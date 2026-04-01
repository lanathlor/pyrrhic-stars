package zone

import (
	"context"
	"encoding/binary"
	"log/slog"
	"math"
	"math/rand"
	"sync"
	"time"

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
	StateResult
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

	State      GameFlowState
	ResultTimer float32

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

	// OnResultEnd is called when the arena result timer expires.
	// Set by the gateway to trigger zone transfer back to hub.
	OnResultEnd func(zoneID string)
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
	defer z.mu.Unlock()
	z.clients[c.PeerID] = c
	if p, ok := z.Players[c.PeerID]; !ok {
		np := entity.NewPlayer(c.PeerID, "gunner")
		np.Username = c.Username
		z.Players[c.PeerID] = np
	} else {
		p.Username = c.Username
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
		z.damageEvents = z.damageEvents[:0]
		switch z.State {
		case StateLobby:
			z.tickLobby()
		case StateSpawned:
			z.tickSpawned()
		case StateFight:
			z.tickFight()
		case StateResult:
			z.tickResult()
		}
	}

	// 3. Broadcast state
	z.broadcastState()
}

func (z *Zone) handleInput(inp inputMsg) {
	switch inp.Opcode {
	case message.OpPlayerInput:
		z.handlePlayerInput(inp.PeerID, inp.Payload)
	case message.OpAbilityInput:
		z.handleAbilityInput(inp.PeerID, inp.Payload)
	case message.OpInteractInput:
		z.handleInteractInput(inp.PeerID, inp.Payload)
	}
}

// =============================================================================
// Input handlers
// =============================================================================

func (z *Zone) handlePlayerInput(peerID uint16, payload []byte) {
	if len(payload) < 16 {
		return
	}
	p, ok := z.Players[peerID]
	if !ok {
		return
	}

	posX := math.Float32frombits(binary.LittleEndian.Uint32(payload[0:4]))
	posY := math.Float32frombits(binary.LittleEndian.Uint32(payload[4:8]))
	posZ := math.Float32frombits(binary.LittleEndian.Uint32(payload[8:12]))
	rotY := math.Float32frombits(binary.LittleEndian.Uint32(payload[12:16]))

	var tick uint32
	if len(payload) >= 20 {
		tick = binary.LittleEndian.Uint32(payload[16:20])
	}

	// Client-authoritative: accept position, clamp to boundaries
	p.Position = entity.Vec3{X: posX, Y: posY, Z: posZ}
	// Clamp to zone boundaries
	if z.Type == ZoneTypeHub {
		p.Position.X = entity.Clamp(p.Position.X, -14.5, 14.5)
		p.Position.Z = entity.Clamp(p.Position.Z, -9.5, 14.5)
	} else {
		p.Position.X = entity.Clamp(p.Position.X, -19.5, 19.5)
		p.Position.Z = entity.Clamp(p.Position.Z, -14.5, 24.5)
	}
	p.RotationY = rotY
	p.LastInput = &entity.PlayerInput{PosX: posX, PosY: posY, PosZ: posZ, RotY: rotY, Tick: tick}

	// Parse animation name + speed (appended after tick)
	off := 20
	if off < len(payload) {
		animLen := int(payload[off])
		off++
		if off+animLen <= len(payload) {
			p.AnimName = string(payload[off : off+animLen])
			off += animLen
		}
		if off+4 <= len(payload) {
			p.AnimSpeed = math.Float32frombits(binary.LittleEndian.Uint32(payload[off : off+4]))
		}
	}
}

func (z *Zone) handleAbilityInput(peerID uint16, payload []byte) {
	if len(payload) < 1 {
		return
	}
	p, ok := z.Players[peerID]
	if !ok || !p.Alive {
		return
	}
	if z.State != StateFight {
		return
	}

	action := payload[0]
	switch action {
	case entity.ActionShoot:
		// Gunner: hitscan, gated by fire cooldown
		if p.ClassName == "gunner" && p.FireCooldown <= 0 {
			p.FireCooldown = 0.18
			// If payload includes aim pitch, use it
			if len(payload) >= 5 {
				p.AimPitch = math.Float32frombits(binary.LittleEndian.Uint32(payload[1:5]))
			}
			evt := combat.ResolvePlayerAttackOnEnemy(p, z.Enemy)
			if evt != nil {
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
			evt := combat.ResolvePlayerAttackOnEnemy(p, z.Enemy)
			if evt != nil {
				z.damageEvents = append(z.damageEvents, *evt)
			}
		}
	case entity.ActionHeavy:
		if p.ClassName == "vanguard" && p.FireCooldown <= 0 {
			p.FireCooldown = 0.8
			evt := combat.ResolvePlayerAttackOnEnemy(p, z.Enemy)
			if evt != nil {
				z.damageEvents = append(z.damageEvents, *evt)
			}
		}
	}
}

func (z *Zone) handleInteractInput(peerID uint16, payload []byte) {
	if len(payload) < 1 {
		return
	}
	p, ok := z.Players[peerID]
	if !ok {
		return
	}

	action := payload[0]
	switch action {
	case message.InteractClassSelect:
		if len(payload) < 3 {
			return
		}
		nameLen := int(payload[1])
		if len(payload) < 2+nameLen {
			return
		}
		className := string(payload[2 : 2+nameLen])
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

// arenaObstacle represents a rectangular obstacle in the arena (XZ plane).
type arenaObstacle struct {
	cx, cz float32 // center
	hx, hz float32 // half-extents
}

// Arena obstacles: 6 pillars (1.5x1.5) + 4 cover boxes
var arenaObstacles = []arenaObstacle{
	// Pillars
	{cx: -8, cz: -6, hx: 0.75, hz: 0.75},
	{cx: 8, cz: -6, hx: 0.75, hz: 0.75},
	{cx: -8, cz: 6, hx: 0.75, hz: 0.75},
	{cx: 8, cz: 6, hx: 0.75, hz: 0.75},
	{cx: 0, cz: -10, hx: 0.75, hz: 0.75},
	{cx: 0, cz: 10, hx: 0.75, hz: 0.75},
	// Cover boxes
	{cx: -5, cz: -2, hx: 1.5, hz: 0.5},
	{cx: 5, cz: 2, hx: 1.5, hz: 0.5},
	{cx: -12, cz: 0, hx: 0.5, hz: 1.5},
	{cx: 12, cz: 0, hx: 0.5, hz: 1.5},
}

const enemyRadius float32 = 1.0

// pushOutOfObstacles resolves collisions between a position and arena obstacles.
func (z *Zone) pushOutOfObstacles(pos *entity.Vec3) {
	for _, obs := range arenaObstacles {
		// Expand obstacle by enemy radius (Minkowski sum)
		exHx := obs.hx + enemyRadius
		exHz := obs.hz + enemyRadius
		dx := pos.X - obs.cx
		dz := pos.Z - obs.cz
		if dx > -exHx && dx < exHx && dz > -exHz && dz < exHz {
			// Inside — push out along shortest axis
			pushX := exHx - abs32(dx)
			pushZ := exHz - abs32(dz)
			if pushX < pushZ {
				if dx > 0 {
					pos.X = obs.cx + exHx
				} else {
					pos.X = obs.cx - exHx
				}
			} else {
				if dz > 0 {
					pos.Z = obs.cz + exHz
				} else {
					pos.Z = obs.cz - exHz
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
			if dist <= entity.MeleeRange {
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
			if combat.CheckAoERadius(e.Position, p.Position, radius) {
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
		z.State = StateResult
		z.ResultTimer = 3.0
		z.broadcastGameFlow(message.FlowShowResult, "VICTORY")
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
		z.State = StateResult
		z.ResultTimer = 3.0
		z.broadcastGameFlow(message.FlowShowResult, "YOU DIED")
	}
}

// =============================================================================
// Result tick
// =============================================================================

func (z *Zone) tickResult() {
	z.ResultTimer -= DeltaTime
	if z.ResultTimer <= 0 {
		if z.OnResultEnd != nil {
			z.OnResultEnd(z.ID)
		} else {
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
		z.broadcastHubState()
		return
	}

	switch z.State {
	case StateLobby:
		z.broadcastLobbyState()
	case StateSpawned, StateFight, StateResult:
		z.broadcastWorldState()
		z.broadcastDamageEvents()
	}
}

func (z *Zone) broadcastLobbyState() {
	// Format: [player_count:1] for each: [peer_id:2][class_len:1][class:...][ready:1]
	buf := make([]byte, 0, 256)
	buf = append(buf, byte(len(z.Players)))
	for _, p := range z.Players {
		b := make([]byte, 2)
		binary.LittleEndian.PutUint16(b, p.PeerID)
		buf = append(buf, b...)
		className := []byte(p.ClassName)
		buf = append(buf, byte(len(className)))
		buf = append(buf, className...)
		if p.Ready {
			buf = append(buf, 1)
		} else {
			buf = append(buf, 0)
		}
	}
	msg := message.Encode(message.OpLobbyState, 0, buf)
	for _, c := range z.clients {
		c.Send(msg)
	}
}

func (z *Zone) broadcastHubState() {
	// Format: [tick:4 LE][player_count:1] per player:
	//   [peer_id:2 LE][x:f32][y:f32][z:f32][rot_y:f32]
	//   [class_len:1][class:...][name_len:1][name:...]
	//   [anim_len:1][anim:...][anim_speed:f32]
	buf := make([]byte, 0, 512)
	buf = appendUint32(buf, z.Tick)
	buf = append(buf, byte(len(z.Players)))
	for _, p := range z.Players {
		buf = appendUint16(buf, p.PeerID)
		buf = appendFloat32(buf, p.Position.X)
		buf = appendFloat32(buf, p.Position.Y)
		buf = appendFloat32(buf, p.Position.Z)
		buf = appendFloat32(buf, p.RotationY)
		classBytes := []byte(p.ClassName)
		buf = append(buf, byte(len(classBytes)))
		buf = append(buf, classBytes...)
		nameBytes := []byte(p.Username)
		buf = append(buf, byte(len(nameBytes)))
		buf = append(buf, nameBytes...)
		animBytes := []byte(p.AnimName)
		buf = append(buf, byte(len(animBytes)))
		buf = append(buf, animBytes...)
		buf = appendFloat32(buf, p.AnimSpeed)
	}
	msg := message.Encode(message.OpHubState, 0, buf)
	for _, c := range z.clients {
		c.Send(msg)
	}
}

func (z *Zone) broadcastWorldState() {
	// Format: [tick:4][player_count:1]...players...[enemy_alive:1]...enemy...[proj_count:1]...projs...
	buf := make([]byte, 0, 512)

	// Tick
	b4 := make([]byte, 4)
	binary.LittleEndian.PutUint32(b4, z.Tick)
	buf = append(buf, b4...)

	// Players
	buf = append(buf, byte(len(z.Players)))
	for _, p := range z.Players {
		buf = appendUint16(buf, p.PeerID)
		buf = appendFloat32(buf, p.Position.X)
		buf = appendFloat32(buf, p.Position.Y)
		buf = appendFloat32(buf, p.Position.Z)
		buf = appendFloat32(buf, p.RotationY)
		buf = appendFloat32(buf, p.Health)
		buf = append(buf, byte(p.State))
		animBytes := []byte(p.AnimName)
		buf = append(buf, byte(len(animBytes)))
		buf = append(buf, animBytes...)
		buf = appendFloat32(buf, p.AnimSpeed)
	}

	// Enemy
	e := z.Enemy
	if e.Alive {
		buf = append(buf, 1)
	} else {
		buf = append(buf, 0)
	}
	buf = appendUint16(buf, e.ID)
	buf = appendFloat32(buf, e.Position.X)
	buf = appendFloat32(buf, e.Position.Y)
	buf = appendFloat32(buf, e.Position.Z)
	buf = appendFloat32(buf, e.RotationY)
	buf = appendFloat32(buf, e.Health)
	buf = append(buf, byte(e.State))
	buf = append(buf, byte(e.Phase))
	buf = appendFloat32(buf, e.RangedTargetPos.X)
	buf = appendFloat32(buf, e.RangedTargetPos.Y)
	buf = appendFloat32(buf, e.RangedTargetPos.Z)
	buf = appendFloat32(buf, e.ChargeDirection.X)
	buf = appendFloat32(buf, e.ChargeDirection.Y)
	buf = appendFloat32(buf, e.ChargeDirection.Z)

	// Projectiles
	buf = append(buf, byte(len(z.Projectiles)))
	for _, proj := range z.Projectiles {
		buf = appendUint32(buf, proj.ID)
		buf = appendFloat32(buf, proj.Position.X)
		buf = appendFloat32(buf, proj.Position.Y)
		buf = appendFloat32(buf, proj.Position.Z)
		buf = appendFloat32(buf, proj.Direction.X)
		buf = appendFloat32(buf, proj.Direction.Y)
		buf = appendFloat32(buf, proj.Direction.Z)
	}

	msg := message.Encode(message.OpWorldState, 0, buf)
	for _, c := range z.clients {
		c.Send(msg)
	}
}

func (z *Zone) broadcastDamageEvents() {
	for _, evt := range z.damageEvents {
		buf := make([]byte, 0, 20)
		buf = appendUint16(buf, evt.TargetPeerID)
		buf = appendFloat32(buf, evt.Amount)
		buf = appendFloat32(buf, evt.HitPos.X)
		buf = appendFloat32(buf, evt.HitPos.Y)
		buf = appendFloat32(buf, evt.HitPos.Z)
		buf = append(buf, evt.SourceType)
		msg := message.Encode(message.OpDamageEvent, 0, buf)
		for _, c := range z.clients {
			c.Send(msg)
		}
	}
}

func (z *Zone) broadcastGameFlow(flowType uint8, text string) {
	z.mu.Lock()
	defer z.mu.Unlock()
	buf := []byte{flowType}
	textBytes := []byte(text)
	buf = append(buf, byte(len(textBytes)))
	buf = append(buf, textBytes...)
	msg := message.Encode(message.OpGameFlowEvent, 0, buf)
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

// Binary encoding helpers
func appendFloat32(buf []byte, v float32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, math.Float32bits(v))
	return append(buf, b...)
}

func appendUint16(buf []byte, v uint16) []byte {
	b := make([]byte, 2)
	binary.LittleEndian.PutUint16(b, v)
	return append(buf, b...)
}

func appendUint32(buf []byte, v uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	return append(buf, b...)
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
