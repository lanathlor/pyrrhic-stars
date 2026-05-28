package ability

import "codex-online/server/internal/entity"

var mendingBeamDef = AbilityDef{
	ID:     IDMendingBeam,
	Name:   "Mending Beam",
	School: entity.SchoolBioarcanotechnic,
	Hit: HitDef{
		Type:  HitAllyTarget,
		Range: 20,
	},
	CommitTime:  3.0,
	ExecuteTime: 0.1,
	GCD:         0.5,
	Costs: []ResourceCost{
		{Resource: entity.ResourceFlux, Amount: 8}, // cost per second during channel
	},
	BaseHeal:         12, // heal per tick during channel
	HealScaling:      "identity",
	Delivery:         uint8(entity.DeliveryBeam),
	Handler:          IDMendingBeam,
	OnCommitTick:     IDMendingBeam,
	CancelConditions: uint8(CancelOnMove) | uint8(CancelOnDamage),

	Sustain:           true,
	SustainCostPerSec: 8,
	SustainEffect:     12,
	SustainInterval:   0.5,
	SustainScaling:    0.05,
	SustainCooldown:   10.0,
}

// mendingBeamHandler validates the initial commit of Mending Beam.
// Resource spending per tick happens in the OnCommitTick handler.
func mendingBeamHandler(_ *Engine, ctx *CommitContext) CommitResult {
	p, ok := ctx.Committer.(*entity.Player)
	if !ok {
		return CommitResult{Reason: ReasonNotAPlayer}
	}

	if p.FluxCommit != nil && len(p.FluxCommit.Pools) > 0 {
		pool := p.FluxCommit.GetPool(mendingBeamDef.School)
		if pool == nil || pool.Current < mendingBeamDef.Costs[0].Amount*p.AffinityCostMult(mendingBeamDef.School) {
			return CommitResult{Reason: "insufficient " + mendingBeamDef.School + " flux"}
		}
	} else {
		flux := p.Resources[entity.ResourceFlux]
		if flux == nil || flux.Current < mendingBeamDef.Costs[0].Amount {
			return CommitResult{Reason: ReasonInsufficientFlux}
		}
	}

	return CommitResult{OK: true}
}
