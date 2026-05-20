package ability

import (
	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
)

// Cleave — fast horizontal sweep. Arc widens with Onslaught tier.
var meleeLightDef = AbilityDef{
	ID: "cleave", Name: "Cleave",
	Hit:     HitDef{Type: HitMeleeArc, Range: 6, ArcDegrees: 120},
	Handler: "cleave_vg",
}

// Cleave tier tuning.
type cleaveTuning struct {
	arc      float32
	damage   float32
	cooldown float32
}

var cleaveTiers = [3]cleaveTuning{
	{arc: 120, damage: 30, cooldown: 0.45}, // standard
	{arc: 200, damage: 40, cooldown: 0.50}, // empowered
	{arc: 360, damage: 50, cooldown: 0.55}, // maximum
}

const cleaveStaminaCost float32 = 10

func cleaveHandler(eng *Engine, ctx *CastContext) CastResult {
	p, ok := ctx.Caster.(*entity.Player)
	if !ok {
		return CastResult{Reason: "invalid caster"}
	}
	if p.Cooldowns["cleave"] > 0 {
		return CastResult{Reason: "cooldown"}
	}
	if !p.SpendResource("stamina", cleaveStaminaCost*p.TenacityEfficiency()) {
		return CastResult{Reason: ReasonInsufficientStamina}
	}

	ons := getOnslaughtState(p)
	tier := ons.Tier()
	tuning := cleaveTiers[tier]

	damage := tuning.damage * p.CasterDamageMult() * ons.DamageMult(p.GearStats.Mastery)

	// Maximum tier (360°) uses AoE circle instead of arc
	if tier == TierMaximum {
		eng.hitBuf = resolveAoECircle(eng.hitBuf, p.Position, p.ID, ctx.Targets, ctx.Obstacles, 6.0, damage, combat.SourcePlayerAttack)
	} else {
		hit := HitDef{Type: HitMeleeArc, Range: 6, ArcDegrees: tuning.arc}
		eng.hitBuf = resolveMeleeArc(eng.hitBuf, p, ctx.Targets, ctx.Obstacles, hit, damage, combat.SourcePlayerAttack)
	}

	// Onslaught: increment by number of enemies hit
	if len(eng.hitBuf) > 0 {
		ons.Increment(len(eng.hitBuf))
	}

	// Threat
	for i := range eng.hitBuf {
		if th, ok := eng.hitBuf[i].Target.(entity.Threateable); ok {
			th.AddThreat(p.ID, eng.hitBuf[i].Amount)
		}
	}

	p.Cooldowns["cleave"] = tuning.cooldown
	p.State = entity.PlayerStateAttack

	return CastResult{OK: true, Events: eng.hitBuf}
}
