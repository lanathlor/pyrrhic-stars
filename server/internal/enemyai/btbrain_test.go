package enemyai

import (
	"math"
	"testing"

	"codex-online/server/internal/ability"
	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
)

func testBrain(def *EnemyDef) (*Brain, *entity.Enemy) {
	e := entity.NewEnemy(0, def.MaxHealth, def.Name)
	e.State = entity.EnemyChase
	e.Position = entity.Vec3{X: 0, Y: 0.1, Z: 0}
	eng := ability.NewEngine(nil)
	b := NewBrainSeeded(def, e, eng, 42)
	b.BoundsMinX = -20
	b.BoundsMaxX = 20
	b.BoundsMinZ = -15
	b.BoundsMaxZ = 50
	return b, e
}

func noSpawn(_, _ entity.Vec3, _, _, _ float32) {}

// TestBrain_DeadStopsVelocity verifies dead enemy zeroes velocity.
func TestBrain_DeadStopsVelocity(t *testing.T) {
	def := testDef()
	b, e := testBrain(def)
	e.Alive = false
	e.Velocity = entity.Vec3{X: 5, Z: 5}

	b.Tick(0.05, testPlayers(), nil, noSpawn, nil)
	if e.Velocity.X != 0 || e.Velocity.Z != 0 {
		t.Errorf("velocity should be zero, got %v", e.Velocity)
	}
}

// TestBrain_PhaseTransitionWaits verifies phase transition returns Running.
func TestBrain_PhaseTransitionWaits(t *testing.T) {
	def := testDef()
	b, e := testBrain(def)
	e.State = entity.EnemyPhaseTransition
	e.StateTimer = 1.0

	b.Tick(0.5, testPlayers(), nil, noSpawn, nil)
	if e.State != entity.EnemyPhaseTransition {
		t.Errorf("should still be transitioning, got %d", e.State)
	}

	// Tick past transition
	b.Tick(0.6, testPlayers(), nil, noSpawn, nil)
	if e.State != entity.EnemyChase {
		t.Errorf("should be chase after transition, got %d", e.State)
	}
}

// TestBrain_ChasesTowardTarget verifies the enemy moves toward the target.
func TestBrain_ChasesTowardTarget(t *testing.T) {
	def := testDef()
	b, e := testBrain(def)
	e.State = entity.EnemyChase
	e.Alive = true
	// Set an existing target so condHasTarget passes
	p := testPlayer(1, entity.Vec3{X: 0, Z: 20})
	players := testPlayers(p)
	e.TargetPlayerID = p.ID

	b.Tick(0.05, players, nil, noSpawn, nil)

	if e.Velocity.Z <= 0 {
		t.Errorf("should move toward target (Z+), got velocity %v", e.Velocity)
	}
}

// TestBrain_MeleeAttackAtRange verifies the enemy attacks when in melee range.
func TestBrain_MeleeAttackAtRange(t *testing.T) {
	def := testDef()
	b, e := testBrain(def)
	e.Alive = true
	e.State = entity.EnemyChase

	p := testPlayer(1, entity.Vec3{X: 0, Z: 2})
	players := testPlayers(p)
	e.TargetPlayerID = p.ID

	b.Tick(0.05, players, nil, noSpawn, nil)

	// Should have transitioned to a telegraph state
	if e.State != entity.EnemyMeleeTelegraph && e.State != entity.EnemyAoETelegraph {
		t.Errorf("should be in telegraph, got state %d", e.State)
	}
}

// TestBrain_FullMeleeCycle verifies the complete attack lifecycle.
func TestBrain_FullMeleeCycle(t *testing.T) {
	def := &EnemyDef{
		Name:      "test_melee_only",
		MaxHealth: 500,
		MoveSpeed: 4.0,
		Radius:    1.0,
		TreeData:  testTreeData(),
		Abilities: []AbilityDef{
			{
				Name: "melee", Type: AbilityMelee,
				TelegraphTime: 0.5, CooldownTime: 0.5,
				BaseWeight: 100, MaxRange: 3.0,
				MeleeRange: 3.0, MeleeDamage: 30.0,
				MeleeConeAngle:   math.Pi,
				DamageSourceType: SourceEnemyMelee,
			},
		},
	}
	b, e := testBrain(def)
	e.Alive = true
	e.State = entity.EnemyChase

	p := testPlayer(1, entity.Vec3{X: 0, Z: 2})
	players := testPlayers(p)
	e.TargetPlayerID = p.ID

	// Tick 1: select + telegraph starts
	b.Tick(0.05, players, nil, noSpawn, nil)
	if e.State != entity.EnemyMeleeTelegraph {
		t.Fatalf("tick 1: expected MeleeTelegraph, got %d", e.State)
	}

	// Tick through telegraph (0.5s) + execute (0.3s) + cooldown (0.5s) = ~1.3s
	// Need ~26 ticks at 0.05s each after the first tick
	for range 30 {
		b.Tick(0.05, players, nil, noSpawn, nil)
	}

	// After ~1.55s total, should be back to chase or starting a new attack
	if e.State == entity.EnemyCooldown || e.State == entity.EnemyMeleeAttack {
		// Still finishing up, tick a few more
		for range 10 {
			b.Tick(0.05, players, nil, noSpawn, nil)
		}
	}

	// By now (2.05s), the cycle should have completed
	validStates := e.State == entity.EnemyChase || e.State == entity.EnemyMeleeTelegraph
	if !validStates {
		t.Errorf("expected Chase or new telegraph, got state %d", e.State)
	}
}

// TestBrain_PatrolAndAggro verifies patrol behavior and aggro detection.
func TestBrain_PatrolAndAggro(t *testing.T) {
	def := &EnemyDef{
		Name:      "test_patrol",
		MaxHealth: 200,
		MoveSpeed: 4.0,
		Radius:    1.0,
		TreeData:  testTreeData(),
		Abilities: []AbilityDef{
			{
				Name: "melee", Type: AbilityMelee,
				TelegraphTime: 0.5, CooldownTime: 0.5,
				BaseWeight: 100, MaxRange: 3.0,
				MeleeRange: 3.0, MeleeDamage: 10.0,
			},
		},
	}
	b, e := testBrain(def)
	e.Alive = true
	e.State = entity.EnemyPatrol
	e.PatrolA = entity.Vec3{X: -5, Z: 0}
	e.PatrolB = entity.Vec3{X: 5, Z: 0}
	e.AggroRadius = 6.0
	// Start slightly offset from PatrolA so we don't flip immediately
	e.Position = entity.Vec3{X: -4, Z: 0}

	// No players nearby — should patrol
	b.Tick(0.05, testPlayers(), nil, noSpawn, nil)
	// PatrolA is at X=-5 and we're at X=-4, so we should move toward PatrolA (X-)
	// OR if already past the waypoint threshold, move toward PatrolB
	if e.Velocity.X == 0 && e.Velocity.Z == 0 {
		t.Error("should be patrolling, got zero velocity")
	}

	// Add player within aggro range
	p := testPlayer(1, entity.Vec3{X: -3, Z: 0})
	players := testPlayers(p)
	b.Tick(0.05, players, nil, noSpawn, nil)
	if e.State != entity.EnemyChase {
		t.Errorf("should aggro to chase, got state %d", e.State)
	}
	if e.TargetPlayerID != p.ID {
		t.Errorf("should target player %d, got %d", p.ID, e.TargetPlayerID)
	}
}

// TestBrain_LeashReset verifies leash behavior.
func TestBrain_LeashReset(t *testing.T) {
	def := testDef()
	b, e := testBrain(def)
	e.Alive = true
	e.State = entity.EnemyChase
	e.LeashOrigin = entity.Vec3{X: 0, Z: 0}
	e.LeashRadius = 10.0
	e.Position = entity.Vec3{X: 15, Z: 0}

	p := testPlayer(1, entity.Vec3{X: 20, Z: 0})
	e.TargetPlayerID = p.ID
	players := testPlayers(p)

	b.Tick(0.05, players, nil, noSpawn, nil)
	if e.State != entity.EnemyPatrol {
		t.Errorf("should leash to patrol, got state %d", e.State)
	}
	if e.Health != e.MaxHealth {
		t.Errorf("should heal to full on leash, got %f", e.Health)
	}
}

// TestBrain_RangedSpawnsProjectile verifies ranged attacks spawn projectiles.
func TestBrain_RangedSpawnsProjectile(t *testing.T) {
	// Use "hallway_ranged" name to get the ranged tree
	def := &EnemyDef{
		Name:      "hallway_ranged",
		MaxHealth: 200,
		MoveSpeed: 3.0,
		Radius:    1.0,
		Abilities: []AbilityDef{
			{
				Name: "bolt", Type: AbilityRanged,
				TelegraphTime: 0.2, CooldownTime: 0.5,
				BaseWeight:      100,
				ProjectileCount: 1, ProjectileSpeed: 20.0,
				ProjectileDamage: 10.0, ProjectileOriginY: 1.5,
				ProjectileLifetime: 3.0,
				DamageSourceType:   SourceEnemyRanged,
			},
		},
		PreferredRange: 8.0,
		BackpedalSpeed: 2.0,
	}
	b, e := testBrain(def)
	e.Alive = true
	e.State = entity.EnemyChase

	p := testPlayer(1, entity.Vec3{X: 0, Z: 10})
	players := testPlayers(p)
	e.TargetPlayerID = p.ID

	projectiles := 0
	spawnFn := func(_, _ entity.Vec3, _, _, _ float32) { projectiles++ }

	// Reactive selector re-evaluates attack branch each tick — should fire quickly with LoS
	for range 80 {
		b.Tick(0.05, players, nil, spawnFn, nil)
		if projectiles > 0 {
			break
		}
	}

	if projectiles == 0 {
		t.Error("should have spawned at least one projectile within 4 seconds")
	}
}

// TestBrain_ChargeHitsPlayer verifies charge damage on contact.
// Uses "guard_captain" name to get the boss tree which has chase-timer-ready attacks.
func TestBrain_ChargeHitsPlayer(t *testing.T) {
	def := &EnemyDef{
		Name:      "guard_captain",
		MaxHealth: 500,
		MoveSpeed: 4.0,
		Radius:    1.0,
		Abilities: []AbilityDef{
			{
				Name: "melee", Type: AbilityMelee,
				TelegraphTime: 0.3, CooldownTime: 0.5,
				BaseWeight: 10, MaxRange: 3.0,
				MeleeRange: 3.0, MeleeDamage: 15.0,
				MeleeConeAngle:   math.Pi,
				DamageSourceType: SourceEnemyMelee,
			},
			{
				Name: "charge", Type: AbilityCharge,
				TelegraphTime: 0.2, CooldownTime: 0.5,
				BaseWeight: 90, MinRange: 4.0,
				FaceTarget:  true,
				ChargeSpeed: 12.0, ChargeDamage: 35.0,
				ChargeMaxDistance: 20.0, ChargeHitRadius: 2.0,
				ChargeStopOnWall: true,
				DamageSourceType: SourceEnemyCharge,
			},
		},
	}
	b, e := testBrain(def)
	e.Alive = true
	e.State = entity.EnemyChase
	e.Position = entity.Vec3{X: 0, Z: 0}

	p := testPlayer(1, entity.Vec3{X: 0, Z: 10})
	p.Health = 100
	players := testPlayers(p)
	e.TargetPlayerID = p.ID

	var allEvents []combat.DamageEvent
	for range 200 {
		events := b.Tick(0.05, players, nil, noSpawn, nil)
		allEvents = append(allEvents, events...)
		if p.Health < 100 {
			break
		}
	}

	if p.Health >= 100 {
		t.Errorf("player should have taken charge damage, health=%f, events=%d", p.Health, len(allEvents))
	}
}

// TestBrain_HallwayMeleeTree verifies the named tree is built correctly.
func TestBrain_HallwayMeleeTree(t *testing.T) {
	b, e := testBrain(DefRegistry["hallway_melee"])
	e.Alive = true
	e.State = entity.EnemyPatrol
	e.PatrolA = entity.Vec3{X: -5}
	e.PatrolB = entity.Vec3{X: 5}
	e.AggroRadius = 5.0
	// Start between patrol points
	e.Position = entity.Vec3{X: 0}

	startX := e.Position.X
	for range 20 {
		b.Tick(0.05, testPlayers(), nil, noSpawn, nil)
	}
	if e.Position.X == startX {
		t.Errorf("should have moved during patrol, still at %f", e.Position.X)
	}
}

// TestBrain_MeleeCommitsDirection verifies that melee attacks with FaceTarget=true
// and TrackTarget=false commit to a direction at telegraph start and don't track.
func TestBrain_MeleeCommitsDirection(t *testing.T) {
	def := &EnemyDef{
		Name:      "test_commit",
		MaxHealth: 500,
		MoveSpeed: 4.0,
		Radius:    1.0,
		TreeData:  testTreeData(),
		Abilities: []AbilityDef{
			{
				Name: "melee", Type: AbilityMelee,
				TelegraphTime: 1.0, CooldownTime: 0.5,
				BaseWeight: 100, MaxRange: 5.0,
				FaceTarget:  true,
				TrackTarget: false,
				MeleeRange:  5.0, MeleeDamage: 10.0,
				MeleeConeAngle: math.Pi,
			},
		},
	}
	b, e := testBrain(def)
	e.Alive = true
	e.State = entity.EnemyChase

	p := testPlayer(1, entity.Vec3{X: 0, Z: 2})
	players := testPlayers(p)
	e.TargetPlayerID = p.ID

	// Tick until telegraph starts
	for range 100 {
		b.Tick(0.05, players, nil, noSpawn, nil)
		if e.State == entity.EnemyMeleeTelegraph {
			break
		}
	}
	if e.State != entity.EnemyMeleeTelegraph {
		t.Fatalf("expected MeleeTelegraph, got %d", e.State)
	}

	// Record the committed rotation
	committedYaw := e.RotationY

	// Move the player 90° to the side during telegraph
	p.Position = entity.Vec3{X: 10, Z: 0}

	// Tick through remaining telegraph — rotation should NOT change
	for range 20 {
		if e.State != entity.EnemyMeleeTelegraph {
			break
		}
		b.Tick(0.05, players, nil, noSpawn, nil)
	}

	if e.RotationY != committedYaw {
		t.Errorf("melee should commit direction: rotation changed from %f to %f", committedYaw, e.RotationY)
	}
}

// TestBrain_RangedTracksTarget verifies that ranged attacks with TrackTarget=true
// keep updating the target position during telegraph.
func TestBrain_RangedTracksTarget(t *testing.T) {
	def := &EnemyDef{
		Name:           "hallway_ranged",
		MaxHealth:      200,
		MoveSpeed:      3.0,
		Radius:         1.0,
		PreferredRange: 8.0,
		BackpedalSpeed: 2.0,
		Abilities: []AbilityDef{
			{
				Name: "bolt", Type: AbilityRanged,
				TelegraphTime: 1.0, CooldownTime: 0.5,
				BaseWeight:      100,
				TrackTarget:     true,
				ProjectileCount: 1, ProjectileSpeed: 20.0,
				ProjectileDamage: 10.0, ProjectileOriginY: 1.5,
				ProjectileLifetime: 3.0,
			},
		},
	}
	b, e := testBrain(def)
	e.Alive = true
	e.State = entity.EnemyChase

	p := testPlayer(1, entity.Vec3{X: 0, Z: 10})
	players := testPlayers(p)
	e.TargetPlayerID = p.ID

	// Tick until ranged telegraph starts
	for range 400 {
		b.Tick(0.05, players, nil, noSpawn, nil)
		if e.State == entity.EnemyRangedTelegraph {
			break
		}
	}
	if e.State != entity.EnemyRangedTelegraph {
		t.Fatalf("expected RangedTelegraph, got %d", e.State)
	}

	initialTargetPos := e.RangedTargetPos

	// Move the player — target pos should update
	p.Position = entity.Vec3{X: 5, Z: 15}
	b.Tick(0.05, players, nil, noSpawn, nil)

	if e.RangedTargetPos == initialTargetPos {
		t.Error("ranged with TrackTarget=true should update target position during telegraph")
	}
}

// TestBrain_GuardCaptainTree verifies the boss tree builds and runs without panic.
func TestBrain_GuardCaptainTree(t *testing.T) {
	b, e := testBrain(&GuardCaptain)
	e.Alive = true
	e.State = entity.EnemyChase

	p := testPlayer(1, entity.Vec3{X: 0, Z: 5})
	players := testPlayers(p)
	e.TargetPlayerID = p.ID

	for range 200 {
		b.Tick(0.05, players, nil, noSpawn, nil)
	}
	if !e.Alive {
		t.Error("boss should still be alive (no player damage)")
	}
}
