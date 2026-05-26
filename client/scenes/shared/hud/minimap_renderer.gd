class_name MinimapRenderer
extends RefCounted

## Minimap drawing logic extracted from shared_hud.gd.
## Holds minimap-specific state (circle poly, floor geometry) and provides
## draw methods that operate on a Control canvas.

const MapData := preload("res://scenes/shared/hud/map_data.gd")

# --- Constants ---
const MINIMAP_RADIUS := 80.0
const MINIMAP_WORLD_RADIUS := 25.0
const MINIMAP_CIRCLE_POINTS := 48

# --- Colors (duplicated from SharedHUD for self-contained rendering) ---
const HUD_BG := Color(0.02, 0.025, 0.035, 0.82)
const HUD_BORDER := Color(0.28, 0.3, 0.36, 0.85)
const HEALTH_GOOD := Color(0.28, 0.78, 0.4, 1.0)
const HEALTH_BAD := Color(0.82, 0.24, 0.24, 1.0)
const POWER_COLOR := Color(0.82, 0.68, 0.24, 1.0)

# --- Persistent state ---
var circle_poly: PackedVector2Array = PackedVector2Array()
var floor_rects: Array = []  # Array of {rect: Rect2, type: String}
var floor_circles: Array = []  # Array of {center: Vector2, radius: float, green: bool}
var current_floor_id: String = ""
var waypoint_target: Vector3 = Vector3.ZERO
var has_waypoint: bool = false
var floor_check_timer: float = 0.0
var environment: Node3D = null

# --- Per-frame draw inputs (set by caller before draw()) ---
var local_peer_id: int = 0
var world_players: Dictionary = {}
var local_player: CharacterBody3D = null
var enemy_positions: Array = []
var npc_positions: Array = []
var player_rot_y: float = 0.0

# =============================================================================
# Public API
# =============================================================================


func on_enter_hub() -> void:
	current_floor_id = ""
	floor_check_timer = 10.0  # force immediate floor check


func on_enter_arena() -> void:
	current_floor_id = "arena"
	has_waypoint = false
	update_floor_geometry(null, 0.0)


func process_floor_detection(delta: float, hub_mode: bool, player: Node) -> void:
	if not hub_mode:
		return
	if not player or not is_instance_valid(player):
		return
	floor_check_timer += delta
	if floor_check_timer >= 0.5:
		floor_check_timer = 0.0
		_detect_floor(player.global_position)


## Draw the full minimap onto the given Control canvas.
## Caller must set local_peer_id, world_players, local_player,
## enemy_positions, npc_positions, player_rot_y before calling.
func draw(ctrl: Control) -> void:
	var map_center := Vector2(ctrl.size.x - MINIMAP_RADIUS - 16.0, MINIMAP_RADIUS + 16.0)
	var scale_factor := MINIMAP_RADIUS / MINIMAP_WORLD_RADIUS

	if circle_poly.is_empty():
		circle_poly = PackedVector2Array()
		for i in MINIMAP_CIRCLE_POINTS:
			var angle := TAU * i / float(MINIMAP_CIRCLE_POINTS)
			circle_poly.append(Vector2(cos(angle), sin(angle)) * MINIMAP_RADIUS)

	var circle_at_center := PackedVector2Array()
	for pt in circle_poly:
		circle_at_center.append(pt + map_center)
	ctrl.draw_colored_polygon(circle_at_center, HUD_BG)

	var local_pos := _get_local_position()

	var minimap_view := {
		center_pos = local_pos,
		map_center = map_center,
		scale = scale_factor,
		clip = circle_at_center,
	}
	_draw_floor_layers(ctrl, minimap_view)
	ctrl.draw_arc(map_center, MINIMAP_RADIUS + 2.0, 0.0, TAU, 64, HUD_BORDER, 2.0)
	_draw_minimap_entities(ctrl, local_pos, map_center, scale_factor)
	_draw_arrow(ctrl, map_center, player_rot_y, Color(1.0, 1.0, 1.0, 0.9), 6.0)

	if has_waypoint:
		var wp_map := _world_to_minimap(waypoint_target, local_pos, map_center, scale_factor)
		_draw_waypoint_indicator(ctrl, map_center, wp_map)

	var font := ThemeDB.fallback_font
	ctrl.draw_string(
		font,
		map_center + Vector2(-3.0, -MINIMAP_RADIUS - 4.0),
		"N",
		HORIZONTAL_ALIGNMENT_CENTER,
		10,
		9,
		Color(0.5, 0.5, 0.5, 0.5)
	)


func _get_local_position() -> Vector3:
	if world_players.has(local_peer_id):
		return world_players[local_peer_id]["pos"]
	if local_player and is_instance_valid(local_player):
		return local_player.global_position
	return Vector3.ZERO


func _draw_floor_layers(ctrl: Control, minimap_view: Dictionary) -> void:
	var type_colors := {
		"floor": Color(0.30, 0.30, 0.34, 1.0),
		"garden": Color(0.12, 0.28, 0.12, 1.0),
		"ground": Color(0.38, 0.38, 0.42, 1.0),
		"green": Color(0.15, 0.38, 0.15, 1.0),
		"wall": Color(0.18, 0.18, 0.22, 1.0),
	}
	for layer in ["floor", "garden", "ground", "green", "wall"]:
		var color: Color = type_colors[layer]
		for entry in floor_rects:
			if entry["type"] == layer:
				_draw_rect_clipped(ctrl, entry["rect"], minimap_view, color)


func _draw_minimap_entities(
	ctrl: Control, local_pos: Vector3, map_center: Vector2, scale_factor: float
) -> void:
	for npc_pos in npc_positions:
		var npc_map := _world_to_minimap(npc_pos, local_pos, map_center, scale_factor)
		if npc_map.distance_to(map_center) <= MINIMAP_RADIUS:
			ctrl.draw_circle(npc_map, 2.5, POWER_COLOR)

	for pid in world_players:
		if pid == local_peer_id:
			continue
		var world_pos: Vector3 = world_players[pid]["pos"]
		var map_pos := _world_to_minimap(world_pos, local_pos, map_center, scale_factor)
		if map_pos.distance_to(map_center) <= MINIMAP_RADIUS:
			ctrl.draw_circle(map_pos, 3.0, HEALTH_GOOD)

	for epos in enemy_positions:
		var enemy_map := _world_to_minimap(epos, local_pos, map_center, scale_factor)
		if enemy_map.distance_to(map_center) <= MINIMAP_RADIUS:
			ctrl.draw_circle(enemy_map, 4.0, HEALTH_BAD)


# =============================================================================
# Floor detection and geometry caching
# =============================================================================


func _detect_floor(player_pos: Vector3) -> void:
	var floor_def := MapData.get_floor_for_position(player_pos)
	var new_id: String = floor_def.get("id", "")
	if new_id != current_floor_id:
		current_floor_id = new_id
		if floor_def.has("target"):
			waypoint_target = floor_def["target"]
			has_waypoint = true
		else:
			has_waypoint = false
	# Always rescan — geometry may have been invalidated
	update_floor_geometry(environment if is_instance_valid(environment) else null, player_pos.y)


func update_floor_geometry(env: Node3D, player_y: float) -> void:
	floor_rects.clear()
	floor_circles.clear()
	if env and is_instance_valid(env):
		var result: Dictionary = MapData.scan_scene(env, player_y)
		floor_rects = result["rects"]
		floor_circles = result["circles"]
	else:
		# Fallback to MapData if no environment available
		var geo := MapData.get_geometry_for_floor(current_floor_id)
		if geo.is_empty():
			return
		var buildings: Array = geo.get("buildings", [])
		for b in buildings:
			var c: Vector2 = b["center"]
			var s: Vector2 = b["size"]
			floor_rects.append(
				{"rect": Rect2(c.x - s.x / 2.0, c.y - s.y / 2.0, s.x, s.y), "type": "wall"}
			)


# =============================================================================
# Minimap drawing primitives
# =============================================================================


func _world_to_minimap(
	world_pos: Vector3, center_pos: Vector3, map_center: Vector2, scale: float
) -> Vector2:
	var dx := (world_pos.x - center_pos.x) * scale
	var dz := (world_pos.z - center_pos.z) * scale
	return map_center + Vector2(dx, dz)


func _draw_arrow(
	ctrl: Control, pos: Vector2, rot_y: float, color: Color, arrow_size: float
) -> void:
	# Arrow pointing in the direction the player faces
	# In Godot: rotation_y = 0 means facing -Z, which is up on minimap
	var angle := -rot_y
	var forward := Vector2(sin(angle), -cos(angle))
	var right := Vector2(forward.y, -forward.x)

	var tip := pos + forward * arrow_size
	var left_pt := pos - forward * arrow_size * 0.6 - right * arrow_size * 0.5
	var right_pt := pos - forward * arrow_size * 0.6 + right * arrow_size * 0.5

	ctrl.draw_colored_polygon(PackedVector2Array([tip, left_pt, right_pt]), color)


func _draw_rect_clipped(ctrl: Control, rect: Rect2, view: Dictionary, color: Color) -> void:
	## Draw a world-space Rect2 (XZ plane) on the minimap, clipped to the circle.
	var center_pos: Vector3 = view.center_pos
	var map_center: Vector2 = view.map_center
	var scale: float = view.scale
	var clip_circle: PackedVector2Array = view.clip
	var world_center := Vector3(
		rect.position.x + rect.size.x / 2.0, 0.0, rect.position.y + rect.size.y / 2.0
	)
	var half_diag := Vector2(rect.size.x, rect.size.y).length() / 2.0 * scale
	var mc := _world_to_minimap(world_center, center_pos, map_center, scale)
	# Skip if entirely outside minimap circle
	if mc.distance_to(map_center) > MINIMAP_RADIUS + half_diag:
		return

	# Transform four corners to minimap screen space
	var tl := _world_to_minimap(
		Vector3(rect.position.x, 0.0, rect.position.y), center_pos, map_center, scale
	)
	var tr := _world_to_minimap(
		Vector3(rect.end.x, 0.0, rect.position.y), center_pos, map_center, scale
	)
	var br := _world_to_minimap(Vector3(rect.end.x, 0.0, rect.end.y), center_pos, map_center, scale)
	var bl := _world_to_minimap(
		Vector3(rect.position.x, 0.0, rect.end.y), center_pos, map_center, scale
	)

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
				ctrl.draw_colored_polygon(poly, color)


func _draw_waypoint_indicator(ctrl: Control, map_center: Vector2, target_map_pos: Vector2) -> void:
	var offset := target_map_pos - map_center
	var dist := offset.length()
	var waypoint_color := Color(0.3, 0.55, 1.0, 0.9)  # flux blue
	if dist < 3.0:
		return
	if dist <= MINIMAP_RADIUS - 6.0:
		# On-map: draw diamond
		_draw_diamond(ctrl, target_map_pos, 5.0, waypoint_color)
	else:
		# Edge: clamp and draw chevron pointing outward
		var dir := offset.normalized()
		var edge_pos := map_center + dir * (MINIMAP_RADIUS - 6.0)
		_draw_chevron(ctrl, edge_pos, dir, waypoint_color)


func _draw_diamond(ctrl: Control, pos: Vector2, half_size: float, color: Color) -> void:
	var pts := PackedVector2Array(
		[
			pos + Vector2(0, -half_size),
			pos + Vector2(half_size, 0),
			pos + Vector2(0, half_size),
			pos + Vector2(-half_size, 0),
		]
	)
	ctrl.draw_colored_polygon(pts, color)


func _draw_chevron(ctrl: Control, pos: Vector2, dir: Vector2, color: Color) -> void:
	var perp := Vector2(dir.y, -dir.x)
	var tip := pos + dir * 6.0
	var left_pt := pos - perp * 4.0
	var right_pt := pos + perp * 4.0
	ctrl.draw_colored_polygon(PackedVector2Array([tip, left_pt, right_pt]), color)
