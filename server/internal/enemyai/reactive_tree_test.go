package enemyai

import (
	"strings"
	"testing"

	"codex-online/server/internal/ability"
	"codex-online/server/internal/entity"
)

// trackCommits ticks the brain n times, teleporting the player to `hold`
// meters from the enemy after every tick (a perfect-distance kiter), and
// returns how many times each ability was committed.
func trackCommits(b *Brain, e *entity.Enemy, p *entity.Player, n int, hold float32) map[string]int {
	commits := map[string]int{}
	players := testPlayers(p)
	prevPhase := b.ctx.Runner.Phase
	for range n {
		b.Tick(0.05, players, nil, noSpawn, nil)
		if prevPhase == RunnerIdle && b.ctx.Runner.Phase == RunnerCommit {
			commits[b.def.Abilities[b.ctx.Runner.AbilIdx].ID]++
		}
		prevPhase = b.ctx.Runner.Phase
		// Hold the player at a fixed distance in front of the enemy.
		p.Position = entity.Vec3{X: e.Position.X, Y: e.Position.Y, Z: e.Position.Z + hold}
	}
	return commits
}

// TestAceras_EvasiveModeWeavesBarrage reproduces the live-play bug where the
// aceras_general committed exactly one pattern during its whole kite phase:
// nested non-reactive composites cached their running child (chase in the
// is_committed branch, chase at the tail of the evasive selector) and never
// re-evaluated the ability guards. Over 30 simulated seconds against a player
// holding at kite range, the evasive kit (spiral cd 10s, aimed cd 12s,
// gravity cd 16s) must produce a steady barrage of multiple distinct patterns.
func TestAceras_EvasiveModeWeavesBarrage(t *testing.T) {
	def := DefRegistry["aceras_general"]
	if def == nil {
		t.Fatal("aceras_general not in registry")
	}
	b, e := testBrain(def)
	e.Alive = true
	e.State = entity.EnemyChase
	e.LeashRadius = 200.0

	p := testPlayer(1, entity.Vec3{X: 0, Z: 12})
	p.MaxHealth = 1e6
	p.Health = p.MaxHealth
	e.TargetPlayerID = p.ID

	commits := trackCommits(b, e, p, 600, 12) // 30s at 20Hz, player holds 12m

	total := 0
	for _, n := range commits {
		total += n
	}
	t.Logf("commits over 30s: %v", commits)
	if len(commits) < 2 {
		t.Errorf("evasive boss should weave multiple distinct patterns, committed only %v", commits)
	}
	if total < 4 {
		t.Errorf("evasive boss should keep a steady barrage (>=4 commits in 30s), got %d: %v", total, commits)
	}
}

// TestGuardCaptain_RecommitsWhileChasing reproduces the "boss sits there and
// occasionally remembers to act" bug: once the phase-1 selector cached chase
// as its running child, the ability branches were never re-evaluated, so a
// boss that could not physically reach a kiting player never cast again.
// A player held at 6m (outside melee, inside fireball range) must eat repeated
// fireball_burst commits (cd 12s → at least 2 in 30s).
func TestGuardCaptain_RecommitsWhileChasing(t *testing.T) {
	def := DefRegistry["guard_captain"]
	if def == nil {
		t.Fatal("guard_captain not in registry")
	}
	b, e := testBrain(def)
	e.Alive = true
	e.State = entity.EnemyChase
	e.LeashRadius = 200.0

	p := testPlayer(1, entity.Vec3{X: 0, Z: 6})
	p.MaxHealth = 1e6
	p.Health = p.MaxHealth
	e.TargetPlayerID = p.ID

	commits := trackCommits(b, e, p, 600, 6) // 30s, player kites at 6m

	t.Logf("commits over 30s: %v", commits)
	total := 0
	for _, n := range commits {
		total += n
	}
	if total < 2 {
		t.Errorf("boss must keep casting at a kiting player, got %d commits: %v", total, commits)
	}
}

// trackCommitOrder is trackCommits but preserves the cast sequence.
func trackCommitOrder(b *Brain, e *entity.Enemy, p *entity.Player, n int, hold float32) []string {
	var order []string
	players := testPlayers(p)
	prevPhase := b.ctx.Runner.Phase
	for range n {
		b.Tick(0.05, players, nil, noSpawn, nil)
		if prevPhase == RunnerIdle && b.ctx.Runner.Phase == RunnerCommit {
			order = append(order, b.def.Abilities[b.ctx.Runner.AbilIdx].ID)
		}
		prevPhase = b.ctx.Runner.Phase
		p.Position = entity.Vec3{X: e.Position.X, Y: e.Position.Y, Z: e.Position.Z + hold}
	}
	return order
}

// TestAceras_EvasiveRotationVaries locks in the weighted evasive kit: the
// pattern order must depend on the RNG seed, not settle into one fixed
// A-B-C cycle (live feedback 2026-07-14: "do spell A, then B, then C, then
// cooldowns, repeat").
func TestAceras_EvasiveRotationVaries(t *testing.T) {
	def := DefRegistry["aceras_general"]
	if def == nil {
		t.Fatal("aceras_general not in registry")
	}
	run := func(seed uint64) []string {
		e := entity.NewEnemy(0, def.MaxHealth, def.Name)
		e.State = entity.EnemyChase
		e.Position = entity.Vec3{X: 0, Y: 0.1, Z: 0}
		e.Alive = true
		e.LeashRadius = 200.0
		b := NewBrainSeeded(def, e, ability.NewEngine(nil), seed)
		b.BoundsMinX, b.BoundsMaxX = -20, 20
		b.BoundsMinZ, b.BoundsMaxZ = -15, 50
		p := testPlayer(1, entity.Vec3{X: 0, Z: 12})
		p.MaxHealth = 1e6
		p.Health = p.MaxHealth
		e.TargetPlayerID = p.ID
		return trackCommitOrder(b, e, p, 1200, 12) // 60s
	}

	orders := map[string]bool{}
	for seed := uint64(1); seed <= 6; seed++ {
		seq := run(seed * 1337)
		if len(seq) < 4 {
			t.Fatalf("seed %d: expected a steady barrage, got %v", seed, seq)
		}
		orders[strings.Join(seq, ",")] = true
	}
	if len(orders) < 2 {
		t.Errorf("rotation is deterministic across seeds — weighted selection not in effect: %v", orders)
	}
}

// SelectAbility must not offer abilities whose cooldown is still running:
// Runner.Start does not re-check cooldowns, so without this filter the
// weighted path casts straight through cooldown_time (hallway mobs paced by
// GCD only), and weighted rotations can never interleave around cooldowns.
func TestSelectAbility_SkipsOnCooldown(t *testing.T) {
	def := DefRegistry["hallway_ranged"]
	if def == nil {
		t.Fatal("hallway_ranged not in registry")
	}
	b, e := testBrain(def)
	e.Alive = true
	p := testPlayer(1, entity.Vec3{Z: 6})
	b.ctx.Reset(0.05, testPlayers(p), nil, noSpawn, nil, nil)
	e.TargetPlayerID = p.ID

	chosen := b.ctx.SelectAbility(6)
	if chosen == nil {
		t.Fatal("expected an ability candidate at range 6")
	}
	// Put everything the def has on cooldown: nothing may be selected.
	if b.ctx.Runner.AbilityCDs == nil {
		b.ctx.Runner.AbilityCDs = make(map[int]float32)
	}
	for i := range def.Abilities {
		b.ctx.Runner.AbilityCDs[i] = 5.0
	}
	if got := b.ctx.SelectAbility(6); got != nil {
		t.Errorf("SelectAbility returned %q while all abilities are on cooldown", got.ID)
	}
}
