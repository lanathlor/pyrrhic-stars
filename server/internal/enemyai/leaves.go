package enemyai

import (
	"math"
	"slices"

	"codex-online/server/internal/ability"
	"codex-online/server/internal/bt"
	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
)

func ctx(v any) *EntityContext { return v.(*EntityContext) } //nolint:revive // unchecked assertion intentional: hot path, panic on wrong type is a programming error

// --- Conditions ---

func condHasTarget(v any) bool {
	c := ctx(v)
	result := c.TargetPlayer() != nil
	c.logCond("has_target", result)
	return result
}

func condTargetInMeleeRange(v any) bool {
	c := ctx(v)
	p := c.TargetPlayer()
	if p == nil {
		if c.Logger != nil {
			c.logCond("target_in_melee", false, "reason", "no_target")
		}
		return false
	}
	dist := c.Enemy.Position.Flat().DistanceTo(p.Position.Flat())
	result := dist <= c.Def.LongestMeleeRange()
	if c.Logger != nil {
		c.logCond("target_in_melee", result, "dist", dist, "range", c.Def.LongestMeleeRange())
	}
	return result
}

func condHasLoS(v any) bool {
	c := ctx(v)
	p := c.TargetPlayer()
	if p == nil {
		return false
	}
	return c.HasLineOfSight(p.Position)
}

func condIsDead(v any) bool {
	c := ctx(v)
	result := !c.Enemy.Alive
	c.logCond("is_dead", result)
	return result
}

func condPhaseTransitioning(v any) bool {
	c := ctx(v)
	result := c.Enemy.State == entity.EnemyPhaseTransition
	if c.Logger != nil {
		c.logCond("phase_transitioning", result, "state", c.Enemy.State)
	}
	return result
}

func condInLeashRange(v any) bool {
	c := ctx(v)
	e := c.Enemy
	if e.LeashRadius <= 0 {
		return true
	}
	distSq := e.Position.Flat().DistanceToSq(e.LeashOrigin.Flat())
	result := distSq <= e.LeashRadius*e.LeashRadius
	if c.Logger != nil {
		c.logCond("in_leash_range", result, "dist_sq", distSq, "radius", e.LeashRadius)
	}
	return result
}

// condPlayerNearby returns a condition factory parameterized by radius.
func condPlayerNearby(radius float32) func(any) bool {
	rSq := radius * radius
	return func(v any) bool {
		c := ctx(v)
		for _, p := range c.Players {
			if !p.Alive {
				continue
			}
			if c.Enemy.Position.Flat().DistanceToSq(p.Position.Flat()) <= rSq {
				if c.Logger != nil {
					c.logCond("player_nearby", true, "radius", radius, "player", p.ID)
				}
				return true
			}
		}
		if c.Logger != nil {
			c.logCond("player_nearby", false, "radius", radius)
		}
		return false
	}
}

// condPhaseEq returns a condition factory parameterized by phase number.
func condPhaseEq(phase int) func(any) bool {
	return func(v any) bool {
		c := ctx(v)
		result := c.Enemy.Phase == phase
		if c.Logger != nil {
			c.logCond("phase_eq", result, "want", phase, "got", c.Enemy.Phase)
		}
		return result
	}
}

// --- Actions ---

func actionStop(v any) bt.Result {
	c := ctx(v)
	c.Enemy.Velocity = entity.Vec3{}
	c.logAction("stop", bt.Success)
	return bt.Success
}

func actionAggroNearest(v any) bt.Result {
	c := ctx(v)
	p := c.NearestPlayer()
	if p == nil {
		return bt.Failure
	}
	c.Enemy.TargetPlayerID = p.ID
	c.Enemy.State = entity.EnemyChase
	if c.Logger != nil {
		c.logAction("aggro_nearest", bt.Success, "target", p.ID)
	}
	return bt.Success
}

func actionWaitTransition(v any) bt.Result {
	c := ctx(v)
	if c.Enemy.StateTimer > 0 {
		c.Enemy.Velocity = entity.Vec3{}
		return bt.Running
	}
	c.Enemy.State = entity.EnemyChase
	return bt.Success
}

func actionLeashReset(v any) bt.Result {
	c := ctx(v)
	e := c.Enemy
	e.Position = e.LeashOrigin
	e.Health = e.MaxHealth
	e.State = entity.EnemyPatrol
	e.Velocity = entity.Vec3{}
	e.ThreatTable = make(map[uint16]float32)
	c.BB.Delete("last_attack")
	c.logAction("leash_reset", bt.Success)
	return bt.Success
}

func actionPatrol(v any) bt.Result {
	c := ctx(v)
	e := c.Enemy

	// Check aggro: if any alive player within AggroRadius, switch to Chase
	for _, p := range c.Players {
		if !p.Alive {
			continue
		}
		distSq := e.Position.Flat().DistanceToSq(p.Position.Flat())
		if distSq <= e.AggroRadius*e.AggroRadius {
			e.TargetPlayerID = p.ID
			e.State = entity.EnemyChase
			return bt.Success
		}
	}

	// Walk toward current patrol target
	var target entity.Vec3
	if e.PatrolTarget == 0 {
		target = e.PatrolA
	} else {
		target = e.PatrolB
	}
	toTarget := target.Sub(e.Position).Flat()
	dist := toTarget.Length()
	if dist < 0.5 {
		e.PatrolTarget = 1 - e.PatrolTarget
		e.Velocity = entity.Vec3{}
		return bt.Running
	}
	dir := toTarget.Normalized()
	spd := c.Def.MoveSpeed * 0.5
	e.Velocity = entity.Vec3{X: dir.X * spd, Z: dir.Z * spd}
	e.RotationY = float32(math.Atan2(float64(-dir.X), float64(-dir.Z)))
	return bt.Running
}

// actionChase moves toward the target with obstacle avoidance and backpedaling.
// Returns Running while moving, Success when in melee range (so the Selector
// re-evaluates the attack branch above).
func actionChase(v any) bt.Result {
	c := ctx(v)
	e := c.Enemy
	def := c.Def

	if !e.Alive || e.State == entity.EnemyPhaseTransition {
		e.Velocity = entity.Vec3{}
		return bt.Failure
	}

	target := c.NearestPlayer()
	if target == nil {
		e.Velocity = entity.Vec3{}
		return bt.Failure
	}
	e.TargetPlayerID = target.ID

	toTarget := target.Position.Sub(e.Position).Flat()
	distance := toTarget.Length()

	if distance > 0.1 {
		dir := toTarget.Normalized()
		e.RotationY = float32(math.Atan2(float64(-dir.X), float64(-dir.Z)))
	}

	spd := def.CurrentMoveSpeed(e.Phase)
	preferred := def.PreferredRange

	if preferred <= 0 {
		meleeRange := def.LongestMeleeRange()
		if distance <= meleeRange*0.8 {
			e.Velocity = entity.Vec3{}
			return bt.Success
		}
		dir := c.AvoidObstacles(toTarget.Normalized(), e.Position, target.Position)
		e.Velocity = entity.Vec3{X: dir.X * spd, Z: dir.Z * spd}
	} else {
		margin := preferred * 0.2
		if distance < preferred-margin {
			bspd := def.CurrentBackpedalSpeed(e.Phase)
			dir := toTarget.Normalized().Neg().Flat()
			e.Velocity = entity.Vec3{X: dir.X * bspd, Z: dir.Z * bspd}
		} else if distance > preferred+margin {
			dir := c.AvoidObstacles(toTarget.Normalized(), e.Position, target.Position)
			e.Velocity = entity.Vec3{X: dir.X * spd, Z: dir.Z * spd}
		} else {
			e.Velocity = entity.Vec3{}
		}
	}

	return bt.Running
}

// --- Attack subtree leaves ---
//
// The attack lifecycle is modeled as a BT Sequence rather than a blackboard FSM.
// Each action handles one phase: select -> telegraph -> execute/charge -> cooldown.
// The Sequence's built-in runningIdx tracking replaces the old atk_phase state.

func attackAborted(e *entity.Enemy) bool {
	return !e.Alive || e.State == entity.EnemyPhaseTransition
}

func isTelegraphState(s entity.EnemyState) bool {
	return s == entity.EnemyMeleeTelegraph || s == entity.EnemyRangedTelegraph ||
		s == entity.EnemyAoETelegraph || s == entity.EnemyChargeTelegraph
}

func condActiveAbilityIsCharge(v any) bool {
	c := ctx(v)
	abil := c.Def.AbilityByIndex(c.Enemy.ActiveAbility)
	result := abil != nil && abil.Category == ability.CategoryCharge
	c.logCond("active_ability_is_charge", result)
	return result
}

// actionSelectAbility picks an ability and enters the telegraph state.
func actionSelectAbility(v any) bt.Result {
	c := ctx(v)
	e := c.Enemy
	if attackAborted(e) {
		return bt.Failure
	}
	target := c.TargetPlayer()
	if target == nil {
		return bt.Failure
	}
	distance := e.Position.Flat().DistanceTo(target.Position.Flat())
	chosen := c.SelectAbility(distance)
	if chosen == nil {
		return bt.Failure
	}
	c.StartAbility(chosen)
	c.logAction("select_ability", bt.Success, "ability", chosen.Name)
	return bt.Success
}

// actionTelegraph waits for the telegraph timer, optionally tracking the target.
func actionTelegraph(v any) bt.Result {
	c := ctx(v)
	e := c.Enemy
	if attackAborted(e) {
		return bt.Failure
	}
	e.Velocity = entity.Vec3{}
	abil := c.Def.AbilityByIndex(e.ActiveAbility)
	if abil != nil && abil.TrackTarget {
		switch abil.Category {
		case ability.CategoryCharge:
			c.faceTargetPlayer()
			if target := c.TargetPlayer(); target != nil {
				dir := target.Position.Sub(e.Position).Flat()
				if dir.Length() > 0.1 {
					e.ChargeDirection = dir.Normalized()
				}
			}
		case ability.CategoryRanged:
			if target := c.TargetPlayer(); target != nil {
				e.RangedTargetPos = target.Position.Add(entity.Vec3{Y: 1.0})
			}
		case ability.CategoryMelee, ability.CategoryAoE:
			c.faceTargetPlayer()
		}
	}
	if e.StateTimer <= 0 {
		c.logAction("telegraph", bt.Success)
		return bt.Success
	}
	return bt.Running
}

// actionExecuteAbility handles melee/ranged/AoE execution: enter attack state,
// wait for the execute timer, then resolve damage.
func actionExecuteAbility(v any) bt.Result {
	c := ctx(v)
	e := c.Enemy
	if attackAborted(e) {
		return bt.Failure
	}
	e.Velocity = entity.Vec3{}
	abil := c.Def.AbilityByIndex(e.ActiveAbility)
	if abil == nil {
		return bt.Failure
	}

	// First tick: transition from telegraph to attack state.
	if isTelegraphState(e.State) {
		resolved := c.Def.ResolveAbility(abil, e.Phase)
		switch abil.Category {
		case ability.CategoryMelee:
			e.State = entity.EnemyMeleeAttack
		case ability.CategoryRanged:
			e.State = entity.EnemyRangedAttack
		case ability.CategoryAoE:
			e.State = entity.EnemyAoESlam
		}
		e.StateTimer = resolved.ExecuteTime
		return bt.Running
	}

	// Wait for execute timer.
	if e.StateTimer > 0 {
		return bt.Running
	}

	// Resolve damage.
	resolved := c.ResolveCurrentAbility()
	switch abil.Category {
	case ability.CategoryMelee, ability.CategoryAoE:
		c.CastMeleeOrAoE(resolved)
	case ability.CategoryRanged:
		c.SpawnProjectiles(resolved)
	}
	c.logAction("execute_ability", bt.Success, "ability", abil.Name)
	return bt.Success
}

// actionChargeDash handles charge movement with per-player hit detection.
func actionChargeDash(v any) bt.Result {
	c := ctx(v)
	e := c.Enemy
	if attackAborted(e) {
		return bt.Failure
	}

	// First tick: transition from telegraph to charge state.
	if e.State != entity.EnemyCharge {
		e.State = entity.EnemyCharge
		e.ChargeDistance = 0
		e.ChargeHitPlayers = []uint16{}
		if e.ChargeDirection.LengthSq() < 0.01 {
			e.ChargeDirection = entity.Vec3{Z: -1}
		}
	}

	resolved := c.ResolveCurrentAbility()
	charge := resolved.Charge
	if charge == nil {
		return bt.Failure
	}
	spd := charge.Speed
	e.Velocity = entity.Vec3{X: e.ChargeDirection.X * spd, Z: e.ChargeDirection.Z * spd}
	e.ChargeDistance += spd * c.Dt

	// Per-player hit detection.
	for _, p := range c.Players {
		if !p.Alive || slices.Contains(e.ChargeHitPlayers, p.ID) {
			continue
		}
		if e.Position.DistanceTo(p.Position) <= charge.HitRadius {
			dealt := p.ApplyDamage(charge.Damage)
			if dealt > 0 {
				*c.Events = append(*c.Events, combat.DamageEvent{
					TargetPeerID: p.ID,
					Amount:       dealt,
					HitPos:       e.Position,
					SourceType:   resolved.DamageSource,
				})
				if c.Logger != nil {
					c.logAction("charge_dash", bt.Running, "phase", "charge_hit", "target", p.ID, "damage", dealt)
				}
			}
			e.ChargeHitPlayers = append(e.ChargeHitPlayers, p.ID)
		}
	}

	// Stop conditions.
	stop := e.ChargeDistance >= charge.MaxDistance
	if charge.StopOnWall {
		stop = stop || combat.IsAtWall(e.Position,
			c.BoundsMinX, c.BoundsMaxX,
			c.BoundsMinZ, c.BoundsMaxZ)
	}
	if charge.StopOnObstacle {
		stop = stop || combat.IsAtObstacle(e.Position, c.Obs, c.Def.Radius)
	}
	if stop {
		e.Velocity = entity.Vec3{}
		c.logAction("charge_dash", bt.Success)
		return bt.Success
	}
	return bt.Running
}

// actionCooldown waits for the ability cooldown before returning to chase.
func actionCooldown(v any) bt.Result {
	c := ctx(v)
	e := c.Enemy
	if attackAborted(e) {
		return bt.Failure
	}
	if e.State != entity.EnemyCooldown {
		c.EnterCooldown()
	}
	e.Velocity = entity.Vec3{}
	if e.StateTimer <= 0 {
		e.State = entity.EnemyChase
		c.logAction("cooldown", bt.Success)
		return bt.Success
	}
	return bt.Running
}

// --- Runner-based leaves ---

func condIsCasting(v any) bool {
	c := ctx(v)
	result := c.IsRunnerBusy()
	c.logCond("is_casting", result)
	return result
}

func condIsCommitted(v any) bool {
	c := ctx(v)
	result := c.Runner.Phase == RunnerCommit
	c.logCond("is_committed", result)
	return result
}

func condCanCast(v any) bool {
	c := ctx(v)
	result := c.Runner.Phase == RunnerIdle
	c.logCond("can_cast", result)
	return result
}

func condCanMove(v any) bool {
	c := ctx(v)
	r := c.Runner
	if r.Phase == RunnerIdle || r.Phase == RunnerCooldown {
		return true
	}
	abil := c.Def.AbilityByIndex(r.AbilIdx)
	if abil == nil {
		return true
	}
	switch r.Phase {
	case RunnerCommit:
		return abil.CanMoveCommitted
	case RunnerExecute:
		return abil.CanMoveExecuting
	}
	return true
}

func actionCastWeighted(v any) bt.Result {
	c := ctx(v)
	if !c.CastWeighted() {
		c.logAction("cast_weighted", bt.Failure)
		return bt.Failure
	}
	c.logAction("cast_weighted", bt.Success, "ability", c.CurrentAbilityID())
	return bt.Success
}

func actionWaitAbility(v any) bt.Result {
	c := ctx(v)
	if c.Runner.Phase == RunnerIdle {
		return bt.Success
	}
	return bt.Running
}

func actionCancelAbility(v any) bt.Result {
	c := ctx(v)
	if c.CancelAbility() {
		c.logAction("cancel_ability", bt.Success)
		return bt.Success
	}
	c.logAction("cancel_ability", bt.Failure)
	return bt.Failure
}

// castByName returns an action leaf that casts a specific ability by ID.
func castByName(abilityID string) func(any) bt.Result {
	return func(v any) bt.Result {
		c := ctx(v)
		if !c.Cast(abilityID) {
			c.logAction("cast", bt.Failure, "ability", abilityID)
			return bt.Failure
		}
		c.logAction("cast", bt.Success, "ability", abilityID)
		return bt.Success
	}
}

// --- Helpers ---

func (ctx *EntityContext) faceTargetPlayer() {
	p := ctx.TargetPlayer()
	if p != nil {
		ctx.FaceToward(p.Position)
	}
}
