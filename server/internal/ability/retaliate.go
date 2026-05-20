package ability

import (
	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
)

// Retaliate — massive shield slam consuming all Devotion charges.
const (
	retaliateBaseDamage    float32 = 30
	retaliatePerCharge     float32 = 2.0
	retaliateArc           float32 = 180
	retaliateRange         float32 = 6
	retaliateGCD           float32 = 1.5
	retaliateThreatMult    float32 = 3.0
)

var retaliateDef = AbilityDef{
	ID:       "retaliate",
	Name:     "Retaliate",
	Handler:  "retaliate",
	Category: CategoryMelee,
}

func retaliateHandler(eng *Engine, ctx *CastContext) CastResult {
	p, ok := ctx.Caster.(*entity.Player)
	if !ok {
		return CastResult{Reason: "invalid caster"}
	}
	if p.GCDTimer > 0 {
		return CastResult{Reason: "gcd"}
	}

	// Retaliate drops guard
	if p.HasBuff("vg_shield_block") {
		EndVgShieldBlock(p)
	}

	// Consume all Devotion charges
	dev := getDevotionState(p)
	charges := dev.ConsumeAll()

	// Damage = base + charges * per_charge * (1 + mastery/100) * CasterDamageMult
	damage := (retaliateBaseDamage + charges*retaliatePerCharge*(1.0+p.GearStats.Mastery/100.0)) * p.CasterDamageMult()

	hit := HitDef{Type: HitAoECone, Range: retaliateRange, ArcDegrees: retaliateArc}
	eng.hitBuf = resolveAoECone(eng.hitBuf[:0], p, ctx.Targets, ctx.Obstacles, hit, damage, combat.SourcePlayerAttack)

	for i := range eng.hitBuf {
		if th, ok := eng.hitBuf[i].Target.(entity.Threateable); ok {
			th.AddThreat(p.ID, eng.hitBuf[i].Amount*retaliateThreatMult)
		}
	}

	p.GCDTimer = retaliateGCD
	p.State = entity.PlayerStateAttack

	return CastResult{OK: true, Events: eng.hitBuf}
}
