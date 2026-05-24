package ability

import (
	"codex-online/server/internal/entity"
)

// Brace — plant feet and fortify stance while blocking.
const (
	braceDuration       float32 = 3.5
	braceDrainReduction float32 = 0.2 // 20% of normal stamina drain while braced
	braceCooldown       float32 = 18
)

// BraceDrainReduction is exported so entity.ApplyDamage can check brace.
const BraceDrainReduction = braceDrainReduction

var braceDef = AbilityDef{
	ID:      "brace",
	Name:    "Brace",
	Handler: "brace",
}

func braceHandler(_ *Engine, ctx *CommitContext) CommitResult {
	p, ok := ctx.Committer.(*entity.Player)
	if !ok {
		return CommitResult{Reason: "invalid caster"}
	}
	// Must be actively shield blocking
	if !p.HasBuff("vg_shield_block") {
		return CommitResult{Reason: "not blocking"}
	}
	if p.Cooldowns["brace"] > 0 {
		return CommitResult{Reason: "cooldown"}
	}
	if p.HasBuff("brace") {
		return CommitResult{Reason: "already braced"}
	}

	p.AddBuff(entity.ActiveBuff{
		ID:       "brace",
		Type:     "brace", // marker buff — actual effect is reducing stamina drain in ApplyDamage
		Value:    braceDrainReduction,
		Duration: braceDuration,
	})

	p.Cooldowns["brace"] = braceCooldown

	return CommitResult{OK: true}
}
