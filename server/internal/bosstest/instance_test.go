package bosstest_test

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"codex-online/server/internal/bosstest"
	"codex-online/server/internal/combatlog"
	"codex-online/server/internal/overflux"
)

// instanceSpec is the YAML schema for a full-dungeon fuzz spec
// (testdata/instances/*.yaml). Unlike boss specs, a run covers the whole
// instance: lobby ready-up, trash packs, both bosses, gates, deaths,
// checkpoint respawns and wipes.
type instanceSpec struct {
	Level               string         `yaml:"level"`
	Runs                int            `yaml:"runs"`
	RespawnDelaySeconds float32        `yaml:"respawn_delay_seconds"`
	Compositions        []instanceComp `yaml:"compositions"`

	// Overflux difficulty configs: each entry re-runs every composition under
	// those conditions. Per-comp bounds only apply to the baseline; each
	// overflux entry asserts its own aggregate win-rate bounds (same
	// convention as the boss fuzz specs).
	Overflux []instanceOverflux `yaml:"overflux"`
}

// instanceOverflux extends the shared overflux config with instance-level
// ordering assertions.
type instanceOverflux struct {
	bosstest.OverfluxSpec `yaml:",inline"`

	// KeepsLethalityOf names another overflux config whose boss-1 lethality
	// (deaths per run in the boss-1 section) this one must retain at least
	// lethalityFloor of. Guards config stacking that accidentally DILUTES
	// difficulty (regression 2026-07: the tempered tree starved its pattern
	// casts, so tempered+volatile — volatile being a pure pattern buff —
	// dropped to 43% of volatile's boss-1 deaths). Deaths are a continuous
	// per-run statistic, far more stable at spec run counts than win-rate
	// deltas (which swing ±9pts on 40-run binary aggregates).
	KeepsLethalityOf string `yaml:"keeps_lethality_of"`
}

// lethalityFloor is the minimum fraction of the reference config's boss-1
// deaths/run a stacked config must retain. Healthy stacking measures 0.8-0.9;
// the starved-pattern regression measured 0.43.
const lethalityFloor = 0.55

type instanceComp struct {
	Name      string   `yaml:"name"`
	Classes   []string `yaml:"classes"`
	Specs     []string `yaml:"specs"`
	Profiles  []string `yaml:"profiles"`
	ChainPull bool     `yaml:"chain_pull"`

	WinRate         *instanceBound `yaml:"win_rate"`
	AvgClearSeconds *instanceBound `yaml:"avg_clear_seconds"` // over cleared runs
	AvgDeaths       *instanceBound `yaml:"avg_deaths"`        // per run
	AvgWipes        *instanceBound `yaml:"avg_wipes"`         // per run
}

type instanceBound struct {
	Min *float64 `yaml:"min"`
	Max *float64 `yaml:"max"`
}

func (b *instanceBound) check(t *testing.T, label string, v float64) {
	t.Helper()
	if b == nil {
		return
	}
	if b.Min != nil && v < *b.Min {
		t.Errorf("%s = %.2f, want >= %.2f", label, v, *b.Min)
	}
	if b.Max != nil && v > *b.Max {
		t.Errorf("%s = %.2f, want <= %.2f", label, v, *b.Max)
	}
}

// instanceCompStats aggregates run results for one composition.
type instanceCompStats struct {
	runs, wins, overtime int
	deaths, wipes        int
	clearSum             float32 // seconds, wins only
	clearMin, clearMax   float32

	totalTicks    int            // all runs
	downtimeTicks int            // all runs
	bossTicks     int            // ticks with a boss engaged (boss-DPS denominator)
	trashTicks    int            // ticks with trash engaged (trash-DPS denominator)
	segSum        map[string]int // segment name → summed duration ticks
	segCount      map[string]int // segment name → runs that completed it
	deathsBySec   map[string]int
	classDmg      map[string]*bosstest.ClassDamage
	commits       map[string]int
}

// instanceSegments is the fixed reporting order of content milestones.
var instanceSegments = []string{"packs", "boss1", "decline", "boss2"}

func TestInstance(t *testing.T) {
	specs, err := filepath.Glob("testdata/instances/*.yaml")
	if err != nil {
		t.Fatalf("glob instance specs: %v", err)
	}
	if len(specs) == 0 {
		t.Skip("no instance specs found")
	}
	for _, specPath := range specs {
		base := strings.TrimSuffix(filepath.Base(specPath), ".yaml")
		t.Run(base, func(t *testing.T) {
			runInstanceSpec(t, specPath)
		})
	}
}

func runInstanceSpec(t *testing.T, specPath string) {
	raw, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("read spec: %v", err)
	}
	var spec instanceSpec
	if err := yaml.Unmarshal(raw, &spec); err != nil {
		t.Fatalf("parse spec: %v", err)
	}
	if spec.Runs == 0 {
		spec.Runs = 30
	}
	respawnTicks := int(spec.RespawnDelaySeconds * 20)

	var report strings.Builder
	fmt.Fprintf(&report, "\n━━━ INSTANCE REPORT: %s (%d runs/comp) ━━━\n", spec.Level, spec.Runs)

	// Baseline: per-comp bounds apply.
	t.Run("baseline", func(t *testing.T) {
		report.WriteString(" baseline:\n")
		for _, comp := range spec.Compositions {
			t.Run(comp.Name, func(t *testing.T) {
				st := runInstanceComp(spec, comp, nil, respawnTicks)
				reportInstanceComp(&report, comp.Name, st)
				checkInstanceComp(t, comp, st)
			})
		}
	})

	// Overflux configs: every comp re-runs under the conditions; assertion is
	// the config's aggregate win-rate bounds, plus optional cross-config
	// lethality ordering (keeps_lethality_of).
	aggBoss1Deaths := make(map[string]float64, len(spec.Overflux))
	for _, oflx := range spec.Overflux {
		t.Run(oflx.Name, func(t *testing.T) {
			fmt.Fprintf(&report, " overflux %s:\n", oflx.Name)
			state := oflx.ToOverfluxState()
			totalRuns, totalWins, boss1Deaths := 0, 0, 0
			for _, comp := range spec.Compositions {
				st := runInstanceComp(spec, comp, state, respawnTicks)
				reportInstanceComp(&report, comp.Name, st)
				totalRuns += st.runs
				totalWins += st.wins
				boss1Deaths += st.deathsBySec["boss1"]
			}
			winRate := float64(totalWins) / float64(totalRuns)
			aggBoss1Deaths[oflx.Name] = float64(boss1Deaths) / float64(totalRuns)
			fmt.Fprintf(&report, "   %-20s win %5.1f%% aggregate | boss1 deaths %.2f/run\n", oflx.Name, winRate*100, aggBoss1Deaths[oflx.Name])
			if oflx.WinRate != nil {
				if oflx.WinRate.Min > 0 && winRate < oflx.WinRate.Min {
					t.Errorf("%s aggregate win rate = %.3f, want >= %.3f", oflx.Name, winRate, oflx.WinRate.Min)
				}
				if oflx.WinRate.Max > 0 && winRate > oflx.WinRate.Max {
					t.Errorf("%s aggregate win rate = %.3f, want <= %.3f", oflx.Name, winRate, oflx.WinRate.Max)
				}
			}
			if oflx.KeepsLethalityOf != "" {
				ref, ok := aggBoss1Deaths[oflx.KeepsLethalityOf]
				if !ok {
					t.Fatalf("%s: keeps_lethality_of references %q, which must be listed BEFORE it", oflx.Name, oflx.KeepsLethalityOf)
				}
				if ref > 0 && aggBoss1Deaths[oflx.Name] < ref*lethalityFloor {
					t.Errorf("%s boss1 deaths = %.2f/run, want >= %.0f%% of %s (%.2f/run) — stacking conditions must not dilute difficulty",
						oflx.Name, aggBoss1Deaths[oflx.Name], lethalityFloor*100, oflx.KeepsLethalityOf, ref)
				}
			}
		})
	}

	t.Log(report.String())
}

// runInstanceComp executes spec.Runs seeded instance runs for one composition
// (optionally under overflux conditions) and aggregates the metrics.
func runInstanceComp(spec instanceSpec, comp instanceComp, oflx *overflux.State, respawnTicks int) instanceCompStats {
	party := make([]bosstest.PuppetConfig, len(comp.Classes))
	for i := range comp.Classes {
		specID := ""
		if i < len(comp.Specs) {
			specID = comp.Specs[i]
		}
		profile := "average"
		if i < len(comp.Profiles) {
			profile = comp.Profiles[i]
		}
		party[i] = bosstest.PuppetConfig{
			Class:   comp.Classes[i],
			Spec:    specID,
			Profile: bosstest.BotProfile(profile),
		}
	}

	st := instanceCompStats{
		segSum:      make(map[string]int),
		segCount:    make(map[string]int),
		deathsBySec: make(map[string]int),
		classDmg:    make(map[string]*bosstest.ClassDamage),
		commits:     make(map[string]int),
	}
	for run := 0; run < spec.Runs; run++ {
		res := bosstest.RunInstance(bosstest.InstanceConfig{
			Level:             spec.Level,
			Party:             party,
			Seed:              uint64(run*7919 + 13),
			ChainPull:         comp.ChainPull,
			RespawnDelayTicks: respawnTicks,
			Overflux:          oflx,
			PuppetTrees:       puppetTrees,
		})
		st.runs++
		st.deaths += res.Deaths
		st.wipes += res.Wipes
		st.totalTicks += res.TotalTicks
		st.downtimeTicks += res.DowntimeTicks
		st.bossTicks += res.BossEngagedTicks
		st.trashTicks += res.TrashEngagedTicks
		for sec, n := range res.DeathsBySection {
			st.deathsBySec[sec] += n
		}
		for key, cd := range res.ClassDamage {
			agg := st.classDmg[key]
			if agg == nil {
				agg = &bosstest.ClassDamage{}
				st.classDmg[key] = agg
			}
			agg.Boss += cd.Boss
			agg.Trash += cd.Trash
		}
		for key, n := range res.AbilityCommits {
			st.commits[key] += n
		}
		// Segment durations: completion-tick deltas in content order.
		prev := 0
		for _, seg := range instanceSegments {
			end, ok := res.SegmentTicks[seg]
			if !ok {
				break // never completed (timeout) — later segments neither
			}
			st.segSum[seg] += end - prev
			st.segCount[seg]++
			prev = end
		}
		if res.Outcome == combatlog.OutcomePlayerWin {
			st.wins++
			st.clearSum += res.ClearSeconds
			if st.clearMin == 0 || res.ClearSeconds < st.clearMin {
				st.clearMin = res.ClearSeconds
			}
			if res.ClearSeconds > st.clearMax {
				st.clearMax = res.ClearSeconds
			}
			if res.OverTime {
				st.overtime++
			}
		}
	}
	return st
}

func (st instanceCompStats) winRate() float64   { return float64(st.wins) / float64(st.runs) }
func (st instanceCompStats) avgDeaths() float64 { return float64(st.deaths) / float64(st.runs) }
func (st instanceCompStats) avgWipes() float64  { return float64(st.wipes) / float64(st.runs) }
func (st instanceCompStats) avgClear() float64 {
	if st.wins == 0 {
		return 0
	}
	return float64(st.clearSum) / float64(st.wins)
}
func (st instanceCompStats) overtimeRate() float64 {
	if st.wins == 0 {
		return 0
	}
	return float64(st.overtime) / float64(st.wins)
}

func reportInstanceComp(report *strings.Builder, name string, st instanceCompStats) {
	fmt.Fprintf(report,
		"   %-22s win %5.1f%% | clear avg %6.1fs [%.1f–%.1f] | overtime %4.1f%% | deaths/run %.2f | wipes/run %.2f\n",
		name, st.winRate()*100, st.avgClear(), st.clearMin, st.clearMax,
		st.overtimeRate()*100, st.avgDeaths(), st.avgWipes())

	// Time: where runs spend it (segment averages over runs that got there)
	// and how much of it is spent out of combat (travel, run-backs).
	line := "     time:  "
	var lineSb310 strings.Builder
	for _, seg := range instanceSegments {
		if n := st.segCount[seg]; n > 0 {
			lineSb310.WriteString(fmt.Sprintf("%s %.1fs → ", seg, float64(st.segSum[seg])/float64(n)*0.05))
		}
	}
	line += lineSb310.String()
	line = strings.TrimSuffix(line, " → ")
	fmt.Fprintf(report, "%s | downtime %.1fs/run\n", line, float64(st.downtimeTicks)/float64(st.runs)*0.05)

	// Deaths: which section they happen in.
	if len(st.deathsBySec) > 0 {
		secs := make([]string, 0, len(st.deathsBySec))
		for sec := range st.deathsBySec {
			secs = append(secs, sec)
		}
		sort.Slice(secs, func(i, j int) bool { return st.deathsBySec[secs[i]] > st.deathsBySec[secs[j]] })
		line = "     deaths/run: "
		var lineSb326 strings.Builder
		for _, sec := range secs {
			lineSb326.WriteString(fmt.Sprintf("%s %.2f, ", sec, float64(st.deathsBySec[sec])/float64(st.runs)))
		}
		line += lineSb326.String()
		fmt.Fprintf(report, "%s\n", strings.TrimSuffix(line, ", "))
	}

	// DPS per party slot: boss DPS over boss-engaged time, trash DPS over
	// trash-engaged time. The two are intensive rates with different
	// denominators — they deliberately don't sum to anything.
	if len(st.classDmg) > 0 {
		keys := make([]string, 0, len(st.classDmg))
		for key := range st.classDmg {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		bossSecs := float64(st.bossTicks) * 0.05
		trashSecs := float64(st.trashTicks) * 0.05
		line = "     dps:   "
		var lineSb344 strings.Builder
		for _, key := range keys {
			cd := st.classDmg[key]
			bossDPS, trashDPS := 0.0, 0.0
			if bossSecs > 0 {
				bossDPS = float64(cd.Boss) / bossSecs
			}
			if trashSecs > 0 {
				trashDPS = float64(cd.Trash) / trashSecs
			}
			lineSb344.WriteString(fmt.Sprintf("%s boss %.1f / trash %.1f, ", key, bossDPS, trashDPS))
		}
		line += lineSb344.String()
		fmt.Fprintf(report, "%s\n", strings.TrimSuffix(line, ", "))
	}

	// What the encounter cast (top commits per run, boss rotation visibility).
	if len(st.commits) > 0 {
		keys := make([]string, 0, len(st.commits))
		for key := range st.commits {
			keys = append(keys, key)
		}
		sort.Slice(keys, func(i, j int) bool { return st.commits[keys[i]] > st.commits[keys[j]] })
		if len(keys) > 8 {
			keys = keys[:8]
		}
		line = "     casts/run: "
		var lineSb369 strings.Builder
		for _, key := range keys {
			lineSb369.WriteString(fmt.Sprintf("%s %.1f, ", key, float64(st.commits[key])/float64(st.runs)))
		}
		line += lineSb369.String()
		fmt.Fprintf(report, "%s\n", strings.TrimSuffix(line, ", "))
	}
}

func checkInstanceComp(t *testing.T, comp instanceComp, st instanceCompStats) {
	t.Helper()
	comp.WinRate.check(t, comp.Name+" win rate", st.winRate())
	comp.AvgClearSeconds.check(t, comp.Name+" avg clear seconds", st.avgClear())
	comp.AvgDeaths.check(t, comp.Name+" avg deaths/run", st.avgDeaths())
	comp.AvgWipes.check(t, comp.Name+" avg wipes/run", st.avgWipes())
}
