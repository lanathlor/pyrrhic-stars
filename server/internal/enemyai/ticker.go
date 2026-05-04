package enemyai

import (
	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
)

// BrainTicker is the interface satisfied by Brain. It decouples the AI system
// from the concrete Brain type.
type BrainTicker interface {
	Tick(dt float32, players []*entity.Player,
		obstacles []combat.Obstacle,
		spawnProjectile func(pos, dir entity.Vec3, speed, damage, lifetime float32),
		castPattern func(pattern *combat.PatternDef, abilityName string, origin, facing entity.Vec3),
	) []combat.DamageEvent
	Enemy() *entity.Enemy
}
