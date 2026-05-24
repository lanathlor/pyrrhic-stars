package system

import (
	"codex-online/server/internal/ability"
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
				AbilityID:    r.AbilityID,
				Amount:       r.Amount,
				PosX:         r.HitPos.X,
				PosY:         r.HitPos.Y,
				PosZ:         r.HitPos.Z,
			})

			if enemy, ok := r.Target.(*entity.Enemy); ok {
				w.AggroEnemy(enemy, r.SourceID)
				if !enemy.Alive {
					w.logCombatDeath(combatlog.FormatEnemyID(r.TargetID), combatlog.FormatPlayerID(r.SourceID), p.ClassID, r.AbilityID)
					checkEnemyGroupDead(w, enemy)
				}
				w.logPhaseChange(enemy)
			}
		}
	}

	// Tick player ability runners (commit→execute→sustain→cooldown lifecycle)
	for _, p := range w.Players {
		if !p.Alive {
			continue
		}
		runner, ok := w.AbilityRunners[p.ID]
		if !ok || !runner.IsBusy() {
			continue
		}

		// Enforce cancel conditions during sustain before ticking
		if runner.Phase == ability.PRunnerSustain && runner.Def != nil {
			cc := runner.Def.CancelConditions
			if cc&uint8(ability.CancelOnDamage) != 0 && p.LastDamageTick > runner.SustainStartTick {
				if id, cd := runner.SustainCooldownOnCancel(); cd > 0 {
					p.Cooldowns[id] = cd
				}
				runner.Cancel()
				if p.Confluence != nil {
					p.Confluence.OnInterrupt()
				}
				runner.SyncToPlayer(p)
				continue
			}
			if cc&uint8(ability.CancelOnMove) != 0 {
				dx := p.Position.X - runner.SustainStartPos.X
				dz := p.Position.Z - runner.SustainStartPos.Z
				if dx*dx+dz*dz > 0.25 {
					if id, cd := runner.SustainCooldownOnCancel(); cd > 0 {
						p.Cooldowns[id] = cd
					}
					runner.Cancel()
					if p.Confluence != nil {
						p.Confluence.OnInterrupt()
					}
					runner.SyncToPlayer(p)
					continue
				}
			}
		}

		prevPhase := runner.Phase
		shouldFire := runner.Tick(dt)

		// When entering sustain from execute, record start position and tick
		if prevPhase == ability.PRunnerExecute && runner.Phase == ability.PRunnerSustain {
			runner.SustainStartPos = p.Position
			runner.SustainStartTick = w.TickNum
		}

		runner.SyncToPlayer(p)

		if shouldFire {
			switch runner.Phase {
			case ability.PRunnerExecute, ability.PRunnerCooldown, ability.PRunnerSustain:
				switch prevPhase {
				case ability.PRunnerCommit:
					// Commit expired → execute ability
					w.enemyTargetBuf = enemiesToTargets(w.enemyTargetBuf, w.Enemies)
					ctx := &ability.CommitContext{
						Committer:    p,
						Targets:      w.enemyTargetBuf,
						Obstacles:    w.Level.Obstacles,
						Allies:       w.Players,
						TargetPeerID: p.ChannelTargetID,
					}
					result := w.AbilityEngine.Commit(runner.AbilityID, ctx)
					for _, h := range result.Heals {
						w.DamageEvents = append(w.DamageEvents, combat.DamageEvent{
							TargetPeerID: h.TargetID,
							SourcePeerID: h.SourceID,
							Amount:       h.Amount,
							Overheal:     h.Overheal,
							HitPos:       h.HitPos,
							SourceType:   h.SourceType,
						})
					}
					for _, r := range result.Events {
						w.DamageEvents = append(w.DamageEvents, combat.DamageEvent{
							TargetPeerID: r.TargetID,
							SourcePeerID: r.SourceID,
							Amount:       r.Amount,
							HitPos:       r.HitPos,
							SourceType:   r.SourceType,
						})
						if enemy, ok := r.Target.(*entity.Enemy); ok {
							enemy.AddThreat(p.ID, r.Amount)
							w.AggroEnemy(enemy, p.ID)
							if !enemy.Alive {
								w.logCombatDeath(
									combatlog.FormatEnemyID(r.TargetID),
									combatlog.FormatPlayerID(p.ID),
									p.ClassID,
									runner.AbilityID,
								)
								checkEnemyGroupDead(w, enemy)
							}
							w.logPhaseChange(enemy)
						}
					}
				case ability.PRunnerSustain:
					// Sustain tick → apply per-tick effect
					applySustainTick(w, p, runner)
				}
			}
		}
	}

	// Tick healing zones
	aliveZones := w.HealingZones[:0]
	for _, zone := range w.HealingZones {
		if zone.Tick(dt) {
			continue // expired
		}
		if zone.ShouldTick(dt) {
			for _, p := range w.Players {
				if !p.Alive || !zone.ContainsPoint(p.Position) {
					continue
				}
				healAmount := zone.HealPerTick
				// Sympathetic Field: amplify zone heal if the target is
				// inside the zone owner's aura radius.
				if owner, ok := w.Players[zone.OwnerID]; ok {
					if r := owner.SympatheticFieldRadius(); r > 0 {
						dx := owner.Position.X - p.Position.X
						dz := owner.Position.Z - p.Position.Z
						if dx*dx+dz*dz <= r*r {
							healAmount *= 1.15
						}
					}
				}
				before := p.Health
				p.Health += healAmount
				if p.Health > p.MaxHealth {
					p.Health = p.MaxHealth
				}
				actual := p.Health - before
				overheal := healAmount - actual
				if actual > 0 || overheal > 0 {
					w.DamageEvents = append(w.DamageEvents, combat.DamageEvent{
						TargetPeerID: p.ID,
						SourcePeerID: zone.OwnerID,
						Amount:       actual,
						Overheal:     overheal,
						HitPos:       p.Position.Add(entity.Vec3{Y: 1.0}),
						SourceType:   combat.SourcePlayerHeal,
					})
				}
			}
		}
		aliveZones = append(aliveZones, zone)
	}
	w.HealingZones = aliveZones

	// Tick player HoTs (Regeneration Protocol)
	for _, p := range w.Players {
		if !p.Alive || len(p.HoTs) == 0 {
			continue
		}
		alive := p.HoTs[:0]
		for i := range p.HoTs {
			hot := &p.HoTs[i]
			hot.Remaining -= dt
			if hot.Remaining <= 0 {
				continue // expired
			}
			// Emergency burst: consume all remaining ticks when HP < threshold
			if hot.BurstThreshold > 0 && p.Health < p.MaxHealth*hot.BurstThreshold {
				remainingTicks := int(hot.Remaining/hot.Interval) + 1
				burst := float32(remainingTicks) * hot.HealPerTick
				before := p.Health
				p.Health += burst
				if p.Health > p.MaxHealth {
					p.Health = p.MaxHealth
				}
				actual := p.Health - before
				if actual > 0 {
					w.DamageEvents = append(w.DamageEvents, combat.DamageEvent{
						TargetPeerID: p.ID,
						SourcePeerID: hot.SourcePeer,
						Amount:       actual,
						HitPos:       p.Position.Add(entity.Vec3{Y: 1.0}),
						SourceType:   combat.SourcePlayerHeal,
					})
				}
				continue // consumed
			}
			hot.TickTimer -= dt
			if hot.TickTimer <= 0 {
				hot.TickTimer += hot.Interval
				before := p.Health
				p.Health += hot.HealPerTick
				if p.Health > p.MaxHealth {
					p.Health = p.MaxHealth
				}
				actual := p.Health - before
				if actual > 0 {
					w.DamageEvents = append(w.DamageEvents, combat.DamageEvent{
						TargetPeerID: p.ID,
						SourcePeerID: hot.SourcePeer,
						Amount:       actual,
						HitPos:       p.Position.Add(entity.Vec3{Y: 1.0}),
						SourceType:   combat.SourcePlayerHeal,
					})
				}
			}
			alive = append(alive, *hot)
		}
		p.HoTs = alive
	}

	// Tick damage links (Vital Circuit)
	aliveLinks := w.DamageLinks[:0]
	for _, link := range w.DamageLinks {
		link.Duration -= dt
		if link.Duration <= 0 {
			// On expiry: heal the lower-HP ally for 30% of the HP difference
			pa, okA := w.Players[link.PeerA]
			pb, okB := w.Players[link.PeerB]
			if okA && okB && pa.Alive && pb.Alive {
				diff := pb.Health - pa.Health
				var healTarget *entity.Player
				if diff > 0 {
					healTarget = pa
				} else if diff < 0 {
					healTarget = pb
					diff = -diff
				}
				if healTarget != nil && diff > 0 {
					healAmount := diff * 0.3
					before := healTarget.Health
					healTarget.Health += healAmount
					if healTarget.Health > healTarget.MaxHealth {
						healTarget.Health = healTarget.MaxHealth
					}
					actual := healTarget.Health - before
					if actual > 0 {
						w.DamageEvents = append(w.DamageEvents, combat.DamageEvent{
							TargetPeerID: healTarget.ID,
							SourcePeerID: link.SourcePeer,
							Amount:       actual,
							HitPos:       healTarget.Position.Add(entity.Vec3{Y: 1.0}),
							SourceType:   combat.SourcePlayerHeal,
						})
					}
				}
			}
			continue // expired
		}
		aliveLinks = append(aliveLinks, link)
	}
	w.DamageLinks = aliveLinks

	// Last Breath expiry: when death prevention buff disappears, caster takes 50% of prevented damage
	for _, p := range w.Players {
		if !p.Alive || p.LastBreathCasterID == 0 {
			continue
		}
		if !p.HasBuff("last_breath") {
			// Buff expired — apply deferred damage to caster
			if p.LastBreathPrevented > 0 {
				if caster, ok := w.Players[p.LastBreathCasterID]; ok && caster.Alive {
					selfDmg := p.LastBreathPrevented * 0.5
					dealt := caster.ApplyDamage(selfDmg)
					if dealt > 0 {
						w.DamageEvents = append(w.DamageEvents, combat.DamageEvent{
							TargetPeerID: caster.ID,
							SourcePeerID: caster.ID,
							Amount:       dealt,
							HitPos:       caster.Position.Add(entity.Vec3{Y: 1.0}),
							SourceType:   combat.SourcePlayerAttack,
						})
					}
				}
			}
			p.LastBreathPrevented = 0
			p.LastBreathCasterID = 0
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

		// Flux regen boost when out of combat (same 5% rate as HP)
		if !inCombat {
			if fc := p.FluxCommit; fc != nil && len(fc.Pools) > 0 {
				for i := range fc.Pools {
					pool := &fc.Pools[i]
					if pool.Current < pool.Max {
						pool.Current += pool.Max * RegenRate * dt
						if pool.Current > pool.Max {
							pool.Current = pool.Max
						}
					}
				}
				p.SyncFluxAggregate()
			} else if r := p.Resources[entity.ResourceFlux]; r != nil && r.Current < r.Max {
				r.Current += r.Max * RegenRate * dt
				if r.Current > r.Max {
					r.Current = r.Max
				}
			}
		}
	}
}

// applySustainTick applies a single sustain tick effect (heal or damage with scaling).
func applySustainTick(w *World, p *entity.Player, runner *ability.PlayerAbilityRunner) {
	def := runner.Def
	if def == nil {
		return
	}

	// Drain flux (school-aware for FluxCommitment players)
	fluxCost := def.SustainCostPerSec * def.SustainInterval
	if fluxCost > 0 {
		var ok bool
		if def.School != "" && p.FluxCommit != nil && len(p.FluxCommit.Pools) > 0 {
			ok = p.SpendFluxBySchool(def.School, fluxCost)
		} else {
			ok = p.SpendResource("flux", fluxCost)
		}
		if !ok {
			if id, cd := runner.SustainCooldownOnCancel(); cd > 0 {
				p.Cooldowns[id] = cd
			}
			runner.Cancel()
			if p.Confluence != nil {
				p.Confluence.OnInterrupt()
			}
			runner.SyncToPlayer(p)
			return
		}
	}

	// Compute scaling multiplier
	multiplier := float32(1.0) + runner.SustainElapsed*def.SustainScaling
	amount := def.SustainEffect * multiplier

	// Transfusion: drain target ally HP, heal all OTHER allies
	if def.SustainHandler == "transfusion" {
		targetID := p.ChannelTargetID
		target, ok := w.Players[targetID]
		if !ok || !target.Alive {
			if id, cd := runner.SustainCooldownOnCancel(); cd > 0 {
				p.Cooldowns[id] = cd
			}
			runner.Cancel()
			runner.SyncToPlayer(p)
			return
		}
		drainAmount := amount
		if target.Health-drainAmount < 1 {
			drainAmount = target.Health - 1
		}
		if drainAmount <= 0 {
			if id, cd := runner.SustainCooldownOnCancel(); cd > 0 {
				p.Cooldowns[id] = cd
			}
			runner.Cancel()
			runner.SyncToPlayer(p)
			return
		}
		target.Health -= drainAmount
		w.DamageEvents = append(w.DamageEvents, combat.DamageEvent{
			TargetPeerID: targetID,
			SourcePeerID: p.ID,
			Amount:       -drainAmount, // negative = drain
			HitPos:       target.Position.Add(entity.Vec3{Y: 1.0}),
			SourceType:   combat.SourcePlayerHeal,
		})
		// Heal all other allies within 10m of caster
		for _, ally := range w.Players {
			if ally.ID == targetID || ally.ID == p.ID || !ally.Alive {
				continue
			}
			dx := p.Position.X - ally.Position.X
			dz := p.Position.Z - ally.Position.Z
			if dx*dx+dz*dz > 100 { // 10m radius
				continue
			}
			before := ally.Health
			ally.Health += drainAmount
			if ally.Health > ally.MaxHealth {
				ally.Health = ally.MaxHealth
			}
			actual := ally.Health - before
			overheal := drainAmount - actual
			if actual > 0 || overheal > 0 {
				w.DamageEvents = append(w.DamageEvents, combat.DamageEvent{
					TargetPeerID: ally.ID,
					SourcePeerID: p.ID,
					Amount:       actual,
					Overheal:     overheal,
					HitPos:       ally.Position.Add(entity.Vec3{Y: 1.0}),
					SourceType:   combat.SourcePlayerHeal,
				})
			}
		}
		return
	}

	// Apply effect: heal ally target or damage enemy
	if def.Hit.Type == ability.HitAllyTarget {
		// Heal: target the channel target ally
		targetID := p.ChannelTargetID
		target, ok := w.Players[targetID]
		if !ok || !target.Alive {
			// Fallback: heal self
			target = p
			targetID = p.ID
		}
		before := target.Health
		target.Health += amount
		if target.Health > target.MaxHealth {
			target.Health = target.MaxHealth
		}
		actual := target.Health - before
		overheal := amount - actual
		if actual > 0 || overheal > 0 {
			w.DamageEvents = append(w.DamageEvents, combat.DamageEvent{
				TargetPeerID: targetID,
				SourcePeerID: p.ID,
				Amount:       actual,
				Overheal:     overheal,
				HitPos:       target.Position.Add(entity.Vec3{Y: 1.0}),
				SourceType:   combat.SourcePlayerHeal,
			})
		}
	} else if def.Hit.Type == ability.HitNearestN {
		// Damage: hit nearest alive enemy, heal lowest ally for 50%
		for _, e := range w.Enemies {
			if e == nil || !e.Alive {
				continue
			}
			dealt := e.TargetApplyDamage(amount)
			if dealt > 0 {
				w.DamageEvents = append(w.DamageEvents, combat.DamageEvent{
					TargetPeerID: e.ID,
					SourcePeerID: p.ID,
					Amount:       dealt,
					HitPos:       e.Position.Add(entity.Vec3{Y: 1.0}),
					SourceType:   combat.SourcePlayerAttack,
				})
				e.AddThreat(p.ID, dealt)
				w.AggroEnemy(e, p.ID)
				if !e.Alive {
					w.logCombatDeath(combatlog.FormatEnemyID(e.ID), combatlog.FormatPlayerID(p.ID), p.ClassID, runner.AbilityID)
					checkEnemyGroupDead(w, e)
				}
				w.logPhaseChange(e)

				// Heal lowest ally for 50% of damage dealt
				healAmount := dealt * 0.5
				if healAmount > 0 {
					var healTarget *entity.Player
					var lowestHP float32 = 999999
					for _, ally := range w.Players {
						if ally.Alive && ally.Health < ally.MaxHealth && ally.Health < lowestHP {
							lowestHP = ally.Health
							healTarget = ally
						}
					}
					if healTarget == nil {
						healTarget = p
					}
					before := healTarget.Health
					healTarget.Health += healAmount
					if healTarget.Health > healTarget.MaxHealth {
						healTarget.Health = healTarget.MaxHealth
					}
					actual := healTarget.Health - before
					if actual > 0 {
						w.DamageEvents = append(w.DamageEvents, combat.DamageEvent{
							TargetPeerID: healTarget.ID,
							SourcePeerID: p.ID,
							Amount:       actual,
							HitPos:       healTarget.Position.Add(entity.Vec3{Y: 1.0}),
							SourceType:   combat.SourcePlayerHeal,
						})
					}
				}
			}
			break // only hit nearest
		}
	}
}
