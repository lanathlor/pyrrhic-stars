package system

import (
	"testing"

	"codex-online/server/internal/ability"
	"codex-online/server/internal/codec"
	"codex-online/server/internal/combat"
	"codex-online/server/internal/enemyai"
	"codex-online/server/internal/entity"
)

func f32p(v float32) *float32 { return &v }

// registerTelegraphTestBoss installs a minimal boss def with a phase-scaled
// ground_slam (6.5 base -> 7.5 phase 2) and a pillar_overload around obstacles.
func registerTelegraphTestBoss() {
	enemyai.DefRegistry["tg_test_boss"] = &enemyai.EnemyDef{
		Name:      "tg_test_boss",
		MaxHealth: 1000,
		Abilities: []ability.AbilityDef{
			{
				ID: "ground_slam", Name: "ground_slam", Category: ability.CategoryAoE,
				CommitTime: 1.5, DamageSource: combat.SourceEnemyAoE,
				Hit: ability.HitDef{Type: ability.HitAoECircle, Radius: 6.5},
			},
			{
				ID: "pillar_overload", Name: "pillar_overload", Category: ability.CategoryAoE,
				CommitTime: 1.0, DamageSource: combat.SourceEnemyPillar,
				Hit: ability.HitDef{Type: ability.HitAoEObstacles, Radius: 9.0},
			},
		},
		Phases: []enemyai.PhaseDef{
			{ // phase 2
				HPThresholdPct:   0.6,
				AbilityOverrides: map[string]enemyai.AbilityOverride{"ground_slam": {AoERadius: f32p(7.5)}},
			},
		},
	}
}

func telegraphTestEnemy(abilityIdx, phase int, state entity.EnemyState) *entity.Enemy {
	e := entity.NewEnemy(1000, 1000, "tg_test_boss")
	e.Alive = true
	e.Position = entity.Vec3{X: 3, Z: -4}
	e.ActiveAbility = abilityIdx
	e.Phase = phase
	e.State = state
	e.StateTimer = 0.5 // mid-commit
	return e
}

func TestBuildTelegraphs_GroundSlamPhaseRadius(t *testing.T) {
	registerTelegraphTestBoss()
	defer delete(enemyai.DefRegistry, "tg_test_boss")

	for _, tc := range []struct {
		phase      int
		wantRadius float32
	}{{1, 6.5}, {2, 7.5}} {
		w := &World{TickNum: 100, Enemies: []*entity.Enemy{telegraphTestEnemy(0, tc.phase, entity.EnemyAoETelegraph)}}
		tgs := buildTelegraphs(w)
		if len(tgs) != 1 {
			t.Fatalf("phase %d: got %d telegraphs, want 1", tc.phase, len(tgs))
		}
		got := tgs[0]
		if got.Shape != codec.TelegraphShapeCircle {
			t.Errorf("phase %d: shape = %d, want circle", tc.phase, got.Shape)
		}
		if got.Radius != tc.wantRadius {
			t.Errorf("phase %d: radius = %v, want %v (phase-resolved)", tc.phase, got.Radius, tc.wantRadius)
		}
		if got.CX != 3 || got.CZ != -4 {
			t.Errorf("phase %d: center = (%v,%v), want enemy pos (3,-4)", tc.phase, got.CX, got.CZ)
		}
		// Window: commit 1.5s = 30 ticks, 0.5s remaining = 10 ticks left.
		if got.ExecuteTick != 110 || got.StartTick != 80 {
			t.Errorf("phase %d: window start=%d exec=%d, want 80/110", tc.phase, got.StartTick, got.ExecuteTick)
		}
		if got.Category != codec.TelegraphCatUnavoidable {
			t.Errorf("phase %d: aoe category = %d, want unavoidable", tc.phase, got.Category)
		}
	}
}

func TestBuildTelegraphs_PillarOverloadMultiRing(t *testing.T) {
	registerTelegraphTestBoss()
	defer delete(enemyai.DefRegistry, "tg_test_boss")

	w := &World{
		TickNum: 50,
		Enemies: []*entity.Enemy{telegraphTestEnemy(1, 1, entity.EnemyAoETelegraph)},
		Obstacles: []combat.Obstacle{
			{CX: -8, CZ: -6, HX: 0.75, HZ: 0.75}, // pillar
			{CX: 8, CZ: -6, HX: 0.75, HZ: 0.75},  // pillar
			{CX: 0, CZ: -15, HX: 20, HZ: 0.25},   // wall (not pillar-like)
		},
	}
	tgs := buildTelegraphs(w)
	if len(tgs) != 1 {
		t.Fatalf("got %d telegraphs, want 1", len(tgs))
	}
	got := tgs[0]
	if got.Shape != codec.TelegraphShapeMulti {
		t.Fatalf("shape = %d, want multi", got.Shape)
	}
	if len(got.Centers) != 2 {
		t.Errorf("got %d centers, want 2 (pillars only, wall excluded)", len(got.Centers))
	}
	// radius = Hit.Radius(9) + max half-extent(0.75)
	if got.Radius != 9.75 {
		t.Errorf("radius = %v, want 9.75", got.Radius)
	}
}

func TestBuildTelegraphs_OnlyTelegraphStates(t *testing.T) {
	registerTelegraphTestBoss()
	defer delete(enemyai.DefRegistry, "tg_test_boss")

	w := &World{TickNum: 100, Enemies: []*entity.Enemy{telegraphTestEnemy(0, 1, entity.EnemyChase)}}
	if tgs := buildTelegraphs(w); len(tgs) != 0 {
		t.Errorf("a chasing enemy should emit no telegraph, got %d", len(tgs))
	}
}
