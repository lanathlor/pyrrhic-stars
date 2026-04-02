package zone

import (
	"context"
	"log/slog"
	"math"
	"math/rand"
	"sync"
	"time"

	"codex-online/server/internal/codec"
	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
	"codex-online/server/internal/message"
)

const (
	TickRate    = 20
	TickPeriod  = time.Second / TickRate
	DeltaTime   = 1.0 / float32(TickRate)
)

// GameFlowState tracks the zone's game state.
type GameFlowState uint8

const (
	StateLobby   GameFlowState = iota
	StateSpawned // players spawned, waiting for arena entry
	StateFight
	StateFightOver
)

// ZoneType distinguishes hub (social) zones from arena (combat) zones.
type ZoneType uint8

const (
	ZoneTypeHub   ZoneType = 0
	ZoneTypeArena ZoneType = 1
)

// Client represents a connected player for sending messages.
type Client struct {
	PeerID   uint16
	Username string
	Send     func([]byte) // queues a message to the client
}

// Zone simulates one game instance (hub or arena).
type Zone struct {
	ID   string
	Type ZoneType
	Tick uint32

	State        GameFlowState
	BossDefeated bool

	Players     map[uint16]*entity.Player
	Enemy       *entity.Enemy
	Projectiles []*entity.Projectile

	// Networking
	clients    map[uint16]*Client
	inputQueue []inputMsg
	mu         sync.Mutex

	// Projectile ID counter
	nextProjID uint32

	// Damage events from this tick (broadcast to clients)
	damageEvents []combat.DamageEvent

	// Spawn positions
	playerSpawns []entity.Vec3
	enemySpawn   entity.Vec3

	// OnPlayerRespawnHub is called when a dead player requests to return to hub.
	// Set by the gateway to trigger zone transfer for a single player.
	OnPlayerRespawnHub func(peerID uint16)
}

type inputMsg struct {
	PeerID  uint16
	Opcode  uint16
	Payload []byte
}

// New creates a zone ready to run.
func New(id string, zoneType ZoneType) *Zone {
	z := &Zone{
		ID:      id,
		Type:    zoneType,
		State:   StateLobby,
		Players: make(map[uint16]*entity.Player),
		clients: make(map[uint16]*Client),
	}
	if zoneType == ZoneTypeArena {
		z.Enemy = entity.NewEnemy(0)
		z.Enemy.Alive = false // dormant until fight starts
		z.playerSpawns = []entity.Vec3{
			{X: -2.0, Y: 0.1, Z: 20.0},
			{X: 0.0, Y: 0.1, Z: 20.0},
			{X: 2.0, Y: 0.1, Z: 20.0},
			{X: -1.0, Y: 0.1, Z: 21.0},
			{X: 1.0, Y: 0.1, Z: 21.0},
		}
		z.enemySpawn = entity.Vec3{X: 0.0, Y: 0.1, Z: 0.0}
	} else {
		// Hub zone: larger spawn area, no enemy
		z.playerSpawns = []entity.Vec3{
			{X: -2.0, Y: 0.1, Z: -5.0},
			{X: 0.0, Y: 0.1, Z: -5.0},
			{X: 2.0, Y: 0.1, Z: -5.0},
			{X: -1.0, Y: 0.1, Z: -3.0},
			{X: 1.0, Y: 0.1, Z: -3.0},
		}
	}
	return z
}

// AddClient adds a connected client to the zone.
func (z *Zone) AddClient(c *Client) {
	z.mu.Lock()
	z.clients[c.PeerID] = c
	if p, ok := z.Players[c.PeerID]; !ok {
		np := entity.NewPlayer(c.PeerID, "gunner")
		np.Username = c.Username
		// Set spawn position immediately so the tick loop never sees origin
		if z.Type == ZoneTypeArena && len(z.playerSpawns) > 0 {
			idx := len(z.Players) % len(z.playerSpawns)
			np.Position = z.playerSpawns[idx]
		} else if len(z.playerSpawns) > 0 {
			idx := len(z.Players) % len(z.playerSpawns)
			np.Position = z.playerSpawns[idx]
		}
		z.Players[c.PeerID] = np
	} else {
		p.Username = c.Username
	}

	// Arena: initialize player fully and send catch-up state
	var catchUpFlow uint8
	if z.Type == ZoneTypeArena {
		z.spawnPlayer(c.PeerID)
		switch z.State {
		case StateLobby:
			z.State = StateSpawned
			catchUpFlow = message.FlowSpawnPlayers
			slog.Info("arena entered, skipping lobby", "zone_id", z.ID)
		case StateSpawned:
			catchUpFlow = message.FlowSpawnPlayers
		case StateFight:
			catchUpFlow = message.FlowFightStart
		case StateFightOver:
			if z.BossDefeated {
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
	delete(z.clients, peerID)
	delete(z.Players, peerID)
}

// QueueInput adds a client message to the input queue for processing on the next tick.
func (z *Zone) QueueInput(peerID, opcode uint16, payload []byte) {
	z.mu.Lock()
	defer z.mu.Unlock()
	z.inputQueue = append(z.inputQueue, inputMsg{PeerID: peerID, Opcode: opcode, Payload: payload})
}

// ClientCount returns the number of connected clients.
func (z *Zone) ClientCount() int {
	z.mu.Lock()
	defer z.mu.Unlock()
	return len(z.clients)
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
	inputs := z.inputQueue
	z.inputQueue = z.inputQueue[:0]
	z.mu.Unlock()

	z.Tick++

	// 1. Process inputs
	for _, inp := range inputs {
		z.handleInput(inp)
	}

	// 2. Process game state
	if z.Type == ZoneTypeHub {
		z.tickHub()
	} else {
		switch z.State {
		case StateLobby:
			z.tickLobby()
		case StateSpawned:
			z.tickSpawned()
		case StateFight:
			z.tickFight()
		case StateFightOver:
			z.tickFightOver()
		}
	}

	// 3. Broadcast state
	z.broadcastState()

	// 4. Clear damage events after broadcast (not before — input processing
	// appends events in step 1 that must survive to step 3).
	z.damageEvents = z.damageEvents[:0]
}

func (z *Zone) handleInput(inp inputMsg) {
	switch inp.Opcode {
	case message.OpPlayerInput:
		z.handlePlayerInput(inp.PeerID, inp.Payload)
	case message.OpAbilityInput:
		z.handleAbilityInput(inp.PeerID, inp.Payload)
	case message.OpInteractInput:
		z.handleInteractInput(inp.PeerID, inp.Payload)
	case message.OpRespawnRequest:
		z.handleRespawnRequest(inp.PeerID, inp.Payload)
	}
}

// =============================================================================
// Input handlers
// =============================================================================

func (z *Zone) handlePlayerInput(peerID uint16, payload []byte) {
	inp := codec.DecodePlayerInput(payload)
	if inp == nil {
		return
	}
	p, ok := z.Players[peerID]
	if !ok {
		return
	}

	// Reject positions that teleport too far from the server-assigned position.
	// This catches the first bogus (0,0,0) frame from a freshly spawned client.
	newPos := entity.Vec3{X: inp.PosX, Y: inp.PosY, Z: inp.PosZ}
	dx := newPos.X - p.Position.X
	dy := newPos.Y - p.Position.Y
	dz := newPos.Z - p.Position.Z
	dist := dx*dx + dy*dy + dz*dz
	if dist > 100.0 { // > 10 units teleport = reject
		return
	}

	// Client-authoritative: accept position, clamp to boundaries
	p.Position = newPos
	if z.Type == ZoneTypeHub {
		p.Position.X = entity.Clamp(p.Position.X, -14.5, 14.5)
		p.Position.Z = entity.Clamp(p.Position.Z, -9.5, 14.5)
	} else {
		p.Position.X = entity.Clamp(p.Position.X, -19.5, 19.5)
		p.Position.Z = entity.Clamp(p.Position.Z, -14.5, 24.5)
	}
	p.RotationY = inp.RotY
	p.LastInput = &entity.PlayerInput{PosX: inp.PosX, PosY: inp.PosY, PosZ: inp.PosZ, RotY: inp.RotY, Tick: inp.Tick}
	p.AnimName = inp.AnimName
	p.AnimSpeed = inp.AnimSpeed
	p.AimPitch = inp.AimPitch
}

func (z *Zone) handleAbilityInput(peerID uint16, payload []byte) {
	inp := codec.DecodeAbilityInput(payload)
	if inp == nil {
		return
	}
	p, ok := z.Players[peerID]
	if !ok || !p.Alive {
		return
	}
	if z.State != StateFight {
		return
	}

	switch inp.Action {
	case entity.ActionShoot:
		// Gunner: hitscan, gated by fire cooldown
		if p.ClassName == "gunner" && p.FireCooldown <= 0 {
			p.FireCooldown = 0.18
			p.State = entity.PlayerStateAttack
			p.AimPitch = inp.AimPitch
			evt := combat.ResolvePlayerAttackOnEnemy(p, z.Enemy, arenaObstacles)
			if evt != nil {
				evt.SourcePeerID = peerID
				z.damageEvents = append(z.damageEvents, *evt)
			}
		}
	case entity.ActionMelee:
		// Vanguard/blade_dancer: melee, gated by cooldown
		if p.FireCooldown <= 0 {
			if p.ClassName == "vanguard" {
				p.FireCooldown = 0.55
			} else {
				p.FireCooldown = 0.3
			}
			p.State = entity.PlayerStateAttack
			evt := combat.ResolvePlayerAttackOnEnemy(p, z.Enemy, arenaObstacles)
			if evt != nil {
				evt.SourcePeerID = peerID
				z.damageEvents = append(z.damageEvents, *evt)
			}
		}
	case entity.ActionHeavy:
		if p.ClassName == "vanguard" && p.FireCooldown <= 0 {
			p.FireCooldown = 0.8
			p.State = entity.PlayerStateAttack
			evt := combat.ResolvePlayerAttackOnEnemy(p, z.Enemy, arenaObstacles)
			if evt != nil {
				evt.SourcePeerID = peerID
				z.damageEvents = append(z.damageEvents, *evt)
			}
		}
	}
}

func (z *Zone) handleInteractInput(peerID uint16, payload []byte) {
	inp := codec.DecodeInteractInput(payload)
	if inp == nil {
		return
	}
	p, ok := z.Players[peerID]
	if !ok {
		return
	}

	switch inp.Action {
	case message.InteractClassSelect:
		className := inp.ClassName
		if className == "gunner" || className == "vanguard" || className == "blade_dancer" {
			p.ClassName = className
			// Re-init class stats
			np := entity.NewPlayer(peerID, className)
			p.Health = np.Health
			p.MaxHealth = np.MaxHealth
			p.Stamina = np.Stamina
			p.MaxStamina = np.MaxStamina
		}
	case message.InteractReadyToggle:
		p.Ready = !p.Ready
		slog.Info("player ready toggled", "peer_id", peerID, "ready", p.Ready)
	case message.InteractExitPortal:
		if z.State == StateFightOver && z.BossDefeated {
			if z.OnPlayerRespawnHub != nil {
				z.OnPlayerRespawnHub(peerID)
			}
		}
	}
}

func (z *Zone) handleRespawnRequest(peerID uint16, payload []byte) {
	respawnType, ok := codec.DecodeRespawnRequest(payload)
	if !ok {
		return
	}
	player := z.Players[peerID]
	if player == nil || player.Alive {
		return
	}

	if respawnType == 1 { // hub
		if z.OnPlayerRespawnHub != nil {
			z.OnPlayerRespawnHub(peerID)
		}
	} else if respawnType == 0 { // arena
		if z.State == StateFightOver || z.State == StateLobby {
			player.Alive = true
			player.Health = player.MaxHealth
			player.State = entity.PlayerStateMove
			player.Position = entity.Vec3{X: 0, Y: 0.1, Z: 20}
		}
	}
}

// =============================================================================
// Lobby tick
// =============================================================================

func (z *Zone) tickLobby() {
	// Check if all players are ready (need at least 1)
	if len(z.Players) < 1 {
		return
	}
	allReady := true
	for _, p := range z.Players {
		if !p.Ready {
			allReady = false
			break
		}
	}
	if !allReady {
		return
	}

	// Spawn players — they'll walk into the arena to trigger fight
	z.spawnPlayers()
	z.State = StateSpawned
	slog.Info("all players ready, spawning", "zone_id", z.ID, "players", len(z.Players))
	z.broadcastGameFlow(message.FlowSpawnPlayers, "")
}

// =============================================================================
// Hub tick
// =============================================================================

func (z *Zone) tickHub() {
	// Client-authoritative movement: positions already updated in handlePlayerInput.
	// Nothing to simulate for the hub.
}

func (z *Zone) spawnPlayers() {
	idx := 0
	for _, p := range z.Players {
		spawnPos := z.playerSpawns[idx%len(z.playerSpawns)]
		p.Position = spawnPos
		p.Health = p.MaxHealth
		p.Alive = true
		p.State = entity.PlayerStateMove
		p.Velocity = entity.Vec3{}
		p.IsRolling = false
		p.RollCooldown = 0
		p.Invincible = false
		p.InvincibleTimer = 0
		idx++
	}
}

// spawnPlayer initializes a single player at the next available spawn point.
func (z *Zone) spawnPlayer(peerID uint16) {
	p, ok := z.Players[peerID]
	if !ok {
		return
	}
	idx := len(z.Players) - 1
	spawnPos := z.playerSpawns[idx%len(z.playerSpawns)]
	p.Position = spawnPos
	p.Health = p.MaxHealth
	p.Alive = true
	p.State = entity.PlayerStateMove
	p.Velocity = entity.Vec3{}
	p.IsRolling = false
	p.RollCooldown = 0
	p.Invincible = false
	p.InvincibleTimer = 0
}

// tickSpawned processes the state where players are spawned but haven't entered the arena yet.
func (z *Zone) tickSpawned() {
	// Client-authoritative movement: positions already updated in handlePlayerInput.
	// Check if any player crossed into the arena
	if z.checkPlayerArenaEntry() {
		z.StartFight()
	}
}

// StartFight transitions to fight state. Called when a player enters the arena trigger.
func (z *Zone) StartFight() {
	if z.State != StateSpawned {
		return
	}
	z.State = StateFight

	// Reset enemy
	z.Enemy.Reset(z.enemySpawn)
	z.Projectiles = nil

	slog.Info("fight started", "zone_id", z.ID, "players", len(z.Players))
	z.broadcastGameFlow(message.FlowFightStart, "")
}

// =============================================================================
// Fight tick
// =============================================================================

func (z *Zone) tickFight() {
	dt := DeltaTime

	// Tick down fire cooldowns (attacks resolved in handleAbilityInput)
	for _, p := range z.Players {
		if p.Alive {
			p.FireCooldown -= dt
			if p.State == entity.PlayerStateAttack && p.FireCooldown <= 0 {
				p.State = entity.PlayerStateMove
			}
		}
	}

	// Process enemy AI
	if z.Enemy.Alive {
		z.processEnemyAI(dt)
	}

	// Process projectiles
	z.processProjectiles(dt)

	// Check fight end conditions
	z.checkFightEnd()
}

// =============================================================================
// Enemy AI
// =============================================================================

func (z *Zone) processEnemyAI(dt float32) {
	e := z.Enemy
	e.StateTimer -= dt

	switch e.State {
	case entity.EnemyChase:
		z.processEnemyChase(dt)
	case entity.EnemyMeleeTelegraph:
		z.processEnemyMeleeTelegraph()
	case entity.EnemyMeleeAttack:
		z.processEnemyMeleeAttack()
	case entity.EnemyRangedTelegraph:
		z.processEnemyRangedTelegraph()
	case entity.EnemyRangedAttack:
		z.processEnemyRangedAttack()
	case entity.EnemyAoETelegraph:
		z.processEnemyAoETelegraph()
	case entity.EnemyAoESlam:
		z.processEnemyAoESlam()
	case entity.EnemyChargeTelegraph:
		z.processEnemyChargeTelegraph()
	case entity.EnemyCharge:
		z.processEnemyCharge(dt)
	case entity.EnemyCooldown:
		z.processEnemyCooldown()
	case entity.EnemyPhaseTransition:
		z.processEnemyPhaseTransition()
	case entity.EnemyDead:
		e.Velocity = entity.Vec3{}
	}

	// Apply velocity
	e.Position = e.Position.Add(e.Velocity.Scale(dt))
	// Clamp to arena walls
	e.Position.X = entity.Clamp(e.Position.X, -19.5, 19.5)
	e.Position.Z = entity.Clamp(e.Position.Z, -14.5, 14.5)
	if e.Position.Y < 0.1 {
		e.Position.Y = 0.1
	}
	// Push out of obstacles (pillars and cover)
	z.pushOutOfObstacles(&e.Position)
}

// Arena obstacles: 6 pillars (1.5x1.5) + 4 cover boxes
var arenaObstacles = []combat.Obstacle{
	// Pillars
	{CX: -8, CZ: -6, HX: 0.75, HZ: 0.75},
	{CX: 8, CZ: -6, HX: 0.75, HZ: 0.75},
	{CX: -8, CZ: 6, HX: 0.75, HZ: 0.75},
	{CX: 8, CZ: 6, HX: 0.75, HZ: 0.75},
	{CX: 0, CZ: -10, HX: 0.75, HZ: 0.75},
	{CX: 0, CZ: 10, HX: 0.75, HZ: 0.75},
	// Cover boxes
	{CX: -5, CZ: -2, HX: 1.5, HZ: 0.5},
	{CX: 5, CZ: 2, HX: 1.5, HZ: 0.5},
	{CX: -12, CZ: 0, HX: 0.5, HZ: 1.5},
	{CX: 12, CZ: 0, HX: 0.5, HZ: 1.5},
}

const enemyRadius float32 = 1.0

// pushOutOfObstacles resolves collisions between a position and arena obstacles.
func (z *Zone) pushOutOfObstacles(pos *entity.Vec3) {
	for _, obs := range arenaObstacles {
		// Expand obstacle by enemy radius (Minkowski sum)
		exHx := obs.HX + enemyRadius
		exHz := obs.HZ + enemyRadius
		dx := pos.X - obs.CX
		dz := pos.Z - obs.CZ
		if dx > -exHx && dx < exHx && dz > -exHz && dz < exHz {
			// Inside — push out along shortest axis
			pushX := exHx - abs32(dx)
			pushZ := exHz - abs32(dz)
			if pushX < pushZ {
				if dx > 0 {
					pos.X = obs.CX + exHx
				} else {
					pos.X = obs.CX - exHx
				}
			} else {
				if dz > 0 {
					pos.Z = obs.CZ + exHz
				} else {
					pos.Z = obs.CZ - exHz
				}
			}
		}
	}
}

func abs32(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}

func (z *Zone) processEnemyChase(dt float32) {
	e := z.Enemy
	e.ChaseTimer += dt

	target := z.getNearestAlivePlayer(e.Position)
	if target == nil {
		e.Velocity = entity.Vec3{}
		return
	}

	e.TargetPlayerID = target.PeerID
	toTarget := target.Position.Sub(e.Position).Flat()
	distance := toTarget.Length()

	if distance > 0.1 {
		dir := toTarget.Normalized()
		e.RotationY = float32(math.Atan2(float64(-dir.X), float64(-dir.Z)))
	}

	// In melee range → attack
	if distance <= entity.MeleeRange {
		attack := z.selectEnemyAttack(distance)
		if attack == entity.EnemyRangedTelegraph {
			attack = entity.EnemyMeleeTelegraph
		}
		if attack == entity.EnemyChargeTelegraph {
			attack = entity.EnemyAoETelegraph
		}
		e.ChangeState(attack)
		return
	}

	// Chase timer threshold
	chaseThreshold := float32(1.5)
	if distance > entity.MeleeRange*3.0 {
		chaseThreshold = 0.5
	}
	if e.ChaseTimer >= chaseThreshold {
		attack := z.selectEnemyAttack(distance)
		// Can't melee from far
		if attack == entity.EnemyMeleeTelegraph && distance > entity.MeleeRange {
			if distance > entity.MeleeRange*2.0 {
				attack = entity.EnemyChargeTelegraph
			} else {
				attack = entity.EnemyRangedTelegraph
			}
		}
		// AoE useless at long range
		if attack == entity.EnemyAoETelegraph && distance > e.GetAoERadius()*1.5 {
			attack = entity.EnemyChargeTelegraph
		}
		// Pick ranged target
		if attack == entity.EnemyRangedTelegraph {
			farthest := z.getFarthestAlivePlayer(e.Position)
			if farthest != nil {
				e.TargetPlayerID = farthest.PeerID
				e.RangedTargetPos = farthest.Position.Add(entity.Vec3{Y: 1.0})
			}
		}
		e.ChangeState(attack)
		return
	}

	// Move toward target
	if distance > entity.MeleeRange*0.8 {
		dir := toTarget.Normalized()
		spd := e.GetMoveSpeed()
		e.Velocity = entity.Vec3{X: dir.X * spd, Z: dir.Z * spd}
	} else {
		e.Velocity = entity.Vec3{}
	}
}

func (z *Zone) selectEnemyAttack(distance float32) entity.EnemyState {
	e := z.Enemy
	weights := e.PhaseWeights()
	attackNames := [4]string{"melee", "ranged", "aoe", "charge"}

	// Distance bias
	if distance <= entity.MeleeRange*2.0 {
		weights[0] = weights[0] * 3 / 2
		weights[1] = 0
		weights[2] = weights[2] * 13 / 10
		weights[3] = weights[3] * 3 / 10
	} else if distance > entity.MeleeRange*3.0 {
		weights[0] = weights[0] * 3 / 10
		weights[1] = weights[1] * 3 / 2
		weights[3] = weights[3] * 3 / 2
	}

	// Anti-repeat
	for i, name := range attackNames {
		if name == e.LastAttack && weights[i] > 1 {
			weights[i] /= 2
		}
	}

	total := 0
	for _, w := range weights {
		total += w
	}
	if total <= 0 {
		total = 1
	}

	roll := rand.Intn(total)
	cumulative := 0
	for i, w := range weights {
		cumulative += w
		if roll < cumulative {
			e.LastAttack = attackNames[i]
			switch i {
			case 0:
				return entity.EnemyMeleeTelegraph
			case 1:
				return entity.EnemyRangedTelegraph
			case 2:
				return entity.EnemyAoETelegraph
			case 3:
				return entity.EnemyChargeTelegraph
			}
		}
	}
	e.LastAttack = "melee"
	return entity.EnemyMeleeTelegraph
}

func (z *Zone) processEnemyMeleeTelegraph() {
	e := z.Enemy
	e.Velocity = entity.Vec3{}
	z.faceEnemyToTarget()
	if e.StateTimer <= 0 {
		e.ChangeState(entity.EnemyMeleeAttack)
	}
}

func (z *Zone) processEnemyMeleeAttack() {
	e := z.Enemy
	e.Velocity = entity.Vec3{}
	if e.StateTimer <= 0 {
		// Deal melee damage to all players in range
		for _, p := range z.Players {
			if !p.Alive {
				continue
			}
			dist := e.Position.DistanceTo(p.Position)
			if dist <= entity.MeleeRange && !combat.SegmentHitsObstacle(e.Position, p.Position, arenaObstacles) {
				dealt := p.ApplyDamage(e.GetMeleeDamage())
				if dealt > 0 {
					hitDir := p.Position.Sub(e.Position).Normalized()
					z.damageEvents = append(z.damageEvents, combat.DamageEvent{
						TargetPeerID: p.PeerID,
						Amount:       dealt,
						HitPos:       e.Position.Add(hitDir),
						SourceType:   combat.SourceEnemyMelee,
					})
				}
			}
		}
		e.ChangeState(entity.EnemyCooldown)
	}
}

func (z *Zone) processEnemyRangedTelegraph() {
	e := z.Enemy
	e.Velocity = entity.Vec3{}
	// Update ranged target position
	if target, ok := z.Players[e.TargetPlayerID]; ok && target.Alive {
		e.RangedTargetPos = target.Position.Add(entity.Vec3{Y: 1.0})
	}
	if e.StateTimer <= 0 {
		e.ChangeState(entity.EnemyRangedAttack)
	}
}

func (z *Zone) processEnemyRangedAttack() {
	e := z.Enemy
	if e.StateTimer <= 0 {
		// Fire projectiles
		count := e.GetRangedBurstCount()
		spreadAngle := float32(5.0 * math.Pi / 180.0)

		baseDir := e.RangedTargetPos.Sub(e.Position.Add(entity.Vec3{Y: 1.5})).Normalized()

		for i := 0; i < count; i++ {
			offset := (float32(i) - float32(count-1)/2.0) * spreadAngle
			dir := rotateVecY(baseDir, offset)
			z.spawnProjectile(
				e.Position.Add(entity.Vec3{Y: 1.5}),
				dir,
				22.0,
				e.GetRangedPerProjectileDamage(),
				5.0,
			)
		}
		e.ChangeState(entity.EnemyCooldown)
	}
}

func (z *Zone) processEnemyAoETelegraph() {
	e := z.Enemy
	e.Velocity = entity.Vec3{}
	if e.StateTimer <= 0 {
		e.ChangeState(entity.EnemyAoESlam)
	}
}

func (z *Zone) processEnemyAoESlam() {
	e := z.Enemy
	e.Velocity = entity.Vec3{}
	if e.StateTimer <= 0 {
		radius := e.GetAoERadius()
		damage := e.GetAoEDamage()
		for _, p := range z.Players {
			if !p.Alive {
				continue
			}
			if combat.CheckAoERadius(e.Position, p.Position, radius, arenaObstacles) {
				dealt := p.ApplyDamage(damage)
				if dealt > 0 {
					z.damageEvents = append(z.damageEvents, combat.DamageEvent{
						TargetPeerID: p.PeerID,
						Amount:       dealt,
						HitPos:       e.Position,
						SourceType:   combat.SourceEnemyAoE,
					})
				}
			}
		}
		e.ChangeState(entity.EnemyCooldown)
	}
}

func (z *Zone) processEnemyChargeTelegraph() {
	e := z.Enemy
	e.Velocity = entity.Vec3{}
	z.faceEnemyToTarget()
	// Lock charge direction
	if target, ok := z.Players[e.TargetPlayerID]; ok && target.Alive {
		dir := target.Position.Sub(e.Position).Flat()
		if dir.Length() > 0.1 {
			e.ChargeDirection = dir.Normalized()
		}
	}
	if e.StateTimer <= 0 {
		e.ChangeState(entity.EnemyCharge)
	}
}

func (z *Zone) processEnemyCharge(dt float32) {
	e := z.Enemy
	spd := e.GetChargeSpeed()
	e.Velocity = entity.Vec3{X: e.ChargeDirection.X * spd, Z: e.ChargeDirection.Z * spd}
	e.ChargeDistance += spd * dt

	// Hit players along path
	for _, p := range z.Players {
		if !p.Alive {
			continue
		}
		// Skip already-hit players
		alreadyHit := false
		for _, hid := range e.ChargeHitPlayers {
			if hid == p.PeerID {
				alreadyHit = true
				break
			}
		}
		if alreadyHit {
			continue
		}
		if e.Position.DistanceTo(p.Position) <= 2.0 {
			dealt := p.ApplyDamage(e.GetChargeDamage())
			if dealt > 0 {
				z.damageEvents = append(z.damageEvents, combat.DamageEvent{
					TargetPeerID: p.PeerID,
					Amount:       dealt,
					HitPos:       e.Position,
					SourceType:   combat.SourceEnemyCharge,
				})
			}
			e.ChargeHitPlayers = append(e.ChargeHitPlayers, p.PeerID)
		}
	}

	// Stop conditions
	if e.ChargeDistance >= e.GetChargeMaxDistance() || z.isAtWall(e.Position) {
		e.Velocity = entity.Vec3{}
		e.ChangeState(entity.EnemyCooldown)
	}
}

func (z *Zone) processEnemyCooldown() {
	e := z.Enemy
	e.Velocity = entity.Vec3{}
	if e.StateTimer <= 0 {
		e.ChaseTimer = 0
		e.ChangeState(entity.EnemyChase)
	}
}

func (z *Zone) processEnemyPhaseTransition() {
	e := z.Enemy
	e.Velocity = entity.Vec3{}
	if e.StateTimer <= 0 {
		e.ChangeState(entity.EnemyChase)
	}
}

// =============================================================================
// Projectiles
// =============================================================================

func (z *Zone) spawnProjectile(pos, dir entity.Vec3, speed, damage, lifetime float32) {
	z.nextProjID++
	proj := entity.NewProjectile(z.nextProjID, 0, pos, dir, speed, damage, lifetime)
	z.Projectiles = append(z.Projectiles, proj)
}

func (z *Zone) processProjectiles(dt float32) {
	alive := z.Projectiles[:0]
	for _, proj := range z.Projectiles {
		proj.Tick(dt)
		if !proj.Alive {
			continue
		}

		// Kill projectile if it hits an obstacle
		if combat.ProjectileHitsObstacle(proj.Position, entity.ProjectileHitRadius, arenaObstacles) {
			proj.Alive = false
			continue
		}

		// Check hit against players (enemy projectiles)
		if proj.OwnerID == 0 {
			for _, p := range z.Players {
				if !p.Alive {
					continue
				}
				if combat.CheckProjectileHit(proj.Position, p.Position, entity.ProjectileHitRadius+0.5) {
					dealt := p.ApplyDamage(proj.Damage)
					if dealt > 0 {
						z.damageEvents = append(z.damageEvents, combat.DamageEvent{
							TargetPeerID: p.PeerID,
							Amount:       dealt,
							HitPos:       proj.Position,
							SourceType:   combat.SourceEnemyRanged,
						})
					}
					proj.Alive = false
					break
				}
			}
		}
		if proj.Alive {
			alive = append(alive, proj)
		}
	}
	z.Projectiles = alive
}

// =============================================================================
// Fight end
// =============================================================================

func (z *Zone) checkFightEnd() {
	if z.Enemy.State == entity.EnemyDead {
		z.State = StateFightOver
		z.BossDefeated = true
		z.Projectiles = nil
		z.broadcastGameFlow(message.FlowBossDead, "")
		return
	}

	allDead := true
	for _, p := range z.Players {
		if p.Alive {
			allDead = false
			break
		}
	}
	if allDead && len(z.Players) > 0 {
		z.State = StateFightOver
		z.BossDefeated = false
		z.Projectiles = nil
		z.Enemy.Reset(z.enemySpawn)
		z.broadcastGameFlow(message.FlowAllDead, "")
	}
}

// =============================================================================
// Result tick
// =============================================================================

func (z *Zone) tickFightOver() {
	for _, p := range z.Players {
		if p.Alive {
			p.FireCooldown -= DeltaTime
		}
	}

	// After a wipe, transition back to lobby once all players have respawned
	if !z.BossDefeated {
		allAlive := true
		for _, p := range z.Players {
			if !p.Alive {
				allAlive = false
				break
			}
		}
		if allAlive && len(z.Players) > 0 {
			z.returnToLobby()
		}
	}
}

func (z *Zone) returnToLobby() {
	z.State = StateLobby
	z.Projectiles = nil
	for _, p := range z.Players {
		p.Ready = false
		p.Alive = true
		p.Health = p.MaxHealth
		p.State = entity.PlayerStateMove
		p.Position = entity.Vec3{X: 0, Y: 0.1, Z: 20.0}
		p.Velocity = entity.Vec3{}
	}
	z.broadcastGameFlow(message.FlowReturnLobby, "")
}

// =============================================================================
// Broadcasting
// =============================================================================

func (z *Zone) broadcastState() {
	z.mu.Lock()
	defer z.mu.Unlock()

	if z.Type == ZoneTypeHub {
		z.broadcastWorldState()
		return
	}

	switch z.State {
	case StateLobby:
		z.broadcastLobbyState()
	case StateSpawned, StateFight, StateFightOver:
		z.broadcastWorldState()
		z.broadcastDamageEvents()
	}
}

func (z *Zone) broadcastLobbyState() {
	infos := make([]codec.LobbyPlayerInfo, 0, len(z.Players))
	for _, p := range z.Players {
		infos = append(infos, codec.LobbyPlayerInfo{
			PeerID:    p.PeerID,
			ClassName: p.ClassName,
			Username:  p.Username,
			Ready:     p.Ready,
		})
	}
	payload := codec.EncodeLobbyState(infos)
	msg := message.Encode(message.OpLobbyState, 0, payload)
	for _, c := range z.clients {
		c.Send(msg)
	}
}

func (z *Zone) broadcastWorldState() {
	players := make([]*entity.Player, 0, len(z.Players))
	for _, p := range z.Players {
		players = append(players, p)
	}
	payload := codec.EncodeWorldState(z.Tick, players, z.Enemy, z.Projectiles)
	msg := message.Encode(message.OpWorldState, 0, payload)
	for _, c := range z.clients {
		c.Send(msg)
	}
}

func (z *Zone) broadcastDamageEvents() {
	for _, evt := range z.damageEvents {
		payload := codec.EncodeDamageEvent(
			evt.TargetPeerID, evt.SourcePeerID, evt.Amount,
			evt.HitPos.X, evt.HitPos.Y, evt.HitPos.Z,
			evt.SourceType,
		)
		msg := message.Encode(message.OpDamageEvent, 0, payload)
		for _, c := range z.clients {
			c.Send(msg)
		}
	}
}

func (z *Zone) broadcastGameFlow(flowType uint8, text string) {
	z.mu.Lock()
	defer z.mu.Unlock()
	payload := codec.EncodeGameFlow(flowType, text)
	msg := message.Encode(message.OpGameFlowEvent, 0, payload)
	for _, c := range z.clients {
		c.Send(msg)
	}
}

// =============================================================================
// Helpers
// =============================================================================

func (z *Zone) getNearestAlivePlayer(pos entity.Vec3) *entity.Player {
	var nearest *entity.Player
	minDist := float32(math.MaxFloat32)
	for _, p := range z.Players {
		if !p.Alive {
			continue
		}
		d := pos.DistanceToSq(p.Position)
		if d < minDist {
			minDist = d
			nearest = p
		}
	}
	return nearest
}

func (z *Zone) getFarthestAlivePlayer(pos entity.Vec3) *entity.Player {
	var farthest *entity.Player
	maxDist := float32(-1)
	for _, p := range z.Players {
		if !p.Alive {
			continue
		}
		d := pos.DistanceToSq(p.Position)
		if d > maxDist {
			maxDist = d
			farthest = p
		}
	}
	return farthest
}

func (z *Zone) faceEnemyToTarget() {
	e := z.Enemy
	target, ok := z.Players[e.TargetPlayerID]
	if !ok || !target.Alive {
		return
	}
	dir := target.Position.Sub(e.Position).Flat()
	if dir.Length() > 0.1 {
		e.RotationY = float32(math.Atan2(float64(-dir.X), float64(-dir.Z)))
	}
}

func (z *Zone) isAtWall(pos entity.Vec3) bool {
	return pos.X <= -19.0 || pos.X >= 19.0 || pos.Z <= -14.0 || pos.Z >= 14.0
}

func rotateVecY(v entity.Vec3, angle float32) entity.Vec3 {
	s := float32(math.Sin(float64(angle)))
	c := float32(math.Cos(float64(angle)))
	return entity.Vec3{
		X: v.X*c + v.Z*s,
		Y: v.Y,
		Z: -v.X*s + v.Z*c,
	}
}

// GetPeerIDs returns the IDs of all connected clients.
func (z *Zone) GetPeerIDs() []uint16 {
	z.mu.Lock()
	defer z.mu.Unlock()
	ids := make([]uint16, 0, len(z.clients))
	for id := range z.clients {
		ids = append(ids, id)
	}
	return ids
}

// Broadcast sends a message to all clients except excludePeerID (0 = send to all).
func (z *Zone) Broadcast(msg []byte, excludePeerID uint16) {
	z.mu.Lock()
	defer z.mu.Unlock()
	for id, c := range z.clients {
		if excludePeerID != 0 && id == excludePeerID {
			continue
		}
		c.Send(msg)
	}
}

// checkPlayerArenaEntry checks if any player has crossed into the arena (z < 12) from the lobby.
func (z *Zone) checkPlayerArenaEntry() bool {
	for _, p := range z.Players {
		if p.Alive && p.Position.Z < 12.0 {
			return true
		}
	}
	return false
}
