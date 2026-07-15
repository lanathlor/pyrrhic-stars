package bosstest_test

import (
	"fmt"
	"testing"

	"codex-online/server/internal/bosstest"
	"codex-online/server/internal/combatlog"
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
		{"blade_blade", []string{"vanguard", "vanguard"}, []string{"blade", "blade"}},
		{"blade_healer", []string{"vanguard", "arcanotechnicien"}, []string{"blade", "harmonist"}},
		{"gunner_healer", []string{"gunner", "arcanotechnicien"}, []string{"assault", "harmonist"}},
		{"gunner_blade", []string{"gunner", "vanguard"}, []string{"assault", "blade"}},
		{"shield_healer", []string{"vanguard", "arcanotechnicien"}, []string{"shield", "harmonist"}},
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
