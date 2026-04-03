package level

import (
	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
)

// NewArenaLevel returns the dungeon level: warmup room → hallway (2 trash packs) → boss room.
//
// Layout (Z axis, north is positive):
//
//	Z=52     Warmup room top
//	Z=40     Hallway entry (ArenaEntryZ — fight starts when player crosses south)
//	Z=32     Pack 1 (4 mobs patrolling X-axis)
//	Z=22     Pack 2 (4 mobs patrolling X-axis)
//	Z=12     Boss room entry (BossRoomEntryZ — gate opens after trash cleared)
//	Z=0      Boss spawn
//	Z=-14.5  Boss room south wall
func NewArenaLevel() *Level {
	return &Level{
		// Players can move in warmup + hallway + boss room
		PlayerBoundsMinX: -19.5,
		PlayerBoundsMaxX: 19.5,
		PlayerBoundsMinZ: -14.5,
		PlayerBoundsMaxZ: 52.0,

		// Enemies can go anywhere in the instance
		EnemyBoundsMinX: -19.5,
		EnemyBoundsMaxX: 19.5,
		EnemyBoundsMinZ: -14.5,
		EnemyBoundsMaxZ: 52.0,

		EnemyRadius: 1.0,
		ArenaEntryZ:    40.0, // hallway entry: fight starts here
		BossRoomEntryZ: 12.0, // boss room gate

		// Boss room obstacles (unchanged from original)
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
			// Hallway cover crates (waist-height, 1.2m)
			{CX: 0, CZ: 27, HX: 1.0, HZ: 0.5, Height: 1.2},
			{CX: -4, CZ: 30, HX: 0.5, HZ: 0.5, Height: 1.2},
			{CX: 4, CZ: 30, HX: 0.5, HZ: 0.5, Height: 1.2},
			{CX: 0, CZ: 18, HX: 1.0, HZ: 0.5, Height: 1.2},
			{CX: -4, CZ: 20, HX: 0.5, HZ: 0.5, Height: 1.2},
			{CX: 4, CZ: 20, HX: 0.5, HZ: 0.5, Height: 1.2},
		},

		PlayerSpawns: []entity.Vec3{
			{X: -2.0, Y: 0.1, Z: 48.0},
			{X: 0.0, Y: 0.1, Z: 48.0},
			{X: 2.0, Y: 0.1, Z: 48.0},
			{X: -1.0, Y: 0.1, Z: 49.0},
			{X: 1.0, Y: 0.1, Z: 49.0},
		},

		EnemySpawns: []EnemySpawnPoint{
			// Pack 1 at Z≈32: 2 melee + 2 ranged (GroupID=1)
			{
				Position: entity.Vec3{X: -3, Y: 0.1, Z: 32}, DefName: "hallway_melee",
				PatrolA: entity.Vec3{X: -6, Y: 0.1, Z: 32}, PatrolB: entity.Vec3{X: 6, Y: 0.1, Z: 32},
				AggroRadius: 10, LeashRadius: 40, GroupID: 1,
			},
			{
				Position: entity.Vec3{X: 3, Y: 0.1, Z: 32}, DefName: "hallway_melee",
				PatrolA: entity.Vec3{X: 6, Y: 0.1, Z: 32}, PatrolB: entity.Vec3{X: -6, Y: 0.1, Z: 32},
				AggroRadius: 10, LeashRadius: 40, GroupID: 1,
			},
			{
				Position: entity.Vec3{X: -5, Y: 0.1, Z: 34}, DefName: "hallway_ranged",
				PatrolA: entity.Vec3{X: -6, Y: 0.1, Z: 34}, PatrolB: entity.Vec3{X: 6, Y: 0.1, Z: 34},
				AggroRadius: 12, LeashRadius: 40, GroupID: 1,
			},
			{
				Position: entity.Vec3{X: 5, Y: 0.1, Z: 34}, DefName: "hallway_ranged",
				PatrolA: entity.Vec3{X: 6, Y: 0.1, Z: 34}, PatrolB: entity.Vec3{X: -6, Y: 0.1, Z: 34},
				AggroRadius: 12, LeashRadius: 40, GroupID: 1,
			},
			// Pack 2 at Z≈22: 2 melee + 2 ranged (GroupID=2)
			{
				Position: entity.Vec3{X: -3, Y: 0.1, Z: 22}, DefName: "hallway_melee",
				PatrolA: entity.Vec3{X: -6, Y: 0.1, Z: 22}, PatrolB: entity.Vec3{X: 6, Y: 0.1, Z: 22},
				AggroRadius: 10, LeashRadius: 40, GroupID: 2,
			},
			{
				Position: entity.Vec3{X: 3, Y: 0.1, Z: 22}, DefName: "hallway_melee",
				PatrolA: entity.Vec3{X: 6, Y: 0.1, Z: 22}, PatrolB: entity.Vec3{X: -6, Y: 0.1, Z: 22},
				AggroRadius: 10, LeashRadius: 40, GroupID: 2,
			},
			{
				Position: entity.Vec3{X: -5, Y: 0.1, Z: 24}, DefName: "hallway_ranged",
				PatrolA: entity.Vec3{X: -6, Y: 0.1, Z: 24}, PatrolB: entity.Vec3{X: 6, Y: 0.1, Z: 24},
				AggroRadius: 12, LeashRadius: 40, GroupID: 2,
			},
			{
				Position: entity.Vec3{X: 5, Y: 0.1, Z: 24}, DefName: "hallway_ranged",
				PatrolA: entity.Vec3{X: 6, Y: 0.1, Z: 24}, PatrolB: entity.Vec3{X: -6, Y: 0.1, Z: 24},
				AggroRadius: 12, LeashRadius: 40, GroupID: 2,
			},
			// Boss at center of boss room (no group)
			{
				Position: entity.Vec3{X: 0, Y: 0.1, Z: 0}, DefName: "guard_captain",
				PatrolA: entity.Vec3{X: -5, Y: 0.1, Z: 0}, PatrolB: entity.Vec3{X: 5, Y: 0.1, Z: 0},
				IsBoss: true, AggroRadius: 15, LeashRadius: 30,
			},
		},
	}
}
