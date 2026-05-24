package ability

import "codex-online/server/internal/entity"

var mendingSurgeDef = AbilityDef{
	ID:     "mending_surge",
	Name:   "Mending Surge",
	School: "bioarcanotechnic",
	Hit: HitDef{
		Type: HitAllyTarget,
	},
	Cooldown: 1.5,
	GCD:      0.8,
	Costs: []ResourceCost{
		{Resource: "flux", Amount: 25},
	},
	BaseHeal:    35,
	HealScaling: "identity",
	Delivery:    uint8(entity.DeliveryDirect),
	Handler:     "mending_surge",
}

func mendingSurgeHandler(_ *Engine, ctx *CommitContext) CommitResult {
	p, ok := ctx.Committer.(*entity.Player)
	if !ok {
		return CommitResult{Reason: "invalid caster"}
	}

	result := resolveHeal(&mendingSurgeDef, p, ctx.Allies, ctx.TargetPeerID)

	// Spend resource from school pool (engine validated sufficiency, handler spends)
	p.SpendFluxBySchool(mendingSurgeDef.School, mendingSurgeDef.Costs[0].Amount)
	p.GCDTimer = mendingSurgeDef.GCD
	if mendingSurgeDef.Cooldown > 0 {
		p.Cooldowns["mending_surge"] = mendingSurgeDef.Cooldown
	}

	// Confluence: grant stack on ability completion.
	if p.Confluence != nil {
		p.Confluence.OnAbilityComplete()
	}

	if result == nil {
		// Everyone at full HP -- commit succeeds, flux spent, no heal emitted.
		return CommitResult{OK: true}
	}

	return CommitResult{
		OK:    true,
		Heals: []HealResult{*result},
	}
}
