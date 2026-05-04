package integration

import (
	"testing"

	"codex-online/server/internal/enemyai"
)

func TestMain(m *testing.M) {
	if err := enemyai.LoadMobs("../../../shared/mobs"); err != nil {
		panic("TestMain: load mobs: " + err.Error())
	}
	m.Run()
}
