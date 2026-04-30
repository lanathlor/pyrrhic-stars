package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"
	"sync"

	"codex-online/server/internal/codec"
	"codex-online/server/internal/container"
	"codex-online/server/internal/entity"
	"codex-online/server/internal/group"
	"codex-online/server/internal/message"
	"codex-online/server/internal/session"
	"codex-online/server/internal/zone"
)

// gateway manages zones, player sessions, and groups.
type gateway struct {
	container *container.Container
	zones     map[string]*zoneInstance
	sessions  *session.Registry
	groups    *group.Manager
	mu        sync.Mutex // protects zones
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
		container: ctr,
		zones:     make(map[string]*zoneInstance),
		sessions:  session.NewRegistry(),
		groups:    group.NewManager(),
	}
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
func (g *gateway) transferPlayer(sess *session.Session, targetZoneID string, targetType zone.ZoneType) {
	// Remove from old zone.
	if sess.ZoneID != "" {
		oldZI := g.getZone(sess.ZoneID)
		if oldZI != nil {
			oldZI.zone.RemoveClient(sess.PeerID)
			disconnMsg := message.Encode(message.OpPeerDisconnected, 0, codec.EncodePeerID(sess.PeerID))
			oldZI.zone.Broadcast(disconnMsg, sess.PeerID)
			if oldZI.zoneType == zone.ZoneTypeArena && oldZI.zone.ClientCount() == 0 {
				g.removeZone(sess.ZoneID)
			}
		}
	}

	// Create/get target zone.
	zi := g.getOrCreateZone(targetZoneID, targetType)

	if targetType == zone.ZoneTypeArena {
		zi.zone.OnPlayerRespawnHub = func(peerID uint16) {
			g.handlePlayerRespawnHub(targetZoneID, peerID)
		}
	}

	// Allocate new peer ID.
	zi.mu.Lock()
	newPeerID := zi.nextID
	zi.nextID++
	zi.mu.Unlock()

	sess.ZoneID = targetZoneID
	sess.PeerID = newPeerID

	// Send zone transfer to client.
	transferPayload := make([]byte, 3)
	transferPayload[0] = byte(targetType)
	binary.BigEndian.PutUint16(transferPayload[1:3], newPeerID)
	sess.Conn.Send(message.Encode(message.OpZoneTransfer, 0, transferPayload))

	// Register in new zone.
	displayName := sess.CharName
	if displayName == "" {
		displayName = sess.Username
	}
	if displayName == "" {
		displayName = fmt.Sprintf("Player_%d", sess.ID)
	}
	zi.zone.AddClient(&zone.Client{
		PeerID:   newPeerID,
		Username: displayName,
		Send:     sess.Conn.Send,
	})

	if sess.Class != "" && sess.Class != entity.ClassGunner {
		zi.zone.QueueInput(newPeerID, message.OpInteractInput, codec.EncodeInteractInput(message.InteractClassSelect, sess.Class))
	}

	// Restore saved position when transferring back to hub.
	if targetType == zone.ZoneTypeHub && sess.CharID != 0 {
		if ch, _ := g.container.Repo.GetCharacterByID(sess.CharID); ch != nil && (ch.PosX != 0 || ch.PosY != 0 || ch.PosZ != 0) {
			zi.zone.SetPlayerPosition(newPeerID, entity.Vec3{
				X: float32(ch.PosX),
				Y: float32(ch.PosY),
				Z: float32(ch.PosZ),
			}, float32(ch.RotY))
		}
	}

	// Notify existing clients about new peer.
	peerMsg := message.Encode(message.OpPeerConnected, 0, codec.EncodePeerID(newPeerID))
	zi.zone.Broadcast(peerMsg, newPeerID)

	// Notify new client about existing peers.
	for _, existingID := range zi.zone.GetPeerIDs() {
		if existingID == newPeerID {
			continue
		}
		sess.Conn.Send(message.Encode(message.OpPeerConnected, 0, codec.EncodePeerID(existingID)))
	}

	slog.Info("player transferred", "player_id", sess.ID, "to_zone", targetZoneID, "new_peer", newPeerID)
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
	g.transferPlayer(sess, zone.ZoneHub, zone.ZoneTypeHub)

	grp := g.groups.GetGroup(sess.ID)
	if grp != nil {
		g.broadcastGroupState(grp)
	}
}

