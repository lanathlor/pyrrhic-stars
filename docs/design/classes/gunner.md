# Gunner

**Gameplay: First-Person Shooter**

Camera: first person. Input: crosshair aiming, real projectile/hitscan. Core loop: aim, shoot, reposition, use cover.

Flux usage: minimal. Primarily martial with tech gadgets.

## Aim and Stability

The Gunner is a literal FPS character inside an MMO. Crosshair aiming, real ballistics (hitscan or projectile depending on weapon), and server-authoritative accuracy. Damage comes from landing shots — not from pressing abilities on cooldown. A Gunner who can't aim deals no damage regardless of gear.

### Server-Authoritative Accuracy Model

The client is open source. Any client-side accuracy mechanic (recoil patterns, scope sway) can be trivially removed by forking the client. The server cannot distinguish "perfect recoil control" from "recoil deleted." Therefore, all accuracy mechanics are server-authoritative.

The server tracks two values per Gunner:

-   **Stability** — decreases with sustained fire, recovers when not firing. Low Stability widens the server-side accuracy cone: shots scatter regardless of where the client claims the crosshair is pointing. Each weapon has a different Stability decay curve and recovery rate.
-   **Steadiness** — decreases with movement, recovers when stationary. Low Steadiness reduces precision hit registration: the server rejects headshot/weak-point claims from moving players regardless of client aim.

The client displays visual feedback (crosshair bloom, camera shake, scope drift) as cosmetic indicators of the server state, but these are informational — the server enforces accuracy independent of client rendering. A modded client that removes visual feedback gains zero advantage because the accuracy penalty is applied server-side to hit detection.

**Micro-bursting** (releasing fire briefly to let Stability recover) is the universal Gunner skill. The client shows the crosshair tightening as feedback, but the reason it works is that the server's Stability value recovers during the pause. The skill expression is firing rhythm — knowing how long to sustain fire before the accuracy cone makes continued shooting wasteful, and how long to pause for recovery.

## Magazine and Reload

Standard ammunition is unlimited but fed from a magazine. When the magazine empties, you must reload — a real animation that locks you out of shooting for its duration.

-   **Tactical reload**: reload with rounds remaining. Faster animation (you skip the charging handle).
-   **Empty reload**: magazine is dry. Slower animation. Getting caught dry during a dodge phase is a DPS loss at best, a death at worst.

The Action stat scales reload speed, ADS speed, and weapon swap speed. Managing your magazine — reloading during safe windows, never getting caught dry during burst phases — is the universal Gunner skill floor.

## Enhanced Rounds

The Gunner's Identity resource (Munitions stat). A secondary ammunition type stored in a separate reserve. Activating Enhanced Rounds loads them into your current magazine — subsequent shots consume Enhanced Rounds instead of standard ammo, dealing bonus damage with spec-specific effects.

Enhanced Rounds are not unlimited. Each spec generates them through different triggers. The Munitions stat scales reserve capacity, passive regen rate, and round potency.

The tension: Enhanced Rounds are your strongest damage. Loading them during a burst window maximizes output. But generating them requires spec-specific play — Assault needs sustained tracking, Marksman needs precision hits, Chasseur needs disruption. The best Gunners weave generation and spending into a natural rhythm without ever losing DPS uptime.

## Combat Roll

Universal Gunner mobility tool. Quick lateral roll with brief i-frames. Short cooldown. Covers less distance than a Vanguard dodge but recovers faster — designed for repositioning within an FPS firefight, not for crossing distance. You can shoot immediately after the roll ends.

The Gunner doesn't have stamina. Your constraint is DPS uptime: every roll is time not shooting. Roll spam keeps you alive but craters your damage. Expert Gunners minimize rolls by reading telegraphs and pre-positioning, rolling only when necessary.

| Spec     | Identity                   | Playstyle                                                     |
| -------- | -------------------------- | ------------------------------------------------------------- |
| Assault  | Aggressive close-mid range | Fast movement, high fire rate. Titanfall energy               |
| Marksman | Long range precision       | Slow, deliberate, hold breath for perfect shots. Sniper Elite |
| Chasseur | Anti-arcanotechnique       | Disruption grenades, EMP, tactical. Rainbow Six               |

## Assault — Pressure System

Automatic rifle, close-to-mid range. Monotarget, constant damage. The skill expression is maintaining maximum uptime while dodging everything.

### Core Mechanic: Pressure

Consecutive hits on the same target stack a damage bonus (max 10 stacks, scaled by Mastery). Stacks reset on miss, target swap, or ~2 second timeout. Pressure rewards pure tracking skill — landing every shot, even while dodging and repositioning, keeps your damage climbing.

At high stacks, Assault's DPS eclipses burst specs on sustained single-target fights. But one missed shot, one forced target swap, one panic roll that drops your aim — back to zero. The mastery ceiling is aim consistency under pressure, not burst optimization.

### Enhanced Round Trigger

Reaching max Pressure stacks (10) generates a batch of Enhanced Rounds. While loaded, Enhanced Rounds gain bonus damage per active Pressure stack — incentivizing you to load them at max stacks and maintain tracking to keep the multiplier.

### Weapon: Assault Rifle

Hitscan, full-auto. Stability decays quickly under sustained fire — the first 5–6 shots land in a tight cone, then the server-side spread blooms significantly. Expert Assault players micro-burst (release and re-press trigger) to recover Stability between bursts, sacrificing marginal fire rate for tighter accuracy. The crosshair bloom on screen reflects the server's Stability state.

### Abilities

**Overclock** — 6–8 second window of increased fire rate and move speed. Moderate cooldown. During Overclock, Stability recovers faster between micro-bursts and Pressure stacks build 50% faster. Not burst — it's a sustained DPS amplifier that rewards extended tracking under the buff. Best used when you already have Pressure stacks to accelerate to cap.

**Rechamber** — trigger a short animation (pulling the charging handle), then hit the button again within a timing window for a damage buff on the next few seconds of fire. Hit the sweet spot: bonus damage on all shots for ~3 seconds. Miss the window: brief lockout (can't fire for ~1 second). The risk/reward weave tool — slot it into safe moments between dodges. At expert play, Rechamber uptime approaches 80%, weaving it between every dodge and Overclock activation.

**Mag Dump** — empty your remaining magazine in a rapid-fire burst at increased fire rate. All remaining rounds (standard or Enhanced) fire in a tight cone over ~1 second. More rounds remaining = more damage. Triggers an empty reload afterward. The burst finisher — dump Enhanced Rounds at max Pressure for peak damage, then reload during the next safe window.

### Assault Loop

Shoot → build Pressure → roll through telegraph → maintain tracking → Rechamber during safe window → Overclock at mid-high stacks → Mag Dump with Enhanced Rounds at max Pressure → reload → repeat.

### Skill Expression

-   **Beginner**: holds trigger until the magazine is dry, Stability bottoms out and shots scatter everywhere. Loses Pressure constantly to missed shots. Rolls reactively and loses tracking. Treats Rechamber as a standalone ability instead of weaving it. Still deals baseline damage — the gun works even if you're sloppy.
-   **Competent**: maintains 5–7 Pressure stacks reliably through micro-bursting to manage Stability. Reloads during safe windows. Weaves Rechamber between dodge phases. Uses Overclock proactively when stacks are mid-range. Consistent DPS with few dead moments.
-   **Expert**: maintains max Pressure through entire fight phases by tracking through dodge rolls and micro-bursting on rhythm to keep Stability in the tight-cone range. Rechamber uptime is near-permanent. Times Overclock → max stacks → Enhanced Rounds → Mag Dump for devastating sustained sequences. Never gets caught dry. Movement is pre-positioned so rolls are rare. Damage output is relentless.

### Risk / Reward

Assault's damage scales with consistency, not cleverness. There's no burst window to time, no combo to memorize. The entire skill expression is "keep shooting, don't miss, don't get hit." This makes Assault the purest aim-skill spec in the game — a Gunner who can track perfectly under pressure will outdamage anyone on sustained single-target fights. But the floor for "perfect tracking under pressure" is extremely high when bosses are throwing telegraphed attacks that demand repositioning.

---

## Marksman — Patience System

Bolt-action rifle, long range. Monotarget, burst damage. The opposite of Assault — every shot is a decision, not a stream.

### Core Mechanic: Patience

The next shot deals bonus damage scaling with time since the last shot (caps at 5 seconds). Patience is passive — stop shooting, start building. The Mastery stat scales the damage bonus curve and reduces time to reach cap.

This creates the anti-Assault rhythm: shoot, wait, shoot. The skill is not just aim (you need to land every shot because each one matters enormously) but also fight reading — knowing when you can afford to hold for a full Patience charge vs. when you need to fire early for phase timing or safety.

### Enhanced Round Trigger

Precision hits on weak points (headshots) generate Enhanced Rounds. A body shot deals damage. A headshot deals damage AND generates fuel for later burst. Expert Marksmen land precision hits consistently enough that Enhanced Rounds are always available when needed.

### Weapon: Bolt-Action Rifle

Projectile (not hitscan — bullet travel time at extreme range). Bolt-action: one shot, then a cycling animation before the next round chambers. Cannot hold trigger. Each shot is deliberate. The cycle time between shots is the natural Patience builder — you physically can't spam.

ADS (aim down sights) narrows the crosshair significantly and adds a scope zoom. Steadiness is critical for Marksman — movement drops Steadiness, and the server rejects precision hits (headshots, weak-point hits) below a Steadiness threshold. Standing still recovers Steadiness. The Action stat reduces Steadiness recovery time and bolt cycling speed.

### Abilities

**Steady** — plant feet and enter a stabilized stance. Instantly maxes Steadiness and locks it at maximum for the duration, guaranteeing precision hit registration. Also reduces flinch from incoming damage. Cannot move during Steady. Moderate duration (~4 seconds). The "I have a window and I'm taking it" ability. Using Steady at the wrong moment roots you in place for a telegraph.

**Killshot** — consumes all available Enhanced Rounds in one devastating shot. Damage scales with rounds consumed (more banked = bigger hit). Long wind-up animation during which you must maintain ADS on target. The ultimate payoff — bank Enhanced Rounds through a fight phase, then unload everything in one perfect shot during a stagger window.

**Disengage** — quick backward jump that creates distance. Brief i-frames during the jump. Automatically exits ADS. The emergency repositioning tool — something got too close, or a telegraph is incoming while you're in Steady. Shorter cooldown than Combat Roll but always moves you backward from facing. Expert Marksmen use it proactively to maintain optimal range.

**Spotter Round** — fire a tracer that marks the target. Marked targets display a weak point indicator visible to you and party members. Marked target takes increased damage from your next shot. Short duration. Does not consume Patience charge (it doesn't count as a "shot" for Patience purposes). The setup tool — Spotter → hold Patience → Killshot is the maximum burst combo.

### Marksman Loop

Spotter Round → hold Patience → ADS + Steady during safe window → precision headshot → cycle bolt → hold Patience again → Killshot when Enhanced Rounds are banked and boss is vulnerable → Disengage when threatened → reposition → repeat.

### Skill Expression

-   **Beginner**: fires as soon as bolt cycles, never builds Patience. Misses headshots, barely generates Enhanced Rounds. Uses Steady in exposed positions and eats telegraphs. Still deals decent damage per shot — bolt-action hits hard at baseline.
-   **Competent**: holds Patience to 3–4 seconds regularly. Lands headshots on stationary or slow-moving targets. Banks 3–4 Enhanced Rounds for Killshot during obvious stagger windows. Uses Disengage reactively for survival. Consistent burst contributor.
-   **Expert**: holds full 5-second Patience charges between every shot. Lands headshots on moving targets mid-mechanic. Banks maximum Enhanced Rounds and times Killshot with party burst windows and boss vulnerability phases. Uses Steady only in confirmed safe windows. Pre-positions at optimal range so Disengage is rarely needed. Every single shot is a precision event that hits harder than most specs' entire rotations.

### Risk / Reward

Marksman is the highest single-hit damage in the game, but only if you can afford to wait. Fights with constant pressure and no safe windows starve Marksman of Patience charge time and Steady uptime. Fights with clear phase transitions and stagger windows are Marksman's paradise — bank charges through the action phase, then deliver devastating shots during the opening. This creates genuine spec-choice decisions per encounter, not just "Marksman is always best."

The headshot requirement for Enhanced Round generation adds an aim-skill gate that doesn't exist for other specs. A Marksman who can't hit weak points doesn't just deal less damage — they lose access to their burst resource entirely.

---

## Chasseur — Quarry System

Shotgun and tactical grenades. AoE burst. The anti-caster: disruption, area denial, and punishing enemies for using abilities.

### Core Mechanic: Quarry

Disrupting a target's ability (interrupting a channel, detonating a grenade during a wind-up, hitting a target during its commit window) marks that target as your Quarry. Quarry grants a damage bonus window against that target. Mastery scales the bonus magnitude and window duration.

This inverts the normal DPS mindset. Other specs deal damage during safe windows. Chasseur deals maximum damage during dangerous windows — when the boss is mid-ability and vulnerable to disruption. The best Chasseur players seek out interrupt opportunities that other classes dodge away from.

### Enhanced Round Trigger

Quarry procs generate Enhanced Rounds. Each successful disruption loads rounds. Chasseur's burst economy is directly tied to how aggressively they play into enemy abilities — passive play starves your resources.

### Weapon: Combat Shotgun

Hitscan, semi-auto, wide spread. High damage at close range, damage falls off sharply with distance. Magazine-fed, moderate reload. The shotgun rewards close-range aggression — exactly where the boss's abilities are most dangerous. Chasseur's weapon forces proximity that its kit then exploits.

### Abilities

**EMP Grenade** — thrown grenade that detonates after a short fuse (or on direct hit). Creates an EMP field for ~3 seconds. Enemies casting or channeling abilities within the field are disrupted (ability cancelled, brief stagger). Triggers Quarry on each disrupted target. The bread-and-butter interrupt tool. Moderate cooldown. Placement is everything — predicting where a boss will start its next ability and pre-placing the EMP is the core Chasseur skill.

**Concussion Charge** — thrown explosive that deals high AoE damage in a radius. No interrupt effect — pure damage. Short delay before detonation. The AoE damage tool. Enhanced Rounds loaded into the shotgun amplify Concussion Charge as well (the charge is loaded with enhanced shrapnel). Best used immediately after a Quarry proc for amplified AoE during the damage window.

**Flashbang** — thrown device that blinds enemies in radius, briefly reducing their accuracy and attack speed. Does not interrupt abilities already in progress but delays the next one. No Quarry proc (it's not a disruption). The defensive utility — buy time for a reload, create a safe window for the team, or delay a dangerous ability to set up EMP timing.

**Breach** — shoulder-charge forward through enemies. Brief i-frames during the charge. Enemies hit take moderate damage and are staggered. If Breach hits an enemy mid-ability, it counts as a disruption (triggers Quarry). The aggressive gap-closer — rush into a boss's face during its wind-up, disrupt the ability, trigger Quarry, and shotgun at point-blank range with the damage bonus. High risk: misread the timing and you charge into a hit.

### Chasseur Loop

Read boss ability telegraphs → EMP pre-placement → disruption triggers Quarry → Breach in → Enhanced shotgun blasts at point blank during Quarry window → Concussion Charge for AoE burst → Flashbang to buy reload time → reposition → read next ability → repeat.

### Skill Expression

-   **Beginner**: throws EMPs reactively after abilities start, misses disruption windows. Stays at mid-range and wastes shotgun damage to falloff. Uses Flashbang and Breach defensively. Still contributes AoE with Concussion Charge — the kit works at range, just worse.
-   **Competent**: pre-places EMPs reliably on boss ability patterns. Closes to shotgun range during Quarry windows. Chains Quarry → Enhanced Rounds → Concussion Charge for solid burst. Uses Breach for repositioning rather than disruption. Consistent interrupt contribution to the group.
-   **Expert**: reads boss ability queues 1–2 abilities ahead and pre-places EMPs with perfect timing. Uses Breach offensively through wind-ups for extra Quarry procs. Maintains near-permanent Quarry uptime on bosses with frequent abilities. Stays at shotgun range throughout fights by reading telegraphs well enough to never need distance. The team's interrupt engine — freeing other players from interrupt responsibility while dealing top-tier AoE burst.

### Risk / Reward

Chasseur's entire damage economy is tied to enemy ability frequency. Against a boss with constant ability usage, Chasseur is a machine — permanent Quarry uptime, Enhanced Rounds flowing, shotgun blasting at close range. Against a boss that primarily auto-attacks with rare ability usage, Chasseur starves — no Quarry procs, no Enhanced Rounds, reduced to a mid-range shotgun with no special ammo. Like Shadow Vanguard, this creates real encounter-dependent spec choices.

The close-range requirement compounds the risk. Chasseur wants to be in the boss's face during its most dangerous moments (ability wind-ups). A Chasseur who can read every telegraph and disrupt every ability is the group's most valuable player. A Chasseur who misreads a single telegraph eats the full hit at point-blank range.

---

## Shared Gunner Identity

The Gunner's universal tension is **uptime and precision**. You deal damage by landing shots — not by pressing abilities on cooldown. Every bullet is a skill check. Every moment not shooting is DPS lost.

Where Vanguard asks "when do I press the button?", Gunner asks "can I hit the target while staying alive?" The reward is immediate, visceral feedback: crosshair on target, bullets land, numbers go up. The punishment is equally immediate: miss, and nothing happens.

No Flux management, no configuration planning, no stamina budgeting. You point, you shoot, you move. The class ceiling is raw mechanical skill — aim, tracking, firing rhythm, and fight awareness. The three specs differentiate HOW you aim (spray tracking vs. single-shot precision vs. close-range disruption) but the fundamental question is always the same: can you put the crosshair where it needs to be?

TTRPG source classes: Horion, Spectre, Chasseur
