package level

import "codex-online/server/internal/entity"

// NewHubLevel returns the hub level definition.
func NewHubLevel() *Level {
	return &Level{
		// Upper plaza + lower streets (200m radius from lift)
		PlayerBoundsMinX: -125.0,
		PlayerBoundsMaxX: 125.0,
		PlayerBoundsMinZ: -160.0,
		PlayerBoundsMaxZ: 160.0,

		PlayerSpawns: []entity.Vec3{
			{X: 3.5, Y: -199.8, Z: -55.0},
			{X: 5.0, Y: -199.8, Z: -55.0},
			{X: 6.5, Y: -199.8, Z: -55.0},
			{X: 4.25, Y: -199.8, Z: -53.5},
			{X: 5.75, Y: -199.8, Z: -53.5},
		},
	}
}
