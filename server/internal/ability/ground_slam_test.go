package ability

import (
	"testing"

	"codex-online/server/internal/entity"
)

func TestGroundSlam_InsufficientStamina(t *testing.T) {
	eng := NewEngine()
	p := newVanguard()
	p.Resources["stamina"].Current = 10

	r := eng.Cast("ground_slam", castCtx(p))
	if r.OK {
		t.Error("should fail with insufficient stamina")
	}
}

func TestGroundSlam_BlockedByCooldown(t *testing.T) {
	eng := NewEngine()
	p := newVanguard()
	e := enemyInFront(100, 1e6)

	eng.Cast("ground_slam", castCtx(p, e))
	p.GCDTimer = 0 // clear lockout for re-cast attempt
	p.Resources["stamina"].Current = 100

	r := eng.Cast("ground_slam", castCtx(p, e))
	if r.OK {
		t.Error("should be blocked by 8s cooldown")
	}
}

func TestGroundSlam_DamageMultApplied(t *testing.T) {
	eng := NewEngine()
	p := newVanguard()
	e := enemyInFront(100, 1000)

	p.AddBuff(entity.ActiveBuff{
		ID: "test_dmg", Type: entity.BuffDamageMult, Value: 2.0, Duration: 5.0,
	})

	r := eng.Cast("ground_slam", castCtx(p, e))
	if !r.OK {
		t.Fatalf("cast failed: %s", r.Reason)
	}
	for _, ev := range r.Events {
		if ev.Amount != 120 {
			t.Errorf("damage = %f, want 120 (60 base * 2.0 buff)", ev.Amount)
		}
	}
}

func TestGroundSlam_SetsAttackState(t *testing.T) {
	eng := NewEngine()
	p := newVanguard()
	e := enemyInFront(100, 500)

	eng.Cast("ground_slam", castCtx(p, e))
	if p.State != entity.PlayerStateAttack {
		t.Errorf("state = %d, want %d (attack)", p.State, entity.PlayerStateAttack)
	}
}

func TestGroundSlam_HitsMultipleInCone(t *testing.T) {
	eng := NewEngine()
	p := newVanguard()
	e1 := enemyInFront(100, 500)
	e2 := enemyInFront(101, 500)
	e2.Position.X = 2 // offset but still in 90° cone at range 7

	r := eng.Cast("ground_slam", castCtx(p, e1, e2))
	if !r.OK {
		t.Fatalf("cast failed: %s", r.Reason)
	}
	hitIDs := map[uint16]bool{}
	for _, ev := range r.Events {
		hitIDs[ev.TargetID] = true
	}
	if !hitIDs[100] {
		t.Error("e1 in cone should be hit")
	}
	if !hitIDs[101] {
		t.Error("e2 in cone should be hit")
	}
}

func TestGroundSlam_SetsCooldown(t *testing.T) {
	eng := NewEngine()
	p := newVanguard()

	eng.Cast("ground_slam", castCtx(p))
	if cd := p.Cooldowns["ground_slam"]; cd != 8.0 {
		t.Errorf("cooldown = %f, want 8.0", cd)
	}
}
