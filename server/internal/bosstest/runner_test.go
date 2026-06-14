package bosstest_test

import (
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"errors"
	"io/fs"

	"codex-online/server/internal/bosstest"
	"codex-online/server/internal/combatlog"
	"codex-online/server/internal/enemyai"
	"codex-online/server/internal/entity"
	"codex-online/server/internal/overflux"
)

var (
	flagOnly      = flag.String("boss.only", "", "run only these bosses (comma-separated)")
	flagTier      = flag.String("boss.tier", "", "run only this tier: injection, scenario, fuzz")
	flagFuzzIters = flag.Int("boss.fuzz-iterations", 0, "override spec run count for fuzz")
	flagOverflux  = flag.String("boss.overflux", "", "run only this overflux config name (empty = all including baseline)")

	puppetTrees *bosstest.PuppetTreeRegistry
)

func TestMain(m *testing.M) {
	// Silence per-spawn INFO logs (e.g. "applying overflux variants"): the fuzz
	// harness spawns thousands of enemies, so production-level logging is spam here.
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))

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
	for b := range strings.SplitSeq(*flagOnly, ",") {
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

	// Injection and scenario tiers are boss-phase specific (single enemy);
	// trash-pack encounters only run the fuzz tier.
	if !spec.IsPack() {
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
			Players(bosstest.FakePlayer{ID: 1, Pos: entity.Vec3{Z: 5}, Class: entity.ClassGunner}).
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
			Players(bosstest.FakePlayer{ID: 1, Pos: entity.Vec3{Z: 5}, Class: entity.ClassGunner}).
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
			Players(bosstest.FakePlayer{ID: 1, Pos: entity.Vec3{Z: 5}, Class: entity.ClassGunner}).
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
			Players(bosstest.FakePlayer{ID: 1, Pos: entity.Vec3{Z: 15}, Class: entity.ClassGunner}).
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
			Players(bosstest.FakePlayer{ID: 1, Pos: entity.Vec3{Z: 2}, Class: entity.ClassVanguard}).
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
			Players(bosstest.FakePlayer{ID: 1, Pos: entity.Vec3{Z: 5}, Class: entity.ClassGunner}).
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
	// Build overflux configs: baseline (nil) + any defined in the spec.
	type oflxEntry struct {
		name string
		spec *bosstest.OverfluxSpec // nil = baseline
	}
	entries := []oflxEntry{{name: "baseline", spec: nil}}
	for i := range spec.Overflux {
		entries = append(entries, oflxEntry{name: spec.Overflux[i].Name, spec: &spec.Overflux[i]})
	}

	// Filter by -boss.overflux flag if set.
	if *flagOverflux != "" {
		var filtered []oflxEntry
		for _, e := range entries {
			if e.name == *flagOverflux {
				filtered = append(filtered, e)
			}
		}
		if len(filtered) == 0 {
			t.Skipf("no overflux config matching %q", *flagOverflux)
		}
		entries = filtered
	}

	// Collect all results across batches for the summary matrix.
	matrix := &bosstest.SummaryMatrix{Boss: spec.Boss}

	for _, entry := range entries {
		t.Run(entry.name, func(t *testing.T) {
			results := runFuzzBatch(t, spec, entry.name, entry.spec)
			matrix.Results = append(matrix.Results, results...)
		})
	}

	// Print the cross-cutting win rate matrix after all batches.
	if len(matrix.Results) > 0 && len(entries) > 1 {
		matrix.PrintMatrix(t)
	}
}

func runFuzzBatch(t *testing.T, spec *bosstest.EncounterSpec, oflxName string, oflxSpec *bosstest.OverfluxSpec) []bosstest.SimResult {
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

	// Build overflux state from spec (nil for baseline).
	var oflxState *overflux.State
	var oflxScore int
	if oflxSpec != nil {
		oflxState = oflxSpec.ToOverfluxState()
		oflxScore = oflxState.TotalScore
		t.Logf("overflux: %s (score=%d, hp_mult=%.2f)", oflxName, oflxScore, oflxState.HPMultiplier())
	}

	var allResults []bosstest.SimResult
	groupID := fmt.Sprintf("fuzz_%s_%s_%d", spec.Label(), oflxName, os.Getpid())
	enemyDefs := spec.EnemyDefs()

	for _, comp := range spec.Compositions {
		party := comp.ToPartyConfigs()
		runsPerComp := runs / len(spec.Compositions)
		compGroupID := fmt.Sprintf("%s_%s", groupID, comp.Name)

		for i := range runsPerComp {
			result := bosstest.RunSimulation(bosstest.SimConfig{
				Boss:        spec.Label(),
				Enemies:     enemyDefs,
				Party:       party,
				Seed:        uint64(i),
				Sink:        sink,
				GroupID:     compGroupID,
				RunID:       fmt.Sprintf("%s_%d", compGroupID, i),
				PuppetTrees: puppetTrees,
				Overflux:    oflxState,
			})
			result.CompName = comp.Name
			result.OverfluxName = oflxName
			result.OverfluxScore = oflxScore
			allResults = append(allResults, result)
		}
	}

	// For overflux batches: only assert per-overflux win_rate, skip global
	// duration/phase/ability/tree assertions (they're tuned for baseline).
	// For baseline: assert everything.
	assertSpec := spec
	if oflxSpec != nil {
		specCopy := *spec
		if oflxSpec.WinRate != nil {
			specCopy.WinRate = *oflxSpec.WinRate
		}
		// Strip global assertions that don't apply to overflux variants.
		specCopy.Duration = bosstest.DurationSpec{}
		specCopy.PhaseReach = nil
		specCopy.AbilityStats = nil
		specCopy.SpecBalance = nil
		// Remove per-comp win rate assertions (inherited from base).
		for i := range specCopy.Compositions {
			specCopy.Compositions[i].WinRate = nil
		}
		assertSpec = &specCopy
	}

	fr := &bosstest.FuzzResults{
		Results: allResults,
		Spec:    assertSpec,
	}
	report := fr.AssertAll(t)
	report.OverfluxName = oflxName
	report.OverfluxScore = oflxScore

	// Tree health: only for baseline runs. Merge each enemy def's trees across
	// all runs, then assert per def (a pack has one tree per mob type).
	if oflxSpec == nil {
		merged := make(map[string]*bosstest.TreeReport)
		var defOrder []string
		for _, r := range allResults {
			for name, rep := range r.TreeReports {
				if rep == nil {
					continue
				}
				if existing, ok := merged[name]; ok {
					bosstest.MergeTreeReport(existing, rep)
				} else {
					merged[name] = bosstest.CloneTreeReport(rep)
					defOrder = append(defOrder, name)
				}
			}
		}
		slices.Sort(defOrder)
		// Single-tree encounters (a boss) keep the unprefixed layout.
		singleTree := len(defOrder) == 1
		for _, name := range defOrder {
			m := merged[name]
			bosstest.ClassifyTreeReport(m)
			label := name
			if singleTree {
				label = ""
			}
			t.Run("tree_health/"+name, func(t *testing.T) {
				bosstest.AssertTreeHealth(t, label, spec.TreeHealth, m, report)
			})
		}
	}

	report.PrintReport(t)
	return allResults
}

// --- Calibration ---

var flagCalibrate = flag.Bool("boss.calibrate", false, "run score calibration (outputs recommended ScorePerRank)")
var flagCalibIters = flag.Int("boss.calibrate-iterations", 150, "iterations per condition per rank")

func TestBoss_Calibrate(t *testing.T) {
	if !*flagCalibrate {
		t.Skip("score calibration disabled (pass -boss.calibrate to enable)")
	}

	specs, err := filepath.Glob("testdata/specs/*.yaml")
	if err != nil {
		t.Fatalf("glob specs: %v", err)
	}

	for _, specPath := range specs {
		base := strings.TrimSuffix(filepath.Base(specPath), ".yaml")
		if !shouldRunBoss(base) {
			continue
		}
		spec, err := bosstest.LoadSpec(specPath)
		if err != nil {
			t.Fatalf("load spec: %v", err)
		}
		spec.ExpandVariants()
		t.Run(base, func(t *testing.T) {
			runCalibration(t, spec)
		})
	}
}

func runCalibration(t *testing.T, spec *bosstest.EncounterSpec) {
	iters := *flagCalibIters
	sink := &combatlog.InMemorySink{}

	// Run baseline
	baselineWinRate := runCalibBatch(t, spec, nil, "baseline", iters, sink)

	type calibResult struct {
		condID  string
		rank    int
		winRate float64
		delta   float64
		score   int
	}
	var results []calibResult

	// Run each condition at each rank individually
	for _, def := range overflux.Registry {
		for rank := 1; rank <= def.MaxRank; rank++ {
			oflx := overflux.NewState([]overflux.ActiveCondition{
				{ID: def.ID, Rank: rank},
			})
			label := fmt.Sprintf("%s_%d", def.ID, rank)
			wr := runCalibBatch(t, spec, oflx, label, iters, sink)
			delta := baselineWinRate - wr
			if delta < 0 {
				delta = 0
			}
			// K=0.5: 2% win-rate drop per score point
			score := int(delta * 50)
			results = append(results, calibResult{
				condID: string(def.ID), rank: rank,
				winRate: wr, delta: delta, score: score,
			})
		}
	}

	// Print calibration table
	var sb strings.Builder
	fmt.Fprintf(&sb, "\n%s\n", strings.Repeat("━", 80))
	fmt.Fprintf(&sb, "  SCORE CALIBRATION: %s (%d iterations per condition)\n", spec.Boss, iters)
	fmt.Fprintf(&sb, "  Baseline win rate: %.1f%%\n", baselineWinRate*100)
	fmt.Fprintf(&sb, "%s\n\n", strings.Repeat("━", 80))
	fmt.Fprintf(&sb, "  %-20s %5s %10s %10s %10s\n", "Condition", "Rank", "Win%", "Delta", "Score")
	fmt.Fprintf(&sb, "  %s\n", strings.Repeat("─", 60))
	for _, r := range results {
		fmt.Fprintf(&sb, "  %-20s %5d %9.1f%% %9.1f%% %10d\n",
			r.condID, r.rank, r.winRate*100, r.delta*100, r.score)
	}

	// Compute recommended ScorePerRank (average score across ranks for ranked conditions)
	fmt.Fprintf(&sb, "\n  %s\n", strings.Repeat("─", 60))
	sb.WriteString("  Recommended ScorePerRank:\n")
	condScores := make(map[string][]int)
	for _, r := range results {
		condScores[r.condID] = append(condScores[r.condID], r.score)
	}
	for condID, scores := range condScores {
		if len(scores) == 1 {
			fmt.Fprintf(&sb, "    %-20s %d (single rank)\n", condID, scores[0])
		} else {
			// For ranked conditions, use score-per-rank = max_score / max_rank
			maxScore := scores[len(scores)-1]
			maxRank := len(scores)
			perRank := max(maxScore/maxRank, 1)
			fmt.Fprintf(&sb, "    %-20s %d (max score %d / %d ranks)\n", condID, perRank, maxScore, maxRank)
		}
	}
	fmt.Fprintf(&sb, "\n%s\n", strings.Repeat("━", 80))
	t.Log(sb.String())
}

func runCalibBatch(t *testing.T, spec *bosstest.EncounterSpec, oflx *overflux.State, label string, iters int, sink combatlog.EventSink) float64 {
	t.Helper()
	wins := 0
	total := 0
	groupID := fmt.Sprintf("calib_%s_%s_%d", spec.Label(), label, os.Getpid())
	enemyDefs := spec.EnemyDefs()

	for _, comp := range spec.Compositions {
		party := comp.ToPartyConfigs()
		runsPerComp := iters / len(spec.Compositions)
		for i := range runsPerComp {
			result := bosstest.RunSimulation(bosstest.SimConfig{
				Boss:        spec.Label(),
				Enemies:     enemyDefs,
				Party:       party,
				Seed:        uint64(i),
				Sink:        sink,
				GroupID:     groupID,
				RunID:       fmt.Sprintf("%s_%d", groupID, i),
				PuppetTrees: puppetTrees,
				Overflux:    oflx,
			})
			total++
			if result.Outcome == combatlog.OutcomePlayerWin {
				wins++
			}
		}
	}
	return float64(wins) / float64(total)
}
