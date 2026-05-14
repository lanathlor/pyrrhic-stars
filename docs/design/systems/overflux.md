# Overflux System

Single difficulty axis. Replaces traditional "key level + affix" duality. One number, one mental model.

## Mechanics

- Modular conditions, each with 1-5 ranks
- Player chooses which conditions to enable and at what rank per run
- Total Overflux = sum of enabled ranks across selected conditions
- No forced affix rotation

**Design lineage:** Hades Pact of Punishment (rank granularity, player agency) + Halo Skulls (modular toggle aesthetic). Ranked toggles, player-composed loadouts, cosmetic-scaling rewards.

## Reward Structure

- Token tier gated by Overflux threshold (Lesser / Greater / Master / Mythic)
- Quantity scales within tier (more Overflux = more tokens of that tier)
- Higher-tier tokens substitute for lower-tier purchases (no waste)
- Cosmetic-only rewards layer on top, scaling with total Overflux regardless of tier
- Leaderboard score = function of total Overflux + time

## Overflux Condition Libraries

Each track has a shared base plus track-specific conditions.

**Shared base (reusable across tracks):** faster enemies, time pressure, reduced healing, harder defensives, environmental hazards.

**M+ specific:** dungeon-flow modifiers, group-coordinated challenges.

**Hunt specific:**
- Tempered — new boss attacks/phases (Extreme Measures equivalent)
- Wounded Prey — regenerates without sustained pressure
- Pack Hunt — additional adds
- Frenzied — no recovery
- Low Visibility

**Raid specific:** TBD.

**Explorer:** likely doesn't need Overflux (mostly achievement-based).

## Difficulty Curve

- Overflux 0: comfortable for skilled threshold-geared players
- Overflux 20: ~30-50% difficulty increase
- Overflux 40: ~80-120% increase
- Overflux 60+: ~200%+ increase, with new mechanics layered in (Hades Extreme Measures pattern — not just stat inflation)

## UI Requirements

- Starter presets for onboarding ("Beginner: Overflux 5 loadout")
- Progressive condition reveal (don't show full Pact menu in tutorial)
- Current selection and total Overflux visible at a glance
