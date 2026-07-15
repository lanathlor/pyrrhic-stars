package bosstest_test

import (
	"testing"

	"codex-online/server/internal/bosstest"
)

// TestSingleTargetDPSOrdering encodes the class-design contract for
// single-target damage: the gunner (assault) is the mono-target specialist
// and must out-damage the blade vanguard on a boss, while the vanguard owns
// multi-target (its cleave identity, deliberately untested here). The blade
// dancer trades raw output for safety and utility (untargeted mobile damage,
// shields), but it is still a DPS spec: it must clearly out-damage the
// shield tank, or nobody queues it.
//
// Measured in the isolated boss sim (single target, no trash) over seeded
// runs with the standard average party. Established 2026-07 when assault
// measured 51 dps/player vs blade's 66; BD floor added when multi_blade
// measured 64% of the tank's damage.
func TestSingleTargetDPSOrdering(t *testing.T) {
	const runs = 40
	const margin = 1.05     // assault must lead by at least 5%
	const bdOverTank = 1.10 // multi_blade must beat the shield tank by 10%

	specDmg := map[string]float64{}
	specPlayers := map[string]int{}
	for run := range runs {
		res := bosstest.RunSimulation(bosstest.SimConfig{
			Boss: "guard_captain",
			Party: []bosstest.PuppetConfig{
				{Class: "arcanotechnicien", Spec: "harmonist", Profile: "average"},
				{Class: "vanguard", Spec: "shield", Profile: "average"},
				{Class: "vanguard", Spec: "blade", Profile: "average"},
				{Class: "gunner", Spec: "assault", Profile: "average"},
				{Class: "blade_dancer", Spec: "multi_blade", Profile: "average"},
			},
			Seed:        uint64(run*7919 + 13),
			PuppetTrees: puppetTrees,
		})
		for spec, d := range res.SpecDamage {
			specDmg[spec] += float64(d)
		}
		for spec, n := range res.SpecPlayers {
			specPlayers[spec] += n
		}
	}
	perPlayer := func(spec string) float64 {
		if specPlayers[spec] == 0 {
			return 0
		}
		return specDmg[spec] / float64(specPlayers[spec])
	}
	assault := perPlayer("assault")
	blade := perPlayer("blade")
	multiBlade := perPlayer("multi_blade")
	shield := perPlayer("shield")
	t.Logf("single-target damage/player: assault %.0f, blade %.0f (ratio %.2f), multi_blade %.0f, shield %.0f, harmonist %.0f",
		assault, blade, assault/blade, multiBlade, shield, perPlayer("harmonist"))
	if assault < blade*margin {
		t.Errorf("assault single-target damage (%.0f/player) must exceed blade vanguard (%.0f/player) by >= %.0f%% — the gunner is the mono-target specialist", assault, blade, (margin-1)*100)
	}
	if assault < multiBlade*margin {
		t.Errorf("assault single-target damage (%.0f/player) must exceed multi_blade (%.0f/player) by >= %.0f%% — the gunner is the mono-target specialist", assault, multiBlade, (margin-1)*100)
	}
	if multiBlade < shield*bdOverTank {
		t.Errorf("multi_blade single-target damage (%.0f/player) must exceed the shield tank (%.0f/player) by >= %.0f%% — the blade dancer is a DPS spec, safety and utility don't excuse tanking the meter", multiBlade, shield, (bdOverTank-1)*100)
	}
}
