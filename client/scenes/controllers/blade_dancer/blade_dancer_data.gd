class_name BladeDancerData

## Static data for Blade Dancer configurations and transition abilities.
## 5 blade configurations, 20 transition abilities (4 per config).

## All 20 transition abilities. TABLE[origin_config][slot] -> ability dict.
## Each ability transitions from origin_config to dest config.
## action_id = 30 + origin_config * 4 + slot
const ABILITY_TABLE := {
	0:  # ORBIT
	[
		{
			name = "Shielded Sweep",
			dest = 1,
			dur = 0.4,
			action_id = 30,
			telegraph = "circle",
			radius = 4.0,
			desc = "8 dmg AoE (4m). 15% DR for 2s."
		},
		{
			name = "Protected Scatter",
			dest = 3,
			dur = 0.4,
			action_id = 31,
			telegraph = "none",
			desc = "5 dmg x3 nearest. 1.5/tick DoT 12s. 10% DR."
		},
		{
			name = "Guarded Thrust",
			dest = 2,
			dur = 0.3,
			action_id = 32,
			telegraph = "none",
			desc = "25 dmg single. +8 shield."
		},
		{
			name = "Fortified Command",
			dest = 4,
			dur = 0.5,
			action_id = 33,
			telegraph = "circle_target",
			radius = 5.0,
			desc = "5 dmg AoE at target (5m). 20% DR for 2s."
		},
	],
	1:  # FAN
	[
		{
			name = "Reaping Guard",
			dest = 0,
			dur = 0.4,
			action_id = 34,
			telegraph = "circle",
			radius = 3.0,
			desc = "8 dmg AoE (3m). +12 shield."
		},
		{
			name = "Slashing Spread",
			dest = 3,
			dur = 0.4,
			action_id = 35,
			telegraph = "circle_target",
			radius = 5.0,
			desc = "8 dmg AoE at target (5m). 1.5/tick DoT 10s."
		},
		{
			name = "Cleaving Pierce",
			dest = 2,
			dur = 0.3,
			action_id = 36,
			telegraph = "none",
			desc = "30 dmg single target."
		},
		{
			name = "Sweeping Hex",
			dest = 4,
			dur = 0.5,
			action_id = 37,
			telegraph = "circle_target",
			radius = 5.0,
			desc = "10 dmg AoE at target (5m)."
		},
	],
	2:  # LANCE
	[
		{
			name = "Piercing Barrier",
			dest = 0,
			dur = 0.4,
			action_id = 38,
			telegraph = "none",
			desc = "18 dmg single. +15 shield."
		},
		{
			name = "Focused Slash",
			dest = 1,
			dur = 0.3,
			action_id = 39,
			telegraph = "circle_target",
			radius = 4.0,
			desc = "15 dmg AoE at target (4m)."
		},
		{
			name = "Targeted Spread",
			dest = 3,
			dur = 0.4,
			action_id = 40,
			telegraph = "none",
			desc = "12 dmg single. 2.0/tick DoT 15s."
		},
		{
			name = "Pinning Strike",
			dest = 4,
			dur = 0.3,
			action_id = 41,
			telegraph = "none",
			desc = "25 dmg single target."
		},
	],
	3:  # SCATTER
	[
		{
			name = "Dispersed Shield",
			dest = 0,
			dur = 0.5,
			action_id = 42,
			telegraph = "none",
			desc = "+18 shield. 15% DR for 2s."
		},
		{
			name = "Converging Strike",
			dest = 2,
			dur = 0.3,
			action_id = 43,
			telegraph = "none",
			desc = "32 dmg single. 1.5/tick DoT 10s."
		},
		{
			name = "Rain of Blades",
			dest = 1,
			dur = 0.4,
			action_id = 44,
			telegraph = "circle_target",
			radius = 5.0,
			desc = "15 dmg AoE at target (5m). 1.0/tick DoT 10s."
		},
		{
			name = "Chaos Bind",
			dest = 4,
			dur = 0.5,
			action_id = 45,
			telegraph = "none",
			desc = "8 dmg x4 nearest enemies."
		},
	],
	4:  # CROWN
	[
		{
			name = "Commanding Ward",
			dest = 0,
			dur = 0.5,
			action_id = 46,
			telegraph = "none",
			desc = "+20 shield."
		},
		{
			name = "Decree Strike",
			dest = 2,
			dur = 0.3,
			action_id = 47,
			telegraph = "none",
			desc = "28 dmg single target."
		},
		{
			name = "Royal Cleave",
			dest = 1,
			dur = 0.3,
			action_id = 48,
			telegraph = "circle",
			radius = 5.0,
			desc = "12 dmg AoE (5m)."
		},
		{
			name = "Sovereign Scatter",
			dest = 3,
			dur = 0.4,
			action_id = 49,
			telegraph = "none",
			desc = "5 dmg x3 nearest. 1.5/tick DoT 12s."
		},
	],
}
