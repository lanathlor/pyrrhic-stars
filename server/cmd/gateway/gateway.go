package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"
	"sync"

	"codex-online/server/internal/character"
	"codex-online/server/internal/codec"
	"codex-online/server/internal/container"
	"codex-online/server/internal/entity"
	"codex-online/server/internal/group"
	"codex-online/server/internal/message"
	"codex-online/server/internal/session"
	"codex-online/server/internal/user"
	"codex-online/server/internal/zone"
)

// gateway manages zones, player sessions, and groups.
type gateway struct {
	container  *container.Container
	zones      map[string]*zoneInstance
	sessions   *session.Registry
	groups     *group.Manager
	users      *user.Service
	characters *character.Service
	mu         sync.Mutex // protects zones
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
		sessions:   session.NewRegistry(),
		groups:     group.NewManager(),
		users:      user.NewService(ctr.Repo),
		characters: character.NewService(ctr.Repo),
	}
}

// getOrCreateZone returns the zone for the given ID, creating it if needed.
// groupSize is used for instance scaling (ignored for open-world zones).
func (g *gateway) getOrCreateZone(zoneID string, zoneType zone.ZoneType, groupSize int) *zoneInstance {
	g.mu.Lock()
	defer g.mu.Unlock()
	zi, ok := g.zones[zoneID]
	if !ok {
		z := zone.New(zoneID, zoneType)
		if zoneType == zone.ZoneTypeInstanced {
			z.SetGroupSize(groupSize)
		}
		z.CombatLogSink = g.container.CombatLogSink
		ctx, cancel := context.WithCancel(context.Background())
		zi = &zoneInstance{zone: z, zoneType: zoneType, cancel: cancel, nextID: 1}
		g.zones[zoneID] = zi
		go z.Run(ctx)
		slog.Info("zone created", "zone_id", zoneID, "type", zoneType, "group_size", groupSize)
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

// joinResponse controls the message sent to the player after joining a zone.
type joinResponse uint8

const (
	joinResponseZoneJoined   joinResponse = iota // OpZoneJoined  [peerID:2][0:1]
	joinResponseZoneTransfer                     // OpZoneTransfer [type:1][peerID:2]
)

// leaveZone removes a player from their current zone, broadcasts their
// departure, and cleans up empty arena zones.
func (g *gateway) leaveZone(sess *session.Session) {
	if sess.ZoneID == "" {
		return
	}
	zi := g.getZone(sess.ZoneID)
	if zi == nil {
		return
	}
	zi.zone.RemoveClient(sess.PeerID)
	disconnMsg := message.Encode(message.OpPeerDisconnected, 0, codec.EncodePeerID(sess.PeerID))
	zi.zone.Broadcast(disconnMsg, sess.PeerID)
	if zi.zoneType == zone.ZoneTypeInstanced && zi.zone.ClientCount() == 0 {
		g.removeZone(sess.ZoneID)
	}
}

// joinZone adds a player to a zone, notifies peers, and sends the appropriate
// response. It handles peer ID allocation, display name resolution, position
// restore (hub zones), and class selection queuing.
func (g *gateway) joinZone(sess *session.Session, zi *zoneInstance, resp joinResponse) {
	// Allocate peer ID.
	zi.mu.Lock()
	peerID := zi.nextID
	zi.nextID++
	zi.mu.Unlock()

	sess.PeerID = peerID
	sess.ZoneID = zi.zone.ID

	// Resolve display name.
	displayName := sess.CharName
	if displayName == "" {
		displayName = sess.Username
	}
	if displayName == "" {
		displayName = fmt.Sprintf("Player_%d", sess.ID)
		sess.Username = displayName
	}

	zi.zone.AddClient(&zone.Client{
		PeerID:   peerID,
		Username: displayName,
		Send:     sess.Conn.Send,
	})

	// Queue class selection for non-gunner characters.
	if sess.Class != "" && sess.Class != entity.ClassGunner {
		zi.zone.QueueInput(peerID, message.OpInteractInput,
			codec.EncodeInteractInput(message.InteractClassSelect, sess.Class))
	}

	// Restore saved position for hub zones.
	if zi.zoneType == zone.ZoneTypeOpenWorld && sess.CharID != 0 {
		if ch, _ := g.container.Repo.GetCharacterByID(sess.CharID); ch != nil && (ch.PosX != 0 || ch.PosY != 0 || ch.PosZ != 0) {
			zi.zone.SetPlayerPosition(peerID, entity.Vec3{
				X: float32(ch.PosX),
				Y: float32(ch.PosY),
				Z: float32(ch.PosZ),
			}, float32(ch.RotY))
		}
	}

	// Notify new peer about existing peers.
	for _, existingID := range zi.zone.GetPeerIDs() {
		if existingID == peerID {
			continue
		}
		sess.Conn.Send(message.Encode(message.OpPeerConnected, 0, codec.EncodePeerID(existingID)))
	}

	// Send response to the joining client.
	switch resp {
	case joinResponseZoneJoined:
		buf := make([]byte, 3)
		binary.BigEndian.PutUint16(buf[0:2], peerID)
		buf[2] = 0
		sess.Conn.Send(message.Encode(message.OpZoneJoined, 0, buf))
	case joinResponseZoneTransfer:
		buf := make([]byte, 3)
		buf[0] = byte(zi.zoneType)
		binary.BigEndian.PutUint16(buf[1:3], peerID)
		sess.Conn.Send(message.Encode(message.OpZoneTransfer, 0, buf))
	}

	// Broadcast new peer to existing peers.
	peerMsg := message.Encode(message.OpPeerConnected, 0, codec.EncodePeerID(peerID))
	zi.zone.Broadcast(peerMsg, peerID)

	slog.Info("peer joined zone", "zone_id", zi.zone.ID, "peer_id", peerID, "username", displayName)
}

// transferPlayer moves a player from their current zone to a new zone.
// groupSize is used for instance scaling when creating a new arena.
func (g *gateway) transferPlayer(sess *session.Session, targetZoneID string, targetType zone.ZoneType, groupSize int) {
	g.leaveZone(sess)

	zi := g.getOrCreateZone(targetZoneID, targetType, groupSize)
	if targetType == zone.ZoneTypeInstanced {
		zi.zone.OnPlayerRespawnHub = func(peerID uint16) {
			g.handlePlayerRespawnHub(targetZoneID, peerID)
		}
	}

	g.joinZone(sess, zi, joinResponseZoneTransfer)
}

// handlePlayerRespawnHub transfers a single dead player back to the hub.
func (g *gateway) handlePlayerRespawnHub(zoneID string, peerID uint16) {
	globalID := g.sessions.ResolveZonePeer(zoneID, peerID)
	if globalID == 0 {
		return
	}
	sess := g.sessions.GetByID(globalID)
	if sess == nil {
		return
	}
	slog.Info("player respawning to hub", "player_id", sess.ID, "from_zone", zoneID)
	g.transferPlayer(sess, zone.ZoneHub, zone.ZoneTypeOpenWorld, 0)

	grp := g.groups.GetGroup(sess.ID)
	if grp != nil {
		g.broadcastGroupState(grp)
	}
}
