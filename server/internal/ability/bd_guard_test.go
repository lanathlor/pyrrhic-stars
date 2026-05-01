package ability

import (
	"math"
	"testing"
)

func TestBDGuard_BuffExpires(t *testing.T) {
	eng := NewEngine()
	p := newBladeDancer()

	eng.Cast("bd_guard", castCtx(p))
	if !p.HasBuff("guard") {
		t.Fatal("guard buff should be applied")
	}

	eng.TickPlayer(p, 1.6, tickCtx())
	if p.HasBuff("guard") {
		t.Error("guard buff should have expired after 1.6s")
	}
}

func TestBDGuard_NoDamageEvents(t *testing.T) {
	eng := NewEngine()
	p := newBladeDancer()
	e := enemyInFront(100, 500)

	r := eng.Cast("bd_guard", castCtx(p, e))
	if !r.OK {
		t.Fatalf("cast failed: %s", r.Reason)
	}
	if len(r.Events) != 0 {
		t.Errorf("events = %d, want 0 (HitNone)", len(r.Events))
	}
}

func TestBDGuard_ReducesDamage(t *testing.T) {
	eng := NewEngine()
	p := newBladeDancer()

	eng.Cast("bd_guard", castCtx(p))

	dealt := p.ApplyDamage(100)
	expected := float32(50.0) // 100 * 0.5
	if math.Abs(float64(dealt-expected)) > 0.1 {
		t.Errorf("dealt = %f, want %f (50%% guard)", dealt, expected)
	}
}

func TestBDGuard_CanRecast(t *testing.T) {
	eng := NewEngine()
	p := newBladeDancer()

	r1 := eng.Cast("bd_guard", castCtx(p))
	if !r1.OK {
		t.Fatalf("first cast failed: %s", r1.Reason)
	}

	// bd_guard has no cooldown or GCD, should be re-castable
	r2 := eng.Cast("bd_guard", castCtx(p))
	if !r2.OK {
		t.Fatalf("second cast failed: %s", r2.Reason)
	}
}
