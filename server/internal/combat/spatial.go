package combat

import (
	"math"

	"codex-online/server/internal/entity"
)

// PushOutOfObstacles resolves collisions between a position and obstacles,
// expanding each obstacle by the given radius (Minkowski sum).
func PushOutOfObstacles(pos *entity.Vec3, obstacles []Obstacle, radius float32) {
	for _, obs := range obstacles {
		exHx := obs.HX + radius
		exHz := obs.HZ + radius
		dx := pos.X - obs.CX
		dz := pos.Z - obs.CZ
		if dx > -exHx && dx < exHx && dz > -exHz && dz < exHz {
			pushX := exHx - abs32(dx)
			pushZ := exHz - abs32(dz)
			if pushX < pushZ {
				if dx > 0 {
					pos.X = obs.CX + exHx
				} else {
					pos.X = obs.CX - exHx
				}
			} else {
				if dz > 0 {
					pos.Z = obs.CZ + exHz
				} else {
					pos.Z = obs.CZ - exHz
				}
			}
		}
	}
}

// IsAtWall checks if a position is at the arena boundary.
func IsAtWall(pos entity.Vec3, minX, maxX, minZ, maxZ float32) bool {
	const margin float32 = 0.5
	return pos.X <= minX+margin || pos.X >= maxX-margin ||
		pos.Z <= minZ+margin || pos.Z >= maxZ-margin
}

// IsAtObstacle returns true if pos is touching an obstacle expanded by radius.
func IsAtObstacle(pos entity.Vec3, obstacles []Obstacle, radius float32) bool {
	const margin float32 = 0.1
	for _, obs := range obstacles {
		exHx := obs.HX + radius + margin
		exHz := obs.HZ + radius + margin
		dx := pos.X - obs.CX
		dz := pos.Z - obs.CZ
		if dx > -exHx && dx < exHx && dz > -exHz && dz < exHz {
			return true
		}
	}
	return false
}

// RotateVecY rotates a direction vector around the Y axis by angle (radians).
func RotateVecY(v entity.Vec3, angle float32) entity.Vec3 {
	s := float32(math.Sin(float64(angle)))
	c := float32(math.Cos(float64(angle)))
	return entity.Vec3{
		X: v.X*c + v.Z*s,
		Y: v.Y,
		Z: -v.X*s + v.Z*c,
	}
}
