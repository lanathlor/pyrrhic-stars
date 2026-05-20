extends Control

## Gunner HUD — bloom crosshair, magazine counter, pressure, enhanced rounds, hit/damage feedback.

const HIT_MARKER_DURATION: float = 0.15
const DAMAGE_FLASH_DURATION: float = 0.3
const RECOIL_DURATION: float = 0.06
const GUNNER_COLOR := Color(0.24, 0.62, 0.95)
const ENHANCED_COLOR := Color(1.0, 0.75, 0.2)
const PRESSURE_COLORS: Array[Color] = [
	Color(0.3, 0.3, 0.35, 0.4),   # 0: dim
	Color(0.3, 0.55, 0.85, 0.8),  # 1-3: blue
	Color(0.85, 0.6, 0.2, 0.9),   # 4-7: orange
	Color(0.95, 0.25, 0.2, 1.0),  # 8-9: red
	Color(1.0, 1.0, 1.0, 1.0),    # 10: white (max)
]

var _hit_marker_timer: float = 0.0
var _damage_flash_timer: float = 0.0
var _recoil_timer: float = 0.0

# Assault state (updated from controller each frame)
var _magazine: int = 30
var _mag_max: int = 30
var _stability: float = 1.0
var _steadiness: float = 1.0
var _pressure_stacks: int = 0
var _munitions: float = 0.0
var _enhanced_loaded: int = 0
var _reloading: bool = false
var _reload_progress: float = 0.0
var _max_munitions: float = 10.0

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


func update_assault_state(
	magazine: int, mag_max: int, stability: float, steadiness: float,
	pressure: int, munitions: float, enhanced_loaded: int,
	reloading: bool, reload_progress: float
) -> void:
	_magazine = magazine
	_mag_max = mag_max
	_stability = stability
	_steadiness = steadiness
	_pressure_stacks = pressure
	_munitions = munitions
	_enhanced_loaded = enhanced_loaded
	_reloading = reloading
	_reload_progress = reload_progress


func on_shoot() -> void:
	_recoil_timer = RECOIL_DURATION


## Custom crosshair drawing — called on the Crosshair child node.
func draw_crosshair(canvas: Control) -> void:
	var center := canvas.size / 2.0
	_draw_crosshair_lines(canvas, center)
	_draw_hit_marker(canvas, center)
	_draw_magazine_counter(canvas, center)
	_draw_pressure_indicator(canvas, center)
	_draw_enhanced_pips(canvas, center)
	if _reloading:
		_draw_reload_bar(canvas, center)


func _draw_crosshair_lines(canvas: Control, center: Vector2) -> void:
	# Bloom: gap scales with combined stability + steadiness
	# Stability: spread from sustained fire. Steadiness: spread from movement.
	var stability_bloom: float = 1.0 - _stability
	var steadiness_bloom: float = (1.0 - _steadiness) * 0.5  # steadiness adds up to half the max bloom
	var bloom: float = clampf(stability_bloom + steadiness_bloom, 0.0, 1.0)
	var gap: float = lerpf(6.0, 28.0, bloom)
	var length: float = 12.0
	var thickness: float = 2.0
	var color := Color.WHITE

	# Momentary recoil kick on top of bloom
	if _recoil_timer > 0.0:
		gap += 4.0

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


func _draw_hit_marker(canvas: Control, center: Vector2) -> void:
	if _hit_marker_timer <= 0.0:
		return
	var t: float = _hit_marker_timer / HIT_MARKER_DURATION
	var hit_color := Color(1.0, 0.2, 0.2, t)
	var x_gap: float = 5.0
	var x_len: float = 10.0
	var x_thick: float = 2.5
	canvas.draw_line(
		center + Vector2(-x_gap - x_len, -x_gap - x_len),
		center + Vector2(-x_gap, -x_gap),
		hit_color, x_thick, true
	)
	canvas.draw_line(
		center + Vector2(x_gap + x_len, -x_gap - x_len),
		center + Vector2(x_gap, -x_gap),
		hit_color, x_thick, true
	)
	canvas.draw_line(
		center + Vector2(-x_gap - x_len, x_gap + x_len),
		center + Vector2(-x_gap, x_gap),
		hit_color, x_thick, true
	)
	canvas.draw_line(
		center + Vector2(x_gap + x_len, x_gap + x_len),
		center + Vector2(x_gap, x_gap),
		hit_color, x_thick, true
	)


## Magazine counter: bottom-right of crosshair area.
func _draw_magazine_counter(canvas: Control, center: Vector2) -> void:
	var pos := center + Vector2(45.0, 20.0)
	# Color based on magazine %
	var pct: float = float(_magazine) / float(maxi(_mag_max, 1))
	var color := Color.WHITE
	if pct < 0.1:
		color = Color(1.0, 0.2, 0.2)
	elif pct < 0.3:
		color = Color(1.0, 0.85, 0.3)

	if _reloading:
		color = Color(0.6, 0.6, 0.65, 0.7)

	# Current ammo — large
	var font := ThemeDB.fallback_font
	var ammo_str := str(_magazine)
	canvas.draw_string(font, pos, ammo_str, HORIZONTAL_ALIGNMENT_RIGHT, -1, 18, color)
	# Max ammo — small, to the right
	var max_str := " / " + str(_mag_max)
	canvas.draw_string(
		font, pos + Vector2(2.0, 0.0), max_str, HORIZONTAL_ALIGNMENT_LEFT, -1, 11,
		Color(0.6, 0.6, 0.65, 0.5)
	)


## Reload progress bar below magazine counter.
func _draw_reload_bar(canvas: Control, center: Vector2) -> void:
	var bar_pos := center + Vector2(25.0, 28.0)
	var bar_w: float = 50.0
	var bar_h: float = 3.0
	# Background
	canvas.draw_rect(Rect2(bar_pos.x, bar_pos.y, bar_w, bar_h), Color(0.2, 0.2, 0.25, 0.5))
	# Fill
	var fill_w: float = bar_w * clampf(_reload_progress, 0.0, 1.0)
	canvas.draw_rect(Rect2(bar_pos.x, bar_pos.y, fill_w, bar_h), GUNNER_COLOR)


## Pressure indicator: 10 marks in arc below crosshair.
func _draw_pressure_indicator(canvas: Control, center: Vector2) -> void:
	var mark_count := 10
	var spacing := 7.0
	var total_w := float(mark_count - 1) * spacing
	var start_x := center.x - total_w / 2.0
	var y := center.y + 38.0
	var mark_w: float = 4.0
	var mark_h: float = 6.0

	for i in mark_count:
		var cx := start_x + float(i) * spacing
		var filled := i < _pressure_stacks
		var mark_color: Color
		if not filled:
			mark_color = PRESSURE_COLORS[0]
		elif _pressure_stacks >= 10:
			mark_color = PRESSURE_COLORS[4]
		elif _pressure_stacks >= 8:
			mark_color = PRESSURE_COLORS[3]
		elif _pressure_stacks >= 4:
			mark_color = PRESSURE_COLORS[2]
		else:
			mark_color = PRESSURE_COLORS[1]
		canvas.draw_rect(
			Rect2(cx - mark_w / 2.0, y - mark_h / 2.0, mark_w, mark_h), mark_color
		)


## Enhanced rounds display: reserve pips (dim blue) + loaded pips (gold).
func _draw_enhanced_pips(canvas: Control, center: Vector2) -> void:
	var reserve := int(_munitions)
	var loaded := _enhanced_loaded
	var total := reserve + loaded
	if total <= 0 and int(_max_munitions) <= 0:
		return
	var pip_count := int(_max_munitions)
	if pip_count <= 0:
		pip_count = 10
	var radius := 2.5
	var spacing := 8.0
	var total_w := float(pip_count - 1) * spacing
	var start_x := center.x - total_w / 2.0
	var y := center.y + 50.0
	var dim := Color(0.3, 0.3, 0.35, 0.3)

	for i in pip_count:
		var cx := start_x + float(i) * spacing
		if i < loaded:
			# Loaded: bright gold
			canvas.draw_circle(Vector2(cx, y), radius, ENHANCED_COLOR)
		elif i < loaded + reserve:
			# Reserve: dim blue
			canvas.draw_circle(Vector2(cx, y), radius, GUNNER_COLOR * Color(1, 1, 1, 0.6))
		else:
			# Empty
			canvas.draw_circle(Vector2(cx, y), radius, dim)
