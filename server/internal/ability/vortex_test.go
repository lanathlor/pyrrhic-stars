package ability

import (
	"testing"

	"codex-online/server/internal/entity"
)

func TestVortex_InsufficientStamina(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()
	p.Resources["stamina"].Current = 10

	r := eng.Commit("vortex", commitCtx(p))
	if r.OK {
		t.Error("should fail with insufficient stamina")
	}
	if r.Reason != ReasonInsufficientStamina {
		t.Errorf("reason = %q, want %q", r.Reason, ReasonInsufficientStamina)
	}
}

func TestVortex_BlockedByGCD(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()
	p.GCDTimer = 0.5

	r := eng.Commit("vortex", commitCtx(p))
	if r.OK {
		t.Error("should be blocked by GCD")
	}
}

func TestVortex_BlockedByCooldown(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()
	e := enemyInFront(100, 1e6)

	eng.Commit("vortex", commitCtx(p, e))

	// Reset GCD but keep cooldown
	p.GCDTimer = 0
	p.Resources["stamina"].Current = 100

	r := eng.Commit("vortex", commitCtx(p, e))
	if r.OK {
		t.Error("should be blocked by cooldown")
	}
}

func TestVortex_OutOfRange(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()
	e := enemyInFront(100, 500)
	e.Position = entity.Vec3{X: 0, Y: 0, Z: -20} // far outside radius

	r := eng.Commit("vortex", commitCtx(p, e))
	if !r.OK {
		t.Fatalf("commit failed: %s", r.Reason)
	}
	if len(r.Events) != 0 {
		t.Error("should not hit enemy out of AoE radius")
	}
}

func TestVortex_StandardTier_TwoHits(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()
	e := enemyInFront(1000, 1e6)
	e.Position = entity.Vec3{X: 0, Y: 0, Z: -3}

	// Commit — first hit immediate
	r := eng.Commit("vortex", commitCtx(p, e))
	if !r.OK {
		t.Fatalf("commit failed: %s", r.Reason)
	}
	if len(r.Events) == 0 {
		t.Fatal("expected initial hit on commit")
	}
	hpAfterCast := e.Health

	// Standard tier: duration=0.6s, 2 hits total.
	// Second hit at 0.3s (interval = 0.6/2 = 0.3s).
	events := eng.TickPlayer(p, 0.35, tickCtx(e))
	if len(events) == 0 {
		t.Error("expected second hit tick")
	}
	hpAfterTick := e.Health
	if hpAfterTick >= hpAfterCast {
		t.Error("expected damage from tick")
	}

	// After duration ends, no more hits
	eng.TickPlayer(p, 0.3, tickCtx(e))
	hpAfterEnd := e.Health
	if hpAfterEnd != hpAfterTick {
		t.Error("expected no damage after vortex ends")
	}
}

func TestVortex_ThreatGenerated(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()
	e := enemyInFront(100, 1e6)
	e.Position = entity.Vec3{X: 0, Y: 0, Z: -3}

	eng.Commit("vortex", commitCtx(p, e))
	if e.ThreatTable[p.ID] <= 0 {
		t.Error("expected threat on enemy from vortex")
	}
}

func TestVortex_SetsCooldown(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()

	eng.Commit("vortex", commitCtx(p))
	if cd := p.Cooldowns["vortex"]; cd != 10.0 {
		t.Errorf("cooldown = %f, want 10.0", cd)
	}
}

func TestVortex_BuildsOnslaught(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()
	e := enemyInFront(1000, 1e6)
	e.Position = entity.Vec3{X: 0, Y: 0, Z: -3}

	eng.Commit("vortex", commitCtx(p, e))

	ons := getOnslaughtState(p)
	if ons.Stacks < 1 {
		t.Error("expected onslaught stacks from vortex hit")
	}
}

func TestVortex_EmpoweredTier_ThreeHits(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()
	e := enemyInFront(10000, 1e6)
	e.Position = entity.Vec3{X: 0, Y: 0, Z: -3}

	// Build to empowered tier (3 stacks)
	ons := getOnslaughtState(p)
	ons.Stacks = 3

	r := eng.Commit("vortex", commitCtx(p, e))
	if !r.OK {
		t.Fatalf("commit failed: %s", r.Reason)
	}

	// Empowered: duration=0.8s, 3 hits total.
	// Tick past interval (0.8/3 ≈ 0.267s) for second hit.
	events1 := eng.TickPlayer(p, 0.3, tickCtx(e))
	if len(events1) == 0 {
		t.Error("expected second hit at ~0.27s")
	}

	// Third hit
	events2 := eng.TickPlayer(p, 0.3, tickCtx(e))
	if len(events2) == 0 {
		t.Error("expected third hit at ~0.53s")
	}
}
