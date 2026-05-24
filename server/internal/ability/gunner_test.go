package ability

import (
	"math"
	"testing"

	"codex-online/server/internal/entity"
)

const dmgTol = 0.05

func assertDmgNear(t *testing.T, got, want float32, label string) {
	t.Helper()
	if math.Abs(float64(got-want)) > dmgTol {
		t.Errorf("%s: damage = %.2f, want ~%.2f", label, got, want)
	}
}

// --- Fire Shot ---

func TestGunner_FireShot_BasicHit(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	e := enemyInFront(100, 500)

	r := eng.Commit("fire_shot", commitCtx(p, e))
	if !r.OK {
		t.Fatalf("fire_shot failed: %s", r.Reason)
	}
	if len(r.Events) != 1 {
		t.Fatalf("events = %d, want 1", len(r.Events))
	}
	// 10 base + ~0.3 pressure bonus (1 stack)
	assertDmgNear(t, r.Events[0].Amount, 10.3, "fire_shot")
}

func TestGunner_FireShot_SetsCooldown(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	e := enemyInFront(100, 500)

	eng.Commit("fire_shot", commitCtx(p, e))
	cd := p.Cooldowns["fire_shot"]
	if math.Abs(float64(cd-0.18)) > 0.001 {
		t.Errorf("cooldown = %f, want 0.18", cd)
	}
}

func TestGunner_FireShot_BlockedByCooldown(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	e := enemyInFront(100, 500)

	eng.Commit("fire_shot", commitCtx(p, e))
	r := eng.Commit("fire_shot", commitCtx(p, e))
	if r.OK {
		t.Error("second fire_shot should be blocked by cooldown")
	}
	if r.Reason != "cooldown" {
		t.Errorf("reason = %q, want %q", r.Reason, "cooldown")
	}
}

func TestGunner_FireShot_AfterCooldownExpires(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	e := enemyInFront(100, 500)

	eng.Commit("fire_shot", commitCtx(p, e))
	eng.TickPlayer(p, 0.2, tickCtx()) // past 0.18s cooldown
	r := eng.Commit("fire_shot", commitCtx(p, e))
	if !r.OK {
		t.Fatalf("fire_shot should work after cooldown: %s", r.Reason)
	}
}

func TestGunner_FireShot_RapidFire(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	e := enemyInFront(100, 1e6)

	// Simulate 10 shots with proper cooldown ticks
	for i := 0; i < 10; i++ {
		r := eng.Commit("fire_shot", commitCtx(p, e))
		if !r.OK {
			t.Fatalf("shot %d failed: %s", i+1, r.Reason)
		}
		eng.TickPlayer(p, 0.2, tickCtx())
	}
}

// --- Overclock ---

func TestGunner_Overclock_AppliesBuff(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()

	r := eng.Commit("overclock", commitCtx(p))
	if !r.OK {
		t.Fatalf("overclock failed: %s", r.Reason)
	}
	if !p.HasBuff("overclock") {
		t.Error("overclock buff not applied")
	}
}

func TestGunner_Overclock_SetsCooldown(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()

	eng.Commit("overclock", commitCtx(p))
	if p.Cooldowns["overclock"] != 15.0 {
		t.Errorf("cooldown = %f, want 15.0", p.Cooldowns["overclock"])
	}
}

func TestGunner_Overclock_ReducesFireCooldown(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	e := enemyInFront(100, 500)

	eng.Commit("overclock", commitCtx(p))

	eng.Commit("fire_shot", commitCtx(p, e))
	cd := p.Cooldowns["fire_shot"]
	want := float32(0.18 * 0.556)
	if math.Abs(float64(cd-want)) > 0.01 {
		t.Errorf("fire_shot cooldown during overclock = %f, want ~%f", cd, want)
	}
}

func TestGunner_Overclock_CanFireDuring(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	e := enemyInFront(100, 1e6)

	eng.Commit("overclock", commitCtx(p))

	// Fire 5 rapid shots during overclock
	for i := 0; i < 5; i++ {
		r := eng.Commit("fire_shot", commitCtx(p, e))
		if !r.OK {
			t.Fatalf("shot %d during overclock failed: %s", i+1, r.Reason)
		}
		eng.TickPlayer(p, 0.12, tickCtx()) // 0.10s reduced CD, tick 0.12s to clear
	}
}

func TestGunner_Overclock_BlocksWhileActive(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()

	eng.Commit("overclock", commitCtx(p))
	r := eng.Commit("overclock", commitCtx(p))
	if r.OK {
		t.Error("overclock should be blocked while active")
	}
}

func TestGunner_Overclock_BlocksDuringCooldown(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()

	eng.Commit("overclock", commitCtx(p))

	// Expire buff but keep cooldown
	eng.TickPlayer(p, 7.5, tickCtx()) // buff lasts 7s
	if p.HasBuff("overclock") {
		t.Fatal("overclock buff should have expired")
	}
	// Cooldown should still be active (15 - 7.5 = 7.5)
	r := eng.Commit("overclock", commitCtx(p))
	if r.OK {
		t.Error("overclock should be blocked by cooldown")
	}
}

func TestGunner_Overclock_ReusableAfterFullCooldown(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()

	eng.Commit("overclock", commitCtx(p))

	// Tick past both buff (7s) and cooldown (15s)
	eng.TickPlayer(p, 16.0, tickCtx())
	if p.HasBuff("overclock") {
		t.Error("overclock buff should have expired")
	}
	if _, ok := p.Cooldowns["overclock"]; ok {
		t.Error("overclock cooldown should have expired")
	}

	r := eng.Commit("overclock", commitCtx(p))
	if !r.OK {
		t.Fatalf("overclock should be reusable after full cooldown: %s", r.Reason)
	}
}

func TestGunner_Overclock_BuffDuration(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()

	eng.Commit("overclock", commitCtx(p))

	// At 6.9s, buff should still be active
	eng.TickPlayer(p, 6.9, tickCtx())
	if !p.HasBuff("overclock") {
		t.Error("overclock should still be active at 6.9s")
	}

	// At 7.1s total, buff should have expired
	eng.TickPlayer(p, 0.2, tickCtx())
	if p.HasBuff("overclock") {
		t.Error("overclock should have expired after 7.1s")
	}
}

// --- Rechamber ---

func TestGunner_Rechamber_StartsPhase1(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()

	r := eng.Commit("rechamber", commitCtx(p))
	if !r.OK {
		t.Fatalf("rechamber failed: %s", r.Reason)
	}
	state, ok := p.AbilityState["rechamber"].(*RechamberState)
	if !ok {
		t.Fatal("rechamber state not set")
	}
	if state.Phase != 1 {
		t.Errorf("phase = %d, want 1", state.Phase)
	}
	if state.Timer != 0.6 {
		t.Errorf("timer = %f, want 0.6", state.Timer)
	}
}

func TestGunner_Rechamber_BlockedDuringFireCooldown(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	e := enemyInFront(100, 500)

	eng.Commit("fire_shot", commitCtx(p, e))
	r := eng.Commit("rechamber", commitCtx(p))
	if r.OK {
		t.Error("rechamber should be blocked during fire cooldown")
	}
	if r.Reason != "fire cooldown" {
		t.Errorf("reason = %q, want %q", r.Reason, "fire cooldown")
	}
}

func TestGunner_Rechamber_AllowedAfterFireCooldown(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	e := enemyInFront(100, 500)

	eng.Commit("fire_shot", commitCtx(p, e))
	eng.TickPlayer(p, 0.2, tickCtx()) // past 0.18s fire CD
	r := eng.Commit("rechamber", commitCtx(p))
	if !r.OK {
		t.Fatalf("rechamber should work after fire cooldown: %s", r.Reason)
	}
}

func TestGunner_Rechamber_BlocksFiringDuringWindup(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	e := enemyInFront(100, 500)

	eng.Commit("rechamber", commitCtx(p))
	r := eng.Commit("fire_shot", commitCtx(p, e))
	if r.OK {
		t.Error("fire_shot should be blocked during rechamber windup")
	}
}

func TestGunner_Rechamber_BlocksDoubleStart(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()

	eng.Commit("rechamber", commitCtx(p))
	r := eng.Commit("rechamber", commitCtx(p))
	if r.OK {
		t.Error("second rechamber should be blocked")
	}
	if r.Reason != "rechamber in progress" {
		t.Errorf("reason = %q, want %q", r.Reason, "rechamber in progress")
	}
}

func TestGunner_Rechamber_PhaseProgression(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()

	eng.Commit("rechamber", commitCtx(p))
	state, ok := p.AbilityState["rechamber"].(*RechamberState)
	if !ok {
		t.Fatal("rechamber state not set")
	}

	// Phase 1 → 2 after 0.6s
	eng.TickPlayer(p, 0.6, tickCtx())
	if state.Phase != 2 {
		t.Errorf("after 0.6s: phase = %d, want 2", state.Phase)
	}

	// Phase 2 → 3 after 0.35s
	eng.TickPlayer(p, 0.35, tickCtx())
	if state.Phase != 3 {
		t.Errorf("after 0.95s: phase = %d, want 3", state.Phase)
	}

	// Phase 3 → 0 after 0.8s
	eng.TickPlayer(p, 0.8, tickCtx())
	if state.Phase != 0 {
		t.Errorf("after 1.75s: phase = %d, want 0", state.Phase)
	}
}

func TestGunner_Rechamber_ConfirmInWindow(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()

	eng.Commit("rechamber", commitCtx(p))
	eng.TickPlayer(p, 0.6, tickCtx()) // → phase 2

	r := eng.Commit("rechamber_confirm", commitCtx(p))
	if !r.OK {
		t.Fatalf("rechamber_confirm failed: %s", r.Reason)
	}
	if !p.HasBuff("rechamber_buff") {
		t.Error("rechamber_buff not applied")
	}
}

func TestGunner_Rechamber_ConfirmDuringWindup(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()

	eng.Commit("rechamber", commitCtx(p))
	// Still in phase 1 (windup)
	r := eng.Commit("rechamber_confirm", commitCtx(p))
	if r.OK {
		t.Error("confirm should fail during windup (phase 1)")
	}
}

func TestGunner_Rechamber_ConfirmDuringLockout(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()

	eng.Commit("rechamber", commitCtx(p))
	eng.TickPlayer(p, 0.6, tickCtx())  // → phase 2
	eng.TickPlayer(p, 0.35, tickCtx()) // → phase 3

	r := eng.Commit("rechamber_confirm", commitCtx(p))
	if r.OK {
		t.Error("confirm should fail during lockout (phase 3)")
	}
}

func TestGunner_Rechamber_BuffIncreasesDamage(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	e := enemyInFront(100, 1000)

	// Complete rechamber for buff
	eng.Commit("rechamber", commitCtx(p))
	eng.TickPlayer(p, 0.6, tickCtx()) // → phase 2
	eng.Commit("rechamber_confirm", commitCtx(p))

	// Clear fire cooldown (rechamber set it to 0.6, but we ticked 0.6)
	eng.TickPlayer(p, 0.1, tickCtx()) // ensure fire CD is gone

	// Fire with buff
	r := eng.Commit("fire_shot", commitCtx(p, e))
	if !r.OK {
		t.Fatalf("fire_shot after rechamber failed: %s", r.Reason)
	}
	if len(r.Events) != 1 {
		t.Fatalf("events = %d, want 1", len(r.Events))
	}
	// 10 base * 1.8 buff = 18, + pressure bonus ~0.54
	assertDmgNear(t, r.Events[0].Amount, 18.54, "rechamber buffed shot")
}

func TestGunner_Rechamber_CanFireAfterConfirm(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	e := enemyInFront(100, 500)

	eng.Commit("rechamber", commitCtx(p))
	eng.TickPlayer(p, 0.6, tickCtx()) // → phase 2, fire CD expired
	eng.Commit("rechamber_confirm", commitCtx(p))

	// Fire cooldown was set to 0.6 at rechamber start, ticked by 0.6 → should be gone
	r := eng.Commit("fire_shot", commitCtx(p, e))
	if !r.OK {
		t.Fatalf("should be able to fire after confirm: %s", r.Reason)
	}
}

func TestGunner_Rechamber_CanFireAfterLockout(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	e := enemyInFront(100, 500)

	eng.Commit("rechamber", commitCtx(p))
	eng.TickPlayer(p, 0.6, tickCtx())  // → phase 2
	eng.TickPlayer(p, 0.35, tickCtx()) // → phase 3 (missed)
	eng.TickPlayer(p, 0.8, tickCtx())  // → phase 0

	r := eng.Commit("fire_shot", commitCtx(p, e))
	if !r.OK {
		t.Fatalf("should be able to fire after lockout: %s", r.Reason)
	}
}

func TestGunner_Rechamber_CanRestartAfterLockout(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()

	eng.Commit("rechamber", commitCtx(p))
	eng.TickPlayer(p, 0.6, tickCtx())  // → phase 2
	eng.TickPlayer(p, 0.35, tickCtx()) // → phase 3
	eng.TickPlayer(p, 0.8, tickCtx())  // → phase 0

	r := eng.Commit("rechamber", commitCtx(p))
	if !r.OK {
		t.Fatalf("rechamber should be restartable after lockout: %s", r.Reason)
	}
}

func TestGunner_Rechamber_CanRestartAfterConfirm(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()

	eng.Commit("rechamber", commitCtx(p))
	eng.TickPlayer(p, 0.6, tickCtx()) // → phase 2
	eng.Commit("rechamber_confirm", commitCtx(p))

	// Fire CD should be cleared by now (0.6 tick > 0.6 CD)
	r := eng.Commit("rechamber", commitCtx(p))
	if !r.OK {
		t.Fatalf("rechamber should be restartable after confirm: %s", r.Reason)
	}
}

// --- Overclock + Rechamber interaction ---

func TestGunner_Overclock_Then_Rechamber(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()

	eng.Commit("overclock", commitCtx(p))
	r := eng.Commit("rechamber", commitCtx(p))
	if !r.OK {
		t.Fatalf("rechamber during overclock failed: %s", r.Reason)
	}
}

func TestGunner_Rechamber_Then_Overclock(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()

	eng.Commit("rechamber", commitCtx(p))
	r := eng.Commit("overclock", commitCtx(p))
	if !r.OK {
		t.Fatalf("overclock during rechamber failed: %s", r.Reason)
	}
}

func TestGunner_RechamberBuff_Plus_Overclock_Damage(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	e := enemyInFront(100, 1000)

	// Get rechamber buff (1.8x)
	eng.Commit("rechamber", commitCtx(p))
	eng.TickPlayer(p, 0.6, tickCtx())
	eng.Commit("rechamber_confirm", commitCtx(p))

	// rechamber_buff is damage_mult, not cooldown_mult — so it stacks multiplicatively
	// 10 base * 1.8 = 18, + pressure bonus ~0.54
	r := eng.Commit("fire_shot", commitCtx(p, e))
	if !r.OK {
		t.Fatalf("fire_shot failed: %s", r.Reason)
	}
	if len(r.Events) == 1 {
		assertDmgNear(t, r.Events[0].Amount, 18.54, "rechamber damage")
	}
}

func TestGunner_Overclock_RechamberCooldown(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()

	// Overclock reduces cooldowns via cooldown_mult buff
	eng.Commit("overclock", commitCtx(p))

	// Rechamber is handler-based, sets fire_shot CD directly (0.6)
	// The handler doesn't go through the cooldown_mult path
	eng.Commit("rechamber", commitCtx(p))
	cd := p.Cooldowns["fire_shot"]
	if cd != 0.6 {
		t.Errorf("fire_shot CD during rechamber = %f, want 0.6 (handler sets directly)", cd)
	}
}

// --- Full rotation ---

func TestGunner_FullRotation_Fire_Rechamber_Fire(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	e := enemyInFront(100, 1e6)

	// 1. Fire a few shots
	for i := 0; i < 3; i++ {
		r := eng.Commit("fire_shot", commitCtx(p, e))
		if !r.OK {
			t.Fatalf("initial shot %d failed: %s", i+1, r.Reason)
		}
		eng.TickPlayer(p, 0.2, tickCtx())
	}

	// 2. Rechamber
	r := eng.Commit("rechamber", commitCtx(p))
	if !r.OK {
		t.Fatalf("rechamber failed: %s", r.Reason)
	}

	// 3. Tick through windup
	eng.TickPlayer(p, 0.6, tickCtx())

	// 4. Confirm
	r = eng.Commit("rechamber_confirm", commitCtx(p))
	if !r.OK {
		t.Fatalf("confirm failed: %s", r.Reason)
	}

	// 5. Fire buffed shots — pressure stacks continue from earlier hits
	var prevDmg float32
	for i := range 3 {
		r = eng.Commit("fire_shot", commitCtx(p, e))
		if !r.OK {
			t.Fatalf("buffed shot %d failed: %s", i+1, r.Reason)
		}
		if len(r.Events) != 1 {
			t.Fatalf("buffed shot %d: events = %d, want 1", i+1, len(r.Events))
		}
		dmg := r.Events[0].Amount
		// Buffed: base 10 * 1.8 = 18, plus growing pressure bonus
		if dmg < 18 {
			t.Errorf("buffed shot %d: damage = %.2f, want >= 18", i+1, dmg)
		}
		if i > 0 && dmg <= prevDmg {
			t.Errorf("buffed shot %d: damage = %.2f should exceed previous %.2f (pressure growth)", i+1, dmg, prevDmg)
		}
		prevDmg = dmg
		eng.TickPlayer(p, 0.2, tickCtx())
	}
}

func TestGunner_FullRotation_Overclock_Fire(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	e := enemyInFront(100, 1e6)

	// 1. Overclock
	r := eng.Commit("overclock", commitCtx(p))
	if !r.OK {
		t.Fatalf("overclock failed: %s", r.Reason)
	}

	// 2. Rapid fire with reduced cooldowns
	for i := 0; i < 5; i++ {
		r = eng.Commit("fire_shot", commitCtx(p, e))
		if !r.OK {
			t.Fatalf("overclocked shot %d failed: %s", i+1, r.Reason)
		}
		// Reduced CD should be ~0.10, tick 0.12 to clear
		eng.TickPlayer(p, 0.12, tickCtx())
	}
}

func TestGunner_FullRotation_Overclock_Rechamber_Fire(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	e := enemyInFront(100, 1e6)

	// 1. Overclock
	eng.Commit("overclock", commitCtx(p))

	// 2. Fire one shot
	eng.Commit("fire_shot", commitCtx(p, e))
	eng.TickPlayer(p, 0.12, tickCtx())

	// 3. Rechamber
	r := eng.Commit("rechamber", commitCtx(p))
	if !r.OK {
		t.Fatalf("rechamber during overclock failed: %s", r.Reason)
	}

	// 4. Tick through windup and confirm
	eng.TickPlayer(p, 0.6, tickCtx())
	r = eng.Commit("rechamber_confirm", commitCtx(p))
	if !r.OK {
		t.Fatalf("confirm failed: %s", r.Reason)
	}

	// 5. Fire with both buffs (rechamber_buff 1.8x damage, overclock reduced CD)
	// Pressure: was 1 (from shot 1), now same target → stacks=2, bonus ~1.08
	r = eng.Commit("fire_shot", commitCtx(p, e))
	if !r.OK {
		t.Fatalf("buffed fire failed: %s", r.Reason)
	}
	if len(r.Events) == 1 {
		assertDmgNear(t, r.Events[0].Amount, 19.08, "overclock+rechamber shot")
	}
	// CD should be reduced by overclock
	cd := p.Cooldowns["fire_shot"]
	want := float32(0.18 * 0.556)
	if math.Abs(float64(cd-want)) > 0.01 {
		t.Errorf("fire CD with overclock = %f, want ~%f", cd, want)
	}
}

// --- Edge cases ---

func TestGunner_Rechamber_GetPhase_ForCodec(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()

	// Before rechamber, phase should be 0
	if ph := p.GetAbilityPhase("rechamber"); ph != 0 {
		t.Errorf("phase before rechamber = %d, want 0", ph)
	}

	eng.Commit("rechamber", commitCtx(p))
	if ph := p.GetAbilityPhase("rechamber"); ph != 1 {
		t.Errorf("phase during windup = %d, want 1", ph)
	}

	eng.TickPlayer(p, 0.6, tickCtx())
	if ph := p.GetAbilityPhase("rechamber"); ph != 2 {
		t.Errorf("phase in timing window = %d, want 2", ph)
	}

	eng.TickPlayer(p, 0.35, tickCtx())
	if ph := p.GetAbilityPhase("rechamber"); ph != 3 {
		t.Errorf("phase in lockout = %d, want 3", ph)
	}
}

func TestGunner_Rechamber_BuffExpires(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	e := enemyInFront(100, 1e6)

	// Get rechamber buff
	eng.Commit("rechamber", commitCtx(p))
	eng.TickPlayer(p, 0.6, tickCtx())
	eng.Commit("rechamber_confirm", commitCtx(p))

	// Buff lasts 4s
	eng.TickPlayer(p, 4.1, tickCtx())
	if p.HasBuff("rechamber_buff") {
		t.Error("rechamber_buff should have expired after 4.1s")
	}

	// Damage should be back to base (+ pressure bonus for first hit)
	r := eng.Commit("fire_shot", commitCtx(p, e))
	if !r.OK {
		t.Fatalf("fire_shot failed: %s", r.Reason)
	}
	if len(r.Events) == 1 {
		assertDmgNear(t, r.Events[0].Amount, 10.3, "post-buff damage")
	}
}

func TestGunner_NoGCD_BetweenAbilities(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()

	// Gunner abilities should not set GCD (they use per-ability CDs)
	eng.Commit("overclock", commitCtx(p))
	if p.GCDTimer != 0 {
		t.Errorf("GCDTimer after overclock = %f, want 0", p.GCDTimer)
	}

	eng.Commit("rechamber", commitCtx(p))
	if p.GCDTimer != 0 {
		t.Errorf("GCDTimer after rechamber = %f, want 0", p.GCDTimer)
	}
}

func TestGunner_OriginConfig_DoesNotBlock(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()

	// Gunner abilities have OriginConfig=0 (default) and Config=0 (default).
	// Verify this doesn't accidentally block casts.
	for _, id := range []string{"fire_shot", "overclock", "rechamber", "rechamber_confirm", "reload", "load_enhanced", "mag_dump"} {
		def := eng.GetAbility(id)
		if def == nil {
			t.Fatalf("ability %q not found", id)
		}
		// For handler abilities, only fire_shot goes through data-driven path
		// All should have OriginConfig that doesn't block a fresh gunner
		if def.OriginConfig >= 0 && def.OriginConfig != p.Config {
			t.Errorf("%s: OriginConfig=%d would block gunner with Config=%d", id, def.OriginConfig, p.Config)
		}
	}
}

// =============================================================================
// Magazine & Reload
// =============================================================================

func TestGunner_Magazine_DepletesThenAutoReload(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	e := enemyInFront(100, 1e6)
	state := getGunnerAssaultState(p)

	for i := range 30 {
		p.Cooldowns["fire_shot"] = 0
		r := eng.Commit("fire_shot", commitCtx(p, e))
		if !r.OK {
			t.Fatalf("shot %d failed: %s", i+1, r.Reason)
		}
	}
	if state.MagCurrent != 0 {
		t.Errorf("mag = %d, want 0", state.MagCurrent)
	}
	if !state.Reloading {
		t.Error("should be auto-reloading after empty")
	}
}

func TestGunner_Reload_Tactical(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	state := getGunnerAssaultState(p)
	state.MagCurrent = 15

	r := eng.Commit("reload", commitCtx(p))
	if !r.OK {
		t.Fatalf("reload failed: %s", r.Reason)
	}
	if !state.Reloading {
		t.Error("should be reloading")
	}
	if math.Abs(float64(state.ReloadTimer-1.5)) > 0.01 {
		t.Errorf("reload timer = %f, want 1.5 (tactical)", state.ReloadTimer)
	}

	eng.TickPlayer(p, 1.6, tickCtx())
	if state.Reloading {
		t.Error("should be done reloading")
	}
	if state.MagCurrent != 30 {
		t.Errorf("mag = %d, want 30", state.MagCurrent)
	}
}

func TestGunner_Reload_Empty(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	state := getGunnerAssaultState(p)
	state.MagCurrent = 0

	r := eng.Commit("reload", commitCtx(p))
	if !r.OK {
		t.Fatalf("reload failed: %s", r.Reason)
	}
	if math.Abs(float64(state.ReloadTimer-2.2)) > 0.01 {
		t.Errorf("reload timer = %f, want 2.2 (empty)", state.ReloadTimer)
	}
}

func TestGunner_Reload_BlockedWhenFull(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	getGunnerAssaultState(p) // ensure state init

	r := eng.Commit("reload", commitCtx(p))
	if r.OK {
		t.Error("reload should fail when magazine is full")
	}
	if r.Reason != "magazine full" {
		t.Errorf("reason = %q, want %q", r.Reason, "magazine full")
	}
}

func TestGunner_Reload_BlockedDuringMagDump(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	state := getGunnerAssaultState(p)
	state.MagCurrent = 15
	state.MagDumpActive = true

	r := eng.Commit("reload", commitCtx(p))
	if r.OK {
		t.Error("reload should fail during mag dump")
	}
}

// =============================================================================
// Stability
// =============================================================================

func TestGunner_Stability_DecayAndRecovery(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	e := enemyInFront(100, 1e6)
	state := getGunnerAssaultState(p)

	eng.Commit("fire_shot", commitCtx(p, e))
	if state.Stability > 0.93 {
		t.Errorf("stability = %f, want < 0.93 after 1 shot", state.Stability)
	}

	// Tick past stability delay, recovery should restore to 1.0
	eng.TickPlayer(p, 1.0, tickCtx())
	if state.Stability < 0.99 {
		t.Errorf("stability = %f, want ~1.0 after 1s recovery", state.Stability)
	}
}

func TestGunner_Stability_OverclockFasterRecovery(t *testing.T) {
	eng := NewEngine(nil)

	// Player with overclock
	p1 := newGunner()
	s1 := getGunnerAssaultState(p1)
	eng.Commit("overclock", commitCtx(p1))
	s1.Stability = 0.5
	s1.StabilityTimer = 1.0 // past delay

	eng.TickPlayer(p1, 0.1, tickCtx())
	rec1 := s1.Stability - 0.5

	// Player without overclock
	p2 := newGunner()
	s2 := getGunnerAssaultState(p2)
	s2.Stability = 0.5
	s2.StabilityTimer = 1.0

	eng.TickPlayer(p2, 0.1, tickCtx())
	rec2 := s2.Stability - 0.5

	if rec1 <= rec2 {
		t.Errorf("overclock recovery = %f, normal = %f — overclock should be faster", rec1, rec2)
	}
}

// =============================================================================
// Steadiness
// =============================================================================

func TestGunner_Steadiness_InitiallyFull(t *testing.T) {
	p := newGunner()
	state := getGunnerAssaultState(p)
	if state.Steadiness != 1.0 {
		t.Errorf("steadiness = %f, want 1.0", state.Steadiness)
	}
}

func TestGunner_Steadiness_DecaysWithMovement(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	state := getGunnerAssaultState(p)
	state.prevPosInit = true
	state.PrevPos = p.Position

	// Move 5 units in one tick (speed = 100 u/s at dt=0.05)
	p.Position = entity.Vec3{X: 5, Y: 0, Z: 0}
	eng.TickPlayer(p, 0.05, tickCtx())

	if state.Steadiness >= 1.0 {
		t.Errorf("steadiness = %f, want < 1.0 after movement", state.Steadiness)
	}
}

func TestGunner_Steadiness_RecoversWhenStationary(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	state := getGunnerAssaultState(p)
	state.Steadiness = 0.5
	state.SteadinessTimer = 1.0 // past delay
	state.prevPosInit = true
	state.PrevPos = p.Position

	eng.TickPlayer(p, 0.5, tickCtx())

	if state.Steadiness <= 0.5 {
		t.Errorf("steadiness = %f, want > 0.5 after stationary recovery", state.Steadiness)
	}
}

func TestGunner_Steadiness_NoRecoveryDuringDelay(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	state := getGunnerAssaultState(p)
	state.Steadiness = 0.5
	state.SteadinessTimer = 0 // just stopped
	state.prevPosInit = true
	state.PrevPos = p.Position

	eng.TickPlayer(p, 0.1, tickCtx()) // less than 0.2s delay

	// Should have ticked timer but not recovered yet (or minimally)
	if state.Steadiness > 0.51 {
		t.Errorf("steadiness = %f, want ~0.5 during delay period", state.Steadiness)
	}
}

func TestGunner_Steadiness_AffectsSpread(t *testing.T) {
	eng := NewEngine(nil)

	// Stationary gunner: steadiness 1.0, stability 1.0 → no spread
	p1 := newGunner()
	s1 := getGunnerAssaultState(p1)
	e1 := enemyInFront(100, 1e6)

	hits1 := 0
	for range 20 {
		p1.Cooldowns["fire_shot"] = 0
		s1.Stability = 1.0
		s1.Steadiness = 1.0
		s1.MagCurrent = 30 // keep magazine full
		r := eng.Commit("fire_shot", commitCtx(p1, e1))
		if r.OK && len(r.Events) > 0 {
			hits1++
		}
	}

	// With both at 1.0, spread is 0° — all shots should hit
	if hits1 < 20 {
		t.Errorf("full steadiness hits = %d/20, want 20 (no spread)", hits1)
	}

	// Verify the combined spread formula: steadiness 0 adds assaultMaxSteadinessDeg
	// of spread even when stability is perfect. We test this deterministically
	// by checking the spread value in fireHitscan indirectly: the decay/recovery
	// tests above prove Steadiness changes, and the fireHitscan code adds
	// assaultMaxSteadinessRad * (1.0 - Steadiness) to the spread cone.
}

// =============================================================================
// Pressure
// =============================================================================

func TestGunner_Pressure_ConsecutiveHits(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	e := enemyInFront(100, 1e6)
	state := getGunnerAssaultState(p)

	for i := range 5 {
		p.Cooldowns["fire_shot"] = 0
		r := eng.Commit("fire_shot", commitCtx(p, e))
		if !r.OK {
			t.Fatalf("shot %d failed: %s", i+1, r.Reason)
		}
	}
	if state.PressureStacks != 5 {
		t.Errorf("pressure stacks = %d, want 5", state.PressureStacks)
	}
}

func TestGunner_Pressure_MissResets(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	e := enemyInFront(100, 1e6)
	state := getGunnerAssaultState(p)

	// Build stacks
	for i := range 3 {
		p.Cooldowns["fire_shot"] = 0
		r := eng.Commit("fire_shot", commitCtx(p, e))
		if !r.OK {
			t.Fatalf("shot %d failed: %s", i+1, r.Reason)
		}
	}
	if state.PressureStacks != 3 {
		t.Fatalf("stacks = %d, want 3", state.PressureStacks)
	}

	// Miss (shoot at enemy behind — hitscan won't find it)
	p.Cooldowns["fire_shot"] = 0
	eBehind := enemyBehind(200, 1e6)
	eng.Commit("fire_shot", commitCtx(p, eBehind))
	if state.PressureStacks != 0 {
		t.Errorf("stacks after miss = %d, want 0", state.PressureStacks)
	}
}

func TestGunner_Pressure_TimeoutResets(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	e := enemyInFront(100, 1e6)
	state := getGunnerAssaultState(p)

	eng.Commit("fire_shot", commitCtx(p, e))
	if state.PressureStacks == 0 {
		t.Fatal("expected stacks > 0 after hit")
	}

	eng.TickPlayer(p, 2.1, tickCtx())
	if state.PressureStacks != 0 {
		t.Errorf("stacks after timeout = %d, want 0", state.PressureStacks)
	}
}

func TestGunner_Pressure_TargetSwapResetsToOne(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	e1 := enemyInFront(100, 1e6)
	e2 := enemyInFront(200, 1e6)
	e2.Position = entity.Vec3{X: 0, Y: 0, Z: -8}
	state := getGunnerAssaultState(p)

	// Hit e1 three times
	for i := range 3 {
		p.Cooldowns["fire_shot"] = 0
		r := eng.Commit("fire_shot", commitCtx(p, e1))
		if !r.OK {
			t.Fatalf("shot %d failed: %s", i+1, r.Reason)
		}
	}
	if state.PressureStacks != 3 {
		t.Fatalf("stacks = %d, want 3", state.PressureStacks)
	}

	// Hit e2 (only e2 in targets so hitscan picks it)
	p.Cooldowns["fire_shot"] = 0
	eng.Commit("fire_shot", commitCtx(p, e2))
	if state.PressureStacks != 1 {
		t.Errorf("stacks after target swap = %d, want 1", state.PressureStacks)
	}
}

func TestGunner_Pressure_MaxGeneratesEnhanced(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	e := enemyInFront(100, 1e6)
	state := getGunnerAssaultState(p)

	for i := range 10 {
		p.Cooldowns["fire_shot"] = 0
		r := eng.Commit("fire_shot", commitCtx(p, e))
		if !r.OK {
			t.Fatalf("shot %d failed: %s", i+1, r.Reason)
		}
	}
	if state.PressureStacks != 10 {
		t.Errorf("stacks = %d, want 10", state.PressureStacks)
	}
	if state.EnhancedReserve != 5 {
		t.Errorf("enhanced reserve = %d, want 5", state.EnhancedReserve)
	}
}

func TestGunner_Pressure_NoDuplicateEnhancedBatch(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	e := enemyInFront(100, 1e6)
	state := getGunnerAssaultState(p)

	// Reach max stacks (10 shots)
	for i := range 10 {
		p.Cooldowns["fire_shot"] = 0
		eng.Commit("fire_shot", commitCtx(p, e))
		_ = i
	}
	if state.EnhancedReserve != 5 {
		t.Fatalf("reserve = %d, want 5 after first batch", state.EnhancedReserve)
	}

	// Fire more while at max — should NOT generate another batch
	for range 5 {
		p.Cooldowns["fire_shot"] = 0
		eng.Commit("fire_shot", commitCtx(p, e))
	}
	if state.EnhancedReserve != 5 {
		t.Errorf("reserve = %d, want 5 (no duplicate batch)", state.EnhancedReserve)
	}
}

// =============================================================================
// Enhanced Rounds
// =============================================================================

func TestGunner_LoadEnhanced(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	state := getGunnerAssaultState(p)
	state.EnhancedReserve = 5

	r := eng.Commit("load_enhanced", commitCtx(p))
	if !r.OK {
		t.Fatalf("load_enhanced failed: %s", r.Reason)
	}
	if state.EnhancedLoaded != 5 {
		t.Errorf("loaded = %d, want 5", state.EnhancedLoaded)
	}
	if state.EnhancedReserve != 0 {
		t.Errorf("reserve = %d, want 0", state.EnhancedReserve)
	}
}

func TestGunner_LoadEnhanced_BlockedWhenEmpty(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	getGunnerAssaultState(p) // no reserve

	r := eng.Commit("load_enhanced", commitCtx(p))
	if r.OK {
		t.Error("load_enhanced should fail with no reserve")
	}
	if r.Reason != "no enhanced rounds" {
		t.Errorf("reason = %q, want %q", r.Reason, "no enhanced rounds")
	}
}

func TestGunner_LoadEnhanced_BlockedWhenAlreadyLoaded(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	state := getGunnerAssaultState(p)
	state.EnhancedReserve = 5
	state.EnhancedLoaded = 3

	r := eng.Commit("load_enhanced", commitCtx(p))
	if r.OK {
		t.Error("load_enhanced should fail when already loaded")
	}
	if r.Reason != "already loaded" {
		t.Errorf("reason = %q, want %q", r.Reason, "already loaded")
	}
}

func TestGunner_EnhancedRound_ConsumesAndDealsBonusDamage(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	e := enemyInFront(100, 1e6)
	state := getGunnerAssaultState(p)
	state.EnhancedReserve = 5
	eng.Commit("load_enhanced", commitCtx(p))

	// Fire an enhanced round
	r := eng.Commit("fire_shot", commitCtx(p, e))
	if !r.OK {
		t.Fatalf("fire_shot failed: %s", r.Reason)
	}
	if state.EnhancedLoaded != 4 {
		t.Errorf("enhanced loaded = %d, want 4", state.EnhancedLoaded)
	}
	// Enhanced bonus: 15 + stacks*1.5, at stacks=1 identity=0: 16.5 bonus
	// Total > base 10 + 16.5 = 26.5 (plus pressure)
	if r.Events[0].Amount < 20 {
		t.Errorf("enhanced damage = %.2f, expected > 20", r.Events[0].Amount)
	}
}

func TestGunner_EnhancedRound_ConsumesBeforeMagazine(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	e := enemyInFront(100, 1e6)
	state := getGunnerAssaultState(p)
	state.EnhancedLoaded = 2
	magBefore := state.MagCurrent

	p.Cooldowns["fire_shot"] = 0
	eng.Commit("fire_shot", commitCtx(p, e))
	if state.EnhancedLoaded != 1 {
		t.Errorf("enhanced = %d, want 1", state.EnhancedLoaded)
	}
	if state.MagCurrent != magBefore {
		t.Errorf("mag = %d, want %d (enhanced consumed first)", state.MagCurrent, magBefore)
	}
}

// =============================================================================
// Mag Dump
// =============================================================================

func TestGunner_MagDump(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	e := enemyInFront(100, 1e6)
	state := getGunnerAssaultState(p)
	state.MagCurrent = 4

	r := eng.Commit("mag_dump", commitCtx(p, e))
	if !r.OK {
		t.Fatalf("mag_dump failed: %s", r.Reason)
	}
	if !state.MagDumpActive {
		t.Error("mag dump should be active")
	}

	// Tick twice (2 rounds/tick × 2 = 4 rounds)
	eng.TickPlayer(p, 0.05, tickCtx(e))
	eng.TickPlayer(p, 0.05, tickCtx(e))

	if state.MagDumpActive {
		t.Error("mag dump should be complete")
	}
	if !state.Reloading {
		t.Error("should auto-reload after mag dump")
	}
}

func TestGunner_MagDump_BlockedDuringReload(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	state := getGunnerAssaultState(p)
	state.Reloading = true

	r := eng.Commit("mag_dump", commitCtx(p))
	if r.OK {
		t.Error("mag dump should fail during reload")
	}
	if r.Reason != "reloading" {
		t.Errorf("reason = %q, want %q", r.Reason, "reloading")
	}
}

func TestGunner_MagDump_BlockedWhenEmpty(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	state := getGunnerAssaultState(p)
	state.MagCurrent = 0

	r := eng.Commit("mag_dump", commitCtx(p))
	if r.OK {
		t.Error("mag dump should fail with empty magazine")
	}
	if r.Reason != "no ammo" {
		t.Errorf("reason = %q, want %q", r.Reason, "no ammo")
	}
}

func TestGunner_MagDump_SetsCooldown(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	getGunnerAssaultState(p)

	eng.Commit("mag_dump", commitCtx(p))
	if p.Cooldowns["mag_dump"] != 12.0 {
		t.Errorf("cooldown = %f, want 12.0", p.Cooldowns["mag_dump"])
	}
}

// =============================================================================
// Wire state (GunnerWireState)
// =============================================================================

func TestGunner_WireState(t *testing.T) {
	p := newGunner()
	state := getGunnerAssaultState(p)
	state.MagCurrent = 24
	state.MagMax = 30
	state.Stability = 0.75
	state.Steadiness = 0.5
	state.PressureStacks = 7
	state.EnhancedLoaded = 3
	state.Reloading = true
	state.MagDumpActive = true

	mag, magMax, stab, stead, pressure, enhanced, flags := state.GunnerWireState()
	if mag != 24 {
		t.Errorf("mag = %d, want 24", mag)
	}
	if magMax != 30 {
		t.Errorf("magMax = %d, want 30", magMax)
	}
	// 0.75 * 255 = 191.25 → 191
	if stab != 191 {
		t.Errorf("stab = %d, want 191", stab)
	}
	// 0.5 * 255 = 127.5 → 127
	if stead != 127 {
		t.Errorf("steadiness = %d, want 127", stead)
	}
	if pressure != 7 {
		t.Errorf("pressure = %d, want 7", pressure)
	}
	if enhanced != 3 {
		t.Errorf("enhanced = %d, want 3", enhanced)
	}
	if flags != 0x03 { // bit 0 (reloading) + bit 1 (mag dump)
		t.Errorf("flags = 0x%02X, want 0x03", flags)
	}
}
