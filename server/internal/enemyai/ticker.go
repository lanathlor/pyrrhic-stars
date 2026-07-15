package enemyai

import (
	"codex-online/server/internal/bt"
	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
	"codex-online/server/internal/overflux"
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

	// SetBus attaches the zone coordination bus (see Bus).
	SetBus(bus *Bus)

	// SetSpawnAddFn attaches the zone's add-summoning callback.
	SetSpawnAddFn(fn func(defName string, pos entity.Vec3, ownerID uint16) bool)

	// SetAllies attaches the zone's enemy slice for add-awareness conditions.
	SetAllies(allies []*entity.Enemy)

	// Overflux variant support.
	ApplyOverfluxVariants(oflx *overflux.State)

	// Tree access (used by bosstest instrumentation).
	Tree() bt.Node
	SetTree(root bt.Node)

	// Debug / dev mode methods.
	ForceCommit(abilityID string) bool
	AbilityIDs() []string
	DefName() string
}
