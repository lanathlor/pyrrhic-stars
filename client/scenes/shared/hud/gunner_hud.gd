extends Control

## Gunner HUD — crosshair, hit/damage feedback, shared spell bar.

const HIT_MARKER_DURATION: float = 0.15
const DAMAGE_FLASH_DURATION: float = 0.3
const RECOIL_DURATION: float = 0.06
const GUNNER_COLOR := Color(0.24, 0.62, 0.95)

var _hit_marker_timer: float = 0.0
var _damage_flash_timer: float = 0.0
var _recoil_timer: float = 0.0
var _munitions: float = 0.0
var _max_munitions: float = 5.0

@onready var crosshair: Control = $Crosshair
@onready var damage_overlay: ColorRect = $DamageOverlay
@onready var ability_bar = $AbilityBar


func _ready() -> void:
	damage_overlay.modulate.a = 0.0
	ability_bar.accent_color = GUNNER_COLOR


func _process(delta: float) -> void:
	if _hit_marker_timer > 0.0:
		_hit_marker_timer -= delta
	if _damage_flash_timer > 0.0:
		_damage_flash_timer -= delta
		damage_overlay.modulate.a = _damage_flash_timer / DAMAGE_FLASH_DURATION * 0.4
	if _recoil_timer > 0.0:
		_recoil_timer -= delta
	crosshair.queue_redraw()


func update_spells(spells: Array) -> void:
	ability_bar.update_spells(spells)


func show_hit_marker() -> void:
	_hit_marker_timer = HIT_MARKER_DURATION


func show_damage_flash() -> void:
	_damage_flash_timer = DAMAGE_FLASH_DURATION


func update_munitions(current: float, max_val: float) -> void:
	_munitions = current
	_max_munitions = max_val


func on_shoot() -> void:
	_recoil_timer = RECOIL_DURATION


## Custom crosshair drawing — called on the Crosshair child node.
func draw_crosshair(canvas: Control) -> void:
	var center := canvas.size / 2.0
	_draw_crosshair_lines(canvas, center)
	_draw_munitions_pips(canvas, center)


func _draw_crosshair_lines(canvas: Control, center: Vector2) -> void:
	var gap: float = 6.0
	var length: float = 12.0
	var thickness: float = 2.0
	var color := Color.WHITE

	if _recoil_timer > 0.0:
		gap = 10.0

	# Horizontal lines
	canvas.draw_rect(
		Rect2(center.x - gap - length, center.y - thickness / 2.0, length, thickness), color
	)
	canvas.draw_rect(Rect2(center.x + gap, center.y - thickness / 2.0, length, thickness), color)
	# Vertical lines
	canvas.draw_rect(
		Rect2(center.x - thickness / 2.0, center.y - gap - length, thickness, length), color
	)
	canvas.draw_rect(Rect2(center.x - thickness / 2.0, center.y + gap, thickness, length), color)
	# Center dot
	canvas.draw_rect(Rect2(center.x - 1.0, center.y - 1.0, 2.0, 2.0), color)

	# X-shaped hit marker (diagonal lines) on confirmed hit
	if _hit_marker_timer > 0.0:
		var t: float = _hit_marker_timer / HIT_MARKER_DURATION
		var hit_color := Color(1.0, 0.2, 0.2, t)
		var x_gap: float = 5.0
		var x_len: float = 10.0
		var x_thick: float = 2.5
		# Top-left to center
		canvas.draw_line(
			center + Vector2(-x_gap - x_len, -x_gap - x_len),
			center + Vector2(-x_gap, -x_gap),
			hit_color,
			x_thick,
			true
		)
		# Top-right to center
		canvas.draw_line(
			center + Vector2(x_gap + x_len, -x_gap - x_len),
			center + Vector2(x_gap, -x_gap),
			hit_color,
			x_thick,
			true
		)
		# Bottom-left to center
		canvas.draw_line(
			center + Vector2(-x_gap - x_len, x_gap + x_len),
			center + Vector2(-x_gap, x_gap),
			hit_color,
			x_thick,
			true
		)
		# Bottom-right to center
		canvas.draw_line(
			center + Vector2(x_gap + x_len, x_gap + x_len),
			center + Vector2(x_gap, x_gap),
			hit_color,
			x_thick,
			true
		)


func _draw_munitions_pips(canvas: Control, center: Vector2) -> void:
	var pip_count := int(_max_munitions)
	if pip_count <= 0:
		return
	var radius := 3.0
	var spacing := 10.0
	var total_w := (pip_count - 1) * spacing
	var start_x := center.x - total_w / 2.0
	var y := center.y + 28.0
	var dim := Color(0.3, 0.3, 0.35, 0.4)

	for i in pip_count:
		var cx := start_x + float(i) * spacing
		var fill := clampf(_munitions - float(i), 0.0, 1.0)
		if fill >= 1.0:
			canvas.draw_circle(Vector2(cx, y), radius, GUNNER_COLOR)
		elif fill > 0.0:
			canvas.draw_circle(Vector2(cx, y), radius, dim)
			var partial_color := GUNNER_COLOR
			partial_color.a = fill
			canvas.draw_circle(Vector2(cx, y), radius * fill, partial_color)
		else:
			canvas.draw_circle(Vector2(cx, y), radius, dim)
