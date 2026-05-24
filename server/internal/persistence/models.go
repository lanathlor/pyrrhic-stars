package persistence

import "time"

// User represents a persistent user account identified by a client-generated UUID.
type User struct {
	ID        string `gorm:"primaryKey;size:36"`
	Username  string `gorm:"size:20"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Character represents a named character belonging to a user.
// A user may have multiple characters of the same class.
type Character struct {
	ID        uint   `gorm:"primaryKey"`
	UserID    string `gorm:"size:36;index"`
	ClassName string `gorm:"size:20"`
	SpecID    string `gorm:"size:20"`
	Name      string `gorm:"size:20;uniqueIndex"`
	PosX      float64
	PosY      float64
	PosZ      float64
	RotY      float64
	CreatedAt time.Time
	UpdatedAt time.Time
}

// CharacterItem represents an item instance owned by a character.
type CharacterItem struct {
	ID          uint   `gorm:"primaryKey"`
	CharacterID uint   `gorm:"index"`
	DefID       string `gorm:"size:40"`
	ILvl        int
	Slot        uint8 // item.SlotID — which equipment slot this item fits
	CreatedAt   time.Time
}

// CharacterEquipment maps a character's equipment slot to an item.
// One row per equipped slot. Items not referenced here are in the bag.
type CharacterEquipment struct {
	ID          uint  `gorm:"primaryKey"`
	CharacterID uint  `gorm:"uniqueIndex:idx_char_slot"`
	SlotID      uint8 `gorm:"uniqueIndex:idx_char_slot"`
	ItemID      uint  `gorm:"index"`
}

// CharacterFluxCommitment stores the flux school allocation for Arcanotechnicien characters.
// One row per school. Total percentages must equal 100.
type CharacterFluxCommitment struct {
	ID          uint   `gorm:"primaryKey"`
	CharacterID uint   `gorm:"uniqueIndex:idx_char_school"`
	School      string `gorm:"size:40;uniqueIndex:idx_char_school"`
	Percentage  uint8
}

// CharacterLoadout stores the 6-slot ability loadout for Arcanotechnicien characters.
type CharacterLoadout struct {
	ID          uint   `gorm:"primaryKey"`
	CharacterID uint   `gorm:"uniqueIndex"`
	Slot0       string `gorm:"size:40"`
	Slot1       string `gorm:"size:40"`
	Slot2       string `gorm:"size:40"`
	Slot3       string `gorm:"size:40"`
	Slot4       string `gorm:"size:40"`
	Slot5       string `gorm:"size:40"`
	UpdatedAt   time.Time
}

// CharacterLoadoutPreset stores a named loadout preset for Arcanotechnicien characters.
// Max 10 presets per character. Unique on (CharacterID, Name).
type CharacterLoadoutPreset struct {
	ID          uint   `gorm:"primaryKey"`
	CharacterID uint   `gorm:"uniqueIndex:idx_char_preset_name"`
	Name        string `gorm:"size:30;uniqueIndex:idx_char_preset_name"`
	Slot0       string `gorm:"size:40"`
	Slot1       string `gorm:"size:40"`
	Slot2       string `gorm:"size:40"`
	Slot3       string `gorm:"size:40"`
	Slot4       string `gorm:"size:40"`
	Slot5       string `gorm:"size:40"`
	Commitment  string `gorm:"size:200"` // "school:pct,school:pct,..."
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
