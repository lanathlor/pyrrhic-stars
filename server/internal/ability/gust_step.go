package ability

import "codex-online/server/internal/entity"

var gustStepDef = AbilityDef{
	ID:       "gust_step",
	Name:     "Gust Step",
	School:   "aerokinetic",
	Hit:      HitDef{Type: HitNone},
	GCD:      0.8,
	Cooldown: 10.0,
	Costs:    []ResourceCost{{Resource: entity.ResourceFlux, Amount: 10}},
	Delivery: uint8(entity.DeliveryDirect),
	Handler:  "gust_step",
}

func gustStepHandler(_ *Engine, ctx *CommitContext) CommitResult {
	p, ok := ctx.Committer.(*entity.Player)
	if !ok {
		return CommitResult{Reason: "invalid caster"}
	}

	// Spend resource from school pool (engine validated sufficiency, handler spends).
	p.SpendFluxBySchool(gustStepDef.School, gustStepDef.Costs[0].Amount)

	// Grant invincibility frames for the displacement.
	p.Invincible = true
	p.InvincibleTimer = 0.15

	// Timing.
	p.GCDTimer = gustStepDef.GCD
	if gustStepDef.Cooldown > 0 {
		p.Cooldowns["gust_step"] = gustStepDef.Cooldown
	}

	// Confluence: grant stack on ability completion.
	if p.Confluence != nil {
		p.Confluence.OnAbilityComplete()
	}

	return CommitResult{OK: true}
}
