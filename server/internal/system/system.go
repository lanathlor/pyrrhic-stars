package system

import (
	"math/rand/v2"

	"codex-online/server/internal/ability"
	"codex-online/server/internal/combat"
	"codex-online/server/internal/combatlog"
	"codex-online/server/internal/enemyai"
	"codex-online/server/internal/entity"
	"codex-online/server/internal/level"
)

// System is a discrete unit of game logic that runs once per tick.
type System interface {
	Tick(w *World, dt float32)
}

// NoOpSystem is a placeholder that does nothing. Used when a system is
// conditionally disabled (e.g. bots in non-dev mode).
type NoOpSystem struct{}

func (NoOpSystem) Tick(*World, float32) {}

// GameFlowState tracks the zone's game state.
type GameFlowState uint8

const (
	StateLobby   GameFlowState = iota
	StateSpawned               // players spawned, waiting for arena entry
	StateFight
	StateFightOver
)

// Client represents a connected player for sending messages.
type Client struct {
	PeerID   uint16
	Username string
	Send     func([]byte)
}

// InputMsg is a queued input from a client.
type InputMsg struct {
	PeerID  uint16
	Opcode  uint16
	Payload []byte
}

// GameFlowEvent is produced by the GameFlowSystem and consumed by the
// NetworkSystem at the end of the tick pipeline.
type GameFlowEvent struct {
	FlowType uint8
	Text     string
}

// World is the shared game state that systems read and write.
type World struct {
	// Identity
	ZoneID   string
	ZoneType uint8 // 0=OpenWorld, 1=Instanced
	RunID    string

	// Tick counter
	TickNum uint32

	// Game state
	State        GameFlowState
	BossDefeated bool
	GateStates   map[string]bool // gate_id → is_closed

	// Group scaling — multiplier applied to all enemy damage (1.0 = no scaling)
	EnemyDamageMult float32

	// Entities
	Players      map[uint16]*entity.Player
	Enemies      []*entity.Enemy
	Projectiles  []*entity.Projectile
	NPCs         []*entity.NPC
	HealingZones []*entity.HealingZone
	DamageLinks  []*entity.DamageLink
	NextProjID   uint32
	NextZoneID   uint32

	// AI brains (parallel to Enemies)
	Brains []enemyai.BrainTicker

	// Level geometry
	Level     *level.Level
	Obstacles []combat.Obstacle // Level.Obstacles + closed gate obstacles (rebuilt on gate change)

	// Ability engine
	AbilityEngine *ability.Engine

	// Per-player ability runners (commit→execute→cooldown lifecycle).
	AbilityRunners map[uint16]*ability.PlayerAbilityRunner

	// Networking
	Clients      map[uint16]*Client
	DamageEvents []combat.DamageEvent

	// Input queue (consumed by InputSystem)
	InputQueue []InputMsg

	// Game flow events (produced by GameFlowSystem, consumed by NetworkSystem)
	GameFlowEvents []GameFlowEvent

	// Callbacks (set by zone/gateway)
	OnPlayerRespawnHub func(peerID uint16)
	BroadcastToAll     func(msg []byte, excludePeerID uint16)

	// SendBuf is a pooled buffer for the broadcast path.
	// Reused every tick to avoid per-call allocations.
	// Capacity should be pre-allocated to the max expected message size (~4KB).
	SendBuf []byte

	// DamageBuf is a pooled buffer for damage event messages.
	// Reused every tick to avoid per-call allocations.
	DamageBuf []byte

	// GameFlowBuf is a pooled buffer for game flow event messages.
	// Reused every tick to avoid per-call allocations.
	GameFlowBuf []byte

	// LobbyBuf is a pooled buffer for lobby state messages.
	// Reused every tick to avoid per-call allocations.
	LobbyBuf []byte

	// Dev mode fields (active only when CODEX_DEV=1).
	DevMode            bool
	TimeScale          float32         // 0 = use default 1.0
	DebugRepeatAbility string          // empty = disabled
	DebugGodModePeers  map[uint16]bool // per-player god mode

	// TestMode enables defensive per-client copies so mock Send can inspect
	// messages. In production and benchmarks (TestMode=false), the pooled
	// buffer is passed directly to Send, matching real socket behavior.
	TestMode bool

	// Reusable player slices for the AI system. Built once per tick from
	// Players map to avoid per-brain map iteration allocations.
	playerSlice     []*entity.Player
	filteredPlayers []*entity.Player

	// Reusable buffer for enemy→Target interface conversion.
	enemyTargetBuf []entity.Target

	// Reusable ability tick context (avoids per-player allocation in CombatSystem).
	abilTickCtx ability.TickContext

	// Pattern engine for bullet-hell projectile patterns.
	PatternEngine *combat.PatternEngine
	PatternRng    *rand.Rand

	// Combat event logger. CombatLogSink is injected by Zone (NullSink if
	// disabled). CombatLogs tracks all active per-group sessions (keyed by
	// enemy GroupID or synthetic -enemyID for solo bosses).
	CombatLogSink combatlog.EventSink
	CombatLogs    map[int]*combatlog.EncounterSession

	// Pre-allocated spawn function for AISystem (avoids per-tick closure).
	spawnEnemyIdx   int
	spawnFn         func(pos, dir entity.Vec3, speed, damage, lifetime float32)
	commitPatternFn func(pattern *combat.PatternDef, abilityName string, origin, facing entity.Vec3)
}

// DeadGroupIDs returns a set of enemy GroupIDs where all members are dead.
func (w *World) DeadGroupIDs() map[int]bool {
	groups := make(map[int]int)
	dead := make(map[int]int)
	for _, e := range w.Enemies {
		if e.GroupID > 0 {
			groups[e.GroupID]++
			if !e.Alive {
				dead[e.GroupID]++
			}
		}
	}
	result := make(map[int]bool)
	for gid, total := range groups {
		if dead[gid] == total {
			result[gid] = true
		}
	}
	return result
}

// IsGateClosed returns whether a gate is currently closed.
func (w *World) IsGateClosed(gateID string) bool {
	return w.GateStates[gateID]
}

// AnyGateClosed returns true if any gate is currently closed.
func (w *World) AnyGateClosed() bool {
	for _, closed := range w.GateStates {
		if closed {
			return true
		}
	}
	return false
}

// ClosedGatePosition returns the position of a closed gate.
// Returns (Vec3{}, false) if the gate doesn't exist or is open.
func (w *World) ClosedGatePosition(gateID string) (entity.Vec3, bool) {
	if !w.GateStates[gateID] {
		return entity.Vec3{}, false
	}
	for i := range w.Level.Gates {
		if w.Level.Gates[i].ID == gateID {
			return w.Level.Gates[i].Position, true
		}
	}
	return entity.Vec3{}, false
}

// ClosedGateZ returns the Z position of the first closed gate, or 0 if none.
// Used by AI/combat for player-side filtering.
func (w *World) ClosedGateZ() (float32, bool) {
	for _, g := range w.Level.Gates {
		if w.GateStates[g.ID] {
			return g.Position.Z, true
		}
	}
	return 0, false
}

// RebuildObstacles reconstructs the combined obstacle list from level geometry
// plus any currently closed gates.
func (w *World) RebuildObstacles() {
	n := len(w.Level.Obstacles)
	for _, g := range w.Level.Gates {
		if w.GateStates[g.ID] {
			n++
		}
	}
	if cap(w.Obstacles) >= n {
		w.Obstacles = w.Obstacles[:len(w.Level.Obstacles)]
	} else {
		w.Obstacles = make([]combat.Obstacle, len(w.Level.Obstacles), n)
	}
	copy(w.Obstacles, w.Level.Obstacles)
	for i := range w.Level.Gates {
		if w.GateStates[w.Level.Gates[i].ID] {
			w.Obstacles = append(w.Obstacles, w.Level.Gates[i].ToObstacle())
		}
	}
}

// InitGateStates sets all gates to their default states and rebuilds obstacles.
func (w *World) InitGateStates() {
	w.GateStates = make(map[string]bool, len(w.Level.Gates))
	for _, g := range w.Level.Gates {
		w.GateStates[g.ID] = g.DefaultClosed
	}
	w.RebuildObstacles()
}

// EnemyDmgMult returns the enemy damage multiplier, defaulting to 1.0 if unset.
func (w *World) EnemyDmgMult() float32 {
	if w.EnemyDamageMult == 0 {
		return 1.0
	}
	return w.EnemyDamageMult
}

// FirstEnemy returns the first enemy or nil.
// Convenience for phase 0 where there's only one enemy.
func (w *World) FirstEnemy() *entity.Enemy {
	if len(w.Enemies) > 0 {
		return w.Enemies[0]
	}
	return nil
}

// AggroEnemy forces an enemy out of patrol into chase, targeting the given player.
// If the enemy belongs to a group, all group members also aggro and a combat log
// session is started for the group.
func (w *World) AggroEnemy(e *entity.Enemy, targetPeerID uint16) {
	if e.State != entity.EnemyPatrol {
		return
	}
	e.State = entity.EnemyChase
	e.ChaseTimer = 0
	e.TargetPlayerID = targetPeerID

	// Start combat logging on first aggro.
	if key := enemySessionKey(e); key != 0 {
		startGroupCombatLog(w, key)
	}

	// Group aggro: wake all allies in the same group
	if e.GroupID > 0 {
		for _, other := range w.Enemies {
			if other == e || !other.Alive || other.GroupID != e.GroupID {
				continue
			}
			if other.State == entity.EnemyPatrol {
				other.State = entity.EnemyChase
				other.ChaseTimer = 0
				other.TargetPlayerID = targetPeerID
			}
		}
	}
}

// SpawnEnemyProjectile creates an enemy-owned projectile in the world.
func (w *World) SpawnEnemyProjectile(enemyIdx int, pos, dir entity.Vec3, speed, damage, lifetime float32) {
	w.NextProjID++
	p := entity.NewProjectile(w.NextProjID, 0, enemyIdx, pos, dir, speed, damage, lifetime)
	w.Projectiles = append(w.Projectiles, p)
}

// SpawnPlayerProjectile creates a player-owned projectile in the world.
func (w *World) SpawnPlayerProjectile(ownerID uint16, pos, dir entity.Vec3, speed, damage, lifetime float32) {
	w.NextProjID++
	p := entity.NewProjectile(w.NextProjID, ownerID, -1, pos, dir, speed, damage, lifetime)
	w.Projectiles = append(w.Projectiles, p)
}

// enemiesToTargets converts an enemy slice to a Target interface slice,
// reusing the provided buffer to avoid allocation.
func enemiesToTargets(dst []entity.Target, enemies []*entity.Enemy) []entity.Target {
	dst = dst[:0]
	for _, e := range enemies {
		dst = append(dst, e)
	}
	return dst
}

// logCombatEvent emits a combat log entry to all active encounter sessions.
// No-op if no sessions are recording.
func (w *World) logCombatEvent(entry combatlog.LogEntry) {
	if len(w.CombatLogs) == 0 {
		return
	}
	var bossHP float32
	var bossPhase int
	if boss := findBoss(w); boss != nil {
		if boss.MaxHealth > 0 {
			bossHP = boss.Health / boss.MaxHealth
		}
		bossPhase = boss.Phase
	}
	for _, session := range w.CombatLogs {
		session.LogEvent(w.TickNum, bossHP, bossPhase, entry)
	}
}

// logPhaseChange checks and logs phase transitions for an enemy across all sessions.
func (w *World) logPhaseChange(enemy *entity.Enemy) {
	for _, session := range w.CombatLogs {
		session.CheckPhaseChange(w.TickNum, enemy.ID, enemy.Phase, enemy.Health/enemy.MaxHealth)
	}
}

// logCombatDeath emits a death event for the given entity.
func (w *World) logCombatDeath(target string, source string, sourceClass string, abilityID string) {
	w.logCombatEvent(combatlog.LogEntry{
		EventType:    combatlog.EventDeath,
		Target:       target,
		SourceEntity: source,
		SourceClass:  sourceClass,
		AbilityID:    abilityID,
	})
}
