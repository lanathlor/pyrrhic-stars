package level

import (
	"log/slog"
	"os"
	"path/filepath"

	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
)

// NewArenaLevel returns the dungeon level: warmup room → hallway (2 trash packs) → boss room.
// Loads geometry from shared/levels/arena.json if available, falls back to hardcoded values.
func NewArenaLevel() *Level {
	l := &Level{
		// Game logic constants (not geometry):
		ArenaEntryZ:    40.0,
		BossRoomEntryZ: 12.0,
		EnemyRadius:    1.0,
	}

	path := levelDataPath("arena")
	if err := loadLevelData(path, l); err != nil {
		slog.Warn("arena level data not found, using hardcoded fallback", "path", path, "err", err)
		return hardcodedArenaLevel()
	}
	return l
}

func hardcodedArenaLevel() *Level { //nolint:funlen // pure data literal
	return &Level{
		// Players can move in warmup + hallway + boss room
		PlayerBoundsMinX: -19.5,
		PlayerBoundsMaxX: 19.5,
		PlayerBoundsMinY: -1.0,
		PlayerBoundsMaxY: 6.0,
		PlayerBoundsMinZ: -14.5,
		PlayerBoundsMaxZ: 52.0,

		// Enemies can go anywhere in the instance
		EnemyBoundsMinX: -19.5,
		EnemyBoundsMaxX: 19.5,
		EnemyBoundsMinZ: -14.5,
		EnemyBoundsMaxZ: 52.0,

		EnemyRadius:    1.0,
		ArenaEntryZ:    40.0,
		BossRoomEntryZ: 12.0,

		Obstacles: []combat.Obstacle{
			// Boss room pillars (1.5x1.5, infinitely tall)
			{CX: -8, CZ: -6, HX: 0.75, HZ: 0.75},
			{CX: 8, CZ: -6, HX: 0.75, HZ: 0.75},
			{CX: -8, CZ: 6, HX: 0.75, HZ: 0.75},
			{CX: 8, CZ: 6, HX: 0.75, HZ: 0.75},
			{CX: 0, CZ: -10, HX: 0.75, HZ: 0.75},
			{CX: 0, CZ: 10, HX: 0.75, HZ: 0.75},
			// Boss room cover boxes (1.2m tall — boss projectiles at Y=1.5 pass over)
			{CX: -5, CZ: -2, HX: 1.5, HZ: 0.5, Height: 1.2},
			{CX: 5, CZ: 2, HX: 1.5, HZ: 0.5, Height: 1.2},
			{CX: -12, CZ: 0, HX: 0.5, HZ: 1.5, Height: 1.2},
			{CX: 12, CZ: 0, HX: 0.5, HZ: 1.5, Height: 1.2},
			// Connector walls between hallway (X ±8) and boss room (X ±20) at Z=13
			{CX: -14, CZ: 11.6, HX: 6.0, HZ: 0.25},
			{CX: 14, CZ: 11.6, HX: 6.0, HZ: 0.25},
			// Hallway side walls (X ±8, Z 12-40)
			{CX: -8, CZ: 26, HX: 0.25, HZ: 14.0},
			{CX: 8, CZ: 26, HX: 0.25, HZ: 14.0},
			// Hallway cover crates (waist-height, 1.2m)
			{CX: 0, CZ: 27, HX: 1.0, HZ: 0.5, Height: 1.2},
			{CX: -4, CZ: 30, HX: 0.5, HZ: 0.5, Height: 1.2},
			{CX: 4, CZ: 30, HX: 0.5, HZ: 0.5, Height: 1.2},
			{CX: 0, CZ: 18, HX: 1.0, HZ: 0.5, Height: 1.2},
			{CX: -4, CZ: 20, HX: 0.5, HZ: 0.5, Height: 1.2},
			{CX: 4, CZ: 20, HX: 0.5, HZ: 0.5, Height: 1.2},
		},

		PlayerSpawns: []PlayerSpawn{
			{Position: entity.Vec3{X: -2.0, Y: 0.1, Z: 48.0}},
			{Position: entity.Vec3{X: 0.0, Y: 0.1, Z: 48.0}},
			{Position: entity.Vec3{X: 2.0, Y: 0.1, Z: 48.0}},
			{Position: entity.Vec3{X: -1.0, Y: 0.1, Z: 49.0}},
			{Position: entity.Vec3{X: 1.0, Y: 0.1, Z: 49.0}},
		},

		EnemySpawns: []EnemySpawnPoint{
			// Pack 1 at Z≈32: 2 melee + 2 ranged (GroupID=1)
			{
				Position: entity.Vec3{X: -3, Y: 0.1, Z: 32}, DefName: EnemyHallwayMelee,
				PatrolA: entity.Vec3{X: -6, Y: 0.1, Z: 32}, PatrolB: entity.Vec3{X: 6, Y: 0.1, Z: 32},
				AggroRadius: 10, LeashRadius: 40, GroupID: 1,
			},
			{
				Position: entity.Vec3{X: 3, Y: 0.1, Z: 32}, DefName: EnemyHallwayMelee,
				PatrolA: entity.Vec3{X: 6, Y: 0.1, Z: 32}, PatrolB: entity.Vec3{X: -6, Y: 0.1, Z: 32},
				AggroRadius: 10, LeashRadius: 40, GroupID: 1,
			},
			{
				Position: entity.Vec3{X: -5, Y: 0.1, Z: 34}, DefName: EnemyHallwayRanged,
				PatrolA: entity.Vec3{X: -6, Y: 0.1, Z: 34}, PatrolB: entity.Vec3{X: 6, Y: 0.1, Z: 34},
				AggroRadius: 12, LeashRadius: 40, GroupID: 1,
			},
			{
				Position: entity.Vec3{X: 5, Y: 0.1, Z: 34}, DefName: EnemyHallwayRanged,
				PatrolA: entity.Vec3{X: 6, Y: 0.1, Z: 34}, PatrolB: entity.Vec3{X: -6, Y: 0.1, Z: 34},
				AggroRadius: 12, LeashRadius: 40, GroupID: 1,
			},
			// Pack 2 at Z≈22: 2 melee + 2 ranged (GroupID=2)
			{
				Position: entity.Vec3{X: -3, Y: 0.1, Z: 22}, DefName: EnemyHallwayMelee,
				PatrolA: entity.Vec3{X: -6, Y: 0.1, Z: 22}, PatrolB: entity.Vec3{X: 6, Y: 0.1, Z: 22},
				AggroRadius: 10, LeashRadius: 40, GroupID: 2,
			},
			{
				Position: entity.Vec3{X: 3, Y: 0.1, Z: 22}, DefName: EnemyHallwayMelee,
				PatrolA: entity.Vec3{X: 6, Y: 0.1, Z: 22}, PatrolB: entity.Vec3{X: -6, Y: 0.1, Z: 22},
				AggroRadius: 10, LeashRadius: 40, GroupID: 2,
			},
			{
				Position: entity.Vec3{X: -5, Y: 0.1, Z: 24}, DefName: EnemyHallwayRanged,
				PatrolA: entity.Vec3{X: -6, Y: 0.1, Z: 24}, PatrolB: entity.Vec3{X: 6, Y: 0.1, Z: 24},
				AggroRadius: 12, LeashRadius: 40, GroupID: 2,
			},
			{
				Position: entity.Vec3{X: 5, Y: 0.1, Z: 24}, DefName: EnemyHallwayRanged,
				PatrolA: entity.Vec3{X: 6, Y: 0.1, Z: 24}, PatrolB: entity.Vec3{X: -6, Y: 0.1, Z: 24},
				AggroRadius: 12, LeashRadius: 40, GroupID: 2,
			},
			// Boss at center of boss room (no group)
			{
				Position: entity.Vec3{X: 0, Y: 0.1, Z: 0}, DefName: "guard_captain",
				PatrolA: entity.Vec3{X: -5, Y: 0.1, Z: 0}, PatrolB: entity.Vec3{X: 5, Y: 0.1, Z: 0},
				IsBoss: true, AggroRadius: 10, LeashRadius: 30,
			},
		},
	}
}

// levelDataPath returns the path to a level JSON file.
// Checks CODEX_LEVELS_DIR env var first, then walks up from CWD looking for shared/levels/.
func levelDataPath(zone string) string {
	dir := os.Getenv("CODEX_LEVELS_DIR")
	if dir != "" {
		return filepath.Join(dir, zone+".json")
	}
	// Walk up from CWD to find shared/levels/
	cwd, _ := os.Getwd()
	for d := cwd; d != "/" && d != "."; d = filepath.Dir(d) {
		candidate := filepath.Join(d, "shared", "levels", zone+".json")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	// Last resort: relative path (works when run from server/)
	return filepath.Join("..", "shared", "levels", zone+".json")
}
