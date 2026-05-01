package ability

import "codex-online/server/internal/entity"

var meleeHeavyDef = AbilityDef{
	ID:         "melee_heavy", Name: "Heavy Attack",
	Hit:        HitDef{Type: HitMeleeArc, Range: 6, ArcDegrees: 120},
	BaseDamage: 45,
	Handler:    "melee_heavy_vg",
}

func meleeHeavyVGHandler(eng *Engine, ctx *CastContext) CastResult {
	p := ctx.Player
	if p.Cooldowns["melee_heavy"] > 0 {
		return CastResult{Reason: "cooldown"}
	}
	if !p.SpendResource("stamina", 20) {
		return CastResult{Reason: ReasonInsufficientStamina}
	}

	def := eng.abilities["melee_heavy"]
	damage := def.BaseDamage * p.DamageMult()
	eng.hitBuf = resolveMeleeArc(eng.hitBuf, p, ctx.Enemies, ctx.Obstacles, def.Hit, damage)

	p.Cooldowns["melee_heavy"] = 0.8
	p.State = entity.PlayerStateAttack

	return CastResult{OK: true, Events: eng.hitBuf}
}
