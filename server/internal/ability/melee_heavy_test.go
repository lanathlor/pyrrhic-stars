package ability

import (
	"testing"

	"codex-online/server/internal/entity"
)

func TestMeleeHeavy_BlockedByCooldown(t *testing.T) {
	eng := NewEngine()
	p := newVanguard()
	e := enemyInFront(100, 500)

	eng.Cast("melee_heavy", castCtx(p, e))
	r := eng.Cast("melee_heavy", castCtx(p, e))
	if r.OK {
		t.Error("second melee_heavy should be blocked by cooldown")
	}
}

func TestMeleeHeavy_InsufficientStamina(t *testing.T) {
	eng := NewEngine()
	p := newVanguard()
	p.Resources["stamina"].Current = 10

	r := eng.Cast("melee_heavy", castCtx(p))
	if r.OK {
		t.Error("should fail with insufficient stamina")
	}
	if r.Reason != "insufficient stamina" {
		t.Errorf("reason = %q, want %q", r.Reason, "insufficient stamina")
	}
}

func TestMeleeHeavy_MissesBehind(t *testing.T) {
	eng := NewEngine()
	p := newVanguard()
	e := enemyBehind(100, 500)

	r := eng.Cast("melee_heavy", castCtx(p, e))
	if !r.OK {
		t.Fatalf("cast failed: %s", r.Reason)
	}
	if len(r.Events) != 0 {
		t.Error("should miss enemy behind player")
	}
}

func TestMeleeHeavy_SetsAttackState(t *testing.T) {
	eng := NewEngine()
	p := newVanguard()
	e := enemyInFront(100, 500)

	eng.Cast("melee_heavy", castCtx(p, e))
	if p.State != entity.PlayerStateAttack {
		t.Errorf("state = %d, want %d (attack)", p.State, entity.PlayerStateAttack)
	}
}

func TestMeleeHeavy_DamageMultApplied(t *testing.T) {
	eng := NewEngine()
	p := newVanguard()
	e := enemyInFront(100, 1000)

	p.AddBuff(entity.ActiveBuff{
		ID: "test_dmg", Type: entity.BuffDamageMult, Value: 2.0, Duration: 5.0,
	})

	r := eng.Cast("melee_heavy", castCtx(p, e))
	if !r.OK {
		t.Fatalf("cast failed: %s", r.Reason)
	}
	if len(r.Events) == 1 && r.Events[0].Amount != 90 {
		t.Errorf("damage = %f, want 90 (45 base * 2.0 buff)", r.Events[0].Amount)
	}
}

func TestMeleeHeavy_SetsCooldown(t *testing.T) {
	eng := NewEngine()
	p := newVanguard()
	e := enemyInFront(100, 500)

	eng.Cast("melee_heavy", castCtx(p, e))
	if cd := p.Cooldowns["melee_heavy"]; cd != 0.8 {
		t.Errorf("cooldown = %f, want 0.8", cd)
	}
}
