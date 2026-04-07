# Prime Hub — Implementation Roadmap

## Phase 0: Gather Props

Source CC0 assets from PolyHaven / Sketchfab, model remaining pieces in Blender. Export as GLB to `client/assets/models/environments/prime_hub/`.

**Architecture pieces:**
- Sci-fi wall panels, floor tiles, ceiling sections (modular kit)
- Brutalist concrete/metal structural beams, columns
- Door frames, blast doors, checkpoint barriers
- Railing sections (for perimeter walkways, lift platform)
- Lift/elevator platform and cage structure

**Furniture & props:**
- Military consoles, tactical displays, holographic terminals
- Benches, seating (clean sci-fi style)
- Food stall / vendor cart structures
- Crates, containers (already have one — need variety)
- Weapon racks, workbenches (armory)
- Fountain or monument (plaza centerpiece)
- Street lamp posts

**Vegetation:**
- Trees (mature, tall — not saplings)
- Bushes, ground cover, hanging planters
- Planter boxes (integrated into architecture)

**Lighting fixtures:**
- Recessed strip light housings (mana crystal strips)
- Wall-mounted sconces
- Overhead industrial lights
- Ground-level path markers

---

## Phase 1: Military Building Interior

**Scope:** Lobby, corridor, briefing room, dungeon lift.

- Lobby: reception area, checkpoint gates (open), holographic directory
- Corridor: connects lobby to briefing room and lift
- Briefing room: mission board wall, tactical displays, seating
- Dungeon lift: replaces current portal door — this is the arena launch point
- Locked doors: 2-3 sealed doors with red indicators (future content)
- Lighting: cool blue-white mana crystal strips, recessed ceiling lights
- Collision: all walls, floor, ceiling, props

**Depends on:** Phase 0 (architecture kit, consoles, doors)

---

## Phase 2: Military Building Exterior + Upper Plaza

**Scope:** Open-air rooftop district with night sky.

- Military building exterior: heavy angular facade, 3-4 stories, main entrance
- WorldEnvironment: starfield sky, faint nebula, distant orbital stations
- Garden layout: winding paths, elevation changes, central clearing
- Perimeter walkways: railings, sky views, screenshot spots
- Garden alcoves: benches, small water features
- Lighting: cool infrastructure (blue-white), warm garden (amber), teal plant accents
- Player spawn point: near military building entrance
- Collision: terrain, railings, garden boundaries

**Depends on:** Phase 0 (vegetation, benches, railings, lighting fixtures), Phase 1 (building interior must connect)

---

## Phase 3: Public Lift

**Scope:** Seamless vertical transport between upper and lower levels.

- Open platform design (not enclosed — players see the city during ride)
- Ride duration: 5-8 seconds
- Animation: platform descends/ascends along vertical shaft
- Visual transition: open sky → enclosing architecture, cool → warm color temperature
- Audio transition: quiet wind → urban buzz (placeholder)
- Multi-player capacity, no queue
- Call button / automatic trigger at both ends
- Collision: platform floor, safety railings

**Depends on:** Phase 0 (lift platform, railings), Phase 2 (upper landing), Phase 4 (lower landing)

Note: Phase 3 and 4 can be built in parallel — the lift just needs connection points defined.

---

## Phase 4: Lower Level — Streets & Plaza

**Scope:** Dense enclosed neighborhood below the upper plaza.

### Central Plaza (~80m x 60m)
- Fountain/monument centerpiece (navigation anchor)
- Food stall props (organic placement, not grid)
- E-sport screen placeholders (large wall-mounted)
- Seating clusters (mixed furniture)
- Overhead: visible upper structure / ceiling creating canyon feel

### Street Network
- Main arteries (~8m wide): well-lit, connect plaza to destinations
- Side streets (~4-5m wide): shops, bars, more character
- Alleys (~2-3m wide): shortcuts, atmosphere, dim lighting
- Street layout is organic — bends, forks, dead-ends

### Key Facades
- Residential building: locked entrance, balconies with warm light, NPC life visible
- Bar/social venue: exterior only, music and light spilling out
- Specialty vendor fronts: arcanotechnic repair, clothing, stims

### Lighting
- Mana crystal lamp posts (warm white ~3500K)
- Holographic signage (noisy, translucent, color-coded by district)
- Window spill from apartments and shops (warm yellow-orange)

**Depends on:** Phase 0 (vendor stalls, lamps, fountain, building panels)

---

## Phase 5: Scene Integration

**Scope:** Assemble everything into one seamless zone.

- Combine all parts into single `prime_hub.tscn`
- Update `main.gd`: load `prime_hub.tscn` instead of `hub.tscn`
- Update spawn points (upper level, near military building)
- Wire dungeon lift to existing arena zone transfer (replaces portal door)
- Update server `internal/level/` hub geometry to match new bounds
- Verify: no loading zones, no teleports — walk from streets to lift to plaza to building to dungeon seamlessly
- Performance pass: occlusion, LOD, draw distance for the 500m zone
- Remove old `hub.tscn` (or archive)

**Depends on:** All previous phases.

---

## Dependency Graph

```
Phase 0 (props)
  ├──► Phase 1 (building interior)
  ├──► Phase 2 (upper plaza) ◄── Phase 1
  ├──► Phase 4 (lower streets)
  │
  Phase 3 (lift) ◄── Phase 2 + Phase 4
  │
  Phase 5 (integration) ◄── All
```
