package enemyai

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"codex-online/server/internal/ability"
	"codex-online/server/internal/bt"
	"codex-online/server/internal/combat"
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
	if def.MaxHealth != 300 {
		t.Errorf("MaxHealth = %f, want 300", def.MaxHealth)
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
	if a.Category != ability.CategoryMelee {
		t.Errorf("ability Category = %d, want %d (melee)", a.Category, ability.CategoryMelee)
	}
	if a.TargetStrategy != ability.TargetNearest {
		t.Errorf("ability TargetStrategy = %d, want %d (nearest)", a.TargetStrategy, ability.TargetNearest)
	}
	if a.CommitTime != 0.3 {
		t.Errorf("CommitTime = %f, want 0.3", a.CommitTime)
	}
	if a.ExecuteTime != 0.2 {
		t.Errorf("ExecuteTime = %f, want 0.2", a.ExecuteTime)
	}
	if a.Cooldown != 0.4 {
		t.Errorf("Cooldown = %f, want 0.4", a.Cooldown)
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
	if a.Hit.Range != 2.5 {
		t.Errorf("Hit.Range = %f, want 2.5", a.Hit.Range)
	}
	if a.BaseDamage != 54 {
		t.Errorf("BaseDamage = %f, want 54", a.BaseDamage)
	}

	// 120 degrees stored as degrees
	if a.Hit.ArcDegrees != 120 {
		t.Errorf("Hit.ArcDegrees = %f, want 120", a.Hit.ArcDegrees)
	}
	if a.DamageSource != combat.SourceEnemyMelee {
		t.Errorf("DamageSource = %d, want %d", a.DamageSource, combat.SourceEnemyMelee)
	}
}

// TestLoadMobs_HallwayRangedFields verifies the YAML-loaded hallway_ranged def.
func TestLoadMobs_HallwayRangedFields(t *testing.T) {
	def := DefRegistry["hallway_ranged"]
	if def == nil {
		t.Fatal("hallway_ranged not in DefRegistry after LoadMobs")
	}

	if def.MaxHealth != 210 {
		t.Errorf("MaxHealth = %f, want 210", def.MaxHealth)
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
	if a.Category != ability.CategoryRanged {
		t.Errorf("ability Category = %d, want %d (ranged)", a.Category, ability.CategoryRanged)
	}
	// energy_bolt is a staggered 3-round salvo (targeted pattern); the ability's
	// Projectile carries only the spawn origin height.
	if a.Projectile.OriginY != 1.2 {
		t.Errorf("Projectile.OriginY = %f, want 1.2", a.Projectile.OriginY)
	}
	if a.Pattern == nil || len(a.Pattern.Emitters) != 1 {
		t.Fatalf("energy_bolt should have a 1-emitter salvo pattern, got %+v", a.Pattern)
	}
	em := a.Pattern.Emitters[0]
	if em.Type != combat.EmitterTargeted {
		t.Errorf("emitter Type = %d, want EmitterTargeted (%d)", em.Type, combat.EmitterTargeted)
	}
	if em.Count != 1 || em.Waves != 3 {
		t.Errorf("salvo Count/Waves = %d/%d, want 1/3", em.Count, em.Waves)
	}
	if em.Projectile.Speed != 18.0 {
		t.Errorf("salvo projectile Speed = %f, want 18", em.Projectile.Speed)
	}
	if !a.TrackTarget {
		t.Error("TrackTarget should be true")
	}
	if a.DamageSource != combat.SourceEnemyRanged {
		t.Errorf("DamageSource = %d, want %d", a.DamageSource, combat.SourceEnemyRanged)
	}
}

// TestLoadMobs_TreeDataPresent verifies YAML-loaded defs have TreeData set.
func TestLoadMobs_TreeDataPresent(t *testing.T) {
	for _, name := range []string{"hallway_melee", testHallwayRanged} {
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

// TestLoadEncounters_GuardCaptain verifies the boss loads from YAML with correct values.
func TestLoadEncounters_GuardCaptain(t *testing.T) {
	def := DefRegistry["guard_captain"]
	if def == nil {
		t.Fatal("guard_captain missing from DefRegistry")
	}
	if def.TreeData == nil {
		t.Error("guard_captain.TreeData should be set (YAML-defined)")
	}
	if def.MaxHealth != 1800 {
		t.Errorf("guard_captain MaxHealth = %f, want 1800", def.MaxHealth)
	}
	if len(def.Abilities) != 5 {
		t.Errorf("guard_captain abilities = %d, want 5", len(def.Abilities))
	}
	if len(def.Phases) != 2 {
		t.Errorf("guard_captain phases = %d, want 2", len(def.Phases))
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
	e := entity.NewEnemy(0, 100, testTest)
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

	e := entity.NewEnemy(0, 100, testTest)
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
		NodeSelector: []any{
			map[string]any{
				NodeSequence: []any{
					LeafIsDead,
					LeafStop,
				},
			},
			LeafChase,
		},
	}
	node, err := bt.BuildTreeFromYAML(data, resolveLeaf)
	if err != nil {
		t.Fatalf("BuildTreeFromYAML: %v", err)
	}

	// Tick with alive enemy — should reach chase (Selector tries sequence which fails on is_dead)
	e := entity.NewEnemy(0, 100, testTest)
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
	if ovr.CommitTime == nil || *ovr.CommitTime != 0.3 {
		t.Errorf("AbilityOverrides[slash].CommitTime = %v, want 0.3", ovr.CommitTime)
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
	for _, name := range []string{"attack", LeafAggroOrPatrol} {
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

// TestResolveLeaf_TargetBeyond verifies target_beyond parameterized condition.
func TestResolveLeaf_TargetBeyond(t *testing.T) {
	node, err := resolveLeaf("target_beyond(5)")
	if err != nil {
		t.Fatalf("resolveLeaf: %v", err)
	}

	e := entity.NewEnemy(0, 100, testTest)
	e.Alive = true
	p := testPlayer(1, entity.Vec3{X: 0, Z: 8})
	e.TargetPlayerID = p.ID
	ctx := &EntityContext{Enemy: e, Def: &EnemyDef{}, Players: testPlayers(p)}
	result := node.Tick(ctx)
	if result != bt.Success {
		t.Errorf("target_beyond(5) with player at Z=8 = %v, want Success", result)
	}

	p.Position = entity.Vec3{X: 0, Z: 3}
	result = node.Tick(ctx)
	if result != bt.Failure {
		t.Errorf("target_beyond(5) with player at Z=3 = %v, want Failure", result)
	}
}

// TestResolveLeaf_PlayersInAoE verifies players_in_aoe parameterized condition.
func TestResolveLeaf_PlayersInAoE(t *testing.T) {
	node, err := resolveLeaf("players_in_aoe(5)")
	if err != nil {
		t.Fatalf("resolveLeaf: %v", err)
	}

	e := entity.NewEnemy(0, 100, testTest)
	e.Alive = true
	p1 := testPlayer(1, entity.Vec3{X: 2, Z: 0})
	p2 := testPlayer(2, entity.Vec3{X: -2, Z: 0})
	ctx := &EntityContext{Enemy: e, Def: &EnemyDef{}, Players: testPlayers(p1, p2)}
	result := node.Tick(ctx)
	if result != bt.Success {
		t.Errorf("players_in_aoe(5) with 2 players at dist 2 = %v, want Success", result)
	}
}

// TestResolveLeaf_ParamInvalidArgs verifies bad params return errors.
func TestResolveLeaf_ParamInvalidArgs(t *testing.T) {
	cases := []struct {
		name string
		leaf string
	}{
		{"player_nearby non-numeric", "player_nearby(abc)"},
		{"phase_eq non-numeric", "phase_eq(abc)"},
		{"player_nearby empty", "player_nearby()"},
		{"target_beyond non-numeric", "target_beyond(abc)"},
		{"players_in_aoe non-numeric", "players_in_aoe(abc)"},
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
	e := entity.NewEnemy(0, 100, testTest)
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
		{NodeSequence, map[string]any{NodeSequence: []any{LeafIsDead, LeafStop}}},
		{NodeSelector, map[string]any{NodeSelector: []any{LeafIsDead, LeafStop}}},
		{NodeReactiveSelector, map[string]any{NodeReactiveSelector: []any{LeafIsDead, LeafStop}}},
	}
	for _, tc := range composites {
		t.Run(tc.name, func(t *testing.T) {
			node, err := bt.BuildTreeFromYAML(tc.data, resolveLeaf)
			if err != nil {
				t.Fatalf("BuildTreeFromYAML: %v", err)
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
		NodeSequence: []any{LeafStop},
		NodeSelector: []any{LeafStop},
	}
	_, err := bt.BuildTreeFromYAML(data, resolveLeaf)
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
		"parallel": []any{LeafStop},
	}
	_, err := bt.BuildTreeFromYAML(data, resolveLeaf)
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
		NodeSequence: LeafStop,
	}
	_, err := bt.BuildTreeFromYAML(data, resolveLeaf)
	if err == nil {
		t.Fatal("expected error for non-list children")
	}
	if !strings.Contains(err.Error(), "must be a list") {
		t.Errorf("error = %q, want mention of 'must be a list'", err.Error())
	}
}

// TestBuildTreeFromData_ErrorUnexpectedType verifies unexpected types (int, float) fail.
func TestBuildTreeFromData_ErrorUnexpectedType(t *testing.T) {
	_, err := bt.BuildTreeFromYAML(42, resolveLeaf)
	if err == nil {
		t.Fatal("expected error for integer tree node")
	}
	if !strings.Contains(err.Error(), "unexpected tree node type") {
		t.Errorf("error = %q, want mention of 'unexpected tree node type'", err.Error())
	}
}

// TestBuildTreeFromData_StringLeaf verifies a single string resolves.
func TestBuildTreeFromData_StringLeaf(t *testing.T) {
	node, err := bt.BuildTreeFromYAML(LeafChase, resolveLeaf)
	if err != nil {
		t.Fatalf("BuildTreeFromYAML: %v", err)
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
		want    ability.AbilityCategory
	}{
		{TypeMelee, ability.CategoryMelee},
		{TypeRanged, ability.CategoryRanged},
		{TypeAoE, ability.CategoryAoE},
		{TypeCharge, ability.CategoryCharge},
	}
	for _, tc := range cases {
		t.Run(tc.typeStr, func(t *testing.T) {
			af := abilityFile{Name: testTest, Type: tc.typeStr, BaseWeight: 100}
			ad, err := convertAbility(af)
			if err != nil {
				t.Fatalf("convertAbility: %v", err)
			}
			if ad.Category != tc.want {
				t.Errorf("Category = %d, want %d", ad.Category, tc.want)
			}
		})
	}
}

// TestConvertAbility_AllTargetStrategies verifies all 3 target strategies.
func TestConvertAbility_AllTargetStrategies(t *testing.T) {
	cases := []struct {
		targetStr string
		want      ability.TargetStrategy
	}{
		{"nearest", ability.TargetNearest},
		{"", ability.TargetNearest}, // default
		{"farthest", ability.TargetFarthest},
		{"current", ability.TargetCurrent},
	}
	for _, tc := range cases {
		name := tc.targetStr
		if name == "" {
			name = "empty_default"
		}
		t.Run(name, func(t *testing.T) {
			af := abilityFile{Name: testTest, Type: TypeMelee, Target: tc.targetStr, BaseWeight: 100}
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
	af := abilityFile{Name: "bad", Type: TypeMelee, Target: "random", BaseWeight: 100}
	_, err := convertAbility(af)
	if err == nil {
		t.Fatal("expected error for unknown target strategy")
	}
}

// TestConvertAbility_DegreeConversions verifies cone and spread angle conversions.
func TestConvertAbility_DegreeConversions(t *testing.T) {
	// Melee cone: degrees stored directly on Hit.ArcDegrees
	meleeAf := abilityFile{
		Name: "cone_test", Type: TypeMelee, BaseWeight: 100,
		MeleeConeDeg: 90, MeleeRange: 3, MeleeDamage: 10,
	}
	meleeAd, err := convertAbility(meleeAf)
	if err != nil {
		t.Fatalf("convertAbility melee: %v", err)
	}
	if meleeAd.Hit.ArcDegrees != 90 {
		t.Errorf("Hit.ArcDegrees = %f, want 90", meleeAd.Hit.ArcDegrees)
	}

	// Ranged spread: degrees converted to radians on Projectile.Spread
	rangedAf := abilityFile{
		Name: "spread_test", Type: TypeRanged, BaseWeight: 100,
		ProjectileSpreadDeg: 30, ProjectileCount: 1,
	}
	rangedAd, err := convertAbility(rangedAf)
	if err != nil {
		t.Fatalf("convertAbility ranged: %v", err)
	}
	expectedSpread := float32(30.0 * math.Pi / 180.0)
	if diff := rangedAd.Projectile.Spread - expectedSpread; diff > 0.001 || diff < -0.001 {
		t.Errorf("Projectile.Spread = %f, want %f", rangedAd.Projectile.Spread, expectedSpread)
	}
}

// TestConvertAbility_ChargeFields verifies charge-specific fields.
func TestConvertAbility_ChargeFields(t *testing.T) {
	af := abilityFile{
		Name: "rush", Type: TypeCharge, BaseWeight: 100,
		ChargeSpeed: 15, ChargeDamage: 40, ChargeMaxDistance: 20,
		ChargeHitRadius: 3, ChargeStopOnWall: true, ChargeStopOnObstacle: true,
		DamageSource: combat.SourceEnemyCharge,
	}
	ad, err := convertAbility(af)
	if err != nil {
		t.Fatalf("convertAbility: %v", err)
	}
	if ad.Charge.Speed != 15 {
		t.Errorf("Charge.Speed = %f, want 15", ad.Charge.Speed)
	}
	if ad.Charge.Damage != 40 {
		t.Errorf("Charge.Damage = %f, want 40", ad.Charge.Damage)
	}
	if ad.Charge.MaxDistance != 20 {
		t.Errorf("Charge.MaxDistance = %f, want 20", ad.Charge.MaxDistance)
	}
	if ad.Charge.HitRadius != 3 {
		t.Errorf("Charge.HitRadius = %f, want 3", ad.Charge.HitRadius)
	}
	if !ad.Charge.StopOnWall {
		t.Error("Charge.StopOnWall should be true")
	}
	if !ad.Charge.StopOnObstacle {
		t.Error("Charge.StopOnObstacle should be true")
	}
	if ad.DamageSource != combat.SourceEnemyCharge {
		t.Errorf("DamageSource = %d, want %d", ad.DamageSource, combat.SourceEnemyCharge)
	}
}

// TestConvertAbility_AoEFields verifies aoe-specific fields.
func TestConvertAbility_AoEFields(t *testing.T) {
	af := abilityFile{
		Name: "blast", Type: TypeAoE, BaseWeight: 100,
		AoERadius: 5, AoEDamage: 25, DamageSource: combat.SourceEnemyAoE,
	}
	ad, err := convertAbility(af)
	if err != nil {
		t.Fatalf("convertAbility: %v", err)
	}
	if ad.Hit.Radius != 5 {
		t.Errorf("Hit.Radius = %f, want 5", ad.Hit.Radius)
	}
	if ad.BaseDamage != 25 {
		t.Errorf("BaseDamage = %f, want 25", ad.BaseDamage)
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
	if a.Category != ability.CategoryRanged {
		t.Errorf("Category = %d, want %d", a.Category, ability.CategoryRanged)
	}
	if a.TargetStrategy != ability.TargetFarthest {
		t.Errorf("TargetStrategy = %d, want %d", a.TargetStrategy, ability.TargetFarthest)
	}
	if a.Projectile.Count != 3 {
		t.Errorf("Projectile.Count = %d, want 3", a.Projectile.Count)
	}
	if a.Projectile.Speed != 20 {
		t.Errorf("Projectile.Speed = %f, want 20", a.Projectile.Speed)
	}
	if a.Projectile.Damage != 10 {
		t.Errorf("Projectile.Damage = %f, want 10", a.Projectile.Damage)
	}
	expectedSpread := float32(15.0 * math.Pi / 180.0)
	if diff := a.Projectile.Spread - expectedSpread; diff > 0.001 || diff < -0.001 {
		t.Errorf("Projectile.Spread = %f, want %f", a.Projectile.Spread, expectedSpread)
	}
	if a.Projectile.OriginY != 1.5 {
		t.Errorf("Projectile.OriginY = %f, want 1.5", a.Projectile.OriginY)
	}
	if a.Projectile.Lifetime != 3.0 {
		t.Errorf("Projectile.Lifetime = %f, want 3", a.Projectile.Lifetime)
	}
	if !a.TrackTarget {
		t.Error("TrackTarget should be true")
	}
	if a.DamageSource != combat.SourceEnemyRanged {
		t.Errorf("DamageSource = %d, want %d", a.DamageSource, combat.SourceEnemyRanged)
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
	if def.Abilities[1].Name != testBoltID {
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
		_, err := bt.BuildTreeFromYAML(data, resolveLeaf)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkResolveLeaf_Simple(b *testing.B) {
	for b.Loop() {
		_, _ = resolveLeaf(LeafChase)
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

func TestParseMobYAML_LifecycleFields(t *testing.T) {
	yamlData := `
name: test_lifecycle
max_health: 100
move_speed: 3.0
abilities:
  - name: slash
    type: melee
    telegraph_time: 0.5
    execute_time: 0.2
    cooldown_time: 1.0
    base_weight: 100
    max_range: 3.0
    melee_range: 3.0
    melee_damage: 10
    cancellable: true
    can_move_committed: true
    can_move_executing: false
tree:
  reactive_selector:
    - sequence: [is_dead, stop]
    - chase
`
	def, err := parseMobYAML([]byte(yamlData))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(def.Abilities) != 1 {
		t.Fatalf("expected 1 ability, got %d", len(def.Abilities))
	}
	a := def.Abilities[0]
	if !a.Cancellable {
		t.Error("Cancellable should be true")
	}
	if !a.CanMoveCommitted {
		t.Error("CanMoveCommitted should be true")
	}
	if a.CanMoveExecuting {
		t.Error("CanMoveExecuting should be false")
	}
}
