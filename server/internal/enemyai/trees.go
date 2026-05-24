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

// attackSubtree picks a weighted-random ability and waits for the runner to
// complete the full lifecycle (commit → execute → cooldown).
//
//	Sequence
//	├── cast_weighted   (pick ability, runner enters commit)
//	└── wait_ability    (Running until runner returns to idle)
func attackSubtree() bt.Node {
	return bt.NewSequence(
		bt.Named("commit_weighted", bt.NewAction(actionCommitWeighted)),
		bt.Named("wait_ability", bt.NewAction(actionWaitAbility)),
	)
}

// combatSubtree is shared across phases. Phase differences come from
// PhaseDef overrides on the EnemyDef (weight changes, cooldown changes,
// speed changes) which are resolved at runtime by the leaf functions.
//
// The is_casting guard prevents the ReactiveSelector from re-entering
// attackSubtree while an ability is already in progress.
func combatSubtree() bt.Node {
	return bt.NewReactiveSelector(
		// Continue active ability (prevent interruption)
		bt.NewSequence(
			bt.Named("is_casting?", bt.NewCondition(condIsCommitted)),
			bt.Named("wait_ability", bt.NewAction(actionWaitAbility)),
		),
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
