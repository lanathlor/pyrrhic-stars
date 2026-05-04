package enemyai

import (
	"math"

	"codex-online/server/internal/combat"
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
			MeleeConeAngle:   math.Pi, // 180° — wide boss sweep
			DamageSourceType: SourceEnemyMelee,
		},
		{
			// Multi-emitter: aimed cone burst → radial spiral ring
			Name:              "fireball_burst",
			Type:              AbilityRanged,
			TargetStrategy:    TargetFarthest,
			TelegraphTime:     1.0,
			ExecuteTime:       0.1,
			CooldownTime:      1.5,
			BaseWeight:        25,
			MinRange:          3.0,
			FaceTarget:        false,
			TrackTarget:       true,
			ProjectileOriginY: 1.5,
			DamageSourceType:  SourceEnemyRanged,
			Pattern: &combat.PatternDef{
				Emitters: []combat.EmitterDef{
					{
						// Phase 1: cone burst aimed at target
						Type:         combat.EmitterCone,
						Count:        5,
						Waves:        3,
						WaveInterval: 0.2,
						ArcAngle:     40 * math.Pi / 180, // 40° spread
						Projectile: combat.ProjectileDef{
							Speed:    12,
							Damage:   10,
							Lifetime: 3,
						},
					},
					{
						// Phase 2: radial spiral ring
						Type:          combat.EmitterRadial,
						Count:         12,
						Waves:         2,
						WaveInterval:  0.15,
						OffsetPerWave: 15 * math.Pi / 180, // 15° spiral offset
						Projectile: combat.ProjectileDef{
							Speed:           6,
							Damage:          8,
							Lifetime:        4,
							AngularVelocity: 0.4, // curves outward
						},
					},
				},
			},
		},
		{
			// Contracting ring → radial burst. Teaches players to dodge inward then outward.
			Name:              "void_barrage",
			Type:              AbilityRanged,
			TargetStrategy:    TargetNearest,
			TelegraphTime:     1.3,
			ExecuteTime:       0.1,
			CooldownTime:      2.0,
			BaseWeight:        15,
			MinRange:          4.0,
			FaceTarget:        false,
			TrackTarget:       true,
			ProjectileOriginY: 0.0, // ground level for the ring
			DamageSourceType:  SourceEnemyRanged,
			Pattern: &combat.PatternDef{
				Emitters: []combat.EmitterDef{
					{
						// Ring contracts inward — dodge through it
						Type:        combat.EmitterRingContract,
						Count:       16,
						Waves:       1,
						StartRadius: 12,
						Projectile: combat.ProjectileDef{
							Speed:    5,
							Damage:   15,
							Lifetime: 3,
						},
					},
					{
						// Then radial burst — punish players who dodged to center
						Type:          combat.EmitterRadial,
						Count:         8,
						Waves:         2,
						WaveInterval:  0.1,
						OffsetPerWave: 22.5 * math.Pi / 180, // half-step offset between waves
						Projectile: combat.ProjectileDef{
							Speed:    10,
							Damage:   8,
							Lifetime: 2.5,
						},
					},
				},
			},
		},
		{
			Name:             "ground_slam",
			Type:             AbilityAoE,
			TargetStrategy:   TargetNearest,
			TelegraphTime:    1.5,
			ExecuteTime:      0.1,
			CooldownTime:     1.5,
			BaseWeight:       15,
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
			BaseWeight:           15,
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
				"melee_swipe":    20,
				"fireball_burst": 25,
				"void_barrage":   20,
				"ground_slam":    15,
				"bull_charge":    20,
			},
			AbilityOverrides: map[string]AbilityOverride{
				"melee_swipe": {
					TelegraphTime: F32(0.9),
				},
				"fireball_burst": {
					TelegraphTime: F32(0.8),
				},
				"void_barrage": {
					TelegraphTime: F32(1.0),
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
			// Phase 3 at 30% HP — enraged, bullet-hell intensifies
			HPThresholdPct:   0.3,
			TransitionTime:   1.5,
			MoveSpeed:        6.0,
			CooldownOverride: 0.4,
			WeightOverrides: map[string]int{
				"melee_swipe":    15,
				"fireball_burst": 25,
				"void_barrage":   25,
				"ground_slam":    15,
				"bull_charge":    20,
			},
			AbilityOverrides: map[string]AbilityOverride{
				"melee_swipe": {
					TelegraphTime: F32(0.7),
					Damage:        F32(35.0),
				},
				"fireball_burst": {
					TelegraphTime: F32(0.6),
				},
				"void_barrage": {
					TelegraphTime: F32(0.8),
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

// DefRegistry maps def names to their definitions. Go-defined defs (Tier 3
// bosses) are registered here directly. YAML-loaded defs (Tier 1/2) are
// added at startup via LoadMobs().
var DefRegistry = map[string]*EnemyDef{
	"guard_captain": &GuardCaptain,
}
