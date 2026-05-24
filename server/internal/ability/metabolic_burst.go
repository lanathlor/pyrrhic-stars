package ability

import (
	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
)

var metabolicBurstDef = AbilityDef{
	ID:     "metabolic_burst",
	Name:   "Metabolic Burst",
	School: "biometabolic",
	Hit: HitDef{
		Type:        HitNearestN,
		Range:       20,
		TargetCount: 1,
	},
	GCD:      0.8,
	Cooldown: 12.0,
	Costs: []ResourceCost{
		{Resource: entity.ResourceFlux, Amount: 40},
	},
	BaseDamage:   25,
	Delivery:     uint8(entity.DeliveryDirect),
	Handler:      "metabolic_burst",
	DamageSource: combat.SourcePlayerAttack,
}

func metabolicBurstHandler(eng *Engine, ctx *CommitContext) CommitResult {
	p, ok := ctx.Committer.(*entity.Player)
	if !ok {
		return CommitResult{Reason: "not a player"}
	}

	// Resolve hit on nearest enemy
	eng.hitBuf = eng.hitBuf[:0]
	eng.hitBuf = resolveHit(eng.hitBuf, &metabolicBurstDef, p, ctx.Targets, ctx.Obstacles, combat.SourcePlayerAttack)

	if len(eng.hitBuf) == 0 {
		return CommitResult{Reason: "no target"}
	}

	// Spend resource from school pool (engine validated sufficiency, handler spends)
	p.SpendFluxBySchool(metabolicBurstDef.School, metabolicBurstDef.Costs[0].Amount)
	p.GCDTimer = metabolicBurstDef.GCD
	p.Cooldowns["metabolic_burst"] = metabolicBurstDef.Cooldown

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

	// AoE heal: heal all alive allies within 8m of the hit enemy for 50% of damage dealt
	healAmount := totalDmg * 0.5
	if healAmount > 0 && ctx.Allies != nil && len(eng.hitBuf) > 0 {
		enemyPos := eng.hitBuf[0].Target.TargetPos()

		for _, ally := range ctx.Allies {
			if !ally.Alive {
				continue
			}
			dx := enemyPos.X - ally.Position.X
			dz := enemyPos.Z - ally.Position.Z
			if dx*dx+dz*dz > 64 { // 8m radius = 64 sq
				continue
			}

			before := ally.Health
			ally.Health += healAmount
			if ally.Health > ally.MaxHealth {
				ally.Health = ally.MaxHealth
			}
			actual := ally.Health - before
			overheal := healAmount - actual

			if actual > 0 || overheal > 0 {
				result.Heals = append(result.Heals, HealResult{
					TargetID:   ally.ID,
					SourceID:   p.ID,
					Amount:     actual,
					Overheal:   overheal,
					HitPos:     ally.Position.Add(entity.Vec3{Y: 1.0}),
					SourceType: combat.SourcePlayerHeal,
				})
			}
		}
	}

	return result
}
