package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"codex-online/server/internal/container"
	"codex-online/server/internal/entity"
	"codex-online/server/internal/group"
	"codex-online/server/internal/message"
	"codex-online/server/internal/persistence"
	"codex-online/server/internal/telemetry"
	"codex-online/server/internal/zone"

	"github.com/coder/websocket"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// playerSession represents a connected player across zone transfers.
type playerSession struct {
	id         uint32 // permanent global ID (assigned once at connect)
	playerUUID string // persistent identity from client
	username   string
	conn       *wsClient
	zoneID     string // current zone
	peerID     uint16 // current zone peer ID
	class      string // selected class
	charID     uint   // selected character ID (persistence primary key)
	charName   string // character display name (shown overhead)
}

// gateway manages zones, player sessions, and groups.
type gateway struct {
	container  *container.Container
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

func newGateway(ctr *container.Container) *gateway {
	return &gateway{
		container:  ctr,
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

	// If arena, set callbacks
	if targetType == zone.ZoneTypeArena {
		zi.zone.OnPlayerRespawnHub = func(peerID uint16) {
			g.handlePlayerRespawnHub(targetZoneID, peerID)
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

	// Register in new zone (character name shown overhead, fall back to account username).
	displayName := sess.charName
	if displayName == "" {
		displayName = sess.username
	}
	if displayName == "" {
		displayName = fmt.Sprintf("Player_%d", sess.id)
	}
	zi.zone.AddClient(&zone.Client{
		PeerID:   newPeerID,
		Username: displayName,
		Send:     client.sendMsg,
	})

	// Set class on the player in the new zone
	if sess.class != "" && sess.class != "gunner" {
		zi.zone.QueueInput(newPeerID, message.OpInteractInput, encodeInteractClassSelect(sess.class))
	}

	// Restore saved position when transferring back to hub.
	if targetType == zone.ZoneTypeHub && sess.charID != 0 {
		if ch, _ := g.container.Repo.GetCharacterByID(sess.charID); ch != nil && (ch.PosX != 0 || ch.PosY != 0 || ch.PosZ != 0) {
			zi.zone.SetPlayerPosition(newPeerID, entity.Vec3{
				X: float32(ch.PosX),
				Y: float32(ch.PosY),
				Z: float32(ch.PosZ),
			}, float32(ch.RotY))
		}
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

// handlePlayerRespawnHub transfers a single dead player back to the hub.
func (g *gateway) handlePlayerRespawnHub(zoneID string, peerID uint16) {
	globalID := g.resolveZonePeerToGlobal(zoneID, peerID)
	if globalID == 0 {
		return
	}
	sess := g.getSessionByID(globalID)
	if sess == nil {
		return
	}
	slog.Info("player respawning to hub", "player_id", sess.id, "from_zone", zoneID)
	g.transferPlayer(sess, "hub", zone.ZoneTypeHub)

	// Re-broadcast group state with updated hub peer ID
	grp := g.groups.GetGroup(sess.id)
	if grp != nil {
		g.broadcastGroupState(grp)
	}
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

	// Initialize persistence.
	dbDriver := os.Getenv("DB_DRIVER")
	if dbDriver == "" {
		dbDriver = "sqlite"
	}
	pgDSN := os.Getenv("POSTGRES_DSN")
	repo, err := persistence.NewGormRepo(dbDriver, pgDSN)
	if err != nil {
		slog.Error("database init failed", "driver", dbDriver, "error", err)
		os.Exit(1)
	}
	slog.Info("database initialized", "driver", dbDriver)

	ctr := container.New(repo)
	gw := newGateway(ctr)

	// Create persistent hub zone at startup.
	gw.getOrCreateZone("hub", zone.ZoneTypeHub)

	// Start periodic position flush (every 30s).
	flushCtx, flushCancel := context.WithCancel(ctx)
	defer flushCancel()
	go gw.periodicFlush(flushCtx)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, req *http.Request) {
		// Authenticate at HTTP handshake via query params.
		playerUUID := req.URL.Query().Get("uuid")
		username := req.URL.Query().Get("username")

		if !isValidUUID(playerUUID) {
			http.Error(w, "invalid or missing uuid", http.StatusUnauthorized)
			return
		}
		username = strings.TrimSpace(username)
		if username == "" {
			username = "Player"
		}
		if len(username) > 20 {
			username = username[:20]
		}

		// Upsert player in DB (only sets username on first creation).
		if err := gw.container.Repo.UpsertPlayer(playerUUID, username); err != nil {
			slog.Error("upsert player", "uuid", playerUUID, "error", err)
			http.Error(w, "auth failed", http.StatusInternalServerError)
			return
		}

		// Use the stored username as authoritative (query param only used for creation).
		player, err := gw.container.Repo.GetPlayer(playerUUID)
		if err != nil || player == nil {
			slog.Error("get player after upsert", "uuid", playerUUID, "error", err)
			http.Error(w, "auth failed", http.StatusInternalServerError)
			return
		}
		username = player.Username

		// Load all characters for the selection screen.
		allChars, _ := gw.container.Repo.GetCharacters(playerUUID)

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
		sess.playerUUID = playerUUID
		sess.username = username
		// sess.class is set when client sends OpSelectCharacter.
		// Default to "gunner" so disconnect-save still works if client disconnects early.
		sess.class = "gunner"
		if len(allChars) > 0 {
			sess.class = allChars[0].ClassName // most recently played
		}

		// Send character list for the selection screen.
		client.sendMsg(encodeCharacterList(username, allChars))
		defer func() {
			// Save character position before cleanup (hub only).
			if sess.playerUUID != "" && sess.class != "" && sess.zoneID == "hub" {
				gw.savePlayerPosition(sess)
			}
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
				// Track class selection in gateway session.
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

	case message.OpSelectCharacter:
		if len(payload) < 4 {
			return
		}
		charID := uint(binary.LittleEndian.Uint32(payload[0:4]))

		char, err := gw.container.Repo.GetCharacterByID(charID)
		if err != nil || char == nil || char.PlayerID != sess.playerUUID {
			client.sendMsg(encodeCharacterError(5, "Character not found"))
			return
		}

		sess.charID = char.ID
		sess.class = char.ClassName
		sess.charName = char.Name
		client.sendMsg(encodeCharacterState(char))

		// Auto-join hub zone.
		gw.joinHubAfterCharSelect(sess, client, char)

	case message.OpCreateCharacter:
		gw.handleCreateCharacter(sess, client, payload)

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

		// Use character name overhead, fall back to account username.
		displayName := sess.charName
		if displayName == "" {
			displayName = sess.username
		}
		if displayName == "" {
			displayName = fmt.Sprintf("Player_%d", sess.id)
			sess.username = displayName
		}

		// Register client in zone with send function
		zi.zone.AddClient(&zone.Client{
			PeerID:   peerID,
			Username: displayName,
			Send:     client.sendMsg,
		})

		// Restore saved position for hub zone (overrides default spawn).
		if zoneType == zone.ZoneTypeHub && sess.charID != 0 {
			if ch, _ := gw.container.Repo.GetCharacterByID(sess.charID); ch != nil && (ch.PosX != 0 || ch.PosY != 0 || ch.PosZ != 0) {
				zi.zone.SetPlayerPosition(peerID, entity.Vec3{
					X: float32(ch.PosX),
					Y: float32(ch.PosY),
					Z: float32(ch.PosZ),
				}, float32(ch.RotY))
			}
		}

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
		slog.Info("peer joined zone", "zone_id", zoneID, "peer_id", peerID, "username", displayName)

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
		// Group arena if grouped, solo arena otherwise
		grp := gw.groups.GetGroup(sess.id)
		var arenaID string
		if grp != nil {
			arenaID = fmt.Sprintf("arena_g%d", grp.ID)
		} else {
			arenaID = fmt.Sprintf("arena_s%d", sess.id)
		}
		slog.Info("player entering portal", "player_id", sess.id, "arena", arenaID)
		gw.transferPlayer(sess, arenaID, zone.ZoneTypeArena)
		if grp != nil {
			gw.broadcastGroupState(grp)
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

// isValidUUID checks that s is a well-formed UUID (36 chars, dashes at 8/13/18/23).
func isValidUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
		} else {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return false
			}
		}
	}
	return true
}

// encodeCharacterState builds an OpCharacterState message.
// Format: [charID:u32 LE][classLen:u8][class:...][nameLen:u8][name:...]
//
//	[posX:f32 LE][posY:f32 LE][posZ:f32 LE][rotY:f32 LE]
func encodeCharacterState(c *persistence.Character) []byte {
	classBytes := []byte(c.ClassName)
	nameBytes := []byte(c.Name)
	buf := make([]byte, 0, 4+1+len(classBytes)+1+len(nameBytes)+16)

	b4 := make([]byte, 4)
	binary.LittleEndian.PutUint32(b4, uint32(c.ID))
	buf = append(buf, b4...)

	buf = append(buf, byte(len(classBytes)))
	buf = append(buf, classBytes...)
	buf = append(buf, byte(len(nameBytes)))
	buf = append(buf, nameBytes...)

	for _, f := range [4]float32{float32(c.PosX), float32(c.PosY), float32(c.PosZ), float32(c.RotY)} {
		b := make([]byte, 4)
		binary.LittleEndian.PutUint32(b, math.Float32bits(f))
		buf = append(buf, b...)
	}

	return message.Encode(message.OpCharacterState, 0, buf)
}

// encodeCharacterList builds an OpCharacterList message.
// Format: [usernameLen:u8][username:...]
//
//	[count:u8] per char: [charID:u32 LE][classLen:u8][class:...][nameLen:u8][name:...]
//	                     [posX:f32 LE][posY:f32 LE][posZ:f32 LE][rotY:f32 LE]
//	[lastCharID:u32 LE]   // first char's ID (ordered by updated_at DESC), or 0
func encodeCharacterList(username string, chars []*persistence.Character) []byte {
	buf := make([]byte, 0, 256)

	usernameBytes := []byte(username)
	buf = append(buf, byte(len(usernameBytes)))
	buf = append(buf, usernameBytes...)

	buf = append(buf, byte(len(chars)))
	for _, c := range chars {
		b4 := make([]byte, 4)
		binary.LittleEndian.PutUint32(b4, uint32(c.ID))
		buf = append(buf, b4...)

		classBytes := []byte(c.ClassName)
		buf = append(buf, byte(len(classBytes)))
		buf = append(buf, classBytes...)

		nameBytes := []byte(c.Name)
		buf = append(buf, byte(len(nameBytes)))
		buf = append(buf, nameBytes...)

		for _, f := range [4]float32{float32(c.PosX), float32(c.PosY), float32(c.PosZ), float32(c.RotY)} {
			b := make([]byte, 4)
			binary.LittleEndian.PutUint32(b, math.Float32bits(f))
			buf = append(buf, b...)
		}
	}

	// Last-played character ID (first entry since ordered by updated_at DESC).
	lastID := make([]byte, 4)
	if len(chars) > 0 {
		binary.LittleEndian.PutUint32(lastID, uint32(chars[0].ID))
	}
	buf = append(buf, lastID...)

	return message.Encode(message.OpCharacterList, 0, buf)
}

// encodeCharacterError builds an OpCharacterError message.
// Format: [code:u8][msgLen:u8][msg:...]
// Error codes: 1=name taken, 2=limit reached, 3=invalid name, 4=invalid class, 5=not found
func encodeCharacterError(code uint8, msg string) []byte {
	msgBytes := []byte(msg)
	buf := make([]byte, 0, 2+len(msgBytes))
	buf = append(buf, code)
	buf = append(buf, byte(len(msgBytes)))
	buf = append(buf, msgBytes...)
	return message.Encode(message.OpCharacterError, 0, buf)
}

// handleCreateCharacter processes OpCreateCharacter.
// Payload: [classLen:u8][class:...][nameLen:u8][name:...]
func (g *gateway) handleCreateCharacter(sess *playerSession, client *wsClient, payload []byte) {
	if len(payload) < 2 {
		return
	}

	// Parse className.
	classLen := int(payload[0])
	if len(payload) < 1+classLen+1 {
		return
	}
	className := string(payload[1 : 1+classLen])

	// Parse charName.
	nameLen := int(payload[1+classLen])
	if len(payload) < 1+classLen+1+nameLen {
		return
	}
	charName := strings.TrimSpace(string(payload[1+classLen+1 : 1+classLen+1+nameLen]))

	// Validate class.
	validClasses := map[string]bool{"gunner": true, "vanguard": true, "blade_dancer": true}
	if !validClasses[className] {
		client.sendMsg(encodeCharacterError(4, "Invalid class"))
		return
	}

	// Validate name: 2-20 chars, alphanumeric + spaces only.
	if len(charName) < 2 || len(charName) > 20 {
		client.sendMsg(encodeCharacterError(3, "Name must be 2-20 characters"))
		return
	}
	for _, r := range charName {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == ' ' || r == '-' || r == '_') {
			client.sendMsg(encodeCharacterError(3, "Name must be alphanumeric (spaces, hyphens, underscores allowed)"))
			return
		}
	}

	// Check character limit.
	count, err := g.container.Repo.CountCharacters(sess.playerUUID)
	if err != nil {
		slog.Error("count characters", "error", err)
		client.sendMsg(encodeCharacterError(2, "Failed to check limit"))
		return
	}
	if count >= 100 {
		client.sendMsg(encodeCharacterError(2, "Character limit reached"))
		return
	}

	// Check name uniqueness.
	taken, err := g.container.Repo.IsCharacterNameTaken(charName)
	if err != nil {
		slog.Error("check name taken", "error", err)
		client.sendMsg(encodeCharacterError(1, "Name already taken"))
		return
	}
	if taken {
		client.sendMsg(encodeCharacterError(1, "Name already taken"))
		return
	}

	// Create character.
	char := &persistence.Character{
		PlayerID:  sess.playerUUID,
		ClassName: className,
		Name:      charName,
	}
	if err := g.container.Repo.CreateCharacter(char); err != nil {
		slog.Error("create character", "error", err)
		client.sendMsg(encodeCharacterError(1, "Name already taken"))
		return
	}

	sess.charID = char.ID
	sess.class = char.ClassName
	sess.charName = char.Name
	client.sendMsg(encodeCharacterState(char))

	// Auto-join hub zone.
	g.joinHubAfterCharSelect(sess, client, char)
}

// joinHubAfterCharSelect handles the shared logic for joining the hub zone
// after a character is selected or created.
func (g *gateway) joinHubAfterCharSelect(sess *playerSession, client *wsClient, char *persistence.Character) {
	zoneID := "hub"
	zi := g.getOrCreateZone(zoneID, zone.ZoneTypeHub)

	zi.mu.Lock()
	peerID := zi.nextID
	zi.nextID++
	zi.mu.Unlock()

	client.peerID = peerID
	client.zoneID = zoneID
	sess.zoneID = zoneID
	sess.peerID = peerID

	displayName := sess.charName
	if displayName == "" {
		displayName = sess.username
	}
	if displayName == "" {
		displayName = fmt.Sprintf("Player_%d", sess.id)
	}

	zi.zone.AddClient(&zone.Client{
		PeerID:   peerID,
		Username: displayName,
		Send:     client.sendMsg,
	})

	// Restore saved position.
	if char != nil && (char.PosX != 0 || char.PosY != 0 || char.PosZ != 0) {
		zi.zone.SetPlayerPosition(peerID, entity.Vec3{
			X: float32(char.PosX),
			Y: float32(char.PosY),
			Z: float32(char.PosZ),
		}, float32(char.RotY))
	}

	// Set class on zone player (if not gunner, which is the default).
	if sess.class != "" && sess.class != "gunner" {
		zi.zone.QueueInput(peerID, message.OpInteractInput, encodeInteractClassSelect(sess.class))
	}

	// Notify about existing peers.
	for _, existingID := range zi.zone.GetPeerIDs() {
		if existingID == peerID {
			continue
		}
		client.sendMsg(message.Encode(message.OpPeerConnected, 0, encodePeerID(existingID)))
	}

	// Send ZoneJoined.
	resp := make([]byte, 3)
	binary.BigEndian.PutUint16(resp[0:2], peerID)
	resp[2] = 0
	client.sendMsg(message.Encode(message.OpZoneJoined, 0, resp))
	slog.Info("character selected, joined hub", "player_id", sess.id, "char_id", sess.charID, "class", sess.class, "peer_id", peerID)

	// Notify existing clients.
	peerMsg := message.Encode(message.OpPeerConnected, 0, encodePeerID(peerID))
	zi.zone.Broadcast(peerMsg, peerID)
}

// savePlayerPosition snapshots a player's hub position to the database.
func (g *gateway) savePlayerPosition(sess *playerSession) {
	zi := g.getZone("hub")
	if zi == nil {
		return
	}
	p := zi.zone.GetPlayer(sess.peerID)
	if p == nil {
		return
	}
	if sess.charID == 0 {
		return
	}
	if err := g.container.Repo.UpdateCharacterPosition(
		sess.charID,
		float64(p.Position.X),
		float64(p.Position.Y),
		float64(p.Position.Z),
		float64(p.RotationY),
	); err != nil {
		slog.Error("save player position", "uuid", sess.playerUUID, "error", err)
	}
}

// periodicFlush saves all hub player positions every 30 seconds.
func (g *gateway) periodicFlush(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			g.flushAllPositions()
		case <-ctx.Done():
			return
		}
	}
}

func (g *gateway) flushAllPositions() {
	g.mu.Lock()
	type snapshot struct {
		playerUUID string
		charID     uint
		peerID     uint16
	}
	var toSave []snapshot
	for _, sess := range g.sessions {
		if sess.playerUUID != "" && sess.zoneID == "hub" && sess.charID != 0 {
			toSave = append(toSave, snapshot{sess.playerUUID, sess.charID, sess.peerID})
		}
	}
	g.mu.Unlock()

	if len(toSave) == 0 {
		return
	}

	hubZI := g.getZone("hub")
	if hubZI == nil {
		return
	}

	saved := 0
	for _, s := range toSave {
		if s.charID == 0 {
			continue
		}
		p := hubZI.zone.GetPlayer(s.peerID)
		if p == nil {
			continue
		}
		if err := g.container.Repo.UpdateCharacterPosition(
			s.charID,
			float64(p.Position.X),
			float64(p.Position.Y),
			float64(p.Position.Z),
			float64(p.RotationY),
		); err != nil {
			slog.Error("periodic flush", "uuid", s.playerUUID, "error", err)
		} else {
			saved++
		}
	}
	if saved > 0 {
		slog.Debug("periodic flush completed", "saved", saved)
	}
}
