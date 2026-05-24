package persistence

// Repository defines the persistence contract for user and character data.
type Repository interface {
	// UpsertUser creates a user if not found. Does NOT update username on existing users.
	UpsertUser(id, username string) error

	// GetUser returns a user by UUID, or nil if not found.
	GetUser(id string) (*User, error)

	// CreateCharacter inserts a new character. Returns error if name is taken.
	CreateCharacter(c *Character) error

	// UpdateCharacterPosition updates position fields by character ID.
	UpdateCharacterPosition(charID uint, posX, posY, posZ, rotY float64) error

	// UpdateCharacterSpec updates the spec for a character.
	UpdateCharacterSpec(charID uint, specID string) error

	// GetCharacterByID returns a character by ID, or nil if not found.
	GetCharacterByID(id uint) (*Character, error)

	// GetCharacters returns all characters for a user, ordered by updated_at DESC.
	GetCharacters(userID string) ([]*Character, error)

	// IsCharacterNameTaken returns true if a character with the given name exists.
	IsCharacterNameTaken(name string) (bool, error)

	// CountCharacters returns the number of characters owned by a user.
	CountCharacters(userID string) (int64, error)

	// CreateItem inserts a new item instance for a character.
	CreateItem(item *CharacterItem) error

	// DeleteItem removes an item instance by ID.
	DeleteItem(itemID uint) error

	// GetItemsByCharacterID returns all items owned by a character.
	GetItemsByCharacterID(charID uint) ([]*CharacterItem, error)

	// SetEquipment equips an item in a slot for a character (upsert).
	SetEquipment(charID uint, slotID uint8, itemID uint) error

	// ClearEquipment removes the item from a slot for a character.
	ClearEquipment(charID uint, slotID uint8) error

	// GetEquipment returns all equipped slot mappings for a character.
	GetEquipment(charID uint) ([]*CharacterEquipment, error)

	// UpsertLoadout creates or updates the loadout for a character.
	UpsertLoadout(charID uint, slots [6]string) error

	// GetLoadout returns the loadout for a character, or nil if none exists.
	GetLoadout(charID uint) (*CharacterLoadout, error)

	// UpsertFluxCommitment replaces the flux commitment for a character.
	UpsertFluxCommitment(charID uint, entries []FluxCommitmentEntry) error

	// GetFluxCommitment returns the flux commitment for a character.
	GetFluxCommitment(charID uint) ([]FluxCommitmentEntry, error)

	// SaveLoadoutPreset creates or updates a named loadout preset. Max 10 per character.
	SaveLoadoutPreset(charID uint, name string, slots [6]string, commitment string) error

	// DeleteLoadoutPreset removes a preset by ID, only if owned by charID.
	DeleteLoadoutPreset(charID uint, presetID uint) error

	// GetLoadoutPresets returns all presets for a character, ordered by name.
	GetLoadoutPresets(charID uint) ([]*CharacterLoadoutPreset, error)
}

// FluxCommitmentEntry is a school+percentage pair for the repo interface.
type FluxCommitmentEntry struct {
	School     string
	Percentage uint8
}
