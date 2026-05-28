package ability

import (
	"testing"

	"codex-online/server/internal/entity"
)

func TestVGBlock_StartAppliesBuffsAndState(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()

	r := eng.Commit(IDVgBlock, commitCtx(p))
	if !r.OK {
		t.Fatalf("commit failed: %s", r.Reason)
	}
	if !p.HasBuff("vg_parry") {
		t.Error("parry buff should be applied")
	}
	if !p.HasBuff(IDVgBlock) {
		t.Error("block buff should be applied")
	}
	b := p.GetBuff(IDVgBlock)
	if b == nil || b.Value != blockDRStart {
		t.Errorf("block DR = %v, want %v", b.Value, blockDRStart)
	}
	if p.State != entity.PlayerStateBlock {
		t.Errorf("state = %d, want %d (block)", p.State, entity.PlayerStateBlock)
	}
}

func TestVGBlock_ParryExpires(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()

	eng.Commit(IDVgBlock, commitCtx(p))

	// Tick past parry window (0.15s)
	eng.TickPlayer(p, 0.2, tickCtx())
	if p.HasBuff("vg_parry") {
		t.Error("parry should have expired after 0.2s")
	}
	if !p.HasBuff(IDVgBlock) {
		t.Error("block should still be active")
	}
}

func TestVGBlock_DamageWhileParrying(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()

	eng.Commit(IDVgBlock, commitCtx(p))

	// During parry: parry(0.0) * block(0.2) = 0 → full block
	dealt := p.ApplyDamage(100)
	if dealt != 0 {
		t.Errorf("dealt = %f, want 0 (full parry)", dealt)
	}
}

func TestVGBlock_DamageAtStart(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()

	eng.Commit(IDVgBlock, commitCtx(p))
	// Expire parry with small ticks (4 x 0.05 = 0.2s)
	for range 4 {
		eng.TickPlayer(p, 0.05, tickCtx())
	}

	// Block DR has decayed slightly over 0.2s: Value = 0.2 + 0.2*0.2 = 0.227
	// So ~22-23% passes through
	dealt := p.ApplyDamage(100)
	if dealt < 22.0 || dealt > 28.0 {
		t.Errorf("dealt = %f, want ~23 (block DR at 0.2s elapsed)", dealt)
	}
}

func TestVGBlock_DRDecaysPartial(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()

	eng.Commit(IDVgBlock, commitCtx(p))
	// Tick 0.75s (halfway through decay) — Value should be ~0.35
	eng.TickPlayer(p, 0.75, tickCtx())

	b := p.GetBuff(IDVgBlock)
	if b == nil {
		t.Fatal("block buff missing")
	}
	// 0.2 + 0.2*0.75 = 0.35
	if b.Value < 0.34 || b.Value > 0.36 {
		t.Errorf("block Value = %f, want ~0.35 at 0.75s", b.Value)
	}
}

func TestVGBlock_DRDecaysFull(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()

	eng.Commit(IDVgBlock, commitCtx(p))
	// Tick 1.5s — Value should be 0.5 (floor)
	eng.TickPlayer(p, 1.5, tickCtx())

	b := p.GetBuff(IDVgBlock)
	if b == nil {
		t.Fatal("block buff missing")
	}
	if b.Value < 0.49 || b.Value > 0.51 {
		t.Errorf("block Value = %f, want ~0.5 at 1.5s", b.Value)
	}
}

func TestVGBlock_DRFloorsAt50(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()

	eng.Commit(IDVgBlock, commitCtx(p))
	// Tick well past decay time — Value should cap at 0.5
	eng.TickPlayer(p, 3.0, tickCtx())

	b := p.GetBuff(IDVgBlock)
	if b == nil {
		t.Fatal("block buff missing after 3s (stamina should still have some left)")
	}
	if b.Value < 0.49 || b.Value > 0.51 {
		t.Errorf("block Value = %f, want 0.5 (floor)", b.Value)
	}
}

func TestVGBlock_StaminaDrains(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()

	eng.Commit(IDVgBlock, commitCtx(p))
	// Use small ticks to avoid partial-regen within a single large dt
	for range 20 {
		eng.TickPlayer(p, 0.05, tickCtx())
	}

	stam := p.GetResource("stamina")
	// 100 - 15*1.0 = 85 (regen delay resets each tick, so no regen)
	if stam < 84.0 || stam > 86.0 {
		t.Errorf("stamina = %f, want ~85 after 1s block", stam)
	}
}

func TestVGBlock_StaminaDepletionEndsBlock(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()

	// Set low stamina
	p.Resources["stamina"].Current = 10

	eng.Commit(IDVgBlock, commitCtx(p))
	// 10 stamina / 15 per sec = ~0.67s → block should end within 1s
	for range 20 {
		eng.TickPlayer(p, 0.05, tickCtx())
	}

	if p.HasBuff(IDVgBlock) {
		t.Error("block should have ended when stamina depleted")
	}
	if p.GetResource("stamina") != 0 {
		t.Errorf("stamina = %f, want 0", p.GetResource("stamina"))
	}
	// Cooldown was set when block ended (~0.67s in), then ticked for remaining ~0.33s
	if p.Cooldowns[IDVgBlock] < 2.5 {
		t.Errorf("cooldown = %f, want >2.5", p.Cooldowns[IDVgBlock])
	}
	if p.State == entity.PlayerStateBlock {
		t.Error("state should no longer be block")
	}
}

func TestVGBlock_StopAction(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()

	eng.Commit(IDVgBlock, commitCtx(p))
	r := eng.Commit(IDVgBlockStop, commitCtx(p))
	if !r.OK {
		t.Fatalf("stop failed: %s", r.Reason)
	}

	if p.HasBuff(IDVgBlock) {
		t.Error("block buff should be removed")
	}
	if p.HasBuff("vg_parry") {
		t.Error("parry buff should be removed")
	}
	if p.Cooldowns[IDVgBlock] < 2.9 || p.Cooldowns[IDVgBlock] > 3.1 {
		t.Errorf("cooldown = %f, want 3.0", p.Cooldowns[IDVgBlock])
	}
	if p.State == entity.PlayerStateBlock {
		t.Error("state should no longer be block")
	}
}

func TestVGBlock_CooldownPreventsReblock(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()

	eng.Commit(IDVgBlock, commitCtx(p))
	eng.Commit(IDVgBlockStop, commitCtx(p))

	r := eng.Commit(IDVgBlock, commitCtx(p))
	if r.OK {
		t.Error("should not be able to re-block during cooldown")
	}
	if r.Reason != ReasonCooldown {
		t.Errorf("reason = %q, want %q", r.Reason, ReasonCooldown)
	}
}

func TestVGBlock_CooldownExpires(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()

	eng.Commit(IDVgBlock, commitCtx(p))
	eng.Commit(IDVgBlockStop, commitCtx(p))

	// Tick past the 3s cooldown
	eng.TickPlayer(p, 3.1, tickCtx())

	r := eng.Commit(IDVgBlock, commitCtx(p))
	if !r.OK {
		t.Fatalf("re-block after cooldown expired failed: %s", r.Reason)
	}
}

func TestVGBlock_AlreadyBlocking(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()

	eng.Commit(IDVgBlock, commitCtx(p))
	r := eng.Commit(IDVgBlock, commitCtx(p))
	if r.OK {
		t.Error("should reject duplicate block start")
	}
	if r.Reason != "already blocking" {
		t.Errorf("reason = %q, want \"already blocking\"", r.Reason)
	}
}

func TestVGBlock_ZeroStaminaPreventsStart(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()
	p.Resources["stamina"].Current = 0

	r := eng.Commit(IDVgBlock, commitCtx(p))
	if r.OK {
		t.Error("should not block with 0 stamina")
	}
}

func TestVGBlock_StopWhenNotBlocking_NoOp(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()

	// Stop without starting — should be a no-op, not crash
	r := eng.Commit(IDVgBlockStop, commitCtx(p))
	if !r.OK {
		t.Errorf("stop when not blocking should succeed (no-op): %s", r.Reason)
	}
	// No cooldown should be set
	if _, ok := p.Cooldowns[IDVgBlock]; ok {
		t.Error("cooldown should not be set when stop is a no-op")
	}
}
