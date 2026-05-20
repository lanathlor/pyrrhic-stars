package entity

var bdMultiBladeSpec = SpecDef{
	ID:          "multi_blade",
	Name:        "Multi Blade",
	Description: "4-6 blades, scattered multi-target sustained.\nAoE constant damage, flowing between configurations.",
	Role:        "DPS",
	Implemented: true,
	MaxHealth:   150,
	Movement: ClassMovement{
		WalkSpeed: 6.0, SprintSpeed: 9.0, JumpVel: 3.5,
		GroundAccel: 20.0, GroundDecel: 15.0, AirAccel: 2.5, AirDecel: 1.0,
		RollSpeed: 15.0, RollDur: 0.2, RollCD: 0.5,
	},
	Resources: map[string]ResourceTemplate{
		"shield":    {Max: 25, Initial: 0, Regen: -5, RegenDelay: 0},
		"resonance": {Max: 100, Initial: 0, Regen: -2, RegenDelay: 0},
	},
	Abilities: []string{
		"bd_guard", "dodge",
		"shielded_sweep", "guarded_thrust", "protected_scatter", "fortified_command",
		"reaping_guard", "cleaving_pierce", "slashing_spread", "sweeping_hex",
		"piercing_barrier", "focused_slash", "targeted_spread", "pinning_strike",
		"dispersed_shield", "rain_of_blades", "converging_strike", "chaos_bind",
		"commanding_ward", "royal_cleave", "decree_strike", "sovereign_scatter",
	},
	ActionMap: func() map[uint8]string {
		m := map[uint8]string{3: "dodge", 4: "bd_guard"}
		bdSpellIDs := []string{
			"shielded_sweep", "guarded_thrust", "protected_scatter", "fortified_command",
			"reaping_guard", "cleaving_pierce", "slashing_spread", "sweeping_hex",
			"piercing_barrier", "focused_slash", "targeted_spread", "pinning_strike",
			"dispersed_shield", "rain_of_blades", "converging_strike", "chaos_bind",
			"commanding_ward", "royal_cleave", "decree_strike", "sovereign_scatter",
		}
		for i, id := range bdSpellIDs {
			m[uint8(30+i)] = id
		}
		return m
	}(),
}

var bdDualBladeSpec = SpecDef{
	ID:          "dual_blade",
	Name:        "Dual Blade",
	Description: "2 blades, separate GCDs, piano burst combos.\nMonotarget burst, highest skill ceiling.",
	Role:        "DPS",
	Implemented: false,
	MaxHealth:   130,
	Movement: ClassMovement{
		WalkSpeed: 6.5, SprintSpeed: 9.5, JumpVel: 4.0,
		GroundAccel: 22.0, GroundDecel: 16.0, AirAccel: 3.0, AirDecel: 1.0,
		RollSpeed: 16.0, RollDur: 0.2, RollCD: 0.4,
	},
	Resources: map[string]ResourceTemplate{
		"shield":    {Max: 15, Initial: 0, Regen: -5, RegenDelay: 0},
		"resonance": {Max: 120, Initial: 0, Regen: -3, RegenDelay: 0},
	},
}

var bladeDancerDef = ClassDef{
	ID:          ClassBladeDancer,
	MaxHealth:   150,
	Movement:    bdMultiBladeSpec.Movement,
	Resources:   bdMultiBladeSpec.Resources,
	Abilities:   bdMultiBladeSpec.Abilities,
	ActionMap:   bdMultiBladeSpec.ActionMap,
	DefaultSpec: "multi_blade",
	Specs:       []*SpecDef{&bdMultiBladeSpec, &bdDualBladeSpec},
}
