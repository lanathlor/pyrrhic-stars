package system

import (
	"codex-online/server/internal/combat"
	"codex-online/server/internal/combatlog"
	"codex-online/server/internal/entity"
)

// PhysicsSystem processes projectile movement and hit detection.
type PhysicsSystem struct{}

func (s *PhysicsSystem) Tick(w *World, dt float32) {
	if w.State != StateFight {
		return
	}

	tickPatternSpawns(w, dt)

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
			if checkProjectilePlayerHits(w, proj) {
				proj.Alive = false
			}
		}
		if proj.Alive {
			alive = append(alive, proj)
		}
	}
	w.Projectiles = alive
}

func tickPatternSpawns(w *World, dt float32) {
	if w.PatternEngine == nil {
		return
	}
	w.PatternEngine.Tick(dt, w.PatternRng)
	for _, req := range w.PatternEngine.DrainSpawns() {
		w.NextProjID++
		p := entity.NewProjectile(w.NextProjID, req.OwnerID, req.EnemyIdx,
			req.Position, req.Direction, req.Speed, req.Damage, req.Lifetime)
		p.VisualTag = req.VisualTag
		p.Acceleration = req.Acceleration
		p.AngularVelocity = req.AngularVelocity
		p.MaxSpeed = req.MaxSpeed
		w.Projectiles = append(w.Projectiles, p)
	}
}

func checkProjectilePlayerHits(w *World, proj *entity.Projectile) bool {
	for _, p := range w.Players {
		if !p.Alive {
			continue
		}
		if combat.CheckProjectileHit(proj.Position, p.Position, entity.ProjectileHitRadius+entity.PlayerHurtRadius) {
			applyProjectileHitToPlayer(w, proj, p)
			return true
		}
	}
	return false
}

func applyProjectileHitToPlayer(w *World, proj *entity.Projectile, p *entity.Player) {
	dealt := p.ApplyDamage(proj.Damage * w.EnemyDmgMult())
	if dealt <= 0 {
		return
	}
	// Add player to threat table of the specific enemy that fired
	if proj.EnemyIdx >= 0 && proj.EnemyIdx < len(w.Enemies) {
		if e := w.Enemies[proj.EnemyIdx]; e != nil && e.Alive {
			e.AddThreat(p.ID, dealt)
		}
	}
	w.DamageEvents = append(w.DamageEvents, combat.DamageEvent{
		TargetPeerID: p.ID,
		Amount:       dealt,
		HitPos:       proj.Position,
		SourceType:   combat.SourceEnemyRanged,
	})

	// Log projectile damage
	var enemyName string
	if proj.EnemyIdx >= 0 && proj.EnemyIdx < len(w.Enemies) {
		if e := w.Enemies[proj.EnemyIdx]; e != nil {
			enemyName = e.DefName
		}
	}
	w.logCombatEvent(combatlog.LogEntry{
		EventType:    combatlog.EventDamage,
		SourceEntity: "enemy_projectile",
		SourceClass:  enemyName,
		Target:       combatlog.FormatPlayerID(p.ID),
		AbilityID:    proj.VisualTag,
		Amount:       dealt,
		PosX:         proj.Position.X,
		PosY:         proj.Position.Y,
		PosZ:         proj.Position.Z,
	})

	// Log death if player died
	if !p.Alive {
		w.logCombatDeath(combatlog.FormatPlayerID(p.ID), "enemy_projectile", enemyName, proj.VisualTag)
	}
}
