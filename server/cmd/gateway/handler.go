package main

import (
	"log/slog"

	"codex-online/server/internal/message"
	"codex-online/server/internal/session"
	"codex-online/server/internal/zone"
)

func handleServerMessage(gw *gateway, sess *session.Session, opcode uint16, payload []byte) {
	if message.IsGroupRelated(opcode) {
		gw.handleGroupMessage(sess, opcode, payload)
		return
	}
	if message.IsUserRelated(opcode) {
		gw.handleUserMessage(sess, opcode, payload)
		return
	}
	if message.IsCharacterRelated(opcode) {
		gw.handleCharacterMessage(sess, opcode, payload)
		return
	}
	if message.IsInventoryRelated(opcode) {
		gw.handleInventoryMessage(sess, opcode, payload)
		return
	}
	if message.IsLoadoutRelated(opcode) {
		gw.handleLoadoutMessage(sess, opcode, payload)
		return
	}

	switch opcode {
	case message.OpJoinZone:
		zoneID := string(payload)
		if zoneID == "" {
			zoneID = zone.ZoneHub
		}
		lvl, err := gw.loadLevel(zoneID)
		if err != nil {
			slog.Warn("level not found for OpJoinZone", "zone_id", zoneID, "error", err)
			return
		}
		zi := gw.getOrCreateZone(zoneID, lvl, 1)
		gw.joinZone(sess, zi, joinResponseZoneJoined)

	default:
		slog.Warn("unknown server opcode", "opcode", opcode)
	}
}

// joinHubAfterCharSelect handles the shared logic for joining the hub zone
// after a character is selected or created. In dev mode, skips hub and
// transfers directly to an arena instance.
func (g *gateway) joinHubAfterCharSelect(sess *session.Session) {
	hubLvl, err := g.loadLevel("hub")
	if err != nil {
		slog.Error("hub level not found", "error", err)
		return
	}
	zi := g.getOrCreateZone(zone.ZoneHub, hubLvl, 0)
	g.joinZone(sess, zi, joinResponseZoneJoined)
}

// devJoinZone joins the player to a zone specified by the dev client.
// Used by the auto-join path to let the editor choose the starting zone.
func (g *gateway) devJoinZone(sess *session.Session, devZone string) {
	baseZone := devZone
	if baseZone == "" {
		baseZone = "arena"
	}
	lvl, err := g.loadLevel(baseZone)
	if err != nil {
		slog.Error("dev join zone: level not found", "zone", baseZone, "error", err)
		return
	}
	instanceID := baseZone
	groupSize := 0
	if lvl.ZoneType == "instanced" {
		instanceID = baseZone + "_dev"
		groupSize = 1
	}
	g.transferPlayer(sess, instanceID, lvl, groupSize)
}
