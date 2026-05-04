package system

import (
	"codex-online/server/internal/ability"
	"codex-online/server/internal/combat"
	"codex-online/server/internal/enemyai"
	"codex-online/server/internal/entity"
	"codex-online/server/internal/level"
)

// System is a discrete unit of game logic that runs once per tick.
type System interface {
	Tick(w *World, dt float32)
}

// GameFlowState tracks the zone's game state.
type GameFlowState uint8

const (
	StateLobby   GameFlowState = iota
	StateSpawned // players spawned, waiting for arena entry
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

	// Tick counter
	TickNum uint32

	// Game state
	State        GameFlowState
	BossDefeated bool
	BossGateActive bool // true when boss room is sealed (boss is fighting)

	// Entities
	Players     map[uint16]*entity.Player
	Enemies     []*entity.Enemy
	Projectiles []*entity.Projectile
	NPCs        []*entity.NPC
	NextProjID  uint32

	// AI brains (parallel to Enemies)
	Brains []enemyai.BrainTicker

	// Level geometry
	Level *level.Level

	// Ability engine
	AbilityEngine *ability.Engine

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

	// Pre-allocated spawn function for AISystem (avoids per-tick closure).
	spawnEnemyIdx int
	spawnFn       func(pos, dir entity.Vec3, speed, damage, lifetime float32)
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
// If the enemy belongs to a group, all group members also aggro.
func (w *World) AggroEnemy(e *entity.Enemy, targetPeerID uint16) {
	if e.State != entity.EnemyPatrol {
		return
	}
	e.State = entity.EnemyChase
	e.ChaseTimer = 0
	e.TargetPlayerID = targetPeerID

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
