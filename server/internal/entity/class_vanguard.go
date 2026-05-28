package entity

var vanguardBladeSpec = SpecDef{
	ID:          SpecBlade,
	Name:        "Blade",
	Description: "Cleave, Upheaval, Vortex, Execution.\nAoE burst with Onslaught momentum. Dynasty Warriors meets Dark Souls.",
	Role:        RoleDPS,
	Implemented: true,
	MaxHealth:   200,
	Movement: ClassMovement{
		WalkSpeed: 5.0, SprintSpeed: 7.0, JumpVel: 3.5,
		GroundAccel: 20.0, GroundDecel: 15.0, AirAccel: 2.5, AirDecel: 1.0,
		RollSpeed: 12.0, RollDur: 0.4, RollCD: 1.0,
	},
	Resources: map[string]ResourceTemplate{
		ResourceStamina: {Max: 100, Initial: 100, Regen: 30, RegenDelay: 0.6},
	},
	Abilities: []string{
		"cleave", "upheaval", AbilityVgBlock, AbilityVgBlockStop,
		AbilityVortex, AbilityExecution, AbilityDodge,
	},
	ActionMap: map[uint8]string{
		1: "cleave", 2: "upheaval", 3: AbilityDodge,
		4: AbilityVgBlock, 5: AbilityVgBlockStop,
		20: AbilityVortex, 21: AbilityExecution,
	},
}

var vanguardShieldSpec = SpecDef{
	ID:          SpecShield,
	Name:        "Shield",
	Description: "Directional block, absorbs for allies.\nMonster Hunter lance — slow, unbreakable.",
	Role:        RoleTank,
	Implemented: true,
	MaxHealth:   280,
	Movement: ClassMovement{
		WalkSpeed: 4.0, SprintSpeed: 5.5, JumpVel: 3.0,
		GroundAccel: 16.0, GroundDecel: 12.0, AirAccel: 2.0, AirDecel: 1.0,
		RollSpeed: 10.0, RollDur: 0.4, RollCD: 1.5,
	},
	Resources: map[string]ResourceTemplate{
		ResourceStamina: {Max: 120, Initial: 120, Regen: 25, RegenDelay: 0.8},
	},
	Abilities: []string{
		"shield_bash", "bull_rush", "vg_shield_block", "vg_shield_block_stop",
		"brace", "retaliate", "dodge",
	},
	ActionMap: map[uint8]string{
		1:  "shield_bash",
		2:  "bull_rush",
		3:  "dodge",
		4:  "vg_shield_block",
		5:  "vg_shield_block_stop",
		20: "brace",
		21: "retaliate",
	},
}

var vanguardShadowSpec = SpecDef{
	ID:          SpecShadow,
	Name:        "Shadow",
	Description: "Counters, flanking, sustained stealth pressure.\nSekiro — dodge, punish, repeat.",
	Role:        RoleDPS,
	Implemented: false,
	MaxHealth:   170,
	Movement: ClassMovement{
		WalkSpeed: 5.5, SprintSpeed: 8.0, JumpVel: 4.0,
		GroundAccel: 24.0, GroundDecel: 18.0, AirAccel: 3.0, AirDecel: 1.0,
		RollSpeed: 14.0, RollDur: 0.35, RollCD: 0.8,
	},
	Resources: map[string]ResourceTemplate{
		ResourceStamina: {Max: 80, Initial: 80, Regen: 35, RegenDelay: 0.4},
	},
}

var vanguardDef = ClassDef{
	ID:          ClassVanguard,
	MaxHealth:   200,
	Movement:    vanguardBladeSpec.Movement,
	Resources:   vanguardBladeSpec.Resources,
	Abilities:   vanguardBladeSpec.Abilities,
	ActionMap:   vanguardBladeSpec.ActionMap,
	DefaultSpec: SpecBlade,
	Specs:       []*SpecDef{&vanguardBladeSpec, &vanguardShieldSpec, &vanguardShadowSpec},
}
