package ability

import (
	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
)

// Shield Bash — quick strike without dropping guard.
const (
	shieldBashDamage     float32 = 15
	shieldBashArc        float32 = 90
	shieldBashRange      float32 = 4
	shieldBashStamina    float32 = 8
	shieldBashGCD        float32 = 0.4
	shieldBashThreatMult float32 = 1.5
)

var shieldBashDef = AbilityDef{
	ID: "shield_bash", Name: "Shield Bash",
	Handler:  "shield_bash",
	Category: CategoryMelee,
}

func shieldBashHandler(eng *Engine, ctx *CastContext) CastResult {
	p, ok := ctx.Caster.(*entity.Player)
	if !ok {
		return CastResult{Reason: "invalid caster"}
	}
	if p.GCDTimer > 0 {
		return CastResult{Reason: "gcd"}
	}
	cost := shieldBashStamina * p.TenacityEfficiency()
	if !p.SpendResource("stamina", cost) {
		return CastResult{Reason: ReasonInsufficientStamina}
	}

	damage := shieldBashDamage * p.CasterDamageMult()
	hit := HitDef{Type: HitMeleeArc, Range: shieldBashRange, ArcDegrees: shieldBashArc}
	eng.hitBuf = resolveMeleeArc(eng.hitBuf[:0], p, ctx.Targets, ctx.Obstacles, hit, damage, combat.SourcePlayerAttack)

	// Stagger debuff (slow) on hit targets
	for i := range eng.hitBuf {
		if enemy, ok := eng.hitBuf[i].Target.(*entity.Enemy); ok && enemy.Alive {
			enemy.AddDebuff(entity.ActiveDebuff{
				ID:       "shield_bash_stagger",
				Type:     entity.DebuffSlow,
				Value:    0.5,
				Duration: 0.5,
				SourceID: p.ID,
			})
		}
		if th, ok := eng.hitBuf[i].Target.(entity.Threateable); ok {
			th.AddThreat(p.ID, eng.hitBuf[i].Amount*shieldBashThreatMult)
		}
	}

	// Small Devotion generation per hit
	if len(eng.hitBuf) > 0 {
		dev := getDevotionState(p)
		dev.AddCharges(float32(len(eng.hitBuf))*5.0, p.GearStats.Mastery)
	}

	p.GCDTimer = shieldBashGCD
	p.State = entity.PlayerStateAttack

	return CastResult{OK: true, Events: eng.hitBuf}
}
