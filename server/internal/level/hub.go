package level

import "codex-online/server/internal/entity"

// NewHubLevel returns the hub level definition.
func NewHubLevel() *Level {
	return &Level{
		// 250m x 250m hub plaza centered around tower
		PlayerBoundsMinX: -125.0,
		PlayerBoundsMaxX: 125.0,
		PlayerBoundsMinZ: -115.0,
		PlayerBoundsMaxZ: 135.0,

		PlayerSpawns: []entity.Vec3{
			{X: -1.5, Y: 0.1, Z: 3.0},
			{X: 0.0, Y: 0.1, Z: 3.0},
			{X: 1.5, Y: 0.1, Z: 3.0},
			{X: -0.75, Y: 0.1, Z: 4.0},
			{X: 0.75, Y: 0.1, Z: 4.0},
		},
	}
}
