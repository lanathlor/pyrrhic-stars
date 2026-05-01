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
	Enemy      *entity.Enemy // for threat tracking
}

// resolveHit executes the hit portion of an ability and appends damage results to dst.
func resolveHit(dst []DamageResult, def *AbilityDef, p *entity.Player, enemies []*entity.Enemy, obstacles []combat.Obstacle) []DamageResult {
	damage := def.BaseDamage * p.DamageMult()

	switch def.Hit.Type {
	case HitNone:
		return dst

	case HitHitscan:
		return resolveHitscan(dst, p, enemies, obstacles, damage)

	case HitMeleeArc:
		return resolveMeleeArc(dst, p, enemies, obstacles, def.Hit, damage)

	case HitAoECircle:
		return resolveAoECircle(dst, p.Position, p.PeerID, enemies, obstacles, def.Hit.Radius, damage)

	case HitAoECone:
		return resolveAoECone(dst, p, enemies, obstacles, def.Hit, damage)

	case HitAoECircleTarget:
		return resolveAoECircleTarget(dst, p, enemies, obstacles, def.Hit.Radius, damage)

	case HitNearestN:
		return resolveNearestN(dst, p, enemies, obstacles, def.Hit.TargetCount, damage)
	}
	return dst
}

func resolveHitscan(dst []DamageResult, p *entity.Player, enemies []*entity.Enemy, obstacles []combat.Obstacle, damage float32) []DamageResult {
	origin := p.EyePosition()
	direction := p.AimDirection()

	var best *entity.Enemy
	bestDistSq := float32(1e18)

	for _, e := range enemies {
		if e == nil || !e.Alive || e.State == entity.EnemyDead {
			continue
		}
		targetCenter := e.Position.Add(entity.Vec3{Y: 1.0})
		if !combat.CheckHitscan(origin, direction, targetCenter, 1.2, 100.0, obstacles) {
			continue
		}
		distSq := e.Position.DistanceToSq(p.Position)
		if distSq < bestDistSq {
			bestDistSq = distSq
			best = e
		}
	}
	if best == nil {
		return dst
	}

	dealt, _ := best.ApplyDamage(damage)
	if dealt == 0 {
		return dst
	}
	return append(dst, DamageResult{
		TargetID:   best.ID,
		SourceID:   p.PeerID,
		Amount:     dealt,
		HitPos:     best.Position.Add(entity.Vec3{Y: 1.0}),
		SourceType: combat.SourcePlayerAttack,
		Enemy:      best,
	})
}

func resolveMeleeArc(dst []DamageResult, p *entity.Player, enemies []*entity.Enemy, obstacles []combat.Obstacle, hit HitDef, damage float32) []DamageResult {
	var best DamageResult
	bestDistSq := float32(1e18)
	found := false

	for _, e := range enemies {
		if e == nil || !e.Alive || e.State == entity.EnemyDead {
			continue
		}
		if !combat.CheckMeleeArc(p.Position, p.Forward(), e.Position, hit.Range, hit.ArcDegrees, obstacles) {
			continue
		}
		dealt, _ := e.ApplyDamage(damage)
		if dealt == 0 {
			continue
		}
		distSq := e.Position.DistanceToSq(p.Position)
		hitDir := e.Position.Sub(p.Position).Normalized()
		if distSq < bestDistSq {
			bestDistSq = distSq
			found = true
			best = DamageResult{
				TargetID:   e.ID,
				SourceID:   p.PeerID,
				Amount:     dealt,
				HitPos:     p.Position.Add(hitDir),
				SourceType: combat.SourcePlayerAttack,
				Enemy:      e,
			}
		}
	}
	if !found {
		return dst
	}
	return append(dst, best)
}

func resolveAoECircle(dst []DamageResult, center entity.Vec3, sourceID uint16, enemies []*entity.Enemy, obstacles []combat.Obstacle, radius, damage float32) []DamageResult {
	for _, e := range enemies {
		if e == nil || !e.Alive || e.State == entity.EnemyDead {
			continue
		}
		if !combat.CheckAoERadius(center, e.Position, radius, obstacles) {
			continue
		}
		dealt, _ := e.ApplyDamage(damage)
		if dealt == 0 {
			continue
		}
		hitDir := e.Position.Sub(center)
		if hitDir.LengthSq() > 0.01 {
			hitDir = hitDir.Normalized()
		}
		dst = append(dst, DamageResult{
			TargetID:   e.ID,
			SourceID:   sourceID,
			Amount:     dealt,
			HitPos:     center.Add(hitDir),
			SourceType: combat.SourcePlayerAttack,
			Enemy:      e,
		})
	}
	return dst
}

func resolveAoECone(dst []DamageResult, p *entity.Player, enemies []*entity.Enemy, obstacles []combat.Obstacle, hit HitDef, damage float32) []DamageResult {
	for _, e := range enemies {
		if e == nil || !e.Alive || e.State == entity.EnemyDead {
			continue
		}
		if !combat.CheckMeleeArc(p.Position, p.Forward(), e.Position, hit.Range, hit.ArcDegrees, obstacles) {
			continue
		}
		dealt, _ := e.ApplyDamage(damage)
		if dealt == 0 {
			continue
		}
		hitDir := e.Position.Sub(p.Position)
		if hitDir.LengthSq() > 0.01 {
			hitDir = hitDir.Normalized()
		}
		dst = append(dst, DamageResult{
			TargetID:   e.ID,
			SourceID:   p.PeerID,
			Amount:     dealt,
			HitPos:     p.Position.Add(hitDir),
			SourceType: combat.SourcePlayerAttack,
			Enemy:      e,
		})
	}
	return dst
}

func resolveAoECircleTarget(dst []DamageResult, p *entity.Player, enemies []*entity.Enemy, obstacles []combat.Obstacle, radius, damage float32) []DamageResult {
	// Find hitscan target first, then AoE around it
	target := findHitscanTarget(p, enemies, obstacles)
	if target == nil {
		return dst
	}
	return resolveAoECircle(dst, target.Position, p.PeerID, enemies, obstacles, radius, damage)
}

func resolveNearestN(dst []DamageResult, p *entity.Player, enemies []*entity.Enemy, obstacles []combat.Obstacle, n int, damage float32) []DamageResult {
	if n <= 0 {
		return dst
	}

	type candidate struct {
		enemy  *entity.Enemy
		distSq float32
	}
	var candidates []candidate
	for _, e := range enemies {
		if e == nil || !e.Alive || e.State == entity.EnemyDead {
			continue
		}
		if len(e.ThreatTable) == 0 {
			continue
		}
		distSq := e.Position.DistanceToSq(p.Position)
		candidates = append(candidates, candidate{e, distSq})
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
		if combat.SegmentHitsObstacle(p.Position, c.enemy.Position, obstacles) {
			continue
		}
		dealt, _ := c.enemy.ApplyDamage(damage)
		if dealt == 0 {
			continue
		}
		hitDir := c.enemy.Position.Sub(p.Position)
		if hitDir.LengthSq() > 0.01 {
			hitDir = hitDir.Normalized()
		}
		dst = append(dst, DamageResult{
			TargetID:   c.enemy.ID,
			SourceID:   p.PeerID,
			Amount:     dealt,
			HitPos:     p.Position.Add(hitDir),
			SourceType: combat.SourcePlayerAttack,
			Enemy:      c.enemy,
		})
		hits++
	}
	return dst
}

// findHitscanTarget finds the nearest enemy hit by the player's hitscan aim.
func findHitscanTarget(p *entity.Player, enemies []*entity.Enemy, obstacles []combat.Obstacle) *entity.Enemy {
	origin := p.EyePosition()
	direction := p.AimDirection()
	var best *entity.Enemy
	bestDistSq := float32(1e18)
	for _, e := range enemies {
		if e == nil || !e.Alive || e.State == entity.EnemyDead {
			continue
		}
		targetCenter := e.Position.Add(entity.Vec3{Y: 1.0})
		if !combat.CheckHitscan(origin, direction, targetCenter, 1.2, 20.0, obstacles) {
			continue
		}
		distSq := e.Position.DistanceToSq(p.Position)
		if distSq < bestDistSq {
			bestDistSq = distSq
			best = e
		}
	}
	return best
}
