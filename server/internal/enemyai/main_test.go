package enemyai

import "testing"

func TestMain(m *testing.M) {
	if err := LoadMobs("../../../shared/mobs"); err != nil {
		panic("TestMain: load mobs: " + err.Error())
	}
	if err := LoadEncounters("../../../shared/encounters"); err != nil {
		panic("TestMain: load encounters: " + err.Error())
	}
	m.Run()
}
