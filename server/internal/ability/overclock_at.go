package ability

import "codex-online/server/internal/entity"

var overclockATDef = AbilityDef{
	ID:       IDOverclockAT,
	Name:     "Overclock",
	School:   entity.SchoolBioarcanotechnic,
	Hit:      HitDef{Type: HitAllyTarget},
	GCD:      0.8,
	Cooldown: 15.0,
	Costs:    []ResourceCost{{Resource: entity.ResourceFlux, Amount: 30}},
	Delivery: uint8(entity.DeliveryDirect),
	Handler:  IDOverclockAT,
}

func overclockATHandler(_ *Engine, ctx *CommitContext) CommitResult {
	p, ok := ctx.Committer.(*entity.Player)
	if !ok {
		return CommitResult{Reason: ReasonInvalidCaster}
	}

	// Validate flux (engine already checked, but handler owns the spend).
	if p.FluxCommit != nil && len(p.FluxCommit.Pools) > 0 {
		pool := p.FluxCommit.GetPool(overclockATDef.School)
		if pool == nil || pool.Current < overclockATDef.Costs[0].Amount*p.AffinityCostMult(overclockATDef.School) {
			return CommitResult{Reason: "insufficient " + overclockATDef.School + " flux"}
		}
	}

	// Find target ally, fall back to self if invalid.
	target := p
	if ally, ok := ctx.Allies[ctx.TargetPeerID]; ok && ally != nil {
		target = ally
	}

	// Spend flux.
	p.SpendFluxBySchool(overclockATDef.School, overclockATDef.Costs[0].Amount)

	// Set GCD and cooldown.
	p.GCDTimer = overclockATDef.GCD
	p.Cooldowns[IDOverclockAT] = overclockATDef.Cooldown

	// Confluence: grant stack on ability completion.
	if p.Confluence != nil {
		p.Confluence.OnAbilityComplete()
	}

	// Apply buffs to target (not caster).
	target.AddBuff(entity.ActiveBuff{
		ID:       "overclock_at_speed",
		Type:     entity.BuffAttackSpeed,
		Value:    1.15,
		Duration: 6.0,
	})
	target.AddBuff(entity.ActiveBuff{
		ID:       "overclock_at_move",
		Type:     entity.BuffMoveSpeed,
		Value:    1.10,
		Duration: 6.0,
	})

	return CommitResult{OK: true}
}
