package ability

import (
	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
)

var bladeSwirlDef = AbilityDef{
	ID: "blade_swirl", Name: "Blade Swirl",
	Handler: "blade_swirl",
}

// BladeSwirlState tracks the blade swirl multi-tick AoE for a vanguard.
type BladeSwirlState struct {
	Timer float32
	Ticks int
}

func bladeSwirlHandler(eng *Engine, ctx *CastContext) CastResult {
	p, ok := ctx.Caster.(*entity.Player)
	if !ok {
		return CastResult{Reason: "invalid caster"}
	}
	if p.Cooldowns["blade_swirl"] > 0 || p.GCDTimer > 0 {
		return CastResult{Reason: "cooldown"}
	}
	if !p.SpendResource("stamina", 25*p.TenacityEfficiency()) {
		return CastResult{Reason: ReasonInsufficientStamina}
	}

	state := getBladeSwirlState(p)
	state.Timer = 1.5
	state.Ticks = 0

	p.Cooldowns["blade_swirl"] = 10.0
	p.GCDTimer = 1.5
	p.State = entity.PlayerStateAttack

	p.AddBuff(entity.ActiveBuff{
		ID:       "blade_swirl",
		Type:     entity.BuffDamageReduction,
		Value:    0.8,
		Duration: 1.5,
	})

	damage := float32(25.0) * p.CasterDamageMult()
	eng.hitBuf = resolveAoECircle(eng.hitBuf, p.Position, p.ID, ctx.Targets, ctx.Obstacles, 6.0, damage, combat.SourcePlayerAttack)
	for i := range eng.hitBuf {
		if th, ok := eng.hitBuf[i].Target.(entity.Threateable); ok {
			th.AddThreat(p.ID, eng.hitBuf[i].Amount)
		}
	}

	return CastResult{OK: true, Events: eng.hitBuf}
}

func bladeSwirlTick(eng *Engine, p *entity.Player, dt float32, ctx *TickContext) []DamageResult {
	state := getBladeSwirlState(p)
	if state.Timer <= 0 {
		return nil
	}
	state.Timer -= dt

	var events []DamageResult
	expectedTicks := int((1.5 - state.Timer) / 0.5)
	if expectedTicks > state.Ticks && ctx != nil {
		tickDmg := float32(25.0) * p.CasterDamageMult()
		eng.hitBuf = resolveAoECircle(eng.hitBuf[:0], p.Position, p.ID, ctx.Targets, ctx.Obstacles, 6.0, tickDmg, combat.SourcePlayerAttack)
		for i := range eng.hitBuf {
			if th, ok := eng.hitBuf[i].Target.(entity.Threateable); ok {
				th.AddThreat(p.ID, eng.hitBuf[i].Amount)
			}
		}
		events = append(events, eng.hitBuf...)
		state.Ticks = expectedTicks
	}

	if state.Timer <= 0 {
		state.Timer = 0
		state.Ticks = 0
	}

	return events
}

func getBladeSwirlState(p *entity.Player) *BladeSwirlState {
	if s, ok := p.AbilityState["blade_swirl"].(*BladeSwirlState); ok {
		return s
	}
	s := &BladeSwirlState{}
	p.AbilityState["blade_swirl"] = s
	return s
}
