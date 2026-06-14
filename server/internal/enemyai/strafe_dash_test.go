package enemyai

import (
	"math"
	"testing"

	"codex-online/server/internal/bt"
	"codex-online/server/internal/entity"
)

func speed2D(v entity.Vec3) float64 {
	return math.Hypot(float64(v.X), float64(v.Z))
}

// TestStrafe_PerpendicularAndFlips verifies the strafe leaf moves the mob
// sideways relative to its target (not toward/away) and periodically flips the
// side it strafes to, so a ranged mob weaves between volleys instead of standing
// still.
func TestStrafe_PerpendicularAndFlips(t *testing.T) {
	def := DefRegistry["hallway_ranged"] // preferred_range 8
	if def == nil {
		t.Fatal("hallway_ranged not loaded")
	}
	b, e := testBrain(def)
	e.Alive = true
	e.State = entity.EnemyChase
	e.Position = entity.Vec3{}
	b.SetTree(bt.NewAction(actionStrafe))

	p := testPlayer(1, entity.Vec3{Z: 8}) // at the preferred range, dead ahead (+Z)
	p.Alive = true
	p.Health = p.MaxHealth
	players := testPlayers(p)
	e.TargetPlayerID = 1

	signs := map[float32]bool{}
	var sawPerp bool
	for range 40 {
		b.Tick(0.05, players, nil, noSpawn, nil)
		// Target is dead ahead in +Z, so strafing is motion along X.
		if math.Abs(float64(e.Velocity.X)) <= math.Abs(float64(e.Velocity.Z)) {
			continue
		}
		sawPerp = true
		if e.Velocity.X > 0 {
			signs[1] = true
		} else if e.Velocity.X < 0 {
			signs[-1] = true
		}
	}
	if !sawPerp {
		t.Fatal("strafe never produced sideways (X-dominant) motion toward a +Z target")
	}
	if !signs[1] || !signs[-1] {
		t.Fatalf("strafe did not flip sides over 2s (saw signs: %v)", signs)
	}
}

func TestStrafe_RootFreezes(t *testing.T) {
	def := DefRegistry["hallway_ranged"]
	b, e := testBrain(def)
	e.Alive = true
	e.State = entity.EnemyChase
	e.Position = entity.Vec3{}
	e.AddDebuff(entity.ActiveDebuff{ID: "root", Type: entity.DebuffRoot, Duration: 5})
	b.SetTree(bt.NewAction(actionStrafe))

	p := testPlayer(1, entity.Vec3{Z: 8})
	p.Alive = true
	p.Health = p.MaxHealth
	e.TargetPlayerID = 1

	b.Tick(0.05, testPlayers(p), nil, noSpawn, nil)
	if speed2D(e.Velocity) > 0.001 {
		t.Fatalf("rooted mob should not move while strafing, got velocity %v", e.Velocity)
	}
}

// TestDash_BurstThenCooldown verifies dash produces a burst of speed above the
// mob's walk speed, then goes quiet for its cooldown, then bursts again.
func TestDash_BurstThenCooldown(t *testing.T) {
	def := DefRegistry["hallway_melee"] // move_speed 5
	b, e := testBrain(def)
	e.Alive = true
	e.State = entity.EnemyChase
	e.Position = entity.Vec3{}
	b.SetTree(bt.NewAction(dashFactory(0.6)))

	p := testPlayer(1, entity.Vec3{Z: 10}) // far, so dash wants to close
	p.Alive = true
	p.Health = p.MaxHealth
	players := testPlayers(p)
	e.TargetPlayerID = 1

	walk := float64(def.CurrentMoveSpeed(1))
	burstThreshold := walk * 1.5

	var speeds []float64
	for range 16 {
		b.Tick(0.05, players, nil, noSpawn, nil)
		speeds = append(speeds, speed2D(e.Velocity))
	}

	if speeds[0] <= burstThreshold {
		t.Fatalf("first dash tick should burst (> %.1f), got %.1f", burstThreshold, speeds[0])
	}
	// Mid-cooldown (after the ~0.25s burst window, before 0.6s cooldown): quiet.
	if speeds[7] > 0.001 {
		t.Fatalf("dash should be idle mid-cooldown, got speed %.1f at tick 8", speeds[7])
	}
	// After the cooldown elapses a second burst fires.
	secondBurst := false
	for _, s := range speeds[12:] {
		if s > burstThreshold {
			secondBurst = true
		}
	}
	if !secondBurst {
		t.Fatalf("dash should burst again after cooldown; tail speeds=%v", speeds[12:])
	}
}

// TestDash_CooldownParam confirms a short cooldown ("relentless") re-bursts far
// more often than a long one over the same window.
func TestDash_CooldownParam(t *testing.T) {
	countBursts := func(cd float32) int {
		def := DefRegistry["hallway_melee"]
		b, e := testBrain(def)
		e.Alive = true
		e.State = entity.EnemyChase
		e.Position = entity.Vec3{}
		b.SetTree(bt.NewAction(dashFactory(cd)))
		p := testPlayer(1, entity.Vec3{Z: 30})
		p.Alive = true
		p.Health = p.MaxHealth
		e.TargetPlayerID = 1

		walk := float64(def.CurrentMoveSpeed(1))
		bursts, wasBursting := 0, false
		for range 40 { // 2 seconds
			b.Tick(0.05, testPlayers(p), nil, noSpawn, nil)
			bursting := speed2D(e.Velocity) > walk*1.5
			if bursting && !wasBursting {
				bursts++
			}
			wasBursting = bursting
		}
		return bursts
	}

	relentless := countBursts(0.6)
	occasional := countBursts(5.0)
	if relentless <= occasional {
		t.Fatalf("short-cooldown dash should burst more often: relentless=%d occasional=%d", relentless, occasional)
	}
}
