package ability

import (
	"testing"

	"codex-online/server/internal/entity"
)

func TestMeleeLight_CostsStamina(t *testing.T) {
	eng := NewEngine()
	p := newVanguard()
	e := enemyInFront(100, 500)

	eng.Cast("melee_light", castCtx(p, e))
	if s := p.GetResource("stamina"); s != 90 {
		t.Errorf("stamina = %f, want 90 (100 - 10)", s)
	}
}

func TestMeleeLight_InsufficientStamina(t *testing.T) {
	eng := NewEngine()
	p := newVanguard()
	p.Resources["stamina"].Current = 5

	r := eng.Cast("melee_light", castCtx(p))
	if r.OK {
		t.Error("should fail with insufficient stamina")
	}
	if r.Reason != "insufficient stamina" {
		t.Errorf("reason = %q, want %q", r.Reason, "insufficient stamina")
	}
}

func TestMeleeLight_SetsCooldown(t *testing.T) {
	eng := NewEngine()
	p := newVanguard()
	e := enemyInFront(100, 500)

	eng.Cast("melee_light", castCtx(p, e))
	if cd := p.Cooldowns["melee_light"]; cd != 0.55 {
		t.Errorf("cooldown = %f, want 0.55", cd)
	}
}

func TestMeleeLight_BlockedByCooldown(t *testing.T) {
	eng := NewEngine()
	p := newVanguard()
	e := enemyInFront(100, 500)

	eng.Cast("melee_light", castCtx(p, e))
	r := eng.Cast("melee_light", castCtx(p, e))
	if r.OK {
		t.Error("second melee_light should be blocked by cooldown")
	}
}

func TestMeleeLight_MissesBehind(t *testing.T) {
	eng := NewEngine()
	p := newVanguard()
	e := enemyBehind(100, 500)

	r := eng.Cast("melee_light", castCtx(p, e))
	if !r.OK {
		t.Fatalf("cast failed: %s", r.Reason)
	}
	if len(r.Events) != 0 {
		t.Error("should miss enemy behind player")
	}
}

func TestMeleeLight_SetsAttackState(t *testing.T) {
	eng := NewEngine()
	p := newVanguard()
	e := enemyInFront(100, 500)

	eng.Cast("melee_light", castCtx(p, e))
	if p.State != entity.PlayerStateAttack {
		t.Errorf("state = %d, want %d (attack)", p.State, entity.PlayerStateAttack)
	}
}

func TestMeleeLight_DamageMultApplied(t *testing.T) {
	eng := NewEngine()
	p := newVanguard()
	e := enemyInFront(100, 1000)

	p.AddBuff(entity.ActiveBuff{
		ID: "test_dmg", Type: entity.BuffDamageMult, Value: 2.0, Duration: 5.0,
	})

	r := eng.Cast("melee_light", castCtx(p, e))
	if !r.OK {
		t.Fatalf("cast failed: %s", r.Reason)
	}
	if len(r.Events) == 1 && r.Events[0].Amount != 60 {
		t.Errorf("damage = %f, want 60 (30 base * 2.0 buff)", r.Events[0].Amount)
	}
}

func TestMeleeLight_ComboWrapsAround(t *testing.T) {
	eng := NewEngine()
	p := newVanguard()
	e := enemyInFront(100, 1e6)

	// Do a full combo (3 steps) then verify it wraps
	for i := 0; i < 3; i++ {
		p.Cooldowns = make(map[string]float32)
		eng.Cast("melee_light", castCtx(p, e))
	}

	// 4th hit should be back to step 0 (30 damage)
	hpBefore := e.Health
	p.Cooldowns = make(map[string]float32)
	eng.Cast("melee_light", castCtx(p, e))
	dealt := hpBefore - e.Health
	if dealt != 30 {
		t.Errorf("4th combo hit = %f, want 30 (wrapped to step 0)", dealt)
	}
}
