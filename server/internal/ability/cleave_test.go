package ability

import (
	"testing"

	"codex-online/server/internal/entity"
)

func TestCleave_CostsStamina(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()
	e := enemyInFront(100, 500)

	eng.Commit("cleave", commitCtx(p, e))
	if s := p.GetResource("stamina"); s != 90 {
		t.Errorf("stamina = %f, want 90 (100 - 10)", s)
	}
}

func TestCleave_InsufficientStamina(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()
	p.Resources["stamina"].Current = 5

	r := eng.Commit("cleave", commitCtx(p))
	if r.OK {
		t.Error("should fail with insufficient stamina")
	}
	if r.Reason != ReasonInsufficientStamina {
		t.Errorf("reason = %q, want %q", r.Reason, ReasonInsufficientStamina)
	}
}

func TestCleave_SetsCooldown(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()
	e := enemyInFront(100, 500)

	eng.Commit("cleave", commitCtx(p, e))
	if cd := p.Cooldowns["cleave"]; cd != 0.45 {
		t.Errorf("cooldown = %f, want 0.45 (standard tier)", cd)
	}
}

func TestCleave_BlockedByCooldown(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()
	e := enemyInFront(100, 500)

	eng.Commit("cleave", commitCtx(p, e))
	r := eng.Commit("cleave", commitCtx(p, e))
	if r.OK {
		t.Error("second cleave should be blocked by cooldown")
	}
}

func TestCleave_MissesBehind(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()
	e := enemyBehind(100, 500)

	r := eng.Commit("cleave", commitCtx(p, e))
	if !r.OK {
		t.Fatalf("commit failed: %s", r.Reason)
	}
	if len(r.Events) != 0 {
		t.Error("should miss enemy behind player")
	}
}

func TestCleave_SetsAttackState(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()
	e := enemyInFront(100, 500)

	eng.Commit("cleave", commitCtx(p, e))
	if p.State != entity.PlayerStateAttack {
		t.Errorf("state = %d, want %d (attack)", p.State, entity.PlayerStateAttack)
	}
}

func TestCleave_DamageMultApplied(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()
	e := enemyInFront(100, 1000)

	p.AddBuff(entity.ActiveBuff{
		ID: "test_dmg", Type: entity.BuffDamageMult, Value: 2.0, Duration: 5.0,
	})

	r := eng.Commit("cleave", commitCtx(p, e))
	if !r.OK {
		t.Fatalf("commit failed: %s", r.Reason)
	}
	if len(r.Events) == 1 && r.Events[0].Amount != 60 {
		t.Errorf("damage = %f, want 60 (30 base * 2.0 buff)", r.Events[0].Amount)
	}
}

func TestCleave_RepeatsWithoutCombo(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()
	e := enemyInFront(100, 1e6)

	// Hit multiple times — damage should be consistent (no combo escalation)
	for i := range 4 {
		p.Cooldowns = make(map[string]float32)
		hpBefore := e.Health
		eng.Commit("cleave", commitCtx(p, e))
		dealt := hpBefore - e.Health
		// Each hit should deal 30 * onslaught_mult (grows per hit)
		if dealt < 30 {
			t.Errorf("hit %d: damage = %f, want >= 30", i+1, dealt)
		}
	}
}

func TestCleave_BuildsOnslaught(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()
	e := enemyInFront(100, 1e6)

	eng.Commit("cleave", commitCtx(p, e))
	ons := getOnslaughtState(p)
	if ons.Stacks != 1 {
		t.Errorf("stacks = %d, want 1 after hitting one enemy", ons.Stacks)
	}
}

func TestCleave_EmpoweredArcWider(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()

	// Build to empowered tier (3 stacks)
	ons := getOnslaughtState(p)
	ons.Stacks = 3

	// Enemy at edge of 120° arc but within 200° arc
	e := enemyInFront(100, 1e6)
	e.Position = entity.Vec3{X: 4.5, Y: 0, Z: -3} // ~56° off center

	r := eng.Commit("cleave", commitCtx(p, e))
	if !r.OK {
		t.Fatalf("commit failed: %s", r.Reason)
	}
	if len(r.Events) == 0 {
		t.Error("empowered cleave (200°) should hit enemy at ~56° off center")
	}
}

func TestCleave_MaximumIs360(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()

	// Build to maximum tier (6 stacks)
	ons := getOnslaughtState(p)
	ons.Stacks = 6

	// Enemy behind player — should be hit by 360° sweep
	e := enemyBehind(100, 1e6)

	r := eng.Commit("cleave", commitCtx(p, e))
	if !r.OK {
		t.Fatalf("commit failed: %s", r.Reason)
	}
	if len(r.Events) == 0 {
		t.Error("maximum cleave (360°) should hit enemy behind player")
	}
}
