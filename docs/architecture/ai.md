# AI & Encounter System

## 1. Design Principles

- **Go-only runtime.** All behavior logic is compiled Go with YAML-driven composition. Type safety, TDD, single-language debugging.
- **Data-driven entities.** 90% of mobs are pure YAML definitions with zero custom code. Custom Go leaf functions only for boss-specific behavior.
- **One binary.** Monolith with clean internal package boundaries.
- **Same pipeline for test and live.** Combat logging, analysis, and the simulation loop are identical in fuzz tests and production.

## 2. Package Structure

```
server/
  internal/
    enemyai/          # BT executor, snapshot builder, leaf registry (currently FSM-based Brain)
    combat/           # spell engine, damage calc, pattern engine
    combatlog/        # logger, repo, API handlers (new)
    zone/             # instancing, zone management
    entity/           # Player, Enemy, Projectile, Vec3

content/              # data files (future: separate repo for world-race protection)
  encounters/         # encounter YAML definitions
  mobs/               # mob YAML definitions
  spells/             # spell YAML definitions
  leaves/
    shared/           # shared Go leaf functions (available to all entities)
    tier1/            # entity-specific Go leaves
    tier2/
```

### Dependency Flow

```
internal/zone → internal/combat → internal/enemyai
             ↘ internal/combatlog

enemyai NEVER imports combat
combat NEVER imports zone
combatlog imports NOTHING (receives events via interface)
```

Enforced by test:

```go
func TestNoCyclicImports(t *testing.T) {
    forbidden := map[string][]string{
        "internal/enemyai":   {"internal/combat", "internal/zone"},
        "internal/combat":    {"internal/zone"},
        "internal/combatlog": {"internal/enemyai", "internal/combat", "internal/zone"},
    }
    // ... assert no forbidden imports
}
```

---

## 3. Behavior Tree System

### 3.1 Node Types

| Node          | Behavior                                                                                                 |
| ------------- | -------------------------------------------------------------------------------------------------------- |
| **Selector**  | Tries children in order. Returns SUCCESS on first child success. Returns FAILURE if all children fail.   |
| **Sequence**  | Runs children in order. Returns FAILURE on first child failure. Returns SUCCESS if all children succeed. |
| **Parallel**  | Runs all children concurrently. Configurable success/fail policy (all, any, N-of-M).                     |
| **Decorator** | Wraps a child. Variants: Inverter, Repeater, Cooldown, Chance(probability), UntilFail, UntilSuccess.     |
| **Condition** | Leaf. Evaluates a boolean. No side effects.                                                              |
| **Action**    | Leaf. Performs a game action. Returns SUCCESS, FAILURE, or RUNNING.                                      |

### 3.2 Core Interface

```go
type Node interface {
    Tick(ctx any) Result
}

type Result int
const (
    Success Result = iota
    Failure
    Running
)
```

### 3.3 Leaf Registry

Three-layer resolution for every leaf name referenced in YAML:

1. **Built-in conditions/actions** — minimal set of truly generic checks.
2. **Shared leaf library** — Go functions in `leaves/shared/`, available to all entities.
3. **Entity-specific leaves** — Go functions in the entity's own leaf file, override shared if same name.

```go
type LeafFunc func(ctx *EntityContext) Result
type ConditionFunc func(ctx *EntityContext) bool

type LeafRegistry struct {
    conditions map[string]ConditionFunc
    actions    map[string]LeafFunc
}

func (r *LeafRegistry) Resolve(name string) (any, bool) {
    if fn, ok := r.conditions[name]; ok { return fn, true }
    if fn, ok := r.actions[name]; ok { return fn, true }
    return nil, false
}
```

### 3.4 YAML Tree Definition

```yaml
mob: shadow_captain
tier: 2
tree:
    selector:
        - sequence:
              - condition: target_casting
              - condition: target_in_range(5)
              - action: punish_caster
        - sequence:
              - condition: is_flanking
              - action: aggressive_melee
        - sequence:
              - condition: target_far
              - action: close_distance
        - action: attack_weighted
```

Trees reference leaf names. The executor resolves them at load time via the registry. Unresolved names are a load-time error, caught by tests.

### 3.5 Built-in Condition Vocabulary

Kept minimal. Most conditions are shared Go leaf functions, not built-ins.

```yaml
- condition: { health_pct_lt: 0.3 }
- condition: { health_pct_gt: 0.5 }
- condition: { has_flag: empowered }
- condition: { counter_gte: { key: absorbed_adds, value: 3 } }
- condition: { timer_expired: hold_aggro }
- condition: { cooldown_ready: void_spiral }
- condition: { target_in_range: 5 }
```

### 3.6 Tiered Tick Rates

| Entity Tier        | Brain Tick Rate | Description                          |
| ------------------ | --------------- | ------------------------------------ |
| Tier 1 (data mobs) | 5 Hz            | Patrol + basic combat. Pure YAML.    |
| Tier 2 (elites)    | 10 Hz           | Shared leaf functions only.          |
| Tier 3 (bosses)    | 20 Hz           | Custom leaf functions per encounter. |

Physics and projectile systems run at full server tick rate (20 Hz) regardless of brain tick rate.

### 3.7 BT Design Rule

**Conditions read. Actions write. Trees decide. Leaves execute.**

If a leaf function contains `if/else` that changes _what_ the entity decides to do, that logic belongs in the tree structure (selector/sequence), not in the leaf. Leaves may contain `if/else` only for _how_ an action executes (e.g., picking from a random pool).

---

## 4. Entity Context

### 4.1 Architecture

`EntityContext` is a Go struct composed from focused interfaces. Pushed into the BT executor as the context for every tick.

```go
type EntityContext struct {
    self    SelfQuery
    threat  ThreatQuery
    percept Perception
    combat  CombatActions
    move    Movement
    memory  EntityMemory
    spawn   SpawnManager
}
```

### 4.2 Snapshot Model

All ctx reads come from a snapshot built once per tick. This eliminates repeated computation within the tick and makes behavior deterministic and testable.

```go
type TickSnapshot struct {
    Self    SelfState
    Threats []ThreatEntry
    Nearby  []EntityState
    Tick    int64
}
```

### 4.3 API Surface

This is the working set for Phase 0-1. The full extended API (minions, siblings, messages, group state, environment, communication, utilities) is documented in [AI Long-term Vision](ai-vision.md).

Entity IDs in the BT context use `string` rather than the current `uint16` peer IDs — the BT layer needs to address players, enemies, and minions uniformly.

#### Self — "what am I?"

```
ctx.EntityID() string
ctx.Health() float32
ctx.HealthPct() float32           // 0.0–1.0
ctx.Position() Vec3
ctx.Facing() float32              // radians
ctx.IsAlive() bool
ctx.Phase() string
ctx.IsCasting() bool
```

#### Threat — "who is aware of me?"

Threat is passive awareness data maintained by the combat system. See §5 for design details.

```
ctx.ThreatTable() []ThreatEntry   // all known players, sorted by damage dealt
ctx.HighestThreat() Entity        // most cumulative damage
ctx.ThreatCount() int
ctx.ThreatInRange(range) []Entity
ctx.NearestThreat() Entity
ctx.FarthestThreat() Entity
```

Entity references returned from queries:

```
entity.EntityID() string
entity.ClassTag() string          // "gunner", "vanguard", "blade_dancer"
entity.HealthPct() float32
entity.Position() Vec3
entity.DistanceTo(other) float32
entity.IsCasting() bool
entity.IsAlive() bool
```

#### Perception — "what's around me?"

```
ctx.EntitiesInRadius(radius) []Entity
ctx.NearestEnemy() Entity
ctx.LineOfSight(entity) bool
ctx.IsFlanking(entity) bool
ctx.EntitiesInCone(angle, range, direction) []Entity
```

#### Combat — "what can I do?"

```
ctx.Cast(spellID, target?) Result
ctx.CastAtPosition(spellID, pos) Result
ctx.IsOnCooldown(spellID) bool
ctx.CooldownRemaining(spellID) float32
ctx.AttackWeighted() Result
ctx.CastPattern(patternID) PatternHandle
```

#### Movement — "where should I go?"

```
ctx.MoveTo(pos) Result
ctx.MoveToEntity(entity) Result
ctx.MoveAwayFrom(entity, distance)
ctx.FaceToward(entity)
ctx.Stop()
```

#### Memory — "what do I remember?"

```
ctx.GetFlag(key) bool
ctx.SetFlag(key)
ctx.ClearFlag(key)
ctx.GetCounter(key) int
ctx.SetCounter(key, value)
ctx.IncrementCounter(key)
ctx.StartTimer(key, duration)
ctx.TimerExpired(key) bool
```

#### Spawn — "my underlings"

```
ctx.Spawn(mobID, pos) Entity
ctx.SpawnWave(mobID, count, pattern) []Entity
ctx.MinionsAliveCount() int
```

---

## 5. Threat System

### 5.1 Design Principle

From the [combat design doc](../design/combat.md):

> No traditional threat table. Shield Vanguard physically blocks space, doesn't "taunt."

Threat in this system is **passive awareness**, not an aggro mechanic. The combat system tracks cumulative player activity against each NPC. The BT reads this data to inform targeting decisions but is never forced to act on it. There is no aggro lock, no taunt, and no forced targeting.

### 5.2 What Threat Tracks

```go
type ThreatEntry struct {
    EntityID  string
    ClassTag  string
    Damage    float32   // cumulative damage dealt to this NPC
    Position  Vec3
    Distance  float32   // current distance to this NPC
    HealthPct float32
    IsCasting bool
    IsAlive   bool
}
```

The combat system accumulates threat passively:

- Damage dealt to the NPC adds to the source's threat value
- No class-specific threat modifiers (no class generates "extra" threat)
- No healing-based threat generation
- Threat resets on NPC death or encounter reset

### 5.3 How BTs Use Threat

Threat is one input signal among many. A boss tree might use it, ignore it, or combine it with spatial and perception data:

```yaml
# Snipe the top damage dealer at range
- sequence:
    - condition: { highest_threat_in_range: { min: 8, max: 15 } }
    - action: snipe_highest_threat

# Punish a caster (uses perception, not threat)
- sequence:
    - condition: player_casting_in_range(5)
    - action: interrupt_cast

# Go for nearest player (ignores threat entirely)
- action: attack_nearest

# Punish stacking by targeting the cluster with highest total threat
- sequence:
    - condition: { threats_clustered_gte: { radius: 4, count: 3 } }
    - action: aoe_on_cluster
```

### 5.4 What Threat Does NOT Do

- **No aggro lock.** No entity is ever "the tank" because of threat numbers.
- **No taunt mechanic.** The Vanguard holds space by physically blocking attacks, not by manipulating threat values.
- **No forced targeting.** The BT always decides; threat is advisory data.
- **No threat modifiers.** No class generates more or less threat per damage point.
- **BT cannot write to threat.** No `AddThreat()`, `SetThreat()`, `ResetThreat()` in the entity context. Threat accumulates passively from combat events.

---

## 6. Spell System

### 6.1 Design Split

| Class        | Spell Implementation      | Rationale                                     |
| ------------ | ------------------------- | --------------------------------------------- |
| Gunner       | Pure YAML data            | Skillshots are data-driven projectiles.       |
| Vanguard     | Pure YAML data            | Melee actions are data-driven effects.        |
| Blade Dancer | Go actions (existing)     | Already implemented, performance-sensitive.   |
| Arcanotechnicien | YAML + Go lifecycle hooks | Flux commitment, channeling, evolving spells. |
| Engineer     | YAML + spawned entity AI  | Deployables use shared BT leaves.             |
| Tutelaire    | YAML, minimal Go hooks    | Aura positioning logic.                       |

### 6.2 Spell Definition (YAML)

```yaml
spell: flux_bolt
class: arcanotechnicien
cast_time: 1.2
cooldown: 0
range: 30
cost:
    flux: 10
targeting:
    type: skillshot
    origin: caster
    direction: camera_facing
    range: 40
    width: 0.5
    piercing: false
effects:
    - type: projectile
      speed: 20
      on_hit:
          - type: damage
            school: fire
            base: 80
            scaling: 0.6
          - type: apply_debuff
            debuff: burning
            duration: 4.0
            stacks: true
```

### 6.3 Targeting Types

| Type        | Description                        | Example                        |
| ----------- | ---------------------------------- | ------------------------------ |
| `skillshot` | Projectile along camera facing     | Flux bolt, void lance          |
| `entity`    | Tab-target or lock-on              | Siphon flux                    |
| `smart`     | Engine picks via strategy function | Emergency heal (lowest_health) |
| `self_aoe`  | AoE around caster                  | Frost nova, blinding flash     |
| `ground`    | Player places a reticle            | Ice wall, flux turret          |

Smart targeting strategies are Go functions registered to the leaf registry using the same pattern.

### 6.4 Pattern Engine (Bullet Hell)

Go provides built-in emitter types: `radial`, `cone`, `line`, `arc`, `ring_contract`, `targeted`, `random_zone`. Patterns are composed from emitters in YAML:

```yaml
spell: void_spiral
phases:
    - emitter: radial
      count: 24
      offset_per_wave: 15
      waves: 8
      wave_interval: 0.2
      projectile:
          speed: 6
          radius: 0.3
          lifetime: 4.0
          on_hit: apply_debuff:void_mark
```

Safe zones can be declared:

```yaml
safe_zones:
    - type: arc
      angle: 45
      direction: random
```

Patterns can be dynamically modified mid-flight via `ctx.ModifyActivePattern()`. The same pattern engine powers both boss bullet-hell attacks and player spell effects.

### 6.5 Lifecycle Hooks (Arcanotechnicien)

Arcanotechnicien spells with Flux commitment, channeling, or evolving behavior use Go lifecycle hooks (`OnCastStart`, `OnCastComplete`, `OnChannelTick`, `OnHit`, `OnProc`, etc.). Most spells implement zero hooks. The hooks are optional methods on a `SpellController` interface.

---

## 7. Encounter Definition

### 7.1 Full Boss Example (Tier 3)

```yaml
boss: hollow_sovereign
tier: 3
health_pool: 2_400_000

phases:
    - id: duelist
      trigger: { health_pct_gt: 0.65 }
      tree:
          selector:
              - sequence:
                    - condition: target_casting
                    - condition: target_in_range(3)
                    - action: punish_caster
              - sequence:
                    - condition: is_flanking
                    - action: aggressive_melee
              - sequence:
                    - condition: target_far
                    - action: ranged_pressure
              - action: standard_melee

    - id: unraveling
      trigger: { health_pct_lte: 0.65, health_pct_gt: 0.30 }
      on_enter: phase_transition_unraveling
      tree:
          selector:
              - sequence:
                    - condition: minion_died_this_tick
                    - action: harvest_fallen_minion
                    - sequence:
                          - condition: { counter_gte: { key: absorbed, value: 3 } }
                          - action: become_empowered
              - sequence:
                    - condition: { cooldown_ready: bullet_hell }
                    - selector:
                          - sequence:
                                - condition: { threats_clustered_gte: { radius: 4, count: 3 } }
                                - action: punish_stack
                          - action: choose_spread_pattern
              - sequence:
                    - condition: { highest_threat_distance_lt: 4 }
                    - action: desperate_melee
              - action: reposition_for_pattern

    - id: hollow
      trigger: { health_pct_lte: 0.30 }
      on_enter: phase_transition_hollow
      tree:
          selector:
              - sequence:
                    - condition: can_layer_patterns
                    - action: overlapping_bullet_hell
              - action: relentless_assault
```

### 7.2 Multi-Entity Boss Example

```yaml
encounter: twin_heralds
group:
    - id: herald_of_flame
      mob: herald_flame
      position: [10, 0, 20]
      role_hint: aggressive
    - id: herald_of_void
      mob: herald_void
      position: [30, 0, 20]
      role_hint: defensive
shared_health: false
enrage_on_partner_death: true
```

Each entity runs its own BT independently. Coordination via messages and sibling queries — see [AI Long-term Vision](ai-vision.md) for the sibling and message APIs.

### 7.3 Data-Driven Mob (Tier 1, Zero Custom Code)

```yaml
mob: forest_stalker
tier: 1
brain:
    tree:
        selector:
            - sequence:
                  - condition: has_target
                  - selector:
                        - sequence:
                              - condition: target_in_range(melee)
                              - action: attack_weighted
                        - action: move_to_target
            - sequence:
                  - condition: has_patrol_path
                  - action: follow_patrol
            - action: idle
combat:
    attacks:
        - id: claw_swipe
          range: 2
          cooldown: 1.5
          weight: 3
        - id: pounce
          range: 8
          cooldown: 6.0
          weight: 1
          condition: target_distance > 4
patrol:
    path: [[10, 0, 20], [30, 0, 20], [30, 0, 40]]
    style: loop
```

---

## 8. Implementation Priority

| Priority | System                          | Rationale                                                            |
| -------- | ------------------------------- | -------------------------------------------------------------------- |
| **1**    | **BT executor + leaf registry** | Port existing boss from FSM to BT. Highest-leverage change.         |
| **2**    | **Tier 1 data-driven mobs**     | Express existing mobs as YAML. Prove the system.                     |
| **3**    | **Pattern engine**              | Bullet-hell patterns for bosses and Arcanotechnicien. Core differentiator.   |
| **4**    | **TDD scenarios**               | Write tests for existing boss behavior. Build confidence.            |
| **5**    | **Combat logger**               | Start writing events. Simplest version, no API yet.                  |
| **6**    | **Fuzz tests + specs**          | Balance testing. Useful once 2+ encounters exist.                    |
| **7**    | **REST API + analysis**         | Useful once combat log data accumulates.                             |

Testing framework details: see [AI Testing & Balance](testing.md).
Combat logging details: see [Combat Logging & Observer](combat_logs.md).
Long-term systems (ML training, OSS extraction): see [AI Long-term Vision](ai-vision.md).
