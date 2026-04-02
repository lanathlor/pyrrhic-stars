package system

import (
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
	ZoneType uint8 // 0=Hub, 1=Arena

	// Tick counter
	TickNum uint32

	// Game state
	State        GameFlowState
	BossDefeated bool

	// Entities
	Players     map[uint16]*entity.Player
	Enemies     []*entity.Enemy
	Projectiles []*entity.Projectile
	NextProjID  uint32

	// AI brains (parallel to Enemies)
	Brains []*enemyai.Brain

	// Level geometry
	Level *level.Level

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
}

// FirstEnemy returns the first enemy or nil.
// Convenience for phase 0 where there's only one enemy.
func (w *World) FirstEnemy() *entity.Enemy {
	if len(w.Enemies) > 0 {
		return w.Enemies[0]
	}
	return nil
}

// SpawnProjectile creates a new projectile in the world.
func (w *World) SpawnProjectile(pos, dir entity.Vec3, speed, damage, lifetime float32) {
	w.NextProjID++
	p := entity.NewProjectile(w.NextProjID, 0, pos, dir, speed, damage, lifetime)
	w.Projectiles = append(w.Projectiles, p)
}
