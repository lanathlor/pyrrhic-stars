package enemyai

import (
	"math"
)

// GuardCaptain is the first dungeon boss — a corrupted guard captain
// encountered in the Derelict City arena. Three-phase fight with melee,
// ranged, AoE, and charge attacks that escalate as HP drops.
var GuardCaptain = EnemyDef{
	Name:      "guard_captain",
	MaxHealth: 2000.0,
	MoveSpeed: 4.0,
	Radius:    1.0,

	AntiRepeat: 2.0,

	Abilities: []AbilityDef{
		{
			Name:             "melee_swipe",
			Type:             AbilityMelee,
			TargetStrategy:   TargetNearest,
			TelegraphTime:    1.2,
			ExecuteTime:      0.3,
			CooldownTime:     1.5,
			BaseWeight:       30,
			MaxRange:         3.0,
			FaceTarget:       true,
			MeleeRange:       3.0,
			MeleeDamage:      30.0,
			MeleeConeAngle:   math.Pi,          // 180° — wide boss sweep
			DamageSourceType: SourceEnemyMelee,
		},
		{
			Name:               "fireball_burst",
			Type:               AbilityRanged,
			TargetStrategy:     TargetFarthest,
			TelegraphTime:      1.0,
			ExecuteTime:        0.1,
			CooldownTime:       1.5,
			BaseWeight:         30,
			MinRange:           3.0,
			FaceTarget:         false,
			TrackTarget:        true,
			ProjectileCount:    1,
			ProjectileSpeed:    22.0,
			ProjectileDamage:   20.0,
			ProjectileSpread:   5.0 * math.Pi / 180.0,
			ProjectileOriginY:  1.5,
			ProjectileLifetime: 5.0,
			DamageSourceType:   SourceEnemyRanged,
		},
		{
			Name:             "ground_slam",
			Type:             AbilityAoE,
			TargetStrategy:   TargetNearest,
			TelegraphTime:    1.5,
			ExecuteTime:      0.1,
			CooldownTime:     1.5,
			BaseWeight:       20,
			MaxRange:         7.5,
			FaceTarget:       true,
			AoERadius:        5.0,
			AoEDamage:        40.0,
			DamageSourceType: SourceEnemyAoE,
		},
		{
			Name:                 "bull_charge",
			Type:                 AbilityCharge,
			TargetStrategy:       TargetNearest,
			TelegraphTime:        1.0,
			CooldownTime:         1.5,
			BaseWeight:           20,
			MinRange:             6.0,
			FaceTarget:           true,
			ChargeSpeed:          12.0,
			ChargeDamage:         35.0,
			ChargeMaxDistance:     15.0,
			ChargeHitRadius:      2.0,
			ChargeStopOnWall:     true,
			ChargeStopOnObstacle: true,
			DamageSourceType:     SourceEnemyCharge,
		},
	},

	Phases: []PhaseDef{
		{
			// Phase 2 at 60% HP
			HPThresholdPct:   0.6,
			TransitionTime:   1.5,
			MoveSpeed:        5.0,
			CooldownOverride: 1.2,
			WeightOverrides: map[string]int{
				"melee_swipe":    25,
				"fireball_burst": 25,
				"ground_slam":    25,
				"bull_charge":    25,
			},
			AbilityOverrides: map[string]AbilityOverride{
				"melee_swipe": {
					TelegraphTime: F32(0.9),
				},
				"fireball_burst": {
					TelegraphTime:   F32(0.8),
					Damage:          F32(15.0),
					ProjectileCount: Int(2),
				},
				"ground_slam": {
					TelegraphTime: F32(1.2),
					AoERadius:     F32(6.0),
				},
				"bull_charge": {
					TelegraphTime:    F32(0.8),
					ChargeSpeed:      F32(14.0),
					ChargeMaxDistance: F32(18.0),
				},
			},
		},
		{
			// Phase 3 at 30% HP — enraged, barely pauses between attacks
			HPThresholdPct:   0.3,
			TransitionTime:   1.5,
			MoveSpeed:        6.0,
			CooldownOverride: 0.4,
			WeightOverrides: map[string]int{
				"melee_swipe":    20,
				"fireball_burst": 20,
				"ground_slam":    25,
				"bull_charge":    35,
			},
			AbilityOverrides: map[string]AbilityOverride{
				"melee_swipe": {
					TelegraphTime: F32(0.7),
					Damage:        F32(35.0),
				},
				"fireball_burst": {
					TelegraphTime:   F32(0.6),
					Damage:          F32(12.0),
					ProjectileCount: Int(3),
				},
				"ground_slam": {
					TelegraphTime: F32(1.0),
					AoERadius:     F32(7.0),
					Damage:        F32(45.0),
				},
				"bull_charge": {
					TelegraphTime:    F32(0.6),
					Damage:           F32(40.0),
					ChargeSpeed:      F32(16.0),
					ChargeMaxDistance: F32(20.0),
				},
			},
		},
	},
}

// HallwayMelee is a simple melee-only grunt for hallway trash packs.
var HallwayMelee = EnemyDef{
	Name:      "hallway_melee",
	MaxHealth: 200.0,
	MoveSpeed: 5.0,
	Radius:    0.8,

	AntiRepeat: 1.0,

	Abilities: []AbilityDef{
		{
			Name:             "melee_slash",
			Type:             AbilityMelee,
			TargetStrategy:   TargetNearest,
			TelegraphTime:    0.5,
			ExecuteTime:      0.2,
			CooldownTime:     0.4,
			BaseWeight:       100,
			MaxRange:         2.5,
			FaceTarget:       true,
			MeleeRange:       2.5,
			MeleeDamage:      15.0,
			MeleeConeAngle:   120.0 * math.Pi / 180.0, // 120° — narrow mob swing
			DamageSourceType: SourceEnemyMelee,
		},
	},
}

// HallwayRanged is a ranged caster for hallway trash packs.
// Keeps distance from players, backpedals when too close.
var HallwayRanged = EnemyDef{
	Name:           "hallway_ranged",
	MaxHealth:      150.0,
	MoveSpeed:      3.5,
	PreferredRange: 8.0,
	BackpedalSpeed: 3.0,
	Radius:         0.8,

	AntiRepeat: 1.0,

	Abilities: []AbilityDef{
		{
			Name:               "energy_bolt",
			Type:               AbilityRanged,
			TargetStrategy:     TargetNearest,
			TelegraphTime:      0.6,
			ExecuteTime:        0.1,
			CooldownTime:       1.2,
			BaseWeight:         100,
			MinRange:           2.0,
			FaceTarget:         false,
			TrackTarget:        true,
			ProjectileCount:    1,
			ProjectileSpeed:    18.0,
			ProjectileDamage:   12.0,
			ProjectileSpread:   0,
			ProjectileOriginY:  1.2,
			ProjectileLifetime: 4.0,
			DamageSourceType:   SourceEnemyRanged,
		},
	},
}

// DefRegistry maps def names to their definitions.
var DefRegistry = map[string]*EnemyDef{
	"guard_captain":  &GuardCaptain,
	"hallway_melee":  &HallwayMelee,
	"hallway_ranged": &HallwayRanged,
}
