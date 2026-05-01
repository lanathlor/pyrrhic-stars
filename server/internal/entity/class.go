package entity

// Class name constants.
const (
	ClassGunner      = "gunner"
	ClassVanguard    = "vanguard"
	ClassBladeDancer = "blade_dancer"
)

// ClassMovement holds per-class movement tuning.
type ClassMovement struct {
	WalkSpeed   float32
	SprintSpeed float32
	JumpVel     float32
	GroundAccel float32
	GroundDecel float32
	AirAccel    float32
	AirDecel    float32
	RollSpeed   float32
	RollDur     float32
	RollCD      float32
}

// ResourceTemplate defines a resource's initial state for a class.
type ResourceTemplate struct {
	Max        float32
	Initial    float32
	Regen      float32 // per-second regen (negative = decay)
	RegenDelay float32 // seconds after spending before regen starts
}

// ClassDef describes a playable class: stats, resources, and available abilities.
type ClassDef struct {
	ID        string
	MaxHealth float32
	Movement  ClassMovement
	Resources map[string]ResourceTemplate // resource_name -> template
	Abilities []string                    // available ability IDs
	ActionMap map[uint8]string            // wire action_id -> ability_id
}

// Classes is the global class registry.
var Classes = map[string]*ClassDef{
	ClassGunner:      &gunnerDef,
	ClassVanguard:    &vanguardDef,
	ClassBladeDancer: &bladeDancerDef,
}
