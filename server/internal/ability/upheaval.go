package ability

import (
	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
)

// Upheaval — upward slash + overhead slam. Cone widens with Onslaught tier.
var meleeHeavyDef = AbilityDef{
	ID: "upheaval", Name: "Upheaval",
	Hit:     HitDef{Type: HitAoECone, Range: 7, ArcDegrees: 60},
	Handler: "upheaval_vg",
}

// Upheaval tier tuning.
type upheavalTuning struct {
	arc     float32
	damage  float32
	lockout float32
	dot     bool // maximum tier applies DoT
}

var upheavalTiers = [3]upheavalTuning{
	{arc: 60, damage: 55, lockout: 0.8, dot: false},  // standard
	{arc: 120, damage: 70, lockout: 1.0, dot: false}, // empowered
	{arc: 120, damage: 70, lockout: 1.0, dot: true},  // maximum
}

const upheavalStaminaCost float32 = 20

func upheavalHandler(eng *Engine, ctx *CommitContext) CommitResult {
	p, ok := ctx.Committer.(*entity.Player)
	if !ok {
		return CommitResult{Reason: "invalid caster"}
	}
	if p.Cooldowns["upheaval"] > 0 {
		return CommitResult{Reason: "cooldown"}
	}
	if !p.SpendResource("stamina", upheavalStaminaCost*p.TenacityEfficiency()) {
		return CommitResult{Reason: ReasonInsufficientStamina}
	}

	ons := getOnslaughtState(p)
	tier := ons.Tier()
	tuning := upheavalTiers[tier]

	damage := tuning.damage * p.CommitterDamageMult() * ons.DamageMult(p.GearStats.Mastery)

	hit := HitDef{Type: HitAoECone, Range: 7, ArcDegrees: tuning.arc}
	eng.hitBuf = resolveAoECone(eng.hitBuf, p, ctx.Targets, ctx.Obstacles, hit, damage, combat.SourcePlayerAttack)

	// Onslaught: increment by number of enemies hit
	if len(eng.hitBuf) > 0 {
		ons.Increment(len(eng.hitBuf))
	}

	// Maximum tier: apply DoT to hit targets
	if tuning.dot {
		for _, evt := range eng.hitBuf {
			p.DoTs = append(p.DoTs, entity.ActiveDoT{
				EnemyID:    evt.TargetID,
				SourcePeer: p.ID,
				AbilityID:  "upheaval",
				Damage:     10.0 * p.CommitterDamageMult(),
				Remaining:  3.0,
				Interval:   1.0,
				TickTimer:  1.0,
			})
		}
	}

	// Threat
	for i := range eng.hitBuf {
		if th, ok := eng.hitBuf[i].Target.(entity.Threateable); ok {
			th.AddThreat(p.ID, eng.hitBuf[i].Amount)
		}
	}

	p.Cooldowns["upheaval"] = tuning.lockout
	p.GCDTimer = tuning.lockout
	p.State = entity.PlayerStateAttack

	return CommitResult{OK: true, Events: eng.hitBuf}
}
