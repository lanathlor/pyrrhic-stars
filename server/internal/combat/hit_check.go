package combat

import (
	"math"

	"codex-online/server/internal/entity"
)

// CheckHitscan tests if a ray from origin in direction hits a sphere at target
// with the given radius, within maxRange.
func CheckHitscan(origin, direction entity.Vec3, target entity.Vec3, targetRadius, maxRange float32) bool {
	dir := direction.Normalized()
	toTarget := target.Sub(origin)
	projection := toTarget.Dot(dir)
	if projection < 0 || projection > maxRange {
		return false
	}
	closestPoint := origin.Add(dir.Scale(projection))
	distSq := closestPoint.DistanceToSq(target)
	return distSq <= targetRadius*targetRadius
}

// CheckMeleeArc tests if a target is within melee range and arc of the attacker.
func CheckMeleeArc(attackerPos, attackerForward, targetPos entity.Vec3, meleeRange, arcDegrees float32) bool {
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
	return angle <= float32(halfArc)
}

// CheckAoERadius tests if a target position is within radius of center.
func CheckAoERadius(center, targetPos entity.Vec3, radius float32) bool {
	return center.DistanceToSq(targetPos) <= radius*radius
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
