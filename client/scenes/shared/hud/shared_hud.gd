extends Control

## Shared HUD overlay — drawn on top of the game world, below class-specific HUDs.
## Contains: player status, group frames, boss frame, damage meter, minimap.
## Managed by main.gd which feeds it data from network events.

const MapData := preload("res://scenes/shared/hud/map_data.gd")

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
var _fight_over: bool = false  # keep boss frame visible after fight ends

# --- Damage Meter (bottom right) ---
var _damage_totals: Dictionary = {}  # pid → float
var _fight_active: bool = false
var _fight_duration: float = 0.0

# --- Minimap (top right) ---
var _enemy_positions: Array = []  # Array of Vector3 for all alive enemies
var _npc_positions: Array = []    # Array of Vector3 for NPCs
var _enemy_alive: bool = false
var _player_rot_y: float = 0.0
var _boss_max_health: float = 2000.0
var _hub_mode: bool = false
var _current_floor_id: String = ""
var _floor_rects: Array = []      # Array of {rect: Rect2, type: String}
var _floor_circles: Array = []    # Array of {center: Vector2, radius: float, green: bool}
var _waypoint_target: Vector3 = Vector3.ZERO
var _has_waypoint: bool = false
var _floor_check_timer: float = 0.0
var _environment: Node3D = null    # ref to current environment scene for scanning

# --- Constants ---
const CLASS_MAX_HP := {
	"gunner": 150.0,
	"vanguard": 200.0,
	"blade_dancer": 150.0,
}
const ENEMY_MAX_HP := 2000.0
const MINIMAP_RADIUS := 80.0
const MINIMAP_WORLD_RADIUS := 25.0  # world units shown in minimap
const MINIMAP_CIRCLE_POINTS := 48   # vertices for circle clipping polygon
var _minimap_circle_poly: PackedVector2Array  # circle polygon centered at origin


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

		# Throttled floor detection for minimap geometry
		if _hub_mode:
			_floor_check_timer += delta
			if _floor_check_timer >= 0.5:
				_floor_check_timer = 0.0
				_detect_floor(_local_player.global_position)

	if _fight_active:
		_fight_duration += delta

	queue_redraw()


func _draw() -> void:
	_draw_player_status()
	_draw_group_frames()
	if _boss_visible:
		_draw_boss_frame()
	if _fight_active or _boss_visible or _fight_over:
		_draw_damage_meter()
	if _hub_mode or _fight_active or _boss_visible or _fight_over:
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
	_fight_over = false
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

	# Enemies — track boss and all alive positions for minimap
	var enemies: Array = data.get("enemies", [])
	_enemy_positions.clear()
	_enemy_alive = false
	# Only update boss visibility if fight is not over (preserve frame on result screen)
	if not _fight_over:
		_boss_visible = false
	for edata in enemies:
		if edata.get("alive", false):
			_enemy_alive = true
			_enemy_positions.append(edata.get("pos", Vector3.ZERO))
			# Boss frame: show only for the guard_captain
			if edata.get("def_name", "") == "guard_captain":
				_boss_health = edata.get("health", 0.0)
				_boss_phase = edata.get("phase", 1)
				_boss_max_health = edata.get("max_health", 2000.0)
				_boss_visible = true

	# NPCs — yellow dots on minimap
	var npcs: Array = data.get("npcs", [])
	_npc_positions.clear()
	for ndata in npcs:
		_npc_positions.append(ndata.get("pos", Vector3.ZERO))


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
	# Only count damage TO enemies (enemy IDs are >= 1000)
	if target >= 1000 and source > 0:
		_damage_totals[source] = _damage_totals.get(source, 0.0) + amount


func on_fight_start() -> void:
	_fight_active = true
	_fight_over = false
	# Boss visibility is driven by update_world_state — guard_captain presence
	_damage_totals.clear()
	_fight_duration = 0.0


func on_fight_end() -> void:
	_fight_active = false
	_fight_over = true
	# Keep boss frame and damage meter visible for result screen


func set_environment(env: Node3D) -> void:
	_environment = env


func on_enter_hub() -> void:
	_hub_mode = true
	_boss_visible = false
	_fight_active = false
	_fight_over = false
	_damage_totals.clear()
	_fight_duration = 0.0
	_world_players.clear()
	_current_floor_id = ""
	_floor_check_timer = 10.0  # force immediate floor check


func on_enter_arena() -> void:
	_hub_mode = false
	_current_floor_id = "arena"
	_has_waypoint = false
	_update_floor_geometry()


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
	var hp_ratio := clampf(_boss_health / maxf(_boss_max_health, 1.0), 0.0, 1.0)
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
	var hp_text := "%d / %d" % [int(_boss_health), int(_boss_max_health)]
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

	# Build circle clipping polygon (centered at map_center)
	if _minimap_circle_poly.is_empty():
		_minimap_circle_poly = PackedVector2Array()
		for i in MINIMAP_CIRCLE_POINTS:
			var angle := TAU * i / float(MINIMAP_CIRCLE_POINTS)
			_minimap_circle_poly.append(Vector2(cos(angle), sin(angle)) * MINIMAP_RADIUS)

	var circle_at_center := PackedVector2Array()
	for pt in _minimap_circle_poly:
		circle_at_center.append(pt + map_center)

	# Background circle — fully opaque dark
	draw_colored_polygon(circle_at_center, Color(0.08, 0.08, 0.10, 1.0))

	# Get local player position for centering
	var local_pos := Vector3.ZERO
	if _world_players.has(_local_peer_id):
		local_pos = _world_players[_local_peer_id]["pos"]
	elif _local_player and is_instance_valid(_local_player):
		local_pos = _local_player.global_position

	# Floor geometry — layered and clipped to circle
	# Color map matching the full map overlay
	var type_colors := {
		"floor": Color(0.30, 0.30, 0.34, 1.0),
		"garden": Color(0.12, 0.28, 0.12, 1.0),
		"ground": Color(0.38, 0.38, 0.42, 1.0),
		"green": Color(0.15, 0.38, 0.15, 1.0),
		"wall": Color(0.18, 0.18, 0.22, 1.0),
	}
	var draw_order := ["floor", "garden", "ground", "green", "wall"]
	for layer in draw_order:
		var color: Color = type_colors[layer]
		for entry in _floor_rects:
			if entry["type"] == layer:
				_draw_minimap_rect_clipped(entry["rect"], local_pos, map_center,
					scale_factor, color, circle_at_center)

	# Border on top of geometry
	draw_arc(map_center, MINIMAP_RADIUS + 2.0, 0.0, TAU, 64, Color(0.5, 0.5, 0.55, 1.0), 2.0)

	# NPCs (yellow dots)
	for npc_pos in _npc_positions:
		var npc_map := _world_to_minimap(npc_pos, local_pos, map_center, scale_factor)
		if npc_map.distance_to(map_center) <= MINIMAP_RADIUS:
			draw_circle(npc_map, 2.5, Color(0.9, 0.75, 0.2, 0.8))

	# Other players (green dots)
	for pid in _world_players:
		if pid == _local_peer_id:
			continue
		var world_pos: Vector3 = _world_players[pid]["pos"]
		var map_pos := _world_to_minimap(world_pos, local_pos, map_center, scale_factor)
		if map_pos.distance_to(map_center) <= MINIMAP_RADIUS:
			draw_circle(map_pos, 3.0, Color(0.3, 0.9, 0.3, 0.9))

	# Enemies (red dots)
	for epos in _enemy_positions:
		var enemy_map := _world_to_minimap(epos, local_pos, map_center, scale_factor)
		if enemy_map.distance_to(map_center) <= MINIMAP_RADIUS:
			draw_circle(enemy_map, 4.0, Color(0.9, 0.2, 0.2, 0.9))

	# Self arrow (center of minimap, rotated)
	_draw_minimap_arrow(map_center, _player_rot_y, Color(1.0, 1.0, 1.0, 0.9), 6.0)

	# Waypoint indicator
	if _has_waypoint:
		var wp_map := _world_to_minimap(_waypoint_target, local_pos, map_center, scale_factor)
		_draw_waypoint_indicator(map_center, wp_map)

	# Cardinal markers
	var font := ThemeDB.fallback_font
	var marker_color := Color(0.5, 0.5, 0.5, 0.5)
	draw_string(font, map_center + Vector2(-3.0, -MINIMAP_RADIUS - 4.0), "N",
		HORIZONTAL_ALIGNMENT_CENTER, 10, 9, marker_color)


func _world_to_minimap(world_pos: Vector3, center_pos: Vector3, map_center: Vector2, scale: float) -> Vector2:
	var dx := (world_pos.x - center_pos.x) * scale
	var dz := (world_pos.z - center_pos.z) * scale  # -Z is forward in Godot, -Y is up on screen
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


func _draw_minimap_rect_clipped(rect: Rect2, center_pos: Vector3, map_center: Vector2,
		scale: float, color: Color, clip_circle: PackedVector2Array) -> void:
	## Draw a world-space Rect2 (XZ plane) on the minimap, clipped to the circle.
	var world_center := Vector3(rect.position.x + rect.size.x / 2.0, 0.0,
		rect.position.y + rect.size.y / 2.0)
	var half_diag := Vector2(rect.size.x, rect.size.y).length() / 2.0 * scale
	var mc := _world_to_minimap(world_center, center_pos, map_center, scale)
	# Skip if entirely outside minimap circle
	if mc.distance_to(map_center) > MINIMAP_RADIUS + half_diag:
		return

	# Transform four corners to minimap screen space
	var tl := _world_to_minimap(Vector3(rect.position.x, 0.0, rect.position.y),
		center_pos, map_center, scale)
	var tr := _world_to_minimap(Vector3(rect.end.x, 0.0, rect.position.y),
		center_pos, map_center, scale)
	var br := _world_to_minimap(Vector3(rect.end.x, 0.0, rect.end.y),
		center_pos, map_center, scale)
	var bl := _world_to_minimap(Vector3(rect.position.x, 0.0, rect.end.y),
		center_pos, map_center, scale)

	# Skip sub-pixel slivers (walls with tiny depth)
	if tl.distance_to(tr) < 1.0 and tl.distance_to(bl) < 1.0:
		return

	var rect_poly := PackedVector2Array([tl, tr, br, bl])
	var clipped := Geometry2D.intersect_polygons(rect_poly, clip_circle)
	for poly in clipped:
		if poly.size() >= 3:
			# Compute area to skip degenerate slivers
			var area := 0.0
			for i in poly.size():
				var j := (i + 1) % poly.size()
				area += poly[i].x * poly[j].y - poly[j].x * poly[i].y
			if absf(area) > 1.0:
				draw_colored_polygon(poly, color)


func _draw_waypoint_indicator(map_center: Vector2, target_map_pos: Vector2) -> void:
	var offset := target_map_pos - map_center
	var dist := offset.length()
	var waypoint_color := Color(0.3, 0.55, 1.0, 0.9)  # flux blue
	if dist < 3.0:
		return
	if dist <= MINIMAP_RADIUS - 6.0:
		# On-map: draw diamond
		_draw_diamond(target_map_pos, 5.0, waypoint_color)
	else:
		# Edge: clamp and draw chevron pointing outward
		var dir := offset.normalized()
		var edge_pos := map_center + dir * (MINIMAP_RADIUS - 6.0)
		_draw_chevron(edge_pos, dir, waypoint_color)


func _draw_diamond(pos: Vector2, half_size: float, color: Color) -> void:
	var pts := PackedVector2Array([
		pos + Vector2(0, -half_size),
		pos + Vector2(half_size, 0),
		pos + Vector2(0, half_size),
		pos + Vector2(-half_size, 0),
	])
	draw_colored_polygon(pts, color)


func _draw_chevron(pos: Vector2, dir: Vector2, color: Color) -> void:
	var perp := Vector2(dir.y, -dir.x)
	var tip := pos + dir * 6.0
	var left_pt := pos - perp * 4.0
	var right_pt := pos + perp * 4.0
	draw_colored_polygon(PackedVector2Array([tip, left_pt, right_pt]), color)


# =============================================================================
# Floor detection and geometry caching
# =============================================================================

func _detect_floor(player_pos: Vector3) -> void:
	var floor_def := MapData.get_floor_for_position(player_pos)
	var new_id: String = floor_def.get("id", "")
	if new_id != _current_floor_id:
		_current_floor_id = new_id
		if floor_def.has("target"):
			_waypoint_target = floor_def["target"]
			_has_waypoint = true
		else:
			_has_waypoint = false
	# Always rescan — geometry may have been invalidated
	_update_floor_geometry()


func _update_floor_geometry() -> void:
	_floor_rects.clear()
	_floor_circles.clear()
	if _environment and is_instance_valid(_environment) and _local_player and is_instance_valid(_local_player):
		var result: Dictionary = MapData.scan_scene(_environment, _local_player.global_position.y)
		_floor_rects = result["rects"]
		_floor_circles = result["circles"]
	else:
		# Fallback to MapData if no environment available
		var geo := MapData.get_geometry_for_floor(_current_floor_id)
		if geo.is_empty():
			return
		var buildings: Array = geo.get("buildings", [])
		for b in buildings:
			var c: Vector2 = b["center"]
			var s: Vector2 = b["size"]
			_floor_rects.append({"rect": Rect2(c.x - s.x / 2.0, c.y - s.y / 2.0, s.x, s.y), "type": "wall"})
