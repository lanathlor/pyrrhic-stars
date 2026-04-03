package level

import "codex-online/server/internal/entity"

// NewHubLevel returns the hub level definition.
func NewHubLevel() *Level {
	return &Level{
		PlayerBoundsMinX: -14.5,
		PlayerBoundsMaxX: 14.5,
		PlayerBoundsMinZ: -9.5,
		PlayerBoundsMaxZ: 14.5,

		PlayerSpawns: []entity.Vec3{
			{X: -2.0, Y: 0.9, Z: -5.0},
			{X: 0.0, Y: 0.9, Z: -5.0},
			{X: 2.0, Y: 0.9, Z: -5.0},
			{X: -1.0, Y: 0.9, Z: -4.0},
			{X: 1.0, Y: 0.9, Z: -4.0},
		},
	}
}
