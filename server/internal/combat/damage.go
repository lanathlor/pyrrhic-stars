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

// ResolvePlayerAttackOnEnemy checks if a player's attack hits the enemy and applies damage.
func ResolvePlayerAttackOnEnemy(player *entity.Player, enemy *entity.Enemy) *DamageEvent {
	if !enemy.Alive || enemy.State == entity.EnemyDead {
		return nil
	}

	switch player.ClassName {
	case "gunner":
		return resolveGunnerShot(player, enemy)
	case "vanguard":
		return resolveVanguardMelee(player, enemy)
	case "blade_dancer":
		return resolveBladeDancerAttack(player, enemy)
	}
	return nil
}

func resolveGunnerShot(player *entity.Player, enemy *entity.Enemy) *DamageEvent {
	origin := player.EyePosition()
	direction := player.AimDirection()
	targetCenter := enemy.Position.Add(entity.Vec3{Y: 1.0})

	if !CheckHitscan(origin, direction, targetCenter, 1.0, 100.0) {
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

func resolveVanguardMelee(player *entity.Player, enemy *entity.Enemy) *DamageEvent {
	if !CheckMeleeArc(player.Position, player.Forward(), enemy.Position, entity.MeleeRange, 120.0) {
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

func resolveBladeDancerAttack(player *entity.Player, enemy *entity.Enemy) *DamageEvent {
	origin := player.EyePosition()
	direction := player.AimDirection()
	targetCenter := enemy.Position.Add(entity.Vec3{Y: 1.0})

	if !CheckHitscan(origin, direction, targetCenter, 1.0, 20.0) {
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
