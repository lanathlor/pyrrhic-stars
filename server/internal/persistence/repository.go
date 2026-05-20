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
}
