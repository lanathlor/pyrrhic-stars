package system

import (
	"testing"

	"codex-online/server/internal/entity"
	"codex-online/server/internal/level"
)

func makeWorld(players map[uint16]*entity.Player, enemies []*entity.Enemy) *World {
	return &World{
		ZoneType: 1, // arena
		TickNum:  100,
		State:    StateFight,
		Players:  players,
		Enemies:  enemies,
		Level:    level.NewArenaLevel(),
	}
}

// --- Unit tests ---

func TestInCombatWhenOnThreatTable(t *testing.T) {
	p := entity.NewPlayer(1, "gunner")
	e := entity.NewEnemy(0, 2000.0, "guard_captain")
	e.Alive = true
	e.AddThreat(1, 10.0)

	w := makeWorld(map[uint16]*entity.Player{1: p}, []*entity.Enemy{e})
	sys := CombatSystem{}
	sys.Tick(w, 0.05)

	if !p.InCombat {
		t.Error("player should be in combat when on threat table")
	}
}

func TestNotInCombatWhenNotOnThreatTable(t *testing.T) {
	p := entity.NewPlayer(1, "gunner")
	e := entity.NewEnemy(0, 2000.0, "guard_captain")
	e.Alive = true
	// no threat added

	w := makeWorld(map[uint16]*entity.Player{1: p}, []*entity.Enemy{e})
	sys := CombatSystem{}
	sys.Tick(w, 0.05)

	if p.InCombat {
		t.Error("player should not be in combat when not on threat table")
	}
}

func TestRegenOnlyOutOfCombat(t *testing.T) {
	p := entity.NewPlayer(1, "gunner")
	p.Health = 100.0 // below max (150)

	e := entity.NewEnemy(0, 2000.0, "guard_captain")
	e.Alive = true
	e.AddThreat(1, 10.0)

	w := makeWorld(map[uint16]*entity.Player{1: p}, []*entity.Enemy{e})
	sys := CombatSystem{}

	// In combat — no regen
	sys.Tick(w, 0.05)
	if p.Health != 100.0 {
		t.Errorf("health = %f during combat, want 100.0 (no regen)", p.Health)
	}

	// Remove from threat table — out of combat, regen should apply
	e.ClearThreat()
	sys.Tick(w, 1.0) // 1 second = 5% of 150 = 7.5 HP
	expected := float32(107.5)
	if p.Health < expected-0.1 || p.Health > expected+0.1 {
		t.Errorf("health = %f after 1s regen, want ~%f", p.Health, expected)
	}
}

func TestMultiplePlayersAllInCombat(t *testing.T) {
	p1 := entity.NewPlayer(1, "gunner")
	p2 := entity.NewPlayer(2, "vanguard")
	p3 := entity.NewPlayer(3, "blade_dancer")

	e := entity.NewEnemy(0, 2000.0, "guard_captain")
	e.Alive = true
	e.AddThreat(1, 10.0)
	e.AddThreat(2, 30.0)
	e.AddThreat(3, 5.0)

	players := map[uint16]*entity.Player{1: p1, 2: p2, 3: p3}
	w := makeWorld(players, []*entity.Enemy{e})
	sys := CombatSystem{}
	sys.Tick(w, 0.05)

	for id, p := range players {
		if !p.InCombat {
			t.Errorf("player %d should be in combat", id)
		}
	}
}

func TestNotInCombatAfterEnemyDies(t *testing.T) {
	p := entity.NewPlayer(1, "gunner")
	e := entity.NewEnemy(0, 2000.0, "guard_captain")
	e.Alive = true
	e.AddThreat(1, 50.0)

	w := makeWorld(map[uint16]*entity.Player{1: p}, []*entity.Enemy{e})
	sys := CombatSystem{}

	// In combat while alive
	sys.Tick(w, 0.05)
	if !p.InCombat {
		t.Fatal("should be in combat while enemy alive")
	}

	// Enemy dies — threat table still has player, but enemy is dead
	e.Alive = false
	sys.Tick(w, 0.05)
	if p.InCombat {
		t.Error("should not be in combat after enemy dies")
	}
}

// --- Integration tests ---

func TestThreatGeneratedOnPlayerAttack(t *testing.T) {
	// Set up a world with a gunner aimed directly at the enemy
	p := entity.NewPlayer(1, "gunner")
	p.Position = entity.Vec3{X: 0, Y: 0.1, Z: 5}
	p.RotationY = 0 // facing -Z (toward enemy at origin)
	p.AimPitch = 0

	e := entity.NewEnemy(0, 2000.0, "guard_captain")
	e.Alive = true
	e.Position = entity.Vec3{X: 0, Y: 0.1, Z: 0}

	w := makeWorld(map[uint16]*entity.Player{1: p}, []*entity.Enemy{e})
	w.State = StateFight

	// Simulate a gunner shoot input
	inputSys := InputSystem{}
	// Build a shoot ability input: action=0 (shoot), aimPitch as float32
	payload := []byte{entity.ActionShoot}
	// Append aim pitch (4 bytes, little-endian float32 = 0.0)
	payload = append(payload, 0, 0, 0, 0)

	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: 0x0031, Payload: payload}}
	inputSys.Tick(w, 0.05)

	if !e.HasThreat(1) {
		t.Error("enemy should have threat from player 1 after being shot")
	}
	if e.ThreatTable[1] <= 0 {
		t.Errorf("threat should be > 0, got %f", e.ThreatTable[1])
	}
}

func TestCombatEndsOnEnemyDeath(t *testing.T) {
	p1 := entity.NewPlayer(1, "gunner")
	p1.Health = 100.0 // below max
	p2 := entity.NewPlayer(2, "vanguard")
	p2.Health = 150.0 // below max (200)

	e := entity.NewEnemy(0, 2000.0, "guard_captain")
	e.Alive = true
	e.AddThreat(1, 10.0)
	e.AddThreat(2, 20.0)

	players := map[uint16]*entity.Player{1: p1, 2: p2}
	w := makeWorld(players, []*entity.Enemy{e})
	sys := CombatSystem{}

	// Both in combat
	sys.Tick(w, 0.05)
	if !p1.InCombat || !p2.InCombat {
		t.Fatal("both players should be in combat")
	}

	// Enemy dies and resets (clears threat table)
	e.Alive = false
	e.ClearThreat()

	// Tick combat — both should be out of combat, regen applies
	hp1Before := p1.Health
	hp2Before := p2.Health
	sys.Tick(w, 1.0) // 1 second

	if p1.InCombat || p2.InCombat {
		t.Error("players should be out of combat after enemy death")
	}
	if p1.Health <= hp1Before {
		t.Errorf("p1 health should have increased from regen, got %f (was %f)", p1.Health, hp1Before)
	}
	if p2.Health <= hp2Before {
		t.Errorf("p2 health should have increased from regen, got %f (was %f)", p2.Health, hp2Before)
	}
}
