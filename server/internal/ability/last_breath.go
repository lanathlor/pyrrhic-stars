package ability

import "codex-online/server/internal/entity"

var lastBreathDef = AbilityDef{
	ID:       IDLastBreath,
	Name:     "Last Breath",
	School:   entity.SchoolBiometabolic,
	Hit:      HitDef{Type: HitAllyTarget},
	GCD:      0.8,
	Cooldown: 60.0,
	Costs:    []ResourceCost{{Resource: entity.ResourceFlux, Amount: 45}},
	Delivery: uint8(entity.DeliveryDirect),
	Handler:  IDLastBreath,
}

func lastBreathHandler(_ *Engine, ctx *CommitContext) CommitResult {
	p, ok := ctx.Committer.(*entity.Player)
	if !ok {
		return CommitResult{Reason: ReasonNotAPlayer}
	}

	// Validate flux
	if p.FluxCommit != nil && len(p.FluxCommit.Pools) > 0 {
		pool := p.FluxCommit.GetPool(lastBreathDef.School)
		if pool == nil || pool.Current < lastBreathDef.Costs[0].Amount*p.AffinityCostMult(lastBreathDef.School) {
			return CommitResult{Reason: "insufficient " + lastBreathDef.School + " flux"}
		}
	} else {
		flux := p.Resources[entity.ResourceFlux]
		if flux == nil || flux.Current < lastBreathDef.Costs[0].Amount {
			return CommitResult{Reason: ReasonInsufficientFlux}
		}
	}

	// Find target ally (or self)
	var target *entity.Player
	if ctx.TargetPeerID != 0 && ctx.Allies != nil {
		if t, ok := ctx.Allies[ctx.TargetPeerID]; ok && t.Alive {
			target = t
		}
	}
	if target == nil {
		target = p
	}

	p.SpendFluxBySchool(lastBreathDef.School, lastBreathDef.Costs[0].Amount)
	p.GCDTimer = lastBreathDef.GCD
	p.Cooldowns[IDLastBreath] = lastBreathDef.Cooldown

	if p.Confluence != nil {
		p.Confluence.OnAbilityComplete()
	}

	// Apply death prevention buff
	target.AddBuff(entity.ActiveBuff{
		ID:       IDLastBreath,
		Type:     entity.BuffDeathPrevention,
		Duration: 4.0,
	})
	target.LastBreathCasterID = p.ID
	target.LastBreathPrevented = 0 // reset accumulator

	return CommitResult{OK: true}
}
