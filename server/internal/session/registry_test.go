package session

import (
	"testing"

	"codex-online/server/internal/network"
)

// newSession registers a session with the given identity fields populated.
func newSession(t *testing.T, r *Registry, uuid, username, charName string) *Session {
	t.Helper()
	client, _ := network.NewTestClient()
	sess := r.Register(client)
	sess.UserUUID = uuid
	sess.Username = username
	sess.CharName = charName
	return sess
}

func TestFindOnlineByUsername(t *testing.T) {
	r := NewRegistry()
	want := newSession(t, r, "uuid-a", "Alice", "AliceTheBrave")
	newSession(t, r, "uuid-b", "Bob", "BobTheBold")

	// Case-insensitive match.
	got := r.FindOnlineByUsername("alice")
	if got == nil || got.ID != want.ID {
		t.Errorf("FindOnlineByUsername(alice) = %v, want session %d", got, want.ID)
	}

	if r.FindOnlineByUsername("Nobody") != nil {
		t.Error("expected nil for unknown username")
	}
}

func TestFindOnlineByCharName(t *testing.T) {
	r := NewRegistry()
	newSession(t, r, "uuid-a", "Alice", "AliceTheBrave")
	want := newSession(t, r, "uuid-b", "Bob", "BobTheBold")

	got := r.FindOnlineByCharName("bobthebold")
	if got == nil || got.ID != want.ID {
		t.Errorf("FindOnlineByCharName = %v, want session %d", got, want.ID)
	}

	if r.FindOnlineByCharName("Ghost") != nil {
		t.Error("expected nil for unknown character name")
	}
}

func TestFindOnlineByUserUUIDAndIsUserOnline(t *testing.T) {
	r := NewRegistry()
	want := newSession(t, r, "uuid-a", "Alice", "AliceTheBrave")

	got := r.FindOnlineByUserUUID("uuid-a")
	if got == nil || got.ID != want.ID {
		t.Errorf("FindOnlineByUserUUID = %v, want session %d", got, want.ID)
	}
	if !r.IsUserOnline("uuid-a") {
		t.Error("IsUserOnline(uuid-a) = false, want true")
	}
	if r.IsUserOnline("uuid-offline") {
		t.Error("IsUserOnline(uuid-offline) = true, want false")
	}

	// After removal, the user is offline.
	r.Remove(want.Conn)
	if r.IsUserOnline("uuid-a") {
		t.Error("IsUserOnline after Remove = true, want false")
	}
}
