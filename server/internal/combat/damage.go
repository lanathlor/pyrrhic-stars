package combat

import "codex-online/server/internal/entity"

// DamageEvent is emitted when damage is dealt, for broadcasting to clients.
type DamageEvent struct {
	TargetPeerID uint16
	SourcePeerID uint16 // peer ID of the attacker (0 for enemy sources)
	Amount       float32
	HitPos       entity.Vec3
	SourceType   uint8 // 0=player_attack, 1=enemy_melee, 2=enemy_ranged, 3=enemy_aoe, 4=enemy_charge
}

// Source type constants.
const (
	SourcePlayerAttack uint8 = 0
	SourceEnemyMelee   uint8 = 1
	SourceEnemyRanged  uint8 = 2
	SourceEnemyAoE     uint8 = 3
	SourceEnemyCharge  uint8 = 4
)

// ResolvePlayerAttackOnEnemy checks if a player's attack hits a single enemy and applies damage.
func ResolvePlayerAttackOnEnemy(player *entity.Player, enemy *entity.Enemy, obstacles []Obstacle) *DamageEvent {
	if !enemy.Alive || enemy.State == entity.EnemyDead {
		return nil
	}

	switch player.ClassName {
	case "gunner":
		return resolveGunnerShot(player, enemy, obstacles)
	case "vanguard":
		return resolveVanguardMelee(player, enemy, obstacles)
	case "blade_dancer":
		return resolveBladeDancerAttack(player, enemy, obstacles)
	}
	return nil
}

// ResolvePlayerAttackOnEnemies checks if a player's attack hits any enemy.
// Returns the damage event and the enemy that was hit.
func ResolvePlayerAttackOnEnemies(player *entity.Player, enemies []*entity.Enemy, obstacles []Obstacle) (*DamageEvent, *entity.Enemy) {
	// For hitscan classes (gunner, blade_dancer), check each enemy and hit the first valid one.
	// For melee classes (vanguard), check each enemy and hit the nearest in range.
	var bestEvt *DamageEvent
	var bestEnemy *entity.Enemy
	bestDistSq := float32(1e18)

	for _, e := range enemies {
		if e == nil || !e.Alive || e.State == entity.EnemyDead {
			continue
		}
		evt := ResolvePlayerAttackOnEnemy(player, e, obstacles)
		if evt == nil {
			continue
		}
		// For hitscan: return first hit (order doesn't matter much, but prefer closer)
		// For melee: prefer nearest
		distSq := e.Position.DistanceToSq(player.Position)
		if distSq < bestDistSq {
			bestDistSq = distSq
			bestEvt = evt
			bestEnemy = e
		}
	}
	return bestEvt, bestEnemy
}

// AoEShapeType identifies the geometry of an AoE check.
type AoEShapeType uint8

const (
	AoECircle AoEShapeType = 0
	AoECone   AoEShapeType = 1
)

// AoEShape describes an AoE attack geometry.
type AoEShape struct {
	Type       AoEShapeType
	Radius     float32
	ArcDegrees float32 // only for AoECone
	Damage     float32
}

// ResolvePlayerAoEOnEnemies checks a player's AoE attack against all enemies,
// returning damage events for every enemy hit (not just the first).
func ResolvePlayerAoEOnEnemies(player *entity.Player, enemies []*entity.Enemy, obstacles []Obstacle, shape AoEShape) []DamageEvent {
	var events []DamageEvent
	for _, e := range enemies {
		if e == nil || !e.Alive || e.State == entity.EnemyDead {
			continue
		}
		var hit bool
		switch shape.Type {
		case AoECircle:
			hit = CheckAoERadius(player.Position, e.Position, shape.Radius, obstacles)
		case AoECone:
			hit = CheckMeleeArc(player.Position, player.Forward(), e.Position, shape.Radius, shape.ArcDegrees, obstacles)
		}
		if !hit {
			continue
		}
		dealt, _ := e.ApplyDamage(shape.Damage)
		if dealt == 0 {
			continue
		}
		hitDir := e.Position.Sub(player.Position)
		if hitDir.LengthSq() > 0.01 {
			hitDir = hitDir.Normalized()
		}
		events = append(events, DamageEvent{
			TargetPeerID: e.ID,
			SourcePeerID: player.PeerID,
			Amount:       dealt,
			HitPos:       player.Position.Add(hitDir),
			SourceType:   SourcePlayerAttack,
		})
	}
	return events
}

// ResolveAoEAtPosition checks an AoE centered on an arbitrary world position
// (e.g. an enemy's location for target-centered spells).
func ResolveAoEAtPosition(center entity.Vec3, sourcePeerID uint16, enemies []*entity.Enemy, obstacles []Obstacle, shape AoEShape) []DamageEvent {
	var events []DamageEvent
	for _, e := range enemies {
		if e == nil || !e.Alive || e.State == entity.EnemyDead {
			continue
		}
		if !CheckAoERadius(center, e.Position, shape.Radius, obstacles) {
			continue
		}
		dealt, _ := e.ApplyDamage(shape.Damage)
		if dealt == 0 {
			continue
		}
		hitDir := e.Position.Sub(center)
		if hitDir.LengthSq() > 0.01 {
			hitDir = hitDir.Normalized()
		}
		events = append(events, DamageEvent{
			TargetPeerID: e.ID,
			SourcePeerID: sourcePeerID,
			Amount:       dealt,
			HitPos:       center.Add(hitDir),
			SourceType:   SourcePlayerAttack,
		})
	}
	return events
}

// ResolveNearestNEnemies hits the N nearest in-combat enemies by proximity.
// Only targets enemies that have threat entries (are in combat with someone).
// LoS is checked from the player to each target.
func ResolveNearestNEnemies(player *entity.Player, enemies []*entity.Enemy, obstacles []Obstacle, n int, damage float32) []DamageEvent {
	if n <= 0 {
		return nil
	}

	// Collect alive, in-combat enemies with distances
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
			continue // not in combat
		}
		distSq := e.Position.DistanceToSq(player.Position)
		candidates = append(candidates, candidate{e, distSq})
	}

	// Sort by distance (simple insertion sort — N is small)
	for i := 1; i < len(candidates); i++ {
		for j := i; j > 0 && candidates[j].distSq < candidates[j-1].distSq; j-- {
			candidates[j], candidates[j-1] = candidates[j-1], candidates[j]
		}
	}

	// Take nearest N with LoS check
	var events []DamageEvent
	hits := 0
	for _, c := range candidates {
		if hits >= n {
			break
		}
		if SegmentHitsObstacle(player.Position, c.enemy.Position, obstacles) {
			continue // blocked by obstacle
		}
		dealt, _ := c.enemy.ApplyDamage(damage)
		if dealt == 0 {
			continue
		}
		hitDir := c.enemy.Position.Sub(player.Position)
		if hitDir.LengthSq() > 0.01 {
			hitDir = hitDir.Normalized()
		}
		events = append(events, DamageEvent{
			TargetPeerID: c.enemy.ID,
			SourcePeerID: player.PeerID,
			Amount:       dealt,
			HitPos:       player.Position.Add(hitDir),
			SourceType:   SourcePlayerAttack,
		})
		hits++
	}
	return events
}

func resolveGunnerShot(player *entity.Player, enemy *entity.Enemy, obstacles []Obstacle) *DamageEvent {
	origin := player.EyePosition()
	direction := player.AimDirection()
	targetCenter := enemy.Position.Add(entity.Vec3{Y: 1.0})

	if !CheckHitscan(origin, direction, targetCenter, 1.2, 100.0, obstacles) {
		return nil
	}

	gunDamage := float32(10.0)
	if player.RechamberBuff {
		gunDamage = 18.0
	}
	dealt, _ := enemy.ApplyDamage(gunDamage)
	if dealt == 0 {
		return nil
	}
	return &DamageEvent{
		TargetPeerID: enemy.ID,
		Amount:       dealt,
		HitPos:       targetCenter,
		SourceType:   SourcePlayerAttack,
	}
}

func resolveVanguardMelee(player *entity.Player, enemy *entity.Enemy, obstacles []Obstacle) *DamageEvent {
	if !CheckMeleeArc(player.Position, player.Forward(), enemy.Position, entity.MeleeRange, 120.0, obstacles) {
		return nil
	}

	// Damage depends on combo step
	var damage float32
	switch player.ComboStep {
	case 0:
		damage = 30.0
	case 1:
		damage = 35.0
	case 2:
		damage = 55.0
	default:
		damage = 30.0
	}

	dealt, _ := enemy.ApplyDamage(damage)
	if dealt == 0 {
		return nil
	}
	hitDir := enemy.Position.Sub(player.Position).Normalized()
	return &DamageEvent{
		TargetPeerID: enemy.ID,
		Amount:       dealt,
		HitPos:       player.Position.Add(hitDir),
		SourceType:   SourcePlayerAttack,
	}
}

func resolveBladeDancerAttack(player *entity.Player, enemy *entity.Enemy, obstacles []Obstacle) *DamageEvent {
	origin := player.EyePosition()
	direction := player.AimDirection()
	targetCenter := enemy.Position.Add(entity.Vec3{Y: 1.0})

	if !CheckHitscan(origin, direction, targetCenter, 1.2, 20.0, obstacles) {
		return nil
	}

	// Damage depends on config and ability
	var damage float32
	if player.Config == 0 { // orbit
		damage = 25.0
	} else { // lance
		damage = 35.0
	}

	dealt, _ := enemy.ApplyDamage(damage)
	if dealt == 0 {
		return nil
	}
	return &DamageEvent{
		TargetPeerID: enemy.ID,
		Amount:       dealt,
		HitPos:       targetCenter,
		SourceType:   SourcePlayerAttack,
	}
}
