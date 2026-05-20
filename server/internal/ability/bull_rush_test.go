package ability

import (
	"testing"

	"codex-online/server/internal/entity"
)

func TestBullRush_HitsEnemy(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()
	enemy := enemyInFront(100, 500)
	// Place enemy close enough for AoE
	enemy.Position = entity.Vec3{X: 0, Y: 0, Z: -3}

	r := eng.Cast("bull_rush", castCtx(p, enemy))
	if !r.OK {
		t.Fatalf("cast failed: %s", r.Reason)
	}
	if len(r.Events) == 0 {
		t.Error("expected damage events")
	}
}

func TestBullRush_DropsGuard(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()

	eng.Cast("vg_shield_block", castCtx(p))
	if !p.HasBuff("vg_shield_block") {
		t.Fatal("should be blocking")
	}

	eng.Cast("bull_rush", castCtx(p))
	if p.HasBuff("vg_shield_block") {
		t.Error("Bull Rush should cancel shield block")
	}
}

func TestBullRush_AppliesRootDebuff(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()
	enemy := enemyInFront(100, 500)
	enemy.Position = entity.Vec3{X: 0, Y: 0, Z: -3}

	eng.Cast("bull_rush", castCtx(p, enemy))

	if !enemy.HasDebuff(entity.DebuffRoot) {
		t.Error("enemy should have root debuff from Bull Rush knockback")
	}
}

func TestBullRush_CostsStamina(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()
	initial := p.GetResource("stamina")

	eng.Cast("bull_rush", castCtx(p))

	if p.GetResource("stamina") >= initial {
		t.Error("stamina should decrease after Bull Rush")
	}
}

func TestBullRush_SetsCooldown(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()

	eng.Cast("bull_rush", castCtx(p))

	if p.Cooldowns["bull_rush"] < 7.9 {
		t.Errorf("cooldown = %f, want ~8.0", p.Cooldowns["bull_rush"])
	}
}

func TestBullRush_CooldownPreventsRecast(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()

	eng.Cast("bull_rush", castCtx(p))
	// Clear GCD so only cooldown blocks
	p.GCDTimer = 0

	r := eng.Cast("bull_rush", castCtx(p))
	if r.OK {
		t.Error("should not cast during cooldown")
	}
}

func TestBullRush_InsufficientStamina(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()
	p.Resources["stamina"].Current = 0

	r := eng.Cast("bull_rush", castCtx(p))
	if r.OK {
		t.Error("should fail with 0 stamina")
	}
}
