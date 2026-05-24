package ability

import (
	"math"
	"testing"
)

func TestBDGuard_BuffExpires(t *testing.T) {
	eng := NewEngine(nil)
	p := newBladeDancer()

	eng.Commit("bd_guard", commitCtx(p))
	if !p.HasBuff("guard") {
		t.Fatal("guard buff should be applied")
	}

	eng.TickPlayer(p, 1.6, tickCtx())
	if p.HasBuff("guard") {
		t.Error("guard buff should have expired after 1.6s")
	}
}

func TestBDGuard_NoDamageEvents(t *testing.T) {
	eng := NewEngine(nil)
	p := newBladeDancer()
	e := enemyInFront(100, 500)

	r := eng.Commit("bd_guard", commitCtx(p, e))
	if !r.OK {
		t.Fatalf("commit failed: %s", r.Reason)
	}
	if len(r.Events) != 0 {
		t.Errorf("events = %d, want 0 (HitNone)", len(r.Events))
	}
}

func TestBDGuard_ReducesDamage(t *testing.T) {
	eng := NewEngine(nil)
	p := newBladeDancer()

	eng.Commit("bd_guard", commitCtx(p))

	dealt := p.ApplyDamage(100)
	expected := float32(50.0) // 100 * 0.5
	if math.Abs(float64(dealt-expected)) > 0.1 {
		t.Errorf("dealt = %f, want %f (50%% guard)", dealt, expected)
	}
}

func TestBDGuard_CanRecast(t *testing.T) {
	eng := NewEngine(nil)
	p := newBladeDancer()

	r1 := eng.Commit("bd_guard", commitCtx(p))
	if !r1.OK {
		t.Fatalf("first commit failed: %s", r1.Reason)
	}

	// bd_guard has no cooldown or GCD, should be re-committable
	r2 := eng.Commit("bd_guard", commitCtx(p))
	if !r2.OK {
		t.Fatalf("second commit failed: %s", r2.Reason)
	}
}
