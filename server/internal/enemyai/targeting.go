package enemyai

import (
	"math"
	"slices"

	"codex-online/server/internal/entity"
)

// NNearestAlivePlayers returns up to n alive players closest to pos, nearest
// first. Used by multi-target ("twin-lock") abilities to fire at several players.
func NNearestAlivePlayers(pos entity.Vec3, players []*entity.Player, n int) []*entity.Player {
	if n <= 0 {
		return nil
	}
	out := make([]*entity.Player, 0, len(players))
	for _, p := range players {
		if p.Alive {
			out = append(out, p)
		}
	}
	slices.SortFunc(out, func(a, b *entity.Player) int {
		da, db := a.Position.DistanceToSq(pos), b.Position.DistanceToSq(pos)
		switch {
		case da < db:
			return -1
		case da > db:
			return 1
		default:
			return 0
		}
	})
	if len(out) > n {
		out = out[:n]
	}
	return out
}

// NearestAlivePlayer returns the closest alive player to pos, or nil.
func NearestAlivePlayer(pos entity.Vec3, players []*entity.Player) *entity.Player {
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
func FarthestAlivePlayer(pos entity.Vec3, players []*entity.Player) *entity.Player {
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
