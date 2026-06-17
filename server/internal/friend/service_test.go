package friend

import (
	"errors"
	"fmt"
	"sync/atomic"
	"testing"

	"codex-online/server/internal/entity"
	"codex-online/server/internal/persistence"
)

// uniq yields a process-unique suffix. The in-memory sqlite uses a shared cache,
// so every test shares one DB; unique names avoid the global UNIQUE(name) clash.
var seq atomic.Uint64

func uniq() uint64 { return seq.Add(1) }

// tu is a test user: its account UUID, account username, and character name.
type tu struct {
	uuid     string
	username string
	charName string
}

// setup builds a service backed by the shared in-memory sqlite repo, plus two
// users (alice, bob) each with one character with a process-unique name.
func setup(t *testing.T) (*Service, persistence.Repository, tu, tu) {
	t.Helper()
	repo, err := persistence.NewGormRepo("sqlite", "")
	if err != nil {
		t.Fatalf("NewGormRepo: %v", err)
	}
	mk := func(prefix, username string) tu {
		n := uniq()
		u := tu{
			uuid:     fmt.Sprintf("uuid-%s-%016d", prefix, n),
			username: username,
			charName: fmt.Sprintf("%sChar%d", username, n),
		}
		if err := repo.UpsertUser(u.uuid, u.username); err != nil {
			t.Fatalf("UpsertUser: %v", err)
		}
		c := &persistence.Character{UserID: u.uuid, ClassName: entity.ClassGunner, Name: u.charName}
		if err := repo.CreateCharacter(c); err != nil {
			t.Fatalf("CreateCharacter: %v", err)
		}
		return u
	}
	return NewService(repo), repo, mk("alice", "Alice"), mk("bobxx", "Bob")
}

func TestRequestSelf(t *testing.T) {
	svc, _, alice, _ := setup(t)
	if _, _, err := svc.Request(alice.uuid, NameTypeCharacter, alice.charName); !errors.Is(err, ErrSelf) {
		t.Errorf("Request(self) err = %v, want ErrSelf", err)
	}
}

func TestRequestUnknown(t *testing.T) {
	svc, _, alice, _ := setup(t)
	if _, _, err := svc.Request(alice.uuid, NameTypeCharacter, "Ghost"); !errors.Is(err, ErrTargetNotFound) {
		t.Errorf("Request(unknown) err = %v, want ErrTargetNotFound", err)
	}
}

func TestRequestAmbiguous(t *testing.T) {
	svc, repo, alice, bob := setup(t)
	// A second account sharing bob's username makes account-name resolution ambiguous.
	if err := repo.UpsertUser(fmt.Sprintf("uuid-dupe-%016d", uniq()), bob.username); err != nil {
		t.Fatalf("UpsertUser: %v", err)
	}
	if _, _, err := svc.Request(alice.uuid, NameTypeAccount, bob.username); !errors.Is(err, ErrAmbiguous) {
		t.Errorf("Request(ambiguous account) err = %v, want ErrAmbiguous", err)
	}
}

func TestRequestThenDuplicate(t *testing.T) {
	svc, _, alice, bob := setup(t)
	if _, auto, err := svc.Request(alice.uuid, NameTypeCharacter, bob.charName); err != nil || auto {
		t.Fatalf("Request = (auto=%v, err=%v), want (false, nil)", auto, err)
	}
	if _, _, err := svc.Request(alice.uuid, NameTypeCharacter, bob.charName); !errors.Is(err, ErrRequestPending) {
		t.Errorf("duplicate Request err = %v, want ErrRequestPending", err)
	}
}

func TestRequestReversePendingAutoAccepts(t *testing.T) {
	svc, _, alice, bob := setup(t)
	if _, _, err := svc.Request(alice.uuid, NameTypeCharacter, bob.charName); err != nil {
		t.Fatalf("alice→bob Request: %v", err)
	}
	// Bob asks Alice while Alice→Bob is pending → auto-accept.
	_, auto, err := svc.Request(bob.uuid, NameTypeCharacter, alice.charName)
	if err != nil || !auto {
		t.Fatalf("bob→alice Request = (auto=%v, err=%v), want (true, nil)", auto, err)
	}
	for _, u := range []tu{alice, bob} {
		friends, err := svc.List(u.uuid)
		if err != nil {
			t.Fatalf("List(%s): %v", u.uuid, err)
		}
		if len(friends) != 1 {
			t.Errorf("%s has %d friends, want 1", u.username, len(friends))
		}
	}
}

func TestRespondAccept(t *testing.T) {
	svc, _, alice, bob := setup(t)
	if _, _, err := svc.Request(alice.uuid, NameTypeCharacter, bob.charName); err != nil {
		t.Fatalf("Request: %v", err)
	}
	pending, err := svc.PendingIncoming(bob.uuid)
	if err != nil || len(pending) != 1 {
		t.Fatalf("PendingIncoming = (%v, %v), want 1 entry", pending, err)
	}
	if pending[0].Name != alice.charName {
		t.Errorf("request display name = %q, want %q", pending[0].Name, alice.charName)
	}
	if err := svc.Respond(bob.uuid, alice.uuid, true); err != nil {
		t.Fatalf("Respond(accept): %v", err)
	}
	if friends, _ := svc.List(bob.uuid); len(friends) != 1 {
		t.Errorf("bob has %d friends after accept, want 1", len(friends))
	}
}

func TestRespondDecline(t *testing.T) {
	svc, _, alice, bob := setup(t)
	if _, _, err := svc.Request(alice.uuid, NameTypeCharacter, bob.charName); err != nil {
		t.Fatalf("Request: %v", err)
	}
	if err := svc.Respond(bob.uuid, alice.uuid, false); err != nil {
		t.Fatalf("Respond(decline): %v", err)
	}
	if pending, _ := svc.PendingIncoming(bob.uuid); len(pending) != 0 {
		t.Error("bob has pending after decline, want 0")
	}
	if friends, _ := svc.List(bob.uuid); len(friends) != 0 {
		t.Error("bob has friends after decline, want 0")
	}
}

func TestRespondNoRequest(t *testing.T) {
	svc, _, alice, bob := setup(t)
	if err := svc.Respond(bob.uuid, alice.uuid, true); !errors.Is(err, ErrNoRequest) {
		t.Errorf("Respond with no request err = %v, want ErrNoRequest", err)
	}
}

func TestRemove(t *testing.T) {
	svc, _, alice, bob := setup(t)
	if _, _, err := svc.Request(alice.uuid, NameTypeCharacter, bob.charName); err != nil {
		t.Fatalf("Request: %v", err)
	}
	if err := svc.Respond(bob.uuid, alice.uuid, true); err != nil {
		t.Fatalf("Respond: %v", err)
	}
	if err := svc.Remove(alice.uuid, bob.uuid); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if friends, _ := svc.List(alice.uuid); len(friends) != 0 {
		t.Error("alice has friends after remove, want 0")
	}
	if friends, _ := svc.List(bob.uuid); len(friends) != 0 {
		t.Error("bob has friends after remove, want 0")
	}
}

func TestDisplayNameFallback(t *testing.T) {
	svc, repo, alice, _ := setup(t)
	// A user with no character falls back to the account username.
	noChar := fmt.Sprintf("uuid-nochr-%016d", uniq())
	if err := repo.UpsertUser(noChar, "NamelessAcct"); err != nil {
		t.Fatalf("UpsertUser: %v", err)
	}
	if err := repo.CreateFriendship(alice.uuid, noChar); err != nil {
		t.Fatalf("CreateFriendship: %v", err)
	}
	if err := repo.AcceptFriendship(alice.uuid, noChar); err != nil {
		t.Fatalf("AcceptFriendship: %v", err)
	}
	friends, err := svc.List(alice.uuid)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(friends) != 1 || friends[0].Name != "NamelessAcct" {
		t.Errorf("friend display = %+v, want name NamelessAcct", friends)
	}
}
