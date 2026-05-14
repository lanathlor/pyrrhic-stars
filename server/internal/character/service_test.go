package character

import (
	"errors"
	"testing"

	"codex-online/server/internal/persistence"
)

// stubRepo implements persistence.Repository for testing.
type stubRepo struct {
	chars      map[uint]*persistence.Character
	namesTaken map[string]bool
	charCount  int64
	createErr  error
}

func newStubRepo() *stubRepo {
	return &stubRepo{
		chars:      make(map[uint]*persistence.Character),
		namesTaken: make(map[string]bool),
	}
}

func (r *stubRepo) UpsertUser(string, string) error           { return nil }
func (r *stubRepo) GetUser(string) (*persistence.User, error) { return nil, nil }
func (r *stubRepo) UpdateCharacterPosition(uint, float64, float64, float64, float64) error {
	return nil
}
func (r *stubRepo) GetCharacters(string) ([]*persistence.Character, error) { return nil, nil }

func (r *stubRepo) GetCharacterByID(id uint) (*persistence.Character, error) {
	c, ok := r.chars[id]
	if !ok {
		return nil, nil
	}
	return c, nil
}

func (r *stubRepo) IsCharacterNameTaken(name string) (bool, error) {
	return r.namesTaken[name], nil
}

func (r *stubRepo) CountCharacters(string) (int64, error) {
	return r.charCount, nil
}

func (r *stubRepo) CreateCharacter(c *persistence.Character) error {
	if r.createErr != nil {
		return r.createErr
	}
	c.ID = uint(len(r.chars) + 1)
	r.chars[c.ID] = c
	return nil
}

// Inventory stubs (not used by character service tests).
func (r *stubRepo) CreateItem(*persistence.CharacterItem) error                      { return nil }
func (r *stubRepo) DeleteItem(uint) error                                            { return nil }
func (r *stubRepo) GetItemsByCharacterID(uint) ([]*persistence.CharacterItem, error) { return nil, nil }
func (r *stubRepo) SetEquipment(uint, uint8, uint) error                             { return nil }
func (r *stubRepo) ClearEquipment(uint, uint8) error                                 { return nil }
func (r *stubRepo) GetEquipment(uint) ([]*persistence.CharacterEquipment, error)     { return nil, nil }

func TestSelect(t *testing.T) {
	repo := newStubRepo()
	repo.chars[1] = &persistence.Character{ID: 1, UserID: "player-1", ClassName: "gunner", Name: "Hero"}

	svc := NewService(repo)

	tests := []struct {
		name     string
		charID   uint
		playerID string
		wantErr  error
	}{
		{"valid", 1, "player-1", nil},
		{"wrong owner", 1, "player-2", ErrNotFound},
		{"not found", 99, "player-1", ErrNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			char, err := svc.Select(tt.charID, tt.playerID)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got err %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && char == nil {
				t.Fatal("expected character, got nil")
			}
		})
	}
}

func TestCreate(t *testing.T) {
	tests := []struct {
		name      string
		className string
		charName  string
		count     int64
		taken     bool
		wantErr   error
	}{
		{"valid", "gunner", "MyHero", 0, false, nil},
		{"valid vanguard", "vanguard", "Tank", 0, false, nil},
		{"valid blade_dancer", "blade_dancer", "Dancer-1", 0, false, nil},
		{"invalid class", "mage", "Hero", 0, false, ErrInvalidClass},
		{"name too short", "gunner", "A", 0, false, ErrNameLength},
		{"name too long", "gunner", "ThisNameIsWayTooLongX", 0, false, ErrNameLength},
		{"invalid chars", "gunner", "He@ro!", 0, false, ErrNameChars},
		{"name taken", "gunner", "Taken", 0, true, ErrNameTaken},
		{"limit reached", "gunner", "Hero", 100, false, ErrLimitReached},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newStubRepo()
			repo.charCount = tt.count
			if tt.taken {
				repo.namesTaken[tt.charName] = true
			}

			svc := NewService(repo)
			char, err := svc.Create("player-1", tt.className, tt.charName)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got err %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && char == nil {
				t.Fatal("expected character, got nil")
			}
		})
	}
}
