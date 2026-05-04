package enemyai

import (
	"fmt"

	"codex-online/server/internal/bt"
)

// buildTree constructs the BT for an enemy. If the definition has TreeData
// (loaded from YAML), the tree is built from data. Otherwise falls through
// to hardcoded Go tree builders for Tier 3 bosses.
func buildTree(def *EnemyDef, _ *EntityContext) *bt.Tree {
	if def.TreeData != nil {
		node, err := buildTreeFromData(def.TreeData)
		if err != nil {
			// TreeData is validated at load time, so this should not happen.
			panic(fmt.Sprintf("buildTree: %s: %v", def.Name, err))
		}
		return bt.NewTree(node)
	}

	if def.Name == "guard_captain" {
		return bt.NewTree(guardCaptainTree())
	}

	// Fallback: check if a registered def with the same name has TreeData.
	// This handles inline test defs that borrow a registered mob's name.
	if reg := DefRegistry[def.Name]; reg != nil && reg.TreeData != nil {
		node, err := buildTreeFromData(reg.TreeData)
		if err != nil {
			panic(fmt.Sprintf("buildTree: %s (registry fallback): %v", def.Name, err))
		}
		return bt.NewTree(node)
	}

	panic(fmt.Sprintf("buildTree: no tree for %q", def.Name))
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
		bt.NewAction(actionSelectAbility),
		bt.NewAction(actionTelegraph),
		bt.NewSelector(
			bt.NewSequence(
				bt.NewCondition(condActiveAbilityIsCharge),
				bt.NewAction(actionChargeDash),
			),
			bt.NewAction(actionExecuteAbility),
		),
		bt.NewAction(actionCooldown),
	)
}

// guardCaptainTree builds a phase-aware boss tree:
//
//	Selector
//	├── Sequence [dead → stop]
//	├── Sequence [transitioning → wait]
//	├── Sequence [no target → aggro/patrol]
//	├── Sequence [out of leash → reset]
//	├── Sequence [phase 3 → phase3 subtree]
//	├── Sequence [phase 2 → phase2 subtree]
//	└── phase1 subtree
func guardCaptainTree() bt.Node {
	return bt.NewReactiveSelector(
		// Dead
		bt.NewSequence(
			bt.NewCondition(condIsDead),
			bt.NewAction(actionStop),
		),
		// Phase transition
		bt.NewSequence(
			bt.NewCondition(condPhaseTransitioning),
			bt.NewAction(actionWaitTransition),
		),
		// No target → try aggro or patrol
		bt.NewSequence(
			bt.NewInverter(bt.NewCondition(condHasTarget)),
			bt.NewSelector(
				bt.NewSequence(
					bt.NewCondition(condPlayerNearby(8)),
					bt.NewAction(actionAggroNearest),
				),
				bt.NewAction(actionPatrol),
			),
		),
		// Leash check
		bt.NewSequence(
			bt.NewInverter(bt.NewCondition(condInLeashRange)),
			bt.NewAction(actionLeashReset),
		),
		// Phase 3 (enraged)
		bt.NewSequence(
			bt.NewCondition(condPhaseEq(3)),
			combatSubtree(),
		),
		// Phase 2
		bt.NewSequence(
			bt.NewCondition(condPhaseEq(2)),
			combatSubtree(),
		),
		// Phase 1 (default)
		combatSubtree(),
	)
}

// combatSubtree is shared across phases. Phase differences come from
// PhaseDef overrides on the EnemyDef (weight changes, cooldown changes,
// speed changes) which are resolved at runtime by the leaf functions.
func combatSubtree() bt.Node {
	return bt.NewReactiveSelector(
		// In melee range with LoS → attack
		bt.NewSequence(
			bt.NewCondition(condTargetInMeleeRange),
			bt.NewCondition(condHasLoS),
			attackSubtree(),
		),
		// Has LoS → attack from range
		bt.NewSequence(
			bt.NewCondition(condHasLoS),
			attackSubtree(),
		),
		// Chase toward target
		bt.NewAction(actionChase),
	)
}
