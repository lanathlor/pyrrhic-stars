package ability

import (
	"testing"
)

func TestBrace_RequiresBlocking(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()

	r := eng.Cast("brace", castCtx(p))
	if r.OK {
		t.Error("Brace should require active shield block")
	}
	if r.Reason != "not blocking" {
		t.Errorf("reason = %q, want \"not blocking\"", r.Reason)
	}
}

func TestBrace_SucceedsWhileBlocking(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()

	eng.Cast("vg_shield_block", castCtx(p))
	r := eng.Cast("brace", castCtx(p))
	if !r.OK {
		t.Fatalf("Brace while blocking failed: %s", r.Reason)
	}
	if !p.HasBuff("brace") {
		t.Error("brace buff should be applied")
	}
}

func TestBrace_DoesNotCancelBlock(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()

	eng.Cast("vg_shield_block", castCtx(p))
	eng.Cast("brace", castCtx(p))

	if !p.HasBuff("vg_shield_block") {
		t.Error("Brace should NOT cancel shield block")
	}
}

func TestBrace_ReducesStaminaDrain(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()

	eng.Cast("vg_shield_block", castCtx(p))
	eng.Cast("brace", castCtx(p))
	eng.TickPlayer(p, 0.2, tickCtx()) // expire parry

	initialStamina := p.GetResource("stamina")

	// Take 100 damage → drain uses pre-DR amount
	// With Brace: drain = 100 * 0.5 * 0.2 = 10
	p.ApplyDamage(100)

	stam := p.GetResource("stamina")
	drainWithBrace := initialStamina - stam
	expectedDrain := float32(100) * ShieldStaminaDrainFraction * BraceDrainReduction // pre-DR based

	if drainWithBrace < expectedDrain*0.8 || drainWithBrace > expectedDrain*1.2 {
		t.Errorf("stamina drain with brace = %f, want ~%f", drainWithBrace, expectedDrain)
	}
}

func TestBrace_SetsLockout(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()

	eng.Cast("vg_shield_block", castCtx(p))
	eng.Cast("brace", castCtx(p))

	if p.GCDTimer < 3.0 {
		t.Errorf("GCD = %f, want >= 3.0 (brace lockout)", p.GCDTimer)
	}
}

func TestBrace_SetsCooldown(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()

	eng.Cast("vg_shield_block", castCtx(p))
	eng.Cast("brace", castCtx(p))

	if p.Cooldowns["brace"] < 11.9 {
		t.Errorf("cooldown = %f, want ~12.0", p.Cooldowns["brace"])
	}
}

func TestBrace_CooldownPreventsRecast(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()

	eng.Cast("vg_shield_block", castCtx(p))
	eng.Cast("brace", castCtx(p))

	// Clear GCD, keep cooldown
	p.GCDTimer = 0
	p.RemoveBuff("brace")

	r := eng.Cast("brace", castCtx(p))
	if r.OK {
		t.Error("should not recast during cooldown")
	}
}

func TestBrace_BuffExpiresNaturally(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()

	eng.Cast("vg_shield_block", castCtx(p))
	eng.Cast("brace", castCtx(p))

	// Tick past brace duration (3.5s)
	eng.TickPlayer(p, 4.0, tickCtx())

	if p.HasBuff("brace") {
		t.Error("brace buff should have expired after 3.5s")
	}
}
