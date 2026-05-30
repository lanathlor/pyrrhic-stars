package system

import (
	"testing"

	"codex-online/server/internal/enemyai"
	"codex-online/server/internal/level"
)

func TestMain(m *testing.M) {
	if err := enemyai.LoadMobs("../../../shared/mobs"); err != nil {
		panic("TestMain: load mobs: " + err.Error())
	}
	if err := enemyai.LoadEncounters("../../../shared/encounters"); err != nil {
		panic("TestMain: load encounters: " + err.Error())
	}
	m.Run()
}

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
