package ability

import (
	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
)

// Bull Rush — charge forward with shield, push enemies.
const (
	bullRushDamage     float32 = 60
	bullRushRange      float32 = 5
	bullRushStamina    float32 = 20
	bullRushCooldown   float32 = 8
	bullRushGCD        float32 = 0.8
	bullRushThreatMult float32 = 2.0
)

var bullRushDef = AbilityDef{
	ID:       "bull_rush",
	Name:     "Bull Rush",
	Handler:  "bull_rush",
	Category: CategoryCharge,
	Charge: &ChargeDef{
		Speed:       12,
		Damage:      bullRushDamage,
		MaxDistance:  12,
		HitRadius:   2.5,
		StopOnWall:  true,
	},
}

func bullRushHandler(eng *Engine, ctx *CastContext) CastResult {
	p, ok := ctx.Caster.(*entity.Player)
	if !ok {
		return CastResult{Reason: "invalid caster"}
	}
	if p.Cooldowns["bull_rush"] > 0 {
		return CastResult{Reason: "cooldown"}
	}
	cost := bullRushStamina * p.TenacityEfficiency()
	if !p.SpendResource("stamina", cost) {
		return CastResult{Reason: ReasonInsufficientStamina}
	}

	// Bull Rush drops guard
	if p.HasBuff("vg_shield_block") {
		EndVgShieldBlock(p)
	}

	// AoE circle at caster position (charge endpoint resolved by movement system;
	// for ability engine we hit targets in a circle around the caster)
	damage := bullRushDamage * p.CasterDamageMult()
	eng.hitBuf = resolveAoECircle(eng.hitBuf[:0], p.Position, p.ID, ctx.Targets, ctx.Obstacles, bullRushRange, damage, combat.SourcePlayerAttack)

	// Root debuff (brief knockback effect)
	for i := range eng.hitBuf {
		if enemy, ok := eng.hitBuf[i].Target.(*entity.Enemy); ok && enemy.Alive {
			enemy.AddDebuff(entity.ActiveDebuff{
				ID:       "bull_rush_knockback",
				Type:     entity.DebuffRoot,
				Value:    1.0,
				Duration: 0.3,
				SourceID: p.ID,
			})
		}
		if th, ok := eng.hitBuf[i].Target.(entity.Threateable); ok {
			th.AddThreat(p.ID, eng.hitBuf[i].Amount*bullRushThreatMult)
		}
	}

	p.Cooldowns["bull_rush"] = bullRushCooldown
	p.GCDTimer = bullRushGCD
	p.State = entity.PlayerStateAttack

	return CastResult{OK: true, Events: eng.hitBuf}
}
