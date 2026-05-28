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

	r := eng.Commit("bull_rush", commitCtx(p, enemy))
	if !r.OK {
		t.Fatalf("commit failed: %s", r.Reason)
	}
	if len(r.Events) == 0 {
		t.Error("expected damage events")
	}
}

func TestBullRush_DropsGuard(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()

	eng.Commit(IDVgShieldBlock, commitCtx(p))
	if !p.HasBuff(IDVgShieldBlock) {
		t.Fatal("should be blocking")
	}

	eng.Commit("bull_rush", commitCtx(p))
	if p.HasBuff(IDVgShieldBlock) {
		t.Error("Bull Rush should cancel shield block")
	}
}

func TestBullRush_AppliesRootDebuff(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()
	enemy := enemyInFront(100, 500)
	enemy.Position = entity.Vec3{X: 0, Y: 0, Z: -3}

	eng.Commit("bull_rush", commitCtx(p, enemy))

	if !enemy.HasDebuff(entity.DebuffRoot) {
		t.Error("enemy should have root debuff from Bull Rush knockback")
	}
}

func TestBullRush_CostsStamina(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()
	initial := p.GetResource("stamina")

	eng.Commit("bull_rush", commitCtx(p))

	if p.GetResource("stamina") >= initial {
		t.Error("stamina should decrease after Bull Rush")
	}
}

func TestBullRush_SetsCooldown(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()

	eng.Commit("bull_rush", commitCtx(p))

	if p.Cooldowns["bull_rush"] < 7.9 {
		t.Errorf("cooldown = %f, want ~8.0", p.Cooldowns["bull_rush"])
	}
}

func TestBullRush_CooldownPreventsRecast(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()

	eng.Commit("bull_rush", commitCtx(p))
	// Clear GCD so only cooldown blocks
	p.GCDTimer = 0

	r := eng.Commit("bull_rush", commitCtx(p))
	if r.OK {
		t.Error("should not commit during cooldown")
	}
}

func TestBullRush_InsufficientStamina(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()
	p.Resources["stamina"].Current = 0

	r := eng.Commit("bull_rush", commitCtx(p))
	if r.OK {
		t.Error("should fail with 0 stamina")
	}
}
