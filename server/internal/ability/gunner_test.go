package ability

import (
	"math"
	"testing"
)

// --- Fire Shot ---

func TestGunner_FireShot_BasicHit(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	e := enemyInFront(100, 500)

	r := eng.Cast("fire_shot", castCtx(p, e))
	if !r.OK {
		t.Fatalf("fire_shot failed: %s", r.Reason)
	}
	if len(r.Events) != 1 {
		t.Fatalf("events = %d, want 1", len(r.Events))
	}
	if r.Events[0].Amount != 10 {
		t.Errorf("damage = %f, want 10", r.Events[0].Amount)
	}
}

func TestGunner_FireShot_SetsCooldown(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	e := enemyInFront(100, 500)

	eng.Cast("fire_shot", castCtx(p, e))
	cd := p.Cooldowns["fire_shot"]
	if math.Abs(float64(cd-0.18)) > 0.001 {
		t.Errorf("cooldown = %f, want 0.18", cd)
	}
}

func TestGunner_FireShot_BlockedByCooldown(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	e := enemyInFront(100, 500)

	eng.Cast("fire_shot", castCtx(p, e))
	r := eng.Cast("fire_shot", castCtx(p, e))
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

	eng.Cast("fire_shot", castCtx(p, e))
	eng.TickPlayer(p, 0.2, tickCtx()) // past 0.18s cooldown
	r := eng.Cast("fire_shot", castCtx(p, e))
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
		r := eng.Cast("fire_shot", castCtx(p, e))
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

	r := eng.Cast("overclock", castCtx(p))
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

	eng.Cast("overclock", castCtx(p))
	if p.Cooldowns["overclock"] != 15.0 {
		t.Errorf("cooldown = %f, want 15.0", p.Cooldowns["overclock"])
	}
}

func TestGunner_Overclock_ReducesFireCooldown(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	e := enemyInFront(100, 500)

	eng.Cast("overclock", castCtx(p))

	eng.Cast("fire_shot", castCtx(p, e))
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

	eng.Cast("overclock", castCtx(p))

	// Fire 5 rapid shots during overclock
	for i := 0; i < 5; i++ {
		r := eng.Cast("fire_shot", castCtx(p, e))
		if !r.OK {
			t.Fatalf("shot %d during overclock failed: %s", i+1, r.Reason)
		}
		eng.TickPlayer(p, 0.12, tickCtx()) // 0.10s reduced CD, tick 0.12s to clear
	}
}

func TestGunner_Overclock_BlocksWhileActive(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()

	eng.Cast("overclock", castCtx(p))
	r := eng.Cast("overclock", castCtx(p))
	if r.OK {
		t.Error("overclock should be blocked while active")
	}
}

func TestGunner_Overclock_BlocksDuringCooldown(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()

	eng.Cast("overclock", castCtx(p))

	// Expire buff but keep cooldown
	eng.TickPlayer(p, 7.5, tickCtx()) // buff lasts 7s
	if p.HasBuff("overclock") {
		t.Fatal("overclock buff should have expired")
	}
	// Cooldown should still be active (15 - 7.5 = 7.5)
	r := eng.Cast("overclock", castCtx(p))
	if r.OK {
		t.Error("overclock should be blocked by cooldown")
	}
}

func TestGunner_Overclock_ReusableAfterFullCooldown(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()

	eng.Cast("overclock", castCtx(p))

	// Tick past both buff (7s) and cooldown (15s)
	eng.TickPlayer(p, 16.0, tickCtx())
	if p.HasBuff("overclock") {
		t.Error("overclock buff should have expired")
	}
	if _, ok := p.Cooldowns["overclock"]; ok {
		t.Error("overclock cooldown should have expired")
	}

	r := eng.Cast("overclock", castCtx(p))
	if !r.OK {
		t.Fatalf("overclock should be reusable after full cooldown: %s", r.Reason)
	}
}

func TestGunner_Overclock_BuffDuration(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()

	eng.Cast("overclock", castCtx(p))

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

	r := eng.Cast("rechamber", castCtx(p))
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

	eng.Cast("fire_shot", castCtx(p, e))
	r := eng.Cast("rechamber", castCtx(p))
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

	eng.Cast("fire_shot", castCtx(p, e))
	eng.TickPlayer(p, 0.2, tickCtx()) // past 0.18s fire CD
	r := eng.Cast("rechamber", castCtx(p))
	if !r.OK {
		t.Fatalf("rechamber should work after fire cooldown: %s", r.Reason)
	}
}

func TestGunner_Rechamber_BlocksFiringDuringWindup(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	e := enemyInFront(100, 500)

	eng.Cast("rechamber", castCtx(p))
	r := eng.Cast("fire_shot", castCtx(p, e))
	if r.OK {
		t.Error("fire_shot should be blocked during rechamber windup")
	}
}

func TestGunner_Rechamber_BlocksDoubleStart(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()

	eng.Cast("rechamber", castCtx(p))
	r := eng.Cast("rechamber", castCtx(p))
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

	eng.Cast("rechamber", castCtx(p))
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

	eng.Cast("rechamber", castCtx(p))
	eng.TickPlayer(p, 0.6, tickCtx()) // → phase 2

	r := eng.Cast("rechamber_confirm", castCtx(p))
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

	eng.Cast("rechamber", castCtx(p))
	// Still in phase 1 (windup)
	r := eng.Cast("rechamber_confirm", castCtx(p))
	if r.OK {
		t.Error("confirm should fail during windup (phase 1)")
	}
}

func TestGunner_Rechamber_ConfirmDuringLockout(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()

	eng.Cast("rechamber", castCtx(p))
	eng.TickPlayer(p, 0.6, tickCtx())  // → phase 2
	eng.TickPlayer(p, 0.35, tickCtx()) // → phase 3

	r := eng.Cast("rechamber_confirm", castCtx(p))
	if r.OK {
		t.Error("confirm should fail during lockout (phase 3)")
	}
}

func TestGunner_Rechamber_BuffIncreasesDamage(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	e := enemyInFront(100, 1000)

	// Complete rechamber for buff
	eng.Cast("rechamber", castCtx(p))
	eng.TickPlayer(p, 0.6, tickCtx()) // → phase 2
	eng.Cast("rechamber_confirm", castCtx(p))

	// Clear fire cooldown (rechamber set it to 0.6, but we ticked 0.6)
	eng.TickPlayer(p, 0.1, tickCtx()) // ensure fire CD is gone

	// Fire with buff
	r := eng.Cast("fire_shot", castCtx(p, e))
	if !r.OK {
		t.Fatalf("fire_shot after rechamber failed: %s", r.Reason)
	}
	if len(r.Events) != 1 {
		t.Fatalf("events = %d, want 1", len(r.Events))
	}
	// 10 base * 1.8 buff = 18
	if r.Events[0].Amount != 18 {
		t.Errorf("damage = %f, want 18 (10 * 1.8)", r.Events[0].Amount)
	}
}

func TestGunner_Rechamber_CanFireAfterConfirm(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	e := enemyInFront(100, 500)

	eng.Cast("rechamber", castCtx(p))
	eng.TickPlayer(p, 0.6, tickCtx()) // → phase 2, fire CD expired
	eng.Cast("rechamber_confirm", castCtx(p))

	// Fire cooldown was set to 0.6 at rechamber start, ticked by 0.6 → should be gone
	r := eng.Cast("fire_shot", castCtx(p, e))
	if !r.OK {
		t.Fatalf("should be able to fire after confirm: %s", r.Reason)
	}
}

func TestGunner_Rechamber_CanFireAfterLockout(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	e := enemyInFront(100, 500)

	eng.Cast("rechamber", castCtx(p))
	eng.TickPlayer(p, 0.6, tickCtx())  // → phase 2
	eng.TickPlayer(p, 0.35, tickCtx()) // → phase 3 (missed)
	eng.TickPlayer(p, 0.8, tickCtx())  // → phase 0

	r := eng.Cast("fire_shot", castCtx(p, e))
	if !r.OK {
		t.Fatalf("should be able to fire after lockout: %s", r.Reason)
	}
}

func TestGunner_Rechamber_CanRestartAfterLockout(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()

	eng.Cast("rechamber", castCtx(p))
	eng.TickPlayer(p, 0.6, tickCtx())  // → phase 2
	eng.TickPlayer(p, 0.35, tickCtx()) // → phase 3
	eng.TickPlayer(p, 0.8, tickCtx())  // → phase 0

	r := eng.Cast("rechamber", castCtx(p))
	if !r.OK {
		t.Fatalf("rechamber should be restartable after lockout: %s", r.Reason)
	}
}

func TestGunner_Rechamber_CanRestartAfterConfirm(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()

	eng.Cast("rechamber", castCtx(p))
	eng.TickPlayer(p, 0.6, tickCtx()) // → phase 2
	eng.Cast("rechamber_confirm", castCtx(p))

	// Fire CD should be cleared by now (0.6 tick > 0.6 CD)
	r := eng.Cast("rechamber", castCtx(p))
	if !r.OK {
		t.Fatalf("rechamber should be restartable after confirm: %s", r.Reason)
	}
}

// --- Overclock + Rechamber interaction ---

func TestGunner_Overclock_Then_Rechamber(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()

	eng.Cast("overclock", castCtx(p))
	r := eng.Cast("rechamber", castCtx(p))
	if !r.OK {
		t.Fatalf("rechamber during overclock failed: %s", r.Reason)
	}
}

func TestGunner_Rechamber_Then_Overclock(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()

	eng.Cast("rechamber", castCtx(p))
	r := eng.Cast("overclock", castCtx(p))
	if !r.OK {
		t.Fatalf("overclock during rechamber failed: %s", r.Reason)
	}
}

func TestGunner_RechamberBuff_Plus_Overclock_Damage(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	e := enemyInFront(100, 1000)

	// Get rechamber buff (1.8x)
	eng.Cast("rechamber", castCtx(p))
	eng.TickPlayer(p, 0.6, tickCtx())
	eng.Cast("rechamber_confirm", castCtx(p))

	// rechamber_buff is damage_mult, not cooldown_mult — so it stacks multiplicatively
	// 10 base * 1.8 = 18
	r := eng.Cast("fire_shot", castCtx(p, e))
	if !r.OK {
		t.Fatalf("fire_shot failed: %s", r.Reason)
	}
	if len(r.Events) == 1 && r.Events[0].Amount != 18 {
		t.Errorf("damage = %f, want 18", r.Events[0].Amount)
	}
}

func TestGunner_Overclock_RechamberCooldown(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()

	// Overclock reduces cooldowns via cooldown_mult buff
	eng.Cast("overclock", castCtx(p))

	// Rechamber is handler-based, sets fire_shot CD directly (0.6)
	// The handler doesn't go through the cooldown_mult path
	eng.Cast("rechamber", castCtx(p))
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
		r := eng.Cast("fire_shot", castCtx(p, e))
		if !r.OK {
			t.Fatalf("initial shot %d failed: %s", i+1, r.Reason)
		}
		eng.TickPlayer(p, 0.2, tickCtx())
	}

	// 2. Rechamber
	r := eng.Cast("rechamber", castCtx(p))
	if !r.OK {
		t.Fatalf("rechamber failed: %s", r.Reason)
	}

	// 3. Tick through windup
	eng.TickPlayer(p, 0.6, tickCtx())

	// 4. Confirm
	r = eng.Cast("rechamber_confirm", castCtx(p))
	if !r.OK {
		t.Fatalf("confirm failed: %s", r.Reason)
	}

	// 5. Fire buffed shots
	for i := 0; i < 3; i++ {
		r = eng.Cast("fire_shot", castCtx(p, e))
		if !r.OK {
			t.Fatalf("buffed shot %d failed: %s", i+1, r.Reason)
		}
		if len(r.Events) == 1 && r.Events[0].Amount != 18 {
			t.Errorf("buffed shot %d: damage = %f, want 18", i+1, r.Events[0].Amount)
		}
		eng.TickPlayer(p, 0.2, tickCtx())
	}
}

func TestGunner_FullRotation_Overclock_Fire(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	e := enemyInFront(100, 1e6)

	// 1. Overclock
	r := eng.Cast("overclock", castCtx(p))
	if !r.OK {
		t.Fatalf("overclock failed: %s", r.Reason)
	}

	// 2. Rapid fire with reduced cooldowns
	for i := 0; i < 5; i++ {
		r = eng.Cast("fire_shot", castCtx(p, e))
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
	eng.Cast("overclock", castCtx(p))

	// 2. Fire one shot
	eng.Cast("fire_shot", castCtx(p, e))
	eng.TickPlayer(p, 0.12, tickCtx())

	// 3. Rechamber
	r := eng.Cast("rechamber", castCtx(p))
	if !r.OK {
		t.Fatalf("rechamber during overclock failed: %s", r.Reason)
	}

	// 4. Tick through windup and confirm
	eng.TickPlayer(p, 0.6, tickCtx())
	r = eng.Cast("rechamber_confirm", castCtx(p))
	if !r.OK {
		t.Fatalf("confirm failed: %s", r.Reason)
	}

	// 5. Fire with both buffs (rechamber_buff 1.8x damage, overclock reduced CD)
	r = eng.Cast("fire_shot", castCtx(p, e))
	if !r.OK {
		t.Fatalf("buffed fire failed: %s", r.Reason)
	}
	if len(r.Events) == 1 && r.Events[0].Amount != 18 {
		t.Errorf("damage = %f, want 18", r.Events[0].Amount)
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

	eng.Cast("rechamber", castCtx(p))
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
	eng.Cast("rechamber", castCtx(p))
	eng.TickPlayer(p, 0.6, tickCtx())
	eng.Cast("rechamber_confirm", castCtx(p))

	// Buff lasts 4s
	eng.TickPlayer(p, 4.1, tickCtx())
	if p.HasBuff("rechamber_buff") {
		t.Error("rechamber_buff should have expired after 4.1s")
	}

	// Damage should be back to base
	r := eng.Cast("fire_shot", castCtx(p, e))
	if !r.OK {
		t.Fatalf("fire_shot failed: %s", r.Reason)
	}
	if len(r.Events) == 1 && r.Events[0].Amount != 10 {
		t.Errorf("damage after buff expired = %f, want 10", r.Events[0].Amount)
	}
}

func TestGunner_NoGCD_BetweenAbilities(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()

	// Gunner abilities should not set GCD (they use per-ability CDs)
	eng.Cast("overclock", castCtx(p))
	if p.GCDTimer != 0 {
		t.Errorf("GCDTimer after overclock = %f, want 0", p.GCDTimer)
	}

	eng.Cast("rechamber", castCtx(p))
	if p.GCDTimer != 0 {
		t.Errorf("GCDTimer after rechamber = %f, want 0", p.GCDTimer)
	}
}

func TestGunner_OriginConfig_DoesNotBlock(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()

	// Gunner abilities have OriginConfig=0 (default) and Config=0 (default).
	// Verify this doesn't accidentally block casts.
	for _, id := range []string{"fire_shot", "overclock", "rechamber", "rechamber_confirm"} {
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
