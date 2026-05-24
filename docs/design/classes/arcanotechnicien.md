# Arcanotechnicien

**Gameplay: Tactical Flux Channeling**

Camera: third person, pulled back for spatial awareness. Input: target-based ability usage, Flux management, ability loadout preparation. Core loop: prepare the right loadout, manage Flux throughput, channel at the right moment, avoid interruption.

The Arcanotechnicien is the only class whose combat kit changes between encounters. Where a Gunner always has Overclock and a Vanguard always has their dodge, an Arcanotechnicien selects 6 abilities from a codex of ~35 before each fight. Two Destroyers can have completely different loadouts. The pre-fight ability loadout is half the skill — the other half is executing under pressure while your team keeps you alive.

Flux usage: primary resource. Flux commitment system is central. The Arcanotechnicien is the heaviest Flux user in the game, and the only class where Flux management IS the gameplay rather than a secondary consideration.

| Spec               | Identity           | Playstyle                                                              |
| ------------------ | ------------------ | ---------------------------------------------------------------------- |
| Destroyer          | Glass cannon       | Huge Flux reserve, massive abilities, long channels, vulnerable           |
| Battlemage         | Melee-range hybrid | Lower Flux, instant combat abilities, weaves strikes with arcanotechnique |
| Harmonist (healer) | Flux-based healer  | Healing zones and beams, positioning-based healing, not whack-a-mole   |

---

## Flux Economy

Three values define combat capability (see [Flux System](../systems/flux.md) for full rules):

-   **Flux Reserve** — maximum energy pool. The Identity stat (Flux) scales this directly.
-   **Afflux per Tick** — how much Flux can be mobilized per server tick. The Tempo stat (Channel) scales this.
-   **Natural Recovery** — passive Flux regeneration per tick.

### Instant vs. Channeled

If an ability's cost <= your Afflux per Tick, the ability fires instantly. If an ability costs more than your Afflux, you must **channel** — standing still, visibly gathering Flux, vulnerable to interruption. Channel duration = ability cost / Afflux per Tick (rounded up to whole ticks).

Channeling is visible to every player in the zone. A glowing Flux accumulation builds around the channeler, scaling in intensity with channel progress. This creates the organic "protect the channeler" dynamic — allies can see when a big ability is building and position accordingly.

**Interruption**: any damage or hard CC during a channel cancels it. The accumulated Flux is returned to reserve — no Flux is lost on interruption. However, all Confluence stacks are instantly reset to zero. Getting interrupted on a 3-second Cataclysm doesn't cost Flux, but it destroys your Confluence momentum and wastes the channel time.

**Early Release**: some channeled abilities (marked in the codex) support early release — you can fire the ability before the channel completes for proportional effect. A Fireball released at 70% channel deals ~70% damage. This adds skill expression: holding to the absolute last tick before a telegraph hits, or releasing early to save yourself. Not all abilities support this — Cataclysm-type abilities are all-or-nothing.

---

## Schools

Schools are the disciplines of arcanotechnique — each represents a different way of converting Flux into a physical phenomenon. Not all schools are available to the Arcanotechnicien — some belong to other classes, and the Advanced Scientific schools are locked entirely.

Each school has a clear identity. Knowing what a school does tells you what its abilities will feel like in combat. Schools are grouped by the nature of their Flux conversion.

### Elemental Schools

| School          | Principle                                                                                       | Combat Identity                                                                                             |
| --------------- | ----------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------- |
| **Fire**        | Rapid thermal conversion — Flux into molecular kinetic energy, causing combustion and explosion | High burst damage, AoE, burn DoTs. Expensive but devastating. The "I need this dead now" school.            |
| **Frost**       | Endothermic conversion — absorbs ambient kinetic energy, creating negative thermal gradient     | Zone control, slows, immobilize. High Flux efficiency. The "control the space" school.                      |
| **Electricity** | Electromagnetic conversion — Flux separates charges at the quantum level, producing current     | Chain damage between targets, near-instant activation, tech synergy. The "fast and reactive" school.        |
| **Shadow**      | Photonic absorption — reconverts incident photons into dissipated Flux, creating total darkness | Optical stealth, perception disruption, sensory debuffs. Countered by Light. The "deny information" school. |

### Kinetic Schools

| School           | Principle                                                                                        | Combat Identity                                                                                                      |
| ---------------- | ------------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------- |
| **Aerokinetic**  | Atmospheric conversion — Flux into macroscopic wind, pressure differentials, and weather effects | Projectile deflection, wind zones, push/pull positioning. The "control the air" school.                              |
| **Gravitonic**   | Gravitational conversion — Flux into local spacetime curvature, altering gravity                 | Weight manipulation, area pulls, levitation, force fields. Requires extreme precision. The "reshape space" school.   |
| **Hydrodynamic** | Fluid conversion — Flux controls water and its state transitions                                 | Terrain manipulation, synergy with Frost/Fire, purification. Dependent on nearby water. The "shape the flow" school. |

### Biological Schools

| School               | Principle                                                                                                | Combat Identity                                                                                                                                                                    |
| -------------------- | -------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Bioarcanotechnic** | Biological interface — Flux interacts with biological and cybernetic systems, manipulates nerve impulses | Huge monotarget heals (high Flux cost), neural augmentation, buffs. Expensive but potent. The "raw power healer" school.                                                           |
| **Biometabolic**     | Bioenergetic conversion — Flux manipulates vital field, extracting or stimulating cellular energy        | Life force redistribution: drain ally HP to heal/buff/damage (low Flux), or spend Flux to damage+heal simultaneously (high Flux). Mono and AoE heals. The "triage surgeon" school. |

### Martial, Cognitive & Pure Schools

| School       | Principle                                                                                                                | Combat Identity                                                                                                                                           |
| ------------ | ------------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Martial**  | Vital energy channeling — not traditional Flux manipulation but channeling one's own energy to transcend physical limits | Combat augmentation, superhuman reflexes, weapon infusion. No Flux cost — uses cooldowns instead. Battlemage-exclusive. The "transcend the body" school.  |
| **Illusion** | Cognitive manipulation — Flux generates informational signals that alter perception and hack cybernetic implants         | Confusion, false targeting, data theft, perception falsification. No physical damage. Countered by hardened minds. The "break the mind" school.           |
| **Pure**     | Raw Flux manipulation — acts on arcanotechnology itself without channeling through a physical phenomenon                 | Ability negation, Flux burn, dispel, amplification. No effect on physical matter. The most versatile and most difficult school. The "counter-arcanotechnique" school. |

### Schools Belonging to Other Classes

The following schools exist in arcanotechnic theory but are not part of the Arcanotechnicien's codex. They are the domain of other classes:

-   **Light** — Tutelaire. Solid light barriers, shields, damage reflection.
-   **Kinesthetic** — Blade Dancer. Pure kinetic energy, telekinetic force, impact absorption.

### Inaccessible: Advanced Scientific Schools

The following schools are not available for player use. They represent military-restricted research, lost techniques, or disciplines too unstable for field deployment.

-   **Ballistic** — trajectory optimization, railgun physics
-   **Magnetic** — magnetic field generation, metal manipulation
-   **Sonic** — sound wave manipulation, destructive resonance
-   **Chronodynamic** — local temporal flow manipulation
-   **Quantum** — quantum state manipulation, teleportation

These schools may appear in enemy abilities, environmental hazards, or lore encounters — but never in the player codex.

### School Affinity

Each spec has **primary** and **secondary** school affinities. Primary schools have full effectiveness — abilities cost base Flux and channel at base speed. Secondary schools work but at +25% Flux cost. Schools outside your affinity are accessible but at +50% cost — worth it for niche utility, punishing for core rotation.

| Spec       | Primary Schools                       | Secondary Schools         |
| ---------- | ------------------------------------- | ------------------------- |
| Destroyer  | Fire, Frost, Electricity              | Gravitonic, Aerokinetic   |
| Battlemage | Electricity, Fire, Martial            | Shadow, Aerokinetic       |
| Harmonist  | Bioarcanotechnic, Biometabolic, Frost | Aerokinetic, Hydrodynamic |

All specs can access Pure as a secondary school — counter-ability use is a core Arcanotechnicien skill regardless of spec.

---

## The Codex (Ability Loadout)

The Arcanotechnicien's codex contains ~40 abilities across all schools. Before combat (at rest areas, between encounters), you prepare a **loadout of 6 abilities** from the codex. These are your combat abilities for the next fight.

### Ability Types

Every ability has a School (which element) and a Type (what it does mechanically):

| Type             | Function                              | Examples                               |
| ---------------- | ------------------------------------- | -------------------------------------- |
| **Destruction**  | Direct damage — instant or channeled  | Fireball, Ice Javelin, Chain Lightning |
| **Affliction**   | DoT, debuffs, persistent effects      | Frostbite, Burn, Static Field          |
| **Protection**   | Shields, barriers, damage reduction   | Frost Ward, Gale Shield, Force Shell   |
| **Enhancement**  | Buffs, self-empowerment, augmentation | Overclock, Lightning Blade, Surge      |
| **Displacement** | Movement, repositioning, teleport     | Shadow Step, Gust Step                 |

### Preparation Rules

-   **6 ability slots** — your full combat kit. Choose wisely.
-   **School affinity applies** — off-affinity abilities cost more Flux (see above).
-   **No duplicate types required** — you can bring 6 Destruction abilities if you want. You'll have zero utility, but you can.
-   **Swappable at rest areas** — between pulls, between dungeon bosses, at hub. Not during combat.
-   **Saved loadouts** — save and name loadout presets for quick swapping. "AoE Boss," "Single Target," "Progression Safe."

### Why This Matters

Every other class walks into every fight with the same abilities. The Arcanotechnicien walks in with a hypothesis: "This boss has a burn phase at 30%, so I'm bringing two Frost control abilities to survive it, plus three Fire nukes for the opening, and a Gravitonic pull for the add wave."

Wrong hypothesis? You have the wrong tools. Right hypothesis? You have exactly the right tools and outperform anyone. The skill floor is picking a reasonable generalist loadout. The skill ceiling is optimizing loadouts per encounter, per party composition, per Overflux condition.

This is the theorycrafting class. The community will maintain spreadsheets of optimal loadouts per boss. Arguments about whether Gravitonic Collapse or Frost Nova is better for the second phase of a specific encounter are the Arcanotechnicien endgame.

---

## Flux Commitment

Before combat, you distribute your Flux Reserve and Regeneration across the schools in your loadout. Total must equal 100%.

**Example**: a Destroyer bringing 3 Fire abilities and 3 Frost abilities might commit 60% Fire / 40% Frost. This means:

-   60% of Flux Reserve is available for Fire abilities, regenerates at 60% of natural recovery rate
-   40% of Flux Reserve is available for Frost abilities, regenerates at 40% of natural recovery rate

### Commitment Tradeoffs

-   **100% single-school**: maximum power in one school, zero versatility. If the fight demands that school, you're a god. If it doesn't, you're useless.
-   **50/50 split**: balanced but neither school hits as hard. Good for encounters with mixed demands.
-   **60/30/10 three-way**: primary school for damage, secondary for control, a sliver for emergency utility. The default "safe" split.

Commitment is set alongside your loadout at rest areas. Part of the same pre-fight preparation. The loadout answers "what can I commit?" and commitment answers "how much can I commit of each?"

---

## Confluence (Shared Class Mechanic)

The Arcanotechnicien's class-wide combat mechanic. Confluence represents aligned Flux streams — the state where your channeling flows without resistance.

**Building Confluence**: each successfully completed ability (instant or fully channeled) adds 1 stack. Max 5 stacks.

**Confluence effects** (per stack):

-   +8% ability power (damage, healing, zone size, buff strength)
-   -3% channel time on channeled abilities
-   Visual escalation: Flux particles orbit the channeler, growing denser and brighter per stack

**Losing Confluence**: getting interrupted drops ALL stacks to zero. Not committing for ~4 seconds begins a 1-stack-per-second decay. Taking damage does NOT drop stacks (only interruption does) — you can eat hits and maintain Confluence as long as you're not mid-channel when they land.

**The Tension**: Confluence rewards sustained committing. But sustained committing means sustained vulnerability. A Destroyer at 5 stacks is dealing 40% bonus damage — but they've been standing still channeling for 10+ seconds to get there. The team has been protecting them. One interrupt resets everything.

Each spec interacts with Confluence differently through their Mastery mechanic (see spec sections).

---

## Destroyer — Overcharge System

Massive AoE burst. Glass cannon. The skill is finding safe windows to channel devastating abilities without getting interrupted.

### Core Mechanic: Overcharge

Channeled abilities can be **held past their completion point**. Once a channel completes, you enter Overcharge — continuing to hold pours additional Flux into the ability, increasing its damage, AoE radius, or duration beyond base values.

Overcharge builds Instability. A visible meter fills as you hold. Release the ability during the **safe window** (the green zone on the meter, scaled by Mastery) for maximum bonus damage. Hold past the safe window into the **red zone** and the ability destabilizes — reduced damage, self-damage from Flux backlash, and a brief commit lockout.

The Mastery stat (Overcharge) widens the safe window and increases the bonus damage ceiling. At low Mastery, the safe window is tight and the risk of backlash is high. At high Mastery, you have a comfortable margin to hold for massive damage.

### Innate: Cataclysm Resonance

When a Destroyer completes a channeled ability (with or without Overcharge), their next ability within 3 seconds gains **Cataclysm Resonance** — increased AoE radius (+30%) and a brief channel speed bonus. This chains big abilities together: finish a Fireball, immediately begin a Frost Nova with wider radius and faster channel.

Cataclysm Resonance is what makes Destroyer feel like a cascading avalanche of destruction. Each completed ability accelerates the next one. But the chain breaks instantly on interruption — you lose Confluence AND the Resonance bonus.

### Camera and Feel

The Destroyer's third-person camera is pulled furthest back of any Arcanotechnicien spec. You need to see the entire arena — incoming telegraphs, ally positions, enemy clusters for AoE targeting. The HUD emphasizes the Overcharge meter (center-bottom, impossible to miss) and Flux commitment bars (segmented by school).


Channeling feels heavy and deliberate. Flux particles stream from the environment toward the channeler, the ground cracks with energy, and the ability effect builds visually at the target location before release. A fully Overcharged Fireball is a miniature sun forming over the target area before detonation. It should feel like you're building a bomb.

### Example Loadout (AoE Boss)

| Slot | Ability             | School      | Type        | Notes                                             |
| ---- | ------------------- | ----------- | ----------- | ------------------------------------------------- |
| 1    | Fireball            | Fire        | Destruction | Long channel, massive AoE, supports early release |
| 2    | Frost Nova          | Frost       | Destruction | Medium channel, AoE burst + slow field            |
| 3    | Chain Lightning     | Electricity | Destruction | Instant, chains between nearby targets            |
| 4    | Gravitonic Collapse | Gravitonic  | Destruction | Channel, pulls enemies together then damages      |
| 5    | Gale Shield         | Aerokinetic | Protection  | Instant, wind barrier deflects projectiles        |
| 6    | Frost Ward          | Frost       | Protection  | Instant, frost barrier on self, explodes on break |

### Destroyer Loop

Position at range → assess enemy positions → commit to Fireball channel → Overcharge into safe window → release → Cataclysm Resonance triggers → immediate Frost Nova with bonus radius → Overcharge again → release → Chain Lightning for instant damage during repositioning → Gravitonic Collapse to group adds → Fireball the cluster → Frost Ward or Gale Shield if a telegraph targets you → repeat.

The rhythm is: **channel → overcharge → release → chain → channel → overcharge → release**. Safe windows between boss mechanics are your DPS windows. Dangerous phases are for repositioning, instant abilities, and shield uptime.

### Skill Expression

-   **Beginner**: channels abilities to base completion without Overcharging, releases immediately. Never builds Confluence past 2 stacks because they stop committing to dodge telegraphs. Uses Frost Ward reactively after taking damage. Still deals respectable burst — the abilities hit hard at baseline.
-   **Competent**: Overcharges into the safe window reliably. Maintains 3-4 Confluence stacks by chaining abilities during openings. Pre-positions at range to minimize interrupt risk. Manages Flux commitment to never run dry on their primary school mid-fight.
-   **Expert**: Overcharges to the edge of the safe window for maximum bonus on every channeled ability. Maintains 5 Confluence stacks through entire fight phases by threading channels between telegraphs. Chains Cataclysm Resonance across 3-4 abilities in a single burst window. Early-releases abilities at the exact moment needed to dodge, preserving Confluence. Swaps loadouts between pulls to optimize for each encounter's AoE patterns. The team builds around protecting them during channel sequences.

### Risk / Reward

Destroyer is the highest AoE burst in the game, but only if uninterrupted. A single interrupt during a 5-stack Overcharged channel doesn't just cancel one ability — it resets Confluence, wastes Flux, triggers Overcharge backlash, and kills your Cataclysm Resonance chain. You go from peak output to zero in one hit.

This makes Destroyer the most team-dependent DPS spec. A Destroyer with a Shield Vanguard blocking for them during channels is devastating. A solo Destroyer with no protection channels nothing safely and deals mediocre damage with instant abilities. The class fantasy is a weapon of mass destruction that requires an escort.

Encounters with constant unavoidable AoE damage are Destroyer's nightmare — every tick is a potential interrupt. Encounters with long, predictable safe windows between dangerous phases are paradise. This creates genuine loadout and spec decisions per boss.

---

## Battlemage — Weave System

Melee-range hybrid. Monotarget, constant damage. The skill is maintaining a perfect rhythm of alternating weapon strikes and abilities without breaking the chain.

### Core Mechanic: Weave

Alternating between a **melee weapon strike** and an **ability** grants a stacking damage bonus (max 8 stacks, scaled by Mastery). Each alternation adds a stack. Two consecutive melee strikes or two consecutive abilities break the chain and reset stacks to zero.

The rhythm is strict: strike → ability → strike → ability. At 8 stacks, every hit (weapon or ability) deals significant bonus damage. The Mastery stat (Weave) scales the per-stack bonus and adds a brief grace period between actions before the chain breaks.

Weave transforms the Arcanotechnicien from a stationary channeler into a melee combatant who happens to use arcanotechnique. You're in the boss's face, sword in one hand, Flux crackling in the other, alternating between physical and arcane damage in a relentless rhythm.

### Weapon: Flux Blade

A one-handed sword infused with electromagnetic Flux. The Battlemage's basic attack — a fast melee swing with no Flux cost. Deals physical damage scaled by Output. The weapon exists to be the "strike" half of the Weave chain.

The Flux Blade has a 3-hit combo: lateral slash → upward cut → thrust. Each hit in the combo is slightly slower but deals more damage. You don't need to complete the combo — any single hit counts as a "strike" for Weave purposes. But completing the full 3-hit combo before the next ability grants a small bonus to the ability's damage (momentum carry).

### Innate: Close Quarters

Abilities committed within melee range (5m of target) have their channel time reduced by 50%. Abilities that would normally require channeling at range become instant or near-instant in melee. This is the mechanic that makes Battlemage work — you're not a channeler who sometimes melees, you're a melee fighter whose abilities are instant because you're always in the enemy's face.

Close Quarters does NOT reduce Flux cost — only channel time. Expensive abilities are still expensive. You'll drain your reserve faster than a Destroyer because you're committing more frequently (every other GCD), even though each individual ability costs less than a Destroyer's Overcharged nukes.

### Camera and Feel

Tighter third-person camera than Destroyer — closer to Vanguard's over-the-shoulder perspective. You need to see melee range, not the whole arena. The HUD emphasizes the Weave stack counter (prominent, center) and the Flux Blade combo indicator.

Combat feels aggressive and rhythmic. The Flux Blade crackles with electricity on each swing. Abilities fire from the off-hand in quick bursts between strikes — a Lightning Blade infuses the weapon, a Static Discharge arcs from your palm mid-combo, a Shadow Step repositions you behind the boss. The fantasy is a warrior-mage in constant, fluid motion.

### Example Loadout (Single Target Boss)

| Slot | Ability          | School      | Type         | Notes                                                              |
| ---- | ---------------- | ----------- | ------------ | ------------------------------------------------------------------ |
| 1    | Lightning Blade  | Electricity | Enhancement  | Instant, infuses weapon with chain lightning for 3 strikes         |
| 2    | Static Discharge | Electricity | Destruction  | Instant at melee range (Close Quarters), AoE burst around target   |
| 3    | Ignition         | Fire        | Destruction  | Instant, single-target burst + burn DoT. Weave filler              |
| 4    | Adrenaline       | Martial     | Enhancement  | No Flux cost, +20% attack speed for 6s. Tightens Weave rhythm      |
| 5    | Shadow Step      | Shadow      | Displacement | Instant, short blink behind target. Repositioning + Weave filler   |
| 6    | Flux Armor       | Electricity | Protection   | Instant, brief electromagnetic shield. Attackers take shock damage |

### Battlemage Loop

Engage at melee range → Flux Blade slash → Lightning Blade (Weave +1) → slash → Static Discharge (Weave +2) → slash → Adrenaline (Weave +3, no Flux cost) → slash → Ignition (Weave +4) → continue alternating → at 8 stacks, every hit is amplified → maintain rhythm through dodges → Shadow Step back after forced repositioning → Flux Armor for unavoidable damage → never break the chain.


The rhythm is a metronome: **strike-ability-strike-ability**. Boss mechanics that force you to stop swinging (long dodges, phase transitions, forced range) are your enemy — they break Weave. Mechanics that you can dodge through while maintaining melee range are opportunities.

### Skill Expression

-   **Beginner**: alternates strike and ability loosely, maintains 2-3 Weave stacks. Uses abilities at range, missing Close Quarters bonus. Breaks the chain frequently by double-striking or panicking into consecutive dodges. Still deals decent damage — the weapon hits hard and the abilities are instant at melee range.
-   **Competent**: maintains 5-6 Weave stacks reliably. Weaves dodges into the strike-ability rhythm without breaking the chain. Uses Shadow Step to stick to repositioning bosses. Manages Flux to never run dry mid-chain. Consistent, relentless DPS.
-   **Expert**: maintains 8 Weave stacks through entire fight phases. Times full Flux Blade combos to land the 3-hit bonus on key abilities. Weaves dodges, repositioning, and defensive abilities into the rhythm without ever double-striking or double-committing. Pre-positions so boss mechanics don't force range. The Weave chain becomes unconscious — pure flow state. DPS output is the highest sustained single-target among Arcanotechnicien specs.

### Risk / Reward

Battlemage is the Arcanotechnicien that plays like a Vanguard. You're in melee range, taking the same risks as a Blade or Shadow, but without stamina-based dodging or a physical parry. Your defense is Flux Armor, Shadow Step, and not getting hit.

The reward is versatility: you deal physical AND Flux damage, making you effective against enemies resistant to either one. You're also the least team-dependent Arcanotechnicien — no long channels that need protecting, no stationary vulnerability windows. You protect yourself by being in constant motion.

The punishment is precision. Weave demands perfect alternation. Under pressure — dodging telegraphs, chasing a repositioning boss, managing Flux — maintaining strike-ability-strike-ability becomes genuinely difficult. Breaking the chain at 7 stacks because you panic-dodged into a double-strike is devastating. And unlike Vanguard, you have no parry, no block, and no stamina-based i-frames. Your survival is pure positioning and ability-based mobility.

---

## Harmonist — Harmony System

Flux-based healer. Positional, channeled, visible, interruptible. The anti-whack-a-mole healer: you don't click health bars, you redistribute life force across the party.

The Harmonist draws from two healing schools that play fundamentally differently. **Bioarcanotechnic** abilities are expensive but powerful — massive single-target heals and beams that burn through Flux. **Biometabolic** abilities are Flux-cheap but cost ally HP — draining one ally to heal another, redistributing health across the group. The Harmonist's skill is mixing both: Biometabolic for sustain, Bioarcanotechnic for emergencies.

Harmonist healing is channeled and visible, just like offensive Arcanotechnicien abilities. Allies must stand in your healing zones or stay near your beams. The Harmonist must be protected during channels. This is not a healer who hides in the back — this is a healer who is the most important target in the fight.

### Core Mechanic: Harmony

Healing an ally with a **different delivery method** than the last heal on that target triggers a **Harmony proc** — a bonus heal scaled by Mastery.

Three delivery methods:

-   **Zone** — persistent ground effect that heals allies standing in it
-   **Beam** — channeled tether between channeler and target, heals while maintained
-   **Direct** — instant or short-commit heal on a specific target

Applying Zone → Beam → Direct to the same ally triggers Harmony twice (on the Beam and on the Direct). Applying Zone → Zone → Zone triggers it zero times. The Mastery stat (Harmony) scales the bonus heal magnitude and adds a brief HoT (heal over time) to each Harmony proc.

This creates a weaving pattern: rotate delivery methods on each ally to maximize Harmony procs. Spamming one type of heal is safe but inefficient. Cycling all three on a target under pressure is optimal but demands attention, positioning, and uninterrupted channels.

### Innate: Sympathetic Field

The Harmonist passively generates a **Sympathetic Field** — a visible area around them (radius scaled by Identity: Flux stat) where all healing effects are amplified by 15%. Allies inside the field receive more healing from zones, beams, and direct heals alike.

This forces the Harmonist to position centrally. You want your field covering as many allies as possible, which means you're near the action, not hiding at max range. The Sympathetic Field is also visible to enemies — in the lore, it's a concentration of stabilized Flux that glows softly. The boss (and in PvP, opponents) can target the Harmonist by following the glow.

### Healing Delivery Methods

**Zones** — place a persistent healing area on the ground. Allies inside heal per tick. Multiple zones can overlap. Zones persist for a duration (no concentration required after placement), but you can only maintain a limited number simultaneously (scaled by Flux stat). Placing a new zone beyond the limit despawns the oldest.

**Beams** — channeled connection between you and a target. Highest single-target healing throughput, but locks you in place and requires line-of-sight to the target. The beam is a visible Flux stream — allies can see who you're healing, and enemies can see your channel. Breaking line-of-sight or getting interrupted ends the beam.

**Direct** — instant or short-commit heal on a target. Lowest throughput but no channel, no positioning requirement beyond range. The emergency tool — someone is about to die and you can't set up a zone or beam in time.

### Camera and Feel

Camera pulled back further than Battlemage but similar to Destroyer — you need to see the arena, ally positions, and incoming telegraphs. The HUD emphasizes ally health (party frames), active zone count, and Harmony proc tracking per ally (which delivery method was used last).

Healing feels like conducting an orchestra. Zones glow on the ground in soft bioluminescent patterns. Beams are visible Flux streams connecting you to your target. Direct heals pulse outward from your hands. At high Confluence stacks, the entire area around you hums with restorative energy. The fantasy is a field medic channeling life force — visible, vital, and vulnerable.


### Example Loadout (Dungeon Healing)

| Slot | Ability       | School           | Type         | Notes                                                              |
| ---- | ------------- | ---------------- | ------------ | ------------------------------------------------------------------ |
| 1    | Mending Surge | Bioarcanotechnic | Enhancement  | Direct. Massive single-target emergency heal. High Flux cost.      |
| 2    | Mending Beam  | Bioarcanotechnic | Enhancement  | Beam. High sustained single-target throughput. Channel. Expensive. |
| 3    | Life Swap     | Biometabolic     | Enhancement  | Direct. Drain healthy ally → empower next heal. Low Flux cost.     |
| 4    | Transfusion   | Biometabolic     | Enhancement  | Beam→Zone. Drain one ally, AoE heal everyone else. Low Flux cost.  |
| 5    | Frost Ward    | Frost            | Protection   | Instant. Frost barrier on ally. Absorbs damage, explodes.          |
| 6    | Gust Step     | Aerokinetic      | Displacement | Instant. Wind-propelled repositioning for Sympathetic Field.       |

### Harmonist Loop

Pre-position centrally → Life Swap the healthy Shield Vanguard → empowered Mending Surge on the dying DPS (Harmony proc: Direct after Direct from different schools) → Mending Beam on the next target taking damage → Transfusion on the Vanguard again to AoE heal the melee cluster → break beam → Frost Ward on whoever is about to eat a telegraph → Gust Step to reposition Sympathetic Field → Life Swap again → cycle.

The rhythm weaves both schools: **Biometabolic to fuel, Bioarcanotechnic to deliver**. Biometabolic abilities are cheap but drain your allies — you're constantly making triage decisions about who can afford to lose HP right now. Bioarcanotechnic abilities are expensive but powerful — you use them when someone needs saving NOW and you can't afford to set up a drain. The interplay between the two schools is the Harmonist's unique skill expression.

### Skill Expression

-   **Beginner**: only uses Mending Surge and Mending Beam. Never touches Biometabolic abilities (too scary to drain allies). Burns through Flux in 30 seconds and has nothing left. Healing is functional but unsustainable — the "I only know Bioarcanotechnic" Harmonist.
-   **Competent**: uses Life Swap on the tank between big hits to fuel empowered heals. Uses Transfusion during safe windows for efficient AoE healing. Mixes both schools to sustain Flux throughout the fight. Cycles delivery methods (beam/direct) for regular Harmony procs. Positions Sympathetic Field to cover 2-3 allies. Consistent, sustainable healing.
-   **Expert**: reads the fight 2-3 GCDs ahead. Life Swaps the healthiest ally at the exact moment before a tank buster so the empowered heal lands on the tank instantly after the hit. Chains Transfusion on a Shield Vanguard during their Brace (they can afford the drain) to top the entire group. Never runs out of Flux because Biometabolic sustains the rotation. Burns Bioarcanotechnic Flux only on genuine emergencies — Mending Surge when someone would die without it, Mending Beam when sustained single-target healing outpaces what Biometabolic redistribution can cover. Manages ally HP pools like a resource — the group is never at full HP, but never in danger. The Harmonist IS the group's HP bar.

### Risk / Reward

Harmonist is the healer that treats the entire party's HP as a shared resource. Biometabolic abilities move health around — taking from the healthy to give to the dying, draining one ally to heal the group. Bioarcanotechnic abilities are the raw power backup — massive heals that burn through Flux when redistribution isn't enough.

The reward is unmatched healing versatility. A Harmonist with both schools can sustain AoE healing for an entire fight (via cheap Biometabolic redistribution), burst-heal a tank through a lethal hit (via expensive Bioarcanotechnic surge), and everything in between. No other healer (Tutelaire's Luminary) has this range.

The risk is trust. Biometabolic healing hurts your allies. A Life Swap at the wrong moment — draining someone about to eat a telegraph — kills them. A Transfusion on an ally who's already low wipes them. The Harmonist must know every ally's HP, every incoming damage event, and every safe window for draining. Misread the fight and your healing kills more people than it saves.

The secondary risk is Flux management. Leaning too hard on Bioarcanotechnic burns you dry. Leaning too hard on Biometabolic leaves your team perpetually low. The optimal play mixes both — Biometabolic sustain with Bioarcanotechnic punctuation — and that mix changes per encounter, per party, per phase.

Encounters are designed to be completable without a healer at sufficient player skill (all damage is avoidable). The Harmonist's value is lowering the execution bar for the team — covering mistakes, enabling greedier DPS positioning, and turning "one hit kills you" into "one hit is survivable." In progression, the Harmonist is the difference between 50 pulls and 200 pulls.

---

## The Codex — Ability List

The full list of abilities available to Arcanotechnicien. Each ability belongs to a School and a Type. Flux costs are relative (Low/Medium/High/Extreme) — actual values depend on character level and gear.

### Fire

**Fireball** — Destruction. High cost. Channeled (supports early release). Massive AoE explosion at target location. Radius and damage scale with channel completion. The Destroyer's signature nuke. At full Overcharge, it leaves a brief burning ground effect.

**Ignition** — Destruction. Medium cost. Instant. Single-target burst that applies a strong burn DoT. Less raw damage than Fireball but instant activation makes it Weave-compatible. The Battlemage's fire option.

**Burn** — Affliction. Low cost. Instant. Applies a burn DoT to target. Low individual damage but efficient Flux-per-damage. Good filler, good for maintaining pressure.

**Flame Wall** — Protection. Medium cost. Channeled. Creates a wall of fire at target location for 6 seconds. Enemies passing through take heavy damage and are briefly slowed. Zone denial tool.

### Frost

**Frost Nova** — Destruction. Medium cost. Channeled (supports early release). AoE burst centered on target location. Deals damage and leaves a slow field for 4 seconds. The control-damage hybrid.

**Ice Javelin** — Destruction. Medium cost. Instant. Single-target projectile. High damage, applies a brief root on hit. The Frost school's burst single-target option.

**Frostbite** — Affliction. Low cost. Instant. Applies a stacking slow + damage-over-time. Each stack increases slow magnitude. At 3 stacks, the target is immobilized for 2 seconds.

**Frost Ward** — Protection. Medium cost. Instant. Applies a frost barrier to target ally. Absorbs damage. When the barrier breaks or expires, it explodes for minor AoE damage and applies Frostbite to nearby enemies.

### Electricity

**Chain Lightning** — Destruction. Medium cost. Instant. Hits primary target then chains to up to 3 nearby enemies. Each chain deals reduced damage. The instant AoE option.

**Lightning** — Destruction. High cost. Channeled. Single-target, devastating damage. Near-instant travel time. The Electricity school's big single-target nuke.

**Lightning Blade** — Enhancement. Medium cost. Instant. Infuses your weapon (or Flux Blade) with electricity for 10 seconds. Weapon strikes chain a small lightning arc to one nearby enemy. The Battlemage's bread-and-butter enhancement.

**Static Field** — Affliction. Medium cost. Instant. Places a persistent field at target location for 8 seconds. Enemies inside take ticking lightning damage and have reduced attack speed.

**Surge** — Enhancement. Medium cost. Instant. Channels electromagnetic Flux along your limbs, granting +20% attack speed for 6 seconds. Flux arcs visibly between your joints. The Battlemage's burst window opener.

**Flux Armor** — Protection. Medium cost. Instant. Wraps the channeler in a crackling electromagnetic field for 4 seconds. Absorbs incoming damage. Enemies who strike the channeler in melee take shock damage. The Battlemage's defensive tool.

### Shadow

**Shadow Step** — Displacement. Low cost. Instant. Short-range blink (8m) in facing direction. Leaves a shadow decoy at origin for 2 seconds that draws one attack. The universal repositioning tool.

**Veil** — Enhancement. Medium cost. Instant. Grants brief optical camouflage (3 seconds). Reduces enemy targeting priority. Not true invisibility — AoE still hits you — but buys breathing room.

**Dread** — Affliction. Medium cost. Instant. Debuffs target: reduced damage output for 6 seconds. A strong defensive debuff for high-damage phases.

### Aerokinetic

**Gale Force** — Destruction. Medium cost. Channeled (supports early release). Cone of pressurized wind that pushes enemies back and deals damage over the channel duration. Longer channel = more pushback. Area denial meets damage.

**Gale Shield** — Protection. Medium cost. Instant. Creates a swirling wind barrier around channeler for 4 seconds. Deflects incoming projectiles and reduces ranged damage taken. Melee attacks unaffected.

**Gust Step** — Displacement. Low cost. Instant. Wind-propelled dash (10m) in facing direction. Faster than Shadow Step, no decoy, but leaves a brief tailwind that gives allies who pass through it +10% move speed for 2 seconds.

**Soothing Wind** — Enhancement. Medium cost. Instant. Places a gentle wind zone at target location for 8 seconds. Allies inside gain minor move speed and heal per tick. The Harmonist's off-school zone option.

### Gravitonic

**Gravitonic Collapse** — Destruction. Extreme cost. Channeled. Creates a gravity well at target location. Pulls all enemies within 10m toward center over 2 seconds, then detonates for massive AoE damage. The ultimate "group them and nuke them" setup tool.

**Force Shell** — Protection. Medium cost. Instant. Surrounds channeler in a gravity field for 3 seconds. Incoming projectiles are deflected. Melee attacks against the channeler deal reduced damage. Does not prevent telegraphed AoE.

**Gravity Crush** — Destruction. High cost. Instant. Single target. Increases local gravity on the target — heavy damage plus a 2-second root (they can't move under the weight). Strong burst CC.

### Hydrodynamic

**Torrent** — Destruction. Medium cost. Channeled. Directed stream of pressurized water at target. Sustained damage that pushes the target slowly backward. Low damage per tick but continuous and disruptive.

**Purifying Mist** — Enhancement. Medium cost. Instant. Places a water mist zone at target location for 6 seconds. Allies inside are cleansed of one DoT effect on entry and gain minor damage reduction. Situational but powerful in DoT-heavy encounters.

### Bioarcanotechnic

Expensive, powerful, monotarget. The "emergency room" school — each ability costs a lot of Flux but hits hard.

**Mending Surge** — Enhancement. High cost. Instant. Massive single-target heal on an ally. The biggest heal in the game per commit. Burns through your Flux reserve fast — you can't spam this, but when someone is about to die, nothing else compares.

**Mending Beam** — Enhancement. High cost. Channeled. Tethers channeler to target ally. Heals for a large amount per tick while channeled. Highest sustained single-target healing throughput. Line-of-sight required. Expensive to maintain — drains Flux every tick.

**Overclock** — Enhancement. Medium cost. Instant. Buffs target for 6 seconds: +15% attack speed, +10% movement speed. Works on self or ally. The offensive support tool.

**Neural Fortification** — Protection. High cost. Instant. Buffs target ally for 6 seconds: +20% damage reduction, immunity to one interrupt effect. The "protect the channeler" ability — a Harmonist can commit this on a Destroyer mid-channel to armor them against one interruption.

**Restoration Matrix** — Enhancement. High cost. Instant. Places a bioarcanotechnic healing zone at target location for 10 seconds. Allies inside heal at high throughput per tick. Expensive to place but powerful sustained AoE healing. The Bioarcanotechnic primary-school Zone heal.

**Neural Purge** — Enhancement. Medium cost. Instant. Cleanses all Flux-based debuffs and one non-Flux debuff from target ally. Grants 2-second debuff immunity after cleanse. The primary-school cleanse.

**Regeneration Protocol** — Enhancement. Medium cost. Instant. Applies a strong HoT to target ally for 12 seconds. If the ally drops below 30% HP while the HoT is active, remaining ticks are consumed instantly as a burst heal. The "insurance policy" — place before a dangerous phase.

### Biometabolic

Life force redistribution. Two modes: drain ally HP as fuel (low Flux cost), or spend Flux to damage enemies and heal allies simultaneously (high Flux cost). This is where the Harmonist's AoE and multi-target healing lives.

**Life Swap** — Enhancement. Low cost. Instant. Drains a portion of target ally's HP and stores it as vital charge. The next healing ability you commit within 4 seconds is empowered by the stored charge (bonus healing proportional to HP drained). The core Biometabolic mechanic — take health from someone healthy to supercharge your next heal on someone dying. Allies must trust their Harmonist.

**Transfusion** — Enhancement. Low cost. Channeled. Tethers channeler to an ally, draining their HP per tick. Simultaneously heals all other allies within 10m for the same amount. The AoE heal — powered by one ally's sacrifice. Best used on a high-HP Shield Vanguard who can afford to donate. Interruptible.

**Vital Circuit** — Enhancement. Low cost. Instant. Links two allies for 8 seconds. Damage taken by either ally is split evenly between them. When the link expires, the ally with lower HP is healed for a portion of the difference. HP equalization — keeps both targets alive longer against focused damage.

**Metabolic Burst** — Enhancement. High cost. Instant. Deals moderate damage to target enemy and heals all allies within 8m of the target for a portion of the damage dealt. The Flux-expensive mode — no ally HP cost, but burns your reserve. The "I need AoE healing NOW and nobody is healthy enough to drain" emergency.

**Vital Drain** — Destruction. Medium cost. Channeled. Tethers channeler to target enemy. Drains health per tick, healing the channeler for a portion of damage dealt. The self-sustain option for any Arcanotechnicien without a healer. Beam is visible and interruptible.

**Metabolic Disruption** — Affliction. Medium cost. Instant. Debuffs target: -15% healing received, -10% movement speed for 8 seconds. Reduces enemy sustain. Niche in PvE, powerful in PvP.

**Vital Bloom** — Enhancement. Low cost. Instant. Sacrifices a portion of the channeler's HP to create a healing zone at target location for 8 seconds. Heal per tick is proportional to HP sacrificed — more health given, stronger zone. The Biometabolic primary-school Zone heal. Self-sacrifice fits the school's identity: HP is the currency.

**Siphon Pulse** — Destruction. Low cost. Instant. Deals minor damage to target enemy. If the target has any active debuff, heals the nearest injured ally for a portion of the damage dealt. The Harmonist's offensive filler — builds Confluence during safe phases and trickle-heals as a bonus. Pairs with Frostbite or Burn for sustained damage and heal loops.

**Last Breath** — Enhancement. High cost. Instant. 60-second cooldown. Target ally cannot die for 4 seconds — lethal damage reduces them to 1 HP instead. When the effect expires, the channeler takes 50% of the damage that was prevented as self-damage. The emergency cooldown — save someone, pay the price yourself.

### Martial (Battlemage Only)

Martial abilities do not consume Flux. They use cooldowns instead — channeling vital energy rather than external Flux. Only accessible to the Battlemage spec.

**Adrenaline** — Enhancement. No Flux cost. 15s cooldown. Instant. +20% attack speed for 6 seconds. The Battlemage's burst window opener — faster strikes mean faster Weave stacking.

**Combat Roll** — Displacement. No Flux cost. 8s cooldown. Instant. Quick lateral roll with brief i-frames. Doesn't consume Flux and has a tight cooldown. The Battlemage's resource-free dodge.

**Precise Strike** — Enhancement. No Flux cost. 12s cooldown. Instant. Next melee hit within 4 seconds is guaranteed to critically hit. The Battlemage's burst punctuation — land this on the 3-hit combo finisher for massive damage.

### Illusion

**Mirage** — Affliction. Medium cost. Instant. Creates a false targeting signal on an ally's position for 4 seconds. The next enemy ability targeting that ally is redirected to the illusion (which takes the hit and vanishes). One-use bodyguard. Does not work on untargeted AoE.

**Data Theft** — Affliction. High cost. Channeled. Hacks a target enemy's perception for 3 seconds: the target's next ability is delayed by 1.5 seconds (cognitive lag). Against bosses, this shortens the reaction window but does not prevent the ability. Powerful when timed with party burst windows.

### Pure

**Flux Negation** — Protection. High cost. Instant. Dispels one active Flux-based effect on target (ally: removes debuff; enemy: removes buff). The counter-arcanotechnique tool. Essential for encounters with dispellable mechanics.

**Flux Burn** — Destruction. High cost. Instant. Deals damage proportional to the target's current Flux reserve (effective against Flux-using enemies). Against non-Flux targets, deals flat moderate damage. Niche but devastating in the right encounter.

**Arcane Silence** — Affliction. Extreme cost. Channeled. Silences target enemy for 3 seconds — prevents all Flux-based abilities. The ultimate shutdown for arcanotechnique-using bosses or PvP targets. Enormous Flux cost makes it a commitment.

---

## Shared Arcanotechnicien Identity

The Arcanotechnicien's universal tension is **preparation and vulnerability**. You walk into every fight with a plan — your loadout, your commitment split, your ability priority. When the plan works, you are the most powerful force on the battlefield: Destroyers erase entire rooms, Battlemages sustain relentless hybrid damage, Harmonists keep parties alive through otherwise lethal mechanics.

When the plan fails — wrong loadout, bad positioning, interrupted at the worst moment — you are helpless. No Flux means no abilities. No channel completion means no damage. No team protection means no safe windows to commit.

Where Gunner asks "can I hit the target?" and Vanguard asks "when do I commit?", the Arcanotechnicien asks "did I prepare correctly, and can I execute the plan?" The skill expression spans two timescales: the strategic (loadout, commitment, ability selection before the fight) and the tactical (channel timing, Overcharge windows, Weave rhythm, Harmony cycling during the fight).

The ability loadout is the class's soul. It's what makes two Arcanotechniciens feel completely different. It's what creates the theorycrafting community. It's the answer to "what should I bring for this boss?" that drives endless discussion. And it's the reason the Arcanotechnicien has the highest skill ceiling that isn't purely mechanical — because half the skill happens before you ever enter combat.

TTRPG source classes: Arcanotechnicien, Arcanotechnicien de Combat, Arcanotechnologue
