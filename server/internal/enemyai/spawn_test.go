package enemyai

import (
	"testing"

	"codex-online/server/internal/ability"
	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
)

const testSummonID = "summon_adds"

func spawnRunnerDef() *EnemyDef {
	return &EnemyDef{
		Name:      "test_summoner",
		MaxHealth: 1500,
		MoveSpeed: 5.0,
		Radius:    1.0,
		Abilities: []ability.AbilityDef{
			{
				ID: testSummonID, Name: testSummonID, Category: ability.CategorySpawn,
				CommitTime: 0.5, ExecuteTime: 0.2, Cooldown: 30.0,
				SpawnDefName:  "test_add",
				SpawnCount:    0, // one per alive player
				SpawnCap:      3,
				SpawnDistance: 4.0,
			},
		},
	}
}

func TestParseSpawnAbility(t *testing.T) {
	yaml := []byte(`
name: test_spawn_boss
max_health: 1000
move_speed: 4.0
radius: 1.0
abilities:
  - name: summon_stalkers
    type: spawn
    telegraph_time: 1.2
    execute_time: 0.2
    cooldown_time: 30.0
    spawn_def_name: aceras_stalker
    spawn_count: 0
    spawn_cap: 3
    spawn_distance: 4.0
tree:
  reactive_selector:
    - sequence: [is_dead, stop]
    - chase
`)
	def, err := parseMobYAML(yaml)
	if err != nil {
		t.Fatalf("parseMobYAML: %v", err)
	}
	if len(def.Abilities) != 1 {
		t.Fatalf("abilities = %d, want 1", len(def.Abilities))
	}
	a := def.Abilities[0]
	if a.Category != ability.CategorySpawn {
		t.Errorf("category = %d, want CategorySpawn", a.Category)
	}
	if a.SpawnDefName != "aceras_stalker" {
		t.Errorf("SpawnDefName = %q, want aceras_stalker", a.SpawnDefName)
	}
	if a.SpawnCap != 3 || a.SpawnDistance != 4.0 {
		t.Errorf("SpawnCap=%d SpawnDistance=%f, want 3 / 4.0", a.SpawnCap, a.SpawnDistance)
	}
}

func TestParseSpawnAbility_MissingDefNameErrors(t *testing.T) {
	yaml := []byte(`
name: test_spawn_boss
max_health: 1000
move_speed: 4.0
radius: 1.0
abilities:
  - name: summon_broken
    type: spawn
    telegraph_time: 1.0
tree:
  reactive_selector:
    - chase
`)
	if _, err := parseMobYAML(yaml); err == nil {
		t.Fatal("expected error for spawn ability without spawn_def_name")
	}
}

// tickRunnerThrough advances the runner past commit+execute.
func tickRunnerThrough(ctx *EntityContext, ticks int) {
	for range ticks {
		ctx.Enemy.StateTimer -= 0.05
		ctx.Runner.Tick(ctx)
	}
}

func TestSpawnAbility_SpawnsBehindEachPlayer(t *testing.T) {
	def := spawnRunnerDef()
	e := entity.NewEnemy(1000, 1500, "test_summoner")
	e.State = entity.EnemyChase
	e.Position = entity.Vec3{X: 0, Y: 0.1, Z: 0}

	p1 := testPlayer(1, entity.Vec3{X: 10, Y: 0.1, Z: 0})
	p2 := testPlayer(2, entity.Vec3{X: 0, Y: 0.1, Z: 8})
	ctx := testCtx(def, e, testPlayers(p1, p2))

	var spawned []entity.Vec3
	ctx.SpawnAddFn = func(defName string, pos entity.Vec3, ownerID uint16) bool {
		if defName != "test_add" {
			t.Errorf("defName = %q, want test_add", defName)
		}
		if ownerID != e.ID {
			t.Errorf("ownerID = %d, want %d", ownerID, e.ID)
		}
		spawned = append(spawned, pos)
		return true
	}

	if !ctx.Commit(testSummonID) {
		t.Fatal("Commit should succeed")
	}
	if e.State != entity.EnemyAoETelegraph {
		t.Fatalf("spawn commit should reuse AoE telegraph state, got %d", e.State)
	}
	tickRunnerThrough(ctx, 20) // 1.0s: past 0.5 commit + 0.2 execute

	if len(spawned) != 2 {
		t.Fatalf("spawned = %d adds, want 2 (one per alive player)", len(spawned))
	}
	// Expect one add behind each player, away from the boss (order is
	// nearest-first): p1 at (10,0) → ≈(14,0); p2 at (0,8) → ≈(0,12).
	near := func(pos entity.Vec3, x, z float32) bool {
		return pos.X > x-0.5 && pos.X < x+0.5 && pos.Z > z-0.5 && pos.Z < z+0.5
	}
	foundP1, foundP2 := false, false
	for _, pos := range spawned {
		if near(pos, 14, 0) {
			foundP1 = true
		}
		if near(pos, 0, 12) {
			foundP2 = true
		}
	}
	if !foundP1 || !foundP2 {
		t.Errorf("spawned = %+v, want one add ≈(14,0) and one ≈(0,12)", spawned)
	}
}

func TestSpawnAbility_RespectsCapAndBounds(t *testing.T) {
	def := spawnRunnerDef()
	e := entity.NewEnemy(1000, 1500, "test_summoner")
	e.State = entity.EnemyChase
	e.Position = entity.Vec3{}

	// 5 players but cap is 3.
	players := testPlayers(
		testPlayer(1, entity.Vec3{X: 18, Z: 0}), // behind → x=22, clamped to bounds (max 20 - radius)
		testPlayer(2, entity.Vec3{X: 0, Z: 8}),
		testPlayer(3, entity.Vec3{X: -6, Z: 0}),
		testPlayer(4, entity.Vec3{X: 0, Z: -6}),
		testPlayer(5, entity.Vec3{X: 4, Z: 4}),
	)
	ctx := testCtx(def, e, players)

	var spawned []entity.Vec3
	ctx.SpawnAddFn = func(_ string, pos entity.Vec3, _ uint16) bool {
		spawned = append(spawned, pos)
		return true
	}

	if !ctx.Commit(testSummonID) {
		t.Fatal("Commit should succeed")
	}
	tickRunnerThrough(ctx, 20)

	if len(spawned) != 3 {
		t.Fatalf("spawned = %d, want 3 (SpawnCap)", len(spawned))
	}
	for i, pos := range spawned {
		if pos.X > ctx.BoundsMaxX || pos.X < ctx.BoundsMinX || pos.Z > ctx.BoundsMaxZ || pos.Z < ctx.BoundsMinZ {
			t.Errorf("add %d at %+v escapes bounds", i, pos)
		}
	}
}

func TestSpawnAbility_PushedOutOfObstacles(t *testing.T) {
	def := spawnRunnerDef()
	def.Abilities[0].SpawnCount = 1
	e := entity.NewEnemy(1000, 1500, "test_summoner")
	e.State = entity.EnemyChase
	e.Position = entity.Vec3{}

	p := testPlayer(1, entity.Vec3{X: 6, Z: 0})
	ctx := testCtx(def, e, testPlayers(p))
	// Obstacle exactly at the naive spawn point (10, 0).
	ctx.Obs = []combat.Obstacle{{CX: 10, CZ: 0, HX: 1, HZ: 1, Height: 4}}

	var spawned []entity.Vec3
	ctx.SpawnAddFn = func(_ string, pos entity.Vec3, _ uint16) bool {
		spawned = append(spawned, pos)
		return true
	}

	if !ctx.Commit(testSummonID) {
		t.Fatal("Commit should succeed")
	}
	tickRunnerThrough(ctx, 20)

	if len(spawned) != 1 {
		t.Fatalf("spawned = %d, want 1", len(spawned))
	}
	pos := spawned[0]
	dx := pos.X - 10
	dz := pos.Z - 0
	if dx > -1 && dx < 1 && dz > -1 && dz < 1 {
		t.Errorf("add at %+v is inside the obstacle", pos)
	}
}
