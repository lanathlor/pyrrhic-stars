package enemyai

import (
	"math"
	"math/rand"

	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
)

// Brain drives one enemy instance using its EnemyDef.
type Brain struct {
	Def   *EnemyDef
	enemy *entity.Enemy

	// Bounds for charge wall-stop detection. Set by zone at init.
	BoundsMinX, BoundsMaxX float32
	BoundsMinZ, BoundsMaxZ float32
}

// NewBrain creates a brain for the given enemy.
func NewBrain(def *EnemyDef, enemy *entity.Enemy) *Brain {
	return &Brain{Def: def, enemy: enemy}
}

// Enemy returns the brain's enemy.
func (b *Brain) Enemy() *entity.Enemy { return b.enemy }

// Tick advances the AI by dt seconds. Returns damage events to emit.
func (b *Brain) Tick(dt float32, players map[uint16]*entity.Player, obstacles []combat.Obstacle, spawnProjectile func(pos, dir entity.Vec3, speed, damage, lifetime float32)) []combat.DamageEvent {
	e := b.enemy

	e.StateTimer -= dt

	var events []combat.DamageEvent

	switch e.State {
	case entity.EnemyChase:
		b.tickChase(dt, players, obstacles)
	case entity.EnemyMeleeTelegraph:
		b.tickMeleeTelegraph(players)
	case entity.EnemyMeleeAttack:
		events = b.tickMeleeAttack(players, obstacles)
	case entity.EnemyRangedTelegraph:
		b.tickRangedTelegraph(players)
	case entity.EnemyRangedAttack:
		b.tickRangedAttack(spawnProjectile)
	case entity.EnemyAoETelegraph:
		b.tickAoETelegraph()
	case entity.EnemyAoESlam:
		events = b.tickAoESlam(players, obstacles)
	case entity.EnemyChargeTelegraph:
		b.tickChargeTelegraph(players)
	case entity.EnemyCharge:
		events = b.tickCharge(dt, players, obstacles)
	case entity.EnemyCooldown:
		b.tickCooldown()
	case entity.EnemyPhaseTransition:
		b.tickPhaseTransition()
	case entity.EnemyDead:
		e.Velocity = entity.Vec3{}
	case entity.EnemyPatrol:
		b.tickPatrol(dt, players)
	}

	// Apply velocity
	e.Position = e.Position.Add(e.Velocity.Scale(dt))

	return events
}

// --- State handlers ---

func (b *Brain) tickChase(dt float32, players map[uint16]*entity.Player, obstacles []combat.Obstacle) {
	e := b.enemy
	def := b.Def

	// Leash check: if mob is too far from spawn, reset to patrol
	if e.LeashRadius > 0 && b.checkLeash() {
		return
	}

	e.ChaseTimer += dt

	target := NearestAlivePlayer(e.Position, players)
	if target == nil {
		e.Velocity = entity.Vec3{}
		return
	}

	e.TargetPlayerID = target.PeerID
	toTarget := target.Position.Sub(e.Position).Flat()
	distance := toTarget.Length()

	if distance > 0.1 {
		dir := toTarget.Normalized()
		e.RotationY = float32(math.Atan2(float64(-dir.X), float64(-dir.Z)))
	}

	hasLoS := !combat.SegmentHitsExpandedObstacle(e.Position, target.Position, obstacles, def.Radius)

	meleeRange := def.LongestMeleeRange()

	// In melee range → attack (only if LoS)
	if distance <= meleeRange && hasLoS {
		ability := b.selectAbility(distance, players)
		if ability != nil {
			// Don't use ranged or charge at melee range — substitute
			if ability.Type == AbilityRanged {
				ability = b.findAbilityByType(AbilityMelee)
			}
			if ability.Type == AbilityCharge {
				ability = b.findAbilityByType(AbilityAoE)
				if ability == nil {
					ability = b.findAbilityByType(AbilityMelee)
				}
			}
			if ability != nil {
				b.startAbility(ability, e, players, obstacles)
			}
		}
		return
	}

	// Chase timer threshold (only attack if LoS)
	// Phase can override chase thresholds for more aggressive behavior
	phase := def.CurrentPhase(e.Phase)
	chaseThreshold := def.ChaseThreshold
	chaseThresholdFar := def.ChaseThresholdFar
	if phase != nil {
		if phase.ChaseThreshold > 0 {
			chaseThreshold = phase.ChaseThreshold
		}
		if phase.ChaseThresholdFar > 0 {
			chaseThresholdFar = phase.ChaseThresholdFar
		}
	}
	if distance > meleeRange*def.FarDistanceMultiplier {
		chaseThreshold = chaseThresholdFar
	}
	if e.ChaseTimer >= chaseThreshold && hasLoS {
		ability := b.selectAbility(distance, players)
		if ability != nil {
			// Can't melee from far — try substitutes
			if ability.Type == AbilityMelee && distance > meleeRange {
				if distance > meleeRange*2.0 {
					ability = b.findAbilityByType(AbilityCharge)
				} else {
					ability = b.findAbilityByType(AbilityRanged)
				}
			}
			// AoE useless at long range
			if ability != nil {
				resolved := b.Def.ResolveAbility(ability, e.Phase)
				if ability.Type == AbilityAoE && distance > resolved.AoERadius*1.5 {
					ability = b.findAbilityByType(AbilityCharge)
				}
			}
			if ability != nil {
				b.startAbility(ability, e, players, obstacles)
				return
			}
		}
		// No usable ability at this range — keep chasing (fall through to movement)
	}

	// Movement: chase, hold position, or backpedal depending on PreferredRange
	spd := def.CurrentMoveSpeed(e.Phase)
	preferred := def.PreferredRange
	if preferred <= 0 {
		// Classic chase — close distance until in melee range
		if distance > meleeRange*0.8 {
			dir := b.avoidObstacles(toTarget.Normalized(), e.Position, target.Position, obstacles)
			e.Velocity = entity.Vec3{X: dir.X * spd, Z: dir.Z * spd}
		} else {
			e.Velocity = entity.Vec3{}
		}
	} else {
		// Ranged — maintain preferred distance, fire while moving
		margin := preferred * 0.2 // dead zone to avoid jitter
		if distance < preferred-margin {
			// Too close — backpedal (slower, catchable)
			bspd := def.CurrentBackpedalSpeed(e.Phase)
			dir := toTarget.Normalized().Neg().Flat()
			e.Velocity = entity.Vec3{X: dir.X * bspd, Z: dir.Z * bspd}
			// Fire while backpedaling if chase timer expired
			if e.ChaseTimer >= chaseThreshold && hasLoS {
				ability := b.selectAbility(distance, players)
				if ability != nil {
					b.startAbility(ability, e, players, obstacles)
				}
			}
		} else if distance > preferred+margin {
			// Too far — close in at full speed
			dir := b.avoidObstacles(toTarget.Normalized(), e.Position, target.Position, obstacles)
			e.Velocity = entity.Vec3{X: dir.X * spd, Z: dir.Z * spd}
		} else {
			// In sweet spot — hold position
			e.Velocity = entity.Vec3{}
		}
	}
}

func (b *Brain) tickMeleeTelegraph(players map[uint16]*entity.Player) {
	e := b.enemy
	e.Velocity = entity.Vec3{}
	// No rotation during telegraph — player can reposition to dodge
	if e.StateTimer <= 0 {
		e.State = entity.EnemyMeleeAttack
		e.StateTimer = 0.3
	}
}

func (b *Brain) tickMeleeAttack(players map[uint16]*entity.Player, obstacles []combat.Obstacle) []combat.DamageEvent {
	e := b.enemy
	e.Velocity = entity.Vec3{}
	if e.StateTimer > 0 {
		return nil
	}

	ability := b.activeAbilityResolved()

	// Frontal cone: enemy's facing direction from RotationY (locked during telegraph).
	// Default cone = 180° (π rad) if not specified.
	coneAngle := ability.MeleeConeAngle
	if coneAngle <= 0 {
		coneAngle = math.Pi // 180° default
	}
	halfAngle := float64(coneAngle) / 2.0
	// Forward direction from RotationY: atan2(-x, -z) → forward = (-sinR, 0, -cosR)
	forwardX := -math.Sin(float64(e.RotationY))
	forwardZ := -math.Cos(float64(e.RotationY))

	var events []combat.DamageEvent
	for _, p := range players {
		if !p.Alive {
			continue
		}
		toPlayer := p.Position.Sub(e.Position).Flat()
		dist := toPlayer.Length()
		if dist > ability.MeleeRange || dist < 0.01 {
			continue
		}
		// Cone check: angle between forward and direction to player
		dx := float64(toPlayer.X) / float64(dist)
		dz := float64(toPlayer.Z) / float64(dist)
		dot := forwardX*dx + forwardZ*dz
		if dot < math.Cos(halfAngle) {
			continue
		}
		if combat.SegmentHitsObstacle(e.Position, p.Position, obstacles) {
			continue
		}
		dealt := p.ApplyDamage(ability.MeleeDamage)
		if dealt > 0 {
			hitDir := toPlayer.Normalized()
			events = append(events, combat.DamageEvent{
				TargetPeerID: p.PeerID,
				Amount:       dealt,
				HitPos:       e.Position.Add(hitDir),
				SourceType:   ability.DamageSourceType,
			})
		}
	}
	b.enterCooldown()
	return events
}

func (b *Brain) tickRangedTelegraph(players map[uint16]*entity.Player) {
	e := b.enemy
	e.Velocity = entity.Vec3{}
	if target, ok := players[e.TargetPlayerID]; ok && target.Alive {
		e.RangedTargetPos = target.Position.Add(entity.Vec3{Y: 1.0})
	}
	if e.StateTimer <= 0 {
		e.State = entity.EnemyRangedAttack
		e.StateTimer = 0.1
	}
}

func (b *Brain) tickRangedAttack(spawnProjectile func(pos, dir entity.Vec3, speed, damage, lifetime float32)) {
	e := b.enemy
	if e.StateTimer > 0 {
		return
	}

	ability := b.activeAbilityResolved()
	baseDir := e.RangedTargetPos.Sub(e.Position.Add(entity.Vec3{Y: ability.ProjectileOriginY})).Normalized()

	count := ability.ProjectileCount
	for i := 0; i < count; i++ {
		offset := (float32(i) - float32(count-1)/2.0) * ability.ProjectileSpread
		dir := combat.RotateVecY(baseDir, offset)
		spawnProjectile(
			e.Position.Add(entity.Vec3{Y: ability.ProjectileOriginY}),
			dir,
			ability.ProjectileSpeed,
			ability.ProjectileDamage,
			ability.ProjectileLifetime,
		)
	}
	b.enterCooldown()
}

func (b *Brain) tickAoETelegraph() {
	e := b.enemy
	e.Velocity = entity.Vec3{}
	if e.StateTimer <= 0 {
		e.State = entity.EnemyAoESlam
		e.StateTimer = 0.1
	}
}

func (b *Brain) tickAoESlam(players map[uint16]*entity.Player, obstacles []combat.Obstacle) []combat.DamageEvent {
	e := b.enemy
	e.Velocity = entity.Vec3{}
	if e.StateTimer > 0 {
		return nil
	}

	ability := b.activeAbilityResolved()
	var events []combat.DamageEvent
	for _, p := range players {
		if !p.Alive {
			continue
		}
		if combat.CheckAoERadius(e.Position, p.Position, ability.AoERadius, obstacles) {
			dealt := p.ApplyDamage(ability.AoEDamage)
			if dealt > 0 {
				events = append(events, combat.DamageEvent{
					TargetPeerID: p.PeerID,
					Amount:       dealt,
					HitPos:       e.Position,
					SourceType:   ability.DamageSourceType,
				})
			}
		}
	}
	b.enterCooldown()
	return events
}

func (b *Brain) tickChargeTelegraph(players map[uint16]*entity.Player) {
	e := b.enemy
	e.Velocity = entity.Vec3{}
	b.faceTarget(players)
	if target, ok := players[e.TargetPlayerID]; ok && target.Alive {
		dir := target.Position.Sub(e.Position).Flat()
		if dir.Length() > 0.1 {
			e.ChargeDirection = dir.Normalized()
		}
	}
	if e.StateTimer <= 0 {
		e.State = entity.EnemyCharge
		e.ChargeDistance = 0
		e.ChargeHitPlayers = []uint16{}
		if e.ChargeDirection.LengthSq() < 0.01 {
			e.ChargeDirection = entity.Vec3{Z: -1}
		}
	}
}

func (b *Brain) tickCharge(dt float32, players map[uint16]*entity.Player, obstacles []combat.Obstacle) []combat.DamageEvent {
	e := b.enemy
	ability := b.activeAbilityResolved()
	spd := ability.ChargeSpeed
	e.Velocity = entity.Vec3{X: e.ChargeDirection.X * spd, Z: e.ChargeDirection.Z * spd}
	e.ChargeDistance += spd * dt

	var events []combat.DamageEvent
	for _, p := range players {
		if !p.Alive || b.isChargeHit(p.PeerID) {
			continue
		}
		if e.Position.DistanceTo(p.Position) <= ability.ChargeHitRadius {
			dealt := p.ApplyDamage(ability.ChargeDamage)
			if dealt > 0 {
				events = append(events, combat.DamageEvent{
					TargetPeerID: p.PeerID,
					Amount:       dealt,
					HitPos:       e.Position,
					SourceType:   ability.DamageSourceType,
				})
			}
			e.ChargeHitPlayers = append(e.ChargeHitPlayers, p.PeerID)
		}
	}

	// Stop conditions
	stop := e.ChargeDistance >= ability.ChargeMaxDistance
	if ability.ChargeStopOnWall {
		stop = stop || combat.IsAtWall(e.Position,
			b.BoundsMinX, b.BoundsMaxX,
			b.BoundsMinZ, b.BoundsMaxZ)
	}
	if ability.ChargeStopOnObstacle {
		stop = stop || combat.IsAtObstacle(e.Position, obstacles, b.Def.Radius)
	}
	if stop {
		e.Velocity = entity.Vec3{}
		b.enterCooldown()
	}
	return events
}

func (b *Brain) tickCooldown() {
	e := b.enemy
	e.Velocity = entity.Vec3{}
	if e.StateTimer <= 0 {
		e.ChaseTimer = 0
		e.State = entity.EnemyChase
	}
}

func (b *Brain) tickPhaseTransition() {
	e := b.enemy
	e.Velocity = entity.Vec3{}
	if e.StateTimer <= 0 {
		e.State = entity.EnemyChase
	}
}

// --- Helpers ---

func (b *Brain) selectAbility(distance float32, players map[uint16]*entity.Player) *AbilityDef {
	e := b.enemy
	def := b.Def
	phase := def.CurrentPhase(e.Phase)

	type candidate struct {
		ability *AbilityDef
		weight  int
	}

	var candidates []candidate
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
			if w, ok := phase.WeightOverrides[a.Name]; ok {
				weight = w
			}
		}

		// Anti-repeat
		if a.Name == e.LastAttack && weight > 1 && def.AntiRepeat > 0 {
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
	roll := rand.Intn(total)
	cumulative := 0
	for _, c := range candidates {
		cumulative += c.weight
		if roll < cumulative {
			return c.ability
		}
	}
	return candidates[0].ability
}

func (b *Brain) startAbility(ability *AbilityDef, e *entity.Enemy, players map[uint16]*entity.Player, obstacles []combat.Obstacle) {
	resolved := b.Def.ResolveAbility(ability, e.Phase)
	e.ActiveAbility = b.abilityIndex(ability)
	e.LastAttack = ability.Name
	e.Velocity = entity.Vec3{}

	switch ability.Type {
	case AbilityMelee:
		e.State = entity.EnemyMeleeTelegraph
		e.StateTimer = resolved.TelegraphTime
		e.MeleeRange = resolved.MeleeRange
		coneAngle := resolved.MeleeConeAngle
		if coneAngle <= 0 {
			coneAngle = math.Pi // default 180°
		}
		e.MeleeConeAngle = coneAngle
	case AbilityRanged:
		e.State = entity.EnemyRangedTelegraph
		e.StateTimer = resolved.TelegraphTime
		// Set ranged target
		target := FarthestAlivePlayer(e.Position, players)
		if target != nil {
			e.TargetPlayerID = target.PeerID
			e.RangedTargetPos = target.Position.Add(entity.Vec3{Y: 1.0})
		}
	case AbilityAoE:
		e.State = entity.EnemyAoETelegraph
		e.StateTimer = resolved.TelegraphTime
	case AbilityCharge:
		e.State = entity.EnemyChargeTelegraph
		e.StateTimer = resolved.TelegraphTime
		e.ChargeDirection = entity.Vec3{}
	}
}

func (b *Brain) enterCooldown() {
	e := b.enemy
	ability := b.Def.AbilityByIndex(e.ActiveAbility)
	cooldown := b.Def.CurrentCooldownTime(ability, e.Phase)
	e.State = entity.EnemyCooldown
	e.StateTimer = cooldown
	e.Velocity = entity.Vec3{}
}

func (b *Brain) activeAbilityResolved() AbilityDef {
	ability := b.Def.AbilityByIndex(b.enemy.ActiveAbility)
	if ability == nil {
		return AbilityDef{}
	}
	return b.Def.ResolveAbility(ability, b.enemy.Phase)
}

func (b *Brain) abilityIndex(a *AbilityDef) int {
	for i := range b.Def.Abilities {
		if &b.Def.Abilities[i] == a {
			return i
		}
	}
	return 0
}

func (b *Brain) findAbilityByType(t AbilityType) *AbilityDef {
	for i := range b.Def.Abilities {
		if b.Def.Abilities[i].Type == t {
			return &b.Def.Abilities[i]
		}
	}
	return nil
}

func (b *Brain) faceTarget(players map[uint16]*entity.Player) {
	e := b.enemy
	target, ok := players[e.TargetPlayerID]
	if !ok || !target.Alive {
		return
	}
	dir := target.Position.Sub(e.Position).Flat()
	if dir.Length() > 0.1 {
		e.RotationY = float32(math.Atan2(float64(-dir.X), float64(-dir.Z)))
	}
}

// avoidObstacles steers a movement direction around obstacles between the
// enemy and the target. Returns the adjusted direction (flat, normalized).
func (b *Brain) avoidObstacles(dir entity.Vec3, from, to entity.Vec3, obstacles []combat.Obstacle) entity.Vec3 {
	obs, blocked := combat.NearestObstacleOnSegment(from, to, obstacles, b.Def.Radius)
	if !blocked {
		return dir
	}
	obstacleCenter := entity.Vec3{X: obs.CX, Z: obs.CZ}
	perpL := entity.Vec3{X: -dir.Z, Z: dir.X}
	perpR := entity.Vec3{X: dir.Z, Z: -dir.X}
	clearance := obs.HX + b.Def.Radius + 0.5
	if obs.HZ+b.Def.Radius+0.5 > clearance {
		clearance = obs.HZ + b.Def.Radius + 0.5
	}
	waypointL := obstacleCenter.Add(perpL.Scale(clearance))
	waypointR := obstacleCenter.Add(perpR.Scale(clearance))
	if waypointL.DistanceToSq(to) < waypointR.DistanceToSq(to) {
		return waypointL.Sub(from).Flat().Normalized()
	}
	return waypointR.Sub(from).Flat().Normalized()
}

func (b *Brain) isChargeHit(peerID uint16) bool {
	for _, hid := range b.enemy.ChargeHitPlayers {
		if hid == peerID {
			return true
		}
	}
	return false
}

// tickPatrol walks between PatrolA/PatrolB at half speed, checking for player aggro.
func (b *Brain) tickPatrol(dt float32, players map[uint16]*entity.Player) {
	e := b.enemy

	// Check aggro: if any alive player within AggroRadius, switch to Chase
	for _, p := range players {
		if !p.Alive {
			continue
		}
		distSq := e.Position.Flat().DistanceToSq(p.Position.Flat())
		if distSq <= e.AggroRadius*e.AggroRadius {
			e.TargetPlayerID = p.PeerID
			e.State = entity.EnemyChase
			e.ChaseTimer = 0
			return
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
		// Reached waypoint, flip target
		e.PatrolTarget = 1 - e.PatrolTarget
		e.Velocity = entity.Vec3{}
		return
	}
	dir := toTarget.Normalized()
	spd := b.Def.MoveSpeed * 0.5 // patrol at half speed
	e.Velocity = entity.Vec3{X: dir.X * spd, Z: dir.Z * spd}
	e.RotationY = float32(math.Atan2(float64(-dir.X), float64(-dir.Z)))
}

// checkLeash resets the enemy to patrol if too far from spawn. Returns true if leashed.
func (b *Brain) checkLeash() bool {
	e := b.enemy
	if e.LeashRadius <= 0 {
		return false
	}
	distSq := e.Position.Flat().DistanceToSq(e.LeashOrigin.Flat())
	if distSq > e.LeashRadius*e.LeashRadius {
		e.Position = e.LeashOrigin
		e.Health = e.MaxHealth
		e.State = entity.EnemyPatrol
		e.Velocity = entity.Vec3{}
		e.ChaseTimer = 0
		e.ThreatTable = make(map[uint16]float32)
		return true
	}
	return false
}
