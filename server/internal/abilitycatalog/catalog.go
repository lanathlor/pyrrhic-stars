// Package abilitycatalog loads and queries YAML ability definitions.
// It provides lookup by ID, affinity-tier resolution per spec, and
// loadout validation. The catalog is read-only after construction.
package abilitycatalog

import (
	"fmt"
	"os"
	"slices"

	"gopkg.in/yaml.v3"
)

// AbilityEntry is a single ability in the catalog (presentational metadata).
type AbilityEntry struct {
	ID          string  `yaml:"id"`
	Name        string  `yaml:"name"`
	School      string  `yaml:"school"`
	AbilityType string  `yaml:"ability_type"`
	Delivery    string  `yaml:"delivery"`
	FluxCost    string  `yaml:"flux_cost"`
	Cooldown    float32 `yaml:"cooldown"`
	CommitTime  float32 `yaml:"commit_time"`
	Description string  `yaml:"description"`
	Implemented bool    `yaml:"implemented"`
}

// SpecAffinity describes which schools are primary and secondary for a spec.
type SpecAffinity struct {
	Primary   []string `yaml:"primary"`
	Secondary []string `yaml:"secondary"`
}

// AbilityWithAffinity pairs an AbilityEntry with its affinity tier for a given spec.
type AbilityWithAffinity struct {
	AbilityEntry
	Affinity string // "primary", "secondary", "off"
}

// Catalog holds all loaded ability data for the Arcanotechnicien class.
type Catalog struct {
	Abilities  []AbilityEntry
	byID       map[string]*AbilityEntry
	Affinities map[string]SpecAffinity
}

// yamlFile mirrors the on-disk YAML structure for unmarshalling.
type yamlFile struct {
	Affinities map[string]SpecAffinity `yaml:"affinities"`
	Abilities  []AbilityEntry          `yaml:"abilities"`
}

// Load reads the YAML catalog from disk and returns a ready-to-query Catalog.
func Load(path string) (*Catalog, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("abilitycatalog load: %w", err)
	}

	var raw yamlFile
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("abilitycatalog unmarshal: %w", err)
	}

	c := &Catalog{
		Abilities:  raw.Abilities,
		Affinities: raw.Affinities,
		byID:       make(map[string]*AbilityEntry, len(raw.Abilities)),
	}

	for i := range c.Abilities {
		c.byID[c.Abilities[i].ID] = &c.Abilities[i]
	}

	return c, nil
}

// GetAbility returns an ability by ID, or nil if not found.
func (c *Catalog) GetAbility(id string) *AbilityEntry {
	return c.byID[id]
}

// AllAbilities returns a copy of every ability in the catalog.
func (c *Catalog) AllAbilities() []AbilityEntry {
	out := make([]AbilityEntry, len(c.Abilities))
	copy(out, c.Abilities)
	return out
}

// Affinity tier constants.
const (
	AffinityPrimary   = "primary"
	AffinitySecondary = "secondary"
	AffinityOff       = "off"
)

// GetAffinityTier returns "primary", "secondary", or "off" for an ability's
// school relative to a given spec. If the spec is unknown, returns "off".
func (c *Catalog) GetAffinityTier(specID, school string) string {
	aff, ok := c.Affinities[specID]
	if !ok {
		return AffinityOff
	}

	if slices.Contains(aff.Primary, school) {
		return AffinityPrimary
	}
	if slices.Contains(aff.Secondary, school) {
		return AffinitySecondary
	}
	return AffinityOff
}

// ValidateLoadout checks that all non-empty slots reference valid, implemented
// ability IDs. Empty strings are treated as unslotted and are always valid.
func (c *Catalog) ValidateLoadout(slots [6]string) bool {
	for _, id := range slots {
		if id == "" {
			continue
		}
		entry := c.byID[id]
		if entry == nil || !entry.Implemented {
			return false
		}
	}
	return true
}

// AbilitiesForSpec returns all abilities annotated with their affinity tier for the
// given spec. The order matches the catalog order.
func (c *Catalog) AbilitiesForSpec(specID string) []AbilityWithAffinity {
	out := make([]AbilityWithAffinity, len(c.Abilities))
	for i, s := range c.Abilities {
		out[i] = AbilityWithAffinity{
			AbilityEntry: s,
			Affinity:     c.GetAffinityTier(specID, s.School),
		}
	}
	return out
}
