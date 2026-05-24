# Flux System

Flux is the energy that powers arcanotechnique abilities. It is NOT a simple mana bar.

Three values define a character's Flux capability:

- **Flux Reserve**: maximum energy pool
- **Afflux per Tick**: how much Flux can be mobilized per server tick (translates from TTRPG "per turn"). This is throughput, not capacity
- **Natural Recovery**: passive Flux regeneration per tick

## Ability Usage Rules

- If Afflux per Tick >= ability cost: ability is instant
- If Afflux per Tick < ability cost: ability requires channeling over multiple ticks. Channeling is visible to all players and interruptible by damage or CC
- On interruption: ability is cancelled, Confluence stacks are reset (no Flux lost)

## Flux Commitment System

Before entering combat (or at rest areas), Flux-using classes commit percentages of their Flux Reserve and Regeneration to specific schools. Example: an Arcanotechnicien might go 60% Givre / 40% Feu.

- Committed percentage determines max Flux available for that school
- Committed percentage determines regen rate for that school
- Total must equal 100%
- A 100% single-school build hits harder but has zero versatility
- A 33/33/33 split is versatile but individually weaker
- Swappable at rest areas, not during combat

This creates meaningful pre-fight decisions and endless theorycrafting: "What's the optimal commitment split for this Mythic+ dungeon?"

## Ability Slots

Each character has limited prepared ability slots (base 2 + Intelligence/5, minimum 4). Abilities are selected from the full codex if affinity requirements are met. Swappable at rest areas.
