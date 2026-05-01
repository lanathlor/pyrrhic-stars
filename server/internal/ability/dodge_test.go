package ability

import "testing"

func TestDodge_CastsSuccessfully(t *testing.T) {
	eng := NewEngine()
	p := newVanguard() // has stamina

	r := eng.Cast("dodge", castCtx(p))
	if !r.OK {
		t.Fatalf("dodge failed: %s", r.Reason)
	}
}

func TestDodge_CostsStamina(t *testing.T) {
	eng := NewEngine()
	p := newVanguard()

	eng.Cast("dodge", castCtx(p))
	if s := p.GetResource("stamina"); s != 80 {
		t.Errorf("stamina = %f, want 80 (100 - 20)", s)
	}
}

func TestDodge_InsufficientStamina(t *testing.T) {
	eng := NewEngine()
	p := newVanguard()
	p.Resources["stamina"].Current = 10

	r := eng.Cast("dodge", castCtx(p))
	if r.OK {
		t.Error("dodge should fail with insufficient stamina")
	}
	if r.Reason != "insufficient stamina" {
		t.Errorf("reason = %q, want %q", r.Reason, "insufficient stamina")
	}
}

func TestDodge_NoDamageEvents(t *testing.T) {
	eng := NewEngine()
	p := newVanguard()
	e := enemyInFront(100, 500)

	r := eng.Cast("dodge", castCtx(p, e))
	if !r.OK {
		t.Fatalf("dodge failed: %s", r.Reason)
	}
	if len(r.Events) != 0 {
		t.Errorf("events = %d, want 0 (HitNone)", len(r.Events))
	}
}

func TestDodge_NoStaminaResource(t *testing.T) {
	eng := NewEngine()
	p := newGunner() // gunner has no stamina

	r := eng.Cast("dodge", castCtx(p))
	if r.OK {
		t.Error("dodge should fail without stamina resource")
	}
}
