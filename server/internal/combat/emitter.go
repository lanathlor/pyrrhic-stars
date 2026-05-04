package combat

import (
	"math"
	"math/rand/v2"

	"codex-online/server/internal/entity"
)

// emitWave computes spawn requests for one wave of an emitter.
// origin is the pattern center, facing is the base direction.
// waveIdx is 0-indexed wave number.
func emitWave(emitter *EmitterDef, origin, facing entity.Vec3, waveIdx int, rng *rand.Rand) []SpawnRequest {
	switch emitter.Type {
	case EmitterRadial:
		return emitRadial(emitter, origin, waveIdx)
	case EmitterCone:
		return emitCone(emitter, origin, facing, waveIdx)
	case EmitterLine:
		return emitLine(emitter, origin, facing, waveIdx)
	case EmitterArc:
		return emitArc(emitter, origin, facing, waveIdx)
	case EmitterRingContract:
		return emitRingContract(emitter, origin, waveIdx)
	case EmitterTargeted:
		return emitTargeted(emitter, origin, facing, waveIdx)
	case EmitterRandomZone:
		return emitRandomZone(emitter, origin, rng)
	default:
		return nil
	}
}

// emitRadial spawns Count projectiles evenly around 360°.
// OffsetPerWave rotates each subsequent wave (spiral arms).
func emitRadial(e *EmitterDef, origin entity.Vec3, waveIdx int) []SpawnRequest {
	count := e.Count
	if count <= 0 {
		return nil
	}
	requests := make([]SpawnRequest, 0, count)
	step := 2 * math.Pi / float64(count)
	baseAngle := float64(e.StartAngle) + float64(e.OffsetPerWave)*float64(waveIdx)

	for i := range count {
		angle := baseAngle + step*float64(i)
		dir := angleToDir(float32(angle))
		requests = append(requests, SpawnRequest{
			Position:        origin,
			Direction:       dir,
			Speed:           e.Projectile.Speed,
			Damage:          e.Projectile.Damage,
			Lifetime:        e.Projectile.Lifetime,
			Acceleration:    e.Projectile.Acceleration,
			MaxSpeed:        e.Projectile.MaxSpeed,
			AngularVelocity: e.Projectile.AngularVelocity,
			Radius:          e.Projectile.Radius,
		})
	}
	return requests
}

// emitCone spawns Count projectiles within ArcAngle centered on facing.
func emitCone(e *EmitterDef, origin, facing entity.Vec3, waveIdx int) []SpawnRequest {
	count := e.Count
	if count <= 0 {
		return nil
	}
	requests := make([]SpawnRequest, 0, count)
	facingAngle := dirToAngle(facing)
	waveOffset := e.OffsetPerWave * float32(waveIdx)

	halfArc := e.ArcAngle / 2
	var step float32
	if count > 1 {
		step = e.ArcAngle / float32(count-1)
	}

	for i := range count {
		var offset float32
		if count == 1 {
			offset = 0
		} else {
			offset = -halfArc + step*float32(i)
		}
		angle := facingAngle + offset + e.StartAngle + waveOffset
		dir := angleToDir(angle)
		requests = append(requests, SpawnRequest{
			Position:        origin,
			Direction:       dir,
			Speed:           e.Projectile.Speed,
			Damage:          e.Projectile.Damage,
			Lifetime:        e.Projectile.Lifetime,
			Acceleration:    e.Projectile.Acceleration,
			MaxSpeed:        e.Projectile.MaxSpeed,
			AngularVelocity: e.Projectile.AngularVelocity,
			Radius:          e.Projectile.Radius,
		})
	}
	return requests
}

// emitLine spawns Count projectiles in a line perpendicular to facing.
// All travel in the facing direction.
func emitLine(e *EmitterDef, origin, facing entity.Vec3, waveIdx int) []SpawnRequest {
	count := e.Count
	if count <= 0 {
		return nil
	}
	requests := make([]SpawnRequest, 0, count)
	facingAngle := dirToAngle(facing)
	waveOffset := e.OffsetPerWave * float32(waveIdx)
	dir := angleToDir(facingAngle + waveOffset)

	// Perpendicular direction for positioning
	perpAngle := facingAngle + math.Pi/2
	perp := angleToDir(perpAngle)

	halfWidth := e.LineWidth / 2
	var step float32
	if count > 1 {
		step = e.LineWidth / float32(count-1)
	}

	for i := range count {
		var lateralOffset float32
		if count == 1 {
			lateralOffset = 0
		} else {
			lateralOffset = -halfWidth + step*float32(i)
		}
		pos := origin.Add(perp.Scale(lateralOffset))
		requests = append(requests, SpawnRequest{
			Position:        pos,
			Direction:       dir,
			Speed:           e.Projectile.Speed,
			Damage:          e.Projectile.Damage,
			Lifetime:        e.Projectile.Lifetime,
			Acceleration:    e.Projectile.Acceleration,
			MaxSpeed:        e.Projectile.MaxSpeed,
			AngularVelocity: e.Projectile.AngularVelocity,
			Radius:          e.Projectile.Radius,
		})
	}
	return requests
}

// emitArc spawns Count projectiles spanning ArcAngle from facing.
// Like radial but only partial.
func emitArc(e *EmitterDef, origin, facing entity.Vec3, waveIdx int) []SpawnRequest {
	// Arc is identical to cone in terms of direction computation.
	return emitCone(e, origin, facing, waveIdx)
}

// emitRingContract spawns a ring at StartRadius, all pointing inward.
func emitRingContract(e *EmitterDef, origin entity.Vec3, waveIdx int) []SpawnRequest {
	count := e.Count
	if count <= 0 {
		return nil
	}
	requests := make([]SpawnRequest, 0, count)
	step := 2 * math.Pi / float64(count)
	baseAngle := float64(e.StartAngle) + float64(e.OffsetPerWave)*float64(waveIdx)

	for i := range count {
		angle := baseAngle + step*float64(i)
		outDir := angleToDir(float32(angle))
		// Spawn at radius, point inward
		pos := origin.Add(outDir.Scale(e.StartRadius))
		inDir := entity.Vec3{X: -outDir.X, Y: outDir.Y, Z: -outDir.Z}
		requests = append(requests, SpawnRequest{
			Position:        pos,
			Direction:       inDir,
			Speed:           e.Projectile.Speed,
			Damage:          e.Projectile.Damage,
			Lifetime:        e.Projectile.Lifetime,
			Acceleration:    e.Projectile.Acceleration,
			MaxSpeed:        e.Projectile.MaxSpeed,
			AngularVelocity: e.Projectile.AngularVelocity,
			Radius:          e.Projectile.Radius,
		})
	}
	return requests
}

// emitTargeted spawns Count projectiles aimed at facing.
// If ArcAngle > 0, spreads them within that cone.
func emitTargeted(e *EmitterDef, origin, facing entity.Vec3, waveIdx int) []SpawnRequest {
	count := e.Count
	if count <= 0 {
		return nil
	}
	if e.ArcAngle > 0 && count > 1 {
		// Spread like a cone, but centered on target direction
		return emitCone(e, origin, facing, waveIdx)
	}
	// All projectiles fire in same direction
	requests := make([]SpawnRequest, 0, count)
	facingAngle := dirToAngle(facing)
	waveOffset := e.OffsetPerWave * float32(waveIdx)
	dir := angleToDir(facingAngle + e.StartAngle + waveOffset)

	for range count {
		requests = append(requests, SpawnRequest{
			Position:        origin,
			Direction:       dir,
			Speed:           e.Projectile.Speed,
			Damage:          e.Projectile.Damage,
			Lifetime:        e.Projectile.Lifetime,
			Acceleration:    e.Projectile.Acceleration,
			MaxSpeed:        e.Projectile.MaxSpeed,
			AngularVelocity: e.Projectile.AngularVelocity,
			Radius:          e.Projectile.Radius,
		})
	}
	return requests
}

// emitRandomZone spawns Count projectiles at random positions within ZoneRadius,
// with random outward directions.
func emitRandomZone(e *EmitterDef, origin entity.Vec3, rng *rand.Rand) []SpawnRequest {
	count := e.Count
	if count <= 0 {
		return nil
	}
	requests := make([]SpawnRequest, 0, count)
	for range count {
		angle := rng.Float32() * 2 * math.Pi
		dist := rng.Float32() * e.ZoneRadius
		pos := origin.Add(angleToDir(angle).Scale(dist))
		dir := angleToDir(rng.Float32() * 2 * math.Pi)
		requests = append(requests, SpawnRequest{
			Position:        pos,
			Direction:       dir,
			Speed:           e.Projectile.Speed,
			Damage:          e.Projectile.Damage,
			Lifetime:        e.Projectile.Lifetime,
			Acceleration:    e.Projectile.Acceleration,
			MaxSpeed:        e.Projectile.MaxSpeed,
			AngularVelocity: e.Projectile.AngularVelocity,
			Radius:          e.Projectile.Radius,
		})
	}
	return requests
}

// angleToDir converts a Y-axis rotation angle to an XZ direction vector.
func angleToDir(angle float32) entity.Vec3 {
	return entity.Vec3{
		X: float32(math.Sin(float64(angle))),
		Y: 0,
		Z: float32(math.Cos(float64(angle))),
	}
}

// dirToAngle converts an XZ direction to a Y-axis rotation angle.
func dirToAngle(dir entity.Vec3) float32 {
	return float32(math.Atan2(float64(dir.X), float64(dir.Z)))
}
