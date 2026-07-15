package enemyai

import (
	"log/slog"
	"math"
	"math/rand/v2"

	"codex-online/server/internal/ability"
	"codex-online/server/internal/bt"
	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
)

// EntityContext bridges BT leaf functions to the game state. It is allocated
// once per Brain and reset each tick. Lazy-cached queries avoid repeated
// computation within a single tick.
type EntityContext struct {
	Enemy           *entity.Enemy
	Def             *EnemyDef
	Engine          *ability.Engine
	BB              *Blackboard
	Rng             *rand.Rand
	Dt              float32
	Players         []*entity.Player
	Obs             []combat.Obstacle
	SpawnFn         func(pos, dir entity.Vec3, speed, damage, lifetime float32)
	CommitPatternFn func(pattern *combat.PatternDef, abilityName string, origin, facing entity.Vec3)
	Events          *[]combat.DamageEvent

	// SpawnAddFn summons a new enemy into the zone (CategorySpawn abilities).
	// Set externally per tick like Bus; nil in isolated unit tests.
	SpawnAddFn func(defName string, pos entity.Vec3, ownerID uint16) bool

	// Allies is the zone's full enemy slice (this enemy included). Set
	// externally per tick; used by add-awareness conditions. Nil-safe.
	Allies []*entity.Enemy

	// Runner owns the ability commit→execute→cooldown lifecycle.
	Runner *AbilityRunner

	// Bus is the zone coordination event log (shared by all brains in the
	// zone). Nil in isolated unit tests; coordination leaves fail open.
	Bus *Bus

	// Logger enables optional BT trace logging. Nil disables logging.
	Logger *slog.Logger

	// Bounds for charge wall detection
	BoundsMinX, BoundsMaxX, BoundsMinZ, BoundsMaxZ float32

	// Lazy-cached queries (reset per tick)
	nearestPlayer  *entity.Player
	farthestPlayer *entity.Player
	nearestCached  bool
	farthestCached bool

	// Reusable buffers
	targetBuf []entity.Target
	commitCtx ability.CommitContext
}

// Reset prepares the context for a new tick. Clears cached queries.
func (ctx *EntityContext) Reset(dt float32, players []*entity.Player,
	obstacles []combat.Obstacle,
	spawnFn func(pos, dir entity.Vec3, speed, damage, lifetime float32),
	commitPatternFn func(pattern *combat.PatternDef, abilityName string, origin, facing entity.Vec3),
	events *[]combat.DamageEvent) {
	ctx.Dt = dt
	ctx.Players = players
	ctx.Obs = obstacles
	ctx.SpawnFn = spawnFn
	ctx.CommitPatternFn = commitPatternFn
	ctx.Events = events
	ctx.nearestCached = false
	ctx.farthestCached = false
	ctx.nearestPlayer = nil
	ctx.farthestPlayer = nil
}

// --- Self ---

func (ctx *EntityContext) HealthPct() float32 {
	if ctx.Enemy.MaxHealth <= 0 {
		return 0
	}
	return ctx.Enemy.Health / ctx.Enemy.MaxHealth
}

func (ctx *EntityContext) Position() entity.Vec3 { return ctx.Enemy.Position }
func (ctx *EntityContext) Phase() int            { return ctx.Enemy.Phase }
func (ctx *EntityContext) IsAlive() bool         { return ctx.Enemy.Alive }

// --- Threat / Targeting ---

func (ctx *EntityContext) NearestPlayer() *entity.Player {
	if !ctx.nearestCached {
		ctx.nearestPlayer = NearestAlivePlayer(ctx.Enemy.Position, ctx.Players)
		ctx.nearestCached = true
	}
	return ctx.nearestPlayer
}

func (ctx *EntityContext) FarthestPlayer() *entity.Player {
	if !ctx.farthestCached {
		ctx.farthestPlayer = FarthestAlivePlayer(ctx.Enemy.Position, ctx.Players)
		ctx.farthestCached = true
	}
	return ctx.farthestPlayer
}

func (ctx *EntityContext) AlivePlayerCount() int {
	n := 0
	for _, p := range ctx.Players {
		if p.Alive {
			n++
		}
	}
	return n
}

func (ctx *EntityContext) TargetPlayer() *entity.Player {
	id := ctx.Enemy.TargetPlayerID
	for _, p := range ctx.Players {
		if p.ID == id && p.Alive {
			return p
		}
	}
	return nil
}

// --- Perception ---

func (ctx *EntityContext) HasLineOfSight(target entity.Vec3) bool {
	return !combat.SegmentHitsExpandedObstacle(ctx.Enemy.Position, target, ctx.Obs, ctx.Def.Radius)
}

// --- Combat ---

// SelectAbility runs weighted random ability selection.
func (ctx *EntityContext) SelectAbility(distance float32) *ability.AbilityDef {
	e := ctx.Enemy
	def := ctx.Def
	phase := def.CurrentPhase(e.Phase)

	type candidate struct {
		ability *ability.AbilityDef
		weight  int
	}

	var buf [8]candidate
	candidates := buf[:0]
	for i := range def.Abilities {
		a := &def.Abilities[i]
		if a.MinRange > 0 && distance < a.MinRange {
			continue
		}
		if a.MaxRange > 0 && distance > a.MaxRange {
			continue
		}
		// Runner.Start does not re-check cooldowns; filtering here is what
		// makes cooldown_time real for weighted selection.
		if cd, ok := ctx.Runner.AbilityCDs[i]; ok && cd > 0 {
			continue
		}

		weight := a.BaseWeight
		if phase != nil {
			if w, ok := phase.WeightOverrides[a.ID]; ok {
				weight = w
			}
		}

		// Anti-repeat
		if a.ID == ctx.BB.GetString("last_attack") && weight > 1 && def.AntiRepeat > 0 {
			weight = int(float32(weight) / def.AntiRepeat)
		}

		if weight > 0 {
			candidates = append(candidates, candidate{a, weight})
		}
	}

	if len(candidates) == 0 {
		return nil
	}

	total := 0
	for _, c := range candidates {
		total += c.weight
	}
	if total <= 0 {
		return candidates[0].ability
	}
	roll := ctx.Rng.IntN(total)
	cumulative := 0
	for _, c := range candidates {
		cumulative += c.weight
		if roll < cumulative {
			return c.ability
		}
	}
	return candidates[0].ability
}

// FindAbilityByCategory returns the first ability matching the given category.
func (ctx *EntityContext) FindAbilityByCategory(c ability.AbilityCategory) *ability.AbilityDef {
	for i := range ctx.Def.Abilities {
		if ctx.Def.Abilities[i].Category == c {
			return &ctx.Def.Abilities[i]
		}
	}
	return nil
}

// AbilityIndex returns the index of an ability in the def's Abilities slice.
func (ctx *EntityContext) AbilityIndex(a *ability.AbilityDef) int {
	for i := range ctx.Def.Abilities {
		if &ctx.Def.Abilities[i] == a {
			return i
		}
	}
	return 0
}

// StartAbility sets up the commit state for an ability on the entity.
// When FaceTarget is true, the enemy faces the target at the moment of
// commitment — after this point, rotation only updates if TrackTarget is set.
func (ctx *EntityContext) StartAbility(abil *ability.AbilityDef) {
	e := ctx.Enemy
	resolved := ctx.Def.ResolveAbility(abil, e.Phase)
	e.ActiveAbility = ctx.AbilityIndex(abil)
	ctx.BB.Set("last_attack", abil.ID)
	e.Velocity = entity.Vec3{}

	setupAbilityCategory(ctx, abil, resolved)
}

// ResolveCurrentAbility returns the resolved AbilityDef for the currently active ability.
func (ctx *EntityContext) ResolveCurrentAbility() ability.AbilityDef {
	abil := ctx.Def.AbilityByIndex(ctx.Enemy.ActiveAbility)
	if abil == nil {
		return ability.AbilityDef{}
	}
	return ctx.Def.ResolveAbility(abil, ctx.Enemy.Phase)
}

// CommitMeleeOrAoE resolves a melee/AoE hit via the ability engine and appends damage events.
func (ctx *EntityContext) CommitMeleeOrAoE(resolved ability.AbilityDef) {
	ctx.fillTargets()

	ctx.commitCtx.Committer = ctx.Enemy
	ctx.commitCtx.Targets = ctx.targetBuf
	ctx.commitCtx.Obstacles = ctx.Obs
	ctx.commitCtx.SourceType = resolved.DamageSource

	result := ctx.Engine.CommitDef(&resolved, &ctx.commitCtx)
	for _, r := range result.Events {
		*ctx.Events = append(*ctx.Events, combat.DamageEvent{
			TargetPeerID: r.TargetID,
			Amount:       r.Amount,
			HitPos:       r.HitPos,
			SourceType:   r.SourceType,
		})
	}
}

// SpawnProjectiles spawns projectiles for a ranged attack.
// If the ability has a Pattern definition, uses the pattern engine for
// bullet-hell style multi-wave emission. Otherwise uses the legacy fan system.
func (ctx *EntityContext) SpawnProjectiles(resolved ability.AbilityDef) {
	e := ctx.Enemy

	var originY float32
	if resolved.Projectile != nil {
		originY = resolved.Projectile.OriginY
	}
	origin := e.Position.Add(entity.Vec3{Y: originY})

	// Aim direction(s): toward the committed target, or - for a twin-lock
	// ability - one toward each of the N nearest players, fired at once.
	for _, baseDir := range ctx.aimDirections(resolved, origin) {
		// Pattern engine path: multi-wave bullet-hell patterns
		if resolved.Pattern != nil && ctx.CommitPatternFn != nil {
			ctx.CommitPatternFn(resolved.Pattern, resolved.Name, origin, baseDir)
			continue
		}

		// Fan path: simple fan of projectiles
		if resolved.Projectile == nil {
			return
		}
		proj := resolved.Projectile
		for i := range proj.Count {
			offset := (float32(i) - float32(proj.Count-1)/2.0) * proj.Spread
			dir := combat.RotateVecY(baseDir, offset)
			ctx.SpawnFn(
				origin,
				dir,
				proj.Speed,
				proj.Damage,
				proj.Lifetime,
			)
		}
	}
}

// aimDirections returns the base direction(s) a ranged ability fires along: a
// single direction toward the committed target (RangedTargetPos), or - when
// MultiTargetCount > 1 - one toward each of the N nearest alive players.
func (ctx *EntityContext) aimDirections(resolved ability.AbilityDef, origin entity.Vec3) []entity.Vec3 {
	if resolved.MultiTargetCount > 1 {
		if targets := NNearestAlivePlayers(ctx.Enemy.Position, ctx.Players, resolved.MultiTargetCount); len(targets) > 0 {
			dirs := make([]entity.Vec3, len(targets))
			for i, p := range targets {
				dirs[i] = p.Position.Sub(origin).Normalized()
			}
			return dirs
		}
	}
	return []entity.Vec3{ctx.Enemy.RangedTargetPos.Sub(origin).Normalized()}
}

// SpawnAdds summons add enemies for a CategorySpawn ability. Placement
// defaults to "behind_players": one add behind each of the N nearest alive
// players, on the far side from this enemy.
func (ctx *EntityContext) SpawnAdds(resolved ability.AbilityDef) {
	if ctx.SpawnAddFn == nil {
		return
	}
	for _, pos := range ctx.spawnPositions(resolved) {
		ctx.SpawnAddFn(resolved.SpawnDefName, pos, ctx.Enemy.ID)
	}
}

// spawnPositions computes clamped, obstacle-free spawn points for an add wave.
func (ctx *EntityContext) spawnPositions(resolved ability.AbilityDef) []entity.Vec3 {
	count := resolved.SpawnCount
	if count <= 0 {
		count = ctx.AlivePlayerCount()
	}
	maxCount := resolved.SpawnCap
	if maxCount <= 0 {
		maxCount = 3
	}
	if count > maxCount {
		count = maxCount
	}
	if count <= 0 {
		return nil
	}

	dist := resolved.SpawnDistance
	if dist <= 0 {
		dist = 4.0
	}
	addRadius := float32(0.8)
	if def := DefRegistry[resolved.SpawnDefName]; def != nil && def.Radius > 0 {
		addRadius = def.Radius
	}

	boss := ctx.Enemy.Position
	positions := make([]entity.Vec3, 0, count)

	if resolved.SpawnPlacement == "at_self" {
		for range count {
			pos := boss
			ctx.clampSpawnPos(&pos, addRadius)
			positions = append(positions, pos)
		}
		return positions
	}

	// "behind_players" (default): behind each player relative to this enemy.
	for _, p := range NNearestAlivePlayers(boss, ctx.Players, count) {
		dir := p.Position.Sub(boss).Flat()
		if dir.Length() < 0.1 {
			dir = entity.Vec3{Z: -1}
		} else {
			dir = dir.Normalized()
		}
		pos := p.Position.Add(dir.Scale(dist))
		ctx.clampSpawnPos(&pos, addRadius)
		positions = append(positions, pos)
	}
	return positions
}

// clampSpawnPos keeps a spawn point inside the zone bounds and outside obstacles.
func (ctx *EntityContext) clampSpawnPos(pos *entity.Vec3, radius float32) {
	pos.X = entity.Clamp(pos.X, ctx.BoundsMinX+radius, ctx.BoundsMaxX-radius)
	pos.Z = entity.Clamp(pos.Z, ctx.BoundsMinZ+radius, ctx.BoundsMaxZ-radius)
	combat.PushOutOfObstacles(pos, ctx.Obs, radius)
}

// EnterCooldown sets the enemy into cooldown state using the GCD.
func (ctx *EntityContext) EnterCooldown() {
	e := ctx.Enemy
	gcd := ctx.Def.CurrentGCD(e.Phase)
	e.State = entity.EnemyCooldown
	e.StateTimer = gcd
	e.Velocity = entity.Vec3{}
}

// --- Runner API (BT interface) ---

// Commit initiates an ability by ID. Returns true if accepted (runner was idle).
func (ctx *EntityContext) Commit(abilityID string) bool {
	return ctx.Runner.Start(ctx, abilityID)
}

// CommitWeighted does weighted random selection then Commit. Returns true if commit started.
func (ctx *EntityContext) CommitWeighted() bool {
	target := ctx.TargetPlayer()
	if target == nil {
		return false
	}
	distance := ctx.Enemy.Position.Flat().DistanceTo(target.Position.Flat())
	chosen := ctx.SelectAbility(distance)
	if chosen == nil {
		return false
	}
	return ctx.Runner.Start(ctx, chosen.ID)
}

// CancelAbility cancels the current ability if in commit phase and cancellable.
func (ctx *EntityContext) CancelAbility() bool {
	return ctx.Runner.Cancel(ctx)
}

// IsRunnerBusy returns true if the runner is in any non-idle phase.
func (ctx *EntityContext) IsRunnerBusy() bool {
	return ctx.Runner.Phase != RunnerIdle
}

// CurrentAbilityID returns the ID of the currently active ability, or "".
func (ctx *EntityContext) CurrentAbilityID() string {
	if ctx.Runner.Phase == RunnerIdle {
		return ""
	}
	abil := ctx.Def.AbilityByIndex(ctx.Runner.AbilIdx)
	if abil == nil {
		return ""
	}
	return abil.ID
}

// --- Movement ---

func (ctx *EntityContext) FaceToward(target entity.Vec3) {
	dir := target.Sub(ctx.Enemy.Position).Flat()
	if dir.Length() > 0.1 {
		ctx.Enemy.RotationY = float32(math.Atan2(float64(-dir.X), float64(-dir.Z)))
	}
}

// bbRetreatSide remembers which way (rotation sign) a pinned kiter is
// escaping, so it commits to one side instead of flip-flopping between the
// mirror-image slide directions each tick.
const bbRetreatSide = "retreat_side"

// RetreatDirection picks a backpedal direction that will not pin the enemy
// against a wall or obstacle. Straight away from the target is preferred;
// when the probe ahead is blocked, rotated candidates are tried and the clear
// one ending farthest from the target wins, so a cornered kiter slides along
// the wall instead of driving into it.

func (ctx *EntityContext) RetreatDirection(away, targetPos entity.Vec3) entity.Vec3 {
	const probeDist = 1.2
	pos := ctx.Enemy.Position
	if sim, ok := ctx.retreatProbe(away, probeDist); ok &&
		sim.Sub(pos).Flat().Length() >= probeDist*0.95 {
		ctx.BB.Delete(bbRetreatSide)
		return away
	}
	// Straight back is pinned. Escaping a corner may require temporarily
	// closing on the target, so feasibility gates first: prefer candidates
	// whose clamped step actually moves the full probe (a clean slide along
	// the wall) over half-clamped diagonals, then rank by distance kept from
	// the target. A committed side wins ties so the escape doesn't oscillate.
	quarter := float32(math.Pi / 4)
	angles := [6]float32{quarter, -quarter, 2 * quarter, -2 * quarter, 3 * quarter, -3 * quarter}
	lockedSide := ctx.BB.GetFloat32(bbRetreatSide)
	best := away
	bestMoved := float32(-1)
	bestDistSq := float32(-1)
	bestSide := float32(0)
	for _, a := range angles {
		cand := combat.RotateVecY(away, a)
		sim, ok := ctx.retreatProbe(cand, probeDist)
		if !ok {
			continue
		}
		moved := sim.Sub(pos).Flat().Length()
		if moved < probeDist*0.6 {
			continue
		}
		side := float32(1)
		if a < 0 {
			side = -1
		}
		distSq := sim.DistanceToSq(targetPos)
		// Side lock acts as a strong bonus, not a hard filter, so a fully
		// blocked side still falls back to the other one.
		score := moved
		if lockedSide != 0 && side == lockedSide {
			score += probeDist
		}
		bestScore := bestMoved
		if lockedSide != 0 && bestSide == lockedSide {
			bestScore += probeDist
		}
		if score > bestScore+0.01 || (score > bestScore-0.01 && distSq > bestDistSq) {
			bestMoved = moved
			bestDistSq = distSq
			bestSide = side
			best = cand
		}
	}
	if bestMoved > 0 {
		ctx.BB.Set(bbRetreatSide, bestSide)
	}
	// All candidates pinned: fully boxed in, the position clamp holds us.
	return best
}

// retreatProbe simulates one movement step clamped to the zone bounds.
// Returns the post-clamp position, and false when it lands inside an obstacle.
func (ctx *EntityContext) retreatProbe(dir entity.Vec3, dist float32) (entity.Vec3, bool) {
	r := ctx.Def.Radius
	probe := ctx.Enemy.Position.Add(dir.Scale(dist))
	probe.X = entity.Clamp(probe.X, ctx.BoundsMinX+r, ctx.BoundsMaxX-r)
	probe.Z = entity.Clamp(probe.Z, ctx.BoundsMinZ+r, ctx.BoundsMaxZ-r)
	if combat.IsAtObstacle(probe, ctx.Obs, r) {
		return probe, false
	}
	return probe, true
}

// AvoidObstacles steers a direction around obstacles between from and to.
func (ctx *EntityContext) AvoidObstacles(dir, from, to entity.Vec3) entity.Vec3 {
	obs, blocked := combat.NearestObstacleOnSegment(from, to, ctx.Obs, ctx.Def.Radius)
	if !blocked {
		return dir
	}
	obstacleCenter := entity.Vec3{X: obs.CX, Z: obs.CZ}
	perpL := entity.Vec3{X: -dir.Z, Z: dir.X}
	perpR := entity.Vec3{X: dir.Z, Z: -dir.X}
	clearance := obs.HX + ctx.Def.Radius + 0.5
	if obs.HZ+ctx.Def.Radius+0.5 > clearance {
		clearance = obs.HZ + ctx.Def.Radius + 0.5
	}
	waypointL := obstacleCenter.Add(perpL.Scale(clearance))
	waypointR := obstacleCenter.Add(perpR.Scale(clearance))
	if waypointL.DistanceToSq(to) < waypointR.DistanceToSq(to) {
		return waypointL.Sub(from).Flat().Normalized()
	}
	return waypointR.Sub(from).Flat().Normalized()
}

// --- Internal helpers ---

func (ctx *EntityContext) fillTargets() {
	ctx.targetBuf = ctx.targetBuf[:0]
	for _, p := range ctx.Players {
		ctx.targetBuf = append(ctx.targetBuf, p)
	}
}

// --- Trace logging ---

// logCond logs a condition evaluation at Debug level. No-op when Logger is nil.
func (ctx *EntityContext) logCond(name string, result bool, extra ...any) {
	if ctx.Logger == nil {
		return
	}
	args := make([]any, 0, 6+len(extra))
	args = append(args, "node", name, "result", result, "enemy", ctx.Enemy.ID)
	args = append(args, extra...)
	ctx.Logger.Debug("bt.cond", args...)
}

// logAction logs an action execution at Debug level. No-op when Logger is nil.
func (ctx *EntityContext) logAction(name string, result bt.Result, extra ...any) {
	if ctx.Logger == nil {
		return
	}
	args := make([]any, 0, 6+len(extra))
	args = append(args, "node", name, "result", result.String(), "enemy", ctx.Enemy.ID)
	args = append(args, extra...)
	ctx.Logger.Debug("bt.action", args...)
}
