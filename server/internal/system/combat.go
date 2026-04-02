package system

import "codex-online/server/internal/entity"

const (
	// CombatExitTicks is the number of ticks after last damage before
	// a player is considered out of combat (5 seconds at 20Hz).
	CombatExitTicks uint32 = 100

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
	}

	// Update combat state per player
	for _, p := range w.Players {
		if !p.Alive {
			continue
		}

		inCombat := false

		// Check if any alive enemy is targeting this player
		for _, e := range w.Enemies {
			if e != nil && e.Alive && e.TargetPlayerID == p.PeerID {
				inCombat = true
				break
			}
		}

		// Check if player took damage recently
		if !inCombat && p.LastDamageTick > 0 && w.TickNum-p.LastDamageTick < CombatExitTicks {
			inCombat = true
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
