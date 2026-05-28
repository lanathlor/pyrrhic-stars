package ability

import "codex-online/server/internal/entity"

var vitalCircuitDef = AbilityDef{
	ID:       "vital_circuit",
	Name:     "Vital Circuit",
	School:   entity.SchoolBiometabolic,
	Hit:      HitDef{Type: HitAllyTarget, Range: 15},
	GCD:      0.8,
	Cooldown: 15.0,
	Costs:    []ResourceCost{{Resource: entity.ResourceFlux, Amount: 8}},
	Delivery: uint8(entity.DeliveryDirect),
	Handler:  "vital_circuit",
}

func vitalCircuitHandler(_ *Engine, ctx *CommitContext) CommitResult {
	p, ok := ctx.Committer.(*entity.Player)
	if !ok {
		return CommitResult{Reason: ReasonNotAPlayer}
	}

	// Validate flux
	if p.FluxCommit != nil && len(p.FluxCommit.Pools) > 0 {
		pool := p.FluxCommit.GetPool(vitalCircuitDef.School)
		if pool == nil || pool.Current < vitalCircuitDef.Costs[0].Amount*p.AffinityCostMult(vitalCircuitDef.School) {
			return CommitResult{Reason: "insufficient " + vitalCircuitDef.School + " flux"}
		}
	} else {
		flux := p.Resources[entity.ResourceFlux]
		if flux == nil || flux.Current < vitalCircuitDef.Costs[0].Amount {
			return CommitResult{Reason: ReasonInsufficientFlux}
		}
	}

	// Must have a valid ally target (can't link to self)
	var target *entity.Player
	if ctx.TargetPeerID != 0 && ctx.Allies != nil {
		if t, ok := ctx.Allies[ctx.TargetPeerID]; ok && t.Alive && t.ID != p.ID {
			target = t
		}
	}
	if target == nil {
		return CommitResult{Reason: ReasonNoValidTarget}
	}

	p.SpendFluxBySchool(vitalCircuitDef.School, vitalCircuitDef.Costs[0].Amount)
	p.GCDTimer = vitalCircuitDef.GCD
	p.Cooldowns["vital_circuit"] = vitalCircuitDef.Cooldown

	if p.Confluence != nil {
		p.Confluence.OnAbilityComplete()
	}

	// Spawn damage link between caster and target
	if ctx.SpawnLink != nil {
		ctx.SpawnLink(&entity.DamageLink{
			SourcePeer: p.ID,
			PeerA:      p.ID,
			PeerB:      target.ID,
			Duration:   8.0,
		})
	}

	return CommitResult{OK: true}
}
