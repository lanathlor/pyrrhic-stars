# Development Phases

The public roadmap on the landing page
(`web/landing/src/components/Roadmap.astro`) is the source of truth for the
order and shape of phases. This document expands on each phase with internal
detail: tech stack, deliverable checklists, week estimates, and what is
explicitly out of scope.

Phase 0.5 (post-clip validation) is internal only. The public roadmap does
not surface it. The clip and the Discord server are the gate for moving on;
until they exist, we have not validated Phase 0.

## Phase 0: Online alpha

**Goal**: five players in a hub, pick a class, group up, walk into a dungeon,
fight one real boss, leave. Server-authoritative, online co-op, four classes
playable.

**Public deliverables** (paraphrased from the landing page):

- Server-authoritative online co-op
- Four classes, five specs playable
- Combat systems: damage, cooldowns, abilities
- First dungeon playable (one boss live)

**Internal detail** (not surfaced on the landing page):

- Go game server: gateway and zone services, WebSocket transport, 20Hz tick
  loop, combat resolution
- Redis: player positions, combat state, Flux reserves
- PostgreSQL: character creation and persistence
- Flux system implemented server-side (reserve, afflux, recovery, instability)
- Resistance system (RMEC, RRAD, RINT)
- Hub area: small military base, modular kit assembly
- One dungeon: derelict city, one boss
- Client: client-predicted, server-authoritative; reconcile on mismatch
- Basic HUD per class
- Basic sound design (free sounds from freesound.org)
- Playable over the internet with friends

**Estimated scope**: 6-8 weeks at 10-15h/week.

**Not in Phase 0**: progression, loot, specs beyond the first five, Mythic+,
Engineer and Tutelaire classes, Steam or itch.io distribution.

**Phase 0 checklist**:

- [ ] Server: gateway and zone services running
- [ ] Server: hub zone and arena zone, zone transfer
- [ ] Server: combat system (damage, cooldowns, abilities)
- [ ] Server: Flux and Resistance systems
- [ ] Server: persistence (character, position on zone transfer)
- [ ] Client: four class controllers (Gunner, Vanguard, Blade Dancer, Arcanotechnicien)
- [ ] Client: five specs total across the four classes
- [ ] Client: HUD per class
- [ ] Content: one boss, fully authored (telegraphs, mechanics, phases)
- [ ] Networking: client prediction and server reconciliation working
- [ ] Playable with friends over the internet
- [ ] Playtest with at least five players simultaneously

## Phase 0.5: Validation (internal only)

**Goal**: prove there is an audience before committing to Phase 1.

**Gate criteria** (must happen before Phase 1 starts):

- 60-second clip: five players, different gameplay modes visible
  simultaneously in one continuous shot
- Post clip: r/godot, r/indiegaming, r/gamedev, Twitter/X, YouTube Shorts,
  TikTok
- Open a Discord server

**Outcomes**:

- If nobody cares: project was fun, move on.
- If people are interested: continue to Phase 1.

## Phase 1: Complete the first dungeon

**Goal**: the first dungeon is a real content experience, not a single-room
demo. The boss fight feels like a fight, not a placeholder.

**Public deliverables** (paraphrased from the landing page):

- More bosses, trash packs, full clear loop
- Combat feel and boss telegraphs

**Internal detail**:

- Two to three bosses total in the first dungeon (the Phase 0 boss plus new
  ones)
- Trash pack authoring: per-pack telegraphs, density tuning
- Full clear loop: start, trash, boss, trash, boss, loot, reset
- Combat feel pass: hit reactions, screen feedback, audio
- Boss telegraphs refined: color, sound, time-to-react
- Modular kit assembly for boss arenas (reusable pieces across bosses)

**Not in Phase 1**: loot beyond cosmetic, Mythic+ modifiers, group finder,
last two classes.

## Phase 2: Items and Progression

**Goal**: a reason to run the dungeon more than once.

**Public deliverables** (paraphrased from the landing page):

- Loot token
- Trade tokens for items
- Complexity modifiers

**Internal detail**:

- Mythic+ timer mode (keystone levels, simple affixes) is the "complexity
  modifiers" pillar
- Companion web app (React/TypeScript): leaderboards, character armory, key
  history
- Account system via Discord OAuth
- Distribution: itch.io early access (10-15 EUR), Steam page with trailer
- Wishlist goal: 10,000 before launch (10-20% historical conversion rate)

**Not in Phase 2**: Engineer and Tutelaire, group finder, outdoor zones,
cosmetic shop.

## Phase 3: Polish

**Goal**: ship-ready for the genre the public cares about.

**Public deliverables** (paraphrased from the landing page):

- Last two classes: Engineer, Tutelaire
- Group finder
- "Next phases..." (placeholder for the long-tail roadmap)

**Internal detail**:

- Specs within existing classes (for example, a Gunner can be Assault,
  Marksman, or Chasseur; mirror the structure in `docs/design/classes/`)
- Two additional dungeons (different themes and kits)
- First outdoor zone
- Cosmetic shop
- Community-driven content feedback loop

**After Phase 3**: the long-term vision (space, PvP, open world, crafting)
lives in `docs/content/long-term.md`. It is not part of the public roadmap
yet.
