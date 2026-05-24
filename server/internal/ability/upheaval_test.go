package ability

import (
	"testing"

	"codex-online/server/internal/entity"
)

func TestUpheaval_BlockedByCooldown(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()
	e := enemyInFront(100, 500)

	eng.Commit("upheaval", commitCtx(p, e))
	p.GCDTimer = 0 // clear GCD for recast attempt
	r := eng.Commit("upheaval", commitCtx(p, e))
	if r.OK {
		t.Error("second upheaval should be blocked by cooldown")
	}
}

func TestUpheaval_InsufficientStamina(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()
	p.Resources["stamina"].Current = 10

	r := eng.Commit("upheaval", commitCtx(p))
	if r.OK {
		t.Error("should fail with insufficient stamina")
	}
	if r.Reason != ReasonInsufficientStamina {
		t.Errorf("reason = %q, want %q", r.Reason, ReasonInsufficientStamina)
	}
}

func TestUpheaval_MissesBehind(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()
	e := enemyBehind(100, 500)

	r := eng.Commit("upheaval", commitCtx(p, e))
	if !r.OK {
		t.Fatalf("commit failed: %s", r.Reason)
	}
	if len(r.Events) != 0 {
		t.Error("should miss enemy behind player (outside 60° cone)")
	}
}

func TestUpheaval_SetsAttackState(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()
	e := enemyInFront(100, 500)

	eng.Commit("upheaval", commitCtx(p, e))
	if p.State != entity.PlayerStateAttack {
		t.Errorf("state = %d, want %d (attack)", p.State, entity.PlayerStateAttack)
	}
}

func TestUpheaval_DamageMultApplied(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()
	e := enemyInFront(100, 1000)

	p.AddBuff(entity.ActiveBuff{
		ID: "test_dmg", Type: entity.BuffDamageMult, Value: 2.0, Duration: 5.0,
	})

	r := eng.Commit("upheaval", commitCtx(p, e))
	if !r.OK {
		t.Fatalf("commit failed: %s", r.Reason)
	}
	if len(r.Events) == 1 && r.Events[0].Amount != 110 {
		t.Errorf("damage = %f, want 110 (55 base * 2.0 buff)", r.Events[0].Amount)
	}
}

func TestUpheaval_SetsCooldown(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()
	e := enemyInFront(100, 500)

	eng.Commit("upheaval", commitCtx(p, e))
	if cd := p.Cooldowns["upheaval"]; cd != 0.8 {
		t.Errorf("cooldown = %f, want 0.8", cd)
	}
}

func TestUpheaval_BuildsOnslaught(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()
	e := enemyInFront(100, 1e6)

	eng.Commit("upheaval", commitCtx(p, e))
	ons := getOnslaughtState(p)
	if ons.Stacks < 1 {
		t.Error("expected onslaught stacks from upheaval hit")
	}
}

func TestUpheaval_EmpoweredWiderCone(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()
	ons := getOnslaughtState(p)
	ons.Stacks = 3 // empowered

	// Enemy at edge: within 120° cone but outside 60°
	e := enemyInFront(100, 1e6)
	e.Position = entity.Vec3{X: 4.5, Y: 0, Z: -3} // ~56° off center

	r := eng.Commit("upheaval", commitCtx(p, e))
	if !r.OK {
		t.Fatalf("commit failed: %s", r.Reason)
	}
	if len(r.Events) == 0 {
		t.Error("empowered upheaval (120° cone) should hit enemy at ~56° off center")
	}
}

func TestUpheaval_MaximumAppliesDoT(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()
	ons := getOnslaughtState(p)
	ons.Stacks = 6 // maximum

	e := enemyInFront(100, 1e6)
	eng.Commit("upheaval", commitCtx(p, e))

	if len(p.DoTs) == 0 {
		t.Error("maximum upheaval should apply DoT to hit target")
	}
}
