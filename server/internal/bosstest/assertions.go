package bosstest

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"strings"
	"testing"
	"time"

	"codex-online/server/internal/combatlog"
)

// FuzzResults holds aggregated results from a batch of simulation runs.
type FuzzResults struct {
	Results []SimResult
	Spec    *EncounterSpec
}

// reportLine captures one row of the summary report.
type reportLine struct {
	label  string
	result string
	spec   string
	pass   bool
	dmg    float64 // used for sorting ability stats by damage
}

// FuzzReport collects all assertion outcomes for the summary table.
type FuzzReport struct {
	Boss          string
	Runs          int
	OverfluxName  string // "baseline" or overflux config name
	OverfluxScore int    // total overflux score (0 = baseline)
	Lines         []reportLine

	// Outcome distribution
	Wins     int
	Losses   int
	Timeouts int

	// Duration stats
	DurationMin time.Duration
	DurationMax time.Duration
	DurationAvg time.Duration

	// Per-composition breakdown
	CompBreakdown []compStats

	// Sections with sub-items
	SpecBalance     []reportLine
	AbilityStats    []reportLine
	TotalBossDamage float64            // grand total damage dealt by boss abilities
	CompDetails     []compDetailReport // per-composition breakdown
	TreeHealth      []reportLine

	// Tree node details (for inline listing)
	DeadNodes []string
	ColdNodes []string // "name (evals=N)"
}

// compStats holds per-composition outcome stats.
type compStats struct {
	Name     string
	Runs     int
	Wins     int
	Losses   int
	Timeouts int
	AvgDur   time.Duration
}

// compDetailReport holds per-composition ability damage and class balance for the report.
type compDetailReport struct {
	Name              string
	TotalBossDamage   float64 // boss damage dealt to players
	TotalPlayerDamage float64 // player damage dealt to boss
	TotalPlayerHeal   float64 // total healing done by players
	TotalDurationSec  float64 // sum of all run durations
	Runs              int
	AbilityShares     []nameShare    // sorted by damage desc
	SpecShares        []nameShare    // sorted alphabetically
	HealShares        []nameShare    // per-spec healing, sorted by amount desc
	SpecPlayerCount   map[string]int // spec → player count (for per-player DPS/HPS)
}

type nameShare struct {
	Name   string
	Share  float64 // 0-100 percent
	RawDmg float64
}

func (r *FuzzReport) add(label, result, spec string, pass bool) {
	r.Lines = append(r.Lines, reportLine{label: label, result: result, spec: spec, pass: pass})
}

func statusIcon(pass bool) string { //nolint:revive // flag-parameter: simple formatting helper
	if pass {
		return "ok"
	}
	return "FAIL"
}

// PrintReport outputs the formatted summary table via t.Log.
func (r *FuzzReport) PrintReport(t *testing.T) {
	t.Helper()
	var sb strings.Builder

	fmt.Fprintf(&sb, "\n%s\n", strings.Repeat("━", 64))
	if r.OverfluxScore > 0 {
		fmt.Fprintf(&sb, "  FUZZ REPORT: %s [overflux: %s, score=%d]\n", r.Boss, r.OverfluxName, r.OverfluxScore)
	} else {
		fmt.Fprintf(&sb, "  FUZZ REPORT: %s [baseline]\n", r.Boss)
	}
	fmt.Fprintf(&sb, "  %d runs | %d wins | %d losses | %d timeouts\n",
		r.Runs, r.Wins, r.Losses, r.Timeouts)
	fmt.Fprintf(&sb, "  duration: avg %s | min %s | max %s\n",
		fmtDuration(r.DurationAvg), fmtDuration(r.DurationMin), fmtDuration(r.DurationMax))
	fmt.Fprintf(&sb, "%s\n\n", strings.Repeat("━", 64))

	r.printCompBreakdownSection(&sb)
	r.printMetricsSection(&sb)
	r.printSpecBalanceSection(&sb)
	r.printAbilityStatsSection(&sb)
	r.printCompDetailsSection(&sb)
	r.printTreeHealthSection(&sb)

	fmt.Fprintf(&sb, "\n%s\n", strings.Repeat("━", 64))
	t.Log(sb.String())
}

func (r *FuzzReport) printCompBreakdownSection(sb *strings.Builder) {
	if len(r.CompBreakdown) <= 1 {
		return
	}
	fmt.Fprintf(sb, "  %-20s %6s %6s %6s %8s   %s\n",
		"Composition", "Runs", "Win%", "Loss%", "T/O%", "Avg Dur")
	fmt.Fprintf(sb, "  %s\n", strings.Repeat("─", 58))
	for _, cs := range r.CompBreakdown {
		winPct := 100.0 * float64(cs.Wins) / float64(cs.Runs)
		lossPct := 100.0 * float64(cs.Losses) / float64(cs.Runs)
		toPct := 100.0 * float64(cs.Timeouts) / float64(cs.Runs)
		fmt.Fprintf(sb, "    %-18s %6d %5.1f%% %5.1f%% %7.1f%%   %s\n",
			cs.Name, cs.Runs, winPct, lossPct, toPct, fmtDuration(cs.AvgDur))
	}
	sb.WriteString("\n")
}

func (r *FuzzReport) printMetricsSection(sb *strings.Builder) {
	fmt.Fprintf(sb, "  %-20s %-14s %-14s %s\n", "Metric", "Result", "Spec", "")
	fmt.Fprintf(sb, "  %s\n", strings.Repeat("─", 58))
	for _, l := range r.Lines {
		fmt.Fprintf(sb, "  %-20s %-14s %-14s [%s]\n", l.label, l.result, l.spec, statusIcon(l.pass))
	}
}

func (r *FuzzReport) printSpecBalanceSection(sb *strings.Builder) {
	if len(r.SpecBalance) == 0 {
		return
	}
	fmt.Fprintf(sb, "\n  %s\n", strings.Repeat("─", 58))
	h := r.SpecBalance[0]
	fmt.Fprintf(sb, "  Spec Balance      sigma=%-8s max=%-8s [%s]\n",
		h.result, h.spec, statusIcon(h.pass))
	for _, l := range r.SpecBalance[1:] {
		fmt.Fprintf(sb, "    %-14s %s\n", l.label, l.result)
	}
}

func (r *FuzzReport) printAbilityStatsSection(sb *strings.Builder) {
	if len(r.AbilityStats) == 0 {
		return
	}
	fmt.Fprintf(sb, "\n  %s\n", strings.Repeat("─", 58))
	fmt.Fprintf(sb, "  Boss Damage (total: %.0f)\n", r.TotalBossDamage)
	for _, l := range r.AbilityStats {
		fmt.Fprintf(sb, "    %-14s %s [%s]\n", l.label, l.result, statusIcon(l.pass))
	}
}

func (r *FuzzReport) printCompDetailsSection(sb *strings.Builder) {
	if len(r.CompDetails) == 0 {
		return
	}
	fmt.Fprintf(sb, "\n  %s\n", strings.Repeat("─", 58))
	sb.WriteString("  Per-Composition Breakdown\n")
	for _, cd := range r.CompDetails {
		r.printSingleCompDetail(sb, cd)
	}
}

func (r *FuzzReport) printSingleCompDetail(sb *strings.Builder, cd compDetailReport) {
	avgDur := cd.TotalDurationSec / float64(cd.Runs)
	partyDPS := cd.TotalPlayerDamage / cd.TotalDurationSec
	fmt.Fprintf(sb, "\n    %s (boss dmg: %.0f | party DPS: %.1f/s | avg %.1fs)\n",
		cd.Name, cd.TotalBossDamage, partyDPS, avgDur)
	// Ability damage shares
	sb.WriteString("      ability:  ")
	for i, a := range cd.AbilityShares {
		if i > 0 {
			sb.WriteString("  ")
		}
		fmt.Fprintf(sb, "%s %.1f%%", a.Name, a.Share)
	}
	sb.WriteString("\n")
	// Class balance (shares)
	sb.WriteString("      spec:     ")
	for i, c := range cd.SpecShares {
		if i > 0 {
			sb.WriteString("  ")
		}
		fmt.Fprintf(sb, "%s %.1f%%", c.Name, c.Share)
	}
	sb.WriteString("\n")
	// Per-player DPS (normalized by player count per spec)
	if cd.TotalDurationSec > 0 {
		sb.WriteString("      dps/player: ")
		for i, c := range cd.SpecShares {
			if i > 0 {
				sb.WriteString("  ")
			}
			n := max(cd.SpecPlayerCount[c.Name], 1)
			playerDPS := c.RawDmg / cd.TotalDurationSec / float64(n)
			fmt.Fprintf(sb, "%s %.1f/s", c.Name, playerDPS)
		}
		sb.WriteString("\n")
	}
	// Healing stats
	if cd.TotalPlayerHeal > 0 && cd.TotalDurationSec > 0 {
		partyHPS := cd.TotalPlayerHeal / cd.TotalDurationSec
		fmt.Fprintf(sb, "      healing:  total %.0f | party HPS: %.1f/s\n", cd.TotalPlayerHeal, partyHPS)
		sb.WriteString("      hps/player: ")
		for i, h := range cd.HealShares {
			if i > 0 {
				sb.WriteString("  ")
			}
			n := max(cd.SpecPlayerCount[h.Name], 1)
			playerHPS := h.RawDmg / cd.TotalDurationSec / float64(n)
			fmt.Fprintf(sb, "%s %.1f/s (%.0f%%)", h.Name, playerHPS, h.Share)
		}
		sb.WriteString("\n")
	}
}

func (r *FuzzReport) printTreeHealthSection(sb *strings.Builder) {
	if len(r.TreeHealth) == 0 {
		return
	}
	fmt.Fprintf(sb, "\n  %s\n", strings.Repeat("─", 58))
	sb.WriteString("  Tree Health\n")
	for _, l := range r.TreeHealth {
		fmt.Fprintf(sb, "    %-14s %-24s [%s]\n", l.label, l.result, statusIcon(l.pass))
	}
	if len(r.DeadNodes) > 0 {
		sb.WriteString("\n    dead:\n")
		for _, name := range r.DeadNodes {
			fmt.Fprintf(sb, "      %s\n", name)
		}
	}
	if len(r.ColdNodes) > 0 {
		sb.WriteString("\n    cold:\n")
		for _, name := range r.ColdNodes {
			fmt.Fprintf(sb, "      %s\n", name)
		}
	}
}

func fmtDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}

// AssertAll runs all spec assertions as Go subtests and returns the report
// for further additions (e.g. tree health) before printing.
func (fr *FuzzResults) AssertAll(t *testing.T) *FuzzReport {
	t.Helper()

	report := &FuzzReport{
		Boss: fr.Spec.Boss,
		Runs: len(fr.Results),
	}

	computeOutcomeStats(fr.Results, report)
	groupByComposition(fr.Results, report)

	t.Run("win_rate", func(t *testing.T) { fr.assertWinRate(t, report) })
	t.Run("duration", func(t *testing.T) { fr.assertDuration(t, report) })
	for _, ps := range fr.Spec.PhaseReach {
		t.Run("phase_reach/"+itoa(ps.Phase), func(t *testing.T) {
			fr.assertPhaseReach(t, ps, report)
		})
	}
	if fr.Spec.SpecBalance != nil {
		t.Run("spec_balance", func(t *testing.T) { fr.assertSpecBalance(t, report) })
	}
	if len(fr.Spec.AbilityStats) > 0 {
		t.Run("ability_stats", func(t *testing.T) {
			fr.assertAbilityStats(t, fr.Spec.AbilityStats, report)
		})
	}

	// Per-composition win rate assertions.
	fr.assertCompWinRates(t, report)

	// Compute per-composition ability damage and class balance (informational).
	fr.computePerCompDetails(report)

	return report
}

// computeOutcomeStats tallies wins/losses/timeouts and duration min/max/avg
// from the results slice, writing directly into report.
func computeOutcomeStats(results []SimResult, report *FuzzReport) {
	var minDur, maxDur, totalDur time.Duration
	for i, r := range results {
		switch r.Outcome {
		case combatlog.OutcomePlayerWin:
			report.Wins++
		case combatlog.OutcomeBossWin:
			report.Losses++
		default:
			report.Timeouts++
		}
		totalDur += r.Duration
		if i == 0 || r.Duration < minDur {
			minDur = r.Duration
		}
		if r.Duration > maxDur {
			maxDur = r.Duration
		}
	}
	if len(results) > 0 {
		report.DurationAvg = totalDur / time.Duration(len(results))
	}
	report.DurationMin = minDur
	report.DurationMax = maxDur
}

// groupByComposition builds the per-composition breakdown and appends it to
// report.CompBreakdown, sorted by win-rate descending.
func groupByComposition(results []SimResult, report *FuzzReport) {
	compMap := make(map[string]*compStats)
	var compOrder []string
	for _, r := range results {
		cs, ok := compMap[r.CompName]
		if !ok {
			cs = &compStats{Name: r.CompName}
			compMap[r.CompName] = cs
			compOrder = append(compOrder, r.CompName)
		}
		cs.Runs++
		cs.AvgDur += r.Duration
		switch r.Outcome {
		case combatlog.OutcomePlayerWin:
			cs.Wins++
		case combatlog.OutcomeBossWin:
			cs.Losses++
		default:
			cs.Timeouts++
		}
	}
	for _, name := range compOrder {
		cs := compMap[name]
		if cs.Runs > 0 {
			cs.AvgDur /= time.Duration(cs.Runs)
		}
		report.CompBreakdown = append(report.CompBreakdown, *cs)
	}
	// Sort compositions by win% descending (sweaty first).
	slices.SortFunc(report.CompBreakdown, func(a, b compStats) int {
		wa := float64(a.Wins) / float64(a.Runs)
		wb := float64(b.Wins) / float64(b.Runs)
		if wa > wb {
			return -1
		}
		if wa < wb {
			return 1
		}
		return 0
	})
}

// AssertTreeHealth validates instrumented tree metrics against spec.
func AssertTreeHealth(t *testing.T, spec TreeHealthSpec, report *TreeReport, fuzzReport *FuzzReport) {
	t.Helper()
	t.Run("dead_nodes", func(t *testing.T) {
		dead := report.DeadCount()
		pass := dead <= spec.MaxDeadNodes
		if !pass {
			t.Errorf("dead nodes = %d, max allowed = %d", dead, spec.MaxDeadNodes)
		}
		if fuzzReport != nil {
			fuzzReport.TreeHealth = append(fuzzReport.TreeHealth, reportLine{
				label:  "dead nodes",
				result: fmt.Sprintf("%d (max %d)", dead, spec.MaxDeadNodes),
				pass:   pass,
			})
			for _, n := range report.Nodes {
				if n.Classification == ClassDead {
					fuzzReport.DeadNodes = append(fuzzReport.DeadNodes, n.Name)
				}
			}
		}
	})
	t.Run("cold_nodes", func(t *testing.T) {
		cold := report.ColdCount()
		pass := cold <= spec.MaxColdNodes
		if !pass {
			t.Errorf("cold nodes = %d, max allowed = %d", cold, spec.MaxColdNodes)
		}
		if fuzzReport != nil {
			fuzzReport.TreeHealth = append(fuzzReport.TreeHealth, reportLine{
				label:  "cold nodes",
				result: fmt.Sprintf("%d (max %d)", cold, spec.MaxColdNodes),
				pass:   pass,
			})
			for _, n := range report.Nodes {
				if n.Classification == ClassCold {
					fuzzReport.ColdNodes = append(fuzzReport.ColdNodes,
						fmt.Sprintf("%s (evals=%d)", n.Name, n.EvalCount))
				}
			}
		}
	})
}

// computePerCompDetails aggregates ability damage shares and class balance
// per composition for the report. Ordered to match CompBreakdown (win% desc).
func (fr *FuzzResults) computePerCompDetails(report *FuzzReport) {
	// Group results by composition.
	byComp := make(map[string][]SimResult)
	for _, r := range fr.Results {
		byComp[r.CompName] = append(byComp[r.CompName], r)
	}

	// Build details in the same order as CompBreakdown (already sorted by win% desc).
	for _, cs := range report.CompBreakdown {
		results := byComp[cs.Name]
		detail := compDetailReport{Name: cs.Name, Runs: len(results)}

		for _, r := range results {
			detail.TotalDurationSec += r.Duration.Seconds()
		}

		aggregateAbilityDamage(results, &detail)
		aggregateSpecDamage(results, &detail)
		aggregateSpecHealing(results, &detail)

		report.CompDetails = append(report.CompDetails, detail)
	}
}

// aggregateAbilityDamage fills detail.TotalBossDamage and detail.AbilityShares
// from the boss-ability damage events across all results.
func aggregateAbilityDamage(results []SimResult, detail *compDetailReport) {
	abilDmg := make(map[string]float64)
	for _, r := range results {
		for name, ar := range r.AbilityStats {
			abilDmg[name] += float64(ar.TotalDamage)
		}
	}
	var totalAbilDmg float64
	for _, d := range abilDmg {
		totalAbilDmg += d
	}
	detail.TotalBossDamage = totalAbilDmg
	for name, d := range abilDmg {
		var pct float64
		if totalAbilDmg > 0 {
			pct = d / totalAbilDmg * 100
		}
		detail.AbilityShares = append(detail.AbilityShares, nameShare{Name: name, Share: pct, RawDmg: d})
	}
	slices.SortFunc(detail.AbilityShares, func(a, b nameShare) int {
		return cmp.Compare(b.RawDmg, a.RawDmg)
	})
}

// aggregateSpecDamage fills detail.TotalPlayerDamage, detail.SpecPlayerCount,
// and detail.SpecShares from player damage events across all results.
func aggregateSpecDamage(results []SimResult, detail *compDetailReport) {
	specDmg := make(map[string]float64)
	specPlayers := make(map[string]int)
	var totalSpecDmg float64
	for _, r := range results {
		for spec, d := range r.SpecDamage {
			specDmg[spec] += float64(d)
			totalSpecDmg += float64(d)
		}
		for spec, n := range r.SpecPlayers {
			if n > specPlayers[spec] {
				specPlayers[spec] = n
			}
		}
	}
	detail.TotalPlayerDamage = totalSpecDmg
	detail.SpecPlayerCount = specPlayers
	// Build spec list from specPlayers (all specs), not specDmg (only specs that dealt damage).
	specs := make([]string, 0, len(specPlayers))
	for spec := range specPlayers {
		specs = append(specs, spec)
	}
	slices.Sort(specs)
	for _, spec := range specs {
		var pct float64
		if totalSpecDmg > 0 {
			pct = specDmg[spec] / totalSpecDmg * 100
		}
		detail.SpecShares = append(detail.SpecShares, nameShare{Name: spec, Share: pct, RawDmg: specDmg[spec]})
	}
}

// aggregateSpecHealing fills detail.TotalPlayerHeal and detail.HealShares
// from healer events across all results.
func aggregateSpecHealing(results []SimResult, detail *compDetailReport) {
	specHeal := make(map[string]float64)
	var totalSpecHeal float64
	for _, r := range results {
		for spec, h := range r.SpecHealing {
			specHeal[spec] += float64(h)
			totalSpecHeal += float64(h)
		}
	}
	detail.TotalPlayerHeal = totalSpecHeal
	if totalSpecHeal == 0 {
		return
	}
	healSpecs := make([]string, 0, len(specHeal))
	for spec := range specHeal {
		healSpecs = append(healSpecs, spec)
	}
	slices.Sort(healSpecs)
	for _, spec := range healSpecs {
		pct := specHeal[spec] / totalSpecHeal * 100
		detail.HealShares = append(detail.HealShares, nameShare{Name: spec, Share: pct, RawDmg: specHeal[spec]})
	}
	// Sort by healing done descending.
	slices.SortFunc(detail.HealShares, func(a, b nameShare) int {
		return cmp.Compare(b.RawDmg, a.RawDmg)
	})
}

func (fr *FuzzResults) assertWinRate(t *testing.T, report *FuzzReport) {
	t.Helper()
	wins := 0
	for _, r := range fr.Results {
		if r.Outcome == combatlog.OutcomePlayerWin {
			wins++
		}
	}
	rate := float64(wins) / float64(len(fr.Results))
	pass := true
	if fr.Spec.WinRate.Min > 0 && rate < fr.Spec.WinRate.Min {
		t.Errorf("win rate = %.3f, want >= %.3f (%d/%d)", rate, fr.Spec.WinRate.Min, wins, len(fr.Results))
		pass = false
	}
	if fr.Spec.WinRate.Max > 0 && rate > fr.Spec.WinRate.Max {
		t.Errorf("win rate = %.3f, want <= %.3f (%d/%d)", rate, fr.Spec.WinRate.Max, wins, len(fr.Results))
		pass = false
	}
	report.add("Win rate",
		fmt.Sprintf("%.1f%%", rate*100),
		fmt.Sprintf("[%.0f%%-%.0f%%]", fr.Spec.WinRate.Min*100, fr.Spec.WinRate.Max*100),
		pass)
}

func (fr *FuzzResults) assertCompWinRates(t *testing.T, report *FuzzReport) {
	t.Helper()

	// Group results by composition name.
	byComp := make(map[string][]SimResult)
	for _, r := range fr.Results {
		byComp[r.CompName] = append(byComp[r.CompName], r)
	}

	// Deduplicate compositions by name (ExpandVariants produces multiples).
	seen := make(map[string]bool)
	for _, comp := range fr.Spec.Compositions {
		if comp.WinRate == nil || seen[comp.Name] {
			continue
		}
		seen[comp.Name] = true
		comp := comp
		t.Run("comp_win_rate/"+comp.Name, func(t *testing.T) {
			results := byComp[comp.Name]
			if len(results) == 0 {
				return
			}
			wins := 0
			for _, r := range results {
				if r.Outcome == combatlog.OutcomePlayerWin {
					wins++
				}
			}
			rate := float64(wins) / float64(len(results))
			pass := true
			if comp.WinRate.Min > 0 && rate < comp.WinRate.Min {
				t.Errorf("%s win rate = %.3f, want >= %.3f (%d/%d)",
					comp.Name, rate, comp.WinRate.Min, wins, len(results))
				pass = false
			}
			if comp.WinRate.Max > 0 && rate > comp.WinRate.Max {
				t.Errorf("%s win rate = %.3f, want <= %.3f (%d/%d)",
					comp.Name, rate, comp.WinRate.Max, wins, len(results))
				pass = false
			}
			report.add(
				"  "+comp.Name,
				fmt.Sprintf("%.1f%%", rate*100),
				fmt.Sprintf("[%.0f%%-%.0f%%]", comp.WinRate.Min*100, comp.WinRate.Max*100),
				pass,
			)
		})
	}
}

func (fr *FuzzResults) assertDuration(t *testing.T, report *FuzzReport) {
	t.Helper()
	var totalDuration time.Duration
	for _, r := range fr.Results {
		totalDuration += r.Duration
	}
	avgSec := totalDuration.Seconds() / float64(len(fr.Results))
	pass := true
	if fr.Spec.Duration.MinSeconds > 0 && avgSec < fr.Spec.Duration.MinSeconds {
		t.Errorf("avg duration = %.1fs, want >= %.1fs", avgSec, fr.Spec.Duration.MinSeconds)
		pass = false
	}
	if fr.Spec.Duration.MaxSeconds > 0 && avgSec > fr.Spec.Duration.MaxSeconds {
		t.Errorf("avg duration = %.1fs, want <= %.1fs", avgSec, fr.Spec.Duration.MaxSeconds)
		pass = false
	}
	report.add("Avg duration",
		fmt.Sprintf("%.1fs", avgSec),
		fmt.Sprintf("[%.0fs-%.0fs]", fr.Spec.Duration.MinSeconds, fr.Spec.Duration.MaxSeconds),
		pass)
}

func (fr *FuzzResults) assertPhaseReach(t *testing.T, ps PhaseSpec, report *FuzzReport) {
	t.Helper()
	reached := 0
	for _, r := range fr.Results {
		if slices.Contains(r.PhasesReached, ps.Phase) {
			reached++
		}
	}
	rate := float64(reached) / float64(len(fr.Results))
	pass := true
	if ps.MinRate > 0 && rate < ps.MinRate {
		t.Errorf("phase %d reach rate = %.3f, want >= %.3f", ps.Phase, rate, ps.MinRate)
		pass = false
	}
	if ps.MaxRate > 0 && rate > ps.MaxRate {
		t.Errorf("phase %d reach rate = %.3f, want <= %.3f", ps.Phase, rate, ps.MaxRate)
		pass = false
	}

	specStr := ""
	if ps.MinRate > 0 {
		specStr = fmt.Sprintf(">=%.0f%%", ps.MinRate*100)
	}
	if ps.MaxRate > 0 {
		if specStr != "" {
			specStr += ", "
		}
		specStr += fmt.Sprintf("<=%.0f%%", ps.MaxRate*100)
	}
	report.add(fmt.Sprintf("Phase %d reach", ps.Phase),
		fmt.Sprintf("%.1f%%", rate*100),
		specStr,
		pass)
}

func (fr *FuzzResults) assertSpecBalance(t *testing.T, report *FuzzReport) {
	t.Helper()
	totalBySpec := make(map[string]float64)
	var grandTotal float64
	for _, r := range fr.Results {
		for spec, dmg := range r.SpecDamage {
			totalBySpec[spec] += float64(dmg)
			grandTotal += float64(dmg)
		}
	}
	if grandTotal == 0 || len(totalBySpec) < 2 {
		return
	}

	shares := make([]float64, 0, len(totalBySpec))
	for _, dmg := range totalBySpec {
		shares = append(shares, dmg/grandTotal)
	}

	mean := 1.0 / float64(len(shares))
	var variance float64
	for _, s := range shares {
		diff := s - mean
		variance += diff * diff
	}
	sigma := math.Sqrt(variance / float64(len(shares)))

	maxSigma := fr.Spec.SpecBalance.MaxDamageShareSigma
	pass := maxSigma <= 0 || sigma <= maxSigma
	if !pass {
		t.Errorf("spec damage share sigma = %.4f, want <= %.4f", sigma, maxSigma)
	}

	// Header line
	report.SpecBalance = append(report.SpecBalance, reportLine{
		result: fmt.Sprintf("%.4f", sigma),
		spec:   fmt.Sprintf("%.2f", maxSigma),
		pass:   pass,
	})
	// Per-spec lines (sorted for stable output)
	specs := make([]string, 0, len(totalBySpec))
	for spec := range totalBySpec {
		specs = append(specs, spec)
	}
	slices.Sort(specs)
	for _, spec := range specs {
		dmg := totalBySpec[spec]
		report.SpecBalance = append(report.SpecBalance, reportLine{
			label:  spec,
			result: fmt.Sprintf("%.1f%%", (dmg/grandTotal)*100),
		})
	}
}

// aggregateAbilityStats merges per-run AbilityResult maps from all results into
// a single map keyed by ability name.
func aggregateAbilityStats(results []SimResult) map[string]*AbilityResult {
	agg := make(map[string]*AbilityResult)
	for _, r := range results {
		for name, ar := range r.AbilityStats {
			if existing, ok := agg[name]; ok {
				existing.Hits += ar.Hits
				existing.Kills += ar.Kills
				existing.Dodges += ar.Dodges
				existing.TotalDamage += ar.TotalDamage
			} else {
				agg[name] = &AbilityResult{
					Name:        name,
					Hits:        ar.Hits,
					Kills:       ar.Kills,
					Dodges:      ar.Dodges,
					TotalDamage: ar.TotalDamage,
				}
			}
		}
	}
	return agg
}

func (fr *FuzzResults) assertAbilityStats(t *testing.T, specs []AbilitySpec, report *FuzzReport) {
	t.Helper()
	agg := aggregateAbilityStats(fr.Results)

	// Compute grand total damage for share percentages.
	var grandTotalDmg float64
	for _, ar := range agg {
		grandTotalDmg += float64(ar.TotalDamage)
	}
	report.TotalBossDamage = grandTotalDmg

	// Spec'd ability assertions
	specNames := make(map[string]bool, len(specs))
	for _, spec := range specs {
		specNames[spec.Ability] = true
		spec := spec
		t.Run(spec.Ability, func(t *testing.T) {
			assertSpecdAbility(t, spec, agg, grandTotalDmg, report)
		})
	}

	appendUnspecdAbilities(agg, specNames, grandTotalDmg, report)

	// Sort ability stats by damage descending for readability.
	slices.SortFunc(report.AbilityStats, func(a, b reportLine) int {
		return cmp.Compare(b.dmg, a.dmg)
	})
}

// assertSpecdAbility runs the kill-rate and dodge-rate assertions for a single
// spec'd ability and appends the result line to the report.
func assertSpecdAbility(t *testing.T, spec AbilitySpec, agg map[string]*AbilityResult, grandTotalDmg float64, report *FuzzReport) {
	t.Helper()
	ar, ok := agg[spec.Ability]
	if !ok {
		report.AbilityStats = append(report.AbilityStats, reportLine{
			label: spec.Ability, result: "no data", pass: true,
		})
		return
	}
	total := ar.Hits + ar.Dodges
	if total == 0 {
		report.AbilityStats = append(report.AbilityStats, reportLine{
			label: spec.Ability, result: "no interactions", pass: true,
		})
		return
	}

	pass := true
	var sharePct float64
	if grandTotalDmg > 0 {
		sharePct = float64(ar.TotalDamage) / grandTotalDmg * 100
	}
	parts := []string{fmt.Sprintf("%4.1f%%", sharePct)}

	if spec.MaxKillRate > 0 {
		killRate := float64(ar.Kills) / float64(total)
		if killRate > spec.MaxKillRate {
			t.Errorf("ability %q kill rate = %.3f, want <= %.3f (%d/%d)",
				spec.Ability, killRate, spec.MaxKillRate, ar.Kills, total)
			pass = false
		}
		parts = append(parts, fmt.Sprintf("kill=%.0f%% (max %.0f%%)", killRate*100, spec.MaxKillRate*100))
	}

	if spec.MinDodgeRate > 0 {
		dodgeRate := float64(ar.Dodges) / float64(total)
		if dodgeRate < spec.MinDodgeRate {
			t.Errorf("ability %q dodge rate = %.3f, want >= %.3f (%d/%d)",
				spec.Ability, dodgeRate, spec.MinDodgeRate, ar.Dodges, total)
			pass = false
		}
		parts = append(parts, fmt.Sprintf("dodge=%.0f%% (min %.0f%%)", dodgeRate*100, spec.MinDodgeRate*100))
	}

	parts = append(parts, fmt.Sprintf("dmg=%.0f hits=%d", ar.TotalDamage, ar.Hits))
	report.AbilityStats = append(report.AbilityStats, reportLine{
		label:  spec.Ability,
		result: strings.Join(parts, "  "),
		pass:   pass,
		dmg:    float64(ar.TotalDamage),
	})
}

// appendUnspecdAbilities appends informational (always-pass) report lines for
// abilities present in the aggregated data but not covered by a spec.
func appendUnspecdAbilities(agg map[string]*AbilityResult, specNames map[string]bool, grandTotalDmg float64, report *FuzzReport) {
	var unspecced []string
	for name := range agg {
		if !specNames[name] {
			unspecced = append(unspecced, name)
		}
	}
	slices.Sort(unspecced)
	for _, name := range unspecced {
		ar := agg[name]
		var sharePct float64
		if grandTotalDmg > 0 {
			sharePct = float64(ar.TotalDamage) / grandTotalDmg * 100
		}
		report.AbilityStats = append(report.AbilityStats, reportLine{
			label:  name,
			result: fmt.Sprintf("%4.1f%%  dmg=%.0f hits=%d kills=%d", sharePct, ar.TotalDamage, ar.Hits, ar.Kills),
			pass:   true,
			dmg:    float64(ar.TotalDamage),
		})
	}
}

// --- Tier Report (injection / scenario) ---

// TierCase holds the outcome of a single injection or scenario test case.
type TierCase struct {
	Name   string
	Setup  string // e.g. "HP=59%, 5 ticks" or "pos=(0,0,15), 3 ticks"
	Assert string // what was checked
	Detail string // post-run state summary
	Passed bool
}

// TierReport collects cases for injection/scenario tiers and prints a summary.
type TierReport struct {
	Boss  string
	Tier  string
	Cases []TierCase
}

// Add records a test case result.
func (tr *TierReport) Add(c TierCase) {
	tr.Cases = append(tr.Cases, c)
}

// Print outputs the tier summary via t.Log.
func (tr *TierReport) Print(t *testing.T) {
	t.Helper()
	if len(tr.Cases) == 0 {
		return
	}

	passed := 0
	for _, c := range tr.Cases {
		if c.Passed {
			passed++
		}
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "\n%s\n", strings.Repeat("━", 64))
	fmt.Fprintf(&sb, "  %s REPORT: %s (%d/%d passed)\n",
		strings.ToUpper(tr.Tier), tr.Boss, passed, len(tr.Cases))
	fmt.Fprintf(&sb, "%s\n\n", strings.Repeat("━", 64))

	for _, c := range tr.Cases {
		fmt.Fprintf(&sb, "  [%s] %s\n", statusIcon(c.Passed), c.Name)
		if c.Setup != "" {
			fmt.Fprintf(&sb, "        setup:  %s\n", c.Setup)
		}
		fmt.Fprintf(&sb, "        assert: %s\n", c.Assert)
		if c.Detail != "" {
			fmt.Fprintf(&sb, "        result: %s\n", c.Detail)
		}
	}

	fmt.Fprintf(&sb, "\n%s\n", strings.Repeat("━", 64))
	t.Log(sb.String())
}

// --- Summary Matrix (win rate: conditions x compositions) ---

// SummaryMatrix collects results across all overflux batches to print a single
// cross-cutting table: rows = compositions (player skill), columns = overflux conditions.
type SummaryMatrix struct {
	Boss    string
	Results []SimResult
}

// PrintMatrix outputs the win rate matrix via t.Log.
func (m *SummaryMatrix) PrintMatrix(t *testing.T) { //nolint:gocognit,funlen // table rendering
	t.Helper()
	if len(m.Results) == 0 {
		return
	}

	oflxOrder, compOrder := m.collectAxes()
	if len(oflxOrder) == 0 || len(compOrder) == 0 {
		return
	}

	type cell struct{ wins, total int }
	data := make(map[string]map[string]*cell)
	for _, r := range m.Results {
		if data[r.CompName] == nil {
			data[r.CompName] = make(map[string]*cell)
		}
		if data[r.CompName][r.OverfluxName] == nil {
			data[r.CompName][r.OverfluxName] = &cell{}
		}
		data[r.CompName][r.OverfluxName].total++
		if r.Outcome == combatlog.OutcomePlayerWin {
			data[r.CompName][r.OverfluxName].wins++
		}
	}

	oflxTotals := make(map[string]*cell)
	for _, r := range m.Results {
		if oflxTotals[r.OverfluxName] == nil {
			oflxTotals[r.OverfluxName] = &cell{}
		}
		oflxTotals[r.OverfluxName].total++
		if r.Outcome == combatlog.OutcomePlayerWin {
			oflxTotals[r.OverfluxName].wins++
		}
	}

	labelW := 18
	for _, c := range compOrder {
		if len(c)+2 > labelW {
			labelW = len(c) + 2
		}
	}
	colW := 12
	for _, o := range oflxOrder {
		if len(o)+3 > colW {
			colW = len(o) + 3
		}
	}

	totalW := labelW + colW*len(oflxOrder) + 4
	var sb strings.Builder
	fmt.Fprintf(&sb, "\n%s\n", strings.Repeat("━", totalW))
	fmt.Fprintf(&sb, "  WIN RATE MATRIX: %s\n", m.Boss)
	fmt.Fprintf(&sb, "%s\n\n", strings.Repeat("━", totalW))

	fmt.Fprintf(&sb, "  %-*s", labelW, "")
	for _, oflx := range oflxOrder {
		fmt.Fprintf(&sb, " %*s", colW, oflx)
	}
	sb.WriteString("\n")
	fmt.Fprintf(&sb, "  %s\n", strings.Repeat("─", labelW+colW*len(oflxOrder)))

	for _, comp := range compOrder {
		fmt.Fprintf(&sb, "  %-*s", labelW, comp)
		for _, oflx := range oflxOrder {
			c := data[comp][oflx]
			if c == nil || c.total == 0 {
				fmt.Fprintf(&sb, " %*s", colW, "-")
			} else {
				pct := 100.0 * float64(c.wins) / float64(c.total)
				fmt.Fprintf(&sb, " %*.1f%%", colW-1, pct)
			}
		}
		sb.WriteString("\n")
	}

	fmt.Fprintf(&sb, "  %s\n", strings.Repeat("─", labelW+colW*len(oflxOrder)))
	fmt.Fprintf(&sb, "  %-*s", labelW, "TOTAL")
	for _, oflx := range oflxOrder {
		c := oflxTotals[oflx]
		if c == nil || c.total == 0 {
			fmt.Fprintf(&sb, " %*s", colW, "-")
		} else {
			pct := 100.0 * float64(c.wins) / float64(c.total)
			fmt.Fprintf(&sb, " %*.1f%%", colW-1, pct)
		}
	}
	sb.WriteString("\n")

	fmt.Fprintf(&sb, "\n%s\n", strings.Repeat("━", totalW))
	t.Log(sb.String())
}

func (m *SummaryMatrix) collectAxes() (oflxOrder, compOrder []string) {
	oflxSeen := make(map[string]bool)
	compSeen := make(map[string]bool)
	for _, r := range m.Results {
		if !oflxSeen[r.OverfluxName] {
			oflxSeen[r.OverfluxName] = true
			oflxOrder = append(oflxOrder, r.OverfluxName)
		}
		if !compSeen[r.CompName] {
			compSeen[r.CompName] = true
			compOrder = append(compOrder, r.CompName)
		}
	}
	return
}
