package entity

var arcanotechnicienDestroyerSpec = SpecDef{
	ID:          "destroyer",
	Name:        "Destroyer",
	Description: "Massive AoE burst. Glass cannon.\nHuge Flux reserve, long channels, vulnerable.",
	Role:        "DPS",
	Implemented: false,
	MaxHealth:   130,
	Movement: ClassMovement{
		WalkSpeed: 4.0, SprintSpeed: 5.5, JumpVel: 3.0,
		GroundAccel: 18.0, GroundDecel: 14.0, AirAccel: 2.0, AirDecel: 1.0,
		RollSpeed: 10.0, RollDur: 0.4, RollCD: 2.0,
	},
	Resources: map[string]ResourceTemplate{
		"flux": {Max: 200, Initial: 200, Regen: 8, RegenDelay: 0},
	},
}

var arcanotechnicienBattlemageSpec = SpecDef{
	ID:          "battlemage",
	Name:        "Battlemage",
	Description: "Melee-range hybrid. Monotarget, constant damage.\nAlternating weapon strikes and spells.",
	Role:        "DPS",
	Implemented: false,
	MaxHealth:   160,
	Movement: ClassMovement{
		WalkSpeed: 5.0, SprintSpeed: 7.0, JumpVel: 3.5,
		GroundAccel: 22.0, GroundDecel: 16.0, AirAccel: 2.5, AirDecel: 1.0,
		RollSpeed: 12.0, RollDur: 0.35, RollCD: 1.5,
	},
	Resources: map[string]ResourceTemplate{
		"flux": {Max: 120, Initial: 120, Regen: 6, RegenDelay: 0},
	},
}

var arcanotechnicienHarmonistSpec = SpecDef{
	ID:          "harmonist",
	Name:        "Harmonist",
	Description: "Flux-based healer. Positional, channeled, visible.\nHealing zones and beams, not whack-a-mole.",
	Role:        "Healer",
	Implemented: true,
	MaxHealth:   170,
	Movement: ClassMovement{
		WalkSpeed: 4.5, SprintSpeed: 6.5, JumpVel: 3.5,
		GroundAccel: 20.0, GroundDecel: 15.0, AirAccel: 2.0, AirDecel: 1.0,
		RollSpeed: 11.0, RollDur: 0.35, RollCD: 1.8,
	},
	Resources: map[string]ResourceTemplate{
		"flux": {Max: 160, Initial: 160, Regen: 7, RegenDelay: 0},
	},
	Abilities: []string{"mending_surge", "mending_beam", "dodge"},
	ActionMap: map[uint8]string{
		3:  "dodge",
		50: "mending_surge",
		51: "mending_beam",
	},
}

var arcanotechnicienDef = ClassDef{
	ID:          ClassArcanotechnicien,
	MaxHealth:   arcanotechnicienHarmonistSpec.MaxHealth,
	Movement:    arcanotechnicienHarmonistSpec.Movement,
	Resources:   arcanotechnicienHarmonistSpec.Resources,
	Abilities:   arcanotechnicienHarmonistSpec.Abilities,
	ActionMap:   arcanotechnicienHarmonistSpec.ActionMap,
	DefaultSpec: "harmonist",
	Specs:       []*SpecDef{&arcanotechnicienDestroyerSpec, &arcanotechnicienBattlemageSpec, &arcanotechnicienHarmonistSpec},
}
