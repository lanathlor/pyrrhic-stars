package enemyai

import "codex-online/server/internal/bt"

// buildTree constructs the BT for an enemy based on its definition name.
func buildTree(def *EnemyDef, _ *EntityContext) *bt.Tree {
	switch def.Name {
	case "guard_captain":
		return bt.NewTree(guardCaptainTree())
	case "hallway_ranged":
		return bt.NewTree(hallwayRangedTree())
	default:
		return bt.NewTree(hallwayMeleeTree())
	}
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

// hallwayMeleeTree builds a simple melee mob tree:
//
//	ReactiveSelector
//	├── Sequence [is_dead → stop]
//	├── Sequence [phase_transitioning → wait_transition]
//	├── Sequence [NOT has_target → patrol_or_aggro]
//	├── Sequence [NOT in_leash_range → leash_reset]
//	├── Sequence [target_in_melee → has_los → attack]
//	└── chase
func hallwayMeleeTree() bt.Node {
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
		// In melee range with LoS → attack
		bt.NewSequence(
			bt.NewCondition(condTargetInMeleeRange),
			bt.NewCondition(condHasLoS),
			attackSubtree(),
		),
		// Chase
		bt.NewAction(actionChase),
	)
}

// hallwayRangedTree builds a ranged mob tree. Same structure as melee but
// the chase action handles backpedaling and preferred range via the EnemyDef.
func hallwayRangedTree() bt.Node {
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
		// Has LoS → attack at range
		bt.NewSequence(
			bt.NewCondition(condHasLoS),
			attackSubtree(),
		),
		// Chase / maintain range
		bt.NewAction(actionChase),
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
