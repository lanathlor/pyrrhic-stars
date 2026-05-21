package ability

import "codex-online/server/internal/entity"

// CheckHarmony evaluates whether a heal triggers the Harmony bonus.
// Harmony fires when the delivery method differs from the last heal on that
// target. The bonus is a flat 20 HP scaled by the healer's Mastery stat.
// Returns 0 if the healer has no Harmony state (non-Harmonist) or if the
// delivery method matches the previous heal.
func CheckHarmony(healer *entity.Player, targetID uint16, method entity.DeliveryMethod) float32 {
	if healer.Harmony == nil {
		return 0
	}

	last, exists := healer.Harmony.LastDelivery[targetID]
	healer.Harmony.LastDelivery[targetID] = method

	if !exists || last == method {
		return 0
	}

	return 20.0 * (1.0 + healer.GearStats.Mastery/100.0)
}
