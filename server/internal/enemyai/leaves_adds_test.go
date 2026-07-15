package enemyai

import (
	"testing"

	"codex-online/server/internal/ability"
	"codex-online/server/internal/bt"
	"codex-online/server/internal/entity"
)

func kiterDef() *EnemyDef {
	return &EnemyDef{
		Name:           "test_kiter",
		MaxHealth:      1500,
		MoveSpeed:      5.0,
		Radius:         1.0,
		PreferredRange: 12.0,
		BackpedalSpeed: 3.5,
		Abilities: []ability.AbilityDef{
			{
				ID: "swipe", Name: "swipe", Category: ability.CategoryMelee,
				CommitTime: 0.5, ExecuteTime: 0.2, Cooldown: 3.0,
				Hit: ability.HitDef{Type: ability.HitAoECone, Range: 3.0, ArcDegrees: 120},
			},
		},
	}
}

func TestCond_MyAddsAlive(t *testing.T) {
	def := kiterDef()
	boss := entity.NewEnemy(1000, 1500, "test_kiter")
	ctx := testCtx(def, boss, nil)

	if condMyAddsAlive(ctx) {
		t.Error("no allies: should be false")
	}

	mine := entity.NewEnemy(1500, 120, "test_add")
	mine.SpawnedBy = boss.ID
	other := entity.NewEnemy(1501, 120, "test_add")
	other.SpawnedBy = 999 // someone else's add
	ctx.Allies = []*entity.Enemy{boss, mine, other}

	if !condMyAddsAlive(ctx) {
		t.Error("owned alive add: should be true")
	}

	mine.Alive = false
	if condMyAddsAlive(ctx) {
		t.Error("owned add dead, other-owner add alive: should be false")
	}
}

func TestCond_AddsEngaged_LingerWindow(t *testing.T) {
	def := kiterDef()
	boss := entity.NewEnemy(1000, 1500, "test_kiter")
	ctx := testCtx(def, boss, nil)
	cond := condAddsEngaged(5.0)

	mine := entity.NewEnemy(1500, 120, "test_add")
	mine.SpawnedBy = boss.ID
	ctx.Allies = []*entity.Enemy{boss, mine}

	if !cond(ctx) {
		t.Fatal("add alive: should be engaged")
	}

	// Add dies: engaged lingers for 5s, refreshed while it was alive.
	mine.Alive = false
	for range 98 { // 4.9s at 0.05
		ctx.BB.TickTimers(0.05)
		if !cond(ctx) {
			t.Fatalf("linger window should hold at %f s", ctx.BB.TimerRemaining("adds_linger"))
		}
	}
	for range 6 { // push past 5.0s
		ctx.BB.TickTimers(0.05)
	}
	if cond(ctx) {
		t.Error("linger expired: should be false at 5.2s after death")
	}
}

func TestCond_EngagedFor(t *testing.T) {
	def := kiterDef()
	boss := entity.NewEnemy(1000, 1500, "test_kiter")
	p := testPlayer(1, entity.Vec3{X: 0, Z: 10})
	ctx := testCtx(def, boss, testPlayers(p))
	cond := condEngagedFor(8.0)

	if cond(ctx) {
		t.Fatal("first evaluation must be false (timer just started)")
	}
	for range 158 { // 7.9s
		ctx.BB.TickTimers(0.05)
	}
	if cond(ctx) {
		t.Error("7.9s in: should still be false")
	}
	for range 4 { // past 8.0s
		ctx.BB.TickTimers(0.05)
	}
	if !cond(ctx) {
		t.Error("8.1s in: should be true")
	}
}

func TestAction_ChaseMelee_IgnoresPreferredRange(t *testing.T) {
	def := kiterDef()
	boss := entity.NewEnemy(1000, 1500, "test_kiter")
	boss.Alive = true
	boss.State = entity.EnemyChase
	boss.Position = entity.Vec3{}
	p := testPlayer(1, entity.Vec3{X: 8, Z: 0}) // inside PreferredRange 12
	ctx := testCtx(def, boss, testPlayers(p))

	// Plain chase backpedals (moves away: negative X velocity).
	if r := actionChase(ctx); r != bt.Running {
		t.Fatalf("chase = %v, want Running", r)
	}
	if boss.Velocity.X >= 0 {
		t.Fatalf("plain chase should backpedal, vel.X = %f", boss.Velocity.X)
	}

	// chase_melee closes in (positive X velocity toward the player).
	boss.Velocity = entity.Vec3{}
	if r := actionChaseMelee(ctx); r != bt.Running {
		t.Fatalf("chase_melee = %v, want Running", r)
	}
	if boss.Velocity.X <= 0 {
		t.Errorf("chase_melee should move toward the player, vel.X = %f", boss.Velocity.X)
	}

	// Within melee reach it stops and reports Success.
	p.Position = entity.Vec3{X: 2, Z: 0}
	if r := actionChaseMelee(ctx); r != bt.Success {
		t.Errorf("chase_melee in range = %v, want Success", r)
	}
}

func TestResolveLeaf_AddsLeaves(t *testing.T) {
	for _, name := range []string{"my_adds_alive", "chase_melee", "adds_engaged(5)", "engaged_for(8)"} {
		if _, err := resolveLeaf(name); err != nil {
			t.Errorf("resolveLeaf(%q): %v", name, err)
		}
	}
}
