# Arcanotechnicien — Spell Codex

Complete spell list. Each spell has a School, Type, Flux cost, cast behavior, and effect.

Delivery column (for healing/support spells): **Z**one, **B**eam, **D**irect, or **—** for non-healing.

Spells marked **NEW** are additions addressing identified design gaps.

---

## Fire

| Spell      | Type        | Cost   | Cast                    | Delivery | Effect                                                                                                                     |
| ---------- | ----------- | ------ | ----------------------- | -------- | -------------------------------------------------------------------------------------------------------------------------- |
| Fireball   | Destruction | High   | Channel (early release) | —        | Massive AoE explosion at target location. Radius/damage scale with channel completion. Overcharged: leaves burning ground. |
| Ignition   | Destruction | Medium | Instant                 | —        | Single-target burst + strong burn DoT. Weave-compatible.                                                                   |
| Burn       | Affliction  | Low    | Instant                 | —        | Burn DoT on target. Low damage, high Flux efficiency. Filler.                                                              |
| Flame Wall | Protection  | Medium | Channel                 | —        | Wall of fire at target location (6s). Enemies passing through take heavy damage + slow. Zone denial.                       |

## Frost

| Spell       | Type        | Cost   | Cast                    | Delivery | Effect                                                                                                        |
| ----------- | ----------- | ------ | ----------------------- | -------- | ------------------------------------------------------------------------------------------------------------- |
| Frost Nova  | Destruction | Medium | Channel (early release) | —        | AoE burst at target location + slow field (4s). Control-damage hybrid.                                        |
| Ice Javelin | Destruction | Medium | Instant                 | —        | Single-target projectile. High damage + brief root.                                                           |
| Frostbite   | Affliction  | Low    | Instant                 | —        | Stacking slow + DoT. At 3 stacks: 2s immobilize.                                                              |
| Frost Ward  | Protection  | Medium | Instant                 | D        | Frost barrier on target ally. Absorbs damage. Explodes on break/expiry, applying Frostbite to nearby enemies. |

## Electricity

| Spell           | Type        | Cost   | Cast    | Delivery | Effect                                                                                                       |
| --------------- | ----------- | ------ | ------- | -------- | ------------------------------------------------------------------------------------------------------------ |
| Chain Lightning | Destruction | Medium | Instant | —        | Hits primary target, chains to up to 3 nearby enemies (reduced damage per chain).                            |
| Lightning       | Destruction | High   | Channel | —        | Single-target devastating damage. Near-instant travel.                                                       |
| Lightning Blade | Enhancement | Medium | Instant | —        | Infuses weapon with electricity (10s). Strikes chain a lightning arc to one nearby enemy. Battlemage staple. |
| Static Field    | Affliction  | Medium | Instant | —        | Persistent field at target (8s). Enemies inside: ticking lightning damage + reduced attack speed.            |
| Surge           | Enhancement | Medium | Instant | —        | +20% attack speed (6s). Flux arcs along your limbs. Battlemage burst opener.                                 |
| Flux Armor      | Protection  | Medium | Instant | —        | Electromagnetic shield (4s). Absorbs damage. Melee attackers take shock damage. Battlemage defensive.        |

## Shadow

| Spell       | Type         | Cost   | Cast    | Delivery | Effect                                                                        |
| ----------- | ------------ | ------ | ------- | -------- | ----------------------------------------------------------------------------- |
| Shadow Step | Displacement | Low    | Instant | —        | 8m blink in facing direction. Leaves shadow decoy (2s) that draws one attack. |
| Veil        | Enhancement  | Medium | Instant | —        | Optical camouflage (3s). Reduces enemy targeting priority. AoE still hits.    |
| Dread       | Affliction   | Medium | Instant | —        | Debuff target: reduced damage output (6s).                                    |

## Aerokinetic

| Spell         | Type         | Cost   | Cast                    | Delivery | Effect                                                                                            |
| ------------- | ------------ | ------ | ----------------------- | -------- | ------------------------------------------------------------------------------------------------- |
| Gale Force    | Destruction  | Medium | Channel (early release) | —        | Cone of wind: pushback + damage over duration. Area denial.                                       |
| Gale Shield   | Protection   | Medium | Instant                 | —        | Wind barrier around caster (4s). Deflects projectiles, reduces ranged damage. Melee unaffected.   |
| Gust Step     | Displacement | Low    | Instant                 | —        | 10m wind dash. Leaves tailwind: allies passing through get +10% move speed (2s).                  |
| Soothing Wind | Enhancement  | Medium | Instant                 | Z        | Wind zone at target (8s). Allies inside: minor heal/tick + move speed. Harmonist off-school zone. |

## Gravitonic

| Spell               | Type        | Cost    | Cast    | Delivery | Effect                                                                                                 |
| ------------------- | ----------- | ------- | ------- | -------- | ------------------------------------------------------------------------------------------------------ |
| Gravitonic Collapse | Destruction | Extreme | Channel | —        | Gravity well at target. Pulls enemies within 10m to center (2s), then detonates for massive AoE.       |
| Force Shell         | Protection  | Medium  | Instant | —        | Gravity field around caster (3s). Deflects projectiles, reduces melee damage taken. No AoE protection. |
| Gravity Crush       | Destruction | High    | Instant | —        | Single target: heavy damage + 2s root (pinned by gravity).                                             |

## Hydrodynamic

| Spell          | Type        | Cost   | Cast    | Delivery | Effect                                                                              |
| -------------- | ----------- | ------ | ------- | -------- | ----------------------------------------------------------------------------------- |
| Torrent        | Destruction | Medium | Channel | —        | Pressurized water stream at target. Sustained damage + slow pushback.               |
| Purifying Mist | Enhancement | Medium | Instant | Z        | Water mist zone at target (6s). Allies inside: cleanse one DoT on entry + minor DR. |

## Bioarcanotechnic

Expensive, powerful, monotarget. The "emergency room" school.

| Spell                         | Type        | Cost   | Cast    | Delivery | Effect                                                                                                                                                                                       |
| ----------------------------- | ----------- | ------ | ------- | -------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Mending Surge                 | Enhancement | High   | Instant | D        | Massive single-target heal. Biggest heal per cast in the game. Burns Flux fast.                                                                                                              |
| Mending Beam                  | Enhancement | High   | Channel | B        | Tether to ally. Large heal/tick while channeled. Highest sustained single-target throughput. LoS required.                                                                                   |
| Overclock                     | Enhancement | Medium | Instant | —        | Buff target (6s): +15% attack speed, +10% move speed. Self or ally.                                                                                                                          |
| Neural Fortification          | Protection  | High   | Instant | —        | Buff target ally (6s): +20% DR, immune to one interrupt. "Protect the caster" spell.                                                                                                         |
| **NEW** Restoration Matrix    | Enhancement | High   | Instant | Z        | Bioarcanotechnic healing zone at target (10s). Allies inside: strong heal/tick. Expensive to place but high throughput. Primary-school Zone heal.                                            |
| **NEW** Neural Purge          | Enhancement | Medium | Instant | D        | Cleanses all Flux-based debuffs + one non-Flux debuff from target ally. Grants 2s debuff immunity after cleanse. Primary-school cleanse.                                                     |
| **NEW** Regeneration Protocol | Enhancement | Medium | Instant | D        | Applies strong HoT on target ally (12s). If ally drops below 30% HP while active, remaining ticks consumed instantly as burst heal. The "insurance policy" — place before a dangerous phase. |

## Biometabolic

Life force redistribution. Drain ally HP as fuel (low Flux), or spend Flux to damage+heal (high Flux).

| Spell                | Type        | Cost   | Cast    | Delivery | Effect                                                                                                                                                                                                                    |
| -------------------- | ----------- | ------ | ------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Life Swap            | Enhancement | Low    | Instant | D        | Drains portion of ally HP, stores as vital charge. Next heal within 4s is empowered (bonus heal = HP drained). Core drain mechanic.                                                                                       |
| Transfusion          | Enhancement | Low    | Channel | B        | Tether to ally, drains their HP/tick. All other allies within 10m healed for the same amount. AoE heal powered by one ally's sacrifice.                                                                                   |
| Vital Circuit        | Enhancement | Low    | Instant | D        | Links two allies (8s). Damage split evenly. On expiry, lower-HP ally healed for portion of HP difference.                                                                                                                 |
| Metabolic Burst      | Enhancement | High   | Instant | D        | Deals moderate damage to target enemy. Heals all allies within 8m of target for portion of damage dealt. No ally HP cost — pure Flux. Emergency AoE.                                                                      |
| Vital Drain          | Destruction | Medium | Channel | —        | Tether to enemy. Drains HP/tick, heals caster for portion dealt. Self-sustain for DPS specs.                                                                                                                              |
| Metabolic Disruption | Affliction  | Medium | Instant | —        | Debuff target: -15% healing received, -10% move speed (8s). PvP tool / niche PvE.                                                                                                                                         |
| **NEW** Vital Bloom  | Enhancement | Low    | Instant | Z        | Sacrifices a portion of caster's HP to create healing zone at target (8s). Heal/tick proportional to HP sacrificed. Self-sacrifice Zone — more HP given, stronger zone. The Biometabolic primary-school Zone heal.        |
| **NEW** Siphon Pulse | Destruction | Low    | Instant | —        | Minor damage to target enemy. If target has any debuff, heals nearest injured ally for portion of damage. Offensive filler that feeds Confluence and trickle-heals. The "safe phase" spell.                               |
| **NEW** Last Breath  | Enhancement | High   | Instant | D        | 60s cooldown. Target ally cannot die for 4 seconds — lethal damage reduces to 1 HP. When effect expires, caster takes 50% of prevented damage as self-damage. The emergency cooldown — save someone, pay for it yourself. |

## Martial (Battlemage Only)

No Flux cost. Cooldown-based. Channeling vital energy, not Flux.

| Spell          | Type         | Cost    | Cast    | Delivery | Effect                                                    |
| -------------- | ------------ | ------- | ------- | -------- | --------------------------------------------------------- |
| Adrenaline     | Enhancement  | No Flux | Instant | —        | 15s CD. +20% attack speed (6s). Battlemage burst window.  |
| Combat Roll    | Displacement | No Flux | Instant | —        | 8s CD. Lateral roll with brief i-frames. Flux-free dodge. |
| Precise Strike | Enhancement  | No Flux | Instant | —        | 12s CD. Next melee hit within 4s is guaranteed crit.      |

## Illusion

| Spell      | Type       | Cost   | Cast    | Delivery | Effect                                                                                                                                            |
| ---------- | ---------- | ------ | ------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| Mirage     | Affliction | Medium | Instant | —        | False targeting signal on ally position (4s). Next targeted enemy ability redirected to illusion. One-use bodyguard. No effect on untargeted AoE. |
| Data Theft | Affliction | High   | Channel | —        | Hacks target perception (3s). Next ability delayed 1.5s. Against bosses: shortens reaction window, doesn't prevent ability.                       |

## Pure

| Spell          | Type        | Cost    | Cast    | Delivery | Effect                                                                                          |
| -------------- | ----------- | ------- | ------- | -------- | ----------------------------------------------------------------------------------------------- |
| Flux Negation  | Protection  | High    | Instant | —        | Dispels one Flux-based effect. Ally: removes debuff. Enemy: removes buff. Counter-magic staple. |
| Flux Burn      | Destruction | High    | Instant | —        | Damage proportional to target's Flux reserve. Flat moderate damage vs non-Flux targets.         |
| Arcane Silence | Affliction  | Extreme | Channel | —        | Silences target (3s) — prevents all Flux-based abilities. Ultimate shutdown. Enormous cost.     |

---

## Harmonist Spell Pool Summary

Spells a Harmonist would realistically consider, organized by role:

### Core Healing

| Spell                 | School           | Delivery | Cost | Why bring it                           |
| --------------------- | ---------------- | -------- | ---- | -------------------------------------- |
| Mending Surge         | Bioarcanotechnic | D        | High | Emergency single-target save           |
| Mending Beam          | Bioarcanotechnic | B        | High | Sustained single-target throughput     |
| Restoration Matrix    | Bioarcanotechnic | Z        | High | Primary-school Zone, strong throughput |
| Life Swap             | Biometabolic     | D        | Low  | Empower next heal, Flux-efficient      |
| Transfusion           | Biometabolic     | B        | Low  | AoE heal via ally drain                |
| Vital Bloom           | Biometabolic     | Z        | Low  | Self-sacrifice Zone, Flux-efficient    |
| Regeneration Protocol | Bioarcanotechnic | D        | Med  | HoT + sub-30% emergency burst          |
| Metabolic Burst       | Biometabolic     | D        | High | AoE emergency when nobody to drain     |

### Protection & Utility

| Spell                | School           | Cost | Why bring it                             |
| -------------------- | ---------------- | ---- | ---------------------------------------- |
| Frost Ward           | Frost            | Med  | Pre-emptive shield on ally               |
| Neural Fortification | Bioarcanotechnic | High | DR + interrupt immunity on ally          |
| Vital Circuit        | Biometabolic     | Low  | Damage split link, equalizes HP          |
| Neural Purge         | Bioarcanotechnic | Med  | Primary-school cleanse + debuff immunity |
| Last Breath          | Biometabolic     | High | Death prevention emergency (60s CD)      |

### Mobility & Offensive

| Spell        | School       | Cost | Why bring it                                      |
| ------------ | ------------ | ---- | ------------------------------------------------- |
| Gust Step    | Aerokinetic  | Low  | Repositioning for Sympathetic Field               |
| Siphon Pulse | Biometabolic | Low  | Offensive filler, builds Confluence, trickle heal |
| Frostbite    | Frost        | Low  | Cheap CC, stacking slow to immobilize             |

### Situational

| Spell          | School           | Cost | Why bring it                                |
| -------------- | ---------------- | ---- | ------------------------------------------- |
| Overclock      | Bioarcanotechnic | Med  | Offensive buff on ally (coordinated groups) |
| Soothing Wind  | Aerokinetic      | Med  | Off-school Zone + move speed                |
| Purifying Mist | Hydrodynamic     | Med  | DoT cleanse zone (DoT-heavy encounters)     |
| Flux Negation  | Pure             | High | Dispel (encounters with Flux debuffs)       |
| Mirage         | Illusion         | Med  | One-shot bodyguard (at +50% cost)           |

### Delivery Method Coverage

| Method     | Bioarcanotechnic                                                      | Biometabolic                                                       | Off-School                 |
| ---------- | --------------------------------------------------------------------- | ------------------------------------------------------------------ | -------------------------- |
| **Zone**   | Restoration Matrix (High)                                             | Vital Bloom (Low, self-sacrifice)                                  | Soothing Wind (Aero, +25%) |
| **Beam**   | Mending Beam (High)                                                   | Transfusion (Low, ally drain)                                      | —                          |
| **Direct** | Mending Surge (High), Regeneration Protocol (Med), Neural Purge (Med) | Life Swap (Low, drain), Metabolic Burst (High), Last Breath (High) | Frost Ward (Frost, absorb) |

Every delivery method now has at least one option in each primary healing school. Harmony cycling works within primary schools alone.

### Example Loadouts

**Standard Dungeon (balanced)**

1. Restoration Matrix (Bioarcanotechnic, Z) — primary Zone
2. Mending Beam (Bioarcanotechnic, B) — primary Beam
3. Life Swap (Biometabolic, D) — empowered heals, Flux-efficient
4. Vital Bloom (Biometabolic, Z) — cheap backup Zone, self-sacrifice
5. Frost Ward (Frost) — pre-emptive shield
6. Gust Step (Aerokinetic) — mobility

Commitment: 50% Bioarcanotechnic / 25% Biometabolic / 15% Frost / 10% Aerokinetic

**Progression (safety-first)**

1. Mending Surge (Bioarcanotechnic, D) — emergency save
2. Mending Beam (Bioarcanotechnic, B) — sustained tank healing
3. Regeneration Protocol (Bioarcanotechnic, D) — HoT insurance
4. Transfusion (Biometabolic, B) — AoE sustain
5. Last Breath (Biometabolic, D) — death prevention
6. Neural Purge (Bioarcanotechnic, D) — cleanse

Commitment: 70% Bioarcanotechnic / 20% Biometabolic / 10% flex
Trade: no Zone heal (skip Harmony cycling), max raw throughput and safety.

**Biometabolic Triage (sustained, Flux-efficient)**

1. Vital Bloom (Biometabolic, Z) — self-sacrifice Zone
2. Transfusion (Biometabolic, B) — ally-drain AoE
3. Life Swap (Biometabolic, D) — empowered next heal
4. Metabolic Burst (Biometabolic, D) — emergency AoE
5. Mending Surge (Bioarcanotechnic, D) — emergency single-target
6. Siphon Pulse (Biometabolic) — offensive filler + trickle heal

Commitment: 25% Bioarcanotechnic / 60% Biometabolic / 15% flex
Trade: low single-target throughput, but nearly infinite sustain. Group never at full HP, but never dies. The "trust your Harmonist" build.
