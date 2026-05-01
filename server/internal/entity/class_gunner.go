package entity

var gunnerDef = ClassDef{
	ID:        ClassGunner,
	MaxHealth: 150,
	Movement: ClassMovement{
		WalkSpeed: 5.5, SprintSpeed: 7.7, JumpVel: 4.0,
		GroundAccel: 25.0, GroundDecel: 18.0, AirAccel: 2.5, AirDecel: 1.0,
		RollSpeed: 14.0, RollDur: 0.3, RollCD: 2.5,
	},
	Resources: map[string]ResourceTemplate{},
	Abilities: []string{"fire_shot", "overclock", "rechamber", "rechamber_confirm", "dodge"},
	ActionMap: map[uint8]string{
		0:  "fire_shot",
		3:  "dodge",
		10: "overclock",
		11: "rechamber",
		12: "rechamber_confirm",
	},
}
