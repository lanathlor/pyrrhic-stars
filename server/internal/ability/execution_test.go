package ability

import (
	"testing"

	"codex-online/server/internal/entity"
)

func TestExecution_InsufficientStamina(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()
	p.Resources["stamina"].Current = 10

	r := eng.Commit("execution", commitCtx(p))
	if r.OK {
		t.Error("should fail with insufficient stamina")
	}
}

func TestExecution_BlockedByCooldown(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()
	e := enemyInFront(100, 1e6)

	eng.Commit("execution", commitCtx(p, e))
	p.GCDTimer = 0
	p.Resources["stamina"].Current = 100

	r := eng.Commit("execution", commitCtx(p, e))
	if r.OK {
		t.Error("should be blocked by 8s cooldown")
	}
}

func TestExecution_DamageMultApplied(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()
	e := enemyInFront(100, 1000)

	p.AddBuff(entity.ActiveBuff{
		ID: "test_dmg", Type: entity.BuffDamageMult, Value: 2.0, Duration: 5.0,
	})

	r := eng.Commit("execution", commitCtx(p, e))
	if !r.OK {
		t.Fatalf("commit failed: %s", r.Reason)
	}
	for _, ev := range r.Events {
		if ev.Amount != 180 {
			t.Errorf("damage = %f, want 180 (90 base * 2.0 buff)", ev.Amount)
		}
	}
}

func TestExecution_SetsAttackState(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()
	e := enemyInFront(100, 500)

	eng.Commit("execution", commitCtx(p, e))
	if p.State != entity.PlayerStateAttack {
		t.Errorf("state = %d, want %d (attack)", p.State, entity.PlayerStateAttack)
	}
}

func TestExecution_NarrowCone(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()
	e1 := enemyInFront(100, 500) // directly in front
	e2 := enemyInFront(101, 500)
	e2.Position.X = 4 // far off center — outside 30° cone

	r := eng.Commit("execution", commitCtx(p, e1, e2))
	if !r.OK {
		t.Fatalf("commit failed: %s", r.Reason)
	}
	hitIDs := map[uint16]bool{}
	for _, ev := range r.Events {
		hitIDs[ev.TargetID] = true
	}
	if !hitIDs[100] {
		t.Error("e1 directly in front should be hit")
	}
	if hitIDs[101] {
		t.Error("e2 far off-center should NOT be hit by 30° cone")
	}
}

func TestExecution_SetsCooldown(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()

	eng.Commit("execution", commitCtx(p))
	if cd := p.Cooldowns["execution"]; cd != 8.0 {
		t.Errorf("cooldown = %f, want 8.0", cd)
	}
}

func TestExecution_BuildsOnslaught(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()
	e := enemyInFront(100, 1e6)

	eng.Commit("execution", commitCtx(p, e))
	ons := getOnslaughtState(p)
	if ons.Stacks < 1 {
		t.Error("expected onslaught stacks from execution hit")
	}
}

func TestExecution_EmpoweredShockwave(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()
	ons := getOnslaughtState(p)
	ons.Stacks = 3 // empowered

	// Place enemy within shockwave range (3) — must be close
	e := enemyInFront(100, 1e6)
	e.Position = entity.Vec3{X: 0, Y: 0, Z: -2.5}

	r := eng.Commit("execution", commitCtx(p, e))
	if !r.OK {
		t.Fatalf("commit failed: %s", r.Reason)
	}
	// Empowered should deal more total damage (primary + shockwave)
	var total float32
	for _, ev := range r.Events {
		total += ev.Amount
	}
	// Primary: 120*1.09 ≈ 130.8 + shockwave: 65.4 = ~196
	if total < 150 {
		t.Errorf("empowered execution total = %f, want > 150", total)
	}
}
