package system

import (
	"testing"

	"codex-online/server/internal/enemyai"
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
