package level

import "codex-online/server/internal/entity"

// NewHubLevel returns the hub level definition.
func NewHubLevel() *Level {
	return &Level{
		// Tower interior X[-12,12] + landing pad extends to X=24
		PlayerBoundsMinX: -12.0,
		PlayerBoundsMaxX: 24.0,
		PlayerBoundsMinZ: -2.0,
		PlayerBoundsMaxZ: 22.0,

		PlayerSpawns: []entity.Vec3{
			{X: -1.5, Y: 0.1, Z: 3.0},
			{X: 0.0, Y: 0.1, Z: 3.0},
			{X: 1.5, Y: 0.1, Z: 3.0},
			{X: -0.75, Y: 0.1, Z: 4.0},
			{X: 0.75, Y: 0.1, Z: 4.0},
		},
	}
}
