package ability

import (
	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
)

var frostWardDef = AbilityDef{
	ID:       "frost_ward",
	Name:     "Frost Ward",
	School:   "frost",
	Hit:      HitDef{Type: HitAllyTarget},
	GCD:      0.8,
	Cooldown: 12.0,
	Costs:    []ResourceCost{{Resource: "flux", Amount: 20}},
	Delivery: uint8(entity.DeliveryDirect),
	Handler:  "frost_ward",
}

const (
	frostWardShieldAmount float32 = 30
	frostWardShieldCap    float32 = 50
	frostWardBuffDuration float32 = 6.0
	frostWardExplosionDmg float32 = 8.0
	frostWardExplosionRad float32 = 5.0 // meters
	frostbiteDuration     float32 = 4.0
	frostbiteSlow         float32 = 0.3 // 30% slow
)

func frostWardHandler(_ *Engine, ctx *CommitContext) CommitResult {
	p, ok := ctx.Committer.(*entity.Player)
	if !ok {
		return CommitResult{Reason: "invalid caster"}
	}

	// Spend resource from school pool (engine validated sufficiency, handler spends).
	p.SpendFluxBySchool(frostWardDef.School, frostWardDef.Costs[0].Amount)

	// Resolve target: use specified ally, fall back to self.
	var target *entity.Player
	if t, ok := ctx.Allies[ctx.TargetPeerID]; ok && t.Alive && t.ID != p.ID {
		target = t
	}
	if target == nil {
		target = p
	}

	// Create shield resource on target if absent.
	if target.Resources["shield"] == nil {
		target.Resources["shield"] = &entity.Resource{Current: 0, Max: frostWardShieldCap}
	}

	// Grant shield, capped at maximum.
	shield := target.Resources["shield"]
	shield.Current += frostWardShieldAmount
	if shield.Current > frostWardShieldCap {
		shield.Current = frostWardShieldCap
	}

	// Add buff to track ward duration.
	target.AddBuff(entity.ActiveBuff{
		ID:       "frost_ward",
		Type:     entity.BuffDamageReduction,
		Value:    1.0, // no extra DR, tracks expiry for explosion
		Duration: frostWardBuffDuration,
	})

	// Mark ward active for tick detection.
	target.AbilityState["frost_ward_active"] = true

	// Timing.
	p.GCDTimer = frostWardDef.GCD
	if frostWardDef.Cooldown > 0 {
		p.Cooldowns["frost_ward"] = frostWardDef.Cooldown
	}

	// Confluence: grant stack on ability completion.
	if p.Confluence != nil {
		p.Confluence.OnAbilityComplete()
	}

	return CommitResult{OK: true}
}

// frostWardTick checks each player for frost ward expiry or shield break and
// triggers the AoE frost explosion when the ward ends.
func frostWardTick(eng *Engine, p *entity.Player, dt float32, ctx *TickContext) []DamageResult {
	wasActive, _ := p.AbilityState["frost_ward_active"].(bool)
	if !wasActive {
		return nil
	}

	buff := p.GetBuff("frost_ward")

	// Detect explosion trigger:
	//   1. Buff was removed externally (dispel, shield break callback, etc.)
	//   2. Buff will expire this tick (duration will go to 0 in the buff tick loop
	//      that runs AFTER this tick handler in engine.TickPlayer)
	shouldExplode := false
	if buff == nil {
		// Buff already removed externally.
		shouldExplode = true
	} else if buff.Duration > 0 && buff.Duration-dt <= 0 {
		// Buff expires this tick.
		shouldExplode = true
	}

	if !shouldExplode {
		return nil
	}

	// Clear ward state.
	p.AbilityState["frost_ward_active"] = false

	// Clear remaining shield from the ward.
	if shield := p.Resources["shield"]; shield != nil {
		shield.Current = 0
	}

	// No targets available -- nothing to explode on.
	if ctx == nil {
		return nil
	}

	// AoE explosion: damage + frostbite to enemies within radius.
	var results []DamageResult
	radSq := frostWardExplosionRad * frostWardExplosionRad

	for _, t := range ctx.Targets {
		if t == nil || !t.TargetAlive() {
			continue
		}
		dx := p.Position.X - t.TargetPos().X
		dz := p.Position.Z - t.TargetPos().Z
		if dx*dx+dz*dz > radSq {
			continue
		}

		dealt := t.TargetApplyDamage(frostWardExplosionDmg)
		if dealt <= 0 {
			continue
		}

		// Apply frostbite slow to enemies.
		if enemy, ok := t.(*entity.Enemy); ok {
			enemy.AddDebuff(entity.ActiveDebuff{
				ID:       "frostbite",
				Type:     entity.DebuffSlow,
				Value:    frostbiteSlow,
				Duration: frostbiteDuration,
				SourceID: p.ID,
			})
		}

		results = append(results, DamageResult{
			TargetID:   t.TargetID(),
			SourceID:   p.ID,
			Amount:     dealt,
			HitPos:     t.TargetPos().Add(entity.Vec3{Y: 1.0}),
			SourceType: combat.SourcePlayerAttack,
			AbilityID:  "frost_ward",
			Target:     t,
		})
	}

	return results
}
