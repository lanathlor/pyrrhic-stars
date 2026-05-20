# Stat System

Six stats per character. Three universal (identical mechanic across all classes), three class/spec-specific (mechanic varies by class or spec).

## Universal Stats

Same name, same mechanic, every class.

-   **Hull** — hit points. More Hull = more HP. The survivability stat.
-   **Output** — damage scaling. More Output = harder hits. The throughput stat.
-   **Plating** — mitigation. More Plating = less damage taken per hit. Flat damage reduction.

## Class-Specific Stats

Named per class. The stat name changes to reflect the class fantasy, and the mechanic is tuned to that class's gameplay.

### Tempo

The rhythm stat. Controls how fast or fluidly the class operates.

| Class        | Name       | Effect                                                  |
| ------------ | ---------- | ------------------------------------------------------- |
| Gunner       | Action     | Fire rate, ADS speed, weapon swap speed                 |
| Vanguard     | Recovery   | Parry window duration, dodge i-frame duration, combo input window, recovery frame speed |
| Arcanotechnicien | Channel    | Cast speed, channel speed                               |
| Engineer     | Deploy     | Deployable cooldown reduction, drone reactivation speed |
| Blade Dancer | Transition | GCD on configuration/position changes                   |
| Tutelaire    | Pulse      | Aura tick rate, beacon recharge speed                   |

### Identity

The class signature stat. Defines the core resource or mechanical identity of the class.

| Class        | Name       | Effect                                                        |
| ------------ | ---------- | ------------------------------------------------------------- |
| Gunner       | Munitions  | Enhanced round reserve size, passive regen rate, enhanced round potency. Standard shots unlimited; enhanced rounds are a finite secondary resource that procs via spec-specific triggers. |
| Vanguard     | Tenacity   | Maximum stamina pool, stamina cost efficiency on all actions, stamina recovery rate. The core stamina resource stat. |
| Arcanotechnicien | Flux       | Maximum Flux pool + Flux regen rate                           |
| Engineer     | Grid       | Total grid capacity, grid recharge rate, power efficiency per deployable. Deployables draw from a shared power budget — run more at lower power or fewer at full power. |
| Blade Dancer | Resonance  | Resonance charge capacity, charge gain per action, charge retention. Dual Blade: scales convergence burst system. Multi Blade: transitions build Resonance, thresholds amplify next transition. |
| Tutelaire    | Presence   | Aura radius, simultaneous light projection count, aura linger duration (brief persistence in areas you've left). |

## Spec-Specific Stat

### Mastery

Named per spec. The build-defining stat — its effect changes based on the character's active specialization.

#### Gunner

| Spec     | Mastery Effect                                                                                                                     |
| -------- | ---------------------------------------------------------------------------------------------------------------------------------- |
| Assault  | Pressure — consecutive hits on same target stack damage (max 10). Resets on miss, target swap, or timeout.                         |
| Marksman | Patience — next shot deals bonus damage scaling with time since last shot (caps at 5s charge).                                     |
| Chasseur | Quarry — disrupting a target's ability grants a damage bonus window against that target. Mastery scales bonus and window duration. |

#### Vanguard

| Spec   | Mastery Effect                                                                                                                                                 |
| ------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Blade  | Onslaught — successive hits without taking damage stack bonus damage. Taking a hit drops stacks.                                                               |
| Shield | Devotion — absorb a percentage of damage taken by nearby allies. Each absorption charges your next ability's damage.                                           |
| Shadow | Afterimage — dodging an attack grants bonus damage on next hit. Consecutive dodge→hit chains amplify the bonus. Resets on taking damage or missing the window. |

#### Arcanotechnicien

| Spec       | Mastery Effect                                                                                                                                                                                  |
| ---------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Destroyer  | Overcharge — channeling past base duration increases damage but builds Instability. Release at the right moment for max damage; hold too long and it backfires. Mastery widens the safe window. |
| Battlemage | Weave — alternating melee and spell attacks grants stacking bonus damage. Consecutive same-type attacks break the chain.                                                                        |
| Harmonist  | Harmony — healing an ally with a different spell type than the last heal on that target (zone/beam/direct) triggers a bonus heal. Mastery scales the bonus.                                     |

#### Engineer

| Spec      | Mastery Effect                                                                                                                                            |
| --------- | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Architect | Synergy — deployables within range of each other gain stacking bonuses. Mastery scales the per-link bonus. Cluster for power or spread for coverage.      |
| Pilot     | Coordination — you and your drone hitting the same target within a short window triggers bonus damage. Mastery scales the bonus and timing tolerance.     |
| Saboteur  | Chain Reaction — when a trap triggers, nearby traps gain potency. Sequential detonations amplify each successive one. Mastery scales chain amplification. |

#### Blade Dancer

| Spec        | Mastery Effect                                                                                                                                                                             |
| ----------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Multi Blade | Flow — each unique configuration transition within a window extends the window and amplifies the next transition. Repeating a transition breaks the chain. Mastery scales per-chain bonus. |
| Dual Blade  | Convergence — staying in one configuration builds focused energy. Burst abilities consume the charge for bonus damage scaling with time held. Mastery scales charge rate.                  |

#### Tutelaire

| Spec        | Mastery Effect                                                                                                                                                                |
| ----------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Guardian    | Anchor — standing still or moving slowly builds stacks increasing damage-reduction aura strength. Moving quickly drops stacks. Mastery scales per-stack reduction.            |
| Adjudicator | Verdict — Judgment marks on enemies build charges as allies hit marked targets. At threshold, the mark detonates for bonus damage. Mastery scales charge rate and detonation. |
| Luminary    | Radiance — healing an ally briefly increases their damage output. Healing multiple unique allies within a window amplifies the buff. Mastery scales per-ally damage bonus.    |

## Scaling

Stats scale with item level (ilvl). The magnitude-vs-ilvl relationship is **quadratic, but gently sloped** — stat gains accelerate slightly toward the top of the ilvl range rather than rising linearly. The curve is shallow by design: higher ilvl always matters, but the per-ilvl jump never spikes hard enough to make a tier feel mandatory. Ilvl bounds (M+ entry, max, T2 craft) are defined in the [Item System](items.md#ilvl-bounds).

Approximate peak-to-threshold ratios across the full ilvl range:

-   **Hull**: ~3x (tripling HP from baseline to BiS)
-   **Output**: ~2-2.5x
-   **Secondary stats** (Tempo, Identity, Mastery, Plating): ~1.5-2x

These ratios ensure gear progression feels meaningful without creating unplayable gaps between tiers.
