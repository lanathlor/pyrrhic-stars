package main

import (
	"log/slog"
	"strings"

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

	switch opcode {
	case message.OpJoinZone:
		zoneID := string(payload)
		if zoneID == "" {
			zoneID = zone.ZoneHub
		}
		zoneType := zone.ZoneTypeOpenWorld
		if strings.HasPrefix(zoneID, "arena") {
			zoneType = zone.ZoneTypeInstanced
		}
		zi := gw.getOrCreateZone(zoneID, zoneType)
		gw.joinZone(sess, zi, joinResponseZoneJoined)

	default:
		slog.Warn("unknown server opcode", "opcode", opcode)
	}
}

// joinHubAfterCharSelect handles the shared logic for joining the hub zone
// after a character is selected or created.
func (g *gateway) joinHubAfterCharSelect(sess *session.Session) {
	zi := g.getOrCreateZone(zone.ZoneHub, zone.ZoneTypeOpenWorld)
	g.joinZone(sess, zi, joinResponseZoneJoined)
}
