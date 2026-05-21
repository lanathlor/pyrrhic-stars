package bosstest

import (
	"codex-online/server/internal/bt"
	"codex-online/server/internal/entity"
)

// specTree returns the default behavior tree for the given class and spec.
// These are generic trees suitable for trash mobs. Boss-specific YAML trees
// override these during boss encounters.
func specTree(class, spec string) *bt.Tree {
	switch class {
	case entity.ClassVanguard:
		if spec == "shield" {
			return vanguardShieldTree()
		}
		return vanguardBladeTree()
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

// vanguardBladeTree builds the BT for Vanguard Blade spec (melee DPS).
// Circles behind boss during melee telegraphs (cone is fixed, not tracking).
// Block is cast while strafing — shield up + movement simultaneously.
func vanguardBladeTree() *bt.Tree {
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

// vanguardShieldTree builds the BT for Vanguard Shield spec (tank).
// Raises shield to block incoming damage, bashes while blocking, uses
// bull_rush and retaliate as burst when safe. Generic tree for trash mobs;
// boss-specific YAML trees add Brace timing and soak strategies.
func vanguardShieldTree() *bt.Tree {
	return bt.NewTree(bt.NewReactiveSelector(
		// Charge: always dodge (can't block geometry attacks)
		bt.NewSequence(
			bt.NewCondition(condHasReactedQuick),
			bt.NewCondition(condInChargePath),
			bt.NewAction(withCast(actionStrafeCharge, "shield_bash")),
		),
		// AoE: block through it if already blocking, otherwise flee
		bt.NewSequence(
			bt.NewCondition(condHasReactedQuick),
			bt.NewCondition(condInAoEDanger),
			bt.NewCondition(condIsBlocking),
			bt.NewAction(castAbilityAction("shield_bash")),
		),
		bt.NewSequence(
			bt.NewCondition(condHasReactedQuick),
			bt.NewCondition(condInAoEDanger),
			bt.NewAction(withCast(actionFleeAoE, "shield_bash")),
		),
		// Ranged: raise shield and tank through projectiles
		bt.NewSequence(
			bt.NewCondition(condHasReactedQuick),
			bt.NewCondition(condTargetedByRanged),
			bt.NewCondition(condIsBlocking),
			bt.NewAction(castAbilityAction("shield_bash")),
		),
		bt.NewSequence(
			bt.NewCondition(condHasReactedQuick),
			bt.NewCondition(condTargetedByRanged),
			bt.NewAction(castAbilityAction("vg_shield_block")),
		),
		// Projectile: hold block if already blocking, sidestep otherwise
		bt.NewSequence(
			bt.NewCondition(condProjectileIncoming),
			bt.NewCondition(condIsBlocking),
			bt.NewAction(castAbilityAction("shield_bash")),
		),
		bt.NewSequence(
			bt.NewCondition(condProjectileIncoming),
			bt.NewAction(withCast(actionSidestepProjectile, "shield_bash")),
		),
		// Melee: block + bash through melee attacks
		bt.NewSequence(
			bt.NewCondition(condInMeleeDanger),
			bt.NewCondition(condIsBlocking),
			bt.NewAction(castAbilityAction("shield_bash")),
		),
		bt.NewSequence(
			bt.NewCondition(condHasReacted),
			bt.NewCondition(condInMeleeDanger),
			bt.NewCondition(condShouldUseDefensive),
			bt.NewAction(castAbilityAction("vg_shield_block")),
		),
		bt.NewSequence(
			bt.NewCondition(condHasReacted),
			bt.NewCondition(condInMeleeDanger),
			bt.NewAction(castAbilityAction("shield_bash")),
		),
		// Safe window: drop stale block
		bt.NewSequence(
			bt.NewCondition(condIsBlocking),
			bt.NewCondition(condBlockStale),
			bt.NewAction(castAbilityAction("vg_shield_block_stop")),
		),
		// Burst: retaliate when Devotion stacked
		bt.NewSequence(
			bt.NewCondition(condHasDevotion),
			bt.NewCondition(canCastAbility("retaliate")),
			bt.NewAction(castAbilityAction("retaliate")),
		),
		// Burst: bull rush on CD
		bt.NewSequence(
			bt.NewCondition(canCastAbility("bull_rush")),
			bt.NewAction(castAbilityAction("bull_rush")),
		),
		// Drop block when safe (no telegraph active)
		bt.NewSequence(
			bt.NewCondition(condIsBlocking),
			bt.NewAction(castAbilityAction("vg_shield_block_stop")),
		),
		// Positioning: get into melee range
		bt.NewSequence(
			bt.NewCondition(condOutOfMelee),
			bt.NewAction(actionAdvance),
		),
		// Filler: shield bash
		bt.NewAction(castAbilityAction("shield_bash")),
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
