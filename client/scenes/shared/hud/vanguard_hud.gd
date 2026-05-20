extends Control

## Vanguard HUD — lock-on reticle, damage/parry feedback, hit marker, shared spell bar.

const DAMAGE_FLASH_DURATION: float = 0.3
const PARRY_FLASH_DURATION: float = 0.25
const HIT_MARKER_DURATION: float = 0.15
const VANGUARD_COLOR := Color(0.82, 0.44, 0.24)
const ONSLAUGHT_DIM := Color(0.25, 0.18, 0.12, 0.4)
const ONSLAUGHT_EMPOWERED := Color(0.82, 0.44, 0.24, 0.95)
const ONSLAUGHT_MAXIMUM := Color(1.0, 0.35, 0.15, 1.0)

var _damage_flash_timer: float = 0.0
var _parry_flash_timer: float = 0.0
var _hit_marker_timer: float = 0.0
var _onslaught_tier: int = 0
var _onslaught_stacks: int = 0

@onready var damage_overlay: ColorRect = $DamageOverlay
@onready var lock_on_reticle: Control = $LockOnReticle
@onready var ability_bar = $AbilityBar


func _ready() -> void:
	damage_overlay.modulate.a = 0.0
	ability_bar.accent_color = VANGUARD_COLOR


func _process(delta: float) -> void:
	if _damage_flash_timer > 0.0:
		_damage_flash_timer -= delta
		damage_overlay.modulate.a = _damage_flash_timer / DAMAGE_FLASH_DURATION * 0.4
	if _parry_flash_timer > 0.0:
		_parry_flash_timer -= delta
		# White flash for parry
		damage_overlay.color = (
			Color(1.0, 1.0, 1.0, 1.0) if _parry_flash_timer > 0.0 else Color(0.8, 0.0, 0.0, 1.0)
		)
		damage_overlay.modulate.a = _parry_flash_timer / PARRY_FLASH_DURATION * 0.3
		if _parry_flash_timer <= 0.0:
			damage_overlay.color = Color(0.8, 0.0, 0.0, 1.0)
	if _hit_marker_timer > 0.0:
		_hit_marker_timer -= delta
	lock_on_reticle.queue_redraw()
	queue_redraw()


func _draw() -> void:
	_draw_hit_marker()
	_draw_onslaught()


func _draw_hit_marker() -> void:
	if _hit_marker_timer <= 0.0:
		return
	var center := size / 2.0
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


func _draw_onslaught() -> void:
	var font := ThemeDB.fallback_font
	var center_x := size.x / 2.0
	# Position well above the ability bar (crosshair area, lower third)
	var y := size.y - 160.0

	# --- Stack counter (big number) ---
	var count_text := "%d" % _onslaught_stacks
	var count_color: Color
	if _onslaught_tier >= 2:
		count_color = ONSLAUGHT_MAXIMUM
		count_color.a = 0.7 + 0.3 * absf(sin(float(Time.get_ticks_msec()) / 250.0))
	elif _onslaught_tier == 1:
		count_color = ONSLAUGHT_EMPOWERED
	else:
		count_color = Color(0.5, 0.45, 0.4, 0.6) if _onslaught_stacks > 0 else ONSLAUGHT_DIM

	draw_string(
		font,
		Vector2(center_x - 10.0, y + 6.0),
		count_text,
		HORIZONTAL_ALIGNMENT_CENTER,
		20.0,
		24,
		count_color
	)

	# --- Tier label (right of number) ---
	var label := "ONSLAUGHT"
	var label_color := ONSLAUGHT_DIM
	if _onslaught_tier == 1:
		label_color = ONSLAUGHT_EMPOWERED
	elif _onslaught_tier >= 2:
		label_color = ONSLAUGHT_MAXIMUM
		label_color.a = 0.7 + 0.3 * absf(sin(float(Time.get_ticks_msec()) / 250.0))
	elif _onslaught_stacks > 0:
		label_color = Color(0.5, 0.45, 0.4, 0.6)

	draw_string(
		font,
		Vector2(center_x + 14.0, y - 6.0),
		label,
		HORIZONTAL_ALIGNMENT_LEFT,
		90.0,
		11,
		label_color
	)

	# --- Tier pips (3 bars below the number) ---
	var pip_w := 14.0
	var pip_h := 3.0
	var pip_gap := 4.0
	var total_w := pip_w * 3.0 + pip_gap * 2.0
	var pip_x := center_x - total_w / 2.0
	var pip_y := y + 12.0

	for i in 3:
		var px := pip_x + i * (pip_w + pip_gap)
		var pip_rect := Rect2(px, pip_y, pip_w, pip_h)
		var lit: bool
		if _onslaught_tier == 0:
			lit = false
		elif _onslaught_tier == 1:
			lit = i < 2
		else:
			lit = true

		if lit:
			var pip_color: Color
			if _onslaught_tier >= 2:
				pip_color = ONSLAUGHT_MAXIMUM
				pip_color.a = 0.7 + 0.3 * absf(sin(float(Time.get_ticks_msec()) / 250.0))
			else:
				pip_color = ONSLAUGHT_EMPOWERED
			draw_rect(pip_rect, pip_color)
		else:
			draw_rect(pip_rect, ONSLAUGHT_DIM)


func update_onslaught(tier: int, stacks: int) -> void:
	_onslaught_tier = tier
	_onslaught_stacks = stacks


func update_spells(spells: Array) -> void:
	ability_bar.update_spells(spells)


func show_damage_flash() -> void:
	damage_overlay.color = Color(0.8, 0.0, 0.0, 1.0)
	_damage_flash_timer = DAMAGE_FLASH_DURATION


func show_parry_flash() -> void:
	_parry_flash_timer = PARRY_FLASH_DURATION


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
