package level

import (
	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
)

// NewArenaLevel returns the arena level definition.
func NewArenaLevel() *Level {
	return &Level{
		// Players can move in the warmup room (Z up to 24.5) and the arena
		PlayerBoundsMinX: -19.5,
		PlayerBoundsMaxX: 19.5,
		PlayerBoundsMinZ: -14.5,
		PlayerBoundsMaxZ: 24.5,

		// Enemy stays within the arena (not the warmup room)
		EnemyBoundsMinX: -19.5,
		EnemyBoundsMaxX: 19.5,
		EnemyBoundsMinZ: -14.5,
		EnemyBoundsMaxZ: 14.5,

		EnemyRadius: 1.0,
		ArenaEntryZ: 12.0,

		// 6 pillars (1.5x1.5, infinitely tall) + 4 cover boxes (half-height, 1.2m)
		Obstacles: []combat.Obstacle{
			// Pillars
			{CX: -8, CZ: -6, HX: 0.75, HZ: 0.75},
			{CX: 8, CZ: -6, HX: 0.75, HZ: 0.75},
			{CX: -8, CZ: 6, HX: 0.75, HZ: 0.75},
			{CX: 8, CZ: 6, HX: 0.75, HZ: 0.75},
			{CX: 0, CZ: -10, HX: 0.75, HZ: 0.75},
			{CX: 0, CZ: 10, HX: 0.75, HZ: 0.75},
			// Cover boxes (1.2m tall — boss projectiles at Y=1.5 pass over)
			{CX: -5, CZ: -2, HX: 1.5, HZ: 0.5, Height: 1.2},
			{CX: 5, CZ: 2, HX: 1.5, HZ: 0.5, Height: 1.2},
			{CX: -12, CZ: 0, HX: 0.5, HZ: 1.5, Height: 1.2},
			{CX: 12, CZ: 0, HX: 0.5, HZ: 1.5, Height: 1.2},
		},

		PlayerSpawns: []entity.Vec3{
			{X: -2.0, Y: 0.1, Z: 20.0},
			{X: 0.0, Y: 0.1, Z: 20.0},
			{X: 2.0, Y: 0.1, Z: 20.0},
			{X: -1.0, Y: 0.1, Z: 21.0},
			{X: 1.0, Y: 0.1, Z: 21.0},
		},
		EnemySpawn: entity.Vec3{X: 0.0, Y: 0.1, Z: 0.0},
	}
}
