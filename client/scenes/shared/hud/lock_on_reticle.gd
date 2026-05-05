extends Control

## Lock-on reticle — tracks enemy position on screen.
## Shows a diamond marker on the target + a small HUD indicator for lock-on state.

var _lock_active: bool = false
var _pulse_timer: float = 0.0


func _process(delta: float) -> void:
	_pulse_timer += delta


func _draw() -> void:
	var target: Node3D = null
	var cam: Camera3D = null
	if has_meta("lock_target"):
		var t = get_meta("lock_target")
		if is_instance_valid(t) and t is Node3D:
			target = t
		else:
			remove_meta("lock_target")
	if has_meta("lock_camera"):
		var c = get_meta("lock_camera")
		if is_instance_valid(c) and c is Camera3D:
			cam = c
		else:
			remove_meta("lock_camera")

	# Lock-on status indicator (top-center)
	if _lock_active and target:
		_draw_lock_indicator()
	elif not _lock_active:
		var hint_color := Color(0.6, 0.6, 0.7, 0.4)
		draw_string(
			ThemeDB.fallback_font,
			Vector2(size.x / 2.0 - 30.0, 33.0),
			"[Q] Lock On",
			HORIZONTAL_ALIGNMENT_LEFT,
			-1,
			13,
			hint_color
		)

	if not target or not cam:
		return

	var world_pos := target.global_position + Vector3(0.0, 2.2, 0.0)
	if cam.is_position_behind(world_pos):
		# Target behind camera — draw edge arrow
		_draw_offscreen_arrow()
		return

	var screen_pos := cam.unproject_position(world_pos)
	_draw_target_reticle(screen_pos)


func _draw_target_reticle(pos: Vector2) -> void:
	var pulse := 0.9 + 0.1 * sin(_pulse_timer * 4.0)
	var radius: float = 24.0 * pulse
	var thickness: float = 2.5
	var color := Color(1.0, 0.75, 0.1, 0.9)

	# Outer diamond
	var points := PackedVector2Array(
		[
			pos + Vector2(0.0, -radius),
			pos + Vector2(radius, 0.0),
			pos + Vector2(0.0, radius),
			pos + Vector2(-radius, 0.0),
			pos + Vector2(0.0, -radius),
		]
	)
	for i in range(points.size() - 1):
		draw_line(points[i], points[i + 1], color, thickness, true)

	# Center dot
	draw_circle(pos, 4.0, color)

	# Small corner ticks for visibility
	var tick_len: float = 6.0
	var offset: float = radius + 4.0
	# Top
	draw_line(
		pos + Vector2(-tick_len / 2, -offset), pos + Vector2(tick_len / 2, -offset), color, 1.5
	)
	# Bottom
	draw_line(pos + Vector2(-tick_len / 2, offset), pos + Vector2(tick_len / 2, offset), color, 1.5)
	# Left
	draw_line(
		pos + Vector2(-offset, -tick_len / 2), pos + Vector2(-offset, tick_len / 2), color, 1.5
	)
	# Right
	draw_line(pos + Vector2(offset, -tick_len / 2), pos + Vector2(offset, tick_len / 2), color, 1.5)


func _draw_lock_indicator() -> void:
	var center_x := size.x / 2.0
	var color := Color(1.0, 0.75, 0.1, 0.7)

	draw_circle(Vector2(center_x, 28.0), 5.0, color)
	draw_circle(Vector2(center_x, 28.0), 3.0, Color(0.0, 0.0, 0.0, 0.5))
	draw_string(
		ThemeDB.fallback_font,
		Vector2(center_x + 10.0, 33.0),
		"LOCKED [Q]",
		HORIZONTAL_ALIGNMENT_LEFT,
		-1,
		14,
		color
	)


func _draw_offscreen_arrow() -> void:
	# Arrow at bottom of screen indicating target is behind
	var center_x := size.x / 2.0
	var y := size.y - 60.0
	var color := Color(1.0, 0.75, 0.1, 0.6)
	var arrow_size: float = 12.0

	draw_line(Vector2(center_x, y + arrow_size), Vector2(center_x - arrow_size, y), color, 2.0)
	draw_line(Vector2(center_x, y + arrow_size), Vector2(center_x + arrow_size, y), color, 2.0)
	draw_string(
		ThemeDB.fallback_font,
		Vector2(center_x - 25.0, y - 5.0),
		"BEHIND",
		HORIZONTAL_ALIGNMENT_CENTER,
		50,
		12,
		color
	)
