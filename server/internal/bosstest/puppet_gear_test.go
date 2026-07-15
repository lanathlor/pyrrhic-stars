package bosstest_test

import (
	"testing"

	"codex-online/server/internal/bosstest"
	"codex-online/server/internal/entity"
	"codex-online/server/internal/item"
)

// Puppets must fight in the same gear live players actually wear: the sims
// previously ran naked puppets, so gear-dependent mechanics (Plating
// mitigation, Hull, Output/Mastery scaling) were invisible to every fuzz
// baseline while shaping live play.
func TestPuppetWearsStarterGear(t *testing.T) {
	pp := bosstest.NewPuppet(1, entity.ClassGunner, "", bosstest.ProfileAverage, 42, "", nil)
	p := pp.Player

	want := item.ComputeStats(item.StarterEquipment(item.StarterILvl))
	if want.Plating <= 0 {
		t.Fatal("starter set must include Plating (items not loaded?)")
	}
	if p.GearStats.Plating != want.Plating {
		t.Errorf("puppet plating = %f, want starter set %f", p.GearStats.Plating, want.Plating)
	}
	if p.GearStats.Hull != want.Hull {
		t.Errorf("puppet hull = %f, want starter set %f", p.GearStats.Hull, want.Hull)
	}

	naked := entity.NewPlayer(2, entity.ClassGunner)
	if p.MaxHealth <= naked.MaxHealth {
		t.Errorf("geared puppet max health %f should exceed naked %f (Hull)", p.MaxHealth, naked.MaxHealth)
	}
	if p.Health != p.MaxHealth {
		t.Errorf("puppet should spawn at full geared health: %f/%f", p.Health, p.MaxHealth)
	}
}

// A composition can raise the party's gear level to test higher-tier balance.
func TestPuppetGearILvlOverride(t *testing.T) {
	pp := bosstest.NewPuppet(1, entity.ClassVanguard, "", bosstest.ProfileAverage, 42, "", nil)
	base := pp.Player.GearStats.Plating

	pp.ApplyGear(50)
	bis := pp.Player.GearStats.Plating
	if bis <= base {
		t.Errorf("ilvl 50 plating %f should exceed ilvl %d plating %f", bis, item.StarterILvl, base)
	}
	want := item.ComputeStats(item.StarterEquipment(50))
	if bis != want.Plating {
		t.Errorf("ilvl 50 plating = %f, want %f", bis, want.Plating)
	}
}
