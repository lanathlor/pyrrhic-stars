package ability

import "codex-online/server/internal/entity"

var vgBlockDef = AbilityDef{
	ID:      "vg_block", Name: "Block",
	Handler: "vg_block",
}

func vgBlockHandler(_ *Engine, ctx *CastContext) CastResult {
	p := ctx.Caster.(*entity.Player)
	if p.HasBuff("vg_block") || p.HasBuff("vg_parry") {
		return CastResult{Reason: "already blocking"}
	}
	p.AddBuff(entity.ActiveBuff{
		ID:       "vg_parry",
		Type:     entity.BuffDamageReduction,
		Value:    0.0,
		Duration: 0.15,
	})
	p.AddBuff(entity.ActiveBuff{
		ID:       "vg_block",
		Type:     entity.BuffDamageReduction,
		Value:    0.3,
		Duration: 1.5,
	})
	p.State = entity.PlayerStateBlock
	return CastResult{OK: true}
}
