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

func resolveGunnerShot(player *entity.Player, enemy *entity.Enemy, obstacles []Obstacle) *DamageEvent {
	origin := player.EyePosition()
	direction := player.AimDirection()
	targetCenter := enemy.Position.Add(entity.Vec3{Y: 1.0})

	if !CheckHitscan(origin, direction, targetCenter, 1.2, 100.0, obstacles) {
		return nil
	}

	const gunDamage float32 = 10.0
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
