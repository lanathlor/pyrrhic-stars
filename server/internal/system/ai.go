package system

import (
	"log/slog"

	"codex-online/server/internal/combat"
	"codex-online/server/internal/combatlog"
	"codex-online/server/internal/enemyai"
	"codex-online/server/internal/entity"
)

// AISystem ticks enemy brains during the fight state.
type AISystem struct{}

func (s *AISystem) Tick(w *World, dt float32) {
	if len(w.Enemies) == 0 {
		return
	}

	// Build player slice once for all brains (avoids per-brain map iteration).
	allPlayers := w.playerSlice[:0]
	for _, p := range w.Players {
		allPlayers = append(allPlayers, p)
	}
	w.playerSlice = allPlayers

	ensureBrainCallbacks(w)

	// Owner-death sweep: adds die with their summoner. Runs here (not in
	// gameflow) so the bosstest simulation pipeline gets it too.
	sweepOrphanedAdds(w)

	// Coordination bus: advance the clock, prune old events, and re-derive
	// combat clusters from current aggro state (GroupID + shared threat).
	if w.Bus == nil {
		w.Bus = enemyai.NewBus()
	}
	w.Bus.BeginTick(w.TickNum, dt)
	w.Bus.RebuildClusters(w.Enemies)

	for i, e := range w.Enemies {
		if e == nil || !e.Alive || i >= len(w.Brains) {
			continue
		}
		// Filter players to same side of every closed gate (lobby, boss, decline).
		w.spawnEnemyIdx = i
		visiblePlayers := allPlayers
		if w.AnyGateClosed() {
			w.filteredPlayers = w.filterPlayersByClosedGates(w.filteredPlayers[:0], allPlayers, e.Position.Z)
			visiblePlayers = w.filteredPlayers
		}
		// Skip patrol enemies with no threat and no nearby visible players.
		// Avoids the full BT tick (60+ node traversals) for idle mobs.
		// propagateGroupAggro wakes them when a group member enters combat.
		if e.State == entity.EnemyPatrol && len(e.ThreatTable) == 0 && !anyPlayerNearby(e.Position, visiblePlayers, 10) {
			continue
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

// ensureBrainCallbacks lazy-inits the closures brains call back into the
// World with (allocated once on first tick, not per tick).
func ensureBrainCallbacks(w *World) {
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
	if w.spawnAddFn == nil {
		w.spawnAddFn = func(defName string, pos entity.Vec3, ownerID uint16) bool {
			return spawnAddEnemy(w, defName, pos, ownerID)
		}
	}
}

func tickEnemyBrain(w *World, idx int, e *entity.Enemy, allPlayers []*entity.Player, dt float32) {
	prevState := e.State
	w.Brains[idx].SetBus(w.Bus)
	w.Brains[idx].SetSpawnAddFn(w.spawnAddFn)
	w.Brains[idx].SetAllies(w.Enemies)
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

	// Log ability commits (telegraph onset). Without these a live debrief
	// cannot tell "enemy never attacked" apart from "attacked and missed" —
	// damage events only record hits that landed.
	if !isTelegraphState(prevState) && isTelegraphState(e.State) {
		w.logCombatEvent(combatlog.LogEntry{
			EventType:    combatlog.EventCommitStart,
			SourceEntity: combatlog.FormatEnemyID(e.ID),
			SourceClass:  e.DefName,
			AbilityID:    resolveEnemyAbilityName(e),
			PosX:         e.Position.X,
			PosY:         e.Position.Y,
			PosZ:         e.Position.Z,
		})
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

// spawnAddEnemy summons a new enemy mid-fight (CategorySpawn abilities).
// The new enemy is appended to both parallel slices (Enemies + Brains),
// inherits the owner's GroupID (joining its bus cluster), and dies with its
// owner via sweepOrphanedAdds. Never removed from the slice — despawn = dead.
func spawnAddEnemy(w *World, defName string, pos entity.Vec3, ownerID uint16) bool {
	def := enemyai.DefRegistry[defName]
	if def == nil {
		slog.Warn("spawn add: unknown enemy def", "def_name", defName, "zone_id", w.ZoneID)
		return false
	}
	if w.NextDynEnemyID == 0 {
		w.NextDynEnemyID = 1500
	}
	if w.NextDynEnemyID >= 2000 {
		slog.Warn("spawn add: dynamic enemy ID range exhausted", "zone_id", w.ZoneID)
		return false
	}
	id := w.NextDynEnemyID
	w.NextDynEnemyID++

	e := buildAddEnemy(w, def, defName, pos, ownerID, id)
	owner := enemyByID(w, ownerID)
	if owner != nil {
		e.GroupID = owner.GroupID
	}
	if p := enemyai.NearestAlivePlayer(pos, w.playerSlice); p != nil {
		e.TargetPlayerID = p.ID
	}

	brain := enemyai.NewBrain(def, e, w.AbilityEngine)
	brain.ApplyOverfluxVariants(w.OverfluxState)
	brain.BoundsMinX = w.Level.EnemyBoundsMinX
	brain.BoundsMaxX = w.Level.EnemyBoundsMaxX
	brain.BoundsMinZ = w.Level.EnemyBoundsMinZ
	brain.BoundsMaxZ = w.Level.EnemyBoundsMaxZ

	w.Enemies = append(w.Enemies, e)
	w.Brains = append(w.Brains, brain)

	// Join the owner's active combat log session, if one is recording.
	if owner != nil {
		if session, ok := w.CombatLogs[enemySessionKey(owner)]; ok {
			session.AddParticipant(combatlog.ParticipantLog{
				EntityID: combatlog.FormatEnemyID(e.ID),
				Name:     e.DefName,
				Class:    "enemy",
			})
		}
	}

	slog.Info("add spawned", "def_name", defName, "id", id, "owner", ownerID, "zone_id", w.ZoneID)
	return true
}

// buildAddEnemy constructs the entity for a mid-fight summon: HP-scaled,
// already chasing, unleashed (adds hunt freely inside the room), and tagged
// with its summoner.
func buildAddEnemy(w *World, def *enemyai.EnemyDef, defName string, pos entity.Vec3, ownerID, id uint16) *entity.Enemy {
	e := entity.NewEnemy(id, def.MaxHealth, defName)
	if m := w.EnemyHPMult; m > 1 {
		e.MaxHealth *= m
		e.Health = e.MaxHealth
	}
	e.Alive = true
	e.State = entity.EnemyChase
	e.Position = pos
	e.LeashOrigin = pos
	e.LeashRadius = 0
	e.AggroRadius = 60
	e.SpawnedBy = ownerID
	return e
}

// enemyByID returns the enemy with the given ID, or nil.
func enemyByID(w *World, id uint16) *entity.Enemy {
	for _, e := range w.Enemies {
		if e != nil && e.ID == id {
			return e
		}
	}
	return nil
}

// sweepOrphanedAdds kills any spawned add whose owner is dead or missing.
func sweepOrphanedAdds(w *World) {
	for _, e := range w.Enemies {
		if e == nil || !e.Alive || e.SpawnedBy == 0 {
			continue
		}
		owner := enemyByID(w, e.SpawnedBy)
		if owner == nil || !owner.Alive {
			killAdd(e)
		}
	}
}

// killAdd marks a spawned add dead (despawn = dead; slices are append-only).
func killAdd(e *entity.Enemy) {
	e.Alive = false
	e.State = entity.EnemyDead
	e.Health = 0
	e.Velocity = entity.Vec3{}
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
