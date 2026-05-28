package character

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"codex-online/server/internal/entity"
	"codex-online/server/internal/persistence"
)

const maxCharacters = 100

var validClasses = map[string]bool{
	entity.ClassGunner:           true,
	entity.ClassVanguard:         true,
	entity.ClassBladeDancer:      true,
	entity.ClassArcanotechnicien: true,
}

var (
	ErrNotFound     = errors.New("character not found")
	ErrInvalidClass = errors.New("invalid class")
	ErrNameLength   = errors.New("name must be 2-20 characters")
	ErrNameChars    = errors.New("name must be alphanumeric (spaces, hyphens, underscores allowed)")
	ErrLimitReached = errors.New("character limit reached")
	ErrNameTaken    = errors.New("name already taken")
)

// Service handles character CRUD and validation.
type Service struct {
	repo persistence.Repository
}

// NewService creates a character service.
func NewService(repo persistence.Repository) *Service {
	return &Service{repo: repo}
}

// Select looks up a character and verifies it belongs to the user.
func (s *Service) Select(charID uint, userUUID string) (*persistence.Character, error) {
	char, err := s.repo.GetCharacterByID(charID)
	if err != nil || char == nil || char.UserID != userUUID {
		return nil, ErrNotFound
	}
	return char, nil
}

// Create validates inputs, checks limits and uniqueness, and persists a new character.
func (s *Service) Create(userUUID, className, charName string) (*persistence.Character, error) {
	if !validClasses[className] {
		return nil, ErrInvalidClass
	}

	charName = strings.TrimSpace(charName)
	if len(charName) < 2 || len(charName) > 20 {
		return nil, ErrNameLength
	}
	for _, r := range charName {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != ' ' && r != '-' && r != '_' {
			return nil, ErrNameChars
		}
	}

	count, err := s.repo.CountCharacters(userUUID)
	if err != nil {
		slog.Error("count characters", "error", err)
		return nil, fmt.Errorf("check limit: %w", err)
	}
	if count >= maxCharacters {
		return nil, ErrLimitReached
	}

	taken, err := s.repo.IsCharacterNameTaken(charName)
	if err != nil {
		slog.Error("check name taken", "error", err)
		return nil, ErrNameTaken
	}
	if taken {
		return nil, ErrNameTaken
	}

	char := &persistence.Character{
		UserID:    userUUID,
		ClassName: className,
		Name:      charName,
	}
	if err := s.repo.CreateCharacter(char); err != nil {
		slog.Error("create character", "error", err)
		return nil, ErrNameTaken
	}

	return char, nil
}
