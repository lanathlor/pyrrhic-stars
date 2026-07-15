package bosstest

import (
	"testing"

	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
	"codex-online/server/internal/system"
)

// TestPuppetInstanceUptime guards puppet fidelity on a mobile battlefield:
// in full-instance runs, the chasing DPS puppets must keep a clear line of
// fire to their target for most of their alive time. Without it the instance
// DPS report measures "how badly the puppet AI copes" instead of class
// output (the boss sim is balanced at sigma 0.07; only the instance showed a
// 5x class gap).
//
// Regression context (2026-07), all fixed and guarded here:
//   - gunner: the advance branch fired no shot, blind-fire drained the
//     magazine, and nothing repositioned around blockers → LoS 51%.
//   - blade_dancer: dodge branches committed transitions while strafing out
//     of range → 80% GCD-busy swinging air, LoS 59%.
//   - both: walk-speed advance never caught moving targets; nearest-target
//     flapping (no stickiness) kept melee chasing crossing mobs.
func TestPuppetInstanceUptime(t *testing.T) {
	const runs = 20
	losFloor := map[string]float64{
		"gunner/assault":           0.55, // 0.51 before the fixes; 0.56-0.80 after
		"blade_dancer/multi_blade": 0.61, // 0.59 before the fixes; 0.62-0.86 after
	}

	party := []PuppetConfig{
		{Class: "arcanotechnicien", Spec: "harmonist", Profile: "average"},
		{Class: "vanguard", Spec: "shield", Profile: "average"},
		{Class: "vanguard", Spec: "blade", Profile: "average"},
		{Class: "gunner", Spec: "assault", Profile: "average"},
		{Class: "blade_dancer", Spec: "multi_blade", Profile: "average"},
	}
	type stats struct{ alive, los int }
	agg := map[string]*stats{}
	for _, pc := range party {
		agg[pc.Class+"/"+pc.Spec] = &stats{}
	}
	dmg := map[string]float64{}
	for run := range runs {
		res := RunInstance(InstanceConfig{
			Level: "arena", Party: party, Seed: uint64(run*7919 + 13),
			RespawnDelayTicks: 60,
			debugHook: func(_ int, w *system.World, insts []enemyInst, puppets []*PlayerPuppet) {
				if w.FightStartTick == 0 {
					return
				}
				for _, pp := range puppets {
					if !pp.Player.Alive {
						continue
					}
					st := agg[pp.Player.ClassID+"/"+pp.Player.SpecID]
					st.alive++
					inst := nearestEnemyInst(insts, pp.Player.Position)
					if inst == nil {
						continue
					}
					eye := pp.Player.Position.Add(entity.Vec3{Y: 1.5})
					tgt := inst.enemy.Position.Add(entity.Vec3{Y: 1.0})
					if !combat.SegmentHitsExpandedObstacle(eye, tgt, w.Obstacles, 0) {
						st.los++
					}
				}
			},
		})
		for key, cd := range res.ClassDamage {
			dmg[key] += float64(cd.Boss + cd.Trash)
		}
	}
	for key, st := range agg {
		los := float64(st.los) / float64(st.alive)
		t.Logf("%-30s los %4.1f%% | dmg/alive-s %5.1f", key, los*100, dmg[key]/(float64(st.alive)*0.05))
		if floor, ok := losFloor[key]; ok && los < floor {
			t.Errorf("%s line-of-fire uptime = %.0f%%, want >= %.0f%% — its puppet is not repositioning/closing on instance targets", key, los*100, floor*100)
		}
	}
}
