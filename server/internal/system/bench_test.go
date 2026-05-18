package system

import (
	"math"
	"math/rand/v2"
	"sync"
	"testing"

	"codex-online/server/internal/ability"
	"codex-online/server/internal/codec"
	"codex-online/server/internal/combat"
	"codex-online/server/internal/enemyai"
	"codex-online/server/internal/entity"
	"codex-online/server/internal/item"
	"codex-online/server/internal/level"
	"codex-online/server/internal/message"
)

// benchWorld creates a realistic fight scenario: 5 players, 9 enemies, 10 projectiles.
func benchWorld() *World {
	lvl := level.NewArenaLevel()

	players := make(map[uint16]*entity.Player, 5)
	for i := uint16(1); i <= 5; i++ {
		p := entity.NewPlayer(i, entity.ClassGunner)
		p.Position = entity.Vec3{X: float32(i) * 2, Y: 0.1, Z: 5}
		p.RotationY = 0
		p.AimPitch = 0
		p.Health = p.MaxHealth * 0.8
		p.Cooldowns["fire_shot"] = 0.1
		p.AddBuff(entity.ActiveBuff{ID: "overclock", Type: entity.BuffCooldownMult, Value: 0.556, Duration: 3.0})
		p.Cooldowns["overclock"] = 5.0
		players[i] = p
	}

	enemies := make([]*entity.Enemy, 9)
	brains := make([]enemyai.BrainTicker, 9)
	for i := 0; i < 9; i++ {
		var def *enemyai.EnemyDef
		if i == 8 {
			def = enemyai.DefRegistry["guard_captain"]
		} else if i%2 == 0 {
			def = enemyai.DefRegistry["hallway_melee"]
		} else {
			def = enemyai.DefRegistry["hallway_ranged"]
		}
		e := entity.NewEnemy(uint16(100+i), def.MaxHealth, def.Name)
		e.Position = entity.Vec3{X: float32(i-4) * 3, Y: 0.1, Z: float32(20 + i*2)}
		e.State = entity.EnemyChase
		e.Alive = true
		e.AddThreat(1, 50)
		e.AddThreat(2, 30)
		enemies[i] = e

		b := enemyai.NewBrain(def, e, ability.NewEngine(nil))
		b.BoundsMinX = lvl.EnemyBoundsMinX
		b.BoundsMaxX = lvl.EnemyBoundsMaxX
		b.BoundsMinZ = lvl.EnemyBoundsMinZ
		b.BoundsMaxZ = lvl.EnemyBoundsMaxZ
		brains[i] = b
	}

	projs := make([]*entity.Projectile, 10)
	for i := 0; i < 10; i++ {
		projs[i] = entity.NewProjectile(uint32(i+1), 0, i%9,
			entity.Vec3{X: float32(i - 5), Y: 1.5, Z: float32(15 + i)},
			entity.Vec3{X: 0, Z: -1},
			20, 15, 5.0)
	}

	return &World{
		ZoneType:      1,
		TickNum:       100,
		State:         StateFight,
		Players:       players,
		Enemies:       enemies,
		Brains:        brains,
		Projectiles:   projs,
		Level:         lvl,
		Clients:       make(map[uint16]*Client),
		AbilityEngine: ability.NewEngine(nil),
		PatternEngine: combat.NewPatternEngine(),
		PatternRng:    rand.New(rand.NewPCG(42, 0)),
		// Pre-allocate pooled buffers so broadcast doesn't allocate in the tick loop.
		SendBuf:     make([]byte, 0, 4096),
		DamageBuf:   make([]byte, 0, 256),
		GameFlowBuf: make([]byte, 0, 256),
		LobbyBuf:    make([]byte, 0, 512),
	}
}

// --- System benchmarks ---

func BenchmarkCombatSystemTick(b *testing.B) {
	w := benchWorld()
	sys := CombatSystem{}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.DamageEvents = w.DamageEvents[:0]
		sys.Tick(w, 0.05)
	}
}

func BenchmarkAISystemTick(b *testing.B) {
	w := benchWorld()
	sys := AISystem{}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.DamageEvents = w.DamageEvents[:0]
		w.Projectiles = w.Projectiles[:0]
		sys.Tick(w, 0.05)
	}
}

func BenchmarkPhysicsSystemTick(b *testing.B) {
	w := benchWorld()
	sys := PhysicsSystem{}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Reset projectiles each iteration
		for j := range w.Projectiles {
			w.Projectiles[j].Alive = true
			w.Projectiles[j].Timer = 0
			w.Projectiles[j].Position = entity.Vec3{X: float32(j - 5), Y: 1.5, Z: float32(15 + j)}
		}
		w.DamageEvents = w.DamageEvents[:0]
		sys.Tick(w, 0.05)
	}
}

func BenchmarkInputSystemTick(b *testing.B) {
	w := benchWorld()
	sys := InputSystem{}
	// 5 movement inputs
	inputs := make([]InputMsg, 5)
	var senderBuffer = make([]byte, 0, 1024)

	for i := uint16(1); i <= 5; i++ {
		inputs[i-1] = InputMsg{
			PeerID:  i,
			Opcode:  0x0030,
			Payload: codec.EncodePlayerInput(senderBuffer, float32(i)*2, 0.1, 5, 0, 100, 0, 0),
		}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.InputQueue = inputs
		sys.Tick(w, 0.05)
	}
}

func BenchmarkHandlePlayerInput(b *testing.B) {
	w := benchWorld()
	payload := codec.EncodePlayerInput(nil, 3.0, 0.1, 6.0, 0.5, 101, 0, 0.1)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Reset player position so teleport check doesn't reject
		w.Players[1].Position = entity.Vec3{X: 2, Y: 0.1, Z: 5}
		handlePlayerInput(w, 1, payload)
	}
}

func BenchmarkHandleAbilityInput(b *testing.B) {
	w := benchWorld()
	payload := codec.EncodeAbilityInput(entity.ActionShoot, 0.1, 0.5)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		delete(w.Players[1].Cooldowns, "fire_shot")
		w.DamageEvents = w.DamageEvents[:0]
		handleAbilityInput(w, 1, payload)
	}
}

func BenchmarkHandleInteractInput(b *testing.B) {
	w := benchWorld()
	payload := codec.EncodeInteractInput(message.InteractClassSelect, entity.ClassVanguard)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handleInteractInput(w, 1, payload)
	}
}

func BenchmarkHandleRespawnRequest(b *testing.B) {
	w := benchWorld()
	w.State = StateFightOver
	payload := codec.EncodeRespawnRequest(0) // arena respawn
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Players[1].Alive = false
		handleRespawnRequest(w, 1, payload)
	}
}

func BenchmarkFullTickPipeline(b *testing.B) {
	w := benchWorld()
	inputSys := InputSystem{}
	combatSys := CombatSystem{}
	aiSys := AISystem{}
	physicsSys := PhysicsSystem{}

	inputs := make([]InputMsg, 5)
	for i := uint16(1); i <= 5; i++ {
		inputs[i-1] = InputMsg{
			PeerID:  i,
			Opcode:  0x0030,
			Payload: codec.EncodePlayerInput(nil, float32(i)*2, 0.1, 5, 0, 100, 0, 0),
		}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.InputQueue = inputs
		w.DamageEvents = w.DamageEvents[:0]
		w.GameFlowEvents = w.GameFlowEvents[:0]

		inputSys.Tick(w, 0.05)
		combatSys.Tick(w, 0.05)
		aiSys.Tick(w, 0.05)
		physicsSys.Tick(w, 0.05)
	}
}

// --- Codec benchmarks ---

func BenchmarkEncodeWorldState(b *testing.B) {
	w := benchWorld()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = codec.EncodeWorldState(w.TickNum, w.Players, w.Enemies, w.Projectiles)
	}
}

// Narrowing: AppendEncodeWorldState with a pre-sized buffer (the production path).
// If this is 0 allocs, the allocs in EncodeWorldState come from the wrapper's buffer management.
func BenchmarkAppendEncodeWorldState(b *testing.B) {
	w := benchWorld()
	buf := make([]byte, 0, 4096)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf = buf[:0]
		buf = codec.AppendEncodeWorldState(buf, w.TickNum, w.Players, w.Enemies, w.Projectiles, nil)
	}
}

func BenchmarkDecodePlayerInput(b *testing.B) {
	payload := codec.EncodePlayerInput(nil, 5.0, 0.1, 10.0, 1.5, 500, 0, 0.2)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = codec.DecodePlayerInput(payload)
	}
}

func BenchmarkDecodeAbilityInput(b *testing.B) {
	payload := codec.EncodeAbilityInput(0, 0.5, 1.2)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = codec.DecodeAbilityInput(payload)
	}
}

func BenchmarkEncodeDamageEvent(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = codec.EncodeDamageEvent(1, 0, 25.0, 5.0, 1.5, 3.0, 1)
	}
}

// Narrowing: broadcastDamageEvents allocates the payload via EncodeDamageEvent (make([]byte,0,21))
// then copies into the pooled DamageBuf. This bench isolates that per-event cost.
func BenchmarkBroadcastDamageEventPooled(b *testing.B) {
	w := benchWorld()
	w.DamageEvents = []combat.DamageEvent{
		{TargetPeerID: 100, SourcePeerID: 1, Amount: 25, HitPos: entity.Vec3{X: 1, Y: 0, Z: 2}, SourceType: 0},
	}
	// Add test clients so broadcast actually runs
	for i := uint16(1); i <= 5; i++ {
		w.Clients[i] = &Client{PeerID: i, Send: func([]byte) {}}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		broadcastDamageEvents(w)
	}
}

// --- Combat hit check benchmarks ---

func BenchmarkCheckHitscan(b *testing.B) {
	origin := entity.Vec3{X: 0, Y: 1.6, Z: 5}
	dir := entity.Vec3{X: 0, Y: 0, Z: -1}
	target := entity.Vec3{X: 0, Y: 0.1, Z: 0}
	obs := level.NewArenaLevel().Obstacles
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		combat.CheckHitscan(origin, dir, target, 0.5, 50, obs)
	}
}

func BenchmarkCheckMeleeArc(b *testing.B) {
	attacker := entity.Vec3{X: 0, Y: 0.1, Z: 0}
	forward := entity.Vec3{X: 0, Z: -1}
	target := entity.Vec3{X: 1, Y: 0.1, Z: -2}
	obs := level.NewArenaLevel().Obstacles
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		combat.CheckMeleeArc(attacker, forward, target, 3.0, 180, obs)
	}
}

func BenchmarkCheckAoERadius(b *testing.B) {
	center := entity.Vec3{X: 0, Y: 0.1, Z: 0}
	target := entity.Vec3{X: 3, Y: 0.1, Z: 0}
	obs := level.NewArenaLevel().Obstacles
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		combat.CheckAoERadius(center, target, 5.0, obs)
	}
}

func BenchmarkSegmentHitsObstacle(b *testing.B) {
	a := entity.Vec3{X: -5, Y: 1.0, Z: 5}
	target := entity.Vec3{X: 5, Y: 1.0, Z: -5}
	obs := level.NewArenaLevel().Obstacles
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		combat.SegmentHitsObstacle(a, target, obs)
	}
}

func BenchmarkPushOutOfObstacles(b *testing.B) {
	obs := level.NewArenaLevel().Obstacles
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pos := entity.Vec3{X: -7.5, Y: 0.1, Z: -5.5}
		combat.PushOutOfObstacles(&pos, obs, 1.0)
	}
}

// --- Multi-instance benchmarks ---
// Simulates a server running N concurrent arena instances simultaneously.
// Each instance: 5 players, 9 enemies (8 trash + 1 boss), 10 projectiles,
// active boss fight (BossGateActive=true, boss in chase), full system pipeline,
// plus WorldState broadcast to all connected clients.

// benchArenaInstance creates a full arena fight scenario ready for ticking.
// Players are actively in combat, boss gate is sealed, boss is fighting.
func benchArenaInstance(instanceID uint16) *World {
	lvl := level.NewArenaLevel()

	// 5 players spread across the boss room
	players := make(map[uint16]*entity.Player, 5)
	for i := uint16(0); i < 5; i++ {
		peerID := instanceID*10 + i + 1
		p := entity.NewPlayer(peerID, entity.ClassGunner)
		p.Position = entity.Vec3{X: float32(i) * 2, Y: 0.1, Z: 5}
		p.RotationY = 0
		p.AimPitch = 0
		p.Health = p.MaxHealth * 0.8
		p.Cooldowns["fire_shot"] = 0.05
		p.AddBuff(entity.ActiveBuff{ID: "overclock", Type: entity.BuffCooldownMult, Value: 0.556, Duration: 3.0})
		players[peerID] = p
	}

	// 8 trash enemies (alternating melee/ranged) + 1 boss
	enemies := make([]*entity.Enemy, 9)
	brains := make([]enemyai.BrainTicker, 9)
	for i := 0; i < 9; i++ {
		var def *enemyai.EnemyDef
		if i == 8 {
			def = enemyai.DefRegistry["guard_captain"]
		} else if i%2 == 0 {
			def = enemyai.DefRegistry["hallway_melee"]
		} else {
			def = enemyai.DefRegistry["hallway_ranged"]
		}
		e := entity.NewEnemy(uint16(10000+int(instanceID)*10+i), def.MaxHealth, def.Name)
		if i == 8 {
			e.IsBoss = true
			e.Position = entity.Vec3{X: 0, Y: 0.1, Z: 3}
		} else {
			e.Position = entity.Vec3{X: float32(i-4) * 3, Y: 0.1, Z: float32(10 + i)}
		}
		e.State = entity.EnemyChase
		e.Alive = true
		e.AddThreat(instanceID*10+1, 50)
		e.AddThreat(instanceID*10+2, 30)
		enemies[i] = e

		b := enemyai.NewBrain(def, e, ability.NewEngine(nil))
		b.BoundsMinX = lvl.EnemyBoundsMinX
		b.BoundsMaxX = lvl.EnemyBoundsMaxX
		b.BoundsMinZ = lvl.EnemyBoundsMinZ
		b.BoundsMaxZ = lvl.EnemyBoundsMaxZ
		brains[i] = b
	}

	// 10 projectiles mid-flight
	projs := make([]*entity.Projectile, 10)
	for i := 0; i < 10; i++ {
		projs[i] = entity.NewProjectile(uint32(1000+int(instanceID)*10+i), 0, i%9,
			entity.Vec3{X: float32(i - 5), Y: 1.5, Z: float32(7 + i)},
			entity.Vec3{X: 0, Z: -1},
			20, 15, 5.0)
	}

	// Clients for network broadcast — no-op Send to avoid harness allocations
	// polluting benchmark results. The full encode + broadcast path still runs;
	// only the test-side accumulation buffer is removed.
	clients := make(map[uint16]*Client, 5)
	for i := uint16(0); i < 5; i++ {
		peerID := instanceID*10 + i + 1
		clients[peerID] = &Client{
			PeerID: peerID,
			Send:   func([]byte) {},
		}
	}

	return &World{
		ZoneType:       1,
		TickNum:        100,
		State:          StateFight,
		BossGateActive: true,
		Players:        players,
		Enemies:        enemies,
		Brains:         brains,
		Projectiles:    projs,
		Level:          lvl,
		Clients:        clients,
		AbilityEngine:  ability.NewEngine(nil),
		// Pre-allocate pooled buffers so broadcast doesn't allocate in the tick loop.
		SendBuf:     make([]byte, 0, 4096),
		DamageBuf:   make([]byte, 0, 256),
		GameFlowBuf: make([]byte, 0, 256),
		LobbyBuf:    make([]byte, 0, 512),
	}
}

// buildInputs creates 5 player inputs for an instance.
func buildInputs(instanceID uint16) []InputMsg {
	inputs := make([]InputMsg, 5)
	for i := uint16(0); i < 5; i++ {
		peerID := instanceID*10 + i + 1
		inputs[i] = InputMsg{
			PeerID:  peerID,
			Opcode:  0x0030,
			Payload: codec.EncodePlayerInput(nil, float32(i)*2, 0.1, 5, 0, 100, 0, 0),
		}
	}
	return inputs
}

// tickInstance simulates one full tick of an arena instance.
func tickInstance(w *World, inputs []InputMsg) {
	w.DamageEvents = w.DamageEvents[:0]
	w.GameFlowEvents = w.GameFlowEvents[:0]

	(&InputSystem{}).Tick(w, 0.05)
	(&GameFlowSystem{}).Tick(w, 0.05)
	(&AISystem{}).Tick(w, 0.05)
	(&CombatSystem{}).Tick(w, 0.05)
	(&PhysicsSystem{}).Tick(w, 0.05)
	(&NetworkSystem{}).Tick(w, 0.05)

	// Drain queued inputs for next tick
	_ = inputs
}

// BenchmarkMultiInstance5 simulates 5 concurrent arena instances (25 players, 5 bosses).
// Represents a server handling 5 simultaneous groups in dungeons.
func BenchmarkMultiInstance5(b *testing.B) {
	instances := make([]*World, 5)
	allInputs := make([][]InputMsg, 5)
	for i := range instances {
		instances[i] = benchArenaInstance(uint16(i))
		allInputs[i] = buildInputs(uint16(i))
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := range instances {
			instances[j].InputQueue = allInputs[j]
			tickInstance(instances[j], allInputs[j])
		}
	}
}

// BenchmarkMultiInstance10 simulates 10 concurrent arena instances (50 players, 10 bosses).
func BenchmarkMultiInstance10(b *testing.B) {
	instances := make([]*World, 10)
	allInputs := make([][]InputMsg, 10)
	for i := range instances {
		instances[i] = benchArenaInstance(uint16(i))
		allInputs[i] = buildInputs(uint16(i))
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := range instances {
			instances[j].InputQueue = allInputs[j]
			tickInstance(instances[j], allInputs[j])
		}
	}
}

// BenchmarkMultiInstance20 simulates 20 concurrent arena instances (100 players, 20 bosses).
// Heavy load: equivalent to 4 full raid groups per boss simultaneously.
func BenchmarkMultiInstance20(b *testing.B) {
	instances := make([]*World, 20)
	allInputs := make([][]InputMsg, 20)
	for i := range instances {
		instances[i] = benchArenaInstance(uint16(i))
		allInputs[i] = buildInputs(uint16(i))
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := range instances {
			instances[j].InputQueue = allInputs[j]
			tickInstance(instances[j], allInputs[j])
		}
	}
}

// BenchmarkMultiInstance50 simulates 50 concurrent arena instances (250 players, 50 bosses).
// Extremely heavy: ~2 full boss encounters per second across the server.
func BenchmarkMultiInstance50(b *testing.B) {
	instances := make([]*World, 50)
	allInputs := make([][]InputMsg, 50)
	for i := range instances {
		instances[i] = benchArenaInstance(uint16(i))
		allInputs[i] = buildInputs(uint16(i))
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := range instances {
			instances[j].InputQueue = allInputs[j]
			tickInstance(instances[j], allInputs[j])
		}
	}
}

// BenchmarkMultiInstance100 simulates 100 concurrent arena instances (500 players, 100 bosses).
// Stress test: all CPU cores under heavy load with significant GC pressure.
func BenchmarkMultiInstance100(b *testing.B) {
	instances := make([]*World, 100)
	allInputs := make([][]InputMsg, 100)
	for i := range instances {
		instances[i] = benchArenaInstance(uint16(i))
		allInputs[i] = buildInputs(uint16(i))
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := range instances {
			instances[j].InputQueue = allInputs[j]
			tickInstance(instances[j], allInputs[j])
		}
	}
}

// BenchmarkMultiInstance50Parallel simulates 50 concurrent instances where each
// instance ticks in its own goroutine. Represents a real server where each zone
// runs in a separate goroutine and all tick concurrently across CPU cores.
func BenchmarkMultiInstance50Parallel(b *testing.B) {
	instances := make([]*World, 50)
	allInputs := make([][]InputMsg, 50)
	for i := range instances {
		instances[i] = benchArenaInstance(uint16(i))
		allInputs[i] = buildInputs(uint16(i))
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		for j := range instances {
			w, inputs := instances[j], allInputs[j]
			wg.Go(func() {
				tickInstance(w, inputs)
			})
		}
		wg.Wait()
	}
}

// BenchmarkMultiInstance100Parallel simulates 100 concurrent instances in parallel.
// Maximum stress test with full goroutine parallelism.
func BenchmarkMultiInstance100Parallel(b *testing.B) {
	instances := make([]*World, 100)
	allInputs := make([][]InputMsg, 100)
	for i := range instances {
		instances[i] = benchArenaInstance(uint16(i))
		allInputs[i] = buildInputs(uint16(i))
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		for j := range instances {
			w, inputs := instances[j], allInputs[j]
			wg.Go(func() {
				tickInstance(w, inputs)
			})
		}
		wg.Wait()
	}
}

// BenchmarkBroadcastOnly measures just the network broadcast overhead:
// encode WorldState once, send to 5 clients. No game simulation.
func BenchmarkBroadcastOnly(b *testing.B) {
	w := benchArenaInstance(0)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		(&NetworkSystem{}).Tick(w, 0.05)
	}
}

// --- Vec3 benchmarks ---

func BenchmarkBrainTickChase(b *testing.B) {
	def := enemyai.DefRegistry["guard_captain"]
	e := entity.NewEnemy(0, def.MaxHealth, def.Name)
	e.State = entity.EnemyChase
	e.Position = entity.Vec3{X: 0, Y: 0.1, Z: 0}

	brain := enemyai.NewBrain(def, e, ability.NewEngine(nil))
	brain.BoundsMinX = -20
	brain.BoundsMaxX = 20
	brain.BoundsMinZ = -15
	brain.BoundsMaxZ = 50

	p := entity.NewPlayer(1, entity.ClassGunner)
	p.Position = entity.Vec3{X: 5, Y: 0.1, Z: 5}
	players := []*entity.Player{p}
	obs := level.NewArenaLevel().Obstacles

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.State = entity.EnemyChase
		e.ChaseTimer = 0
		e.Position = entity.Vec3{X: 0, Y: 0.1, Z: 0}
		brain.Tick(0.05, players, obs, func(_, _ entity.Vec3, _, _, _ float32) {}, nil)
	}
}

func BenchmarkBrainTickMeleeAttack(b *testing.B) {
	def := enemyai.DefRegistry["guard_captain"]
	e := entity.NewEnemy(0, def.MaxHealth, def.Name)
	brain := enemyai.NewBrain(def, e, ability.NewEngine(nil))
	brain.BoundsMinX = -20
	brain.BoundsMaxX = 20
	brain.BoundsMinZ = -15
	brain.BoundsMaxZ = 50

	p := entity.NewPlayer(1, entity.ClassGunner)
	p.Position = entity.Vec3{X: 0, Y: 0.1, Z: -2}
	players := []*entity.Player{p}
	obs := level.NewArenaLevel().Obstacles

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.State = entity.EnemyMeleeAttack
		e.StateTimer = 0
		e.ActiveAbility = 0
		e.RotationY = 0
		e.Position = entity.Vec3{X: 0, Y: 0.1, Z: 0}
		p.Health = p.MaxHealth
		p.Alive = true
		p.State = entity.PlayerStateMove
		brain.Tick(0.05, players, obs, nil, nil)
	}
}

// Narrowing: melee attack that misses (player out of range). If 0 allocs,
// the alloc in BrainTickMeleeAttack is the []DamageEvent append on hit.
func BenchmarkBrainTickMeleeAttackMiss(b *testing.B) {
	def := enemyai.DefRegistry["guard_captain"]
	e := entity.NewEnemy(0, def.MaxHealth, def.Name)
	brain := enemyai.NewBrain(def, e, ability.NewEngine(nil))
	brain.BoundsMinX = -20
	brain.BoundsMaxX = 20
	brain.BoundsMinZ = -15
	brain.BoundsMaxZ = 50

	p := entity.NewPlayer(1, entity.ClassGunner)
	p.Position = entity.Vec3{X: 0, Y: 0.1, Z: -50} // far away — miss
	players := []*entity.Player{p}
	obs := level.NewArenaLevel().Obstacles

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.State = entity.EnemyMeleeAttack
		e.StateTimer = 0
		e.ActiveAbility = 0
		e.RotationY = 0
		e.Position = entity.Vec3{X: 0, Y: 0.1, Z: 0}
		brain.Tick(0.05, players, obs, nil, nil)
	}
}

// --- Vec3 benchmarks ---

func BenchmarkVec3DistanceTo(b *testing.B) {
	a := entity.Vec3{X: 1, Y: 2, Z: 3}
	o := entity.Vec3{X: 10, Y: 20, Z: 30}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = a.DistanceTo(o)
	}
}

func BenchmarkVec3Normalized(b *testing.B) {
	v := entity.Vec3{X: 3, Y: 4, Z: 5}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = v.Normalized()
	}
}

// --- Pattern engine system benchmarks ---
// These measure the pattern engine cost within the full system pipeline context.

// BenchmarkPhysicsWithPatterns_5Active measures PhysicsSystem.Tick with 5 active
// patterns each spawning 16 projectiles per wave (80 new projectiles/tick).
func BenchmarkPhysicsWithPatterns_5Active(b *testing.B) {
	w := benchWorld()
	sys := PhysicsSystem{}

	spiralDef := &combat.PatternDef{
		Emitters: []combat.EmitterDef{{
			Type:          combat.EmitterRadial,
			Count:         16,
			Waves:         1000,
			WaveInterval:  0.05,
			OffsetPerWave: 10 * math.Pi / 180,
			Projectile: combat.ProjectileDef{
				Speed:           8,
				Damage:          10,
				Lifetime:        4,
				AngularVelocity: 0.3,
			},
		}},
	}
	for i := range 5 {
		w.PatternEngine.Spawn(spiralDef, "fire_spiral", 0, i%len(w.Enemies),
			entity.Vec3{X: float32(i) * 3, Z: 20}, entity.Vec3{Z: -1})
	}
	// Let first waves fire
	w.PatternEngine.Tick(0.05, w.PatternRng)
	w.PatternEngine.DrainSpawns()

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		// Reset projectiles and pattern timer to avoid accumulation
		w.Projectiles = w.Projectiles[:0]
		w.DamageEvents = w.DamageEvents[:0]
		for _, ap := range w.PatternEngine.Active {
			ap.WaveTimer = 0.04 // about to fire
			ap.WaveIdx = 1      // prevent completion
		}
		sys.Tick(w, 0.05)
	}
}

// BenchmarkPhysicsWithPatterns_10Active_32Count is the stress scenario:
// 10 patterns, 32 projectiles each = 320 new projectiles spawned per tick,
// plus 200 existing projectiles being ticked.
func BenchmarkPhysicsWithPatterns_10Active_32Count(b *testing.B) {
	w := benchWorld()
	sys := PhysicsSystem{}

	denseDef := &combat.PatternDef{
		Emitters: []combat.EmitterDef{{
			Type:          combat.EmitterRadial,
			Count:         32,
			Waves:         1000,
			WaveInterval:  0.05,
			OffsetPerWave: 8 * math.Pi / 180,
			Projectile: combat.ProjectileDef{
				Speed:           6,
				Damage:          8,
				Lifetime:        5,
				AngularVelocity: 0.4,
				Acceleration:    2.0,
				MaxSpeed:        12,
			},
		}},
	}
	for i := range 10 {
		w.PatternEngine.Spawn(denseDef, "void_storm", 0, i%len(w.Enemies),
			entity.Vec3{X: float32(i-5) * 4, Z: 25}, entity.Vec3{Z: -1})
	}
	// Fire first waves
	w.PatternEngine.Tick(0.05, w.PatternRng)
	w.PatternEngine.DrainSpawns()

	// Pre-populate 200 existing curved projectiles (from previous ticks)
	existingProjs := make([]*entity.Projectile, 200)
	for i := range existingProjs {
		existingProjs[i] = &entity.Projectile{
			ID:              uint32(10000 + i),
			OwnerID:         0,
			EnemyIdx:        i % len(w.Enemies),
			Position:        entity.Vec3{X: float32(i%20 - 10), Y: 1.5, Z: float32(i/20 + 5)},
			Direction:       entity.Vec3{X: 0.1, Z: -1},
			Speed:           8,
			Damage:          10,
			Lifetime:        5,
			Timer:           float32(i%40) * 0.05,
			Alive:           true,
			AngularVelocity: 0.3,
			Acceleration:    1.0,
			MaxSpeed:        12,
		}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		// Reset state for consistent measurement
		w.Projectiles = append(w.Projectiles[:0], existingProjs...)
		for _, p := range w.Projectiles {
			p.Alive = true
			p.Timer = float32(int(p.Timer*20)%40) * 0.05
		}
		w.DamageEvents = w.DamageEvents[:0]
		for _, ap := range w.PatternEngine.Active {
			ap.WaveTimer = 0.04
			ap.WaveIdx = 1 // prevent completion
		}
		sys.Tick(w, 0.05)
	}
}

// --- Gear/Stat hot-path benchmarks ---
// These measure the impact of gear stats on the tick pipeline.
// Players have full gear kits applied so TempoMult, CasterDamageMult,
// Plating, and Identity scaling are exercised every frame.

// applyGear sets gear stats on all players in the world, simulating
// a fully equipped party. This is not called per-tick — it's setup.
func applyGear(w *World) {
	for _, p := range w.Players {
		p.GearStats = entity.GearStats{
			Hull:     80,
			Output:   55,
			Plating:  15,
			Tempo:    20,
			Identity: 12,
			Mastery:  8,
		}
		p.RecalcStats()
		p.Health = p.MaxHealth * 0.8
	}
}

// BenchmarkCombatSystemTick_WithGear measures CombatSystem.Tick when all
// players have gear stats. Exercises TempoMult (cooldown scaling),
// CasterDamageMult (Output), and Identity scaling in ability ticks.
func BenchmarkCombatSystemTick_WithGear(b *testing.B) {
	w := benchWorld()
	applyGear(w)
	sys := CombatSystem{}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		w.DamageEvents = w.DamageEvents[:0]
		sys.Tick(w, 0.05)
	}
}

// BenchmarkFullTickPipeline_WithGear measures the full tick pipeline
// with gear stats active on all players. This is the primary hot-path
// benchmark for stat/gear regression detection.
func BenchmarkFullTickPipeline_WithGear(b *testing.B) {
	w := benchWorld()
	applyGear(w)
	inputSys := InputSystem{}
	combatSys := CombatSystem{}
	aiSys := AISystem{}
	physicsSys := PhysicsSystem{}

	inputs := make([]InputMsg, 5)
	for i := uint16(1); i <= 5; i++ {
		inputs[i-1] = InputMsg{
			PeerID:  i,
			Opcode:  0x0030,
			Payload: codec.EncodePlayerInput(nil, float32(i)*2, 0.1, 5, 0, 100, 0, 0),
		}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		w.InputQueue = inputs
		w.DamageEvents = w.DamageEvents[:0]
		w.GameFlowEvents = w.GameFlowEvents[:0]

		inputSys.Tick(w, 0.05)
		combatSys.Tick(w, 0.05)
		aiSys.Tick(w, 0.05)
		physicsSys.Tick(w, 0.05)
	}
}

// BenchmarkMultiInstance100_WithGear is BenchmarkMultiInstance100 but with
// all players equipped. Detects per-tick allocation regressions from stats.
func BenchmarkMultiInstance100_WithGear(b *testing.B) {
	instances := make([]*World, 100)
	allInputs := make([][]InputMsg, 100)
	for i := range instances {
		instances[i] = benchArenaInstance(uint16(i))
		applyGear(instances[i])
		allInputs[i] = buildInputs(uint16(i))
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		for j := range instances {
			instances[j].InputQueue = allInputs[j]
			tickInstance(instances[j], allInputs[j])
		}
	}
}

// --- ComputeStats integration benchmark ---
// Measures the cost of recomputing stats from equipped gear.
// This runs on gear change (equip/unequip), not per-tick.

func BenchmarkComputeStatsAndRecalc(b *testing.B) {
	// Seed item defs
	item.DefRegistry["bench_frame"] = &item.ItemDef{
		ID: "bench_frame", Slot: item.SlotFrame,
		StatLines: []item.StatLine{
			{Stat: item.StatHull, Value: 90},
			{Stat: item.StatPlating, Value: 12},
			{Stat: item.StatMastery, Value: 5},
		},
	}
	item.DefRegistry["bench_core"] = &item.ItemDef{
		ID: "bench_core", Slot: item.SlotPowerCore,
		StatLines: []item.StatLine{
			{Stat: item.StatHull, Value: 20},
			{Stat: item.StatOutput, Value: 22},
			{Stat: item.StatTempo, Value: 8},
		},
	}
	item.DefRegistry["bench_weapon"] = &item.ItemDef{
		ID: "bench_weapon", Slot: item.SlotPrimaryWeapon,
		StatLines: []item.StatLine{
			{Stat: item.StatOutput, Value: 25},
			{Stat: item.StatIdentity, Value: 10},
		},
	}
	item.DefRegistry["bench_tool"] = &item.ItemDef{
		ID: "bench_tool", Slot: item.SlotSecondaryTool,
		StatLines: []item.StatLine{
			{Stat: item.StatOutput, Value: 8},
			{Stat: item.StatTempo, Value: 8},
		},
	}
	item.DefRegistry["bench_aug"] = &item.ItemDef{
		ID: "bench_aug", Slot: item.SlotAugment,
		StatLines: []item.StatLine{
			{Stat: item.StatOutput, Value: 12},
			{Stat: item.StatIdentity, Value: 8},
		},
	}
	item.DefRegistry["bench_mod"] = &item.ItemDef{
		ID: "bench_mod", Slot: item.SlotModule,
		StatLines: []item.StatLine{
			{Stat: item.StatHull, Value: 40},
			{Stat: item.StatOutput, Value: 8},
		},
	}

	equipped := [item.SlotCount]*item.Item{
		{DefID: "bench_frame", ILvl: 35, Slot: item.SlotFrame},
		{DefID: "bench_core", ILvl: 35, Slot: item.SlotPowerCore},
		{DefID: "bench_weapon", ILvl: 35, Slot: item.SlotPrimaryWeapon},
		{DefID: "bench_tool", ILvl: 30, Slot: item.SlotSecondaryTool},
		{DefID: "bench_aug", ILvl: 32, Slot: item.SlotAugment},
		{DefID: "bench_mod", ILvl: 28, Slot: item.SlotModule},
	}

	p := entity.NewPlayer(1, entity.ClassGunner)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		stats := item.ComputeStats(equipped)
		p.GearStats = entity.GearStats{
			Hull:     stats.Hull,
			Output:   stats.Output,
			Plating:  stats.Plating,
			Tempo:    stats.Tempo,
			Identity: stats.Identity,
			Mastery:  stats.Mastery,
		}
		p.RecalcStats()
	}
}

// BenchmarkFullTickWithPatterns measures the complete tick pipeline with active patterns.
// Represents a realistic boss fight: 5 players, 9 enemies, boss firing patterns,
// 100+ projectiles in flight.
func BenchmarkFullTickWithPatterns(b *testing.B) {
	w := benchWorld()
	inputSys := InputSystem{}
	combatSys := CombatSystem{}
	aiSys := AISystem{}
	physicsSys := PhysicsSystem{}

	// Boss fires a spiral pattern
	spiralDef := &combat.PatternDef{
		Emitters: []combat.EmitterDef{{
			Type:          combat.EmitterRadial,
			Count:         16,
			Waves:         1000,
			WaveInterval:  0.05,
			OffsetPerWave: 12 * math.Pi / 180,
			Projectile: combat.ProjectileDef{
				Speed:           8,
				Damage:          12,
				Lifetime:        3.5,
				AngularVelocity: 0.3,
			},
		}},
	}
	w.PatternEngine.Spawn(spiralDef, "fireball_burst", 0, 8,
		entity.Vec3{X: 0, Y: 1.5, Z: 25}, entity.Vec3{Z: -1})
	// First wave
	w.PatternEngine.Tick(0.05, w.PatternRng)
	w.PatternEngine.DrainSpawns()

	// 50 existing projectiles in flight
	for i := range 50 {
		p := entity.NewProjectile(uint32(5000+i), 0, 8,
			entity.Vec3{X: float32(i%10 - 5), Y: 1.5, Z: float32(10 + i/10)},
			entity.Vec3{Z: -1}, 8, 12, 3.5)
		p.AngularVelocity = 0.3
		w.Projectiles = append(w.Projectiles, p)
	}

	inputs := make([]InputMsg, 5)
	for i := uint16(1); i <= 5; i++ {
		inputs[i-1] = InputMsg{
			PeerID:  i,
			Opcode:  0x0030,
			Payload: codec.EncodePlayerInput(nil, float32(i)*2, 0.1, 5, 0, 100, 0, 0),
		}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		w.InputQueue = inputs
		w.DamageEvents = w.DamageEvents[:0]
		w.GameFlowEvents = w.GameFlowEvents[:0]
		// Reset projectiles to consistent state
		for _, p := range w.Projectiles {
			p.Alive = true
			p.Timer = 0.5
		}
		for _, ap := range w.PatternEngine.Active {
			ap.WaveTimer = 0.04
			ap.WaveIdx = 1 // prevent completion
		}

		inputSys.Tick(w, 0.05)
		combatSys.Tick(w, 0.05)
		aiSys.Tick(w, 0.05)
		physicsSys.Tick(w, 0.05)
	}
}
