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
	Enemy         *entity.Enemy
	Def           *EnemyDef
	Engine        *ability.Engine
	BB            *Blackboard
	Rng           *rand.Rand
	Dt            float32
	Players       []*entity.Player
	Obs           []combat.Obstacle
	SpawnFn       func(pos, dir entity.Vec3, speed, damage, lifetime float32)
	CommitPatternFn func(pattern *combat.PatternDef, abilityName string, origin, facing entity.Vec3)
	Events        *[]combat.DamageEvent

	// Runner owns the ability commit→execute→cooldown lifecycle.
	Runner *AbilityRunner

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
	commitCtx   ability.CommitContext
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

	switch abil.Category {
	case ability.CategoryMelee:
		e.State = entity.EnemyMeleeTelegraph
		e.StateTimer = resolved.CommitTime
		e.MeleeRange = resolved.Hit.Range
		arcDeg := resolved.Hit.ArcDegrees
		if arcDeg <= 0 {
			arcDeg = 180
		}
		e.MeleeConeAngle = arcDeg * math.Pi / 180.0
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
	baseDir := e.RangedTargetPos.Sub(origin).Normalized()

	// Pattern engine path: multi-wave bullet-hell patterns
	if resolved.Pattern != nil && ctx.CommitPatternFn != nil {
		ctx.CommitPatternFn(resolved.Pattern, resolved.Name, origin, baseDir)
		return
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
