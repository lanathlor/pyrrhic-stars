package bosstest_test

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"errors"
	"io/fs"

	"codex-online/server/internal/bosstest"
	"codex-online/server/internal/combatlog"
	"codex-online/server/internal/enemyai"
	"codex-online/server/internal/entity"
)

var (
	flagOnly      = flag.String("boss.only", "", "run only these bosses (comma-separated)")
	flagTier      = flag.String("boss.tier", "", "run only this tier: injection, scenario, fuzz")
	flagFuzzIters = flag.Int("boss.fuzz-iterations", 0, "override spec run count for fuzz")

	puppetTrees *bosstest.PuppetTreeRegistry
)

func TestMain(m *testing.M) {
	if err := enemyai.LoadMobs("../../../shared/mobs"); err != nil {
		panic("TestMain: load mobs: " + err.Error())
	}
	if err := enemyai.LoadEncounters("../../../shared/encounters"); err != nil {
		panic("TestMain: load encounters: " + err.Error())
	}

	// Load YAML puppet trees (optional — missing dir is not fatal)
	reg, err := bosstest.LoadPuppetTrees("testdata/puppets")
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		panic("TestMain: load puppet trees: " + err.Error())
	}
	puppetTrees = reg

	m.Run()
}

// shouldRunBoss checks if the given boss should be tested based on -boss.only flag.
func shouldRunBoss(name string) bool {
	if *flagOnly == "" {
		return true
	}
	for _, b := range strings.Split(*flagOnly, ",") {
		if strings.TrimSpace(b) == name {
			return true
		}
	}
	return false
}

// shouldRunTier checks if the given tier should be tested based on -boss.tier flag.
func shouldRunTier(tier string) bool {
	if *flagTier == "" {
		return true
	}
	return strings.TrimSpace(*flagTier) == tier
}

// --- Dynamic Boss Discovery ---

func TestBoss(t *testing.T) {
	specs, err := filepath.Glob("testdata/specs/*.yaml")
	if err != nil {
		t.Fatalf("glob specs: %v", err)
	}
	if len(specs) == 0 {
		t.Fatal("no spec files found in testdata/specs/")
	}

	for _, specPath := range specs {
		base := strings.TrimSuffix(filepath.Base(specPath), ".yaml")
		if !shouldRunBoss(base) {
			continue
		}
		specPath := specPath
		t.Run(base, func(t *testing.T) {
			runBossTests(t, specPath)
		})
	}
}

func runBossTests(t *testing.T, specPath string) {
	spec, err := bosstest.LoadSpec(specPath)
	if err != nil {
		t.Fatalf("load spec: %v", err)
	}

	if shouldRunTier("injection") {
		t.Run("injection", func(t *testing.T) {
			runInjectionTests(t, spec.Boss)
		})
	}
	if shouldRunTier("scenario") {
		t.Run("scenario", func(t *testing.T) {
			runScenarioTests(t, spec.Boss)
		})
	}
	if shouldRunTier("fuzz") {
		t.Run("fuzz", func(t *testing.T) {
			runFuzzTests(t, spec)
		})
	}
}

// --- Tier 1: State Injection ---

func runInjectionTests(t *testing.T, boss string) {
	report := &bosstest.TierReport{Boss: boss, Tier: "injection"}

	t.Run("dead_stops", func(t *testing.T) {
		result := bosstest.NewScenario(boss).
			HP(0).
			Players(bosstest.FakePlayer{ID: 1, Pos: entity.Vec3{Z: 5}, Class: "gunner"}).
			Ticks(1).
			Run()
		result.AssertDead(t)
		report.Add(bosstest.TierCase{
			Name:   "dead_stops",
			Setup:  "HP=0%, 1 tick",
			Assert: "boss is dead",
			Detail: result.Summary(),
			Passed: !t.Failed(),
		})
	})

	t.Run("phase_transition_at_60pct", func(t *testing.T) {
		result := bosstest.NewScenario(boss).
			HP(0.59).
			Players(bosstest.FakePlayer{ID: 1, Pos: entity.Vec3{Z: 5}, Class: "gunner"}).
			Ticks(5).
			Run()
		result.AssertPhase(t, 2)
		report.Add(bosstest.TierCase{
			Name:   "phase_transition_at_60pct",
			Setup:  "HP=59%, 5 ticks",
			Assert: "phase == 2",
			Detail: result.Summary(),
			Passed: !t.Failed(),
		})
	})

	t.Run("phase_transition_at_30pct", func(t *testing.T) {
		result := bosstest.NewScenario(boss).
			HP(0.29).
			Players(bosstest.FakePlayer{ID: 1, Pos: entity.Vec3{Z: 5}, Class: "gunner"}).
			Ticks(5).
			Run()
		result.AssertPhase(t, 3)
		report.Add(bosstest.TierCase{
			Name:   "phase_transition_at_30pct",
			Setup:  "HP=29%, 5 ticks",
			Assert: "phase == 3",
			Detail: result.Summary(),
			Passed: !t.Failed(),
		})
	})

	report.Print(t)
}

// --- Tier 2: Scripted Scenarios ---

func runScenarioTests(t *testing.T, boss string) {
	report := &bosstest.TierReport{Boss: boss, Tier: "scenario"}

	t.Run("chases_when_target_far", func(t *testing.T) {
		result := bosstest.NewScenario(boss).
			Players(bosstest.FakePlayer{ID: 1, Pos: entity.Vec3{Z: 15}, Class: "gunner"}).
			Ticks(3).
			Run()
		result.AssertAlive(t)
		report.Add(bosstest.TierCase{
			Name:   "chases_when_target_far",
			Setup:  "player at Z=15, 3 ticks",
			Assert: "boss alive (chasing, not stuck)",
			Detail: result.Summary(),
			Passed: !t.Failed(),
		})
	})

	t.Run("attacks_in_melee_range", func(t *testing.T) {
		result := bosstest.NewScenario(boss).
			Players(bosstest.FakePlayer{ID: 1, Pos: entity.Vec3{Z: 2}, Class: "vanguard"}).
			Ticks(60).
			Run()
		result.AssertDamageDealt(t)
		report.Add(bosstest.TierCase{
			Name:   "attacks_in_melee_range",
			Setup:  "vanguard at Z=2, 60 ticks",
			Assert: "damage dealt to player",
			Detail: result.Summary(),
			Passed: !t.Failed(),
		})
	})

	t.Run("survives_many_ticks", func(t *testing.T) {
		result := bosstest.NewScenario(boss).
			Players(bosstest.FakePlayer{ID: 1, Pos: entity.Vec3{Z: 5}, Class: "gunner"}).
			Ticks(200).
			Run()
		result.AssertAlive(t)
		report.Add(bosstest.TierCase{
			Name:   "survives_many_ticks",
			Setup:  "player at Z=5, 200 ticks",
			Assert: "boss alive (no self-damage/crash)",
			Detail: result.Summary(),
			Passed: !t.Failed(),
		})
	})

	report.Print(t)
}

// --- Tier 3: Full Fuzz Simulation ---

func runFuzzTests(t *testing.T, spec *bosstest.EncounterSpec) {
	runs := spec.Runs
	if *flagFuzzIters > 0 {
		runs = *flagFuzzIters
	}

	// Use ClickHouse if available, fall back to in-memory
	var sink combatlog.EventSink
	if chSink := bosstest.TryClickHouseSink(); chSink != nil {
		sink = chSink
		defer func() {
			if err := chSink.Close(); err != nil {
				t.Errorf("clickhouse flush: %v", err)
			}
		}()
		t.Log("using ClickHouse sink")
	} else {
		sink = &combatlog.InMemorySink{}
	}

	spec.ExpandVariants()

	var allResults []bosstest.SimResult
	groupID := fmt.Sprintf("fuzz_%s_%d", spec.Boss, os.Getpid())

	for _, comp := range spec.Compositions {
		party := comp.ToPartyConfigs()
		runsPerComp := runs / len(spec.Compositions)
		compGroupID := fmt.Sprintf("%s_%s", groupID, comp.Name)

		for i := range runsPerComp {
			result := bosstest.RunSimulation(bosstest.SimConfig{
				Boss:        spec.Boss,
				Party:       party,
				Seed:        uint64(i),
				Sink:        sink,
				GroupID:     compGroupID,
				RunID:       fmt.Sprintf("%s_%d", compGroupID, i),
				PuppetTrees: puppetTrees,
			})
			result.CompName = comp.Name
			allResults = append(allResults, result)
		}
	}

	fr := &bosstest.FuzzResults{
		Results: allResults,
		Spec:    spec,
	}
	report := fr.AssertAll(t)

	// Tree health: merge reports from all runs
	var merged bosstest.TreeReport
	for _, r := range allResults {
		if r.TreeReport == nil {
			continue
		}
		if len(merged.Nodes) == 0 {
			merged = *r.TreeReport
		} else {
			merged.TotalTicks += r.TreeReport.TotalTicks
			for i := range merged.Nodes {
				if i < len(r.TreeReport.Nodes) {
					merged.Nodes[i].EvalCount += r.TreeReport.Nodes[i].EvalCount
					merged.Nodes[i].SuccessCount += r.TreeReport.Nodes[i].SuccessCount
					merged.Nodes[i].FailCount += r.TreeReport.Nodes[i].FailCount
					merged.Nodes[i].RunningCount += r.TreeReport.Nodes[i].RunningCount
				}
			}
		}
	}
	if len(merged.Nodes) > 0 {
		for i := range merged.Nodes {
			n := &merged.Nodes[i]
			n.Classification = bosstest.ClassifyFromCounts(n.EvalCount, n.SuccessCount, merged.TotalTicks)
		}
		t.Run("tree_health", func(t *testing.T) {
			bosstest.AssertTreeHealth(t, spec.TreeHealth, &merged, report)
		})
	}

	report.PrintReport(t)
}
