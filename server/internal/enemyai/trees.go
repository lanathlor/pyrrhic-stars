package enemyai

import (
	"fmt"

	"codex-online/server/internal/bt"
)

// buildTree constructs the BT for an enemy. The definition must have TreeData
// (loaded from YAML). All enemies — including Tier 3 bosses — are now
// data-driven.
func buildTree(def *EnemyDef, _ *EntityContext) *bt.Tree {
	if def.TreeData != nil {
		node, err := bt.BuildTreeFromYAML(def.TreeData, resolveLeaf)
		if err != nil {
			// TreeData is validated at load time, so this should not happen.
			panic(fmt.Sprintf("buildTree: %s: %v", def.Name, err))
		}
		return bt.NewTree(node)
	}

	// Fallback: check if a registered def with the same name has TreeData.
	// This handles inline test defs that borrow a registered mob's name.
	if reg := DefRegistry[def.Name]; reg != nil && reg.TreeData != nil {
		node, err := bt.BuildTreeFromYAML(reg.TreeData, resolveLeaf)
		if err != nil {
			panic(fmt.Sprintf("buildTree: %s (registry fallback): %v", def.Name, err))
		}
		return bt.NewTree(node)
	}

	panic(fmt.Sprintf("buildTree: no tree for %q (missing TreeData — was it loaded from YAML?)", def.Name))
}

// attackSubtree models the full attack lifecycle as a BT Sequence:
//
//	Sequence
//	├── actionSelectAbility     (pick ability, enter telegraph)
//	├── actionTelegraph         (wait telegraph timer, optional tracking)
//	├── Selector                (branch by ability type)
//	│   ├── Sequence [is_charge → charge_dash]
//	│   └── actionExecuteAbility (melee/ranged/AoE resolution)
//	└── actionCooldown          (wait cooldown, return to chase)
func attackSubtree() bt.Node {
	return bt.NewSequence(
		bt.Named("select_ability", bt.NewAction(actionSelectAbility)),
		bt.Named("telegraph", bt.NewAction(actionTelegraph)),
		bt.NewSelector(
			bt.NewSequence(
				bt.Named("is_charge?", bt.NewCondition(condActiveAbilityIsCharge)),
				bt.Named("charge_dash", bt.NewAction(actionChargeDash)),
			),
			bt.Named("execute_ability", bt.NewAction(actionExecuteAbility)),
		),
		bt.Named("cooldown", bt.NewAction(actionCooldown)),
	)
}

// combatSubtree is shared across phases. Phase differences come from
// PhaseDef overrides on the EnemyDef (weight changes, cooldown changes,
// speed changes) which are resolved at runtime by the leaf functions.
func combatSubtree() bt.Node {
	return bt.NewReactiveSelector(
		// In melee range with LoS → attack
		bt.NewSequence(
			bt.Named("in_melee?", bt.NewCondition(condTargetInMeleeRange)),
			bt.Named("has_los?", bt.NewCondition(condHasLoS)),
			attackSubtree(),
		),
		// Has LoS → attack from range
		bt.NewSequence(
			bt.Named("has_los?", bt.NewCondition(condHasLoS)),
			attackSubtree(),
		),
		// Chase toward target
		bt.Named("chase", bt.NewAction(actionChase)),
	)
}
