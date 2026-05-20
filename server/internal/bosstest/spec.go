package bosstest

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// EncounterSpec defines expected balance values for a boss encounter.
// Tests fail if simulation results drift outside these ranges.
type EncounterSpec struct {
	Boss         string         `yaml:"boss"`
	Runs         int            `yaml:"runs"`
	Compositions []CompSpec     `yaml:"compositions"`
	WinRate      RangeSpec      `yaml:"win_rate"`
	Duration     DurationSpec   `yaml:"duration"`
	PhaseReach   []PhaseSpec    `yaml:"phase_reach"`
	TreeHealth   TreeHealthSpec `yaml:"tree_health"`
	ClassBalance *BalanceSpec   `yaml:"class_balance"`
	AbilityStats []AbilitySpec  `yaml:"ability_stats"`
}

// CompSpec defines a party composition for testing.
type CompSpec struct {
	Name     string   `yaml:"name"`
	Classes  []string `yaml:"classes"`
	Specs    []string `yaml:"specs"`    // spec IDs (empty string = class default)
	Profiles []string `yaml:"profiles"`
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
	if spec.Boss == "" {
		return nil, fmt.Errorf("spec %q missing 'boss' field", path)
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
		configs[i] = PuppetConfig{
			Class:   cs.Classes[i],
			Spec:    spec,
			Profile: profile,
		}
	}
	return configs
}
