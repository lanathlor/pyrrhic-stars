package enemyai

import (
	"math"
	"testing"

	"codex-online/server/internal/entity"
)

// --- Helpers ---

// testTreeData returns a YAML-style tree spec that mirrors the hallway_melee
// tree — the original default for unnamed defs in buildTree.
func testTreeData() any {
	return map[string]any{
		"reactive_selector": []any{
			map[string]any{"sequence": []any{"is_dead", "stop"}},
			map[string]any{"sequence": []any{"phase_transitioning", "wait_transition"}},
			map[string]any{"sequence": []any{"!has_target", "aggro_or_patrol"}},
			map[string]any{"sequence": []any{"!in_leash_range", "leash_reset"}},
			map[string]any{"sequence": []any{"target_in_melee_range", "has_los", "attack"}},
			"chase",
		},
	}
}

func testDef() *EnemyDef {
	return &EnemyDef{
		Name:       "test_enemy",
		MaxHealth:  1000,
		MoveSpeed:  4.0,
		Radius:     1.0,
		AntiRepeat: 2.0,
		TreeData:   testTreeData(),
		Abilities: []AbilityDef{
			{
				Name: "melee", Type: AbilityMelee,
				TelegraphTime: 1.0, CooldownTime: 1.0,
				BaseWeight: 50, MaxRange: 3.0,
				MeleeRange: 3.0, MeleeDamage: 30.0, MeleeConeAngle: math.Pi,
				DamageSourceType: SourceEnemyMelee,
			},
			{
				Name: "ranged", Type: AbilityRanged,
				TelegraphTime: 0.8, CooldownTime: 1.0,
				BaseWeight: 50, MinRange: 3.0,
				ProjectileCount: 1, ProjectileSpeed: 20.0,
				ProjectileDamage: 15.0, ProjectileOriginY: 1.5,
				ProjectileLifetime: 5.0,
				DamageSourceType:   SourceEnemyRanged,
			},
			{
				Name: "aoe", Type: AbilityAoE,
				TelegraphTime: 1.2, CooldownTime: 1.5,
				BaseWeight: 30, MaxRange: 7.0,
				AoERadius: 5.0, AoEDamage: 40.0,
				DamageSourceType: SourceEnemyAoE,
			},
			{
				Name: "charge", Type: AbilityCharge,
				TelegraphTime: 1.0, CooldownTime: 1.5,
				BaseWeight: 20, MinRange: 6.0,
				ChargeSpeed: 12.0, ChargeDamage: 35.0,
				ChargeMaxDistance: 15.0, ChargeHitRadius: 2.0,
				ChargeStopOnWall: true, ChargeStopOnObstacle: true,
				DamageSourceType: SourceEnemyCharge,
			},
		},
		Phases: []PhaseDef{
			{
				HPThresholdPct:   0.6,
				MoveSpeed:        5.0,
				CooldownOverride: 0.8,
				WeightOverrides:  map[string]int{"melee": 25, "ranged": 25, "aoe": 25, "charge": 25},
				AbilityOverrides: map[string]AbilityOverride{
					"melee": {TelegraphTime: F32(0.8), Damage: F32(35.0)},
				},
			},
		},
	}
}

func testPlayer(id uint16, pos entity.Vec3) *entity.Player {
	p := entity.NewPlayer(id, entity.ClassGunner)
	p.Position = pos
	return p
}

func testPlayers(players ...*entity.Player) []*entity.Player {
	return players
}

// --- Targeting tests ---

func TestNearestAlivePlayer(t *testing.T) {
	p1 := testPlayer(1, entity.Vec3{X: 5, Z: 5})
	p2 := testPlayer(2, entity.Vec3{X: 1, Z: 1})
	p3 := testPlayer(3, entity.Vec3{X: 10, Z: 10})
	p3.Alive = false

	result := NearestAlivePlayer(entity.Vec3{}, testPlayers(p1, p2, p3))
	if result == nil || result.ID != 2 {
		t.Errorf("expected player 2 (nearest), got %v", result)
	}
}

func TestNearestAlivePlayerNoneAlive(t *testing.T) {
	p1 := testPlayer(1, entity.Vec3{X: 1})
	p1.Alive = false
	if got := NearestAlivePlayer(entity.Vec3{}, testPlayers(p1)); got != nil {
		t.Errorf("expected nil, got peer %d", got.ID)
	}
}

func TestNearestAlivePlayerEmpty(t *testing.T) {
	if got := NearestAlivePlayer(entity.Vec3{}, nil); got != nil {
		t.Error("expected nil for empty slice")
	}
}

func TestFarthestAlivePlayer(t *testing.T) {
	p1 := testPlayer(1, entity.Vec3{X: 1, Z: 1})
	p2 := testPlayer(2, entity.Vec3{X: 10, Z: 10})
	p3 := testPlayer(3, entity.Vec3{X: 20, Z: 20})
	p3.Alive = false

	result := FarthestAlivePlayer(entity.Vec3{}, testPlayers(p1, p2, p3))
	if result == nil || result.ID != 2 {
		t.Errorf("expected player 2 (farthest alive), got %v", result)
	}
}

// --- EnemyDef tests ---

func TestCurrentPhase(t *testing.T) {
	def := testDef()
	if def.CurrentPhase(1) != nil {
		t.Error("phase 1 should return nil")
	}
	p2 := def.CurrentPhase(2)
	if p2 == nil {
		t.Fatal("phase 2 should not be nil")
	}
	if p2.MoveSpeed != 5.0 {
		t.Errorf("phase 2 move speed = %f, want 5.0", p2.MoveSpeed)
	}
}

func TestResolveAbilityBase(t *testing.T) {
	def := testDef()
	melee := &def.Abilities[0]
	resolved := def.ResolveAbility(melee, 1)
	if resolved.TelegraphTime != 1.0 {
		t.Errorf("base telegraph = %f, want 1.0", resolved.TelegraphTime)
	}
	if resolved.MeleeDamage != 30.0 {
		t.Errorf("base damage = %f, want 30.0", resolved.MeleeDamage)
	}
}

func TestResolveAbilityPhaseOverride(t *testing.T) {
	def := testDef()
	melee := &def.Abilities[0]
	resolved := def.ResolveAbility(melee, 2)
	if resolved.TelegraphTime != 0.8 {
		t.Errorf("phase 2 telegraph = %f, want 0.8", resolved.TelegraphTime)
	}
	if resolved.MeleeDamage != 35.0 {
		t.Errorf("phase 2 damage = %f, want 35.0", resolved.MeleeDamage)
	}
}

func TestCurrentMoveSpeed(t *testing.T) {
	def := testDef()
	if got := def.CurrentMoveSpeed(1); got != 4.0 {
		t.Errorf("phase 1 speed = %f, want 4.0", got)
	}
	if got := def.CurrentMoveSpeed(2); got != 5.0 {
		t.Errorf("phase 2 speed = %f, want 5.0", got)
	}
}

func TestCurrentBackpedalSpeed(t *testing.T) {
	def := testDef()
	// No BackpedalSpeed set -> 50% of MoveSpeed
	if got := def.CurrentBackpedalSpeed(1); got != 2.0 {
		t.Errorf("phase 1 backpedal = %f, want 2.0", got)
	}
}

func TestCurrentCooldownTime(t *testing.T) {
	def := testDef()
	melee := &def.Abilities[0]
	if got := def.CurrentCooldownTime(melee, 1); got != 1.0 {
		t.Errorf("phase 1 cooldown = %f, want 1.0", got)
	}
	if got := def.CurrentCooldownTime(melee, 2); got != 0.8 {
		t.Errorf("phase 2 cooldown = %f, want 0.8 (override)", got)
	}
}

func TestLongestMeleeRange(t *testing.T) {
	def := testDef()
	if got := def.LongestMeleeRange(); got != 3.0 {
		t.Errorf("longest melee range = %f, want 3.0", got)
	}
}

func TestAbilityByIndex(t *testing.T) {
	def := testDef()
	if got := def.AbilityByIndex(0); got == nil || got.Name != "melee" { //nolint:goconst // test data
		t.Error("index 0 should be melee")
	}
	if got := def.AbilityByIndex(-1); got != nil {
		t.Error("index -1 should be nil")
	}
	if got := def.AbilityByIndex(100); got != nil {
		t.Error("index 100 should be nil")
	}
}

// --- Brain.Enemy() ---

func TestBrainEnemyReturnsEnemy(t *testing.T) {
	def := testDef()
	b, e := testBrain(def)
	if b.Enemy() != e {
		t.Error("Brain.Enemy() should return the enemy passed to NewBrain")
	}
}

// --- Registry ---

func TestDefRegistryContainsAll(t *testing.T) {
	expected := []string{"guard_captain", "hallway_melee", "hallway_ranged"}
	for _, name := range expected {
		if _, ok := DefRegistry[name]; !ok {
			t.Errorf("DefRegistry missing %s", name)
		}
	}
}

func TestGuardCaptainAbilityCount(t *testing.T) {
	if len(GuardCaptain.Abilities) != 5 {
		t.Errorf("guard captain abilities = %d, want 5", len(GuardCaptain.Abilities))
	}
}

func TestGuardCaptainPhaseOverrides(t *testing.T) {
	// fireball_burst (index 1) phase 2: telegraph shortens
	resolved := GuardCaptain.ResolveAbility(&GuardCaptain.Abilities[1], 2)
	if resolved.TelegraphTime != 0.8 {
		t.Errorf("phase 2 fireball telegraph = %f, want 0.8", resolved.TelegraphTime)
	}
	// void_barrage (index 2) phase 2: telegraph shortens
	vb := GuardCaptain.ResolveAbility(&GuardCaptain.Abilities[2], 2)
	if vb.TelegraphTime != 1.0 {
		t.Errorf("phase 2 void_barrage telegraph = %f, want 1.0", vb.TelegraphTime)
	}
}

// --- FaceToward ---

func TestFaceTowardRight(t *testing.T) {
	from := entity.Vec3{X: 0, Y: 0, Z: 0}
	to := entity.Vec3{X: 5, Y: 0, Z: 0}
	yaw := FaceToward(from, to)
	// Facing +X -> atan2(-5, 0) = -pi/2
	expected := float32(math.Atan2(-5, 0))
	if diff := yaw - expected; diff > 0.01 || diff < -0.01 {
		t.Errorf("FaceToward(right) = %f, want %f", yaw, expected)
	}
}

func TestFaceTowardForward(t *testing.T) {
	from := entity.Vec3{X: 0, Y: 0, Z: 0}
	to := entity.Vec3{X: 0, Y: 0, Z: -5}
	yaw := FaceToward(from, to)
	// Facing -Z -> atan2(0, 5) = 0
	if yaw > 0.01 || yaw < -0.01 {
		t.Errorf("FaceToward(forward) = %f, want ~0", yaw)
	}
}

func TestFaceTowardSamePosition(t *testing.T) {
	pos := entity.Vec3{X: 3, Y: 1, Z: 5}
	yaw := FaceToward(pos, pos)
	if yaw != 0 {
		t.Errorf("FaceToward(same) = %f, want 0", yaw)
	}
}
