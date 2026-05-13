package enemyai

import (
	"slices"

	"codex-online/server/internal/ability"
	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
)

// RunnerPhase tracks where the enemy is in the ability lifecycle.
type RunnerPhase uint8

const (
	RunnerIdle     RunnerPhase = iota
	RunnerCommit               // telegraph window (CommitTime)
	RunnerExecute              // execution window (ExecuteTime)
	RunnerCooldown             // post-ability cooldown
)

// AbilityRunner owns the commit→execute→cooldown lifecycle for one enemy.
// The BT issues Cast/Cancel commands; the runner advances the state machine.
// Timer management: Brain.Tick decrements Enemy.StateTimer each tick.
// The runner sets StateTimer at phase transitions and reads it to detect expiry.
type AbilityRunner struct {
	Phase      RunnerPhase
	AbilIdx    int             // index into EnemyDef.Abilities
	AbilityCDs map[int]float32 // per-ability cooldown timers (index → remaining seconds)
}

// Start initiates an ability by ID. Returns true if accepted (runner was idle).
func (r *AbilityRunner) Start(ctx *EntityContext, abilityID string) bool {
	if r.Phase != RunnerIdle {
		return false
	}

	// Find ability by ID
	idx := -1
	for i := range ctx.Def.Abilities {
		if ctx.Def.Abilities[i].ID == abilityID {
			idx = i
			break
		}
	}
	if idx < 0 {
		return false
	}

	abil := &ctx.Def.Abilities[idx]
	e := ctx.Enemy
	resolved := ctx.Def.ResolveAbility(abil, e.Phase)

	r.Phase = RunnerCommit
	r.AbilIdx = idx
	e.ActiveAbility = idx
	ctx.BB.Set("last_attack", abil.ID)
	e.Velocity = entity.Vec3{}

	// Set enemy state and timer based on ability category.
	// Mirrors the logic previously in EntityContext.StartAbility.
	switch abil.Category {
	case ability.CategoryMelee:
		e.State = entity.EnemyMeleeTelegraph
		e.StateTimer = resolved.CommitTime
		e.MeleeRange = resolved.Hit.Range
		arcDeg := resolved.Hit.ArcDegrees
		if arcDeg <= 0 {
			arcDeg = 180
		}
		e.MeleeConeAngle = arcDeg * 3.14159265 / 180.0
		if resolved.FaceTarget {
			if p := ctx.TargetPlayer(); p != nil {
				ctx.FaceToward(p.Position)
			}
		}
	case ability.CategoryRanged:
		e.State = entity.EnemyRangedTelegraph
		e.StateTimer = resolved.CommitTime
		var target *entity.Player
		switch abil.TargetStrategy {
		case ability.TargetFarthest:
			target = FarthestAlivePlayer(e.Position, ctx.Players)
		case ability.TargetCurrent:
			target = ctx.TargetPlayer()
		default: // TargetNearest
			target = NearestAlivePlayer(e.Position, ctx.Players)
		}
		if target != nil {
			e.TargetPlayerID = target.ID
			e.RangedTargetPos = target.Position.Add(entity.Vec3{Y: 1.0})
		}
	case ability.CategoryAoE:
		e.State = entity.EnemyAoETelegraph
		e.StateTimer = resolved.CommitTime
		if resolved.FaceTarget {
			if p := ctx.TargetPlayer(); p != nil {
				ctx.FaceToward(p.Position)
			}
		}
	case ability.CategoryCharge:
		e.State = entity.EnemyChargeTelegraph
		e.StateTimer = resolved.CommitTime
		if resolved.FaceTarget {
			if p := ctx.TargetPlayer(); p != nil {
				ctx.FaceToward(p.Position)
				dir := p.Position.Sub(e.Position).Flat()
				if dir.Length() > 0.1 {
					e.ChargeDirection = dir.Normalized()
				}
			}
		}
		if e.ChargeDirection.LengthSq() < 0.01 {
			e.ChargeDirection = entity.Vec3{Z: -1}
		}
	}

	return true
}

// ForceStart unconditionally resets the runner and starts the given ability.
// Used by dev mode force-cast: cancels any in-progress ability, clears the
// ability's cooldown, and initiates the cast.
func (r *AbilityRunner) ForceStart(ctx *EntityContext, abilityID string) bool {
	// Reset runner and enemy state unconditionally.
	r.Phase = RunnerIdle
	ctx.Enemy.State = entity.EnemyChase
	ctx.Enemy.Velocity = entity.Vec3{}
	ctx.Enemy.StateTimer = 0

	// Clear this ability's per-ability cooldown.
	for i := range ctx.Def.Abilities {
		if ctx.Def.Abilities[i].ID == abilityID {
			delete(r.AbilityCDs, i)
			break
		}
	}

	return r.Start(ctx, abilityID)
}

// Cancel aborts the current ability if in commit phase and the ability is cancellable.
func (r *AbilityRunner) Cancel(ctx *EntityContext) bool {
	if r.Phase != RunnerCommit {
		return false
	}
	abil := ctx.Def.AbilityByIndex(r.AbilIdx)
	if abil == nil {
		return false
	}
	resolved := ctx.Def.ResolveAbility(abil, ctx.Enemy.Phase)
	if !resolved.Cancellable {
		return false
	}
	r.Phase = RunnerIdle
	ctx.Enemy.State = entity.EnemyChase
	ctx.Enemy.Velocity = entity.Vec3{}
	return true
}

// Tick advances the ability lifecycle by one step. Called after the BT tick,
// before velocity application. Brain.Tick decrements Enemy.StateTimer.
func (r *AbilityRunner) Tick(ctx *EntityContext) {
	// Always tick per-ability cooldowns, even when idle.
	for idx, cd := range r.AbilityCDs {
		cd -= ctx.Dt
		if cd <= 0 {
			delete(r.AbilityCDs, idx)
		} else {
			r.AbilityCDs[idx] = cd
		}
	}

	switch r.Phase {
	case RunnerIdle:
		return
	case RunnerCommit:
		r.tickCommit(ctx)
	case RunnerExecute:
		r.tickExecute(ctx)
	case RunnerCooldown:
		r.tickCooldown(ctx)
	}
}

// IsAbilityReady returns true if the named ability's per-ability cooldown has expired.
func (r *AbilityRunner) IsAbilityReady(ctx *EntityContext, abilityID string) bool {
	for i := range ctx.Def.Abilities {
		if ctx.Def.Abilities[i].ID == abilityID {
			if r.AbilityCDs == nil {
				return true
			}
			_, onCD := r.AbilityCDs[i]
			return !onCD
		}
	}
	return false
}

func (r *AbilityRunner) tickCommit(ctx *EntityContext) {
	e := ctx.Enemy
	if attackAborted(e) {
		r.Phase = RunnerIdle
		return
	}

	abil := ctx.Def.AbilityByIndex(r.AbilIdx)
	if abil == nil {
		r.Phase = RunnerIdle
		return
	}

	// Enforce movement restriction
	if !abil.CanMoveCommitted {
		e.Velocity = entity.Vec3{}
	}

	// Track target during commit
	if abil.TrackTarget {
		switch abil.Category {
		case ability.CategoryCharge:
			ctx.faceTargetPlayer()
			if target := ctx.TargetPlayer(); target != nil {
				dir := target.Position.Sub(e.Position).Flat()
				if dir.Length() > 0.1 {
					e.ChargeDirection = dir.Normalized()
				}
			}
		case ability.CategoryRanged:
			if target := ctx.TargetPlayer(); target != nil {
				e.RangedTargetPos = target.Position.Add(entity.Vec3{Y: 1.0})
			}
		case ability.CategoryMelee, ability.CategoryAoE:
			ctx.faceTargetPlayer()
		}
	}

	// Wait for commit timer to expire
	if e.StateTimer > 0 {
		return
	}

	// Transition to execute phase
	resolved := ctx.Def.ResolveAbility(abil, e.Phase)
	r.Phase = RunnerExecute

	switch abil.Category {
	case ability.CategoryMelee:
		e.State = entity.EnemyMeleeAttack
	case ability.CategoryRanged:
		e.State = entity.EnemyRangedAttack
	case ability.CategoryAoE:
		e.State = entity.EnemyAoESlam
	case ability.CategoryCharge:
		e.State = entity.EnemyCharge
		e.ChargeDistance = 0
		e.ChargeHitPlayers = []uint16{}
		if e.ChargeDirection.LengthSq() < 0.01 {
			e.ChargeDirection = entity.Vec3{Z: -1}
		}
	}
	e.StateTimer = resolved.ExecuteTime
}

func (r *AbilityRunner) tickExecute(ctx *EntityContext) {
	e := ctx.Enemy
	if attackAborted(e) {
		r.Phase = RunnerIdle
		return
	}

	abil := ctx.Def.AbilityByIndex(r.AbilIdx)
	if abil == nil {
		r.Phase = RunnerIdle
		return
	}

	// Charge abilities have special per-tick execution
	if abil.Category == ability.CategoryCharge {
		r.tickCharge(ctx, abil)
		return
	}

	// Enforce movement restriction
	if !abil.CanMoveExecuting {
		e.Velocity = entity.Vec3{}
	}

	// Wait for execute timer
	if e.StateTimer > 0 {
		return
	}

	// Resolve damage
	resolved := ctx.Def.ResolveAbility(abil, e.Phase)
	switch abil.Category {
	case ability.CategoryMelee, ability.CategoryAoE:
		ctx.CastMeleeOrAoE(resolved)
	case ability.CategoryRanged:
		ctx.SpawnProjectiles(resolved)
	}

	r.enterCooldown(ctx, abil)
}

func (r *AbilityRunner) tickCharge(ctx *EntityContext, abil *ability.AbilityDef) {
	e := ctx.Enemy
	resolved := ctx.Def.ResolveAbility(abil, e.Phase)
	charge := resolved.Charge
	if charge == nil {
		r.enterCooldown(ctx, abil)
		return
	}

	spd := charge.Speed
	e.Velocity = entity.Vec3{X: e.ChargeDirection.X * spd, Z: e.ChargeDirection.Z * spd}
	e.ChargeDistance += spd * ctx.Dt

	// Per-player hit detection
	for _, p := range ctx.Players {
		if !p.Alive || slices.Contains(e.ChargeHitPlayers, p.ID) {
			continue
		}
		if e.Position.DistanceTo(p.Position) <= charge.HitRadius {
			dealt := p.ApplyDamage(charge.Damage)
			if dealt > 0 {
				*ctx.Events = append(*ctx.Events, combat.DamageEvent{
					TargetPeerID: p.ID,
					Amount:       dealt,
					HitPos:       e.Position,
					SourceType:   resolved.DamageSource,
				})
			}
			e.ChargeHitPlayers = append(e.ChargeHitPlayers, p.ID)
		}
	}

	// Stop conditions
	stop := e.ChargeDistance >= charge.MaxDistance
	if charge.StopOnWall {
		stop = stop || combat.IsAtWall(e.Position,
			ctx.BoundsMinX, ctx.BoundsMaxX,
			ctx.BoundsMinZ, ctx.BoundsMaxZ)
	}
	if charge.StopOnObstacle {
		stop = stop || combat.IsAtObstacle(e.Position, ctx.Obs, ctx.Def.Radius)
	}
	if stop {
		e.Velocity = entity.Vec3{}
		r.enterCooldown(ctx, abil)
	}
}

func (r *AbilityRunner) tickCooldown(ctx *EntityContext) {
	e := ctx.Enemy
	if attackAborted(e) {
		r.Phase = RunnerIdle
		return
	}
	e.Velocity = entity.Vec3{}
	if e.StateTimer <= 0 {
		r.Phase = RunnerIdle
		e.State = entity.EnemyChase
	}
}

// enterCooldown transitions the runner into cooldown phase.
// Sets per-ability cooldown for the used ability, and a short GCD for the global state.
func (r *AbilityRunner) enterCooldown(ctx *EntityContext, abil *ability.AbilityDef) {
	e := ctx.Enemy

	// Per-ability cooldown
	abilityCooldown := ctx.Def.CurrentCooldownTime(abil, e.Phase)
	if abilityCooldown > 0 {
		if r.AbilityCDs == nil {
			r.AbilityCDs = make(map[int]float32)
		}
		r.AbilityCDs[r.AbilIdx] = abilityCooldown
	}

	// Global cooldown (short recovery before next ability)
	gcd := ctx.Def.CurrentGCD(e.Phase)
	r.Phase = RunnerCooldown
	e.State = entity.EnemyCooldown
	e.StateTimer = gcd
	e.Velocity = entity.Vec3{}
}
