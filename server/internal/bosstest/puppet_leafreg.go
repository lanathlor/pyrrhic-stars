package bosstest

import (
	"fmt"
	"strconv"
	"strings"

	"codex-online/server/internal/bt"
)

// puppetLeafEntry describes a single leaf in the puppet registry.
type puppetLeafEntry struct {
	isCond bool
	cond   func(any) bool
	action func(any) bt.Result
}

// puppetLeafRegistry maps YAML leaf names to puppet BT nodes.
// These operate on *PuppetContext (not *EntityContext like enemy AI leaves).
var puppetLeafRegistry = map[string]puppetLeafEntry{
	// Conditions — reaction & awareness
	"has_reacted":       {isCond: true, cond: condHasReacted},
	"has_reacted_quick": {isCond: true, cond: condHasReactedQuick},

	// Conditions — danger detection
	"in_aoe_danger":           {isCond: true, cond: condInAoEDanger},
	"in_charge_path":          {isCond: true, cond: condInChargePath},
	"in_melee_danger":         {isCond: true, cond: condInMeleeDanger},
	"targeted_by_ranged":      {isCond: true, cond: condTargetedByRanged},
	"projectile_incoming":     {isCond: true, cond: condProjectileIncoming},
	"boss_telegraphing_melee": {isCond: true, cond: condBossTelegraphingMelee},

	// Conditions — positioning
	"too_close":    {isCond: true, cond: condTooClose},
	"too_far":      {isCond: true, cond: condTooFar},
	"out_of_melee": {isCond: true, cond: condOutOfMelee},

	// Conditions — abilities & resources
	"should_use_defensive": {isCond: true, cond: condShouldUseDefensive},
	"can_transition":       {isCond: true, cond: condCanTransition},
	"should_reload":        {isCond: true, cond: condShouldReload},
	"is_blocking":          {isCond: true, cond: condIsBlocking},
	"block_stale":          {isCond: true, cond: condBlockStale},
	"has_devotion":         {isCond: true, cond: condHasDevotion},
	"stamina_low":          {isCond: true, cond: condStaminaLow},

	// Actions — movement (no commit)
	"strafe_charge":       {action: actionStrafeCharge},
	"flee_aoe":            {action: actionFleeAoE},
	"sidestep_projectile": {action: actionSidestepProjectile},
	"advance":             {action: actionAdvance},
	"backstep":            {action: actionBackstep},
	"strafe_melee_cone":   {action: actionStrafeMeleeCone},
	"strafe_ranged":       {action: actionStrafeRanged},
	"kite_and_shoot":      {action: actionKiteAndShoot},

	// Actions — abilities
	"commit_best_transition": {action: actionCommitBestTransition},
	"commit_block":           {action: actionCommitBlock},

	// Conditions — healer
	"ally_needs_emergency": {isCond: true, cond: condAllyNeedsEmergency},
	"flux_low":             {isCond: true, cond: condFluxLow},
	"has_vital_charge":     {isCond: true, cond: condHasVitalCharge},
	"is_channeling":        {isCond: true, cond: condIsChanneling},

	// Actions — healer movement
	"move_to_center": {action: actionMoveToCenter},
}

// movementActions maps movement leaf names to their action functions.
// When a movement action receives a parameter, it's wrapped with withCommit.
var movementActions = map[string]func(any) bt.Result{
	"strafe_charge":       actionStrafeCharge,
	"flee_aoe":            actionFleeAoE,
	"sidestep_projectile": actionSidestepProjectile,
	"advance":             actionAdvance,
	"backstep":            actionBackstep,
	"strafe_melee_cone":   actionStrafeMeleeCone,
	"strafe_ranged":       actionStrafeRanged,
}

// puppetParamFactories maps parameterized leaf base names to factories.
var puppetParamFactories = map[string]func(string) (puppetLeafEntry, error){
	"can_commit": func(arg string) (puppetLeafEntry, error) {
		return puppetLeafEntry{isCond: true, cond: canCommitAbility(arg)}, nil
	},
	"commit": func(arg string) (puppetLeafEntry, error) {
		return puppetLeafEntry{action: commitAbilityAction(arg)}, nil
	},

	// Healer conditions (parameterized)
	"ally_below_hp": func(arg string) (puppetLeafEntry, error) {
		pct, err := strconv.ParseFloat(arg, 32)
		if err != nil {
			return puppetLeafEntry{}, fmt.Errorf("ally_below_hp: invalid pct %q: %w", arg, err)
		}
		return puppetLeafEntry{isCond: true, cond: allyBelowHPPct(float32(pct))}, nil
	},
	"has_flux": func(arg string) (puppetLeafEntry, error) {
		amount, err := strconv.ParseFloat(arg, 32)
		if err != nil {
			return puppetLeafEntry{}, fmt.Errorf("has_flux: invalid amount %q: %w", arg, err)
		}
		return puppetLeafEntry{isCond: true, cond: condHasFlux(float32(amount))}, nil
	},
	// has_school_flux(school:amount) — e.g. has_school_flux(bioarcanotechnic:40)
	"has_school_flux": func(arg string) (puppetLeafEntry, error) {
		parts := strings.SplitN(arg, ":", 2)
		if len(parts) != 2 {
			return puppetLeafEntry{}, fmt.Errorf("has_school_flux: expected school:amount, got %q", arg)
		}
		amount, err := strconv.ParseFloat(parts[1], 32)
		if err != nil {
			return puppetLeafEntry{}, fmt.Errorf("has_school_flux: invalid amount %q: %w", parts[1], err)
		}
		return puppetLeafEntry{isCond: true, cond: condHasSchoolFlux(parts[0], float32(amount))}, nil
	},
	// school_flux_low(school) — e.g. school_flux_low(biometabolic)
	"school_flux_low": func(arg string) (puppetLeafEntry, error) {
		return puppetLeafEntry{isCond: true, cond: condSchoolFluxLow(arg)}, nil
	},
	"zone_active": func(arg string) (puppetLeafEntry, error) {
		return puppetLeafEntry{isCond: true, cond: condZoneActive(arg)}, nil
	},

	// Healer actions (parameterized)
	"heal_lowest": func(arg string) (puppetLeafEntry, error) {
		return puppetLeafEntry{action: healLowest(arg)}, nil
	},
	"heal_tank": func(arg string) (puppetLeafEntry, error) {
		return puppetLeafEntry{action: healTank(arg)}, nil
	},
	"drain_healthiest": func(arg string) (puppetLeafEntry, error) {
		return puppetLeafEntry{action: drainHealthiest(arg)}, nil
	},
	"place_zone": func(arg string) (puppetLeafEntry, error) {
		return puppetLeafEntry{action: placeZone(arg)}, nil
	},
}

func puppetEntryToNode(e puppetLeafEntry) bt.Node {
	if e.isCond {
		return bt.NewCondition(e.cond)
	}
	return bt.NewAction(e.action)
}

// resolvePuppetLeaf converts a leaf name string into a bt.Node.
// Handles "!" prefix, parameterized leaves, and movement+commit combos.
func resolvePuppetLeaf(name string) (bt.Node, error) {
	// Inverter prefix
	if strings.HasPrefix(name, "!") {
		inner, err := resolvePuppetLeaf(name[1:])
		if err != nil {
			return nil, err
		}
		label := "?"
		if nn, ok := inner.(*bt.NamedNode); ok {
			label = nn.Label
		}
		return bt.Named("!"+label, bt.NewInverter(inner)), nil
	}

	// Parameterized: name(arg)
	if idx := strings.Index(name, "("); idx > 0 && strings.HasSuffix(name, ")") {
		baseName := name[:idx]
		arg := name[idx+1 : len(name)-1]

		// Check explicit param factories first (can_commit, commit)
		if factory, ok := puppetParamFactories[baseName]; ok {
			entry, err := factory(arg)
			if err != nil {
				return nil, err
			}
			return bt.Named(name, puppetEntryToNode(entry)), nil
		}

		// Movement action + ability = withCommit (or withTransition for "transition")
		if moveAction, ok := movementActions[baseName]; ok {
			if arg == "transition" {
				wrapped := withTransition(moveAction)
				return bt.Named(name, bt.NewAction(wrapped)), nil
			}
			wrapped := withCommit(moveAction, arg)
			return bt.Named(name, bt.NewAction(wrapped)), nil
		}

		return nil, fmt.Errorf("unknown parameterized puppet leaf: %q", baseName)
	}

	// Simple lookup
	entry, ok := puppetLeafRegistry[name]
	if !ok {
		return nil, fmt.Errorf("unknown puppet leaf: %q", name)
	}
	return bt.Named(name, puppetEntryToNode(entry)), nil
}
