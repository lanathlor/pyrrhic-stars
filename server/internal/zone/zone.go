package zone

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"os"
	"sync"
	"time"

	"codex-online/server/internal/ability"
	"codex-online/server/internal/bot"
	"codex-online/server/internal/codec"
	"codex-online/server/internal/combat"
	"codex-online/server/internal/combatlog"
	"codex-online/server/internal/enemyai"
	"codex-online/server/internal/entity"
	"codex-online/server/internal/level"
	"codex-online/server/internal/message"
	"codex-online/server/internal/overflux"
	"codex-online/server/internal/system"
)

const (
	TickRate   = 20
	TickPeriod = time.Second / TickRate
	DeltaTime  = 1.0 / float32(TickRate)
)

// ZoneType distinguishes open-world (persistent) zones from instanced (combat) zones.
type ZoneType uint8

const (
	ZoneTypeOpenWorld ZoneType = 0
	ZoneTypeInstanced ZoneType = 1
)

// Client is a re-export of system.Client for use by the gateway.
type Client = system.Client

// Zone simulates one game instance (open-world or instanced).
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

	// onPlayerReturnToOpenWorld is called when a dead player requests to leave
	// an instance. Set by the gateway via SetOnPlayerReturnToOpenWorld.
	onPlayerReturnToOpenWorld func(peerID uint16)

	// onBossDefeated is called when the boss dies. Receives peer IDs of all
	// players present and the overflux score for reward calculation.
	onBossDefeated func(peerIDs []uint16, overfluxScore int)

	// CombatLogSink receives combat events. Set before Run(). NullSink if nil.
	CombatLogSink combatlog.EventSink

	// replayBuf is a reusable buffer for encoding WorldState frames.
	// Shared across all active replay recorders to avoid redundant encoding.
	replayBuf []byte

	// botMgr manages dev-mode bot spawning and behavior. Nil when not in dev mode.
	botMgr *bot.Manager
}

// New creates a zone ready to run. The level defines geometry, spawns, zone type,
// and all spatial data. Pass oflx for instanced zones with overflux conditions
// (nil for open-world zones). Set CombatLogSink on the returned Zone before
// calling Run() to enable combat event logging.
func New(id string, lvl *level.Level, oflx *overflux.State) *Zone {
	var zoneType ZoneType
	if lvl.ZoneType == "instanced" {
		zoneType = ZoneTypeInstanced
	}

	z := &Zone{
		ID:   id,
		Type: zoneType,
	}

	l := lvl

	devMode := os.Getenv("CODEX_DEV") == "1"
	z.world = system.World{
		ZoneID:          id,
		ZoneType:        uint8(zoneType),
		RunID:           fmt.Sprintf("%s_%d", id, time.Now().UnixMilli()),
		EnemyDamageMult: 1.0,
		DevMode:         devMode,
		Players:         make(map[uint16]*entity.Player),
		Clients:         make(map[uint16]*system.Client),
		Level:           l,
		AbilityEngine:   ability.NewEngine(slog.Default().With("zone_id", id)),
		AbilityRunners:  make(map[uint16]*ability.PlayerAbilityRunner),
		PatternEngine:   combat.NewPatternEngine(),
		PatternRng:      rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 0)),
		OverfluxState:   oflx,
	}
	if devMode {
		z.world.DebugGodModePeers = make(map[uint16]bool)
		z.botMgr = bot.NewManager("")
		z.botMgr.OnRescale = func(totalPlayers int) {
			z.RescaleForPlayerCount(totalPlayers)
		}
	}

	// Initialize obstacles (Level.Obstacles + closed gate obstacles).
	z.world.InitGateStates()

	// Spawn enemies from level data (all zone types).
	spawnEnemies(z, l)
	for i, e := range z.world.Enemies {
		if i < len(l.EnemySpawns) {
			e.Reset(l.EnemySpawns[i].Position, entity.EnemyPatrol)
		}
	}

	// Spawn NPCs from level data (all zone types).
	for i, sp := range l.NPCSpawns {
		npc := entity.NewNPC(uint16(2000+i), sp.DefName, sp.Speed, sp.IdleDuration, sp.Waypoints)
		z.world.NPCs = append(z.world.NPCs, npc)
	}

	z.systems = buildSystemPipeline(z.botMgr)

	return z
}

func buildSystemPipeline(botMgr *bot.Manager) []system.System {
	var botSys system.System = &system.NoOpSystem{}
	if botMgr != nil {
		botSys = &bot.System{Manager: botMgr}
	}
	return []system.System{
		botSys,
		&system.CombatSystem{},
		&system.InputSystem{},
		&system.NPCSystem{},
		&system.GameFlowSystem{},
		&system.AISystem{},
		&system.PhysicsSystem{},
		&system.NetworkSystem{},
	}
}

func spawnEnemies(z *Zone, l *level.Level) {
	for i, sp := range l.EnemySpawns {
		def := enemyai.DefRegistry[sp.DefName]
		if def == nil {
			slog.Warn("unknown enemy def", "def_name", sp.DefName)
			continue
		}
		enemy := entity.NewEnemy(uint16(1000+i), def.MaxHealth, sp.DefName)
		enemy.Alive = false
		enemy.IsBoss = sp.IsBoss
		enemy.PatrolA = sp.PatrolA
		enemy.PatrolB = sp.PatrolB
		enemy.AggroRadius = sp.AggroRadius
		enemy.LeashOrigin = sp.Position
		enemy.LeashRadius = sp.LeashRadius
		enemy.GroupID = sp.GroupID
		brain := enemyai.NewBrain(def, enemy, z.world.AbilityEngine)
		brain.ApplyOverfluxVariants(z.world.OverfluxState)
		brain.BoundsMinX = l.EnemyBoundsMinX
		brain.BoundsMaxX = l.EnemyBoundsMaxX
		brain.BoundsMinZ = l.EnemyBoundsMinZ
		brain.BoundsMaxZ = l.EnemyBoundsMaxZ
		z.world.Enemies = append(z.world.Enemies, enemy)
		z.world.Brains = append(z.world.Brains, brain)
		if enemy.IsBoss {
			z.world.Boss = enemy
		}
	}
}

// SetOnPlayerReturnToOpenWorld sets the callback invoked when a dead player
// requests to leave an instance. Thread-safe: acquires z.mu to synchronize
// with the tick goroutine that reads this callback in processTick.
func (z *Zone) SetOnPlayerReturnToOpenWorld(fn func(peerID uint16)) {
	z.mu.Lock()
	z.onPlayerReturnToOpenWorld = fn
	z.mu.Unlock()
}

// SetOnBossDefeated sets the callback invoked when the boss is defeated.
func (z *Zone) SetOnBossDefeated(fn func(peerIDs []uint16, overfluxScore int)) {
	z.mu.Lock()
	z.onBossDefeated = fn
	z.mu.Unlock()
}

// OverfluxState returns the overflux conditions for this zone, or nil.
func (z *Zone) OverfluxState() *overflux.State {
	return z.world.OverfluxState
}

// SetGroupSize configures instance scaling based on the number of players.
// HP scales from 1x (solo) to 4x (5 players), damage from 1x to 2x.
// Must be called before the fight starts. Safe to call from any goroutine.
func (z *Zone) SetGroupSize(n int) {
	z.mu.Lock()
	z.rescaleEnemies(n)
	z.mu.Unlock()
}

// RescaleForPlayerCount adjusts enemy HP and damage scaling for the current
// total player count (humans + bots). Safe to call mid-fight: preserves each
// enemy's current HP percentage. Safe to call from any goroutine.
func (z *Zone) RescaleForPlayerCount(n int) {
	z.mu.Lock()
	z.rescaleEnemies(n)
	z.mu.Unlock()
}

func (z *Zone) rescaleEnemies(n int) {
	if n < 1 {
		n = 1
	}
	hpMult := float32(1.0 + 0.75*float64(n-1))
	if z.world.OverfluxState != nil {
		hpMult *= z.world.OverfluxState.HPMultiplier()
	}
	dmgMult := float32(1.0 + 0.25*float64(n-1))
	if z.world.OverfluxState != nil {
		dmgMult *= z.world.OverfluxState.DamageMultiplier()
	}
	z.world.EnemyDamageMult = dmgMult
	for _, e := range z.world.Enemies {
		if e.BaseMaxHealth == 0 {
			e.BaseMaxHealth = e.MaxHealth
		}
		newMax := e.BaseMaxHealth * hpMult
		if e.MaxHealth > 0 {
			ratio := e.Health / e.MaxHealth
			e.MaxHealth = newMax
			e.Health = newMax * ratio
		} else {
			e.MaxHealth = newMax
			e.Health = newMax
		}
	}
}

// AddClient adds a connected client to the zone.
func (z *Zone) AddClient(c *Client) {
	z.mu.Lock()
	z.world.Clients[c.PeerID] = c
	if p, ok := z.world.Players[c.PeerID]; !ok {
		np := entity.NewPlayer(c.PeerID, entity.ClassGunner)
		np.Username = c.Username
		// Set spawn position immediately so the tick loop never sees origin
		if len(z.world.Level.PlayerSpawns) > 0 {
			idx := len(z.world.Players) % len(z.world.Level.PlayerSpawns)
			np.Position = z.world.Level.PlayerSpawns[idx].Position
			np.RotationY = z.world.Level.SpawnYaw
		}
		if z.world.DevMode {
			np.GodMode = true
		}
		z.world.Players[c.PeerID] = np
	} else {
		p.Username = c.Username
	}

	// Initialize player fully for instanced zones (spawn at correct position).
	if z.Type == ZoneTypeInstanced {
		system.SpawnPlayer(&z.world, c.PeerID)
	}

	// Send catch-up flow event so late-joiners see the correct UI state.
	var catchUpFlow uint8
	if z.world.BossDefeated {
		catchUpFlow = message.FlowBossDead
	} else if z.world.WipeHandled {
		catchUpFlow = message.FlowAllDead
	}

	// Rescale enemies for new player count (already under z.mu).
	if len(z.world.Enemies) > 0 {
		z.rescaleEnemies(len(z.world.Players))
	}
	z.mu.Unlock()

	// Send catch-up to the joining client (and broadcast if we just changed state)
	if catchUpFlow > 0 {
		payload := codec.EncodeGameFlow(catchUpFlow, "")
		msg := message.Encode(message.OpGameFlowEvent, 0, payload)
		c.Send(msg)
	}

	z.sendGateCatchUp(c)
}

// sendGateCatchUp notifies a joining client about gates in non-default state.
func (z *Zone) sendGateCatchUp(c *Client) {
	for _, g := range z.world.Level.Gates {
		isClosed := z.world.GateStates[g.ID]
		if isClosed == g.DefaultClosed {
			continue
		}
		flowType := message.FlowGateOpen
		if isClosed {
			flowType = message.FlowGateClose
		}
		payload := codec.EncodeGameFlow(flowType, g.ID)
		msg := message.Encode(message.OpGameFlowEvent, 0, payload)
		c.Send(msg)
	}
}

// RemoveClient removes a disconnected client and any bots they owned.
func (z *Zone) RemoveClient(peerID uint16) {
	z.mu.Lock()
	defer z.mu.Unlock()
	if z.botMgr != nil {
		// Dismiss bots without triggering OnRescale (which calls
		// RescaleForPlayerCount → z.mu.Lock, causing a reentrant deadlock).
		// We rescale directly below since we already hold z.mu.
		z.botMgr.DismissAllForOwnerNoRescale(peerID, &z.world)
	}
	delete(z.world.Clients, peerID)
	delete(z.world.Players, peerID)
	delete(z.world.AbilityRunners, peerID)
	// Rescale directly (already under z.mu).
	if len(z.world.Enemies) > 0 || z.botMgr != nil {
		z.rescaleEnemies(len(z.world.Players))
	}
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

// TickNum returns the zone's current tick counter.
func (z *Zone) TickNum() uint32 {
	z.mu.Lock()
	defer z.mu.Unlock()
	return z.world.TickNum
}

// IncrementTick advances the tick counter by one. Used in tests to simulate
// elapsed time without running the full tick loop.
func (z *Zone) IncrementTick() {
	z.mu.Lock()
	z.world.TickNum++
	z.mu.Unlock()
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
	// Copy the callbacks into the world so systems can access them
	z.world.OnPlayerReturnToOpenWorld = z.onPlayerReturnToOpenWorld
	z.world.OnBossDefeated = z.onBossDefeated
	// Wire combat log sink
	z.world.CombatLogSink = z.CombatLogSink
	// Snapshot client list for broadcast (prevents race with RemoveClient)
	z.world.ClientSnapshot = z.world.ClientSnapshot[:0]
	for _, c := range z.world.Clients {
		z.world.ClientSnapshot = append(z.world.ClientSnapshot, c)
	}
	z.mu.Unlock()

	z.world.TickNum++
	dt := DeltaTime
	if z.world.DevMode && z.world.TimeScale > 0 {
		dt *= z.world.TimeScale
	}
	for _, sys := range z.systems {
		sys.Tick(&z.world, dt)
	}

	// Record replay frame for every active encounter session.
	z.recordReplayFrames()
}

// recordReplayFrames encodes the WorldState once and appends it to all active
// replay recorders. No-op when no encounter sessions are recording.
func (z *Zone) recordReplayFrames() {
	hasSessions := len(z.world.CombatLogs) > 0
	if !hasSessions {
		return
	}

	// Encode WorldState once, share the buffer across all recorders.
	z.replayBuf = z.replayBuf[:0]
	z.replayBuf = codec.AppendEncodeWorldState(
		z.replayBuf, z.world.TickNum,
		z.world.Players, z.world.Enemies, z.world.Projectiles, z.world.NPCs,
	)

	for _, session := range z.world.CombatLogs {
		if session.Recorder != nil {
			session.Recorder.AppendFrame(z.replayBuf)
		}
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

// Portals returns the level's portal definitions.
func (z *Zone) Portals() []level.PortalDef {
	return z.world.Level.Portals
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

// SetPlayerGear updates a player's computed gear stats and recalculates derived
// stats (MaxHealth, etc.). Called by the gateway on join and on equip/unequip.
func (z *Zone) SetPlayerGear(peerID uint16, stats entity.GearStats) {
	z.mu.Lock()
	defer z.mu.Unlock()
	if p, ok := z.world.Players[peerID]; ok {
		p.GearStats = stats
		p.RecalcStats()
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
