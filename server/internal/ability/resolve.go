package ability

import (
	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
)

// DamageResult is emitted by ability resolution when damage is dealt.
type DamageResult struct {
	TargetID   uint16
	SourceID   uint16
	Amount     float32
	HitPos     entity.Vec3
	SourceType uint8
	AbilityID  string
	Target     entity.Target // the hit entity (caller type-asserts for threat/aggro)
}

// resolveHit executes the hit portion of an ability and appends damage results to dst.
func resolveHit(dst []DamageResult, def *AbilityDef, caster entity.Caster, targets []entity.Target, obstacles []combat.Obstacle, sourceType uint8) []DamageResult {
	damage := def.BaseDamage * caster.CasterDamageMult()

	switch def.Hit.Type {
	case HitNone:
		return dst

	case HitHitscan:
		return resolveHitscan(dst, caster, targets, obstacles, damage, sourceType)

	case HitMeleeArc:
		return resolveMeleeArc(dst, caster, targets, obstacles, def.Hit, damage, sourceType)

	case HitAoECircle:
		return resolveAoECircle(dst, caster.CasterPos(), caster.CasterID(), targets, obstacles, def.Hit.Radius, damage, sourceType)

	case HitAoECone:
		return resolveAoECone(dst, caster, targets, obstacles, def.Hit, damage, sourceType)

	case HitAoECircleTarget:
		return resolveAoECircleTarget(dst, caster, targets, obstacles, def.Hit.Radius, damage, sourceType)

	case HitNearestN:
		return resolveNearestN(dst, caster, targets, obstacles, def.Hit.TargetCount, damage, sourceType)
	}
	return dst
}

func resolveHitscan(dst []DamageResult, caster entity.Caster, targets []entity.Target, obstacles []combat.Obstacle, damage float32, sourceType uint8) []DamageResult {
	return resolveHitscanDir(dst, caster.CasterEyePos(), caster.CasterAimDir(), targets, obstacles, damage, sourceType, caster.CasterID())
}

// resolveHitscanDir fires a hitscan ray from origin in direction and returns
// damage results for the nearest hit. Used by the gunner assault handler to
// inject stability-cone-offset directions.
func resolveHitscanDir(dst []DamageResult, origin, direction entity.Vec3, targets []entity.Target, obstacles []combat.Obstacle, damage float32, sourceType uint8, sourceID uint16) []DamageResult {
	var best entity.Target
	bestDistSq := float32(1e18)

	for _, t := range targets {
		if t == nil || !t.TargetAlive() {
			continue
		}
		targetCenter := t.TargetPos().Add(entity.Vec3{Y: 1.0})
		if !combat.CheckHitscan(origin, direction, targetCenter, 1.2, 100.0, obstacles) {
			continue
		}
		distSq := t.TargetPos().Sub(origin).LengthSq()
		if distSq < bestDistSq {
			bestDistSq = distSq
			best = t
		}
	}
	if best == nil {
		return dst
	}

	dealt := best.TargetApplyDamage(damage)
	if dealt == 0 {
		return dst
	}
	return append(dst, DamageResult{
		TargetID:   best.TargetID(),
		SourceID:   sourceID,
		Amount:     dealt,
		HitPos:     best.TargetPos().Add(entity.Vec3{Y: 1.0}),
		SourceType: sourceType,
		Target:     best,
	})
}

func resolveMeleeArc(dst []DamageResult, caster entity.Caster, targets []entity.Target, obstacles []combat.Obstacle, hit HitDef, damage float32, sourceType uint8) []DamageResult {
	var best DamageResult
	bestDistSq := float32(1e18)
	found := false

	for _, t := range targets {
		if t == nil || !t.TargetAlive() {
			continue
		}
		if !combat.CheckMeleeArc(caster.CasterPos(), caster.CasterForward(), t.TargetPos(), hit.Range, hit.ArcDegrees, obstacles) {
			continue
		}
		dealt := t.TargetApplyDamage(damage)
		if dealt == 0 {
			continue
		}
		distSq := t.TargetPos().DistanceToSq(caster.CasterPos())
		hitDir := t.TargetPos().Sub(caster.CasterPos()).Normalized()
		if distSq < bestDistSq {
			bestDistSq = distSq
			found = true
			best = DamageResult{
				TargetID:   t.TargetID(),
				SourceID:   caster.CasterID(),
				Amount:     dealt,
				HitPos:     caster.CasterPos().Add(hitDir),
				SourceType: sourceType,
				Target:     t,
			}
		}
	}
	if !found {
		return dst
	}
	return append(dst, best)
}

func resolveAoECircle(dst []DamageResult, center entity.Vec3, sourceID uint16, targets []entity.Target, obstacles []combat.Obstacle, radius, damage float32, sourceType uint8) []DamageResult {
	for _, t := range targets {
		if t == nil || !t.TargetAlive() {
			continue
		}
		if !combat.CheckAoERadius(center, t.TargetPos(), radius, obstacles) {
			continue
		}
		dealt := t.TargetApplyDamage(damage)
		if dealt == 0 {
			continue
		}
		hitDir := t.TargetPos().Sub(center)
		if hitDir.LengthSq() > 0.01 {
			hitDir = hitDir.Normalized()
		}
		dst = append(dst, DamageResult{
			TargetID:   t.TargetID(),
			SourceID:   sourceID,
			Amount:     dealt,
			HitPos:     center.Add(hitDir),
			SourceType: sourceType,
			Target:     t,
		})
	}
	return dst
}

func resolveAoECone(dst []DamageResult, caster entity.Caster, targets []entity.Target, obstacles []combat.Obstacle, hit HitDef, damage float32, sourceType uint8) []DamageResult {
	for _, t := range targets {
		if t == nil || !t.TargetAlive() {
			continue
		}
		if !combat.CheckMeleeArc(caster.CasterPos(), caster.CasterForward(), t.TargetPos(), hit.Range, hit.ArcDegrees, obstacles) {
			continue
		}
		dealt := t.TargetApplyDamage(damage)
		if dealt == 0 {
			continue
		}
		hitDir := t.TargetPos().Sub(caster.CasterPos())
		if hitDir.LengthSq() > 0.01 {
			hitDir = hitDir.Normalized()
		}
		dst = append(dst, DamageResult{
			TargetID:   t.TargetID(),
			SourceID:   caster.CasterID(),
			Amount:     dealt,
			HitPos:     caster.CasterPos().Add(hitDir),
			SourceType: sourceType,
			Target:     t,
		})
	}
	return dst
}

func resolveAoECircleTarget(dst []DamageResult, caster entity.Caster, targets []entity.Target, obstacles []combat.Obstacle, radius, damage float32, sourceType uint8) []DamageResult {
	// Find hitscan target first, then AoE around it
	target := findHitscanTarget(caster, targets, obstacles)
	if target == nil {
		return dst
	}
	return resolveAoECircle(dst, target.TargetPos(), caster.CasterID(), targets, obstacles, radius, damage, sourceType)
}

func resolveNearestN(dst []DamageResult, caster entity.Caster, targets []entity.Target, obstacles []combat.Obstacle, n int, damage float32, sourceType uint8) []DamageResult {
	if n <= 0 {
		return dst
	}

	type candidate struct {
		target entity.Target
		distSq float32
	}
	var buf [16]candidate
	candidates := buf[:0]
	for _, t := range targets {
		if t == nil || !t.TargetAlive() {
			continue
		}
		// For enemy targets, skip those not yet engaged (no threat)
		if enemy, ok := t.(*entity.Enemy); ok && len(enemy.ThreatTable) == 0 {
			continue
		}
		distSq := t.TargetPos().DistanceToSq(caster.CasterPos())
		candidates = append(candidates, candidate{t, distSq})
	}

	// Sort by distance (insertion sort — N is small)
	for i := 1; i < len(candidates); i++ {
		for j := i; j > 0 && candidates[j].distSq < candidates[j-1].distSq; j-- {
			candidates[j], candidates[j-1] = candidates[j-1], candidates[j]
		}
	}

	hits := 0
	for _, c := range candidates {
		if hits >= n {
			break
		}
		if combat.SegmentHitsObstacle(caster.CasterPos(), c.target.TargetPos(), obstacles) {
			continue
		}
		dealt := c.target.TargetApplyDamage(damage)
		if dealt == 0 {
			continue
		}
		hitDir := c.target.TargetPos().Sub(caster.CasterPos())
		if hitDir.LengthSq() > 0.01 {
			hitDir = hitDir.Normalized()
		}
		dst = append(dst, DamageResult{
			TargetID:   c.target.TargetID(),
			SourceID:   caster.CasterID(),
			Amount:     dealt,
			HitPos:     caster.CasterPos().Add(hitDir),
			SourceType: sourceType,
			Target:     c.target,
		})
		hits++
	}
	return dst
}

// findHitscanTarget finds the nearest target hit by the caster's hitscan aim.
func findHitscanTarget(caster entity.Caster, targets []entity.Target, obstacles []combat.Obstacle) entity.Target {
	origin := caster.CasterEyePos()
	direction := caster.CasterAimDir()
	var best entity.Target
	bestDistSq := float32(1e18)
	for _, t := range targets {
		if t == nil || !t.TargetAlive() {
			continue
		}
		targetCenter := t.TargetPos().Add(entity.Vec3{Y: 1.0})
		if !combat.CheckHitscan(origin, direction, targetCenter, 1.2, 20.0, obstacles) {
			continue
		}
		distSq := t.TargetPos().DistanceToSq(caster.CasterPos())
		if distSq < bestDistSq {
			bestDistSq = distSq
			best = t
		}
	}
	return best
}
