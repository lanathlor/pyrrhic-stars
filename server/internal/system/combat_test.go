package system

import (
	"math"
	"testing"

	"codex-online/server/internal/codec"
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

// --- Overclock / Rechamber timer tests ---

func TestOverclockTimerExpires(t *testing.T) {
	p := entity.NewPlayer(1, "gunner")
	p.OverclockActive = true
	p.OverclockTimer = 7.0
	p.OverclockCooldown = 15.0

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	sys := CombatSystem{}

	// Tick 3 seconds — still active
	sys.Tick(w, 3.0)
	if !p.OverclockActive {
		t.Error("overclock should still be active after 3s")
	}
	if p.OverclockTimer <= 0 {
		t.Error("overclock timer should be > 0 after 3s")
	}

	// Tick another 5 seconds — should expire (8s total > 7s)
	sys.Tick(w, 5.0)
	if p.OverclockActive {
		t.Error("overclock should be inactive after 8s total")
	}
}

func TestOverclockCooldownTicksDown(t *testing.T) {
	p := entity.NewPlayer(1, "gunner")
	p.OverclockCooldown = 15.0

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	sys := CombatSystem{}

	sys.Tick(w, 10.0)
	if p.OverclockCooldown != 5.0 {
		t.Errorf("overclock cooldown = %f, want 5.0", p.OverclockCooldown)
	}

	sys.Tick(w, 10.0)
	if p.OverclockCooldown != 0.0 {
		t.Errorf("overclock cooldown = %f, want 0.0 (clamped)", p.OverclockCooldown)
	}
}

func TestRechamberPhaseTransitions(t *testing.T) {
	p := entity.NewPlayer(1, "gunner")
	p.RechamberPhase = 1
	p.RechamberTimer = 0.6

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	sys := CombatSystem{}

	// Phase 1 windup -> phase 2 timing window
	sys.Tick(w, 0.7)
	if p.RechamberPhase != 2 {
		t.Errorf("expected phase 2, got %d", p.RechamberPhase)
	}
	if p.RechamberTimer < 0.3 || p.RechamberTimer > 0.36 {
		t.Errorf("timing window timer = %f, want ~0.35", p.RechamberTimer)
	}

	// Phase 2 timing window -> phase 3 lockout
	sys.Tick(w, 0.4)
	if p.RechamberPhase != 3 {
		t.Errorf("expected phase 3, got %d", p.RechamberPhase)
	}

	// Phase 3 lockout -> phase 0 idle
	sys.Tick(w, 0.9)
	if p.RechamberPhase != 0 {
		t.Errorf("expected phase 0, got %d", p.RechamberPhase)
	}
}

func TestRechamberBuffExpires(t *testing.T) {
	p := entity.NewPlayer(1, "gunner")
	p.RechamberBuff = true
	p.RechamberBuffTimer = 4.0

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	sys := CombatSystem{}

	sys.Tick(w, 2.0)
	if !p.RechamberBuff {
		t.Error("rechamber buff should still be active after 2s")
	}

	sys.Tick(w, 3.0)
	if p.RechamberBuff {
		t.Error("rechamber buff should expire after 5s total")
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

func TestOverclockInputActivates(t *testing.T) {
	p := entity.NewPlayer(1, "gunner")
	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)

	payload := []byte{entity.ActionOverclock, 0, 0, 0, 0} // action + 4 bytes aim pitch
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: 0x0031, Payload: payload}}
	inputSys := InputSystem{}
	inputSys.Tick(w, 0.05)

	if !p.OverclockActive {
		t.Error("overclock should be active after input")
	}
	if p.OverclockTimer != 7.0 {
		t.Errorf("overclock timer = %f, want 7.0", p.OverclockTimer)
	}
	if p.OverclockCooldown != 15.0 {
		t.Errorf("overclock cooldown = %f, want 15.0", p.OverclockCooldown)
	}
}

func TestOverclockBlockedDuringCooldown(t *testing.T) {
	p := entity.NewPlayer(1, "gunner")
	p.OverclockCooldown = 5.0

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	payload := []byte{entity.ActionOverclock, 0, 0, 0, 0}
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: 0x0031, Payload: payload}}
	inputSys := InputSystem{}
	inputSys.Tick(w, 0.05)

	if p.OverclockActive {
		t.Error("overclock should not activate during cooldown")
	}
}

func TestOverclockFireRateBoost(t *testing.T) {
	p := entity.NewPlayer(1, "gunner")
	p.OverclockActive = true
	p.Position = entity.Vec3{X: 0, Y: 0.1, Z: 5}

	e := entity.NewEnemy(0, 2000.0, "guard_captain")
	e.Alive = true
	e.Position = entity.Vec3{X: 0, Y: 0.1, Z: 0}

	w := makeWorld(map[uint16]*entity.Player{1: p}, []*entity.Enemy{e})
	payload := []byte{entity.ActionShoot, 0, 0, 0, 0}
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: 0x0031, Payload: payload}}
	inputSys := InputSystem{}
	inputSys.Tick(w, 0.05)

	if p.FireCooldown != 0.10 {
		t.Errorf("fire cooldown = %f, want 0.10 (overclock)", p.FireCooldown)
	}
}

func TestRechamberInputStartsWindup(t *testing.T) {
	p := entity.NewPlayer(1, "gunner")
	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)

	payload := []byte{entity.ActionRechamber, 0, 0, 0, 0}
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: 0x0031, Payload: payload}}
	inputSys := InputSystem{}
	inputSys.Tick(w, 0.05)

	if p.RechamberPhase != 1 {
		t.Errorf("rechamber phase = %d, want 1 (windup)", p.RechamberPhase)
	}
	if p.RechamberTimer != 0.6 {
		t.Errorf("rechamber timer = %f, want 0.6", p.RechamberTimer)
	}
	if p.FireCooldown != 0.6 {
		t.Errorf("fire cooldown = %f, want 0.6 (locked during windup)", p.FireCooldown)
	}
}

func TestRechamberConfirmDuringWindow(t *testing.T) {
	p := entity.NewPlayer(1, "gunner")
	p.RechamberPhase = 2
	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)

	payload := []byte{entity.ActionRechamberConfirm, 0, 0, 0, 0}
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: 0x0031, Payload: payload}}
	inputSys := InputSystem{}
	inputSys.Tick(w, 0.05)

	if !p.RechamberBuff {
		t.Error("rechamber buff should be active after confirm in timing window")
	}
	if p.RechamberBuffTimer != 4.0 {
		t.Errorf("rechamber buff timer = %f, want 4.0", p.RechamberBuffTimer)
	}
	if p.RechamberPhase != 0 {
		t.Errorf("rechamber phase = %d, want 0 (reset after confirm)", p.RechamberPhase)
	}
}

func TestRechamberConfirmOutsideWindowIgnored(t *testing.T) {
	p := entity.NewPlayer(1, "gunner")
	p.RechamberPhase = 1 // still in windup, not timing window
	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)

	payload := []byte{entity.ActionRechamberConfirm, 0, 0, 0, 0}
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: 0x0031, Payload: payload}}
	inputSys := InputSystem{}
	inputSys.Tick(w, 0.05)

	if p.RechamberBuff {
		t.Error("rechamber buff should not activate outside timing window")
	}
	if p.RechamberPhase != 1 {
		t.Errorf("rechamber phase should remain 1, got %d", p.RechamberPhase)
	}
}

func TestRechamberBlockedDuringFireCooldown(t *testing.T) {
	p := entity.NewPlayer(1, "gunner")
	p.FireCooldown = 0.1

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	payload := []byte{entity.ActionRechamber, 0, 0, 0, 0}
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: 0x0031, Payload: payload}}
	inputSys := InputSystem{}
	inputSys.Tick(w, 0.05)

	if p.RechamberPhase != 0 {
		t.Error("rechamber should not start during fire cooldown")
	}
}

// --- Vanguard: Blade Swirl tests ---

func TestBladeSwirlMultiTick(t *testing.T) {
	p := entity.NewPlayer(1, "vanguard")
	p.Position = entity.Vec3{X: 0, Y: 0.1, Z: 0}
	p.BladeSwirl = true
	p.BladeSwirlTimer = 1.5
	p.BladeSwirlTicks = 0

	e := entity.NewEnemy(0, 2000.0, "guard_captain")
	e.Alive = true
	e.Position = entity.Vec3{X: 2, Y: 0.1, Z: 0} // within 4.0 radius

	w := makeWorld(map[uint16]*entity.Player{1: p}, []*entity.Enemy{e})
	sys := CombatSystem{}

	// After 0.55s: (1.5-0.95)/0.5 = 1.1 -> expectedTicks=1, should deliver 1 tick
	sys.Tick(w, 0.55)
	if p.BladeSwirlTicks != 1 {
		t.Errorf("after 0.55s: BladeSwirlTicks = %d, want 1", p.BladeSwirlTicks)
	}
	if len(w.DamageEvents) != 1 {
		t.Errorf("after 0.55s: DamageEvents = %d, want 1", len(w.DamageEvents))
	}

	// After another 0.5s (1.05s total): (1.5-0.45)/0.5 = 2.1 -> expectedTicks=2
	sys.Tick(w, 0.5)
	if p.BladeSwirlTicks != 2 {
		t.Errorf("after 1.05s: BladeSwirlTicks = %d, want 2", p.BladeSwirlTicks)
	}
	if len(w.DamageEvents) != 2 {
		t.Errorf("after 1.05s: DamageEvents = %d, want 2", len(w.DamageEvents))
	}

	// After another 0.5s (1.55s total): timer expired, swirl should end
	sys.Tick(w, 0.5)
	if p.BladeSwirl {
		t.Error("BladeSwirl should be false after timer expires")
	}
}

func TestBladeSwirlCooldownPreventsReuse(t *testing.T) {
	p := entity.NewPlayer(1, "vanguard")
	p.Stamina = 100.0
	p.BladeSwirlCooldown = 5.0

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	payload := []byte{entity.ActionBladeSwirl, 0, 0, 0, 0}
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: 0x0031, Payload: payload}}
	inputSys := InputSystem{}
	inputSys.Tick(w, 0.05)

	if p.BladeSwirl {
		t.Error("BladeSwirl should not activate during cooldown")
	}
	if p.Stamina != 100.0 {
		t.Errorf("stamina should be unchanged at 100.0, got %f", p.Stamina)
	}
}

func TestGroundSlamCooldownPreventsReuse(t *testing.T) {
	p := entity.NewPlayer(1, "vanguard")
	p.Stamina = 100.0
	p.GroundSlamCooldown = 3.0

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	payload := []byte{entity.ActionGroundSlam, 0, 0, 0, 0}
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: 0x0031, Payload: payload}}
	inputSys := InputSystem{}
	inputSys.Tick(w, 0.05)

	if p.GroundSlamCooldown != 3.0 {
		t.Errorf("GroundSlamCooldown should remain 3.0, got %f", p.GroundSlamCooldown)
	}
	if p.Stamina != 100.0 {
		t.Errorf("stamina should be unchanged at 100.0, got %f", p.Stamina)
	}
}

func TestGroundSlamConsumesStamina(t *testing.T) {
	p := entity.NewPlayer(1, "vanguard")
	p.Stamina = 100.0
	p.Position = entity.Vec3{X: 0, Y: 0.1, Z: 0}

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	payload := []byte{entity.ActionGroundSlam, 0, 0, 0, 0}
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: 0x0031, Payload: payload}}
	inputSys := InputSystem{}
	inputSys.Tick(w, 0.05)

	if p.Stamina != 80.0 {
		t.Errorf("stamina = %f, want 80.0 (100 - 20)", p.Stamina)
	}
	if p.GroundSlamCooldown != 8.0 {
		t.Errorf("GroundSlamCooldown = %f, want 8.0", p.GroundSlamCooldown)
	}
	if p.FireCooldown != 1.2 {
		t.Errorf("FireCooldown = %f, want 1.2", p.FireCooldown)
	}
}

func TestBladeSwirlCooldownTicksDown(t *testing.T) {
	p := entity.NewPlayer(1, "vanguard")
	p.BladeSwirlCooldown = 10.0

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	sys := CombatSystem{}

	sys.Tick(w, 4.0)
	if p.BladeSwirlCooldown < 5.9 || p.BladeSwirlCooldown > 6.1 {
		t.Errorf("BladeSwirlCooldown = %f, want ~6.0", p.BladeSwirlCooldown)
	}

	sys.Tick(w, 7.0)
	if p.BladeSwirlCooldown != 0.0 {
		t.Errorf("BladeSwirlCooldown = %f, want 0.0 (clamped)", p.BladeSwirlCooldown)
	}
}

func TestGroundSlamCooldownTicksDown(t *testing.T) {
	p := entity.NewPlayer(1, "vanguard")
	p.GroundSlamCooldown = 8.0

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	sys := CombatSystem{}

	sys.Tick(w, 3.0)
	if p.GroundSlamCooldown < 4.9 || p.GroundSlamCooldown > 5.1 {
		t.Errorf("GroundSlamCooldown = %f, want ~5.0", p.GroundSlamCooldown)
	}

	sys.Tick(w, 6.0)
	if p.GroundSlamCooldown != 0.0 {
		t.Errorf("GroundSlamCooldown = %f, want 0.0 (clamped)", p.GroundSlamCooldown)
	}
}

func TestBladeSwirlBlockedByInsufficientStamina(t *testing.T) {
	p := entity.NewPlayer(1, "vanguard")
	p.Stamina = 20.0 // need 25

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	payload := []byte{entity.ActionBladeSwirl, 0, 0, 0, 0}
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: 0x0031, Payload: payload}}
	inputSys := InputSystem{}
	inputSys.Tick(w, 0.05)

	if p.BladeSwirl {
		t.Error("BladeSwirl should not activate with insufficient stamina")
	}
	if p.Stamina != 20.0 {
		t.Errorf("stamina should be unchanged at 20.0, got %f", p.Stamina)
	}
}

// TestBladeSwirl3xIntegration fires 3 consecutive Blade Swirls at a nearby enemy,
// ticking cooldowns between each activation. Verifies damage events are produced
// and enemy HP decreases across all 3 activations + multi-tick damage.
func TestBladeSwirl3xIntegration(t *testing.T) {
	p := entity.NewPlayer(1, "vanguard")
	p.Stamina = 200.0    // enough for 3 swirls (40 each = 120)
	p.MaxStamina = 200.0
	p.Position = entity.Vec3{X: 0, Y: 0.1, Z: 0}
	p.RotationY = 0

	e := entity.NewEnemy(1000, 500.0, "test_enemy")
	e.Position = entity.Vec3{X: 2, Y: 0.1, Z: 0} // 2m away, within 6m radius
	e.State = entity.EnemyChase                    // not patrol, so AggroEnemy won't interfere

	enemies := []*entity.Enemy{e}
	w := makeWorld(map[uint16]*entity.Player{1: p}, enemies)

	inputSys := InputSystem{}
	combatSys := CombatSystem{}
	payload := []byte{entity.ActionBladeSwirl, 0, 0, 0, 0, 0, 0, 0, 0} // action + aim_pitch(4) + rot_y(4)

	totalDamageEvents := 0
	startHP := e.Health
	t.Logf("start: enemy HP=%.0f, player stamina=%.0f", e.Health, p.Stamina)

	for swirl := 0; swirl < 3; swirl++ {
		// Fire blade swirl
		w.DamageEvents = w.DamageEvents[:0]
		w.InputQueue = []InputMsg{{PeerID: 1, Opcode: 0x0031, Payload: payload}}
		inputSys.Tick(w, 0.05)

		if !p.BladeSwirl {
			t.Fatalf("swirl %d: BladeSwirl should be active", swirl+1)
		}

		eventsFromInput := len(w.DamageEvents)
		t.Logf("swirl %d: immediate hits=%d, enemy HP=%.0f, stamina=%.0f, fire_cd=%.2f, swirl_cd=%.2f",
			swirl+1, eventsFromInput, e.Health, p.Stamina, p.FireCooldown, p.BladeSwirlCooldown)
		totalDamageEvents += eventsFromInput

		// Tick combat system for the full 1.5s duration + 1 extra tick for float rounding
		for tick := 0; tick < 32; tick++ {
			w.DamageEvents = w.DamageEvents[:0]
			combatSys.Tick(w, 0.05)
			totalDamageEvents += len(w.DamageEvents)
		}

		t.Logf("swirl %d after ticks: enemy HP=%.0f, blade_swirl=%v, fire_cd=%.2f, swirl_cd=%.2f",
			swirl+1, e.Health, p.BladeSwirl, p.FireCooldown, p.BladeSwirlCooldown)

		if p.BladeSwirl {
			t.Errorf("swirl %d: BladeSwirl should have ended after 1.5s", swirl+1)
		}

		// Tick down cooldowns: 10s swirl CD, 1.5s fire CD already elapsed
		// Need to tick down swirl cooldown fully (10s - 1.5s already ticked = 8.5s remaining)
		for tick := 0; tick < 200; tick++ { // 10s at 0.05s/tick
			combatSys.Tick(w, 0.05)
		}

		t.Logf("swirl %d after cooldown: fire_cd=%.2f, swirl_cd=%.2f, stamina=%.0f",
			swirl+1, p.FireCooldown, p.BladeSwirlCooldown, p.Stamina)
	}

	totalDamage := startHP - e.Health
	t.Logf("FINAL: enemy HP=%.0f (took %.0f damage), total damage events=%d", e.Health, totalDamage, totalDamageEvents)

	// Each swirl should produce: 1 immediate hit + 2 multi-tick hits = 3 hits per swirl
	// 3 swirls x 3 hits x 25 damage = 225 total damage (if enemy in range for all ticks)
	if totalDamageEvents < 3 {
		t.Errorf("expected at least 3 total damage events across 3 swirls, got %d", totalDamageEvents)
	}
	if totalDamage < 50 {
		t.Errorf("expected at least 50 total damage, got %.0f", totalDamage)
	}
	if e.Health >= startHP {
		t.Error("enemy HP should have decreased")
	}
}

// TestSwirlSlamSwirlSlamIntegration fires Blade Swirl → Ground Slam → Blade Swirl → Ground Slam,
// ticking cooldowns between each. Verifies all 4 abilities activate and deal damage.
func TestSwirlSlamSwirlSlamIntegration(t *testing.T) {
	p := entity.NewPlayer(1, "vanguard")
	p.Stamina = 300.0
	p.MaxStamina = 300.0
	p.Position = entity.Vec3{X: 0, Y: 0.1, Z: 0}
	p.RotationY = float32(3.14159) // facing +Z

	e := entity.NewEnemy(1000, 2000.0, "test_enemy")
	e.Position = entity.Vec3{X: 0, Y: 0.1, Z: 3} // 3m ahead, within both AoE ranges
	e.State = entity.EnemyChase

	enemies := []*entity.Enemy{e}
	w := makeWorld(map[uint16]*entity.Player{1: p}, enemies)

	inputSys := InputSystem{}
	combatSys := CombatSystem{}

	swirlPayload := []byte{entity.ActionBladeSwirl, 0, 0, 0, 0, 0, 0, 0, 0}
	slamPayload := []byte{entity.ActionGroundSlam, 0, 0, 0, 0, 0, 0, 0, 0}

	type step struct {
		name    string
		payload []byte
	}
	steps := []step{
		{"Blade Swirl 1", swirlPayload},
		{"Ground Slam 1", slamPayload},
		{"Blade Swirl 2", swirlPayload},
		{"Ground Slam 2", slamPayload},
	}

	startHP := e.Health
	totalEvents := 0

	for i, s := range steps {
		hpBefore := e.Health
		w.DamageEvents = w.DamageEvents[:0]

		// Fire the ability
		w.InputQueue = []InputMsg{{PeerID: 1, Opcode: 0x0031, Payload: s.payload}}
		inputSys.Tick(w, 0.05)

		eventsFromInput := len(w.DamageEvents)
		totalEvents += eventsFromInput

		isSwirl := s.payload[0] == entity.ActionBladeSwirl
		isSwirlActive := p.BladeSwirl

		t.Logf("step %d [%s]: input_hits=%d, HP=%.0f→%.0f, swirl=%v, fire_cd=%.2f, swirl_cd=%.2f, slam_cd=%.2f, stamina=%.0f",
			i+1, s.name, eventsFromInput, hpBefore, e.Health, isSwirlActive,
			p.FireCooldown, p.BladeSwirlCooldown, p.GroundSlamCooldown, p.Stamina)

		if isSwirl && !isSwirlActive {
			t.Errorf("step %d [%s]: BladeSwirl should be active", i+1, s.name)
		}
		if !isSwirl && eventsFromInput == 0 {
			t.Errorf("step %d [%s]: Ground Slam should produce at least 1 hit (enemy is 3m away, cone radius 7m)", i+1, s.name)
		}

		// If blade swirl, tick through its full duration (1.6s = 32 ticks)
		if isSwirl {
			for tick := 0; tick < 32; tick++ {
				w.DamageEvents = w.DamageEvents[:0]
				combatSys.Tick(w, 0.05)
				totalEvents += len(w.DamageEvents)
			}
			if p.BladeSwirl {
				t.Errorf("step %d [%s]: BladeSwirl should have ended", i+1, s.name)
			}
		}

		// Tick down ALL cooldowns: swirl (10s), slam (8s), fire (1.2-1.5s)
		// Tick 220 times (11s) to clear everything
		for tick := 0; tick < 220; tick++ {
			combatSys.Tick(w, 0.05)
		}

		t.Logf("step %d [%s] after cooldown: fire_cd=%.2f, swirl_cd=%.2f, slam_cd=%.2f",
			i+1, s.name, p.FireCooldown, p.BladeSwirlCooldown, p.GroundSlamCooldown)

		if p.BladeSwirlCooldown > 0 {
			t.Errorf("step %d: BladeSwirlCooldown should be 0 after 11s, got %.2f", i+1, p.BladeSwirlCooldown)
		}
		if p.GroundSlamCooldown > 0 {
			t.Errorf("step %d: GroundSlamCooldown should be 0 after 11s, got %.2f", i+1, p.GroundSlamCooldown)
		}
		if p.FireCooldown > 0 {
			t.Errorf("step %d: FireCooldown should be 0 after 11s, got %.2f", i+1, p.FireCooldown)
		}
	}

	totalDamage := startHP - e.Health
	t.Logf("FINAL: enemy HP=%.0f (took %.0f damage), total events=%d, stamina=%.0f",
		e.Health, totalDamage, totalEvents, p.Stamina)

	// 2 swirls × 4 hits × 25 dmg = 200 + 2 slams × 1 hit × 60 dmg = 120 → 320 total
	if totalEvents < 4 {
		t.Errorf("expected at least 4 total damage events, got %d", totalEvents)
	}
	if totalDamage < 200 {
		t.Errorf("expected at least 200 total damage, got %.0f", totalDamage)
	}
}

func TestVanguardStaminaRegen(t *testing.T) {
	p := entity.NewPlayer(1, "vanguard")
	p.Stamina = 50.0
	p.StaminaDelay = 0 // no delay, regen should start immediately

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	sys := CombatSystem{}

	t.Logf("before: stamina=%.1f delay=%.2f regen=%.1f max=%.1f", p.Stamina, p.StaminaDelay, p.StaminaRegen, p.MaxStamina)

	// Tick 1 second — should regen 30 stamina
	sys.Tick(w, 1.0)
	t.Logf("after 1s: stamina=%.1f delay=%.2f", p.Stamina, p.StaminaDelay)

	if p.Stamina < 79.0 || p.Stamina > 81.0 {
		t.Errorf("stamina after 1s regen = %.1f, want ~80.0 (50 + 30)", p.Stamina)
	}

	// Spend stamina, verify delay
	p.Stamina = 50.0
	p.StaminaDelay = 0.6
	before := p.Stamina
	sys.Tick(w, 0.3) // 0.3s < 0.6s delay — no regen yet
	t.Logf("during delay: stamina=%.1f delay=%.2f", p.Stamina, p.StaminaDelay)
	if p.Stamina != before {
		t.Errorf("stamina should not regen during delay, got %.1f (was %.1f)", p.Stamina, before)
	}

	sys.Tick(w, 0.5) // delay expires at 0.3+0.5=0.8 > 0.6, then 0.2s of regen
	t.Logf("after delay: stamina=%.1f delay=%.2f", p.Stamina, p.StaminaDelay)
	if p.Stamina <= 50.0 {
		t.Errorf("stamina should have started regening after delay expired, got %.1f", p.Stamina)
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

// =============================================================================
// Blade Dancer — comprehensive test of all 20 spells with 4 enemies
// =============================================================================

// TestAllBladeDancerSpells fires every spell exactly once in multi-enemy context.
// Enemy layout (player at origin, facing +Z with rotY=PI):
//   - eFront     (1000): (0, 0.1, 3)    — in aim line, 3m from player. Hit by ST, PC(r>=3), TC, NearestN.
//   - eNearFront (1001): (1, 0.1, 3.5)  — 1.1m from eFront, 3.6m from player. Hit by TC(r>=2) but NOT PC(r=3).
//   - eSide      (1002): (3, 0.1, 0)    — 3m from player, 4.2m from eFront. Hit by PC(r>=3) but NOT TC(r=4 centered on eFront).
//   - eFar       (1003): (0, 0.1, 20)   — 20m away. Never hit.
//
// For NearestN spells: eFront, eNearFront, eSide all have threat (in combat).
// eFar has no threat (out of combat) so NearestN skips it.
func TestAllBladeDancerSpells(t *testing.T) {
	const hp float32 = 5000.0

	type spellExpect struct {
		spellIdx      int
		name          string
		originCfg     int
		destCfg       int
		frontDmg      float32 // eFront (1000)
		nearFrontDmg  float32 // eNearFront (1001)
		sideDmg       float32 // eSide (1002)
		farDmg        float32 // eFar (1003) — always 0
		shieldGain    float32
		hasDR         bool
		hasDoT        bool
		dotPerTick    float32
		dotTotalTicks int // ticks over full duration
		dotTargets    int // how many enemies get DoT
	}

	// Distances:
	// eFront to player: 3.0m
	// eNearFront to player: sqrt(1+12.25) = 3.64m
	// eNearFront to eFront: sqrt(1+0.25) = 1.12m
	// eSide to player: 3.0m
	// eSide to eFront: sqrt(9+9) = 4.24m
	//
	// PC circle r=3: hits eFront (3m=boundary), eSide (3m=boundary), misses eNearFront (3.64m)
	// PC circle r=4: hits eFront, eNearFront (3.64m), eSide (3m)
	// PC circle r=5: hits eFront, eNearFront, eSide
	// TC circle r=4 (centered on eFront at Z=3): hits eNearFront (1.12m), misses eSide (4.24m > 4)
	//   also hits eFront itself (0m = center)
	// TC circle r=5 (centered on eFront): hits eNearFront (1.12m), hits eSide (4.24m < 5)
	//   also hits eFront itself
	// NearestN: sorted by dist to player: eFront(3m), eSide(3m), eNearFront(3.64m). All in combat.

	spells := []spellExpect{
		// From Orbit (config 0)
		// 0: Shielded Sweep — PC circle r=4: eFront(3m)=HIT, eNearFront(3.64m)=HIT, eSide(3m)=HIT
		{0, "Shielded Sweep", 0, 1, 8, 8, 8, 0, 0, true, false, 0, 0, 0},
		// 1: Guarded Thrust — ST: eFront only
		{1, "Guarded Thrust", 0, 2, 25, 0, 0, 0, 8, false, false, 0, 0, 0},
		// 2: Protected Scatter — NearestN(3): hits 3 nearest in-combat = eFront+eSide+eNearFront
		{2, "Protected Scatter", 0, 3, 5, 5, 5, 0, 0, true, true, 1.5, 11, 3},
		// 3: Fortified Command — TC circle r=5 at eFront: eFront(0m)=HIT, eNearFront(1.12m)=HIT, eSide(4.24m)=HIT
		{3, "Fortified Command", 0, 4, 5, 5, 5, 0, 0, true, false, 0, 0, 0},

		// From Fan (config 1)
		// 4: Reaping Guard — PC circle r=3: eFront(3m)=HIT, eNearFront(3.64m)=MISS, eSide(3m)=HIT
		{4, "Reaping Guard", 1, 0, 8, 0, 8, 0, 12, false, false, 0, 0, 0},
		// 5: Cleaving Pierce — ST: eFront only
		{5, "Cleaving Pierce", 1, 2, 30, 0, 0, 0, 0, false, false, 0, 0, 0},
		// 6: Slashing Spread — TC circle r=5 at eFront: eFront+eNearFront+eSide all HIT
		{6, "Slashing Spread", 1, 3, 8, 8, 8, 0, 0, false, true, 1.5, 9, 3},
		// 7: Sweeping Hex — TC circle r=5 at eFront: all 3 HIT
		{7, "Sweeping Hex", 1, 4, 10, 10, 10, 0, 0, false, false, 0, 0, 0},

		// From Lance (config 2)
		// 8: Piercing Barrier — ST: eFront only
		{8, "Piercing Barrier", 2, 0, 18, 0, 0, 0, 15, false, false, 0, 0, 0},
		// 9: Focused Slash — TC circle r=4 at eFront: eFront(0m)=HIT, eNearFront(1.12m)=HIT, eSide(4.24m>4)=MISS
		{9, "Focused Slash", 2, 1, 15, 15, 0, 0, 0, false, false, 0, 0, 0},
		// 10: Targeted Spread — ST + DoT: eFront only
		{10, "Targeted Spread", 2, 3, 12, 0, 0, 0, 0, false, true, 2.0, 14, 1},
		// 11: Pinning Strike — ST: eFront only
		{11, "Pinning Strike", 2, 4, 25, 0, 0, 0, 0, false, false, 0, 0, 0},

		// From Scatter (config 3)
		// 12: Dispersed Shield — self-buff: no damage
		{12, "Dispersed Shield", 3, 0, 0, 0, 0, 0, 18, true, false, 0, 0, 0},
		// 13: Rain of Blades — TC circle r=5 at eFront: all 3 HIT + DoT
		{13, "Rain of Blades", 3, 1, 15, 15, 15, 0, 0, false, true, 1.0, 9, 3},
		// 14: Converging Strike — ST + DoT: eFront only
		{14, "Converging Strike", 3, 2, 32, 0, 0, 0, 0, false, true, 1.5, 9, 1},
		// 15: Chaos Bind — NearestN(4): all 3 in-combat hit (only 3 in combat, max 4)
		{15, "Chaos Bind", 3, 4, 8, 8, 8, 0, 0, false, false, 0, 0, 0},

		// From Crown (config 4)
		// 16: Commanding Ward — self-buff: no damage
		{16, "Commanding Ward", 4, 0, 0, 0, 0, 0, 20, false, false, 0, 0, 0},
		// 17: Royal Cleave — PC circle r=5: all 3 HIT
		{17, "Royal Cleave", 4, 1, 12, 12, 12, 0, 0, false, false, 0, 0, 0},
		// 18: Decree Strike — ST: eFront only
		{18, "Decree Strike", 4, 2, 28, 0, 0, 0, 0, false, false, 0, 0, 0},
		// 19: Sovereign Scatter — NearestN(3): hits 3 nearest in-combat
		{19, "Sovereign Scatter", 4, 3, 5, 5, 5, 0, 0, false, true, 1.5, 11, 3},
	}

	for _, sp := range spells {
		t.Run(sp.name, func(t *testing.T) {
			p := entity.NewPlayer(1, "blade_dancer")
			p.Position = entity.Vec3{X: 0, Y: 0.1, Z: 0}
			p.RotationY = float32(math.Pi) // face +Z
			p.AimPitch = 0
			p.Config = sp.originCfg
			p.GCDTimer = 0

			eFront := entity.NewEnemy(1000, hp, "test")
			eFront.Position = entity.Vec3{X: 0, Y: 0.1, Z: 3}
			eFront.State = entity.EnemyChase
			eFront.ThreatTable[1] = 10.0 // in combat (needed for NearestN)

			eNearFront := entity.NewEnemy(1001, hp, "test")
			eNearFront.Position = entity.Vec3{X: 1, Y: 0.1, Z: 3.5}
			eNearFront.State = entity.EnemyChase
			eNearFront.ThreatTable[1] = 5.0

			eSide := entity.NewEnemy(1002, hp, "test")
			eSide.Position = entity.Vec3{X: 3, Y: 0.1, Z: 0}
			eSide.State = entity.EnemyChase
			eSide.ThreatTable[1] = 5.0

			eFar := entity.NewEnemy(1003, hp, "test")
			eFar.Position = entity.Vec3{X: 0, Y: 0.1, Z: 20}
			eFar.State = entity.EnemyChase
			// eFar has NO threat — out of combat, NearestN should skip

			enemies := []*entity.Enemy{eFront, eNearFront, eSide, eFar}
			w := makeWorld(map[uint16]*entity.Player{1: p}, enemies)

			actionID := entity.ActionBDSpellBase + uint8(sp.spellIdx)
			payload := codec.EncodeAbilityInput(actionID, 0, float32(math.Pi))

			inputSys := InputSystem{}
			w.InputQueue = []InputMsg{{PeerID: 1, Opcode: 0x0031, Payload: payload}}
			inputSys.Tick(w, 0.05)

			// Config transition
			if p.Config != sp.destCfg {
				t.Errorf("config = %d, want %d", p.Config, sp.destCfg)
			}
			if p.GCDTimer <= 0 {
				t.Error("GCD should be set")
			}

			// Check immediate damage on all 4 enemies
			frontDmg := hp - eFront.Health
			nearFrontDmg := hp - eNearFront.Health
			sideDmg := hp - eSide.Health
			farDmg := hp - eFar.Health

			if frontDmg != sp.frontDmg {
				t.Errorf("eFront dmg = %.1f, want %.1f", frontDmg, sp.frontDmg)
			}
			if nearFrontDmg != sp.nearFrontDmg {
				t.Errorf("eNearFront dmg = %.1f, want %.1f", nearFrontDmg, sp.nearFrontDmg)
			}
			if sideDmg != sp.sideDmg {
				t.Errorf("eSide dmg = %.1f, want %.1f", sideDmg, sp.sideDmg)
			}
			if farDmg != sp.farDmg {
				t.Errorf("eFar dmg = %.1f, want 0 (should never be hit)", farDmg)
			}

			// Shield
			if sp.shieldGain > 0 {
				expected := sp.shieldGain
				if expected > 25.0 {
					expected = 25.0
				}
				if p.BDShieldHP != expected {
					t.Errorf("shield = %.0f, want %.0f", p.BDShieldHP, expected)
				}
			} else if p.BDShieldHP != 0 {
				t.Errorf("shield = %.0f, want 0", p.BDShieldHP)
			}

			// DR
			if sp.hasDR && (p.BDDRFactor <= 0 || p.BDDRTimer <= 0) {
				t.Errorf("DR should be active: factor=%.2f timer=%.2f", p.BDDRFactor, p.BDDRTimer)
			}
			if !sp.hasDR && p.BDDRFactor > 0 && p.BDDRTimer > 0 {
				t.Errorf("DR should NOT be active: factor=%.2f timer=%.2f", p.BDDRFactor, p.BDDRTimer)
			}

			// DoT
			if sp.hasDoT {
				if len(w.BDDoTs) != sp.dotTargets {
					t.Fatalf("DoT count = %d, want %d targets", len(w.BDDoTs), sp.dotTargets)
				}
				for _, dot := range w.BDDoTs {
					if dot.Damage != sp.dotPerTick {
						t.Errorf("DoT tick dmg = %.1f, want %.1f", dot.Damage, sp.dotPerTick)
					}
				}

				// Tick DoTs to completion for eFront
				combatSys := CombatSystem{}
				frontHPBefore := eFront.Health
				dotDuration := w.BDDoTs[0].Remaining
				ticks := int((dotDuration+1.0)/0.05) + 1
				for i := 0; i < ticks; i++ {
					w.DamageEvents = w.DamageEvents[:0]
					combatSys.Tick(w, 0.05)
				}
				frontDotDmg := frontHPBefore - eFront.Health
				expectedDotDmg := sp.dotPerTick * float32(sp.dotTotalTicks)
				if frontDotDmg < expectedDotDmg-1 || frontDotDmg > expectedDotDmg+1 {
					t.Errorf("eFront DoT total = %.0f, want ~%.0f (%d ticks x %.1f)",
						frontDotDmg, expectedDotDmg, sp.dotTotalTicks, sp.dotPerTick)
				}
				if len(w.BDDoTs) != 0 {
					t.Errorf("DoTs should be expired, got %d", len(w.BDDoTs))
				}
			} else if len(w.BDDoTs) != 0 {
				t.Errorf("expected no DoTs, got %d", len(w.BDDoTs))
			}

			// eFar never touched
			if eFar.Health != hp {
				t.Errorf("eFar HP = %.0f, want %.0f", eFar.Health, hp)
			}
		})
	}
}
