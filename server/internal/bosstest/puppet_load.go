package bosstest

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"codex-online/server/internal/bt"

	"gopkg.in/yaml.v3"
)

// puppetTreeFile is the YAML schema for a puppet behavior tree definition.
type puppetTreeFile struct {
	Class          string   `yaml:"class"`
	Bosses         []string `yaml:"bosses"`
	Profiles       []string `yaml:"profiles"`
	PreferredRange *float32 `yaml:"preferred_range"`
	Tree           any      `yaml:"tree"`
}

// puppetTreeDef is a parsed and validated puppet tree definition.
type puppetTreeDef struct {
	Class          string
	Bosses         []string // empty = matches all bosses
	Profiles       []string // empty = matches all profiles
	PreferredRange *float32 // nil = use class default
	Tree           *bt.Tree
}

// ResolvedPuppet holds the result of a puppet tree lookup.
type ResolvedPuppet struct {
	Tree           *bt.Tree
	PreferredRange *float32 // nil = no override
}

// PuppetTreeRegistry holds all loaded puppet tree definitions and resolves
// the best match for a given (class, boss, profile) triple.
type PuppetTreeRegistry struct {
	defs []puppetTreeDef
}

// LoadPuppetTrees reads all .yaml files from dir, parses each into a
// puppetTreeDef, and returns a registry. Trees are validated at load time.
func LoadPuppetTrees(dir string) (*PuppetTreeRegistry, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("LoadPuppetTrees: read dir %q: %w", dir, err)
	}

	reg := &PuppetTreeRegistry{}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("LoadPuppetTrees: read %q: %w", e.Name(), err)
		}

		def, err := parsePuppetTreeYAML(data)
		if err != nil {
			return nil, fmt.Errorf("LoadPuppetTrees: parse %q: %w", e.Name(), err)
		}
		reg.defs = append(reg.defs, *def)
	}

	return reg, nil
}

// parsePuppetTreeYAML unmarshals YAML bytes into a puppetTreeDef.
func parsePuppetTreeYAML(data []byte) (*puppetTreeDef, error) {
	var pf puppetTreeFile
	if err := yaml.Unmarshal(data, &pf); err != nil {
		return nil, fmt.Errorf("yaml unmarshal: %w", err)
	}
	if pf.Class == "" {
		return nil, errors.New("puppet tree missing 'class'")
	}
	if pf.Tree == nil {
		return nil, fmt.Errorf("puppet tree %q missing 'tree'", pf.Class)
	}

	// Validate tree at load time (fail fast on unknown leaves)
	root, err := bt.BuildTreeFromYAML(pf.Tree, resolvePuppetLeaf)
	if err != nil {
		return nil, fmt.Errorf("puppet tree %q: %w", pf.Class, err)
	}

	return &puppetTreeDef{
		Class:          pf.Class,
		Bosses:         pf.Bosses,
		Profiles:       pf.Profiles,
		PreferredRange: pf.PreferredRange,
		Tree:           bt.NewTree(root),
	}, nil
}

// Resolve finds the best matching puppet tree for the given class, boss, and profile.
//
// Priority (highest first):
//  1. class + boss match + profile match (both lists non-empty and contain the value)
//  2. class + boss match (bosses non-empty, profiles empty)
//  3. class + profile match (bosses empty, profiles non-empty)
//  4. class only (both bosses and profiles empty)
//
// Returns nil if no YAML tree matches (caller should fall back to hardcoded Go tree).
func (r *PuppetTreeRegistry) Resolve(class, boss string, profile BotProfile) *ResolvedPuppet {
	if r == nil {
		return nil
	}

	bestScore := -1
	var best *puppetTreeDef

	for i := range r.defs {
		d := &r.defs[i]
		if d.Class != class {
			continue
		}

		bossMatch := len(d.Bosses) == 0 || containsStr(d.Bosses, boss)
		profileMatch := len(d.Profiles) == 0 || containsStr(d.Profiles, string(profile))

		if !bossMatch || !profileMatch {
			continue
		}

		score := matchScore(d.Bosses, d.Profiles)
		if score > bestScore {
			bestScore = score
			best = d
		}
	}

	if best == nil {
		return nil
	}
	return &ResolvedPuppet{
		Tree:           best.Tree,
		PreferredRange: best.PreferredRange,
	}
}

// matchScore ranks how specific a definition is.
// Both specified = 3, boss only = 2, profile only = 1, neither = 0.
func matchScore(bosses, profiles []string) int {
	score := 0
	if len(bosses) > 0 {
		score += 2
	}
	if len(profiles) > 0 {
		score++
	}
	return score
}

func containsStr(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
