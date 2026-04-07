package system

import (
	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
)

// PhysicsSystem processes projectile movement and hit detection.
type PhysicsSystem struct{}

func (s *PhysicsSystem) Tick(w *World, dt float32) {
	if w.State != StateFight {
		return
	}

	alive := w.Projectiles[:0]
	for _, proj := range w.Projectiles {
		proj.Tick(dt)
		if !proj.Alive {
			continue
		}

		// Kill projectile if it hits an obstacle
		if combat.ProjectileHitsObstacle(proj.Position, entity.ProjectileHitRadius, w.Level.Obstacles) {
			proj.Alive = false
			continue
		}

		// Check hit against players (enemy projectiles)
		if proj.OwnerID == 0 {
			for _, p := range w.Players {
				if !p.Alive {
					continue
				}
				if combat.CheckProjectileHit(proj.Position, p.Position, entity.ProjectileHitRadius+0.5) {
					dealt := p.ApplyDamage(proj.Damage)
					if dealt > 0 {
						// Add player to threat table of the specific enemy that fired
					if proj.EnemyIdx >= 0 && proj.EnemyIdx < len(w.Enemies) {
						if e := w.Enemies[proj.EnemyIdx]; e != nil && e.Alive {
							e.AddThreat(p.PeerID, dealt)
						}
					}
						w.DamageEvents = append(w.DamageEvents, combat.DamageEvent{
							TargetPeerID: p.PeerID,
							Amount:       dealt,
							HitPos:       proj.Position,
							SourceType:   combat.SourceEnemyRanged,
						})
					}
					proj.Alive = false
					break
				}
			}
		}
		if proj.Alive {
			alive = append(alive, proj)
		}
	}
	w.Projectiles = alive
}
