package main

import (
	"log/slog"

	"codex-online/server/internal/codec"
	"codex-online/server/internal/item"
	"codex-online/server/internal/merchant"
	"codex-online/server/internal/message"
	"codex-online/server/internal/session"
)

// handleMerchantMessage dispatches merchant-related opcodes.
func (g *gateway) handleMerchantMessage(sess *session.Session, opcode uint16, payload []byte) {
	if sess.CharID == 0 {
		return
	}
	switch opcode {
	case message.OpMerchantInteract:
		g.handleMerchantOpen(sess, payload)
	case message.OpMerchantBuy:
		g.handleMerchantBuyItem(sess, payload)
	default:
		slog.Warn("unknown merchant opcode", "opcode", opcode)
	}
}

// handleMerchantOpen sends the full merchant state to the client.
func (g *gateway) handleMerchantOpen(sess *session.Session, payload []byte) {
	_, ok := codec.DecodeMerchantInteract(payload)
	if !ok {
		return
	}
	state, err := g.merchant.GetState(sess.CharID)
	if err != nil {
		slog.Error("merchant get state", "char_id", sess.CharID, "error", err)
		return
	}
	tiers := buildMerchantTiers(state)
	msg := codec.EncodeMerchantState(state.ScripBalance, state.BestScore, state.Season, uint16(state.MaxScore), tiers)
	sess.Conn.Send(message.Encode(message.OpMerchantState, 0, msg))
}

// handleMerchantBuyItem processes a purchase request and sends the result.
func (g *gateway) handleMerchantBuyItem(sess *session.Session, payload []byte) {
	tier, defID, ok := codec.DecodeMerchantBuy(payload)
	if !ok {
		return
	}
	itemID, newBalance, err := g.merchant.BuyItem(sess.CharID, int(tier), defID)
	if err != nil {
		slog.Warn("merchant buy failed", "char_id", sess.CharID, "tier", tier, "def", defID, "error", err)
		sess.Conn.Send(message.Encode(message.OpMerchantBuyResult, 0,
			codec.EncodeMerchantBuyError(err.Error())))
		return
	}
	sess.Conn.Send(message.Encode(message.OpMerchantBuyResult, 0,
		codec.EncodeMerchantBuySuccess(newBalance, uint32(itemID))))
	// Send updated inventory so the client sees the new item immediately.
	zi := g.getZone(sess.ZoneID)
	if zi != nil {
		g.loadAndApplyGear(sess, zi)
	}
}

// buildMerchantTiers builds codec tier info from the player's merchant state.
func buildMerchantTiers(state *merchant.PlayerState) []codec.MerchantTierInfo {
	tiers := make([]codec.MerchantTierInfo, len(merchant.Tiers))
	for i, td := range merchant.Tiers {
		unlocked := merchant.IsTierUnlocked(i, state.BestScore, state.MaxScore)
		var items []codec.MerchantItemInfo
		for _, defID := range merchant.MerchantItems {
			def := item.DefRegistry[defID]
			if def == nil {
				continue
			}
			tmpItem := &item.Item{DefID: defID, ILvl: td.ILvl, Slot: def.Slot}
			statLines := item.ComputeStatsForItem(tmpItem)
			codecLines := make([]codec.InventoryStatLine, len(statLines))
			for j, sl := range statLines {
				codecLines[j] = codec.InventoryStatLine{Stat: uint8(sl.Stat), Value: sl.Value}
			}
			items = append(items, codec.MerchantItemInfo{
				DefID:     defID,
				Name:      def.Name,
				SlotID:    uint8(def.Slot),
				StatLines: codecLines,
			})
		}
		tiers[i] = codec.MerchantTierInfo{
			ILvl:     td.ILvl,
			Unlocked: unlocked,
			Price:    td.Price,
			Items:    items,
		}
	}
	return tiers
}
