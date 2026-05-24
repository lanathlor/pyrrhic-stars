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
	case entity.ClassArcanotechnicien:
		return harmonistTree()
	default:
		return gunnerTree()
	}
}

// gunnerTree builds the BT for a ranged DPS that can shoot while moving.
// Gunner fires while dodging — all dodges use quick reaction (no DodgeGreed)
// because withCommit means zero DPS loss while repositioning.
func gunnerTree() *bt.Tree {
	return bt.NewTree(bt.NewReactiveSelector(
		// Mechanics: dodge dangerous attacks while shooting
		bt.NewSequence(
			bt.NewCondition(condHasReactedQuick),
			bt.NewCondition(condInChargePath),
			bt.NewAction(withCommit(actionStrafeCharge, "fire_shot")),
		),
		bt.NewSequence(
			bt.NewCondition(condHasReactedQuick),
			bt.NewCondition(condInAoEDanger),
			bt.NewAction(withCommit(actionFleeAoE, "fire_shot")),
		),
		bt.NewSequence(
			bt.NewCondition(condHasReactedQuick),
			bt.NewCondition(condTargetedByRanged),
			bt.NewAction(withCommit(actionStrafeRanged, "fire_shot")),
		),
		// Projectile dodge: sidestep while shooting (immediate, no reaction gate)
		bt.NewSequence(
			bt.NewCondition(condProjectileIncoming),
			bt.NewAction(withCommit(actionSidestepProjectile, "fire_shot")),
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
			bt.NewCondition(canCommitAbility("overclock")),
			bt.NewAction(commitAbilityAction("overclock")),
		),
		// Tactical reload when magazine is low (1.5s vs 2.2s empty)
		bt.NewSequence(
			bt.NewCondition(condShouldReload),
			bt.NewAction(commitAbilityAction("reload")),
		),
		// Filler: spam fire_shot
		bt.NewAction(commitAbilityAction("fire_shot")),
	))
}

// vanguardBladeTree builds the BT for Vanguard Blade spec (melee DPS).
// Circles behind boss during melee telegraphs (cone is fixed, not tracking).
// Block is committed while strafing — shield up + movement simultaneously.
func vanguardBladeTree() *bt.Tree {
	return bt.NewTree(bt.NewReactiveSelector(
		// Melee telegraph: strafe behind boss + block (shield up while circling)
		bt.NewSequence(
			bt.NewCondition(condHasReactedQuick),
			bt.NewCondition(condInMeleeDanger),
			bt.NewCondition(condShouldUseDefensive),
			bt.NewAction(withCommit(actionStrafeMeleeCone, "vg_block")),
		),
		// Melee telegraph: strafe behind boss + attack (when not blocking)
		bt.NewSequence(
			bt.NewCondition(condHasReactedQuick),
			bt.NewCondition(condInMeleeDanger),
			bt.NewAction(withCommit(actionStrafeMeleeCone, "cleave")),
		),
		// AoE: flee radius while attacking
		bt.NewSequence(
			bt.NewCondition(condHasReactedQuick),
			bt.NewCondition(condInAoEDanger),
			bt.NewAction(withCommit(actionFleeAoE, "cleave")),
		),
		// Charge: strafe out of path while attacking
		bt.NewSequence(
			bt.NewCondition(condHasReactedQuick),
			bt.NewCondition(condInChargePath),
			bt.NewAction(withCommit(actionStrafeCharge, "cleave")),
		),
		// Ranged telegraph: strafe while attacking
		bt.NewSequence(
			bt.NewCondition(condHasReactedQuick),
			bt.NewCondition(condTargetedByRanged),
			bt.NewAction(withCommit(actionStrafeRanged, "cleave")),
		),
		// Projectile dodge: sidestep while attacking (immediate, no reaction gate)
		bt.NewSequence(
			bt.NewCondition(condProjectileIncoming),
			bt.NewAction(withCommit(actionSidestepProjectile, "cleave")),
		),
		// Positioning: get into melee range
		bt.NewSequence(
			bt.NewCondition(condOutOfMelee),
			bt.NewAction(actionAdvance),
		),
		// Rotation: big cooldowns first
		bt.NewSequence(
			bt.NewCondition(canCommitAbility("vortex")),
			bt.NewAction(commitAbilityAction("vortex")),
		),
		bt.NewSequence(
			bt.NewCondition(canCommitAbility("execution")),
			bt.NewAction(commitAbilityAction("execution")),
		),
		bt.NewSequence(
			bt.NewCondition(canCommitAbility("upheaval")),
			bt.NewAction(commitAbilityAction("upheaval")),
		),
		// Filler: light combo
		bt.NewAction(commitAbilityAction("cleave")),
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
			bt.NewAction(withCommit(actionStrafeCharge, "shield_bash")),
		),
		// AoE: block through it if already blocking, otherwise flee
		bt.NewSequence(
			bt.NewCondition(condHasReactedQuick),
			bt.NewCondition(condInAoEDanger),
			bt.NewCondition(condIsBlocking),
			bt.NewAction(commitAbilityAction("shield_bash")),
		),
		bt.NewSequence(
			bt.NewCondition(condHasReactedQuick),
			bt.NewCondition(condInAoEDanger),
			bt.NewAction(withCommit(actionFleeAoE, "shield_bash")),
		),
		// Ranged: raise shield and tank through projectiles
		bt.NewSequence(
			bt.NewCondition(condHasReactedQuick),
			bt.NewCondition(condTargetedByRanged),
			bt.NewCondition(condIsBlocking),
			bt.NewAction(commitAbilityAction("shield_bash")),
		),
		bt.NewSequence(
			bt.NewCondition(condHasReactedQuick),
			bt.NewCondition(condTargetedByRanged),
			bt.NewAction(commitAbilityAction("vg_shield_block")),
		),
		// Projectile: hold block if already blocking, sidestep otherwise
		bt.NewSequence(
			bt.NewCondition(condProjectileIncoming),
			bt.NewCondition(condIsBlocking),
			bt.NewAction(commitAbilityAction("shield_bash")),
		),
		bt.NewSequence(
			bt.NewCondition(condProjectileIncoming),
			bt.NewAction(withCommit(actionSidestepProjectile, "shield_bash")),
		),
		// Melee: block + bash through melee attacks
		bt.NewSequence(
			bt.NewCondition(condInMeleeDanger),
			bt.NewCondition(condIsBlocking),
			bt.NewAction(commitAbilityAction("shield_bash")),
		),
		bt.NewSequence(
			bt.NewCondition(condHasReacted),
			bt.NewCondition(condInMeleeDanger),
			bt.NewCondition(condShouldUseDefensive),
			bt.NewAction(commitAbilityAction("vg_shield_block")),
		),
		bt.NewSequence(
			bt.NewCondition(condHasReacted),
			bt.NewCondition(condInMeleeDanger),
			bt.NewAction(commitAbilityAction("shield_bash")),
		),
		// Safe window: drop stale block
		bt.NewSequence(
			bt.NewCondition(condIsBlocking),
			bt.NewCondition(condBlockStale),
			bt.NewAction(commitAbilityAction("vg_shield_block_stop")),
		),
		// Burst: retaliate when Devotion stacked
		bt.NewSequence(
			bt.NewCondition(condHasDevotion),
			bt.NewCondition(canCommitAbility("retaliate")),
			bt.NewAction(commitAbilityAction("retaliate")),
		),
		// Burst: bull rush on CD
		bt.NewSequence(
			bt.NewCondition(canCommitAbility("bull_rush")),
			bt.NewAction(commitAbilityAction("bull_rush")),
		),
		// Drop block when safe (no telegraph active)
		bt.NewSequence(
			bt.NewCondition(condIsBlocking),
			bt.NewAction(commitAbilityAction("vg_shield_block_stop")),
		),
		// Positioning: get into melee range
		bt.NewSequence(
			bt.NewCondition(condOutOfMelee),
			bt.NewAction(actionAdvance),
		),
		// Filler: shield bash
		bt.NewAction(commitAbilityAction("shield_bash")),
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
			bt.NewAction(actionCommitBestTransition),
		),
		// Filler: best available transition
		bt.NewAction(actionCommitBestTransition),
	))
}

// harmonistTree builds the BT for a Harmonist healer.
// Modern WoW-style healer: weaves DPS (Siphon Pulse) between heals.
// ~40-60% of GCDs are spent on damage when group HP is healthy.
//
// Key differences from DPS/tank trees:
// - No withCommit on dodge branches (healer must interrupt cast to dodge)
// - !is_channeling guards prevent breaking active channels for non-emergency heals
// - Zones are prioritized over channels (instant, fire-and-forget)
// - Life Swap before channels (cheap setup that empowers filler heals)
// - Siphon Pulse as DPS filler (0 flux, heals lowest ally for 50% of damage)
func harmonistTree() *bt.Tree {
	return bt.NewTree(bt.NewReactiveSelector(
		// --- Dodge mechanics (pure dodge, no DPS — healer must interrupt channel to dodge) ---
		bt.NewSequence(
			bt.NewCondition(condHasReactedQuick),
			bt.NewCondition(condInChargePath),
			bt.NewAction(actionStrafeCharge),
		),
		bt.NewSequence(
			bt.NewCondition(condHasReactedQuick),
			bt.NewCondition(condInAoEDanger),
			bt.NewAction(actionFleeAoE),
		),
		bt.NewSequence(
			bt.NewCondition(condHasReactedQuick),
			bt.NewCondition(condInMeleeDanger),
			bt.NewAction(actionStrafeMeleeCone),
		),
		bt.NewSequence(
			bt.NewCondition(condHasReactedQuick),
			bt.NewCondition(condTargetedByRanged),
			bt.NewAction(actionStrafeRanged),
		),
		bt.NewSequence(
			bt.NewCondition(condProjectileIncoming),
			bt.NewAction(actionSidestepProjectile),
		),

		// --- Emergency heal: ally below 30% HP ---
		bt.NewSequence(
			bt.NewCondition(condAllyNeedsEmergency),
			bt.NewCondition(not(condIsChanneling)),
			bt.NewCondition(canCommitAbility("mending_surge")),
			bt.NewCondition(condHasSchoolFlux("bioarcanotechnic", 40)),
			bt.NewAction(healLowest("mending_surge")),
		),

		// --- Zone placement (fire-and-forget, high throughput per GCD) ---
		bt.NewSequence(
			bt.NewCondition(not(condIsChanneling)),
			bt.NewCondition(not(condZoneActive("restoration_matrix"))),
			bt.NewCondition(canCommitAbility("restoration_matrix")),
			bt.NewCondition(condHasSchoolFlux("bioarcanotechnic", 50)),
			bt.NewAction(placeZone("restoration_matrix")),
		),

		// --- Vital Bloom zone (in loadout for healing-focused builds) ---
		bt.NewSequence(
			bt.NewCondition(not(condIsChanneling)),
			bt.NewCondition(not(condZoneActive("vital_bloom"))),
			bt.NewCondition(canCommitAbility("vital_bloom")),
			bt.NewCondition(condHasSchoolFlux("biometabolic", 8)),
			bt.NewAction(placeZone("vital_bloom")),
		),

		// --- Life Swap empowerment (cheap setup for Vital Charge) ---
		bt.NewSequence(
			bt.NewCondition(not(condIsChanneling)),
			bt.NewCondition(not(condHasVitalCharge)),
			bt.NewCondition(canCommitAbility("life_swap")),
			bt.NewCondition(condHasSchoolFlux("biometabolic", 5)),
			bt.NewAction(drainHealthiest("life_swap")),
		),

		// --- Tank sustain: mending_beam channel on damaged tank ---
		bt.NewSequence(
			bt.NewCondition(allyBelowHPPct(0.60)),
			bt.NewCondition(not(condIsChanneling)),
			bt.NewCondition(canCommitAbility("mending_beam")),
			bt.NewCondition(condHasSchoolFlux("bioarcanotechnic", 8)),
			bt.NewAction(healTank("mending_beam")),
		),

		// --- Group sustain: transfusion when biometabolic flux is healthy ---
		bt.NewSequence(
			bt.NewCondition(allyBelowHPPct(0.75)),
			bt.NewCondition(not(condIsChanneling)),
			bt.NewCondition(not(condSchoolFluxLow("biometabolic"))),
			bt.NewCondition(canCommitAbility("transfusion")),
			bt.NewAction(healLowest("transfusion")),
		),

		// --- Reposition for Sympathetic Field coverage ---
		bt.NewSequence(
			bt.NewCondition(not(condIsChanneling)),
			bt.NewCondition(condTooFar),
			bt.NewAction(actionMoveToCenter),
		),

		// --- Spot heal: Mending Surge when ally below 70% and flux available ---
		bt.NewSequence(
			bt.NewCondition(not(condIsChanneling)),
			bt.NewCondition(allyBelowHPPct(0.70)),
			bt.NewCondition(canCommitAbility("mending_surge")),
			bt.NewCondition(condHasSchoolFlux("bioarcanotechnic", 40)),
			bt.NewAction(healLowest("mending_surge")),
		),

		// --- Group sustain: transfusion when biometabolic flux is healthy ---
		// Only in loadout for healing-focused builds (bad profile).
		bt.NewSequence(
			bt.NewCondition(allyBelowHPPct(0.85)),
			bt.NewCondition(not(condIsChanneling)),
			bt.NewCondition(not(condSchoolFluxLow("biometabolic"))),
			bt.NewCondition(canCommitAbility("transfusion")),
			bt.NewAction(healLowest("transfusion")),
		),

		// --- Sustained DPS: Vital Drain (channel on boss, cancel-on-damage) ---
		// Only in loadout for DPS-healer builds (sweaty/average profile).
		bt.NewSequence(
			bt.NewCondition(not(condIsChanneling)),
			bt.NewCondition(not(allyBelowHPPct(0.80))),
			bt.NewCondition(canCommitAbility("vital_drain")),
			bt.NewCondition(condHasSchoolFlux("biometabolic", 3)),
			bt.NewAction(commitAbilityAction("vital_drain")),
		),

		// --- DPS filler: Siphon Pulse (0 flux, heals lowest ally) ---
		bt.NewSequence(
			bt.NewCondition(not(condIsChanneling)),
			bt.NewAction(commitAbilityAction("siphon_pulse")),
		),
	))
}

// not wraps a condition function to return its negation.
func not(cond func(any) bool) func(any) bool {
	return func(v any) bool {
		return !cond(v)
	}
}
