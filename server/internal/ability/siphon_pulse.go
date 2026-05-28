package ability

import (
	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
)

var siphonPulseDef = AbilityDef{
	ID:     IDSiphonPulse,
	Name:   "Siphon Pulse",
	School: entity.SchoolBiometabolic,
	Hit: HitDef{
		Type:        HitNearestN,
		Range:       20,
		TargetCount: 1,
	},
	GCD:          0.8,
	BaseDamage:   12,
	Delivery:     uint8(entity.DeliveryDirect),
	Handler:      IDSiphonPulse,
	DamageSource: combat.SourcePlayerAttack,
}

func siphonPulseHandler(eng *Engine, ctx *CommitContext) CommitResult {
	p, ok := ctx.Committer.(*entity.Player)
	if !ok {
		return CommitResult{Reason: ReasonNotAPlayer}
	}

	// Resolve hit on nearest enemy
	eng.hitBuf = eng.hitBuf[:0]
	eng.hitBuf = resolveHit(eng.hitBuf, &siphonPulseDef, p, ctx.Targets, ctx.Obstacles, combat.SourcePlayerAttack)

	if len(eng.hitBuf) == 0 {
		return CommitResult{Reason: "no target"}
	}

	p.GCDTimer = siphonPulseDef.GCD

	// Confluence: grant stack on ability completion.
	if p.Confluence != nil {
		p.Confluence.OnAbilityComplete()
	}

	// Total damage dealt
	var totalDmg float32
	for _, hit := range eng.hitBuf {
		totalDmg += hit.Amount
	}

	result := CommitResult{OK: true, Events: eng.hitBuf}

	// Heal lowest HP ally for 50% of damage dealt
	healAmount := totalDmg * 0.5
	if healAmount > 0 && ctx.Allies != nil {
		var target *entity.Player
		var lowestHP float32 = 999999
		for _, ally := range ctx.Allies {
			if ally.Alive && ally.Health < ally.MaxHealth && ally.Health < lowestHP {
				lowestHP = ally.Health
				target = ally
			}
		}
		if target == nil {
			target = p // heal self if all full
		}

		before := target.Health
		target.Health += healAmount
		if target.Health > target.MaxHealth {
			target.Health = target.MaxHealth
		}
		actual := target.Health - before
		overheal := healAmount - actual

		if actual > 0 || overheal > 0 {
			result.Heals = []HealResult{{
				TargetID:   target.ID,
				SourceID:   p.ID,
				Amount:     actual,
				Overheal:   overheal,
				HitPos:     target.Position.Add(entity.Vec3{Y: 1.0}),
				SourceType: combat.SourcePlayerHeal,
			}}
		}
	}

	return result
}
