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

// ResolveAoEAtPosition checks an AoE centered on an arbitrary world position.
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
