// Package friend implements the persistent friends subsystem: resolving players
// by account or character name, mutual friend requests, and friend listing.
// It depends only on persistence (no session import) to avoid an import cycle;
// online status is layered on by the gateway.
package friend

import (
	"errors"

	"codex-online/server/internal/persistence"
)

// Name resolution types for incoming requests.
const (
	NameTypeAccount   uint8 = 0
	NameTypeCharacter uint8 = 1
)

var (
	ErrSelf           = errors.New("cannot friend yourself")
	ErrTargetNotFound = errors.New("player not found")
	ErrAlreadyFriends = errors.New("already friends")
	ErrRequestPending = errors.New("friend request already pending")
	ErrAmbiguous      = errors.New("multiple accounts share that name; use character name")
	ErrNoRequest      = errors.New("no pending request from that player")
)

// Service handles friend resolution and the request lifecycle.
type Service struct {
	repo persistence.Repository
}

// NewService creates a friend service.
func NewService(repo persistence.Repository) *Service {
	return &Service{repo: repo}
}

// Entry is a resolved friend or pending request with a display name.
type Entry struct {
	UserID string
	Name   string // best display name (latest character name, else username)
}

// ResolveTarget maps a typed name to a target User.ID.
// nameType 0=account username, 1=character name.
func (s *Service) ResolveTarget(nameType uint8, name string) (string, error) {
	if nameType == NameTypeCharacter {
		char, err := s.repo.GetCharacterByName(name)
		if err != nil {
			return "", err
		}
		if char == nil {
			return "", ErrTargetNotFound
		}
		return char.UserID, nil
	}
	users, err := s.repo.GetUsersByUsername(name)
	if err != nil {
		return "", err
	}
	switch len(users) {
	case 0:
		return "", ErrTargetNotFound
	case 1:
		return users[0].ID, nil
	default:
		return "", ErrAmbiguous
	}
}

// Request creates a pending request from requesterID to the resolved target.
// If a reverse pending request already exists it auto-accepts (mutual cross-request).
// Returns the target User.ID and whether it became an immediate friendship.
func (s *Service) Request(requesterID string, nameType uint8, name string) (string, bool, error) {
	targetID, err := s.ResolveTarget(nameType, name)
	if err != nil {
		return "", false, err
	}
	if targetID == requesterID {
		return "", false, ErrSelf
	}

	existing, err := s.repo.GetFriendship(requesterID, targetID)
	if err != nil {
		return "", false, err
	}
	if existing != nil {
		switch {
		case existing.Status == persistence.FriendStatusAccepted:
			return targetID, false, ErrAlreadyFriends
		case existing.RequesterID == requesterID:
			// We already asked them.
			return targetID, false, ErrRequestPending
		default:
			// They already asked us → accept their request.
			if err := s.repo.AcceptFriendship(existing.RequesterID, existing.AddresseeID); err != nil {
				return "", false, err
			}
			return targetID, true, nil
		}
	}

	if err := s.repo.CreateFriendship(requesterID, targetID); err != nil {
		return "", false, err
	}
	return targetID, false, nil
}

// Respond accepts or declines a pending request from requesterID to addresseeID.
func (s *Service) Respond(addresseeID, requesterID string, accept bool) error { //nolint:revive // accept mirrors the wire-level invite-reply flag, not control coupling
	f, err := s.repo.GetFriendship(requesterID, addresseeID)
	if err != nil {
		return err
	}
	// Must be a pending request addressed to this user.
	if f == nil || f.Status != persistence.FriendStatusPending || f.AddresseeID != addresseeID {
		return ErrNoRequest
	}
	if accept {
		return s.repo.AcceptFriendship(requesterID, addresseeID)
	}
	return s.repo.DeleteFriendship(requesterID, addresseeID)
}

// Remove deletes any friendship/request between the two users.
func (s *Service) Remove(userID, friendID string) error {
	return s.repo.DeleteFriendship(userID, friendID)
}

// List returns accepted friends (UserID + display name) for a user.
func (s *Service) List(userID string) ([]Entry, error) {
	fs, err := s.repo.GetAcceptedFriends(userID)
	if err != nil {
		return nil, err
	}
	entries := make([]Entry, 0, len(fs))
	for _, f := range fs {
		other := f.RequesterID
		if other == userID {
			other = f.AddresseeID
		}
		entries = append(entries, Entry{UserID: other, Name: s.displayName(other)})
	}
	return entries, nil
}

// PendingIncoming returns pending requests addressed to userID.
func (s *Service) PendingIncoming(userID string) ([]Entry, error) {
	fs, err := s.repo.GetPendingIncoming(userID)
	if err != nil {
		return nil, err
	}
	entries := make([]Entry, 0, len(fs))
	for _, f := range fs {
		entries = append(entries, Entry{UserID: f.RequesterID, Name: s.displayName(f.RequesterID)})
	}
	return entries, nil
}

// displayName returns the best human label for a user: their most recently
// updated character name, falling back to the account username, then the UUID.
func (s *Service) displayName(userID string) string {
	if chars, err := s.repo.GetCharacters(userID); err == nil && len(chars) > 0 {
		return chars[0].Name
	}
	if u, err := s.repo.GetUser(userID); err == nil && u != nil && u.Username != "" {
		return u.Username
	}
	return userID
}
