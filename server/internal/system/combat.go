package system

import "codex-online/server/internal/entity"

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
	// Tick down fire cooldowns and reset attack state
	for _, p := range w.Players {
		if !p.Alive {
			continue
		}
		p.FireCooldown -= dt
		if p.State == entity.PlayerStateAttack && p.FireCooldown <= 0 {
			p.State = entity.PlayerStateMove
		}
		// Blade dancer guard timer
		if p.GuardActive {
			p.GuardTimer -= dt
			if p.GuardTimer <= 0 {
				p.GuardActive = false
				p.GuardTimer = 0
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
			if e != nil && e.Alive && e.HasThreat(p.PeerID) {
				// When the boss gate is active, only count enemies on the same side
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
