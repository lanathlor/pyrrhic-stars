package ability

import (
	"math"
	"testing"
)

func TestTempo_CooldownDrainsFaster(t *testing.T) {
	eng := NewEngine(nil)

	// Gunner with Tempo 100 => 2x speed
	pFast := newGunner()
	pFast.GearStats.Tempo = 100
	eFast := enemyInFront(100, 500)

	r := eng.Commit("fire_shot", commitCtx(pFast, eFast))
	if !r.OK {
		t.Fatalf("fire_shot failed: %s", r.Reason)
	}
	cdBefore := pFast.Cooldowns["fire_shot"]
	if math.Abs(float64(cdBefore-0.18)) > 0.001 {
		t.Fatalf("initial cooldown = %f, want 0.18", cdBefore)
	}

	eng.TickPlayer(pFast, 0.05, tickCtx())
	cdFast := pFast.Cooldowns["fire_shot"]
	// With Tempo 100 (2x), 0.05s tick drains 0.10s of cooldown
	// Remaining: 0.18 - 0.10 = 0.08
	wantFast := float32(0.18 - 0.05*2.0)
	if math.Abs(float64(cdFast-wantFast)) > 0.001 {
		t.Errorf("fast cooldown = %f, want ~%f", cdFast, wantFast)
	}

	// Gunner with Tempo 0 => 1x speed (baseline)
	pSlow := newGunner()
	eSlow := enemyInFront(101, 500)

	eng.Commit("fire_shot", commitCtx(pSlow, eSlow))
	eng.TickPlayer(pSlow, 0.05, tickCtx())
	cdSlow := pSlow.Cooldowns["fire_shot"]
	// Without Tempo, 0.05s tick drains 0.05s of cooldown
	// Remaining: 0.18 - 0.05 = 0.13
	wantSlow := float32(0.18 - 0.05)
	if math.Abs(float64(cdSlow-wantSlow)) > 0.001 {
		t.Errorf("slow cooldown = %f, want ~%f", cdSlow, wantSlow)
	}

	// The fast player's cooldown should have drained more
	if cdFast >= cdSlow {
		t.Errorf("fast cooldown (%f) should be less than slow (%f)", cdFast, cdSlow)
	}
}

func TestTempo_GCDDrainsFaster(t *testing.T) {
	eng := NewEngine(nil)

	// Vanguard with Tempo 100 => 2x speed
	// vortex (blade_swirl) sets GCD to 0.6 (standard tier)
	pFast := newVanguard()
	pFast.GearStats.Tempo = 100
	eFast := enemyInFront(100, 1000)
	eFast.Position.Z = -3

	r := eng.Commit("vortex", commitCtx(pFast, eFast))
	if !r.OK {
		t.Fatalf("blade_swirl failed: %s", r.Reason)
	}
	if pFast.GCDTimer != 0.6 {
		t.Fatalf("GCD after commit = %f, want 0.6", pFast.GCDTimer)
	}

	eng.TickPlayer(pFast, 0.2, tickCtx(eFast))
	// With Tempo 100 (2x), 0.2s tick drains 0.4s of GCD
	// Remaining: 0.6 - 0.4 = 0.2
	if math.Abs(float64(pFast.GCDTimer-0.2)) > 0.02 {
		t.Errorf("fast GCD = %f, want ~0.2", pFast.GCDTimer)
	}

	// Vanguard with Tempo 0 => 1x speed (baseline)
	pSlow := newVanguard()
	eSlow := enemyInFront(101, 1000)
	eSlow.Position.Z = -3

	eng.Commit("vortex", commitCtx(pSlow, eSlow))
	eng.TickPlayer(pSlow, 0.2, tickCtx(eSlow))
	// Without Tempo, 0.2s tick drains 0.2s of GCD
	// Remaining: 0.6 - 0.2 = 0.4
	if math.Abs(float64(pSlow.GCDTimer-0.4)) > 0.02 {
		t.Errorf("slow GCD = %f, want ~0.4", pSlow.GCDTimer)
	}

	if pFast.GCDTimer >= pSlow.GCDTimer {
		t.Errorf("fast GCD (%f) should be less than slow (%f)", pFast.GCDTimer, pSlow.GCDTimer)
	}
}

func TestTempo_ParryWindowExtended(t *testing.T) {
	eng := NewEngine(nil)

	// Vanguard with Tempo 100 => 2x multiplier
	pFast := newVanguard()
	pFast.GearStats.Tempo = 100

	r := eng.Commit("vg_block", commitCtx(pFast))
	if !r.OK {
		t.Fatalf("vg_block failed: %s", r.Reason)
	}
	b := pFast.GetBuff("vg_parry")
	if b == nil {
		t.Fatal("parry buff not found")
	}
	// With Tempo 100 (2x), parry window = 0.15 * 2.0 = 0.30
	wantDuration := float32(blockParryTime * 2.0)
	if math.Abs(float64(b.Duration-wantDuration)) > 0.001 {
		t.Errorf("fast parry duration = %f, want %f", b.Duration, wantDuration)
	}

	// Vanguard with Tempo 0 => 1x multiplier (baseline)
	pSlow := newVanguard()

	eng.Commit("vg_block", commitCtx(pSlow))
	bSlow := pSlow.GetBuff("vg_parry")
	if bSlow == nil {
		t.Fatal("parry buff not found (slow)")
	}
	if math.Abs(float64(bSlow.Duration-blockParryTime)) > 0.001 {
		t.Errorf("slow parry duration = %f, want %f", bSlow.Duration, blockParryTime)
	}
}

func TestTempo_RechamberFaster(t *testing.T) {
	eng := NewEngine(nil)

	// Gunner with Tempo 100 => 2x speed
	pFast := newGunner()
	pFast.GearStats.Tempo = 100

	r := eng.Commit("rechamber", commitCtx(pFast))
	if !r.OK {
		t.Fatalf("rechamber failed: %s", r.Reason)
	}
	state, ok := pFast.AbilityState["rechamber"].(*RechamberState)
	if !ok {
		t.Fatal("rechamber state not found")
	}
	if state.Timer != 0.6 {
		t.Fatalf("initial timer = %f, want 0.6", state.Timer)
	}

	eng.TickPlayer(pFast, 0.15, tickCtx())
	// With Tempo 100 (2x), 0.15s tick drains 0.30s from timer
	// Remaining: 0.6 - 0.30 = 0.30
	if math.Abs(float64(state.Timer-0.30)) > 0.01 {
		t.Errorf("fast timer = %f, want ~0.30", state.Timer)
	}

	// Gunner with Tempo 0 => 1x speed (baseline)
	pSlow := newGunner()

	eng.Commit("rechamber", commitCtx(pSlow))
	stateSlow, ok := pSlow.AbilityState["rechamber"].(*RechamberState)
	if !ok {
		t.Fatal("rechamber state not found")
	}

	eng.TickPlayer(pSlow, 0.15, tickCtx())
	// Without Tempo, 0.15s tick drains 0.15s
	// Remaining: 0.6 - 0.15 = 0.45
	if math.Abs(float64(stateSlow.Timer-0.45)) > 0.01 {
		t.Errorf("slow timer = %f, want ~0.45", stateSlow.Timer)
	}

	if state.Timer >= stateSlow.Timer {
		t.Errorf("fast timer (%f) should be less than slow (%f)", state.Timer, stateSlow.Timer)
	}
}

func TestTempo_StaminaRegenFaster(t *testing.T) {
	eng := NewEngine(nil)

	// Vanguard with Tempo 100 => 2x regen speed
	pFast := newVanguard()
	pFast.GearStats.Tempo = 100
	pFast.Resources["stamina"].Current = 0

	eng.TickPlayer(pFast, 1.0, tickCtx())
	// Base regen 30/s, with Tempo 100 (2x) => 60/s over 1.0s = 60
	stamFast := pFast.GetResource("stamina")
	if math.Abs(float64(stamFast-60)) > 1.0 {
		t.Errorf("fast stamina = %f, want ~60", stamFast)
	}

	// Vanguard with Tempo 0 => 1x regen speed (baseline)
	pSlow := newVanguard()
	pSlow.Resources["stamina"].Current = 0

	eng.TickPlayer(pSlow, 1.0, tickCtx())
	// Base regen 30/s, no Tempo => 30/s over 1.0s = 30
	stamSlow := pSlow.GetResource("stamina")
	if math.Abs(float64(stamSlow-30)) > 1.0 {
		t.Errorf("slow stamina = %f, want ~30", stamSlow)
	}

	if stamFast <= stamSlow {
		t.Errorf("fast stamina (%f) should be greater than slow (%f)", stamFast, stamSlow)
	}
}

func TestTempo_OverclockPlusTempo(t *testing.T) {
	eng := NewEngine(nil)

	// Gunner with Tempo 50 => 1.5x speed
	pFast := newGunner()
	pFast.GearStats.Tempo = 50
	eFast := enemyInFront(100, 500)

	// Overclock applies BuffCooldownMult 0.556
	eng.Commit("overclock", commitCtx(pFast))
	// Clear overclock's own cooldown so it doesn't interfere
	delete(pFast.Cooldowns, "overclock")

	eng.Commit("fire_shot", commitCtx(pFast, eFast))
	cdSet := pFast.Cooldowns["fire_shot"]
	// Cooldown set: 0.18 * 0.556 ~ 0.10
	wantCD := float32(0.18 * 0.556)
	if math.Abs(float64(cdSet-wantCD)) > 0.01 {
		t.Fatalf("initial cooldown with overclock = %f, want ~%f", cdSet, wantCD)
	}

	eng.TickPlayer(pFast, 0.05, tickCtx())
	// With Tempo 50 (1.5x), 0.05s tick drains 0.075s
	// Remaining: ~0.10 - 0.075 = ~0.025
	cdRemaining := pFast.Cooldowns["fire_shot"]
	wantRemaining := wantCD - 0.05*1.5
	if math.Abs(float64(cdRemaining-wantRemaining)) > 0.01 {
		t.Errorf("fast cooldown remaining = %f, want ~%f", cdRemaining, wantRemaining)
	}

	// Gunner with Tempo 0 => 1x speed (baseline)
	pSlow := newGunner()
	eSlow := enemyInFront(101, 500)

	eng.Commit("overclock", commitCtx(pSlow))
	delete(pSlow.Cooldowns, "overclock")

	eng.Commit("fire_shot", commitCtx(pSlow, eSlow))
	eng.TickPlayer(pSlow, 0.05, tickCtx())
	// Without Tempo, 0.05s tick drains 0.05s
	// Remaining: ~0.10 - 0.05 = ~0.05
	cdSlowRemaining := pSlow.Cooldowns["fire_shot"]
	wantSlowRemaining := wantCD - 0.05
	if math.Abs(float64(cdSlowRemaining-wantSlowRemaining)) > 0.01 {
		t.Errorf("slow cooldown remaining = %f, want ~%f", cdSlowRemaining, wantSlowRemaining)
	}

	if cdRemaining >= cdSlowRemaining {
		t.Errorf("fast remaining (%f) should be less than slow (%f)", cdRemaining, cdSlowRemaining)
	}
}
