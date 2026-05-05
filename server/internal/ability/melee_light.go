package ability

import (
	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
)

var meleeLightDef = AbilityDef{
	ID: "melee_light", Name: "Light Attack",
	Hit:     HitDef{Type: HitMeleeArc, Range: 6, ArcDegrees: 120},
	Handler: "melee_light_vg",
}

// ComboState tracks melee combo step.
type ComboState struct {
	Step int
}

func meleeLightVGHandler(eng *Engine, ctx *CastContext) CastResult {
	p, ok := ctx.Caster.(*entity.Player)
	if !ok {
		return CastResult{Reason: "invalid caster"}
	}
	if p.Cooldowns["melee_light"] > 0 {
		return CastResult{Reason: "cooldown"}
	}
	if !p.SpendResource("stamina", 10) {
		return CastResult{Reason: ReasonInsufficientStamina}
	}

	combo := getComboState(p)
	var damage float32
	switch combo.Step {
	case 1:
		damage = 35.0
	case 2:
		damage = 55.0
	default:
		damage = 30.0
	}
	damage *= p.DamageMult()

	def := eng.abilities["melee_light"]
	eng.hitBuf = resolveMeleeArc(eng.hitBuf, p, ctx.Targets, ctx.Obstacles, def.Hit, damage, combat.SourcePlayerAttack)

	combo.Step = (combo.Step + 1) % 3
	p.Cooldowns["melee_light"] = 0.55
	p.State = entity.PlayerStateAttack

	return CastResult{OK: true, Events: eng.hitBuf}
}

func getComboState(p *entity.Player) *ComboState {
	if s, ok := p.AbilityState["combo"].(*ComboState); ok {
		return s
	}
	s := &ComboState{}
	p.AbilityState["combo"] = s
	return s
}
