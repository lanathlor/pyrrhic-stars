package main

import (
	"log/slog"

	"codex-online/server/internal/message"
	"codex-online/server/internal/session"
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
	if message.IsMerchantRelated(opcode) {
		gw.handleMerchantMessage(sess, opcode, payload)
		return
	}
	if message.IsFriendRelated(opcode) {
		gw.handleFriendMessage(sess, opcode, payload)
		return
	}

	switch opcode {
	case message.OpJoinZone:
		zoneID := string(payload)
		if zoneID == "" {
			zoneID = defaultOpenWorldZone
		}
		lvl, err := gw.loadLevel(zoneID)
		if err != nil {
			slog.Warn("level not found for OpJoinZone", "zone_id", zoneID, "error", err)
			return
		}
		zi := gw.getOrCreateZone(zoneID, lvl, 1, nil)
		gw.joinZone(sess, zi, joinResponseZoneJoined, "")

	default:
		slog.Warn("unknown server opcode", "opcode", opcode)
	}
}

// joinHubAfterCharSelect handles the shared logic for joining the open-world
// zone after a character is selected or created. In dev mode, skips the
// open-world zone and transfers directly to an instanced zone.
func (g *gateway) joinHubAfterCharSelect(sess *session.Session) {
	lvl, err := g.loadLevel(defaultOpenWorldZone)
	if err != nil {
		slog.Error("open-world level not found", "error", err)
		return
	}
	zi := g.getOrCreateZone(defaultOpenWorldZone, lvl, 0, nil)
	g.joinZone(sess, zi, joinResponseZoneJoined, "")
}

// devJoinZone joins the player to a zone specified by the dev client.
// Used by the auto-join path to let the editor choose the starting zone.
func (g *gateway) devJoinZone(sess *session.Session, devZone string) {
	baseZone := devZone
	if baseZone == "" {
		baseZone = defaultPortalTarget
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
	g.transferPlayer(sess, instanceID, lvl, groupSize, nil, "")
}
