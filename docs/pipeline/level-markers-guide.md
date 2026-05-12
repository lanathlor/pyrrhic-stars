# Level Markers Guide — Godot to Go Server

This guide walks you through placing level markers in Godot scenes so the collision exporter can generate JSON for the Go server. No GDScript needed — everything is done through the editor UI.

**Exporter location:** `client/addons/collision_exporter/collision_exporter.gd`
**Output:** `shared/levels/<zone_name>.json`

---

## Table of Contents

1. [Core Concepts](#1-core-concepts)
2. [How to Add a Node to a Group](#2-how-to-add-a-node-to-a-group)
3. [How to Add Metadata to a Node](#3-how-to-add-metadata-to-a-node)
4. [Portal (Dungeon Entrance)](#4-portal-dungeon-entrance)
5. [Player Spawn Point](#5-player-spawn-point)
6. [Enemy Spawn Point](#6-enemy-spawn-point)
7. [NPC Spawn Point (with Patrol Path)](#7-npc-spawn-point-with-patrol-path)
8. [Zone Trigger](#8-zone-trigger)
9. [Obstacle (Collision)](#9-obstacle-collision)
10. [Elevator](#10-elevator)
11. [Bounds](#11-bounds)
12. [Running the Exporter](#12-running-the-exporter)
13. [Troubleshooting](#13-troubleshooting)

---

## 1. Core Concepts

The exporter walks the entire scene tree and looks for two things on each node:

- **Group membership** — determines *what type* of marker this node is (e.g., `server_portal`, `server_spawn_player`).
- **Metadata properties** — key/value pairs that configure the marker (e.g., `target_zone = "arena"`).

The node's **global position** in the scene is used as the world-space coordinate. Move the node in the 3D viewport to place the marker where you want it.

### Scene Organization

Keep all marker nodes under a single organizer node for clarity:

```
MyScene (root)
  +-- ... (your visual geometry, meshes, lights, etc.)
  +-- LevelData (Node3D)        <-- organizer, no groups needed
      +-- DungeonPortal         <-- server_portal
      +-- SpawnPlayer1          <-- server_spawn_player
      +-- SpawnPlayer2          <-- server_spawn_player
      +-- EnemyPack1Melee1      <-- server_spawn_enemy
      +-- Bounds                <-- server_bounds
      +-- ...
```

---

## 2. How to Add a Node to a Group

1. Select the node in the **Scene** dock (left panel).
2. In the **Inspector** (right panel), click the **Node** tab (next to Inspector tab, at the top).
3. Click the **Groups** sub-tab.
4. In the text field, type the group name (e.g., `server_portal`).
5. Click **Add**.

The node now has a small icon next to its name in the scene tree indicating it belongs to a group.

> **Alternative:** You can also right-click the node in the Scene dock > **Groups...** to open the same dialog.

---

## 3. How to Add Metadata to a Node

Metadata is how you attach key/value configuration to nodes (like `target_zone`, `def_name`, etc.).

1. Select the node in the **Scene** dock.
2. In the **Inspector** (right panel), scroll to the very bottom.
3. You'll see a section called **"Metadata"** (collapsed by default — click to expand it, or it may say "Add Metadata").
4. Click **Add Metadata** (or the `+` button).
5. In the dialog:
   - **Name:** type the key (e.g., `target_zone`) — do NOT include `metadata/` prefix, Godot adds it internally.
   - **Type:** pick the correct type (String, float, int, bool, Vector3 — see each section below).
   - **Value:** enter the value.
6. Click **Add**.

The metadata now appears in the Inspector under the Metadata section and can be edited inline.

> **Tip:** If you need to edit metadata later, just expand the Metadata section and change values directly. To remove a metadata entry, click the trash icon next to it.

---

## 4. Portal (Dungeon Entrance)

A portal marks where players interact to transfer to another zone.

### Setup

1. Create a **Node3D** (`Add Child Node > Node3D`).
2. Name it something descriptive (e.g., `DungeonPortal`, `ArenaEntrance`).
3. Position it where the player should stand to interact.
4. Add to group: **`server_portal`**.
5. Add metadata:

| Key                  | Type   | Required | Example     | Description                          |
|----------------------|--------|----------|-------------|--------------------------------------|
| `target_zone`        | String | Yes      | `arena`     | Zone the portal leads to             |
| `interaction_radius` | float  | No       | `4.0`       | How close the player must be (default: 4.0) |
| `condition`          | String | No       | `boss_dead` | Only active when condition is met    |

### Example

```
DungeonPortal (Node3D)
  Group: server_portal
  Position: (33, 102, 5.5)
  Metadata:
    target_zone = "arena"          (String)
    interaction_radius = 4.0       (float)
```

### What Gets Exported

```json
{
  "portals": [
    {
      "name": "DungeonPortal",
      "x": 33.0, "y": 102.0, "z": 5.5,
      "target_zone": "arena",
      "interaction_radius": 4.0
    }
  ]
}
```

---

## 5. Player Spawn Point

Where players appear when entering or respawning in the zone. Place multiple for party spread.

### Setup

1. Create a **Node3D**.
2. Name it `SpawnPlayer1`, `SpawnPlayer2`, etc.
3. Position it where the player should appear. The Y coordinate should be just above the floor (e.g., `0.1` above ground level).
4. Add to group: **`server_spawn_player`**.
5. (Optional) Add metadata:

| Key         | Type   | Required | Example           | Description                              |
|-------------|--------|----------|-------------------|------------------------------------------|
| `condition` | String | No       | `pack_1_cleared`  | Spawn only activates when condition is met |

### Condition Values

| Condition         | When It Activates                        |
|-------------------|------------------------------------------|
| *(empty/absent)*  | Always active (default spawn)            |
| `pack_1_cleared`  | All enemies in group 1 are dead          |
| `pack_2_cleared`  | All enemies in group 2 are dead          |
| `boss_dead`       | Boss is dead                             |

When multiple condition tiers are satisfied, the server picks the **highest-progression** set. For example, if both default spawns and `pack_1_cleared` spawns exist and pack 1 is dead, players respawn at the `pack_1_cleared` locations.

### Example — Arena with Checkpoint

```
LevelData (Node3D)
  +-- SpawnPlayer1 (Node3D)  — group: server_spawn_player, pos: (-2, 0.1, 48)
  +-- SpawnPlayer2 (Node3D)  — group: server_spawn_player, pos: (0, 0.1, 48)
  +-- SpawnPlayer3 (Node3D)  — group: server_spawn_player, pos: (2, 0.1, 48)
  +-- CheckpointSpawn1 (Node3D) — group: server_spawn_player, pos: (-1, 0.1, 26)
  |     metadata: condition = "pack_1_cleared"
  +-- CheckpointSpawn2 (Node3D) — group: server_spawn_player, pos: (1, 0.1, 26)
        metadata: condition = "pack_1_cleared"
```

---

## 6. Enemy Spawn Point

Where enemies appear. Supports patrol routes, group linking, and boss flags.

### Setup

1. Create a **Node3D**.
2. Name it descriptively (e.g., `Pack1Melee1`, `BossGuardCaptain`).
3. Position it at the enemy's spawn/idle location.
4. Add to group: **`server_spawn_enemy`**.
5. Add metadata:

| Key             | Type    | Required | Example               | Description                               |
|-----------------|---------|----------|-----------------------|-------------------------------------------|
| `def_name`      | String  | Yes      | `hallway_melee`       | Enemy definition name (matches YAML def)  |
| `patrol_a`      | Vector3 | No       | `(-6, 0.1, 32)`      | First patrol endpoint                     |
| `patrol_b`      | Vector3 | No       | `(6, 0.1, 32)`       | Second patrol endpoint                    |
| `aggro_radius`  | float   | No       | `10.0`                | Detection range (default: 10.0)           |
| `leash_radius`  | float   | No       | `30.0`                | Max chase range (default: 30.0)           |
| `group_id`      | int     | No       | `1`                   | Links enemies into a pack (for conditions)|
| `is_boss`       | bool    | No       | `true`                | Marks as boss encounter                   |
| `condition`     | String  | No       | `pack_1_cleared`      | Only spawns when condition is met         |

### Patrol Routes with Path3D (Advanced)

Instead of `patrol_a`/`patrol_b` metadata, you can add a **Path3D** child node for multi-point patrol:

1. Select your enemy spawn Node3D.
2. `Add Child Node > Path3D`.
3. With the Path3D selected, you'll see **path editing tools** in the 3D viewport toolbar.
4. Click the **Add Point** tool (pencil+ icon) in the toolbar.
5. Click in the 3D viewport to place waypoints. Each click adds a point to the curve.
6. To move existing points: switch to the **Select** tool and drag points.
7. To delete a point: select it and press Delete.

> **Important:** Path3D waypoints are in **local space** relative to the Path3D node. The exporter converts them to world space automatically using `global_transform * curve.get_point_position(i)`.

When a Path3D child exists with 2+ points, it overrides `patrol_a`/`patrol_b`.

### Example — Enemy Pack

```
Pack1Melee1 (Node3D)
  Group: server_spawn_enemy
  Position: (-3, 0.1, 32)
  Metadata:
    def_name = "hallway_melee"     (String)
    patrol_a = (-6, 0.1, 32)      (Vector3)
    patrol_b = (6, 0.1, 32)       (Vector3)
    aggro_radius = 10.0            (float)
    leash_radius = 40.0            (float)
    group_id = 1                   (int)
```

### Example — Boss

```
BossGuardCaptain (Node3D)
  Group: server_spawn_enemy
  Position: (0, 0.1, 0)
  Metadata:
    def_name = "guard_captain"     (String)
    patrol_a = (-5, 0.1, 0)       (Vector3)
    patrol_b = (5, 0.1, 0)        (Vector3)
    aggro_radius = 10.0            (float)
    leash_radius = 30.0            (float)
    is_boss = true                 (bool)
```

---

## 7. NPC Spawn Point (with Patrol Path)

Non-combat NPCs (citizens, merchants, guards) that walk waypoint routes in the hub.

### Setup

1. Create a **Node3D**.
2. Name it (e.g., `NPC_Citizen1`, `NPC_Merchant`).
3. Position it at the NPC's starting location.
4. Add to group: **`server_spawn_npc`**.
5. Add metadata:

| Key             | Type   | Required | Example    | Description                        |
|-----------------|--------|----------|------------|------------------------------------|
| `def_name`      | String | Yes      | `citizen`  | NPC type                           |
| `speed`         | float  | No       | `1.8`      | Walk speed (default: 1.5)          |
| `idle_duration` | float  | No       | `4.0`      | Seconds idle at each waypoint (default: 4.0) |

### Adding Waypoints

Waypoints define the NPC's patrol loop. Use a Path3D child:

1. Select your NPC Node3D.
2. `Add Child Node > Path3D`.
3. With the Path3D selected, use the **Add Point** tool in the 3D viewport toolbar.
4. Click to place each waypoint in order.

If no Path3D child exists, the NPC's own position is used as a single waypoint (stationary NPC — useful for merchants).

### Example — Patrolling Citizen

```
NPC_Citizen1 (Node3D)
  Group: server_spawn_npc
  Position: (-20, -199.95, -70)
  Metadata:
    def_name = "citizen"       (String)
    speed = 1.8                (float)
    idle_duration = 4.0        (float)
  +-- Path3D
        Curve3D with 4 points:
          (0, 0, 0)           — local offset from parent = world (-20, -199.95, -70)
          (0, 0, 40)          — world (-20, -199.95, -30)
          (-20, 0, 40)        — world (-40, -199.95, -30)
          (-20, 0, 0)         — world (-40, -199.95, -70)
```

### Example — Stationary Merchant (no Path3D)

```
NPC_Merchant (Node3D)
  Group: server_spawn_npc
  Position: (15, -199.95, -45)
  Metadata:
    def_name = "merchant"      (String)
    speed = 0.0                (float)
    idle_duration = 999.0      (float)
```

---

## 8. Zone Trigger

Invisible threshold lines that the server uses to track player progression (e.g., "entered the arena", "entered boss room").

### Setup

1. Create a **Node3D**.
2. Name it descriptively (e.g., `ArenaEntry`, `BossRoomEntry`).
3. Position it on the threshold line. Only the axis specified by `axis` metadata matters (default: Z).
4. Add to group: **`server_zone_trigger`**.
5. Add metadata:

| Key          | Type   | Required | Example            | Description                                |
|--------------|--------|----------|--------------------|--------------------------------------------|
| `trigger_id` | String | Yes      | `arena_entry`      | Identifier the server uses to match logic  |
| `axis`       | String | No       | `z`                | Which axis the threshold is on (default: z)|

### Current Trigger IDs

| Trigger ID         | Used For                                        |
|--------------------|-------------------------------------------------|
| `arena_entry`      | Marks transition from warmup lobby to hallway   |
| `boss_room_entry`  | Marks transition from hallway to boss room      |

### Example

```
ArenaEntry (Node3D)
  Group: server_zone_trigger
  Position: (0, 0, 40)
  Metadata:
    trigger_id = "arena_entry"  (String)
    axis = "z"                  (String)
```

The server reads this as: "When a player's Z coordinate crosses 40, they've entered the arena."

---

## 9. Obstacle (Collision)

Server-side collision boxes for walls, pillars, crates, etc. Two node types are supported.

### Option A: CSGBox3D

If your geometry is already a CSGBox3D, just add it to the group. The exporter reads its size and position automatically.

1. Select the CSGBox3D node.
2. Add to group: **`server_collision`**.
3. Done — no metadata needed.

### Option B: StaticBody3D with CollisionShape3D

For mesh-based geometry, add a StaticBody3D with BoxShape3D collision children:

1. Create a **StaticBody3D**.
2. Add a **CollisionShape3D** child with a **BoxShape3D** shape.
3. Add the StaticBody3D to group: **`server_collision`**.

The exporter reads each CollisionShape3D child's box size and position.

---

## 10. Elevator

Moving platforms with vertical travel.

### Setup

1. Select or create the **Node3D** that represents the elevator platform.
2. Add to group: **`server_elevator`**.
3. Add metadata:

| Key        | Type  | Required | Example  | Description                     |
|------------|-------|----------|----------|---------------------------------|
| `half_x`   | float | No       | `4.0`   | Half-width on X (default: 4.0) |
| `half_z`   | float | No       | `4.0`   | Half-width on Z (default: 4.0) |
| `bottom_y` | float | No       | `-200.0`| Lowest Y position              |
| `top_y`    | float | No       | `0.0`   | Highest Y position             |
| `speed`    | float | No       | `10.0`  | Travel speed (units/sec)       |
| `offset_x` | float | No       | `0.0`   | X offset from node position    |
| `offset_z` | float | No       | `0.0`   | Z offset from node position    |

The elevator's center X/Z is derived from the node's global position (plus offsets).

---

## 11. Bounds

Defines the playable area limits for the zone. Only one per scene.

### Setup

1. Create a **Node3D** named `Bounds`.
2. Add to group: **`server_bounds`**.
3. Add metadata:

| Key     | Type  | Required | Example  |
|---------|-------|----------|----------|
| `min_x` | float | Yes      | `-19.5`  |
| `max_x` | float | Yes      | `19.5`   |
| `min_y` | float | Yes      | `-1.0`   |
| `max_y` | float | Yes      | `6.0`    |
| `min_z` | float | Yes      | `-14.5`  |
| `max_z` | float | Yes      | `52.0`   |

If no `server_bounds` node exists, bounds are computed automatically from obstacles (with a 1-unit margin).

---

## 12. Running the Exporter

1. Open the scene you want to export in the Godot editor (e.g., `arena.tscn`).
2. Go to **File > Run Script** (or press `Ctrl+Shift+X`).
3. Navigate to `client/addons/collision_exporter/collision_exporter.gd` and select it.
4. Check the **Output** panel at the bottom for the export summary:
   ```
   collision_exporter: exporting zone 'arena' from res://scenes/environments/arena/arena.tscn
   collision_exporter: wrote /path/to/shared/levels/arena.json (10 obstacles, 0 elevators, 5 player spawns, 9 enemy spawns, 0 npc spawns, 0 portals, 2 zone triggers)
   ```
5. The JSON file is written to `shared/levels/<zone_name>.json`.

### After Exporting

- The Go server reads from `shared/levels/` at startup.
- Run `cd server && go test ./internal/level/...` to validate the JSON loads correctly.
- Commit both the `.tscn` changes and the updated `.json` file.

---

## 13. Troubleshooting

**"collision_exporter: no scene open in editor"**
You need to have the scene open as the active tab before running the script.

**Wrong zone name in output**
The exporter infers the zone name from the scene path. Scenes under `prime_hub/` map to `hub`. If your scene isn't detected, check `_infer_zone_name()` in the exporter.

**Metadata not appearing in export**
- Make sure metadata keys are spelled exactly right (case-sensitive).
- Check the type: `target_zone` must be String, `aggro_radius` must be float, `group_id` must be int.
- Expand the Metadata section in the Inspector to verify values.

**Path3D waypoints at wrong positions**
Path3D curve points are in the Path3D node's local space. The exporter converts to world space via `global_transform`. If points look wrong, check that neither the Path3D nor its parent have unexpected rotations or scales.

**Exported values are 0 or defaults**
The node might not be in the correct group, or metadata keys might be misspelled. Double-check the group name matches exactly (e.g., `server_spawn_enemy`, not `server_enemy_spawn`).
