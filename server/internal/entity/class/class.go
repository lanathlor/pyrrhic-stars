package class

// Class name constants.
const (
	Gunner      = "gunner"
	Vanguard    = "vanguard"
	BladeDancer = "blade_dancer"
)

// Movement holds per-class movement tuning.
type Movement struct {
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

// Def describes a playable class: stats, resources, and available abilities.
type Def struct {
	ID        string
	MaxHealth float32
	Movement  Movement
	Resources map[string]ResourceTemplate // resource_name -> template
	Abilities []string                    // available ability IDs
	ActionMap map[uint8]string            // wire action_id -> ability_id
}

// Registry is the global class registry.
var Registry = map[string]*Def{
	Gunner:      &gunnerDef,
	Vanguard:    &vanguardDef,
	BladeDancer: &bladeDancerDef,
}
