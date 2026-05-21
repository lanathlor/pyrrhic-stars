package ability

import (
	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
)

var lifeSwapDef = AbilityDef{
	ID:   "life_swap",
	Name: "Life Swap",
	Hit: HitDef{
		Type:  HitAllyTarget,
		Range: 15,
	},
	GCD:      0.6,
	Costs:    []ResourceCost{{Resource: "flux", Amount: 5}},
	Handler:  "life_swap",
	Delivery: uint8(entity.DeliveryDirect),
}

func lifeSwapHandler(_ *Engine, ctx *CastContext) CastResult {
	p, ok := ctx.Caster.(*entity.Player)
	if !ok {
		return CastResult{Reason: "not a player"}
	}

	var target *entity.Player
	if ctx.Allies != nil && ctx.TargetPeerID != 0 {
		if t, ok := ctx.Allies[ctx.TargetPeerID]; ok && t.Alive && t.ID != p.ID {
			target = t
		}
	}
	if target == nil {
		return CastResult{Reason: "no valid target"}
	}

	drain := target.Health * 0.20
	if target.Health-drain < 1 {
		drain = target.Health - 1
	}
	if drain <= 0 {
		return CastResult{Reason: "target too low"}
	}

	p.SpendResource("flux", lifeSwapDef.Costs[0].Amount)
	target.Health -= drain

	p.VitalCharge = drain
	p.VitalChargeTimer = 4.0

	p.GCDTimer = lifeSwapDef.GCD

	if p.Confluence != nil {
		p.Confluence.OnSpellComplete()
	}

	return CastResult{
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
