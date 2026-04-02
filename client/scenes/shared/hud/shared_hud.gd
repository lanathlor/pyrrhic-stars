extends Control

## Shared HUD overlay — drawn on top of the game world, below class-specific HUDs.
## Contains: player status, group frames, boss frame, damage meter, minimap.
## Managed by main.gd which feeds it data from network events.

# --- References (set by main.gd) ---
var _local_player: CharacterBody3D = null
var _local_class: String = "gunner"
var _local_peer_id: int = 0
var _player_names: Dictionary = {}  # ref to main.gd's dict

# --- Player Status (bottom center) ---
var _player_health: float = 100.0
var _player_max_health: float = 150.0
var _player_resource: float = 0.0
var _player_max_resource: float = 0.0

# --- Group Frames (left side) ---
var _group_member_pids: Array = []  # peer IDs from group state
var _group_member_names: Dictionary = {}  # pid → username from group state

# --- World State (from server ticks) ---
var _world_players: Dictionary = {}  # pid → {pos: Vector3, health: float, rot_y: float}

# --- Boss Frame (top center) ---
var _boss_visible: bool = false
var _boss_name: String = "Arena Guardian"
var _boss_health: float = 2000.0
var _boss_phase: int = 1

# --- Damage Meter (bottom right) ---
var _damage_totals: Dictionary = {}  # pid → float
var _fight_active: bool = false
var _fight_duration: float = 0.0

# --- Minimap (top right) ---
var _enemy_pos: Vector3 = Vector3.ZERO
var _enemy_alive: bool = false
var _player_rot_y: float = 0.0

# --- Constants ---
const CLASS_MAX_HP := {
	"gunner": 150.0,
	"vanguard": 200.0,
	"blade_dancer": 150.0,
}
const ENEMY_MAX_HP := 2000.0
const MINIMAP_RADIUS := 60.0
const MINIMAP_WORLD_RADIUS := 25.0  # world units shown in minimap


func _process(delta: float) -> void:
	# Read local player state each frame for responsive bars
	if _local_player and is_instance_valid(_local_player):
		_player_health = _local_player.health
		_player_max_health = _local_player.max_health
		_player_rot_y = _local_player.rotation.y
		# Duck-type resource (only Vanguard has stamina)
		if "stamina" in _local_player:
			_player_resource = _local_player.stamina
			_player_max_resource = _local_player.max_stamina
		else:
			_player_resource = 0.0
			_player_max_resource = 0.0

	if _fight_active:
		_fight_duration += delta

	queue_redraw()


func _draw() -> void:
	_draw_player_status()
	_draw_group_frames()
	if _boss_visible:
		_draw_boss_frame()
	if _fight_active or _boss_visible:
		_draw_damage_meter()
	if _boss_visible or _fight_active:
		_draw_minimap()


# =============================================================================
# Public API (called by main.gd)
# =============================================================================

func set_local_player(player: CharacterBody3D, class_name_str: String, peer_id: int) -> void:
	_local_player = player
	_local_class = class_name_str
	_local_peer_id = peer_id
	_player_max_health = CLASS_MAX_HP.get(class_name_str, 150.0)


func clear_local_player() -> void:
	_local_player = null
	_boss_visible = false
	_fight_active = false
	_damage_totals.clear()
	_world_players.clear()


func set_player_names(names: Dictionary) -> void:
	_player_names = names


func update_world_state(data: Dictionary) -> void:
	# Players
	var players_data: Array = data.get("players", [])
	_world_players.clear()
	for p in players_data:
		var pid: int = p["peer_id"]
		_world_players[pid] = {
			"pos": p.get("pos", Vector3.ZERO),
			"health": p.get("health", 0.0),
			"rot_y": p.get("rot_y", 0.0),
		}

	# Enemy / boss
	var enemy: Dictionary = data.get("enemy", {})
	if not enemy.is_empty():
		_enemy_alive = enemy.get("alive", false)
		_enemy_pos = enemy.get("pos", Vector3.ZERO)
		_boss_health = enemy.get("health", 0.0)
		_boss_phase = enemy.get("phase", 1)


func update_group_members(data: Dictionary) -> void:
	var members: Array = data.get("members", [])
	_group_member_pids.clear()
	_group_member_names.clear()
	for m in members:
		var pid: int = m.get("peer_id", 0)
		_group_member_pids.append(pid)
		var uname: String = m.get("username", "")
		if uname != "":
			_group_member_names[pid] = uname


func on_damage_event(data: Dictionary) -> void:
	if not _fight_active:
		return
	var target: int = data.get("target_peer_id", -1)
	var source: int = data.get("source_peer_id", 0)
	var amount: float = data.get("amount", 0.0)
	# Only count damage TO the enemy (target 0 = enemy)
	if target == 0 and source > 0:
		_damage_totals[source] = _damage_totals.get(source, 0.0) + amount


func on_fight_start() -> void:
	_fight_active = true
	_boss_visible = true
	_damage_totals.clear()
	_fight_duration = 0.0


func on_fight_end() -> void:
	_fight_active = false
	# Keep boss frame and damage meter visible for result screen


func on_enter_hub() -> void:
	_boss_visible = false
	_fight_active = false
	_damage_totals.clear()
	_fight_duration = 0.0
	_world_players.clear()


# =============================================================================
# Drawing — Player Status (bottom center)
# =============================================================================

func _draw_player_status() -> void:
	if not _local_player or not is_instance_valid(_local_player):
		return

	var font := ThemeDB.fallback_font
	var center_x := size.x / 2.0
	var bar_w := 220.0
	var bar_h := 14.0
	var bar_x := center_x - bar_w / 2.0
	var bar_y := size.y - 85.0

	# HP bar background
	var bg_rect := Rect2(bar_x, bar_y, bar_w, bar_h)
	draw_rect(bg_rect, Color(0.1, 0.1, 0.1, 0.7))

	# HP bar fill (green)
	var hp_ratio := clampf(_player_health / maxf(_player_max_health, 1.0), 0.0, 1.0)
	var hp_color := Color(0.2, 0.8, 0.2) if hp_ratio > 0.3 else Color(0.8, 0.2, 0.2)
	if hp_ratio > 0.0:
		draw_rect(Rect2(bar_x, bar_y, bar_w * hp_ratio, bar_h), hp_color)

	# HP bar border
	draw_rect(bg_rect, Color(0.3, 0.3, 0.3, 0.8), false, 1.0)

	# HP numbers
	var hp_text := "%d / %d" % [int(_player_health), int(_player_max_health)]
	draw_string(font, Vector2(center_x - 30.0, bar_y + 11.0), hp_text,
		HORIZONTAL_ALIGNMENT_CENTER, 60, 11, Color(1.0, 1.0, 1.0, 0.9))

	# Resource bar (only if player has a resource)
	if _player_max_resource > 0.0:
		var res_h := 8.0
		var res_y := bar_y + bar_h + 3.0
		var res_rect := Rect2(bar_x, res_y, bar_w, res_h)
		draw_rect(res_rect, Color(0.1, 0.1, 0.1, 0.6))

		var res_ratio := clampf(_player_resource / maxf(_player_max_resource, 1.0), 0.0, 1.0)
		var res_color := Color(0.85, 0.75, 0.2)  # gold for stamina
		if res_ratio > 0.0:
			draw_rect(Rect2(bar_x, res_y, bar_w * res_ratio, res_h), res_color)
		draw_rect(res_rect, Color(0.3, 0.3, 0.3, 0.6), false, 1.0)

		var res_text := "%d / %d" % [int(_player_resource), int(_player_max_resource)]
		draw_string(font, Vector2(center_x - 25.0, res_y + 7.0), res_text,
			HORIZONTAL_ALIGNMENT_CENTER, 50, 9, Color(1.0, 1.0, 1.0, 0.7))


# =============================================================================
# Drawing — Group Frames (left side)
# =============================================================================

func _draw_group_frames() -> void:
	# In arena: use WorldState players directly (authoritative peer IDs + live health)
	# In hub: use group member list (no health data available)
	var has_world_data := not _world_players.is_empty()

	var pids_to_show: Array = []
	if has_world_data:
		# Arena: show all players from WorldState except self
		for pid in _world_players:
			if pid != _local_peer_id:
				pids_to_show.append(pid)
	else:
		# Hub: show group members except self
		for pid in _group_member_pids:
			if pid != _local_peer_id:
				pids_to_show.append(pid)

	if pids_to_show.is_empty():
		return

	var font := ThemeDB.fallback_font
	var frame_x := 10.0
	var frame_y := 200.0
	var frame_w := 170.0
	var frame_h := 40.0
	var frame_gap := 4.0

	var drawn := 0
	for pid in pids_to_show:
		if drawn >= 4:
			break

		var y := frame_y + drawn * (frame_h + frame_gap)

		# Frame background
		draw_rect(Rect2(frame_x, y, frame_w, frame_h), Color(0.05, 0.05, 0.1, 0.7))

		# Player name
		var uname: String = _group_member_names.get(pid, _player_names.get(pid, "Player_%d" % pid))
		if uname.length() > 14:
			uname = uname.substr(0, 14)
		draw_string(font, Vector2(frame_x + 6.0, y + 14.0), uname,
			HORIZONTAL_ALIGNMENT_LEFT, frame_w - 12.0, 12, Color(0.9, 0.9, 0.9, 0.9))

		# HP bar
		var hp_bar_x := frame_x + 6.0
		var hp_bar_y := y + 20.0
		var hp_bar_w := frame_w - 12.0
		var hp_bar_h := 10.0
		draw_rect(Rect2(hp_bar_x, hp_bar_y, hp_bar_w, hp_bar_h), Color(0.15, 0.15, 0.15, 0.8))

		# Get max HP from class
		var max_health: float = 150.0
		if NetworkManager.player_info.has(pid):
			var cls: String = NetworkManager.player_info[pid].get("class_name", "gunner")
			max_health = CLASS_MAX_HP.get(cls, 150.0)

		# Health: from WorldState if available, otherwise assume full
		var health: float = max_health
		if _world_players.has(pid):
			health = _world_players[pid]["health"]

		var ratio := clampf(health / maxf(max_health, 1.0), 0.0, 1.0)
		var bar_color := Color(0.2, 0.8, 0.2) if ratio > 0.3 else Color(0.8, 0.2, 0.2)
		if ratio > 0.0:
			draw_rect(Rect2(hp_bar_x, hp_bar_y, hp_bar_w * ratio, hp_bar_h), bar_color)
		draw_rect(Rect2(hp_bar_x, hp_bar_y, hp_bar_w, hp_bar_h), Color(0.3, 0.3, 0.3, 0.5), false, 1.0)

		# HP numbers (small)
		var hp_text := "%d" % int(health)
		draw_string(font, Vector2(hp_bar_x + 4.0, hp_bar_y + 8.0), hp_text,
			HORIZONTAL_ALIGNMENT_LEFT, hp_bar_w, 9, Color(1.0, 1.0, 1.0, 0.7))

		# Frame border
		draw_rect(Rect2(frame_x, y, frame_w, frame_h), Color(0.25, 0.25, 0.3, 0.6), false, 1.0)

		drawn += 1


# =============================================================================
# Drawing — Boss Frame (top center)
# =============================================================================

func _draw_boss_frame() -> void:
	var font := ThemeDB.fallback_font
	var center_x := size.x / 2.0
	var bar_w := 400.0
	var bar_h := 18.0
	var bar_x := center_x - bar_w / 2.0
	var name_y := 18.0
	var bar_y := 28.0

	# Boss name
	draw_string(font, Vector2(center_x - 80.0, name_y), _boss_name,
		HORIZONTAL_ALIGNMENT_CENTER, 160, 16, Color(0.9, 0.85, 0.7, 0.95))

	# Phase indicator
	var phase_text := "P%d" % _boss_phase
	var phase_color: Color
	match _boss_phase:
		1: phase_color = Color(0.3, 0.8, 0.3)
		2: phase_color = Color(0.9, 0.7, 0.2)
		3: phase_color = Color(0.9, 0.2, 0.2)
		_: phase_color = Color(0.5, 0.5, 0.5)
	draw_string(font, Vector2(bar_x + bar_w + 8.0, bar_y + 13.0), phase_text,
		HORIZONTAL_ALIGNMENT_LEFT, 30, 12, phase_color)

	# Bar background
	var bg_rect := Rect2(bar_x, bar_y, bar_w, bar_h)
	draw_rect(bg_rect, Color(0.1, 0.05, 0.05, 0.75))

	# Bar fill — color shifts by phase
	var hp_ratio := clampf(_boss_health / ENEMY_MAX_HP, 0.0, 1.0)
	var bar_color: Color
	match _boss_phase:
		1: bar_color = Color(0.2, 0.75, 0.2)
		2: bar_color = Color(0.9, 0.65, 0.15)
		3: bar_color = Color(0.85, 0.15, 0.15)
		_: bar_color = Color(0.5, 0.5, 0.5)
	if hp_ratio > 0.0:
		draw_rect(Rect2(bar_x, bar_y, bar_w * hp_ratio, bar_h), bar_color)

	# Border
	draw_rect(bg_rect, Color(0.4, 0.35, 0.3, 0.8), false, 1.5)

	# HP numbers
	var hp_text := "%d / %d" % [int(_boss_health), int(ENEMY_MAX_HP)]
	draw_string(font, Vector2(center_x - 50.0, bar_y + 14.0), hp_text,
		HORIZONTAL_ALIGNMENT_CENTER, 100, 12, Color(1.0, 1.0, 1.0, 0.9))


# =============================================================================
# Drawing — Damage Meter (bottom right)
# =============================================================================

func _draw_damage_meter() -> void:
	if _damage_totals.is_empty():
		return

	var font := ThemeDB.fallback_font
	var meter_w := 180.0
	var meter_x := size.x - meter_w - 10.0
	var entry_h := 20.0
	var title_y := size.y - 160.0

	# Sort players by damage (descending)
	var sorted_pids: Array = _damage_totals.keys()
	sorted_pids.sort_custom(func(a, b): return _damage_totals[a] > _damage_totals[b])
	var max_damage: float = _damage_totals.get(sorted_pids[0], 1.0) if sorted_pids.size() > 0 else 1.0
	if max_damage <= 0.0:
		max_damage = 1.0

	# Background
	var entry_count := mini(sorted_pids.size(), 5)
	var bg_h := 20.0 + entry_count * entry_h + 8.0
	draw_rect(Rect2(meter_x - 4.0, title_y - 4.0, meter_w + 8.0, bg_h),
		Color(0.03, 0.03, 0.06, 0.6))

	# Title
	var title := "Damage"
	if _fight_duration > 0.0:
		var total_dmg: float = 0.0
		for pid in _damage_totals:
			total_dmg += _damage_totals[pid]
		var dps := total_dmg / maxf(_fight_duration, 1.0)
		title = "Damage (%.0f DPS)" % dps
	draw_string(font, Vector2(meter_x, title_y + 12.0), title,
		HORIZONTAL_ALIGNMENT_LEFT, meter_w, 11, Color(0.7, 0.7, 0.7, 0.8))

	# Entries
	var class_colors := {
		"gunner": Color(0.3, 0.7, 1.0),
		"vanguard": Color(0.9, 0.4, 0.3),
		"blade_dancer": Color(0.4, 0.9, 0.7),
	}

	for i in entry_count:
		var pid: int = sorted_pids[i]
		var dmg: float = _damage_totals[pid]
		var y := title_y + 20.0 + i * entry_h

		# Bar
		var ratio := dmg / max_damage
		var cls: String = "gunner"
		if NetworkManager.player_info.has(pid):
			cls = NetworkManager.player_info[pid].get("class_name", "gunner")
		var bar_color: Color = class_colors.get(cls, Color(0.5, 0.5, 0.5))
		bar_color.a = 0.5
		draw_rect(Rect2(meter_x, y + 2.0, meter_w * ratio, entry_h - 4.0), bar_color)

		# Name
		var uname: String = _player_names.get(pid, "Player_%d" % pid)
		if uname.length() > 10:
			uname = uname.substr(0, 10)
		draw_string(font, Vector2(meter_x + 4.0, y + 14.0), uname,
			HORIZONTAL_ALIGNMENT_LEFT, meter_w * 0.55, 10, Color(0.9, 0.9, 0.9, 0.9))

		# Damage number
		var dmg_text: String
		if dmg >= 1000.0:
			dmg_text = "%.1fk" % (dmg / 1000.0)
		else:
			dmg_text = "%d" % int(dmg)
		draw_string(font, Vector2(meter_x + meter_w - 50.0, y + 14.0), dmg_text,
			HORIZONTAL_ALIGNMENT_RIGHT, 46, 10, Color(1.0, 1.0, 1.0, 0.8))

	# Border
	draw_rect(Rect2(meter_x - 4.0, title_y - 4.0, meter_w + 8.0, bg_h),
		Color(0.25, 0.25, 0.3, 0.5), false, 1.0)


# =============================================================================
# Drawing — Minimap (top right)
# =============================================================================

func _draw_minimap() -> void:
	var map_center := Vector2(size.x - MINIMAP_RADIUS - 16.0, MINIMAP_RADIUS + 16.0)
	var scale_factor := MINIMAP_RADIUS / MINIMAP_WORLD_RADIUS

	# Background circle
	draw_circle(map_center, MINIMAP_RADIUS + 2.0, Color(0.0, 0.0, 0.0, 0.5))
	draw_arc(map_center, MINIMAP_RADIUS + 2.0, 0.0, TAU, 64, Color(0.3, 0.3, 0.35, 0.7), 1.5)

	# Get local player position for centering
	var local_pos := Vector3.ZERO
	if _world_players.has(_local_peer_id):
		local_pos = _world_players[_local_peer_id]["pos"]
	elif _local_player and is_instance_valid(_local_player):
		local_pos = _local_player.global_position

	# Self arrow (center of minimap, rotated)
	_draw_minimap_arrow(map_center, _player_rot_y, Color(1.0, 1.0, 1.0, 0.9), 6.0)

	# Other players (green dots)
	for pid in _world_players:
		if pid == _local_peer_id:
			continue
		var world_pos: Vector3 = _world_players[pid]["pos"]
		var map_pos := _world_to_minimap(world_pos, local_pos, map_center, scale_factor)
		if map_pos.distance_to(map_center) <= MINIMAP_RADIUS:
			draw_circle(map_pos, 3.0, Color(0.3, 0.9, 0.3, 0.9))

	# Enemy (red dot)
	if _enemy_alive:
		var enemy_map := _world_to_minimap(_enemy_pos, local_pos, map_center, scale_factor)
		if enemy_map.distance_to(map_center) <= MINIMAP_RADIUS:
			draw_circle(enemy_map, 4.0, Color(0.9, 0.2, 0.2, 0.9))

	# Cardinal markers
	var font := ThemeDB.fallback_font
	var marker_color := Color(0.5, 0.5, 0.5, 0.5)
	draw_string(font, map_center + Vector2(-3.0, -MINIMAP_RADIUS - 4.0), "N",
		HORIZONTAL_ALIGNMENT_CENTER, 10, 9, marker_color)


func _world_to_minimap(world_pos: Vector3, center_pos: Vector3, map_center: Vector2, scale: float) -> Vector2:
	var dx := (world_pos.x - center_pos.x) * scale
	var dz := -(world_pos.z - center_pos.z) * scale  # Z is forward in Godot, up on minimap
	return map_center + Vector2(dx, dz)


func _draw_minimap_arrow(pos: Vector2, rot_y: float, color: Color, arrow_size: float) -> void:
	# Arrow pointing in the direction the player faces
	# In Godot: rotation_y = 0 means facing -Z, which is up on minimap
	var angle := -rot_y  # negate because screen Y is down
	var forward := Vector2(sin(angle), -cos(angle))
	var right := Vector2(forward.y, -forward.x)

	var tip := pos + forward * arrow_size
	var left_pt := pos - forward * arrow_size * 0.6 - right * arrow_size * 0.5
	var right_pt := pos - forward * arrow_size * 0.6 + right * arrow_size * 0.5

	draw_colored_polygon(PackedVector2Array([tip, left_pt, right_pt]), color)
