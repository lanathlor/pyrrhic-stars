# AI Testing & Balance Validation

Testing framework for the AI and encounter systems. Same pipeline for fuzz simulations and live play.

Related: [AI & Encounter System](ai.md) for BT and encounter format, [Combat Logging & Observer](combat_logs.md) for event persistence.

---

## 1. Three Test Modes

| Mode               | What it Tests                        | Speed            | When to Run        |
| ------------------ | ------------------------------------ | ---------------- | ------------------ |
| State injection    | Every node reachable, no dead leaves | us per node      | Every commit       |
| Scripted scenarios | Specific behavior correctness (TDD)  | us per scenario  | Every commit       |
| Full simulation    | Integration, balance, tree coverage  | ~100ms per fight | PR merge + nightly |

---

## 2. State Injection Tests

Fabricate exact snapshots that trigger each node. No fight simulation.

```go
func TestCoverage_HollowSovereign(t *testing.T) {
    def := loadEncounter(t, "encounters/hollow_sovereign.yaml")
    tree := mustBuildTree(t, def)

    for _, node := range tree.AllConditionNodes() {
        t.Run("reachable/"+node.Name, func(t *testing.T) {
            ctx := snapshotForCondition(node.Name, true)
            ancestors := tree.AncestorConditions(node)
            for _, a := range ancestors {
                ctx = ctx.With(conditionSatisfied(a.Name))
            }
            instrumented := instrument(tree)
            instrumented.Tick(ctx)
            assert.True(t, instrumented.Node(node.Name).WasEvaluated())
        })
    }
}
```

---

## 3. Scripted Scenarios (TDD)

Set up a specific moment, tick once, assert behavior.

```go
func TestHollowSovereign_PunishesStacking(t *testing.T) {
    boss := spawnBoss(t, loadEncounter(t, "encounters/hollow_sovereign.yaml"))
    sim := NewScenario(boss).
        WithPhase("unraveling").
        WithCooldownReady("bullet_hell").
        WithPlayers(
            FakePlayer{Pos: Vec3{20, 0, 20}, Class: "gunner"},
            FakePlayer{Pos: Vec3{20, 0, 21}, Class: "arcanotechnicien"},
            FakePlayer{Pos: Vec3{21, 0, 20}, Class: "vanguard"},
        ).Build()

    result := sim.TickOnce()
    assert.Equal(t, "radial_explosion", result.PatternCast)
}
```

---

## 4. Full Simulation (Fuzz)

### 4.1 Player Profiles

Stochastic puppets, not BTs. These are stimuli, not things being tested. No tree, no node coverage tracking for players.

```go
type PlayerProfile struct {
    Name        string
    DodgeSkill  float64  // 0.0-1.0
    PotionSkill float64
    KiteSkill   float64
    DPS         float64
}

var (
    SweatyPlayer  = PlayerProfile{Name: "sweaty",  DodgeSkill: 0.9, PotionSkill: 0.9, KiteSkill: 0.8, DPS: 12}
    AveragePlayer = PlayerProfile{Name: "average", DodgeSkill: 0.4, PotionSkill: 0.6, KiteSkill: 0.3, DPS: 10}
    BadPlayer     = PlayerProfile{Name: "bad",     DodgeSkill: 0.1, PotionSkill: 0.2, KiteSkill: 0.0, DPS: 7}
)
```

### 4.2 Simulation Loop

```go
func (s *Simulation) RunUntilResolved(maxTicks int) SimResult {
    for tick := 0; tick < maxTicks; tick++ {
        s.buildSnapshots()
        s.boss.tree.Tick(s.boss.ctx)
        for _, bot := range s.party {
            bot.profile.Tick(bot.entity, s.boss.entity, bot.rng)
        }
        s.physics.Step()
        // log every event to combat log (same pipeline as live)
        if s.boss.IsDead() { return SimResult{Outcome: PlayerWin, Ticks: tick} }
        if s.AllPlayersDead() { return SimResult{Outcome: BossWin, Ticks: tick} }
    }
    return SimResult{Outcome: Timeout, Ticks: maxTicks}
}
```

No rendering, physics-lite. A single fight completes in ~50-100ms. 1000 runs in under 2 minutes.

### 4.3 Instrumented Tree

```go
type InstrumentedNode struct {
    inner        Node
    name         string
    evalCount    atomic.Int64
    successCount atomic.Int64
    failCount    atomic.Int64
    runningCount atomic.Int64
}
```

Node classifications after fuzzing:

| Classification | Criteria               | Meaning                              |
| -------------- | ---------------------- | ------------------------------------ |
| Dead           | 0 evaluations          | Structurally unreachable in practice |
| Cold           | Evaluated, 0 successes | Condition never met                  |
| Hot            | >90% tick hit rate     | Performance concern or dominant path |
| Rare           | <1% hit rate           | Edge case (may be fine)              |
| Healthy        | Normal                 | Working as intended                  |

### 4.4 Fuzz Test Structure

The test runs fights and logs to DB. No inline analysis.

```go
func TestFuzz_HollowSovereign(t *testing.T) {
    db := testDB(t)
    logger := combatlog.NewLogger(db)
    defer logger.Close()
    groupID := fmt.Sprintf("fuzz_%s_%d", t.Name(), time.Now().Unix())

    def := loadEncounter(t, "encounters/hollow_sovereign.yaml")
    spec := loadSpec(t, "encounters/hollow_sovereign.spec.yaml")
    tree := mustBuildInstrumentedTree(t, def)

    for i := 0; i < flagFuzzIterations(); i++ {
        rng := rand.New(rand.NewSource(int64(i)))
        party := generateFuzzParty(rng, spec.PartySize)
        instanceID := fmt.Sprintf("%s_%d", groupID, i)

        sim := NewSimulation(def, party, SimOpts{
            Logger: logger, GroupID: groupID,
            InstanceID: instanceID, Tree: tree, Rng: rng,
        })
        result := sim.RunUntilResolved(12000)
        logger.LogInstance(InstanceLog{...})
    }
    logger.Flush()

    // assert against spec (all queries hit DB)
    asserter := spec.NewAssert(spec, combatlog.NewRepo(db))
    asserter.Run(t, groupID)
    asserter.AssertTreeHealth(t, groupID, tree.Report())

    t.Logf("analyze: http://localhost:3000/group/%s", groupID)
}
```

---

## 5. Encounter Spec System

### 5.1 Spec File

Lives alongside the encounter YAML. Defines expected balance values. Tests fail the build if balance drifts.

```yaml
encounter: hollow_sovereign
party_size: 5

outcomes:
    win_rate:
        min: 0.30
        max: 0.65
    timeout_rate:
        max: 0.02

outcomes_by_profile:
    all_sweaty:
        win_rate: { min: 0.75, max: 0.95 }
    mixed_average:
        win_rate: { min: 0.25, max: 0.55 }
    all_floor_tank:
        win_rate: { max: 0.15 }

duration:
    avg_seconds: { min: 180, max: 600 }
    p95_seconds: { max: 720 }

phases:
    duelist:
        reach_rate: { min: 1.0 }
    unraveling:
        reach_rate: { min: 0.70 }
    hollow:
        reach_rate: { min: 0.40, max: 0.85 }

class_balance:
    max_sigma: 2.0
    max_damage_share: 0.30
    overrides:
        vanguard: { exclude_from_dps_balance: true }
        tutelaire: { exclude_from_dps_balance: true }

abilities:
    void_spiral:
        kill_share: { max: 0.40 }
        dodge_rate_by_profile:
            sweaty: { min: 0.70 }
            average: { min: 0.20 }
    melee_combo:
        kill_share: { max: 0.30 }

tree:
    dead_nodes: { max: 0 }
    cold_nodes: { max: 3 }
```

### 5.2 Assertion Layer

Queries the database (same path as the REST API), not the simulation directly. Assertions are standard Go subtests:

```
TestFuzz_HollowSovereign/outcomes/win_rate
TestFuzz_HollowSovereign/outcomes_by_profile/all_sweaty
TestFuzz_HollowSovereign/class_balance/sigma/gunner
TestFuzz_HollowSovereign/abilities/void_spiral/kill_share
TestFuzz_HollowSovereign/abilities/void_spiral/dodge_rate/sweaty
TestFuzz_HollowSovereign/tree/dead_nodes
```

### 5.3 Auto-Generated Suggestions

The report engine flags problems automatically:

- Class DPS > 1.5s from mean: "Consider a boss mechanic that disrupts {class}"
- Ability kill share > 50%: "May be overtuned or counterplay unclear"
- Phase reached < 10%: "Most players will never see this content"
- All sweaty win rate < 50%: "Boss is likely overtuned"
- All average win rate > 80%: "Boss may be undertuned"

---

## 6. Selective Testing

```go
var (
    flagOnly    = flag.String("only", "", "comma-separated entity names")
    flagChanged = flag.String("changed", "", "git base branch to diff against")
    flagTier    = flag.Int("tier", 0, "only test entities of this tier")
)
```

When a shared leaf changes, the test runner walks all YAML files to find which trees reference it. Only those entities are tested.

```bash
go test ./tests/... -changed=main            # only edited/new entities
go test ./tests/... -only=hollow_sovereign   # one boss
go test ./tests/... -tier=3                  # all bosses
go test ./tests/...                          # everything (CI full suite)
```

---

## 7. CI Pipeline

```yaml
jobs:
    ai-quick:
        name: "AI: changed entities"
        runs-on: ubuntu-latest
        steps:
            - uses: actions/checkout@v4
              with: { fetch-depth: 0 }
            - run: go test ./tests/... -changed=origin/main -v

    ai-full:
        name: "AI: full suite"
        if: github.ref == 'refs/heads/main'
        runs-on: ubuntu-latest
        services:
            clickhouse: { image: "clickhouse/clickhouse-server:latest" }
        steps:
            - uses: actions/checkout@v4
            - run: go test ./tests/... -fuzz-iterations=5000 -v
```
