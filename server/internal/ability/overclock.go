package ability

import "codex-online/server/internal/entity"

var overclockDef = AbilityDef{
	ID: IDOverclock, Name: "Overclock",
	Handler: IDOverclock,
}

func overclockHandler(_ *Engine, ctx *CommitContext) CommitResult {
	p, ok := ctx.Committer.(*entity.Player)
	if !ok {
		return CommitResult{Reason: ReasonInvalidCaster}
	}
	if p.HasBuff(IDOverclock) {
		return CommitResult{Reason: "already active"}
	}
	if cd := p.Cooldowns[IDOverclock]; cd > 0 {
		return CommitResult{Reason: ReasonCooldown}
	}
	p.AddBuff(entity.ActiveBuff{
		ID:       IDOverclock,
		Type:     entity.BuffCooldownMult,
		Value:    0.556, // 0.10/0.18 ~ 0.556
		Duration: 7.0,
	})
	p.Cooldowns[IDOverclock] = 15.0
	return CommitResult{OK: true}
}
