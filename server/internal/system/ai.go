package system

import (
	"codex-online/server/internal/combat"
	"codex-online/server/internal/combatlog"
	"codex-online/server/internal/enemyai"
	"codex-online/server/internal/entity"
)

// AISystem ticks enemy brains during the fight state.
type AISystem struct{}

func (s *AISystem) Tick(w *World, dt float32) {
	if !w.CombatActive() {
		return
	}

	// Build player slice once for all brains (avoids per-brain map iteration).
	allPlayers := w.playerSlice[:0]
	for _, p := range w.Players {
		allPlayers = append(allPlayers, p)
	}
	w.playerSlice = allPlayers

	// Lazy-init a single spawn closure on the World (allocated once, not per tick).
	if w.spawnFn == nil {
		w.spawnFn = func(pos, dir entity.Vec3, speed, damage, lifetime float32) {
			w.SpawnEnemyProjectile(w.spawnEnemyIdx, pos, dir, speed, damage, lifetime)
		}
	}
	if w.commitPatternFn == nil && w.PatternEngine != nil {
		w.commitPatternFn = func(pattern *combat.PatternDef, abilityName string, origin, facing entity.Vec3) {
			w.PatternEngine.Spawn(pattern, abilityName, 0, w.spawnEnemyIdx, origin, facing)
		}
	}

	for i, e := range w.Enemies {
		if e == nil || !e.Alive || i >= len(w.Brains) {
			continue
		}
		// Skip patrol enemies with no threat and no nearby players.
		// Avoids the full BT tick (60+ node traversals) for idle mobs.
		// propagateGroupAggro wakes them when a group member enters combat.
		if e.State == entity.EnemyPatrol && len(e.ThreatTable) == 0 && !anyPlayerNearby(e.Position, allPlayers, 10) {
			continue
		}
		w.spawnEnemyIdx = i
		visiblePlayers := allPlayers
		if gateZ, ok := w.ClosedGateZ(); ok {
			w.filteredPlayers = playersOnSameSide(w.filteredPlayers[:0], allPlayers, e.Position.Z, gateZ)
			visiblePlayers = w.filteredPlayers
		}
		tickEnemyBrain(w, i, e, visiblePlayers, dt)
	}

	// Dev mode: auto-repeat a specific ability on the boss.
	// Only force-commit when the boss is idle (chase state) so the current
	// ability completes its full commit→execute→cooldown cycle first.
	if w.DevMode && w.DebugRepeatAbility != "" {
		for i, brain := range w.Brains {
			if i < len(w.Enemies) && w.Enemies[i].IsBoss && w.Enemies[i].Alive {
				if w.Enemies[i].State == entity.EnemyChase {
					brain.ForceCommit(w.DebugRepeatAbility)
				}
			}
		}
	}

	// Group aggro propagation: if any mob in a group is chasing, wake all
	// patrol members of that group.
	propagateGroupAggro(w)
}

func tickEnemyBrain(w *World, idx int, e *entity.Enemy, allPlayers []*entity.Player, dt float32) {
	prevState := e.State
	events := w.Brains[idx].Tick(dt, allPlayers, w.Obstacles, w.spawnFn, w.commitPatternFn)

	// Apply group-size damage scaling to direct hits (melee, AoE, charge).
	if mult := w.EnemyDmgMult(); mult != 1.0 {
		for j := range events {
			events[j].Amount *= mult
		}
	}

	// Detect proximity aggro: brain transitioned patrol→chase directly
	// (bypassing AggroEnemy). Start combat log session for this enemy's group.
	if prevState == entity.EnemyPatrol && e.State != entity.EnemyPatrol {
		if key := enemySessionKey(e); key != 0 {
			startGroupCombatLog(w, key)
		}
	}
	for _, evt := range events {
		if _, ok := w.Players[evt.TargetPeerID]; ok {
			e.AddThreat(evt.TargetPeerID, evt.Amount)
		}

		// Log enemy damage
		abilName := resolveEnemyAbilityName(e)
		w.logCombatEvent(combatlog.LogEntry{
			EventType:    combatlog.EventDamage,
			SourceEntity: combatlog.FormatEnemyID(e.ID),
			SourceClass:  e.DefName,
			Target:       combatlog.FormatPlayerID(evt.TargetPeerID),
			AbilityID:    abilName,
			Amount:       evt.Amount,
			PosX:         evt.HitPos.X,
			PosY:         evt.HitPos.Y,
			PosZ:         evt.HitPos.Z,
		})

		// Log death if player died
		if p, ok := w.Players[evt.TargetPeerID]; ok && !p.Alive {
			w.logCombatDeath(combatlog.FormatPlayerID(evt.TargetPeerID), combatlog.FormatEnemyID(e.ID), e.DefName, abilName)
		}
	}
	w.DamageEvents = append(w.DamageEvents, events...)
	w.Level.ClampEnemy(&e.Position)
	combat.PushOutOfObstacles(&e.Position, w.Obstacles, w.Level.EnemyRadius)
}

// playersOnSameSide filters players to those on the same side of gateZ as enemyZ.
// Uses the provided dst slice to avoid allocation.
func playersOnSameSide(dst []*entity.Player, players []*entity.Player, enemyZ, gateZ float32) []*entity.Player {
	enemyInBossRoom := enemyZ < gateZ
	for _, p := range players {
		playerInBossRoom := p.Position.Z < gateZ
		if playerInBossRoom == enemyInBossRoom {
			dst = append(dst, p)
		}
	}
	return dst
}

// resolveEnemyAbilityName looks up the current ability name for an enemy from its def.
func resolveEnemyAbilityName(e *entity.Enemy) string {
	def := enemyai.DefRegistry[e.DefName]
	if def == nil {
		return ""
	}
	abil := def.AbilityByIndex(e.ActiveAbility)
	if abil == nil {
		return ""
	}
	return abil.Name
}

// anyPlayerNearby returns true if any alive player is within radius of pos.
// Used as a cheap pre-check to skip full BT ticks for idle patrol enemies.
func anyPlayerNearby(pos entity.Vec3, players []*entity.Player, radius float32) bool {
	rSq := radius * radius
	for _, p := range players {
		if p.Alive && p.Position.DistanceToSq(pos) <= rSq {
			return true
		}
	}
	return false
}

// propagateGroupAggro ensures that if any mob in a group has left patrol
// (e.g. due to proximity aggro), all other patrol members in the same group
// also switch to chase. Uses O(n²) scan instead of a map to avoid allocation.
func propagateGroupAggro(w *World) {
	for _, e := range w.Enemies {
		if e == nil || !e.Alive || e.GroupID == 0 || e.State != entity.EnemyPatrol {
			continue
		}
		// Check if any group member is already aggroed
		for _, other := range w.Enemies {
			if other == e || other == nil || !other.Alive || other.GroupID != e.GroupID {
				continue
			}
			if other.State != entity.EnemyPatrol {
				e.State = entity.EnemyChase
				e.ChaseTimer = 0
				e.TargetPlayerID = other.TargetPlayerID
				break
			}
		}
	}
}
