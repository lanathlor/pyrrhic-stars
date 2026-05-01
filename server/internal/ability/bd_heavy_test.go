package ability

import (
	"testing"

	"codex-online/server/internal/entity"
)

func TestBDHeavy_BasicHit(t *testing.T) {
	eng := NewEngine()
	p := newBladeDancer()
	e := enemyInFront(100, 500)

	r := eng.Cast("bd_heavy", castCtx(p, e))
	if !r.OK {
		t.Fatalf("bd_heavy failed: %s", r.Reason)
	}
	if len(r.Events) != 1 {
		t.Fatalf("events = %d, want 1", len(r.Events))
	}
	if r.Events[0].Amount != 35 {
		t.Errorf("damage = %f, want 35", r.Events[0].Amount)
	}
}

func TestBDHeavy_SetsCooldown(t *testing.T) {
	eng := NewEngine()
	p := newBladeDancer()
	e := enemyInFront(100, 500)

	eng.Cast("bd_heavy", castCtx(p, e))
	if cd := p.Cooldowns["bd_heavy"]; cd != 0.5 {
		t.Errorf("cooldown = %f, want 0.5", cd)
	}
}

func TestBDHeavy_BlockedByCooldown(t *testing.T) {
	eng := NewEngine()
	p := newBladeDancer()
	e := enemyInFront(100, 500)

	eng.Cast("bd_heavy", castCtx(p, e))
	r := eng.Cast("bd_heavy", castCtx(p, e))
	if r.OK {
		t.Error("second bd_heavy should be blocked by cooldown")
	}
}

func TestBDHeavy_MissesBehind(t *testing.T) {
	eng := NewEngine()
	p := newBladeDancer()
	e := enemyBehind(100, 500)

	r := eng.Cast("bd_heavy", castCtx(p, e))
	if !r.OK {
		t.Fatalf("cast failed: %s", r.Reason)
	}
	if len(r.Events) != 0 {
		t.Error("should miss enemy behind")
	}
}

func TestBDHeavy_DamageMultApplied(t *testing.T) {
	eng := NewEngine()
	p := newBladeDancer()
	e := enemyInFront(100, 1000)

	p.AddBuff(entity.ActiveBuff{
		ID: "test_dmg", Type: entity.BuffDamageMult, Value: 2.0, Duration: 5.0,
	})

	r := eng.Cast("bd_heavy", castCtx(p, e))
	if !r.OK {
		t.Fatalf("cast failed: %s", r.Reason)
	}
	if len(r.Events) == 1 && r.Events[0].Amount != 70 {
		t.Errorf("damage = %f, want 70 (35 base * 2.0 buff)", r.Events[0].Amount)
	}
}
