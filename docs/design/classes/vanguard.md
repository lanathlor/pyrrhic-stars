# Vanguard

**Gameplay: Souls-like Action Melee**

Camera: third person, close over-the-shoulder. Input: directional attacks, dodge rolls, parry timing. Core loop: read telegraphs, commit to swings, punish windows.

Flux usage: minimal. Used narratively to explain superhuman martial ability.

## Stamina and Commitment

Every Vanguard action costs **stamina** — attacks, dodges, blocks, parries. Stamina recovers passively when not acting. The Tenacity stat scales maximum pool, cost efficiency, and recovery rate. Running out of stamina mid-fight is a death sentence: no dodges, no blocks, no attacks.

**Commitment frames**: once an attack animation starts, you cannot dodge or block until it finishes. Heavier attacks lock you in longer. Every swing is a bet that nothing will hit you before you recover. This is the core Vanguard tension — not _which_ button to press, but _when_ to press it.

**Perfect Dodge**: dodge-rolling through an attack within a tight timing window (i-frames overlap the hit) grants a brief damage and attack speed buff on your next strike. Recovery stat scales i-frame duration and dodge recovery speed. The universal reward for reading telegraphs.

**Parry**: a timed input that, if landed in the parry window, stuns the attacker briefly and opens a riposte window. Tighter window than dodge, you stand still, but the reward is higher — stun + counter opportunity. Recovery stat scales parry window duration. Each spec flavors the parry differently.

| Spec   | Identity                 | Playstyle                                                                                       |
| ------ | ------------------------ | ----------------------------------------------------------------------------------------------- |
| Blade  | AoE burst melee          | Greatsword combos, building momentum through unbroken chains. Dynasty Warriors meets Dark Souls |
| Shield | Directional tank         | Greatshield blocking, physical interposition, stamina budgeting. Monster Hunter lance           |
| Shadow | Evasion counter-attacker | Twin daggers, counter-attacks fueled by dodged attacks. Sekiro meets Bloodborne                 |

## Blade — Momentum System

Two-handed greatsword. AoE burst. The skill is staying aggressive without getting punished.

### Core Mechanic: Onslaught

A hit counter that tracks consecutive landed attacks without taking damage. Each stack grants bonus damage (scaled by Mastery: Onslaught). At thresholds, abilities transform into empowered versions:

-   **0–2 stacks**: standard abilities
-   **3–5 stacks**: empowered — wider AoE, more damage, longer commitment
-   **6+ stacks**: maximum — massive AoE, best damage, longest commitment

Taking any hit resets stacks to zero. The class fantasy is a warrior who builds into an unstoppable storm of steel — but one mistake brings you back to nothing.

### Abilities

**Cleave** — fast horizontal sweep, 120° arc. Low stamina cost, short commitment. The safe Momentum builder.

-   _Empowered (3+)_: arc widens to 200°, increased damage
-   _Maximum (6+)_: full 360° sweep, highest AoE coverage

**Upheaval** — upward slash into overhead slam. Higher damage, 60° cone, moderate stamina, longer commitment.

-   _Empowered (3+)_: slam creates a shockwave extending the cone to 120°, knockback on smaller enemies
-   _Maximum (6+)_: shockwave leaves a brief ground DoT zone in the cone

**Vortex** — spinning advance that moves you forward through enemies. Hits twice along the path.

-   _Empowered (3+)_: extended travel, hits three times
-   _Maximum (6+)_: hits four times, pulls enemies toward the center of the path

**Execution** — wind up, then devastating overhead chop. Highest single-hit damage, narrow impact, longest commitment, highest stamina cost. The punish tool.

-   _Empowered (3+)_: impact creates a small forward shockwave behind the target
-   _Maximum (6+)_: shockwave becomes a ground-targeted cone (you aim it during the wind-up)

### Blade Parry

Blade's parry is a greatsword counter-swing. On success: the parried target staggers and the counter-swing itself deals damage, counting as a hit toward Onslaught. A successful parry mid-chain _builds_ Momentum instead of just preserving it.

### Skill Expression

-   **Beginner**: mashes Cleave, builds Momentum slowly, loses stacks to avoidable damage. Still functional — Cleave is fast and forgiving.
-   **Competent**: maintains 3+ stacks reliably, uses empowered abilities during openings. Chains Cleave → Upheaval for consistent AoE. Dodges most telegraphs.
-   **Expert**: maintains 6+ stacks through entire encounters by threading Perfect Dodges between swings. Uses Maximum Execution during stagger windows for devastating burst. Parries mid-chain to build faster. Never loses stacks to avoidable damage.

### Risk / Reward

Every empowered ability hits harder but commits you longer. At 6 stacks, Maximum Execution is the highest-damage ability in the Blade kit — but its wind-up and commitment are so long that one unexpected telegraph wipes your stacks AND eats a hit. The best Blade players know exactly how greedy they can afford to be.

---

## Shield — Bulwark System

Tower shield and one-handed weapon. Directional tank. The skill is positioning, stamina budgeting, and reading the fight for your whole team.

### Core Mechanic: Directional Block

Hold block to raise your shield. Absorbs all damage from attacks hitting the shield's facing cone (~120° forward). Each blocked hit drains stamina proportional to damage absorbed.

**Guard Break**: if stamina reaches zero while blocking, your guard shatters — brief stagger and increased damage taken. Stamina management is life. Knowing when to block, when to dodge, and when to drop guard for recovery is the entire skill floor.

**Bulwark Zone**: allies directly behind your shield cone take reduced damage from attacks that pass through your facing. This is not a taunt, not an aura, not magic — you are physically in the way. The boss still targets whoever it wants. Your job is to BE between the boss and whoever it's targeting.

### Abilities

**Shield Bash** — quick shield strike without dropping guard. Low damage, briefly staggers target, low stamina cost. Usable while blocking. The "contribute damage without compromising defense" tool.

**Bull Rush** — charge forward with shield raised. Pushes enemies in path back, interrupts abilities. Drops guard during the charge. Moderate stamina cost. The repositioning tool — close distance to interpose for an ally, push adds out of a zone, interrupt an enemy mid-ability.

**Brace** — plant feet and fortify stance. Massively reduced stamina drain on block for 3–4 seconds. Cannot move or attack during Brace. The "big hit incoming" button — survive a major telegraph without emptying your stamina bar.

**Retaliate** — drop guard and deliver a massive two-handed shield slam. Consumes all Devotion charges. Damage scales with charges consumed. Long commitment, leaves you fully exposed. The tank's DPS payoff — earned by absorbing damage for your team.

### Guard Parry

Shield's parry is a timed shield thrust at the moment of impact. On success: zero stamina drain, reflects a portion of blocked damage back to the attacker, and generates bonus Devotion charges. On mistimed attempt: guard drops briefly — worse than just blocking.

Expert Shield players Guard Parry as their primary blocking method — preserving stamina and building Devotion simultaneously. Normal blocking is the fallback when timing isn't safe.

### Devotion Economy

The Mastery stat (Devotion) governs how much absorbed damage converts to offensive charges. Every hit you block for yourself or shield for allies via Bulwark Zone adds to your Devotion pool. Retaliate dumps the pool for burst damage.

The tension: hold Devotion for a bigger Retaliate, or spend early for safety? A big Retaliate during a DPS window is optimal, but if the boss phases or repositions before you can dump, those charges are wasted. Reading the fight's rhythm matters as much as reading individual telegraphs.

### Skill Expression

-   **Beginner**: holds block toward boss, uses Shield Bash occasionally. Burns through stamina, gets Guard Broken on big hits.
-   **Competent**: angles shield toward incoming attacks, uses Brace for big telegraphs. Repositions with Bull Rush to protect targeted allies. Manages stamina to never get Guard Broken.
-   **Expert**: Guard Parries consistently, preserving stamina and building Devotion fast. Reads boss targeting instantly and interposes before the attack starts. Times Retaliate dumps during DPS windows. Fluidly alternates between blocking, dodging, and attacking to optimize stamina economy.

### Risk / Reward

Shield Vanguard is the only tank that doesn't use magic. Tutelaire Guardian projects auras and solid light barriers — protecting the team at range with Flux. Shield stands in melee range and physically takes the hits. Your survivability is your stamina bar and your timing, not a resource that regenerates from a stat. Mastery of Guard Parry is what separates a Shield that runs dry in 10 seconds from one that blocks an entire boss phase without breaking.

---

## Shadow — Counter Flow

Twin daggers. Evasion counter-attacker. The skill is converting enemy aggression into your own damage.

### Core Mechanic: Counter Charges

Every attack you dodge through (Perfect Quickstep) or parry generates a **Counter Charge**. Shadow's offensive abilities consume charges. You cannot generate burst from nothing — the boss's attacks are your fuel.

This inverts the normal melee dynamic. Other Vanguard specs dodge to survive. Shadow dodges to deal damage. Aggressive bosses with fast, frequent attacks are Shadow's best matchup. Passive bosses with long idle phases are the worst.

### Abilities

**Quickstep** — replaces the standard dodge roll. Shorter distance, faster recovery, lower stamina cost. Shadow trades dodge distance for dodge frequency — more dodges per stamina bar means more Counter Charges per fight. Each Perfect Quickstep (i-frames through an attack) generates 1 Counter Charge.

**Riposte** — costs 1 Counter Charge. Fast stab, high single-target damage. If used immediately after a Perfect Quickstep: bonus damage (the instant punish). Consecutive Ripostes within a window deal escalating damage — each one stronger than the last. The dump ability for spending banked charges.

**Shadow Strike** — costs 3 Counter Charges. Dash through the target, appearing behind them. Heavy damage plus a vulnerability debuff (target takes increased damage from all sources for a short duration). The group utility burst — set up a damage window for the whole party.

**Vanish** — no charge cost, moderate cooldown. Brief invisibility (2–3 seconds). Next attack from stealth is a guaranteed critical hit. The "boss isn't attacking me" button — when you can't generate charges naturally, Vanish lets you contribute. Also drops boss targeting to force it onto another player (strategic in group play).

### Shadow Parry

Shadow's parry is a dagger deflection — blade catches blade. On success: generates 2 Counter Charges instead of 1 (compared to Perfect Quickstep). Higher risk than dodging (tighter window, you stand still), but more efficient charge generation for players who can land the timing.

### Afterimage Chains

The Mastery stat (Afterimage) governs dodge→hit chain bonuses. Each consecutive dodge→attack cycle within a rhythm window amplifies the damage bonus. Breaking the rhythm (taking damage, pausing too long, missing a dodge timing) resets the chain.

At high Afterimage mastery, a sustained chain becomes devastating — each Riposte in the chain hits harder than the last, and the Afterimage bonus stacks on top of the Riposte escalation. An expert Shadow in full flow is the highest sustained single-target DPS in the Vanguard kit.

### Skill Expression

-   **Beginner**: quicksteps away from attacks, occasionally lands a Riposte. Uses Vanish defensively as an emergency escape.
-   **Competent**: generates Counter Charges consistently, chains 2–3 Ripostes during openings. Uses Shadow Strike for vulnerability windows during group burst.
-   **Expert**: maintains indefinite Quickstep→Riposte chains, dodging THROUGH every attack and immediately punishing. Banks charges during heavy telegraph phases, dumps entire pools in rapid Riposte chains during stagger windows. Uses Vanish offensively for guaranteed crits on Shadow Strike. Actively seeks boss attention — more attacks incoming means more fuel.

### Risk / Reward

Shadow's damage potential is the highest single-target in the Vanguard kit, but it's entirely dependent on the encounter. Against a boss with constant melee pressure, Shadow is a machine. Against a boss that idles or only uses ranged abilities, Shadow starves. This creates real spec-choice decisions per boss, not just a "best spec" default.

---

## Shared Vanguard Identity

The Vanguard's universal tension is **proximity and commitment**. You are always in melee range. Every swing locks you in. Every GCD is a bet that nothing will hit you before you recover.

The reward is simplicity of entry: no Flux commitment, no pre-fight setup, no resource management beyond stamina. You walk in and fight. The class ceiling is pure execution — damage limited only by how well you read the fight and how precisely you time your actions.

Where Blade Dancer's complexity is mental (tracking states, planning transitions), Vanguard's complexity is physical (reading animations, timing dodges, managing stamina).

TTRPG source classes: Maitre d'Armes, Tutelaire (partial), Sentinelle
