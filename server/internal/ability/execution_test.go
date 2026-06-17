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
		ID: tcTestDmg, Type: entity.BuffDamageMult, Value: 2.0, Duration: 5.0,
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

func TestExecution_EmpoweredShockwaveCatchesPeripheral(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()
	ons := getOnslaughtState(p)
	ons.Stacks = 3 // empowered: shockwave arc 60°, range 3

	// Primary target: directly in front, inside the narrow 30° primary cone.
	primary := enemyInFront(100, 1e6)
	primary.Position = entity.Vec3{X: 0, Y: 0, Z: -2.5}
	// Peripheral target: close (within shockwave range 3) but off-axis so it
	// falls outside the 30° primary cone yet inside the 60° shockwave cone.
	peripheral := enemyInFront(101, 1e6)
	peripheral.Position = entity.Vec3{X: 1.0, Y: 0, Z: -2.0}

	r := eng.Commit("execution", commitCtx(p, primary, peripheral))
	if !r.OK {
		t.Fatalf("commit failed: %s", r.Reason)
	}

	hits := map[uint16][]float32{}
	for _, ev := range r.Events {
		hits[ev.TargetID] = append(hits[ev.TargetID], ev.Amount)
	}
	if len(hits[100]) != 1 {
		t.Errorf("primary target hit %d times, want exactly 1 (no double-dip)", len(hits[100]))
	}
	if len(hits[101]) != 1 {
		t.Errorf("peripheral target hit %d times, want exactly 1 (shockwave)", len(hits[101]))
	}
}

// A target standing inside both the primary cone and the shockwave cone must
// only take the primary hit — the shockwave rewards catching EXTRA enemies, it
// must not stack a second 0.5x hit on whoever the primary already struck.
func TestExecution_NoShockwaveDoubleDip(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()
	ons := getOnslaughtState(p)
	ons.Stacks = 6 // maximum: shockwave arc 90°, range 5

	// Single enemy directly in front and close: inside primary AND shockwave.
	e := enemyInFront(100, 1e6)
	e.Position = entity.Vec3{X: 0, Y: 0, Z: -2.5}

	r := eng.Commit("execution", commitCtx(p, e))
	if !r.OK {
		t.Fatalf("commit failed: %s", r.Reason)
	}

	var hits int
	var total float32
	for _, ev := range r.Events {
		if ev.TargetID == 100 {
			hits++
			total += ev.Amount
		}
	}
	if hits != 1 {
		t.Fatalf("target hit %d times, want exactly 1 (primary only, no shockwave double-dip)", hits)
	}
	// Max tier base 150 * 1.18 onslaught mult ≈ 177 primary; must NOT include the
	// extra 0.5x shockwave (~88 more) on the same target.
	if total > 200 {
		t.Errorf("single-target damage = %f, want primary-only (~177); double-dip leaked in", total)
	}
}
