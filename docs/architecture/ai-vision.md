# AI System — Long-term Vision

Systems planned for Phase 2 and beyond. For current architecture, see [AI & Encounter System](ai.md). For testing, see [AI Testing & Balance](testing.md).

---

## 1. Extended Entity Context API

The [AI & Encounter System](ai.md) defines a focused EntityContext with ~40 methods for Phase 0-1. This section documents the full API planned for later phases: minion management, multi-entity bosses, inter-entity communication, environment queries, and analysis utilities.

### 1.1 Minions — "my underlings"

Full minion management beyond the basic `Spawn` / `SpawnWave` / `MinionsAliveCount` in the core API.

```
ctx.SpawnNearSelf(mobID, radius?) Entity
ctx.SpawnAtTarget(mobID, target, offset?) Entity
ctx.SpawnWave(mobID, count, pattern, params?) []Entity
    // patterns: "circle_around_self", "circle_around_target",
    //           "line_between", "random_in_radius", "at_positions"

ctx.Minions() []Entity
ctx.MinionsAlive() []Entity
ctx.MinionsOfType(mobID) []Entity
ctx.MinionDiedThisTick() Entity
ctx.MinionsInRadius(radius) []Entity

// commands (override minion's BT temporarily)
ctx.Command(entity, commandID, params?)
ctx.CommandAll(commandID, params?)
ctx.CommandGroup(entities, commandID, params?)
ctx.Release(entity)
ctx.ReleaseAll()

// command vocabulary:
//   "focus_target", "move_to_position", "tether_to",
//   "sacrifice", "shield_formation", "detonate"

// aggregates (from snapshot, cached)
ctx.MinionsCentroid() Vec3
ctx.MinionsHealthPctAvg() float32
ctx.MinionsFormationReady(formationID) bool
```

### 1.2 Siblings — "my peers" (multi-entity bosses)

For boss encounters with 2-3 distinct peer entities (e.g., twin bosses). No hierarchy; each runs its own independent BT.

```
ctx.HasSiblings() bool
ctx.Siblings() []Entity
ctx.Sibling(id) Entity
ctx.SiblingsAlive() []Entity
ctx.SiblingsAliveCount() int
ctx.SiblingDiedThisTick() Entity

// deep read (from snapshot, read-only)
ctx.SiblingHealthPct(id) float32
ctx.SiblingPhase(id) string
ctx.SiblingTarget(id) Entity
ctx.SiblingIsChanneling(id) bool
ctx.SiblingHasFlag(id, key) bool  // read their flags, never set
ctx.SiblingPosition(id) Vec3
```

### 1.3 Messages — "suggestions between peers"

Fire-and-forget. Receiver can ignore. Messages expire after N ticks (default 5, ~250ms at 20Hz). No ack system.

```
ctx.Send(siblingID, messageType, data?)
ctx.Broadcast(messageType, data?)

ctx.Messages() []Message
ctx.MessagesOfType(type) []Message
ctx.HasMessage(type) bool
ctx.ConsumeMessage(type) *Message  // pops, won't trigger again
```

### 1.4 Group State — "shared encounter blackboard"

Any sibling can read or write. Use sparingly; most coordination goes through messages.

```
ctx.GroupGet(key) any
ctx.GroupSet(key, value)
ctx.GroupIncrement(key, amount?)
```

### 1.5 Environment — "the arena"

```
ctx.ArenaCenter() Vec3
ctx.ArenaRadius() float32
ctx.ArenaBounds() Bounds
ctx.IsInArena(pos Vec3) bool
```

### 1.6 Communication — "theatrical moments"

```
ctx.Say(text)
ctx.Emote(emoteID)
ctx.ChangeMusic(trackID)
ctx.ScreenShake(intensity, duration)
ctx.CameraFocus(entity, duration)
```

### 1.7 Utilities — "analysis helpers"

All computed lazily from the snapshot. First call calculates, subsequent calls in same tick return cached result.

```
ctx.ThreatsByClass() map[string][]ThreatEntry
ctx.ThreatsInRange(range) []Entity
ctx.ThreatsOutsideRange(range) []Entity

ctx.TicksToReach(entity) int      // distance / speed * tickRate (no pathfinding)
ctx.TicksToReachPos(pos) int
ctx.CentroidOfThreats() Vec3
ctx.ThreatsClustered(radius) []Cluster
ctx.MostIsolatedThreat() Entity

ctx.ThreatsChanneling() []Entity
ctx.ThreatsBelowHealth(pct) []Entity
ctx.ThreatsWithBuff(id) []Entity
ctx.ThreatsTargetingMe() []Entity

ctx.DangerScore() float32         // heuristic: incoming damage pressure
ctx.IsSurrounded(threshold?) bool
ctx.OpenFlankDirection() *float32
```

### 1.8 Expensive Query Caching

Queries that hit the navmesh (pathfinding, cover, flank positions) use TTL caching. The BT author doesn't know about the cache; the function always returns a result.

```go
type ExpensiveQueryCache struct {
    results map[string]cachedResult
}

type cachedResult struct {
    value    any
    validFor int // ticks remaining
}
```

Default TTLs: pathfinding 250ms (5 ticks at 20Hz), cover positions 500ms, LOS checks 100ms.

### 1.9 Extended Movement

```
ctx.Strafe(direction, distance)
ctx.FaceDirection(angle)
ctx.IsMoving() bool
ctx.NearestCoverFrom(entity) *Vec3
ctx.FlankPosition(entity) *Vec3
ctx.RetreatPosition() *Vec3
ctx.IsAtPosition(pos, tolerance) bool
```

### 1.10 Extended Combat

```
ctx.CommitPatternDelayed(patternID, delay) PatternHandle
ctx.ModifyActivePattern(params)
ctx.ActivePatternCount() int
ctx.CancelPattern(handle)
ctx.CancelAllPatterns()

ctx.ExecuteCombo(attackList) Result
ctx.CancelCombo()

ctx.AvailableAttacks() []AttackInfo
ctx.StartCooldown(id, duration)
```

### 1.11 Extended Self

```
ctx.HasBuff(id) bool
ctx.HasDebuff(id) bool
ctx.IsInCombo() bool
ctx.IsImmune() bool
ctx.SetImmune(duration)
```

### 1.12 Extended Perception

```
ctx.AlliesInRadius(radius) []Entity
ctx.AliveAllies() []Entity
ctx.AliveAlliesCount() int
ctx.IsBehind(entity) bool         // backstab angle check
ctx.AngleTo(entity) float32
```

### 1.13 Extended Entity Reference

When the full API is available, entity references returned from queries gain additional methods:

```
entity.RoleTag() string           // "dps", "tank", "healer", "support"
entity.Facing() float32
entity.CommitID() string
entity.CommitRemaining() float32
entity.HasBuff(id) bool
entity.IsLockedOn(target) bool    // hybrid targeting
entity.IsAimingAt(target, cone) bool
```

---

## 2. Open Source Library Extraction

### 2.1 Two Libraries

#### `bt` — Pure BT library, zero game knowledge

```
github.com/user/bt/
  node.go           # Node interface, Result type
  selector.go       # Selector, Sequence, Parallel, Decorator
  registry.go       # LeafRegistry (string -> function)
  yaml.go           # BuildFromYAML
  instrument.go     # InstrumentedNode wrapper
  coverage.go       # CoverageReport, NodeInfo, classifications
```

~500 lines. Any domain: games, robotics, dialog, trading.

#### `bttest` — Testing toolkit, game-agnostic via generics

```
github.com/user/bttest/
  simulation.go     # Simulation[Ctx, Outcome] interface
  fuzz.go           # Fuzz() runner
  spec.go           # Spec[Outcome], RangeSpec, Assert()
  report.go         # FuzzResult[Outcome]
```

~300 lines.

### 2.2 Interface Contract

```go
// bt/node.go
type Node interface {
    Tick(ctx any) Result
}

// bttest/simulation.go
type Simulation[Ctx any, Outcome any] interface {
    Setup(rng *rand.Rand) Ctx
    Tick(ctx Ctx) Ctx
    Resolved(ctx Ctx) (bool, Outcome)
    MaxTicks() int
}
```

### 2.3 Example: Dungeon Crawler

Single example ships with the libraries. One room, one hero, one mob, 1D positioning.

```
examples/dungeon/
  main_test.go       # matrix test: mob tree x player skill
  entity.go          # ~50 lines
  simulation.go      # ~80 lines
  player.go          # PlayerProfile (stochastic)
  trees/
    smart_goblin.yaml
    dumb_goblin.yaml
  specs/
    smart_goblin_sweaty.spec.yaml
    smart_goblin_average.spec.yaml
  leaves/
    conditions.go    # ~40 lines
    actions.go       # ~40 lines
```

The test runs a 2x3 matrix (smart/dumb goblin x sweaty/average/bad player) and outputs coverage per matchup. The dumb goblin has dead nodes and lower win rates. The smart goblin adapts -- its coverage shifts with player skill level. The spec catches regressions.

Build inside the game first. Extract when 2-3 encounters prove the API is stable.

---

## 3. ML Bot Training System

### 3.1 Overview

Train RL agents to serve as bot players in dungeons and raids. The same simulation loop serves both fuzz testing and ML training.

### 3.2 Hardware

- **Training:** AMD Radeon RX 7900 XTX, ROCm + PyTorch, Linux.
- **Inference:** CPU only via ONNX Runtime in Go. No GPU in production.

### 3.3 Architecture

```
Go Simulation (headless, fast)
    |
    | gRPC: Reset() -> Observation
    |        Step(action) -> (Observation, Reward, Done)
    | parallel: 16-64 environments simultaneously
    |
    v
Python Training (PyTorch + ROCm)
    |
    | Phase 1: Imitation learning from combat logs
    | Phase 2: PPO self-play against boss BTs
    | Phase 3: Population training for style diversity
    |
    v
ONNX Export (~50-500KB model file)
    |
    v
Go Inference (ONNX Runtime, us per decision, no GPU)
```

### 3.4 Network Architecture

```python
class PlayerPolicy(nn.Module):
    def __init__(self, obs_dim, action_dim):
        # entity encoder: variable number of entities
        self.entity_encoder = nn.TransformerEncoder(
            nn.TransformerEncoderLayer(d_model=64, nhead=4),
            num_layers=2
        )
        # temporal memory
        self.lstm = nn.LSTM(input_size=64, hidden_size=128, num_layers=2)
        # action head
        self.action_head = nn.Sequential(
            nn.Linear(128, 64), nn.ReLU(), nn.Linear(64, action_dim)
        )
        # target selection (pointer network)
        self.target_head = PointerNetwork(128, 64)
```

Model size: 1-5M parameters. Trains in hours on 7900XTX, not days.

### 3.5 Observation Space

```go
type Observation struct {
    SelfHealthPct     float32
    SelfFluxPct       float32
    SelfPosition      [3]float32  // Vec3
    SelfFacing        float32
    SelfCooldowns     [8]float32

    BossHealthPct     float32
    BossPosition      [3]float32  // Vec3
    BossFacing        float32
    BossIsChanneling   float32
    BossCommitID       int
    BossCommitProgress float32
    BossPhase         int

    Allies            [4]AllyObs
    Projectiles       [32]ProjectileObs
    SelfThreatRank    int
}
```

### 3.6 Reward Design

Hierarchical, with mechanic-specific signals auto-generated from encounter YAML.

```
Layer 1: Personal survival
  - damage_taken * -0.005
  - damage_dealt * +0.003

Layer 2: Role fulfillment (role-specific)
  Tank:   projectiles_intercepted * +0.1, boss_facing_away_from_raid * +0.02
  Healer: effective_healing * +0.005, ally_died_with_cooldowns_available * -0.5
  DPS:    damage_dealt * +0.008, has_boss_aggro * -0.1

Layer 3: Mechanics (auto-generated from encounter YAML)
  Soak:   soaking +0.3/tick, could_soak_but_didnt -0.15, failed -1.0
  Spread: too_close -0.2, correct_distance +0.05
  Stack:  in_stack +0.1, out_of_stack -0.1

Layer 4: Team outcome
  Boss died: +2.0, Raid wiped: -2.0
```

Mechanic rewards are auto-generated from the encounter YAML:

```go
func MechanicRewards(encounter EncounterDef) []RewardModifier {
    for _, phase := range encounter.Phases {
        for _, mech := range phase.Mechanics {
            switch mech.Type {
            case "soak":
                // auto-generate soak reward with configured thresholds
            case "spread":
                // auto-generate spread reward
            case "tank_buster":
                // auto-generate tank buster reward
            }
        }
    }
}
```

### 3.7 Cooperative Training

Role-conditioned policies trained together:

```python
policies = {
    "tank":   TankPolicy(obs_dim, tank_action_dim),
    "healer": HealerPolicy(obs_dim, healer_action_dim),
    "melee":  MeleeDPSPolicy(obs_dim, melee_action_dim),
    "ranged": RangedDPSPolicy(obs_dim, ranged_action_dim),
}

# each episode: assemble full party, train all policies together
# policies co-adapt: tank learns to soak, healer learns cooldown timing
```

**Centralized Training, Decentralized Execution (CTDE):**

- During training: centralized critic sees all agents' observations and actions. Solves credit assignment.
- During inference: each agent only sees its own observation. No global state needed in production.

**Communication channel:** each agent outputs an 8-float message vector per tick, visible to allies next tick. Agents learn their own coordination protocol (e.g., "I'm soaking, don't come").

### 3.8 Deployment in Go

```go
import ort "github.com/yalue/onnxruntime_go"

type MLBrain struct {
    session *ort.Session
}

func (b *MLBrain) Decide(obs Observation) Action {
    input := obs.ToFloat32Slice()
    output, _ := b.session.Run(input)
    return Action(argmax(output))
}
```

Humanization layer degrades perfect play to feel human:

```go
type HumanizedBrain struct {
    ml            *MLBrain
    reactionDelay int       // ticks before acting
    mistakeRate   float32   // probability of random action
    tunnelVision  float32   // probability of ignoring a threat
}
```

Train ML to play perfectly, then degrade with human-like imperfections. A "good" bot: `reactionDelay: 2, mistakeRate: 0.02`. An "average" bot: `reactionDelay: 5, mistakeRate: 0.08`.

---

## 4. Content Release Lifecycle

### 4.1 Tier System

Content is organized into tiers that release sequentially:

```
content/ (private repo)
  encounters/tier1/
  encounters/tier2/  (unreleased)
  abilities/tier1/
  abilities/tier2/
  leaves/tier1/
  leaves/tier2/
```

### 4.2 World Race Protection

```
Tier development (private repo):
  encounters/tierN/, leaves/tierN/, abilities/tierN/

Tier goes live on server:
  Compiled binary from private repo. Players experience encounters blind.

World first claimed:
  Run release-tier.sh tierN
  Copies finished files to public repo (no git history leak).
  Community can study encounter design and create custom content.
```

### 4.3 Release Script

```bash
#!/bin/bash
TIER=$1
cp -r "$PRIVATE_REPO/encounters/$TIER" "$PUBLIC_REPO/encounters/$TIER"
cp -r "$PRIVATE_REPO/abilities/$TIER" "$PUBLIC_REPO/abilities/$TIER"
cp -r "$PRIVATE_REPO/leaves/$TIER" "$PUBLIC_REPO/leaves/$TIER"
cd "$PUBLIC_REPO"
git add . && git commit -m "Release $TIER" && git tag "release-$TIER"
git push origin main --tags
```

Private repo stays private forever. Public repo receives snapshots.
