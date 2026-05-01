package ability

import (
	"testing"

	"codex-online/server/internal/entity"
)

func TestVGBlock_ParryExpires(t *testing.T) {
	eng := NewEngine()
	p := newVanguard()

	eng.Cast("vg_block", castCtx(p))
	if !p.HasBuff("vg_parry") {
		t.Fatal("parry buff should be applied")
	}

	// Tick past parry (0.15s)
	eng.TickPlayer(p, 0.2, tickCtx())
	if p.HasBuff("vg_parry") {
		t.Error("parry should have expired after 0.2s")
	}
	if !p.HasBuff("vg_block") {
		t.Error("block should still be active")
	}
}

func TestVGBlock_BlockExpires(t *testing.T) {
	eng := NewEngine()
	p := newVanguard()

	eng.Cast("vg_block", castCtx(p))

	eng.TickPlayer(p, 1.6, tickCtx())
	if p.HasBuff("vg_block") {
		t.Error("block should have expired after 1.6s")
	}
}

func TestVGBlock_DamageWhileBlocking(t *testing.T) {
	eng := NewEngine()
	p := newVanguard()

	eng.Cast("vg_block", castCtx(p))

	// Expire parry, keep block
	eng.TickPlayer(p, 0.2, tickCtx())

	// Take damage — block DR is 0.3 (70% blocked, 30% passes)
	dealt := p.ApplyDamage(100)
	if dealt < 29.9 || dealt > 30.1 {
		t.Errorf("dealt = %f, want ~30 (70%% block)", dealt)
	}
}

func TestVGBlock_DamageWhileParrying(t *testing.T) {
	eng := NewEngine()
	p := newVanguard()

	eng.Cast("vg_block", castCtx(p))

	// During parry window: both parry (0.0) and block (0.3) active
	// Multiplicative: 0.0 * 0.3 = 0 → full block
	dealt := p.ApplyDamage(100)
	if dealt != 0 {
		t.Errorf("dealt = %f, want 0 (full parry)", dealt)
	}
}

func TestVGBlock_SetsBlockState(t *testing.T) {
	eng := NewEngine()
	p := newVanguard()

	eng.Cast("vg_block", castCtx(p))
	if p.State != entity.PlayerStateBlock {
		t.Errorf("state = %d, want %d (block)", p.State, entity.PlayerStateBlock)
	}
}

func TestVGBlock_CanReblockAfterExpiry(t *testing.T) {
	eng := NewEngine()
	p := newVanguard()

	eng.Cast("vg_block", castCtx(p))
	eng.TickPlayer(p, 1.6, tickCtx()) // expire all buffs

	r := eng.Cast("vg_block", castCtx(p))
	if !r.OK {
		t.Fatalf("re-block after expiry failed: %s", r.Reason)
	}
}
