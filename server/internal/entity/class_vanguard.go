package entity

var vanguardDef = ClassDef{
	ID:        ClassVanguard,
	MaxHealth: 200,
	Movement: ClassMovement{
		WalkSpeed: 5.0, SprintSpeed: 7.0, JumpVel: 3.5,
		GroundAccel: 20.0, GroundDecel: 15.0, AirAccel: 2.5, AirDecel: 1.0,
		RollSpeed: 12.0, RollDur: 0.4, RollCD: 1.0,
	},
	Resources: map[string]ResourceTemplate{
		"stamina": {Max: 100, Initial: 100, Regen: 30, RegenDelay: 0.6},
	},
	Abilities: []string{
		"melee_light", "melee_heavy", "vg_block",
		"blade_swirl", "ground_slam", "dodge",
	},
	ActionMap: map[uint8]string{
		1:  "melee_light",
		2:  "melee_heavy",
		3:  "dodge",
		4:  "vg_block",
		20: "blade_swirl",
		21: "ground_slam",
	},
}
