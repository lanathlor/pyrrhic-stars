package system

import (
	"codex-online/server/internal/ability"
	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
)

const (
	// RegenRate is the fraction of MaxHealth regenerated per second
	// when out of combat (5% = full heal in 20 seconds).
	RegenRate float32 = 0.05
)

// CombatSystem handles per-player combat state tracking, cooldown ticks,
// and out-of-combat HP regeneration.
type CombatSystem struct{}

// Tick runs the combat system for one frame.
func (s *CombatSystem) Tick(w *World, dt float32) {
	// Delegate per-player ability ticking to the engine
	for _, p := range w.Players {
		if !p.Alive {
			continue
		}

		ctx := &ability.TickContext{
			Targets:   enemiesToTargets(w.Enemies),
			Obstacles: w.Level.Obstacles,
		}
		results := w.AbilityEngine.TickPlayer(p, dt, ctx)

		// Convert tick results (DoT damage, blade_swirl ticks) to combat events
		for _, r := range results {
			w.DamageEvents = append(w.DamageEvents, combat.DamageEvent{
				TargetPeerID: r.TargetID,
				SourcePeerID: r.SourceID,
				Amount:       r.Amount,
				HitPos:       r.HitPos,
				SourceType:   r.SourceType,
			})
			if enemy, ok := r.Target.(*entity.Enemy); ok {
				w.AggroEnemy(enemy, r.SourceID)
			}
		}
	}

	// Update combat state per player
	for _, p := range w.Players {
		if !p.Alive {
			continue
		}

		inCombat := false

		// Player is in combat if they're on any alive enemy's threat table
		// and on the same side of the boss gate
		playerInBossRoom := w.Level != nil && p.Position.Z < w.Level.BossRoomEntryZ
		for _, e := range w.Enemies {
			if e != nil && e.Alive && e.HasThreat(p.ID) {
				if w.BossGateActive {
					enemyInBossRoom := e.Position.Z < w.Level.BossRoomEntryZ
					if playerInBossRoom != enemyInBossRoom {
						continue
					}
				}
				inCombat = true
				break
			}
		}

		p.InCombat = inCombat

		// HP regen when out of combat
		if !inCombat && p.Health < p.MaxHealth {
			p.Health += p.MaxHealth * RegenRate * dt
			if p.Health > p.MaxHealth {
				p.Health = p.MaxHealth
			}
		}
	}
}
