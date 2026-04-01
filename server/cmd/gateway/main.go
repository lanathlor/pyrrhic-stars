package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"codex-online/server/internal/group"
	"codex-online/server/internal/message"
	"codex-online/server/internal/telemetry"
	"codex-online/server/internal/zone"

	"github.com/coder/websocket"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// playerSession represents a connected player across zone transfers.
type playerSession struct {
	id       uint32 // permanent global ID (assigned once at connect)
	username string
	conn     *wsClient
	zoneID   string // current zone
	peerID   uint16 // current zone peer ID
	class    string // selected class
}

// gateway manages zones, player sessions, and groups.
type gateway struct {
	zones      map[string]*zoneInstance
	sessions   map[uint32]*playerSession // globalID → session
	connMap    map[*wsClient]uint32      // conn → globalID
	groups     *group.Manager
	nextPlayer uint32
	mu         sync.Mutex
}

type zoneInstance struct {
	zone     *zone.Zone
	zoneType zone.ZoneType
	cancel   context.CancelFunc
	nextID   uint16
	mu       sync.Mutex
}

func newGateway() *gateway {
	return &gateway{
		zones:      make(map[string]*zoneInstance),
		sessions:   make(map[uint32]*playerSession),
		connMap:    make(map[*wsClient]uint32),
		groups:     group.NewManager(),
		nextPlayer: 1,
	}
}

// registerSession creates a global player session for a new connection.
func (g *gateway) registerSession(client *wsClient) *playerSession {
	g.mu.Lock()
	defer g.mu.Unlock()
	id := g.nextPlayer
	g.nextPlayer++
	sess := &playerSession{
		id:    id,
		conn:  client,
		class: "gunner",
	}
	g.sessions[id] = sess
	g.connMap[client] = id
	return sess
}

// getSession returns the session for a wsClient.
func (g *gateway) getSession(client *wsClient) *playerSession {
	g.mu.Lock()
	defer g.mu.Unlock()
	id, ok := g.connMap[client]
	if !ok {
		return nil
	}
	return g.sessions[id]
}

// removeSession cleans up a disconnected player's session.
func (g *gateway) removeSession(client *wsClient) *playerSession {
	g.mu.Lock()
	defer g.mu.Unlock()
	id, ok := g.connMap[client]
	if !ok {
		return nil
	}
	sess := g.sessions[id]
	delete(g.connMap, client)
	delete(g.sessions, id)
	return sess
}

// getOrCreateZone returns the zone for the given ID, creating it if needed.
func (g *gateway) getOrCreateZone(zoneID string, zoneType zone.ZoneType) *zoneInstance {
	g.mu.Lock()
	defer g.mu.Unlock()
	zi, ok := g.zones[zoneID]
	if !ok {
		z := zone.New(zoneID, zoneType)
		ctx, cancel := context.WithCancel(context.Background())
		zi = &zoneInstance{zone: z, zoneType: zoneType, cancel: cancel, nextID: 1}
		g.zones[zoneID] = zi
		go z.Run(ctx)
		slog.Info("zone created", "zone_id", zoneID, "type", zoneType)
	}
	return zi
}

// getZone returns an existing zone or nil.
func (g *gateway) getZone(zoneID string) *zoneInstance {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.zones[zoneID]
}

func (g *gateway) removeZone(zoneID string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if zi, ok := g.zones[zoneID]; ok {
		zi.cancel()
		delete(g.zones, zoneID)
		slog.Info("zone removed", "zone_id", zoneID)
	}
}

// transferPlayer moves a player from their current zone to a new zone.
func (g *gateway) transferPlayer(sess *playerSession, targetZoneID string, targetType zone.ZoneType) {
	client := sess.conn

	// Remove from old zone
	if sess.zoneID != "" {
		oldZI := g.getZone(sess.zoneID)
		if oldZI != nil {
			oldZI.zone.RemoveClient(sess.peerID)
			disconnMsg := message.Encode(message.OpPeerDisconnected, 0, encodePeerID(sess.peerID))
			oldZI.zone.Broadcast(disconnMsg, sess.peerID)
			// Clean up empty arena zones
			if oldZI.zoneType == zone.ZoneTypeArena && oldZI.zone.ClientCount() == 0 {
				g.removeZone(sess.zoneID)
			}
		}
	}

	// Create/get target zone
	zi := g.getOrCreateZone(targetZoneID, targetType)

	// If arena, set OnResultEnd callback
	if targetType == zone.ZoneTypeArena {
		zi.zone.OnResultEnd = func(zoneID string) {
			g.handleArenaResultEnd(zoneID)
		}
	}

	// Allocate new peer ID
	zi.mu.Lock()
	newPeerID := zi.nextID
	zi.nextID++
	zi.mu.Unlock()

	// Update session
	sess.zoneID = targetZoneID
	sess.peerID = newPeerID
	client.zoneID = targetZoneID
	client.peerID = newPeerID

	// Send zone transfer to client
	transferPayload := make([]byte, 3)
	transferPayload[0] = byte(targetType)
	binary.BigEndian.PutUint16(transferPayload[1:3], newPeerID)
	client.sendMsg(message.Encode(message.OpZoneTransfer, 0, transferPayload))

	// Register in new zone
	username := sess.username
	if username == "" {
		username = fmt.Sprintf("Player_%d", sess.id)
	}
	zi.zone.AddClient(&zone.Client{
		PeerID:   newPeerID,
		Username: username,
		Send:     client.sendMsg,
	})

	// Set class on the player in the new zone
	if sess.class != "" && sess.class != "gunner" {
		zi.zone.QueueInput(newPeerID, message.OpInteractInput, encodeInteractClassSelect(sess.class))
	}

	// Notify existing clients about new peer
	peerMsg := message.Encode(message.OpPeerConnected, 0, encodePeerID(newPeerID))
	zi.zone.Broadcast(peerMsg, newPeerID)

	// Notify new client about existing peers
	for _, existingID := range zi.zone.GetPeerIDs() {
		if existingID == newPeerID {
			continue
		}
		existingMsg := message.Encode(message.OpPeerConnected, 0, encodePeerID(existingID))
		client.sendMsg(existingMsg)
	}

	slog.Info("player transferred", "player_id", sess.id, "to_zone", targetZoneID, "new_peer", newPeerID)
}

// handleArenaResultEnd is called when an arena zone's result timer expires.
func (g *gateway) handleArenaResultEnd(zoneID string) {
	slog.Info("arena result ended, returning players to hub", "zone_id", zoneID)
	// Collect all sessions in this zone
	g.mu.Lock()
	var sessionsToMove []*playerSession
	for _, sess := range g.sessions {
		if sess.zoneID == zoneID {
			sessionsToMove = append(sessionsToMove, sess)
		}
	}
	g.mu.Unlock()

	// Transfer each player back to hub
	for _, sess := range sessionsToMove {
		g.transferPlayer(sess, "hub", zone.ZoneTypeHub)
	}

	// Destroy the arena zone
	g.removeZone(zoneID)
}

func encodeInteractClassSelect(className string) []byte {
	nameBytes := []byte(className)
	buf := make([]byte, 2+len(nameBytes))
	buf[0] = 0 // InteractClassSelect
	buf[1] = byte(len(nameBytes))
	copy(buf[2:], nameBytes)
	return buf
}

// wsClient wraps a WebSocket connection.
type wsClient struct {
	conn   *websocket.Conn
	peerID uint16
	zoneID string
	send   chan []byte
	ctx    context.Context
	cancel context.CancelFunc
}

func newWSClient(conn *websocket.Conn) *wsClient {
	ctx, cancel := context.WithCancel(context.Background())
	c := &wsClient{
		conn:   conn,
		send:   make(chan []byte, 256),
		ctx:    ctx,
		cancel: cancel,
	}
	go c.writePump()
	return c
}

func (c *wsClient) writePump() {
	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				return
			}
			if err := c.conn.Write(c.ctx, websocket.MessageBinary, msg); err != nil {
				return
			}
		case <-c.ctx.Done():
			return
		}
	}
}

func (c *wsClient) sendMsg(data []byte) {
	select {
	case c.send <- data:
	default:
		slog.Warn("send buffer full, dropping message", "peer_id", c.peerID)
	}
}

func (c *wsClient) close() {
	c.cancel()
	close(c.send)
}

func main() {
	ctx := context.Background()

	shutdown, err := telemetry.Init(ctx)
	if err != nil {
		slog.Error("telemetry init failed", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := shutdown(ctx); err != nil {
			slog.Error("telemetry shutdown", "error", err)
		}
	}()

	gw := newGateway()

	// Create persistent hub zone at startup.
	gw.getOrCreateZone("hub", zone.ZoneTypeHub)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, req *http.Request) {
		conn, err := websocket.Accept(w, req, &websocket.AcceptOptions{
			InsecureSkipVerify: true, // Allow any origin for dev.
		})
		if err != nil {
			slog.Error("websocket accept", "error", err)
			return
		}

		_, connSpan := telemetry.Tracer().Start(req.Context(), "connection",
			trace.WithAttributes(attribute.String("remote_addr", req.RemoteAddr)),
		)

		client := newWSClient(conn)
		sess := gw.registerSession(client)
		defer func() {
			// Remove from group
			if g, disbanded := gw.groups.LeaveGroup(sess.id); !disbanded && g != nil {
				gw.broadcastGroupState(g)
			}
			// Remove from current zone
			if client.zoneID != "" {
				zi := gw.getZone(client.zoneID)
				if zi != nil {
					zi.zone.RemoveClient(client.peerID)
					disconnMsg := message.Encode(message.OpPeerDisconnected, 0, encodePeerID(client.peerID))
					zi.zone.Broadcast(disconnMsg, client.peerID)
					// Clean up empty arena zones (never remove hub)
					if zi.zoneType == zone.ZoneTypeArena && zi.zone.ClientCount() == 0 {
						gw.removeZone(client.zoneID)
					}
				}
			}
			gw.removeSession(client)
			client.close()
			conn.CloseNow()
			connSpan.End()
		}()

		slog.Info("new connection", "remote_addr", req.RemoteAddr, "player_id", sess.id)

		for {
			_, data, readErr := conn.Read(client.ctx)
			if readErr != nil {
				slog.Info("connection closed", "player_id", sess.id, "peer_id", client.peerID, "error", readErr)
				return
			}

			opcode, _, payload, decErr := message.Decode(data)
			if decErr != nil {
				slog.Error("decode", "error", decErr)
				continue
			}

			// Server-handled messages (zone mgmt + group)
			if message.IsServerHandled(opcode) {
				handleServerMessage(gw, client, opcode, payload)
				continue
			}

			// Client input messages → route to zone simulation
			if message.IsClientInput(opcode) {
				// Track class selection in gateway session
				if opcode == message.OpInteractInput && len(payload) >= 3 && payload[0] == message.InteractClassSelect {
					nameLen := int(payload[1])
					if len(payload) >= 2+nameLen {
						className := string(payload[2 : 2+nameLen])
						sess.class = className
					}
				}
				if client.zoneID != "" {
					zi := gw.getZone(client.zoneID)
					if zi != nil {
						zi.zone.QueueInput(client.peerID, opcode, payload)
					}
				}
				continue
			}

			// Legacy relay messages (for backward compat during migration)
			if client.zoneID != "" {
				zi := gw.getZone(client.zoneID)
				if zi != nil {
					outMsg := message.Encode(opcode, client.peerID, payload)
					excludeSender := message.BroadcastExcludeSender(opcode)
					if excludeSender {
						zi.zone.Broadcast(outMsg, client.peerID)
					} else {
						zi.zone.Broadcast(outMsg, 0)
					}
				}
			}
		}
	})

	addr := ":7777"
	if envAddr := os.Getenv("GATEWAY_ADDR"); envAddr != "" {
		addr = envAddr
	}

	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Graceful shutdown on SIGINT/SIGTERM.
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt)
		<-sigCh
		slog.Info("shutting down...")
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(shutCtx)
	}()

	slog.Info("gateway listening", "addr", addr)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		slog.Error("listen failed", "error", err)
		os.Exit(1)
	}
}

func handleServerMessage(gw *gateway, client *wsClient, opcode uint16, payload []byte) {
	sess := gw.getSession(client)
	if sess == nil {
		return
	}

	switch opcode {
	case message.OpSetUsername:
		if len(payload) < 1 {
			return
		}
		nameLen := int(payload[0])
		if len(payload) < 1+nameLen {
			return
		}
		name := strings.TrimSpace(string(payload[1 : 1+nameLen]))
		if name == "" {
			name = fmt.Sprintf("Player_%d", sess.id)
		}
		if len(name) > 20 {
			name = name[:20]
		}
		sess.username = name
		slog.Info("username set", "player_id", sess.id, "username", name)

	case message.OpJoinZone:
		zoneID := string(payload)
		if zoneID == "" {
			zoneID = "hub"
		}

		// Determine zone type from ID
		zoneType := zone.ZoneTypeHub
		if strings.HasPrefix(zoneID, "arena") {
			zoneType = zone.ZoneTypeArena
		}

		zi := gw.getOrCreateZone(zoneID, zoneType)

		zi.mu.Lock()
		peerID := zi.nextID
		zi.nextID++
		zi.mu.Unlock()

		client.peerID = peerID
		client.zoneID = zoneID
		sess.zoneID = zoneID
		sess.peerID = peerID

		// Set username fallback
		username := sess.username
		if username == "" {
			username = fmt.Sprintf("Player_%d", sess.id)
			sess.username = username
		}

		// Register client in zone with send function
		zi.zone.AddClient(&zone.Client{
			PeerID:   peerID,
			Username: username,
			Send:     client.sendMsg,
		})

		// Notify the new client about all existing peers (excluding self)
		for _, existingID := range zi.zone.GetPeerIDs() {
			if existingID == peerID {
				continue
			}
			existingMsg := message.Encode(message.OpPeerConnected, 0, encodePeerID(existingID))
			client.sendMsg(existingMsg)
		}

		// Send ZoneJoined response
		resp := make([]byte, 3)
		binary.BigEndian.PutUint16(resp[0:2], peerID)
		resp[2] = 0 // isHost is no longer meaningful

		client.sendMsg(message.Encode(message.OpZoneJoined, 0, resp))
		slog.Info("peer joined zone", "zone_id", zoneID, "peer_id", peerID, "username", username)

		// Notify existing clients about the new peer
		peerMsg := message.Encode(message.OpPeerConnected, 0, encodePeerID(peerID))
		zi.zone.Broadcast(peerMsg, peerID)

	case message.OpGroupCreate:
		g, err := gw.groups.CreateGroup(sess.id)
		if err != nil {
			sendGroupError(client, err.Error())
			return
		}
		slog.Info("group created", "group_id", g.ID, "leader", sess.id)
		gw.broadcastGroupState(g)

	case message.OpGroupInvite:
		if len(payload) < 2 {
			return
		}
		targetPeerID := binary.LittleEndian.Uint16(payload[0:2])
		// Resolve peer ID in current zone to global player ID
		targetGlobalID := gw.resolveZonePeerToGlobal(sess.zoneID, targetPeerID)
		if targetGlobalID == 0 {
			sendGroupError(client, "player not found")
			return
		}
		invite, err := gw.groups.InvitePlayer(sess.id, targetGlobalID)
		if err != nil {
			sendGroupError(client, err.Error())
			return
		}
		// Send invite notification to target
		targetSess := gw.getSessionByID(targetGlobalID)
		if targetSess != nil {
			sendGroupInviteRecv(targetSess.conn, invite.GroupID, sess.username)
		}
		slog.Info("group invite sent", "from", sess.id, "to", targetGlobalID, "group", invite.GroupID)

	case message.OpGroupInviteReply:
		if len(payload) < 5 {
			return
		}
		groupID := binary.LittleEndian.Uint32(payload[0:4])
		accept := payload[4] == 1
		if accept {
			g, err := gw.groups.AcceptInvite(sess.id, groupID)
			if err != nil {
				sendGroupError(client, err.Error())
				return
			}
			slog.Info("group invite accepted", "player", sess.id, "group", groupID)
			gw.broadcastGroupState(g)
		} else {
			gw.groups.DeclineInvite(sess.id, groupID)
			slog.Info("group invite declined", "player", sess.id, "group", groupID)
		}

	case message.OpGroupLeave:
		g, disbanded := gw.groups.LeaveGroup(sess.id)
		slog.Info("player left group", "player", sess.id, "disbanded", disbanded)
		if !disbanded && g != nil {
			gw.broadcastGroupState(g)
		}
		// Send empty group state to the leaving player
		sendEmptyGroupState(client)

	case message.OpGroupKick:
		if len(payload) < 2 {
			return
		}
		targetPeerID := binary.LittleEndian.Uint16(payload[0:2])
		targetGlobalID := gw.resolveZonePeerToGlobal(sess.zoneID, targetPeerID)
		if targetGlobalID == 0 {
			sendGroupError(client, "player not found")
			return
		}
		g, err := gw.groups.KickPlayer(sess.id, targetGlobalID)
		if err != nil {
			sendGroupError(client, err.Error())
			return
		}
		slog.Info("player kicked from group", "leader", sess.id, "target", targetGlobalID)
		gw.broadcastGroupState(g)
		// Notify kicked player
		targetSess := gw.getSessionByID(targetGlobalID)
		if targetSess != nil {
			sendEmptyGroupState(targetSess.conn)
		}

	case message.OpEnterPortal:
		if sess.zoneID != "hub" {
			sendGroupError(client, "can only enter portal from hub")
			return
		}
		grp := gw.groups.GetGroup(sess.id)
		if grp != nil {
			// Grouped: only leader can trigger
			if grp.LeaderID != sess.id {
				sendGroupError(client, "only the group leader can enter the portal")
				return
			}
			arenaID := fmt.Sprintf("arena_g%d", grp.ID)
			slog.Info("group entering portal", "group_id", grp.ID, "arena", arenaID)
			// Transfer all group members
			for _, memberID := range grp.Members {
				memberSess := gw.getSessionByID(memberID)
				if memberSess != nil {
					gw.transferPlayer(memberSess, arenaID, zone.ZoneTypeArena)
				}
			}
		} else {
			// Solo: own instance
			arenaID := fmt.Sprintf("arena_s%d", sess.id)
			slog.Info("solo player entering portal", "player_id", sess.id, "arena", arenaID)
			gw.transferPlayer(sess, arenaID, zone.ZoneTypeArena)
		}

	default:
		slog.Warn("unknown server opcode", "opcode", opcode)
	}
}

func encodePeerID(id uint16) []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, id)
	return b
}

// getSessionByID returns a session by global player ID.
func (g *gateway) getSessionByID(playerID uint32) *playerSession {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.sessions[playerID]
}

// resolveZonePeerToGlobal finds the global player ID for a zone peer ID.
func (g *gateway) resolveZonePeerToGlobal(zoneID string, peerID uint16) uint32 {
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, sess := range g.sessions {
		if sess.zoneID == zoneID && sess.peerID == peerID {
			return sess.id
		}
	}
	return 0
}

// broadcastGroupState sends OpGroupState to all members of a group.
func (g *gateway) broadcastGroupState(grp *group.Group) {
	buf := encodeGroupState(g, grp)
	msg := message.Encode(message.OpGroupState, 0, buf)
	for _, memberID := range grp.Members {
		sess := g.getSessionByID(memberID)
		if sess != nil {
			sess.conn.sendMsg(msg)
		}
	}
}

// encodeGroupState builds the OpGroupState payload.
func encodeGroupState(gw *gateway, grp *group.Group) []byte {
	buf := make([]byte, 0, 128)
	b4 := make([]byte, 4)
	binary.LittleEndian.PutUint32(b4, grp.ID)
	buf = append(buf, b4...)

	// Leader peer ID in current zone
	leaderSess := gw.getSessionByID(grp.LeaderID)
	leaderPeer := uint16(0)
	if leaderSess != nil {
		leaderPeer = leaderSess.peerID
	}
	b2 := make([]byte, 2)
	binary.LittleEndian.PutUint16(b2, leaderPeer)
	buf = append(buf, b2...)

	buf = append(buf, byte(len(grp.Members)))
	for _, memberID := range grp.Members {
		sess := gw.getSessionByID(memberID)
		peerID := uint16(0)
		name := ""
		if sess != nil {
			peerID = sess.peerID
			name = sess.username
		}
		b2 := make([]byte, 2)
		binary.LittleEndian.PutUint16(b2, peerID)
		buf = append(buf, b2...)
		nameBytes := []byte(name)
		buf = append(buf, byte(len(nameBytes)))
		buf = append(buf, nameBytes...)
	}
	return buf
}

func sendGroupError(client *wsClient, errMsg string) {
	buf := []byte{1} // error code 1 = generic
	msgBytes := []byte(errMsg)
	buf = append(buf, byte(len(msgBytes)))
	buf = append(buf, msgBytes...)
	client.sendMsg(message.Encode(message.OpGroupError, 0, buf))
}

func sendGroupInviteRecv(client *wsClient, groupID uint32, leaderName string) {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, groupID)
	nameBytes := []byte(leaderName)
	buf = append(buf, byte(len(nameBytes)))
	buf = append(buf, nameBytes...)
	client.sendMsg(message.Encode(message.OpGroupInviteRecv, 0, buf))
}

func sendEmptyGroupState(client *wsClient) {
	// group_id=0 means "not in a group"
	buf := make([]byte, 7) // 4 bytes group_id(0) + 2 bytes leader(0) + 1 byte count(0)
	client.sendMsg(message.Encode(message.OpGroupState, 0, buf))
}
