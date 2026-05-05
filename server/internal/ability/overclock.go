package ability

import "codex-online/server/internal/entity"

var overclockDef = AbilityDef{
	ID: "overclock", Name: "Overclock",
	Handler: "overclock",
}

func overclockHandler(_ *Engine, ctx *CastContext) CastResult {
	p, ok := ctx.Caster.(*entity.Player)
	if !ok {
		return CastResult{Reason: "invalid caster"}
	}
	if p.HasBuff("overclock") {
		return CastResult{Reason: "already active"}
	}
	if cd := p.Cooldowns["overclock"]; cd > 0 {
		return CastResult{Reason: "cooldown"}
	}
	p.AddBuff(entity.ActiveBuff{
		ID:       "overclock",
		Type:     entity.BuffCooldownMult,
		Value:    0.556, // 0.10/0.18 ~ 0.556
		Duration: 7.0,
	})
	p.Cooldowns["overclock"] = 15.0
	return CastResult{OK: true}
}
