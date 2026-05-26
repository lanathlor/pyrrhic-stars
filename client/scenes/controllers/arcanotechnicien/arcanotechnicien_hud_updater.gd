extends Node

## Arcanotechnicien HUD updater: abilities, channel bar, party frames.

var ctrl: Node


func _ready() -> void:
	ctrl = get_parent()


func update_hud() -> void:
	update_abilities()
	ctrl.hud.update_gcd(ctrl._gcd_timer / ctrl.gcd_duration if ctrl._gcd_timer > 0.0 else 0.0)
	ctrl.hud.update_confluence(ctrl._confluence_tier, ctrl._confluence_stacks)
	if ctrl.vfx:
		ctrl.vfx.update_confluence(ctrl._confluence_tier, ctrl._confluence_stacks)
	ctrl.hud.update_flux(ctrl.flux, ctrl.max_flux, ctrl.flux_pools)
	update_channel()
	update_party()


func update_abilities() -> void:
	var ability_data: Array = []
	# Use server catalog if populated, otherwise fall back to hardcoded HARMONIST_ABILITIES.
	if AbilityCatalog.catalog.size() > 0:
		for i in 6:
			var ability_id: String = (
				AbilityCatalog.current_loadout[i]
				if i < AbilityCatalog.current_loadout.size()
				else ""
			)
			if ability_id == "":
				ability_data.append(
					{
						name = "Empty",
						keybind = ctrl.SLOT_KEYBINDS[i],
						desc = "",
						cooldown = 0.0,
						cooldown_max = 0.0
					}
				)
				continue
			var entry: Dictionary = AbilityCatalog.get_ability(ability_id)
			(
				ability_data
				. append(
					{
						name = entry.get("name", ability_id),
						keybind = ctrl.SLOT_KEYBINDS[i],
						desc = entry.get("description", ""),
						cooldown = ctrl._cooldowns[i],
						cooldown_max = entry.get("cooldown", 0.0),
					}
				)
			)
	else:
		for i in ctrl.HARMONIST_ABILITIES.size():
			var ability: Dictionary = ctrl.HARMONIST_ABILITIES[i]
			(
				ability_data
				. append(
					{
						name = ability.name,
						keybind = ability.keybind,
						desc = ability.desc,
						cooldown = ctrl._cooldowns[i],
						cooldown_max = ability.cooldown_max,
					}
				)
			)
	ctrl.hud.update_abilities(ability_data)


func update_channel() -> void:
	if ctrl.state == ctrl.State.CHANNELING and not ctrl._committing_ability.is_empty():
		if ctrl.combat._sustaining:
			ctrl.hud.update_sustain(
				ctrl._committing_ability.get("name", ""), ctrl.combat._sustain_elapsed
			)
		else:
			var total_dur: float = ctrl._committing_ability.get(
				"dur", ctrl._committing_ability.get("commit_time", 1.0)
			)
			var elapsed: float = total_dur - ctrl._cast_timer
			var progress: float = clampf(elapsed / maxf(total_dur, 0.01), 0.0, 1.0)
			ctrl.hud.update_channel(progress, ctrl._committing_ability.get("name", ""))
	else:
		ctrl.hud.hide_channel()


func update_party() -> void:
	var party: Array = []
	for p in GameManager.players:
		if not is_instance_valid(p) or not p.visible:
			continue
		if p == ctrl:
			continue
		var pid: int = p.peer_id if "peer_id" in p else 0
		var p_health: float = p.health if "health" in p else 0.0
		var p_max_health: float = p.max_health if "max_health" in p else 150.0
		var cls: String = "unknown"
		var uname: String = "Player_%d" % pid
		if NetworkManager.player_info.has(pid):
			cls = NetworkManager.player_info[pid].get("class_name", "unknown")
			var info_name: String = NetworkManager.player_info[pid].get("username", "")
			if info_name != "":
				uname = info_name
		(
			party
			. append(
				{
					"peer_id": pid,
					"name": uname,
					"health": p_health,
					"max_health": p_max_health,
					"class_name": cls,
				}
			)
		)
	ctrl.hud.update_party(party)
