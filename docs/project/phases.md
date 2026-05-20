# Development Phases

## Phase 0: Proof of Concept (2 weekends)

**Goal**: answer "does multi-gameplay-mode co-op feel good?"

Deliverables:

- Godot project with 3 player controllers:
  - Gunner: FPS camera, WASD movement, one gun (raycast), crosshair HUD
  - Vanguard: third-person camera, WASD + dodge roll, one melee swing (hitbox), stamina bar
  - Blade Dancer: third-person target-lock, 4 abilities, 2 blade configurations, state display
- One arena (CSG geometry: flat floor, cover boxes, pillars)
- One enemy: walks toward nearest player, melee swing (telegraphed, 1s wind-up), ranged projectile at farthest player (0.5s laser warning). Two attack patterns on a loop
- Local multiplayer via Godot ENet (two instances on localhost)
- No server. No persistence. No progression. Hardcoded stats

Success criteria: two players controlling different class types fighting the same enemy feels fun and fair.

Output: 60-second gameplay recording.

## Phase 0.5: Validation (1 week)

Post the 60-second clip:

- r/godot, r/indiegaming, r/gamedev
- Twitter/X (create project account)
- YouTube Shorts / TikTok

Open a Discord server. If nobody cares: project was fun, move on. If people are interested: continue.

## Phase 1: Playable with Friends (6-8 weeks, 10-15h/week)

Deliverables:

- Go game server: tick loop, WebSocket transport, player connections, combat resolution
- Redis integration: player positions, combat state, Flux reserves
- PostgreSQL integration: character creation and persistence
- 3 playable classes: Gunner (Assault spec), Vanguard (Blade spec), Blade Dancer (Multi Blade spec)
- Flux system implemented server-side (reserve, afflux, recovery, instability)
- Resistance system (RMEC, RRAD, RINT)
- One hub area (small military base, modular kit assembly)
- One dungeon: derelict city, 2 bosses, trash packs
- Basic HUD per class
- Playable over internet with friends
- Basic sound design (free sounds from freesound.org)

NOT included: progression, loot, specs, Flux commitment UI, Mythic+.

## Phase 2: Monetizable Alpha (6-8 weeks)

Deliverables:

- Add Arcanotechnicien class (Destroyer spec) with Flux commitment UI
- Character progression via affinity growth
- Loot system (drops from bosses and trash)
- 2 additional dungeons (different themes/kits)
- Basic Mythic+ timer mode (keystone levels, simple affixes)
- Account system via Discord OAuth
- Companion web app (React/TypeScript): leaderboards, character armory, key history
- Distribution via itch.io (10-15 EUR early access)
- Steam page with trailer (wishlists!)

## Phase 3: Growth (ongoing)

- Engineer and Tutelaire classes
- More dungeons
- Specs within existing classes
- First outdoor zone
- Cosmetic shop
- Community-driven content feedback loop

## Phase 0 Checklist

This is the ONLY thing that matters right now.

- [ ] Install Godot 4
- [ ] Create project with the directory structure above (client/ only)
- [ ] Build Gunner FPS controller (camera, WASD, one raycast gun)
- [ ] Build Vanguard melee controller (third-person camera, WASD, dodge, one swing)
- [ ] Build Blade Dancer controller (target-lock camera, 4 abilities, 2 configurations)
- [ ] Build one arena (CSG boxes: floor, walls, pillars, cover)
- [ ] Build one enemy (move toward nearest, melee swing with telegraph, ranged projectile with laser warning)
- [ ] Set up Godot ENet local multiplayer (2 instances)
- [ ] Playtest with a friend
- [ ] Record 60-second clip
- [ ] Post clip online
- [ ] Open Discord

Do NOT start the Go server before this is done.
Do NOT design more classes before this is done.
Do NOT think about monetization before this is done.

Two weekends. Then decide if you continue.
