package container

import (
	"testing"

	"codex-online/server/internal/persistence"
)

type fakeRepo struct{ persistence.Repository }

func TestNew(t *testing.T) {
	repo := &fakeRepo{}
	c := New(repo)
	if c == nil {
		t.Fatal("New returned nil")
	}
	if c.Repo != repo {
		t.Error("Repo field not set")
	}
}
