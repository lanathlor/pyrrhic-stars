package user

import (
	"fmt"
	"strings"

	"codex-online/server/internal/persistence"
)

// Service handles user-level operations (account, username).
type Service struct {
	repo persistence.Repository
}

// NewService creates a user service.
func NewService(repo persistence.Repository) *Service {
	return &Service{repo: repo}
}

// CleanUsername sanitizes a raw username string. Returns a fallback if empty.
func (s *Service) CleanUsername(raw string, fallbackID uint32) string {
	name := strings.TrimSpace(raw)
	if name == "" {
		name = fmt.Sprintf("Player_%d", fallbackID)
	}
	if len(name) > 20 {
		name = name[:20]
	}
	return name
}
