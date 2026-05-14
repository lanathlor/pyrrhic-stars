# Seasonal System

## Universal Season Clock

One season for everything. M+ seasons, raid tiers, crafting schemas, Overflux condition rotations all share the same clock.

**Cadence:** 3-4 month seasons (PoE league cadence, ARPG industry norm).

**Content release alignment:** Don't release raid mid-M+-season. Align tentpole content to season start. If raid isn't ready, delay the season rather than desync the calendar.

### Content Cadence

Every season has at least one major content addition. Not all tracks every season.

Example seasonal patterns:
- S1: new M+ dungeon + Overflux affixes + crafting recipes
- S2: new raid tier + Overflux affixes
- S3: new harsh region + new hunts + Overflux affixes
- S4: new M+ dungeon + new world boss + recipes

Every season also has universal updates: Overflux rotation, decay tick, leaderboard reset, balance pass. Communicate seasonal spotlight upfront ("Season N spotlight: Adventure track expansion").

## Seasonal Decay

Decay applies at season tick only, NOT continuously. Clean, predictable, single event per season transition.

### Rules

- Purely ilvl-based — no activity tracking, no "active vs inactive" distinction
- Linear squash: decay magnitude scales linearly with ilvl above the ilvl 15 floor — top-ilvl gear loses the most, gear near the floor loses little, gear at or below ilvl 15 loses nothing (see [Heritage Floor](#heritage-floor-ilvl-15))
- Same rule applies to every player uniformly
- Old gear stays at decayed value — NOT restored through play
- Every season is a real rebuild arc, not a maintenance treadmill

### Decay Curve

The squash model is **linear**, fixed by two anchor points: ilvl 20 (entry high-level gear) recalibrates to ilvl 15, and ilvl 50 (max) recalibrates to ilvl 25. The line through those anchors:

    post_decay_ilvl = (ilvl + 25) / 3        (rounded to nearest integer)

Equivalently, the amount squashed is `decay = (2 x ilvl - 25) / 3` — it rises linearly with ilvl, so top gear loses the most and gear near the floor loses little. Two clamps keep it consistent with the [Heritage Floor](#heritage-floor-ilvl-15): the result never drops below ilvl 15, and gear already at or below ilvl 15 is exempt entirely. The same line extends past the cap to cover T2 craft at ilvl 55, which recalibrates to ~ilvl 27 (ilvl bounds defined in the [Item System](../systems/items.md#ilvl-bounds)).

For shipping, the linear curve is discretized into **stepped bands** — easy to communicate, easy to tune per band:

- ilvl 51-55 (T2 craft) -> ilvl 25-27
- ilvl 41-50 -> ilvl 22-25
- ilvl 31-40 -> ilvl 19-22
- ilvl 21-30 -> ilvl 15-18
- ilvl 16-20 -> ilvl 15 (floor clamp)
- ilvl <= 15 -> no decay (Heritage Floor)

### Heritage Floor (ilvl 15)

- The floor is **ilvl 15** — the gear level from open-world solo and easy farm activities, the playable baseline in every season
- Eternal — gear at or below ilvl 15 never decays
- Entry tier for new players
- Return point for inactive players
- Tier 2 craft has a higher floor of ilvl 19 (15 + 4) — permanent advantage for explorer-track investment

### Quality Grade Interaction

Master-crafted gear decays at 50-70% of baseline rate. Compounds over seasons into meaningful gap between basic and master crafts. Quality is persistent on the item across seasons.

### Communication

Don't call it "decay" in UI — use "seasonal recalibration" or similar. Show recent trajectory, highlight gains not losses. Season summary at transition: "Season N closed. Your gear has recalibrated. New gear available at the tier X vendor."

### Cosmetics Never Decay

Visual appearance, glamour, mounts, titles, achievements all permanent. Old-season gear becomes glamour archive — visible record of seasons played.

## Token Economy

### Acquisition

- Deterministic, no RNG drops
- Tokens drop from clears, scaling with [Overflux](overflux.md)
- Spent at vendors for specific gear pieces
- Player chooses what to buy

### Vendor Tier Gating

Overflux tier completion in all dungeons unlocks vendor tier. Completion-based, not currency-grind-based.

### Token Tiers

Lesser -> Greater -> Master -> Mythic. Each tier gated by Overflux threshold. Quantity scales within tier. Higher tokens substitute for lower (no waste).

### Token Reset

Tokens reset at season transition (or convert at unfavorable rate to low-tier carryover). Persistent tokens would defeat the seasonal rebuild loop.
