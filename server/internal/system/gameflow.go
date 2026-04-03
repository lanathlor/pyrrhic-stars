package system

import (
	"log/slog"

	"codex-online/server/internal/entity"
	"codex-online/server/internal/message"
)

// GameFlowSystem manages the arena FSM: Lobby -> Spawned -> Fight -> FightOver.
// It does NOT run AI or physics; those are handled by subsequent systems
// in the pipeline. It only manages state transitions and fight end conditions.
type GameFlowSystem struct{}

func (s *GameFlowSystem) Tick(w *World, dt float32) {
	if w.ZoneType == 0 { // Hub
		// Hub: no game flow to manage.
		return
	}

	switch w.State {
	case StateLobby:
		tickLobby(w)
	case StateSpawned:
		tickSpawned(w)
	case StateFight:
		checkFightEnd(w)
	case StateFightOver:
		tickFightOver(w, dt)
	}
}

func tickLobby(w *World) {
	// Check if all players are ready (need at least 1)
	if len(w.Players) < 1 {
		return
	}
	allReady := true
	for _, p := range w.Players {
		if !p.Ready {
			allReady = false
			break
		}
	}
	if !allReady {
		return
	}

	// Spawn players — they'll walk into the arena to trigger fight
	spawnPlayers(w)
	w.State = StateSpawned
	slog.Info("all players ready, spawning", "zone_id", w.ZoneID, "players", len(w.Players))
	w.GameFlowEvents = append(w.GameFlowEvents, GameFlowEvent{
		FlowType: message.FlowSpawnPlayers,
	})
}

func tickSpawned(w *World) {
	// Check if any player crossed into the arena
	if checkPlayerArenaEntry(w) {
		startFight(w)
	}
}

func startFight(w *World) {
	if w.State != StateSpawned {
		return
	}
	w.State = StateFight

	// Reset enemy
	enemy := w.FirstEnemy()
	if enemy != nil {
		enemy.Reset(w.Level.EnemySpawn)
	}
	w.Projectiles = nil

	slog.Info("fight started", "zone_id", w.ZoneID, "players", len(w.Players))
	w.GameFlowEvents = append(w.GameFlowEvents, GameFlowEvent{
		FlowType: message.FlowFightStart,
	})
}

func checkFightEnd(w *World) {
	enemy := w.FirstEnemy()
	if enemy != nil && enemy.State == entity.EnemyDead {
		w.State = StateFightOver
		w.BossDefeated = true
		w.Projectiles = nil
		w.GameFlowEvents = append(w.GameFlowEvents, GameFlowEvent{
			FlowType: message.FlowBossDead,
		})
		return
	}

	allDead := true
	for _, p := range w.Players {
		if p.Alive {
			allDead = false
			break
		}
	}
	if allDead && len(w.Players) > 0 {
		w.State = StateFightOver
		w.BossDefeated = false
		w.Projectiles = nil
		if enemy != nil {
			enemy.Reset(w.Level.EnemySpawn)
		}
		w.GameFlowEvents = append(w.GameFlowEvents, GameFlowEvent{
			FlowType: message.FlowAllDead,
		})
	}
}

func tickFightOver(w *World, dt float32) {
	for _, p := range w.Players {
		if p.Alive {
			p.FireCooldown -= dt
		}
	}

	// After a wipe, transition back to lobby once all players have respawned
	if !w.BossDefeated {
		allAlive := true
		for _, p := range w.Players {
			if !p.Alive {
				allAlive = false
				break
			}
		}
		if allAlive && len(w.Players) > 0 {
			returnToLobby(w)
		}
	}
}

func returnToLobby(w *World) {
	w.State = StateLobby
	w.Projectiles = nil
	for _, p := range w.Players {
		p.Ready = false
		p.Alive = true
		p.Health = p.MaxHealth
		p.State = entity.PlayerStateMove
		p.Position = entity.Vec3{X: 0, Y: 0.1, Z: 20.0}
		p.Velocity = entity.Vec3{}
	}
	w.GameFlowEvents = append(w.GameFlowEvents, GameFlowEvent{
		FlowType: message.FlowReturnLobby,
	})
}

func checkPlayerArenaEntry(w *World) bool {
	if w.Level.ArenaEntryZ <= 0 {
		return false
	}
	for _, p := range w.Players {
		if p.Alive && p.Position.Z < w.Level.ArenaEntryZ {
			return true
		}
	}
	return false
}

// SpawnPlayers initializes all players at spawn points. Exported for use
// by zone.AddClient catch-up logic.
func SpawnPlayers(w *World) {
	spawnPlayers(w)
}

func spawnPlayers(w *World) {
	idx := 0
	for _, p := range w.Players {
		spawnPos := w.Level.PlayerSpawns[idx%len(w.Level.PlayerSpawns)]
		p.Position = spawnPos
		p.Health = p.MaxHealth
		p.Alive = true
		p.State = entity.PlayerStateMove
		p.Velocity = entity.Vec3{}
		p.IsRolling = false
		p.RollCooldown = 0
		p.Invincible = false
		p.InvincibleTimer = 0
		p.SpawnTick = w.TickNum
		idx++
	}
}

// SpawnPlayer initializes a single player at the next available spawn point.
// Exported for use by zone.AddClient catch-up logic.
func SpawnPlayer(w *World, peerID uint16) {
	p, ok := w.Players[peerID]
	if !ok {
		return
	}
	idx := len(w.Players) - 1
	spawnPos := w.Level.PlayerSpawns[idx%len(w.Level.PlayerSpawns)]
	p.Position = spawnPos
	p.Health = p.MaxHealth
	p.Alive = true
	p.State = entity.PlayerStateMove
	p.Velocity = entity.Vec3{}
	p.IsRolling = false
	p.RollCooldown = 0
	p.Invincible = false
	p.InvincibleTimer = 0
	p.SpawnTick = w.TickNum
}
