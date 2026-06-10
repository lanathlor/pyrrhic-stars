package enemyai

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"

	"codex-online/server/internal/ability"
	"codex-online/server/internal/bt"
	"codex-online/server/internal/combat"

	"gopkg.in/yaml.v3"
)

// YAML ability type constants.
const (
	TypeMelee  = "melee"
	TypeRanged = "ranged"
	TypeAoE    = "aoe"
	TypeCharge = "charge"
)

// mobFile is the YAML schema for a Tier 1 mob definition.
type mobFile struct {
	Name             string            `yaml:"name"`
	Tier             int               `yaml:"tier"`
	MaxHealth        float32           `yaml:"max_health"`
	MoveSpeed        float32           `yaml:"move_speed"`
	Radius           float32           `yaml:"radius"`
	PreferredRange   float32           `yaml:"preferred_range"`
	BackpedalSpeed   float32           `yaml:"backpedal_speed"`
	AntiRepeat       float32           `yaml:"anti_repeat"`
	Abilities        []abilityFile     `yaml:"abilities"`
	Phases           []phaseFile       `yaml:"phases"`
	Tree             any               `yaml:"tree"`
	OverfluxVariants map[string]string `yaml:"overflux_variants"` // condition_id -> filename
}

// variantFile is the YAML schema for an overflux variant definition.
type variantFile struct {
	Tree             any                       `yaml:"tree"`
	Abilities        []abilityFile             `yaml:"abilities"`
	AbilityOverrides map[string]variantOvrFile `yaml:"ability_overrides"`
}

// variantOvrFile represents a single ability override in a variant.
// Uses the same fields as abilityOvrFile plus a full pattern replacement.
type variantOvrFile struct {
	TelegraphTime     *float32     `yaml:"telegraph_time"`
	Damage            *float32     `yaml:"damage"`
	ProjectileCount   *int         `yaml:"projectile_count"`
	AoERadius         *float32     `yaml:"aoe_radius"`
	ChargeSpeed       *float32     `yaml:"charge_speed"`
	ChargeMaxDistance *float32     `yaml:"charge_max_distance"`
	CooldownTime      *float32     `yaml:"cooldown_time"`
	Pattern           *patternFile `yaml:"pattern"`
}

type abilityFile struct {
	Name             string  `yaml:"name"`
	Type             string  `yaml:"type"`   // melee, ranged, aoe, charge
	Target           string  `yaml:"target"` // nearest, farthest, current
	TelegraphTime    float32 `yaml:"telegraph_time"`
	ExecuteTime      float32 `yaml:"execute_time"`
	CooldownTime     float32 `yaml:"cooldown_time"`
	BaseWeight       int     `yaml:"base_weight"`
	MinRange         float32 `yaml:"min_range"`
	MaxRange         float32 `yaml:"max_range"`
	FaceTarget       bool    `yaml:"face_target"`
	TrackTarget      bool    `yaml:"track_target"`
	Cancellable      bool    `yaml:"cancellable"`
	CanMoveCommitted bool    `yaml:"can_move_committed"`
	CanMoveExecuting bool    `yaml:"can_move_executing"`

	// Melee
	MeleeRange   float32 `yaml:"melee_range"`
	MeleeDamage  float32 `yaml:"melee_damage"`
	MeleeConeDeg float32 `yaml:"melee_cone_deg"` // degrees

	// Ranged
	ProjectileCount     int     `yaml:"projectile_count"`
	ProjectileSpeed     float32 `yaml:"projectile_speed"`
	ProjectileDamage    float32 `yaml:"projectile_damage"`
	ProjectileSpreadDeg float32 `yaml:"projectile_spread_deg"` // degrees
	ProjectileOriginY   float32 `yaml:"projectile_origin_y"`
	ProjectileLifetime  float32 `yaml:"projectile_lifetime"`

	// AoE
	AoERadius float32 `yaml:"aoe_radius"`
	AoEDamage float32 `yaml:"aoe_damage"`

	// Charge
	ChargeSpeed          float32 `yaml:"charge_speed"`
	ChargeDamage         float32 `yaml:"charge_damage"`
	ChargeMaxDistance    float32 `yaml:"charge_max_distance"`
	ChargeHitRadius      float32 `yaml:"charge_hit_radius"`
	ChargeStopOnWall     bool    `yaml:"charge_stop_on_wall"`
	ChargeStopOnObstacle bool    `yaml:"charge_stop_on_obstacle"`

	DamageSource uint8 `yaml:"damage_source"`

	// Pattern: bullet-hell emitter composition (replaces ProjectileCount/Spread)
	Pattern *patternFile `yaml:"pattern"`
}

type patternFile struct {
	Emitters []emitterFile `yaml:"emitters"`
}

type emitterFile struct {
	Type          string   `yaml:"type"` // radial, cone, line, arc, ring_contract, targeted, random_zone
	Count         int      `yaml:"count"`
	Waves         int      `yaml:"waves"`
	WaveInterval  float32  `yaml:"wave_interval"`
	OffsetPerWave float32  `yaml:"offset_per_wave"` // degrees
	StartAngle    float32  `yaml:"start_angle"`     // degrees
	ArcAngle      float32  `yaml:"arc_angle"`       // degrees
	LineWidth     float32  `yaml:"line_width"`
	StartRadius   float32  `yaml:"start_radius"`
	ZoneRadius    float32  `yaml:"zone_radius"`
	AimAtTarget   bool     `yaml:"aim_at_target"`
	Projectile    projFile `yaml:"projectile"`
}

type projFile struct {
	Speed           float32 `yaml:"speed"`
	Damage          float32 `yaml:"damage"`
	Lifetime        float32 `yaml:"lifetime"`
	Radius          float32 `yaml:"radius"`
	Acceleration    float32 `yaml:"acceleration"`
	MaxSpeed        float32 `yaml:"max_speed"`
	AngularVelocity float32 `yaml:"angular_velocity"` // degrees/s
}

type phaseFile struct {
	HPThresholdPct   float32                   `yaml:"hp_threshold_pct"`
	TransitionTime   float32                   `yaml:"transition_time"`
	MoveSpeed        float32                   `yaml:"move_speed"`
	BackpedalSpeed   float32                   `yaml:"backpedal_speed"`
	CooldownOverride float32                   `yaml:"cooldown_override"`
	WeightOverrides  map[string]int            `yaml:"weight_overrides"`
	AbilityOverrides map[string]abilityOvrFile `yaml:"ability_overrides"`
}

type abilityOvrFile struct {
	TelegraphTime     *float32 `yaml:"telegraph_time"`
	Damage            *float32 `yaml:"damage"`
	ProjectileCount   *int     `yaml:"projectile_count"`
	AoERadius         *float32 `yaml:"aoe_radius"`
	ChargeSpeed       *float32 `yaml:"charge_speed"`
	ChargeMaxDistance *float32 `yaml:"charge_max_distance"`
	CooldownTime      *float32 `yaml:"cooldown_time"`
}

// MobsDir returns the mobs directory path.
// Checks CODEX_MOBS_DIR env var first, then falls back to ../shared/mobs/.
func MobsDir() string {
	dir := os.Getenv("CODEX_MOBS_DIR")
	if dir == "" {
		dir = filepath.Join("..", "shared", "mobs")
	}
	return dir
}

// EncountersDir returns the encounters directory path.
// Checks CODEX_ENCOUNTERS_DIR env var first, then falls back to ../shared/encounters/.
func EncountersDir() string {
	dir := os.Getenv("CODEX_ENCOUNTERS_DIR")
	if dir == "" {
		dir = filepath.Join("..", "shared", "encounters")
	}
	return dir
}

// LoadEncounters reads all .yaml files from dir, parses each into an EnemyDef,
// and registers them in DefRegistry. Existing entries with the same name
// are overwritten. Uses the same schema as LoadMobs (Tier 3 bosses are
// expressed identically to Tier 1/2 mobs in YAML).
func LoadEncounters(dir string) error {
	return loadDir(dir, "LoadEncounters")
}

// LoadMobs reads all .yaml files from dir, parses each into an EnemyDef,
// and registers them in DefRegistry. Existing entries with the same name
// are overwritten.
func LoadMobs(dir string) error {
	return loadDir(dir, "LoadMobs")
}

// loadDir is the shared implementation for LoadEncounters and LoadMobs.
// It loads base YAML files and resolves any overflux variant references.
func loadDir(dir, caller string) error { //nolint:funlen // two-pass loader with variant resolution
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("%s: read dir %q: %w", caller, dir, err)
	}

	// Collect base files (skip variant files which are referenced by include).
	referenced := make(map[string]bool)
	var baseFiles []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		baseFiles = append(baseFiles, e.Name())
	}

	// First pass: parse base files to discover variant references.
	type pending struct {
		name string
		data []byte
	}
	var bases []pending
	for _, fname := range baseFiles {
		data, err := os.ReadFile(filepath.Join(dir, fname))
		if err != nil {
			return fmt.Errorf("%s: read %q: %w", caller, fname, err)
		}
		// Peek at overflux_variants to skip variant files in the main parse.
		var peek struct {
			OverfluxVariants map[string]string `yaml:"overflux_variants"`
		}
		_ = yaml.Unmarshal(data, &peek)
		for _, vf := range peek.OverfluxVariants {
			referenced[vf] = true
		}
		bases = append(bases, pending{name: fname, data: data})
	}

	// Second pass: parse base files (skip referenced variant files).
	for _, b := range bases {
		if referenced[b.name] {
			continue
		}
		def, err := parseMobYAML(b.data)
		if err != nil {
			return fmt.Errorf("%s: parse %q: %w", caller, b.name, err)
		}

		// Load overflux variants.
		var peek struct {
			OverfluxVariants map[string]string `yaml:"overflux_variants"`
		}
		_ = yaml.Unmarshal(b.data, &peek)
		if len(peek.OverfluxVariants) > 0 {
			def.OverfluxVariants = make(map[string]*OverfluxVariant, len(peek.OverfluxVariants))
			for condID, vfName := range peek.OverfluxVariants {
				vData, err := os.ReadFile(filepath.Join(dir, vfName))
				if err != nil {
					return fmt.Errorf("%s: load variant %q for %q: %w", caller, vfName, def.Name, err)
				}
				variant, err := parseVariantYAML(vData, def.Name, condID)
				if err != nil {
					return fmt.Errorf("%s: parse variant %q for %q: %w", caller, vfName, def.Name, err)
				}
				def.OverfluxVariants[condID] = variant
			}
		}

		DefRegistry[def.Name] = def
	}
	return nil
}

// parseVariantYAML parses an overflux variant YAML file into an OverfluxVariant.
func parseVariantYAML(data []byte, baseName, condID string) (*OverfluxVariant, error) {
	var vf variantFile
	if err := yaml.Unmarshal(data, &vf); err != nil {
		return nil, fmt.Errorf("yaml unmarshal: %w", err)
	}

	variant := &OverfluxVariant{}

	// Tree replacement
	if vf.Tree != nil {
		if _, err := bt.BuildTreeFromYAML(vf.Tree, resolveLeaf); err != nil {
			return nil, fmt.Errorf("variant %s/%s tree: %w", baseName, condID, err)
		}
		variant.TreeData = vf.Tree
	}

	// Additional abilities
	for _, af := range vf.Abilities {
		ad, err := convertAbility(af)
		if err != nil {
			return nil, fmt.Errorf("variant %s/%s ability %q: %w", baseName, condID, af.Name, err)
		}
		variant.Abilities = append(variant.Abilities, ad)
	}

	// Ability overrides
	if len(vf.AbilityOverrides) > 0 {
		variant.AbilityOverrides = make(map[string]AbilityOverride, len(vf.AbilityOverrides))
		for abilID, ovr := range vf.AbilityOverrides {
			ao := AbilityOverride{
				CommitTime:        ovr.TelegraphTime,
				Damage:            ovr.Damage,
				ProjectileCount:   ovr.ProjectileCount,
				AoERadius:         ovr.AoERadius,
				ChargeSpeed:       ovr.ChargeSpeed,
				ChargeMaxDistance: ovr.ChargeMaxDistance,
				CooldownTime:      ovr.CooldownTime,
			}
			if ovr.Pattern != nil {
				p, err := convertPattern(ovr.Pattern)
				if err != nil {
					return nil, fmt.Errorf("variant %s/%s override %q pattern: %w", baseName, condID, abilID, err)
				}
				ao.Pattern = p
			}
			variant.AbilityOverrides[abilID] = ao
		}
	}

	return variant, nil
}

// parseMobYAML unmarshals YAML bytes into an EnemyDef.
func parseMobYAML(data []byte) (*EnemyDef, error) {
	var mf mobFile
	if err := yaml.Unmarshal(data, &mf); err != nil {
		return nil, fmt.Errorf("yaml unmarshal: %w", err)
	}
	if mf.Name == "" {
		return nil, errors.New("mob definition missing 'name'")
	}
	if mf.Tree == nil {
		return nil, fmt.Errorf("mob %q missing 'tree'", mf.Name)
	}

	// Validate tree at load time (fail fast on unknown leaves)
	if _, err := bt.BuildTreeFromYAML(mf.Tree, resolveLeaf); err != nil {
		return nil, fmt.Errorf("mob %q tree: %w", mf.Name, err)
	}

	def := &EnemyDef{
		Name:           mf.Name,
		MaxHealth:      mf.MaxHealth,
		MoveSpeed:      mf.MoveSpeed,
		Radius:         mf.Radius,
		PreferredRange: mf.PreferredRange,
		BackpedalSpeed: mf.BackpedalSpeed,
		AntiRepeat:     mf.AntiRepeat,
		TreeData:       mf.Tree,
	}

	for _, af := range mf.Abilities {
		ad, err := convertAbility(af)
		if err != nil {
			return nil, fmt.Errorf("mob %q ability %q: %w", mf.Name, af.Name, err)
		}
		def.Abilities = append(def.Abilities, ad)
	}

	for _, pf := range mf.Phases {
		def.Phases = append(def.Phases, convertPhase(pf))
	}

	return def, nil
}

func convertAbility(af abilityFile) (ability.AbilityDef, error) {
	ad := ability.AbilityDef{
		ID:               af.Name,
		Name:             af.Name,
		CommitTime:       af.TelegraphTime,
		ExecuteTime:      af.ExecuteTime,
		Cooldown:         af.CooldownTime,
		BaseWeight:       af.BaseWeight,
		MinRange:         af.MinRange,
		MaxRange:         af.MaxRange,
		FaceTarget:       af.FaceTarget,
		TrackTarget:      af.TrackTarget,
		Cancellable:      af.Cancellable,
		CanMoveCommitted: af.CanMoveCommitted,
		CanMoveExecuting: af.CanMoveExecuting,
		DamageSource:     af.DamageSource,
	}

	if err := applyAbilityCategory(&ad, af); err != nil {
		return ability.AbilityDef{}, err
	}

	if err := applyTargetStrategy(&ad, af.Target); err != nil {
		return ability.AbilityDef{}, err
	}

	if af.Pattern != nil {
		p, err := convertPattern(af.Pattern)
		if err != nil {
			return ability.AbilityDef{}, fmt.Errorf("pattern: %w", err)
		}
		ad.Pattern = p
	}

	return ad, nil
}

func applyAbilityCategory(ad *ability.AbilityDef, af abilityFile) error {
	switch af.Type {
	case TypeMelee:
		ad.Category = ability.CategoryMelee
		ad.BaseDamage = af.MeleeDamage
		ad.Hit = ability.HitDef{
			Type:       ability.HitAoECone,
			Range:      af.MeleeRange,
			ArcDegrees: af.MeleeConeDeg,
		}
	case TypeRanged:
		ad.Category = ability.CategoryRanged
		ad.Projectile = &ability.ProjectileDef{
			Count:    af.ProjectileCount,
			Speed:    af.ProjectileSpeed,
			Damage:   af.ProjectileDamage,
			Spread:   af.ProjectileSpreadDeg * math.Pi / 180.0,
			OriginY:  af.ProjectileOriginY,
			Lifetime: af.ProjectileLifetime,
		}
	case TypeAoE:
		ad.Category = ability.CategoryAoE
		ad.BaseDamage = af.AoEDamage
		ad.Hit = ability.HitDef{
			Type:   ability.HitAoECircle,
			Radius: af.AoERadius,
		}
	case TypeCharge:
		ad.Category = ability.CategoryCharge
		ad.Charge = &ability.ChargeDef{
			Speed:          af.ChargeSpeed,
			Damage:         af.ChargeDamage,
			MaxDistance:    af.ChargeMaxDistance,
			HitRadius:      af.ChargeHitRadius,
			StopOnWall:     af.ChargeStopOnWall,
			StopOnObstacle: af.ChargeStopOnObstacle,
		}
	default:
		return fmt.Errorf("unknown ability type %q", af.Type)
	}
	return nil
}

func applyTargetStrategy(ad *ability.AbilityDef, target string) error {
	switch target {
	case "nearest", "":
		ad.TargetStrategy = ability.TargetNearest
	case "farthest":
		ad.TargetStrategy = ability.TargetFarthest
	case "current":
		ad.TargetStrategy = ability.TargetCurrent
	default:
		return fmt.Errorf("unknown target strategy %q", target)
	}
	return nil
}

func convertPattern(pf *patternFile) (*combat.PatternDef, error) {
	if len(pf.Emitters) == 0 {
		return nil, errors.New("pattern has no emitters")
	}
	def := &combat.PatternDef{
		Emitters: make([]combat.EmitterDef, 0, len(pf.Emitters)),
	}
	for i, ef := range pf.Emitters {
		etype, err := parseEmitterType(ef.Type)
		if err != nil {
			return nil, fmt.Errorf("emitter[%d]: %w", i, err)
		}
		waves := ef.Waves
		if waves == 0 {
			waves = 1
		}
		def.Emitters = append(def.Emitters, combat.EmitterDef{
			Type:          etype,
			Count:         ef.Count,
			Waves:         waves,
			WaveInterval:  ef.WaveInterval,
			OffsetPerWave: ef.OffsetPerWave * math.Pi / 180.0,
			StartAngle:    ef.StartAngle * math.Pi / 180.0,
			ArcAngle:      ef.ArcAngle * math.Pi / 180.0,
			LineWidth:     ef.LineWidth,
			StartRadius:   ef.StartRadius,
			ZoneRadius:    ef.ZoneRadius,
			AimAtTarget:   ef.AimAtTarget,
			Projectile: combat.ProjectileDef{
				Speed:           ef.Projectile.Speed,
				Damage:          ef.Projectile.Damage,
				Lifetime:        ef.Projectile.Lifetime,
				Radius:          ef.Projectile.Radius,
				Acceleration:    ef.Projectile.Acceleration,
				MaxSpeed:        ef.Projectile.MaxSpeed,
				AngularVelocity: ef.Projectile.AngularVelocity * math.Pi / 180.0,
			},
		})
	}
	return def, nil
}

func parseEmitterType(s string) (combat.EmitterType, error) {
	switch s {
	case "radial":
		return combat.EmitterRadial, nil
	case "cone":
		return combat.EmitterCone, nil
	case "line":
		return combat.EmitterLine, nil
	case "arc":
		return combat.EmitterArc, nil
	case "ring_contract":
		return combat.EmitterRingContract, nil
	case "targeted":
		return combat.EmitterTargeted, nil
	case "random_zone":
		return combat.EmitterRandomZone, nil
	default:
		return 0, fmt.Errorf("unknown emitter type %q", s)
	}
}

func convertPhase(pf phaseFile) PhaseDef {
	pd := PhaseDef{
		HPThresholdPct:   pf.HPThresholdPct,
		TransitionTime:   pf.TransitionTime,
		MoveSpeed:        pf.MoveSpeed,
		BackpedalSpeed:   pf.BackpedalSpeed,
		CooldownOverride: pf.CooldownOverride,
		WeightOverrides:  pf.WeightOverrides,
	}
	if len(pf.AbilityOverrides) > 0 {
		pd.AbilityOverrides = make(map[string]AbilityOverride, len(pf.AbilityOverrides))
		for name, ovr := range pf.AbilityOverrides {
			pd.AbilityOverrides[name] = AbilityOverride{
				CommitTime:        ovr.TelegraphTime,
				Damage:            ovr.Damage,
				ProjectileCount:   ovr.ProjectileCount,
				AoERadius:         ovr.AoERadius,
				ChargeSpeed:       ovr.ChargeSpeed,
				ChargeMaxDistance: ovr.ChargeMaxDistance,
				CooldownTime:      ovr.CooldownTime,
			}
		}
	}
	return pd
}
