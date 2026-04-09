package level

import (
	"log/slog"

	"codex-online/server/internal/entity"
)

// NewHubLevel returns the hub level definition.
// Loads geometry from shared/levels/hub.json if available, falls back to hardcoded values.
func NewHubLevel() *Level {
	l := &Level{}

	path := levelDataPath("hub")
	if err := loadLevelData(path, l); err != nil {
		slog.Warn("hub level data not found, using hardcoded fallback", "path", path, "err", err)
		return hardcodedHubLevel()
	}
	return l
}

func hardcodedHubLevel() *Level {
	return &Level{
		// Upper plaza + lower streets
		PlayerBoundsMinX: -125.0,
		PlayerBoundsMaxX: 125.0,
		PlayerBoundsMinY: -210.0,
		PlayerBoundsMaxY: 110.0,
		PlayerBoundsMinZ: -160.0,
		PlayerBoundsMaxZ: 160.0,

		Elevators: []ElevatorVolume{
			{
				CenterX: 5.0, CenterZ: -55.0,
				HalfX: 4.0, HalfZ: 4.0,
				BottomY: -200.0, TopY: 0.0,
				Speed: 10.0,
			},
			{
				CenterX: 0.0, CenterZ: 0.0,
				HalfX: 2.0, HalfZ: 2.0,
				BottomY: 0.0, TopY: 100.0,
				Speed: 12.5,
			},
		},

		PlayerSpawns: []entity.Vec3{
			{X: 3.5, Y: -199.8, Z: -55.0},
			{X: 5.0, Y: -199.8, Z: -55.0},
			{X: 6.5, Y: -199.8, Z: -55.0},
			{X: 4.25, Y: -199.8, Z: -53.5},
			{X: 5.75, Y: -199.8, Z: -53.5},
		},
	}
}
