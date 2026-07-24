package bosstest_test

import (
	"fmt"
	"testing"

	"codex-online/server/internal/bosstest"
	"codex-online/server/internal/combatlog"
	"codex-online/server/internal/entity"
)

// TestDuoCompExploration is a diagnostic probe, not an assertion: it measures
// how different duo compositions fare against the aceras_general to tell a
// comp problem apart from an encounter problem. Run with:
//
//	go test ./internal/bosstest/ -run TestDuoCompExploration -v -boss.tier=probe
func TestDuoCompExploration(t *testing.T) {
	if *flagTier != "probe" {
		t.Skip("probe only (-boss.tier=probe)")
	}
	const runs = 150
	comps := []struct {
		name    string
		classes []string
		specs   []string
	}{
		{"blade_blade", []string{entity.ClassVanguard, entity.ClassVanguard}, []string{entity.SpecBlade, entity.SpecBlade}},
		{"blade_healer", []string{entity.ClassVanguard, entity.ClassArcanotechnicien}, []string{entity.SpecBlade, entity.SpecHarmonist}},
		{"gunner_healer", []string{entity.ClassGunner, entity.ClassArcanotechnicien}, []string{entity.SpecAssault, entity.SpecHarmonist}},
		{"gunner_blade", []string{entity.ClassGunner, entity.ClassVanguard}, []string{entity.SpecAssault, entity.SpecBlade}},
		{"shield_healer", []string{entity.ClassVanguard, entity.ClassArcanotechnicien}, []string{entity.SpecShield, entity.SpecHarmonist}},
	}
	for _, comp := range comps {
		wins := 0
		var totalSecs float64
		for i := range runs {
			party := make([]bosstest.PuppetConfig, len(comp.classes))
			for j := range comp.classes {
				party[j] = bosstest.PuppetConfig{
					Class:   comp.classes[j],
					Spec:    comp.specs[j],
					Profile: bosstest.ProfileAverage,
				}
			}
			res := bosstest.RunSimulation(bosstest.SimConfig{
				Boss:        "aceras_general",
				Party:       party,
				Seed:        uint64(1000 + i*17),
				PuppetTrees: puppetTrees,
			})
			if res.Outcome == combatlog.OutcomePlayerWin {
				wins++
			}
			totalSecs += res.Duration.Seconds()
		}
		fmt.Printf("PROBE %-14s win %5.1f%%  avg %5.1fs\n",
			comp.name, 100*float64(wins)/runs, totalSecs/runs)
	}
}
