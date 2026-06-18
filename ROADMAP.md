# Pyrrhic Stars - Roadmap & Progress

> Last updated: 2026-03-30

## Current Phase: 0 — Proof of Concept

**Goal**: answer "does multi-gameplay-mode co-op feel good?"

---

## Infrastructure

-   [x] Repo scaffold (client/, server/, blender/, docs/)
-   [x] Nix flake dev shell (godot, blender, go, redis, postgres, uv, just)
-   [x] Blender asset workflow
-   [x] Docker Compose (gateway, zone, chat, redis, postgres)
-   [x] Justfile task runner
-   [x] Game design docs split into structured tree
-   [ ] CI pipeline (lint, test, build)

---

## Phase 0 — Proof of Concept

Status: **NOT STARTED**

### Player Controllers

-   [x] Gunner (FPS)
    -   [x] First-person camera rig
    -   [x] WASD movement
    -   [x] One raycast gun
    -   [x] Crosshair HUD
-   [x] Vanguard (Souls-like)
    -   [x] Third-person camera
    -   [x] WASD movement
    -   [x] Dodge roll
    -   [x] One melee swing (hitbox)
    -   [x] Stamina bar HUD
-   [x] Blade Dancer (Combo Fighter)
    -   [x] Third-person target-lock camera
    -   [x] 4 abilities
    -   [x] 2 blade configurations (Orbit, Lance)
    -   [x] Configuration state display HUD

### Arena

-   [x] CSG blockout (flat floor, walls, pillars, cover boxes)

### Enemy

-   [x] Basic enemy scene
    -   [x] Move toward nearest player
    -   [x] Melee swing with 1s telegraph
    -   [x] Ranged projectile with 0.5s laser warning
    -   [x] Two attack patterns on a loop

### Multiplayer

-   [x] Godot ENet local multiplayer (two instances on localhost)
-   [x] Player spawning and class selection

### Validation (Phase 0.5)

-   [ ] Playtest with a friend (or girlfriend)
-   [ ] Record 60-second gameplay clip
-   [ ] Post clip (r/godot, r/indiegaming, r/gamedev, Twitter/X)
-   [ ] Open Discord server

---

## Phase 1 — Playable with Friends

Status: **BLOCKED** (waiting on Phase 0 completion)

### Server

-   [x] Go gateway service (WebSocket transport, player connections)
-   [ ] Go zone service (tick loop, game simulation)
-   [ ] Go chat service (Redis pub/sub)
-   [ ] Redis integration (player positions, combat state, Flux)
-   [ ] PostgreSQL integration (character creation, persistence)

### Game Systems (server-side)

-   [ ] Flux system (reserve, afflux, recovery, instability)
-   [ ] Resistance system (RMEC, RRAD, RINT)
-   [ ] Defense system (Score de Defense)
-   [ ] Affinity validation

### Client

-   [x] Client networking layer (WebSocket, server-reconciled movement)
-   [ ] 3 playable classes polished: Gunner (Assault), Vanguard (Blade), Blade Dancer (Multi)
-   [ ] Basic HUD per class
-   [ ] Basic sound design

### Content

-   [ ] Hub area (small military base, modular kit)
-   [ ] Derelict City dungeon
    -   [ ] Asset kit (streets, facades, interiors, rubble, alien tech)
    -   [ ] 2 boss encounters
    -   [ ] Trash pack encounters
    -   [ ] Lighting pass (dark blue ambient, alien spotlights, volumetric fog, rain)

### Infrastructure

-   [ ] Playable over internet
-   [ ] Basic deployment (k3s or manual)

---

## Phase 2 — Monetizable Alpha

Status: **BLOCKED** (waiting on Phase 1)

-   [ ] Arcanotechnicien class (Destroyer spec) with Flux commitment UI
-   [ ] Character progression (affinity growth)
-   [ ] Loot system (boss + trash drops)
-   [ ] 2 additional dungeons (new themes/kits)
-   [ ] Mythic+ timer mode (keystones, affixes)
-   [ ] Account system (Discord OAuth)
-   [ ] Companion web app (React/TS: leaderboards, armory, key history)
-   [ ] Distribution via itch.io (10-15 EUR)
-   [ ] Steam page with trailer

---

## Phase 3 — Growth

Status: **BLOCKED** (waiting on Phase 2)

-   [ ] Engineer class (Architect / Pilot / Saboteur)
-   [ ] Tutelaire class (Guardian / Adjudicator)
-   [ ] Additional specs for existing classes
-   [ ] More dungeons
-   [ ] First outdoor zone
-   [ ] Cosmetic shop
-   [ ] Community feedback loop

---

## Long-term Vision (no timeline)

-   [ ] Space dogfighting
-   [ ] Arena PvP
-   [ ] Large-scale PvP (Planetside 2 style)
-   [ ] Open world zones (1x1 km+)
-   [ ] Megacity capital hub
-   [ ] Ship flight in atmosphere
-   [ ] Exploration system
-   [ ] Harvesting & crafting
-   [ ] Player market
