extends Node

## Vanguard HUD updater: ability bar, onslaught/devotion display.

var ctrl: Node


func _ready() -> void:
	ctrl = get_parent()


func update_hud() -> void:
	if ctrl.spec_id == "shield":
		_update_shield_hud()
	else:
		_update_blade_hud()


func _update_shield_hud() -> void:
	ctrl.hud.update_devotion(ctrl._devotion_tier, ctrl._devotion_stacks)
	(
		ctrl
		. hud
		. update_abilities(
			[
				{
					name = "Shield Bash",
					keybind = "LMB",
					desc = "Quick bash. Works during block.",
					cooldown = 0.0,
					cooldown_max = 0.0,
					stamina_cost = ctrl.SHIELD_BASH_STAMINA
				},
				{
					name = "Shield Block",
					keybind = "RMB",
					desc = "Block damage. Drain stamina on hit.",
					cooldown = ctrl._shield_block_cooldown,
					cooldown_max = ctrl.SHIELD_BLOCK_COOLDOWN
				},
				{
					name = "Bull Rush",
					keybind = "R",
					desc = "Charge forward. AoE at end.",
					cooldown = ctrl._bull_rush_cooldown,
					cooldown_max = ctrl.BULL_RUSH_COOLDOWN,
					stamina_cost = ctrl.BULL_RUSH_STAMINA
				},
				{
					name = "Dodge",
					keybind = "C",
					desc = "I-frame dodge.",
					cooldown = 0.0,
					cooldown_max = 0.0,
					stamina_cost = ctrl.dodge_stamina_cost
				},
				{
					name = "Brace",
					keybind = "F",
					desc = "Plant feet. Reduces stamina drain while blocking.",
					cooldown = ctrl._brace_cooldown,
					cooldown_max = ctrl.BRACE_COOLDOWN
				},
				{
					name = "Retaliate",
					keybind = "T",
					desc = "Consume Devotion. Massive frontal slam.",
					cooldown = ctrl._retaliate_cooldown,
					cooldown_max = ctrl.RETALIATE_COOLDOWN
				},
			]
		)
	)


func _update_blade_hud() -> void:
	ctrl.hud.update_onslaught(ctrl._onslaught_tier, ctrl._onslaught_stacks)
	ctrl.hud.update_abilities(_build_blade_ability_bar())


func _build_blade_ability_bar() -> Array[Dictionary]:
	var tier_suffix: String = ""
	if ctrl._onslaught_tier == 1:
		tier_suffix = "+"
	elif ctrl._onslaught_tier == 2:
		tier_suffix = "++"
	return [
		{
			name = "Cleave" + tier_suffix,
			keybind = "LMB",
			desc = "Fast sweep. Arc widens with Onslaught.",
			cooldown = 0.0,
			cooldown_max = 0.0,
			stamina_cost = ctrl.CLEAVE_STAMINA
		},
		{
			name = "Block",
			keybind = "RMB",
			desc = "Parry counter-swing builds Onslaught.",
			cooldown = ctrl._block_cooldown,
			cooldown_max = 3.0
		},
		{
			name = "Upheaval" + tier_suffix,
			keybind = "R",
			desc = "Cone slam. Wider at empowered, DoT at max.",
			cooldown = 0.0,
			cooldown_max = 0.0,
			stamina_cost = ctrl.UPHEAVAL_STAMINA
		},
		{
			name = "Dodge",
			keybind = "C",
			desc = "I-frame dodge. Preserves Onslaught.",
			cooldown = 0.0,
			cooldown_max = 0.0,
			stamina_cost = ctrl.dodge_stamina_cost
		},
		{
			name = "Vortex" + tier_suffix,
			keybind = "F",
			desc = "Forward spin dash. More hits at higher tier.",
			cooldown = ctrl._vortex_cooldown,
			cooldown_max = ctrl.VORTEX_COOLDOWN,
			stamina_cost = ctrl.VORTEX_STAMINA
		},
		{
			name = "Execution" + tier_suffix,
			keybind = "T",
			desc = "Devastating chop. Shockwave at empowered+.",
			cooldown = ctrl._execution_cooldown,
			cooldown_max = ctrl.EXECUTION_COOLDOWN,
			stamina_cost = ctrl.EXECUTION_STAMINA
		},
	]
