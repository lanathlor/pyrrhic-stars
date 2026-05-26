class_name SpecData

## Static spec/class data constants. Referenced by main.gd to avoid bloating
## the orchestrator script with dictionary literals.

const CLASS_INFO := {
	"gunner":
	{
		"name": "Gunner",
		"genre": "FPS",
		"desc": "Fast movement, high fire rate.\nRelentless aggression."
	},
	"vanguard":
	{
		"name": "Vanguard",
		"genre": "Souls-like",
		"desc": "Big AoE swings, punish windows.\nHeavy and deliberate."
	},
	"blade_dancer":
	{
		"name": "Blade Dancer",
		"genre": "State Machine",
		"desc": "5 configurations, 4 abilities each.\nHighest skill ceiling."
	},
	"arcanotechnicien":
	{
		"name": "Arcanotechnicien",
		"genre": "Tactical Caster",
		"desc": "6 prepared abilities, Flux channeling.\nPosition, channel, protect."
	},
}

const DEFAULT_SPECS := {
	"gunner": "assault",
	"vanguard": "blade",
	"blade_dancer": "multi_blade",
	"arcanotechnicien": "harmonist",
}

const SPEC_INFO := {
	"gunner":
	[
		{
			"id": "assault",
			"name": "Assault",
			"role": "DPS",
			"target": "Monotarget",
			"damage": "Constant",
			"desc":
			"High fire rate, aggressive repositioning.\nRelentless aggression with movement mastery.",
			"mastery": "Pressure — consecutive hits stack damage (max 10). Resets on miss or swap.",
			"implemented": true
		},
		{
			"id": "marksman",
			"name": "Marksman",
			"role": "DPS",
			"target": "Monotarget",
			"damage": "Burst",
			"desc": "Slow, deliberate, perfect shots.\nSniper Elite — hold breath, one shot.",
			"mastery": "Patience — next shot bonus scales with time since last shot (caps 5s).",
			"implemented": false
		},
		{
			"id": "chasseur",
			"name": "Chasseur",
			"role": "DPS",
			"target": "AoE",
			"damage": "Burst",
			"desc": "Grenades, EMP, area denial.\nRainbow Six tactical disruption.",
			"mastery": "Quarry — disrupting a target's ability grants a damage bonus window.",
			"implemented": false
		},
	],
	"vanguard":
	[
		{
			"id": "blade",
			"name": "Blade",
			"role": "DPS",
			"target": "AoE",
			"damage": "Burst",
			"desc":
			(
				"Blade swirl, ground slam, commit-to-cleave.\n"
				+ "AoE burst damage, Dynasty Warriors meets Dark Souls."
			),
			"mastery": "Onslaught — successive hits without taking damage stack bonus damage.",
			"implemented": true
		},
		{
			"id": "shield",
			"name": "Shield",
			"role": "Tank",
			"target": "",
			"damage": "",
			"desc":
			"Directional block, absorbs for allies.\nMonster Hunter lance — slow, unbreakable.",
			"mastery": "Devotion — absorb ally damage, charges your next ability.",
			"implemented": true
		},
		{
			"id": "shadow",
			"name": "Shadow",
			"role": "DPS",
			"target": "Monotarget",
			"damage": "Constant",
			"desc":
			"Counters, flanking, sustained stealth pressure.\nSekiro — dodge, punish, repeat.",
			"mastery": "Afterimage — dodging an attack grants bonus damage on next hit.",
			"implemented": false
		},
	],
	"blade_dancer":
	[
		{
			"id": "multi_blade",
			"name": "Multi Blade",
			"role": "DPS",
			"target": "AoE",
			"damage": "Constant",
			"desc":
			"4-6 blades, scattered multi-target sustained.\nFlowing between 5 configurations.",
			"mastery": "Flow — unique config transitions extend and amplify the chain.",
			"implemented": true
		},
		{
			"id": "dual_blade",
			"name": "Dual Blade",
			"role": "DPS",
			"target": "Monotarget",
			"damage": "Burst",
			"desc":
			"2 blades, separate GCDs, piano burst combos.\nHighest skill ceiling in the game.",
			"mastery": "Convergence — staying in one config builds energy for a burst.",
			"implemented": false
		},
	],
	"arcanotechnicien":
	[
		{
			"id": "destroyer",
			"name": "Destroyer",
			"role": "DPS",
			"target": "AoE",
			"damage": "Burst",
			"desc":
			(
				"Massive AoE burst. Long channels, "
				+ "Overcharge risk/reward.\n"
				+ "Glass cannon with devastating abilities."
			),
			"mastery":
			"Overcharge — hold past completion for bonus damage. Miss the window, suffer backlash.",
			"implemented": false
		},
		{
			"id": "battlemage",
			"name": "Battlemage",
			"role": "DPS",
			"target": "Monotarget",
			"damage": "Constant",
			"desc":
			"Melee-range hybrid. Alternate strikes and abilities.\nWarrior-mage in constant motion.",
			"mastery":
			"Weave — alternating weapon strikes and abilities stacks damage bonus (max 8).",
			"implemented": false
		},
		{
			"id": "harmonist",
			"name": "Harmonist",
			"role": "Healer",
			"target": "Ally",
			"damage": "",
			"desc":
			"Flux-based positional healer. Zone, Beam, Direct.\nRedistribute life force, not whack-a-mole.",
			"mastery":
			"Harmony — cycling delivery methods (Zone/Beam/Direct) triggers bonus heals.",
			"implemented": true
		},
	],
}
