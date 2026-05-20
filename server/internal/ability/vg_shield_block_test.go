package ability

import (
	"testing"

	"codex-online/server/internal/entity"
)

func TestShieldBlock_StartAppliesBuffsAndState(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()

	r := eng.Cast("vg_shield_block", castCtx(p))
	if !r.OK {
		t.Fatalf("cast failed: %s", r.Reason)
	}
	if !p.HasBuff("vg_shield_parry") {
		t.Error("shield parry buff should be applied")
	}
	if !p.HasBuff("vg_shield_block") {
		t.Error("shield block buff should be applied")
	}
	b := p.GetBuff("vg_shield_block")
	if b == nil || b.Value != shieldBlockDR {
		t.Errorf("block DR = %v, want %v", b.Value, shieldBlockDR)
	}
	if p.State != entity.PlayerStateBlock {
		t.Errorf("state = %d, want %d (block)", p.State, entity.PlayerStateBlock)
	}
}

func TestShieldBlock_ParryExpires(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()

	eng.Cast("vg_shield_block", castCtx(p))
	eng.TickPlayer(p, 0.15, tickCtx())
	if p.HasBuff("vg_shield_parry") {
		t.Error("shield parry should have expired after 0.15s")
	}
	if !p.HasBuff("vg_shield_block") {
		t.Error("shield block should still be active")
	}
}

func TestShieldBlock_ConstantDR(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()

	eng.Cast("vg_shield_block", castCtx(p))
	// Tick past parry, then well into block — DR should stay constant
	eng.TickPlayer(p, 0.5, tickCtx())
	eng.TickPlayer(p, 1.0, tickCtx())
	eng.TickPlayer(p, 1.0, tickCtx())

	b := p.GetBuff("vg_shield_block")
	if b == nil {
		t.Fatal("shield block buff should still be active")
	}
	if b.Value != shieldBlockDR {
		t.Errorf("DR = %f, want %f (constant, no decay)", b.Value, shieldBlockDR)
	}
}

func TestShieldBlock_NoPersecondStaminaDrain(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()
	initialStamina := p.GetResource("stamina")

	eng.Cast("vg_shield_block", castCtx(p))
	// Tick 1 second — Shield block should NOT drain stamina per-second
	for range 20 {
		eng.TickPlayer(p, 0.05, tickCtx())
	}

	stam := p.GetResource("stamina")
	// Stamina should be unchanged (no per-second drain; regen delay timer is reset on cast)
	if stam < initialStamina-1 {
		t.Errorf("stamina = %f, want ~%f (no per-second drain)", stam, initialStamina)
	}
}

func TestShieldBlock_DamageDrainsStamina(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()

	eng.Cast("vg_shield_block", castCtx(p))
	// Expire parry
	eng.TickPlayer(p, 0.2, tickCtx())

	initialStamina := p.GetResource("stamina")
	// Take 100 damage while blocking.
	// Stamina drain uses pre-DR amount: 100 * 0.5 (drain fraction) * 1.0 (tenacity) = 50
	p.ApplyDamage(100)

	stam := p.GetResource("stamina")
	expectedDrain := float32(100) * ShieldStaminaDrainFraction // pre-DR based
	expectedStam := initialStamina - expectedDrain
	if stam < expectedStam-2.0 || stam > expectedStam+2.0 {
		t.Errorf("stamina = %f, want ~%f (after blocking 100 damage)", stam, expectedStam)
	}
}

func TestShieldBlock_DamageGeneratesDevotion(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()

	eng.Cast("vg_shield_block", castCtx(p))
	eng.TickPlayer(p, 0.2, tickCtx()) // expire parry

	p.ApplyDamage(100)

	dev := getDevotionState(p)
	if dev.Charges <= 0 {
		t.Error("Devotion charges should increase from blocked damage")
	}
}

func TestShieldBlock_GuardParryReflect(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()
	enemy := enemyInFront(100, 500)

	eng.Cast("vg_shield_block", castCtx(p, enemy))

	// Parry is active — take damage to trigger reflect
	p.ApplyDamage(100)

	// Resolve reflect on next tick
	events := eng.TickPlayer(p, 0.01, tickCtx(enemy))
	if len(events) == 0 {
		t.Error("expected parry reflect damage events")
	}

	// Check bonus Devotion from parry (2x rate)
	dev := getDevotionState(p)
	if dev.Charges <= 0 {
		t.Error("Devotion should be generated from Guard Parry")
	}
}

func TestShieldBlock_GuardBreakOnStaminaDepletion(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()
	p.Resources["stamina"].Current = 1 // very low stamina

	eng.Cast("vg_shield_block", castCtx(p))
	eng.TickPlayer(p, 0.2, tickCtx()) // expire parry

	// Take heavy damage — should drain remaining stamina below 0
	p.ApplyDamage(200)

	// Next tick should trigger Guard Break
	eng.TickPlayer(p, 0.05, tickCtx())

	if p.HasBuff("vg_shield_block") {
		t.Error("shield block should have ended on Guard Break")
	}
	if p.State != entity.PlayerStateStagger {
		t.Errorf("state = %d, want %d (stagger)", p.State, entity.PlayerStateStagger)
	}
	if !p.HasBuff("guard_break") {
		t.Error("guard_break vulnerability buff should be applied")
	}
	b := p.GetBuff("guard_break")
	if b == nil || b.Value != guardBreakVulnMult {
		t.Errorf("guard_break value = %v, want %v", b.Value, guardBreakVulnMult)
	}
}

func TestShieldBlock_StopAction(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()

	eng.Cast("vg_shield_block", castCtx(p))
	r := eng.Cast("vg_shield_block_stop", castCtx(p))
	if !r.OK {
		t.Fatalf("stop failed: %s", r.Reason)
	}
	if p.HasBuff("vg_shield_block") {
		t.Error("shield block buff should be removed")
	}
	if p.Cooldowns["vg_shield_block"] < 1.4 || p.Cooldowns["vg_shield_block"] > 1.6 {
		t.Errorf("cooldown = %f, want 1.5", p.Cooldowns["vg_shield_block"])
	}
}

func TestShieldBlock_CooldownPreventsReblock(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()

	eng.Cast("vg_shield_block", castCtx(p))
	eng.Cast("vg_shield_block_stop", castCtx(p))

	r := eng.Cast("vg_shield_block", castCtx(p))
	if r.OK {
		t.Error("should not re-block during cooldown")
	}
}

func TestShieldBlock_CooldownExpires(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()

	eng.Cast("vg_shield_block", castCtx(p))
	eng.Cast("vg_shield_block_stop", castCtx(p))
	eng.TickPlayer(p, 2.0, tickCtx())

	r := eng.Cast("vg_shield_block", castCtx(p))
	if !r.OK {
		t.Fatalf("re-block after cooldown should succeed: %s", r.Reason)
	}
}

func TestShieldBlock_ZeroStaminaPreventsStart(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()
	p.Resources["stamina"].Current = 0

	r := eng.Cast("vg_shield_block", castCtx(p))
	if r.OK {
		t.Error("should not block with 0 stamina")
	}
}

func TestShieldBlock_DoesNotResetOnslaught(t *testing.T) {
	p := newShieldVanguard()
	// Shield spec should not have Onslaught, but verify damage doesn't panic
	p.ApplyDamage(50)
	// No onslaught state should exist
	if _, ok := p.AbilityState["onslaught"]; ok {
		t.Error("Shield spec should not have onslaught state")
	}
}
