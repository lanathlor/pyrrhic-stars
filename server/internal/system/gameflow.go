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

// clearTimeLimitTicks returns this instance's dungeon clear timer in ticks. The
// limit lives on the level (set per-instance from the Godot editor); when the
// level does not specify one it falls back to InstanceTimeLimitTicks.
func clearTimeLimitTicks(w *World) uint32 {
	if w.Level != nil && w.Level.ClearTimeSeconds > 0 {
		return uint32(w.Level.ClearTimeSeconds * 20)
	}
	return InstanceTimeLimitTicks
}

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
				Text:     strconv.Itoa(int(clearTimeLimitTicks(w) / 20)),
			})
			slog.Info("lobby fight start", "zone_id", w.ZoneID)
		}
	}
}

// bossNumOf returns the encounter number for a boss, defaulting legacy
// single-boss data (BossNum unset) to 1.
func bossNumOf(e *entity.Enemy) int {
	if e.BossNum > 0 {
		return e.BossNum
	}
	return 1
}

// defaultBossGateID is the gate assumed for legacy single-boss level data
// that predates per-boss gate IDs.
const defaultBossGateID = "boss_gate"

// bossGateIDOf returns the gate a boss controls, defaulting legacy data
// to defaultBossGateID.
func bossGateIDOf(e *entity.Enemy) string {
	if e.BossGateID != "" {
		return e.BossGateID
	}
	return defaultBossGateID
}

// bossEventName builds the boss-scoped gate trigger name, e.g. "boss1_dead".
func bossEventName(num int, suffix string) string {
	return "boss" + strconv.Itoa(num) + "_" + suffix
}

// finalBossNum returns the highest BossNum among all bosses (the boss whose
// death ends the run), or 0 if the zone has no boss.
func finalBossNum(w *World) int {
	final := 0
	for _, e := range w.Enemies {
		if e != nil && e.IsBoss {
			if n := bossNumOf(e); n > final {
				final = n
			}
		}
	}
	return final
}

// bossByGate returns the boss controlling the given gate, or nil.
func bossByGate(w *World, gateID string) *entity.Enemy {
	for _, e := range w.Enemies {
		if e != nil && e.IsBoss && bossGateIDOf(e) == gateID {
			return e
		}
	}
	return nil
}

// ActiveBoss returns the boss currently engaged in combat (alive and out of
// patrol/idle), or nil if no boss fight is running.
func ActiveBoss(w *World) *entity.Enemy {
	for _, e := range w.Enemies {
		if e != nil && e.IsBoss && e.Alive &&
			e.State != entity.EnemyPatrol && e.State != entity.EnemyIdle {
			return e
		}
	}
	return nil
}

// checkBossState detects boss aggro/reset per boss and emits boss flow events.
// Gate state changes are handled by processGateEvents which reacts to these events.
func checkBossState(w *World) {
	for _, e := range w.Enemies {
		if e == nil || !e.IsBoss || !e.Alive {
			continue
		}
		checkOneBossState(w, e)
	}
}

func checkOneBossState(w *World, boss *entity.Enemy) {
	num := bossNumOf(boss)
	gateID := bossGateIDOf(boss)

	bossInCombat := boss.State != entity.EnemyPatrol && boss.State != entity.EnemyIdle

	// Track whether we already emitted boss_activated this fight using gate state
	// (if the boss's gate is already closed, it was already activated).
	bossWasActivated := w.IsGateClosed(gateID)

	if bossInCombat && !bossWasActivated {
		w.GameFlowEvents = append(w.GameFlowEvents, GameFlowEvent{
			FlowType:  message.FlowBossActivated,
			GateEvent: bossEventName(num, "activated"),
		})
		slog.Info("boss activated", "zone_id", w.ZoneID, "boss", boss.DefName)
	}

	if bossWasActivated {
		// Check if any player is still in the boss room: same side of the
		// closed gate as the boss's spawn point. Dead players count — a
		// corpse in the room is a wipe (checkFightEnd owns that: it records
		// boss_win and resets), not an abandoned room. Respawning moves them
		// to a checkpoint outside, which is when this reset legitimately
		// fires. Counting only alive players mislabeled a solo in-room death
		// as "timeout" (live run 2026-07-14).
		gatePos, closed := w.ClosedGatePosition(gateID)
		if !closed {
			return
		}
		spawnSide := boss.LeashOrigin.Z < gatePos.Z
		anyPlayerInBossRoom := false
		for _, p := range w.Players {
			if (p.Position.Z < gatePos.Z) == spawnSide {
				anyPlayerInBossRoom = true
				break
			}
		}
		if !anyPlayerInBossRoom {
			// Reset boss — gate will open via bossN_reset → processGateEvents
			finalizeGroupCombatLog(w, enemySessionKey(boss), combatlog.OutcomeTimeout)
			despawnSpawnedAdds(w, boss.ID)
			bossIdx := enemyIndex(w, boss)
			if bossIdx >= 0 && bossIdx < len(w.Level.EnemySpawns) {
				boss.Reset(w.Level.EnemySpawns[bossIdx].Position, entity.EnemyPatrol)
			}
			w.Projectiles = nil
			slog.Info("boss reset — no players in boss room", "zone_id", w.ZoneID, "boss", boss.DefName)
			w.GameFlowEvents = append(w.GameFlowEvents, GameFlowEvent{
				FlowType:  message.FlowBossReset,
				GateEvent: bossEventName(num, "reset"),
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
		// Boss events carry an explicit boss-scoped name ("boss1_dead");
		// the legacy FlowEventName mapping is also tried so old level data
		// with un-scoped triggers ("boss_dead") keeps working.
		names := [2]string{
			w.GameFlowEvents[ei].GateEvent,
			level.FlowEventName[w.GameFlowEvents[ei].FlowType],
		}
		for _, eventName := range names {
			if eventName == "" {
				continue
			}
			for gi := range w.Level.Gates {
				if applyGateEvent(w, &w.Level.Gates[gi], eventName) {
					changed = true
				}
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

	// Remove threat for players on the far side of the gate, on the boss
	// this gate belongs to (falling back to the final boss for legacy gates).
	boss := bossByGate(w, g.ID)
	if boss == nil {
		boss = findBoss(w)
	}
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
	w.BossDeadHandled = nil
	for i, e := range w.Enemies {
		if i < len(w.Level.EnemySpawns) {
			e.Reset(w.Level.EnemySpawns[i].Position, entity.EnemyPatrol)
		}
	}
}

// ResetAliveEnemies returns alive enemies to patrol at their spawn point.
// Dead enemies are left dead — progress is preserved. Mid-fight spawned adds
// (no level spawn point) are despawned instead.
func ResetAliveEnemies(w *World) {
	w.Projectiles = nil
	despawnSpawnedAdds(w, 0)
	for i, e := range w.Enemies {
		if !e.Alive {
			continue
		}
		if i < len(w.Level.EnemySpawns) {
			e.Reset(w.Level.EnemySpawns[i].Position, entity.EnemyPatrol)
		}
	}
}

// despawnSpawnedAdds kills mid-fight spawned adds. ownerID 0 = all adds;
// otherwise only adds summoned by that enemy.
func despawnSpawnedAdds(w *World, ownerID uint16) {
	for _, e := range w.Enemies {
		if e == nil || !e.Alive || e.SpawnedBy == 0 {
			continue
		}
		if ownerID == 0 || e.SpawnedBy == ownerID {
			killAdd(e)
		}
	}
}

func checkFightEnd(w *World) {
	if checkBossDeaths(w) {
		return // run ended in victory: skip the wipe check
	}
	checkWipe(w)
}

// checkBossDeaths handles newly dead bosses. A non-final boss ends its
// encounter and unlocks the next section (via its bossN_dead gate trigger);
// only the final boss (highest BossNum) ends the run. Returns true when the
// final boss died this call.
func checkBossDeaths(w *World) bool {
	final := finalBossNum(w)
	for _, boss := range w.Enemies {
		if boss == nil || !boss.IsBoss || boss.State != entity.EnemyDead {
			continue
		}
		num := bossNumOf(boss)
		if w.BossDeadHandled[num] {
			continue
		}
		if w.BossDeadHandled == nil {
			w.BossDeadHandled = make(map[int]bool)
		}
		w.BossDeadHandled[num] = true

		if num != final {
			// Mid-run boss: unlock progression, keep the run going.
			finalizeGroupCombatLog(w, enemySessionKey(boss), combatlog.OutcomePlayerWin)
			w.Projectiles = nil
			w.GameFlowEvents = append(w.GameFlowEvents, GameFlowEvent{
				FlowType:  message.FlowMidBossDead,
				GateEvent: bossEventName(num, "dead"),
			})
			slog.Info("mid boss defeated", "zone_id", w.ZoneID, "boss", boss.DefName)
			continue
		}

		// Final boss → victory (guard: only trigger once via BossDefeated flag)
		if w.BossDefeated {
			continue
		}
		finalizeAllCombatLogs(w, combatlog.OutcomePlayerWin)
		w.BossDefeated = true
		w.Projectiles = nil
		w.GameFlowEvents = append(w.GameFlowEvents, GameFlowEvent{
			FlowType:  message.FlowBossDead,
			GateEvent: bossEventName(num, "dead"),
		})
		notifyBossDefeated(w)
		return true
	}
	return false
}

// notifyBossDefeated invokes the zone's run-completion callback with the
// participant list, overflux score, and timer verdict.
func notifyBossDefeated(w *World) {
	if w.OnBossDefeated == nil {
		return
	}
	peerIDs := make([]uint16, 0, len(w.Players))
	for id := range w.Players {
		peerIDs = append(peerIDs, id)
	}
	score := 0
	if w.OverfluxState != nil {
		score = w.OverfluxState.TotalScore
	}
	overTime := w.FightStartTick != 0 &&
		w.TickNum-w.FightStartTick > clearTimeLimitTicks(w)
	w.OnBossDefeated(peerIDs, score, overTime)
}

// checkWipe fires the all-dead flow once every player is down (guard: only
// trigger once via WipeHandled flag; reset when any player respawns in
// handleRespawnRequest).
func checkWipe(w *World) {
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

// deadBossNums collects the BossNums of all dead bosses (nil if none),
// used to unlock mid-run checkpoint spawns.
func deadBossNums(w *World) map[int]bool {
	var m map[int]bool
	for _, e := range w.Enemies {
		if e != nil && e.IsBoss && !e.Alive {
			if m == nil {
				m = make(map[int]bool)
			}
			m[bossNumOf(e)] = true
		}
	}
	return m
}

// respawnZoneState builds the spawn-condition state for respawn/unstuck paths.
func respawnZoneState(w *World) level.ZoneState {
	return level.ZoneState{
		BossDefeated: w.BossDefeated,
		DeadGroupIDs: w.DeadGroupIDs(),
		DeadBossNums: deadBossNums(w),
	}
}

// combatGateSealed reports whether any combat gate — one that closes on a
// fight trigger, i.e. a boss room seal — is currently closed. Progression
// gates (open_on only, like the decline gate) never block respawn.
func combatGateSealed(w *World) bool {
	for i := range w.Level.Gates {
		g := &w.Level.Gates[i]
		if len(g.CloseOn) > 0 && w.GateStates[g.ID] {
			return true
		}
	}
	return false
}

// SpawnPlayers initializes all players at spawn points.
func SpawnPlayers(w *World) {
	deadGroups := w.DeadGroupIDs()
	deadBosses := deadBossNums(w)
	idx := 0
	for _, p := range w.Players {
		spawnPos := pickSpawnPoint(w.Level.PlayerSpawns, level.ZoneState{BossDefeated: w.BossDefeated, DeadGroupIDs: deadGroups, DeadBossNums: deadBosses}, idx)
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
	spawnPos := pickSpawnPoint(w.Level.PlayerSpawns, level.ZoneState{BossDefeated: w.BossDefeated, DeadGroupIDs: deadGroups, DeadBossNums: deadBossNums(w)}, idx)
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

// findBoss returns the FINAL boss enemy (highest BossNum) or nil. Uses the
// cached Boss pointer on World when available, falling back to a linear scan.
func findBoss(w *World) *entity.Enemy {
	if w.Boss != nil {
		return w.Boss
	}
	for _, e := range w.Enemies {
		if e.IsBoss && (w.Boss == nil || bossNumOf(e) > bossNumOf(w.Boss)) {
			w.Boss = e
		}
	}
	return w.Boss
}

// findBossIndex returns the index of findBoss's result in the Enemies slice.
func findBossIndex(w *World) int {
	return enemyIndex(w, findBoss(w))
}

// enemyIndex returns the index of the given enemy pointer in w.Enemies, or -1.
func enemyIndex(w *World, target *entity.Enemy) int {
	if target == nil {
		return -1
	}
	for i, e := range w.Enemies {
		if e == target {
			return i
		}
	}
	return -1
}
