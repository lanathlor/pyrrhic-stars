package system

import (
	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
)

// AISystem ticks enemy brains during the fight state.
type AISystem struct{}

func (s *AISystem) Tick(w *World, dt float32) {
	if w.State != StateFight {
		return
	}
	for i, e := range w.Enemies {
		if e == nil || !e.Alive || i >= len(w.Brains) {
			continue
		}
		idx := i
		spawnFn := func(pos, dir entity.Vec3, speed, damage, lifetime float32) {
			w.SpawnEnemyProjectile(idx, pos, dir, speed, damage, lifetime)
		}
		// When the boss gate is active, enemies only see players on
		// their side of the gate (Z < BossRoomEntryZ = boss room).
		visiblePlayers := w.Players
		if w.BossGateActive {
			visiblePlayers = playersOnSameSide(w.Players, e.Position.Z, w.Level.BossRoomEntryZ)
		}
		events := w.Brains[i].Tick(dt, visiblePlayers, w.Level.Obstacles, spawnFn)
		for _, evt := range events {
			if _, ok := w.Players[evt.TargetPeerID]; ok {
				e.AddThreat(evt.TargetPeerID, evt.Amount)
			}
		}
		w.DamageEvents = append(w.DamageEvents, events...)
		w.Level.ClampEnemy(&e.Position)
		combat.PushOutOfObstacles(&e.Position, w.Level.Obstacles, w.Level.EnemyRadius)
	}

	// Group aggro propagation: if any mob in a group is chasing, wake all
	// patrol members of that group.
	propagateGroupAggro(w)
}

// playersOnSameSide returns only the players that are on the same side of
// gateZ as the given enemyZ. Enemies in the boss room (Z < gateZ) only see
// players in the boss room, and vice-versa.
func playersOnSameSide(players map[uint16]*entity.Player, enemyZ, gateZ float32) map[uint16]*entity.Player {
	enemyInBossRoom := enemyZ < gateZ
	filtered := make(map[uint16]*entity.Player, len(players))
	for id, p := range players {
		playerInBossRoom := p.Position.Z < gateZ
		if playerInBossRoom == enemyInBossRoom {
			filtered[id] = p
		}
	}
	return filtered
}

// propagateGroupAggro ensures that if any mob in a group has left patrol
// (e.g. due to proximity aggro), all other patrol members in the same group
// also switch to chase.
func propagateGroupAggro(w *World) {
	// Collect which groups have an aggroed member and their target
	type groupAggro struct {
		targetPeerID uint16
	}
	aggroed := map[int]groupAggro{}
	for _, e := range w.Enemies {
		if e == nil || !e.Alive || e.GroupID == 0 {
			continue
		}
		if e.State != entity.EnemyPatrol {
			aggroed[e.GroupID] = groupAggro{targetPeerID: e.TargetPlayerID}
		}
	}

	// Wake any remaining patrol mobs in aggroed groups
	for _, e := range w.Enemies {
		if e == nil || !e.Alive || e.GroupID == 0 {
			continue
		}
		if e.State == entity.EnemyPatrol {
			if ga, ok := aggroed[e.GroupID]; ok {
				e.State = entity.EnemyChase
				e.ChaseTimer = 0
				e.TargetPlayerID = ga.targetPeerID
			}
		}
	}
}
