extends Control

## Blade Dancer HUD -- 5-config display, shared ability bar, lock-on, damage flash.

const SHIELD_MAX: float = 25.0
const DAMAGE_FLASH_DURATION: float = 0.3
const HIT_MARKER_DURATION: float = 0.15
const CONFIG_NAMES: Array[String] = ["ORBIT", "FAN", "LANCE", "SCATTER", "CROWN"]
const CONFIG_COLORS: Array[Color] = [
	Color(0.2, 0.8, 0.9, 1.0),  # Orbit -- cyan
	Color(1.0, 0.5, 0.1, 1.0),  # Fan -- orange
	Color(0.9, 0.2, 0.1, 1.0),  # Lance -- red
	Color(0.6, 0.2, 0.9, 1.0),  # Scatter -- purple
	Color(1.0, 0.85, 0.3, 1.0),  # Crown -- gold
]
const ABILITY_KEYBINDS: Array[String] = ["LMB", "RMB", "R", "E"]
const PANEL_BG := Color(0.02, 0.025, 0.035, 0.82)
const PANEL_FILL := Color(0.04, 0.05, 0.07, 0.45)
const PANEL_INSET := Color(0.11, 0.12, 0.15, 0.3)
const PANEL_BORDER := Color(0.28, 0.3, 0.36, 0.85)
const TEXT_MUTED := Color(0.66, 0.7, 0.77, 0.9)
const FLOW_DIM := Color(0.15, 0.2, 0.25, 0.4)
const FLOW_EMPOWERED := Color(0.2, 0.75, 0.9, 0.95)
const FLOW_MAXIMUM := Color(0.4, 0.95, 1.0, 1.0)

var _damage_flash_timer: float = 0.0
var _hit_marker_timer: float = 0.0
var _current_config: int = 0
var _gcd_ratio: float = 0.0
var _current_abilities: Array = []
var _shield_hp: float = 0.0
var _flow_tier: int = 0
var _flow_stacks: int = 0

@onready var damage_overlay: ColorRect = $DamageOverlay
@onready var lock_on_reticle: Control = $LockOnReticle
@onready var config_display: Control = $ConfigDisplay
@onready var ability_bar = $AbilityBar


func _ready() -> void:
	damage_overlay.modulate.a = 0.0
	config_display.draw.connect(_draw_config)
	ability_bar.accent_color = _get_current_color()
	ability_bar.custom_tooltip_draw = _draw_custom_tooltip


func _process(delta: float) -> void:
	if _damage_flash_timer > 0.0:
		_damage_flash_timer -= delta
		damage_overlay.modulate.a = _damage_flash_timer / DAMAGE_FLASH_DURATION * 0.4
	if _hit_marker_timer > 0.0:
		_hit_marker_timer -= delta
	lock_on_reticle.queue_redraw()
	queue_redraw()


func _draw() -> void:
	var center := size / 2.0
	_draw_gcd_arc(center)
	_draw_hit_marker(center)
	_draw_flow()
	_draw_shield_bar(center)
	config_display.queue_redraw()


func _draw_gcd_arc(center: Vector2) -> void:
	if _gcd_ratio <= 0.01:
		return
	var radius := 22.0
	var thickness := 3.0
	var arc_color := _get_current_color()
	arc_color.a = 0.7
	var start_angle := -PI / 2.0
	var sweep_angle := _gcd_ratio * TAU
	var segments := 32
	for i in range(segments):
		var a1 := start_angle + sweep_angle * (float(i) / float(segments))
		var a2 := start_angle + sweep_angle * (float(i + 1) / float(segments))
		draw_line(
			center + Vector2(cos(a1), sin(a1)) * radius,
			center + Vector2(cos(a2), sin(a2)) * radius,
			arc_color,
			thickness,
			true
		)


func _draw_hit_marker(center: Vector2) -> void:
	if _hit_marker_timer <= 0.0:
		return
	var t: float = _hit_marker_timer / HIT_MARKER_DURATION
	var color := Color(1.0, 0.2, 0.2, t)
	var gap: float = 5.0
	var x_len: float = 10.0
	var thick: float = 2.5
	draw_line(
		center + Vector2(-gap - x_len, -gap - x_len),
		center + Vector2(-gap, -gap),
		color,
		thick,
		true
	)
	draw_line(
		center + Vector2(gap + x_len, -gap - x_len), center + Vector2(gap, -gap), color, thick, true
	)
	draw_line(
		center + Vector2(-gap - x_len, gap + x_len), center + Vector2(-gap, gap), color, thick, true
	)
	draw_line(
		center + Vector2(gap + x_len, gap + x_len), center + Vector2(gap, gap), color, thick, true
	)


func _draw_shield_bar(center: Vector2) -> void:
	if _shield_hp <= 0.1:
		return
	var bar_w := 120.0
	var bar_h := 8.0
	var bar_x := center.x - bar_w / 2.0
	var bar_y := size.y - 136.0
	var fill := clampf(_shield_hp / SHIELD_MAX, 0.0, 1.0)
	_draw_status_bar(Rect2(bar_x, bar_y, bar_w, bar_h), fill, Color(0.7, 0.9, 1.0, 0.85))
	draw_string(
		ThemeDB.fallback_font,
		Vector2(bar_x + bar_w + 6.0, bar_y + 7.0),
		"%.0f" % _shield_hp,
		HORIZONTAL_ALIGNMENT_LEFT,
		40.0,
		10,
		Color(0.7, 0.9, 1.0, 0.9)
	)


func update_config(config_value: int) -> void:
	_current_config = clampi(config_value, 0, 4)
	ability_bar.accent_color = _get_current_color()


func update_abilities(abilities: Array) -> void:
	_current_abilities = abilities
	var bar_abilities: Array = []
	for i in abilities.size():
		var s: Dictionary = abilities[i].duplicate()
		s["keybind"] = ABILITY_KEYBINDS[i] if i < ABILITY_KEYBINDS.size() else "?"
		s["cooldown"] = 0.0  # BD uses GCD, not per-ability cooldowns
		s["cooldown_max"] = 0.0
		bar_abilities.append(s)
	ability_bar.update_abilities(bar_abilities)


func update_gcd(ratio: float) -> void:
	_gcd_ratio = clampf(ratio, 0.0, 1.0)
	ability_bar.update_gcd(_gcd_ratio)


func update_shield(shield_value: float) -> void:
	_shield_hp = maxf(shield_value, 0.0)


func update_flow(tier: int, stacks: int) -> void:
	_flow_tier = tier
	_flow_stacks = stacks


func show_damage_flash() -> void:
	damage_overlay.color = Color(0.8, 0.0, 0.0, 1.0)
	_damage_flash_timer = DAMAGE_FLASH_DURATION


func show_hit_marker() -> void:
	_hit_marker_timer = HIT_MARKER_DURATION


func show_lock_on() -> void:
	lock_on_reticle.visible = true
	lock_on_reticle._lock_active = true


func hide_lock_on() -> void:
	lock_on_reticle._lock_active = false
	lock_on_reticle.remove_meta("lock_target")
	lock_on_reticle.remove_meta("lock_camera")


func update_lock_on(target: Node3D, cam: Camera3D) -> void:
	lock_on_reticle.set_meta("lock_target", target)
	lock_on_reticle.set_meta("lock_camera", cam)


func _get_current_color() -> Color:
	if _current_config >= 0 and _current_config < CONFIG_COLORS.size():
		return CONFIG_COLORS[_current_config]
	return CONFIG_COLORS[0]


func _get_config_color(cfg: int) -> Color:
	if cfg >= 0 and cfg < CONFIG_COLORS.size():
		return CONFIG_COLORS[cfg]
	return CONFIG_COLORS[0]


func _get_config_name(cfg: int) -> String:
	if cfg >= 0 and cfg < CONFIG_NAMES.size():
		return CONFIG_NAMES[cfg]
	return CONFIG_NAMES[0]


# --- Flow mastery display (chain counter + tier pips) ---


func _draw_flow() -> void:
	var font := ThemeDB.fallback_font
	var center_x := size.x / 2.0
	var y := size.y - 160.0

	_draw_flow_counter(font, center_x, y)
	_draw_flow_label(font, center_x, y)
	_draw_flow_pips(center_x, y)


func _get_flow_color() -> Color:
	if _flow_tier >= 2:
		var c := FLOW_MAXIMUM
		c.a = 0.7 + 0.3 * absf(sin(float(Time.get_ticks_msec()) / 250.0))
		return c
	if _flow_tier == 1:
		return FLOW_EMPOWERED
	if _flow_stacks > 0:
		return Color(0.3, 0.45, 0.5, 0.6)
	return FLOW_DIM


func _draw_flow_counter(font: Font, center_x: float, y: float) -> void:
	draw_string(
		font,
		Vector2(center_x - 10.0, y + 6.0),
		"%d" % _flow_stacks,
		HORIZONTAL_ALIGNMENT_CENTER,
		20.0,
		24,
		_get_flow_color()
	)


func _draw_flow_label(font: Font, center_x: float, y: float) -> void:
	draw_string(
		font,
		Vector2(center_x + 14.0, y - 6.0),
		"FLOW",
		HORIZONTAL_ALIGNMENT_LEFT,
		90.0,
		11,
		_get_flow_color()
	)


func _draw_flow_pips(center_x: float, y: float) -> void:
	var pip_w := 14.0
	var pip_h := 3.0
	var pip_gap := 4.0
	var total_w := pip_w * 3.0 + pip_gap * 2.0
	var pip_x := center_x - total_w / 2.0
	var pip_y := y + 12.0

	for i in 3:
		var px := pip_x + i * (pip_w + pip_gap)
		var pip_rect := Rect2(px, pip_y, pip_w, pip_h)
		var lit := _flow_tier >= 2 or (_flow_tier == 1 and i < 2)

		if lit:
			draw_rect(pip_rect, _get_flow_color())
		else:
			draw_rect(pip_rect, FLOW_DIM)


# --- Config display (drawn on ConfigDisplay control) ---


func _draw_config() -> void:
	var display := config_display
	var center_x := display.size.x / 2.0
	var config_name := _get_config_name(_current_config)
	var color := _get_current_color()

	var font := ThemeDB.fallback_font
	display.draw_string(
		font,
		Vector2(center_x - 50.0, 24.0),
		config_name,
		HORIZONTAL_ALIGNMENT_CENTER,
		100,
		24,
		color
	)

	var pip_y := 36.0
	var pip_spacing := 16.0
	var total_w := pip_spacing * 4.0
	var pip_start_x := center_x - total_w / 2.0
	for i in 5:
		var pip_x := pip_start_x + i * pip_spacing
		var pip_color := _get_config_color(i)
		var pip_size := 5.0 if i == _current_config else 3.0
		if i != _current_config:
			pip_color.a = 0.4
		display.draw_circle(Vector2(pip_x, pip_y), pip_size, pip_color)


# --- Custom tooltip content for Blade Dancer (config transitions) ---


func _draw_custom_tooltip(bar: Control, ability: Dictionary, tip_rect: Rect2) -> void:
	var font := ThemeDB.fallback_font
	var dest_cfg: int = ability.get("dest", 0)
	var dest_name := _get_config_name(dest_cfg)
	var dest_color := _get_config_color(dest_cfg)

	# Transition arrow: CURRENT -> DEST
	var from_name := _get_config_name(_current_config)
	var transition_text := "%s -> %s" % [from_name, dest_name]
	bar.draw_string(
		font,
		Vector2(tip_rect.position.x + 8.0, tip_rect.position.y + 32.0),
		transition_text,
		HORIZONTAL_ALIGNMENT_LEFT,
		tip_rect.size.x - 16.0,
		10,
		dest_color
	)

	bar.draw_circle(
		Vector2(tip_rect.position.x + tip_rect.size.x - 14.0, tip_rect.position.y + 28.0),
		4.0,
		dest_color
	)


func _draw_panel(canvas: CanvasItem, rect: Rect2, accent: Color) -> void:
	canvas.draw_rect(rect, PANEL_FILL)
	canvas.draw_rect(rect, PANEL_BORDER, false, 1.0)
	canvas.draw_rect(
		Rect2(rect.position + Vector2(1.0, 1.0), rect.size - Vector2(2.0, 2.0)), accent, false, 1.0
	)


func _draw_status_bar(rect: Rect2, ratio: float, fill_color: Color) -> void:
	draw_rect(rect, PANEL_BG)
	if ratio > 0.0:
		draw_rect(Rect2(rect.position, Vector2(rect.size.x * ratio, rect.size.y)), fill_color)
	draw_rect(rect, PANEL_BORDER, false, 1.0)
