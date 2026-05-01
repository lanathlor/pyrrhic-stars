package ability

import (
	"testing"

	"codex-online/server/internal/entity"
)

func TestBDMelee_BasicHit(t *testing.T) {
	eng := NewEngine()
	p := newBladeDancer()
	e := enemyInFront(100, 500)

	r := eng.Cast("bd_melee", castCtx(p, e))
	if !r.OK {
		t.Fatalf("bd_melee failed: %s", r.Reason)
	}
	if len(r.Events) != 1 {
		t.Fatalf("events = %d, want 1", len(r.Events))
	}
	if r.Events[0].Amount != 25 {
		t.Errorf("damage = %f, want 25", r.Events[0].Amount)
	}
}

func TestBDMelee_SetsCooldown(t *testing.T) {
	eng := NewEngine()
	p := newBladeDancer()
	e := enemyInFront(100, 500)

	eng.Cast("bd_melee", castCtx(p, e))
	if cd := p.Cooldowns["bd_melee"]; cd != 0.3 {
		t.Errorf("cooldown = %f, want 0.3", cd)
	}
}

func TestBDMelee_BlockedByCooldown(t *testing.T) {
	eng := NewEngine()
	p := newBladeDancer()
	e := enemyInFront(100, 500)

	eng.Cast("bd_melee", castCtx(p, e))
	r := eng.Cast("bd_melee", castCtx(p, e))
	if r.OK {
		t.Error("second bd_melee should be blocked by cooldown")
	}
}

func TestBDMelee_MissesBehind(t *testing.T) {
	eng := NewEngine()
	p := newBladeDancer()
	e := enemyBehind(100, 500)

	r := eng.Cast("bd_melee", castCtx(p, e))
	if !r.OK {
		t.Fatalf("cast failed: %s", r.Reason)
	}
	if len(r.Events) != 0 {
		t.Error("should miss enemy behind")
	}
}

func TestBDMelee_DamageMultApplied(t *testing.T) {
	eng := NewEngine()
	p := newBladeDancer()
	e := enemyInFront(100, 1000)

	p.AddBuff(entity.ActiveBuff{
		ID: "test_dmg", Type: entity.BuffDamageMult, Value: 2.0, Duration: 5.0,
	})

	r := eng.Cast("bd_melee", castCtx(p, e))
	if !r.OK {
		t.Fatalf("cast failed: %s", r.Reason)
	}
	if len(r.Events) == 1 && r.Events[0].Amount != 50 {
		t.Errorf("damage = %f, want 50 (25 base * 2.0 buff)", r.Events[0].Amount)
	}
}

func TestBDMelee_SetsAttackState(t *testing.T) {
	eng := NewEngine()
	p := newBladeDancer()
	e := enemyInFront(100, 500)

	eng.Cast("bd_melee", castCtx(p, e))
	if p.State != entity.PlayerStateAttack {
		t.Errorf("state = %d, want %d (attack)", p.State, entity.PlayerStateAttack)
	}
}
