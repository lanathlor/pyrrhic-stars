package main

import (
	"log/slog"

	"codex-online/server/internal/codec"
	"codex-online/server/internal/item"
	"codex-online/server/internal/message"
	"codex-online/server/internal/session"
)

// handleInventoryMessage processes equip/unequip opcodes and sends back
// the updated inventory state.
func (g *gateway) handleInventoryMessage(sess *session.Session, opcode uint16, payload []byte) {
	if sess.CharID == 0 {
		return
	}

	switch opcode {
	case message.OpEquipItem:
		itemID, slotID, ok := codec.DecodeEquipItem(payload)
		if !ok {
			return
		}
		if err := g.inventory.Equip(sess.CharID, uint(itemID), item.SlotID(slotID)); err != nil {
			slog.Warn("equip failed", "char_id", sess.CharID, "item_id", itemID, "error", err)
			return
		}

	case message.OpUnequipItem:
		slotID, ok := codec.DecodeUnequipItem(payload)
		if !ok {
			return
		}
		if err := g.inventory.Unequip(sess.CharID, item.SlotID(slotID)); err != nil {
			slog.Warn("unequip failed", "char_id", sess.CharID, "slot_id", slotID, "error", err)
			return
		}

	default:
		slog.Warn("unknown inventory opcode", "opcode", opcode)
		return
	}

	// Reload and send updated state.
	zi := g.getZone(sess.ZoneID)
	if zi == nil {
		return
	}
	g.loadAndApplyGear(sess, zi)
}
