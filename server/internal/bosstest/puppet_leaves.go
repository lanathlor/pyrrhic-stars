package bosstest

import (
	"math"

	"codex-online/server/internal/ability"
	"codex-online/server/internal/bt"
	"codex-online/server/internal/entity"
)

// --- Helper to extract PuppetContext from BT any parameter ---

func pctx(v any) *PuppetContext {
	pc, ok := v.(*PuppetContext)
	if !ok {
		panic("bosstest: BT context is not *PuppetContext")
	}
	return pc
}

// --- Conditions ---

// condHasReacted returns true if enough time has passed since telegraph onset.
func condHasReacted(v any) bool {
	c := pctx(v)
	return c.Puppet.HasReacted()
}

// condHasReactedQuick returns true if ReactionTime (not DodgeGreed) has elapsed.
// Used for AoE and charge branches where immediate escape is critical.
func condHasReactedQuick(v any) bool {
	c := pctx(v)
	return c.Puppet.HasReactedQuick()
}

// condInAoEDanger checks if the puppet is within the boss AoE radius + safety margin.
func condInAoEDanger(v any) bool {
	c := pctx(v)
	if c.Boss.State != entity.EnemyAoETelegraph && c.Boss.State != entity.EnemyAoESlam {
		return false
	}
	abil := c.ActiveAbil
	if abil == nil || abil.Category != ability.CategoryAoE {
		return false
	}
	radius := abil.Hit.Radius + c.Puppet.Params.SafetyMargin
	dist := c.Puppet.DistToBoss(c.Boss)
	return dist < radius
}

// condInChargePath checks if the puppet is in the path of a charge.
func condInChargePath(v any) bool {
	c := pctx(v)
	if c.Boss.State != entity.EnemyChargeTelegraph && c.Boss.State != entity.EnemyCharge {
		return false
	}
	abil := c.ActiveAbil
	if abil == nil || abil.Category != ability.CategoryCharge || abil.Charge == nil {
		return false
	}

	chargeDir := c.Boss.ChargeDirection.Flat()
	if chargeDir.LengthSq() < 0.01 {
		return false
	}
	chargeDir = chargeDir.Normalized()

	// Distance from puppet to the charge line (boss position + chargeDir)
	toPlayer := c.Puppet.Player.Position.Sub(c.Boss.Position).Flat()
	// Project onto charge direction
	proj := toPlayer.Dot(chargeDir)
	if proj < 0 {
		return false // behind the boss, not in path
	}
	// Perpendicular distance to charge line
	projVec := chargeDir.Scale(proj)
	perpVec := toPlayer.Sub(projVec)
	perpDist := perpVec.Length()

	hitRadius := abil.Charge.HitRadius + c.Puppet.Params.SafetyMargin
	return perpDist < hitRadius
}

// condInMeleeDanger checks if the puppet is within melee cone + range.
func condInMeleeDanger(v any) bool {
	c := pctx(v)
	if c.Boss.State != entity.EnemyMeleeTelegraph && c.Boss.State != entity.EnemyMeleeAttack {
		return false
	}
	abil := c.ActiveAbil
	if abil == nil || abil.Category != ability.CategoryMelee {
		return false
	}

	dist := c.Puppet.DistToBoss(c.Boss)
	meleeRange := abil.Hit.Range + c.Puppet.Params.SafetyMargin
	if dist > meleeRange {
		return false
	}

	// Check if in cone
	bossForward := c.Boss.Forward()
	toPlayer := c.Puppet.Player.Position.Sub(c.Boss.Position).Flat().Normalized()
	dot := bossForward.Flat().Normalized().Dot(toPlayer)
	arcDeg := abil.Hit.ArcDegrees
	if arcDeg <= 0 {
		arcDeg = 180
	}
	coneHalf := arcDeg * math.Pi / 180.0 / 2
	return dot > float32(math.Cos(float64(coneHalf)))
}

// condTargetedByRanged checks if the boss is telegraphing a ranged ability,
// this puppet is likely the target, and it hasn't already dodged this telegraph.
func condTargetedByRanged(v any) bool {
	c := pctx(v)
	if c.Boss.State != entity.EnemyRangedTelegraph {
		return false
	}
	abil := c.ActiveAbil
	if abil == nil || abil.Category != ability.CategoryRanged {
		return false
	}
	// Already dodged this telegraph? Don't keep dodging.
	if c.Puppet.dodgedThisTelegraph {
		return false
	}
	// Check if this puppet is the target
	if c.Boss.TargetPlayerID == c.Puppet.Player.ID {
		return true
	}
	return isLikelyTarget(c)
}

// isLikelyTarget checks if this puppet matches the boss's targeting strategy.
func isLikelyTarget(c *PuppetContext) bool {
	if c.ActiveAbil == nil {
		return false
	}
	myDist := c.Puppet.DistToBoss(c.Boss)
	strategy := c.ActiveAbil.TargetStrategy

	for _, other := range c.AllPuppets {
		if other == c.Puppet || !other.Player.Alive {
			continue
		}
		otherDist := other.Player.Position.Flat().DistanceTo(c.Boss.Position.Flat())
		switch strategy {
		case ability.TargetNearest:
			if otherDist < myDist {
				return false // someone else is closer
			}
		case ability.TargetFarthest:
			if otherDist > myDist {
				return false // someone else is farther
			}
		}
	}
	return true
}

// condProjectileIncoming checks if any enemy projectile threatens the puppet.
// Uses closest-approach geometry: calculates perpendicular distance from the puppet
// to the projectile's trajectory line. This catches both straight and spiraling
// projectiles without over-triggering on distant/non-threatening ones.
func condProjectileIncoming(v any) bool {
	c := pctx(v)
	pos := c.Puppet.Player.Position
	for _, p := range c.World.Projectiles {
		if p == nil || !p.Alive || p.OwnerID != 0 {
			continue // skip player-owned or dead projectiles
		}
		toMe := pos.Sub(p.Position).Flat()
		dist := toMe.Length()
		if dist < 0.5 {
			return true // already on top of us
		}
		dirNorm := p.Direction.Flat().Normalized()
		dot := toMe.Normalized().Dot(dirNorm)
		if dot < 0 {
			continue // moving away from us
		}
		// Closest approach: perpendicular distance from puppet to projectile trajectory
		// cross2D = |toMe.X * dirNorm.Z - toMe.Z * dirNorm.X|
		crossMag := toMe.X*dirNorm.Z - toMe.Z*dirNorm.X
		if crossMag < 0 {
			crossMag = -crossMag
		}
		// Threat if trajectory passes within ~1.0 unit of puppet and arrival is soon
		if crossMag < 1.0 {
			timeToImpact := dist / p.Speed
			if timeToImpact < 1.5 {
				return true
			}
		}
	}
	return false
}

// condTooClose checks if puppet is closer than preferred range.
func condTooClose(v any) bool {
	c := pctx(v)
	dist := c.Puppet.DistToBoss(c.Boss)
	return dist < c.Puppet.preferredRange-1.0
}

// condTooFar checks if puppet is farther than preferred range.
func condTooFar(v any) bool {
	c := pctx(v)
	dist := c.Puppet.DistToBoss(c.Boss)
	return dist > c.Puppet.preferredRange+1.0
}

// condOutOfMelee checks if a melee puppet is too far to hit.
// Blocks during AoE/charge (prevents walk-back into danger zone).
// Allows advance during melee/ranged attacks and cooldowns.
func condOutOfMelee(v any) bool {
	c := pctx(v)
	s := c.Boss.State
	if s == entity.EnemyAoETelegraph || s == entity.EnemyAoESlam ||
		s == entity.EnemyChargeTelegraph || s == entity.EnemyCharge {
		return false
	}
	dist := c.Puppet.DistToBoss(c.Boss)
	return dist > c.Puppet.preferredRange+1.0
}

// condBossTelegraphingMelee checks if boss is in melee telegraph state.
func condBossTelegraphingMelee(v any) bool {
	c := pctx(v)
	return c.Boss.State == entity.EnemyMeleeTelegraph
}

// condShouldUseDefensive rolls against DefensiveUse probability.
func condShouldUseDefensive(v any) bool {
	c := pctx(v)
	return c.Puppet.Rng.Float32() < c.Puppet.Params.DefensiveUse
}

// canCastAbility creates a condition that checks if a specific ability is castable.
func canCastAbility(abilityID string) func(any) bool {
	return func(v any) bool {
		c := pctx(v)
		return c.Puppet.CanCast(abilityID)
	}
}

// condCanTransition checks if a blade dancer has a valid transition from current config.
func condCanTransition(v any) bool {
	c := pctx(v)
	p := c.Puppet.Player
	if p.GCDTimer > 0 {
		return false
	}
	// Find any transition spell that matches current config
	config := p.Config
	for action := uint8(30); action < 50; action++ {
		abilID, ok := p.ActionMap[action]
		if !ok {
			continue
		}
		// Transition spells encode origin as (action-30)/4
		originConfig := int(action-30) / 4
		if originConfig == config {
			if cd, ok := p.Cooldowns[abilID]; !ok || cd <= 0 {
				return true
			}
		}
	}
	return false
}

// --- Actions ---

// actionStrafeCharge moves perpendicular to the boss charge direction.
func actionStrafeCharge(v any) bt.Result {
	c := pctx(v)
	chargeDir := c.Boss.ChargeDirection.Flat()
	if chargeDir.LengthSq() < 0.01 {
		chargeDir = c.Boss.Forward().Flat()
	}
	c.Puppet.MovePerpendicular(chargeDir, c.Dt)
	return bt.Success
}

// actionFleeAoE moves away from the boss center.
func actionFleeAoE(v any) bt.Result {
	c := pctx(v)
	c.Puppet.MoveAwayFrom(c.Boss.Position, c.Dt, 1.3)
	return bt.Success
}

// actionSidestepProjectile moves perpendicular to the nearest threatening projectile.
// Matches condProjectileIncoming criteria: closest-approach < 1.0 and approaching.
func actionSidestepProjectile(v any) bt.Result {
	c := pctx(v)
	pos := c.Puppet.Player.Position

	var nearestDir entity.Vec3
	nearestDist := float32(math.MaxFloat32)

	for _, p := range c.World.Projectiles {
		if p == nil || !p.Alive || p.OwnerID != 0 {
			continue
		}
		toMe := pos.Sub(p.Position).Flat()
		dist := toMe.Length()
		if dist >= nearestDist {
			continue
		}
		dirNorm := p.Direction.Flat().Normalized()
		dot := toMe.Normalized().Dot(dirNorm)
		if dot < 0 {
			continue
		}
		crossMag := toMe.X*dirNorm.Z - toMe.Z*dirNorm.X
		if crossMag < 0 {
			crossMag = -crossMag
		}
		if crossMag < 1.0 && dist/p.Speed < 1.5 {
			nearestDist = dist
			nearestDir = dirNorm
		}
	}

	if nearestDist < math.MaxFloat32 {
		c.Puppet.MovePerpendicular(nearestDir, c.Dt)
	}
	return bt.Success
}

// actionAdvance moves toward preferred range from boss.
func actionAdvance(v any) bt.Result {
	c := pctx(v)
	dir := c.Puppet.Player.Position.Sub(c.Boss.Position).Flat()
	if dir.LengthSq() < 0.01 {
		dir = entity.Vec3{Z: 1}
	}
	dir = dir.Normalized()
	target := c.Boss.Position.Add(dir.Scale(c.Puppet.preferredRange))
	c.Puppet.MoveToward(target, c.Dt)
	return bt.Success
}

// actionBackstep moves away from boss.
func actionBackstep(v any) bt.Result {
	c := pctx(v)
	c.Puppet.MoveAwayFrom(c.Boss.Position, c.Dt, 1.2)
	return bt.Success
}

// actionStrafeMeleeCone moves perpendicular to the boss's forward direction
// to exit the melee cone while staying in melee range.
func actionStrafeMeleeCone(v any) bt.Result {
	c := pctx(v)
	bossForward := c.Boss.Forward().Flat()
	if bossForward.LengthSq() < 0.01 {
		bossForward = entity.Vec3{Z: -1}
	}
	c.Puppet.MovePerpendicular(bossForward, c.Dt)
	return bt.Success
}

// actionStrafeRanged moves perpendicular to the boss's forward direction to dodge ranged attacks.
func actionStrafeRanged(v any) bt.Result {
	c := pctx(v)
	bossForward := c.Boss.Forward().Flat()
	if bossForward.LengthSq() < 0.01 {
		bossForward = entity.Vec3{Z: -1}
	}
	c.Puppet.MovePerpendicular(bossForward, c.Dt)
	c.Puppet.dodgedThisTelegraph = true
	return bt.Success
}

// castAbilityAction creates an action that casts a specific ability.
func castAbilityAction(abilityID string) func(any) bt.Result {
	return func(v any) bt.Result {
		c := pctx(v)
		if c.Puppet.TryCast(c, abilityID) {
			return bt.Success
		}
		// GCD/cooldown blocked — not a BT failure, just waiting
		if c.Puppet.Player.GCDTimer > 0 {
			return bt.Running
		}
		return bt.Failure
	}
}

// withCast wraps a dodge/movement action to also attempt casting an ability.
// This models skilled players who maintain DPS while repositioning.
func withCast(moveAction func(any) bt.Result, abilityID string) func(any) bt.Result {
	return func(v any) bt.Result {
		result := moveAction(v)
		c := pctx(v)
		c.Puppet.TryCast(c, abilityID)
		return result
	}
}

// withTransition wraps a dodge/movement action to also attempt a BD transition.
// This models blade dancers who maintain DPS while repositioning.
func withTransition(moveAction func(any) bt.Result) func(any) bt.Result {
	return func(v any) bt.Result {
		result := moveAction(v)
		actionCastBestTransition(v)
		return result
	}
}

// actionKiteAndShoot moves away AND tries to fire (gunner-specific).
func actionKiteAndShoot(v any) bt.Result {
	c := pctx(v)
	c.Puppet.MoveAwayFrom(c.Boss.Position, c.Dt, 1.0)
	c.Puppet.TryCast(c, "fire_shot")
	return bt.Success
}

// actionCastBestTransition picks the optimal blade dancer transition.
func actionCastBestTransition(v any) bt.Result {
	c := pctx(v)
	p := c.Puppet.Player
	config := p.Config
	distToBoss := p.Position.Sub(c.Boss.Position).Flat().Length()

	// Collect available transitions from current config
	type candidate struct {
		abilID   string
		canReach bool // true if spell can hit the boss at current distance
	}
	var candidates []candidate

	for action := uint8(30); action < 50; action++ {
		abilID, ok := p.ActionMap[action]
		if !ok {
			continue
		}
		originConfig := int(action-30) / 4
		if originConfig != config {
			continue
		}
		if cd, exists := p.Cooldowns[abilID]; exists && cd > 0 {
			continue
		}

		canReach := true
		if def := c.World.AbilityEngine.GetAbility(abilID); def != nil {
			// AoECircle is caster-centered — can only hit if boss is within radius
			if def.Hit.Type == ability.HitAoECircle && distToBoss > def.Hit.Radius {
				canReach = false
			}
		}
		candidates = append(candidates, candidate{abilID, canReach})
	}

	if len(candidates) == 0 {
		return bt.Failure
	}

	// High MechanicIQ: prefer spells that can reach the boss
	// Low MechanicIQ: random choice (ignores range)
	var pick candidate
	if c.Puppet.Rng.Float32() < c.Puppet.Params.MechanicIQ {
		// Pick first reachable candidate; fall back to first unreachable if none
		pick = candidates[0]
		for _, cand := range candidates {
			if cand.canReach {
				pick = cand
				break
			}
		}
	} else {
		pick = candidates[c.Puppet.Rng.IntN(len(candidates))]
	}

	if c.Puppet.TryCast(c, pick.abilID) {
		return bt.Success
	}
	if p.GCDTimer > 0 {
		return bt.Running
	}
	return bt.Failure
}

// condShouldReload checks if the gunner magazine is low enough for a tactical reload.
// Tactical reload (1.5s) is much faster than empty auto-reload (2.2s).
func condShouldReload(v any) bool {
	c := pctx(v)
	p := c.Puppet.Player
	state, ok := p.AbilityState["gunner_assault"].(*ability.GunnerAssaultState)
	if !ok || state == nil {
		return false
	}
	return !state.Reloading && !state.MagDumpActive &&
		state.MagCurrent > 0 && state.MagCurrent <= 5 && state.EnhancedLoaded <= 0
}

// actionCastBlock casts vanguard block.
func actionCastBlock(v any) bt.Result {
	c := pctx(v)
	if c.Puppet.TryCast(c, "vg_block") {
		return bt.Success
	}
	return bt.Failure
}
