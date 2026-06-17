// Package settings stores per-user client settings (graphics, audio, keybinds)
// as an opaque JSON document. The server is the authoritative store so settings
// follow the account across devices; it never interprets the contents.
package settings

import (
	"encoding/json"
	"errors"

	"codex-online/server/internal/persistence"
)

// maxDataBytes caps the size of a stored settings document.
const maxDataBytes = 64 * 1024

// ErrTooLarge is returned when a settings document exceeds maxDataBytes.
var ErrTooLarge = errors.New("settings: document too large")

// ErrInvalidJSON is returned when a settings document is not valid JSON.
var ErrInvalidJSON = errors.New("settings: invalid JSON")

// Service provides read/write access to user settings documents.
type Service struct {
	repo persistence.Repository
}

// NewService builds a settings service backed by the given repository.
func NewService(repo persistence.Repository) *Service {
	return &Service{repo: repo}
}

// Get returns the stored settings document for a user, or "{}" if none exists.
func (s *Service) Get(userID string) (string, error) {
	rec, err := s.repo.GetUserSettings(userID)
	if err != nil {
		return "", err
	}
	if rec == nil || rec.Data == "" {
		return "{}", nil
	}
	return rec.Data, nil
}

// Save validates and stores the settings document for a user.
func (s *Service) Save(userID, data string) error {
	if len(data) > maxDataBytes {
		return ErrTooLarge
	}
	if !json.Valid([]byte(data)) {
		return ErrInvalidJSON
	}
	return s.repo.UpsertUserSettings(userID, data)
}
