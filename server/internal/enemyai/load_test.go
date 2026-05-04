package enemyai

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"codex-online/server/internal/bt"
	"codex-online/server/internal/entity"
)

// TestLoadMobs_HallwayMeleeFields verifies the YAML-loaded hallway_melee def
// matches the expected stats and abilities exactly.
func TestLoadMobs_HallwayMeleeFields(t *testing.T) {
	def := DefRegistry["hallway_melee"]
	if def == nil {
		t.Fatal("hallway_melee not in DefRegistry after LoadMobs")
	}

	if def.Name != "hallway_melee" {
		t.Errorf("Name = %q, want hallway_melee", def.Name)
	}
	if def.MaxHealth != 200 {
		t.Errorf("MaxHealth = %f, want 200", def.MaxHealth)
	}
	if def.MoveSpeed != 5.0 {
		t.Errorf("MoveSpeed = %f, want 5", def.MoveSpeed)
	}
	if def.Radius != 0.8 {
		t.Errorf("Radius = %f, want 0.8", def.Radius)
	}
	if def.AntiRepeat != 1.0 {
		t.Errorf("AntiRepeat = %f, want 1", def.AntiRepeat)
	}
	if def.PreferredRange != 0 {
		t.Errorf("PreferredRange = %f, want 0", def.PreferredRange)
	}
	if len(def.Abilities) != 1 {
		t.Fatalf("Abilities len = %d, want 1", len(def.Abilities))
	}

	a := def.Abilities[0]
	if a.Name != "melee_slash" {
		t.Errorf("ability Name = %q, want melee_slash", a.Name)
	}
	if a.Type != AbilityMelee {
		t.Errorf("ability Type = %d, want %d (melee)", a.Type, AbilityMelee)
	}
	if a.TargetStrategy != TargetNearest {
		t.Errorf("ability TargetStrategy = %d, want %d (nearest)", a.TargetStrategy, TargetNearest)
	}
	if a.TelegraphTime != 0.5 {
		t.Errorf("TelegraphTime = %f, want 0.5", a.TelegraphTime)
	}
	if a.ExecuteTime != 0.2 {
		t.Errorf("ExecuteTime = %f, want 0.2", a.ExecuteTime)
	}
	if a.CooldownTime != 0.4 {
		t.Errorf("CooldownTime = %f, want 0.4", a.CooldownTime)
	}
	if a.BaseWeight != 100 {
		t.Errorf("BaseWeight = %d, want 100", a.BaseWeight)
	}
	if a.MaxRange != 2.5 {
		t.Errorf("MaxRange = %f, want 2.5", a.MaxRange)
	}
	if !a.FaceTarget {
		t.Error("FaceTarget should be true")
	}
	if a.MeleeRange != 2.5 {
		t.Errorf("MeleeRange = %f, want 2.5", a.MeleeRange)
	}
	if a.MeleeDamage != 15 {
		t.Errorf("MeleeDamage = %f, want 15", a.MeleeDamage)
	}

	// 120 degrees → radians
	expectedCone := float32(120.0 * math.Pi / 180.0)
	if diff := a.MeleeConeAngle - expectedCone; diff > 0.001 || diff < -0.001 {
		t.Errorf("MeleeConeAngle = %f, want %f (120 deg)", a.MeleeConeAngle, expectedCone)
	}
	if a.DamageSourceType != SourceEnemyMelee {
		t.Errorf("DamageSourceType = %d, want %d", a.DamageSourceType, SourceEnemyMelee)
	}
}

// TestLoadMobs_HallwayRangedFields verifies the YAML-loaded hallway_ranged def.
func TestLoadMobs_HallwayRangedFields(t *testing.T) {
	def := DefRegistry["hallway_ranged"]
	if def == nil {
		t.Fatal("hallway_ranged not in DefRegistry after LoadMobs")
	}

	if def.MaxHealth != 150 {
		t.Errorf("MaxHealth = %f, want 150", def.MaxHealth)
	}
	if def.MoveSpeed != 3.5 {
		t.Errorf("MoveSpeed = %f, want 3.5", def.MoveSpeed)
	}
	if def.PreferredRange != 8.0 {
		t.Errorf("PreferredRange = %f, want 8", def.PreferredRange)
	}
	if def.BackpedalSpeed != 3.0 {
		t.Errorf("BackpedalSpeed = %f, want 3", def.BackpedalSpeed)
	}
	if len(def.Abilities) != 1 {
		t.Fatalf("Abilities len = %d, want 1", len(def.Abilities))
	}

	a := def.Abilities[0]
	if a.Name != "energy_bolt" {
		t.Errorf("ability Name = %q, want energy_bolt", a.Name)
	}
	if a.Type != AbilityRanged {
		t.Errorf("ability Type = %d, want %d (ranged)", a.Type, AbilityRanged)
	}
	if a.ProjectileCount != 1 {
		t.Errorf("ProjectileCount = %d, want 1", a.ProjectileCount)
	}
	if a.ProjectileSpeed != 18.0 {
		t.Errorf("ProjectileSpeed = %f, want 18", a.ProjectileSpeed)
	}
	if a.ProjectileDamage != 12.0 {
		t.Errorf("ProjectileDamage = %f, want 12", a.ProjectileDamage)
	}
	if !a.TrackTarget {
		t.Error("TrackTarget should be true")
	}
	if a.DamageSourceType != SourceEnemyRanged {
		t.Errorf("DamageSourceType = %d, want %d", a.DamageSourceType, SourceEnemyRanged)
	}
}

// TestLoadMobs_TreeDataPresent verifies YAML-loaded defs have TreeData set.
func TestLoadMobs_TreeDataPresent(t *testing.T) {
	for _, name := range []string{"hallway_melee", "hallway_ranged"} {
		def := DefRegistry[name]
		if def == nil {
			t.Errorf("%s missing from DefRegistry", name)
			continue
		}
		if def.TreeData == nil {
			t.Errorf("%s.TreeData is nil, should be set from YAML", name)
		}
	}
}

// TestLoadMobs_GuardCaptainUntouched verifies the Go-defined boss is unaffected.
func TestLoadMobs_GuardCaptainUntouched(t *testing.T) {
	def := DefRegistry["guard_captain"]
	if def == nil {
		t.Fatal("guard_captain missing from DefRegistry")
	}
	if def.TreeData != nil {
		t.Error("guard_captain.TreeData should be nil (Go-defined)")
	}
	if def.MaxHealth != 2000 {
		t.Errorf("guard_captain MaxHealth = %f, want 2000", def.MaxHealth)
	}
}

// TestBuildTreeFromData_Melee verifies a tree built from data ticks correctly.
func TestBuildTreeFromData_Melee(t *testing.T) {
	def := DefRegistry["hallway_melee"]
	if def == nil {
		t.Fatal("hallway_melee not loaded")
	}

	e := entity.NewEnemy(0, def.MaxHealth, def.Name)
	e.Alive = true
	e.State = entity.EnemyChase
	e.PatrolA = entity.Vec3{X: -5}
	e.PatrolB = entity.Vec3{X: 5}
	e.AggroRadius = 8.0
	e.LeashRadius = 20.0
	e.LeashOrigin = entity.Vec3{}

	b := NewBrainSeeded(def, e, nil, 42)
	b.BoundsMinX = -20
	b.BoundsMaxX = 20
	b.BoundsMinZ = -15
	b.BoundsMaxZ = 50

	// No players: should patrol
	e.State = entity.EnemyPatrol
	b.Tick(0.05, nil, nil, noSpawn, nil)
	if e.State != entity.EnemyPatrol {
		t.Errorf("expected patrol with no players, got state %d", e.State)
	}

	// Add player in aggro range: should chase
	p := testPlayer(1, entity.Vec3{X: 0, Z: 5})
	players := testPlayers(p)
	b.Tick(0.05, players, nil, noSpawn, nil)
	if e.State != entity.EnemyChase {
		t.Errorf("expected chase after aggro, got state %d", e.State)
	}

	// Move player to melee range: should telegraph
	p.Position = entity.Vec3{X: 0, Z: 1}
	for range 100 {
		b.Tick(0.05, players, nil, noSpawn, nil)
		if e.State == entity.EnemyMeleeTelegraph {
			break
		}
	}
	if e.State != entity.EnemyMeleeTelegraph {
		t.Errorf("expected melee telegraph, got state %d", e.State)
	}
}

// TestBuildTreeFromData_ErrorUnknownLeaf verifies unknown leaves fail at parse time.
func TestBuildTreeFromData_ErrorUnknownLeaf(t *testing.T) {
	yaml := []byte(`
name: bad_mob
tier: 1
max_health: 100
move_speed: 4.0
radius: 0.8
abilities:
  - name: hit
    type: melee
    melee_range: 2.0
    melee_damage: 10
    base_weight: 100
tree:
  sequence: [does_not_exist, stop]
`)
	_, err := parseMobYAML(yaml)
	if err == nil {
		t.Fatal("expected error for unknown leaf 'does_not_exist'")
	}
}

// TestBuildTreeFromData_ErrorMalformed verifies malformed YAML fails.
func TestBuildTreeFromData_ErrorMalformed(t *testing.T) {
	_, err := parseMobYAML([]byte(`not: valid: yaml: {{`))
	if err == nil {
		t.Fatal("expected error for malformed YAML")
	}
}

// TestBuildTreeFromData_ErrorMissingName verifies missing name fails.
func TestBuildTreeFromData_ErrorMissingName(t *testing.T) {
	yaml := []byte(`
tier: 1
max_health: 100
move_speed: 4.0
radius: 0.8
tree:
  sequence: [stop]
`)
	_, err := parseMobYAML(yaml)
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

// TestBuildTreeFromData_ErrorMissingTree verifies missing tree fails.
func TestBuildTreeFromData_ErrorMissingTree(t *testing.T) {
	yaml := []byte(`
name: no_tree_mob
tier: 1
max_health: 100
move_speed: 4.0
radius: 0.8
`)
	_, err := parseMobYAML(yaml)
	if err == nil {
		t.Fatal("expected error for missing tree")
	}
}

// TestResolveLeaf_Inverter verifies "!" prefix wraps in Inverter.
func TestResolveLeaf_Inverter(t *testing.T) {
	node, err := resolveLeaf("!is_dead")
	if err != nil {
		t.Fatalf("resolveLeaf: %v", err)
	}
	// Inverter wrapping is_dead: alive enemy should get Success (inverted from Failure)
	e := entity.NewEnemy(0, 100, "test")
	e.Alive = true
	ctx := &EntityContext{Enemy: e, Def: &EnemyDef{}}
	result := node.Tick(ctx)
	if result != bt.Success {
		t.Errorf("!is_dead on alive enemy = %v, want Success", result)
	}
}

// TestResolveLeaf_Parameterized verifies parameterized leaf parsing.
func TestResolveLeaf_Parameterized(t *testing.T) {
	node, err := resolveLeaf("player_nearby(5)")
	if err != nil {
		t.Fatalf("resolveLeaf: %v", err)
	}

	e := entity.NewEnemy(0, 100, "test")
	e.Alive = true
	p := testPlayer(1, entity.Vec3{X: 0, Z: 3})
	ctx := &EntityContext{Enemy: e, Def: &EnemyDef{}, Players: testPlayers(p)}
	result := node.Tick(ctx)
	if result != bt.Success {
		t.Errorf("player_nearby(5) with player at Z=3 = %v, want Success", result)
	}
}

// TestResolveLeaf_UnknownErrors verifies unknown leaf names return errors.
func TestResolveLeaf_UnknownErrors(t *testing.T) {
	_, err := resolveLeaf("totally_made_up")
	if err == nil {
		t.Error("expected error for unknown leaf")
	}
}

// TestBuildTreeFromData_NestedComposites verifies deeply nested tree building.
func TestBuildTreeFromData_NestedComposites(t *testing.T) {
	data := map[string]any{
		"selector": []any{
			map[string]any{
				"sequence": []any{
					"is_dead",
					"stop",
				},
			},
			"chase",
		},
	}
	node, err := buildTreeFromData(data)
	if err != nil {
		t.Fatalf("buildTreeFromData: %v", err)
	}

	// Tick with alive enemy — should reach chase (Selector tries sequence which fails on is_dead)
	e := entity.NewEnemy(0, 100, "test")
	e.Alive = true
	e.State = entity.EnemyChase
	p := testPlayer(1, entity.Vec3{X: 0, Z: 10})
	ctx := &EntityContext{
		Enemy:   e,
		Def:     &EnemyDef{MoveSpeed: 4},
		Players: testPlayers(p),
		Dt:      0.05,
		BB:      NewBlackboard(),
	}
	result := node.Tick(ctx)
	if result != bt.Running {
		t.Errorf("alive enemy in selector should reach chase (Running), got %v", result)
	}
}

// TestLoadMobs_PhaseOverrides verifies YAML phase definitions are parsed correctly.
func TestLoadMobs_PhaseParsing(t *testing.T) {
	yaml := []byte(`
name: phased_mob
tier: 1
max_health: 1000
move_speed: 4.0
radius: 1.0
abilities:
  - name: slash
    type: melee
    telegraph_time: 0.5
    cooldown_time: 0.5
    base_weight: 100
    max_range: 3.0
    melee_range: 3.0
    melee_damage: 20
    damage_source: 1
phases:
  - hp_threshold_pct: 0.6
    transition_time: 1.5
    move_speed: 5.0
    cooldown_override: 0.8
    weight_overrides:
      slash: 100
    ability_overrides:
      slash:
        telegraph_time: 0.3
        damage: 30
tree:
  sequence: [is_dead, stop]
`)
	def, err := parseMobYAML(yaml)
	if err != nil {
		t.Fatalf("parseMobYAML: %v", err)
	}

	if len(def.Phases) != 1 {
		t.Fatalf("Phases len = %d, want 1", len(def.Phases))
	}
	ph := def.Phases[0]
	if ph.HPThresholdPct != 0.6 {
		t.Errorf("HPThresholdPct = %f, want 0.6", ph.HPThresholdPct)
	}
	if ph.TransitionTime != 1.5 {
		t.Errorf("TransitionTime = %f, want 1.5", ph.TransitionTime)
	}
	if ph.MoveSpeed != 5.0 {
		t.Errorf("MoveSpeed = %f, want 5", ph.MoveSpeed)
	}
	if ph.CooldownOverride != 0.8 {
		t.Errorf("CooldownOverride = %f, want 0.8", ph.CooldownOverride)
	}
	if ph.WeightOverrides["slash"] != 100 {
		t.Errorf("WeightOverrides[slash] = %d, want 100", ph.WeightOverrides["slash"])
	}
	ovr := ph.AbilityOverrides["slash"]
	if ovr.TelegraphTime == nil || *ovr.TelegraphTime != 0.3 {
		t.Errorf("AbilityOverrides[slash].TelegraphTime = %v, want 0.3", ovr.TelegraphTime)
	}
	if ovr.Damage == nil || *ovr.Damage != 30 {
		t.Errorf("AbilityOverrides[slash].Damage = %v, want 30", ovr.Damage)
	}
}

// --- resolveLeaf exhaustive tests ---

// TestResolveLeaf_AllSimpleLeaves verifies every entry in leafRegistry resolves.
func TestResolveLeaf_AllSimpleLeaves(t *testing.T) {
	for name := range leafRegistry {
		t.Run(name, func(t *testing.T) {
			node, err := resolveLeaf(name)
			if err != nil {
				t.Fatalf("resolveLeaf(%q): %v", name, err)
			}
			if node == nil {
				t.Fatalf("resolveLeaf(%q) returned nil node", name)
			}
		})
	}
}

// TestResolveLeaf_BuiltinSubtrees verifies attack and aggro_or_patrol resolve.
func TestResolveLeaf_BuiltinSubtrees(t *testing.T) {
	for _, name := range []string{"attack", "aggro_or_patrol"} {
		t.Run(name, func(t *testing.T) {
			node, err := resolveLeaf(name)
			if err != nil {
				t.Fatalf("resolveLeaf(%q): %v", name, err)
			}
			if node == nil {
				t.Fatalf("resolveLeaf(%q) returned nil node", name)
			}
		})
	}
}

// TestResolveLeaf_InverterBuiltin verifies "!" prefix works with subtrees.
func TestResolveLeaf_InverterBuiltin(t *testing.T) {
	// Should not error — inverting a subtree is valid
	node, err := resolveLeaf("!has_target")
	if err != nil {
		t.Fatalf("resolveLeaf(!has_target): %v", err)
	}
	if node == nil {
		t.Fatal("expected non-nil node")
	}
}

// TestResolveLeaf_ParamInvalidArgs verifies bad params return errors.
func TestResolveLeaf_ParamInvalidArgs(t *testing.T) {
	cases := []struct {
		name    string
		leaf    string
	}{
		{"player_nearby non-numeric", "player_nearby(abc)"},
		{"phase_eq non-numeric", "phase_eq(abc)"},
		{"player_nearby empty", "player_nearby()"},
		{"unknown parameterized", "unknown_factory(5)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := resolveLeaf(tc.leaf)
			if err == nil {
				t.Errorf("expected error for %q", tc.leaf)
			}
		})
	}
}

// TestResolveLeaf_PhaseEq verifies phase_eq parameterized condition.
func TestResolveLeaf_PhaseEq(t *testing.T) {
	node, err := resolveLeaf("phase_eq(1)")
	if err != nil {
		t.Fatalf("resolveLeaf: %v", err)
	}
	e := entity.NewEnemy(0, 100, "test")
	e.Alive = true
	e.Phase = 1
	ctx := &EntityContext{Enemy: e, Def: &EnemyDef{}}
	result := node.Tick(ctx)
	if result != bt.Success {
		t.Errorf("phase_eq(1) with phase=1 = %v, want Success", result)
	}
	e.Phase = 2
	result = node.Tick(ctx)
	if result != bt.Failure {
		t.Errorf("phase_eq(1) with phase=2 = %v, want Failure", result)
	}
}

// --- buildTreeFromData exhaustive tests ---

// TestBuildTreeFromData_AllCompositeTypes tests each composite type individually.
func TestBuildTreeFromData_AllCompositeTypes(t *testing.T) {
	composites := []struct {
		name string
		data map[string]any
	}{
		{"sequence", map[string]any{"sequence": []any{"is_dead", "stop"}}},
		{"selector", map[string]any{"selector": []any{"is_dead", "stop"}}},
		{"reactive_selector", map[string]any{"reactive_selector": []any{"is_dead", "stop"}}},
	}
	for _, tc := range composites {
		t.Run(tc.name, func(t *testing.T) {
			node, err := buildTreeFromData(tc.data)
			if err != nil {
				t.Fatalf("buildTreeFromData: %v", err)
			}
			if node == nil {
				t.Fatal("expected non-nil node")
			}
		})
	}
}

// TestBuildTreeFromData_ErrorMultipleKeys verifies map with >1 key fails.
func TestBuildTreeFromData_ErrorMultipleKeys(t *testing.T) {
	data := map[string]any{
		"sequence": []any{"stop"},
		"selector": []any{"stop"},
	}
	_, err := buildTreeFromData(data)
	if err == nil {
		t.Fatal("expected error for map with multiple keys")
	}
	if !strings.Contains(err.Error(), "exactly one key") {
		t.Errorf("error = %q, want mention of 'exactly one key'", err.Error())
	}
}

// TestBuildTreeFromData_ErrorUnknownComposite verifies unknown composite type fails.
func TestBuildTreeFromData_ErrorUnknownComposite(t *testing.T) {
	data := map[string]any{
		"parallel": []any{"stop"},
	}
	_, err := buildTreeFromData(data)
	if err == nil {
		t.Fatal("expected error for unknown composite 'parallel'")
	}
	if !strings.Contains(err.Error(), "unknown composite") {
		t.Errorf("error = %q, want mention of 'unknown composite'", err.Error())
	}
}

// TestBuildTreeFromData_ErrorChildrenNotList verifies non-list children fail.
func TestBuildTreeFromData_ErrorChildrenNotList(t *testing.T) {
	data := map[string]any{
		"sequence": "stop",
	}
	_, err := buildTreeFromData(data)
	if err == nil {
		t.Fatal("expected error for non-list children")
	}
	if !strings.Contains(err.Error(), "must be a list") {
		t.Errorf("error = %q, want mention of 'must be a list'", err.Error())
	}
}

// TestBuildTreeFromData_ErrorUnexpectedType verifies unexpected types (int, float) fail.
func TestBuildTreeFromData_ErrorUnexpectedType(t *testing.T) {
	_, err := buildTreeFromData(42)
	if err == nil {
		t.Fatal("expected error for integer tree node")
	}
	if !strings.Contains(err.Error(), "unexpected tree node type") {
		t.Errorf("error = %q, want mention of 'unexpected tree node type'", err.Error())
	}
}

// TestBuildTreeFromData_StringLeaf verifies a single string resolves.
func TestBuildTreeFromData_StringLeaf(t *testing.T) {
	node, err := buildTreeFromData("chase")
	if err != nil {
		t.Fatalf("buildTreeFromData: %v", err)
	}
	if node == nil {
		t.Fatal("expected non-nil node")
	}
}

// --- convertAbility tests ---

// TestConvertAbility_AllTypes verifies all 4 ability types parse correctly.
func TestConvertAbility_AllTypes(t *testing.T) {
	cases := []struct {
		typeStr string
		want    AbilityType
	}{
		{"melee", AbilityMelee},
		{"ranged", AbilityRanged},
		{"aoe", AbilityAoE},
		{"charge", AbilityCharge},
	}
	for _, tc := range cases {
		t.Run(tc.typeStr, func(t *testing.T) {
			af := abilityFile{Name: "test", Type: tc.typeStr, BaseWeight: 100}
			ad, err := convertAbility(af)
			if err != nil {
				t.Fatalf("convertAbility: %v", err)
			}
			if ad.Type != tc.want {
				t.Errorf("Type = %d, want %d", ad.Type, tc.want)
			}
		})
	}
}

// TestConvertAbility_AllTargetStrategies verifies all 3 target strategies.
func TestConvertAbility_AllTargetStrategies(t *testing.T) {
	cases := []struct {
		targetStr string
		want      TargetStrategy
	}{
		{"nearest", TargetNearest},
		{"", TargetNearest}, // default
		{"farthest", TargetFarthest},
		{"current", TargetCurrent},
	}
	for _, tc := range cases {
		name := tc.targetStr
		if name == "" {
			name = "empty_default"
		}
		t.Run(name, func(t *testing.T) {
			af := abilityFile{Name: "test", Type: "melee", Target: tc.targetStr, BaseWeight: 100}
			ad, err := convertAbility(af)
			if err != nil {
				t.Fatalf("convertAbility: %v", err)
			}
			if ad.TargetStrategy != tc.want {
				t.Errorf("TargetStrategy = %d, want %d", ad.TargetStrategy, tc.want)
			}
		})
	}
}

// TestConvertAbility_ErrorUnknownType verifies unknown ability type fails.
func TestConvertAbility_ErrorUnknownType(t *testing.T) {
	af := abilityFile{Name: "bad", Type: "beam", BaseWeight: 100}
	_, err := convertAbility(af)
	if err == nil {
		t.Fatal("expected error for unknown ability type")
	}
}

// TestConvertAbility_ErrorUnknownTarget verifies unknown target strategy fails.
func TestConvertAbility_ErrorUnknownTarget(t *testing.T) {
	af := abilityFile{Name: "bad", Type: "melee", Target: "random", BaseWeight: 100}
	_, err := convertAbility(af)
	if err == nil {
		t.Fatal("expected error for unknown target strategy")
	}
}

// TestConvertAbility_DegreeConversions verifies cone and spread angle conversions.
func TestConvertAbility_DegreeConversions(t *testing.T) {
	af := abilityFile{
		Name: "test", Type: "ranged", BaseWeight: 100,
		MeleeConeDeg:        90,
		ProjectileSpreadDeg: 30,
	}
	ad, err := convertAbility(af)
	if err != nil {
		t.Fatalf("convertAbility: %v", err)
	}
	expectedCone := float32(90.0 * math.Pi / 180.0)
	if diff := ad.MeleeConeAngle - expectedCone; diff > 0.001 || diff < -0.001 {
		t.Errorf("MeleeConeAngle = %f, want %f", ad.MeleeConeAngle, expectedCone)
	}
	expectedSpread := float32(30.0 * math.Pi / 180.0)
	if diff := ad.ProjectileSpread - expectedSpread; diff > 0.001 || diff < -0.001 {
		t.Errorf("ProjectileSpread = %f, want %f", ad.ProjectileSpread, expectedSpread)
	}
}

// TestConvertAbility_ChargeFields verifies charge-specific fields.
func TestConvertAbility_ChargeFields(t *testing.T) {
	af := abilityFile{
		Name: "rush", Type: "charge", BaseWeight: 100,
		ChargeSpeed: 15, ChargeDamage: 40, ChargeMaxDistance: 20,
		ChargeHitRadius: 3, ChargeStopOnWall: true, ChargeStopOnObstacle: true,
		DamageSource: SourceEnemyCharge,
	}
	ad, err := convertAbility(af)
	if err != nil {
		t.Fatalf("convertAbility: %v", err)
	}
	if ad.ChargeSpeed != 15 {
		t.Errorf("ChargeSpeed = %f, want 15", ad.ChargeSpeed)
	}
	if ad.ChargeDamage != 40 {
		t.Errorf("ChargeDamage = %f, want 40", ad.ChargeDamage)
	}
	if ad.ChargeMaxDistance != 20 {
		t.Errorf("ChargeMaxDistance = %f, want 20", ad.ChargeMaxDistance)
	}
	if ad.ChargeHitRadius != 3 {
		t.Errorf("ChargeHitRadius = %f, want 3", ad.ChargeHitRadius)
	}
	if !ad.ChargeStopOnWall {
		t.Error("ChargeStopOnWall should be true")
	}
	if !ad.ChargeStopOnObstacle {
		t.Error("ChargeStopOnObstacle should be true")
	}
	if ad.DamageSourceType != SourceEnemyCharge {
		t.Errorf("DamageSourceType = %d, want %d", ad.DamageSourceType, SourceEnemyCharge)
	}
}

// TestConvertAbility_AoEFields verifies aoe-specific fields.
func TestConvertAbility_AoEFields(t *testing.T) {
	af := abilityFile{
		Name: "blast", Type: "aoe", BaseWeight: 100,
		AoERadius: 5, AoEDamage: 25, DamageSource: SourceEnemyAoE,
	}
	ad, err := convertAbility(af)
	if err != nil {
		t.Fatalf("convertAbility: %v", err)
	}
	if ad.AoERadius != 5 {
		t.Errorf("AoERadius = %f, want 5", ad.AoERadius)
	}
	if ad.AoEDamage != 25 {
		t.Errorf("AoEDamage = %f, want 25", ad.AoEDamage)
	}
}

// --- LoadMobs error/edge cases ---

// TestLoadMobs_ErrorMissingDir verifies missing directory returns error.
func TestLoadMobs_ErrorMissingDir(t *testing.T) {
	err := LoadMobs("/nonexistent/path/to/mobs")
	if err == nil {
		t.Fatal("expected error for missing directory")
	}
}

// TestLoadMobs_SkipsNonYAML verifies non-.yaml files and dirs are skipped.
func TestLoadMobs_SkipsNonYAML(t *testing.T) {
	dir := t.TempDir()

	// Write a valid YAML mob
	validYAML := []byte(`
name: temp_mob
tier: 1
max_health: 100
move_speed: 4.0
radius: 0.8
tree:
  sequence: [is_dead, stop]
`)
	if err := os.WriteFile(filepath.Join(dir, "valid.yaml"), validYAML, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("not a mob"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "subdir"), 0755); err != nil {
		t.Fatal(err)
	}

	// Save and restore DefRegistry state
	prev := DefRegistry["temp_mob"]
	defer func() {
		if prev != nil {
			DefRegistry["temp_mob"] = prev
		} else {
			delete(DefRegistry, "temp_mob")
		}
	}()

	err := LoadMobs(dir)
	if err != nil {
		t.Fatalf("LoadMobs: %v", err)
	}
	if DefRegistry["temp_mob"] == nil {
		t.Error("expected temp_mob to be loaded")
	}
}

// TestLoadMobs_ErrorInvalidYAML verifies a bad file in dir fails the whole load.
func TestLoadMobs_ErrorInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "bad.yaml"), []byte(`not: valid: yaml: {{`), 0644); err != nil {
		t.Fatal(err)
	}

	err := LoadMobs(dir)
	if err == nil {
		t.Fatal("expected error for invalid YAML file in dir")
	}
}

// TestLoadMobs_EmptyDir succeeds with no files.
func TestLoadMobs_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	err := LoadMobs(dir)
	if err != nil {
		t.Fatalf("LoadMobs on empty dir: %v", err)
	}
}

// --- parseMobYAML additional tests ---

// TestParseMobYAML_RangedAbility verifies ranged ability fields parse correctly.
func TestParseMobYAML_RangedAbility(t *testing.T) {
	yml := []byte(`
name: ranged_test
tier: 1
max_health: 100
move_speed: 4.0
radius: 0.8
preferred_range: 10
backpedal_speed: 2.5
abilities:
  - name: bolt
    type: ranged
    target: farthest
    telegraph_time: 0.3
    cooldown_time: 0.5
    base_weight: 100
    max_range: 15
    track_target: true
    projectile_count: 3
    projectile_speed: 20
    projectile_damage: 10
    projectile_spread_deg: 15
    projectile_origin_y: 1.5
    projectile_lifetime: 3.0
    damage_source: 2
tree:
  sequence: [is_dead, stop]
`)
	def, err := parseMobYAML(yml)
	if err != nil {
		t.Fatalf("parseMobYAML: %v", err)
	}
	if def.PreferredRange != 10 {
		t.Errorf("PreferredRange = %f, want 10", def.PreferredRange)
	}
	if def.BackpedalSpeed != 2.5 {
		t.Errorf("BackpedalSpeed = %f, want 2.5", def.BackpedalSpeed)
	}
	a := def.Abilities[0]
	if a.Type != AbilityRanged {
		t.Errorf("Type = %d, want %d", a.Type, AbilityRanged)
	}
	if a.TargetStrategy != TargetFarthest {
		t.Errorf("TargetStrategy = %d, want %d", a.TargetStrategy, TargetFarthest)
	}
	if a.ProjectileCount != 3 {
		t.Errorf("ProjectileCount = %d, want 3", a.ProjectileCount)
	}
	if a.ProjectileSpeed != 20 {
		t.Errorf("ProjectileSpeed = %f, want 20", a.ProjectileSpeed)
	}
	if a.ProjectileDamage != 10 {
		t.Errorf("ProjectileDamage = %f, want 10", a.ProjectileDamage)
	}
	expectedSpread := float32(15.0 * math.Pi / 180.0)
	if diff := a.ProjectileSpread - expectedSpread; diff > 0.001 || diff < -0.001 {
		t.Errorf("ProjectileSpread = %f, want %f", a.ProjectileSpread, expectedSpread)
	}
	if a.ProjectileOriginY != 1.5 {
		t.Errorf("ProjectileOriginY = %f, want 1.5", a.ProjectileOriginY)
	}
	if a.ProjectileLifetime != 3.0 {
		t.Errorf("ProjectileLifetime = %f, want 3", a.ProjectileLifetime)
	}
	if !a.TrackTarget {
		t.Error("TrackTarget should be true")
	}
	if a.DamageSourceType != SourceEnemyRanged {
		t.Errorf("DamageSourceType = %d, want %d", a.DamageSourceType, SourceEnemyRanged)
	}
}

// TestParseMobYAML_MultipleAbilities verifies multiple abilities parse.
func TestParseMobYAML_MultipleAbilities(t *testing.T) {
	yml := []byte(`
name: multi_ability
tier: 1
max_health: 500
move_speed: 4.0
radius: 1.0
abilities:
  - name: slash
    type: melee
    base_weight: 60
    melee_range: 2.5
    melee_damage: 20
    damage_source: 1
  - name: bolt
    type: ranged
    base_weight: 40
    projectile_count: 1
    projectile_speed: 15
    projectile_damage: 12
    damage_source: 2
tree:
  sequence: [is_dead, stop]
`)
	def, err := parseMobYAML(yml)
	if err != nil {
		t.Fatalf("parseMobYAML: %v", err)
	}
	if len(def.Abilities) != 2 {
		t.Fatalf("Abilities len = %d, want 2", len(def.Abilities))
	}
	if def.Abilities[0].Name != "slash" {
		t.Errorf("Abilities[0].Name = %q, want slash", def.Abilities[0].Name)
	}
	if def.Abilities[1].Name != "bolt" {
		t.Errorf("Abilities[1].Name = %q, want bolt", def.Abilities[1].Name)
	}
}

// TestParseMobYAML_ErrorBadAbilityType verifies unknown ability type in YAML.
func TestParseMobYAML_ErrorBadAbilityType(t *testing.T) {
	yml := []byte(`
name: bad_type
tier: 1
max_health: 100
move_speed: 4.0
radius: 0.8
abilities:
  - name: laser
    type: beam
    base_weight: 100
tree:
  sequence: [is_dead, stop]
`)
	_, err := parseMobYAML(yml)
	if err == nil {
		t.Fatal("expected error for unknown ability type 'beam'")
	}
}

// TestParseMobYAML_ErrorBadTargetStrategy verifies unknown target in YAML.
func TestParseMobYAML_ErrorBadTargetStrategy(t *testing.T) {
	yml := []byte(`
name: bad_target
tier: 1
max_health: 100
move_speed: 4.0
radius: 0.8
abilities:
  - name: hit
    type: melee
    target: random
    base_weight: 100
tree:
  sequence: [is_dead, stop]
`)
	_, err := parseMobYAML(yml)
	if err == nil {
		t.Fatal("expected error for unknown target strategy 'random'")
	}
}

// TestParseMobYAML_MultiplePhases verifies multiple phase definitions.
func TestParseMobYAML_MultiplePhases(t *testing.T) {
	yml := []byte(`
name: multi_phase
tier: 2
max_health: 2000
move_speed: 4.0
radius: 1.0
abilities:
  - name: slash
    type: melee
    base_weight: 100
    melee_range: 3.0
    melee_damage: 20
    damage_source: 1
phases:
  - hp_threshold_pct: 0.6
    transition_time: 1.0
    move_speed: 5.0
  - hp_threshold_pct: 0.3
    transition_time: 2.0
    move_speed: 6.0
    backpedal_speed: 4.0
    cooldown_override: 0.5
tree:
  sequence: [is_dead, stop]
`)
	def, err := parseMobYAML(yml)
	if err != nil {
		t.Fatalf("parseMobYAML: %v", err)
	}
	if len(def.Phases) != 2 {
		t.Fatalf("Phases len = %d, want 2", len(def.Phases))
	}
	if def.Phases[0].HPThresholdPct != 0.6 {
		t.Errorf("Phase[0].HPThresholdPct = %f, want 0.6", def.Phases[0].HPThresholdPct)
	}
	if def.Phases[1].HPThresholdPct != 0.3 {
		t.Errorf("Phase[1].HPThresholdPct = %f, want 0.3", def.Phases[1].HPThresholdPct)
	}
	if def.Phases[1].BackpedalSpeed != 4.0 {
		t.Errorf("Phase[1].BackpedalSpeed = %f, want 4", def.Phases[1].BackpedalSpeed)
	}
	if def.Phases[1].CooldownOverride != 0.5 {
		t.Errorf("Phase[1].CooldownOverride = %f, want 0.5", def.Phases[1].CooldownOverride)
	}
}

// --- Benchmarks ---

var hallwayMeleeYAML = []byte(`
name: hallway_melee
tier: 1
max_health: 200
move_speed: 5.0
radius: 0.8
anti_repeat: 1.0
abilities:
  - name: melee_slash
    type: melee
    target: nearest
    telegraph_time: 0.5
    execute_time: 0.2
    cooldown_time: 0.4
    base_weight: 100
    max_range: 2.5
    face_target: true
    melee_range: 2.5
    melee_damage: 15
    melee_cone_deg: 120
    damage_source: 1
tree:
  reactive_selector:
    - sequence: [is_dead, stop]
    - sequence: [phase_transitioning, wait_transition]
    - sequence: ["!has_target", aggro_or_patrol]
    - sequence: ["!in_leash_range", leash_reset]
    - sequence: [target_in_melee_range, has_los, attack]
    - chase
`)

func BenchmarkParseMobYAML(b *testing.B) {
	for b.Loop() {
		_, err := parseMobYAML(hallwayMeleeYAML)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBuildTreeFromData(b *testing.B) {
	data := testTreeData()
	b.ResetTimer()
	for b.Loop() {
		_, err := buildTreeFromData(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkResolveLeaf_Simple(b *testing.B) {
	for b.Loop() {
		_, _ = resolveLeaf("chase")
	}
}

func BenchmarkResolveLeaf_Inverter(b *testing.B) {
	for b.Loop() {
		_, _ = resolveLeaf("!has_target")
	}
}

func BenchmarkResolveLeaf_Parameterized(b *testing.B) {
	for b.Loop() {
		_, _ = resolveLeaf("player_nearby(8)")
	}
}

func BenchmarkResolveLeaf_Subtree(b *testing.B) {
	for b.Loop() {
		_, _ = resolveLeaf("attack")
	}
}

func BenchmarkLoadMobs(b *testing.B) {
	dir := b.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "melee.yaml"), hallwayMeleeYAML, 0644); err != nil {
		b.Fatal(err)
	}

	// Save and restore registry state
	prev := DefRegistry["hallway_melee"]
	b.Cleanup(func() {
		if prev != nil {
			DefRegistry["hallway_melee"] = prev
		}
	})

	b.ResetTimer()
	for b.Loop() {
		_ = LoadMobs(dir)
	}
}

func BenchmarkBrainTick_YAMLMelee(b *testing.B) {
	def := DefRegistry["hallway_melee"]
	if def == nil {
		b.Fatal("hallway_melee not loaded")
	}

	br, e := testBrain(def)
	e.Alive = true
	e.State = entity.EnemyChase
	e.PatrolA = entity.Vec3{X: -5}
	e.PatrolB = entity.Vec3{X: 5}
	e.AggroRadius = 8.0
	e.LeashRadius = 20.0
	e.TargetPlayerID = 1

	p := testPlayer(1, entity.Vec3{X: 0, Z: 1})
	p.Health = 1e9
	players := testPlayers(p)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		br.Tick(0.05, players, nil, noSpawn, nil)
	}
}

func BenchmarkBrainTick_YAMLRanged(b *testing.B) {
	def := DefRegistry["hallway_ranged"]
	if def == nil {
		b.Fatal("hallway_ranged not loaded")
	}

	br, e := testBrain(def)
	e.Alive = true
	e.State = entity.EnemyChase
	e.PatrolA = entity.Vec3{X: -5}
	e.PatrolB = entity.Vec3{X: 5}
	e.AggroRadius = 8.0
	e.LeashRadius = 20.0
	e.TargetPlayerID = 1

	p := testPlayer(1, entity.Vec3{X: 0, Z: 12})
	p.Health = 1e9
	players := testPlayers(p)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		br.Tick(0.05, players, nil, noSpawn, nil)
	}
}
