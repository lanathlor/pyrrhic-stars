package ability

import "codex-online/server/internal/entity"

var vitalDrainDef = AbilityDef{
	ID:     IDVitalDrain,
	Name:   "Vital Drain",
	School: entity.SchoolBiometabolic,
	Hit: HitDef{
		Type:        HitNearestN,
		Range:       20,
		TargetCount: 1,
	},
	CommitTime:       0.5,
	ExecuteTime:      0.1,
	GCD:              0.5,
	BaseDamage:       8,
	Delivery:         uint8(entity.DeliveryBeam),
	Handler:          IDVitalDrain,
	CancelConditions: uint8(CancelOnDamage),

	Sustain:           true,
	SustainCostPerSec: 3,
	SustainEffect:     8,
	SustainInterval:   0.5,
	SustainScaling:    0.05,
	SustainCooldown:   8.0,
}

// vitalDrainHandler validates the initial commit of Vital Drain.
// Sustain damage + healing happens in applySustainTick.
func vitalDrainHandler(_ *Engine, ctx *CommitContext) CommitResult {
	p, ok := ctx.Committer.(*entity.Player)
	if !ok {
		return CommitResult{Reason: ReasonNotAPlayer}
	}

	if p.FluxCommit != nil && len(p.FluxCommit.Pools) > 0 {
		pool := p.FluxCommit.GetPool(vitalDrainDef.School)
		if pool == nil || pool.Current < vitalDrainDef.SustainCostPerSec*p.AffinityCostMult(vitalDrainDef.School) {
			return CommitResult{Reason: "insufficient " + vitalDrainDef.School + " flux"}
		}
	} else {
		flux := p.Resources[entity.ResourceFlux]
		if flux == nil || flux.Current < vitalDrainDef.SustainCostPerSec {
			return CommitResult{Reason: ReasonInsufficientFlux}
		}
	}

	return CommitResult{OK: true}
}
