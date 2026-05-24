package main

import (
	"log/slog"

	"codex-online/server/internal/codec"
	"codex-online/server/internal/message"
	"codex-online/server/internal/persistence"
	"codex-online/server/internal/session"
)

func (g *gateway) handleLoadoutMessage(sess *session.Session, opcode uint16, payload []byte) {
	if sess.CharID == 0 {
		return
	}

	switch opcode {
	case message.OpSetLoadout:
		slots, ok := codec.DecodeSetLoadout(payload)
		if !ok {
			return
		}
		// Validate each slot against the ability catalog.
		if g.catalog != nil && !g.catalog.ValidateLoadout(slots) {
			slog.Warn("invalid loadout", "char_id", sess.CharID)
			return
		}
		// Persist.
		if err := g.container.Repo.UpsertLoadout(sess.CharID, slots); err != nil {
			slog.Error("persist loadout", "char_id", sess.CharID, "error", err)
			return
		}
		// Forward to zone so ActionMap updates.
		zi := g.getZone(sess.ZoneID)
		if zi != nil {
			zi.zone.QueueInput(sess.PeerID, message.OpSetLoadout, payload)
		}
		// Confirm to client.
		sess.Conn.Send(message.Encode(message.OpLoadoutState, 0, codec.EncodeLoadoutState(slots)))

	case message.OpSetFluxCommitment:
		entries, ok := codec.DecodeFluxCommitment(payload)
		if !ok {
			return
		}
		// Validate: percentages must sum to 100.
		var total uint8
		for _, e := range entries {
			total += e.Percentage
		}
		if total != 100 {
			slog.Warn("invalid flux commitment: total != 100", "char_id", sess.CharID, "total", total)
			return
		}
		// Persist.
		repoEntries := make([]persistence.FluxCommitmentEntry, len(entries))
		for i, e := range entries {
			repoEntries[i] = persistence.FluxCommitmentEntry{School: e.School, Percentage: e.Percentage}
		}
		if err := g.container.Repo.UpsertFluxCommitment(sess.CharID, repoEntries); err != nil {
			slog.Error("persist flux commitment", "char_id", sess.CharID, "error", err)
			return
		}
		// Forward to zone so FluxCommitment updates on the player entity.
		zi := g.getZone(sess.ZoneID)
		if zi != nil {
			zi.zone.QueueInput(sess.PeerID, message.OpSetFluxCommitment, payload)
		}
		// Confirm to client.
		sess.Conn.Send(message.Encode(message.OpFluxCommitState, 0, codec.EncodeFluxCommitState(entries)))

	default:
		slog.Warn("unknown loadout opcode", "opcode", opcode)
	}
}
