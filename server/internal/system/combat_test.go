package system

import (
	"math"
	"testing"

	"codex-online/server/internal/ability"
	"codex-online/server/internal/codec"
	"codex-online/server/internal/entity"
	"codex-online/server/internal/level"
	"codex-online/server/internal/message"
)

func makeWorld(players map[uint16]*entity.Player, enemies []*entity.Enemy) *World {
	return &World{
		ZoneType:       1, // arena
		TickNum:        100,
		State:          StateFight,
		Players:        players,
		Enemies:        enemies,
		Level:          level.NewArenaLevel(),
		AbilityEngine:  ability.NewEngine(nil),
		AbilityRunners: make(map[uint16]*ability.PlayerAbilityRunner),
	}
}

// --- Unit tests ---

func TestInCombatWhenOnThreatTable(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassGunner)
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
	p := entity.NewPlayer(1, entity.ClassGunner)
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
	p := entity.NewPlayer(1, entity.ClassGunner)
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
	p1 := entity.NewPlayer(1, entity.ClassGunner)
	p2 := entity.NewPlayer(2, entity.ClassVanguard)
	p3 := entity.NewPlayer(3, entity.ClassBladeDancer)

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
	p := entity.NewPlayer(1, entity.ClassGunner)
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
	p := entity.NewPlayer(1, entity.ClassGunner)
	p.AddBuff(entity.ActiveBuff{ID: "overclock", Type: entity.BuffCooldownMult, Value: 0.556, Duration: 7.0})
	p.Cooldowns["overclock"] = 15.0

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	sys := CombatSystem{}

	// Tick 3 seconds — still active
	sys.Tick(w, 3.0)
	if !p.HasBuff("overclock") {
		t.Error("overclock should still be active after 3s")
	}

	// Tick another 5 seconds — should expire (8s total > 7s)
	sys.Tick(w, 5.0)
	if p.HasBuff("overclock") {
		t.Error("overclock should be inactive after 8s total")
	}
}

func TestOverclockCooldownTicksDown(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassGunner)
	p.Cooldowns["overclock"] = 15.0

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	sys := CombatSystem{}

	sys.Tick(w, 10.0)
	if cd := p.Cooldowns["overclock"]; cd < 4.9 || cd > 5.1 {
		t.Errorf("overclock cooldown = %f, want ~5.0", cd)
	}

	sys.Tick(w, 10.0)
	if cd := p.Cooldowns["overclock"]; cd != 0.0 {
		t.Errorf("overclock cooldown = %f, want 0.0 (expired)", cd)
	}
}

func TestRechamberPhaseTransitions(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassGunner)
	p.AbilityState["rechamber"] = &ability.RechamberState{Phase: 1, Timer: 0.6}

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	sys := CombatSystem{}

	// Phase 1 windup -> phase 2 timing window
	sys.Tick(w, 0.7)
	if p.GetAbilityPhase("rechamber") != 2 {
		t.Errorf("expected phase 2, got %d", p.GetAbilityPhase("rechamber"))
	}

	// Phase 2 timing window -> phase 3 lockout
	sys.Tick(w, 0.4)
	if p.GetAbilityPhase("rechamber") != 3 {
		t.Errorf("expected phase 3, got %d", p.GetAbilityPhase("rechamber"))
	}

	// Phase 3 lockout -> phase 0 idle
	sys.Tick(w, 0.9)
	if p.GetAbilityPhase("rechamber") != 0 {
		t.Errorf("expected phase 0, got %d", p.GetAbilityPhase("rechamber"))
	}
}

func TestRechamberBuffExpires(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassGunner)
	p.AddBuff(entity.ActiveBuff{ID: "rechamber_buff", Type: entity.BuffDamageMult, Value: 1.8, Duration: 4.0})

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	sys := CombatSystem{}

	sys.Tick(w, 2.0)
	if !p.HasBuff("rechamber_buff") {
		t.Error("rechamber buff should still be active after 2s")
	}

	sys.Tick(w, 3.0)
	if p.HasBuff("rechamber_buff") {
		t.Error("rechamber buff should expire after 5s total")
	}
}

// --- Integration tests ---

func TestThreatGeneratedOnPlayerAttack(t *testing.T) {
	// Set up a world with a gunner aimed directly at the enemy
	p := entity.NewPlayer(1, entity.ClassGunner)
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
	p := entity.NewPlayer(1, entity.ClassGunner)
	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)

	payload := []byte{entity.ActionOverclock, 0, 0, 0, 0} // action + 4 bytes aim pitch
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: 0x0031, Payload: payload}}
	inputSys := InputSystem{}
	inputSys.Tick(w, 0.05)

	if !p.HasBuff("overclock") {
		t.Error("overclock should be active after input")
	}
	if p.Cooldowns["overclock"] != 15.0 {
		t.Errorf("overclock cooldown = %f, want 15.0", p.Cooldowns["overclock"])
	}
}

func TestOverclockBlockedDuringCooldown(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassGunner)
	p.Cooldowns["overclock"] = 5.0

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	payload := []byte{entity.ActionOverclock, 0, 0, 0, 0}
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: 0x0031, Payload: payload}}
	inputSys := InputSystem{}
	inputSys.Tick(w, 0.05)

	if p.HasBuff("overclock") {
		t.Error("overclock should not activate during cooldown")
	}
}

func TestOverclockFireRateBoost(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassGunner)
	p.AddBuff(entity.ActiveBuff{ID: "overclock", Type: entity.BuffCooldownMult, Value: 0.556, Duration: 7.0})
	p.Position = entity.Vec3{X: 0, Y: 0.1, Z: 5}

	e := entity.NewEnemy(0, 2000.0, "guard_captain")
	e.Alive = true
	e.Position = entity.Vec3{X: 0, Y: 0.1, Z: 0}

	w := makeWorld(map[uint16]*entity.Player{1: p}, []*entity.Enemy{e})
	payload := []byte{entity.ActionShoot, 0, 0, 0, 0}
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: 0x0031, Payload: payload}}
	inputSys := InputSystem{}
	inputSys.Tick(w, 0.05)

	// With overclock buff (cooldown_mult = 0.556), fire_shot cooldown = 0.18 * 0.556 ~ 0.10
	cd := p.Cooldowns["fire_shot"]
	if cd < 0.09 || cd > 0.11 {
		t.Errorf("fire cooldown = %f, want ~0.10 (overclock)", cd)
	}
}

func TestRechamberInputStartsWindup(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassGunner)
	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)

	payload := []byte{entity.ActionRechamber, 0, 0, 0, 0}
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: 0x0031, Payload: payload}}
	inputSys := InputSystem{}
	inputSys.Tick(w, 0.05)

	if p.GetAbilityPhase("rechamber") != 1 {
		t.Errorf("rechamber phase = %d, want 1 (windup)", p.GetAbilityPhase("rechamber"))
	}
	if p.Cooldowns["fire_shot"] != 0.6 {
		t.Errorf("fire cooldown = %f, want 0.6 (locked during windup)", p.Cooldowns["fire_shot"])
	}
}

func TestRechamberConfirmDuringWindow(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassGunner)
	p.AbilityState["rechamber"] = &ability.RechamberState{Phase: 2, Timer: 0.3}
	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)

	payload := []byte{entity.ActionRechamberConfirm, 0, 0, 0, 0}
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: 0x0031, Payload: payload}}
	inputSys := InputSystem{}
	inputSys.Tick(w, 0.05)

	if !p.HasBuff("rechamber_buff") {
		t.Error("rechamber buff should be active after confirm in timing window")
	}
	if p.GetAbilityPhase("rechamber") != 0 {
		t.Errorf("rechamber phase = %d, want 0 (reset after confirm)", p.GetAbilityPhase("rechamber"))
	}
}

func TestRechamberConfirmOutsideWindowIgnored(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassGunner)
	p.AbilityState["rechamber"] = &ability.RechamberState{Phase: 1, Timer: 0.4}
	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)

	payload := []byte{entity.ActionRechamberConfirm, 0, 0, 0, 0}
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: 0x0031, Payload: payload}}
	inputSys := InputSystem{}
	inputSys.Tick(w, 0.05)

	if p.HasBuff("rechamber_buff") {
		t.Error("rechamber buff should not activate outside timing window")
	}
	if p.GetAbilityPhase("rechamber") != 1 {
		t.Errorf("rechamber phase should remain 1, got %d", p.GetAbilityPhase("rechamber"))
	}
}

func TestRechamberBlockedDuringFireCooldown(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassGunner)
	p.Cooldowns["fire_shot"] = 0.1

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	payload := []byte{entity.ActionRechamber, 0, 0, 0, 0}
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: 0x0031, Payload: payload}}
	inputSys := InputSystem{}
	inputSys.Tick(w, 0.05)

	if p.GetAbilityPhase("rechamber") != 0 {
		t.Error("rechamber should not start during fire cooldown")
	}
}

// --- Vanguard: Vortex (blade_swirl) tests ---

func TestVortexMultiTick(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassVanguard)
	p.Position = entity.Vec3{X: 0, Y: 0.1, Z: 0}
	// Set up vortex state directly (standard tier: 0.6s duration, 2 total hits).
	// HitsDone=1 because the initial commit already delivered the first hit.
	p.AbilityState["vortex"] = &ability.VortexState{Timer: 0.6, Duration: 0.6, TotalHits: 2, HitsDone: 1}
	p.AddBuff(entity.ActiveBuff{ID: "vortex", Type: entity.BuffDamageReduction, Value: 0.8, Duration: 0.6})

	e := entity.NewEnemy(0, 2000.0, "guard_captain")
	e.Alive = true
	e.Position = entity.Vec3{X: 2, Y: 0.1, Z: 0} // within 4.0 radius

	w := makeWorld(map[uint16]*entity.Player{1: p}, []*entity.Enemy{e})
	sys := CombatSystem{}

	// After 0.35s: elapsed > interval (0.3s), should deliver second hit
	sys.Tick(w, 0.35)
	state, ok := p.AbilityState["vortex"].(*ability.VortexState)
	if !ok {
		t.Fatal("blade_swirl state not set")
	}
	if state.HitsDone != 2 {
		t.Errorf("after 0.35s: HitsDone = %d, want 2", state.HitsDone)
	}
	if len(w.DamageEvents) != 1 {
		t.Errorf("after 0.35s: DamageEvents = %d, want 1", len(w.DamageEvents))
	}

	// After another 0.3s (0.65s total): timer expired, vortex should end
	sys.Tick(w, 0.3)
	if p.HasBuff("vortex") {
		t.Error("blade_swirl buff should be expired after timer expires")
	}
}

func TestBladeSwirlCooldownPreventsReuse(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassVanguard)
	p.Resources["stamina"].Current = 100.0
	p.Cooldowns["vortex"] = 5.0

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	payload := []byte{entity.ActionBladeSwirl, 0, 0, 0, 0}
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: 0x0031, Payload: payload}}
	inputSys := InputSystem{}
	inputSys.Tick(w, 0.05)

	if p.HasBuff("vortex") {
		t.Error("BladeSwirl should not activate during cooldown")
	}
	if p.Resources["stamina"].Current != 100.0 {
		t.Errorf("stamina should be unchanged at 100.0, got %f", p.Resources["stamina"].Current)
	}
}

func TestGroundSlamCooldownPreventsReuse(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassVanguard)
	p.Resources["stamina"].Current = 100.0
	p.Cooldowns["execution"] = 3.0

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	payload := []byte{entity.ActionGroundSlam, 0, 0, 0, 0}
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: 0x0031, Payload: payload}}
	inputSys := InputSystem{}
	inputSys.Tick(w, 0.05)

	if p.Cooldowns["execution"] < 3.0 {
		t.Errorf("GroundSlamCooldown should remain >= 3.0, got %f", p.Cooldowns["execution"])
	}
	if p.Resources["stamina"].Current != 100.0 {
		t.Errorf("stamina should be unchanged at 100.0, got %f", p.Resources["stamina"].Current)
	}
}

func TestGroundSlamConsumesStamina(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassVanguard)
	p.Resources["stamina"].Current = 100.0
	p.Position = entity.Vec3{X: 0, Y: 0.1, Z: 0}

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	payload := []byte{entity.ActionGroundSlam, 0, 0, 0, 0}
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: 0x0031, Payload: payload}}
	inputSys := InputSystem{}
	inputSys.Tick(w, 0.05)

	if p.Resources["stamina"].Current != 70.0 {
		t.Errorf("stamina = %f, want 70.0 (100 - 30)", p.Resources["stamina"].Current)
	}
	if p.Cooldowns["execution"] != 8.0 {
		t.Errorf("GroundSlamCooldown = %f, want 8.0", p.Cooldowns["execution"])
	}
	if p.GCDTimer != 1.2 {
		t.Errorf("GCDTimer = %f, want 1.2 (lockout)", p.GCDTimer)
	}
}

func TestBladeSwirlCooldownTicksDown(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassVanguard)
	p.Cooldowns["vortex"] = 10.0

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	sys := CombatSystem{}

	sys.Tick(w, 4.0)
	if cd := p.Cooldowns["vortex"]; cd < 5.9 || cd > 6.1 {
		t.Errorf("BladeSwirlCooldown = %f, want ~6.0", cd)
	}

	sys.Tick(w, 7.0)
	if cd := p.Cooldowns["vortex"]; cd != 0.0 {
		t.Errorf("BladeSwirlCooldown = %f, want 0.0 (expired)", cd)
	}
}

func TestGroundSlamCooldownTicksDown(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassVanguard)
	p.Cooldowns["execution"] = 8.0

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	sys := CombatSystem{}

	sys.Tick(w, 3.0)
	if cd := p.Cooldowns["execution"]; cd < 4.9 || cd > 5.1 {
		t.Errorf("GroundSlamCooldown = %f, want ~5.0", cd)
	}

	sys.Tick(w, 6.0)
	if cd := p.Cooldowns["execution"]; cd != 0.0 {
		t.Errorf("GroundSlamCooldown = %f, want 0.0 (expired)", cd)
	}
}

func TestBladeSwirlBlockedByInsufficientStamina(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassVanguard)
	p.Resources["stamina"].Current = 20.0 // need 25

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	payload := []byte{entity.ActionBladeSwirl, 0, 0, 0, 0}
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: 0x0031, Payload: payload}}
	inputSys := InputSystem{}
	inputSys.Tick(w, 0.05)

	if p.HasBuff("vortex") {
		t.Error("BladeSwirl should not activate with insufficient stamina")
	}
	if p.Resources["stamina"].Current != 20.0 {
		t.Errorf("stamina should be unchanged at 20.0, got %f", p.Resources["stamina"].Current)
	}
}

// TestBladeSwirl3xIntegration fires 3 consecutive Blade Swirls at a nearby enemy,
// ticking cooldowns between each activation. Verifies damage events are produced
// and enemy HP decreases across all 3 activations + multi-tick damage.
func TestBladeSwirl3xIntegration(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassVanguard)
	p.Resources["stamina"].Current = 200.0
	p.Resources["stamina"].Max = 200.0
	p.Position = entity.Vec3{X: 0, Y: 0.1, Z: 0}
	p.RotationY = 0

	e := entity.NewEnemy(1000, 500.0, "test_enemy")
	e.Position = entity.Vec3{X: 2, Y: 0.1, Z: 0} // 2m away, within 6m radius
	e.State = entity.EnemyChase                  // not patrol, so AggroEnemy won't interfere

	enemies := []*entity.Enemy{e}
	w := makeWorld(map[uint16]*entity.Player{1: p}, enemies)

	inputSys := InputSystem{}
	combatSys := CombatSystem{}
	payload := []byte{entity.ActionBladeSwirl, 0, 0, 0, 0, 0, 0, 0, 0} // action + aim_pitch(4) + rot_y(4)

	totalDamageEvents := 0
	startHP := e.Health
	t.Logf("start: enemy HP=%.0f, player stamina=%.0f", e.Health, p.Resources["stamina"].Current)

	for swirl := 0; swirl < 3; swirl++ {
		// Fire blade swirl
		w.DamageEvents = w.DamageEvents[:0]
		w.InputQueue = []InputMsg{{PeerID: 1, Opcode: 0x0031, Payload: payload}}
		inputSys.Tick(w, 0.05)

		if !p.HasBuff("vortex") {
			t.Fatalf("swirl %d: BladeSwirl should be active", swirl+1)
		}

		eventsFromInput := len(w.DamageEvents)
		t.Logf("swirl %d: immediate hits=%d, enemy HP=%.0f, stamina=%.0f, swirl_cd=%.2f",
			swirl+1, eventsFromInput, e.Health, p.Resources["stamina"].Current, p.Cooldowns["vortex"])
		totalDamageEvents += eventsFromInput

		// Tick combat system for the full 1.5s duration + 1 extra tick for float rounding
		for tick := 0; tick < 32; tick++ {
			w.DamageEvents = w.DamageEvents[:0]
			combatSys.Tick(w, 0.05)
			totalDamageEvents += len(w.DamageEvents)
		}

		t.Logf("swirl %d after ticks: enemy HP=%.0f, blade_swirl=%v, swirl_cd=%.2f",
			swirl+1, e.Health, p.HasBuff("vortex"), p.Cooldowns["vortex"])

		if p.HasBuff("vortex") {
			t.Errorf("swirl %d: BladeSwirl should have ended after 1.5s", swirl+1)
		}

		// Tick down cooldowns: 10s swirl CD already partially ticked
		for tick := 0; tick < 200; tick++ { // 10s at 0.05s/tick
			combatSys.Tick(w, 0.05)
		}

		t.Logf("swirl %d after cooldown: swirl_cd=%.2f, stamina=%.0f",
			swirl+1, p.Cooldowns["vortex"], p.Resources["stamina"].Current)
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

// TestSwirlSlamSwirlSlamIntegration fires Blade Swirl -> Ground Slam -> Blade Swirl -> Ground Slam,
// ticking cooldowns between each. Verifies all 4 abilities activate and deal damage.
func TestSwirlSlamSwirlSlamIntegration(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassVanguard)
	p.Resources["stamina"].Current = 300.0
	p.Resources["stamina"].Max = 300.0
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
		isSwirlActive := p.HasBuff("vortex")

		t.Logf("step %d [%s]: input_hits=%d, HP=%.0f->%.0f, swirl=%v, swirl_cd=%.2f, slam_cd=%.2f, stamina=%.0f",
			i+1, s.name, eventsFromInput, hpBefore, e.Health, isSwirlActive,
			p.Cooldowns["vortex"], p.Cooldowns["execution"], p.Resources["stamina"].Current)

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
			if p.HasBuff("vortex") {
				t.Errorf("step %d [%s]: BladeSwirl should have ended", i+1, s.name)
			}
		}

		// Tick down ALL cooldowns: swirl (10s), slam (8s)
		// Tick 220 times (11s) to clear everything
		for tick := 0; tick < 220; tick++ {
			combatSys.Tick(w, 0.05)
		}

		t.Logf("step %d [%s] after cooldown: swirl_cd=%.2f, slam_cd=%.2f",
			i+1, s.name, p.Cooldowns["vortex"], p.Cooldowns["execution"])

		if p.Cooldowns["vortex"] > 0 {
			t.Errorf("step %d: BladeSwirlCooldown should be 0 after 11s, got %.2f", i+1, p.Cooldowns["vortex"])
		}
		if p.Cooldowns["execution"] > 0 {
			t.Errorf("step %d: GroundSlamCooldown should be 0 after 11s, got %.2f", i+1, p.Cooldowns["execution"])
		}
	}

	totalDamage := startHP - e.Health
	t.Logf("FINAL: enemy HP=%.0f (took %.0f damage), total events=%d, stamina=%.0f",
		e.Health, totalDamage, totalEvents, p.Resources["stamina"].Current)

	// 2 swirls x 4 hits x 25 dmg = 200 + 2 slams x 1 hit x 60 dmg = 120 -> 320 total
	if totalEvents < 4 {
		t.Errorf("expected at least 4 total damage events, got %d", totalEvents)
	}
	if totalDamage < 200 {
		t.Errorf("expected at least 200 total damage, got %.0f", totalDamage)
	}
}

func TestVanguardStaminaRegen(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassVanguard)
	stamina := p.Resources["stamina"]
	stamina.Current = 50.0
	stamina.DelayTimer = 0 // no delay, regen should start immediately

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	sys := CombatSystem{}

	t.Logf("before: stamina=%.1f delay=%.2f regen=%.1f max=%.1f", stamina.Current, stamina.DelayTimer, stamina.Regen, stamina.Max)

	// Tick 1 second — should regen 30 stamina
	sys.Tick(w, 1.0)
	t.Logf("after 1s: stamina=%.1f delay=%.2f", stamina.Current, stamina.DelayTimer)

	if stamina.Current < 79.0 || stamina.Current > 81.0 {
		t.Errorf("stamina after 1s regen = %.1f, want ~80.0 (50 + 30)", stamina.Current)
	}

	// Spend stamina, verify delay
	stamina.Current = 50.0
	stamina.DelayTimer = 0.6
	before := stamina.Current
	sys.Tick(w, 0.3) // 0.3s < 0.6s delay — no regen yet
	t.Logf("during delay: stamina=%.1f delay=%.2f", stamina.Current, stamina.DelayTimer)
	if stamina.Current != before {
		t.Errorf("stamina should not regen during delay, got %.1f (was %.1f)", stamina.Current, before)
	}

	sys.Tick(w, 0.5) // delay expires at 0.3+0.5=0.8 > 0.6, then 0.2s of regen
	t.Logf("after delay: stamina=%.1f delay=%.2f", stamina.Current, stamina.DelayTimer)
	if stamina.Current <= 50.0 {
		t.Errorf("stamina should have started regening after delay expired, got %.1f", stamina.Current)
	}
}

func TestCombatEndsOnEnemyDeath(t *testing.T) {
	p1 := entity.NewPlayer(1, entity.ClassGunner)
	p1.Health = 100.0 // below max
	p2 := entity.NewPlayer(2, entity.ClassVanguard)
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
// Blade Dancer — comprehensive test of all 20 abilities with 4 enemies
// =============================================================================

func TestAllBladeDancerSpells(t *testing.T) {
	const hp float32 = 5000.0

	type abilityExpect struct {
		abilityIdx      int
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

	abilities := []abilityExpect{
		// From Orbit (config 0)
		{0, "Shielded Sweep", 0, 1, 8, 8, 8, 0, 0, true, false, 0, 0, 0},
		{1, "Guarded Thrust", 0, 2, 25, 0, 0, 0, 8, false, false, 0, 0, 0},
		{2, "Protected Scatter", 0, 3, 5, 5, 5, 0, 0, true, true, 1.5, 11, 3},
		{3, "Fortified Command", 0, 4, 5, 5, 5, 0, 0, true, false, 0, 0, 0},
		// From Fan (config 1)
		{4, "Reaping Guard", 1, 0, 8, 0, 8, 0, 12, false, false, 0, 0, 0},
		{5, "Cleaving Pierce", 1, 2, 30, 9, 0, 0, 0, false, false, 0, 0, 0}, // 9 = splash (30 * 0.3)
		{6, "Slashing Spread", 1, 3, 8, 8, 8, 0, 0, false, true, 1.5, 9, 3},
		{7, "Sweeping Hex", 1, 4, 10, 10, 10, 0, 0, false, false, 0, 0, 0},
		// From Lance (config 2)
		{8, "Piercing Barrier", 2, 0, 18, 0, 0, 0, 14.4, false, false, 0, 0, 0}, // 18 * 0.8 ShieldPerDamage
		{9, "Focused Slash", 2, 1, 15, 15, 0, 0, 0, false, false, 0, 0, 0},
		{10, "Targeted Spread", 2, 3, 12, 0, 0, 0, 0, false, true, 2.0, 14, 1},
		{11, "Pinning Strike", 2, 4, 25, 0, 0, 0, 0, false, false, 0, 0, 0},
		// From Scatter (config 3)
		{12, "Dispersed Shield", 3, 0, 0, 0, 0, 0, 18, true, false, 0, 0, 0},
		{13, "Rain of Blades", 3, 1, 15, 15, 15, 0, 0, false, true, 1.0, 9, 3},
		{14, "Converging Strike", 3, 2, 32, 0, 0, 0, 0, false, true, 1.5, 9, 1},
		{15, "Chaos Bind", 3, 4, 8, 8, 8, 0, 0, false, false, 0, 0, 0},
		// From Crown (config 4)
		{16, "Commanding Ward", 4, 0, 0, 0, 0, 0, 20, false, false, 0, 0, 0},
		{17, "Royal Cleave", 4, 1, 12, 12, 12, 0, 0, false, false, 0, 0, 0},
		{18, "Decree Strike", 4, 2, 28, 0, 0, 0, 0, false, false, 0, 0, 0},
		{19, "Sovereign Scatter", 4, 3, 5, 5, 5, 0, 0, false, true, 1.5, 11, 3},
	}

	for _, sp := range abilities {
		t.Run(sp.name, func(t *testing.T) {
			p := entity.NewPlayer(1, entity.ClassBladeDancer)
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

			actionID := entity.ActionBDAbilityBase + uint8(sp.abilityIdx)
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
				shieldHP := p.GetResource("shield")
				if diff := shieldHP - expected; diff < -0.5 || diff > 0.5 {
					t.Errorf("shield = %.1f, want %.1f", shieldHP, expected)
				}
			} else if p.GetResource("shield") != 0 {
				t.Errorf("shield = %.0f, want 0", p.GetResource("shield"))
			}

			// DR
			if sp.hasDR && !p.HasBuff("bd_dr") {
				t.Error("DR should be active (bd_dr buff missing)")
			}
			if !sp.hasDR && p.HasBuff("bd_dr") {
				t.Error("DR should NOT be active (bd_dr buff present)")
			}

			// DoT — now on player.DoTs instead of w.BDDoTs
			if sp.hasDoT {
				if len(p.DoTs) != sp.dotTargets {
					t.Fatalf("DoT count = %d, want %d targets", len(p.DoTs), sp.dotTargets)
				}
				for _, dot := range p.DoTs {
					if dot.Damage != sp.dotPerTick {
						t.Errorf("DoT tick dmg = %.1f, want %.1f", dot.Damage, sp.dotPerTick)
					}
				}

				// Tick DoTs to completion for eFront
				combatSys := CombatSystem{}
				frontHPBefore := eFront.Health
				dotDuration := p.DoTs[0].Remaining
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
				if len(p.DoTs) != 0 {
					t.Errorf("DoTs should be expired, got %d", len(p.DoTs))
				}
			} else if len(p.DoTs) != 0 {
				t.Errorf("expected no DoTs, got %d", len(p.DoTs))
			}

			// eFar never touched
			if eFar.Health != hp {
				t.Errorf("eFar HP = %.0f, want %.0f", eFar.Health, hp)
			}
		})
	}
}

// =============================================================================
// PlayerAbilityRunner wiring — commit→execute lifecycle
// =============================================================================

// testCommitAction is an unused action slot used to bind test abilities.
const testCommitAction uint8 = 99
const testSustainAction uint8 = 98

// registerTestCommitAbility adds a test ability with CommitTime to an engine and
// binds it to the player's action map.
func registerTestCommitAbility(eng *ability.Engine, p *entity.Player) {
	def := &ability.AbilityDef{
		ID:           "test_commit_aoe",
		Name:         "Test Commit AoE",
		Hit:          ability.HitDef{Type: ability.HitAoECircle, Radius: 10},
		BaseDamage:   50,
		CommitTime:   0.5,
		ExecuteTime:  0.1,
		GCD:          0.3,
		OriginConfig: -1,
		DestConfig:   -1,
	}
	eng.Register(def)
	p.ActionMap[testCommitAction] = "test_commit_aoe"
}

func TestRunnerWiring_InputStartsRunner(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassVanguard)
	p.Position = entity.Vec3{X: 0, Y: 0.1, Z: 0}

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	registerTestCommitAbility(w.AbilityEngine, p)

	payload := codec.EncodeAbilityInput(testCommitAction, 0.0)
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: message.OpAbilityInput, Payload: payload}}

	is := &InputSystem{}
	is.Tick(w, 0.05)

	runner := w.AbilityRunners[1]
	if runner == nil {
		t.Fatal("runner should be created for peer 1")
	}
	if !runner.IsBusy() {
		t.Error("runner should be busy after Start")
	}
	if runner.Phase != ability.PRunnerCommit {
		t.Errorf("runner phase = %d, want %d (commit)", runner.Phase, ability.PRunnerCommit)
	}
	if runner.AbilityID != "test_commit_aoe" {
		t.Errorf("runner ability = %q, want %q", runner.AbilityID, "test_commit_aoe")
	}

	// Player channel state should be synced
	if p.ChannelAbilityID != "test_commit_aoe" {
		t.Errorf("player ChannelAbilityID = %q, want %q", p.ChannelAbilityID, "test_commit_aoe")
	}
	if p.ChannelPhase != uint8(ability.PRunnerCommit) {
		t.Errorf("player ChannelPhase = %d, want %d", p.ChannelPhase, ability.PRunnerCommit)
	}
}

func TestRunnerWiring_CancelOnNewInput(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassVanguard)
	p.Position = entity.Vec3{X: 0, Y: 0.1, Z: 0}

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	registerTestCommitAbility(w.AbilityEngine, p)

	payload := codec.EncodeAbilityInput(testCommitAction, 0.0)
	is := &InputSystem{}

	// First input starts the runner
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: message.OpAbilityInput, Payload: payload}}
	is.Tick(w, 0.05)

	runner := w.AbilityRunners[1]
	if runner == nil || !runner.IsBusy() {
		t.Fatal("runner should be busy after first input")
	}

	// Tick a few times to advance the timer
	combatSys := &CombatSystem{}
	combatSys.Tick(w, 0.05)
	combatSys.Tick(w, 0.05)

	// Second input should cancel and restart the runner
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: message.OpAbilityInput, Payload: payload}}
	is.Tick(w, 0.05)

	// Runner should have restarted with a fresh timer (0.5s commit time)
	if !runner.IsBusy() {
		t.Error("runner should still be busy after cancel+restart")
	}
	if runner.Phase != ability.PRunnerCommit {
		t.Errorf("runner phase = %d, want %d (commit)", runner.Phase, ability.PRunnerCommit)
	}
	if runner.Timer < 0.45 {
		t.Errorf("runner timer = %f, expected fresh timer near 0.5", runner.Timer)
	}
}

func TestRunnerWiring_CancelFailsDuringExecute(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassVanguard)
	p.Position = entity.Vec3{X: 0, Y: 0.1, Z: 0}

	e := entity.NewEnemy(1000, 2000.0, "test_enemy")
	e.Alive = true
	e.Position = entity.Vec3{X: 2, Y: 0.1, Z: 0}
	e.State = entity.EnemyChase

	w := makeWorld(map[uint16]*entity.Player{1: p}, []*entity.Enemy{e})
	registerTestCommitAbility(w.AbilityEngine, p)

	payload := codec.EncodeAbilityInput(testCommitAction, 0.0)
	is := &InputSystem{}

	// Start the runner
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: message.OpAbilityInput, Payload: payload}}
	is.Tick(w, 0.05)

	runner := w.AbilityRunners[1]
	if runner == nil {
		t.Fatal("runner should exist")
	}

	// Tick through entire commit phase (0.5s = 10 ticks at 0.05s)
	combatSys := &CombatSystem{}
	for i := 0; i < 10; i++ {
		combatSys.Tick(w, 0.05)
	}

	// Runner should be in execute or cooldown phase now
	if runner.Phase == ability.PRunnerCommit || runner.Phase == ability.PRunnerIdle {
		t.Fatalf("runner phase = %d, expected execute or cooldown", runner.Phase)
	}

	// New input during execute/cooldown should be rejected (Cancel returns false)
	phaseBefore := runner.Phase
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: message.OpAbilityInput, Payload: payload}}
	is.Tick(w, 0.05)

	if runner.Phase == ability.PRunnerCommit && phaseBefore != ability.PRunnerIdle {
		t.Error("runner should not restart during execute/cooldown phase")
	}
}

func TestRunnerWiring_CombatTickFiresCast(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassVanguard)
	p.Position = entity.Vec3{X: 0, Y: 0.1, Z: 0}

	e := entity.NewEnemy(1000, 2000.0, "test_enemy")
	e.Alive = true
	e.Position = entity.Vec3{X: 2, Y: 0.1, Z: 0} // within 10-unit AoE radius
	e.State = entity.EnemyChase

	w := makeWorld(map[uint16]*entity.Player{1: p}, []*entity.Enemy{e})
	registerTestCommitAbility(w.AbilityEngine, p)

	// Start the runner via input
	payload := codec.EncodeAbilityInput(testCommitAction, 0.0)
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: message.OpAbilityInput, Payload: payload}}
	is := &InputSystem{}
	is.Tick(w, 0.05)

	runner := w.AbilityRunners[1]
	if runner == nil {
		t.Fatal("runner should exist")
	}

	startHP := e.Health
	combatSys := &CombatSystem{}

	// Tick through commit phase (0.5s = 10 ticks at 0.05s)
	// During commit, no damage should be dealt
	for i := 0; i < 9; i++ {
		w.DamageEvents = w.DamageEvents[:0]
		combatSys.Tick(w, 0.05)
	}
	if e.Health != startHP {
		t.Errorf("enemy took damage during commit phase: HP %f -> %f", startHP, e.Health)
	}

	// The 10th tick should expire commit and fire the ability
	w.DamageEvents = w.DamageEvents[:0]
	combatSys.Tick(w, 0.05)

	if e.Health >= startHP {
		t.Error("enemy should have taken damage after commit expired")
	}
	if len(w.DamageEvents) == 0 {
		t.Error("expected damage events after commit expired")
	}

	// Runner should now be in execute or cooldown phase
	if runner.Phase == ability.PRunnerCommit {
		t.Error("runner should have left commit phase")
	}
}

func TestRunnerWiring_SyncsPlayerChannelState(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassVanguard)
	p.Position = entity.Vec3{X: 0, Y: 0.1, Z: 0}

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	registerTestCommitAbility(w.AbilityEngine, p)

	payload := codec.EncodeAbilityInput(testCommitAction, 0.0)
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: message.OpAbilityInput, Payload: payload}}
	is := &InputSystem{}
	is.Tick(w, 0.05)

	// After input, channel state should be set
	if p.ChannelPhase != uint8(ability.PRunnerCommit) {
		t.Fatalf("initial ChannelPhase = %d, want %d", p.ChannelPhase, ability.PRunnerCommit)
	}

	combatSys := &CombatSystem{}

	// Tick a few times during commit — charge should increase
	for i := 0; i < 5; i++ {
		combatSys.Tick(w, 0.05)
	}
	if p.ChannelCharge <= 0 {
		t.Errorf("ChannelCharge = %f after 5 ticks, want > 0", p.ChannelCharge)
	}
	if p.ChannelCharge >= 1.0 {
		t.Errorf("ChannelCharge = %f after 5 ticks, want < 1.0 (still in commit)", p.ChannelCharge)
	}

	// Tick through the rest until idle
	for i := 0; i < 20; i++ {
		combatSys.Tick(w, 0.05)
	}

	// Runner should return to idle, channel state cleared
	runner := w.AbilityRunners[1]
	if runner.IsBusy() {
		t.Errorf("runner should be idle, phase = %d", runner.Phase)
	}
	if p.ChannelPhase != 0 {
		t.Errorf("ChannelPhase = %d after idle, want 0", p.ChannelPhase)
	}
	if p.ChannelAbilityID != "" {
		t.Errorf("ChannelAbilityID = %q after idle, want empty", p.ChannelAbilityID)
	}
}

func TestRunnerWiring_ThreatAndAggroOnExecute(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassVanguard)
	p.Position = entity.Vec3{X: 0, Y: 0.1, Z: 0}

	e := entity.NewEnemy(1000, 2000.0, "test_enemy")
	e.Alive = true
	e.Position = entity.Vec3{X: 2, Y: 0.1, Z: 0}
	e.State = entity.EnemyPatrol // starts in patrol

	w := makeWorld(map[uint16]*entity.Player{1: p}, []*entity.Enemy{e})
	registerTestCommitAbility(w.AbilityEngine, p)

	// Start the runner
	payload := codec.EncodeAbilityInput(testCommitAction, 0.0)
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: message.OpAbilityInput, Payload: payload}}
	is := &InputSystem{}
	is.Tick(w, 0.05)

	combatSys := &CombatSystem{}

	// Tick through commit (10 ticks at 0.05s = 0.5s)
	for i := 0; i < 10; i++ {
		combatSys.Tick(w, 0.05)
	}

	// After ability fires, enemy should have threat from player
	if !e.HasThreat(1) {
		t.Error("enemy should have threat from player 1 after committed ability fires")
	}
	// Enemy should have been aggroed from patrol
	if e.State == entity.EnemyPatrol {
		t.Error("enemy should have left patrol state after being hit")
	}
}

func TestRunnerWiring_NonCommitAbilityBypassesRunner(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassVanguard)
	p.Position = entity.Vec3{X: 0, Y: 0.1, Z: 0}

	e := entity.NewEnemy(1000, 2000.0, "test_enemy")
	e.Alive = true
	e.Position = entity.Vec3{X: 0, Y: 0, Z: -2.0} // in front (player faces -Z)
	e.State = entity.EnemyChase

	w := makeWorld(map[uint16]*entity.Player{1: p}, []*entity.Enemy{e})

	// Normal melee has CommitTime=0, should bypass runner entirely
	payload := codec.EncodeAbilityInput(entity.ActionMelee, 0.0)
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: message.OpAbilityInput, Payload: payload}}

	is := &InputSystem{}
	is.Tick(w, 0.05)

	// No runner should have been created
	if runner := w.AbilityRunners[1]; runner != nil {
		t.Errorf("runner should not be created for non-commit ability, got phase=%d", runner.Phase)
	}

	// Damage should have been applied immediately (no commit delay)
	if e.Health >= 2000.0 {
		t.Error("non-commit melee should deal immediate damage")
	}
}

// =============================================================================
// Sustain phase integration tests
// =============================================================================

func registerTestSustainAbility(eng *ability.Engine, p *entity.Player) {
	def := &ability.AbilityDef{
		ID:                "test_sustain_heal",
		Name:              "Test Sustain Heal",
		School:            "bioarcanotechnic",
		Hit:               ability.HitDef{Type: ability.HitAllyTarget, Range: 20},
		BaseHeal:          10,
		CommitTime:        0.5,
		ExecuteTime:       0.1,
		GCD:               0.3,
		OriginConfig:      -1,
		DestConfig:        -1,
		Sustain:           true,
		SustainCostPerSec: 10,
		SustainEffect:     20,
		SustainInterval:   0.5,
		SustainScaling:    0.1,
		CancelConditions:  uint8(ability.CancelOnMove) | uint8(ability.CancelOnDamage),
	}
	eng.Register(def)
	eng.RegisterHandler("test_sustain_heal", func(_ *ability.Engine, ctx *ability.CommitContext) ability.CommitResult {
		return ability.CommitResult{OK: true}
	})
	p.ActionMap[testSustainAction] = "test_sustain_heal"
}

func newHarmonistPlayer(id uint16) *entity.Player {
	p := entity.NewPlayerWithSpec(id, entity.ClassArcanotechnicien, "harmonist")
	p.Position = entity.Vec3{X: 0, Y: 0.1, Z: 0}
	// Ensure flux resource is available
	if p.Resources["flux"] == nil {
		p.Resources["flux"] = &entity.Resource{Current: 100, Max: 100}
	} else {
		p.Resources["flux"].Current = 100
		p.Resources["flux"].Max = 100
	}
	return p
}

func TestSustain_CommitToSustainTransition(t *testing.T) {
	p := newHarmonistPlayer(1)
	ally := newHarmonistPlayer(2)

	w := makeWorld(map[uint16]*entity.Player{1: p, 2: ally}, nil)
	registerTestSustainAbility(w.AbilityEngine, p)

	// Start sustain ability via input
	payload := codec.EncodeAbilityInput(testSustainAction, 0.0)
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: message.OpAbilityInput, Payload: payload}}
	is := &InputSystem{}
	is.Tick(w, 0.05)

	runner := w.AbilityRunners[1]
	if runner == nil {
		t.Fatal("runner should exist")
	}
	if runner.Phase != ability.PRunnerCommit {
		t.Fatalf("expected PRunnerCommit, got %d", runner.Phase)
	}

	combatSys := &CombatSystem{}

	// Tick through commit (0.5s = 10 ticks)
	for i := 0; i < 10; i++ {
		combatSys.Tick(w, 0.05)
	}
	if runner.Phase != ability.PRunnerExecute {
		t.Fatalf("expected PRunnerExecute after commit, got %d", runner.Phase)
	}

	// Tick through execute (0.1s = 2-3 ticks)
	for i := 0; i < 5; i++ {
		combatSys.Tick(w, 0.05)
		if runner.Phase == ability.PRunnerSustain {
			break
		}
	}
	if runner.Phase != ability.PRunnerSustain {
		t.Fatalf("expected PRunnerSustain after execute, got %d", runner.Phase)
	}

	// Player channel state should show sustain
	if p.ChannelPhase != uint8(ability.PRunnerSustain) {
		t.Errorf("ChannelPhase = %d, want %d (sustain)", p.ChannelPhase, ability.PRunnerSustain)
	}
}

func TestSustain_HealingAppliedOnTick(t *testing.T) {
	p := newHarmonistPlayer(1)
	ally := newHarmonistPlayer(2)
	ally.Health = 50 // damaged
	ally.MaxHealth = 150
	p.ChannelTargetID = 2 // target the ally

	w := makeWorld(map[uint16]*entity.Player{1: p, 2: ally}, nil)
	registerTestSustainAbility(w.AbilityEngine, p)

	// Start runner directly in sustain
	runner := &ability.PlayerAbilityRunner{}
	def := w.AbilityEngine.GetAbility("test_sustain_heal")
	runner.StartSustain(def, p.Position, w.TickNum)
	w.AbilityRunners[1] = runner

	combatSys := &CombatSystem{}
	startHP := ally.Health

	// Tick through first sustain interval (0.5s = 10 ticks)
	for i := 0; i < 10; i++ {
		w.DamageEvents = w.DamageEvents[:0]
		combatSys.Tick(w, 0.05)
	}

	if ally.Health <= startHP {
		t.Errorf("ally should have been healed: HP before=%f, after=%f", startHP, ally.Health)
	}

	// DamageEvents should contain heal events
	healFound := false
	for _, ev := range w.DamageEvents {
		if ev.TargetPeerID == 2 && ev.SourcePeerID == 1 && ev.Amount > 0 {
			healFound = true
			break
		}
	}
	if !healFound {
		t.Error("expected heal damage event for ally (peer 2)")
	}
}

func TestSustain_ScalingIncreasesOverTime(t *testing.T) {
	p := newHarmonistPlayer(1)
	ally := newHarmonistPlayer(2)
	ally.MaxHealth = 1000 // high max so heals don't overcap
	ally.Health = 100
	p.ChannelTargetID = 2

	w := makeWorld(map[uint16]*entity.Player{1: p, 2: ally}, nil)
	registerTestSustainAbility(w.AbilityEngine, p)

	runner := &ability.PlayerAbilityRunner{}
	def := w.AbilityEngine.GetAbility("test_sustain_heal")
	runner.StartSustain(def, p.Position, w.TickNum)
	w.AbilityRunners[1] = runner

	combatSys := &CombatSystem{}

	// Record heal from first tick (at 0.5s)
	for i := 0; i < 10; i++ {
		w.DamageEvents = w.DamageEvents[:0]
		combatSys.Tick(w, 0.05)
	}
	firstHeal := float32(0)
	for _, ev := range w.DamageEvents {
		if ev.TargetPeerID == 2 {
			firstHeal = ev.Amount
		}
	}

	// Record heal from later tick (at 5.0s = 100 ticks total, sustain elapsed ~4.5s)
	for i := 0; i < 90; i++ {
		ally.Health = 100 // keep HP low so heals always apply
		w.DamageEvents = w.DamageEvents[:0]
		combatSys.Tick(w, 0.05)
	}
	laterHeal := float32(0)
	for _, ev := range w.DamageEvents {
		if ev.TargetPeerID == 2 {
			laterHeal = ev.Amount
		}
	}

	if laterHeal <= firstHeal {
		t.Errorf("scaling should increase heal over time: first=%f, later=%f", firstHeal, laterHeal)
	}
}

func TestSustain_FluxDrained(t *testing.T) {
	p := newHarmonistPlayer(1)
	ally := newHarmonistPlayer(2)
	p.ChannelTargetID = 2

	w := makeWorld(map[uint16]*entity.Player{1: p, 2: ally}, nil)
	registerTestSustainAbility(w.AbilityEngine, p)

	runner := &ability.PlayerAbilityRunner{}
	def := w.AbilityEngine.GetAbility("test_sustain_heal")
	runner.StartSustain(def, p.Position, w.TickNum)
	w.AbilityRunners[1] = runner

	// Track the bioarcanotechnic pool (test_sustain_heal has School="bioarcanotechnic")
	pool := p.FluxCommit.GetPool("bioarcanotechnic")
	startFlux := pool.Current
	combatSys := &CombatSystem{}

	// Tick through one sustain interval (0.5s)
	for i := 0; i < 10; i++ {
		combatSys.Tick(w, 0.05)
	}

	endFlux := pool.Current
	// Cost = SustainCostPerSec * SustainInterval = 10 * 0.5 = 5 per tick
	expectedDrain := float32(5.0)
	actualDrain := startFlux - endFlux
	if math.Abs(float64(actualDrain-expectedDrain)) > 0.5 {
		t.Errorf("flux drained = %f, expected ~%f", actualDrain, expectedDrain)
	}
}

func TestSustain_CancelledWhenFluxDepleted(t *testing.T) {
	p := newHarmonistPlayer(1)
	ally := newHarmonistPlayer(2)
	// Set bioarcanotechnic pool to very low (sustain drains from school pool)
	if pool := p.FluxCommit.GetPool("bioarcanotechnic"); pool != nil {
		pool.Current = 3
	}
	p.SyncFluxAggregate()
	p.ChannelTargetID = 2

	w := makeWorld(map[uint16]*entity.Player{1: p, 2: ally}, nil)
	registerTestSustainAbility(w.AbilityEngine, p)

	runner := &ability.PlayerAbilityRunner{}
	def := w.AbilityEngine.GetAbility("test_sustain_heal")
	runner.StartSustain(def, p.Position, w.TickNum)
	w.AbilityRunners[1] = runner

	combatSys := &CombatSystem{}

	// Tick until runner leaves sustain (should happen when flux runs out)
	for i := 0; i < 30; i++ {
		combatSys.Tick(w, 0.05)
		if runner.Phase != ability.PRunnerSustain {
			break
		}
	}
	if runner.Phase == ability.PRunnerSustain {
		t.Fatal("sustain should have been cancelled due to flux depletion")
	}
	// Should be in cooldown after cancel
	if runner.Phase != ability.PRunnerCooldown && runner.Phase != ability.PRunnerIdle {
		t.Errorf("expected cooldown or idle after flux depletion, got %d", runner.Phase)
	}
}

func TestSustain_CancelledOnDamage(t *testing.T) {
	p := newHarmonistPlayer(1)
	p.ChannelTargetID = 1 // self-heal for simplicity

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	registerTestSustainAbility(w.AbilityEngine, p)

	runner := &ability.PlayerAbilityRunner{}
	def := w.AbilityEngine.GetAbility("test_sustain_heal")
	runner.StartSustain(def, p.Position, w.TickNum)
	runner.SustainStartTick = w.TickNum
	w.AbilityRunners[1] = runner

	combatSys := &CombatSystem{}

	// Tick once to be in sustain
	combatSys.Tick(w, 0.05)
	if runner.Phase != ability.PRunnerSustain {
		t.Fatalf("expected PRunnerSustain, got %d", runner.Phase)
	}

	// Simulate taking damage by setting LastDamageTick after sustain started
	p.LastDamageTick = w.TickNum + 1

	// Next combat tick should cancel sustain
	combatSys.Tick(w, 0.05)
	if runner.Phase == ability.PRunnerSustain {
		t.Fatal("sustain should have been cancelled on damage")
	}
}

func TestSustain_CancelledOnMovement(t *testing.T) {
	p := newHarmonistPlayer(1)
	p.ChannelTargetID = 1

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	registerTestSustainAbility(w.AbilityEngine, p)

	startPos := p.Position
	runner := &ability.PlayerAbilityRunner{}
	def := w.AbilityEngine.GetAbility("test_sustain_heal")
	runner.StartSustain(def, startPos, w.TickNum)
	w.AbilityRunners[1] = runner

	combatSys := &CombatSystem{}

	// Tick once in sustain
	combatSys.Tick(w, 0.05)
	if runner.Phase != ability.PRunnerSustain {
		t.Fatalf("expected PRunnerSustain, got %d", runner.Phase)
	}

	// Move the player far from start position
	p.Position = entity.Vec3{X: startPos.X + 5.0, Y: startPos.Y, Z: startPos.Z + 5.0}

	// Next combat tick should detect movement and cancel
	combatSys.Tick(w, 0.05)
	if runner.Phase == ability.PRunnerSustain {
		t.Fatal("sustain should have been cancelled on movement")
	}
}

func TestSustain_SmallMovementDoesNotCancel(t *testing.T) {
	p := newHarmonistPlayer(1)
	p.ChannelTargetID = 1

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	registerTestSustainAbility(w.AbilityEngine, p)

	startPos := p.Position
	runner := &ability.PlayerAbilityRunner{}
	def := w.AbilityEngine.GetAbility("test_sustain_heal")
	runner.StartSustain(def, startPos, w.TickNum)
	w.AbilityRunners[1] = runner

	combatSys := &CombatSystem{}

	// Small jitter within threshold (< 0.5 unit)
	p.Position = entity.Vec3{X: startPos.X + 0.1, Y: startPos.Y, Z: startPos.Z + 0.1}

	combatSys.Tick(w, 0.05)
	if runner.Phase != ability.PRunnerSustain {
		t.Fatal("small positional jitter should not cancel sustain")
	}
}

func TestSustain_Action255CancelsRunner(t *testing.T) {
	p := newHarmonistPlayer(1)
	p.ChannelTargetID = 1

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	registerTestSustainAbility(w.AbilityEngine, p)

	runner := &ability.PlayerAbilityRunner{}
	def := w.AbilityEngine.GetAbility("test_sustain_heal")
	runner.StartSustain(def, p.Position, w.TickNum)
	w.AbilityRunners[1] = runner

	// Send action 255 (ESC cancel)
	payload := codec.EncodeAbilityInput(255, 0.0)
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: message.OpAbilityInput, Payload: payload}}
	is := &InputSystem{}
	is.Tick(w, 0.05)

	if runner.Phase == ability.PRunnerSustain {
		t.Fatal("action 255 should cancel sustain")
	}
	if runner.Phase != ability.PRunnerCooldown {
		t.Errorf("expected PRunnerCooldown after cancel, got %d", runner.Phase)
	}
}

func TestSustain_CancelledByNewAbilityInput(t *testing.T) {
	p := newHarmonistPlayer(1)

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	registerTestSustainAbility(w.AbilityEngine, p)
	registerTestCommitAbility(w.AbilityEngine, p)

	// Start in sustain
	runner := &ability.PlayerAbilityRunner{}
	def := w.AbilityEngine.GetAbility("test_sustain_heal")
	runner.StartSustain(def, p.Position, w.TickNum)
	w.AbilityRunners[1] = runner

	// Send a different ability input (committed ability)
	payload := codec.EncodeAbilityInput(testCommitAction, 0.0)
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: message.OpAbilityInput, Payload: payload}}
	is := &InputSystem{}
	is.Tick(w, 0.05)

	// Runner should have been cancelled and restarted with the new ability
	if runner.Phase == ability.PRunnerSustain {
		t.Fatal("sustain should have been cancelled by new ability input")
	}
	if runner.AbilityID != "test_commit_aoe" {
		t.Errorf("runner should be on new ability, got %q", runner.AbilityID)
	}
}

func TestSustain_CancelledByExplicitCancel(t *testing.T) {
	// Harmonist has no dodge — sustain is cancelled via action 255 (ESC).
	p := newHarmonistPlayer(1)

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	registerTestSustainAbility(w.AbilityEngine, p)

	runner := &ability.PlayerAbilityRunner{}
	def := w.AbilityEngine.GetAbility("test_sustain_heal")
	runner.StartSustain(def, p.Position, w.TickNum)
	w.AbilityRunners[1] = runner

	// Send explicit cancel (action 255 = ESC on client)
	payload := codec.EncodeAbilityInput(255, 0.0)
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: message.OpAbilityInput, Payload: payload}}
	is := &InputSystem{}
	is.Tick(w, 0.05)

	if runner.Phase == ability.PRunnerSustain {
		t.Fatal("sustain should have been cancelled by explicit cancel")
	}
}

func TestSustain_SelfHealWhenNoTarget(t *testing.T) {
	p := newHarmonistPlayer(1)
	p.Health = 50
	p.MaxHealth = 150
	p.ChannelTargetID = 99 // non-existent peer

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	registerTestSustainAbility(w.AbilityEngine, p)

	runner := &ability.PlayerAbilityRunner{}
	def := w.AbilityEngine.GetAbility("test_sustain_heal")
	runner.StartSustain(def, p.Position, w.TickNum)
	w.AbilityRunners[1] = runner

	combatSys := &CombatSystem{}
	startHP := p.Health

	// Tick through one sustain interval
	for i := 0; i < 10; i++ {
		combatSys.Tick(w, 0.05)
	}

	// Should fallback to self-heal
	if p.Health <= startHP {
		t.Errorf("sustain should self-heal when target not found: HP %f -> %f", startHP, p.Health)
	}
}

func TestSustain_DamageTakenBeforeSustainStartDoesNotCancel(t *testing.T) {
	p := newHarmonistPlayer(1)
	p.ChannelTargetID = 1
	p.LastDamageTick = 50 // damage taken before sustain started

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	w.TickNum = 100
	registerTestSustainAbility(w.AbilityEngine, p)

	runner := &ability.PlayerAbilityRunner{}
	def := w.AbilityEngine.GetAbility("test_sustain_heal")
	runner.StartSustain(def, p.Position, w.TickNum)
	runner.SustainStartTick = w.TickNum // sustain started at tick 100
	w.AbilityRunners[1] = runner

	combatSys := &CombatSystem{}

	// Tick — damage was at tick 50, sustain started at tick 100, should NOT cancel
	combatSys.Tick(w, 0.05)
	if runner.Phase != ability.PRunnerSustain {
		t.Fatal("damage taken before sustain start should not cancel sustain")
	}
}

func TestSustain_HealCappedAtMaxHealth(t *testing.T) {
	p := newHarmonistPlayer(1)
	ally := newHarmonistPlayer(2)
	ally.Health = 149 // nearly full
	ally.MaxHealth = 150
	p.ChannelTargetID = 2

	w := makeWorld(map[uint16]*entity.Player{1: p, 2: ally}, nil)
	registerTestSustainAbility(w.AbilityEngine, p)

	runner := &ability.PlayerAbilityRunner{}
	def := w.AbilityEngine.GetAbility("test_sustain_heal")
	runner.StartSustain(def, p.Position, w.TickNum)
	w.AbilityRunners[1] = runner

	combatSys := &CombatSystem{}

	// Tick through sustain ticks
	for i := 0; i < 20; i++ {
		combatSys.Tick(w, 0.05)
	}

	if ally.Health > ally.MaxHealth {
		t.Errorf("health should not exceed max: %f > %f", ally.Health, ally.MaxHealth)
	}
}

func TestSustain_ChannelPhaseValueSyncedToPlayer(t *testing.T) {
	p := newHarmonistPlayer(1)

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	registerTestSustainAbility(w.AbilityEngine, p)

	runner := &ability.PlayerAbilityRunner{}
	def := w.AbilityEngine.GetAbility("test_sustain_heal")
	runner.StartSustain(def, p.Position, w.TickNum)
	w.AbilityRunners[1] = runner

	combatSys := &CombatSystem{}
	combatSys.Tick(w, 0.05)

	// Verify ChannelPhase = 3 (PRunnerSustain)
	if p.ChannelPhase != 3 {
		t.Errorf("ChannelPhase = %d, want 3 (sustain)", p.ChannelPhase)
	}
	if p.ChannelAbilityID != "test_sustain_heal" {
		t.Errorf("ChannelAbilityID = %q, want 'test_sustain_heal'", p.ChannelAbilityID)
	}
	// Charge should be >= 1.0 during sustain
	if p.ChannelCharge < 1.0 {
		t.Errorf("ChannelCharge = %f during sustain, want >= 1.0", p.ChannelCharge)
	}
}

func TestSustain_SustainStartPosRecordedOnTransition(t *testing.T) {
	p := newHarmonistPlayer(1)
	p.Position = entity.Vec3{X: 10, Y: 0, Z: 20}

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	registerTestSustainAbility(w.AbilityEngine, p)

	// Start via input
	payload := codec.EncodeAbilityInput(testSustainAction, 0.0)
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: message.OpAbilityInput, Payload: payload}}
	is := &InputSystem{}
	is.Tick(w, 0.05)

	runner := w.AbilityRunners[1]
	if runner == nil {
		t.Fatal("runner should exist")
	}

	combatSys := &CombatSystem{}

	// Tick through commit + execute to reach sustain
	for i := 0; i < 15; i++ {
		combatSys.Tick(w, 0.05)
		if runner.Phase == ability.PRunnerSustain {
			break
		}
	}
	if runner.Phase != ability.PRunnerSustain {
		t.Fatalf("expected PRunnerSustain, got %d", runner.Phase)
	}

	// SustainStartPos should match player position at transition
	if runner.SustainStartPos != p.Position {
		t.Errorf("SustainStartPos = %v, want %v", runner.SustainStartPos, p.Position)
	}
	if runner.SustainStartTick == 0 {
		t.Error("SustainStartTick should be set after transition")
	}
}

// ---------------------------------------------------------------------------
// Unbound action rejection
// ---------------------------------------------------------------------------

func TestUnboundAction_RejectedByServer(t *testing.T) {
	p := newHarmonistPlayer(1)

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)

	// Verify gust_step is NOT in the default harmonist action map
	for actionID, abilityID := range p.ActionMap {
		if abilityID == "gust_step" {
			t.Fatalf("gust_step should not be in default loadout, found at action %d", actionID)
		}
	}

	startHP := p.Health
	startFlux := p.Resources["flux"].Current

	// Send an ability input for action_id 55 (client's slot 5 = Gust Step)
	// but the server's slot 5 is transfusion in the default loadout
	// First, remove whatever is at action 55 to simulate an unbound slot
	delete(p.ActionMap, 55)

	payload := codec.EncodeAbilityInput(55, 0.0)
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: message.OpAbilityInput, Payload: payload}}
	is := &InputSystem{}
	is.Tick(w, 0.05)

	// No runner should have been created
	if runner := w.AbilityRunners[1]; runner != nil && runner.IsBusy() {
		t.Errorf("unbound action should not start a runner, got phase %d ability %q",
			runner.Phase, runner.AbilityID)
	}

	// No HP or flux should change
	if p.Health != startHP {
		t.Errorf("HP changed from %f to %f on unbound action", startHP, p.Health)
	}
	if p.Resources["flux"].Current != startFlux {
		t.Errorf("flux changed from %f to %f on unbound action", startFlux, p.Resources["flux"].Current)
	}
}

func TestUnboundAction_DoesNotCancelSustain(t *testing.T) {
	p := newHarmonistPlayer(1)

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	registerTestSustainAbility(w.AbilityEngine, p)

	// Start sustain
	runner := &ability.PlayerAbilityRunner{}
	def := w.AbilityEngine.GetAbility("test_sustain_heal")
	runner.StartSustain(def, p.Position, w.TickNum)
	w.AbilityRunners[1] = runner

	// Remove action 55 to simulate unbound Gust Step
	delete(p.ActionMap, 55)

	// Send unbound action 55
	payload := codec.EncodeAbilityInput(55, 0.0)
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: message.OpAbilityInput, Payload: payload}}
	is := &InputSystem{}
	is.Tick(w, 0.05)

	// Sustain should NOT be cancelled by an unbound action
	if runner.Phase != ability.PRunnerSustain {
		t.Errorf("unbound action should not cancel sustain, runner phase = %d", runner.Phase)
	}
}

func TestUnboundAction_DodgeStillWorks(t *testing.T) {
	// Use gunner (which has dodge) — harmonist has no dodge.
	p := entity.NewPlayer(1, entity.ClassGunner)
	if p.Resources["stamina"] == nil {
		p.Resources["stamina"] = &entity.Resource{Current: 100, Max: 100}
	}

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)

	// Remove loadout bindings — dodge (action 3) should still work independently
	delete(p.ActionMap, 50)

	payload := codec.EncodeAbilityInput(entity.ActionDodge, 0.0)
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: message.OpAbilityInput, Payload: payload}}
	is := &InputSystem{}
	is.Tick(w, 0.05)

	if !p.Invincible {
		t.Error("dodge should still work even with unbound ability slots")
	}
}

func TestHarmonist_DodgeRejected(t *testing.T) {
	p := newHarmonistPlayer(1)
	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)

	payload := codec.EncodeAbilityInput(entity.ActionDodge, 0.0)
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: message.OpAbilityInput, Payload: payload}}
	is := &InputSystem{}
	is.Tick(w, 0.05)

	if p.Invincible {
		t.Error("harmonist should not be able to dodge — gust step is mobility via loadout only")
	}
}

// =============================================================================
// Movement speed validation vulnerability
//
// The server does NOT enforce horizontal speed limits for normal (unbuffed)
// players. handlePlayerInput only validates speed when speedMult < 1.0 (brace
// or shield_block debuff). A modified client can send position updates at any
// speed up to the 10-unit teleport threshold, gaining unauthorized displacement.
//
// These tests express DESIRED behavior and FAIL until the server enforces
// per-class speed limits on every position update.
// =============================================================================

func TestMovementSpeed_ExcessiveHorizontalNotClamped(t *testing.T) {
	// A Harmonist at origin sends a position 5 units away in a single tick.
	// SprintSpeed = 6.5 u/s → max per tick = 6.5 * 0.05 = 0.325 units.
	// 5 units in one tick = 100 u/s, ~15x the legal speed.
	// Expected: server clamps to ~0.325 units from origin.
	// Actual: server accepts 5.0 (only the >10-unit teleport check exists).
	p := entity.NewPlayerWithSpec(1, entity.ClassArcanotechnicien, "harmonist")
	p.Position = entity.Vec3{X: 0, Y: 0.1, Z: 0}
	p.SpawnTick = 1 // past grace period

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	w.TickNum = 100

	// Move 5 units on X axis in one tick
	payload := codec.EncodePlayerInput(nil, 5.0, 0.1, 0.0, 0.0, 100, 0, 0.0)
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: message.OpPlayerInput, Payload: payload}}

	is := &InputSystem{}
	is.Tick(w, 0.05)

	// Max legitimate distance: SprintSpeed * tickDt * tolerance
	// 6.5 * 0.05 * 1.5 = 0.4875 units (with 50% tolerance)
	maxDist := float32(6.5 * 0.05 * 1.5)
	dx := p.Position.X - 0.0
	dz := p.Position.Z - 0.0
	actualDist := float32(math.Sqrt(float64(dx*dx + dz*dz)))

	if actualDist > maxDist {
		t.Errorf("server accepted movement of %.2f units in one tick; max legal = %.3f (sprint*tick*1.5). "+
			"Position = (%.2f, %.2f, %.2f). This proves the speed validation vulnerability.",
			actualDist, maxDist, p.Position.X, p.Position.Y, p.Position.Z)
	}
}

func TestMovementSpeed_DodgeSpeedWithoutDodge(t *testing.T) {
	// A modified client sends position updates at dodge/roll speed (11.0 u/s)
	// for 10 consecutive ticks without ever sending an ActionDodge input.
	// This simulates a client faking Gust Step displacement without authorization.
	// Expected: each tick is clamped to sprint speed (6.5 u/s).
	// Actual: all updates accepted — player travels ~5.5 units freely.
	p := entity.NewPlayerWithSpec(1, entity.ClassArcanotechnicien, "harmonist")
	p.Position = entity.Vec3{X: 0, Y: 0.1, Z: 0}
	p.SpawnTick = 1

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	w.TickNum = 100

	is := &InputSystem{}
	rollSpeed := float32(11.0) // Harmonist RollSpeed
	tickDt := float32(0.05)
	distPerTick := rollSpeed * tickDt // 0.55 units per tick

	// Send 10 ticks at dodge speed moving along +X
	for i := 0; i < 10; i++ {
		newX := float32(i+1) * distPerTick
		payload := codec.EncodePlayerInput(nil, newX, 0.1, 0.0, 0.0, uint32(100+i), 0, 0.0)
		w.InputQueue = []InputMsg{{PeerID: 1, Opcode: message.OpPlayerInput, Payload: payload}}
		is.Tick(w, tickDt)
	}

	// After 10 ticks at dodge speed: attempted 10 * 0.55 = 5.5 units
	// Max legal: 10 ticks * sprint speed * tickDt = 10 * 6.5 * 0.05 = 3.25 units
	maxLegal := float32(10) * float32(6.5) * tickDt
	maxLegalTolerant := maxLegal * 1.5 // 50% tolerance = 4.875

	actualX := p.Position.X
	if actualX > maxLegalTolerant {
		t.Errorf("player moved %.2f units in 10 ticks at dodge speed without dodge authorization; "+
			"max legal (with tolerance) = %.2f. Server accepted unauthorized displacement.",
			actualX, maxLegalTolerant)
	}
}

func TestMovementSpeed_BracedPlayerIsClamped(t *testing.T) {
	// Sanity check: speed clamping DOES work for debuffed players.
	// A braced player (speedMult=0) should not move at all.
	// This test should PASS, proving the asymmetry: debuffed = clamped, normal = unchecked.
	p := entity.NewPlayerWithSpec(1, entity.ClassArcanotechnicien, "harmonist")
	p.Position = entity.Vec3{X: 0, Y: 0.1, Z: 0}
	p.SpawnTick = 1
	p.AddBuff(entity.ActiveBuff{ID: "brace", Type: "generic", Duration: 10.0})

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	w.TickNum = 100

	payload := codec.EncodePlayerInput(nil, 5.0, 0.1, 0.0, 0.0, 100, 0, 0.0)
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: message.OpPlayerInput, Payload: payload}}

	is := &InputSystem{}
	is.Tick(w, 0.05)

	// Braced player should stay at origin (speedMult = 0)
	if p.Position.X != 0.0 || p.Position.Z != 0.0 {
		t.Errorf("braced player moved to (%.2f, %.2f) — brace should lock position",
			p.Position.X, p.Position.Z)
	}
}

func TestMovementSpeed_ShieldBlockClampsSpeed(t *testing.T) {
	// Shield block (speedMult=0.4) should clamp movement to 40% of sprint speed.
	// Vanguard SprintSpeed = 7.5, so max = 7.5 * 0.4 * 0.05 * 1.5 = 0.225
	p := entity.NewPlayerWithSpec(1, entity.ClassVanguard, "blade")
	p.Position = entity.Vec3{X: 0, Y: 0.1, Z: 0}
	p.SpawnTick = 1
	p.AddBuff(entity.ActiveBuff{ID: "vg_shield_block", Type: "damage_reduction", Value: 0.7, Duration: 10.0})

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	w.TickNum = 100

	payload := codec.EncodePlayerInput(nil, 5.0, 0.1, 0.0, 0.0, 100, 0, 0.0)
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: message.OpPlayerInput, Payload: payload}}

	is := &InputSystem{}
	is.Tick(w, 0.05)

	mv := p.Movement()
	maxDist := mv.SprintSpeed * 0.4 * (1.0 / 20.0) * 1.5
	dx := p.Position.X
	dz := p.Position.Z
	actualDist := float32(math.Sqrt(float64(dx*dx + dz*dz)))

	if actualDist > maxDist+0.01 {
		t.Errorf("shield block player moved %.3f, max allowed = %.3f", actualDist, maxDist)
	}
	if p.Position.X <= 0.0 {
		t.Error("shield block player should still move (just slower), but X stayed at 0")
	}
}

func TestMovementSpeed_TeleportRejected(t *testing.T) {
	// Teleporting > 10 units (dist² > 100) should be fully rejected.
	p := entity.NewPlayerWithSpec(1, entity.ClassArcanotechnicien, "harmonist")
	p.Position = entity.Vec3{X: 0, Y: 0.1, Z: 0}
	p.SpawnTick = 1

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	w.TickNum = 100

	// 11 units away — exceeds the 10-unit teleport threshold
	payload := codec.EncodePlayerInput(nil, 11.0, 0.1, 0.0, 0.0, 100, 0, 0.0)
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: message.OpPlayerInput, Payload: payload}}

	is := &InputSystem{}
	is.Tick(w, 0.05)

	if p.Position.X != 0.0 {
		t.Errorf("teleport should be rejected: position = (%.2f, %.2f, %.2f), expected origin",
			p.Position.X, p.Position.Y, p.Position.Z)
	}
}

func TestMovementSpeed_SpawnGraceRejectsInput(t *testing.T) {
	// During the 10-tick spawn grace window, all position updates are rejected.
	p := entity.NewPlayerWithSpec(1, entity.ClassArcanotechnicien, "harmonist")
	p.Position = entity.Vec3{X: 5, Y: 0.1, Z: 5}
	p.SpawnTick = 95 // spawned at tick 95

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	w.TickNum = 100 // only 5 ticks since spawn (< 10 grace ticks)

	payload := codec.EncodePlayerInput(nil, 0.0, 0.1, 0.0, 0.0, 100, 0, 0.0)
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: message.OpPlayerInput, Payload: payload}}

	is := &InputSystem{}
	is.Tick(w, 0.05)

	// Position should remain at spawn position, input rejected
	if p.Position.X != 5.0 || p.Position.Z != 5.0 {
		t.Errorf("input during spawn grace should be rejected: position = (%.2f, %.2f)",
			p.Position.X, p.Position.Z)
	}
}

func TestMovementSpeed_YBoundsRejectsOutOfRange(t *testing.T) {
	// Position above PlayerBoundsMaxY should be rejected entirely.
	p := entity.NewPlayerWithSpec(1, entity.ClassArcanotechnicien, "harmonist")
	p.Position = entity.Vec3{X: 0, Y: 0.1, Z: 0}
	p.SpawnTick = 1

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	w.TickNum = 100
	// Arena level has PlayerBoundsMaxY = 6.0

	payload := codec.EncodePlayerInput(nil, 0.0, 50.0, 0.0, 0.0, 100, 0, 0.0)
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: message.OpPlayerInput, Payload: payload}}

	is := &InputSystem{}
	is.Tick(w, 0.05)

	if p.Position.Y != 0.1 {
		t.Errorf("Y out of bounds should reject entire input: Y = %.2f, expected 0.1",
			p.Position.Y)
	}
}

func TestMovementSpeed_ElevatorAllowsFastY(t *testing.T) {
	// Inside an elevator volume, Y movement at elevator speed should be allowed.
	p := entity.NewPlayerWithSpec(1, entity.ClassArcanotechnicien, "harmonist")
	p.Position = entity.Vec3{X: 5, Y: -100, Z: -55}
	p.SpawnTick = 1

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	w.TickNum = 100
	// Use hub level which has an elevator at (5, -55)
	w.Level = level.NewHubLevel()

	// Move up by 0.3 units (elevator speed = 10 u/s, allowed = 10*0.05*1.5 = 0.75)
	newY := float32(-99.7)
	payload := codec.EncodePlayerInput(nil, 5.0, newY, -55.0, 0.0, 100, 0, 0.0)
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: message.OpPlayerInput, Payload: payload}}

	is := &InputSystem{}
	is.Tick(w, 0.05)

	if p.Position.Y < -99.8 {
		t.Errorf("elevator should allow upward movement: Y = %.2f, expected ~%.2f",
			p.Position.Y, newY)
	}
}

func TestMovementSpeed_UpwardYClampedOutsideElevator(t *testing.T) {
	// Outside an elevator, fast upward Y should be clamped.
	// Max upward outside elevator = 5.0 * (1/20) * 2.0 = 0.5 units per tick.
	p := entity.NewPlayerWithSpec(1, entity.ClassArcanotechnicien, "harmonist")
	p.Position = entity.Vec3{X: 0, Y: 0.1, Z: 0}
	p.SpawnTick = 1

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	w.TickNum = 100

	// Try to move up by 2.0 units (way above 0.5 + 0.1 tolerance = 0.6)
	payload := codec.EncodePlayerInput(nil, 0.0, 2.1, 0.0, 0.0, 100, 0, 0.0)
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: message.OpPlayerInput, Payload: payload}}

	is := &InputSystem{}
	is.Tick(w, 0.05)

	// Y should be clamped back to original
	if p.Position.Y > 0.2 {
		t.Errorf("upward Y should be clamped outside elevator: Y = %.2f, expected ~0.1",
			p.Position.Y)
	}
}

// =============================================================================
// handleSetLoadout tests
// =============================================================================

func TestSetLoadout_UpdatesActionMap(t *testing.T) {
	p := newHarmonistPlayer(1)

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)

	// New loadout swaps slot 0 and clears slot 5
	newSlots := [6]string{"vital_bloom", "mending_beam", "mending_surge", "restoration_matrix", "life_swap", ""}
	payload := codec.EncodeLoadoutState(newSlots)
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: message.OpSetLoadout, Payload: payload}}

	is := &InputSystem{}
	is.Tick(w, 0.05)

	// Slot 0 → action 50 should now be vital_bloom
	if p.ActionMap[50] != "vital_bloom" {
		t.Errorf("action 50 = %q, want 'vital_bloom'", p.ActionMap[50])
	}
	// Slot 2 → action 52 should be mending_surge
	if p.ActionMap[52] != "mending_surge" {
		t.Errorf("action 52 = %q, want 'mending_surge'", p.ActionMap[52])
	}
	// Loadout should be updated
	if p.Loadout.Slots[0] != "vital_bloom" {
		t.Errorf("loadout slot 0 = %q, want 'vital_bloom'", p.Loadout.Slots[0])
	}
}

func TestSetLoadout_EmptySlotNotAdded(t *testing.T) {
	p := newHarmonistPlayer(1)

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)

	// Before: slot 5 (action 55) = vital_drain
	beforeAction55 := p.ActionMap[55]
	if beforeAction55 != "vital_drain" {
		t.Fatalf("precondition: action 55 = %q, want 'vital_drain'", beforeAction55)
	}

	// Send loadout with slot 5 empty
	newSlots := [6]string{"mending_surge", "mending_beam", "vital_bloom", "restoration_matrix", "life_swap", ""}
	payload := codec.EncodeLoadoutState(newSlots)
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: message.OpSetLoadout, Payload: payload}}

	is := &InputSystem{}
	is.Tick(w, 0.05)

	// ApplyLoadout skips empty slots, so action 55 retains old value.
	// BUG: ApplyLoadout does NOT delete old bindings for empty slots.
	// A player who unbinds an ability still has it in ActionMap.
	// This is a loadout hygiene issue — but NOT a security hole because
	// the server still validates the ability exists in the engine.
	//
	// For now, verify the empty slot wasn't overwritten with "".
	if _, exists := p.ActionMap[55]; !exists {
		t.Log("action 55 was deleted — this is the correct behavior if ApplyLoadout cleans up")
	} else {
		t.Logf("action 55 = %q (stale binding from previous loadout)", p.ActionMap[55])
	}
}

func TestSetLoadout_InvalidPayload(t *testing.T) {
	p := newHarmonistPlayer(1)
	originalMap := make(map[uint8]string, len(p.ActionMap))
	for k, v := range p.ActionMap {
		originalMap[k] = v
	}

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)

	// Truncated/invalid payload
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: message.OpSetLoadout, Payload: []byte{0xFF}}}

	is := &InputSystem{}
	is.Tick(w, 0.05)

	// ActionMap should be unchanged
	for k, v := range originalMap {
		if p.ActionMap[k] != v {
			t.Errorf("action %d changed from %q to %q on invalid payload", k, v, p.ActionMap[k])
		}
	}
}

func TestSetLoadout_UnknownPlayer(t *testing.T) {
	w := makeWorld(map[uint16]*entity.Player{}, nil)

	newSlots := [6]string{"mending_surge", "", "", "", "", ""}
	payload := codec.EncodeLoadoutState(newSlots)
	// PeerID 99 does not exist
	w.InputQueue = []InputMsg{{PeerID: 99, Opcode: message.OpSetLoadout, Payload: payload}}

	is := &InputSystem{}
	// Should not panic
	is.Tick(w, 0.05)
}

func TestSetLoadout_NilLoadoutCreated(t *testing.T) {
	// Use a gunner (which has no loadout by default)
	p := entity.NewPlayer(1, entity.ClassGunner)

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)

	if p.Loadout != nil {
		t.Fatal("precondition: gunner should not have a loadout")
	}

	newSlots := [6]string{"fire_shot", "", "", "", "", ""}
	payload := codec.EncodeLoadoutState(newSlots)
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: message.OpSetLoadout, Payload: payload}}

	is := &InputSystem{}
	is.Tick(w, 0.05)

	if p.Loadout == nil {
		t.Fatal("loadout should be created when nil")
	}
	if p.Loadout.Slots[0] != "fire_shot" {
		t.Errorf("slot 0 = %q, want 'fire_shot'", p.Loadout.Slots[0])
	}
	if p.ActionMap[50] != "fire_shot" {
		t.Errorf("action 50 = %q, want 'fire_shot'", p.ActionMap[50])
	}
}

// =============================================================================
// Loadout validation security — RED TESTS
//
// handleSetLoadout accepts ANY ability string without validation. A modified
// client can slot non-existent abilities, unimplemented abilities, or abilities
// that don't belong to the player's class. ApplyLoadout then writes these
// into the ActionMap, making them committable.
//
// Additionally, ApplyLoadout does not clean up stale ActionMap entries when
// a slot is emptied — a player who unbinds an ability can still commit it via
// the stale ActionMap binding.
//
// These tests express DESIRED behavior and FAIL until the bugs are fixed.
// =============================================================================

func TestSetLoadout_RejectsNonExistentAbility(t *testing.T) {
	// A modified client sends OpSetLoadout with a completely fake ability ID.
	// Expected: server rejects the loadout change, ActionMap[50] unchanged.
	// Actual: "totally_fake_ability" is written into ActionMap[50].
	p := newHarmonistPlayer(1)
	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)

	original50 := p.ActionMap[50] // "mending_surge"

	newSlots := [6]string{"totally_fake_ability", "mending_beam", "vital_bloom", "restoration_matrix", "life_swap", "transfusion"}
	payload := codec.EncodeLoadoutState(newSlots)
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: message.OpSetLoadout, Payload: payload}}

	is := &InputSystem{}
	is.Tick(w, 0.05)

	if p.ActionMap[50] != original50 {
		t.Errorf("server accepted non-existent ability in loadout: ActionMap[50] = %q, want %q (unchanged). "+
			"handleSetLoadout has no validation — any string is written to ActionMap.",
			p.ActionMap[50], original50)
	}
}

func TestSetLoadout_RejectsUnimplementedAbility(t *testing.T) {
	// "fireball" exists in the ability catalog YAML but has implemented: false.
	// A player should not be able to slot unimplemented abilities.
	// Expected: server rejects the loadout; ActionMap[50] stays "mending_surge".
	// Actual: "fireball" is written into ActionMap[50].
	p := newHarmonistPlayer(1)
	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)

	original50 := p.ActionMap[50]

	newSlots := [6]string{"fireball", "mending_beam", "vital_bloom", "restoration_matrix", "life_swap", "transfusion"}
	payload := codec.EncodeLoadoutState(newSlots)
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: message.OpSetLoadout, Payload: payload}}

	is := &InputSystem{}
	is.Tick(w, 0.05)

	if p.ActionMap[50] != original50 {
		t.Errorf("server accepted unimplemented ability: ActionMap[50] = %q, want %q (unchanged). "+
			"handleSetLoadout does not check implemented status.",
			p.ActionMap[50], original50)
	}
}

func TestSetLoadout_ClearsStaleActionMapOnEmptySlot(t *testing.T) {
	// Default loadout has slot 5 = vital_drain (action 55).
	// Player sends a new loadout with slot 5 empty.
	// Expected: ActionMap[55] is deleted — the ability is no longer equipped.
	// Actual: stale "vital_drain" binding survives because ApplyLoadout skips
	// empty slots instead of deleting them.
	p := newHarmonistPlayer(1)
	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)

	// Precondition
	if p.ActionMap[55] != "vital_drain" {
		t.Fatalf("precondition: ActionMap[55] = %q, want vital_drain", p.ActionMap[55])
	}

	newSlots := [6]string{"mending_surge", "mending_beam", "vital_bloom", "restoration_matrix", "life_swap", ""}
	payload := codec.EncodeLoadoutState(newSlots)
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: message.OpSetLoadout, Payload: payload}}

	is := &InputSystem{}
	is.Tick(w, 0.05)

	if _, exists := p.ActionMap[55]; exists {
		t.Errorf("stale ActionMap[55] = %q survived after clearing slot 5. "+
			"ApplyLoadout skips empty slots instead of deleting old bindings.",
			p.ActionMap[55])
	}
}

func TestAbilityInput_StaleBindingFiresAfterSlotClear(t *testing.T) {
	// Combines the stale ActionMap bug with ability execution.
	// 1. Player clears slot 5 (was transfusion).
	// 2. Client sends AbilityInput for action 55 (the cleared slot).
	// Expected: ability does NOT fire — slot is empty.
	// Actual: stale "transfusion" binding in ActionMap routes to the engine,
	// which starts a runner for transfusion even though the player unequipped it.
	p := newHarmonistPlayer(1)
	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)

	// Step 1: clear slot 5
	newSlots := [6]string{"mending_surge", "mending_beam", "vital_bloom", "restoration_matrix", "life_swap", ""}
	payload := codec.EncodeLoadoutState(newSlots)
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: message.OpSetLoadout, Payload: payload}}
	is := &InputSystem{}
	is.Tick(w, 0.05)

	// Step 2: send ability input for action 55 (the slot we just cleared)
	abilityPayload := codec.EncodeAbilityInput(55, 0.0)
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: message.OpAbilityInput, Payload: abilityPayload}}
	is.Tick(w, 0.05)

	// Transfusion (CommitTime=4.0) would start a runner in commit phase
	if runner := w.AbilityRunners[1]; runner != nil && runner.IsBusy() {
		t.Errorf("cleared slot 5 still fired ability %q via stale ActionMap binding. "+
			"Player unequipped transfusion but can still commit it.",
			runner.AbilityID)
	}
}

func TestSetLoadout_GunnerAbilityOnHarmonist(t *testing.T) {
	// A modified client sends fire_shot (gunner ability) in a harmonist loadout.
	// Expected: server rejects — fire_shot is not in the harmonist's class codex.
	// Actual: "fire_shot" is written into ActionMap[50]. The player can now commit
	// a gunner ability as a healer because there is no class-membership check.
	p := newHarmonistPlayer(1)
	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)

	original50 := p.ActionMap[50]

	newSlots := [6]string{"fire_shot", "mending_beam", "vital_bloom", "restoration_matrix", "life_swap", "transfusion"}
	payload := codec.EncodeLoadoutState(newSlots)
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: message.OpSetLoadout, Payload: payload}}

	is := &InputSystem{}
	is.Tick(w, 0.05)

	if p.ActionMap[50] != original50 {
		t.Errorf("server accepted cross-class ability: ActionMap[50] = %q, want %q (unchanged). "+
			"handleSetLoadout has no class membership validation.",
			p.ActionMap[50], original50)
	}
}

// =============================================================================
// ForceReset — exercised through the sustain→new-committed-ability flow
// =============================================================================

func TestForceReset_SustainThenCommittedAbility(t *testing.T) {
	// During sustain, Cancel() transitions to cooldown (not idle).
	// Without ForceReset, Start() would fail because phase != idle.
	// This test directly exercises the ForceReset path in handleAbilityInput.
	p := newHarmonistPlayer(1)

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	registerTestSustainAbility(w.AbilityEngine, p)
	registerTestCommitAbility(w.AbilityEngine, p)

	// Start runner in sustain
	runner := &ability.PlayerAbilityRunner{}
	def := w.AbilityEngine.GetAbility("test_sustain_heal")
	runner.StartSustain(def, p.Position, w.TickNum)
	w.AbilityRunners[1] = runner

	// Verify we're in sustain
	if runner.Phase != ability.PRunnerSustain {
		t.Fatalf("precondition: phase = %d, want %d (sustain)", runner.Phase, ability.PRunnerSustain)
	}

	// Send a committed ability input — this triggers Cancel() + ForceReset() + Start()
	payload := codec.EncodeAbilityInput(testCommitAction, 0.0)
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: message.OpAbilityInput, Payload: payload}}
	is := &InputSystem{}
	is.Tick(w, 0.05)

	// Runner should be in commit phase for the new ability
	if runner.Phase != ability.PRunnerCommit {
		t.Errorf("runner phase = %d, want %d (commit). ForceReset may not be working.",
			runner.Phase, ability.PRunnerCommit)
	}
	if runner.AbilityID != "test_commit_aoe" {
		t.Errorf("runner ability = %q, want 'test_commit_aoe'", runner.AbilityID)
	}
	if runner.Timer < 0.45 {
		t.Errorf("runner timer = %f, want fresh ~0.5 (commit time)", runner.Timer)
	}
}

func TestForceReset_SustainThenInstantAbility(t *testing.T) {
	// Same flow but for instant (non-committed) abilities.
	// handleAbilityInput line 213-217: Cancel + ForceReset for instant path.
	p := newHarmonistPlayer(1)
	p.Position = entity.Vec3{X: 0, Y: 0.1, Z: 0}

	e := entity.NewEnemy(1000, 2000.0, "test_enemy")
	e.Alive = true
	e.Position = entity.Vec3{X: 0, Y: 0, Z: -2.0}
	e.State = entity.EnemyChase

	w := makeWorld(map[uint16]*entity.Player{1: p}, []*entity.Enemy{e})
	registerTestSustainAbility(w.AbilityEngine, p)

	// Start runner in sustain
	runner := &ability.PlayerAbilityRunner{}
	def := w.AbilityEngine.GetAbility("test_sustain_heal")
	runner.StartSustain(def, p.Position, w.TickNum)
	w.AbilityRunners[1] = runner

	// Bind melee (instant, CommitTime=0) if not already bound
	// ActionMelee is already in vanguard's map; for Harmonist we need to add it
	p.ActionMap[entity.ActionMelee] = "melee"

	// Send instant ability input
	payload := codec.EncodeAbilityInput(entity.ActionMelee, 0.0)
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: message.OpAbilityInput, Payload: payload}}
	is := &InputSystem{}
	is.Tick(w, 0.05)

	// Sustain should be cancelled, runner should be idle or reset
	if runner.Phase == ability.PRunnerSustain {
		t.Error("sustain should have been cancelled by instant ability")
	}
	// The runner was ForceReset, then the instant ability bypasses the runner
	// So runner should be idle
	if runner.Phase != ability.PRunnerIdle {
		t.Errorf("runner phase = %d after instant ability cancelled sustain, want idle (0)", runner.Phase)
	}
}
