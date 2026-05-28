package ability

import (
	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
)

var lifeSwapDef = AbilityDef{
	ID:     IDLifeSwap,
	Name:   "Life Swap",
	School: entity.SchoolBiometabolic,
	Hit: HitDef{
		Type:  HitAllyTarget,
		Range: 15,
	},
	GCD:      0.6,
	Costs:    []ResourceCost{{Resource: entity.ResourceFlux, Amount: 5}},
	Handler:  IDLifeSwap,
	Delivery: uint8(entity.DeliveryDirect),
}

func lifeSwapHandler(_ *Engine, ctx *CommitContext) CommitResult {
	p, ok := ctx.Committer.(*entity.Player)
	if !ok {
		return CommitResult{Reason: ReasonNotAPlayer}
	}

	var target *entity.Player
	if ctx.Allies != nil && ctx.TargetPeerID != 0 {
		if t, ok := ctx.Allies[ctx.TargetPeerID]; ok && t.Alive && t.ID != p.ID {
			target = t
		}
	}
	if target == nil {
		return CommitResult{Reason: ReasonNoValidTarget}
	}

	drain := target.Health * 0.20
	if target.Health-drain < 1 {
		drain = target.Health - 1
	}
	if drain <= 0 {
		return CommitResult{Reason: "target too low"}
	}

	p.SpendFluxBySchool(lifeSwapDef.School, lifeSwapDef.Costs[0].Amount)
	target.Health -= drain

	p.VitalCharge = drain
	p.VitalChargeTimer = 4.0

	p.GCDTimer = lifeSwapDef.GCD

	if p.Confluence != nil {
		p.Confluence.OnAbilityComplete()
	}

	return CommitResult{
		OK: true,
		Heals: []HealResult{{
			TargetID:   target.ID,
			SourceID:   p.ID,
			Amount:     -drain,
			HitPos:     target.Position.Add(entity.Vec3{Y: 1.0}),
			SourceType: combat.SourcePlayerHeal,
		}},
	}
}
