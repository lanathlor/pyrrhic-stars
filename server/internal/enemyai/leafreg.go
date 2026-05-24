package enemyai

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"codex-online/server/internal/bt"
)

// leafEntry describes a single leaf in the registry.
type leafEntry struct {
	isCond bool
	cond   func(any) bool
	action func(any) bt.Result
}

// leafRegistry maps string names used in YAML tree definitions to BT leaf
// constructors. Populated once at package init — no mutex needed.
var leafRegistry = map[string]leafEntry{
	// Conditions
	"is_dead":                  {isCond: true, cond: condIsDead},
	"has_target":               {isCond: true, cond: condHasTarget},
	"target_in_melee_range":    {isCond: true, cond: condTargetInMeleeRange},
	"has_los":                  {isCond: true, cond: condHasLoS},
	"phase_transitioning":      {isCond: true, cond: condPhaseTransitioning},
	"in_leash_range":           {isCond: true, cond: condInLeashRange},
	"active_ability_is_charge": {isCond: true, cond: condActiveAbilityIsCharge},
	"is_committed":             {isCond: true, cond: condIsCommitted},
	"can_commit":               {isCond: true, cond: condCanCommit},
	"can_move":                 {isCond: true, cond: condCanMove},

	// Actions
	"stop":                 {action: actionStop},
	"aggro_nearest":        {action: actionAggroNearest},
	"set_target_clustered": {action: actionSetTargetClustered},
	"set_target_lowest_hp": {action: actionSetTargetLowestHP},
	"set_target_nearest":   {action: actionSetTargetNearest},
	"wait_transition": {action: actionWaitTransition},
	"leash_reset":     {action: actionLeashReset},
	"patrol":          {action: actionPatrol},
	"chase":           {action: actionChase},
	"select_ability":  {action: actionSelectAbility},
	"telegraph":       {action: actionTelegraph},
	"execute_ability": {action: actionExecuteAbility},
	"charge_dash":     {action: actionChargeDash},
	"cooldown":        {action: actionCooldown},
	"commit_weighted":   {action: actionCommitWeighted},
	"wait_ability":    {action: actionWaitAbility},
	"cancel_ability":  {action: actionCancelAbility},
}

// paramFactories maps parameterized leaf base names to factories that accept
// a parsed argument and return a leafEntry.
var paramFactories = map[string]func(string) (leafEntry, error){
	"player_nearby": func(arg string) (leafEntry, error) {
		v, err := strconv.ParseFloat(arg, 32)
		if err != nil {
			return leafEntry{}, fmt.Errorf("player_nearby: invalid radius %q: %w", arg, err)
		}
		return leafEntry{isCond: true, cond: condPlayerNearby(float32(v))}, nil
	},
	"phase_eq": func(arg string) (leafEntry, error) {
		v, err := strconv.Atoi(arg)
		if err != nil {
			return leafEntry{}, fmt.Errorf("phase_eq: invalid phase %q: %w", arg, err)
		}
		return leafEntry{isCond: true, cond: condPhaseEq(v)}, nil
	},
	"target_beyond": func(arg string) (leafEntry, error) {
		v, err := strconv.ParseFloat(arg, 32)
		if err != nil {
			return leafEntry{}, fmt.Errorf("target_beyond: invalid distance %q: %w", arg, err)
		}
		return leafEntry{isCond: true, cond: condTargetBeyond(float32(v))}, nil
	},
	"players_in_aoe": func(arg string) (leafEntry, error) {
		v, err := strconv.ParseFloat(arg, 32)
		if err != nil {
			return leafEntry{}, fmt.Errorf("players_in_aoe: invalid radius %q: %w", arg, err)
		}
		return leafEntry{isCond: true, cond: condPlayersInAoE(float32(v))}, nil
	},
	"n_players_clustered": func(arg string) (leafEntry, error) {
		v, err := strconv.Atoi(arg)
		if err != nil {
			return leafEntry{}, fmt.Errorf("n_players_clustered: invalid count %q: %w", arg, err)
		}
		return leafEntry{isCond: true, cond: condNPlayersClustered(v)}, nil
	},
	"any_below_hp_pct": func(arg string) (leafEntry, error) {
		v, err := strconv.ParseFloat(arg, 32)
		if err != nil {
			return leafEntry{}, fmt.Errorf("any_below_hp_pct: invalid pct %q: %w", arg, err)
		}
		return leafEntry{isCond: true, cond: condAnyBelowHPPct(float32(v))}, nil
	},
	"commit": func(arg string) (leafEntry, error) {
		if arg == "" {
			return leafEntry{}, errors.New("commit: missing ability ID")
		}
		return leafEntry{action: commitByName(arg)}, nil
	},
	"ability_ready": func(arg string) (leafEntry, error) {
		if arg == "" {
			return leafEntry{}, errors.New("ability_ready: missing ability ID")
		}
		return leafEntry{isCond: true, cond: condAbilityReady(arg)}, nil
	},
}

// resolveLeaf converts a leaf name string into a bt.Node. It handles:
//   - "!" prefix for inversion
//   - built-in subtrees: "attack", "aggro_or_patrol"
//   - parameterized leaves: "player_nearby(8)", "phase_eq(2)"
//   - simple leaf lookup from leafRegistry
func resolveLeaf(name string) (bt.Node, error) {
	// Inverter prefix
	if strings.HasPrefix(name, "!") {
		inner, err := resolveLeaf(name[1:])
		if err != nil {
			return nil, err
		}
		return bt.Named("!"+unwrapName(inner), bt.NewInverter(inner)), nil
	}

	// Built-in subtrees
	switch name {
	case "attack":
		return attackSubtree(), nil
	case "aggro_or_patrol":
		return aggroOrPatrolSubtree(), nil
	case "combat_subtree":
		return combatSubtree(), nil
	}

	// Parameterized leaves: name(arg)
	if idx := strings.Index(name, "("); idx > 0 && strings.HasSuffix(name, ")") {
		baseName := name[:idx]
		arg := name[idx+1 : len(name)-1]
		factory, ok := paramFactories[baseName]
		if !ok {
			return nil, fmt.Errorf("unknown parameterized leaf: %q", baseName)
		}
		entry, err := factory(arg)
		if err != nil {
			return nil, err
		}
		return bt.Named(name, entryToNode(entry)), nil
	}

	// Simple lookup
	entry, ok := leafRegistry[name]
	if !ok {
		return nil, fmt.Errorf("unknown leaf: %q", name)
	}
	return bt.Named(name, entryToNode(entry)), nil
}

// unwrapName extracts the label from a NamedNode, or returns "" if not named.
func unwrapName(n bt.Node) string {
	if nn, ok := n.(*bt.NamedNode); ok {
		return nn.Label
	}
	return "?"
}

func entryToNode(e leafEntry) bt.Node {
	if e.isCond {
		return bt.NewCondition(e.cond)
	}
	return bt.NewAction(e.action)
}

// aggroOrPatrolSubtree builds the standard "no target" handler:
//
//	Selector
//	├── Sequence [player_nearby(8) → aggro_nearest]
//	└── patrol
func aggroOrPatrolSubtree() bt.Node {
	return bt.NewSelector(
		bt.NewSequence(
			bt.Named("player_nearby(8)", bt.NewCondition(condPlayerNearby(8))),
			bt.Named("aggro_nearest", bt.NewAction(actionAggroNearest)),
		),
		bt.Named("patrol", bt.NewAction(actionPatrol)),
	)
}
