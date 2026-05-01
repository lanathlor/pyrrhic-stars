package ability

import (
	"testing"

	"codex-online/server/internal/entity"
)

func TestBladeSwirl_InsufficientStamina(t *testing.T) {
	eng := NewEngine()
	p := newVanguard()
	p.Resources["stamina"].Current = 10

	r := eng.Cast("blade_swirl", castCtx(p))
	if r.OK {
		t.Error("should fail with insufficient stamina")
	}
	if r.Reason != "insufficient stamina" {
		t.Errorf("reason = %q, want %q", r.Reason, "insufficient stamina")
	}
}

func TestBladeSwirl_BlockedByGCD(t *testing.T) {
	eng := NewEngine()
	p := newVanguard()
	p.GCDTimer = 0.5

	r := eng.Cast("blade_swirl", castCtx(p))
	if r.OK {
		t.Error("should be blocked by GCD")
	}
}

func TestBladeSwirl_BlockedByCooldown(t *testing.T) {
	eng := NewEngine()
	p := newVanguard()
	e := enemyInFront(100, 1e6)

	eng.Cast("blade_swirl", castCtx(p, e))

	// Reset GCD but keep cooldown
	p.GCDTimer = 0
	p.Resources["stamina"].Current = 100

	r := eng.Cast("blade_swirl", castCtx(p, e))
	if r.OK {
		t.Error("should be blocked by cooldown")
	}
}

func TestBladeSwirl_OutOfRange(t *testing.T) {
	eng := NewEngine()
	p := newVanguard()
	e := enemyInFront(100, 500)
	e.Position = entity.Vec3{X: 0, Y: 0, Z: -20} // far outside 6 radius

	r := eng.Cast("blade_swirl", castCtx(p, e))
	if !r.OK {
		t.Fatalf("cast failed: %s", r.Reason)
	}
	if len(r.Events) != 0 {
		t.Error("should not hit enemy out of AoE radius")
	}
}

func TestBladeSwirl_FullTickSequence(t *testing.T) {
	eng := NewEngine()
	p := newVanguard()
	e := enemyInFront(100, 1e6)
	e.Position = entity.Vec3{X: 0, Y: 0, Z: -3}

	eng.Cast("blade_swirl", castCtx(p, e))
	hpAfterCast := e.Health

	// Tick 1 at 0.5s
	events1 := eng.TickPlayer(p, 0.5, tickCtx(e))
	if len(events1) == 0 {
		t.Error("expected tick 1 at 0.5s")
	}

	// Tick 2 at 1.0s
	events2 := eng.TickPlayer(p, 0.5, tickCtx(e))
	if len(events2) == 0 {
		t.Error("expected tick 2 at 1.0s")
	}

	// After 1.5s total, swirl ends — no more ticks
	eng.TickPlayer(p, 0.5, tickCtx(e))
	hpAfterSwirl := e.Health

	// Should have dealt damage from cast + 2 ticks (3 total AoE hits including cast)
	totalDamage := hpAfterCast - hpAfterSwirl
	if totalDamage <= 0 {
		t.Error("expected tick damage during swirl")
	}
}

func TestBladeSwirl_ThreatGenerated(t *testing.T) {
	eng := NewEngine()
	p := newVanguard()
	e := enemyInFront(100, 1e6)
	e.Position = entity.Vec3{X: 0, Y: 0, Z: -3}

	eng.Cast("blade_swirl", castCtx(p, e))
	if e.ThreatTable[p.PeerID] <= 0 {
		t.Error("expected threat on enemy from blade_swirl")
	}
}

func TestBladeSwirl_SetsCooldown(t *testing.T) {
	eng := NewEngine()
	p := newVanguard()

	eng.Cast("blade_swirl", castCtx(p))
	if cd := p.Cooldowns["blade_swirl"]; cd != 10.0 {
		t.Errorf("cooldown = %f, want 10.0", cd)
	}
}
