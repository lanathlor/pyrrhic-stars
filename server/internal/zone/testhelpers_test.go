package zone

import (
	"testing"

	"codex-online/server/internal/level"
)

// Test constants shared across zone test files.
const (
	testPlayerName = "TestPlayer"
)

func testArenaLevel(t testing.TB) *level.Level {
	t.Helper()
	l, err := level.Load("arena")
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func testHubLevel(t testing.TB) *level.Level {
	t.Helper()
	l, err := level.Load("hub")
	if err != nil {
		t.Fatal(err)
	}
	return l
}
