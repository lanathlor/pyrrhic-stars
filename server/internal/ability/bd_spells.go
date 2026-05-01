package ability

import "codex-online/server/internal/entity"

// bdTransitionSpells builds all 20 Blade Dancer transition spell definitions.
func bdTransitionSpells() []*AbilityDef {
	type bdSpell struct {
		id        string
		name      string
		originCfg int
		destCfg   int

		hit         HitType
		damage      float32
		radius      float32
		targetCount int

		shieldHP   float32
		drFactor   float32
		drDuration float32

		dotDamage   float32
		dotDuration float32
		dotInterval float32
	}

	spells := []bdSpell{
		// From Orbit (Defense)
		{"shielded_sweep", "Shielded Sweep", 0, 1, HitAoECircle, 8, 4, 0, 0, 0.85, 2.0, 0, 0, 0},
		{"guarded_thrust", "Guarded Thrust", 0, 2, HitHitscan, 25, 0, 0, 8, 0, 0, 0, 0, 0},
		{"protected_scatter", "Protected Scatter", 0, 3, HitNearestN, 5, 0, 3, 0, 0.9, 1.5, 1.5, 12, 1},
		{"fortified_command", "Fortified Command", 0, 4, HitAoECircleTarget, 5, 5, 0, 0, 0.8, 2.0, 0, 0, 0},
		// From Fan (AoE Damage)
		{"reaping_guard", "Reaping Guard", 1, 0, HitAoECircle, 8, 3, 0, 12, 0, 0, 0, 0, 0},
		{"cleaving_pierce", "Cleaving Pierce", 1, 2, HitHitscan, 30, 0, 0, 0, 0, 0, 0, 0, 0},
		{"slashing_spread", "Slashing Spread", 1, 3, HitAoECircleTarget, 8, 5, 0, 0, 0, 0, 1.5, 10, 1},
		{"sweeping_hex", "Sweeping Hex", 1, 4, HitAoECircleTarget, 10, 5, 0, 0, 0, 0, 0, 0, 0},
		// From Lance (Single-target Damage)
		{"piercing_barrier", "Piercing Barrier", 2, 0, HitHitscan, 18, 0, 0, 15, 0, 0, 0, 0, 0},
		{"focused_slash", "Focused Slash", 2, 1, HitAoECircleTarget, 15, 4, 0, 0, 0, 0, 0, 0, 0},
		{"targeted_spread", "Targeted Spread", 2, 3, HitHitscan, 12, 0, 0, 0, 0, 0, 2.0, 15, 1},
		{"pinning_strike", "Pinning Strike", 2, 4, HitHitscan, 25, 0, 0, 0, 0, 0, 0, 0, 0},
		// From Scatter (Multi-target DoT)
		{"dispersed_shield", "Dispersed Shield", 3, 0, HitNone, 0, 0, 0, 18, 0.85, 2.0, 0, 0, 0},
		{"rain_of_blades", "Rain of Blades", 3, 1, HitAoECircleTarget, 15, 5, 0, 0, 0, 0, 1.0, 10, 1},
		{"converging_strike", "Converging Strike", 3, 2, HitHitscan, 32, 0, 0, 0, 0, 0, 1.5, 10, 1},
		{"chaos_bind", "Chaos Bind", 3, 4, HitNearestN, 8, 0, 4, 0, 0, 0, 0, 0, 0},
		// From Crown (Utility/Control)
		{"commanding_ward", "Commanding Ward", 4, 0, HitNone, 0, 0, 0, 20, 0, 0, 0, 0, 0},
		{"royal_cleave", "Royal Cleave", 4, 1, HitAoECircle, 12, 5, 0, 0, 0, 0, 0, 0, 0},
		{"decree_strike", "Decree Strike", 4, 2, HitHitscan, 28, 0, 0, 0, 0, 0, 0, 0, 0},
		{"sovereign_scatter", "Sovereign Scatter", 4, 3, HitNearestN, 5, 0, 3, 0, 0, 0, 1.5, 12, 1},
	}

	result := make([]*AbilityDef, 0, len(spells))
	for _, s := range spells {
		def := &AbilityDef{
			ID:           s.id,
			Name:         s.name,
			Hit:          HitDef{Type: s.hit, Radius: s.radius, TargetCount: s.targetCount},
			BaseDamage:   s.damage,
			GCD:          0.5,
			OriginConfig: s.originCfg,
			DestConfig:   s.destCfg,
			ShieldGrant:  s.shieldHP,
			ShieldCap:    25,
		}

		if s.hit == HitHitscan {
			def.Hit.Range = 20
		}

		if s.drFactor > 0 {
			def.SelfBuffs = append(def.SelfBuffs, BuffEffect{
				ID:       "bd_dr",
				Type:     entity.BuffDamageReduction,
				Value:    s.drFactor,
				Duration: s.drDuration,
			})
		}

		if s.dotDamage > 0 {
			def.TargetDoTs = append(def.TargetDoTs, DoTEffect{
				Damage:   s.dotDamage,
				Duration: s.dotDuration,
				Interval: s.dotInterval,
			})
		}

		result = append(result, def)
	}
	return result
}
