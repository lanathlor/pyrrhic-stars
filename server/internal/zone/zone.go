package zone

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"codex-online/server/internal/codec"
	"codex-online/server/internal/enemyai"
	"codex-online/server/internal/entity"
	"codex-online/server/internal/level"
	"codex-online/server/internal/message"
	"codex-online/server/internal/system"
)

const (
	TickRate   = 20
	TickPeriod = time.Second / TickRate
	DeltaTime  = 1.0 / float32(TickRate)
)

// Re-export types from system package so gateway and tests can use them
// without importing system directly.
type GameFlowState = system.GameFlowState

const (
	StateLobby    = system.StateLobby
	StateSpawned  = system.StateSpawned
	StateFight    = system.StateFight
	StateFightOver = system.StateFightOver
)

// ZoneType distinguishes hub (social) zones from arena (combat) zones.
type ZoneType uint8

const (
	ZoneTypeHub   ZoneType = 0
	ZoneTypeArena ZoneType = 1
)

// Client is a re-export of system.Client for use by the gateway.
type Client = system.Client

// Zone simulates one game instance (hub or arena).
// It is a pure container: all game logic lives in system packages.
// The tick loop drains inputs, increments the tick counter, and runs
// the system pipeline in order.
type Zone struct {
	ID   string
	Type ZoneType

	world   system.World
	systems []system.System

	pendingInputs []system.InputMsg
	mu            sync.Mutex

	// OnPlayerRespawnHub is called when a dead player requests to return to hub.
	// Set by the gateway to trigger zone transfer for a single player.
	OnPlayerRespawnHub func(peerID uint16)
}

// New creates a zone ready to run. The level defines geometry, spawns, and
// obstacles. Pass nil to use the default level for the zone type.
func New(id string, zoneType ZoneType, lvl ...*level.Level) *Zone {
	z := &Zone{
		ID:   id,
		Type: zoneType,
	}

	// Resolve level: use provided level or create default from zone type.
	var l *level.Level
	if len(lvl) > 0 && lvl[0] != nil {
		l = lvl[0]
	} else if zoneType == ZoneTypeArena {
		l = level.NewArenaLevel()
	} else {
		l = level.NewHubLevel()
	}

	z.world = system.World{
		ZoneID:   id,
		ZoneType: uint8(zoneType),
		State:    system.StateLobby,
		Players:  make(map[uint16]*entity.Player),
		Clients:  make(map[uint16]*system.Client),
		Level:    l,
	}

	if zoneType == ZoneTypeArena {
		for i, sp := range l.EnemySpawns {
			def := enemyai.DefRegistry[sp.DefName]
			if def == nil {
				slog.Warn("unknown enemy def", "def_name", sp.DefName)
				continue
			}
			enemy := entity.NewEnemy(uint16(1000+i), def.MaxHealth, sp.DefName)
			enemy.Alive = false // dormant until fight starts
			enemy.IsBoss = sp.IsBoss
			enemy.PatrolA = sp.PatrolA
			enemy.PatrolB = sp.PatrolB
			enemy.AggroRadius = sp.AggroRadius
			enemy.LeashOrigin = sp.Position
			enemy.LeashRadius = sp.LeashRadius
			enemy.GroupID = sp.GroupID
			brain := enemyai.NewBrain(def, enemy)
			// Brain bounds = full instance bounds (leash handles area restriction)
			brain.BoundsMinX = l.EnemyBoundsMinX
			brain.BoundsMaxX = l.EnemyBoundsMaxX
			brain.BoundsMinZ = l.EnemyBoundsMinZ
			brain.BoundsMaxZ = l.EnemyBoundsMaxZ
			z.world.Enemies = append(z.world.Enemies, enemy)
			z.world.Brains = append(z.world.Brains, brain)
		}
		// Activate all enemies immediately — they patrol from the start
		system.InitInstance(&z.world)
	}

	// Spawn hub NPCs from level data
	if zoneType == ZoneTypeHub {
		for i, sp := range l.NPCSpawns {
			npc := entity.NewNPC(uint16(2000+i), sp.DefName, sp.Speed, sp.IdleDuration, sp.Waypoints)
			z.world.NPCs = append(z.world.NPCs, npc)
		}
	}

	// Build system pipeline based on zone type.
	if zoneType == ZoneTypeHub {
		z.systems = []system.System{
			&system.InputSystem{},
			&system.NPCSystem{},
			&system.CombatSystem{},
			&system.NetworkSystem{},
		}
	} else {
		z.systems = []system.System{
			&system.InputSystem{},
			&system.GameFlowSystem{},
			&system.AISystem{},
			&system.CombatSystem{},
			&system.PhysicsSystem{},
			&system.NetworkSystem{},
		}
	}

	return z
}

// AddClient adds a connected client to the zone.
func (z *Zone) AddClient(c *Client) {
	z.mu.Lock()
	z.world.Clients[c.PeerID] = c
	if p, ok := z.world.Players[c.PeerID]; !ok {
		np := entity.NewPlayer(c.PeerID, "gunner")
		np.Username = c.Username
		// Set spawn position immediately so the tick loop never sees origin
		if len(z.world.Level.PlayerSpawns) > 0 {
			idx := len(z.world.Players) % len(z.world.Level.PlayerSpawns)
			np.Position = z.world.Level.PlayerSpawns[idx]
			np.RotationY = z.world.Level.SpawnYaw
		}
		z.world.Players[c.PeerID] = np
	} else {
		p.Username = c.Username
	}

	// Arena: initialize player fully and send catch-up state
	var catchUpFlow uint8
	if z.Type == ZoneTypeArena {
		system.SpawnPlayer(&z.world, c.PeerID)
		switch z.world.State {
		case system.StateLobby:
			z.world.State = system.StateSpawned
			catchUpFlow = message.FlowSpawnPlayers
			slog.Info("arena entered, skipping lobby", "zone_id", z.ID)
		case system.StateSpawned:
			catchUpFlow = message.FlowSpawnPlayers
		case system.StateFight:
			catchUpFlow = message.FlowFightStart
		case system.StateFightOver:
			if z.world.BossDefeated {
				catchUpFlow = message.FlowBossDead
			} else {
				catchUpFlow = message.FlowAllDead
			}
		}
	}
	z.mu.Unlock()

	// Send catch-up to the joining client (and broadcast if we just changed state)
	if catchUpFlow > 0 {
		payload := codec.EncodeGameFlow(catchUpFlow, "")
		msg := message.Encode(message.OpGameFlowEvent, 0, payload)
		c.Send(msg)
	}
}

// RemoveClient removes a disconnected client.
func (z *Zone) RemoveClient(peerID uint16) {
	z.mu.Lock()
	defer z.mu.Unlock()
	delete(z.world.Clients, peerID)
	delete(z.world.Players, peerID)
}

// QueueInput adds a client message to the input queue for processing on the next tick.
func (z *Zone) QueueInput(peerID, opcode uint16, payload []byte) {
	z.mu.Lock()
	defer z.mu.Unlock()
	z.pendingInputs = append(z.pendingInputs, system.InputMsg{PeerID: peerID, Opcode: opcode, Payload: payload})
}

// ClientCount returns the number of connected clients.
func (z *Zone) ClientCount() int {
	z.mu.Lock()
	defer z.mu.Unlock()
	return len(z.world.Clients)
}

// Run starts the zone tick loop. Blocks until ctx is cancelled.
func (z *Zone) Run(ctx context.Context) {
	ticker := time.NewTicker(TickPeriod)
	defer ticker.Stop()
	slog.Info("zone started", "zone_id", z.ID, "tick_rate", TickRate)
	for {
		select {
		case <-ticker.C:
			z.processTick()
		case <-ctx.Done():
			slog.Info("zone stopped", "zone_id", z.ID)
			return
		}
	}
}

func (z *Zone) processTick() {
	z.mu.Lock()
	// Drain pending inputs into the world's input queue
	z.world.InputQueue = append(z.world.InputQueue[:0], z.pendingInputs...)
	z.pendingInputs = z.pendingInputs[:0]
	// Copy the callback into the world so systems can access it
	z.world.OnPlayerRespawnHub = z.OnPlayerRespawnHub
	z.mu.Unlock()

	z.world.TickNum++
	for _, sys := range z.systems {
		sys.Tick(&z.world, DeltaTime)
	}
}

// GetPeerIDs returns the IDs of all connected clients.
func (z *Zone) GetPeerIDs() []uint16 {
	z.mu.Lock()
	defer z.mu.Unlock()
	ids := make([]uint16, 0, len(z.world.Clients))
	for id := range z.world.Clients {
		ids = append(ids, id)
	}
	return ids
}

// GetPlayer returns the player entity for a peer ID, or nil if not found.
func (z *Zone) GetPlayer(peerID uint16) *entity.Player {
	z.mu.Lock()
	defer z.mu.Unlock()
	return z.world.Players[peerID]
}

// SetPlayerPosition overrides a player's position and rotation.
func (z *Zone) SetPlayerPosition(peerID uint16, pos entity.Vec3, rotY float32) {
	z.mu.Lock()
	defer z.mu.Unlock()
	if p, ok := z.world.Players[peerID]; ok {
		p.Position = pos
		p.RotationY = rotY
	}
}

// Broadcast sends a message to all clients except excludePeerID (0 = send to all).
func (z *Zone) Broadcast(msg []byte, excludePeerID uint16) {
	z.mu.Lock()
	defer z.mu.Unlock()
	for id, c := range z.world.Clients {
		if excludePeerID != 0 && id == excludePeerID {
			continue
		}
		c.Send(msg)
	}
}
