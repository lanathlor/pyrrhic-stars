package ability

import "codex-online/server/internal/entity"

var transfusionDef = AbilityDef{
	ID:     "transfusion",
	Name:   "Transfusion",
	School: "biometabolic",
	Hit: HitDef{
		Type:  HitAllyTarget,
		Range: 15,
	},
	CommitTime:       4.0,
	ExecuteTime:      0.1,
	GCD:              0.5,
	Costs:            []ResourceCost{{Resource: "flux", Amount: 3}},
	Handler:          "transfusion",
	OnCommitTick:     "transfusion",
	CancelConditions: uint8(CancelOnMove) | uint8(CancelOnDamage),
	Delivery:         uint8(entity.DeliveryBeam),

	Sustain:           true,
	SustainCostPerSec: 3,
	SustainEffect:     8,
	SustainInterval:   0.5,
	SustainScaling:    0.05,
	SustainCooldown:   12.0,
	SustainHandler:    "transfusion",
}

// transfusionHandler validates the initial commit of Transfusion.
// Per-tick drain + AoE heal happens in the OnCommitTick handler.
func transfusionHandler(_ *Engine, ctx *CommitContext) CommitResult {
	p, ok := ctx.Committer.(*entity.Player)
	if !ok {
		return CommitResult{Reason: "not a player"}
	}

	if p.FluxCommit != nil && len(p.FluxCommit.Pools) > 0 {
		pool := p.FluxCommit.GetPool(transfusionDef.School)
		if pool == nil || pool.Current < transfusionDef.Costs[0].Amount*p.AffinityCostMult(transfusionDef.School) {
			return CommitResult{Reason: "insufficient " + transfusionDef.School + " flux"}
		}
	} else {
		flux := p.Resources["flux"]
		if flux == nil || flux.Current < transfusionDef.Costs[0].Amount {
			return CommitResult{Reason: "insufficient flux"}
		}
	}

	// Store the channel target for per-tick drain
	p.ChannelTargetID = ctx.TargetPeerID

	return CommitResult{OK: true}
}
