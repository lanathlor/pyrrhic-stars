# Blade Dancer

**Gameplay: Positional State Machine**

Camera: third person, pulled back to see blade positions. Input: target-lock, 4 ability buttons that change based on current blade configuration. Core loop: chain configuration transitions to set up optimal ability sequences.

Flux usage: moderate. Powers telekinetic blade control.

The blades exist in configurations (states). Each of the 4 abilities does something different depending on current configuration AND transitions the blades to a new configuration.

## Configurations

- **Orbit**: blades spinning around the player. Defensive abilities (shields, reflects)
- **Fan**: blades spread in an arc ahead. AoE abilities (sweeps, slashes)
- **Lance**: blades stacked in a line at target. Single target burst (piercing strikes)
- **Scatter**: blades flying to multiple targets. Multi-target DoTs

No cooldowns, small GCD. Skill expression is speed of decision-making and planning transitions ahead. Beginners mash and blades do stuff. Experts chain transitions like chess moves.

| Spec | Identity | Playstyle |
|---|---|---|
| Single Blade | One massive blade | Slower transitions, bigger hits, simpler states. Beginner-friendly |
| Multi Blade | 4-6 blades | Full state machine, complex, highest skill ceiling in the game |

TTRPG source: new class, no direct TTRPG equivalent
