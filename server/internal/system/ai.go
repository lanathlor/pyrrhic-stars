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
		events := w.Brains[i].Tick(dt, w.Players, w.Level.Obstacles, w.SpawnProjectile)
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
