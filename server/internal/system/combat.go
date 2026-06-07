package system

import (
	"math"

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
	tickAbilityResults(w, dt)
	tickAbilityRunners(w, dt)
	tickHealingZones(w, dt)
	tickHoTs(w, dt)
	tickDamageLinks(w, dt)
	tickLastBreath(w)
	tickCombatState(w, dt)
}

func tickAbilityResults(w *World, dt float32) {
	for _, p := range w.Players {
		if !p.Alive {
			continue
		}
		tickPlayerAbilityResult(w, p, dt)
	}
}

func tickPlayerAbilityResult(w *World, p *entity.Player, dt float32) {
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
	w.abilTickCtx.Obstacles = w.Obstacles
	results := w.AbilityEngine.TickPlayer(p, dt, &w.abilTickCtx)

	logExpiredBuffs(w, p, prevBuffs, nPrev)
	processAbilityTickResults(w, p, results)
}

func logExpiredBuffs(w *World, p *entity.Player, prevBuffs [8]string, nPrev int) {
	if len(w.CombatLogs) == 0 {
		return
	}
	for i := range nPrev {
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

func processAbilityTickResults(w *World, p *entity.Player, results []ability.DamageResult) {
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

func tickAbilityRunners(w *World, dt float32) {
	for _, p := range w.Players {
		if !p.Alive {
			continue
		}
		runner, ok := w.AbilityRunners[p.ID]
		if !ok || !runner.IsBusy() {
			continue
		}

		if checkRunnerCancelConditions(p, runner) {
			continue
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
					executeCommitPhase(w, p, runner)
				case ability.PRunnerSustain:
					// Sustain tick → apply per-tick effect
					applySustainTick(w, p, runner)
				}
			}
		}
	}
}

// checkRunnerCancelConditions enforces cancel conditions during sustain.
// Returns true if the runner was cancelled and the caller should skip to the next player.
func checkRunnerCancelConditions(p *entity.Player, runner *ability.PlayerAbilityRunner) bool {
	if runner.Phase != ability.PRunnerSustain || runner.Def == nil {
		return false
	}
	cc := runner.Def.CancelConditions
	if cc&uint8(ability.CancelOnDamage) != 0 && p.LastDamageTick > runner.SustainStartTick {
		cancelRunner(p, runner)
		return true
	}
	if cc&uint8(ability.CancelOnMove) != 0 {
		dx := p.Position.X - runner.SustainStartPos.X
		dz := p.Position.Z - runner.SustainStartPos.Z
		if dx*dx+dz*dz > 0.25 {
			cancelRunner(p, runner)
			return true
		}
	}
	return false
}

func cancelRunner(p *entity.Player, runner *ability.PlayerAbilityRunner) {
	if id, cd := runner.SustainCooldownOnCancel(); cd > 0 {
		p.Cooldowns[id] = cd
	}
	runner.Cancel()
	if p.Confluence != nil {
		p.Confluence.OnInterrupt()
	}
	runner.SyncToPlayer(p)
}

func executeCommitPhase(w *World, p *entity.Player, runner *ability.PlayerAbilityRunner) {
	w.enemyTargetBuf = enemiesToTargets(w.enemyTargetBuf, w.Enemies)
	ctx := &ability.CommitContext{
		Committer:    p,
		Targets:      w.enemyTargetBuf,
		Obstacles:    w.Obstacles,
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
}

func tickHealingZones(w *World, dt float32) {
	aliveZones := w.HealingZones[:0]
	for _, zone := range w.HealingZones {
		if zone.Tick(dt) {
			continue // expired
		}
		if zone.ShouldTick(dt) {
			applyHealingZoneToPlayers(w, zone)
		}
		aliveZones = append(aliveZones, zone)
	}
	w.HealingZones = aliveZones
}

func applyHealingZoneToPlayers(w *World, zone *entity.HealingZone) {
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

func tickHoTs(w *World, dt float32) {
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
			if tickSingleHoT(w, p, hot, dt) {
				alive = append(alive, *hot)
			}
		}
		p.HoTs = alive
	}
}

// tickSingleHoT processes one HoT tick. Returns false if the HoT was consumed (burst).
func tickSingleHoT(w *World, p *entity.Player, hot *entity.ActiveHoT, dt float32) bool {
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
		return false // consumed
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
	return true
}

func tickDamageLinks(w *World, dt float32) {
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
}

func tickLastBreath(w *World) {
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
}

func tickCombatState(w *World, dt float32) {
	for _, p := range w.Players {
		if !p.Alive {
			continue
		}
		p.InCombat = isPlayerInCombat(w, p)
		if !p.InCombat {
			regenPlayerResources(p, dt)
		}
	}
}

func isPlayerInCombat(w *World, p *entity.Player) bool {
	gateZ, gateActive := w.ClosedGateZ()
	playerBeyondGate := gateActive && p.Position.Z < gateZ
	for _, e := range w.Enemies {
		if e != nil && e.Alive && e.HasThreat(p.ID) {
			if gateActive {
				enemyBeyondGate := e.Position.Z < gateZ
				if playerBeyondGate != enemyBeyondGate {
					continue
				}
			}
			return true
		}
	}
	return false
}

func regenPlayerResources(p *entity.Player, dt float32) {
	// HP regen when out of combat
	if p.Health < p.MaxHealth {
		p.Health += p.MaxHealth * RegenRate * dt
		if p.Health > p.MaxHealth {
			p.Health = p.MaxHealth
		}
	}

	// Flux regen boost when out of combat (same 5% rate as HP)
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
			cancelRunner(p, runner)
			return
		}
	}

	// Compute scaling multiplier
	multiplier := float32(1.0) + runner.SustainElapsed*def.SustainScaling
	amount := def.SustainEffect * multiplier

	if def.SustainHandler == ability.IDTransfusion {
		handleTransfusionSustain(w, p, runner, amount)
		return
	}

	switch def.Hit.Type {
	case ability.HitAllyTarget:
		applySustainAllyHeal(w, p, amount)
	case ability.HitNearestN:
		applySustainEnemyDamage(w, p, runner, amount)
	}
}

func handleTransfusionSustain(w *World, p *entity.Player, runner *ability.PlayerAbilityRunner, amount float32) {
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
}

func applySustainAllyHeal(w *World, p *entity.Player, amount float32) {
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
}

func applySustainEnemyDamage(w *World, p *entity.Player, runner *ability.PlayerAbilityRunner, amount float32) {
	// Find nearest alive enemy by distance to player.
	var nearest *entity.Enemy
	bestDistSq := float32(math.MaxFloat32)
	for _, e := range w.Enemies {
		if e == nil || !e.Alive {
			continue
		}
		distSq := e.Position.DistanceToSq(p.Position)
		if distSq < bestDistSq {
			bestDistSq = distSq
			nearest = e
		}
	}
	if nearest == nil {
		return
	}

	dealt := nearest.TargetApplyDamage(amount)
	if dealt > 0 {
		w.DamageEvents = append(w.DamageEvents, combat.DamageEvent{
			TargetPeerID: nearest.ID,
			SourcePeerID: p.ID,
			Amount:       dealt,
			HitPos:       nearest.Position.Add(entity.Vec3{Y: 1.0}),
			SourceType:   combat.SourcePlayerAttack,
		})
		nearest.AddThreat(p.ID, dealt)
		w.AggroEnemy(nearest, p.ID)
		if !nearest.Alive {
			w.logCombatDeath(combatlog.FormatEnemyID(nearest.ID), combatlog.FormatPlayerID(p.ID), p.ClassID, runner.AbilityID)
			checkEnemyGroupDead(w, nearest)
		}
		w.logPhaseChange(nearest)
		healLowestAllyForAmount(w, p, dealt*0.5)
	}
}

func healLowestAllyForAmount(w *World, p *entity.Player, healAmount float32) {
	if healAmount <= 0 {
		return
	}
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
