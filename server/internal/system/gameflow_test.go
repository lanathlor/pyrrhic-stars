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
	w := &World{
		ZoneID:        testArenaZoneID,
		ZoneType:      1,
		TickNum:       100,
		Players:       players,
		Enemies:       enemies,
		Level:         lvl,
		Clients:       make(map[uint16]*Client),
		AbilityEngine: ability.NewEngine(nil),
	}
	w.InitGateStates()
	return w
}

// ---------------------------------------------------------------------------
// checkFightEnd
// ---------------------------------------------------------------------------

func TestCheckFightEnd(t *testing.T) {
	tests := []struct {
		name            string
		setupEnemies    func() []*entity.Enemy
		setupPlayers    func() map[uint16]*entity.Player
		bossGateActive  bool // pre-set gate state to avoid checkBossGate side effects
		wantDefeated    bool
		wantWipeHandled bool
		wantFlowType    uint8
		wantTransition  bool
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
			bossGateActive:  true, // gate already active, boss in combat
			wantDefeated:    false,
			wantWipeHandled: true,
			wantFlowType:    message.FlowAllDead,
			wantTransition:  true,
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
			bossGateActive:  true,
			wantDefeated:    false,
			wantWipeHandled: true,
			wantFlowType:    message.FlowAllDead,
			wantTransition:  true,
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
			wantDefeated:   false,
			wantTransition: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			enemies := tc.setupEnemies()
			players := tc.setupPlayers()
			w := makeArenaWorld(t, players, enemies)
			w.GateStates["boss_gate"] = tc.bossGateActive

			sys := &GameFlowSystem{}
			sys.Tick(w, 0.05)

			if w.BossDefeated != tc.wantDefeated {
				t.Errorf("BossDefeated = %v, want %v", w.BossDefeated, tc.wantDefeated)
			}
			if w.WipeHandled != tc.wantWipeHandled {
				t.Errorf("WipeHandled = %v, want %v", w.WipeHandled, tc.wantWipeHandled)
			}
			// After a wipe, gates should be reset (all open)
			if tc.wantWipeHandled && w.AnyGateClosed() {
				t.Error("gates should be reset (all open) after wipe")
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
// Cooldowns tick via CombatSystem (no state gating)
// ---------------------------------------------------------------------------

func TestCooldownsTick(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassGunner)
	p.Alive = true
	p.Cooldowns[ability.IDFireShot] = 1.0

	w := makeArenaWorld(t, map[uint16]*entity.Player{1: p}, nil)
	w.BossDefeated = true

	// Cooldowns are ticked by CombatSystem (via AbilityEngine.TickPlayer).
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
// checkBossState + processGateEvents (data-driven gate system)
// ---------------------------------------------------------------------------

func TestCheckBossState_AggroClosesGate(t *testing.T) {
	boss := entity.NewEnemy(0, 2000, "guard_captain")
	boss.IsBoss = true
	boss.State = entity.EnemyChase // not patrol = in combat
	boss.Position = entity.Vec3{X: 0, Y: 0.1, Z: 0}

	p := entity.NewPlayer(1, entity.ClassGunner)
	p.Position = entity.Vec3{X: 0, Y: 0.1, Z: 5} // in boss room (Z < gate at 12)

	w := makeArenaWorld(t, map[uint16]*entity.Player{1: p}, []*entity.Enemy{boss})

	sys := &GameFlowSystem{}
	sys.Tick(w, 0.05)

	if !w.IsGateClosed("boss_gate") {
		t.Error("boss gate should be closed after boss enters combat")
	}

	foundActivated := false
	foundGateClose := false
	for _, evt := range w.GameFlowEvents {
		if evt.FlowType == message.FlowBossActivated {
			foundActivated = true
		}
		if evt.FlowType == message.FlowGateClose && evt.Text == "boss_gate" {
			foundGateClose = true
		}
	}
	if !foundActivated {
		t.Error("expected FlowBossActivated event")
	}
	if !foundGateClose {
		t.Error("expected FlowGateClose event for boss_gate")
	}
}

func TestCheckBossState_NoPlayersInBossRoomResetsBoss(t *testing.T) {
	boss := entity.NewEnemy(0, 2000, "guard_captain")
	boss.IsBoss = true
	boss.State = entity.EnemyChase
	boss.Health = 1000 // damaged
	boss.Position = entity.Vec3{X: 0, Y: 0.1, Z: 0}

	p := entity.NewPlayer(1, entity.ClassGunner)
	// Player is outside boss room (Z > gate at 12)
	p.Position = entity.Vec3{X: 0, Y: 0.1, Z: 20}

	enemies := []*entity.Enemy{boss}
	w := makeArenaWorld(t, map[uint16]*entity.Player{1: p}, enemies)
	w.GateStates["boss_gate"] = true
	w.RebuildObstacles()

	// Add projectile that should be cleared on reset
	w.Projectiles = []*entity.Projectile{
		entity.NewProjectile(1, 0, 0, entity.Vec3{}, entity.Vec3{X: 1}, 10, 10, 5),
	}

	sys := &GameFlowSystem{}
	sys.Tick(w, 0.05)

	if w.IsGateClosed("boss_gate") {
		t.Error("boss gate should be open when no players in boss room")
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

	foundReset := false
	foundGateOpen := false
	for _, evt := range w.GameFlowEvents {
		if evt.FlowType == message.FlowBossReset {
			foundReset = true
		}
		if evt.FlowType == message.FlowGateOpen && evt.Text == "boss_gate" {
			foundGateOpen = true
		}
	}
	if !foundReset {
		t.Error("expected FlowBossReset event")
	}
	if !foundGateOpen {
		t.Error("expected FlowGateOpen event for boss_gate")
	}
}

func TestCheckBossState_PushesPlayersNearGate(t *testing.T) {
	boss := entity.NewEnemy(0, 2000, "guard_captain")
	boss.IsBoss = true
	boss.State = entity.EnemyChase
	boss.Position = entity.Vec3{X: 0, Y: 0.1, Z: 0}

	// Player right at the gate threshold (gate is at Z=12)
	p := entity.NewPlayer(1, entity.ClassGunner)
	p.Position = entity.Vec3{X: 0, Y: 0.1, Z: 11.0}

	w := makeArenaWorld(t, map[uint16]*entity.Player{1: p}, []*entity.Enemy{boss})

	sys := &GameFlowSystem{}
	sys.Tick(w, 0.05)

	// Player near gate should be pushed into boss room (Z = 12 + (-3) = 9)
	if p.Position.Z >= 10.0 {
		t.Errorf("player Z = %f, should have been pushed below gate threshold", p.Position.Z)
	}
}

func TestCheckBossState_RemovesThreatForOutsidePlayers(t *testing.T) {
	boss := entity.NewEnemy(0, 2000, "guard_captain")
	boss.IsBoss = true
	boss.State = entity.EnemyChase
	boss.Position = entity.Vec3{X: 0, Y: 0.1, Z: 0}
	boss.AddThreat(1, 100)
	boss.AddThreat(2, 50)

	// Player 1 in boss room
	p1 := entity.NewPlayer(1, entity.ClassGunner)
	p1.Position = entity.Vec3{X: 0, Y: 0.1, Z: 5}

	// Player 2 outside boss room (Z > gate at 12)
	p2 := entity.NewPlayer(2, entity.ClassVanguard)
	p2.Position = entity.Vec3{X: 0, Y: 0.1, Z: 20}

	w := makeArenaWorld(t, map[uint16]*entity.Player{1: p1, 2: p2}, []*entity.Enemy{boss})

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

func TestCheckBossState_DeadBossSkipped(t *testing.T) {
	boss := entity.NewEnemy(0, 2000, "guard_captain")
	boss.IsBoss = true
	boss.Alive = false
	boss.State = entity.EnemyDead

	w := makeArenaWorld(t, map[uint16]*entity.Player{}, []*entity.Enemy{boss})

	sys := &GameFlowSystem{}
	sys.Tick(w, 0.05)

	// Dead boss should not trigger gate logic (boss dead -> checkFightEnd handles it)
	if w.IsGateClosed("boss_gate") {
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
// pickSpawnPoint -- conditional spawn selection
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

// ---------------------------------------------------------------------------
// checkLobbyReady
// ---------------------------------------------------------------------------

func makeLobbyWorld(t testing.TB) *World {
	t.Helper()
	lvl := testArenaLevel(t)
	p1 := entity.NewPlayer(1, entity.ClassGunner)
	p2 := entity.NewPlayer(2, entity.ClassVanguard)
	w := &World{
		ZoneID:        testArenaZoneID,
		ZoneType:      1,
		TickNum:       100,
		Players:       map[uint16]*entity.Player{1: p1, 2: p2},
		Enemies:       nil,
		Level:         lvl,
		Clients:       make(map[uint16]*Client),
		AbilityEngine: ability.NewEngine(nil),
		LobbyActive:   true,
	}
	w.InitGateStates()
	return w
}

func TestCheckLobbyReady_AllReadyStartsCountdown(t *testing.T) {
	w := makeLobbyWorld(t)
	w.Players[1].Ready = true
	w.Players[2].Ready = true

	checkLobbyReady(w)

	if w.LobbyCountdown == 0 {
		t.Fatal("expected countdown to start when all ready")
	}
	if w.LobbyCountdown != LobbyCountdownTicks-1 {
		t.Errorf("countdown = %d, want %d (decremented on first tick)", w.LobbyCountdown, LobbyCountdownTicks-1)
	}
}

func TestCheckLobbyReady_NotAllReady(t *testing.T) {
	w := makeLobbyWorld(t)
	w.Players[1].Ready = true
	w.Players[2].Ready = false

	checkLobbyReady(w)

	if w.LobbyCountdown != 0 {
		t.Errorf("countdown = %d, want 0 (not all ready)", w.LobbyCountdown)
	}
}

func TestCheckLobbyReady_UnreadyCancelsCountdown(t *testing.T) {
	w := makeLobbyWorld(t)
	w.LobbyCountdown = 50 // mid-countdown
	w.Players[1].Ready = true
	w.Players[2].Ready = false

	checkLobbyReady(w)

	if w.LobbyCountdown != 0 {
		t.Errorf("countdown = %d, want 0 (cancelled)", w.LobbyCountdown)
	}
}

func TestCheckLobbyReady_CountdownExpireEmitsFightStart(t *testing.T) {
	w := makeLobbyWorld(t)
	w.Players[1].Ready = true
	w.Players[2].Ready = true
	w.LobbyCountdown = 1 // about to expire

	checkLobbyReady(w)

	if w.LobbyActive {
		t.Error("lobby should be inactive after countdown expires")
	}
	if w.LobbyCountdown != 0 {
		t.Errorf("countdown = %d, want 0", w.LobbyCountdown)
	}
	if w.Players[1].Ready || w.Players[2].Ready {
		t.Error("players should be unreadied after fight start")
	}

	found := false
	for _, evt := range w.GameFlowEvents {
		if evt.FlowType == message.FlowFightStart {
			found = true
		}
	}
	if !found {
		t.Error("expected FlowFightStart event")
	}
}

func TestCheckLobbyReady_InactiveNoOp(t *testing.T) {
	w := makeLobbyWorld(t)
	w.LobbyActive = false
	w.Players[1].Ready = true
	w.Players[2].Ready = true

	checkLobbyReady(w)

	if w.LobbyCountdown != 0 {
		t.Errorf("countdown = %d, want 0 (lobby inactive)", w.LobbyCountdown)
	}
}

func TestInitInstance_SetsLobbyActive(t *testing.T) {
	lvl := testArenaLevel(t)
	e := entity.NewEnemy(0, 2000, "guard_captain")
	e.IsBoss = true
	w := &World{
		ZoneID:        testArenaZoneID,
		ZoneType:      1,
		Players:       make(map[uint16]*entity.Player),
		Enemies:       []*entity.Enemy{e},
		Level:         lvl,
		Clients:       make(map[uint16]*Client),
		AbilityEngine: ability.NewEngine(nil),
	}

	InitInstance(w)

	if !w.LobbyActive {
		t.Error("expected LobbyActive=true after InitInstance")
	}
	if w.LobbyCountdown != 0 {
		t.Errorf("countdown = %d, want 0", w.LobbyCountdown)
	}
}
