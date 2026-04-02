package system

import "codex-online/server/internal/combat"

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
			if p, ok := w.Players[evt.TargetPeerID]; ok {
				p.LastDamageTick = w.TickNum
			}
		}
		w.DamageEvents = append(w.DamageEvents, events...)
		w.Level.ClampEnemy(&e.Position)
		combat.PushOutOfObstacles(&e.Position, w.Level.Obstacles, w.Level.EnemyRadius)
	}
}
