package entity

var arcanotechnicienDestroyerSpec = SpecDef{
	ID:          SpecDestroyer,
	Name:        "Destroyer",
	Description: "Massive AoE burst. Glass cannon.\nHuge Flux reserve, long channels, vulnerable.",
	Role:        RoleDPS,
	Implemented: false,
	MaxHealth:   130,
	Movement: ClassMovement{
		WalkSpeed: 4.0, SprintSpeed: 5.5, JumpVel: 3.0,
		GroundAccel: 18.0, GroundDecel: 14.0, AirAccel: 2.0, AirDecel: 1.0,
		RollSpeed: 10.0, RollDur: 0.4, RollCD: 2.0,
	},
	Resources: map[string]ResourceTemplate{
		ResourceFlux: {Max: 200, Initial: 200, Regen: 8, RegenDelay: 0},
	},
	PrimarySchools:   []string{SchoolFire, SchoolFrost, SchoolElectricity},
	SecondarySchools: []string{SchoolGravitonic, SchoolAerokinetic, SchoolPure},
}

var arcanotechnicienBattlemageSpec = SpecDef{
	ID:          SpecBattlemage,
	Name:        "Battlemage",
	Description: "Melee-range hybrid. Monotarget, constant damage.\nAlternating weapon strikes and abilities.",
	Role:        RoleDPS,
	Implemented: false,
	MaxHealth:   160,
	Movement: ClassMovement{
		WalkSpeed: 5.0, SprintSpeed: 7.0, JumpVel: 3.5,
		GroundAccel: 22.0, GroundDecel: 16.0, AirAccel: 2.5, AirDecel: 1.0,
		RollSpeed: 12.0, RollDur: 0.35, RollCD: 1.5,
	},
	Resources: map[string]ResourceTemplate{
		ResourceFlux: {Max: 120, Initial: 120, Regen: 6, RegenDelay: 0},
	},
	PrimarySchools:   []string{SchoolElectricity, SchoolFire, SchoolMartial},
	SecondarySchools: []string{SchoolShadow, SchoolAerokinetic, SchoolPure},
}

// harmonistDefaultLoadout is the default loadout for the Harmonist spec.
// Players can customize this by swapping abilities from the class codex.
var harmonistDefaultLoadout = Loadout{
	Slots: [6]string{
		"siphon_pulse",
		AbilityMendingBeam,
		"mending_surge",
		"restoration_matrix",
		"life_swap",
		"vital_drain",
	},
}

var arcanotechnicienHarmonistSpec = SpecDef{
	ID:          SpecHarmonist,
	Name:        "Harmonist",
	Description: "Flux-based healer. Positional, channeled, visible.\nHealing zones and beams, not whack-a-mole.",
	Role:        RoleHealer,
	Implemented: true,
	MaxHealth:   170,
	Movement: ClassMovement{
		WalkSpeed: 4.5, SprintSpeed: 6.5, JumpVel: 3.5,
		GroundAccel: 20.0, GroundDecel: 15.0, AirAccel: 2.0, AirDecel: 1.0,
		RollSpeed: 11.0, RollDur: 0.35, RollCD: 1.8,
	},
	Resources: map[string]ResourceTemplate{
		ResourceFlux: {Max: 160, Initial: 160, Regen: 3, RegenDelay: 0},
	},
	Abilities:        []string{"siphon_pulse", "mending_surge", AbilityMendingBeam, "vital_bloom", "restoration_matrix", "life_swap", "transfusion", "vital_drain", "overclock_at", "neural_fortification", "regen_protocol", "vital_circuit", "metabolic_burst", "last_breath", "gust_step", "frost_ward"},
	ActionMap:        map[uint8]string{},
	PrimarySchools:   []string{SchoolBioarcanotechnic, SchoolBiometabolic, SchoolFrost},
	SecondarySchools: []string{SchoolAerokinetic, SchoolHydrodynamic, SchoolPure},
}

var arcanotechnicienDef = ClassDef{
	ID:          ClassArcanotechnicien,
	MaxHealth:   arcanotechnicienHarmonistSpec.MaxHealth,
	Movement:    arcanotechnicienHarmonistSpec.Movement,
	Resources:   arcanotechnicienHarmonistSpec.Resources,
	Abilities:   arcanotechnicienHarmonistSpec.Abilities,
	ActionMap:   arcanotechnicienHarmonistSpec.ActionMap,
	DefaultSpec: SpecHarmonist,
	Specs:       []*SpecDef{&arcanotechnicienDestroyerSpec, &arcanotechnicienBattlemageSpec, &arcanotechnicienHarmonistSpec},
}
