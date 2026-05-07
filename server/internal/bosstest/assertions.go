package bosstest

import (
	"fmt"
	"math"
	"sort"
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
	Boss  string
	Runs  int
	Lines []reportLine

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
	ClassBalance    []reportLine
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
	Name            string
	TotalBossDamage float64
	AbilityShares   []nameShare // sorted by damage desc
	ClassShares     []nameShare // sorted alphabetically
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

	sb.WriteString(fmt.Sprintf("\n%s\n", strings.Repeat("━", 64)))
	sb.WriteString(fmt.Sprintf("  FUZZ REPORT: %s\n", r.Boss))
	sb.WriteString(fmt.Sprintf("  %d runs | %d wins | %d losses | %d timeouts\n",
		r.Runs, r.Wins, r.Losses, r.Timeouts))
	sb.WriteString(fmt.Sprintf("  duration: avg %s | min %s | max %s\n",
		fmtDuration(r.DurationAvg), fmtDuration(r.DurationMin), fmtDuration(r.DurationMax)))
	sb.WriteString(fmt.Sprintf("%s\n\n", strings.Repeat("━", 64)))

	// Per-composition breakdown
	if len(r.CompBreakdown) > 1 {
		sb.WriteString(fmt.Sprintf("  %-20s %6s %6s %6s %8s   %s\n",
			"Composition", "Runs", "Win%", "Loss%", "T/O%", "Avg Dur"))
		sb.WriteString(fmt.Sprintf("  %s\n", strings.Repeat("─", 58)))
		for _, cs := range r.CompBreakdown {
			winPct := 100.0 * float64(cs.Wins) / float64(cs.Runs)
			lossPct := 100.0 * float64(cs.Losses) / float64(cs.Runs)
			toPct := 100.0 * float64(cs.Timeouts) / float64(cs.Runs)
			sb.WriteString(fmt.Sprintf("    %-18s %6d %5.1f%% %5.1f%% %7.1f%%   %s\n",
				cs.Name, cs.Runs, winPct, lossPct, toPct, fmtDuration(cs.AvgDur)))
		}
		sb.WriteString("\n")
	}

	// Main metrics table
	sb.WriteString(fmt.Sprintf("  %-20s %-14s %-14s %s\n", "Metric", "Result", "Spec", ""))
	sb.WriteString(fmt.Sprintf("  %s\n", strings.Repeat("─", 58)))
	for _, l := range r.Lines {
		sb.WriteString(fmt.Sprintf("  %-20s %-14s %-14s [%s]\n", l.label, l.result, l.spec, statusIcon(l.pass)))
	}

	// Class balance
	if len(r.ClassBalance) > 0 {
		sb.WriteString(fmt.Sprintf("\n  %s\n", strings.Repeat("─", 58)))
		h := r.ClassBalance[0]
		sb.WriteString(fmt.Sprintf("  Class Balance     sigma=%-8s max=%-8s [%s]\n",
			h.result, h.spec, statusIcon(h.pass)))
		for _, l := range r.ClassBalance[1:] {
			sb.WriteString(fmt.Sprintf("    %-14s %s\n", l.label, l.result))
		}
	}

	// Ability stats (overall)
	if len(r.AbilityStats) > 0 {
		sb.WriteString(fmt.Sprintf("\n  %s\n", strings.Repeat("─", 58)))
		sb.WriteString(fmt.Sprintf("  Boss Damage (total: %.0f)\n", r.TotalBossDamage))
		for _, l := range r.AbilityStats {
			sb.WriteString(fmt.Sprintf("    %-14s %s [%s]\n", l.label, l.result, statusIcon(l.pass)))
		}
	}

	// Per-composition boss damage + class balance
	if len(r.CompDetails) > 0 {
		sb.WriteString(fmt.Sprintf("\n  %s\n", strings.Repeat("─", 58)))
		sb.WriteString("  Per-Composition Breakdown\n")
		for _, cd := range r.CompDetails {
			sb.WriteString(fmt.Sprintf("\n    %s (boss dmg: %.0f)\n", cd.Name, cd.TotalBossDamage))
			// Ability damage shares
			sb.WriteString("      ability:  ")
			for i, a := range cd.AbilityShares {
				if i > 0 {
					sb.WriteString("  ")
				}
				sb.WriteString(fmt.Sprintf("%s %.1f%%", a.Name, a.Share))
			}
			sb.WriteString("\n")
			// Class balance
			sb.WriteString("      class:    ")
			for i, c := range cd.ClassShares {
				if i > 0 {
					sb.WriteString("  ")
				}
				sb.WriteString(fmt.Sprintf("%s %.1f%%", c.Name, c.Share))
			}
			sb.WriteString("\n")
		}
	}

	// Tree health
	if len(r.TreeHealth) > 0 {
		sb.WriteString(fmt.Sprintf("\n  %s\n", strings.Repeat("─", 58)))
		sb.WriteString("  Tree Health\n")
		for _, l := range r.TreeHealth {
			sb.WriteString(fmt.Sprintf("    %-14s %-24s [%s]\n", l.label, l.result, statusIcon(l.pass)))
		}
		if len(r.DeadNodes) > 0 {
			sb.WriteString("\n    dead:\n")
			for _, name := range r.DeadNodes {
				sb.WriteString(fmt.Sprintf("      %s\n", name))
			}
		}
		if len(r.ColdNodes) > 0 {
			sb.WriteString("\n    cold:\n")
			for _, name := range r.ColdNodes {
				sb.WriteString(fmt.Sprintf("      %s\n", name))
			}
		}
	}

	sb.WriteString(fmt.Sprintf("\n%s\n", strings.Repeat("━", 64)))
	t.Log(sb.String())
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

	// Compute outcome distribution + duration stats
	var minDur, maxDur, totalDur time.Duration
	for i, r := range fr.Results {
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
	if len(fr.Results) > 0 {
		report.DurationAvg = totalDur / time.Duration(len(fr.Results))
	}
	report.DurationMin = minDur
	report.DurationMax = maxDur

	// Compute per-composition breakdown
	compMap := make(map[string]*compStats)
	var compOrder []string
	for _, r := range fr.Results {
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
	sort.Slice(report.CompBreakdown, func(i, j int) bool {
		wi := float64(report.CompBreakdown[i].Wins) / float64(report.CompBreakdown[i].Runs)
		wj := float64(report.CompBreakdown[j].Wins) / float64(report.CompBreakdown[j].Runs)
		return wi > wj
	})

	t.Run("win_rate", func(t *testing.T) { fr.assertWinRate(t, report) })
	t.Run("duration", func(t *testing.T) { fr.assertDuration(t, report) })
	for _, ps := range fr.Spec.PhaseReach {
		ps := ps
		t.Run("phase_reach/"+itoa(ps.Phase), func(t *testing.T) {
			fr.assertPhaseReach(t, ps, report)
		})
	}
	if fr.Spec.ClassBalance != nil {
		t.Run("class_balance", func(t *testing.T) { fr.assertClassBalance(t, report) })
	}
	if len(fr.Spec.AbilityStats) > 0 {
		t.Run("ability_stats", func(t *testing.T) {
			fr.assertAbilityStats(t, fr.Spec.AbilityStats, report)
		})
	}

	// Compute per-composition ability damage and class balance (informational).
	fr.computePerCompDetails(report)

	return report
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
		detail := compDetailReport{Name: cs.Name}

		// Ability damage aggregation.
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
		sort.Slice(detail.AbilityShares, func(i, j int) bool {
			return detail.AbilityShares[i].RawDmg > detail.AbilityShares[j].RawDmg
		})

		// Class damage aggregation.
		classDmg := make(map[string]float64)
		var totalClassDmg float64
		for _, r := range results {
			for class, d := range r.ClassDamage {
				classDmg[class] += float64(d)
				totalClassDmg += float64(d)
			}
		}
		classes := make([]string, 0, len(classDmg))
		for class := range classDmg {
			classes = append(classes, class)
		}
		sort.Strings(classes)
		for _, class := range classes {
			var pct float64
			if totalClassDmg > 0 {
				pct = classDmg[class] / totalClassDmg * 100
			}
			detail.ClassShares = append(detail.ClassShares, nameShare{Name: class, Share: pct})
		}

		report.CompDetails = append(report.CompDetails, detail)
	}
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
		for _, p := range r.PhasesReached {
			if p == ps.Phase {
				reached++
				break
			}
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

func (fr *FuzzResults) assertClassBalance(t *testing.T, report *FuzzReport) {
	t.Helper()
	totalByClass := make(map[string]float64)
	var grandTotal float64
	for _, r := range fr.Results {
		for class, dmg := range r.ClassDamage {
			totalByClass[class] += float64(dmg)
			grandTotal += float64(dmg)
		}
	}
	if grandTotal == 0 || len(totalByClass) < 2 {
		return
	}

	shares := make([]float64, 0, len(totalByClass))
	for _, dmg := range totalByClass {
		shares = append(shares, dmg/grandTotal)
	}

	mean := 1.0 / float64(len(shares))
	var variance float64
	for _, s := range shares {
		diff := s - mean
		variance += diff * diff
	}
	sigma := math.Sqrt(variance / float64(len(shares)))

	maxSigma := fr.Spec.ClassBalance.MaxDamageShareSigma
	pass := maxSigma <= 0 || sigma <= maxSigma
	if !pass {
		t.Errorf("class damage share sigma = %.4f, want <= %.4f", sigma, maxSigma)
	}

	// Header line
	report.ClassBalance = append(report.ClassBalance, reportLine{
		result: fmt.Sprintf("%.4f", sigma),
		spec:   fmt.Sprintf("%.2f", maxSigma),
		pass:   pass,
	})
	// Per-class lines (sorted for stable output)
	classes := make([]string, 0, len(totalByClass))
	for class := range totalByClass {
		classes = append(classes, class)
	}
	sort.Strings(classes)
	for _, class := range classes {
		dmg := totalByClass[class]
		report.ClassBalance = append(report.ClassBalance, reportLine{
			label:  class,
			result: fmt.Sprintf("%.1f%%", (dmg/grandTotal)*100),
		})
	}
}

func (fr *FuzzResults) assertAbilityStats(t *testing.T, specs []AbilitySpec, report *FuzzReport) {
	t.Helper()
	agg := make(map[string]*AbilityResult)
	for _, r := range fr.Results {
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
		})
	}

	// Unspec'd abilities: show damage stats for visibility (always pass)
	var unspecced []string
	for name := range agg {
		if !specNames[name] {
			unspecced = append(unspecced, name)
		}
	}
	sort.Strings(unspecced)
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

	// Sort ability stats by damage descending for readability.
	sort.Slice(report.AbilityStats, func(i, j int) bool {
		return report.AbilityStats[i].dmg > report.AbilityStats[j].dmg
	})
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
	sb.WriteString(fmt.Sprintf("\n%s\n", strings.Repeat("━", 64)))
	sb.WriteString(fmt.Sprintf("  %s REPORT: %s (%d/%d passed)\n",
		strings.ToUpper(tr.Tier), tr.Boss, passed, len(tr.Cases)))
	sb.WriteString(fmt.Sprintf("%s\n\n", strings.Repeat("━", 64)))

	for _, c := range tr.Cases {
		sb.WriteString(fmt.Sprintf("  [%s] %s\n", statusIcon(c.Passed), c.Name))
		if c.Setup != "" {
			sb.WriteString(fmt.Sprintf("        setup:  %s\n", c.Setup))
		}
		sb.WriteString(fmt.Sprintf("        assert: %s\n", c.Assert))
		if c.Detail != "" {
			sb.WriteString(fmt.Sprintf("        result: %s\n", c.Detail))
		}
	}

	sb.WriteString(fmt.Sprintf("\n%s\n", strings.Repeat("━", 64)))
	t.Log(sb.String())
}
