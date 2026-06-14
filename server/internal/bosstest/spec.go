package bosstest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"codex-online/server/internal/overflux"

	"gopkg.in/yaml.v3"
)

// OverfluxSpec defines overflux conditions to test. Each entry produces a
// separate fuzz batch with the given conditions applied.
type OverfluxSpec struct {
	Name       string             `yaml:"name"` // label for reporting (e.g. "fortified_3")
	Conditions []OverfluxCondSpec `yaml:"conditions"`
	WinRate    *RangeSpec         `yaml:"win_rate"` // optional: per-overflux assertion
}

// OverfluxCondSpec is a single overflux condition with its rank.
type OverfluxCondSpec struct {
	ID   string `yaml:"id"`   // e.g. "enemy_hp"
	Rank int    `yaml:"rank"` // 1-5
}

// ToOverfluxState converts the spec conditions into an overflux.State.
func (s OverfluxSpec) ToOverfluxState() *overflux.State {
	var conditions []overflux.ActiveCondition
	for _, c := range s.Conditions {
		conditions = append(conditions, overflux.ActiveCondition{
			ID:   overflux.ConditionID(c.ID),
			Rank: c.Rank,
		})
	}
	return overflux.NewState(conditions)
}

// EncounterSpec defines expected balance values for an encounter. An encounter
// is either a single boss (Boss set) or a trash-mob pack (Pack set). Tests fail
// if simulation results drift outside these ranges.
type EncounterSpec struct {
	Boss         string          `yaml:"boss"` // single-enemy encounter (mutually exclusive with pack)
	Name         string          `yaml:"name"` // label for pack encounters (boss encounters use Boss)
	Pack         []PackEntrySpec `yaml:"pack"` // multi-enemy trash pack
	Runs         int             `yaml:"runs"`
	Compositions []CompSpec      `yaml:"compositions"`
	Overflux     []OverfluxSpec  `yaml:"overflux"` // optional: test under overflux conditions
	WinRate      RangeSpec       `yaml:"win_rate"`
	Duration     DurationSpec    `yaml:"duration"`
	PhaseReach   []PhaseSpec     `yaml:"phase_reach"`
	TreeHealth   TreeHealthSpec  `yaml:"tree_health"`
	SpecBalance  *BalanceSpec    `yaml:"spec_balance"`
	AbilityStats []AbilitySpec   `yaml:"ability_stats"`
}

// PackEntrySpec is one mob type in a trash pack: spawn Count copies of Def.
type PackEntrySpec struct {
	Def   string `yaml:"def"`
	Count int    `yaml:"count"`
}

// IsPack reports whether this spec describes a multi-enemy trash pack.
func (s *EncounterSpec) IsPack() bool { return len(s.Pack) > 0 }

// Label returns the encounter's display/identifier name: the boss def name for
// boss encounters, or the pack Name for pack encounters.
func (s *EncounterSpec) Label() string {
	if s.Boss != "" {
		return s.Boss
	}
	return s.Name
}

// EnemyDefs returns the flat list of enemy def names to spawn: [Boss] for a
// boss encounter, or each pack entry's Def repeated Count times for a pack.
func (s *EncounterSpec) EnemyDefs() []string {
	if !s.IsPack() {
		return []string{s.Boss}
	}
	var defs []string
	for _, e := range s.Pack {
		count := max(e.Count, 1)
		for range count {
			defs = append(defs, e.Def)
		}
	}
	return defs
}

// CompSpec defines a party composition for testing.
type CompSpec struct {
	Name     string     `yaml:"name"`
	Classes  []string   `yaml:"classes"`
	Specs    []string   `yaml:"specs"` // spec IDs (empty string = class default)
	Profiles []string   `yaml:"profiles"`
	Loadouts [][]string `yaml:"loadouts"` // optional per-player ability loadouts
	WinRate  *RangeSpec `yaml:"win_rate"` // optional per-comp win rate assertion
}

// RangeSpec is a min/max float range.
type RangeSpec struct {
	Min float64 `yaml:"min"`
	Max float64 `yaml:"max"`
}

// DurationSpec bounds fight duration in seconds.
type DurationSpec struct {
	MinSeconds float64 `yaml:"min_seconds"`
	MaxSeconds float64 `yaml:"max_seconds"`
}

// PhaseSpec defines expected reach rate for a phase.
type PhaseSpec struct {
	Phase   int     `yaml:"phase"`
	MinRate float64 `yaml:"min_rate"`
	MaxRate float64 `yaml:"max_rate"`
}

// TreeHealthSpec defines maximum allowed dead/cold nodes.
type TreeHealthSpec struct {
	MaxDeadNodes int `yaml:"max_dead_nodes"`
	MaxColdNodes int `yaml:"max_cold_nodes"`
}

// BalanceSpec defines class balance bounds.
type BalanceSpec struct {
	MaxDamageShareSigma float64 `yaml:"max_damage_share_sigma"`
}

// AbilitySpec defines expected behavior for a specific ability.
type AbilitySpec struct {
	Ability      string  `yaml:"ability"`
	MaxKillRate  float64 `yaml:"max_kill_rate"`
	MinDodgeRate float64 `yaml:"min_dodge_rate"`
}

// LoadSpec reads and parses an encounter spec YAML file.
func LoadSpec(path string) (*EncounterSpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load spec %q: %w", path, err)
	}
	var spec EncounterSpec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("parse spec %q: %w", path, err)
	}
	if spec.Boss == "" && !spec.IsPack() {
		return nil, fmt.Errorf("spec %q must set either 'boss' or 'pack'", path)
	}
	// Pack encounters fall back to the file base name for their label.
	if spec.IsPack() && spec.Name == "" {
		spec.Name = strings.TrimSuffix(filepath.Base(path), ".yaml")
	}
	if spec.Runs == 0 {
		spec.Runs = 1000
	}
	return &spec, nil
}

// ToPartyConfigs converts a CompSpec to PuppetConfig slice.
func (cs CompSpec) ToPartyConfigs() []PuppetConfig {
	configs := make([]PuppetConfig, len(cs.Classes))
	for i := range cs.Classes {
		profile := ProfileAverage
		if i < len(cs.Profiles) {
			profile = BotProfile(cs.Profiles[i])
		}
		var spec string
		if i < len(cs.Specs) {
			spec = cs.Specs[i]
		}
		var loadout []string
		if i < len(cs.Loadouts) && len(cs.Loadouts[i]) > 0 {
			loadout = cs.Loadouts[i]
		}
		configs[i] = PuppetConfig{
			Class:   cs.Classes[i],
			Spec:    spec,
			Profile: profile,
			Loadout: loadout,
		}
	}
	return configs
}

// ExpandVariants expands pipe-separated specs into multiple compositions.
// e.g., specs: [assault, blade|shield, multi_blade] produces two CompSpecs
// with the same Name (so reporting groups them), one with blade, one with shield.
func (s *EncounterSpec) ExpandVariants() {
	var expanded []CompSpec
	for _, cs := range s.Compositions {
		expanded = append(expanded, cs.expandSpecs()...)
	}
	s.Compositions = expanded
}

func (cs CompSpec) expandSpecs() []CompSpec {
	slotOptions := make([][]string, len(cs.Specs))
	for i, s := range cs.Specs {
		slotOptions[i] = strings.Split(s, "|")
	}
	// Cartesian product of all slot options.
	combos := [][]string{{}}
	for _, opts := range slotOptions {
		var next [][]string
		for _, existing := range combos {
			for _, o := range opts {
				c := make([]string, len(existing), len(existing)+1)
				copy(c, existing)
				next = append(next, append(c, o))
			}
		}
		combos = next
	}
	results := make([]CompSpec, len(combos))
	for i, specs := range combos {
		results[i] = CompSpec{
			Name:     cs.Name,
			Classes:  cs.Classes,
			Specs:    specs,
			Profiles: cs.Profiles,
			Loadouts: cs.Loadouts,
			WinRate:  cs.WinRate,
		}
	}
	return results
}
