package item

// StatID identifies one of the six character stats.
type StatID uint8

const (
	StatHull     StatID = 0
	StatOutput   StatID = 1
	StatPlating  StatID = 2
	StatTempo    StatID = 3
	StatIdentity StatID = 4
	StatMastery  StatID = 5
	StatCount    StatID = 6 // sentinel — not a real stat
)

// SlotID identifies an equipment slot.
type SlotID uint8

const (
	SlotFrame         SlotID = 0
	SlotPowerCore     SlotID = 1
	SlotPrimaryWeapon SlotID = 2
	SlotSecondaryTool SlotID = 3
	SlotAugment       SlotID = 4
	SlotModule        SlotID = 5
	SlotCount         SlotID = 6 // sentinel — not a real slot
)

// SlotName returns a human-readable name for the slot.
func SlotName(s SlotID) string {
	switch s {
	case SlotFrame:
		return "Frame"
	case SlotPowerCore:
		return "Power Core"
	case SlotPrimaryWeapon:
		return "Primary Weapon"
	case SlotSecondaryTool:
		return "Secondary / Tool"
	case SlotAugment:
		return "Augment"
	case SlotModule:
		return "Module"
	default:
		return "Unknown"
	}
}

// StatName returns a human-readable name for the stat.
func StatName(s StatID) string {
	switch s {
	case StatHull:
		return "Hull"
	case StatOutput:
		return "Output"
	case StatPlating:
		return "Plating"
	case StatTempo:
		return "Tempo"
	case StatIdentity:
		return "Identity"
	case StatMastery:
		return "Mastery"
	default:
		return "Unknown"
	}
}

// StatLine is a single stat contribution on an item.
// Value is the base value at ilvl 1; it scales linearly with ilvl.
type StatLine struct {
	Stat  StatID  `yaml:"stat"`
	Value float32 `yaml:"value"`
}

// Item is a concrete item instance owned by a character.
type Item struct {
	ID    uint   // persistence primary key
	DefID string // references ItemDef.ID
	ILvl  int    // item level (determines stat magnitudes)
	Slot  SlotID // which equipment slot this fits
}

// Stats holds the six aggregated stat values for a character.
type Stats struct {
	Hull     float32
	Output   float32
	Plating  float32
	Tempo    float32
	Identity float32
	Mastery  float32
}

// Get returns the stat value for the given StatID.
func (s Stats) Get(id StatID) float32 {
	switch id {
	case StatHull:
		return s.Hull
	case StatOutput:
		return s.Output
	case StatPlating:
		return s.Plating
	case StatTempo:
		return s.Tempo
	case StatIdentity:
		return s.Identity
	case StatMastery:
		return s.Mastery
	default:
		return 0
	}
}
