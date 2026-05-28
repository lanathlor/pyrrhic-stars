package ability

import (
	"math"
	"testing"

	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
)

// --- helpers ---

func newEnemyCommitter(id uint16, hp float32) *entity.Enemy {
	e := entity.NewEnemy(id, hp, "test_boss")
	e.Position = entity.Vec3{X: 0, Y: 0.1, Z: 0}
	e.RotationY = 0 // facing -Z
	return e
}

func playerTarget(id uint16, pos entity.Vec3) *entity.Player {
	p := entity.NewPlayer(id, entity.ClassGunner)
	p.Position = pos
	return p
}

// --- CommitDef tests ---

func TestCastDef_EnemyMelee_HitsPlayer(t *testing.T) {
	eng := NewEngine(nil)
	e := newEnemyCommitter(200, 1000)
	p := playerTarget(1, entity.Vec3{X: 0, Y: 0.1, Z: -2})

	def := &AbilityDef{
		ID:         IDEnemyMelee,
		BaseDamage: 30,
		Hit:        HitDef{Type: HitAoECone, Range: 3.0, ArcDegrees: 180},
	}

	ctx := &CommitContext{
		Committer:  e,
		Targets:    []entity.Target{p},
		SourceType: combat.SourceEnemyMelee,
	}
	result := eng.CommitDef(def, ctx)
	if !result.OK {
		t.Fatalf("CommitDef failed: %s", result.Reason)
	}
	if len(result.Events) != 1 {
		t.Fatalf("events = %d, want 1", len(result.Events))
	}
	if result.Events[0].Amount != 30 {
		t.Errorf("damage = %f, want 30", result.Events[0].Amount)
	}
	if result.Events[0].TargetID != 1 {
		t.Errorf("targetID = %d, want 1", result.Events[0].TargetID)
	}
	if result.Events[0].SourceType != combat.SourceEnemyMelee {
		t.Errorf("sourceType = %d, want %d", result.Events[0].SourceType, combat.SourceEnemyMelee)
	}
}

func TestCastDef_EnemyMelee_HitsMultiplePlayers(t *testing.T) {
	eng := NewEngine(nil)
	e := newEnemyCommitter(200, 1000)
	p1 := playerTarget(1, entity.Vec3{X: 1, Y: 0.1, Z: -2})
	p2 := playerTarget(2, entity.Vec3{X: -1, Y: 0.1, Z: -2})

	def := &AbilityDef{
		ID:         "enemy_sweep",
		BaseDamage: 25,
		Hit:        HitDef{Type: HitAoECone, Range: 5.0, ArcDegrees: 180},
	}

	ctx := &CommitContext{
		Committer: e,
		Targets:   []entity.Target{p1, p2},
	}
	result := eng.CommitDef(def, ctx)
	if !result.OK {
		t.Fatalf("CommitDef failed: %s", result.Reason)
	}
	if len(result.Events) != 2 {
		t.Errorf("events = %d, want 2 (both players in cone)", len(result.Events))
	}
}

func TestCastDef_EnemyMelee_MissesPlayerBehind(t *testing.T) {
	eng := NewEngine(nil)
	e := newEnemyCommitter(200, 1000)
	p := playerTarget(1, entity.Vec3{X: 0, Y: 0.1, Z: 2}) // behind enemy

	def := &AbilityDef{
		ID:         IDEnemyMelee,
		BaseDamage: 30,
		Hit:        HitDef{Type: HitAoECone, Range: 3.0, ArcDegrees: 180},
	}

	ctx := &CommitContext{
		Committer: e,
		Targets:   []entity.Target{p},
	}
	result := eng.CommitDef(def, ctx)
	if !result.OK {
		t.Fatalf("CommitDef failed: %s", result.Reason)
	}
	if len(result.Events) != 0 {
		t.Error("should miss player behind enemy")
	}
}

func TestCastDef_EnemyMelee_MissesPlayerOutOfRange(t *testing.T) {
	eng := NewEngine(nil)
	e := newEnemyCommitter(200, 1000)
	p := playerTarget(1, entity.Vec3{X: 0, Y: 0.1, Z: -10}) // in front but far

	def := &AbilityDef{
		ID:         IDEnemyMelee,
		BaseDamage: 30,
		Hit:        HitDef{Type: HitAoECone, Range: 3.0, ArcDegrees: 180},
	}

	ctx := &CommitContext{
		Committer: e,
		Targets:   []entity.Target{p},
	}
	result := eng.CommitDef(def, ctx)
	if !result.OK {
		t.Fatal("commit should succeed")
	}
	if len(result.Events) != 0 {
		t.Error("should miss player out of range")
	}
}

func TestCastDef_EnemyMelee_SkipsDeadPlayer(t *testing.T) {
	eng := NewEngine(nil)
	e := newEnemyCommitter(200, 1000)
	p := playerTarget(1, entity.Vec3{X: 0, Y: 0.1, Z: -2})
	p.Alive = false

	def := &AbilityDef{
		ID:         IDEnemyMelee,
		BaseDamage: 30,
		Hit:        HitDef{Type: HitAoECone, Range: 3.0, ArcDegrees: 180},
	}

	ctx := &CommitContext{
		Committer: e,
		Targets:   []entity.Target{p},
	}
	result := eng.CommitDef(def, ctx)
	if len(result.Events) != 0 {
		t.Error("should skip dead player")
	}
}

func TestCastDef_EnemyAoE_HitsPlayersInRadius(t *testing.T) {
	eng := NewEngine(nil)
	e := newEnemyCommitter(200, 1000)
	pClose := playerTarget(1, entity.Vec3{X: 3, Y: 0.1, Z: 0})
	pFar := playerTarget(2, entity.Vec3{X: 20, Y: 0.1, Z: 0})

	def := &AbilityDef{
		ID:         "enemy_slam",
		BaseDamage: 40,
		Hit:        HitDef{Type: HitAoECircle, Radius: 5.0},
	}

	ctx := &CommitContext{
		Committer:  e,
		Targets:    []entity.Target{pClose, pFar},
		SourceType: combat.SourceEnemyAoE,
	}
	result := eng.CommitDef(def, ctx)
	if !result.OK {
		t.Fatalf("CommitDef failed: %s", result.Reason)
	}
	if len(result.Events) != 1 {
		t.Errorf("events = %d, want 1 (only close player)", len(result.Events))
	}
	if len(result.Events) == 1 && result.Events[0].TargetID != 1 {
		t.Errorf("targetID = %d, want 1 (close player)", result.Events[0].TargetID)
	}
}

func TestCastDef_EnemyAoE_SourceTypePreserved(t *testing.T) {
	eng := NewEngine(nil)
	e := newEnemyCommitter(200, 1000)
	p := playerTarget(1, entity.Vec3{X: 2, Y: 0.1, Z: 0})

	def := &AbilityDef{
		ID:         "enemy_slam",
		BaseDamage: 40,
		Hit:        HitDef{Type: HitAoECircle, Radius: 5.0},
	}

	ctx := &CommitContext{
		Committer:  e,
		Targets:    []entity.Target{p},
		SourceType: combat.SourceEnemyAoE,
	}
	result := eng.CommitDef(def, ctx)
	if len(result.Events) != 1 {
		t.Fatal("expected 1 hit")
	}
	if result.Events[0].SourceType != combat.SourceEnemyAoE {
		t.Errorf("sourceType = %d, want %d", result.Events[0].SourceType, combat.SourceEnemyAoE)
	}
}

// --- Non-player caster skips player-specific logic ---

func TestCastDef_EnemyCaster_SkipsGCDCheck(t *testing.T) {
	eng := NewEngine(nil)
	e := newEnemyCommitter(200, 1000)
	p := playerTarget(1, entity.Vec3{X: 0, Y: 0.1, Z: -2})

	def := &AbilityDef{
		ID:         IDEnemyMelee,
		BaseDamage: 30,
		Hit:        HitDef{Type: HitAoECone, Range: 3.0, ArcDegrees: 180},
	}

	// Enemies don't have GCD — should always succeed
	ctx := &CommitContext{Committer: e, Targets: []entity.Target{p}}
	result := eng.CommitDef(def, ctx)
	if !result.OK {
		t.Errorf("enemy commit should not be gated by GCD: %s", result.Reason)
	}
}

func TestCastDef_EnemyCaster_SkipsCooldownCheck(t *testing.T) {
	eng := NewEngine(nil)
	e := newEnemyCommitter(200, 1000)

	def := &AbilityDef{
		ID:       "enemy_ability",
		Cooldown: 5.0, // would block a player
		Hit:      HitDef{Type: HitNone},
	}

	// Commit twice — enemies don't track cooldowns through the engine
	ctx := &CommitContext{Committer: e}
	r1 := eng.CommitDef(def, ctx)
	r2 := eng.CommitDef(def, ctx)
	if !r1.OK || !r2.OK {
		t.Error("enemy should be able to commit repeatedly without cooldown gating")
	}
}

func TestCastDef_EnemyCaster_SkipsResourceCheck(t *testing.T) {
	eng := NewEngine(nil)
	e := newEnemyCommitter(200, 1000)

	def := &AbilityDef{
		ID:    "enemy_ability",
		Hit:   HitDef{Type: HitNone},
		Costs: []ResourceCost{{Resource: "stamina", Amount: 999}},
	}

	ctx := &CommitContext{Committer: e}
	result := eng.CommitDef(def, ctx)
	if !result.OK {
		t.Errorf("enemy should not be gated by resource costs: %s", result.Reason)
	}
}

func TestCastDef_EnemyCaster_SkipsPostCastEffects(t *testing.T) {
	eng := NewEngine(nil)
	e := newEnemyCommitter(200, 1000)
	p := playerTarget(1, entity.Vec3{X: 0, Y: 0.1, Z: -2})

	def := &AbilityDef{
		ID:         "enemy_slash",
		BaseDamage: 30,
		Hit:        HitDef{Type: HitAoECone, Range: 3.0, ArcDegrees: 180},
		Cooldown:   5.0,
		GCD:        1.0,
		SelfBuffs: []BuffEffect{
			{ID: "test_buff", Type: "damage_mult", Value: 1.5, Duration: 5.0},
		},
	}

	ctx := &CommitContext{Committer: e, Targets: []entity.Target{p}}
	result := eng.CommitDef(def, ctx)
	if !result.OK {
		t.Fatalf("commit failed: %s", result.Reason)
	}
	// Verify no player-specific side effects leaked onto the enemy
	if len(result.Events) != 1 {
		t.Errorf("events = %d, want 1", len(result.Events))
	}
}

func TestCastDef_DeadEnemy_Rejected(t *testing.T) {
	eng := NewEngine(nil)
	e := newEnemyCommitter(200, 1000)
	e.Alive = false

	def := &AbilityDef{
		ID:  "enemy_ability",
		Hit: HitDef{Type: HitNone},
	}

	ctx := &CommitContext{Committer: e}
	result := eng.CommitDef(def, ctx)
	if result.OK {
		t.Error("dead enemy should not be able to commit")
	}
	if result.Reason != "dead" {
		t.Errorf("reason = %q, want %q", result.Reason, "dead")
	}
}

// --- DamageResult.Target field ---

func TestCastDef_DamageResult_HasTarget(t *testing.T) {
	eng := NewEngine(nil)
	e := newEnemyCommitter(200, 1000)
	p := playerTarget(1, entity.Vec3{X: 0, Y: 0.1, Z: -2})

	def := &AbilityDef{
		ID:         IDEnemyMelee,
		BaseDamage: 30,
		Hit:        HitDef{Type: HitAoECone, Range: 3.0, ArcDegrees: 180},
	}

	ctx := &CommitContext{Committer: e, Targets: []entity.Target{p}}
	result := eng.CommitDef(def, ctx)
	if len(result.Events) != 1 {
		t.Fatal("expected 1 event")
	}
	if result.Events[0].Target == nil {
		t.Fatal("DamageResult.Target should not be nil")
	}
	hitPlayer, ok := result.Events[0].Target.(*entity.Player)
	if !ok {
		t.Fatal("Target should be *entity.Player")
	}
	if hitPlayer.ID != 1 {
		t.Errorf("hit player ID = %d, want 1", hitPlayer.ID)
	}
}

// --- Resolve with enemy caster ---

func TestResolveHitscan_EnemyCaster(t *testing.T) {
	e := newEnemyCommitter(200, 1000)
	p := playerTarget(1, entity.Vec3{X: 0, Y: 0.1, Z: -5})

	results := resolveHitscan(nil, e, []entity.Target{p}, nil, 20, combat.SourceEnemyMelee)
	if len(results) != 1 {
		t.Fatalf("hits = %d, want 1", len(results))
	}
	if results[0].Amount != 20 {
		t.Errorf("damage = %f, want 20", results[0].Amount)
	}
	if results[0].SourceID != 200 {
		t.Errorf("sourceID = %d, want 200 (enemy ID)", results[0].SourceID)
	}
}

func TestResolveAoECone_EnemyCaster(t *testing.T) {
	e := newEnemyCommitter(200, 1000)
	p1 := playerTarget(1, entity.Vec3{X: 1, Y: 0.1, Z: -3})
	p2 := playerTarget(2, entity.Vec3{X: -1, Y: 0.1, Z: -3})
	pBehind := playerTarget(3, entity.Vec3{X: 0, Y: 0.1, Z: 3})

	hit := HitDef{Type: HitAoECone, Range: 5.0, ArcDegrees: 180}
	results := resolveAoECone(nil, e, []entity.Target{p1, p2, pBehind}, nil, hit, 25, combat.SourceEnemyMelee)

	if len(results) != 2 {
		t.Errorf("hits = %d, want 2 (two in front)", len(results))
	}
	for _, r := range results {
		if r.TargetID == 3 {
			t.Error("should not hit player behind enemy")
		}
	}
}

func TestResolveAoECone_EnemyRotated(t *testing.T) {
	e := newEnemyCommitter(200, 1000)
	e.RotationY = float32(-math.Pi / 2) // facing +X

	pRight := playerTarget(1, entity.Vec3{X: 3, Y: 0.1, Z: 0}) // in front (+X)
	pLeft := playerTarget(2, entity.Vec3{X: -3, Y: 0.1, Z: 0}) // behind (-X)

	hit := HitDef{Type: HitAoECone, Range: 5.0, ArcDegrees: 180}
	results := resolveAoECone(nil, e, []entity.Target{pRight, pLeft}, nil, hit, 25, 0)

	if len(results) != 1 {
		t.Fatalf("hits = %d, want 1", len(results))
	}
	if results[0].TargetID != 1 {
		t.Errorf("should hit player to the right (in front), got targetID=%d", results[0].TargetID)
	}
}

func TestResolveMeleeArc_EnemyCaster(t *testing.T) {
	e := newEnemyCommitter(200, 1000)
	p := playerTarget(1, entity.Vec3{X: 0, Y: 0.1, Z: -4})

	hit := HitDef{Type: HitMeleeArc, Range: 6.0, ArcDegrees: 120}
	results := resolveMeleeArc(nil, e, []entity.Target{p}, nil, hit, 35, combat.SourceEnemyMelee)

	if len(results) != 1 {
		t.Fatalf("hits = %d, want 1", len(results))
	}
	if results[0].Amount != 35 {
		t.Errorf("damage = %f, want 35", results[0].Amount)
	}
}

func TestResolveAoECircle_EnemyCaster(t *testing.T) {
	e := newEnemyCommitter(200, 1000)
	p := playerTarget(1, entity.Vec3{X: 3, Y: 0.1, Z: 0})

	results := resolveAoECircle(nil, e.CommitterPos(), e.CommitterID(), []entity.Target{p}, nil, 5.0, 40, combat.SourceEnemyAoE)

	if len(results) != 1 {
		t.Fatalf("hits = %d, want 1", len(results))
	}
	if results[0].SourceID != 200 {
		t.Errorf("sourceID = %d, want 200", results[0].SourceID)
	}
	if results[0].SourceType != combat.SourceEnemyAoE {
		t.Errorf("sourceType = %d, want %d", results[0].SourceType, combat.SourceEnemyAoE)
	}
}

// --- Obstacle blocking ---

func TestCastDef_EnemyMelee_ObstacleBlocksLOS(t *testing.T) {
	eng := NewEngine(nil)
	e := newEnemyCommitter(200, 1000)
	p := playerTarget(1, entity.Vec3{X: 0, Y: 0.1, Z: -4})
	obs := combat.Obstacle{CX: 0, CZ: -2, HX: 2, HZ: 0.5, Height: 3}

	def := &AbilityDef{
		ID:         IDEnemyMelee,
		BaseDamage: 30,
		Hit:        HitDef{Type: HitAoECone, Range: 5.0, ArcDegrees: 180},
	}

	ctx := &CommitContext{
		Committer: e,
		Targets:   []entity.Target{p},
		Obstacles: []combat.Obstacle{obs},
	}
	result := eng.CommitDef(def, ctx)
	if len(result.Events) != 0 {
		t.Error("obstacle should block melee hit")
	}
}

// --- Bidirectional: player → player (PvP readiness) ---

func TestCastDef_PlayerTargetsPlayer(t *testing.T) {
	eng := NewEngine(nil)
	attacker := newGunner()
	attacker.ID = 10
	victim := playerTarget(20, entity.Vec3{X: 0, Y: 0.1, Z: -5})

	def := &AbilityDef{
		ID:         "pvp_slash",
		BaseDamage: 15,
		Hit:        HitDef{Type: HitAoECone, Range: 6.0, ArcDegrees: 120},
	}

	ctx := &CommitContext{Committer: attacker, Targets: []entity.Target{victim}}
	result := eng.CommitDef(def, ctx)
	if !result.OK {
		t.Fatalf("PvP commit failed: %s", result.Reason)
	}
	if len(result.Events) != 1 {
		t.Fatalf("events = %d, want 1", len(result.Events))
	}
	if result.Events[0].TargetID != 20 {
		t.Errorf("targetID = %d, want 20", result.Events[0].TargetID)
	}
}

// --- Benchmarks ---

func BenchmarkCastDef_EnemyMelee(b *testing.B) {
	eng := NewEngine(nil)
	e := newEnemyCommitter(200, 1000)
	targets := make([]entity.Target, 5)
	for i := range targets {
		p := playerTarget(uint16(i+1), entity.Vec3{X: float32(i-2) * 1.5, Y: 0.1, Z: -2})
		p.Health = 1e9
		targets[i] = p
	}
	def := &AbilityDef{
		ID:         IDEnemyMelee,
		BaseDamage: 30,
		Hit:        HitDef{Type: HitAoECone, Range: 5.0, ArcDegrees: 180},
	}
	ctx := &CommitContext{
		Committer:  e,
		Targets:    targets,
		SourceType: combat.SourceEnemyMelee,
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		for _, t := range targets {
			t.(*entity.Player).Health = 1e9
			t.(*entity.Player).Alive = true
		}
		eng.CommitDef(def, ctx)
	}
}

func BenchmarkCastDef_EnemyAoE(b *testing.B) {
	eng := NewEngine(nil)
	e := newEnemyCommitter(200, 1000)
	targets := make([]entity.Target, 5)
	for i := range targets {
		p := playerTarget(uint16(i+1), entity.Vec3{X: float32(i-2) * 1.5, Y: 0.1, Z: float32(i)})
		p.Health = 1e9
		targets[i] = p
	}
	def := &AbilityDef{
		ID:         "enemy_slam",
		BaseDamage: 40,
		Hit:        HitDef{Type: HitAoECircle, Radius: 8.0},
	}
	ctx := &CommitContext{
		Committer:  e,
		Targets:    targets,
		SourceType: combat.SourceEnemyAoE,
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		for _, t := range targets {
			t.(*entity.Player).Health = 1e9
			t.(*entity.Player).Alive = true
		}
		eng.CommitDef(def, ctx)
	}
}

func BenchmarkCastDef_EnemyMelee_Miss(b *testing.B) {
	eng := NewEngine(nil)
	e := newEnemyCommitter(200, 1000)
	p := playerTarget(1, entity.Vec3{X: 0, Y: 0.1, Z: 20}) // far behind
	def := &AbilityDef{
		ID:         IDEnemyMelee,
		BaseDamage: 30,
		Hit:        HitDef{Type: HitAoECone, Range: 3.0, ArcDegrees: 180},
	}
	ctx := &CommitContext{
		Committer: e,
		Targets:   []entity.Target{p},
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		eng.CommitDef(def, ctx)
	}
}

func BenchmarkResolveAoECone_5Players(b *testing.B) {
	e := newEnemyCommitter(200, 1000)
	targets := make([]entity.Target, 5)
	for i := range targets {
		p := playerTarget(uint16(i+1), entity.Vec3{X: float32(i-2) * 1.5, Y: 0.1, Z: -2})
		p.Health = 1e9
		targets[i] = p
	}
	hit := HitDef{Type: HitAoECone, Range: 5.0, ArcDegrees: 180}
	var buf []DamageResult

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		for _, t := range targets {
			t.(*entity.Player).Health = 1e9
			t.(*entity.Player).Alive = true
		}
		buf = resolveAoECone(buf[:0], e, targets, nil, hit, 25, 0)
	}
}
