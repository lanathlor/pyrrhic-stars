package ability

import "codex-online/server/internal/entity"

var mendingBeamDef = AbilityDef{
	ID:   "mending_beam",
	Name: "Mending Beam",
	Hit: HitDef{
		Type:  HitAllyTarget,
		Range: 20,
	},
	CommitTime:  3.0,
	ExecuteTime: 0.1,
	GCD:         0.5,
	Costs: []ResourceCost{
		{Resource: "flux", Amount: 10}, // cost per second during channel
	},
	BaseHeal:         15, // heal per tick during channel
	HealScaling:      "identity",
	Handler:          "mending_beam",
	OnCommitTick:     "mending_beam",
	CancelConditions: uint8(CancelOnMove) | uint8(CancelOnDamage),
}

// mendingBeamHandler validates the initial cast of Mending Beam.
// Resource spending per tick happens in the OnCommitTick handler.
func mendingBeamHandler(_ *Engine, ctx *CastContext) CastResult {
	p, ok := ctx.Caster.(*entity.Player)
	if !ok {
		return CastResult{Reason: "not a player"}
	}

	flux := p.Resources["flux"]
	if flux == nil || flux.Current < 10 {
		return CastResult{Reason: "insufficient flux"}
	}

	return CastResult{OK: true}
}
