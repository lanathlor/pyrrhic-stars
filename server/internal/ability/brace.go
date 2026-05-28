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
	ID:      IDBrace,
	Name:    "Brace",
	Handler: IDBrace,
}

func braceHandler(_ *Engine, ctx *CommitContext) CommitResult {
	p, ok := ctx.Committer.(*entity.Player)
	if !ok {
		return CommitResult{Reason: ReasonInvalidCaster}
	}
	// Must be actively shield blocking
	if !p.HasBuff(IDVgShieldBlock) {
		return CommitResult{Reason: "not blocking"}
	}
	if p.Cooldowns[IDBrace] > 0 {
		return CommitResult{Reason: ReasonCooldown}
	}
	if p.HasBuff(IDBrace) {
		return CommitResult{Reason: "already braced"}
	}

	p.AddBuff(entity.ActiveBuff{
		ID:       IDBrace,
		Type:     IDBrace, // marker buff — actual effect is reducing stamina drain in ApplyDamage
		Value:    braceDrainReduction,
		Duration: braceDuration,
	})

	p.Cooldowns[IDBrace] = braceCooldown

	return CommitResult{OK: true}
}
