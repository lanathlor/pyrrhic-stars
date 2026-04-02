package enemyai

import (
	"math"

	"codex-online/server/internal/entity"
)

// NearestAlivePlayer returns the closest alive player to pos, or nil.
func NearestAlivePlayer(pos entity.Vec3, players map[uint16]*entity.Player) *entity.Player {
	var best *entity.Player
	bestDist := float32(math.MaxFloat32)
	for _, p := range players {
		if !p.Alive {
			continue
		}
		d := p.Position.DistanceToSq(pos)
		if d < bestDist {
			bestDist = d
			best = p
		}
	}
	return best
}

// FarthestAlivePlayer returns the farthest alive player from pos, or nil.
func FarthestAlivePlayer(pos entity.Vec3, players map[uint16]*entity.Player) *entity.Player {
	var best *entity.Player
	var bestDist float32
	for _, p := range players {
		if !p.Alive {
			continue
		}
		d := p.Position.DistanceToSq(pos)
		if d > bestDist {
			bestDist = d
			best = p
		}
	}
	return best
}

// FaceToward computes the yaw rotation to face from pos toward target (Godot convention).
func FaceToward(from, to entity.Vec3) float32 {
	dir := to.Sub(from).Flat()
	if dir.Length() > 0.1 {
		return float32(math.Atan2(float64(-dir.X), float64(-dir.Z)))
	}
	return 0
}
