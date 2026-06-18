# Blade Dancer

**Gameplay: Positional Combo Fighter**

Camera: third person, pulled back to see blade positions. Input: target-lock, 4 ability buttons that change based on current blade configuration. Core loop: chain configuration transitions to set up optimal ability sequences.

Flux usage: moderate. Powers telekinetic blade control.

The blades exist in configurations (states). Each of the 4 abilities does something different depending on current configuration AND transitions the blades to a new configuration.

## Multi Blade — Configuration System

5 configurations, 4 abilities each. Every ability transitions to a different configuration. No cooldowns, small GCD.

### Core Rule: Configuration Power Distribution

Each configuration has a **core capability** (defense, AoE, single-target, etc.). When an ability transitions between configurations:

-   **~2/3 of the ability's power** comes from the configuration it's **going to** (the destination defines the ability's primary effect)
-   **~1/3 of the ability's power** comes from the configuration it's **leaving** (a residual echo of where you were)

This is a design guideline, not a strict formula. The intent is reactive gameplay: you move to the configuration you need right now, and you carry a trace of where you just were.

**Example:** You're under pressure → go to Orbit (defensive). You defended successfully → leave Orbit toward Fan. That Fan ability is mostly AoE damage (destination) but carries a small defensive component (origin). The system rewards reading the fight and flowing between states naturally.

### Configurations

Each configuration has a core capability and 4 abilities that each transition to one of the other 4 configurations.

| Configuration | Core Capability      | Fantasy                                                                |
| ------------- | -------------------- | ---------------------------------------------------------------------- |
| **Orbit**     | Defense              | Blades spinning around the player. Shields, damage reduction, reflects |
| **Fan**       | AoE damage           | Blades spread in an arc ahead. Sweeps, slashes, wave attacks           |
| **Lance**     | Single-target damage | Blades stacked in a line at target. Piercing strikes, focused hits     |
| **Scatter**   | Multi-target DoT     | Blades flying to multiple enemies. Sustained ticking damage, pressure  |
| **Crown**     | Utility / Control    | Blades hovering above in a halo. Crowd control, buffs, debuffs         |

### Transition Table

Each cell is an ability. Row = current configuration (leaving), Column = destination configuration (going to).

| From \ To   | → Orbit          | → Fan          | → Lance           | → Scatter         | → Crown           |
| ----------- | ---------------- | -------------- | ----------------- | ----------------- | ----------------- |
| **Orbit**   | —                | Shielded Sweep | Guarded Thrust    | Protected Scatter | Fortified Command |
| **Fan**     | Reaping Guard    | —              | Cleaving Pierce   | Slashing Spread   | Sweeping Hex      |
| **Lance**   | Piercing Barrier | Focused Slash  | —                 | Targeted Spread   | Pinning Strike    |
| **Scatter** | Dispersed Shield | Rain of Blades | Converging Strike | —                 | Chaos Bind        |
| **Crown**   | Commanding Ward  | Royal Cleave   | Decree Strike     | Sovereign Scatter | —                 |

### Ability Definitions

#### From Orbit (leaving Defense)

**Shielded Sweep** (Orbit → Fan)
AoE damage sweep with defensive residue. Blades unfurl from orbit into a wide arc, slashing all enemies in a cone. Grants a brief damage reduction buff (fading shield from the spin momentum).

**Guarded Thrust** (Orbit → Lance)
Focused single-target strike with defensive residue. Blades collapse from orbit into a piercing line at the target. On hit, grants a small personal shield (carried from defensive stance).

**Protected Scatter** (Orbit → Scatter)
Multi-target DoT application with defensive residue. Blades scatter outward from orbit, latching onto nearby enemies and ticking damage. Committer gains a small damage reduction per target hit (defensive echo scaling with exposure).

**Fortified Command** (Orbit → Crown)
Utility/CC with defensive residue. Blades rise from orbit into a halo, releasing a pulse that slows nearby enemies. Committer gains brief CC immunity (fortified transition from defensive state).

---

#### From Fan (leaving AoE)

**Reaping Guard** (Fan → Orbit)
Defensive barrier with AoE residue. Blades sweep inward into a tight orbit, forming a shield. Enemies the blades pass through on the way in take minor damage (reaping arc).

**Cleaving Pierce** (Fan → Lance)
Single-target hit with AoE residue. Blades converge from spread arc onto one target for a heavy strike. Enemies adjacent to the target take minor splash damage (lingering arc momentum).

**Slashing Spread** (Fan → Scatter)
Multi-target DoTs with AoE residue. Blades scatter from their arc formation, each seeking a different enemy. On application, each target takes a small initial AoE burst around itself before the DoT begins (carried slash energy).

**Sweeping Hex** (Fan → Crown)
CC/debuff with AoE residue. Blades rise from arc into crown formation, cursing all enemies in the arc's path. Applies a damage vulnerability debuff to all targets hit (wide-angle hex).

---

#### From Lance (leaving Single-target)

**Piercing Barrier** (Lance → Orbit)
Defensive shield with single-target residue. Blades withdraw from the target into orbit, dealing a parting strike on the way out. Forms a shield whose strength scales with the parting hit (momentum converted to defense).

**Focused Slash** (Lance → Fan)
AoE damage with single-target residue. Blades explode from their stacked line into a wide fan. The primary target takes a focused hit, then the fan sweeps for AoE behind it (piercing energy dispersing outward).

**Targeted Spread** (Lance → Scatter)
Multi-target DoTs with single-target residue. Blades fragment from lance formation, each seeking a different enemy. The original target receives a stronger DoT than secondary targets (focused residue on primary).

**Pinning Strike** (Lance → Crown)
CC/debuff with single-target residue. Blades pull back into crown position. The target takes a final hit and is briefly rooted in place (pinned by the departing lance).

---

#### From Scatter (leaving Multi-target DoT)

**Dispersed Shield** (Scatter → Orbit)
Defensive barrier with DoT residue. Blades recall from all targets into orbit. The shield pulses with residual energy — enemies that strike the committer take minor ticking damage (thorns effect from carried DoT energy).

**Rain of Blades** (Scatter → Fan)
AoE damage with DoT residue. Blades converge from scattered positions through a sweeping arc. Large AoE hit that leaves a brief ground effect dealing damage over time (scattered energy pooling into one zone).

**Converging Strike** (Scatter → Lance)
Single-target burst with DoT residue. All scattered blades converge simultaneously onto one target. Heavy hit that also applies a strong short-duration bleed (all that distributed pressure focused into one wound).

**Chaos Bind** (Scatter → Crown)
CC/debuff with DoT residue. Blades rise from scattered targets into crown. Each target they leave gets a brief snare, and all affected targets receive a debuff that ticks for minor damage (lingering chaos).

---

#### From Crown (leaving Utility/Control)

**Commanding Ward** (Crown → Orbit)
Defensive shield with utility residue. Blades descend from halo into orbit. Strong shield that also cleanses one debuff from the committer (purifying command carried from crown authority).

**Royal Cleave** (Crown → Fan)
AoE damage with utility residue. Blades descend from crown into a sweeping arc. AoE hit that also applies a brief slow to all targets struck (authoritative strike that commands enemies to halt).

**Decree Strike** (Crown → Lance)
Single-target hit with utility residue. Blades descend from crown into a focused lance. Heavy hit that marks the target with a vulnerability debuff — the target takes increased damage for a short duration (royal decree of weakness).

**Sovereign Scatter** (Crown → Scatter)
Multi-target DoTs with utility residue. Blades scatter from crown to all nearby enemies. Applies DoTs that also reduce target movement speed for the duration (sovereign will binding them in place).

### Skill Expression

No cooldowns, small GCD. Beginners mash and blades do stuff. Experts plan 2-3 transitions ahead like chess moves.

-   **Reactive play:** boss telegraphs a big hit → go to Orbit from wherever you are. All 4 paths into Orbit provide defense, just flavored differently.
-   **Offensive windows:** boss is stunned → chain Fan and Lance transitions to maximize damage during the window.
-   **Flow state:** the best players never stop transitioning. Every GCD is a decision about what you need NOW and what you'll need NEXT.

| Spec        | Identity                 | Playstyle                                                                            |
| ----------- | ------------------------ | ------------------------------------------------------------------------------------ |
| Multi Blade | 4-6 blades, AoE constant | Full combo system, complex, highest skill ceiling in the game                       |
| Dual Blade  | 2 blades, mono burst     | Each blade on its own GCD with 3 positions and 2 abilities per position. Piano gameplay |

## Dual Blade

Two telekinetic blades, each controlled independently on separate GCDs. The player manages two parallel combo flows, like playing piano with two hands.

Each blade has 3 positions:

-   **Close** — blade orbits near the player
-   **Mid** — blade hovers at mid range
-   **Far** — blade stationed at distance on target

Each position has 2 abilities per blade (12 abilities total across both blades). Every ability is bound to a position — using an ability repositions that blade and triggers its GCD.

### Resonance — The Burst Mechanic

When both blades occupy the **same position**, a **Resonance** triggers automatically. Resonance is a powerful burst ability whose strength scales with the **number of actions (GCDs) spent across both blades since the last Resonance**.

After Resonance, the blades **split** to the two remaining positions. The player chooses which blade goes where.

This creates the core tension: **keep the blades apart** to build charge, then **deliberately converge** at the right moment for maximum burst. Every GCD is a micro-decision — "if I use this Mid ability on Blade A, Blade B is already Mid, that'll trigger early."

#### The 3 Resonances

Each position produces a different Resonance ability:

| Position | Resonance | Effect | Splits To |
|---|---|---|---|
| **Close** | **Clash** | Defensive burst — high damage + brief damage reduction | Mid + Far |
| **Mid** | **Cross** | Balanced burst — high damage, no secondary effect | Close + Far |
| **Far** | **Impale** | Offensive burst — highest damage, no defensive upside | Close + Mid |

#### Post-Split Dynamics

Where the blades land after Resonance shapes the next cycle:

-   **After Clash (Close)** — blades split to Mid + Far. No blade nearby for defense. You traded safety for damage, now you're exposed until you bring a blade back Close.
-   **After Cross (Mid)** — blades split to Close + Far. Maximum separation. Hardest to accidentally re-converge, giving the longest buildup runway for the next Resonance.
-   **After Impale (Far)** — blades split to Close + Mid. Both blades relatively close together. Shorter runway before accidental convergence. High risk — you can build charge faster but mistakes come faster too.

Expert players plan not just this Resonance, but the setup for the next one.

### Skill Expression

**The skill is avoidance, not memorization.** You're tracking two independent blade paths and ensuring they never accidentally cross — until you want them to.

-   **Beginner:** converges every 3-4 actions. Small Resonances, fired at random positions. Still functional — the spec has a low floor.
-   **Competent:** holds 6-8 actions, converges deliberately at the position they want. Solid burst on a reasonable cycle.
-   **Expert:** holds 12+ actions, threading two blade paths that never overlap, dodging boss mechanics without accidentally converging, then choosing the perfect position and moment to slam them together. Plans 2 Resonance cycles ahead based on post-split positioning.

### Risk / Reward

-   **Boss telegraphs are the enemy of your burst.** A telegraph forces you to pull a blade Close for a defensive ability — but your other blade is already Close. You either eat the hit to protect your charge, or save yourself and pop a weak Resonance. Real decisions.
-   **Longer buildup = bigger payoff but more exposure.** Every extra GCD is another chance for boss mechanics or positioning pressure to accidentally converge your blades.
-   **Accidental convergence is the punishment for sloppy tracking.** Not a failed combo — a wasted burst fired at the wrong time with weak charge. You don't lose a resource, you lose an opportunity.

TTRPG source: new class, no direct TTRPG equivalent
