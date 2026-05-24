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
		zoneType := zone.ZoneTypeOpenWorld
		if strings.HasPrefix(zoneID, "arena") {
			zoneType = zone.ZoneTypeInstanced
		}
		zi := gw.getOrCreateZone(zoneID, zoneType, 1)
		gw.joinZone(sess, zi, joinResponseZoneJoined)

	default:
		slog.Warn("unknown server opcode", "opcode", opcode)
	}
}

// joinHubAfterCharSelect handles the shared logic for joining the hub zone
// after a character is selected or created. In dev mode, skips hub and
// transfers directly to an arena instance.
func (g *gateway) joinHubAfterCharSelect(sess *session.Session) {
	if g.devMode {
		g.transferPlayer(sess, "arena_dev", zone.ZoneTypeInstanced, 1)
		return
	}
	zi := g.getOrCreateZone(zone.ZoneHub, zone.ZoneTypeOpenWorld, 0)
	g.joinZone(sess, zi, joinResponseZoneJoined)
}

// devJoinZone joins the player to a zone specified by the dev client.
// Used by the auto-join path to let the editor choose the starting zone.
func (g *gateway) devJoinZone(sess *session.Session, devZone string) {
	if devZone == "" || devZone == "arena" {
		devZone = "arena_dev"
	}
	zoneType := zone.ZoneTypeInstanced
	groupSize := 1
	if devZone == "hub" {
		zoneType = zone.ZoneTypeOpenWorld
		groupSize = 0
	}
	g.transferPlayer(sess, devZone, zoneType, groupSize)
}
