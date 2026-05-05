package system

import (
	"codex-online/server/internal/combat"
	"codex-online/server/internal/combatlog"
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

		// Snapshot buff IDs before tick to detect expiry
		var prevBuffs [8]string
		nPrev := 0
		if len(w.CombatLogs) > 0 {
			for i := range p.Buffs {
				if nPrev < len(prevBuffs) {
					prevBuffs[nPrev] = p.Buffs[i].ID
					nPrev++
				}
			}
		}

		w.enemyTargetBuf = enemiesToTargets(w.enemyTargetBuf, w.Enemies)
		w.abilTickCtx.Targets = w.enemyTargetBuf
		w.abilTickCtx.Obstacles = w.Level.Obstacles
		results := w.AbilityEngine.TickPlayer(p, dt, &w.abilTickCtx)

		// Detect expired buffs
		if len(w.CombatLogs) > 0 {
			for i := 0; i < nPrev; i++ {
				found := false
				for j := range p.Buffs {
					if p.Buffs[j].ID == prevBuffs[i] {
						found = true
						break
					}
				}
				if !found {
					w.logCombatEvent(combatlog.LogEntry{
						EventType:    combatlog.EventBuffRemove,
						SourceEntity: combatlog.FormatPlayerID(p.ID),
						SourceClass:  p.ClassID,
						Target:       combatlog.FormatPlayerID(p.ID),
						AbilityID:    prevBuffs[i],
					})
				}
			}
		}

		// Convert tick results (DoT damage, blade_swirl ticks) to combat events
		for _, r := range results {
			w.DamageEvents = append(w.DamageEvents, combat.DamageEvent{
				TargetPeerID: r.TargetID,
				SourcePeerID: r.SourceID,
				Amount:       r.Amount,
				HitPos:       r.HitPos,
				SourceType:   r.SourceType,
			})

			// Log DoT / tick damage
			w.logCombatEvent(combatlog.LogEntry{
				EventType:    combatlog.EventBuffTick,
				SourceEntity: combatlog.FormatPlayerID(r.SourceID),
				SourceClass:  p.ClassID,
				Target:       combatlog.FormatEnemyID(r.TargetID),
				Amount:       r.Amount,
				PosX:         r.HitPos.X,
				PosY:         r.HitPos.Y,
				PosZ:         r.HitPos.Z,
			})

			if enemy, ok := r.Target.(*entity.Enemy); ok {
				w.AggroEnemy(enemy, r.SourceID)
				if !enemy.Alive {
					w.logCombatDeath(combatlog.FormatEnemyID(r.TargetID), combatlog.FormatPlayerID(r.SourceID), p.ClassID)
					checkEnemyGroupDead(w, enemy)
				}
				w.logPhaseChange(enemy)
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
