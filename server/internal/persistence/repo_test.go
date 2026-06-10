package persistence

import (
	"strings"
	"testing"
	"time"

	"codex-online/server/internal/entity"
)

func newTestRepo(t *testing.T) Repository {
	t.Helper()
	repo, err := NewGormRepo("sqlite", "")
	if err != nil {
		t.Fatalf("NewGormRepo: %v", err)
	}
	return repo
}

func TestUpsertUser(t *testing.T) {
	repo := newTestRepo(t)
	id := "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee"

	// First call creates the user.
	if err := repo.UpsertUser(id, "Alice"); err != nil {
		t.Fatalf("UpsertUser (create): %v", err)
	}
	p, err := repo.GetUser(id)
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if p == nil {
		t.Fatal("expected user, got nil")
	}
	if p.Username != "Alice" {
		t.Errorf("username = %q, want %q", p.Username, "Alice")
	}

	// Second call with a different username must NOT overwrite.
	if err := repo.UpsertUser(id, "AliceRenamed"); err != nil {
		t.Fatalf("UpsertUser (noop): %v", err)
	}
	p, err = repo.GetUser(id)
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if p.Username != "Alice" {
		t.Errorf("username = %q after re-upsert, want %q (should not overwrite)", p.Username, "Alice")
	}
}

func TestGetUserNotFound(t *testing.T) {
	repo := newTestRepo(t)
	p, err := repo.GetUser("nonexistent-uuid")
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if p != nil {
		t.Errorf("expected nil, got %+v", p)
	}
}

func TestCreateCharacter(t *testing.T) {
	repo := newTestRepo(t)
	userID := "11111111-2222-4333-8444-555555555555"
	if err := repo.UpsertUser(userID, "Bob"); err != nil {
		t.Fatalf("UpsertUser: %v", err)
	}

	c := &Character{UserID: userID, ClassName: entity.ClassGunner, Name: "BobGunner", PosX: 1.0, PosY: 2.0, PosZ: 3.0, RotY: 0.5}
	if err := repo.CreateCharacter(c); err != nil {
		t.Fatalf("CreateCharacter: %v", err)
	}
	if c.ID == 0 {
		t.Fatal("expected non-zero ID after creation")
	}

	got, err := repo.GetCharacterByID(c.ID)
	if err != nil {
		t.Fatalf("GetCharacterByID: %v", err)
	}
	if got == nil {
		t.Fatal("expected character, got nil")
	}
	if got.Name != "BobGunner" {
		t.Errorf("Name = %q, want %q", got.Name, "BobGunner")
	}
	if got.PosX != 1.0 {
		t.Errorf("PosX = %f, want 1.0", got.PosX)
	}

	// Duplicate name must fail.
	dup := &Character{UserID: userID, ClassName: entity.ClassVanguard, Name: "BobGunner", PosX: 5.0}
	if err := repo.CreateCharacter(dup); err == nil {
		t.Fatal("expected error for duplicate name, got nil")
	}
}

func TestCreateCharacter_MultiplePerClass(t *testing.T) {
	repo := newTestRepo(t)
	userID := "22222222-3333-4444-8555-666666666666"
	if err := repo.UpsertUser(userID, "Multi"); err != nil {
		t.Fatalf("UpsertUser: %v", err)
	}

	names := []string{"Gunner1", "Gunner2", "Gunner3"}
	for _, name := range names {
		c := &Character{UserID: userID, ClassName: entity.ClassGunner, Name: name}
		if err := repo.CreateCharacter(c); err != nil {
			t.Fatalf("CreateCharacter(%s): %v", name, err)
		}
	}

	count, err := repo.CountCharacters(userID)
	if err != nil {
		t.Fatalf("CountCharacters: %v", err)
	}
	if count != 3 {
		t.Errorf("count = %d, want 3", count)
	}
}

func TestUpdateCharacterPosition(t *testing.T) {
	repo := newTestRepo(t)
	userID := "33333333-4444-4555-8666-777777777777"
	if err := repo.UpsertUser(userID, "Mover"); err != nil {
		t.Fatalf("UpsertUser: %v", err)
	}

	c := &Character{UserID: userID, ClassName: entity.ClassVanguard, Name: "MoverChar", PosX: 0, PosY: 0, PosZ: 0, RotY: 0}
	if err := repo.CreateCharacter(c); err != nil {
		t.Fatalf("CreateCharacter: %v", err)
	}

	if err := repo.UpdateCharacterPosition(c.ID, 10.5, 20.5, 30.5, 1.5); err != nil {
		t.Fatalf("UpdateCharacterPosition: %v", err)
	}

	got, err := repo.GetCharacterByID(c.ID)
	if err != nil {
		t.Fatalf("GetCharacterByID: %v", err)
	}
	if got.PosX != 10.5 || got.PosY != 20.5 || got.PosZ != 30.5 || got.RotY != 1.5 {
		t.Errorf("position = (%f,%f,%f,%f), want (10.5,20.5,30.5,1.5)", got.PosX, got.PosY, got.PosZ, got.RotY)
	}
}

func TestGetCharacterByID(t *testing.T) {
	repo := newTestRepo(t)
	userID := "44444444-5555-4666-8777-888888888888"
	if err := repo.UpsertUser(userID, "Finder"); err != nil {
		t.Fatalf("UpsertUser: %v", err)
	}

	c := &Character{UserID: userID, ClassName: "arcanotechnicien", Name: "FindMe"}
	if err := repo.CreateCharacter(c); err != nil {
		t.Fatalf("CreateCharacter: %v", err)
	}

	got, err := repo.GetCharacterByID(c.ID)
	if err != nil {
		t.Fatalf("GetCharacterByID: %v", err)
	}
	if got == nil || got.Name != "FindMe" {
		t.Errorf("got %+v, want character with Name=FindMe", got)
	}

	// Non-existent ID returns nil.
	got, err = repo.GetCharacterByID(99999)
	if err != nil {
		t.Fatalf("GetCharacterByID(99999): %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for non-existent ID, got %+v", got)
	}
}

func TestGetCharacters(t *testing.T) {
	repo := newTestRepo(t)
	userID := "55555555-6666-4777-8888-999999999999"
	if err := repo.UpsertUser(userID, "Lister"); err != nil {
		t.Fatalf("UpsertUser: %v", err)
	}

	names := []string{"First", "Second", "Third"}
	for _, name := range names {
		c := &Character{UserID: userID, ClassName: entity.ClassGunner, Name: name}
		if err := repo.CreateCharacter(c); err != nil {
			t.Fatalf("CreateCharacter(%s): %v", name, err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	chars, err := repo.GetCharacters(userID)
	if err != nil {
		t.Fatalf("GetCharacters: %v", err)
	}
	if len(chars) != 3 {
		t.Fatalf("got %d characters, want 3", len(chars))
	}
	// Most recently created should be first (updated_at DESC).
	if chars[0].Name != "Third" {
		t.Errorf("first character Name = %q, want %q", chars[0].Name, "Third")
	}
}

func TestIsCharacterNameTaken(t *testing.T) {
	repo := newTestRepo(t)
	userID := "66666666-7777-4888-8999-aaaaaaaaaaaa"
	if err := repo.UpsertUser(userID, "Namer"); err != nil {
		t.Fatalf("UpsertUser: %v", err)
	}

	taken, err := repo.IsCharacterNameTaken("UniqueName")
	if err != nil {
		t.Fatalf("IsCharacterNameTaken: %v", err)
	}
	if taken {
		t.Error("expected false for unused name, got true")
	}

	c := &Character{UserID: userID, ClassName: entity.ClassGunner, Name: "UniqueName"}
	if err := repo.CreateCharacter(c); err != nil {
		t.Fatalf("CreateCharacter: %v", err)
	}

	taken, err = repo.IsCharacterNameTaken("UniqueName")
	if err != nil {
		t.Fatalf("IsCharacterNameTaken: %v", err)
	}
	if !taken {
		t.Error("expected true after creation, got false")
	}
}

func TestCountCharacters(t *testing.T) {
	repo := newTestRepo(t)
	userID := "77777777-8888-4999-8aaa-bbbbbbbbbbbb"
	if err := repo.UpsertUser(userID, "Counter"); err != nil {
		t.Fatalf("UpsertUser: %v", err)
	}

	count, err := repo.CountCharacters(userID)
	if err != nil {
		t.Fatalf("CountCharacters: %v", err)
	}
	if count != 0 {
		t.Errorf("count = %d for new user, want 0", count)
	}

	for i, name := range []string{"Char1", "Char2"} {
		c := &Character{UserID: userID, ClassName: entity.ClassVanguard, Name: name}
		if err := repo.CreateCharacter(c); err != nil {
			t.Fatalf("CreateCharacter #%d: %v", i, err)
		}
	}

	count, err = repo.CountCharacters(userID)
	if err != nil {
		t.Fatalf("CountCharacters: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d after 2 creations, want 2", count)
	}
}

// createTestCharacter is a helper that creates a user and character, returning the character ID.
func createTestCharacter(t *testing.T, repo Repository, userID, username, charName string) uint {
	t.Helper()
	if err := repo.UpsertUser(userID, username); err != nil {
		t.Fatalf("UpsertUser: %v", err)
	}
	c := &Character{UserID: userID, ClassName: "arcanotechnicien", Name: charName}
	if err := repo.CreateCharacter(c); err != nil {
		t.Fatalf("CreateCharacter: %v", err)
	}
	return c.ID
}

func TestUpsertLoadoutCreate(t *testing.T) {
	repo := newTestRepo(t)
	charID := createTestCharacter(t, repo, "88888888-9999-4aaa-8bbb-cccccccccccc", "Caster", "ArcaneOne")

	slots := [6]string{"fireball", "ice_lance", "shield", "blink", "arcane_blast", "meteor"}
	if err := repo.UpsertLoadout(charID, slots); err != nil {
		t.Fatalf("UpsertLoadout (create): %v", err)
	}

	got, err := repo.GetLoadout(charID)
	if err != nil {
		t.Fatalf("GetLoadout: %v", err)
	}
	if got == nil {
		t.Fatal("expected loadout, got nil")
	}
	gotSlots := [6]string{got.Slot0, got.Slot1, got.Slot2, got.Slot3, got.Slot4, got.Slot5}
	if gotSlots != slots {
		t.Errorf("slots = %v, want %v", gotSlots, slots)
	}
	if got.CharacterID != charID {
		t.Errorf("CharacterID = %d, want %d", got.CharacterID, charID)
	}
}

func TestUpsertLoadoutUpdate(t *testing.T) {
	repo := newTestRepo(t)
	charID := createTestCharacter(t, repo, "99999999-aaaa-4bbb-8ccc-dddddddddddd", "Updater", "ArcaneTwo")

	initial := [6]string{"fireball", "ice_lance", "shield", "blink", "arcane_blast", "meteor"}
	if err := repo.UpsertLoadout(charID, initial); err != nil {
		t.Fatalf("UpsertLoadout (create): %v", err)
	}

	updated := [6]string{"chain_lightning", "ice_lance", "barrier", "teleport", "arcane_blast", "comet"}
	if err := repo.UpsertLoadout(charID, updated); err != nil {
		t.Fatalf("UpsertLoadout (update): %v", err)
	}

	got, err := repo.GetLoadout(charID)
	if err != nil {
		t.Fatalf("GetLoadout: %v", err)
	}
	if got == nil {
		t.Fatal("expected loadout after update, got nil")
	}
	gotSlots := [6]string{got.Slot0, got.Slot1, got.Slot2, got.Slot3, got.Slot4, got.Slot5}
	if gotSlots != updated {
		t.Errorf("slots after update = %v, want %v", gotSlots, updated)
	}
}

func TestGetLoadoutNotFound(t *testing.T) {
	repo := newTestRepo(t)

	got, err := repo.GetLoadout(99999)
	if err != nil {
		t.Fatalf("GetLoadout: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for non-existent character, got %+v", got)
	}
}

func TestUpsertLoadoutEmptySlots(t *testing.T) {
	repo := newTestRepo(t)
	charID := createTestCharacter(t, repo, "aaaaaaaa-bbbb-4ccc-8ddd-ffffffffffff", "Blank", "ArcaneEmpty")

	empty := [6]string{"", "", "", "", "", ""}
	if err := repo.UpsertLoadout(charID, empty); err != nil {
		t.Fatalf("UpsertLoadout (empty): %v", err)
	}

	got, err := repo.GetLoadout(charID)
	if err != nil {
		t.Fatalf("GetLoadout: %v", err)
	}
	if got == nil {
		t.Fatal("expected loadout with empty slots, got nil")
	}
	gotSlots := [6]string{got.Slot0, got.Slot1, got.Slot2, got.Slot3, got.Slot4, got.Slot5}
	if gotSlots != empty {
		t.Errorf("slots = %v, want all empty strings", gotSlots)
	}
}

// ---------------------------------------------------------------------------
// GetScrip
// ---------------------------------------------------------------------------

func TestGetScripMissing(t *testing.T) {
	repo := newTestRepo(t)
	charID := createTestCharacter(t, repo, "aaaa0001-0000-4000-8000-000000000001", "ScripUser1", "ScripChar1")

	balance, err := repo.GetScrip(charID, 1)
	if err != nil {
		t.Fatalf("GetScrip: %v", err)
	}
	if balance != 0 {
		t.Errorf("balance = %d for missing row, want 0", balance)
	}
}

func TestGetScripAfterAdd(t *testing.T) {
	repo := newTestRepo(t)
	charID := createTestCharacter(t, repo, "aaaa0001-0000-4000-8000-000000000002", "ScripUser2", "ScripChar2")

	if err := repo.AddScrip(charID, 1, 50); err != nil {
		t.Fatalf("AddScrip: %v", err)
	}

	balance, err := repo.GetScrip(charID, 1)
	if err != nil {
		t.Fatalf("GetScrip: %v", err)
	}
	if balance != 50 {
		t.Errorf("balance = %d, want 50", balance)
	}
}

// ---------------------------------------------------------------------------
// AddScrip
// ---------------------------------------------------------------------------

func TestAddScripCreatesRow(t *testing.T) {
	repo := newTestRepo(t)
	charID := createTestCharacter(t, repo, "aaaa0002-0000-4000-8000-000000000001", "ScripUser3", "ScripChar3")

	if err := repo.AddScrip(charID, 2, 100); err != nil {
		t.Fatalf("AddScrip (first): %v", err)
	}

	balance, err := repo.GetScrip(charID, 2)
	if err != nil {
		t.Fatalf("GetScrip: %v", err)
	}
	if balance != 100 {
		t.Errorf("balance = %d after first AddScrip, want 100", balance)
	}
}

func TestAddScripAccumulates(t *testing.T) {
	repo := newTestRepo(t)
	charID := createTestCharacter(t, repo, "aaaa0002-0000-4000-8000-000000000002", "ScripUser4", "ScripChar4")

	if err := repo.AddScrip(charID, 1, 40); err != nil {
		t.Fatalf("AddScrip (first): %v", err)
	}
	if err := repo.AddScrip(charID, 1, 60); err != nil {
		t.Fatalf("AddScrip (second): %v", err)
	}

	balance, err := repo.GetScrip(charID, 1)
	if err != nil {
		t.Fatalf("GetScrip: %v", err)
	}
	if balance != 100 {
		t.Errorf("balance = %d after two adds, want 100", balance)
	}
}

func TestAddScripSeasonsAreIndependent(t *testing.T) {
	repo := newTestRepo(t)
	charID := createTestCharacter(t, repo, "aaaa0002-0000-4000-8000-000000000003", "ScripUser5", "ScripChar5")

	if err := repo.AddScrip(charID, 1, 200); err != nil {
		t.Fatalf("AddScrip season 1: %v", err)
	}
	if err := repo.AddScrip(charID, 2, 300); err != nil {
		t.Fatalf("AddScrip season 2: %v", err)
	}

	bal1, err := repo.GetScrip(charID, 1)
	if err != nil {
		t.Fatalf("GetScrip season 1: %v", err)
	}
	bal2, err := repo.GetScrip(charID, 2)
	if err != nil {
		t.Fatalf("GetScrip season 2: %v", err)
	}
	if bal1 != 200 {
		t.Errorf("season 1 balance = %d, want 200", bal1)
	}
	if bal2 != 300 {
		t.Errorf("season 2 balance = %d, want 300", bal2)
	}
}

// ---------------------------------------------------------------------------
// DeductScrip
// ---------------------------------------------------------------------------

func TestDeductScripExactBalance(t *testing.T) {
	repo := newTestRepo(t)
	charID := createTestCharacter(t, repo, "aaaa0003-0000-4000-8000-000000000001", "ScripUser6", "ScripChar6")

	if err := repo.AddScrip(charID, 1, 75); err != nil {
		t.Fatalf("AddScrip: %v", err)
	}
	if err := repo.DeductScrip(charID, 1, 75); err != nil {
		t.Fatalf("DeductScrip (exact): %v", err)
	}

	balance, err := repo.GetScrip(charID, 1)
	if err != nil {
		t.Fatalf("GetScrip: %v", err)
	}
	if balance != 0 {
		t.Errorf("balance = %d after full deduction, want 0", balance)
	}
}

func TestDeductScripInsufficientBalance(t *testing.T) {
	repo := newTestRepo(t)
	charID := createTestCharacter(t, repo, "aaaa0003-0000-4000-8000-000000000002", "ScripUser7", "ScripChar7")

	if err := repo.AddScrip(charID, 1, 10); err != nil {
		t.Fatalf("AddScrip: %v", err)
	}

	err := repo.DeductScrip(charID, 1, 20)
	if err == nil {
		t.Fatal("expected error deducting more than balance, got nil")
	}
	if !strings.Contains(err.Error(), "insufficient") {
		t.Errorf("error = %q, want it to contain \"insufficient\"", err.Error())
	}
}

func TestDeductScripMissingRow(t *testing.T) {
	repo := newTestRepo(t)
	charID := createTestCharacter(t, repo, "aaaa0003-0000-4000-8000-000000000003", "ScripUser8", "ScripChar8")

	err := repo.DeductScrip(charID, 1, 50)
	if err == nil {
		t.Fatal("expected error deducting from missing row, got nil")
	}
}

// ---------------------------------------------------------------------------
// GetWatermark
// ---------------------------------------------------------------------------

func TestGetWatermarkMissing(t *testing.T) {
	repo := newTestRepo(t)
	charID := createTestCharacter(t, repo, "aaaa0004-0000-4000-8000-000000000001", "WMUser1", "WMChar1")

	score, err := repo.GetWatermark(charID, 1)
	if err != nil {
		t.Fatalf("GetWatermark: %v", err)
	}
	if score != 0 {
		t.Errorf("score = %d for missing row, want 0", score)
	}
}

func TestGetWatermarkAfterUpdate(t *testing.T) {
	repo := newTestRepo(t)
	charID := createTestCharacter(t, repo, "aaaa0004-0000-4000-8000-000000000002", "WMUser2", "WMChar2")

	if err := repo.UpdateWatermark(charID, 1, 500); err != nil {
		t.Fatalf("UpdateWatermark: %v", err)
	}

	score, err := repo.GetWatermark(charID, 1)
	if err != nil {
		t.Fatalf("GetWatermark: %v", err)
	}
	if score != 500 {
		t.Errorf("score = %d, want 500", score)
	}
}

// ---------------------------------------------------------------------------
// UpdateWatermark
// ---------------------------------------------------------------------------

func TestUpdateWatermarkCreatesRow(t *testing.T) {
	repo := newTestRepo(t)
	charID := createTestCharacter(t, repo, "aaaa0005-0000-4000-8000-000000000001", "WMUser3", "WMChar3")

	if err := repo.UpdateWatermark(charID, 1, 100); err != nil {
		t.Fatalf("UpdateWatermark (first): %v", err)
	}

	score, err := repo.GetWatermark(charID, 1)
	if err != nil {
		t.Fatalf("GetWatermark: %v", err)
	}
	if score != 100 {
		t.Errorf("score = %d after first UpdateWatermark, want 100", score)
	}
}

func TestUpdateWatermarkHigherScoreUpdates(t *testing.T) {
	repo := newTestRepo(t)
	charID := createTestCharacter(t, repo, "aaaa0005-0000-4000-8000-000000000002", "WMUser4", "WMChar4")

	if err := repo.UpdateWatermark(charID, 1, 100); err != nil {
		t.Fatalf("UpdateWatermark (initial): %v", err)
	}
	if err := repo.UpdateWatermark(charID, 1, 250); err != nil {
		t.Fatalf("UpdateWatermark (higher): %v", err)
	}

	score, err := repo.GetWatermark(charID, 1)
	if err != nil {
		t.Fatalf("GetWatermark: %v", err)
	}
	if score != 250 {
		t.Errorf("score = %d after higher update, want 250", score)
	}
}

func TestUpdateWatermarkLowerScoreIgnored(t *testing.T) {
	repo := newTestRepo(t)
	charID := createTestCharacter(t, repo, "aaaa0005-0000-4000-8000-000000000003", "WMUser5", "WMChar5")

	if err := repo.UpdateWatermark(charID, 1, 300); err != nil {
		t.Fatalf("UpdateWatermark (initial): %v", err)
	}
	if err := repo.UpdateWatermark(charID, 1, 150); err != nil {
		t.Fatalf("UpdateWatermark (lower): %v", err)
	}

	score, err := repo.GetWatermark(charID, 1)
	if err != nil {
		t.Fatalf("GetWatermark: %v", err)
	}
	if score != 300 {
		t.Errorf("score = %d after lower update, want 300 (watermark must not decrease)", score)
	}
}

func TestUpdateWatermarkSeasonsAreIndependent(t *testing.T) {
	repo := newTestRepo(t)
	charID := createTestCharacter(t, repo, "aaaa0005-0000-4000-8000-000000000004", "WMUser6", "WMChar6")

	if err := repo.UpdateWatermark(charID, 1, 400); err != nil {
		t.Fatalf("UpdateWatermark season 1: %v", err)
	}
	if err := repo.UpdateWatermark(charID, 2, 600); err != nil {
		t.Fatalf("UpdateWatermark season 2: %v", err)
	}

	sc1, err := repo.GetWatermark(charID, 1)
	if err != nil {
		t.Fatalf("GetWatermark season 1: %v", err)
	}
	sc2, err := repo.GetWatermark(charID, 2)
	if err != nil {
		t.Fatalf("GetWatermark season 2: %v", err)
	}
	if sc1 != 400 {
		t.Errorf("season 1 watermark = %d, want 400", sc1)
	}
	if sc2 != 600 {
		t.Errorf("season 2 watermark = %d, want 600", sc2)
	}
}
