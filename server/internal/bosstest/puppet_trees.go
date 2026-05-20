package bosstest

import (
	"codex-online/server/internal/bt"
	"codex-online/server/internal/entity"
)

// classTree returns the behavior tree for the given class.
func classTree(class string) *bt.Tree {
	switch class {
	case entity.ClassVanguard:
		return vanguardTree()
	case entity.ClassBladeDancer:
		return bladeDancerTree()
	default:
		return gunnerTree()
	}
}

// gunnerTree builds the BT for a ranged DPS that can shoot while moving.
// Gunner fires while dodging — all dodges use quick reaction (no DodgeGreed)
// because withCast means zero DPS loss while repositioning.
func gunnerTree() *bt.Tree {
	return bt.NewTree(bt.NewReactiveSelector(
		// Mechanics: dodge dangerous attacks while shooting
		bt.NewSequence(
			bt.NewCondition(condHasReactedQuick),
			bt.NewCondition(condInChargePath),
			bt.NewAction(withCast(actionStrafeCharge, "fire_shot")),
		),
		bt.NewSequence(
			bt.NewCondition(condHasReactedQuick),
			bt.NewCondition(condInAoEDanger),
			bt.NewAction(withCast(actionFleeAoE, "fire_shot")),
		),
		bt.NewSequence(
			bt.NewCondition(condHasReactedQuick),
			bt.NewCondition(condTargetedByRanged),
			bt.NewAction(withCast(actionStrafeRanged, "fire_shot")),
		),
		// Projectile dodge: sidestep while shooting (immediate, no reaction gate)
		bt.NewSequence(
			bt.NewCondition(condProjectileIncoming),
			bt.NewAction(withCast(actionSidestepProjectile, "fire_shot")),
		),
		// Positioning: kite if too close (still shooting)
		bt.NewSequence(
			bt.NewCondition(condTooClose),
			bt.NewAction(actionKiteAndShoot),
		),
		// Positioning: advance if too far
		bt.NewSequence(
			bt.NewCondition(condTooFar),
			bt.NewAction(actionAdvance),
		),
		// Rotation: use overclock buff when available (15s CD)
		bt.NewSequence(
			bt.NewCondition(canCastAbility("overclock")),
			bt.NewAction(castAbilityAction("overclock")),
		),
		// Tactical reload when magazine is low (1.5s vs 2.2s empty)
		bt.NewSequence(
			bt.NewCondition(condShouldReload),
			bt.NewAction(castAbilityAction("reload")),
		),
		// Filler: spam fire_shot
		bt.NewAction(castAbilityAction("fire_shot")),
	))
}

// vanguardTree builds the BT for a melee tank that stays in melee range.
// Vanguard circles behind boss during melee telegraphs (cone is fixed, not tracking).
// Block is cast while strafing — shield up + movement simultaneously.
func vanguardTree() *bt.Tree {
	return bt.NewTree(bt.NewReactiveSelector(
		// Melee telegraph: strafe behind boss + block (shield up while circling)
		bt.NewSequence(
			bt.NewCondition(condHasReactedQuick),
			bt.NewCondition(condInMeleeDanger),
			bt.NewCondition(condShouldUseDefensive),
			bt.NewAction(withCast(actionStrafeMeleeCone, "vg_block")),
		),
		// Melee telegraph: strafe behind boss + attack (when not blocking)
		bt.NewSequence(
			bt.NewCondition(condHasReactedQuick),
			bt.NewCondition(condInMeleeDanger),
			bt.NewAction(withCast(actionStrafeMeleeCone, "cleave")),
		),
		// AoE: flee radius while attacking
		bt.NewSequence(
			bt.NewCondition(condHasReactedQuick),
			bt.NewCondition(condInAoEDanger),
			bt.NewAction(withCast(actionFleeAoE, "cleave")),
		),
		// Charge: strafe out of path while attacking
		bt.NewSequence(
			bt.NewCondition(condHasReactedQuick),
			bt.NewCondition(condInChargePath),
			bt.NewAction(withCast(actionStrafeCharge, "cleave")),
		),
		// Ranged telegraph: strafe while attacking
		bt.NewSequence(
			bt.NewCondition(condHasReactedQuick),
			bt.NewCondition(condTargetedByRanged),
			bt.NewAction(withCast(actionStrafeRanged, "cleave")),
		),
		// Projectile dodge: sidestep while attacking (immediate, no reaction gate)
		bt.NewSequence(
			bt.NewCondition(condProjectileIncoming),
			bt.NewAction(withCast(actionSidestepProjectile, "cleave")),
		),
		// Positioning: get into melee range
		bt.NewSequence(
			bt.NewCondition(condOutOfMelee),
			bt.NewAction(actionAdvance),
		),
		// Rotation: big cooldowns first
		bt.NewSequence(
			bt.NewCondition(canCastAbility("vortex")),
			bt.NewAction(castAbilityAction("vortex")),
		),
		bt.NewSequence(
			bt.NewCondition(canCastAbility("execution")),
			bt.NewAction(castAbilityAction("execution")),
		),
		bt.NewSequence(
			bt.NewCondition(canCastAbility("upheaval")),
			bt.NewAction(castAbilityAction("upheaval")),
		),
		// Filler: light combo
		bt.NewAction(castAbilityAction("cleave")),
	))
}

// bladeDancerTree builds the BT for a mobile melee with config transitions.
// BD rotation is entirely configuration transitions — no basic attacks.
func bladeDancerTree() *bt.Tree {
	return bt.NewTree(bt.NewReactiveSelector(
		// Mechanics: dodge while transitioning (all use quick reaction — zero DPS cost)
		bt.NewSequence(
			bt.NewCondition(condHasReactedQuick),
			bt.NewCondition(condInChargePath),
			bt.NewAction(withTransition(actionStrafeCharge)),
		),
		bt.NewSequence(
			bt.NewCondition(condHasReactedQuick),
			bt.NewCondition(condInAoEDanger),
			bt.NewAction(withTransition(actionFleeAoE)),
		),
		bt.NewSequence(
			bt.NewCondition(condHasReactedQuick),
			bt.NewCondition(condInMeleeDanger),
			bt.NewAction(withTransition(actionStrafeMeleeCone)),
		),
		bt.NewSequence(
			bt.NewCondition(condHasReactedQuick),
			bt.NewCondition(condTargetedByRanged),
			bt.NewAction(withTransition(actionStrafeRanged)),
		),
		// Projectile dodge: sidestep while transitioning (immediate, no reaction gate)
		bt.NewSequence(
			bt.NewCondition(condProjectileIncoming),
			bt.NewAction(withTransition(actionSidestepProjectile)),
		),
		// Positioning: get into melee range
		bt.NewSequence(
			bt.NewCondition(condOutOfMelee),
			bt.NewAction(actionAdvance),
		),
		// Rotation: all transitions
		bt.NewSequence(
			bt.NewCondition(condCanTransition),
			bt.NewAction(actionCastBestTransition),
		),
		// Filler: best available transition
		bt.NewAction(actionCastBestTransition),
	))
}
