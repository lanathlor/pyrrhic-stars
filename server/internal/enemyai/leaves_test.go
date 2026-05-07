package enemyai

import (
	"math/rand/v2"
	"testing"

	"codex-online/server/internal/ability"
	"codex-online/server/internal/bt"
	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
)

// testCtx creates an EntityContext for isolated leaf testing.
func testCtx(def *EnemyDef, e *entity.Enemy, players []*entity.Player) *EntityContext {
	bb := NewBlackboard()
	events := &[]combat.DamageEvent{}
	ctx := &EntityContext{
		Enemy:      e,
		Def:        def,
		Engine:     ability.NewEngine(nil),
		BB:         bb,
		Rng:        rand.New(rand.NewPCG(0, 0)),
		Runner:     &AbilityRunner{},
		Players:    players,
		Dt:         0.05,
		Events:     events,
		BoundsMinX: -20, BoundsMaxX: 20,
		BoundsMinZ: -15, BoundsMaxZ: 50,
		SpawnFn: func(_, _ entity.Vec3, _, _, _ float32) {},
	}
	return ctx
}

func simpleMeleeDef() *EnemyDef {
	return &EnemyDef{
		Name:      "test_leaf",
		MaxHealth: 500,
		MoveSpeed: 4.0,
		Radius:    1.0,
		Abilities: []ability.AbilityDef{
			{
				ID: "melee", Name: "melee", Category: ability.CategoryMelee,
				CommitTime: 0.5, ExecuteTime: 0.2, Cooldown: 0.5,
				BaseWeight: 100, MaxRange: 3.0,
				BaseDamage:   20.0,
				Hit:          ability.HitDef{Type: ability.HitAoECone, Range: 3.0, ArcDegrees: 180},
				DamageSource: combat.SourceEnemyMelee,
			},
		},
	}
}

// ============================================================
// Conditions
// ============================================================

func TestCond_HasTarget(t *testing.T) {
	def := simpleMeleeDef()
	e := entity.NewEnemy(0, 500, "test")
	p := testPlayer(1, entity.Vec3{X: 0, Z: 5})

	// No target set
	ctx := testCtx(def, e, testPlayers(p))
	if condHasTarget(ctx) {
		t.Error("should be false when TargetPlayerID is 0")
	}

	// Target set to alive player
	e.TargetPlayerID = p.ID
	ctx.nearestCached = false
	if !condHasTarget(ctx) {
		t.Error("should be true when target is alive")
	}

	// Target set but player is dead
	p.Alive = false
	if condHasTarget(ctx) {
		t.Error("should be false when target is dead")
	}
}

func TestCond_TargetInMeleeRange(t *testing.T) {
	def := simpleMeleeDef()
	e := entity.NewEnemy(0, 500, "test")
	e.Position = entity.Vec3{X: 0, Z: 0}

	tests := []struct {
		name     string
		playerZ  float32
		expected bool
	}{
		{"within_range", 2.0, true},
		{"at_boundary", 3.0, true},
		{"beyond_range", 4.0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := testPlayer(1, entity.Vec3{X: 0, Z: tt.playerZ})
			e.TargetPlayerID = p.ID
			ctx := testCtx(def, e, testPlayers(p))
			if got := condTargetInMeleeRange(ctx); got != tt.expected {
				t.Errorf("dist=%.1f: got %v, want %v", tt.playerZ, got, tt.expected)
			}
		})
	}

	// No target
	t.Run("no_target", func(t *testing.T) {
		e.TargetPlayerID = 99
		ctx := testCtx(def, e, testPlayers())
		if condTargetInMeleeRange(ctx) {
			t.Error("should be false with no target")
		}
	})
}

func TestCond_HasLoS(t *testing.T) {
	def := simpleMeleeDef()
	e := entity.NewEnemy(0, 500, "test")
	e.Position = entity.Vec3{X: 0, Z: 0}
	p := testPlayer(1, entity.Vec3{X: 0, Z: 10})
	e.TargetPlayerID = p.ID

	// Clear LoS
	ctx := testCtx(def, e, testPlayers(p))
	ctx.Obs = nil
	if !condHasLoS(ctx) {
		t.Error("should have LoS without obstacles")
	}

	// Blocked LoS
	ctx.Obs = []combat.Obstacle{{CX: 0, CZ: 5, HX: 3, HZ: 1}}
	if condHasLoS(ctx) {
		t.Error("should not have LoS with obstacle blocking path")
	}

	// No target
	e.TargetPlayerID = 99
	ctx2 := testCtx(def, e, testPlayers(p))
	if condHasLoS(ctx2) {
		t.Error("should be false with no target")
	}
}

func TestCond_IsDead(t *testing.T) {
	def := simpleMeleeDef()
	e := entity.NewEnemy(0, 500, "test")

	ctx := testCtx(def, e, nil)
	if condIsDead(ctx) {
		t.Error("alive enemy should not be dead")
	}

	e.Alive = false
	if !condIsDead(ctx) {
		t.Error("dead enemy should be dead")
	}
}

func TestCond_PhaseTransitioning(t *testing.T) {
	def := simpleMeleeDef()
	e := entity.NewEnemy(0, 500, "test")

	ctx := testCtx(def, e, nil)
	if condPhaseTransitioning(ctx) {
		t.Error("should be false in chase state")
	}

	e.State = entity.EnemyPhaseTransition
	if !condPhaseTransitioning(ctx) {
		t.Error("should be true in phase transition state")
	}
}

func TestCond_InLeashRange(t *testing.T) {
	def := simpleMeleeDef()
	e := entity.NewEnemy(0, 500, "test")
	e.LeashOrigin = entity.Vec3{X: 0, Z: 0}

	ctx := testCtx(def, e, nil)

	// No leash radius → always in range
	e.LeashRadius = 0
	if !condInLeashRange(ctx) {
		t.Error("should be true with no leash radius")
	}

	// Within leash
	e.LeashRadius = 10
	e.Position = entity.Vec3{X: 5, Z: 0}
	if !condInLeashRange(ctx) {
		t.Error("should be true at distance 5 with radius 10")
	}

	// At boundary
	e.Position = entity.Vec3{X: 10, Z: 0}
	if !condInLeashRange(ctx) {
		t.Error("should be true at boundary (10 == 10)")
	}

	// Beyond leash
	e.Position = entity.Vec3{X: 11, Z: 0}
	if condInLeashRange(ctx) {
		t.Error("should be false at distance 11 with radius 10")
	}
}

func TestCond_PlayerNearby(t *testing.T) {
	def := simpleMeleeDef()
	e := entity.NewEnemy(0, 500, "test")
	e.Position = entity.Vec3{X: 0, Z: 0}

	cond := condPlayerNearby(5)

	// No players
	ctx := testCtx(def, e, testPlayers())
	if cond(ctx) {
		t.Error("should be false with no players")
	}

	// Player in range
	p := testPlayer(1, entity.Vec3{X: 3, Z: 0})
	ctx = testCtx(def, e, testPlayers(p))
	if !cond(ctx) {
		t.Error("should be true with player at distance 3 (radius 5)")
	}

	// Player out of range
	p.Position = entity.Vec3{X: 6, Z: 0}
	ctx = testCtx(def, e, testPlayers(p))
	if cond(ctx) {
		t.Error("should be false with player at distance 6 (radius 5)")
	}

	// Dead player in range
	p.Position = entity.Vec3{X: 2, Z: 0}
	p.Alive = false
	ctx = testCtx(def, e, testPlayers(p))
	if cond(ctx) {
		t.Error("should be false with dead player in range")
	}
}

func TestCond_PhaseEq(t *testing.T) {
	def := simpleMeleeDef()
	e := entity.NewEnemy(0, 500, "test")

	ctx := testCtx(def, e, nil)

	cond2 := condPhaseEq(2)
	cond1 := condPhaseEq(1)

	e.Phase = 1
	if !cond1(ctx) {
		t.Error("phase 1 should match condPhaseEq(1)")
	}
	if cond2(ctx) {
		t.Error("phase 1 should not match condPhaseEq(2)")
	}

	e.Phase = 2
	if !cond2(ctx) {
		t.Error("phase 2 should match condPhaseEq(2)")
	}
}

func TestCond_TargetBeyond(t *testing.T) {
	def := simpleMeleeDef()
	e := entity.NewEnemy(0, 500, "test")
	e.Position = entity.Vec3{X: 0, Z: 0}

	tests := []struct {
		name      string
		playerZ   float32
		threshold float32
		expected  bool
	}{
		{"beyond", 10.0, 5.0, true},
		{"at_boundary", 5.0, 5.0, false},
		{"within", 3.0, 5.0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := testPlayer(1, entity.Vec3{X: 0, Z: tt.playerZ})
			e.TargetPlayerID = p.ID
			cond := condTargetBeyond(tt.threshold)
			ctx := testCtx(def, e, testPlayers(p))
			if got := cond(ctx); got != tt.expected {
				t.Errorf("dist=%.1f threshold=%.1f: got %v, want %v", tt.playerZ, tt.threshold, got, tt.expected)
			}
		})
	}

	// No target
	t.Run("no_target", func(t *testing.T) {
		e.TargetPlayerID = 99
		cond := condTargetBeyond(5)
		ctx := testCtx(def, e, testPlayers())
		if cond(ctx) {
			t.Error("should be false with no target")
		}
	})
}

func TestCond_PlayersInAoE(t *testing.T) {
	def := simpleMeleeDef()
	e := entity.NewEnemy(0, 500, "test")
	e.Position = entity.Vec3{X: 0, Z: 0}

	cond := condPlayersInAoE(5)

	// 0 players
	t.Run("no_players", func(t *testing.T) {
		ctx := testCtx(def, e, testPlayers())
		if cond(ctx) {
			t.Error("should be false with no players")
		}
	})

	// 1 player in range (need 2+)
	t.Run("one_in_range", func(t *testing.T) {
		p := testPlayer(1, entity.Vec3{X: 3, Z: 0})
		ctx := testCtx(def, e, testPlayers(p))
		if cond(ctx) {
			t.Error("should be false with only 1 player in range")
		}
	})

	// 2 players in range
	t.Run("two_in_range", func(t *testing.T) {
		p1 := testPlayer(1, entity.Vec3{X: 2, Z: 0})
		p2 := testPlayer(2, entity.Vec3{X: -2, Z: 0})
		ctx := testCtx(def, e, testPlayers(p1, p2))
		if !cond(ctx) {
			t.Error("should be true with 2 players in range")
		}
	})

	// 2 players but one out of range
	t.Run("one_out_of_range", func(t *testing.T) {
		p1 := testPlayer(1, entity.Vec3{X: 2, Z: 0})
		p2 := testPlayer(2, entity.Vec3{X: 8, Z: 0})
		ctx := testCtx(def, e, testPlayers(p1, p2))
		if cond(ctx) {
			t.Error("should be false with only 1 player in range")
		}
	})

	// 3 players, 2 in range
	t.Run("three_players_two_in", func(t *testing.T) {
		p1 := testPlayer(1, entity.Vec3{X: 1, Z: 0})
		p2 := testPlayer(2, entity.Vec3{X: -1, Z: 0})
		p3 := testPlayer(3, entity.Vec3{X: 10, Z: 0})
		ctx := testCtx(def, e, testPlayers(p1, p2, p3))
		if !cond(ctx) {
			t.Error("should be true with 2+ players in range")
		}
	})

	// Dead player in range doesn't count
	t.Run("dead_player_ignored", func(t *testing.T) {
		p1 := testPlayer(1, entity.Vec3{X: 2, Z: 0})
		p2 := testPlayer(2, entity.Vec3{X: -2, Z: 0})
		p2.Alive = false
		ctx := testCtx(def, e, testPlayers(p1, p2))
		if cond(ctx) {
			t.Error("should be false with only 1 alive player in range")
		}
	})
}

func TestCond_ActiveAbilityIsCharge(t *testing.T) {
	def := &EnemyDef{
		Name:      "test_charge",
		MaxHealth: 500,
		Abilities: []ability.AbilityDef{
			{ID: "melee", Name: "melee", Category: ability.CategoryMelee},
			{ID: "charge", Name: "charge", Category: ability.CategoryCharge},
		},
	}
	e := entity.NewEnemy(0, 500, "test")

	ctx := testCtx(def, e, nil)

	e.ActiveAbility = 0
	if condActiveAbilityIsCharge(ctx) {
		t.Error("melee ability should not be charge")
	}

	e.ActiveAbility = 1
	if !condActiveAbilityIsCharge(ctx) {
		t.Error("charge ability should be charge")
	}

	// Out of bounds index
	e.ActiveAbility = 99
	if condActiveAbilityIsCharge(ctx) {
		t.Error("invalid index should not be charge")
	}
}

// ============================================================
// Actions
// ============================================================

func TestAction_Stop(t *testing.T) {
	def := simpleMeleeDef()
	e := entity.NewEnemy(0, 500, "test")
	e.Velocity = entity.Vec3{X: 5, Z: 3}

	ctx := testCtx(def, e, nil)
	r := actionStop(ctx)
	if r != bt.Success {
		t.Errorf("want Success, got %v", r)
	}
	if e.Velocity.X != 0 || e.Velocity.Z != 0 {
		t.Errorf("velocity should be zeroed, got %v", e.Velocity)
	}
}

func TestAction_AggroNearest(t *testing.T) {
	def := simpleMeleeDef()
	e := entity.NewEnemy(0, 500, "test")
	e.Position = entity.Vec3{X: 0, Z: 0}

	// No players
	ctx := testCtx(def, e, testPlayers())
	if r := actionAggroNearest(ctx); r != bt.Failure {
		t.Errorf("should fail with no players, got %v", r)
	}

	// With players
	p1 := testPlayer(1, entity.Vec3{X: 10, Z: 0})
	p2 := testPlayer(2, entity.Vec3{X: 3, Z: 0})
	ctx = testCtx(def, e, testPlayers(p1, p2))
	if r := actionAggroNearest(ctx); r != bt.Success {
		t.Errorf("should succeed with players, got %v", r)
	}
	if e.TargetPlayerID != 2 {
		t.Errorf("should target nearest (p2), got %d", e.TargetPlayerID)
	}
	if e.State != entity.EnemyChase {
		t.Errorf("should set state to chase, got %d", e.State)
	}
}

func TestAction_WaitTransition(t *testing.T) {
	def := simpleMeleeDef()
	e := entity.NewEnemy(0, 500, "test")
	e.State = entity.EnemyPhaseTransition
	e.Velocity = entity.Vec3{X: 3, Z: 3}

	ctx := testCtx(def, e, nil)

	// Timer still running
	e.StateTimer = 0.5
	if r := actionWaitTransition(ctx); r != bt.Running {
		t.Errorf("should return Running while timer > 0, got %v", r)
	}
	if e.Velocity.X != 0 || e.Velocity.Z != 0 {
		t.Error("should zero velocity during transition")
	}

	// Timer expired
	e.StateTimer = 0
	if r := actionWaitTransition(ctx); r != bt.Success {
		t.Errorf("should return Success when timer <= 0, got %v", r)
	}
	if e.State != entity.EnemyChase {
		t.Errorf("should set state to chase, got %d", e.State)
	}
}

func TestAction_LeashReset(t *testing.T) {
	def := simpleMeleeDef()
	e := entity.NewEnemy(0, 500, "test")
	e.LeashOrigin = entity.Vec3{X: 5, Z: 5}
	e.Position = entity.Vec3{X: 20, Z: 20}
	e.Health = 100
	e.ThreatTable = map[uint16]float32{1: 50, 2: 30}
	e.Velocity = entity.Vec3{X: 3, Z: 3}

	ctx := testCtx(def, e, nil)
	ctx.BB.Set("last_attack", "melee")

	r := actionLeashReset(ctx)
	if r != bt.Success {
		t.Errorf("want Success, got %v", r)
	}
	if e.Position != e.LeashOrigin {
		t.Errorf("should teleport to leash origin, got %v", e.Position)
	}
	if e.Health != e.MaxHealth {
		t.Errorf("should heal to full, got %f", e.Health)
	}
	if e.State != entity.EnemyPatrol {
		t.Errorf("should enter patrol, got %d", e.State)
	}
	if e.Velocity.X != 0 || e.Velocity.Z != 0 {
		t.Error("should zero velocity")
	}
	if len(e.ThreatTable) != 0 {
		t.Error("should clear threat table")
	}
	if ctx.BB.GetString("last_attack") != "" {
		t.Error("should clear last_attack from blackboard")
	}
}

func TestAction_Patrol_Movement(t *testing.T) {
	def := simpleMeleeDef()
	e := entity.NewEnemy(0, 500, "test")
	e.Position = entity.Vec3{X: 0, Z: 0}
	e.PatrolA = entity.Vec3{X: -10, Z: 0}
	e.PatrolB = entity.Vec3{X: 10, Z: 0}
	e.PatrolTarget = 0 // heading toward PatrolA
	e.AggroRadius = 5.0

	ctx := testCtx(def, e, testPlayers())
	r := actionPatrol(ctx)
	if r != bt.Running {
		t.Errorf("should return Running while patrolling, got %v", r)
	}
	if e.Velocity.X >= 0 {
		t.Errorf("should move toward PatrolA (X-), got velocity %v", e.Velocity)
	}
}

func TestAction_Patrol_FlipsAtWaypoint(t *testing.T) {
	def := simpleMeleeDef()
	e := entity.NewEnemy(0, 500, "test")
	e.PatrolA = entity.Vec3{X: -10, Z: 0}
	e.PatrolB = entity.Vec3{X: 10, Z: 0}
	e.PatrolTarget = 0
	e.AggroRadius = 5.0
	// Place very close to PatrolA (within 0.5 threshold)
	e.Position = entity.Vec3{X: -9.8, Z: 0}

	ctx := testCtx(def, e, testPlayers())
	actionPatrol(ctx)

	if e.PatrolTarget != 1 {
		t.Errorf("should flip to PatrolB, got target %d", e.PatrolTarget)
	}
}

func TestAction_Patrol_Aggro(t *testing.T) {
	def := simpleMeleeDef()
	e := entity.NewEnemy(0, 500, "test")
	e.Position = entity.Vec3{X: 0, Z: 0}
	e.PatrolA = entity.Vec3{X: -10, Z: 0}
	e.PatrolB = entity.Vec3{X: 10, Z: 0}
	e.AggroRadius = 5.0

	p := testPlayer(1, entity.Vec3{X: 3, Z: 0})
	ctx := testCtx(def, e, testPlayers(p))
	r := actionPatrol(ctx)
	if r != bt.Success {
		t.Errorf("should return Success on aggro, got %v", r)
	}
	if e.State != entity.EnemyChase {
		t.Errorf("should enter chase, got %d", e.State)
	}
	if e.TargetPlayerID != p.ID {
		t.Errorf("should target aggroing player, got %d", e.TargetPlayerID)
	}
}

func TestAction_Patrol_IgnoresDeadPlayers(t *testing.T) {
	def := simpleMeleeDef()
	e := entity.NewEnemy(0, 500, "test")
	e.Position = entity.Vec3{X: 0, Z: 0}
	e.PatrolA = entity.Vec3{X: -10, Z: 0}
	e.PatrolB = entity.Vec3{X: 10, Z: 0}
	e.AggroRadius = 5.0

	p := testPlayer(1, entity.Vec3{X: 2, Z: 0})
	p.Alive = false
	ctx := testCtx(def, e, testPlayers(p))
	r := actionPatrol(ctx)
	if r != bt.Running {
		t.Errorf("should patrol (not aggro dead player), got %v", r)
	}
}

func TestAction_Chase_MeleeClosesDistance(t *testing.T) {
	def := simpleMeleeDef()
	e := entity.NewEnemy(0, 500, "test")
	e.Position = entity.Vec3{X: 0, Z: 0}
	e.Alive = true
	e.State = entity.EnemyChase

	p := testPlayer(1, entity.Vec3{X: 0, Z: 10})
	ctx := testCtx(def, e, testPlayers(p))
	r := actionChase(ctx)
	if r != bt.Running {
		t.Errorf("should return Running while closing, got %v", r)
	}
	if e.Velocity.Z <= 0 {
		t.Errorf("should move toward target (Z+), got %v", e.Velocity)
	}
}

func TestAction_Chase_MeleeReturnsSuccessInRange(t *testing.T) {
	def := simpleMeleeDef()
	e := entity.NewEnemy(0, 500, "test")
	e.Position = entity.Vec3{X: 0, Z: 0}
	e.Alive = true
	e.State = entity.EnemyChase

	// Player at distance 2.0, melee range is 3.0, threshold is 3.0*0.8=2.4
	p := testPlayer(1, entity.Vec3{X: 0, Z: 2})
	ctx := testCtx(def, e, testPlayers(p))
	r := actionChase(ctx)
	if r != bt.Success {
		t.Errorf("should return Success when in melee range, got %v", r)
	}
	if e.Velocity.X != 0 || e.Velocity.Z != 0 {
		t.Errorf("should stop in melee range, got %v", e.Velocity)
	}
}

func TestAction_Chase_RangedBackpedals(t *testing.T) {
	def := &EnemyDef{
		Name:           "ranged",
		MaxHealth:      200,
		MoveSpeed:      3.0,
		PreferredRange: 8.0,
		BackpedalSpeed: 2.5,
		Radius:         1.0,
		Abilities:      []ability.AbilityDef{{ID: "bolt", Name: "bolt", Category: ability.CategoryRanged}},
	}
	e := entity.NewEnemy(0, 200, "ranged")
	e.Position = entity.Vec3{X: 0, Z: 0}
	e.Alive = true
	e.State = entity.EnemyChase

	// Player too close (distance 3 < preferred-margin 6.4)
	p := testPlayer(1, entity.Vec3{X: 0, Z: 3})
	ctx := testCtx(def, e, testPlayers(p))
	r := actionChase(ctx)
	if r != bt.Running {
		t.Errorf("should return Running while backpedaling, got %v", r)
	}
	if e.Velocity.Z >= 0 {
		t.Errorf("should backpedal (Z-), got %v", e.Velocity)
	}
}

func TestAction_Chase_RangedAdvances(t *testing.T) {
	def := &EnemyDef{
		Name:           "ranged",
		MaxHealth:      200,
		MoveSpeed:      3.0,
		PreferredRange: 8.0,
		BackpedalSpeed: 2.5,
		Radius:         1.0,
		Abilities:      []ability.AbilityDef{{ID: "bolt", Name: "bolt", Category: ability.CategoryRanged}},
	}
	e := entity.NewEnemy(0, 200, "ranged")
	e.Position = entity.Vec3{X: 0, Z: 0}
	e.Alive = true
	e.State = entity.EnemyChase

	// Player too far (distance 15 > preferred+margin 9.6)
	p := testPlayer(1, entity.Vec3{X: 0, Z: 15})
	ctx := testCtx(def, e, testPlayers(p))
	r := actionChase(ctx)
	if r != bt.Running {
		t.Errorf("should return Running while advancing, got %v", r)
	}
	if e.Velocity.Z <= 0 {
		t.Errorf("should advance (Z+), got %v", e.Velocity)
	}
}

func TestAction_Chase_RangedStopsInSweetSpot(t *testing.T) {
	def := &EnemyDef{
		Name:           "ranged",
		MaxHealth:      200,
		MoveSpeed:      3.0,
		PreferredRange: 8.0,
		BackpedalSpeed: 2.5,
		Radius:         1.0,
		Abilities:      []ability.AbilityDef{{ID: "bolt", Name: "bolt", Category: ability.CategoryRanged}},
	}
	e := entity.NewEnemy(0, 200, "ranged")
	e.Position = entity.Vec3{X: 0, Z: 0}
	e.Alive = true
	e.State = entity.EnemyChase

	// Player at preferred range (distance 8, within margin ±1.6)
	p := testPlayer(1, entity.Vec3{X: 0, Z: 8})
	ctx := testCtx(def, e, testPlayers(p))
	r := actionChase(ctx)
	if r != bt.Running {
		t.Errorf("should return Running in sweet spot, got %v", r)
	}
	if e.Velocity.X != 0 || e.Velocity.Z != 0 {
		t.Errorf("should stop in sweet spot, got %v", e.Velocity)
	}
}

func TestAction_Chase_FailsWhenDead(t *testing.T) {
	def := simpleMeleeDef()
	e := entity.NewEnemy(0, 500, "test")
	e.Alive = false
	e.Velocity = entity.Vec3{X: 5, Z: 5}

	p := testPlayer(1, entity.Vec3{X: 0, Z: 5})
	ctx := testCtx(def, e, testPlayers(p))
	r := actionChase(ctx)
	if r != bt.Failure {
		t.Errorf("should fail when dead, got %v", r)
	}
	if e.Velocity.X != 0 || e.Velocity.Z != 0 {
		t.Error("should zero velocity on failure")
	}
}

func TestAction_Chase_FailsNoPlayers(t *testing.T) {
	def := simpleMeleeDef()
	e := entity.NewEnemy(0, 500, "test")
	e.Alive = true
	e.State = entity.EnemyChase

	ctx := testCtx(def, e, testPlayers())
	r := actionChase(ctx)
	if r != bt.Failure {
		t.Errorf("should fail with no players, got %v", r)
	}
}

func TestAction_SelectAbility_Success(t *testing.T) {
	def := simpleMeleeDef()
	e := entity.NewEnemy(0, 500, "test")
	e.Alive = true
	e.State = entity.EnemyChase

	p := testPlayer(1, entity.Vec3{X: 0, Z: 2})
	e.TargetPlayerID = p.ID
	ctx := testCtx(def, e, testPlayers(p))
	ctx.Rng = rand.New(rand.NewPCG(42, 42))

	r := actionSelectAbility(ctx)
	if r != bt.Success {
		t.Errorf("should succeed, got %v", r)
	}
	if e.State != entity.EnemyMeleeTelegraph {
		t.Errorf("should enter telegraph, got state %d", e.State)
	}
}

func TestAction_SelectAbility_FailsNoValidAbility(t *testing.T) {
	def := &EnemyDef{
		Name:      "test",
		MaxHealth: 500,
		Abilities: []ability.AbilityDef{
			{ID: "bolt", Name: "bolt", Category: ability.CategoryRanged, BaseWeight: 100, MinRange: 5.0},
		},
	}
	e := entity.NewEnemy(0, 500, "test")
	e.Alive = true

	// Player at distance 2 — below MinRange 5
	p := testPlayer(1, entity.Vec3{X: 0, Z: 2})
	e.TargetPlayerID = p.ID
	ctx := testCtx(def, e, testPlayers(p))
	ctx.Rng = rand.New(rand.NewPCG(42, 42))

	r := actionSelectAbility(ctx)
	if r != bt.Failure {
		t.Errorf("should fail when no ability in range, got %v", r)
	}
}

func TestAction_SelectAbility_FailsWhenDead(t *testing.T) {
	def := simpleMeleeDef()
	e := entity.NewEnemy(0, 500, "test")
	e.Alive = false

	p := testPlayer(1, entity.Vec3{X: 0, Z: 2})
	e.TargetPlayerID = p.ID
	ctx := testCtx(def, e, testPlayers(p))

	r := actionSelectAbility(ctx)
	if r != bt.Failure {
		t.Errorf("should fail when dead, got %v", r)
	}
}

func TestAction_Telegraph_WaitsForTimer(t *testing.T) {
	def := simpleMeleeDef()
	e := entity.NewEnemy(0, 500, "test")
	e.Alive = true
	e.State = entity.EnemyMeleeTelegraph
	e.StateTimer = 0.3
	e.ActiveAbility = 0
	e.Velocity = entity.Vec3{X: 5, Z: 5}

	ctx := testCtx(def, e, testPlayers())
	r := actionTelegraph(ctx)
	if r != bt.Running {
		t.Errorf("should return Running while timer > 0, got %v", r)
	}
	if e.Velocity.X != 0 || e.Velocity.Z != 0 {
		t.Error("should zero velocity during telegraph")
	}
}

func TestAction_Telegraph_CompletesWhenTimerExpires(t *testing.T) {
	def := simpleMeleeDef()
	e := entity.NewEnemy(0, 500, "test")
	e.Alive = true
	e.State = entity.EnemyMeleeTelegraph
	e.StateTimer = 0
	e.ActiveAbility = 0

	ctx := testCtx(def, e, testPlayers())
	r := actionTelegraph(ctx)
	if r != bt.Success {
		t.Errorf("should return Success when timer <= 0, got %v", r)
	}
}

func TestAction_Telegraph_FailsWhenAborted(t *testing.T) {
	def := simpleMeleeDef()
	e := entity.NewEnemy(0, 500, "test")
	e.Alive = false
	e.State = entity.EnemyMeleeTelegraph
	e.StateTimer = 0.5
	e.ActiveAbility = 0

	ctx := testCtx(def, e, testPlayers())
	r := actionTelegraph(ctx)
	if r != bt.Failure {
		t.Errorf("should fail when dead, got %v", r)
	}
}

func TestAction_Telegraph_TrackTargetRanged(t *testing.T) {
	def := &EnemyDef{
		Name:      "test",
		MaxHealth: 500,
		Abilities: []ability.AbilityDef{
			{ID: "bolt", Name: "bolt", Category: ability.CategoryRanged, TrackTarget: true, CommitTime: 1.0},
		},
	}
	e := entity.NewEnemy(0, 500, "test")
	e.Alive = true
	e.State = entity.EnemyRangedTelegraph
	e.StateTimer = 0.5
	e.ActiveAbility = 0
	e.RangedTargetPos = entity.Vec3{X: 0, Z: 10, Y: 1}

	p := testPlayer(1, entity.Vec3{X: 5, Z: 15})
	e.TargetPlayerID = p.ID
	ctx := testCtx(def, e, testPlayers(p))

	actionTelegraph(ctx)
	if e.RangedTargetPos.X != 5 || e.RangedTargetPos.Z != 15 {
		t.Errorf("should update target pos, got %v", e.RangedTargetPos)
	}
}

func TestAction_ExecuteAbility_TransitionsFromTelegraph(t *testing.T) {
	def := simpleMeleeDef()
	e := entity.NewEnemy(0, 500, "test")
	e.Alive = true
	e.State = entity.EnemyMeleeTelegraph
	e.ActiveAbility = 0

	ctx := testCtx(def, e, testPlayers())
	r := actionExecuteAbility(ctx)
	if r != bt.Running {
		t.Errorf("first tick should return Running (entering attack state), got %v", r)
	}
	if e.State != entity.EnemyMeleeAttack {
		t.Errorf("should transition to MeleeAttack, got %d", e.State)
	}
	if e.StateTimer != 0.2 {
		t.Errorf("should set execute time 0.2, got %f", e.StateTimer)
	}
}

func TestAction_ExecuteAbility_WaitsForExecuteTimer(t *testing.T) {
	def := simpleMeleeDef()
	e := entity.NewEnemy(0, 500, "test")
	e.Alive = true
	e.State = entity.EnemyMeleeAttack
	e.StateTimer = 0.1
	e.ActiveAbility = 0

	ctx := testCtx(def, e, testPlayers())
	r := actionExecuteAbility(ctx)
	if r != bt.Running {
		t.Errorf("should return Running while execute timer > 0, got %v", r)
	}
}

func TestAction_ExecuteAbility_ResolvesDamage(t *testing.T) {
	def := simpleMeleeDef()
	e := entity.NewEnemy(0, 500, "test")
	e.Alive = true
	e.State = entity.EnemyMeleeAttack
	e.StateTimer = 0
	e.ActiveAbility = 0
	e.Position = entity.Vec3{X: 0, Z: 0}
	e.RotationY = 0 // facing -Z

	// Player within melee range and cone
	p := testPlayer(1, entity.Vec3{X: 0, Z: -2})
	events := &[]combat.DamageEvent{}
	ctx := testCtx(def, e, testPlayers(p))
	ctx.Events = events

	r := actionExecuteAbility(ctx)
	if r != bt.Success {
		t.Errorf("should return Success after resolving, got %v", r)
	}
	if len(*events) == 0 {
		t.Error("should have generated damage events")
	}
}

func TestAction_ChargeDash_InitializesCharge(t *testing.T) {
	def := &EnemyDef{
		Name:      "test",
		MaxHealth: 500,
		Radius:    1.0,
		Abilities: []ability.AbilityDef{
			{
				ID: "charge", Name: "charge", Category: ability.CategoryCharge,
				Charge: &ability.ChargeDef{
					Speed: 10, Damage: 20, MaxDistance: 15,
					HitRadius: 2.0, StopOnWall: true,
				},
			},
		},
	}
	e := entity.NewEnemy(0, 500, "test")
	e.Alive = true
	e.State = entity.EnemyChargeTelegraph
	e.ActiveAbility = 0
	e.ChargeDirection = entity.Vec3{X: 0, Z: 1}

	ctx := testCtx(def, e, testPlayers())
	r := actionChargeDash(ctx)
	if r != bt.Running {
		t.Errorf("first tick should return Running, got %v", r)
	}
	if e.State != entity.EnemyCharge {
		t.Errorf("should enter charge state, got %d", e.State)
	}
	// First tick: init + movement, so distance = speed * dt = 10 * 0.05 = 0.5
	if e.ChargeDistance != 10*0.05 {
		t.Errorf("charge distance should be speed*dt, got %f", e.ChargeDistance)
	}
}

func TestAction_ChargeDash_StopsAtMaxDistance(t *testing.T) {
	def := &EnemyDef{
		Name:      "test",
		MaxHealth: 500,
		Radius:    1.0,
		Abilities: []ability.AbilityDef{
			{
				ID: "charge", Name: "charge", Category: ability.CategoryCharge,
				Charge: &ability.ChargeDef{
					Speed: 10, Damage: 20, MaxDistance: 5,
					HitRadius: 2.0,
				},
			},
		},
	}
	e := entity.NewEnemy(0, 500, "test")
	e.Alive = true
	e.State = entity.EnemyCharge
	e.ActiveAbility = 0
	e.ChargeDirection = entity.Vec3{X: 0, Z: 1}
	e.ChargeDistance = 4.9
	e.ChargeHitPlayers = []uint16{}

	ctx := testCtx(def, e, testPlayers())
	r := actionChargeDash(ctx)
	// ChargeDistance = 4.9 + 10*0.05 = 5.4 >= 5.0 → stop
	if r != bt.Success {
		t.Errorf("should stop at max distance, got %v", r)
	}
	if e.Velocity.X != 0 || e.Velocity.Z != 0 {
		t.Error("should zero velocity on stop")
	}
}

func TestAction_ChargeDash_StopsAtWall(t *testing.T) {
	def := &EnemyDef{
		Name:      "test",
		MaxHealth: 500,
		Radius:    1.0,
		Abilities: []ability.AbilityDef{
			{
				ID: "charge", Name: "charge", Category: ability.CategoryCharge,
				Charge: &ability.ChargeDef{
					Speed: 10, Damage: 20, MaxDistance: 100,
					HitRadius: 2.0, StopOnWall: true,
				},
			},
		},
	}
	e := entity.NewEnemy(0, 500, "test")
	e.Alive = true
	e.State = entity.EnemyCharge
	e.ActiveAbility = 0
	e.ChargeDirection = entity.Vec3{X: 1, Z: 0}
	e.ChargeDistance = 0
	e.ChargeHitPlayers = []uint16{}
	// Position at the wall boundary
	e.Position = entity.Vec3{X: 19.6, Z: 0}

	ctx := testCtx(def, e, testPlayers())
	r := actionChargeDash(ctx)
	if r != bt.Success {
		t.Errorf("should stop at wall, got %v", r)
	}
}

func TestAction_ChargeDash_HitsPlayerOnce(t *testing.T) {
	def := &EnemyDef{
		Name:      "test",
		MaxHealth: 500,
		Radius:    1.0,
		Abilities: []ability.AbilityDef{
			{
				ID: "charge", Name: "charge", Category: ability.CategoryCharge,
				Charge: &ability.ChargeDef{
					Speed: 10, Damage: 20, MaxDistance: 100,
					HitRadius: 3.0,
				},
				DamageSource: combat.SourceEnemyCharge,
			},
		},
	}
	e := entity.NewEnemy(0, 500, "test")
	e.Alive = true
	e.State = entity.EnemyCharge
	e.ActiveAbility = 0
	e.ChargeDirection = entity.Vec3{X: 0, Z: 1}
	e.ChargeDistance = 0
	e.ChargeHitPlayers = []uint16{}
	e.Position = entity.Vec3{X: 0, Z: 0}

	p := testPlayer(1, entity.Vec3{X: 0, Z: 1})
	events := &[]combat.DamageEvent{}
	ctx := testCtx(def, e, testPlayers(p))
	ctx.Events = events

	// First tick: hits player
	actionChargeDash(ctx)
	if len(*events) != 1 {
		t.Fatalf("should hit player once, got %d events", len(*events))
	}
	if (*events)[0].Amount != 20 {
		t.Errorf("should deal 20 damage, got %f", (*events)[0].Amount)
	}

	// Second tick: same player should NOT be hit again
	*events = (*events)[:0]
	actionChargeDash(ctx)
	if len(*events) != 0 {
		t.Error("should not double-hit the same player")
	}
}

func TestAction_Cooldown_EntersAndWaits(t *testing.T) {
	def := simpleMeleeDef()
	e := entity.NewEnemy(0, 500, "test")
	e.Alive = true
	e.State = entity.EnemyMeleeAttack
	e.ActiveAbility = 0

	ctx := testCtx(def, e, testPlayers())

	// First tick: enters cooldown
	r := actionCooldown(ctx)
	if r != bt.Running {
		t.Errorf("should return Running on cooldown entry, got %v", r)
	}
	if e.State != entity.EnemyCooldown {
		t.Errorf("should enter cooldown state, got %d", e.State)
	}

	// Timer expires
	e.StateTimer = 0
	r = actionCooldown(ctx)
	if r != bt.Success {
		t.Errorf("should return Success when cooldown done, got %v", r)
	}
	if e.State != entity.EnemyChase {
		t.Errorf("should return to chase, got %d", e.State)
	}
}

func TestAction_Cooldown_FailsWhenAborted(t *testing.T) {
	def := simpleMeleeDef()
	e := entity.NewEnemy(0, 500, "test")
	e.Alive = false
	e.State = entity.EnemyCooldown
	e.StateTimer = 1.0

	ctx := testCtx(def, e, testPlayers())
	r := actionCooldown(ctx)
	if r != bt.Failure {
		t.Errorf("should fail when dead, got %v", r)
	}
}

// ============================================================
// Runner-based leaves
// ============================================================

func TestCond_IsCasting(t *testing.T) {
	def := simpleMeleeDef()
	e := entity.NewEnemy(0, 500, "test")
	e.TargetPlayerID = 1
	p := testPlayer(1, entity.Vec3{X: 0, Z: 2})

	ctx := testCtx(def, e, testPlayers(p))
	if condIsCasting(ctx) {
		t.Error("should be false when idle")
	}

	ctx.Cast("melee")
	if !condIsCasting(ctx) {
		t.Error("should be true when casting")
	}
}

func TestCond_IsCommitted(t *testing.T) {
	def := simpleMeleeDef()
	e := entity.NewEnemy(0, 500, "test")
	e.TargetPlayerID = 1
	p := testPlayer(1, entity.Vec3{X: 0, Z: 2})

	ctx := testCtx(def, e, testPlayers(p))
	if condIsCommitted(ctx) {
		t.Error("should be false when idle")
	}

	ctx.Cast("melee")
	if !condIsCommitted(ctx) {
		t.Error("should be true during commit")
	}
}

func TestCond_CanCast(t *testing.T) {
	def := simpleMeleeDef()
	e := entity.NewEnemy(0, 500, "test")
	e.TargetPlayerID = 1
	p := testPlayer(1, entity.Vec3{X: 0, Z: 2})

	ctx := testCtx(def, e, testPlayers(p))
	if !condCanCast(ctx) {
		t.Error("should be true when idle")
	}

	ctx.Cast("melee")
	if condCanCast(ctx) {
		t.Error("should be false when busy")
	}
}

func TestCond_CanMove(t *testing.T) {
	def := simpleMeleeDef()
	e := entity.NewEnemy(0, 500, "test")
	e.TargetPlayerID = 1
	p := testPlayer(1, entity.Vec3{X: 0, Z: 2})

	ctx := testCtx(def, e, testPlayers(p))

	// Idle → can move
	if !condCanMove(ctx) {
		t.Error("should be true when idle")
	}

	// Committed with CanMoveCommitted=false → cannot move
	ctx.Cast("melee")
	if condCanMove(ctx) {
		t.Error("should be false during commit with CanMoveCommitted=false")
	}
}

func TestAction_CastWeighted_Success(t *testing.T) {
	def := simpleMeleeDef()
	e := entity.NewEnemy(0, 500, "test")
	e.Alive = true
	e.State = entity.EnemyChase
	e.TargetPlayerID = 1
	p := testPlayer(1, entity.Vec3{X: 0, Z: 2})

	ctx := testCtx(def, e, testPlayers(p))

	r := actionCastWeighted(ctx)
	if r != bt.Success {
		t.Errorf("should succeed, got %v", r)
	}
	if ctx.Runner.Phase != RunnerCommit {
		t.Error("runner should be in commit")
	}
}

func TestAction_CastWeighted_FailsWhenBusy(t *testing.T) {
	def := simpleMeleeDef()
	e := entity.NewEnemy(0, 500, "test")
	e.Alive = true
	e.State = entity.EnemyChase
	e.TargetPlayerID = 1
	p := testPlayer(1, entity.Vec3{X: 0, Z: 2})

	ctx := testCtx(def, e, testPlayers(p))
	ctx.Cast("melee")

	r := actionCastWeighted(ctx)
	if r != bt.Failure {
		t.Errorf("should fail when runner is busy, got %v", r)
	}
}

func TestAction_WaitAbility(t *testing.T) {
	def := simpleMeleeDef()
	e := entity.NewEnemy(0, 500, "test")
	e.TargetPlayerID = 1
	p := testPlayer(1, entity.Vec3{X: 0, Z: 2})

	ctx := testCtx(def, e, testPlayers(p))

	// Idle → immediate success
	r := actionWaitAbility(ctx)
	if r != bt.Success {
		t.Errorf("should return Success when idle, got %v", r)
	}

	// Busy → running
	ctx.Cast("melee")
	r = actionWaitAbility(ctx)
	if r != bt.Running {
		t.Errorf("should return Running when busy, got %v", r)
	}
}

func TestAction_CancelAbility(t *testing.T) {
	def := simpleMeleeDef()
	def.Abilities[0].Cancellable = true
	e := entity.NewEnemy(0, 500, "test")
	e.TargetPlayerID = 1
	p := testPlayer(1, entity.Vec3{X: 0, Z: 2})

	ctx := testCtx(def, e, testPlayers(p))
	ctx.Cast("melee")

	r := actionCancelAbility(ctx)
	if r != bt.Success {
		t.Errorf("should succeed for cancellable ability, got %v", r)
	}
}

func TestAction_CastByName(t *testing.T) {
	def := simpleMeleeDef()
	e := entity.NewEnemy(0, 500, "test")
	e.Alive = true
	e.State = entity.EnemyChase
	e.TargetPlayerID = 1
	p := testPlayer(1, entity.Vec3{X: 0, Z: 2})

	ctx := testCtx(def, e, testPlayers(p))

	castMelee := castByName("melee")
	r := castMelee(ctx)
	if r != bt.Success {
		t.Errorf("should succeed, got %v", r)
	}
	if ctx.CurrentAbilityID() != "melee" {
		t.Errorf("expected 'melee', got %q", ctx.CurrentAbilityID())
	}
}

func TestAction_CastByName_UnknownAbility(t *testing.T) {
	def := simpleMeleeDef()
	e := entity.NewEnemy(0, 500, "test")
	e.State = entity.EnemyChase

	ctx := testCtx(def, e, nil)

	castBogus := castByName("nonexistent")
	r := castBogus(ctx)
	if r != bt.Failure {
		t.Errorf("should fail for unknown ability, got %v", r)
	}
}
