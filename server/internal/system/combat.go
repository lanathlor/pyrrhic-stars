package system

import (
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

		// Gunner: Overclock buff timer
		if p.OverclockActive {
			p.OverclockTimer -= dt
			if p.OverclockTimer <= 0 {
				p.OverclockActive = false
				p.OverclockTimer = 0
			}
		}
		// Gunner: Overclock cooldown (always tick)
		if p.OverclockCooldown > 0 {
			p.OverclockCooldown -= dt
			if p.OverclockCooldown < 0 {
				p.OverclockCooldown = 0
			}
		}

		// Gunner: Rechamber phase transitions
		switch p.RechamberPhase {
		case 1: // windup
			p.RechamberTimer -= dt
			if p.RechamberTimer <= 0 {
				p.RechamberPhase = 2
				p.RechamberTimer = 0.35
			}
		case 2: // timing window
			p.RechamberTimer -= dt
			if p.RechamberTimer <= 0 {
				p.RechamberPhase = 3
				p.RechamberTimer = 0.8
			}
		case 3: // lockout
			p.RechamberTimer -= dt
			if p.RechamberTimer <= 0 {
				p.RechamberPhase = 0
				p.RechamberTimer = 0
			}
		}

		// Gunner: Rechamber damage buff timer
		if p.RechamberBuff {
			p.RechamberBuffTimer -= dt
			if p.RechamberBuffTimer <= 0 {
				p.RechamberBuff = false
				p.RechamberBuffTimer = 0
			}
		}

		// Vanguard: Blade Swirl multi-tick AoE
		if p.BladeSwirl {
			p.BladeSwirlTimer -= dt
			// Deliver additional damage ticks every 0.5s
			expectedTicks := int((1.5 - p.BladeSwirlTimer) / 0.5)
			if expectedTicks > p.BladeSwirlTicks {
				shape := combat.AoEShape{Type: combat.AoECircle, Radius: 6.0, Damage: 25.0}
				events := combat.ResolvePlayerAoEOnEnemies(p, w.Enemies, w.Level.Obstacles, shape)
				for _, evt := range events {
					evt.SourcePeerID = p.PeerID
					w.DamageEvents = append(w.DamageEvents, evt)
					for _, e := range w.Enemies {
						if e != nil && e.ID == evt.TargetPeerID {
							e.AddThreat(p.PeerID, evt.Amount)
							w.AggroEnemy(e, p.PeerID)
							break
						}
					}
				}
				p.BladeSwirlTicks = expectedTicks
			}
			if p.BladeSwirlTimer <= 0 {
				p.BladeSwirl = false
				p.BladeSwirlTimer = 0
				p.BladeSwirlTicks = 0
			}
		}

		// Vanguard: Blade Swirl cooldown
		if p.BladeSwirlCooldown > 0 {
			p.BladeSwirlCooldown -= dt
			if p.BladeSwirlCooldown < 0 {
				p.BladeSwirlCooldown = 0
			}
		}

		// Vanguard: Ground Slam cooldown
		if p.GroundSlamCooldown > 0 {
			p.GroundSlamCooldown -= dt
			if p.GroundSlamCooldown < 0 {
				p.GroundSlamCooldown = 0
			}
		}

		// Vanguard: stamina regen (with delay after spending)
		if p.MaxStamina > 0 && p.Stamina < p.MaxStamina {
			regenDt := dt
			if p.StaminaDelay > 0 {
				p.StaminaDelay -= dt
				if p.StaminaDelay < 0 {
					regenDt = -p.StaminaDelay // leftover time after delay expired
					p.StaminaDelay = 0
				} else {
					regenDt = 0
				}
			}
			if regenDt > 0 {
				p.Stamina += p.StaminaRegen * regenDt
				if p.Stamina > p.MaxStamina {
					p.Stamina = p.MaxStamina
				}
			}
		}

		// Blade Dancer: shield decay (5 HP/sec so it doesn't persist forever)
		if p.BDShieldHP > 0 {
			p.BDShieldHP -= 5.0 * dt
			if p.BDShieldHP < 0 {
				p.BDShieldHP = 0
			}
		}

		// Blade Dancer: DR buff timer
		if p.BDDRTimer > 0 {
			p.BDDRTimer -= dt
			if p.BDDRTimer <= 0 {
				p.BDDRTimer = 0
				p.BDDRFactor = 0
			}
		}

		// Blade Dancer: GCD timer
		if p.GCDTimer > 0 {
			p.GCDTimer -= dt
			if p.GCDTimer < 0 {
				p.GCDTimer = 0
			}
		}
	}

	// Tick Blade Dancer DoTs
	alive := w.BDDoTs[:0]
	for i := range w.BDDoTs {
		dot := &w.BDDoTs[i]
		dot.Remaining -= dt
		if dot.Remaining <= 0 {
			continue // expired
		}
		dot.TickTimer -= dt
		if dot.TickTimer <= 0 {
			dot.TickTimer += dot.Interval
			// Find enemy and apply tick damage
			for _, e := range w.Enemies {
				if e != nil && e.ID == dot.EnemyID && e.Alive {
					dealt, _ := e.ApplyDamage(dot.Damage)
					if dealt > 0 {
						w.DamageEvents = append(w.DamageEvents, combat.DamageEvent{
							TargetPeerID: e.ID,
							SourcePeerID: dot.SourcePeer,
							Amount:       dealt,
							HitPos:       e.Position.Add(entity.Vec3{Y: 1.0}),
							SourceType:   combat.SourcePlayerAttack,
						})
						e.AddThreat(dot.SourcePeer, dealt)
					}
					break
				}
			}
		}
		alive = append(alive, *dot)
	}
	w.BDDoTs = alive

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
