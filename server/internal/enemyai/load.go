package enemyai

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"

	"codex-online/server/internal/bt"

	"gopkg.in/yaml.v3"
)

// mobFile is the YAML schema for a Tier 1 mob definition.
type mobFile struct {
	Name           string        `yaml:"name"`
	Tier           int           `yaml:"tier"`
	MaxHealth      float32       `yaml:"max_health"`
	MoveSpeed      float32       `yaml:"move_speed"`
	Radius         float32       `yaml:"radius"`
	PreferredRange float32       `yaml:"preferred_range"`
	BackpedalSpeed float32       `yaml:"backpedal_speed"`
	AntiRepeat     float32       `yaml:"anti_repeat"`
	Abilities      []abilityFile `yaml:"abilities"`
	Phases         []phaseFile   `yaml:"phases"`
	Tree           any           `yaml:"tree"`
}

type abilityFile struct {
	Name           string  `yaml:"name"`
	Type           string  `yaml:"type"`   // melee, ranged, aoe, charge
	Target         string  `yaml:"target"` // nearest, farthest, current
	TelegraphTime  float32 `yaml:"telegraph_time"`
	ExecuteTime    float32 `yaml:"execute_time"`
	CooldownTime   float32 `yaml:"cooldown_time"`
	BaseWeight     int     `yaml:"base_weight"`
	MinRange       float32 `yaml:"min_range"`
	MaxRange       float32 `yaml:"max_range"`
	FaceTarget     bool    `yaml:"face_target"`
	TrackTarget    bool    `yaml:"track_target"`

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
	ChargeMaxDistance     float32 `yaml:"charge_max_distance"`
	ChargeHitRadius      float32 `yaml:"charge_hit_radius"`
	ChargeStopOnWall     bool    `yaml:"charge_stop_on_wall"`
	ChargeStopOnObstacle bool    `yaml:"charge_stop_on_obstacle"`

	DamageSource uint8 `yaml:"damage_source"`
}

type phaseFile struct {
	HPThresholdPct   float32                    `yaml:"hp_threshold_pct"`
	TransitionTime   float32                    `yaml:"transition_time"`
	MoveSpeed        float32                    `yaml:"move_speed"`
	BackpedalSpeed   float32                    `yaml:"backpedal_speed"`
	CooldownOverride float32                    `yaml:"cooldown_override"`
	WeightOverrides  map[string]int             `yaml:"weight_overrides"`
	AbilityOverrides map[string]abilityOvrFile  `yaml:"ability_overrides"`
}

type abilityOvrFile struct {
	TelegraphTime    *float32 `yaml:"telegraph_time"`
	Damage           *float32 `yaml:"damage"`
	ProjectileCount  *int     `yaml:"projectile_count"`
	AoERadius        *float32 `yaml:"aoe_radius"`
	ChargeSpeed      *float32 `yaml:"charge_speed"`
	ChargeMaxDistance *float32 `yaml:"charge_max_distance"`
	CooldownTime     *float32 `yaml:"cooldown_time"`
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

// LoadMobs reads all .yaml files from dir, parses each into an EnemyDef,
// and registers them in DefRegistry. Existing entries with the same name
// are overwritten.
func LoadMobs(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("LoadMobs: read dir %q: %w", dir, err)
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return fmt.Errorf("LoadMobs: read %q: %w", e.Name(), err)
		}
		def, err := parseMobYAML(data)
		if err != nil {
			return fmt.Errorf("LoadMobs: parse %q: %w", e.Name(), err)
		}
		DefRegistry[def.Name] = def
	}
	return nil
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
	if _, err := buildTreeFromData(mf.Tree); err != nil {
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

func convertAbility(af abilityFile) (AbilityDef, error) {
	ad := AbilityDef{
		Name:          af.Name,
		TelegraphTime: af.TelegraphTime,
		ExecuteTime:   af.ExecuteTime,
		CooldownTime:  af.CooldownTime,
		BaseWeight:    af.BaseWeight,
		MinRange:      af.MinRange,
		MaxRange:      af.MaxRange,
		FaceTarget:    af.FaceTarget,
		TrackTarget:   af.TrackTarget,

		MeleeRange:     af.MeleeRange,
		MeleeDamage:    af.MeleeDamage,
		MeleeConeAngle: af.MeleeConeDeg * math.Pi / 180.0,

		ProjectileCount:    af.ProjectileCount,
		ProjectileSpeed:    af.ProjectileSpeed,
		ProjectileDamage:   af.ProjectileDamage,
		ProjectileSpread:   af.ProjectileSpreadDeg * math.Pi / 180.0,
		ProjectileOriginY:  af.ProjectileOriginY,
		ProjectileLifetime: af.ProjectileLifetime,

		AoERadius: af.AoERadius,
		AoEDamage: af.AoEDamage,

		ChargeSpeed:          af.ChargeSpeed,
		ChargeDamage:         af.ChargeDamage,
		ChargeMaxDistance:     af.ChargeMaxDistance,
		ChargeHitRadius:      af.ChargeHitRadius,
		ChargeStopOnWall:     af.ChargeStopOnWall,
		ChargeStopOnObstacle: af.ChargeStopOnObstacle,

		DamageSourceType: af.DamageSource,
	}

	switch af.Type {
	case "melee":
		ad.Type = AbilityMelee
	case "ranged":
		ad.Type = AbilityRanged
	case "aoe":
		ad.Type = AbilityAoE
	case "charge":
		ad.Type = AbilityCharge
	default:
		return AbilityDef{}, fmt.Errorf("unknown ability type %q", af.Type)
	}

	switch af.Target {
	case "nearest", "":
		ad.TargetStrategy = TargetNearest
	case "farthest":
		ad.TargetStrategy = TargetFarthest
	case "current":
		ad.TargetStrategy = TargetCurrent
	default:
		return AbilityDef{}, fmt.Errorf("unknown target strategy %q", af.Target)
	}

	return ad, nil
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
			pd.AbilityOverrides[name] = AbilityOverride(ovr)
		}
	}
	return pd
}

// buildTreeFromData recursively constructs a bt.Node from parsed YAML data.
// Each element is either:
//   - a string (leaf name, optionally prefixed with "!" for inversion)
//   - a map with one key (composite type: "sequence", "selector", "reactive_selector")
func buildTreeFromData(data any) (bt.Node, error) {
	switch v := data.(type) {
	case string:
		return resolveLeaf(v)

	case map[string]any:
		if len(v) != 1 {
			return nil, fmt.Errorf("tree node map must have exactly one key, got %d", len(v))
		}
		for key, val := range v {
			children, ok := val.([]any)
			if !ok {
				return nil, fmt.Errorf("composite %q: children must be a list", key)
			}
			nodes := make([]bt.Node, 0, len(children))
			for i, child := range children {
				n, err := buildTreeFromData(child)
				if err != nil {
					return nil, fmt.Errorf("composite %q child %d: %w", key, i, err)
				}
				nodes = append(nodes, n)
			}
			switch key {
			case "sequence":
				return bt.NewSequence(nodes...), nil
			case "selector":
				return bt.NewSelector(nodes...), nil
			case "reactive_selector":
				return bt.NewReactiveSelector(nodes...), nil
			default:
				return nil, fmt.Errorf("unknown composite type: %q", key)
			}
		}
	}

	return nil, fmt.Errorf("unexpected tree node type %T", data)
}
