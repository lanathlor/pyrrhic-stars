package entity

var gunnerAssaultSpec = SpecDef{
	ID:          SpecAssault,
	Name:        "Assault",
	Description: "High fire rate, aggressive repositioning.\nRelentless aggression with movement mastery.",
	Role:        RoleDPS,
	Implemented: true,
	MaxHealth:   150,
	Movement: ClassMovement{
		WalkSpeed: 5.5, SprintSpeed: 7.7, JumpVel: 4.0,
		GroundAccel: 25.0, GroundDecel: 18.0, AirAccel: 2.5, AirDecel: 1.0,
		RollSpeed: 14.0, RollDur: 0.3, RollCD: 2.5,
	},
	Resources: map[string]ResourceTemplate{
		"munitions": {Max: 5, Initial: 5, Regen: 0.10, RegenDelay: 0},
	},
	Abilities: []string{"fire_shot", "overclock", "rechamber", "rechamber_confirm", AbilityDodge, "reload", "load_enhanced", "mag_dump"},
	ActionMap: map[uint8]string{
		0: "fire_shot", 3: AbilityDodge,
		10: "overclock", 11: "rechamber", 12: "rechamber_confirm",
		13: "reload", 14: "load_enhanced", 15: "mag_dump",
	},
}

var gunnerMarksmanSpec = SpecDef{
	ID:          "marksman",
	Name:        "Marksman",
	Description: "Slow, deliberate, perfect shots.\nSniper Elite — hold breath, one shot.",
	Role:        RoleDPS,
	Implemented: false,
	MaxHealth:   120,
	Movement: ClassMovement{
		WalkSpeed: 4.5, SprintSpeed: 6.5, JumpVel: 3.5,
		GroundAccel: 20.0, GroundDecel: 15.0, AirAccel: 2.0, AirDecel: 1.0,
		RollSpeed: 12.0, RollDur: 0.3, RollCD: 3.0,
	},
	Resources: map[string]ResourceTemplate{
		"munitions": {Max: 3, Initial: 3, Regen: 0.05, RegenDelay: 0},
	},
}

var gunnerChasseurSpec = SpecDef{
	ID:          "chasseur",
	Name:        "Chasseur",
	Description: "Grenades, EMP, area denial.\nRainbow Six tactical disruption.",
	Role:        RoleDPS,
	Implemented: false,
	MaxHealth:   140,
	Movement: ClassMovement{
		WalkSpeed: 5.0, SprintSpeed: 7.0, JumpVel: 3.5,
		GroundAccel: 22.0, GroundDecel: 16.0, AirAccel: 2.5, AirDecel: 1.0,
		RollSpeed: 13.0, RollDur: 0.3, RollCD: 2.5,
	},
	Resources: map[string]ResourceTemplate{
		"munitions": {Max: 4, Initial: 4, Regen: 0.08, RegenDelay: 0},
	},
}

var gunnerDef = ClassDef{
	ID:          ClassGunner,
	MaxHealth:   150,
	Movement:    gunnerAssaultSpec.Movement,
	Resources:   gunnerAssaultSpec.Resources,
	Abilities:   gunnerAssaultSpec.Abilities,
	ActionMap:   gunnerAssaultSpec.ActionMap,
	DefaultSpec: SpecAssault,
	Specs:       []*SpecDef{&gunnerAssaultSpec, &gunnerMarksmanSpec, &gunnerChasseurSpec},
}
