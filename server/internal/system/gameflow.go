package system

import (
	"log/slog"
	"slices"
	"strconv"

	"codex-online/server/internal/combatlog"
	"codex-online/server/internal/entity"
	"codex-online/server/internal/level"
	"codex-online/server/internal/message"
)

// GameFlowSystem detects combat events (boss activation, boss death, wipe)
// and manages data-driven gates. Runs every tick for all zone types.
type GameFlowSystem struct{}

func (s *GameFlowSystem) Tick(w *World, _ float32) {
	checkLobbyReady(w)
	if len(w.Enemies) > 0 {
		checkBossState(w)
		checkFightEnd(w)
	}
	processGateEvents(w)
}

// LobbyCountdownTicks is the number of ticks to count down after all players
// are ready before starting the fight (5 seconds at 20Hz).
const LobbyCountdownTicks int32 = 100

// InstanceTimeLimitTicks is the dungeon completion timer (5 minutes at 20Hz).
// The timer starts at fight start (ready gate gone). Defeating the boss after it
// expires is an over-time finish: reduced scrip and no watermark progress.
const InstanceTimeLimitTicks uint32 = 300 * 20

// InstanceTimeLimitSeconds is the time limit in whole seconds, sent to the
// client in the FlowFightStart event so the HUD count-down matches the server.
const InstanceTimeLimitSeconds = InstanceTimeLimitTicks / 20

// checkLobbyReady manages the lobby ready-up phase for instanced zones.
// When all human players are ready, starts a countdown. When the countdown
// expires, emits FlowFightStart and deactivates the lobby.
func checkLobbyReady(w *World) {
	if !w.LobbyActive {
		return
	}

	humanCount, readyCount := 0, 0
	for _, p := range w.Players {
		if entity.IsBotID(p.ID) {
			continue
		}
		humanCount++
		if p.Ready {
			readyCount++
		}
	}
	if humanCount == 0 {
		return
	}

	allReady := readyCount == humanCount

	if allReady && w.LobbyCountdown == 0 {
		w.LobbyCountdown = LobbyCountdownTicks
		slog.Info("lobby countdown started", "zone_id", w.ZoneID, "players", humanCount)
	}

	if !allReady && w.LobbyCountdown > 0 {
		w.LobbyCountdown = 0
		slog.Info("lobby countdown cancelled", "zone_id", w.ZoneID, "ready", readyCount, "total", humanCount)
	}

	if w.LobbyCountdown > 0 {
		w.LobbyCountdown--
		if w.LobbyCountdown == 0 {
			w.LobbyActive = false
			w.FightStartTick = w.TickNum
			for _, p := range w.Players {
				p.Ready = false
			}
			w.GameFlowEvents = append(w.GameFlowEvents, GameFlowEvent{
				FlowType: message.FlowFightStart,
				Text:     strconv.Itoa(int(InstanceTimeLimitSeconds)),
			})
			slog.Info("lobby fight start", "zone_id", w.ZoneID)
		}
	}
}

// checkBossState detects boss aggro/reset and emits boss flow events.
// Gate state changes are handled by processGateEvents which reacts to these events.
func checkBossState(w *World) {
	boss := findBoss(w)
	if boss == nil || !boss.Alive {
		return
	}

	bossInCombat := boss.State != entity.EnemyPatrol && boss.State != entity.EnemyIdle

	// Track whether we already emitted boss_activated this fight using gate state
	// (if the boss gate is already closed, boss was already activated).
	bossWasActivated := w.IsGateClosed("boss_gate")

	if bossInCombat && !bossWasActivated {
		w.GameFlowEvents = append(w.GameFlowEvents, GameFlowEvent{
			FlowType: message.FlowBossActivated,
		})
		slog.Info("boss activated", "zone_id", w.ZoneID)
	}

	if bossWasActivated {
		// Check if any alive player is in the boss room (past the gate)
		gateZ, _ := w.ClosedGatePosition("boss_gate")
		anyPlayerInBossRoom := false
		for _, p := range w.Players {
			if p.Alive && p.Position.Z < gateZ.Z {
				anyPlayerInBossRoom = true
				break
			}
		}
		if !anyPlayerInBossRoom {
			// Reset boss — gate will open via FlowBossReset → processGateEvents
			finalizeGroupCombatLog(w, boss.GroupID, combatlog.OutcomeTimeout)
			bossIdx := findBossIndex(w)
			if bossIdx >= 0 && bossIdx < len(w.Level.EnemySpawns) {
				boss.Reset(w.Level.EnemySpawns[bossIdx].Position, entity.EnemyPatrol)
			}
			w.Projectiles = nil
			slog.Info("boss reset — no players in boss room", "zone_id", w.ZoneID)
			w.GameFlowEvents = append(w.GameFlowEvents, GameFlowEvent{
				FlowType: message.FlowBossReset,
			})
		}
	}
}

// processGateEvents checks all flow events emitted this tick and opens/closes
// gates whose triggers match. Emits FlowGateClose/FlowGateOpen for each change.
func processGateEvents(w *World) {
	if len(w.Level.Gates) == 0 {
		return
	}

	// Snapshot the flow events emitted so far this tick (before we add gate events).
	n := len(w.GameFlowEvents)
	if n == 0 {
		return
	}

	changed := false
	for ei := range n {
		eventName := level.FlowEventName[w.GameFlowEvents[ei].FlowType]
		if eventName == "" {
			continue
		}
		for gi := range w.Level.Gates {
			if applyGateEvent(w, &w.Level.Gates[gi], eventName) {
				changed = true
			}
		}
	}

	if changed {
		w.RebuildObstacles()
	}
}

// applyGateEvent checks if eventName should close or open a gate, applies the
// state change, emits flow events, and returns true if the gate changed state.
func applyGateEvent(w *World, g *level.GateDef, eventName string) bool {
	if !w.GateStates[g.ID] && slices.Contains(g.CloseOn, eventName) {
		w.GateStates[g.ID] = true
		w.GameFlowEvents = append(w.GameFlowEvents, GameFlowEvent{
			FlowType: message.FlowGateClose,
			Text:     g.ID,
		})
		slog.Info("gate closed", "gate_id", g.ID, "trigger", eventName, "zone_id", w.ZoneID)
		pushPlayersOnGateClose(w, g)
		return true
	}
	if w.GateStates[g.ID] && slices.Contains(g.OpenOn, eventName) {
		w.GateStates[g.ID] = false
		w.GameFlowEvents = append(w.GameFlowEvents, GameFlowEvent{
			FlowType: message.FlowGateOpen,
			Text:     g.ID,
		})
		slog.Info("gate opened", "gate_id", g.ID, "trigger", eventName, "zone_id", w.ZoneID)
		return true
	}
	return false
}

// pushPlayersOnGateClose pushes players overlapping a gate when it closes,
// and removes boss threat for players on the wrong side.
func pushPlayersOnGateClose(w *World, g *level.GateDef) {
	if g.PushAxis == "" {
		return
	}

	for _, p := range w.Players {
		if !p.Alive {
			continue
		}
		var playerPos, gatePos, halfExt float32
		switch g.PushAxis {
		case "x":
			playerPos, gatePos, halfExt = p.Position.X, g.Position.X, g.HalfExtents.X
		case "z":
			playerPos, gatePos, halfExt = p.Position.Z, g.Position.Z, g.HalfExtents.Z
		default:
			continue
		}

		// Push players within the gate's thickness range
		if playerPos >= gatePos-halfExt-2.0 && playerPos <= gatePos+halfExt+2.0 {
			switch g.PushAxis {
			case "x":
				p.Position.X = g.Position.X + g.PushOffset
			case "z":
				p.Position.Z = g.Position.Z + g.PushOffset
			}
		}
	}

	// Remove boss threat for players on the far side of the gate
	boss := findBoss(w)
	if boss == nil {
		return
	}
	for _, p := range w.Players {
		var playerPos, gatePos float32
		switch g.PushAxis {
		case "x":
			playerPos, gatePos = p.Position.X, g.Position.X
		case "z":
			playerPos, gatePos = p.Position.Z, g.Position.Z
		}
		// Players on the opposite side of the gate from the push direction lose threat.
		onFarSide := (g.PushOffset < 0 && playerPos >= gatePos) ||
			(g.PushOffset > 0 && playerPos <= gatePos)
		if onFarSide {
			delete(boss.ThreatTable, p.ID)
		}
	}
}

// InitInstance activates all enemies in patrol state. Called once when the
// instanced zone is created: enemies are alive and patrolling from the start.
func InitInstance(w *World) {
	w.Projectiles = nil
	w.InitGateStates()
	w.LobbyActive = true
	w.LobbyCountdown = 0
	w.FightStartTick = 0
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
	// Boss dead → victory (guard: only trigger once via BossDefeated flag)
	boss := findBoss(w)
	if boss != nil && boss.State == entity.EnemyDead && !w.BossDefeated {
		finalizeAllCombatLogs(w, combatlog.OutcomePlayerWin)
		w.BossDefeated = true
		w.Projectiles = nil
		w.GameFlowEvents = append(w.GameFlowEvents, GameFlowEvent{
			FlowType: message.FlowBossDead,
		})
		if w.OnBossDefeated != nil {
			peerIDs := make([]uint16, 0, len(w.Players))
			for id := range w.Players {
				peerIDs = append(peerIDs, id)
			}
			score := 0
			if w.OverfluxState != nil {
				score = w.OverfluxState.TotalScore
			}
			overTime := w.FightStartTick != 0 &&
				w.TickNum-w.FightStartTick > InstanceTimeLimitTicks
			w.OnBossDefeated(peerIDs, score, overTime)
		}
		return
	}

	// All players dead → wipe (guard: only trigger once via WipeHandled flag;
	// reset when any player respawns in handleRespawnRequest).
	if w.WipeHandled {
		return
	}
	allDead := true
	humanCount := 0
	for _, p := range w.Players {
		if !entity.IsBotID(p.ID) {
			humanCount++
		}
		if p.Alive {
			allDead = false
			break
		}
	}
	if allDead && humanCount > 0 {
		finalizeAllCombatLogs(w, combatlog.OutcomeBossWin)
		w.WipeHandled = true
		w.Projectiles = nil
		ResetAliveEnemies(w)
		// Emit FlowAllDead; processGateEvents will open gates that have
		// "all_dead" in their open_on list, sending FlowGateOpen to clients.
		w.GameFlowEvents = append(w.GameFlowEvents, GameFlowEvent{
			FlowType: message.FlowAllDead,
		})
	}
}

// pickSpawnPoint selects the best spawn point for a player given the current zone state.
// It picks the highest-progression checkpoint whose condition is satisfied,
// then round-robins among spawns at that tier.
func pickSpawnPoint(spawns []level.PlayerSpawn, state level.ZoneState, idx int) entity.Vec3 {
	if len(spawns) == 0 {
		return entity.Vec3{Y: 0.1}
	}
	// Find the highest priority among satisfied conditions
	bestPriority := -1
	for _, s := range spawns {
		if level.EvalCondition(s.Condition, state) {
			if p := level.ConditionPriority(s.Condition); p > bestPriority {
				bestPriority = p
			}
		}
	}
	if bestPriority < 0 {
		// Nothing satisfied — fall back to first spawn
		return spawns[0].Position
	}
	// Count eligible spawns at best tier, then index without allocating.
	count := 0
	for _, s := range spawns {
		if level.EvalCondition(s.Condition, state) &&
			level.ConditionPriority(s.Condition) == bestPriority {
			count++
		}
	}
	target := idx % count
	for _, s := range spawns {
		if level.EvalCondition(s.Condition, state) &&
			level.ConditionPriority(s.Condition) == bestPriority {
			if target == 0 {
				return s.Position
			}
			target--
		}
	}
	return spawns[0].Position
}

// SpawnPlayers initializes all players at spawn points.
func SpawnPlayers(w *World) {
	deadGroups := w.DeadGroupIDs()
	idx := 0
	for _, p := range w.Players {
		spawnPos := pickSpawnPoint(w.Level.PlayerSpawns, level.ZoneState{BossDefeated: w.BossDefeated, DeadGroupIDs: deadGroups}, idx)
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
	deadGroups := w.DeadGroupIDs()
	spawnPos := pickSpawnPoint(w.Level.PlayerSpawns, level.ZoneState{BossDefeated: w.BossDefeated, DeadGroupIDs: deadGroups}, idx)
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

// findBoss returns the boss enemy or nil. Uses the cached Boss pointer on
// World when available, falling back to a linear scan.
func findBoss(w *World) *entity.Enemy {
	if w.Boss != nil {
		return w.Boss
	}
	for _, e := range w.Enemies {
		if e.IsBoss {
			w.Boss = e
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
