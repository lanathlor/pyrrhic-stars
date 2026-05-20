# Item System

Every character has 6 equipment slots. Each piece carries exactly 3 stat lines. A full kit provides 18 stat lines total.

## Equipment Slots

| Slot             | Primary Stat     | Secondary Stat | Flex Lines |
| ---------------- | ---------------- | -------------- | ---------- |
| Frame            | Hull             | Plating        | 1          |
| Power Core       | Output           | Tempo          | 1          |
| Primary Weapon   | Output           | Mastery        | 1          |
| Secondary / Tool | Output (smaller) | Identity       | 1          |
| Augment          | Mastery          | Identity       | 1          |
| Module           | —                | —              | 3          |

**Primary** and **Secondary** stat lines are fixed per slot and always present. They define the slot's role in the kit.

**Flex** lines are the gearing decision. What goes in the flex slot depends on item quality (see below).

## Slot Roles

-   **Frame** — the survivability piece. Hull + Plating make this the defensive anchor.
-   **Power Core** — the engine. Output for damage, Tempo for speed. Core throughput piece.
-   **Primary Weapon** — the spec piece. Output + Mastery ties damage directly to spec identity.
-   **Secondary / Tool** — the utility piece. Smaller Output + Identity reinforces class mechanics.
-   **Augment** — the build-defining piece. Mastery + Identity + a flex line = the piece that most shapes your build.
-   **Module** — the wild card. 3 flex lines, no fixed stats. Pure customization.

## Flex Lines and Quality

Flex lines are the only gearing decision in the system. Their behavior depends on item quality grade:

-   **Specialist craft** (highest quality): player picks the flex stat freely. Full build control.
-   **Lower quality**: flex stat is predetermined from a small class-specific whitelist. Still useful, but less control.

There are **no RNG stat rolls**. A given item at a given ilvl with a given quality always has the same stats. Deterministic gearing.

## Item Level Scaling

All stat magnitudes scale with ilvl. The primary/secondary/flex distinction determines which stat appears, not how much. Ilvl determines magnitude via the scaling curves defined in the [Stat System](stats.md).

### Ilvl Bounds

-   **Heritage Floor: ilvl 15** — the non-decaying playable baseline, from open-world solo and easy farm activities. Detailed in the [Seasonal System](seasons.md#heritage-floor-ilvl-15).
-   **M+ entry: ilvl 20** — gear awarded at M+1 (the lowest Overflux M+ tier). Higher Overflux scales rewards up toward the cap. See [Content Tracks](../content/tracks.md#m-track).
-   **Max ilvl: 50** — current-season BiS through normal endgame loops.
-   **Tier 2 craft: ilvl 55** — the only ilvl source above the normal cap. Reserved for Explorer-track-gated crafting (harsh-region world boss materials). Pairs with the higher Heritage Floor of ilvl 19 documented in the [Seasonal System](seasons.md#heritage-floor-ilvl-15).

### Context Bonus

A gear item used in the context of its obtention gets a **+10 ilvl bonus** (the "native track" bonus referenced in [Content Tracks](../content/tracks.md#universal-substrate-designed-once)). M+ gear is +10 ilvl while running M+; hunt gear is +10 ilvl during hunts; raid gear is +10 ilvl in raid. The bonus rewards on-track specialization without locking gear into one activity — every item remains usable cross-track at its base ilvl.

## Stat Budget Distribution

Across a full 6-piece kit (18 stat lines):

| Stat     | Appearances     | Sources                                    |
| -------- | --------------- | ------------------------------------------ |
| Hull     | 1               | Frame                                      |
| Output   | 3 (one smaller) | Power Core, Primary Weapon, Secondary/Tool |
| Plating  | 1               | Frame                                      |
| Tempo    | 1               | Power Core                                 |
| Mastery  | 2               | Primary Weapon, Augment                    |
| Identity | 2               | Secondary/Tool, Augment                    |
| Flex     | 6               | 1 per slot + 3 from Module                 |

The 6 flex lines are where gearing gets interesting. A player can stack one stat, spread evenly, or target specific breakpoints.

## Spec Swapping and Gear

All 6 gear pieces are universal within a class. Changing spec does not require changing equipment. Stats remain identical — Mastery already morphs its effect per spec (e.g., Vanguard Mastery applies as Onslaught in Blade, Devotion in Shield, Afterimage in Shadow).

Weapons visually adapt to the active spec (Vanguard greatsword becomes a tower shield or twin daggers, Gunner rifle becomes a sniper or tactical kit) but remain a single item. No separate farming, no ilvl mismatches, no inventory bloat.

## Open Design Questions

-   **Engineer/Pilot drone stat budget**: does the drone inherit only from Pilot's Mastery + Grid, or from the full kit? Affects how Pilot gear differs from Architect.
-   **Module's 3 flex lines**: pure freedom, or pulled from a class-specific utility pool?
