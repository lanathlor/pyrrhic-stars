package enemyai

import (
	"math"
	"slices"

	"codex-online/server/internal/ability"
	"codex-online/server/internal/bt"
	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
)

func ctx(v any) *EntityContext { return v.(*EntityContext) } //nolint:revive,forcetypeassert // intentional: panic on wrong type is a programming error

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

const (
	// pillarCampRadius is how close to a pillar (obstacle edge) a player must be
	// to count as hugging it.
	pillarCampRadius float32 = 2.5
	// pillarBossProximity is how close the boss must be to the hugging player to
	// count as cheese. In the pillar cheese the boss is right on the other side
	// of the pillar but cannot hit through it; a ranged group whiffing the boss
	// from across the room is far away and must not trigger the punish.
	pillarBossProximity float32 = 6.0
	// pillarCheeseSeconds is how long the boss must go without landing any
	// damage before a pillar-hugging player is judged to be cheesing. The boss
	// commits constantly during the cheese but every ability whiffs (the player
	// ducks behind the pillar), so its damage output dries up. pillar_overload's
	// own damage (SourceEnemyPillar) is excluded from the drought, so once the
	// cheese starts the punish keeps refiring on cooldown; the moment the boss
	// lands any real hit the drought resets and the punish stops.
	pillarCheeseSeconds float32 = 7.0
)

// condPlayerCampingPillar is true when an alive player is hugging a pillar with
// the boss right next to them AND the boss has not landed any real damage for
// pillarCheeseSeconds. Together these identify the pillar cheese: a player
// orbiting a pillar the boss is stuck on, so its committed abilities keep
// whiffing. Proximity alone is not enough (players fight near pillars normally)
// and the boss-adjacency rules out a ranged group whiffing from range, so the
// damage drought distinguishes a cheesed boss from one trading blows. Gates
// pillar_overload.
func condPlayerCampingPillar(v any) bool {
	c := ctx(v)

	proxSq := pillarBossProximity * pillarBossProximity
	camping := false
	for _, p := range c.Players {
		if !p.Alive {
			continue
		}
		if !combat.IsAtPillar(p.Position, c.Obs, pillarCampRadius) {
			continue
		}
		if c.Enemy.Position.Flat().DistanceToSq(p.Position.Flat()) > proxSq {
			continue
		}
		camping = true
		break
	}
	if !camping {
		if c.Logger != nil {
			c.logCond("player_camping_pillar", false, "camping", false)
		}
		return false
	}

	cheesing := c.Enemy.SecsSinceDealtDamage >= pillarCheeseSeconds
	if c.Logger != nil {
		c.logCond("player_camping_pillar", cheesing, "secs_since_dealt", c.Enemy.SecsSinceDealtDamage)
	}
	return cheesing
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

// condTargetBeyond returns a condition factory parameterized by distance.
// Returns true when the current target is farther than d.
func condTargetBeyond(d float32) func(any) bool {
	return func(v any) bool {
		c := ctx(v)
		p := c.TargetPlayer()
		if p == nil {
			if c.Logger != nil {
				c.logCond("target_beyond", false, "reason", "no_target")
			}
			return false
		}
		dist := c.Enemy.Position.Flat().DistanceTo(p.Position.Flat())
		result := dist > d
		if c.Logger != nil {
			c.logCond("target_beyond", result, "dist", dist, "threshold", d)
		}
		return result
	}
}

// condPlayersInAoE returns a condition factory parameterized by radius.
// Returns true when 2+ alive players are within radius of the boss.
func condPlayersInAoE(radius float32) func(any) bool {
	rSq := radius * radius
	return func(v any) bool {
		c := ctx(v)
		count := 0
		for _, p := range c.Players {
			if !p.Alive {
				continue
			}
			if c.Enemy.Position.Flat().DistanceToSq(p.Position.Flat()) <= rSq {
				count++
			}
		}
		result := count >= 2
		if c.Logger != nil {
			c.logCond("players_in_aoe", result, "radius", radius, "count", count)
		}
		return result
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

// condNPlayersClustered returns a condition factory: true if any alive player
// has minNeighbors+ alive allies within 6m. O(N²), trivial for N=5.
func condNPlayersClustered(minNeighbors int) func(any) bool {
	const clusterRadius float32 = 6.0
	rSq := clusterRadius * clusterRadius
	return func(v any) bool {
		c := ctx(v)
		for _, p := range c.Players {
			if !p.Alive {
				continue
			}
			neighbors := 0
			for _, q := range c.Players {
				if q == p || !q.Alive {
					continue
				}
				if p.Position.Flat().DistanceToSq(q.Position.Flat()) <= rSq {
					neighbors++
				}
			}
			if neighbors >= minNeighbors {
				if c.Logger != nil {
					c.logCond("n_players_clustered", true, "min", minNeighbors, "player", p.ID, "neighbors", neighbors)
				}
				return true
			}
		}
		if c.Logger != nil {
			c.logCond("n_players_clustered", false, "min", minNeighbors)
		}
		return false
	}
}

// condAnyBelowHPPct returns a condition factory: true if any alive player has
// Health/MaxHealth below the given percentage (0-100). O(N), very cheap.
func condAnyBelowHPPct(pct float32) func(any) bool {
	threshold := pct / 100.0
	return func(v any) bool {
		c := ctx(v)
		for _, p := range c.Players {
			if !p.Alive {
				continue
			}
			if p.MaxHealth > 0 && p.Health/p.MaxHealth < threshold {
				if c.Logger != nil {
					c.logCond("any_below_hp_pct", true, "pct", pct, "player", p.ID,
						"hp_pct", p.Health/p.MaxHealth*100)
				}
				return true
			}
		}
		if c.Logger != nil {
			c.logCond("any_below_hp_pct", false, "pct", pct)
		}
		return false
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

// actionSetTargetClustered finds the alive player with the most alive allies
// within 6m and sets them as the target. Tiebreaker: farthest from boss (to
// spread damage away from the tank). O(N²), trivial for N=5.
func actionSetTargetClustered(v any) bt.Result {
	const clusterRadius float32 = 6.0
	rSq := clusterRadius * clusterRadius
	c := ctx(v)

	var best *entity.Player
	bestNeighbors := -1
	var bestDistSq float32

	for _, p := range c.Players {
		if !p.Alive {
			continue
		}
		neighbors := 0
		for _, q := range c.Players {
			if q == p || !q.Alive {
				continue
			}
			if p.Position.Flat().DistanceToSq(q.Position.Flat()) <= rSq {
				neighbors++
			}
		}
		distSq := c.Enemy.Position.Flat().DistanceToSq(p.Position.Flat())
		if neighbors > bestNeighbors || (neighbors == bestNeighbors && distSq > bestDistSq) {
			best = p
			bestNeighbors = neighbors
			bestDistSq = distSq
		}
	}

	if best == nil {
		return bt.Failure
	}
	c.Enemy.TargetPlayerID = best.ID
	if c.Logger != nil {
		c.logAction("set_target_clustered", bt.Success, "target", best.ID, "neighbors", bestNeighbors)
	}
	return bt.Success
}

// actionSetTargetLowestHP finds the alive player with the lowest HP percentage
// and sets them as the target. O(N), very cheap.
func actionSetTargetLowestHP(v any) bt.Result {
	c := ctx(v)
	var best *entity.Player
	bestPct := float32(2.0) // > 100%

	for _, p := range c.Players {
		if !p.Alive || p.MaxHealth <= 0 {
			continue
		}
		pct := p.Health / p.MaxHealth
		if pct < bestPct {
			bestPct = pct
			best = p
		}
	}

	if best == nil {
		return bt.Failure
	}
	c.Enemy.TargetPlayerID = best.ID
	if c.Logger != nil {
		c.logAction("set_target_lowest_hp", bt.Success, "target", best.ID, "hp_pct", bestPct*100)
	}
	return bt.Success
}

// actionSetTargetNearest sets the target to the nearest alive player without
// changing enemy state. Unlike aggro_nearest, this is a pure retarget action.
func actionSetTargetNearest(v any) bt.Result {
	c := ctx(v)
	p := c.NearestPlayer()
	if p == nil {
		return bt.Failure
	}
	c.Enemy.TargetPlayerID = p.ID
	if c.Logger != nil {
		c.logAction("set_target_nearest", bt.Success, "target", p.ID)
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

	// If externally aggroed (e.g., by taking damage via AggroEnemy), abort
	// patrol so the BT falls through to the chase branch.
	if e.State != entity.EnemyPatrol {
		return bt.Failure
	}

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
	if e.HasDebuff(entity.DebuffRoot) {
		e.Velocity = entity.Vec3{}
		return bt.Running
	}
	dir := toTarget.Normalized()
	spd := c.Def.MoveSpeed * 0.5
	if slow := e.GetDebuffValue(entity.DebuffSlow); slow > 0 {
		spd *= (1.0 - slow)
	}
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

	// Debuff checks: root freezes movement, slow reduces speed
	if e.HasDebuff(entity.DebuffRoot) {
		e.Velocity = entity.Vec3{}
		return bt.Running
	}
	slowMult := float32(1.0)
	if slow := e.GetDebuffValue(entity.DebuffSlow); slow > 0 {
		slowMult = 1.0 - slow
	}

	spd := def.CurrentMoveSpeed(e.Phase) * slowMult
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
		switch {
		case distance < preferred-margin:
			bspd := def.CurrentBackpedalSpeed(e.Phase) * slowMult
			dir := toTarget.Normalized().Neg().Flat()
			e.Velocity = entity.Vec3{X: dir.X * bspd, Z: dir.Z * bspd}
		case distance > preferred+margin:
			dir := c.AvoidObstacles(toTarget.Normalized(), e.Position, target.Position)
			e.Velocity = entity.Vec3{X: dir.X * spd, Z: dir.Z * spd}
		default:
			e.Velocity = entity.Vec3{}
		}
	}

	return bt.Running
}

// Blackboard keys + tuning for strafe/dash movement.
const (
	bbStrafeSide = "strafe_side" // flag set => strafe to the mob's left
	bbStrafeFlip = "strafe_flip" // timer until the next strafe-side flip
	bbDashActive = "dash_active" // timer: remaining dash burst window
	bbDashCD     = "dash_cd"     // timer: dash cooldown

	strafeFlipPeriod = float32(0.9) // seconds between strafe direction flips
	strafeSpeedMult  = float32(0.9) // strafe slightly slower than full move speed
	dashDuration     = float32(0.25)
	dashSpeedMult    = float32(2.5) // dash speed relative to move speed
)

// actionStrafe circle-strafes the current target: it sidesteps perpendicular to
// the target, flipping direction on a timer so it weaves rather than orbiting,
// with a small radial nudge to hold its preferred kite distance. Ranged mobs use
// it between volleys so they reposition instead of standing still. Returns Running.
func actionStrafe(v any) bt.Result {
	c := ctx(v)
	e := c.Enemy
	def := c.Def

	if !e.Alive || e.State == entity.EnemyPhaseTransition {
		e.Velocity = entity.Vec3{}
		return bt.Failure
	}

	target := c.TargetPlayer()
	if target == nil {
		target = c.NearestPlayer()
	}
	if target == nil {
		e.Velocity = entity.Vec3{}
		return bt.Failure
	}
	e.TargetPlayerID = target.ID

	toTarget := target.Position.Sub(e.Position).Flat()
	distance := toTarget.Length()
	if distance < 0.1 {
		e.Velocity = entity.Vec3{}
		return bt.Failure
	}
	dir := toTarget.Normalized()
	e.RotationY = float32(math.Atan2(float64(-dir.X), float64(-dir.Z)))

	if e.HasDebuff(entity.DebuffRoot) {
		e.Velocity = entity.Vec3{}
		return bt.Running
	}
	slowMult := float32(1.0)
	if slow := e.GetDebuffValue(entity.DebuffSlow); slow > 0 {
		slowMult = 1.0 - slow
	}

	side := nextStrafeSide(c.BB)
	moveDir := strafeMoveDir(dir, side, distance, def.PreferredRange)
	if moveDir.Length() < 0.01 {
		e.Velocity = entity.Vec3{}
		return bt.Running
	}
	moveDir = moveDir.Normalized()
	moveDir = c.AvoidObstacles(moveDir, e.Position, e.Position.Add(moveDir.Scale(2)))

	spd := def.CurrentMoveSpeed(e.Phase) * slowMult * strafeSpeedMult
	e.Velocity = entity.Vec3{X: moveDir.X * spd, Z: moveDir.Z * spd}
	if c.Logger != nil {
		c.logAction("strafe", bt.Running, "side", side, "dist", distance)
	}
	return bt.Running
}

// nextStrafeSide returns +1 or -1, flipping the stored strafe side each time the
// flip timer expires so the mob weaves instead of orbiting one way forever.
func nextStrafeSide(bb *Blackboard) float32 {
	if bb.TimerExpired(bbStrafeFlip) {
		if bb.GetFlag(bbStrafeSide) {
			bb.ClearFlag(bbStrafeSide)
		} else {
			bb.SetFlag(bbStrafeSide)
		}
		bb.StartTimer(bbStrafeFlip, strafeFlipPeriod)
	}
	if bb.GetFlag(bbStrafeSide) {
		return -1
	}
	return 1
}

// strafeMoveDir returns the (flat, un-normalized) strafe direction: perpendicular
// to the target, nudged radially to hold the preferred kite band.
func strafeMoveDir(dir entity.Vec3, side, distance, preferred float32) entity.Vec3 {
	moveDir := entity.Vec3{X: -dir.Z, Z: dir.X}.Scale(side)
	if preferred > 0 {
		radialErr := entity.Clamp((distance-preferred)/(preferred*0.5), -1, 1)
		moveDir = moveDir.Add(dir.Scale(radialErr * 0.6))
	}
	return moveDir.Flat()
}

// dashFactory builds a dash action: a short cooldowned burst of speed toward the
// current target to close distance (movement only, no damage). cooldown sets how
// often it fires - a long value for an occasional gap-closer, a short one for a
// relentless chaser. Returns Failure while on cooldown so the BT falls through to
// plain chase, and Running during the burst.
func dashFactory(cooldown float32) func(any) bt.Result {
	return func(v any) bt.Result {
		c := ctx(v)
		e := c.Enemy
		def := c.Def

		if !e.Alive || e.State == entity.EnemyPhaseTransition {
			e.Velocity = entity.Vec3{}
			return bt.Failure
		}

		dashing := !c.BB.TimerExpired(bbDashActive)
		if !dashing && !c.BB.TimerExpired(bbDashCD) {
			e.Velocity = entity.Vec3{}
			return bt.Failure // on cooldown, not mid-burst (chase fallback takes over)
		}

		target := c.TargetPlayer()
		if target == nil {
			target = c.NearestPlayer()
		}
		if target == nil {
			e.Velocity = entity.Vec3{}
			return bt.Failure
		}
		e.TargetPlayerID = target.ID

		toTarget := target.Position.Sub(e.Position).Flat()
		distance := toTarget.Length()
		if distance < 0.1 {
			e.Velocity = entity.Vec3{}
			return bt.Failure
		}
		dir := toTarget.Normalized()
		e.RotationY = float32(math.Atan2(float64(-dir.X), float64(-dir.Z)))

		if e.HasDebuff(entity.DebuffRoot) {
			e.Velocity = entity.Vec3{}
			return bt.Running
		}
		slowMult := float32(1.0)
		if slow := e.GetDebuffValue(entity.DebuffSlow); slow > 0 {
			slowMult = 1.0 - slow
		}

		if !dashing {
			// Begin a new burst and put the dash on cooldown (measured from start).
			c.BB.StartTimer(bbDashActive, dashDuration)
			c.BB.StartTimer(bbDashCD, cooldown)
		}

		moveDir := c.AvoidObstacles(dir, e.Position, target.Position)
		spd := def.CurrentMoveSpeed(e.Phase) * slowMult * dashSpeedMult
		e.Velocity = entity.Vec3{X: moveDir.X * spd, Z: moveDir.Z * spd}
		if c.Logger != nil {
			c.logAction("dash", bt.Running, "dist", distance)
		}
		return bt.Running
	}
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
		c.CommitMeleeOrAoE(resolved)
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

func condIsCommitted(v any) bool {
	c := ctx(v)
	result := c.IsRunnerBusy()
	c.logCond("is_committed", result)
	return result
}

func condCanCommit(v any) bool {
	c := ctx(v)
	result := c.Runner.Phase == RunnerIdle
	c.logCond("can_commit", result)
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

func actionCommitWeighted(v any) bt.Result {
	c := ctx(v)
	if !c.CommitWeighted() {
		c.logAction("commit_weighted", bt.Failure)
		return bt.Failure
	}
	c.logAction("commit_weighted", bt.Success, "ability", c.CurrentAbilityID())
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

// condAbilityReady returns a condition factory that checks per-ability cooldown.
func condAbilityReady(abilityID string) func(any) bool {
	return func(v any) bool {
		c := ctx(v)
		result := c.Runner.IsAbilityReady(c, abilityID)
		c.logCond("ability_ready", result, "ability", abilityID)
		return result
	}
}

// commitByName returns an action leaf that commits a specific ability by ID.
func commitByName(abilityID string) func(any) bt.Result {
	return func(v any) bt.Result {
		c := ctx(v)
		if !c.Commit(abilityID) {
			c.logAction("commit", bt.Failure, "ability", abilityID)
			return bt.Failure
		}
		c.logAction("commit", bt.Success, "ability", abilityID)
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
