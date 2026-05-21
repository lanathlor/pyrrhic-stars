package ability

import "codex-online/server/internal/entity"

var mendingSurgeDef = AbilityDef{
	ID:   "mending_surge",
	Name: "Mending Surge",
	Hit: HitDef{
		Type: HitAllyTarget,
	},
	Cooldown: 1.5,
	GCD:      0.8,
	Costs: []ResourceCost{
		{Resource: "flux", Amount: 40},
	},
	BaseHeal:    80,
	HealScaling: "identity",
	Delivery:    uint8(entity.DeliveryDirect),
	Handler:     "mending_surge",
}

func mendingSurgeHandler(_ *Engine, ctx *CastContext) CastResult {
	p, ok := ctx.Caster.(*entity.Player)
	if !ok {
		return CastResult{Reason: "invalid caster"}
	}

	result := resolveHeal(&mendingSurgeDef, p, ctx.Allies, ctx.TargetPeerID)

	// Spend resource (engine validated sufficiency, handler spends)
	p.SpendResource("flux", mendingSurgeDef.Costs[0].Amount)
	p.GCDTimer = mendingSurgeDef.GCD
	if mendingSurgeDef.Cooldown > 0 {
		p.Cooldowns["mending_surge"] = mendingSurgeDef.Cooldown
	}

	if result == nil {
		// Everyone at full HP -- cast succeeds, flux spent, no heal emitted.
		return CastResult{OK: true}
	}

	return CastResult{
		OK:    true,
		Heals: []HealResult{*result},
	}
}
