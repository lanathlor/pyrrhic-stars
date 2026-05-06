package entity

var bladeDancerDef = ClassDef{
	ID:        ClassBladeDancer,
	MaxHealth: 150,
	Movement: ClassMovement{
		WalkSpeed: 6.0, SprintSpeed: 9.0, JumpVel: 3.5,
		GroundAccel: 20.0, GroundDecel: 15.0, AirAccel: 2.5, AirDecel: 1.0,
		RollSpeed: 15.0, RollDur: 0.2, RollCD: 0.5,
	},
	Resources: map[string]ResourceTemplate{
		"shield": {Max: 25, Initial: 0, Regen: -5, RegenDelay: 0},
	},
	Abilities: []string{
		"bd_guard", "dodge",
		// BD transition spells (30-49)
		"shielded_sweep", "guarded_thrust", "protected_scatter", "fortified_command",
		"reaping_guard", "cleaving_pierce", "slashing_spread", "sweeping_hex",
		"piercing_barrier", "focused_slash", "targeted_spread", "pinning_strike",
		"dispersed_shield", "rain_of_blades", "converging_strike", "chaos_bind",
		"commanding_ward", "royal_cleave", "decree_strike", "sovereign_scatter",
	},
	ActionMap: func() map[uint8]string {
		m := map[uint8]string{
			3: "dodge",
			4: "bd_guard",
		}
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
