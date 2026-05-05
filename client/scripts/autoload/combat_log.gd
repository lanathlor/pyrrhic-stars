extends Node

## Logs combat events during a fight and writes to JSON after fight ends.
## Output: res://test_output/combat_log.json (client/test_output/ on disk).

var output_dir: String = "res://test_output/"

var _events: Array[Dictionary] = []
var _fight_active: bool = false
var _fight_start_time: float = 0.0
var _summary: Dictionary = {}


func _ready() -> void:
	DirAccess.make_dir_recursive_absolute(ProjectSettings.globalize_path(output_dir))


func start_fight() -> void:
	_events.clear()
	_fight_active = true
	_fight_start_time = Time.get_ticks_msec() / 1000.0
	_summary = {
		"total_boss_damage_dealt": 0.0,
		"total_player_damage_dealt": 0.0,
		"attacks_used": {},
		"attacks_hit": {},
		"attacks_missed": {},
		"phase_transitions": [],
		"player_dodges": 0,
		"player_blocks": 0,
	}
	_log("fight_start", {})


func end_fight(result: String) -> void:
	if not _fight_active:
		return
	_log("fight_end", {"result": result})
	_fight_active = false
	_write_log()


func log_boss_attack(
	attack_type: String, phase: int, boss_pos: Vector3, target_pos: Vector3
) -> void:
	if not _fight_active:
		return
	_summary["attacks_used"][attack_type] = _summary["attacks_used"].get(attack_type, 0) + 1
	_log(
		"boss_attack",
		{
			"attack": attack_type,
			"phase": phase,
			"boss_pos": _v3(boss_pos),
			"target_pos": _v3(target_pos),
		}
	)


func log_boss_hit(
	attack_type: String, damage: float, player_name: String, player_pos: Vector3
) -> void:
	if not _fight_active:
		return
	_summary["total_boss_damage_dealt"] += damage
	_summary["attacks_hit"][attack_type] = _summary["attacks_hit"].get(attack_type, 0) + 1
	_log(
		"boss_hit",
		{
			"attack": attack_type,
			"damage": damage,
			"target": player_name,
			"target_pos": _v3(player_pos),
		}
	)


func log_boss_miss(attack_type: String) -> void:
	if not _fight_active:
		return
	_summary["attacks_missed"][attack_type] = _summary["attacks_missed"].get(attack_type, 0) + 1
	_log("boss_miss", {"attack": attack_type})


func log_player_damage(amount: float, player_pos: Vector3, boss_state: String) -> void:
	if not _fight_active:
		return
	_summary["total_player_damage_dealt"] += amount
	_log(
		"player_damage",
		{
			"damage": amount,
			"player_pos": _v3(player_pos),
			"boss_state": boss_state,
		}
	)


func log_phase_transition(phase: int, boss_hp: float, time: float) -> void:
	if not _fight_active:
		return
	_summary["phase_transitions"].append({"phase": phase, "hp": boss_hp, "time": time})
	_log("phase_transition", {"phase": phase, "boss_hp": boss_hp})


func log_boss_stuck(boss_pos: Vector3, target_pos: Vector3, stuck_time: float) -> void:
	if not _fight_active:
		return
	_log(
		"boss_stuck",
		{
			"boss_pos": _v3(boss_pos),
			"target_pos": _v3(target_pos),
			"stuck_time": snappedf(stuck_time, 0.01),
		}
	)


func _log(event_type: String, data: Dictionary) -> void:
	var entry := {"t": snappedf(_elapsed(), 0.01), "event": event_type}
	entry.merge(data)
	_events.append(entry)


func _elapsed() -> float:
	return Time.get_ticks_msec() / 1000.0 - _fight_start_time


func _write_log() -> void:
	var output := {
		"summary": _summary,
		"fight_duration": snappedf(_elapsed(), 0.01),
		"total_events": _events.size(),
		"events": _events,
	}
	var json := JSON.stringify(output, "  ")
	var path := ProjectSettings.globalize_path(output_dir + "combat_log.json")
	var file := FileAccess.open(path, FileAccess.WRITE)
	if file:
		file.store_string(json)
		print("[CombatLog] Written %d events to %s" % [_events.size(), path])


func _v3(v: Vector3) -> Array:
	return [snappedf(v.x, 0.1), snappedf(v.y, 0.1), snappedf(v.z, 0.1)]
