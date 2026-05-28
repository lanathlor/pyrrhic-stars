package ability

import (
	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
)

// Shield Bash — quick strike without dropping guard.
const (
	shieldBashDamage         float32 = 40
	shieldBashArc            float32 = 90
	shieldBashRange          float32 = 4
	shieldBashStamina        float32 = 8
	shieldBashGCD            float32 = 0.4
	shieldBashBlockedStamina float32 = 14  // higher cost while blocking
	shieldBashBlockedGCD     float32 = 0.7 // slower while blocking
	shieldBashThreatMult     float32 = 1.5
)

var shieldBashDef = AbilityDef{
	ID: "shield_bash", Name: "Shield Bash",
	Handler:  "shield_bash",
	Category: CategoryMelee,
}

func shieldBashHandler(eng *Engine, ctx *CommitContext) CommitResult {
	p, ok := ctx.Committer.(*entity.Player)
	if !ok {
		return CommitResult{Reason: ReasonInvalidCaster}
	}
	if p.GCDTimer > 0 {
		return CommitResult{Reason: ReasonGCD}
	}

	// Higher cost and slower GCD while blocking
	staminaCost := shieldBashStamina
	gcd := shieldBashGCD
	if p.HasBuff(IDVgShieldBlock) {
		staminaCost = shieldBashBlockedStamina
		gcd = shieldBashBlockedGCD
	}

	cost := staminaCost * p.TenacityEfficiency()
	if !p.SpendResource("stamina", cost) {
		return CommitResult{Reason: ReasonInsufficientStamina}
	}

	damage := shieldBashDamage * p.CommitterDamageMult()
	hit := HitDef{Type: HitMeleeArc, Range: shieldBashRange, ArcDegrees: shieldBashArc}
	eng.hitBuf = resolveMeleeArc(eng.hitBuf[:0], p, ctx.Targets, ctx.Obstacles, hit, damage, combat.SourcePlayerAttack)

	// Stagger debuff (slow) on hit targets
	for i := range eng.hitBuf {
		if enemy, ok := eng.hitBuf[i].Target.(*entity.Enemy); ok && enemy.Alive {
			enemy.AddDebuff(entity.ActiveDebuff{
				ID:       "shield_bash_stagger",
				Type:     entity.DebuffSlow,
				Value:    0.5,
				Duration: 0.5,
				SourceID: p.ID,
			})
		}
		if th, ok := eng.hitBuf[i].Target.(entity.Threateable); ok {
			th.AddThreat(p.ID, eng.hitBuf[i].Amount*shieldBashThreatMult)
		}
	}

	// Small Devotion generation per hit
	if len(eng.hitBuf) > 0 {
		dev := getDevotionState(p)
		dev.AddCharges(float32(len(eng.hitBuf))*8.0, p.GearStats.Mastery)
	}

	p.GCDTimer = gcd
	p.State = entity.PlayerStateAttack

	return CommitResult{OK: true, Events: eng.hitBuf}
}
