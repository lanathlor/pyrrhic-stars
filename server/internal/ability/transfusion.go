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
}

// transfusionHandler validates the initial cast of Transfusion.
// Per-tick drain + AoE heal happens in the OnCommitTick handler.
func transfusionHandler(_ *Engine, ctx *CastContext) CastResult {
	p, ok := ctx.Caster.(*entity.Player)
	if !ok {
		return CastResult{Reason: "not a player"}
	}

	flux := p.Resources["flux"]
	if flux == nil || flux.Current < 3 {
		return CastResult{Reason: "insufficient flux"}
	}

	// Store the channel target for per-tick drain
	p.ChannelTargetID = ctx.TargetPeerID

	return CastResult{OK: true}
}
