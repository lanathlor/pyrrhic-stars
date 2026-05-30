package system

import (
	"testing"

	"codex-online/server/internal/ability"
	"codex-online/server/internal/entity"
	"codex-online/server/internal/level"
	"codex-online/server/internal/message"
)

// makeArenaWorld creates a minimal arena world for gameflow tests.
func makeArenaWorld(t testing.TB, players map[uint16]*entity.Player, enemies []*entity.Enemy) *World {
	t.Helper()
	lvl := testArenaLevel(t)
	return &World{
		ZoneID:        testArenaZoneID,
		ZoneType:      1,
		TickNum:       100,
		State:         StateLobby,
		Players:       players,
		Enemies:       enemies,
		Level:         lvl,
		Clients:       make(map[uint16]*Client),
		AbilityEngine: ability.NewEngine(nil),
	}
}

// ---------------------------------------------------------------------------
// GameFlowSystem.Tick — hub skip
// ---------------------------------------------------------------------------

func TestGameFlowSystem_SkipsInHub(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassGunner)
	p.Ready = true
	w := &World{
		ZoneType: 0, // Hub
		State:    StateLobby,
		Players:  map[uint16]*entity.Player{1: p},
		Level:    testHubLevel(t),
	}

	sys := &GameFlowSystem{}
	sys.Tick(w, 0.05)

	if w.State != StateLobby {
		t.Errorf("hub zone should remain in StateLobby, got %d", w.State)
	}
	if len(w.GameFlowEvents) > 0 {
		t.Errorf("hub zone should produce no game flow events, got %d", len(w.GameFlowEvents))
	}
}

// ---------------------------------------------------------------------------
// tickLobby
// ---------------------------------------------------------------------------

func TestTickLobby(t *testing.T) {
	tests := []struct {
		name          string
		players       map[uint16]*entity.Player
		wantState     GameFlowState
		wantFlowEvent bool
	}{
		{
			name:          "no players stays in lobby",
			players:       map[uint16]*entity.Player{},
			wantState:     StateLobby,
			wantFlowEvent: false,
		},
		{
			name: "not all ready stays in lobby",
			players: func() map[uint16]*entity.Player {
				p1 := entity.NewPlayer(1, entity.ClassGunner)
				p1.Ready = true
				p2 := entity.NewPlayer(2, entity.ClassVanguard)
				p2.Ready = false
				return map[uint16]*entity.Player{1: p1, 2: p2}
			}(),
			wantState:     StateLobby,
			wantFlowEvent: false,
		},
		{
			name: "all ready transitions to StateSpawned",
			players: func() map[uint16]*entity.Player {
				p1 := entity.NewPlayer(1, entity.ClassGunner)
				p1.Ready = true
				p2 := entity.NewPlayer(2, entity.ClassVanguard)
				p2.Ready = true
				return map[uint16]*entity.Player{1: p1, 2: p2}
			}(),
			wantState:     StateSpawned,
			wantFlowEvent: true,
		},
		{
			name: "single player ready transitions",
			players: func() map[uint16]*entity.Player {
				p := entity.NewPlayer(1, entity.ClassGunner)
				p.Ready = true
				return map[uint16]*entity.Player{1: p}
			}(),
			wantState:     StateSpawned,
			wantFlowEvent: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := makeArenaWorld(t, tc.players, nil)
			w.State = StateLobby

			sys := &GameFlowSystem{}
			sys.Tick(w, 0.05)

			if w.State != tc.wantState {
				t.Errorf("state = %d, want %d", w.State, tc.wantState)
			}
			hasFlowEvent := len(w.GameFlowEvents) > 0
			if hasFlowEvent != tc.wantFlowEvent {
				t.Errorf("hasFlowEvent = %v, want %v", hasFlowEvent, tc.wantFlowEvent)
			}
			if tc.wantFlowEvent && len(w.GameFlowEvents) > 0 {
				if w.GameFlowEvents[0].FlowType != message.FlowSpawnPlayers {
					t.Errorf("flow type = %d, want FlowSpawnPlayers (%d)",
						w.GameFlowEvents[0].FlowType, message.FlowSpawnPlayers)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// tickSpawned
// ---------------------------------------------------------------------------

func TestTickSpawned(t *testing.T) {
	tests := []struct {
		name      string
		players   map[uint16]*entity.Player
		wantState GameFlowState
		wantEvent bool
	}{
		{
			name:      "no players stays spawned",
			players:   map[uint16]*entity.Player{},
			wantState: StateSpawned,
			wantEvent: false,
		},
		{
			name: "with players transitions to StateFight",
			players: func() map[uint16]*entity.Player {
				p := entity.NewPlayer(1, entity.ClassGunner)
				return map[uint16]*entity.Player{1: p}
			}(),
			wantState: StateFight,
			wantEvent: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := makeArenaWorld(t, tc.players, nil)
			w.State = StateSpawned

			sys := &GameFlowSystem{}
			sys.Tick(w, 0.05)

			if w.State != tc.wantState {
				t.Errorf("state = %d, want %d", w.State, tc.wantState)
			}
			if tc.wantEvent {
				if len(w.GameFlowEvents) == 0 {
					t.Fatal("expected FlowFightStart event")
				}
				if w.GameFlowEvents[0].FlowType != message.FlowFightStart {
					t.Errorf("flow type = %d, want FlowFightStart (%d)",
						w.GameFlowEvents[0].FlowType, message.FlowFightStart)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// checkFightEnd
// ---------------------------------------------------------------------------

func TestCheckFightEnd(t *testing.T) {
	tests := []struct {
		name           string
		setupEnemies   func() []*entity.Enemy
		setupPlayers   func() map[uint16]*entity.Player
		bossGateActive bool // pre-set gate state to avoid checkBossGate side effects
		wantState      GameFlowState
		wantDefeated   bool
		wantFlowType   uint8
		wantTransition bool
	}{
		{
			name: "boss dead triggers victory",
			setupEnemies: func() []*entity.Enemy {
				e := entity.NewEnemy(0, 2000, "guard_captain")
				e.IsBoss = true
				e.State = entity.EnemyDead
				e.Alive = false
				return []*entity.Enemy{e}
			},
			setupPlayers: func() map[uint16]*entity.Player {
				p := entity.NewPlayer(1, entity.ClassGunner)
				p.Position = entity.Vec3{X: 0, Y: 0.1, Z: 5} // in boss room
				return map[uint16]*entity.Player{1: p}
			},
			bossGateActive: true,
			wantState:      StateFightOver,
			wantDefeated:   true,
			wantFlowType:   message.FlowBossDead,
			wantTransition: true,
		},
		{
			name: "all players dead triggers wipe",
			setupEnemies: func() []*entity.Enemy {
				e := entity.NewEnemy(0, 2000, "guard_captain")
				e.IsBoss = true
				e.Alive = true
				e.State = entity.EnemyChase
				return []*entity.Enemy{e}
			},
			setupPlayers: func() map[uint16]*entity.Player {
				p := entity.NewPlayer(1, entity.ClassGunner)
				p.Alive = false
				p.Health = 0
				p.Position = entity.Vec3{X: 0, Y: 0.1, Z: 5} // in boss room
				return map[uint16]*entity.Player{1: p}
			},
			bossGateActive: true, // gate already active, boss in combat
			wantState:      StateFightOver,
			wantDefeated:   false,
			wantFlowType:   message.FlowAllDead,
			wantTransition: true,
		},
		{
			name: "some alive no transition",
			setupEnemies: func() []*entity.Enemy {
				e := entity.NewEnemy(0, 2000, "guard_captain")
				e.IsBoss = true
				e.Alive = true
				e.State = entity.EnemyChase
				return []*entity.Enemy{e}
			},
			setupPlayers: func() map[uint16]*entity.Player {
				p1 := entity.NewPlayer(1, entity.ClassGunner)
				p1.Position = entity.Vec3{X: 0, Y: 0.1, Z: 5} // in boss room
				p2 := entity.NewPlayer(2, entity.ClassVanguard)
				p2.Alive = false
				p2.Health = 0
				p2.Position = entity.Vec3{X: 0, Y: 0.1, Z: 5} // in boss room
				return map[uint16]*entity.Player{1: p1, 2: p2}
			},
			bossGateActive: true,
			wantState:      StateFight,
			wantDefeated:   false,
			wantTransition: false,
		},
		{
			name: "human dead but bot alive no wipe",
			setupEnemies: func() []*entity.Enemy {
				e := entity.NewEnemy(0, 2000, "guard_captain")
				e.IsBoss = true
				e.Alive = true
				e.State = entity.EnemyChase
				return []*entity.Enemy{e}
			},
			setupPlayers: func() map[uint16]*entity.Player {
				human := entity.NewPlayer(1, entity.ClassGunner)
				human.Alive = false
				human.Health = 0
				human.Position = entity.Vec3{X: 0, Y: 0.1, Z: 5}
				bot := entity.NewPlayer(entity.BotIDBase, entity.ClassVanguard)
				bot.Alive = true
				bot.Position = entity.Vec3{X: 0, Y: 0.1, Z: 5}
				return map[uint16]*entity.Player{1: human, entity.BotIDBase: bot}
			},
			bossGateActive: true,
			wantState:      StateFight,
			wantDefeated:   false,
			wantTransition: false,
		},
		{
			name: "human dead and bot dead triggers wipe",
			setupEnemies: func() []*entity.Enemy {
				e := entity.NewEnemy(0, 2000, "guard_captain")
				e.IsBoss = true
				e.Alive = true
				e.State = entity.EnemyChase
				return []*entity.Enemy{e}
			},
			setupPlayers: func() map[uint16]*entity.Player {
				human := entity.NewPlayer(1, entity.ClassGunner)
				human.Alive = false
				human.Health = 0
				human.Position = entity.Vec3{X: 0, Y: 0.1, Z: 5}
				bot := entity.NewPlayer(entity.BotIDBase, entity.ClassVanguard)
				bot.Alive = false
				bot.Health = 0
				bot.Position = entity.Vec3{X: 0, Y: 0.1, Z: 5}
				return map[uint16]*entity.Player{1: human, entity.BotIDBase: bot}
			},
			bossGateActive: true,
			wantState:      StateFightOver,
			wantDefeated:   false,
			wantFlowType:   message.FlowAllDead,
			wantTransition: true,
		},
		{
			name: "boss alive players alive no transition",
			setupEnemies: func() []*entity.Enemy {
				e := entity.NewEnemy(0, 2000, "guard_captain")
				e.IsBoss = true
				e.Alive = true
				e.State = entity.EnemyChase
				return []*entity.Enemy{e}
			},
			setupPlayers: func() map[uint16]*entity.Player {
				p := entity.NewPlayer(1, entity.ClassGunner)
				p.Position = entity.Vec3{X: 0, Y: 0.1, Z: 5} // in boss room
				return map[uint16]*entity.Player{1: p}
			},
			bossGateActive: true,
			wantState:      StateFight,
			wantDefeated:   false,
			wantTransition: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			enemies := tc.setupEnemies()
			players := tc.setupPlayers()
			w := makeArenaWorld(t, players, enemies)
			w.State = StateFight
			w.BossGateActive = tc.bossGateActive

			sys := &GameFlowSystem{}
			sys.Tick(w, 0.05)

			if w.State != tc.wantState {
				t.Errorf("state = %d, want %d", w.State, tc.wantState)
			}
			if w.BossDefeated != tc.wantDefeated {
				t.Errorf("BossDefeated = %v, want %v", w.BossDefeated, tc.wantDefeated)
			}
			if tc.wantTransition {
				found := false
				for _, evt := range w.GameFlowEvents {
					if evt.FlowType == tc.wantFlowType {
						found = true
						break
					}
				}
				if !found {
					types := make([]uint8, len(w.GameFlowEvents))
					for i, evt := range w.GameFlowEvents {
						types[i] = evt.FlowType
					}
					t.Errorf("expected flow type %d in events, got %v", tc.wantFlowType, types)
				}
			} else if len(w.GameFlowEvents) > 0 {
				types := make([]uint8, len(w.GameFlowEvents))
				for i, evt := range w.GameFlowEvents {
					types[i] = evt.FlowType
				}
				t.Errorf("expected no game flow events, got %v", types)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// tickFightOver — wipe recovery
// ---------------------------------------------------------------------------

func TestTickFightOver_WipeAllRespawnReturnsToLobby(t *testing.T) {
	p1 := entity.NewPlayer(1, entity.ClassGunner)
	p1.Alive = true
	p1.Health = p1.MaxHealth

	p2 := entity.NewPlayer(2, entity.ClassVanguard)
	p2.Alive = true
	p2.Health = p2.MaxHealth

	w := makeArenaWorld(t, map[uint16]*entity.Player{1: p1, 2: p2}, nil)
	w.State = StateFightOver
	w.BossDefeated = false

	sys := &GameFlowSystem{}
	sys.Tick(w, 0.05)

	// All alive + not boss defeated should transition back to StateSpawned
	if w.State != StateSpawned {
		t.Errorf("state = %d, want StateSpawned (%d)", w.State, StateSpawned)
	}
	found := false
	for _, evt := range w.GameFlowEvents {
		if evt.FlowType == message.FlowReturnLobby {
			found = true
		}
	}
	if !found {
		t.Error("expected FlowReturnLobby event")
	}
}

func TestTickFightOver_BossDefeatedNoReturn(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassGunner)
	p.Alive = true

	w := makeArenaWorld(t, map[uint16]*entity.Player{1: p}, nil)
	w.State = StateFightOver
	w.BossDefeated = true

	sys := &GameFlowSystem{}
	sys.Tick(w, 0.05)

	// Boss defeated: should stay in StateFightOver
	if w.State != StateFightOver {
		t.Errorf("state = %d, want StateFightOver", w.State)
	}
}

func TestTickFightOver_WipeSomeDeadNoReturn(t *testing.T) {
	p1 := entity.NewPlayer(1, entity.ClassGunner)
	p1.Alive = true

	p2 := entity.NewPlayer(2, entity.ClassVanguard)
	p2.Alive = false
	p2.Health = 0

	w := makeArenaWorld(t, map[uint16]*entity.Player{1: p1, 2: p2}, nil)
	w.State = StateFightOver
	w.BossDefeated = false

	sys := &GameFlowSystem{}
	sys.Tick(w, 0.05)

	// Not all alive yet, should stay in StateFightOver
	if w.State != StateFightOver {
		t.Errorf("state = %d, want StateFightOver", w.State)
	}
}

func TestTickFightOver_CooldownsTick(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassGunner)
	p.Alive = true
	p.Cooldowns[ability.IDFireShot] = 1.0

	w := makeArenaWorld(t, map[uint16]*entity.Player{1: p}, nil)
	w.State = StateFightOver
	w.BossDefeated = true

	// Cooldowns are now ticked by CombatSystem (via AbilityEngine.TickPlayer),
	// not by GameFlowSystem. Verify they still tick during fight-over state.
	sys := &CombatSystem{}
	sys.Tick(w, 0.5)

	cd := p.Cooldowns[ability.IDFireShot]
	if cd > 0.6 || cd < 0.4 {
		t.Errorf("fire_shot cooldown = %f, want ~0.5", cd)
	}
}

// ---------------------------------------------------------------------------
// InitInstance
// ---------------------------------------------------------------------------

func TestInitInstance(t *testing.T) {
	lvl := testArenaLevel(t)
	enemies := make([]*entity.Enemy, len(lvl.EnemySpawns))
	for i, sp := range lvl.EnemySpawns {
		e := entity.NewEnemy(uint16(i), 100, sp.DefName)
		e.Position = entity.Vec3{X: 999, Y: 999, Z: 999} // moved away from spawn
		e.State = entity.EnemyChase
		e.Health = 50 // damaged
		enemies[i] = e
	}

	w := &World{
		Enemies: enemies,
		Level:   lvl,
	}

	// Add some projectiles that should be cleared
	w.Projectiles = []*entity.Projectile{
		entity.NewProjectile(1, 0, 0, entity.Vec3{}, entity.Vec3{X: 1}, 10, 10, 5),
	}

	InitInstance(w)

	if len(w.Projectiles) != 0 {
		t.Errorf("projectiles = %d, want 0", len(w.Projectiles))
	}

	for i, e := range enemies {
		if e.Health != e.MaxHealth {
			t.Errorf("enemy %d health = %f, want %f", i, e.Health, e.MaxHealth)
		}
		if e.State != entity.EnemyPatrol {
			t.Errorf("enemy %d state = %d, want EnemyPatrol", i, e.State)
		}
		if i < len(lvl.EnemySpawns) {
			sp := lvl.EnemySpawns[i]
			if e.Position != sp.Position {
				t.Errorf("enemy %d position = %v, want %v", i, e.Position, sp.Position)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// ResetAliveEnemies
// ---------------------------------------------------------------------------

func TestResetAliveEnemies(t *testing.T) {
	lvl := testArenaLevel(t)

	eAlive := entity.NewEnemy(0, 200, "hallway_melee")
	eAlive.Position = entity.Vec3{X: 50, Y: 0, Z: 50}
	eAlive.State = entity.EnemyChase
	eAlive.Health = 100

	eDead := entity.NewEnemy(1, 200, "hallway_melee")
	eDead.Alive = false
	eDead.Health = 0
	eDead.State = entity.EnemyDead
	eDead.Position = entity.Vec3{X: 77, Y: 0, Z: 77}

	w := &World{
		Enemies:     []*entity.Enemy{eAlive, eDead},
		Level:       lvl,
		Projectiles: []*entity.Projectile{entity.NewProjectile(1, 0, 0, entity.Vec3{}, entity.Vec3{X: 1}, 10, 10, 5)},
	}

	ResetAliveEnemies(w)

	if len(w.Projectiles) != 0 {
		t.Errorf("projectiles = %d, want 0", len(w.Projectiles))
	}

	// Alive enemy should be reset to spawn and patrol
	if eAlive.State != entity.EnemyPatrol {
		t.Errorf("alive enemy state = %d, want EnemyPatrol", eAlive.State)
	}
	if eAlive.Health != eAlive.MaxHealth {
		t.Errorf("alive enemy health = %f, want %f", eAlive.Health, eAlive.MaxHealth)
	}
	if eAlive.Position == (entity.Vec3{X: 50, Y: 0, Z: 50}) {
		t.Error("alive enemy position should have been reset to spawn point")
	}

	// Dead enemy should remain dead
	if eDead.Alive {
		t.Error("dead enemy should remain dead")
	}
	if eDead.Health != 0 {
		t.Errorf("dead enemy health = %f, want 0", eDead.Health)
	}
}

// ---------------------------------------------------------------------------
// SpawnPlayers
// ---------------------------------------------------------------------------

func TestSpawnPlayers(t *testing.T) {
	lvl := testArenaLevel(t)
	p1 := entity.NewPlayer(1, entity.ClassGunner)
	p1.Position = entity.Vec3{X: 0, Y: 0, Z: 0}
	p1.Health = 50
	p1.Alive = false

	p2 := entity.NewPlayer(2, entity.ClassVanguard)
	p2.Position = entity.Vec3{X: 0, Y: 0, Z: 0}
	p2.Health = 50
	p2.IsRolling = true
	p2.RollCooldown = 5.0

	w := &World{
		TickNum: 200,
		Players: map[uint16]*entity.Player{1: p1, 2: p2},
		Level:   lvl,
	}

	SpawnPlayers(w)

	for _, p := range w.Players {
		if !p.Alive {
			t.Errorf("player %d should be alive after spawn", p.ID)
		}
		if p.Health != p.MaxHealth {
			t.Errorf("player %d health = %f, want %f", p.ID, p.Health, p.MaxHealth)
		}
		if p.State != entity.PlayerStateMove {
			t.Errorf("player %d state = %d, want PlayerStateMove", p.ID, p.State)
		}
		if p.IsRolling {
			t.Errorf("player %d should not be rolling after spawn", p.ID)
		}
		if p.RollCooldown != 0 {
			t.Errorf("player %d roll cooldown = %f, want 0", p.ID, p.RollCooldown)
		}
		if p.SpawnTick != w.TickNum {
			t.Errorf("player %d SpawnTick = %d, want %d", p.ID, p.SpawnTick, w.TickNum)
		}
		if p.RotationY != lvl.SpawnYaw {
			t.Errorf("player %d RotationY = %f, want %f", p.ID, p.RotationY, lvl.SpawnYaw)
		}
		if p.Velocity != (entity.Vec3{}) {
			t.Errorf("player %d velocity should be zero", p.ID)
		}
	}
}

// ---------------------------------------------------------------------------
// SpawnPlayer
// ---------------------------------------------------------------------------

func TestSpawnPlayer(t *testing.T) {
	lvl := testArenaLevel(t)
	p := entity.NewPlayer(1, entity.ClassGunner)
	p.Health = 10
	p.Alive = false
	p.Position = entity.Vec3{X: 99, Y: 99, Z: 99}

	w := &World{
		TickNum: 300,
		Players: map[uint16]*entity.Player{1: p},
		Level:   lvl,
	}

	SpawnPlayer(w, 1)

	if !p.Alive {
		t.Error("player should be alive after SpawnPlayer")
	}
	if p.Health != p.MaxHealth {
		t.Errorf("health = %f, want %f", p.Health, p.MaxHealth)
	}
	if p.SpawnTick != 300 {
		t.Errorf("SpawnTick = %d, want 300", p.SpawnTick)
	}
	if p.State != entity.PlayerStateMove {
		t.Errorf("state = %d, want PlayerStateMove", p.State)
	}
}

func TestSpawnPlayer_UnknownPeer(t *testing.T) {
	lvl := testArenaLevel(t)
	w := &World{
		Players: map[uint16]*entity.Player{},
		Level:   lvl,
	}
	// Should not panic
	SpawnPlayer(w, 42)
}

// ---------------------------------------------------------------------------
// checkBossGate
// ---------------------------------------------------------------------------

func TestCheckBossGate_AggroClosesGate(t *testing.T) {
	boss := entity.NewEnemy(0, 2000, "guard_captain")
	boss.IsBoss = true
	boss.State = entity.EnemyChase // not patrol = in combat
	boss.Position = entity.Vec3{X: 0, Y: 0.1, Z: 0}

	p := entity.NewPlayer(1, entity.ClassGunner)
	p.Position = entity.Vec3{X: 0, Y: 0.1, Z: 5} // in boss room (Z < BossRoomEntryZ=12)

	w := makeArenaWorld(t, map[uint16]*entity.Player{1: p}, []*entity.Enemy{boss})
	w.State = StateFight
	w.BossGateActive = false

	sys := &GameFlowSystem{}
	sys.Tick(w, 0.05)

	if !w.BossGateActive {
		t.Error("boss gate should be active after boss enters combat")
	}

	found := false
	for _, evt := range w.GameFlowEvents {
		if evt.FlowType == message.FlowBossActivated {
			found = true
		}
	}
	if !found {
		t.Error("expected FlowBossActivated event")
	}
}

func TestCheckBossGate_NoPlayersInBossRoomResetsBoss(t *testing.T) {
	boss := entity.NewEnemy(0, 2000, "guard_captain")
	boss.IsBoss = true
	boss.State = entity.EnemyChase
	boss.Health = 1000 // damaged
	boss.Position = entity.Vec3{X: 0, Y: 0.1, Z: 0}

	p := entity.NewPlayer(1, entity.ClassGunner)
	// Player is outside boss room (Z > BossRoomEntryZ=12)
	p.Position = entity.Vec3{X: 0, Y: 0.1, Z: 20}

	enemies := []*entity.Enemy{boss}
	w := makeArenaWorld(t, map[uint16]*entity.Player{1: p}, enemies)
	w.State = StateFight
	w.BossGateActive = true

	// Add projectile that should be cleared on reset
	w.Projectiles = []*entity.Projectile{
		entity.NewProjectile(1, 0, 0, entity.Vec3{}, entity.Vec3{X: 1}, 10, 10, 5),
	}

	sys := &GameFlowSystem{}
	sys.Tick(w, 0.05)

	if w.BossGateActive {
		t.Error("boss gate should be deactivated when no players in boss room")
	}
	if boss.Health != boss.MaxHealth {
		t.Errorf("boss health = %f, want %f (should be reset)", boss.Health, boss.MaxHealth)
	}
	if boss.State != entity.EnemyPatrol {
		t.Errorf("boss state = %d, want EnemyPatrol (should be reset)", boss.State)
	}
	if len(w.Projectiles) != 0 {
		t.Errorf("projectiles = %d, want 0", len(w.Projectiles))
	}

	found := false
	for _, evt := range w.GameFlowEvents {
		if evt.FlowType == message.FlowBossReset {
			found = true
		}
	}
	if !found {
		t.Error("expected FlowBossReset event")
	}
}

func TestCheckBossGate_PushesPlayersNearGate(t *testing.T) {
	lvl := testArenaLevel(t)
	boss := entity.NewEnemy(0, 2000, "guard_captain")
	boss.IsBoss = true
	boss.State = entity.EnemyChase
	boss.Position = entity.Vec3{X: 0, Y: 0.1, Z: 0}

	// Player right at the gate threshold
	p := entity.NewPlayer(1, entity.ClassGunner)
	p.Position = entity.Vec3{X: 0, Y: 0.1, Z: lvl.BossRoomEntryZ - 1.0}

	w := makeArenaWorld(t, map[uint16]*entity.Player{1: p}, []*entity.Enemy{boss})
	w.State = StateFight
	w.BossGateActive = false

	sys := &GameFlowSystem{}
	sys.Tick(w, 0.05)

	// Player near gate should be pushed into boss room
	if p.Position.Z >= lvl.BossRoomEntryZ-2.0 {
		t.Errorf("player Z = %f, should have been pushed below gate threshold", p.Position.Z)
	}
}

func TestCheckBossGate_RemovesThreatForOutsidePlayers(t *testing.T) {
	boss := entity.NewEnemy(0, 2000, "guard_captain")
	boss.IsBoss = true
	boss.State = entity.EnemyChase
	boss.Position = entity.Vec3{X: 0, Y: 0.1, Z: 0}
	boss.AddThreat(1, 100)
	boss.AddThreat(2, 50)

	// Player 1 in boss room
	p1 := entity.NewPlayer(1, entity.ClassGunner)
	p1.Position = entity.Vec3{X: 0, Y: 0.1, Z: 5}

	// Player 2 outside boss room (Z > BossRoomEntryZ=12)
	p2 := entity.NewPlayer(2, entity.ClassVanguard)
	p2.Position = entity.Vec3{X: 0, Y: 0.1, Z: 20}

	w := makeArenaWorld(t, map[uint16]*entity.Player{1: p1, 2: p2}, []*entity.Enemy{boss})
	w.State = StateFight
	w.BossGateActive = false

	sys := &GameFlowSystem{}
	sys.Tick(w, 0.05)

	// Player 2's threat should be removed (they're outside)
	if boss.HasThreat(2) {
		t.Error("player 2 outside boss room should have threat removed")
	}
	// Player 1's threat should remain
	if !boss.HasThreat(1) {
		t.Error("player 1 inside boss room should keep threat")
	}
}

func TestCheckBossGate_DeadBossSkipped(t *testing.T) {
	boss := entity.NewEnemy(0, 2000, "guard_captain")
	boss.IsBoss = true
	boss.Alive = false
	boss.State = entity.EnemyDead

	w := makeArenaWorld(t, map[uint16]*entity.Player{}, []*entity.Enemy{boss})
	w.State = StateFight

	sys := &GameFlowSystem{}
	sys.Tick(w, 0.05)

	// Dead boss should not trigger gate logic (boss dead -> checkFightEnd handles it)
	if w.BossGateActive {
		t.Error("dead boss should not activate gate")
	}
}

// ---------------------------------------------------------------------------
// findBoss, findBossIndex
// ---------------------------------------------------------------------------

func TestFindBoss(t *testing.T) {
	tests := []struct {
		name    string
		enemies []*entity.Enemy
		wantNil bool
		wantID  uint16
	}{
		{
			name:    "no enemies",
			enemies: nil,
			wantNil: true,
		},
		{
			name: "no boss among enemies",
			enemies: []*entity.Enemy{
				func() *entity.Enemy {
					e := entity.NewEnemy(0, 200, "trash")
					e.IsBoss = false
					return e
				}(),
			},
			wantNil: true,
		},
		{
			name: "boss found",
			enemies: []*entity.Enemy{
				func() *entity.Enemy {
					e := entity.NewEnemy(0, 200, "trash")
					return e
				}(),
				func() *entity.Enemy {
					e := entity.NewEnemy(1, 2000, "guard_captain")
					e.IsBoss = true
					return e
				}(),
			},
			wantNil: false,
			wantID:  1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := &World{Enemies: tc.enemies}
			boss := findBoss(w)
			if tc.wantNil {
				if boss != nil {
					t.Errorf("expected nil boss, got ID=%d", boss.ID)
				}
			} else {
				if boss == nil {
					t.Fatal("expected boss, got nil")
				}
				if boss.ID != tc.wantID {
					t.Errorf("boss ID = %d, want %d", boss.ID, tc.wantID)
				}
			}
		})
	}
}

func TestFindBossIndex(t *testing.T) {
	tests := []struct {
		name    string
		enemies []*entity.Enemy
		want    int
	}{
		{
			name:    "no enemies",
			enemies: nil,
			want:    -1,
		},
		{
			name: "boss at index 2",
			enemies: []*entity.Enemy{
				entity.NewEnemy(0, 200, "trash"),
				entity.NewEnemy(1, 200, "trash"),
				func() *entity.Enemy {
					e := entity.NewEnemy(2, 2000, "boss")
					e.IsBoss = true
					return e
				}(),
			},
			want: 2,
		},
		{
			name: "no boss",
			enemies: []*entity.Enemy{
				entity.NewEnemy(0, 200, "trash"),
			},
			want: -1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := &World{Enemies: tc.enemies}
			got := findBossIndex(w)
			if got != tc.want {
				t.Errorf("findBossIndex = %d, want %d", got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Full wipe → respawn → fight: boss must not chase after reset
// ---------------------------------------------------------------------------

func TestWipeAndRespawn_BossStaysPatrol(t *testing.T) {
	boss := entity.NewEnemy(0, 2000, "guard_captain")
	boss.IsBoss = true
	boss.State = entity.EnemyChase
	boss.Position = entity.Vec3{X: 0, Y: 0.1, Z: 0} // in boss room
	boss.TargetPlayerID = 1
	boss.AddThreat(1, 100)

	p := entity.NewPlayer(1, entity.ClassGunner)
	p.Alive = false
	p.Health = 0
	p.State = entity.PlayerStateDead
	p.Position = entity.Vec3{X: 0, Y: 0.1, Z: 5} // died in boss room

	enemies := []*entity.Enemy{boss}
	players := map[uint16]*entity.Player{1: p}
	w := makeArenaWorld(t, players, enemies)
	w.State = StateFight
	w.BossGateActive = true

	sys := &GameFlowSystem{}

	// Tick 1: all dead → checkBossGate resets boss, checkFightEnd → StateFightOver
	sys.Tick(w, 0.05)
	if w.State != StateFightOver {
		t.Fatalf("after wipe: state = %d, want StateFightOver", w.State)
	}

	// Simulate player respawn (client sends respawn)
	p.Alive = true
	p.Health = p.MaxHealth

	// Tick 2: all alive + !bossDefeated → returnToLobby → StateSpawned
	w.GameFlowEvents = w.GameFlowEvents[:0]
	sys.Tick(w, 0.05)
	if w.State != StateSpawned {
		t.Fatalf("after respawn: state = %d, want StateSpawned", w.State)
	}

	// Tick 3: players present → StateFight
	w.GameFlowEvents = w.GameFlowEvents[:0]
	sys.Tick(w, 0.05)
	if w.State != StateFight {
		t.Fatalf("after spawned: state = %d, want StateFight", w.State)
	}

	// Verify boss is patrol with no stale target
	if boss.State != entity.EnemyPatrol {
		t.Errorf("boss state = %d, want EnemyPatrol", boss.State)
	}
	if boss.TargetPlayerID != 0 {
		t.Errorf("boss TargetPlayerID = %d, want 0 (should be cleared on reset)", boss.TargetPlayerID)
	}
	if boss.HasThreat(1) {
		t.Error("boss should have empty threat table after reset")
	}
}

// ---------------------------------------------------------------------------
// returnToLobby (tested indirectly via tickFightOver)
// ---------------------------------------------------------------------------

func TestReturnToLobby_ResetsPlayerState(t *testing.T) {
	lvl := testArenaLevel(t)

	p := entity.NewPlayer(1, entity.ClassGunner)
	p.Ready = true
	p.Health = 50
	p.Alive = false
	p.Position = entity.Vec3{X: 5, Y: 2, Z: 5}
	p.Velocity = entity.Vec3{X: 1, Y: 1, Z: 1}

	eAlive := entity.NewEnemy(0, 200, "hallway_melee")
	eAlive.State = entity.EnemyChase
	eAlive.Position = entity.Vec3{X: 99, Y: 0, Z: 99}

	eDead := entity.NewEnemy(1, 200, "hallway_melee")
	eDead.Alive = false
	eDead.State = entity.EnemyDead
	eDead.Health = 0

	w := makeArenaWorld(t, map[uint16]*entity.Player{1: p}, []*entity.Enemy{eAlive, eDead})
	w.State = StateFightOver
	w.BossDefeated = false
	w.Level = lvl

	// Make player alive to trigger returnToLobby
	p.Alive = true
	p.Health = p.MaxHealth

	sys := &GameFlowSystem{}
	sys.Tick(w, 0.05)

	if w.State != StateSpawned {
		t.Errorf("state = %d, want StateSpawned", w.State)
	}
	if !p.Alive {
		t.Error("player should be alive")
	}
	if p.Health != p.MaxHealth {
		t.Errorf("health = %f, want %f", p.Health, p.MaxHealth)
	}
	if p.Ready {
		t.Error("player ready should be reset to false")
	}
	if p.Position.Z != 48.0 {
		t.Errorf("player Z = %f, want 48.0 (warmup position)", p.Position.Z)
	}

	// Alive enemy should be reset
	if eAlive.State != entity.EnemyPatrol {
		t.Errorf("alive enemy state = %d, want EnemyPatrol", eAlive.State)
	}

	// Dead enemy should remain dead
	if eDead.Alive {
		t.Error("dead enemy should remain dead")
	}
}

// ---------------------------------------------------------------------------
// pickSpawnPoint — conditional spawn selection
// ---------------------------------------------------------------------------

func TestPickSpawnPoint(t *testing.T) {
	spawns := []level.PlayerSpawn{
		{Position: entity.Vec3{X: 0, Y: 0.1, Z: 48}, Condition: ""},
		{Position: entity.Vec3{X: 1, Y: 0.1, Z: 48}, Condition: ""},
		{Position: entity.Vec3{X: 0, Y: 0.1, Z: 26}, Condition: "pack_1_cleared"},
		{Position: entity.Vec3{X: 0, Y: 0.1, Z: 0}, Condition: "boss_dead"},
	}

	t.Run("default_no_progress", func(t *testing.T) {
		pos := pickSpawnPoint(spawns, level.ZoneState{}, 0)
		if pos.Z != 48 {
			t.Errorf("Z = %f, want 48 (default spawn)", pos.Z)
		}
	})

	t.Run("default_round_robin", func(t *testing.T) {
		pos := pickSpawnPoint(spawns, level.ZoneState{}, 1)
		if pos.X != 1 {
			t.Errorf("X = %f, want 1 (second default spawn)", pos.X)
		}
	})

	t.Run("pack1_cleared", func(t *testing.T) {
		pos := pickSpawnPoint(spawns, level.ZoneState{DeadGroupIDs: map[int]bool{1: true}}, 0)
		if pos.Z != 26 {
			t.Errorf("Z = %f, want 26 (pack_1_cleared checkpoint)", pos.Z)
		}
	})

	t.Run("boss_dead_highest_priority", func(t *testing.T) {
		pos := pickSpawnPoint(spawns, level.ZoneState{BossDefeated: true, DeadGroupIDs: map[int]bool{1: true, 2: true}}, 0)
		if pos.Z != 0 {
			t.Errorf("Z = %f, want 0 (boss_dead checkpoint)", pos.Z)
		}
	})

	t.Run("empty_spawns", func(t *testing.T) {
		pos := pickSpawnPoint(nil, level.ZoneState{}, 0)
		if pos.Y != 0.1 {
			t.Errorf("Y = %f, want 0.1 (fallback)", pos.Y)
		}
	})
}
