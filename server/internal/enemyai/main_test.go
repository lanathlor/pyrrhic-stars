package enemyai

import "testing"

func TestMain(m *testing.M) {
	if err := LoadMobs("../../../shared/mobs"); err != nil {
		panic("TestMain: load mobs: " + err.Error())
	}
	m.Run()
}
