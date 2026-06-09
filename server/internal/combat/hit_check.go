package combat

import (
	"math"

	"codex-online/server/internal/entity"
)

// Obstacle represents a rectangular obstacle in the zone.
type Obstacle struct {
	CX, CZ float32 // center (XZ plane)
	HX, HZ float32 // half-extents (XZ plane)
	BaseY  float32 // Y of obstacle bottom face
	Height float32 // full height from BaseY (0 = infinitely tall)
}

// SegmentHitsObstacle checks if a line segment from a to b (on the XZ plane)
// intersects any obstacle. Uses slab intersection (ray-vs-AABB in 2D).
// Obstacles that contain point a (the origin) are skipped — you can shoot
// out of geometry you're standing in.
func SegmentHitsObstacle(a, b entity.Vec3, obstacles []Obstacle) bool { //nolint:gocognit
	dx := b.X - a.X
	dy := b.Y - a.Y
	dz := b.Z - a.Z
	length := float32(math.Sqrt(float64(dx*dx + dz*dz)))
	if length < 1e-6 {
		return false
	}

	for _, obs := range obstacles {
		minX := obs.CX - obs.HX
		maxX := obs.CX + obs.HX
		minZ := obs.CZ - obs.HZ
		maxZ := obs.CZ + obs.HZ

		// Skip obstacles that contain the origin or target point
		if a.X >= minX && a.X <= maxX && a.Z >= minZ && a.Z <= maxZ {
			continue
		}
		if b.X >= minX && b.X <= maxX && b.Z >= minZ && b.Z <= maxZ {
			continue
		}

		var tMin, tMax float32 = 0, 1
		var hit bool

		if tMin, tMax, hit = slabIntersect2D(a.X, dx, minX, maxX, tMin, tMax); !hit {
			continue
		}
		if tMin, tMax, hit = slabIntersect2D(a.Z, dz, minZ, maxZ, tMin, tMax); !hit {
			continue
		}

		// Height check: if obstacle has a finite height, the segment
		// only hits if it passes through the obstacle's vertical extent.
		if obs.Height > 0 {
			obsTop := obs.BaseY + obs.Height
			yAtEntry := a.Y + dy*tMin
			yAtExit := a.Y + dy*tMax
			if yAtEntry > obsTop && yAtExit > obsTop {
				continue // ray passes over the obstacle
			}
			if yAtEntry < obs.BaseY && yAtExit < obs.BaseY {
				continue // ray passes under the obstacle
			}
		}

		return true
	}
	return false
}

// CheckHitscan tests if a ray from origin in direction hits a sphere at target
// with the given radius, within maxRange. Returns false if an obstacle blocks LOS.
// CheckHitscan tests if a ray from origin in direction hits a vertical cylinder
// centered on target with the given radius and height 2.5m (feet to head).
// The cylinder extends from target.Y-1.0 to target.Y+1.5.
func CheckHitscan(origin, direction entity.Vec3, target entity.Vec3, targetRadius, maxRange float32, obstacles []Obstacle) bool {
	dir := direction.Normalized()

	// Find closest approach on XZ plane (cylinder test, ignoring Y)
	toTarget2D := entity.Vec3{X: target.X - origin.X, Z: target.Z - origin.Z}
	dir2D := entity.Vec3{X: dir.X, Z: dir.Z}
	dir2DLen := dir2D.Length()
	if dir2DLen < 1e-6 {
		// Shooting straight up/down — check XZ distance directly
		dxz := toTarget2D.Length()
		return dxz <= targetRadius
	}
	dir2DN := entity.Vec3{X: dir2D.X / dir2DLen, Z: dir2D.Z / dir2DLen}

	proj := toTarget2D.X*dir2DN.X + toTarget2D.Z*dir2DN.Z
	if proj < 0 {
		return false
	}

	// 3D range check
	toTarget3D := target.Sub(origin)
	proj3D := toTarget3D.Dot(dir)
	if proj3D > maxRange {
		return false
	}

	// Perpendicular XZ distance to the ray
	closestX := origin.X + dir2DN.X*proj
	closestZ := origin.Z + dir2DN.Z*proj
	dxSq := (closestX - target.X) * (closestX - target.X)
	dzSq := (closestZ - target.Z) * (closestZ - target.Z)
	if dxSq+dzSq > targetRadius*targetRadius {
		return false
	}

	// Y check: ray must pass through the cylinder height [target.Y-1, target.Y+1.5]
	t := proj / dir2DLen
	hitY := origin.Y + dir.Y*t
	if hitY < target.Y-1.0 || hitY > target.Y+1.5 {
		return false
	}

	// LoS check along the actual bullet path to the hit point
	hitPoint := entity.Vec3{X: closestX, Y: hitY, Z: closestZ}
	return !SegmentHitsObstacle(origin, hitPoint, obstacles)
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
		// Skip if projectile is above or below this obstacle
		if obs.Height > 0 {
			if pos.Y > obs.BaseY+obs.Height || pos.Y < obs.BaseY {
				continue
			}
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
	dy := b.Y - a.Y
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
		var hit bool

		if tMin, tMax, hit = slabIntersect2D(a.X, dx, minX, maxX, tMin, tMax); !hit {
			continue
		}
		if tMin, tMax, hit = slabIntersect2D(a.Z, dz, minZ, maxZ, tMin, tMax); !hit {
			continue
		}

		// Height check: skip obstacles on different floors
		if obs.Height > 0 {
			obsTop := obs.BaseY + obs.Height
			yAtEntry := a.Y + dy*tMin
			yAtExit := a.Y + dy*tMax
			if yAtEntry > obsTop && yAtExit > obsTop {
				continue
			}
			if yAtEntry < obs.BaseY && yAtExit < obs.BaseY {
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
	dy := b.Y - a.Y
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
		var hit bool

		if tMin, tMax, hit = slabIntersect2D(a.X, dx, minX, maxX, tMin, tMax); !hit {
			continue
		}
		if tMin, tMax, hit = slabIntersect2D(a.Z, dz, minZ, maxZ, tMin, tMax); !hit {
			continue
		}

		// Height check: skip obstacles on different floors
		if obs.Height > 0 {
			obsTop := obs.BaseY + obs.Height
			yAtEntry := a.Y + dy*tMin
			yAtExit := a.Y + dy*tMax
			if yAtEntry > obsTop && yAtExit > obsTop {
				continue
			}
			if yAtEntry < obs.BaseY && yAtExit < obs.BaseY {
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

// slabIntersect2D performs a 2D AABB slab intersection test along one axis.
// Returns the updated tMin/tMax and false if the segment misses the slab.
// ax is the ray origin component, dx is the ray direction component,
// minV/maxV are the slab bounds.
func slabIntersect2D(ax, dx, minV, maxV, tMin, tMax float32) (float32, float32, bool) {
	if abs32(dx) < 1e-6 {
		if ax < minV || ax > maxV {
			return tMin, tMax, false
		}
		return tMin, tMax, true
	}
	invD := 1.0 / dx
	t1 := (minV - ax) * invD
	t2 := (maxV - ax) * invD
	if t1 > t2 {
		t1, t2 = t2, t1
	}
	if t1 > tMin {
		tMin = t1
	}
	if t2 < tMax {
		tMax = t2
	}
	return tMin, tMax, tMin <= tMax
}
