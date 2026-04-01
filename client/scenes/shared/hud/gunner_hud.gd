extends Control

## Gunner HUD — crosshair, health bar, hit/damage feedback, roll cooldown.

@onready var crosshair: Control = $Crosshair
@onready var health_bar: ProgressBar = $HealthBar
@onready var damage_overlay: ColorRect = $DamageOverlay

var _hit_marker_timer: float = 0.0
var _damage_flash_timer: float = 0.0
var _recoil_timer: float = 0.0
var _roll_cooldown_ratio: float = 0.0  # 0 = ready, 1 = just used

const HIT_MARKER_DURATION: float = 0.15
const DAMAGE_FLASH_DURATION: float = 0.3
const RECOIL_DURATION: float = 0.06


func _ready() -> void:
	damage_overlay.modulate.a = 0.0


func _process(delta: float) -> void:
	if _hit_marker_timer > 0.0:
		_hit_marker_timer -= delta
	if _damage_flash_timer > 0.0:
		_damage_flash_timer -= delta
		damage_overlay.modulate.a = _damage_flash_timer / DAMAGE_FLASH_DURATION * 0.4
	if _recoil_timer > 0.0:
		_recoil_timer -= delta
	crosshair.queue_redraw()


func update_health(current: float, maximum: float) -> void:
	health_bar.max_value = maximum
	health_bar.value = current


func update_roll_cooldown(remaining: float, total: float) -> void:
	if total > 0.0:
		_roll_cooldown_ratio = remaining / total
	else:
		_roll_cooldown_ratio = 0.0


func show_hit_marker() -> void:
	_hit_marker_timer = HIT_MARKER_DURATION


func show_damage_flash() -> void:
	_damage_flash_timer = DAMAGE_FLASH_DURATION


func on_shoot() -> void:
	_recoil_timer = RECOIL_DURATION


## Custom crosshair + cooldown drawing — called on the Crosshair child node.
func draw_crosshair(canvas: Control) -> void:
	var center := canvas.size / 2.0
	_draw_crosshair_lines(canvas, center)
	_draw_roll_cooldown(canvas, center)


func _draw_crosshair_lines(canvas: Control, center: Vector2) -> void:
	var gap: float = 6.0
	var length: float = 12.0
	var thickness: float = 2.0
	var color := Color.WHITE

	if _recoil_timer > 0.0:
		gap = 10.0

	# Horizontal lines
	canvas.draw_rect(Rect2(center.x - gap - length, center.y - thickness / 2.0, length, thickness), color)
	canvas.draw_rect(Rect2(center.x + gap, center.y - thickness / 2.0, length, thickness), color)
	# Vertical lines
	canvas.draw_rect(Rect2(center.x - thickness / 2.0, center.y - gap - length, thickness, length), color)
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
		canvas.draw_line(center + Vector2(-x_gap - x_len, -x_gap - x_len), center + Vector2(-x_gap, -x_gap), hit_color, x_thick, true)
		# Top-right to center
		canvas.draw_line(center + Vector2(x_gap + x_len, -x_gap - x_len), center + Vector2(x_gap, -x_gap), hit_color, x_thick, true)
		# Bottom-left to center
		canvas.draw_line(center + Vector2(-x_gap - x_len, x_gap + x_len), center + Vector2(-x_gap, x_gap), hit_color, x_thick, true)
		# Bottom-right to center
		canvas.draw_line(center + Vector2(x_gap + x_len, x_gap + x_len), center + Vector2(x_gap, x_gap), hit_color, x_thick, true)


func _draw_roll_cooldown(canvas: Control, _center: Vector2) -> void:
	var radius: float = 22.0
	var arc_width: float = 3.0
	var point_count: int = 32
	# Bottom-right corner, above health bar
	var arc_center := Vector2(360.0, canvas.size.y - 35.0)

	if _roll_cooldown_ratio <= 0.0:
		# Ready — draw full subtle circle
		canvas.draw_arc(arc_center, radius, 0.0, TAU, point_count, Color(0.3, 0.9, 1.0, 0.5), arc_width, true)
		# Small "C" label
		canvas.draw_string(ThemeDB.fallback_font, arc_center + Vector2(-4.0, 5.0), "C", HORIZONTAL_ALIGNMENT_CENTER, -1, 12, Color(0.3, 0.9, 1.0, 0.6))
	else:
		# On cooldown — draw background ring + fill arc
		var fill := 1.0 - _roll_cooldown_ratio
		canvas.draw_arc(arc_center, radius, 0.0, TAU, point_count, Color(0.4, 0.4, 0.4, 0.3), arc_width, true)
		if fill > 0.01:
			var start_angle: float = -PI / 2.0
			var end_angle: float = start_angle + fill * TAU
			canvas.draw_arc(arc_center, radius, start_angle, end_angle, point_count, Color(0.3, 0.9, 1.0, 0.8), arc_width, true)
		# Dimmed label
		canvas.draw_string(ThemeDB.fallback_font, arc_center + Vector2(-4.0, 5.0), "C", HORIZONTAL_ALIGNMENT_CENTER, -1, 12, Color(0.5, 0.5, 0.5, 0.4))
