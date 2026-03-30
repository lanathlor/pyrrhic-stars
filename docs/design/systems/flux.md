# Flux System

Flux is the energy that powers arcanotechnique abilities. It is NOT a simple mana bar.

Three values define a character's Flux capability:

- **Flux Reserve**: maximum energy pool
- **Afflux per Tick**: how much Flux can be mobilized per server tick (translates from TTRPG "per turn"). This is throughput, not capacity
- **Natural Recovery**: passive Flux regeneration per tick

## Spellcasting Rules

- If Afflux per Tick >= spell cost: spell is instant
- If Afflux per Tick < spell cost: spell requires channeling over multiple ticks. Channeling is visible to all players and interruptible by damage or CC
- On interruption: half accumulated Flux is lost
- On Flux overload (exceeding Afflux capacity): instability effects (stat debuffs, internal damage, spell failure)

## Flux Commitment System

Before entering combat (or at rest areas), Flux-using classes commit percentages of their Flux Reserve and Regeneration to specific schools. Example: an Arcanist might go 60% Givre / 40% Feu.

- Committed percentage determines max Flux available for that school
- Committed percentage determines regen rate for that school
- Total must equal 100%
- A 100% single-school build hits harder but has zero versatility
- A 33/33/33 split is versatile but individually weaker
- Swappable at rest areas, not during combat

This creates meaningful pre-fight decisions and endless theorycrafting: "What's the optimal commitment split for this Mythic+ dungeon?"

## Spell Slots

Each character has limited prepared spell slots (base 2 + Intelligence/5, minimum 4). Spells are selected from the full codex if affinity requirements are met. Swappable at rest areas.
