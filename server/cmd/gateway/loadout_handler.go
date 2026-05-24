package main

import (
	"log/slog"
	"strconv"
	"strings"

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
		g.handleSetLoadout(sess, payload)

	case message.OpSetFluxCommitment:
		g.handleSetFluxCommitment(sess, payload)

	case message.OpSavePreset:
		g.handleSavePreset(sess, payload)

	case message.OpDeletePreset:
		presetID, ok := codec.DecodeDeletePreset(payload)
		if !ok {
			return
		}
		if err := g.container.Repo.DeleteLoadoutPreset(sess.CharID, uint(presetID)); err != nil {
			slog.Error("delete preset", "char_id", sess.CharID, "error", err)
			return
		}
		g.sendPresetList(sess)

	default:
		slog.Warn("unknown loadout opcode", "opcode", opcode)
	}
}

// handleSetLoadout validates, persists, and applies a new ability loadout.
func (g *gateway) handleSetLoadout(sess *session.Session, payload []byte) {
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
}

// handleSetFluxCommitment validates, persists, and applies a flux commitment update.
func (g *gateway) handleSetFluxCommitment(sess *session.Session, payload []byte) {
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
}

// handleSavePreset validates and persists an ability loadout preset.
func (g *gateway) handleSavePreset(sess *session.Session, payload []byte) {
	name, slots, commitment, ok := codec.DecodeSavePreset(payload)
	if !ok {
		return
	}
	// Validate name.
	if len(name) == 0 || len(name) > 30 {
		slog.Warn("invalid preset name", "char_id", sess.CharID)
		return
	}
	// Validate slots against catalog.
	if g.catalog != nil && !g.catalog.ValidateLoadout(slots) {
		slog.Warn("invalid preset loadout", "char_id", sess.CharID)
		return
	}
	// Validate commitment: parse CSV, check sum.
	if commitment != "" && !validateCommitmentCSV(commitment) {
		slog.Warn("invalid preset commitment", "char_id", sess.CharID)
		return
	}
	// Persist.
	if err := g.container.Repo.SaveLoadoutPreset(sess.CharID, name, slots, commitment); err != nil {
		slog.Error("save preset", "char_id", sess.CharID, "error", err)
		return
	}
	g.sendPresetList(sess)
}

func (g *gateway) sendPresetList(sess *session.Session) {
	presets, err := g.container.Repo.GetLoadoutPresets(sess.CharID)
	if err != nil {
		slog.Error("load presets", "char_id", sess.CharID, "error", err)
		return
	}
	infos := make([]codec.PresetInfo, len(presets))
	for i, p := range presets {
		infos[i] = codec.PresetInfo{
			ID:         uint32(p.ID),
			Name:       p.Name,
			Slots:      [6]string{p.Slot0, p.Slot1, p.Slot2, p.Slot3, p.Slot4, p.Slot5},
			Commitment: p.Commitment,
		}
	}
	sess.Conn.Send(message.Encode(message.OpPresetList, 0, codec.EncodePresetList(infos)))
}

func validateCommitmentCSV(csv string) bool {
	parts := strings.Split(csv, ",")
	var total int
	for _, part := range parts {
		kv := strings.SplitN(part, ":", 2)
		if len(kv) != 2 {
			return false
		}
		pct, err := strconv.Atoi(kv[1])
		if err != nil || pct < 0 || pct > 100 {
			return false
		}
		total += pct
	}
	return total == 100
}
