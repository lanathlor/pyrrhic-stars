package system

import (
	"log/slog"

	"codex-online/server/internal/entity"
	"codex-online/server/internal/message"
)

// GameFlowSystem manages the arena FSM: Lobby -> Spawned -> Fight -> FightOver.
// During Fight, trash mobs activate first; once all trash is dead, the boss activates.
type GameFlowSystem struct{}

func (s *GameFlowSystem) Tick(w *World, dt float32) {
	if w.ZoneType == 0 { // Hub
		return
	}

	switch w.State {
	case StateLobby:
		tickLobby(w)
	case StateSpawned:
		tickSpawned(w)
	case StateFight:
		checkBossGate(w)
		checkFightEnd(w)
	case StateFightOver:
		tickFightOver(w, dt)
	}
}

func tickLobby(w *World) {
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

	SpawnPlayers(w)
	w.State = StateSpawned
	slog.Info("all players ready, spawning", "zone_id", w.ZoneID, "players", len(w.Players))
	w.GameFlowEvents = append(w.GameFlowEvents, GameFlowEvent{
		FlowType: message.FlowSpawnPlayers,
	})
}

func tickSpawned(w *World) {
	// Instance is already initialized — transition to fight immediately
	// when any player is present.
	if len(w.Players) > 0 {
		w.State = StateFight
		slog.Info("fight active", "zone_id", w.ZoneID, "players", len(w.Players))
		w.GameFlowEvents = append(w.GameFlowEvents, GameFlowEvent{
			FlowType: message.FlowFightStart,
		})
	}
}

// checkBossGate manages the boss room gate:
// - Close gate when boss leaves patrol (aggro)
// - Reset boss and open gate when no alive player is in the boss room
func checkBossGate(w *World) {
	boss := findBoss(w)
	if boss == nil || !boss.Alive {
		return
	}

	bossInCombat := boss.State != entity.EnemyPatrol && boss.State != entity.EnemyIdle

	if bossInCombat && !w.BossGateActive {
		// Boss just aggroed — close the gate
		w.BossGateActive = true
		// Push any players near the gate into the boss room
		for _, p := range w.Players {
			if p.Alive && p.Position.Z >= w.Level.BossRoomEntryZ-2.0 && p.Position.Z <= w.Level.BossRoomEntryZ+2.0 {
				p.Position.Z = w.Level.BossRoomEntryZ - 3.0
			}
		}
		// Remove threat for players stuck outside the boss room
		for _, p := range w.Players {
			if p.Position.Z >= w.Level.BossRoomEntryZ {
				delete(boss.ThreatTable, p.PeerID)
			}
		}
		w.GameFlowEvents = append(w.GameFlowEvents, GameFlowEvent{
			FlowType: message.FlowBossActivated,
		})
		slog.Info("boss gate closed", "zone_id", w.ZoneID)
	}

	if w.BossGateActive {
		// Check if any alive player is in the boss room (Z < BossRoomEntryZ)
		anyPlayerInBossRoom := false
		for _, p := range w.Players {
			if p.Alive && p.Position.Z < w.Level.BossRoomEntryZ {
				anyPlayerInBossRoom = true
				break
			}
		}
		if !anyPlayerInBossRoom {
			// Reset boss and open gate
			bossIdx := findBossIndex(w)
			if bossIdx >= 0 && bossIdx < len(w.Level.EnemySpawns) {
				boss.Reset(w.Level.EnemySpawns[bossIdx].Position, entity.EnemyPatrol)
			}
			w.BossGateActive = false
			w.Projectiles = nil
			slog.Info("boss reset — no players in boss room", "zone_id", w.ZoneID)
			w.GameFlowEvents = append(w.GameFlowEvents, GameFlowEvent{
				FlowType: message.FlowBossReset,
			})
		}
	}
}

// InitInstance activates all enemies in patrol state. Called once when the
// arena zone is created — enemies are alive and patrolling from the start.
func InitInstance(w *World) {
	w.Projectiles = nil
	for i, e := range w.Enemies {
		if i < len(w.Level.EnemySpawns) {
			e.Reset(w.Level.EnemySpawns[i].Position, entity.EnemyPatrol)
		}
	}
}

// ResetAliveEnemies returns alive enemies to patrol at their spawn point.
// Dead enemies are left dead — progress is preserved.
func ResetAliveEnemies(w *World) {
	w.Projectiles = nil
	for i, e := range w.Enemies {
		if !e.Alive {
			continue
		}
		if i < len(w.Level.EnemySpawns) {
			e.Reset(w.Level.EnemySpawns[i].Position, entity.EnemyPatrol)
		}
	}
}

func checkFightEnd(w *World) {
	// Boss dead → victory
	boss := findBoss(w)
	if boss != nil && boss.State == entity.EnemyDead {
		w.State = StateFightOver
		w.BossDefeated = true
		w.Projectiles = nil
		w.GameFlowEvents = append(w.GameFlowEvents, GameFlowEvent{
			FlowType: message.FlowBossDead,
		})
		return
	}

	// Check all players dead → wipe
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
		w.BossGateActive = false
		// Reset alive enemies to patrol, keep dead ones dead
		for i, e := range w.Enemies {
			if !e.Alive {
				continue
			}
			if i < len(w.Level.EnemySpawns) {
				e.Reset(w.Level.EnemySpawns[i].Position, entity.EnemyPatrol)
			}
		}
		w.GameFlowEvents = append(w.GameFlowEvents, GameFlowEvent{
			FlowType: message.FlowAllDead,
		})
	}
}

func tickFightOver(w *World, _ float32) {
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
	w.State = StateSpawned
	w.Projectiles = nil

	// Reset players to warmup
	for _, p := range w.Players {
		p.Ready = false
		p.Alive = true
		p.Health = p.MaxHealth
		p.State = entity.PlayerStateMove
		p.Position = entity.Vec3{X: 0, Y: 0.1, Z: 48.0}
		p.Velocity = entity.Vec3{}
	}

	// Reset alive enemies to patrol — dead ones stay dead
	ResetAliveEnemies(w)

	w.GameFlowEvents = append(w.GameFlowEvents, GameFlowEvent{
		FlowType: message.FlowReturnLobby,
	})
}

// SpawnPlayers initializes all players at spawn points.
func SpawnPlayers(w *World) {
	idx := 0
	for _, p := range w.Players {
		spawnPos := w.Level.PlayerSpawns[idx%len(w.Level.PlayerSpawns)]
		p.Position = spawnPos
		p.RotationY = w.Level.SpawnYaw
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
func SpawnPlayer(w *World, peerID uint16) {
	p, ok := w.Players[peerID]
	if !ok {
		return
	}
	idx := len(w.Players) - 1
	spawnPos := w.Level.PlayerSpawns[idx%len(w.Level.PlayerSpawns)]
	p.Position = spawnPos
	p.RotationY = w.Level.SpawnYaw
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

// findBoss returns the boss enemy or nil.
func findBoss(w *World) *entity.Enemy {
	for _, e := range w.Enemies {
		if e.IsBoss {
			return e
		}
	}
	return nil
}

// findBossIndex returns the index of the boss in the Enemies slice.
func findBossIndex(w *World) int {
	for i, e := range w.Enemies {
		if e.IsBoss {
			return i
		}
	}
	return -1
}
