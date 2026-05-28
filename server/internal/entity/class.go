package entity

// Class name constants.
const (
	ClassGunner           = "gunner"
	ClassVanguard         = "vanguard"
	ClassBladeDancer      = "blade_dancer"
	ClassArcanotechnicien = "arcanotechnicien"
)

// Spec name constants.
const (
	SpecHarmonist  = "harmonist"
	SpecShield     = "shield"
	SpecDestroyer  = "destroyer"
	SpecBattlemage = "battlemage"
	SpecBlade      = "blade"
	SpecShadow     = "shadow"
	SpecAssault    = "assault"
)

// Role constants.
const (
	RoleDPS    = "DPS"
	RoleHealer = "Healer"
	RoleTank   = "Tank"
)

// School name constants.
const (
	SchoolBioarcanotechnic = "bioarcanotechnic"
	SchoolBiometabolic     = "biometabolic"
	SchoolFrost            = "frost"
	SchoolFire             = "fire"
	SchoolElectricity      = "electricity"
	SchoolAerokinetic      = "aerokinetic"
	SchoolPure             = "pure"
	SchoolShadow           = "shadow"
	SchoolGravitonic       = "gravitonic"
	SchoolMartial          = "martial"
	SchoolHydrodynamic     = "hydrodynamic"
)

// Resource name constants.
const (
	ResourceFlux    = "flux"
	ResourceStamina = "stamina"
	ResourceShield  = "shield"
)

// Ability ID constants referenced in class definitions.
const (
	AbilityDodge       = "dodge"
	AbilityVgBlock     = "vg_block"
	AbilityVgBlockStop = "vg_block_stop"
	AbilityVortex      = "vortex"
	AbilityExecution   = "execution"
	AbilityMendingBeam = "mending_beam"
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

// SpecDef describes a specialization within a class.
type SpecDef struct {
	ID          string
	Name        string // display name: "Assault"
	Description string // short UI blurb
	Role        string // "DPS", "Tank", "Healer"
	Implemented bool   // false = coming soon
	MaxHealth   float32
	Movement    ClassMovement
	Resources   map[string]ResourceTemplate
	Abilities   []string
	ActionMap   map[uint8]string

	// School affinity (Arcanotechnicien): primary = 1.0x cost, secondary = 1.25x, off = 1.5x.
	PrimarySchools   []string
	SecondarySchools []string
}

// ClassDef describes a playable class: stats, resources, and available abilities.
type ClassDef struct {
	ID          string
	MaxHealth   float32
	Movement    ClassMovement
	Resources   map[string]ResourceTemplate // resource_name -> template
	Abilities   []string                    // available ability IDs
	ActionMap   map[uint8]string            // wire action_id -> ability_id
	Specs       []*SpecDef                  // specializations (first is default)
	DefaultSpec string                      // default spec ID
}

// GetSpec returns the spec with the given ID, or nil if not found.
func (c *ClassDef) GetSpec(id string) *SpecDef {
	for _, s := range c.Specs {
		if s.ID == id {
			return s
		}
	}
	return nil
}

// FirstSpec returns the default spec (first in the list).
func (c *ClassDef) FirstSpec() *SpecDef {
	if len(c.Specs) == 0 {
		return nil
	}
	if c.DefaultSpec != "" {
		if s := c.GetSpec(c.DefaultSpec); s != nil {
			return s
		}
	}
	return c.Specs[0]
}

// Classes is the global class registry.
var Classes = map[string]*ClassDef{
	ClassGunner:           &gunnerDef,
	ClassVanguard:         &vanguardDef,
	ClassBladeDancer:      &bladeDancerDef,
	ClassArcanotechnicien: &arcanotechnicienDef,
}
