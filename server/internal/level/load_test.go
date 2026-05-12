package level

import (
	"os"
	"path/filepath"
	"testing"

	"codex-online/server/internal/entity"
)

func writeTempJSON(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "test.json")
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadLevelData_Valid(t *testing.T) {
	p := writeTempJSON(t, `{
		"version": 2,
		"zone": "test",
		"source_scene": "res://test.tscn",
		"bounds": { "min_x": -10, "max_x": 10, "min_y": -1, "max_y": 5, "min_z": -10, "max_z": 10 },
		"obstacles": [
			{ "name": "Pillar", "center": [-8, 2, -6], "half_extents": [0.75, 2, 0.75] }
		],
		"elevators": [
			{ "name": "Lift", "center_x": 5, "center_z": -55, "half_x": 4, "half_z": 4, "bottom_y": -200, "top_y": 0, "speed": 10 }
		],
		"player_spawns": [ { "x": 0, "y": 0.1, "z": 5 } ],
		"enemy_spawns": [
			{ "x": 0, "y": 0.1, "z": 0, "def_name": "mob", "patrol_a": { "x": -5, "y": 0.1, "z": 0 }, "patrol_b": { "x": 5, "y": 0.1, "z": 0 }, "aggro_radius": 10, "leash_radius": 30, "group_id": 1 }
		]
	}`)
	l := &Level{}
	if err := loadLevelData(p, l); err != nil {
		t.Fatal(err)
	}

	// Bounds
	if l.PlayerBoundsMinX != -10 || l.PlayerBoundsMaxX != 10 {
		t.Errorf("X bounds = [%f, %f]", l.PlayerBoundsMinX, l.PlayerBoundsMaxX)
	}
	if l.PlayerBoundsMinY != -1 || l.PlayerBoundsMaxY != 5 {
		t.Errorf("Y bounds = [%f, %f]", l.PlayerBoundsMinY, l.PlayerBoundsMaxY)
	}

	// Obstacles
	if len(l.Obstacles) != 1 {
		t.Fatalf("obstacles len = %d, want 1", len(l.Obstacles))
	}
	obs := l.Obstacles[0]
	if obs.CX != -8 || obs.CZ != -6 || obs.HX != 0.75 || obs.HZ != 0.75 {
		t.Errorf("obstacle = %+v", obs)
	}
	if obs.Height != 4 { // half_extents[1]=2 * 2 = 4
		t.Errorf("obstacle height = %f, want 4", obs.Height)
	}

	// Elevators
	if len(l.Elevators) != 1 {
		t.Fatalf("elevators len = %d, want 1", len(l.Elevators))
	}
	ev := l.Elevators[0]
	if ev.BottomY != -200 || ev.TopY != 0 || ev.Speed != 10 {
		t.Errorf("elevator = %+v", ev)
	}

	// Spawns
	if len(l.PlayerSpawns) != 1 || l.PlayerSpawns[0].Position.X != 0 {
		t.Errorf("player spawns = %+v", l.PlayerSpawns)
	}
	if len(l.EnemySpawns) != 1 || l.EnemySpawns[0].DefName != "mob" {
		t.Errorf("enemy spawns = %+v", l.EnemySpawns)
	}
	if l.EnemySpawns[0].GroupID != 1 {
		t.Errorf("enemy group_id = %d, want 1", l.EnemySpawns[0].GroupID)
	}
}

func TestLoadLevelData_MissingFile(t *testing.T) {
	l := &Level{}
	err := loadLevelData("/nonexistent/path.json", l)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadLevelData_BadJSON(t *testing.T) {
	p := writeTempJSON(t, `{ not valid }`)
	l := &Level{}
	err := loadLevelData(p, l)
	if err == nil {
		t.Fatal("expected error for bad JSON")
	}
}

func TestLoadLevelData_WrongVersion(t *testing.T) {
	p := writeTempJSON(t, `{
		"version": 99,
		"zone": "test",
		"source_scene": "res://test.tscn",
		"bounds": { "min_x": 0, "max_x": 0, "min_y": 0, "max_y": 0, "min_z": 0, "max_z": 0 },
		"obstacles": [],
		"player_spawns": []
	}`)
	l := &Level{}
	err := loadLevelData(p, l)
	if err == nil {
		t.Fatal("expected error for wrong version")
	}
}

func TestLoadLevelData_PreservesGameLogic(t *testing.T) {
	p := writeTempJSON(t, `{
		"version": 2,
		"zone": "arena",
		"source_scene": "res://arena.tscn",
		"bounds": { "min_x": -19.5, "max_x": 19.5, "min_y": -1, "max_y": 6, "min_z": -14.5, "max_z": 52 },
		"obstacles": [],
		"player_spawns": []
	}`)
	l := &Level{
		ArenaEntryZ:    40.0,
		BossRoomEntryZ: 12.0,
		EnemyRadius:    1.0,
	}
	if err := loadLevelData(p, l); err != nil {
		t.Fatal(err)
	}
	// Game logic fields must be untouched
	if l.ArenaEntryZ != 40.0 {
		t.Errorf("ArenaEntryZ = %f, want 40", l.ArenaEntryZ)
	}
	if l.BossRoomEntryZ != 12.0 {
		t.Errorf("BossRoomEntryZ = %f, want 12", l.BossRoomEntryZ)
	}
	if l.EnemyRadius != 1.0 {
		t.Errorf("EnemyRadius = %f, want 1", l.EnemyRadius)
	}
}

func TestLoadLevelData_CoverHeight(t *testing.T) {
	p := writeTempJSON(t, `{
		"version": 2,
		"zone": "test",
		"source_scene": "res://test.tscn",
		"bounds": { "min_x": -10, "max_x": 10, "min_y": -1, "max_y": 5, "min_z": -10, "max_z": 10 },
		"obstacles": [
			{ "name": "Cover", "center": [-5, 0.6, -2], "half_extents": [1.5, 0.6, 0.5] }
		],
		"player_spawns": []
	}`)
	l := &Level{}
	if err := loadLevelData(p, l); err != nil {
		t.Fatal(err)
	}
	if l.Obstacles[0].Height != 1.2 {
		t.Errorf("cover height = %f, want 1.2", l.Obstacles[0].Height)
	}
}

func TestLoadArenaJSON(t *testing.T) {
	path := "../../../shared/levels/arena.json"
	if _, err := os.Stat(path); err != nil {
		t.Skip("arena.json not found, skipping integration test")
	}
	l := &Level{ArenaEntryZ: 40.0, BossRoomEntryZ: 12.0, EnemyRadius: 1.0}
	if err := loadLevelData(path, l); err != nil {
		t.Fatal(err)
	}
	if len(l.Obstacles) != 20 {
		t.Errorf("obstacles len = %d, want 20", len(l.Obstacles))
	}
	if len(l.PlayerSpawns) != 5 {
		t.Errorf("player_spawns len = %d, want 5", len(l.PlayerSpawns))
	}
	if len(l.EnemySpawns) != 9 {
		t.Errorf("enemy_spawns len = %d, want 9", len(l.EnemySpawns))
	}
}

func TestLoadHubJSON(t *testing.T) {
	path := "../../../shared/levels/hub.json"
	if _, err := os.Stat(path); err != nil {
		t.Skip("hub.json not found, skipping integration test")
	}
	l := &Level{}
	if err := loadLevelData(path, l); err != nil {
		t.Fatal(err)
	}
	if len(l.Elevators) != 2 {
		t.Errorf("elevators len = %d, want 2", len(l.Elevators))
	}
	if l.Elevators[0].BottomY != -200 {
		t.Errorf("public lift bottom_y = %f, want -200", l.Elevators[0].BottomY)
	}
}

func TestLoadLevelData_V3Features(t *testing.T) {
	p := writeTempJSON(t, `{
		"version": 3,
		"zone": "test",
		"source_scene": "res://test.tscn",
		"bounds": { "min_x": -10, "max_x": 10, "min_y": -1, "max_y": 5, "min_z": -10, "max_z": 10 },
		"obstacles": [],
		"player_spawns": [
			{ "x": 0, "y": 0.1, "z": 48 },
			{ "x": 0, "y": 0.1, "z": 20, "condition": "pack_1_cleared" },
			{ "x": 0, "y": 0.1, "z": 0, "condition": "boss_dead" }
		],
		"enemy_spawns": [
			{
				"x": 0, "y": 0.1, "z": 32, "def_name": "mob",
				"patrol_a": { "x": -5, "y": 0.1, "z": 32 },
				"patrol_b": { "x": 5, "y": 0.1, "z": 32 },
				"patrol_waypoints": [
					{ "x": -5, "y": 0.1, "z": 32 },
					{ "x": 0, "y": 0.1, "z": 30 },
					{ "x": 5, "y": 0.1, "z": 32 }
				],
				"aggro_radius": 10, "leash_radius": 30,
				"condition": "default"
			}
		],
		"portals": [
			{ "name": "Portal1", "x": 33, "y": 102, "z": 5.5, "target_zone": "arena", "interaction_radius": 4.0 }
		],
		"zone_triggers": [
			{ "name": "Entry", "trigger_id": "arena_entry", "axis": "z", "threshold": 40 },
			{ "name": "BossGate", "trigger_id": "boss_room_entry", "axis": "z", "threshold": 12 }
		]
	}`)

	l := &Level{ArenaEntryZ: 99, BossRoomEntryZ: 99} // will be overwritten by zone_triggers
	if err := loadLevelData(p, l); err != nil {
		t.Fatal(err)
	}

	// Player spawns with conditions
	if len(l.PlayerSpawns) != 3 {
		t.Fatalf("player spawns = %d, want 3", len(l.PlayerSpawns))
	}
	if l.PlayerSpawns[1].Condition != "pack_1_cleared" {
		t.Errorf("spawn[1] condition = %q, want pack_1_cleared", l.PlayerSpawns[1].Condition)
	}
	if l.PlayerSpawns[2].Condition != "boss_dead" {
		t.Errorf("spawn[2] condition = %q, want boss_dead", l.PlayerSpawns[2].Condition)
	}

	// Enemy spawn with patrol waypoints
	if len(l.EnemySpawns) != 1 {
		t.Fatalf("enemy spawns = %d, want 1", len(l.EnemySpawns))
	}
	es := l.EnemySpawns[0]
	if len(es.PatrolWaypoints) != 3 {
		t.Errorf("patrol waypoints = %d, want 3", len(es.PatrolWaypoints))
	}
	// PatrolA/B should be overridden by first/last waypoint
	if es.PatrolA.X != -5 {
		t.Errorf("PatrolA.X = %f, want -5 (first waypoint)", es.PatrolA.X)
	}
	if es.PatrolB.X != 5 {
		t.Errorf("PatrolB.X = %f, want 5 (last waypoint)", es.PatrolB.X)
	}
	if es.Condition != "default" {
		t.Errorf("enemy condition = %q, want default", es.Condition)
	}

	// Portals
	if len(l.Portals) != 1 {
		t.Fatalf("portals = %d, want 1", len(l.Portals))
	}
	portal := l.Portals[0]
	if portal.Name != "Portal1" || portal.TargetZone != "arena" {
		t.Errorf("portal = %+v", portal)
	}
	if portal.InteractionRadius != 4.0 {
		t.Errorf("portal radius = %f, want 4", portal.InteractionRadius)
	}

	// Zone triggers override existing values
	if l.ArenaEntryZ != 40 {
		t.Errorf("ArenaEntryZ = %f, want 40", l.ArenaEntryZ)
	}
	if l.BossRoomEntryZ != 12 {
		t.Errorf("BossRoomEntryZ = %f, want 12", l.BossRoomEntryZ)
	}
}

func TestLoadLevelData_V2BackwardCompat(t *testing.T) {
	// v2 JSON should still load without errors — no new fields present
	p := writeTempJSON(t, `{
		"version": 2,
		"zone": "test",
		"source_scene": "res://test.tscn",
		"bounds": { "min_x": -10, "max_x": 10, "min_y": -1, "max_y": 5, "min_z": -10, "max_z": 10 },
		"obstacles": [],
		"player_spawns": [ { "x": 0, "y": 0.1, "z": 5 } ]
	}`)
	l := &Level{ArenaEntryZ: 40.0}
	if err := loadLevelData(p, l); err != nil {
		t.Fatal(err)
	}
	// v2 spawns should have empty condition
	if l.PlayerSpawns[0].Condition != "" {
		t.Errorf("v2 spawn condition = %q, want empty", l.PlayerSpawns[0].Condition)
	}
	// No portals or zone triggers
	if len(l.Portals) != 0 {
		t.Errorf("v2 portals = %d, want 0", len(l.Portals))
	}
	// ArenaEntryZ should be untouched (no zone_triggers in v2 JSON)
	if l.ArenaEntryZ != 40.0 {
		t.Errorf("ArenaEntryZ = %f, want 40 (preserved)", l.ArenaEntryZ)
	}
}

// =============================================================================
// ClampPlayer / ClampEnemy
// =============================================================================

func TestClampPlayerWithinBounds(t *testing.T) {
	l := &Level{
		PlayerBoundsMinX: -10, PlayerBoundsMaxX: 10,
		PlayerBoundsMinZ: -10, PlayerBoundsMaxZ: 10,
	}
	pos := entity.Vec3{X: 5, Y: 1, Z: -3}
	l.ClampPlayer(&pos)
	if pos.X != 5 || pos.Z != -3 {
		t.Errorf("in-bounds position changed: %v", pos)
	}
}

func TestClampPlayerOutOfBounds(t *testing.T) {
	l := &Level{
		PlayerBoundsMinX: -10, PlayerBoundsMaxX: 10,
		PlayerBoundsMinZ: -10, PlayerBoundsMaxZ: 10,
	}
	pos := entity.Vec3{X: 15, Y: 1, Z: -20}
	l.ClampPlayer(&pos)
	if pos.X != 10 {
		t.Errorf("X = %f, want 10 (clamped)", pos.X)
	}
	if pos.Z != -10 {
		t.Errorf("Z = %f, want -10 (clamped)", pos.Z)
	}
}

func TestClampEnemyWithinBounds(t *testing.T) {
	l := &Level{
		EnemyBoundsMinX: -20, EnemyBoundsMaxX: 20,
		EnemyBoundsMinZ: -15, EnemyBoundsMaxZ: 50,
	}
	pos := entity.Vec3{X: 5, Y: 1, Z: 10}
	l.ClampEnemy(&pos)
	if pos.X != 5 || pos.Z != 10 {
		t.Errorf("in-bounds position changed: %v", pos)
	}
}

func TestClampEnemyOutOfBounds(t *testing.T) {
	l := &Level{
		EnemyBoundsMinX: -20, EnemyBoundsMaxX: 20,
		EnemyBoundsMinZ: -15, EnemyBoundsMaxZ: 50,
	}
	pos := entity.Vec3{X: 25, Y: -1, Z: 60}
	l.ClampEnemy(&pos)
	if pos.X != 20 {
		t.Errorf("X = %f, want 20 (clamped)", pos.X)
	}
	if pos.Z != 50 {
		t.Errorf("Z = %f, want 50 (clamped)", pos.Z)
	}
	if pos.Y != 0.1 {
		t.Errorf("Y = %f, want 0.1 (min floor)", pos.Y)
	}
}

// =============================================================================
// levelDataPath
// =============================================================================

func TestLevelDataPathDefault(t *testing.T) {
	t.Setenv("CODEX_LEVELS_DIR", "")
	path := levelDataPath("arena")
	if _, err := os.Stat(path); err != nil {
		t.Errorf("levelDataPath returned %q which does not exist: %v", path, err)
	}
}

func TestLevelDataPathCustomDir(t *testing.T) {
	t.Setenv("CODEX_LEVELS_DIR", "/tmp/levels")
	path := levelDataPath("hub")
	expected := filepath.Join("/tmp/levels", "hub.json")
	if path != expected {
		t.Errorf("path = %q, want %q", path, expected)
	}
}

// =============================================================================
// NewArenaLevel / NewHubLevel (hardcoded fallback)
// =============================================================================

func TestNewArenaLevelFallback(t *testing.T) {
	t.Setenv("CODEX_LEVELS_DIR", "/nonexistent")
	l := NewArenaLevel()
	if l == nil {
		t.Fatal("NewArenaLevel returned nil")
	}
	if l.ArenaEntryZ != 40.0 {
		t.Errorf("ArenaEntryZ = %f, want 40", l.ArenaEntryZ)
	}
	if l.EnemyRadius != 1.0 {
		t.Errorf("EnemyRadius = %f, want 1", l.EnemyRadius)
	}
	if len(l.PlayerSpawns) != 5 {
		t.Errorf("PlayerSpawns = %d, want 5", len(l.PlayerSpawns))
	}
	if len(l.Obstacles) == 0 {
		t.Error("expected obstacles in hardcoded arena")
	}
}

func TestNewHubLevelFallback(t *testing.T) {
	t.Setenv("CODEX_LEVELS_DIR", "/nonexistent")
	l := NewHubLevel()
	if l == nil {
		t.Fatal("NewHubLevel returned nil")
	}
	if len(l.PlayerSpawns) != 5 {
		t.Errorf("PlayerSpawns = %d, want 5", len(l.PlayerSpawns))
	}
	if len(l.Elevators) != 2 {
		t.Errorf("Elevators = %d, want 2", len(l.Elevators))
	}
}
