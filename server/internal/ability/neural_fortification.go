package ability

import "codex-online/server/internal/entity"

var neuralFortificationDef = AbilityDef{
	ID:       "neural_fortification",
	Name:     "Neural Fortification",
	School:   entity.SchoolBioarcanotechnic,
	Hit:      HitDef{Type: HitAllyTarget},
	GCD:      0.8,
	Cooldown: 20.0,
	Costs:    []ResourceCost{{Resource: entity.ResourceFlux, Amount: 40}},
	Delivery: uint8(entity.DeliveryDirect),
	Handler:  "neural_fortification",
}

func neuralFortificationHandler(_ *Engine, ctx *CommitContext) CommitResult {
	p, ok := ctx.Committer.(*entity.Player)
	if !ok {
		return CommitResult{Reason: ReasonInvalidCaster}
	}

	// Spend resource from school pool (engine validated sufficiency, handler spends).
	p.SpendFluxBySchool(neuralFortificationDef.School, neuralFortificationDef.Costs[0].Amount)

	// Resolve target: use specified ally, fall back to self.
	var target *entity.Player
	if t, ok := ctx.Allies[ctx.TargetPeerID]; ok && t.Alive && t.ID != p.ID {
		target = t
	}
	if target == nil {
		target = p
	}

	// Apply +20% damage reduction buff (value 0.8 = incoming damage multiplied by 0.8).
	target.AddBuff(entity.ActiveBuff{
		ID:       "neural_fort_dr",
		Type:     entity.BuffDamageReduction,
		Value:    0.8,
		Duration: 6.0,
	})

	// Apply CC immunity buff.
	target.AddBuff(entity.ActiveBuff{
		ID:       "neural_fort_cc",
		Type:     entity.BuffCCImmunity,
		Value:    1.0,
		Duration: 6.0,
	})

	// Timing.
	p.GCDTimer = neuralFortificationDef.GCD
	if neuralFortificationDef.Cooldown > 0 {
		p.Cooldowns["neural_fortification"] = neuralFortificationDef.Cooldown
	}

	// Confluence: grant stack on ability completion.
	if p.Confluence != nil {
		p.Confluence.OnAbilityComplete()
	}

	return CommitResult{OK: true}
}
