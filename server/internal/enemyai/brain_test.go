package enemyai

import (
	"math"
	"testing"

	"codex-online/server/internal/ability"
	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
)

// --- Helpers ---

func testDef() *EnemyDef {
	return &EnemyDef{
		Name:                  "test_enemy",
		MaxHealth:             1000,
		MoveSpeed:             4.0,
		Radius:                1.0,
		ChaseThreshold:        1.5,
		ChaseThresholdFar:     0.5,
		FarDistanceMultiplier: 3.0,
		AntiRepeat:            2.0,
		Abilities: []AbilityDef{
			{
				Name: "melee", Type: AbilityMelee,
				TelegraphTime: 1.0, CooldownTime: 1.0,
				BaseWeight: 50, MaxRange: 3.0,
				MeleeRange: 3.0, MeleeDamage: 30.0, MeleeConeAngle: math.Pi,
				DamageSourceType: SourceEnemyMelee,
			},
			{
				Name: "ranged", Type: AbilityRanged,
				TelegraphTime: 0.8, CooldownTime: 1.0,
				BaseWeight: 50, MinRange: 3.0,
				ProjectileCount: 1, ProjectileSpeed: 20.0,
				ProjectileDamage: 15.0, ProjectileOriginY: 1.5,
				ProjectileLifetime: 5.0,
				DamageSourceType:   SourceEnemyRanged,
			},
			{
				Name: "aoe", Type: AbilityAoE,
				TelegraphTime: 1.2, CooldownTime: 1.5,
				BaseWeight: 30, MaxRange: 7.0,
				AoERadius: 5.0, AoEDamage: 40.0,
				DamageSourceType: SourceEnemyAoE,
			},
			{
				Name: "charge", Type: AbilityCharge,
				TelegraphTime: 1.0, CooldownTime: 1.5,
				BaseWeight: 20, MinRange: 6.0,
				ChargeSpeed: 12.0, ChargeDamage: 35.0,
				ChargeMaxDistance: 15.0, ChargeHitRadius: 2.0,
				ChargeStopOnWall: true, ChargeStopOnObstacle: true,
				DamageSourceType: SourceEnemyCharge,
			},
		},
		Phases: []PhaseDef{
			{
				HPThresholdPct:   0.6,
				MoveSpeed:        5.0,
				CooldownOverride: 0.8,
				WeightOverrides:  map[string]int{"melee": 25, "ranged": 25, "aoe": 25, "charge": 25},
				AbilityOverrides: map[string]AbilityOverride{
					"melee": {TelegraphTime: F32(0.8), Damage: F32(35.0)},
				},
			},
		},
	}
}

func testBrain(def *EnemyDef) (*Brain, *entity.Enemy) {
	e := entity.NewEnemy(0, def.MaxHealth, def.Name)
	e.State = entity.EnemyChase
	e.Position = entity.Vec3{X: 0, Y: 0.1, Z: 0}
	eng := ability.NewEngine(nil)
	b := NewBrain(def, e, eng)
	b.BoundsMinX = -20
	b.BoundsMaxX = 20
	b.BoundsMinZ = -15
	b.BoundsMaxZ = 50
	return b, e
}

func testPlayer(id uint16, pos entity.Vec3) *entity.Player {
	p := entity.NewPlayer(id, entity.ClassGunner)
	p.Position = pos
	return p
}

func testPlayers(players ...*entity.Player) map[uint16]*entity.Player {
	m := make(map[uint16]*entity.Player, len(players))
	for _, p := range players {
		m[p.ID] = p
	}
	return m
}

// --- Targeting tests ---

func TestNearestAlivePlayer(t *testing.T) {
	p1 := testPlayer(1, entity.Vec3{X: 5, Z: 5})
	p2 := testPlayer(2, entity.Vec3{X: 1, Z: 1})
	p3 := testPlayer(3, entity.Vec3{X: 10, Z: 10})
	p3.Alive = false

	result := NearestAlivePlayer(entity.Vec3{}, testPlayers(p1, p2, p3))
	if result == nil || result.ID != 2 {
		t.Errorf("expected player 2 (nearest), got %v", result)
	}
}

func TestNearestAlivePlayerNoneAlive(t *testing.T) {
	p1 := testPlayer(1, entity.Vec3{X: 1})
	p1.Alive = false
	if got := NearestAlivePlayer(entity.Vec3{}, testPlayers(p1)); got != nil {
		t.Errorf("expected nil, got peer %d", got.ID)
	}
}

func TestNearestAlivePlayerEmpty(t *testing.T) {
	if got := NearestAlivePlayer(entity.Vec3{}, map[uint16]*entity.Player{}); got != nil {
		t.Error("expected nil for empty map")
	}
}

func TestFarthestAlivePlayer(t *testing.T) {
	p1 := testPlayer(1, entity.Vec3{X: 1, Z: 1})
	p2 := testPlayer(2, entity.Vec3{X: 10, Z: 10})
	p3 := testPlayer(3, entity.Vec3{X: 20, Z: 20})
	p3.Alive = false

	result := FarthestAlivePlayer(entity.Vec3{}, testPlayers(p1, p2, p3))
	if result == nil || result.ID != 2 {
		t.Errorf("expected player 2 (farthest alive), got %v", result)
	}
}

// --- EnemyDef tests ---

func TestCurrentPhase(t *testing.T) {
	def := testDef()
	if def.CurrentPhase(1) != nil {
		t.Error("phase 1 should return nil")
	}
	p2 := def.CurrentPhase(2)
	if p2 == nil {
		t.Fatal("phase 2 should not be nil")
	}
	if p2.MoveSpeed != 5.0 {
		t.Errorf("phase 2 move speed = %f, want 5.0", p2.MoveSpeed)
	}
}

func TestResolveAbilityBase(t *testing.T) {
	def := testDef()
	melee := &def.Abilities[0]
	resolved := def.ResolveAbility(melee, 1)
	if resolved.TelegraphTime != 1.0 {
		t.Errorf("base telegraph = %f, want 1.0", resolved.TelegraphTime)
	}
	if resolved.MeleeDamage != 30.0 {
		t.Errorf("base damage = %f, want 30.0", resolved.MeleeDamage)
	}
}

func TestResolveAbilityPhaseOverride(t *testing.T) {
	def := testDef()
	melee := &def.Abilities[0]
	resolved := def.ResolveAbility(melee, 2)
	if resolved.TelegraphTime != 0.8 {
		t.Errorf("phase 2 telegraph = %f, want 0.8", resolved.TelegraphTime)
	}
	if resolved.MeleeDamage != 35.0 {
		t.Errorf("phase 2 damage = %f, want 35.0", resolved.MeleeDamage)
	}
}

func TestCurrentMoveSpeed(t *testing.T) {
	def := testDef()
	if got := def.CurrentMoveSpeed(1); got != 4.0 {
		t.Errorf("phase 1 speed = %f, want 4.0", got)
	}
	if got := def.CurrentMoveSpeed(2); got != 5.0 {
		t.Errorf("phase 2 speed = %f, want 5.0", got)
	}
}

func TestCurrentBackpedalSpeed(t *testing.T) {
	def := testDef()
	// No BackpedalSpeed set → 50% of MoveSpeed
	if got := def.CurrentBackpedalSpeed(1); got != 2.0 {
		t.Errorf("phase 1 backpedal = %f, want 2.0", got)
	}
}

func TestCurrentCooldownTime(t *testing.T) {
	def := testDef()
	melee := &def.Abilities[0]
	if got := def.CurrentCooldownTime(melee, 1); got != 1.0 {
		t.Errorf("phase 1 cooldown = %f, want 1.0", got)
	}
	if got := def.CurrentCooldownTime(melee, 2); got != 0.8 {
		t.Errorf("phase 2 cooldown = %f, want 0.8 (override)", got)
	}
}

func TestLongestMeleeRange(t *testing.T) {
	def := testDef()
	if got := def.LongestMeleeRange(); got != 3.0 {
		t.Errorf("longest melee range = %f, want 3.0", got)
	}
}

func TestAbilityByIndex(t *testing.T) {
	def := testDef()
	if got := def.AbilityByIndex(0); got == nil || got.Name != "melee" { //nolint:goconst // test data
		t.Error("index 0 should be melee")
	}
	if got := def.AbilityByIndex(-1); got != nil {
		t.Error("index -1 should be nil")
	}
	if got := def.AbilityByIndex(100); got != nil {
		t.Error("index 100 should be nil")
	}
}

// --- Brain state handler tests ---

func TestTickCooldownTransitionsToChase(t *testing.T) {
	def := testDef()
	b, e := testBrain(def)
	e.State = entity.EnemyCooldown
	e.StateTimer = 0.5

	b.Tick(0.6, testPlayers(), nil, nil)
	if e.State != entity.EnemyChase {
		t.Errorf("state = %d, want EnemyChase", e.State)
	}
	if e.ChaseTimer != 0 {
		t.Errorf("chase timer should be 0 after cooldown, got %f", e.ChaseTimer)
	}
}

func TestTickCooldownStaysInCooldown(t *testing.T) {
	def := testDef()
	b, e := testBrain(def)
	e.State = entity.EnemyCooldown
	e.StateTimer = 1.0

	b.Tick(0.5, testPlayers(), nil, nil)
	if e.State != entity.EnemyCooldown {
		t.Errorf("state = %d, want EnemyCooldown", e.State)
	}
}

func TestTickPhaseTransitionToChase(t *testing.T) {
	def := testDef()
	b, e := testBrain(def)
	e.State = entity.EnemyPhaseTransition
	e.StateTimer = 0.5

	b.Tick(0.6, testPlayers(), nil, nil)
	if e.State != entity.EnemyChase {
		t.Errorf("state = %d, want EnemyChase", e.State)
	}
}

func TestTickDeadStopsVelocity(t *testing.T) {
	def := testDef()
	b, e := testBrain(def)
	e.State = entity.EnemyDead
	e.Velocity = entity.Vec3{X: 5, Z: 5}

	b.Tick(0.05, testPlayers(), nil, nil)
	if e.Velocity.X != 0 || e.Velocity.Z != 0 {
		t.Errorf("velocity should be zero, got %v", e.Velocity)
	}
}

func TestTickMeleeTelegraphTransitions(t *testing.T) {
	def := testDef()
	b, e := testBrain(def)
	e.State = entity.EnemyMeleeTelegraph
	e.StateTimer = 0.3

	b.Tick(0.4, testPlayers(), nil, nil)
	if e.State != entity.EnemyMeleeAttack {
		t.Errorf("state = %d, want EnemyMeleeAttack", e.State)
	}
	if e.StateTimer != 0.3 {
		t.Errorf("attack timer = %f, want 0.3", e.StateTimer)
	}
}

func TestTickMeleeAttackDealsDamage(t *testing.T) {
	def := testDef()
	b, e := testBrain(def)
	e.State = entity.EnemyMeleeAttack
	e.StateTimer = 0 // ready to attack
	e.ActiveAbility = 0
	e.RotationY = 0 // facing -Z

	p := testPlayer(1, entity.Vec3{X: 0, Y: 0.1, Z: -2}) // in front, 2m away
	events := b.Tick(0.05, testPlayers(p), nil, nil)

	if len(events) == 0 {
		t.Fatal("expected at least one damage event")
	}
	if events[0].TargetPeerID != 1 {
		t.Errorf("target = %d, want 1", events[0].TargetPeerID)
	}
	if events[0].Amount <= 0 {
		t.Error("damage should be > 0")
	}
	if e.State != entity.EnemyCooldown {
		t.Errorf("state after attack = %d, want EnemyCooldown", e.State)
	}
}

func TestTickMeleeAttackMissesOutOfCone(t *testing.T) {
	def := testDef()
	b, e := testBrain(def)
	e.State = entity.EnemyMeleeAttack
	e.StateTimer = 0
	e.ActiveAbility = 0
	e.RotationY = 0 // facing -Z

	p := testPlayer(1, entity.Vec3{X: 0, Y: 0.1, Z: 2}) // behind, 2m away
	events := b.Tick(0.05, testPlayers(p), nil, nil)
	if len(events) != 0 {
		t.Errorf("expected no damage for target behind enemy, got %d events", len(events))
	}
}

func TestTickMeleeAttackMissesOutOfRange(t *testing.T) {
	def := testDef()
	b, e := testBrain(def)
	e.State = entity.EnemyMeleeAttack
	e.StateTimer = 0
	e.ActiveAbility = 0
	e.RotationY = 0

	p := testPlayer(1, entity.Vec3{X: 0, Y: 0.1, Z: -10}) // in front but 10m away
	events := b.Tick(0.05, testPlayers(p), nil, nil)
	if len(events) != 0 {
		t.Errorf("expected no damage for target out of range, got %d", len(events))
	}
}

func TestTickMeleeAttackSkipsDeadPlayer(t *testing.T) {
	def := testDef()
	b, e := testBrain(def)
	e.State = entity.EnemyMeleeAttack
	e.StateTimer = 0
	e.ActiveAbility = 0
	e.RotationY = 0

	p := testPlayer(1, entity.Vec3{X: 0, Y: 0.1, Z: -2})
	p.Alive = false
	events := b.Tick(0.05, testPlayers(p), nil, nil)
	if len(events) != 0 {
		t.Errorf("expected no damage for dead player, got %d", len(events))
	}
}

func TestTickRangedTelegraphTransitions(t *testing.T) {
	def := testDef()
	b, e := testBrain(def)
	e.State = entity.EnemyRangedTelegraph
	e.StateTimer = 0.3
	e.TargetPlayerID = 1
	p := testPlayer(1, entity.Vec3{X: 5, Y: 0.1, Z: 5})

	b.Tick(0.4, testPlayers(p), nil, nil)
	if e.State != entity.EnemyRangedAttack {
		t.Errorf("state = %d, want EnemyRangedAttack", e.State)
	}
}

func TestTickRangedTelegraphTracksTarget(t *testing.T) {
	def := testDef()
	b, e := testBrain(def)
	e.State = entity.EnemyRangedTelegraph
	e.StateTimer = 1.0 // not transitioning yet
	e.TargetPlayerID = 1
	p := testPlayer(1, entity.Vec3{X: 10, Y: 0.1, Z: 10})

	b.Tick(0.05, testPlayers(p), nil, nil)
	// RangedTargetPos should be updated to player position + Y offset
	if e.RangedTargetPos.X != 10 || e.RangedTargetPos.Y != 1.1 {
		t.Errorf("ranged target = %v, want (10, 1.1, 10)", e.RangedTargetPos)
	}
}

func TestTickRangedAttackSpawnsProjectile(t *testing.T) {
	def := testDef()
	b, e := testBrain(def)
	e.State = entity.EnemyRangedAttack
	e.StateTimer = 0
	e.ActiveAbility = 1
	e.RangedTargetPos = entity.Vec3{X: 10, Y: 1.5, Z: 0}

	spawned := 0
	spawnFn := func(_, _ entity.Vec3, speed, damage, _ float32) {
		spawned++
		if speed != 20.0 {
			t.Errorf("projectile speed = %f, want 20.0", speed)
		}
		if damage != 15.0 {
			t.Errorf("projectile damage = %f, want 15.0", damage)
		}
	}

	b.Tick(0.05, testPlayers(), nil, spawnFn)
	if spawned != 1 {
		t.Errorf("spawned = %d, want 1", spawned)
	}
	if e.State != entity.EnemyCooldown {
		t.Errorf("state = %d, want EnemyCooldown", e.State)
	}
}

func TestTickAoETelegraphTransitions(t *testing.T) {
	def := testDef()
	b, e := testBrain(def)
	e.State = entity.EnemyAoETelegraph
	e.StateTimer = 0.3

	b.Tick(0.4, testPlayers(), nil, nil)
	if e.State != entity.EnemyAoESlam {
		t.Errorf("state = %d, want EnemyAoESlam", e.State)
	}
}

func TestTickAoESlamDealsDamage(t *testing.T) {
	def := testDef()
	b, e := testBrain(def)
	e.State = entity.EnemyAoESlam
	e.StateTimer = 0
	e.ActiveAbility = 2 // aoe ability

	p := testPlayer(1, entity.Vec3{X: 3, Y: 0.1, Z: 0}) // 3m, within 5m radius
	events := b.Tick(0.05, testPlayers(p), nil, nil)

	if len(events) == 0 {
		t.Fatal("expected damage from AoE")
	}
	if events[0].Amount <= 0 {
		t.Error("AoE damage should be > 0")
	}
	if e.State != entity.EnemyCooldown {
		t.Errorf("state = %d, want EnemyCooldown", e.State)
	}
}

func TestTickAoESlamMissesOutOfRadius(t *testing.T) {
	def := testDef()
	b, e := testBrain(def)
	e.State = entity.EnemyAoESlam
	e.StateTimer = 0
	e.ActiveAbility = 2

	p := testPlayer(1, entity.Vec3{X: 10, Y: 0.1, Z: 0}) // 10m, outside 5m radius
	events := b.Tick(0.05, testPlayers(p), nil, nil)
	if len(events) != 0 {
		t.Errorf("expected no damage outside AoE radius, got %d", len(events))
	}
}

func TestTickChargeTelegraphSetsDirection(t *testing.T) {
	def := testDef()
	b, e := testBrain(def)
	e.State = entity.EnemyChargeTelegraph
	e.StateTimer = 1.0
	e.TargetPlayerID = 1

	p := testPlayer(1, entity.Vec3{X: 10, Y: 0.1, Z: 0})
	b.Tick(0.05, testPlayers(p), nil, nil)

	// Should set charge direction toward player
	if e.ChargeDirection.Length() < 0.5 {
		t.Error("charge direction should be set during telegraph")
	}
}

func TestTickChargeTelegraphTransitions(t *testing.T) {
	def := testDef()
	b, e := testBrain(def)
	e.State = entity.EnemyChargeTelegraph
	e.StateTimer = 0.3
	e.TargetPlayerID = 1

	p := testPlayer(1, entity.Vec3{X: 10, Y: 0.1, Z: 0})
	b.Tick(0.4, testPlayers(p), nil, nil)

	if e.State != entity.EnemyCharge {
		t.Errorf("state = %d, want EnemyCharge", e.State)
	}
	if e.ChargeDistance != 0 {
		t.Error("charge distance should be 0 at start")
	}
}

func TestTickChargeDealsDamage(t *testing.T) {
	def := testDef()
	b, e := testBrain(def)
	e.State = entity.EnemyCharge
	e.ActiveAbility = 3 // charge
	e.ChargeDirection = entity.Vec3{X: 1, Z: 0}
	e.ChargeHitPlayers = []uint16{}
	e.Position = entity.Vec3{X: 0, Y: 0.1, Z: 0}

	p := testPlayer(1, entity.Vec3{X: 0.5, Y: 0.1, Z: 0}) // within hit radius
	events := b.Tick(0.05, testPlayers(p), nil, nil)

	if len(events) == 0 {
		t.Fatal("expected damage from charge hit")
	}
	if events[0].Amount <= 0 {
		t.Error("charge damage should be > 0")
	}
}

func TestTickChargeDoesNotHitSamePlayerTwice(t *testing.T) {
	def := testDef()
	b, e := testBrain(def)
	e.State = entity.EnemyCharge
	e.ActiveAbility = 3
	e.ChargeDirection = entity.Vec3{X: 1, Z: 0}
	e.ChargeHitPlayers = []uint16{1} // already hit
	e.Position = entity.Vec3{X: 0, Y: 0.1, Z: 0}

	p := testPlayer(1, entity.Vec3{X: 0.5, Y: 0.1, Z: 0})
	events := b.Tick(0.05, testPlayers(p), nil, nil)

	if len(events) != 0 {
		t.Errorf("should not hit same player twice, got %d events", len(events))
	}
}

func TestTickChargeStopsAtMaxDistance(t *testing.T) {
	def := testDef()
	b, e := testBrain(def)
	e.State = entity.EnemyCharge
	e.ActiveAbility = 3
	e.ChargeDirection = entity.Vec3{X: 1, Z: 0}
	e.ChargeDistance = 14.5
	e.ChargeHitPlayers = []uint16{}

	// dt=0.05, speed=12 → distance += 0.6, total = 15.1 > 15.0 max
	b.Tick(0.05, testPlayers(), nil, nil)
	if e.State != entity.EnemyCooldown {
		t.Errorf("state = %d, want EnemyCooldown (max distance reached)", e.State)
	}
}

func TestTickChargeStopsAtWall(t *testing.T) {
	def := testDef()
	b, e := testBrain(def)
	e.State = entity.EnemyCharge
	e.ActiveAbility = 3
	e.ChargeDirection = entity.Vec3{X: 1, Z: 0}
	e.ChargeDistance = 0
	e.ChargeHitPlayers = []uint16{}
	e.Position = entity.Vec3{X: 19.6, Y: 0.1, Z: 0} // near wall at X=20

	b.Tick(0.05, testPlayers(), nil, nil)
	if e.State != entity.EnemyCooldown {
		t.Errorf("state = %d, want EnemyCooldown (at wall)", e.State)
	}
}

func TestTickChargeStopsAtObstacle(t *testing.T) {
	def := testDef()
	b, e := testBrain(def)
	e.State = entity.EnemyCharge
	e.ActiveAbility = 3
	e.ChargeDirection = entity.Vec3{X: 1, Z: 0}
	e.ChargeDistance = 0
	e.ChargeHitPlayers = []uint16{}
	e.Position = entity.Vec3{X: 5, Y: 0.1, Z: 0}

	obs := []combat.Obstacle{{CX: 5.2, CZ: 0, HX: 0.5, HZ: 0.5}}
	b.Tick(0.05, testPlayers(), obs, nil)
	if e.State != entity.EnemyCooldown {
		t.Errorf("state = %d, want EnemyCooldown (at obstacle)", e.State)
	}
}

// --- Patrol & Leash ---

func TestTickPatrolWalksTowardWaypoint(t *testing.T) {
	def := testDef()
	b, e := testBrain(def)
	e.State = entity.EnemyPatrol
	e.PatrolA = entity.Vec3{X: -5, Z: 0}
	e.PatrolB = entity.Vec3{X: 5, Z: 0}
	e.PatrolTarget = 1 // heading to B
	e.Position = entity.Vec3{X: 0, Y: 0.1, Z: 0}

	b.Tick(0.05, testPlayers(), nil, nil)
	// Should have velocity toward +X (PatrolB)
	if e.Velocity.X <= 0 {
		t.Errorf("velocity.X = %f, should be positive (toward PatrolB)", e.Velocity.X)
	}
}

func TestTickPatrolFlipsAtWaypoint(t *testing.T) {
	def := testDef()
	b, e := testBrain(def)
	e.State = entity.EnemyPatrol
	e.PatrolA = entity.Vec3{X: -5, Z: 0}
	e.PatrolB = entity.Vec3{X: 5, Z: 0}
	e.PatrolTarget = 1
	e.Position = entity.Vec3{X: 4.8, Y: 0.1, Z: 0} // within 0.5 of PatrolB

	b.Tick(0.05, testPlayers(), nil, nil)
	if e.PatrolTarget != 0 {
		t.Errorf("patrol target = %d, want 0 (flipped to A)", e.PatrolTarget)
	}
}

func TestTickPatrolAggroOnNearbyPlayer(t *testing.T) {
	def := testDef()
	b, e := testBrain(def)
	e.State = entity.EnemyPatrol
	e.AggroRadius = 10
	e.PatrolA = entity.Vec3{X: -5, Z: 0}
	e.PatrolB = entity.Vec3{X: 5, Z: 0}

	p := testPlayer(1, entity.Vec3{X: 3, Y: 0.1, Z: 0}) // within 10m
	b.Tick(0.05, testPlayers(p), nil, nil)

	if e.State != entity.EnemyChase {
		t.Errorf("state = %d, want EnemyChase (aggroed)", e.State)
	}
	if e.TargetPlayerID != 1 {
		t.Errorf("target = %d, want 1", e.TargetPlayerID)
	}
}

func TestTickPatrolNoAggroFarPlayer(t *testing.T) {
	def := testDef()
	b, e := testBrain(def)
	e.State = entity.EnemyPatrol
	e.AggroRadius = 5
	e.PatrolA = entity.Vec3{X: -5, Z: 0}
	e.PatrolB = entity.Vec3{X: 5, Z: 0}

	p := testPlayer(1, entity.Vec3{X: 20, Y: 0.1, Z: 0}) // beyond 5m aggro
	b.Tick(0.05, testPlayers(p), nil, nil)

	if e.State != entity.EnemyPatrol {
		t.Errorf("state = %d, want EnemyPatrol (player too far)", e.State)
	}
}

func TestCheckLeashResets(t *testing.T) {
	def := testDef()
	b, e := testBrain(def)
	e.State = entity.EnemyChase
	e.LeashRadius = 10
	e.LeashOrigin = entity.Vec3{X: 0, Y: 0.1, Z: 0}
	e.Position = entity.Vec3{X: 15, Y: 0.1, Z: 0} // 15m > 10m leash
	e.Health = 500
	e.AddThreat(1, 100)

	p := testPlayer(1, entity.Vec3{X: 20, Y: 0.1, Z: 0})
	b.Tick(0.05, testPlayers(p), nil, nil)

	if e.State != entity.EnemyPatrol {
		t.Errorf("state = %d, want EnemyPatrol (leashed)", e.State)
	}
	if e.Health != e.MaxHealth {
		t.Errorf("health = %f, want %f (full after leash)", e.Health, e.MaxHealth)
	}
	if e.Position != e.LeashOrigin {
		t.Errorf("position should be leash origin, got %v", e.Position)
	}
	if len(e.ThreatTable) != 0 {
		t.Error("threat table should be cleared after leash")
	}
}

func TestCheckLeashNoResetWithinRadius(t *testing.T) {
	def := testDef()
	b, e := testBrain(def)
	e.State = entity.EnemyChase
	e.LeashRadius = 20
	e.LeashOrigin = entity.Vec3{X: 0, Y: 0.1, Z: 0}
	e.Position = entity.Vec3{X: 5, Y: 0.1, Z: 0}

	p := testPlayer(1, entity.Vec3{X: 3, Y: 0.1, Z: 0})
	b.Tick(0.05, testPlayers(p), nil, nil)

	if e.State == entity.EnemyPatrol {
		t.Error("should not leash within radius")
	}
}

// --- Chase behavior ---

func TestTickChaseMovesTowardPlayer(t *testing.T) {
	def := testDef()
	b, e := testBrain(def)
	e.State = entity.EnemyChase
	e.Position = entity.Vec3{X: 0, Y: 0.1, Z: 0}

	p := testPlayer(1, entity.Vec3{X: 10, Y: 0.1, Z: 0})
	b.Tick(0.05, testPlayers(p), nil, nil)

	if e.Velocity.X <= 0 {
		t.Errorf("velocity.X = %f, should be positive (chasing toward +X)", e.Velocity.X)
	}
}

func TestTickChaseStopsWhenNoPlayers(t *testing.T) {
	def := testDef()
	b, e := testBrain(def)
	e.State = entity.EnemyChase

	b.Tick(0.05, testPlayers(), nil, nil)
	if e.Velocity.X != 0 || e.Velocity.Z != 0 {
		t.Errorf("velocity should be zero with no players, got %v", e.Velocity)
	}
}

// --- selectAbility ---

func TestSelectAbilityRespectsRange(t *testing.T) {
	def := testDef()
	b, e := testBrain(def)
	_ = e

	p := testPlayer(1, entity.Vec3{X: 2, Y: 0.1, Z: 0}) // 2m = within melee range, below ranged MinRange
	chosen := b.selectAbility(2.0, testPlayers(p))

	if chosen == nil {
		t.Fatal("expected an ability")
	}
	// At 2m: melee (max 3) ok, ranged (min 3) excluded, aoe (max 7) ok, charge (min 6) excluded
	if chosen.Type == AbilityRanged || chosen.Type == AbilityCharge {
		t.Errorf("ability %s should not be selectable at 2m range", chosen.Name)
	}
}

func TestSelectAbilityAntiRepeat(t *testing.T) {
	def := &EnemyDef{
		Name:       "test",
		MaxHealth:  100,
		MoveSpeed:  4,
		AntiRepeat: 100.0, // heavy anti-repeat
		Abilities: []AbilityDef{
			{Name: "a", Type: AbilityMelee, BaseWeight: 10, MaxRange: 5, MeleeRange: 5},
			{Name: "b", Type: AbilityMelee, BaseWeight: 10, MaxRange: 5, MeleeRange: 5},
		},
	}
	b, e := testBrain(def)

	// After using "a" 100 times with extreme anti-repeat, "b" should dominate
	e.LastAttack = "a"
	bCount := 0
	for i := 0; i < 100; i++ {
		chosen := b.selectAbility(3.0, testPlayers())
		if chosen != nil && chosen.Name == "b" {
			bCount++
		}
	}
	// With weight 10/100 = 0 for "a" vs weight 10 for "b", "b" should almost always win
	if bCount < 80 {
		t.Errorf("anti-repeat: 'b' selected %d/100, expected > 80", bCount)
	}
}

func TestSelectAbilityReturnsNilWhenNoCandidates(t *testing.T) {
	def := &EnemyDef{
		Name:      "test",
		MaxHealth: 100,
		MoveSpeed: 4,
		Abilities: []AbilityDef{
			{Name: "melee", Type: AbilityMelee, BaseWeight: 10, MaxRange: 3, MeleeRange: 3},
		},
	}
	b, _ := testBrain(def)

	// Target at 10m, melee max 3m → no candidates
	chosen := b.selectAbility(10.0, testPlayers())
	if chosen != nil {
		t.Errorf("expected nil at 10m, got %s", chosen.Name)
	}
}

// --- avoidObstacles ---

func TestAvoidObstaclesNoBlocker(t *testing.T) {
	def := testDef()
	b, e := testBrain(def)

	dir := entity.Vec3{X: 1, Z: 0}
	from := e.Position
	to := entity.Vec3{X: 10, Z: 0}

	result := b.avoidObstacles(dir, from, to, nil)
	// No obstacles → direction unchanged
	if result.X < 0.9 {
		t.Errorf("direction should be ~(1,0,0), got %v", result)
	}
}

func TestAvoidObstaclesWithBlocker(t *testing.T) {
	def := testDef()
	b, e := testBrain(def)
	e.Position = entity.Vec3{X: 0, Y: 0.1, Z: 0}

	obs := []combat.Obstacle{{CX: 5, CZ: 0, HX: 1, HZ: 1}}
	dir := entity.Vec3{X: 1, Z: 0}
	to := entity.Vec3{X: 10, Z: 0}

	result := b.avoidObstacles(dir, e.Position, to, obs)
	// Should steer around the obstacle — Z component should be nonzero
	if result.Z == 0 {
		t.Error("expected obstacle avoidance to add Z component")
	}
}

// --- Velocity application ---

func TestTickAppliesVelocity(t *testing.T) {
	def := testDef()
	b, e := testBrain(def)
	e.State = entity.EnemyCharge
	e.ActiveAbility = 3
	e.ChargeDirection = entity.Vec3{X: 1, Z: 0}
	e.ChargeHitPlayers = []uint16{}

	startX := e.Position.X
	b.Tick(0.1, testPlayers(), nil, nil)

	if e.Position.X <= startX {
		t.Errorf("position.X should advance during charge, was %f now %f", startX, e.Position.X)
	}
}

// --- startAbility ---

func TestStartAbilitySetsState(t *testing.T) {
	def := testDef()
	b, e := testBrain(def)

	melee := &def.Abilities[0]
	players := testPlayers(testPlayer(1, entity.Vec3{X: 2, Z: 0}))
	b.startAbility(melee, e, players, nil)

	if e.State != entity.EnemyMeleeTelegraph {
		t.Errorf("state = %d, want EnemyMeleeTelegraph", e.State)
	}
	if e.StateTimer != 1.0 {
		t.Errorf("telegraph time = %f, want 1.0", e.StateTimer)
	}
	if e.LastAttack != "melee" {
		t.Errorf("last attack = %s, want melee", e.LastAttack)
	}
}

func TestStartAbilityRangedSetsTarget(t *testing.T) {
	def := testDef()
	b, e := testBrain(def)

	ranged := &def.Abilities[1]
	p := testPlayer(1, entity.Vec3{X: 10, Y: 0.1, Z: 10})
	b.startAbility(ranged, e, testPlayers(p), nil)

	if e.State != entity.EnemyRangedTelegraph {
		t.Errorf("state = %d, want EnemyRangedTelegraph", e.State)
	}
	if e.TargetPlayerID != 1 {
		t.Errorf("target = %d, want 1", e.TargetPlayerID)
	}
}

func TestStartAbilityChargeSetsState(t *testing.T) {
	def := testDef()
	b, e := testBrain(def)

	charge := &def.Abilities[3]
	p := testPlayer(1, entity.Vec3{X: 10, Y: 0.1, Z: 0})
	b.startAbility(charge, e, testPlayers(p), nil)

	if e.State != entity.EnemyChargeTelegraph {
		t.Errorf("state = %d, want EnemyChargeTelegraph", e.State)
	}
}

// --- faceTarget ---

func TestFaceTargetRotatesEnemy(t *testing.T) {
	def := testDef()
	b, e := testBrain(def)
	e.TargetPlayerID = 1
	e.RotationY = 0

	p := testPlayer(1, entity.Vec3{X: 5, Y: 0.1, Z: 0}) // to the right
	b.faceTarget(testPlayers(p))

	// Facing right (X=5, Z=0) → RotationY should be ~ -π/2
	expected := float32(math.Atan2(-5.0, 0.0))
	if diff := e.RotationY - expected; diff > 0.01 || diff < -0.01 {
		t.Errorf("rotY = %f, want %f", e.RotationY, expected)
	}
}

func TestFaceTargetNoopDeadTarget(t *testing.T) {
	def := testDef()
	b, e := testBrain(def)
	e.TargetPlayerID = 1
	e.RotationY = 0

	p := testPlayer(1, entity.Vec3{X: 5, Y: 0.1, Z: 0})
	p.Alive = false
	b.faceTarget(testPlayers(p))

	if e.RotationY != 0 {
		t.Errorf("rotY should not change for dead target, got %f", e.RotationY)
	}
}

// --- Registry ---

func TestDefRegistryContainsAll(t *testing.T) {
	expected := []string{"guard_captain", "hallway_melee", "hallway_ranged"}
	for _, name := range expected {
		if _, ok := DefRegistry[name]; !ok {
			t.Errorf("DefRegistry missing %s", name)
		}
	}
}

func TestGuardCaptainHasFourAbilities(t *testing.T) {
	if len(GuardCaptain.Abilities) != 4 {
		t.Errorf("guard captain abilities = %d, want 4", len(GuardCaptain.Abilities))
	}
}

func TestGuardCaptainPhaseOverrides(t *testing.T) {
	resolved := GuardCaptain.ResolveAbility(&GuardCaptain.Abilities[1], 2) // fireball_burst phase 2
	if resolved.ProjectileCount != 2 {
		t.Errorf("phase 2 projectile count = %d, want 2", resolved.ProjectileCount)
	}
	if resolved.ProjectileDamage != 15.0 {
		t.Errorf("phase 2 damage = %f, want 15.0", resolved.ProjectileDamage)
	}
}

// --- Brain.Enemy() ---

func TestBrainEnemyReturnsEnemy(t *testing.T) {
	def := testDef()
	b, e := testBrain(def)
	if b.Enemy() != e {
		t.Error("Brain.Enemy() should return the enemy passed to NewBrain")
	}
}

// --- findAbilityByType ---

func TestFindAbilityByType(t *testing.T) {
	def := testDef()
	b, _ := testBrain(def)

	got := b.findAbilityByType(AbilityMelee)
	if got == nil || got.Name != "melee" {
		t.Errorf("findAbilityByType(Melee) = %v, want melee", got)
	}

	got2 := b.findAbilityByType(AbilityRanged)
	if got2 == nil || got2.Name != "ranged" {
		t.Errorf("findAbilityByType(Ranged) = %v, want ranged", got2)
	}

	got3 := b.findAbilityByType(AbilityAoE)
	if got3 == nil || got3.Name != "aoe" {
		t.Errorf("findAbilityByType(AoE) = %v, want aoe", got3)
	}

	got4 := b.findAbilityByType(AbilityCharge)
	if got4 == nil || got4.Name != "charge" {
		t.Errorf("findAbilityByType(Charge) = %v, want charge", got4)
	}

	// Type not present
	got5 := b.findAbilityByType(AbilityType(99))
	if got5 != nil {
		t.Errorf("findAbilityByType(99) = %v, want nil", got5)
	}
}

// --- FaceToward ---

func TestFaceTowardRight(t *testing.T) {
	from := entity.Vec3{X: 0, Y: 0, Z: 0}
	to := entity.Vec3{X: 5, Y: 0, Z: 0}
	yaw := FaceToward(from, to)
	// Facing +X → atan2(-5, 0) = -pi/2
	expected := float32(math.Atan2(-5, 0))
	if diff := yaw - expected; diff > 0.01 || diff < -0.01 {
		t.Errorf("FaceToward(right) = %f, want %f", yaw, expected)
	}
}

func TestFaceTowardForward(t *testing.T) {
	from := entity.Vec3{X: 0, Y: 0, Z: 0}
	to := entity.Vec3{X: 0, Y: 0, Z: -5}
	yaw := FaceToward(from, to)
	// Facing -Z → atan2(0, 5) = 0
	if yaw > 0.01 || yaw < -0.01 {
		t.Errorf("FaceToward(forward) = %f, want ~0", yaw)
	}
}

func TestFaceTowardSamePosition(t *testing.T) {
	pos := entity.Vec3{X: 3, Y: 1, Z: 5}
	yaw := FaceToward(pos, pos)
	if yaw != 0 {
		t.Errorf("FaceToward(same) = %f, want 0", yaw)
	}
}

// --- resolveEngineDef tests ---

func TestResolveEngineDef_Melee(t *testing.T) {
	def := testDef()
	b, e := testBrain(def)
	e.ActiveAbility = 0

	resolved := b.activeAbilityResolved()
	b.resolveEngineDef(resolved)

	if b.defBuf.ID != "melee" {
		t.Errorf("ID = %q, want %q", b.defBuf.ID, "melee")
	}
	if b.defBuf.BaseDamage != 30.0 {
		t.Errorf("BaseDamage = %f, want 30.0", b.defBuf.BaseDamage)
	}
	if b.defBuf.Hit.Type != ability.HitAoECone {
		t.Errorf("Hit.Type = %d, want HitAoECone (%d)", b.defBuf.Hit.Type, ability.HitAoECone)
	}
	if b.defBuf.Hit.Range != 3.0 {
		t.Errorf("Hit.Range = %f, want 3.0", b.defBuf.Hit.Range)
	}
	// π radians → 180 degrees
	if math.Abs(float64(b.defBuf.Hit.ArcDegrees-180)) > 0.01 {
		t.Errorf("Hit.ArcDegrees = %f, want 180", b.defBuf.Hit.ArcDegrees)
	}
}

func TestResolveEngineDef_MeleeConeAngleConversion(t *testing.T) {
	tests := []struct {
		name        string
		coneAngle   float32
		wantDegrees float32
	}{
		{"90 degrees", float32(math.Pi / 2), 90},
		{"180 degrees (π)", float32(math.Pi), 180},
		{"360 degrees (2π)", float32(2 * math.Pi), 360},
		{"60 degrees", float32(math.Pi / 3), 60},
		{"custom 1.0 rad", 1.0, float32(1.0 * 180.0 / math.Pi)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def := &EnemyDef{
				Name:      "cone_test",
				MaxHealth: 100,
				MoveSpeed: 4,
				Abilities: []AbilityDef{
					{
						Name: "melee", Type: AbilityMelee,
						MeleeRange: 3, MeleeDamage: 10,
						MeleeConeAngle: tt.coneAngle,
					},
				},
			}
			b, e := testBrain(def)
			e.ActiveAbility = 0
			resolved := b.activeAbilityResolved()
			b.resolveEngineDef(resolved)
			if math.Abs(float64(b.defBuf.Hit.ArcDegrees-tt.wantDegrees)) > 0.1 {
				t.Errorf("ArcDegrees = %f, want %f", b.defBuf.Hit.ArcDegrees, tt.wantDegrees)
			}
		})
	}
}

func TestResolveEngineDef_MeleeDefaultConeAngle(t *testing.T) {
	def := &EnemyDef{
		Name:      "default_cone",
		MaxHealth: 100,
		MoveSpeed: 4,
		Abilities: []AbilityDef{
			{
				Name: "melee", Type: AbilityMelee,
				MeleeRange: 3, MeleeDamage: 10,
				MeleeConeAngle: 0, // zero → should default to π (180°)
			},
		},
	}
	b, e := testBrain(def)
	e.ActiveAbility = 0
	resolved := b.activeAbilityResolved()
	b.resolveEngineDef(resolved)
	if math.Abs(float64(b.defBuf.Hit.ArcDegrees-180)) > 0.1 {
		t.Errorf("ArcDegrees = %f, want 180 (default)", b.defBuf.Hit.ArcDegrees)
	}
}

func TestResolveEngineDef_AoE(t *testing.T) {
	def := testDef()
	b, e := testBrain(def)
	e.ActiveAbility = 2 // aoe ability

	resolved := b.activeAbilityResolved()
	b.resolveEngineDef(resolved)

	if b.defBuf.ID != "aoe" {
		t.Errorf("ID = %q, want %q", b.defBuf.ID, "aoe")
	}
	if b.defBuf.BaseDamage != 40.0 {
		t.Errorf("BaseDamage = %f, want 40.0", b.defBuf.BaseDamage)
	}
	if b.defBuf.Hit.Type != ability.HitAoECircle {
		t.Errorf("Hit.Type = %d, want HitAoECircle (%d)", b.defBuf.Hit.Type, ability.HitAoECircle)
	}
	if b.defBuf.Hit.Radius != 5.0 {
		t.Errorf("Hit.Radius = %f, want 5.0", b.defBuf.Hit.Radius)
	}
}

func TestResolveEngineDef_Ranged_ReturnsMinimal(t *testing.T) {
	def := testDef()
	b, e := testBrain(def)
	e.ActiveAbility = 1 // ranged ability

	resolved := b.activeAbilityResolved()
	b.resolveEngineDef(resolved)

	if b.defBuf.ID != "ranged" {
		t.Errorf("ID = %q, want %q", b.defBuf.ID, "ranged")
	}
	// Ranged is handled by brain (projectile spawning), engine def is minimal
	if b.defBuf.BaseDamage != 0 {
		t.Errorf("BaseDamage = %f, want 0 (ranged uses projectile system)", b.defBuf.BaseDamage)
	}
}

func TestResolveEngineDef_PhaseOverride(t *testing.T) {
	def := testDef()
	b, e := testBrain(def)
	e.ActiveAbility = 0
	e.Phase = 2 // phase 2: melee damage overridden to 35

	resolved := b.activeAbilityResolved()
	b.resolveEngineDef(resolved)

	if b.defBuf.BaseDamage != 35.0 {
		t.Errorf("BaseDamage = %f, want 35.0 (phase 2 override)", b.defBuf.BaseDamage)
	}
}

// --- playersToTargets tests ---

func TestPlayersToTargets_Empty(t *testing.T) {
	targets := playersToTargets(map[uint16]*entity.Player{})
	if len(targets) != 0 {
		t.Errorf("len = %d, want 0", len(targets))
	}
}

func TestPlayersToTargets_MultiPlayers(t *testing.T) {
	players := testPlayers(
		testPlayer(1, entity.Vec3{X: 1}),
		testPlayer(2, entity.Vec3{X: 2}),
		testPlayer(3, entity.Vec3{X: 3}),
	)
	targets := playersToTargets(players)
	if len(targets) != 3 {
		t.Fatalf("len = %d, want 3", len(targets))
	}
	// Verify all are valid entity.Target and map back to correct players
	seen := map[uint16]bool{}
	for _, tgt := range targets {
		p, ok := tgt.(*entity.Player)
		if !ok {
			t.Fatal("target is not *entity.Player")
		}
		seen[p.ID] = true
	}
	for _, id := range []uint16{1, 2, 3} {
		if !seen[id] {
			t.Errorf("player %d not in targets", id)
		}
	}
}

func TestPlayersToTargets_ImplementsTargetInterface(t *testing.T) {
	p := testPlayer(1, entity.Vec3{X: 5, Y: 1, Z: 3})
	targets := playersToTargets(testPlayers(p))
	if len(targets) != 1 {
		t.Fatal("expected 1 target")
	}
	tgt := targets[0]
	if tgt.TargetID() != 1 {
		t.Errorf("TargetID() = %d, want 1", tgt.TargetID())
	}
	if !tgt.TargetAlive() {
		t.Error("TargetAlive() should be true")
	}
	if tgt.TargetPos() != p.Position {
		t.Errorf("TargetPos() = %v, want %v", tgt.TargetPos(), p.Position)
	}
}

// --- Benchmarks ---

func BenchmarkResolveEngineDef_Melee(b *testing.B) {
	def := testDef()
	br, e := testBrain(def)
	e.ActiveAbility = 0
	resolved := br.activeAbilityResolved()

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		br.resolveEngineDef(resolved)
	}
}

func BenchmarkResolveEngineDef_AoE(b *testing.B) {
	def := testDef()
	br, e := testBrain(def)
	e.ActiveAbility = 2
	resolved := br.activeAbilityResolved()

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		br.resolveEngineDef(resolved)
	}
}

func BenchmarkFillTargets_5(b *testing.B) {
	players := make(map[uint16]*entity.Player, 5)
	for i := range 5 {
		players[uint16(i+1)] = testPlayer(uint16(i+1), entity.Vec3{X: float32(i)})
	}
	br, _ := testBrain(testDef())
	// Pre-allocate to warm the buffer
	br.fillTargets(players)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		br.fillTargets(players)
	}
}

func BenchmarkTickMeleeAttack_Hit(b *testing.B) {
	def := testDef()
	br, e := testBrain(def)
	p := testPlayer(1, entity.Vec3{X: 0, Y: 0.1, Z: -2})
	p.Health = 1e9
	players := testPlayers(p)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		e.State = entity.EnemyMeleeAttack
		e.StateTimer = 0
		e.ActiveAbility = 0
		e.RotationY = 0
		p.Health = 1e9
		p.Alive = true
		br.Tick(0.05, players, nil, nil)
	}
}

func BenchmarkTickAoESlam_Hit(b *testing.B) {
	def := testDef()
	br, e := testBrain(def)
	p := testPlayer(1, entity.Vec3{X: 3, Y: 0.1, Z: 0})
	p.Health = 1e9
	players := testPlayers(p)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		e.State = entity.EnemyAoESlam
		e.StateTimer = 0
		e.ActiveAbility = 2
		p.Health = 1e9
		p.Alive = true
		br.Tick(0.05, players, nil, nil)
	}
}

func BenchmarkSelectAbility(b *testing.B) {
	def := testDef()
	br, _ := testBrain(def)
	players := testPlayers()

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		br.selectAbility(2.5, players)
	}
}
