# Content Tracks

Everything is a track. Each instantiates the same pattern with different content.

```
Track = (gameplay loop) + (gear path) + (Overflux modulation) + (universal substrate)
```

## Universal Substrate (designed once)

[Overflux](../systems/overflux.md) scaffolding, [token economy](../systems/seasons.md#token-economy), vendor pattern, [seasonal decay](../systems/seasons.md), [context bonus](../systems/items.md#context-bonus) (+10 ilvl in native track), cosmetic permanence, bot testing framework, telemetry/replay/leaderboard infrastructure.

## Track-Specific

Each track defines: gameplay mechanics, Overflux condition library (with overlap), visual/thematic identity, cosmetic rewards, encounter content.

**Current tracks:** M+, Raid, Explorer, Adventure.

---

## M+ Track

5-player instanced group content with Halo+Furi action combat — NOT WoW-clone formula.

**Why M+ first for beta:** content density multiplier (dungeons x Overflux conditions x ranks), 5-player content socially stickier at low concurrent, high replayability per authored hour, solo-dev-friendly cadence, bot framework gives strong tuning signal at this group size.

### Overflux Applied to M+

- M+1 awards ilvl 20 gear; reward ilvl scales up with Overflux toward the cap defined in the [Item System](../systems/items.md#ilvl-bounds)
- Overflux tier completion in all dungeons (4-6 at launch) unlocks vendor tier — all-or-nothing, no sub-cap partial unlock
- Forces engagement with all dungeons
- Overflux composition is player-chosen — no forced weekly affix rotation

### Encounter Design Vocabulary

- "Encounters" not "trash pulls" (Doom Eternal/Halo encounter model)
- Enemy archetypes: flanker, suppressor, area-denier, pressure threat
- Trash design > boss design (encounter density is the main cognitive load)
- Combat tempo, arena geometry, behavior-tree variance are first-class design elements
- No WoW-style affixes. No CC/kick. Overflux conditions modify combat-feel parameters: pattern density, AI aggression, encounter density, parry windows, terrain effects

---

## Adventure Track

Monster Hunter-style hunts. Big complex solo or small-group mobs in the open world.

### Hunt Design

- 5-15 minute fights (shorter than MH proper)
- Each hunt has unique movesets, terrain, mechanics
- Distinct "feel" per hunt — different monster, different mechanics, different terrain
- Bot framework MC-tests cleanly (boss-like single-mob encounters)

### Hunt-Specific Overflux Conditions

Tempered, Wounded Prey, Pack Hunt, Frenzied, Low Visibility (see [Overflux System](../systems/overflux.md)).

### Materials and Crafting

- Standard hunt mats -> Tier 1 craft
- Apex hunts (explorer-gated world bosses) -> Tier 2 craft materials
- Hunt-specific cosmetics and weapons with thematic identity

### Progression

Achievement-based, parallels Explorer model. "Adventurer" is a playstyle/track, not a class. Higher-tier hunts unlock higher-tier materials and cosmetics.

**Open world as primary content surface.** Not vestigial like WoW post-leveling content. Has its own identity, progression, crafting integration, apex bosses.

---

## Explorer Track

Not a class. Any combat class can pursue. Earned through play, not chosen at character creation.

### Earned Via Challenges

- Landmarks (discovery): hidden vistas, ruins, peaks, lore artifacts
- Survival: time in harsh zones, journeys, endurance trials
- NOT earned from world boss kills (those are separate gear rewards)

### Region-Specific Progression

- Each new region = its own exploration track (volcano, frozen wastes, toxic marsh, etc.)
- Progress per region, not globally
- Creates ongoing social value (always need a friend progressed in regions you haven't done)
- Solo-dev content multiplier (every new region adds an exploration track)

### Persistence

- Exploration progression persists across seasons (region unlocks, achievements, lore, cosmetics all permanent)
- Unlocks harsh-region access for Tier 2 craft world bosses
- Cosmetic rewards: titles ("Volcano Surveyor"), region-themed mounts/glamour, account-wide flair

### Group Dynamic

Explorer enables access; group brings combat. Explorer + Adventurer pairing is ideal for Tier 2 craft expeditions.

---

## Raid Track

Lower priority, post-beta addition. Higher authoring bar.

- Larger group content (size TBD)
- Phased multi-mechanic encounters (Furi-style boss fights with bullet hell phases)
- Bot framework for fight tuning
- Probably launches at Season 1 alongside cosmetic shop
