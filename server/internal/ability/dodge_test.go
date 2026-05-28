package ability

import "testing"

func TestDodge_CastsSuccessfully(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard() // has stamina

	r := eng.Commit(IDDodge, commitCtx(p))
	if !r.OK {
		t.Fatalf("dodge failed: %s", r.Reason)
	}
}

func TestDodge_CostsStamina(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()

	eng.Commit(IDDodge, commitCtx(p))
	if s := p.GetResource("stamina"); s != 80 {
		t.Errorf("stamina = %f, want 80 (100 - 20)", s)
	}
}

func TestDodge_InsufficientStamina(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()
	p.Resources["stamina"].Current = 10

	r := eng.Commit(IDDodge, commitCtx(p))
	if r.OK {
		t.Error("dodge should fail with insufficient stamina")
	}
	if r.Reason != ReasonInsufficientStamina {
		t.Errorf("reason = %q, want %q", r.Reason, ReasonInsufficientStamina)
	}
}

func TestDodge_NoDamageEvents(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()
	e := enemyInFront(100, 500)

	r := eng.Commit(IDDodge, commitCtx(p, e))
	if !r.OK {
		t.Fatalf("dodge failed: %s", r.Reason)
	}
	if len(r.Events) != 0 {
		t.Errorf("events = %d, want 0 (HitNone)", len(r.Events))
	}
}

func TestDodge_NoStaminaResource(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner() // gunner has no stamina

	r := eng.Commit(IDDodge, commitCtx(p))
	if r.OK {
		t.Error("dodge should fail without stamina resource")
	}
}
