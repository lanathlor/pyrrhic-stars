package enemyai

import (
	"testing"

	"codex-online/server/internal/ability"
	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
)

const testMeleeID = "melee"

func meleeRunnerDef() *EnemyDef {
	return &EnemyDef{
		Name:      "test_runner",
		MaxHealth: 500,
		MoveSpeed: 4.0,
		Radius:    1.0,
		Abilities: []ability.AbilityDef{
			{
				ID: testMeleeID, Name: testMeleeID, Category: ability.CategoryMelee,
				CommitTime: 0.5, ExecuteTime: 0.2, Cooldown: 0.5,
				BaseWeight: 100, MaxRange: 3.0,
				BaseDamage:   20.0,
				Hit:          ability.HitDef{Type: ability.HitAoECone, Range: 3.0, ArcDegrees: 180},
				DamageSource: combat.SourceEnemyMelee,
			},
		},
	}
}

func chargeRunnerDef() *EnemyDef {
	return &EnemyDef{
		Name:      "test_runner_charge",
		MaxHealth: 500,
		MoveSpeed: 4.0,
		Radius:    1.0,
		Abilities: []ability.AbilityDef{
			{
				ID: "charge", Name: "charge", Category: ability.CategoryCharge,
				CommitTime: 0.5, Cooldown: 1.0,
				BaseWeight: 100, MinRange: 5.0,
				Charge: &ability.ChargeDef{
					Speed:       12.0,
					Damage:      30.0,
					MaxDistance:  15.0,
					HitRadius:   2.0,
					StopOnWall:  true,
				},
				DamageSource: combat.SourceEnemyCharge,
				FaceTarget:   true,
			},
		},
	}
}

func TestRunner_MeleeLifecycle(t *testing.T) {
	def := meleeRunnerDef()
	e := entity.NewEnemy(0, 500, "test")
	e.State = entity.EnemyChase
	e.TargetPlayerID = 1
	p := testPlayer(1, entity.Vec3{X: 0, Z: 2})

	ctx := testCtx(def, e, testPlayers(p))

	// Cast melee ability
	if !ctx.Cast(testMeleeID) {
		t.Fatal("Cast should succeed when runner is idle")
	}
	if ctx.Runner.Phase != RunnerCommit {
		t.Fatalf("expected RunnerCommit, got %d", ctx.Runner.Phase)
	}
	if e.State != entity.EnemyMeleeTelegraph {
		t.Fatalf("expected MeleeTelegraph, got %d", e.State)
	}
	if e.StateTimer != 0.5 {
		t.Fatalf("expected StateTimer=0.5, got %f", e.StateTimer)
	}

	// Double cast rejected
	if ctx.Cast(testMeleeID) {
		t.Error("double Cast should fail when runner is busy")
	}

	// Tick through commit phase (10 ticks @ 0.05 = 0.5s)
	for i := 0; i < 10; i++ {
		e.StateTimer -= 0.05
		ctx.Runner.Tick(ctx)
	}

	if ctx.Runner.Phase != RunnerExecute {
		t.Fatalf("expected RunnerExecute after commit, got %d", ctx.Runner.Phase)
	}
	if e.State != entity.EnemyMeleeAttack {
		t.Fatalf("expected MeleeAttack, got %d", e.State)
	}

	// Tick through execute phase (0.2s execute time)
	for i := 0; i < 20; i++ {
		e.StateTimer -= 0.05
		ctx.Runner.Tick(ctx)
		if ctx.Runner.Phase != RunnerExecute {
			break
		}
	}

	if ctx.Runner.Phase != RunnerCooldown {
		t.Fatalf("expected RunnerCooldown after execute, got %d (timer=%f)", ctx.Runner.Phase, e.StateTimer)
	}
	if e.State != entity.EnemyCooldown {
		t.Fatalf("expected EnemyCooldown, got %d", e.State)
	}

	// Tick through cooldown (10 ticks @ 0.05 = 0.5s)
	for i := 0; i < 10; i++ {
		e.StateTimer -= 0.05
		ctx.Runner.Tick(ctx)
	}

	if ctx.Runner.Phase != RunnerIdle {
		t.Fatalf("expected RunnerIdle after cooldown, got %d", ctx.Runner.Phase)
	}
	if e.State != entity.EnemyChase {
		t.Fatalf("expected EnemyChase after cooldown, got %d", e.State)
	}
}

func TestRunner_CancelDuringCommit(t *testing.T) {
	def := meleeRunnerDef()
	def.Abilities[0].Cancellable = true

	e := entity.NewEnemy(0, 500, "test")
	e.State = entity.EnemyChase
	e.TargetPlayerID = 1
	p := testPlayer(1, entity.Vec3{X: 0, Z: 2})

	ctx := testCtx(def, e, testPlayers(p))
	ctx.Cast(testMeleeID)

	if ctx.Runner.Phase != RunnerCommit {
		t.Fatal("should be in commit phase")
	}

	if !ctx.CancelAbility() {
		t.Fatal("cancel should succeed for cancellable ability in commit phase")
	}
	if ctx.Runner.Phase != RunnerIdle {
		t.Fatalf("expected RunnerIdle after cancel, got %d", ctx.Runner.Phase)
	}
	if e.State != entity.EnemyChase {
		t.Fatalf("expected EnemyChase after cancel, got %d", e.State)
	}
}

func TestRunner_CancelRejected_NotCancellable(t *testing.T) {
	def := meleeRunnerDef()
	// Cancellable defaults to false

	e := entity.NewEnemy(0, 500, "test")
	e.State = entity.EnemyChase
	e.TargetPlayerID = 1
	p := testPlayer(1, entity.Vec3{X: 0, Z: 2})

	ctx := testCtx(def, e, testPlayers(p))
	ctx.Cast(testMeleeID)

	if ctx.CancelAbility() {
		t.Error("cancel should fail for non-cancellable ability")
	}
	if ctx.Runner.Phase != RunnerCommit {
		t.Error("should still be in commit phase")
	}
}

func TestRunner_CancelRejected_DuringExecute(t *testing.T) {
	def := meleeRunnerDef()
	def.Abilities[0].Cancellable = true

	e := entity.NewEnemy(0, 500, "test")
	e.State = entity.EnemyChase
	e.TargetPlayerID = 1
	p := testPlayer(1, entity.Vec3{X: 0, Z: 2})

	ctx := testCtx(def, e, testPlayers(p))
	ctx.Cast(testMeleeID)

	// Tick past commit
	for i := 0; i < 10; i++ {
		e.StateTimer -= 0.05
		ctx.Runner.Tick(ctx)
	}
	if ctx.Runner.Phase != RunnerExecute {
		t.Fatal("should be in execute phase")
	}

	if ctx.CancelAbility() {
		t.Error("cancel should fail during execute phase even for cancellable ability")
	}
}

func TestRunner_MovementEnforcement(t *testing.T) {
	def := meleeRunnerDef()
	// CanMoveCommitted defaults to false

	e := entity.NewEnemy(0, 500, "test")
	e.State = entity.EnemyChase
	e.TargetPlayerID = 1
	p := testPlayer(1, entity.Vec3{X: 0, Z: 2})

	ctx := testCtx(def, e, testPlayers(p))
	ctx.Cast(testMeleeID)

	// Set velocity as if BT tried to move
	e.Velocity = entity.Vec3{X: 5, Z: 5}
	e.StateTimer -= 0.05
	ctx.Runner.Tick(ctx)

	// Velocity should be zeroed by runner during commit
	if e.Velocity.X != 0 || e.Velocity.Z != 0 {
		t.Errorf("velocity should be zeroed during commit when CanMoveCommitted=false, got %v", e.Velocity)
	}
}

func TestRunner_MovementAllowed(t *testing.T) {
	def := meleeRunnerDef()
	def.Abilities[0].CanMoveCommitted = true

	e := entity.NewEnemy(0, 500, "test")
	e.State = entity.EnemyChase
	e.TargetPlayerID = 1
	p := testPlayer(1, entity.Vec3{X: 0, Z: 2})

	ctx := testCtx(def, e, testPlayers(p))
	ctx.Cast(testMeleeID)

	e.Velocity = entity.Vec3{X: 5, Z: 5}
	e.StateTimer -= 0.05
	ctx.Runner.Tick(ctx)

	// Velocity should be preserved
	if e.Velocity.X != 5 || e.Velocity.Z != 5 {
		t.Errorf("velocity should be preserved when CanMoveCommitted=true, got %v", e.Velocity)
	}
}

func TestRunner_AbortOnDeath(t *testing.T) {
	def := meleeRunnerDef()
	e := entity.NewEnemy(0, 500, "test")
	e.State = entity.EnemyChase
	e.TargetPlayerID = 1
	p := testPlayer(1, entity.Vec3{X: 0, Z: 2})

	ctx := testCtx(def, e, testPlayers(p))
	ctx.Cast(testMeleeID)

	// Kill the enemy
	e.Alive = false
	e.StateTimer -= 0.05
	ctx.Runner.Tick(ctx)

	if ctx.Runner.Phase != RunnerIdle {
		t.Fatalf("expected RunnerIdle after death, got %d", ctx.Runner.Phase)
	}
}

func TestRunner_AbortOnPhaseTransition(t *testing.T) {
	def := meleeRunnerDef()
	e := entity.NewEnemy(0, 500, "test")
	e.State = entity.EnemyChase
	e.TargetPlayerID = 1
	p := testPlayer(1, entity.Vec3{X: 0, Z: 2})

	ctx := testCtx(def, e, testPlayers(p))
	ctx.Cast(testMeleeID)

	// Phase transition
	e.State = entity.EnemyPhaseTransition
	e.StateTimer -= 0.05
	ctx.Runner.Tick(ctx)

	if ctx.Runner.Phase != RunnerIdle {
		t.Fatalf("expected RunnerIdle after phase transition, got %d", ctx.Runner.Phase)
	}
}

func TestRunner_ChargeLifecycle(t *testing.T) {
	def := chargeRunnerDef()
	e := entity.NewEnemy(0, 500, "test")
	e.State = entity.EnemyChase
	e.TargetPlayerID = 1
	// Player far away for charge
	p := testPlayer(1, entity.Vec3{X: 0, Z: 10})

	ctx := testCtx(def, e, testPlayers(p))

	if !ctx.Cast("charge") {
		t.Fatal("Cast should succeed")
	}
	if e.State != entity.EnemyChargeTelegraph {
		t.Fatalf("expected ChargeTelegraph, got %d", e.State)
	}

	// Tick through commit
	for i := 0; i < 10; i++ {
		e.StateTimer -= 0.05
		ctx.Runner.Tick(ctx)
	}

	if ctx.Runner.Phase != RunnerExecute {
		t.Fatalf("expected RunnerExecute, got %d", ctx.Runner.Phase)
	}
	if e.State != entity.EnemyCharge {
		t.Fatalf("expected EnemyCharge, got %d", e.State)
	}

	// Tick charge until max distance (12 speed * 0.05dt = 0.6 per tick, 15/0.6 = 25 ticks)
	for i := 0; i < 30; i++ {
		e.StateTimer -= 0.05
		ctx.Runner.Tick(ctx)
		// Apply velocity like Brain.Tick does
		e.Position = e.Position.Add(e.Velocity.Scale(0.05))
		if ctx.Runner.Phase != RunnerExecute {
			break
		}
	}

	if ctx.Runner.Phase != RunnerCooldown {
		t.Fatalf("expected RunnerCooldown after charge, got %d", ctx.Runner.Phase)
	}
}

func TestRunner_CastWeighted(t *testing.T) {
	def := meleeRunnerDef()
	e := entity.NewEnemy(0, 500, "test")
	e.State = entity.EnemyChase
	e.TargetPlayerID = 1
	p := testPlayer(1, entity.Vec3{X: 0, Z: 2})

	ctx := testCtx(def, e, testPlayers(p))

	if !ctx.CastWeighted() {
		t.Fatal("CastWeighted should succeed")
	}
	if ctx.Runner.Phase != RunnerCommit {
		t.Fatalf("expected RunnerCommit, got %d", ctx.Runner.Phase)
	}
}

func TestRunner_CastUnknownAbility(t *testing.T) {
	def := meleeRunnerDef()
	e := entity.NewEnemy(0, 500, "test")
	e.State = entity.EnemyChase

	ctx := testCtx(def, e, nil)

	if ctx.Cast("nonexistent") {
		t.Error("Cast should fail for unknown ability")
	}
	if ctx.Runner.Phase != RunnerIdle {
		t.Error("runner should stay idle after failed cast")
	}
}

func TestRunner_ContextQueries(t *testing.T) {
	def := meleeRunnerDef()
	e := entity.NewEnemy(0, 500, "test")
	e.State = entity.EnemyChase
	e.TargetPlayerID = 1
	p := testPlayer(1, entity.Vec3{X: 0, Z: 2})

	ctx := testCtx(def, e, testPlayers(p))

	// Idle state
	if ctx.IsRunnerBusy() {
		t.Error("should not be busy when idle")
	}
	if ctx.CurrentAbilityID() != "" {
		t.Error("should return empty string when idle")
	}

	// After cast
	ctx.Cast(testMeleeID)
	if !ctx.IsRunnerBusy() {
		t.Error("should be busy after cast")
	}
	if ctx.CurrentAbilityID() != testMeleeID {
		t.Errorf("expected 'melee', got %q", ctx.CurrentAbilityID())
	}
}
