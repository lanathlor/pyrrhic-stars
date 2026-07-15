package level

import (
	"os"
	"path/filepath"
	"testing"

	"codex-online/server/internal/combat"
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
		InstanceEntryZ: 40.0,
		EnemyRadius:    1.0,
	}
	if err := loadLevelData(p, l); err != nil {
		t.Fatal(err)
	}
	// Game logic fields must be untouched
	if l.InstanceEntryZ != 40.0 {
		t.Errorf("InstanceEntryZ = %f, want 40", l.InstanceEntryZ)
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
	l := &Level{}
	if err := loadLevelData(path, l); err != nil {
		t.Fatal(err)
	}
	if l.ZoneType != "instanced" {
		t.Errorf("ZoneType = %q, want %q", l.ZoneType, "instanced")
	}
	if l.EnemyRadius != 1.0 {
		t.Errorf("EnemyRadius = %f, want 1", l.EnemyRadius)
	}
	if len(l.Obstacles) != 44 { // +1 2026-07-14: AcerasScreen (the General's door screen)
		t.Errorf("obstacles len = %d, want 44", len(l.Obstacles))
	}
	if len(l.PlayerSpawns) != 8 {
		t.Errorf("player_spawns len = %d, want 8 (5 lobby + 3 boss-1 checkpoints)", len(l.PlayerSpawns))
	}
	if len(l.EnemySpawns) != 17 {
		t.Errorf("enemy_spawns len = %d, want 17 (2 hallway packs + boss 1 + 2 decline packs + boss 2)", len(l.EnemySpawns))
	}
	if len(l.Gates) != 4 {
		t.Errorf("gates len = %d, want 4", len(l.Gates))
	}
	if len(l.Portals) != 1 {
		t.Fatalf("portals len = %d, want 1", len(l.Portals))
	}
	if l.Portals[0].TargetZone != "hub" {
		t.Errorf("portal target_zone = %q, want %q", l.Portals[0].TargetZone, "hub")
	}

	// Multi-boss fields survive the export round-trip.
	bosses := 0
	for _, s := range l.EnemySpawns {
		if !s.IsBoss {
			continue
		}
		bosses++
		switch s.DefName {
		case "guard_captain":
			if s.BossNum != 1 || s.BossGateID != "boss_gate" || s.AggroMaxZ != 14.0 {
				t.Errorf("guard_captain spawn = %+v", s)
			}
		case "aceras_general":
			if s.BossNum != 2 || s.BossGateID != "aceras_gate" || s.AggroMaxZ != -40.0 {
				t.Errorf("aceras_general spawn = %+v", s)
			}
		default:
			t.Errorf("unexpected boss def %q", s.DefName)
		}
	}
	if bosses != 2 {
		t.Errorf("boss spawns = %d, want 2", bosses)
	}
	checkpoints := 0
	for _, s := range l.PlayerSpawns {
		if s.Condition == "boss_1_dead" {
			checkpoints++
		}
	}
	if checkpoints != 3 {
		t.Errorf("boss_1_dead checkpoint spawns = %d, want 3", checkpoints)
	}
}

// TestArenaBossScreen_BlocksCorridorLoS guards the boss-1 entrance screen:
// every corridor position that is aggro-immune (Z >= the boss's AggroMaxZ,
// behind the screen) must have NO line of sight to any point of the boss's
// patrol band. Closer positions may see the boss — but there the boss can
// aggro back, so there is no risk-free sniping spot anywhere.
func TestArenaBossScreen_BlocksCorridorLoS(t *testing.T) {
	l, err := Load("arena")
	if err != nil {
		t.Fatal(err)
	}
	var aggroMaxZ float32
	for _, sp := range l.EnemySpawns {
		if sp.DefName == "guard_captain" {
			aggroMaxZ = sp.AggroMaxZ
		}
	}
	if aggroMaxZ == 0 {
		t.Fatal("guard_captain spawn has no aggro_max_z")
	}

	const eyeY = 1.5
	const bodyRadius = 0.4
	standable := func(p entity.Vec3) bool {
		for _, o := range l.Obstacles {
			if p.X > o.CX-o.HX-bodyRadius && p.X < o.CX+o.HX+bodyRadius &&
				p.Z > o.CZ-o.HZ-bodyRadius && p.Z < o.CZ+o.HZ+bodyRadius {
				return false
			}
		}
		return true
	}

	// Sample the aggro-immune corridor: full width, screen line to lobby gate.
	var shooters []entity.Vec3
	for x := float32(-11); x <= 11; x++ {
		for z := aggroMaxZ; z <= 45; z += 0.5 {
			p := entity.Vec3{X: x, Y: eyeY, Z: z}
			if standable(p) {
				shooters = append(shooters, p)
			}
		}
	}
	// Boss patrol band (x -5..5 at z=0) padded by its radius.
	var targets []entity.Vec3
	for x := float32(-6); x <= 6; x++ {
		targets = append(targets, entity.Vec3{X: x, Y: eyeY, Z: 0})
	}
	for _, s := range shooters {
		for _, b := range targets {
			if !combat.SegmentHitsExpandedObstacle(s, b, l.Obstacles, 0) {
				t.Fatalf("boss visible from aggro-immune corridor spot: shooter (%.1f, %.1f) sees patrol point (%.1f, 0)",
					s.X, s.Z, b.X)
			}
		}
	}
}

// TestArenaNavmesh_DeclineWalkable guards the decline bake: the navmesh floor
// must descend monotonically from the boss-1 room (y≈0) to the Aceras room
// floor (y≈-4) along the corridor centerline.
func TestArenaNavmesh_DeclineWalkable(t *testing.T) {
	l, err := Load("arena")
	if err != nil {
		t.Fatal(err)
	}
	if l.Navmesh == nil {
		t.Fatal("arena has no navmesh")
	}
	prevY := float32(1.0)
	nearY := float32(0.0)
	for z := float32(-16); z >= -39; z -= 1.0 {
		y, ok := l.Navmesh.SampleY(0, z, nearY)
		if !ok {
			t.Fatalf("no walkable navmesh polygon at (0, %.1f) — decline bake is broken", z)
		}
		if y > prevY+0.01 {
			t.Errorf("floor rises at z=%.1f: y=%.2f > prev %.2f", z, y, prevY)
		}
		prevY = y
		nearY = y
	}
	if prevY > -3.5 {
		t.Errorf("decline bottom y = %.2f, want ≈-4", prevY)
	}
	// The Aceras room floor is walkable at its center.
	if y, ok := l.Navmesh.SampleY(0, -58, -4); !ok || y > -3.5 {
		t.Errorf("Aceras room floor: y=%.2f ok=%v, want ≈-4", y, ok)
	}
}

func TestArenaJSON_BoundsContainAllSpawnsAndPortals(t *testing.T) {
	l, err := Load("arena")
	if err != nil {
		t.Fatal(err)
	}
	for i, sp := range l.PlayerSpawns {
		if sp.Position.Z < l.PlayerBoundsMinZ || sp.Position.Z > l.PlayerBoundsMaxZ {
			t.Errorf("player spawn %d Z=%.1f outside bounds [%.1f, %.1f]",
				i, sp.Position.Z, l.PlayerBoundsMinZ, l.PlayerBoundsMaxZ)
		}
		if sp.Position.X < l.PlayerBoundsMinX || sp.Position.X > l.PlayerBoundsMaxX {
			t.Errorf("player spawn %d X=%.1f outside bounds [%.1f, %.1f]",
				i, sp.Position.X, l.PlayerBoundsMinX, l.PlayerBoundsMaxX)
		}
	}
	for i, p := range l.Portals {
		if p.Position.Z-p.InteractionRadius < l.PlayerBoundsMinZ || p.Position.Z+p.InteractionRadius > l.PlayerBoundsMaxZ {
			t.Errorf("portal %d %q Z=%.1f (radius=%.1f) exceeds bounds [%.1f, %.1f]",
				i, p.Name, p.Position.Z, p.InteractionRadius, l.PlayerBoundsMinZ, l.PlayerBoundsMaxZ)
		}
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
	if l.ZoneType != "open_world" {
		t.Errorf("ZoneType = %q, want %q", l.ZoneType, "open_world")
	}
	if l.EnemyRadius != 0 {
		t.Errorf("EnemyRadius = %f, want 0", l.EnemyRadius)
	}
	if len(l.Elevators) != 2 {
		t.Errorf("elevators len = %d, want 2", len(l.Elevators))
	}
	foundPublicLift := false
	for _, ev := range l.Elevators {
		if ev.BottomY == -200 {
			foundPublicLift = true
			break
		}
	}
	if !foundPublicLift {
		t.Error("no elevator with bottom_y = -200 (public lift)")
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
			{ "name": "Entry", "trigger_id": "instance_entry", "axis": "z", "threshold": 40 },
			{ "name": "BossGate", "trigger_id": "boss_room_entry", "axis": "z", "threshold": 12 }
		]
	}`)

	l := &Level{InstanceEntryZ: 99} // will be overwritten by zone_triggers
	if err := loadLevelData(p, l); err != nil {
		t.Fatal(err)
	}

	// Player spawns with conditions
	if len(l.PlayerSpawns) != 3 {
		t.Fatalf("player spawns = %d, want 3", len(l.PlayerSpawns))
	}
	if l.PlayerSpawns[1].Condition != testCondPack1Cleared {
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
	if l.InstanceEntryZ != 40 {
		t.Errorf("InstanceEntryZ = %f, want 40", l.InstanceEntryZ)
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
	l := &Level{InstanceEntryZ: 40.0}
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
	// InstanceEntryZ should be untouched (no zone_triggers in v2 JSON)
	if l.InstanceEntryZ != 40.0 {
		t.Errorf("InstanceEntryZ = %f, want 40 (preserved)", l.InstanceEntryZ)
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
// Load function
// =============================================================================

func TestLoad(t *testing.T) {
	l, err := Load("arena")
	if err != nil {
		t.Fatal(err)
	}
	if l.ZoneType != "instanced" {
		t.Errorf("ZoneType = %q, want instanced", l.ZoneType)
	}
	if l.EnemyRadius != 1.0 {
		t.Errorf("EnemyRadius = %f, want 1", l.EnemyRadius)
	}
	if len(l.PlayerSpawns) != 8 {
		t.Errorf("PlayerSpawns = %d, want 8", len(l.PlayerSpawns))
	}
}

func TestLoad_MissingFile(t *testing.T) {
	t.Setenv("CODEX_LEVELS_DIR", "/nonexistent")
	_, err := Load("does_not_exist")
	if err == nil {
		t.Fatal("expected error for missing level file")
	}
}

func TestLoad_DefaultZoneType(t *testing.T) {
	// v2-3 JSON without zone_type should default to "open_world"
	dir := t.TempDir()
	writeTestJSON(t, dir, "test_default", `{
		"version": 3,
		"zone": "test_default",
		"bounds": {"min_x":-10,"max_x":10,"min_z":-10,"max_z":10},
		"obstacles": [],
		"player_spawns": [{"x":0,"y":0,"z":0}]
	}`)
	t.Setenv("CODEX_LEVELS_DIR", dir)
	l, err := Load("test_default")
	if err != nil {
		t.Fatal(err)
	}
	if l.ZoneType != "open_world" {
		t.Errorf("ZoneType = %q, want open_world", l.ZoneType)
	}
}

func TestLoadLevelData_V4Features(t *testing.T) {
	p := writeTempJSON(t, `{
		"version": 4,
		"zone": "test",
		"zone_type": "instanced",
		"enemy_radius": 1.5,
		"source_scene": "res://test.tscn",
		"bounds": { "min_x": -10, "max_x": 10, "min_y": -1, "max_y": 5, "min_z": -10, "max_z": 10 },
		"obstacles": [],
		"player_spawns": [{ "x": 0, "y": 0.1, "z": 5 }],
		"zone_triggers": [
			{ "name": "Entry", "trigger_id": "instance_entry", "axis": "z", "threshold": 40 }
		]
	}`)
	l := &Level{}
	if err := loadLevelData(p, l); err != nil {
		t.Fatal(err)
	}
	if l.ZoneType != "instanced" {
		t.Errorf("ZoneType = %q, want %q", l.ZoneType, "instanced")
	}
	if l.EnemyRadius != 1.5 {
		t.Errorf("EnemyRadius = %f, want 1.5", l.EnemyRadius)
	}
	if l.InstanceEntryZ != 40 {
		t.Errorf("InstanceEntryZ = %f, want 40", l.InstanceEntryZ)
	}
}

func TestLoadLevelData_ClearTimeSeconds(t *testing.T) {
	p := writeTempJSON(t, `{
		"version": 6,
		"zone": "test",
		"zone_type": "instanced",
		"clear_time_seconds": 600,
		"source_scene": "res://test.tscn",
		"bounds": { "min_x": -10, "max_x": 10, "min_y": -1, "max_y": 5, "min_z": -10, "max_z": 10 },
		"obstacles": [],
		"player_spawns": [{ "x": 0, "y": 0.1, "z": 5 }]
	}`)
	l := &Level{}
	if err := loadLevelData(p, l); err != nil {
		t.Fatal(err)
	}
	if l.ClearTimeSeconds != 600 {
		t.Errorf("ClearTimeSeconds = %f, want 600", l.ClearTimeSeconds)
	}
}

func TestLoadLevelData_V7BossFields(t *testing.T) {
	p := writeTempJSON(t, `{
		"version": 7,
		"zone": "test",
		"source_scene": "res://test.tscn",
		"bounds": { "min_x": -10, "max_x": 10, "min_y": -1, "max_y": 5, "min_z": -80, "max_z": 10 },
		"obstacles": [],
		"player_spawns": [{ "x": 0, "y": 0.1, "z": 5 }],
		"enemy_spawns": [
			{
				"x": 0, "y": 0.1, "z": 0, "def_name": "boss1", "is_boss": true,
				"patrol_a": { "x": -5, "y": 0.1, "z": 0 }, "patrol_b": { "x": 5, "y": 0.1, "z": 0 },
				"aggro_radius": 10, "leash_radius": 30,
				"boss_num": 1, "boss_gate_id": "boss_gate", "aggro_max_z": 12.0
			},
			{
				"x": 0, "y": -3.9, "z": -58, "def_name": "boss2", "is_boss": true,
				"patrol_a": { "x": -6, "y": -3.9, "z": -58 }, "patrol_b": { "x": 6, "y": -3.9, "z": -58 },
				"aggro_radius": 12, "leash_radius": 40,
				"boss_num": 2, "boss_gate_id": "aceras_gate", "aggro_max_z": -40.0
			},
			{
				"x": 3, "y": 0.1, "z": 22, "def_name": "mob",
				"patrol_a": { "x": -5, "y": 0.1, "z": 22 }, "patrol_b": { "x": 5, "y": 0.1, "z": 22 },
				"aggro_radius": 10, "leash_radius": 30
			}
		]
	}`)
	l := &Level{}
	if err := loadLevelData(p, l); err != nil {
		t.Fatal(err)
	}
	if len(l.EnemySpawns) != 3 {
		t.Fatalf("enemy spawns = %d, want 3", len(l.EnemySpawns))
	}
	b1 := l.EnemySpawns[0]
	if b1.BossNum != 1 || b1.BossGateID != "boss_gate" || b1.AggroMaxZ != 12.0 {
		t.Errorf("boss1 spawn = %+v, want BossNum=1 BossGateID=boss_gate AggroMaxZ=12", b1)
	}
	b2 := l.EnemySpawns[1]
	if b2.BossNum != 2 || b2.BossGateID != "aceras_gate" || b2.AggroMaxZ != -40.0 {
		t.Errorf("boss2 spawn = %+v, want BossNum=2 BossGateID=aceras_gate AggroMaxZ=-40", b2)
	}
	mob := l.EnemySpawns[2]
	if mob.BossNum != 0 || mob.BossGateID != "" || mob.AggroMaxZ != 0 {
		t.Errorf("plain mob spawn = %+v, want zero-value boss fields", mob)
	}
}

func TestLoadLevelData_ClearTimeDefaultsZeroWhenAbsent(t *testing.T) {
	p := writeTempJSON(t, `{
		"version": 6,
		"zone": "test",
		"source_scene": "res://test.tscn",
		"bounds": { "min_x": -10, "max_x": 10, "min_y": -1, "max_y": 5, "min_z": -10, "max_z": 10 },
		"obstacles": [],
		"player_spawns": [{ "x": 0, "y": 0.1, "z": 5 }]
	}`)
	l := &Level{}
	if err := loadLevelData(p, l); err != nil {
		t.Fatal(err)
	}
	if l.ClearTimeSeconds != 0 {
		t.Errorf("ClearTimeSeconds = %f, want 0 (unset, gameflow falls back to default)", l.ClearTimeSeconds)
	}
}

func writeTestJSON(t *testing.T, dir, name, content string) {
	t.Helper()
	err := os.WriteFile(filepath.Join(dir, name+".json"), []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}
}

// TestArenaBunker_ScreenAndCeilings guards the bunker rework (2026-07-14):
// the section past the guard captain is enclosed (ceilings are client-side
// visuals and must NOT leak into server obstacles or the navmesh), and the
// General's doorway has the same screen protection the captain's entrance
// has: a freestanding wall just inside the gate that breaks line of sight
// through the opening.
func TestArenaBunker_ScreenAndCeilings(t *testing.T) {
	l, err := Load("arena")
	if err != nil {
		t.Fatal(err)
	}

	// Door screen: a wide, non-pillar obstacle a few meters past the aceras
	// gate (z=-40), centered on the door gap.
	foundScreen := false
	for _, o := range l.Obstacles {
		if o.CX > -1 && o.CX < 1 && o.CZ < -41 && o.CZ > -48 && o.HX >= 5 {
			foundScreen = true
		}
	}
	if !foundScreen {
		t.Error("no door screen obstacle inside the General's room (want a wide wall at x≈0, z≈-44)")
	}

	// Ceilings must not leak into obstacles: nothing hovering at or above the
	// decline wall tops (y=5).
	for _, o := range l.Obstacles {
		if o.BaseY >= 4.5 {
			t.Errorf("obstacle with BaseY %.1f at (%.1f, %.1f) — ceiling leaked into server obstacles", o.BaseY, o.CX, o.CZ)
		}
	}

	// Ceilings must not pollute the navmesh: inside the bunker (everything
	// past the decline gate at z=-15) every walkable vertex stays at floor
	// level (floors span y ≈ -4.25..0.25). Scoped to the bunker because the
	// open-air area has pre-existing bake islands on set-dressing prop roofs.
	if l.Navmesh == nil {
		t.Fatal("arena has no navmesh")
	}
	for _, poly := range l.Navmesh.Polys {
		for _, v := range poly.Vertices {
			if v.Z < -14 && v.Y > 2.0 {
				t.Fatalf("navmesh vertex at y=%.2f (%.1f, %.1f) — ceiling polluted the bunker bake", v.Y, v.X, v.Z)
			}
		}
	}
}
