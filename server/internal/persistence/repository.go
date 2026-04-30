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

	// GetCharacterByID returns a character by ID, or nil if not found.
	GetCharacterByID(id uint) (*Character, error)

	// GetCharacters returns all characters for a user, ordered by updated_at DESC.
	GetCharacters(userID string) ([]*Character, error)

	// IsCharacterNameTaken returns true if a character with the given name exists.
	IsCharacterNameTaken(name string) (bool, error)

	// CountCharacters returns the number of characters owned by a user.
	CountCharacters(userID string) (int64, error)
}
