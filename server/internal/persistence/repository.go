package persistence

// Repository defines the persistence contract for player and character data.
type Repository interface {
	// UpsertPlayer creates a player if not found. Does NOT update username on existing players.
	UpsertPlayer(id, username string) error

	// GetPlayer returns a player by UUID, or nil if not found.
	GetPlayer(id string) (*Player, error)

	// CreateCharacter inserts a new character. Returns error if name is taken.
	CreateCharacter(c *Character) error

	// UpdateCharacterPosition updates position fields by character ID.
	UpdateCharacterPosition(charID uint, posX, posY, posZ, rotY float64) error

	// GetCharacterByID returns a character by ID, or nil if not found.
	GetCharacterByID(id uint) (*Character, error)

	// GetCharacters returns all characters for a player, ordered by updated_at DESC.
	GetCharacters(playerID string) ([]*Character, error)

	// IsCharacterNameTaken returns true if a character with the given name exists.
	IsCharacterNameTaken(name string) (bool, error)

	// CountCharacters returns the number of characters owned by a player.
	CountCharacters(playerID string) (int64, error)
}
