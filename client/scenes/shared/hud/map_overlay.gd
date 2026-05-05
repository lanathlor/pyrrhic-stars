extends Control

## Full-screen map overlay toggled with M key.
## Scans the environment scene's CSG collision nodes at runtime for a true
## top-down view instead of relying on hardcoded geometry data.

const MapData := preload("res://scenes/shared/hud/map_data.gd")

const PADDING := 0.15
const FLOOR_COLOR := Color(0.30, 0.30, 0.34, 1.0)  # plaza, large ground surfaces
const GARDEN_COLOR := Color(0.12, 0.28, 0.12, 1.0)  # large grass/garden areas
const GROUND_COLOR := Color(0.38, 0.38, 0.42, 1.0)  # roads, sidewalks, small paths
const WALL_COLOR := Color(0.18, 0.18, 0.22, 1.0)  # buildings, walls
const GREEN_COLOR := Color(0.15, 0.38, 0.15, 1.0)  # hedges, trees, planters
const BG_COLOR := Color(0.06, 0.06, 0.08, 0.9)  # void / unexplored
const BORDER_COLOR := Color(0.45, 0.45, 0.50, 0.6)  # subtle outlines
const WAYPOINT_COLOR := Color(0.3, 0.55, 1.0, 0.8)
const PLAYER_COLOR := Color(0.3, 0.9, 0.3, 0.9)
const ENEMY_COLOR := Color(0.9, 0.2, 0.2, 0.9)
const NPC_COLOR := Color(0.9, 0.75, 0.2, 0.8)
const SELF_COLOR := Color(1.0, 1.0, 1.0, 0.95)
# Y tolerance: how close a node's Y must be to the player's floor to be included
const Y_TOLERANCE := 15.0
# Minimum XZ footprint area to be drawn (skip tiny collision slivers)
const MIN_FOOTPRINT := 0.1
# Height threshold: boxes taller than this relative to XZ are walls, not floors
const WALL_HEIGHT_RATIO := 0.5

var _current_floor_id: String = ""
var _floor_name: String = ""
var _scanned_rects: Array = []  # Array of {rect: Rect2, type: String} — "ground", "wall", "green"
var _scanned_circles: Array = []  # Array of {center: Vector2, radius: float} for cylinders
var _floor_center: Vector2 = Vector2.ZERO
var _floor_size: Vector2 = Vector2(100.0, 100.0)

var _player_pos: Vector3 = Vector3.ZERO
var _player_rot_y: float = 0.0
var _world_players: Dictionary = {}
var _local_peer_id: int = 0
var _player_names: Dictionary = {}
var _npc_positions: Array = []
var _enemy_positions: Array = []
var _waypoint_path: PackedVector3Array = PackedVector3Array()
var _waypoint_target: Vector3 = Vector3.ZERO
var _has_waypoint: bool = false

var _map_scale: float = 1.0
var _map_offset: Vector2 = Vector2.ZERO


func _process(_delta: float) -> void:
	if visible:
		queue_redraw()


func toggle() -> void:
	visible = not visible
	if visible:
		_detect_floor()
		_recompute_scale()


func scan_environment(env: Node3D) -> void:
	## Scan the environment using the shared MapData scanner.
	_scanned_rects.clear()
	_scanned_circles.clear()
	if not env or not is_instance_valid(env):
		return

	var result: Dictionary = MapData.scan_scene(env, _player_pos.y)
	_scanned_rects = result["rects"]
	_scanned_circles = result["circles"]

	# Compute bounds from scanned geometry
	var min_x := INF
	var max_x := -INF
	var min_z := INF
	var max_z := -INF
	for entry in _scanned_rects:
		var rect: Rect2 = entry["rect"]
		min_x = minf(min_x, rect.position.x)
		max_x = maxf(max_x, rect.end.x)
		min_z = minf(min_z, rect.position.y)
		max_z = maxf(max_z, rect.end.y)
	for circ in _scanned_circles:
		min_x = minf(min_x, circ["center"].x - circ["radius"])
		max_x = maxf(max_x, circ["center"].x + circ["radius"])
		min_z = minf(min_z, circ["center"].y - circ["radius"])
		max_z = maxf(max_z, circ["center"].y + circ["radius"])

	if min_x < INF:
		_floor_center = Vector2((min_x + max_x) / 2.0, (min_z + max_z) / 2.0)
		_floor_size = Vector2(max_x - min_x, max_z - min_z)
	else:
		var geo := MapData.get_geometry_for_floor(_current_floor_id)
		_floor_center = geo.get("center", Vector2.ZERO)
		_floor_size = geo.get("size", Vector2(100.0, 100.0))


func _detect_floor() -> void:
	if _current_floor_id == "arena":
		return
	var floor_def := MapData.get_floor_for_position(_player_pos)
	var floor_id: String = floor_def.get("id", _current_floor_id)
	var floor_name: String = floor_def.get("name", _floor_name)
	if floor_id.is_empty():
		floor_id = _current_floor_id
	_current_floor_id = floor_id
	_floor_name = floor_name

	# Waypoint: only in hub floors, not arena
	if floor_id != "arena" and floor_def.has("target"):
		_waypoint_target = floor_def["target"]
		_has_waypoint = true
	else:
		_has_waypoint = false


func set_floor(floor_id: String, floor_name: String) -> void:
	_current_floor_id = floor_id
	_floor_name = floor_name
	if floor_id != "arena":
		var floor_def := MapData.get_floor_for_position(_player_pos)
		if floor_def.has("target"):
			_waypoint_target = floor_def["target"]
			_has_waypoint = true
		else:
			_has_waypoint = false
	else:
		_has_waypoint = false


func reset_floor() -> void:
	_current_floor_id = ""


func set_local_info(peer_id: int, names: Dictionary) -> void:
	_local_peer_id = peer_id
	_player_names = names


func update_state(data: Dictionary) -> void:
	_player_pos = data.get("player_pos", Vector3.ZERO)
	_player_rot_y = data.get("player_rot_y", 0.0)
	_world_players = data.get("players", {})
	_npc_positions = data.get("npcs", [])
	_enemy_positions = data.get("enemies", [])


func set_waypoint_path(path: PackedVector3Array) -> void:
	_waypoint_path = path


# =============================================================================
# Scale computation
# =============================================================================


func _recompute_scale() -> void:
	var vp := size
	var usable := vp * (1.0 - 2.0 * PADDING)
	if _floor_size.x <= 0 or _floor_size.y <= 0:
		_map_scale = 1.0
		_map_offset = vp / 2.0
		return
	var scale_x := usable.x / _floor_size.x
	var scale_y := usable.y / _floor_size.y
	_map_scale = minf(scale_x, scale_y)
	_map_offset = vp / 2.0


# =============================================================================
# Drawing
# =============================================================================


func _draw() -> void:
	if not visible:
		return

	var font := ThemeDB.fallback_font

	# Full-screen background (void)
	draw_rect(Rect2(Vector2.ZERO, size), BG_COLOR)

	# Draw in layers: floors -> gardens -> paths -> green details -> walls
	# Pass 1: large floor surfaces (plaza ground, lobby floor)
	for entry in _scanned_rects:
		if entry["type"] == "floor":
			draw_rect(_world_rect_to_screen(entry["rect"]), FLOOR_COLOR)

	# Pass 2: garden areas (large grass)
	for entry in _scanned_rects:
		if entry["type"] == "garden":
			draw_rect(_world_rect_to_screen(entry["rect"]), GARDEN_COLOR)

	# Pass 3: paths and sidewalks
	for entry in _scanned_rects:
		if entry["type"] == "ground":
			draw_rect(_world_rect_to_screen(entry["rect"]), GROUND_COLOR)

	# Pass 4: small greenery (hedges, planters)
	for entry in _scanned_rects:
		if entry["type"] == "green":
			draw_rect(_world_rect_to_screen(entry["rect"]), GREEN_COLOR)
	for circ in _scanned_circles:
		var sp := _world_to_screen(Vector3(circ["center"].x, 0.0, circ["center"].y))
		var screen_radius: float = circ["radius"] * _map_scale
		if screen_radius < 1.5:
			screen_radius = 1.5
		var is_green: bool = circ.get("green", false)
		if is_green:
			draw_circle(sp, screen_radius, GREEN_COLOR)

	# Pass 3: walls/buildings (on top)
	for entry in _scanned_rects:
		if entry["type"] == "wall":
			var sr := _world_rect_to_screen(entry["rect"])
			draw_rect(sr, WALL_COLOR)
			draw_rect(sr, BORDER_COLOR, false, 1.0)

	# Non-green cylinders
	for circ in _scanned_circles:
		if not circ.get("green", false):
			var sp := _world_to_screen(Vector3(circ["center"].x, 0.0, circ["center"].y))
			var screen_radius: float = circ["radius"] * _map_scale
			if screen_radius < 1.5:
				screen_radius = 1.5
			draw_circle(sp, screen_radius, WALL_COLOR)

	# Waypoint path
	if _waypoint_path.size() >= 2:
		for i in range(1, _waypoint_path.size()):
			var a := _world_to_screen(_waypoint_path[i - 1])
			var b := _world_to_screen(_waypoint_path[i])
			draw_line(a, b, WAYPOINT_COLOR, 2.0)

	# Waypoint target diamond
	if _has_waypoint:
		var wp_screen := _world_to_screen(_waypoint_target)
		_draw_diamond(wp_screen, 8.0, WAYPOINT_COLOR)

	# NPCs
	for npc_pos in _npc_positions:
		var sp := _world_to_screen(npc_pos)
		if _is_on_map(sp):
			draw_circle(sp, 4.0, NPC_COLOR)

	# Enemies
	for epos in _enemy_positions:
		var sp := _world_to_screen(epos)
		if _is_on_map(sp):
			draw_circle(sp, 5.0, ENEMY_COLOR)

	# Other players
	for pid in _world_players:
		if pid == _local_peer_id:
			continue
		var pdata: Dictionary = _world_players[pid]
		var sp := _world_to_screen(pdata.get("pos", Vector3.ZERO))
		if _is_on_map(sp):
			draw_circle(sp, 4.0, PLAYER_COLOR)
			var uname: String = _player_names.get(pid, "")
			if uname != "":
				if uname.length() > 12:
					uname = uname.substr(0, 12)
				draw_string(
					font,
					sp + Vector2(-20.0, -8.0),
					uname,
					HORIZONTAL_ALIGNMENT_CENTER,
					40,
					10,
					PLAYER_COLOR
				)

	# Self arrow
	var self_screen := _world_to_screen(_player_pos)
	_draw_arrow(self_screen, _player_rot_y, SELF_COLOR, 10.0)

	# Floor name at top center
	draw_string(
		font,
		Vector2(size.x / 2.0 - 60.0, 30.0),
		_floor_name,
		HORIZONTAL_ALIGNMENT_CENTER,
		120,
		16,
		Color(0.8, 0.8, 0.85, 0.9)
	)

	# Close hint at bottom
	draw_string(
		font,
		Vector2(size.x / 2.0 - 30.0, size.y - 20.0),
		"[M] Close",
		HORIZONTAL_ALIGNMENT_CENTER,
		60,
		12,
		Color(0.5, 0.5, 0.5, 0.6)
	)

	# Legend at bottom-left
	var legend_y := size.y - 80.0
	var legend_x := 20.0
	draw_circle(Vector2(legend_x, legend_y), 4.0, SELF_COLOR)
	draw_string(
		font,
		Vector2(legend_x + 10.0, legend_y + 4.0),
		"You",
		HORIZONTAL_ALIGNMENT_LEFT,
		40,
		10,
		Color(0.6, 0.6, 0.6, 0.7)
	)
	draw_circle(Vector2(legend_x, legend_y + 16.0), 4.0, PLAYER_COLOR)
	draw_string(
		font,
		Vector2(legend_x + 10.0, legend_y + 20.0),
		"Player",
		HORIZONTAL_ALIGNMENT_LEFT,
		40,
		10,
		Color(0.6, 0.6, 0.6, 0.7)
	)
	draw_circle(Vector2(legend_x, legend_y + 32.0), 4.0, NPC_COLOR)
	draw_string(
		font,
		Vector2(legend_x + 10.0, legend_y + 36.0),
		"NPC",
		HORIZONTAL_ALIGNMENT_LEFT,
		40,
		10,
		Color(0.6, 0.6, 0.6, 0.7)
	)
	_draw_diamond(Vector2(legend_x, legend_y + 48.0), 4.0, WAYPOINT_COLOR)
	draw_string(
		font,
		Vector2(legend_x + 10.0, legend_y + 52.0),
		"Objective",
		HORIZONTAL_ALIGNMENT_LEFT,
		60,
		10,
		Color(0.6, 0.6, 0.6, 0.7)
	)


# =============================================================================
# Coordinate transforms
# =============================================================================


func _world_to_screen(world_pos: Vector3) -> Vector2:
	var dx := (world_pos.x - _floor_center.x) * _map_scale
	var dy := (world_pos.z - _floor_center.y) * _map_scale
	return _map_offset + Vector2(dx, dy)


func _world_rect_to_screen(rect: Rect2) -> Rect2:
	var tl := _world_to_screen(Vector3(rect.position.x, 0.0, rect.position.y))
	var br := _world_to_screen(Vector3(rect.end.x, 0.0, rect.end.y))
	return Rect2(tl, br - tl)


func _is_on_map(screen_pos: Vector2) -> bool:
	return (
		screen_pos.x >= 0.0
		and screen_pos.x <= size.x
		and screen_pos.y >= 0.0
		and screen_pos.y <= size.y
	)


# =============================================================================
# Shape drawing
# =============================================================================


func _draw_diamond(pos: Vector2, half_size: float, color: Color) -> void:
	var pts := PackedVector2Array(
		[
			pos + Vector2(0, -half_size),
			pos + Vector2(half_size, 0),
			pos + Vector2(0, half_size),
			pos + Vector2(-half_size, 0),
		]
	)
	draw_colored_polygon(pts, color)


func _draw_arrow(pos: Vector2, rot_y: float, color: Color, arrow_size: float) -> void:
	var angle := -rot_y
	var forward := Vector2(sin(angle), -cos(angle))
	var right := Vector2(forward.y, -forward.x)

	var tip := pos + forward * arrow_size
	var left_pt := pos - forward * arrow_size * 0.6 - right * arrow_size * 0.5
	var right_pt := pos - forward * arrow_size * 0.6 + right * arrow_size * 0.5
	draw_colored_polygon(PackedVector2Array([tip, left_pt, right_pt]), color)
