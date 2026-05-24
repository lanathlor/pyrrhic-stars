package ability

import "codex-online/server/internal/entity"

var regenProtocolDef = AbilityDef{
	ID:       "regen_protocol",
	Name:     "Regeneration Protocol",
	School:   "bioarcanotechnic",
	Hit:      HitDef{Type: HitAllyTarget},
	GCD:      0.8,
	Cooldown: 18.0,
	Costs:    []ResourceCost{{Resource: entity.ResourceFlux, Amount: 20}},
	Delivery: uint8(entity.DeliveryDirect),
	Handler:  "regen_protocol",
}

func regenProtocolHandler(_ *Engine, ctx *CommitContext) CommitResult {
	p, ok := ctx.Committer.(*entity.Player)
	if !ok {
		return CommitResult{Reason: "not a player"}
	}

	// Validate flux
	if p.FluxCommit != nil && len(p.FluxCommit.Pools) > 0 {
		pool := p.FluxCommit.GetPool(regenProtocolDef.School)
		if pool == nil || pool.Current < regenProtocolDef.Costs[0].Amount*p.AffinityCostMult(regenProtocolDef.School) {
			return CommitResult{Reason: "insufficient " + regenProtocolDef.School + " flux"}
		}
	} else {
		flux := p.Resources[entity.ResourceFlux]
		if flux == nil || flux.Current < regenProtocolDef.Costs[0].Amount {
			return CommitResult{Reason: "insufficient flux"}
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

	p.SpendFluxBySchool(regenProtocolDef.School, regenProtocolDef.Costs[0].Amount)
	p.GCDTimer = regenProtocolDef.GCD
	p.Cooldowns["regen_protocol"] = regenProtocolDef.Cooldown

	if p.Confluence != nil {
		p.Confluence.OnAbilityComplete()
	}

	// Apply HoT: 12s duration, 5 HP/tick, 1s interval, burst at <30% HP
	healPerTick := float32(5.0) * (1.0 + p.GearStats.Identity/100.0)
	if p.Confluence != nil {
		healPerTick *= p.Confluence.AbilityPowerMult()
	}

	target.HoTs = append(target.HoTs, entity.ActiveHoT{
		ID:             "regen_protocol",
		SourcePeer:     p.ID,
		HealPerTick:    healPerTick,
		Remaining:      12.0,
		Interval:       1.0,
		TickTimer:      1.0,
		BurstThreshold: 0.3,
	})

	return CommitResult{OK: true}
}
