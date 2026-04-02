package combat

import (
	"math"

	"codex-online/server/internal/entity"
)

// Obstacle represents a rectangular obstacle in the arena.
type Obstacle struct {
	CX, CZ float32 // center (XZ plane)
	HX, HZ float32 // half-extents (XZ plane)
	Height float32 // obstacle height from ground (0 = infinitely tall)
}

// SegmentHitsObstacle checks if a line segment from a to b (on the XZ plane)
// intersects any obstacle. Uses slab intersection (ray-vs-AABB in 2D).
// Obstacles that contain point a (the origin) are skipped — you can shoot
// out of geometry you're standing in.
func SegmentHitsObstacle(a, b entity.Vec3, obstacles []Obstacle) bool {
	dx := b.X - a.X
	dz := b.Z - a.Z
	length := float32(math.Sqrt(float64(dx*dx + dz*dz)))
	if length < 1e-6 {
		return false
	}

	for _, obs := range obstacles {
		// Compute entry/exit t along the segment for both axes
		minX := obs.CX - obs.HX
		maxX := obs.CX + obs.HX
		minZ := obs.CZ - obs.HZ
		maxZ := obs.CZ + obs.HZ

		// Skip obstacles that contain the origin or target point —
		// entities standing inside geometry can still shoot/be hit.
		if a.X >= minX && a.X <= maxX && a.Z >= minZ && a.Z <= maxZ {
			continue
		}
		if b.X >= minX && b.X <= maxX && b.Z >= minZ && b.Z <= maxZ {
			continue
		}

		var tMin, tMax float32 = 0, 1

		// X slab
		if abs32(dx) < 1e-6 {
			// Ray parallel to X slab — check if origin inside
			if a.X < minX || a.X > maxX {
				continue
			}
		} else {
			invD := 1.0 / dx
			t1 := (minX - a.X) * invD
			t2 := (maxX - a.X) * invD
			if t1 > t2 {
				t1, t2 = t2, t1
			}
			if t1 > tMin {
				tMin = t1
			}
			if t2 < tMax {
				tMax = t2
			}
			if tMin > tMax {
				continue
			}
		}

		// Z slab
		if abs32(dz) < 1e-6 {
			if a.Z < minZ || a.Z > maxZ {
				continue
			}
		} else {
			invD := 1.0 / dz
			t1 := (minZ - a.Z) * invD
			t2 := (maxZ - a.Z) * invD
			if t1 > t2 {
				t1, t2 = t2, t1
			}
			if t1 > tMin {
				tMin = t1
			}
			if t2 < tMax {
				tMax = t2
			}
			if tMin > tMax {
				continue
			}
		}

		// Segment intersects this obstacle
		return true
	}
	return false
}

// CheckHitscan tests if a ray from origin in direction hits a sphere at target
// with the given radius, within maxRange. Returns false if an obstacle blocks LOS.
func CheckHitscan(origin, direction entity.Vec3, target entity.Vec3, targetRadius, maxRange float32, obstacles []Obstacle) bool {
	dir := direction.Normalized()
	toTarget := target.Sub(origin)
	projection := toTarget.Dot(dir)
	if projection < 0 || projection > maxRange {
		return false
	}
	closestPoint := origin.Add(dir.Scale(projection))
	distSq := closestPoint.DistanceToSq(target)
	if distSq > targetRadius*targetRadius {
		return false
	}

	// Check if any obstacle blocks the line of sight
	return !SegmentHitsObstacle(origin, target, obstacles)
}

// CheckMeleeArc tests if a target is within melee range and arc of the attacker.
// Returns false if an obstacle blocks LOS.
func CheckMeleeArc(attackerPos, attackerForward, targetPos entity.Vec3, meleeRange, arcDegrees float32, obstacles []Obstacle) bool {
	toTarget := targetPos.Sub(attackerPos).Flat()
	dist := toTarget.Length()
	if dist > meleeRange {
		return false
	}
	if dist < 0.01 {
		return true
	}
	forward := attackerForward.Flat().Normalized()
	targetDir := toTarget.Normalized()
	angle := forward.AngleTo(targetDir)
	halfArc := arcDegrees / 2.0 * (math.Pi / 180.0)
	if angle > float32(halfArc) {
		return false
	}

	// Check if any obstacle blocks the line of sight
	return !SegmentHitsObstacle(attackerPos, targetPos, obstacles)
}

// CheckAoERadius tests if a target position is within radius of center.
func CheckAoERadius(center, targetPos entity.Vec3, radius float32, obstacles []Obstacle) bool {
	if center.DistanceToSq(targetPos) > radius*radius {
		return false
	}
	return !SegmentHitsObstacle(center, targetPos, obstacles)
}

// CheckProjectileHit tests if a projectile at projPos hits a target at targetPos.
func CheckProjectileHit(projPos, targetPos entity.Vec3, hitRadius float32) bool {
	// Use flat distance (Y tolerance of 2m for jumping players)
	dx := projPos.X - targetPos.X
	dz := projPos.Z - targetPos.Z
	flatDistSq := dx*dx + dz*dz
	dy := projPos.Y - (targetPos.Y + 1.0) // target center mass
	if dy > 2.0 || dy < -2.0 {
		return false
	}
	return flatDistSq <= hitRadius*hitRadius
}

// ProjectileHitsObstacle checks if a projectile at pos overlaps any obstacle.
// Respects obstacle height — projectiles above a short obstacle pass over it.
func ProjectileHitsObstacle(pos entity.Vec3, radius float32, obstacles []Obstacle) bool {
	for _, obs := range obstacles {
		// Skip if projectile is above this obstacle
		if obs.Height > 0 && pos.Y > obs.Height {
			continue
		}
		// Expand obstacle by projectile radius (Minkowski sum)
		exHx := obs.HX + radius
		exHz := obs.HZ + radius
		dx := pos.X - obs.CX
		dz := pos.Z - obs.CZ
		if dx > -exHx && dx < exHx && dz > -exHz && dz < exHz {
			return true
		}
	}
	return false
}

// SegmentHitsExpandedObstacle is like SegmentHitsObstacle but expands each
// obstacle by the given radius before testing. Use this for AI line-of-sight
// checks where the entity has a body radius that must clear the obstacle.
// Unlike SegmentHitsObstacle, it does NOT skip obstacles containing the origin.
func SegmentHitsExpandedObstacle(a, b entity.Vec3, obstacles []Obstacle, radius float32) bool {
	dx := b.X - a.X
	dz := b.Z - a.Z
	length := float32(math.Sqrt(float64(dx*dx + dz*dz)))
	if length < 1e-6 {
		return false
	}

	for _, obs := range obstacles {
		minX := obs.CX - obs.HX - radius
		maxX := obs.CX + obs.HX + radius
		minZ := obs.CZ - obs.HZ - radius
		maxZ := obs.CZ + obs.HZ + radius

		var tMin, tMax float32 = 0, 1

		// X slab
		if abs32(dx) < 1e-6 {
			if a.X < minX || a.X > maxX {
				continue
			}
		} else {
			invD := 1.0 / dx
			t1 := (minX - a.X) * invD
			t2 := (maxX - a.X) * invD
			if t1 > t2 {
				t1, t2 = t2, t1
			}
			if t1 > tMin {
				tMin = t1
			}
			if t2 < tMax {
				tMax = t2
			}
			if tMin > tMax {
				continue
			}
		}

		// Z slab
		if abs32(dz) < 1e-6 {
			if a.Z < minZ || a.Z > maxZ {
				continue
			}
		} else {
			invD := 1.0 / dz
			t1 := (minZ - a.Z) * invD
			t2 := (maxZ - a.Z) * invD
			if t1 > t2 {
				t1, t2 = t2, t1
			}
			if t1 > tMin {
				tMin = t1
			}
			if t2 < tMax {
				tMax = t2
			}
			if tMin > tMax {
				continue
			}
		}

		return true
	}
	return false
}

// NearestObstacleOnSegment returns the center of the first obstacle that
// blocks the segment from a to b (expanded by radius). Returns false if clear.
func NearestObstacleOnSegment(a, b entity.Vec3, obstacles []Obstacle, radius float32) (Obstacle, bool) {
	dx := b.X - a.X
	dz := b.Z - a.Z
	length := float32(math.Sqrt(float64(dx*dx + dz*dz)))
	if length < 1e-6 {
		return Obstacle{}, false
	}

	bestT := float32(2.0)
	bestObs := Obstacle{}
	found := false

	for _, obs := range obstacles {
		minX := obs.CX - obs.HX - radius
		maxX := obs.CX + obs.HX + radius
		minZ := obs.CZ - obs.HZ - radius
		maxZ := obs.CZ + obs.HZ + radius

		var tMin, tMax float32 = 0, 1

		if abs32(dx) < 1e-6 {
			if a.X < minX || a.X > maxX {
				continue
			}
		} else {
			invD := 1.0 / dx
			t1 := (minX - a.X) * invD
			t2 := (maxX - a.X) * invD
			if t1 > t2 {
				t1, t2 = t2, t1
			}
			if t1 > tMin {
				tMin = t1
			}
			if t2 < tMax {
				tMax = t2
			}
			if tMin > tMax {
				continue
			}
		}

		if abs32(dz) < 1e-6 {
			if a.Z < minZ || a.Z > maxZ {
				continue
			}
		} else {
			invD := 1.0 / dz
			t1 := (minZ - a.Z) * invD
			t2 := (maxZ - a.Z) * invD
			if t1 > t2 {
				t1, t2 = t2, t1
			}
			if t1 > tMin {
				tMin = t1
			}
			if t2 < tMax {
				tMax = t2
			}
			if tMin > tMax {
				continue
			}
		}

		if tMin < bestT {
			bestT = tMin
			bestObs = obs
			found = true
		}
	}
	return bestObs, found
}

func abs32(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}
