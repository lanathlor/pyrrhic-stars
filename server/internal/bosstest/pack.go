package bosstest

import (
	"codex-online/server/internal/combat"
	"codex-online/server/internal/enemyai"
	"codex-online/server/internal/entity"
)

// packGroupID is the shared GroupID assigned to every enemy in a trash pack so
// that the real CombatSystem.checkEnemyGroupDead finalizes the encounter when
// the whole pack dies (it keys the combat log by GroupID, not the boss's -1).
const packGroupID = 1

// packFormation returns spawn positions for n pack enemies, spread along X near
// the arena centre so they don't stack on spawn. Enemies are forced into chase
// immediately, so exact placement only needs to avoid overlap.
func packFormation(n int) []entity.Vec3 {
	positions := make([]entity.Vec3, n)
	for i := range positions {
		x := (float32(i) - float32(n-1)/2) * 2.5
		positions[i] = entity.Vec3{X: x, Y: 0.1, Z: 2}
	}
	return positions
}

// sourceTypeAbilities maps each enemy damage SourceType to the name of the
// ability that produces it, across all given defs. Enemy→player DamageEvents
// carry no per-enemy id (SourcePeerID is 0), so this map attributes pack damage
// to a specific mob ability by SourceType. When two defs share a SourceType the
// later one wins; for the hallway pack melee and ranged use distinct source
// types (SourceEnemyMelee vs SourceEnemyRanged) so attribution is exact.
func sourceTypeAbilities(defs []*enemyai.EnemyDef) map[uint8]string {
	m := make(map[uint8]string)
	for _, def := range defs {
		for i := range def.Abilities {
			a := &def.Abilities[i]
			if a.DamageSource == combat.SourcePlayerAttack {
				continue // not an enemy-sourced ability
			}
			m[a.DamageSource] = a.Name
		}
	}
	return m
}
