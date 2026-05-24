# Combat Logging & Observer System

Two complementary systems:

1. **Combat Event Logger** — server-side, writes discrete events (damage, heal, death, buff) to ClickHouse for analytics, DPS meters, and balance analysis.
2. **Unified Observer** — client-side, renders tick-by-tick state from a pluggable data source, supporting live play, replay, and spectating.

The event logger feeds the analysis REST API and the fuzz test assertion layer. The fight export reconstructs full tick state for the observer's replay mode.

Related: [AI & Encounter System](ai.md) for encounter definitions, [AI Testing & Balance](testing.md) for fuzz simulation that feeds the logger.

---

## 1. Combat Event Logger

### 1.1 Design

The combat logger is infrastructure, not test code. Same pipeline for fuzz simulations and live play. Fuzz tests produce data; analysis lives in the API.

### 1.2 Logger

```go
type Logger struct {
    buffer        chan LogEntry
    conn          driver.Conn    // clickhouse-go native connection
    batchSize     int            // default 1000
    flushInterval time.Duration  // default 500ms
}
```

-   Non-blocking: `Log()` writes to a buffered channel. If full, drops silently. Never blocks the game loop.
-   Async flush: a background goroutine batches entries and writes via ClickHouse native batch insert.
-   Zero overhead when disabled: swap with `NullSink` (implements the `EventSink` interface, does nothing).

### 1.2.1 Why ClickHouse

Combat events are append-only analytical data — aggregations, percentiles, time-series queries. ClickHouse is purpose-built for this workload:

-   Columnar storage with extreme compression on repetitive event data
-   Analytical queries (DPS percentiles, group-by class, phase timing) run orders of magnitude faster than row-oriented databases
-   Single binary, ~200MB RAM idle, no JVM — trivial to self-host
-   Native Go driver: `github.com/ClickHouse/clickhouse-go/v2`

PostgreSQL remains the right choice for relational data (accounts, characters, groups) if needed later. Combat events are not relational.

### 1.3 Event Schema

```go
type LogEntry struct {
    GroupID       string
    InstanceID    string
    EncounterID   string
    Tick          int
    Timestamp     time.Duration
    Source        string
    SourceClass   string
    Target        string
    EventType     EventType       // damage, heal, buff_apply, buff_remove, buff_tick,
                                  // commit_start, commit_end, cooldown_start, cooldown_end,
                                  // dodge, death
    AbilityID     string
    Amount        float32
    Overkill      float32
    School        string
    IsCrit        bool
    IsDodged      bool
    Phase         string
    BossHealth    float32
    PosX          float32
    PosY          float32
    PosZ          float32
    ResourceType  string
    ResourceDelta float32
    ResourceAfter float32
}
```

### 1.4 Instance Metadata

```go
type InstanceLog struct {
    InstanceID   string
    GroupID      string
    EncounterID  string
    StartedAt    time.Time
    Duration     time.Duration
    Outcome      Outcome         // player_win, boss_win, timeout
    Source       LogSource       // simulation, live
    Participants []ParticipantLog
}

type ParticipantLog struct {
    InstanceID  string
    EntityID    string
    Name        string
    Class       string
    IsBot       bool
    BotProfile  string
}
```

### 1.5 Database Schema

```sql
CREATE TABLE instances (
    instance_id  String,
    group_id     String,
    encounter_id String,
    started_at   DateTime,
    duration_ms  UInt32,
    outcome      LowCardinality(String),
    source       LowCardinality(String),
    created_at   DateTime DEFAULT now()
) ENGINE = MergeTree()
ORDER BY (encounter_id, group_id, instance_id);

CREATE TABLE participants (
    instance_id String,
    entity_id   String,
    name        String,
    class       LowCardinality(String),
    is_bot      Bool DEFAULT false,
    bot_profile String DEFAULT ''
) ENGINE = MergeTree()
ORDER BY (instance_id, entity_id);

CREATE TABLE combat_events (
    instance_id    String,
    group_id       String,
    encounter_id   LowCardinality(String),
    tick           UInt32,
    timestamp_ms   UInt32,
    source         String,
    source_class   LowCardinality(String),
    target         String,
    event_type     UInt8,
    ability_id     LowCardinality(String),
    amount         Float32 DEFAULT 0,
    overkill       Float32 DEFAULT 0,
    school         LowCardinality(String) DEFAULT '',
    is_crit        Bool DEFAULT false,
    is_dodged      Bool DEFAULT false,
    phase          LowCardinality(String) DEFAULT '',
    boss_health    Float32 DEFAULT 0,
    pos_x          Float32 DEFAULT 0,
    pos_y          Float32 DEFAULT 0,
    pos_z          Float32 DEFAULT 0,
    resource_type  LowCardinality(String) DEFAULT '',
    resource_delta Float32 DEFAULT 0,
    resource_after Float32 DEFAULT 0
) ENGINE = MergeTree()
ORDER BY (encounter_id, group_id, instance_id, tick);
```

`LowCardinality(String)` gives dictionary encoding on columns with few distinct values (class names, ability IDs, encounter IDs) — significant compression gains. The `ORDER BY` clause doubles as the primary index; ClickHouse doesn't need secondary indexes for most analytical queries.

### 1.6 REST API

Controller-Service-Repo pattern.

```
GET /api/v1/logs/instances                       ?group_id=&encounter_id=&outcome=
GET /api/v1/logs/instances/{id}
GET /api/v1/logs/instances/{id}/events            ?source=&type=&phase=
GET /api/v1/logs/instances/{id}/timeline
GET /api/v1/logs/instances/{id}/dps               ?entity_id=
GET /api/v1/logs/instances/{id}/damage-taken
GET /api/v1/logs/instances/{id}/deaths
GET /api/v1/logs/instances/{id}/buffs

GET /api/v1/logs/stats/dps                        ?group_id=&encounter_id=&class=
GET /api/v1/logs/stats/deaths                     ?group_id=&encounter_id=
GET /api/v1/logs/stats/outcomes                   ?group_id=&encounter_id=
GET /api/v1/logs/stats/phases                     ?group_id=&encounter_id=
GET /api/v1/logs/stats/abilities                  ?group_id=&encounter_id=
GET /api/v1/logs/stats/percentiles                ?encounter_id=&class=&metric=
```

---

## 2. Unified Observer System

The game client is a state-driven renderer: it receives authoritative tick data from the server, reconciles entity positions, triggers animations from ability events, and applies lerp/snap corrections. This architecture means the client is fundamentally a replay tool that happens to also accept player input.

The Unified Observer formalizes this — a single rendering pipeline driven by a pluggable data source, supporting three modes with shared camera controls, HUD binding, and optional deep state inspection.

### 2.1 Client Modes

The client operates in exactly one mode per session. All three modes share the same rendering path, animation system, HUD, and camera controls. They differ only in data source, input handling, and state visibility.

#### Live Player Mode (existing)

-   **Data source**: WebSocket connection to game server
-   **Input**: Enabled — player actions sent to server
-   **Optimistic updates**: Enabled — local prediction with server reconciliation
-   **State visibility**: Full state for local player, filtered for others
-   **Camera**: Locked to local player entity
-   **HUD**: Bound to local player entity

#### Replay Mode

-   **Data source**: Local JSON file (fight export)
-   **Input**: Disabled
-   **Optimistic updates**: Disabled
-   **State visibility**: God-mode — full state for every participant at every tick
-   **Camera**: Free-fly, click-to-follow, or first-person spectate (see §4)
-   **HUD**: Bound to selected entity (click to rebind)
-   **Tick controller**: Pause, play, speed multiplier, seek slider, rewind

#### Live Spectator Mode (Admin)

-   **Data source**: WebSocket connection to game server (spectator channel)
-   **Input**: Disabled — no entity spawned
-   **Optimistic updates**: Disabled
-   **State visibility**: God-mode — full state for every participant
-   **Camera**: Free-fly, click-to-follow, or first-person spectate (see §4)
-   **HUD**: Bound to selected entity (click to rebind)
-   **Tick controller**: Live indicator; optional short rewind buffer

---

## 3. Data Source Interface

### 3.1 TickSource

```
TickSource
+-- next_frame() -> TickFrame | null
+-- seek(tick: int)          // replay only; no-op for live modes
+-- can_seek() -> bool
+-- playback_speed: float    // 1.0 = real-time; replay only
+-- mode() -> player | replay | spectator
```

### 3.2 TickFrame

Universal per-tick payload consumed by the renderer. Same structure regardless of source.

```
TickFrame
+-- tick: int
+-- timestamp: Duration           // offset from encounter start
+-- phase: string
+-- entities: map[EntityID] -> EntityState
+-- events: []TickEvent
```

### 3.3 EntityState (God-Mode)

In Player Mode, the client receives full state for the local player and a reduced projection for others. In Replay and Spectator modes, every entity carries the full payload.

```
EntityState
+-- entity_id: string
+-- name: string
+-- class: string
+-- position: Vec3
+-- facing: float
+-- health: float
+-- health_max: float
+-- resources: map[ResourceType] -> ResourceState
+-- cooldowns: map[AbilityID] -> CooldownState
+-- buffs: []BuffState
+-- commit: CommitState | null
+-- is_alive: bool
+-- bt_trace: BTTrace | null       // only if deep inspection enabled

ResourceState
+-- current: float
+-- max: float
+-- type: string                   // flux, energy, stamina, etc.

CooldownState
+-- ability_id: string
+-- remaining: Duration
+-- total: Duration
+-- charges: int | null

BuffState
+-- buff_id: string
+-- name: string
+-- stacks: int
+-- remaining: Duration
+-- total: Duration
+-- is_debuff: bool
+-- source: EntityID

CommitState
+-- ability_id: string
+-- elapsed: Duration
+-- total: Duration
+-- target: EntityID | null
```

### 3.4 TickEvent

Discrete events that occurred during this tick. The renderer uses these to trigger animations, spawn VFX, and update the combat log feed.

```
TickEvent
+-- type: EventType
+-- source: EntityID
+-- target: EntityID | null
+-- ability_id: string | null
+-- amount: float | null
+-- is_crit: bool
+-- school: string | null
+-- detail: string | null           // human-readable for log feed

EventType: damage | heal | buff_apply | buff_remove | buff_tick
         | commit_start | commit_end | commit_interrupt
         | cooldown_start | cooldown_end
         | death | resurrect | phase_change
         | position_update
```

---

## 4. Camera System

Three camera modes, switchable at runtime via hotkeys or UI toggle. Available in Replay and Spectator modes.

### 4.1 Free-Fly

Detached camera with no entity binding. WASD movement, mouse-look, scroll to adjust speed. The HUD hides or shows a minimal "no entity selected" state.

### 4.2 Follow

Bound to a selected entity. Third-person orbit camera (existing game camera behavior). Click any entity in the world to rebind. The HUD fully binds to the selected entity — their cooldowns, resources, buffs, commit bar.

### 4.3 First-Person Spectate

Same as Follow but positioned at the entity's head/shoulder. Useful for evaluating encounter readability from a specific player's perspective ("can the healer actually see the telegraph from where they stand?").

### 4.4 Entity Selection

Clicking an entity in any camera mode:

-   Rebinds the HUD to that entity's EntityState
-   In Follow/First-Person mode, moves the camera to that entity
-   In Free-Fly mode, rebinds the HUD but leaves the camera detached

A sidebar lists all participants with class icons, allowing selection without clicking in the 3D world.

---

## 5. Fight Export Format

A single JSON file per encounter instance containing the full tick history. Serves both the Replay Mode client and future web-based replay tools.

### 5.1 API Endpoint

```
GET /api/v1/encounters/{instanceID}/export      -> fight JSON
GET /api/v1/encounters/{instanceID}/export/bt   -> BT trace JSON (separate, optional)
```

### 5.2 File Structure

```json
{
	"version": 1,
	"encounter_id": "hollow_sovereign",
	"instance_id": "uuid",
	"group_id": "fuzz-run-42",
	"source": "simulation",
	"start_time": "2026-04-29T12:00:00Z",
	"duration_ms": 312000,
	"outcome": "kill",

	"participants": [
		{
			"entity_id": "player_1",
			"name": "Valentin",
			"class": "gunner",
			"is_player": true
		},
		{
			"entity_id": "hollow_sovereign",
			"name": "The Hollow Sovereign",
			"class": "boss",
			"is_player": false
		}
	],

	"encounter_metadata": {
		"arena_id": "shattered_throne",
		"arena_bounds": { "min": [0, 0, 0], "max": [60, 0, 60] },
		"phases": ["awakening", "unraveling", "collapse"]
	},

	"ticks": [
		{
			"tick": 0,
			"timestamp_ms": 0,
			"phase": "awakening",
			"entities": {
				"player_1": { "/* full EntityState */": "" },
				"hollow_sovereign": { "/* full EntityState */": "" }
			},
			"events": [
				{
					"type": "commit_start",
					"source": "player_1",
					"ability_id": "rapid_fire"
				}
			]
		}
	]
}
```

### 5.3 Size Considerations

A 5-minute fight at 20 Hz (server tick rate) = 6,000 tick frames. With 5 entities and full state per tick, expect 20-60 MB uncompressed. Gzip brings this to 2-6 MB.

For BT traces, the file grows significantly. Traces are stored in a separate companion file (`{instance_id}.bt.json`) to keep the base export lean.

**Delta compression** (future optimization): only emit changed fields per tick instead of full state. Reduces export size by ~80% but adds reconstruction complexity.

---

## 6. Tick Controller (Replay Mode)

A media-player-style control bar. Only active in Replay mode; hidden in Live Spectator mode (replaced by a "LIVE" badge).

### 6.1 Controls

| Control       | Behavior                                               |
| ------------- | ------------------------------------------------------ |
| Play / Pause  | Toggle tick advancement                                |
| Speed         | 0.25x, 0.5x, 1x, 2x, 4x, 8x multiplier                 |
| Seek slider   | Scrub to arbitrary tick; triggers state reconstruction |
| Step forward  | Advance exactly one tick (while paused)                |
| Step backward | Rewind exactly one tick (while paused)                 |
| Jump to event | Click a combat log entry to seek to that tick          |

### 6.2 State Reconstruction on Seek

Seeking to an arbitrary tick requires the client to know full entity state at that tick. Two strategies:

-   **Replay from zero** (default): Walk the tick array from tick 0 to target. For typical fight lengths (< 6,000 ticks at 20 Hz), this completes quickly and is simple to implement.
-   **Snapshot checkpoints**: The export file includes a full state snapshot every N ticks (e.g., every 200). Seek to the nearest prior snapshot, then replay forward. Only implement if replay-from-zero becomes a perceivable delay.

### 6.3 Live Spectator Rewind Buffer

In Spectator mode, the client retains the last N seconds of tick frames in a ring buffer (e.g., 30 seconds = 600 frames at 20 Hz). The admin can pause the live feed and scrub within this buffer, then resume live. Client-side only — no server support required.

---

## 7. Deep State Inspection

Optional layers for developer-level insight into entity internals. Available in both Replay and Spectator modes.

### 7.1 Cooldown & Buff Inspector

Panel displayed when an entity is selected. Reads directly from EntityState at the current tick:

-   Ability icons with cooldown sweep timers
-   Buff/debuff bar with duration, stacks, source entity
-   Resource bars (flux, energy, stamina, etc.)
-   Commit bar with ability name and progress

No additional server-side work required — this data is already in the god-mode EntityState.

### 7.2 Behavior Tree Visualizer

Displays the BT decision path for a selected NPC entity at the current tick.

#### BT Trace Data

```
BTTrace
+-- tick: int
+-- entity_id: string
+-- nodes: []BTNodeEval

BTNodeEval
+-- node_id: string
+-- node_type: sequence | selector | condition | action | decorator
+-- name: string
+-- result: success | failure | running
+-- detail: string | null           // e.g. "health_pct=0.27 < 0.30 -> true"
+-- children_evaluated: int
```

#### Server-Side Emission

BT tracing is opt-in per encounter instance:

-   Server admin command: `enable_bt_trace <instance_id>`
-   Fuzz test flag: `--bt-trace` (for targeted reruns)

When enabled, the BT executor emits a BTTrace per entity per tick. In Spectator mode, streamed alongside tick frames. In the export file, stored in a companion file.

#### Client Rendering

A collapsible tree panel showing:

-   Full tree structure for the selected entity
-   Each node colored by result at the current tick (green = success, red = failure, yellow = running, gray = not evaluated)
-   Condition nodes show evaluation detail on hover
-   Active execution path highlighted

---

## 8. Server-Side Changes

### 8.1 Spectator Connection

A new connection type. The admin authenticates with elevated permissions and connects to a running encounter instance.

-   No entity spawned — the spectator is invisible
-   Server adds spectator to tick broadcast list
-   Tick frames use god-mode EntityState serializer (full state for all entities)
-   Gameplay input from spectator connection is silently dropped
-   Multiple spectators can observe the same instance

### 8.2 God-Mode Serializer

A serialization path that emits full EntityState for every participant. Used by:

-   The fight export endpoint
-   The spectator tick broadcast

This is the existing per-player serializer minus the filtering step that strips cooldowns/resources/buffs from non-local entities.

### 8.3 Combat Log Event Types

Event types required in the combat logger beyond damage/heal/death:

| Event Type     | Fields                                             |
| -------------- | -------------------------------------------------- |
| cooldown_start | ability_id, duration, charges                      |
| cooldown_end   | ability_id                                         |
| buff_apply     | buff_id, target, source, duration, stacks          |
| buff_remove    | buff_id, target, reason (expired/dispelled/death)  |
| buff_tick      | buff_id, target, amount, type (damage/heal)        |
| commit_start   | ability_id, source, target, commit_time            |
| commit_end     | ability_id, source, result (completed/interrupted) |

---

## 9. Open Questions

1. **Entity spawn/despawn mid-fight**: Adds that spawn during phase transitions need explicit spawn/despawn events in the tick stream. The current TickFrame.entities map handles this implicitly (entity appears/disappears), but the replay client needs to instantiate/destroy scene nodes in response.

2. **Arena geometry**: The replay client needs the encounter arena to render a meaningful 3D scene. The export file should reference which arena to load via `arena_id`.

3. **AoE and telegraph visualization**: Ground indicators and boss telegraphs are currently driven by game logic. The replay client needs to reconstruct these from events. Consider adding explicit `telegraph_start` / `telegraph_end` events to the log, or deriving them from ability metadata + commit timing.

4. **Replay file distribution**: For player-facing replays (post-launch), how are fight exports shared? Direct download from the API, or a replay listing/browser? Defer to post-launch planning.

5. **Delta compression**: Full state per tick at 20 Hz produces large export files (20-60 MB). Delta compression could reduce this by ~80% but adds reconstruction complexity. Implement only if file size becomes a problem.
