package ability

import (
	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
)

// Execution — windup + devastating overhead chop. Narrow impact, highest single-hit.
var groundSlamDef = AbilityDef{
	ID: "execution", Name: "Execution",
	Hit:     HitDef{Type: HitAoECone, Range: 7, ArcDegrees: 30},
	Handler: "execution_vg",
}

// Execution tier tuning.
type executionTuning struct {
	damage       float32
	lockout      float32
	shockwaveArc float32 // 0 = no shockwave
	shockwaveRng float32
}

var executionTiers = [3]executionTuning{
	{damage: 90, lockout: 1.2, shockwaveArc: 0, shockwaveRng: 0},   // standard
	{damage: 120, lockout: 1.4, shockwaveArc: 60, shockwaveRng: 3}, // empowered
	{damage: 150, lockout: 1.6, shockwaveArc: 90, shockwaveRng: 5}, // maximum
}

const (
	executionCooldown    float32 = 8.0
	executionStaminaCost float32 = 30
	executionPrimaryArc  float32 = 30
	executionPrimaryRng  float32 = 7
)

func executionVGHandler(eng *Engine, ctx *CommitContext) CommitResult {
	p, ok := ctx.Committer.(*entity.Player)
	if !ok {
		return CommitResult{Reason: ReasonInvalidCaster}
	}
	if p.Cooldowns["execution"] > 0 || p.GCDTimer > 0 {
		return CommitResult{Reason: ReasonCooldown}
	}
	if !p.SpendResource("stamina", executionStaminaCost*p.TenacityEfficiency()) {
		return CommitResult{Reason: ReasonInsufficientStamina}
	}

	ons := getOnslaughtState(p)
	tier := ons.Tier()
	tuning := executionTiers[tier]

	damage := tuning.damage * p.CommitterDamageMult() * ons.DamageMult(p.GearStats.Mastery)

	// Primary hit: narrow cone
	hit := HitDef{Type: HitAoECone, Range: executionPrimaryRng, ArcDegrees: executionPrimaryArc}
	eng.hitBuf = resolveAoECone(eng.hitBuf, p, ctx.Targets, ctx.Obstacles, hit, damage, combat.SourcePlayerAttack)

	// Empowered/Maximum: secondary shockwave (wider, shorter range, half damage).
	// It only catches EXTRA enemies the primary cone missed — no double-dipping
	// a second 0.5x hit on whoever the primary already struck.
	if tuning.shockwaveArc > 0 {
		primaryHit := make(map[uint16]bool, len(eng.hitBuf))
		for i := range eng.hitBuf {
			primaryHit[eng.hitBuf[i].TargetID] = true
		}
		shockHit := HitDef{Type: HitAoECone, Range: tuning.shockwaveRng, ArcDegrees: tuning.shockwaveArc}
		shockDmg := damage * 0.5
		eng.hitBuf = resolveAoEConeExcluding(eng.hitBuf, p, ctx.Targets, ctx.Obstacles, shockHit, shockDmg, combat.SourcePlayerAttack, primaryHit)
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

	p.Cooldowns["execution"] = executionCooldown
	p.GCDTimer = tuning.lockout
	p.State = entity.PlayerStateAttack

	return CommitResult{OK: true, Events: eng.hitBuf}
}
