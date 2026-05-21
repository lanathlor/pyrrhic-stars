extends Control

## Arcanotechnicien HUD -- Confluence display, shared ability bar, lock-on, damage/heal flash.

const DAMAGE_FLASH_DURATION: float = 0.3
const HEAL_FLASH_DURATION: float = 0.4
const HIT_MARKER_DURATION: float = 0.15

const ARCANOTECHNICIEN_COLOR := Color(0.3, 0.65, 0.85)

const CONFLUENCE_DIM := Color(0.15, 0.25, 0.35, 0.4)
const CONFLUENCE_ACTIVE := Color(0.3, 0.65, 0.85, 0.95)
const CONFLUENCE_MAX := Color(0.5, 0.85, 1.0, 1.0)

var _damage_flash_timer: float = 0.0
var _heal_flash_timer: float = 0.0
var _hit_marker_timer: float = 0.0
var _confluence_tier: int = 0
var _confluence_stacks: int = 0
var _gcd_ratio: float = 0.0

@onready var damage_overlay: ColorRect = $DamageOverlay
@onready var lock_on_reticle: Control = $LockOnReticle
@onready var ability_bar = $AbilityBar


func _ready() -> void:
	damage_overlay.modulate.a = 0.0
	ability_bar.accent_color = ARCANOTECHNICIEN_COLOR


func _process(delta: float) -> void:
	if _damage_flash_timer > 0.0:
		_damage_flash_timer -= delta
		damage_overlay.color = Color(0.8, 0.0, 0.0, 1.0)
		damage_overlay.modulate.a = _damage_flash_timer / DAMAGE_FLASH_DURATION * 0.4
	if _heal_flash_timer > 0.0:
		_heal_flash_timer -= delta
		damage_overlay.color = Color(0.2, 0.8, 0.4, 1.0)
		damage_overlay.modulate.a = _heal_flash_timer / HEAL_FLASH_DURATION * 0.25
	if _hit_marker_timer > 0.0:
		_hit_marker_timer -= delta
	lock_on_reticle.queue_redraw()
	queue_redraw()


func _draw() -> void:
	_draw_hit_marker()
	_draw_confluence()


func _draw_hit_marker() -> void:
	if _hit_marker_timer <= 0.0:
		return
	var center := size / 2.0
	var t: float = _hit_marker_timer / HIT_MARKER_DURATION
	var color := Color(0.3, 0.85, 0.5, t)
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
		center + Vector2(gap + x_len, -gap - x_len),
		center + Vector2(gap, -gap),
		color,
		thick,
		true
	)
	draw_line(
		center + Vector2(-gap - x_len, gap + x_len),
		center + Vector2(-gap, gap),
		color,
		thick,
		true
	)
	draw_line(
		center + Vector2(gap + x_len, gap + x_len),
		center + Vector2(gap, gap),
		color,
		thick,
		true
	)


func _draw_confluence() -> void:
	var font := ThemeDB.fallback_font
	var center_x := size.x / 2.0
	var y := size.y - 160.0

	# --- Stack counter (big number) ---
	var count_text := "%d" % _confluence_stacks
	var count_color: Color
	if _confluence_tier >= 2:
		count_color = CONFLUENCE_MAX
		count_color.a = 0.7 + 0.3 * absf(sin(float(Time.get_ticks_msec()) / 300.0))
	elif _confluence_tier == 1:
		count_color = CONFLUENCE_ACTIVE
	else:
		count_color = Color(0.35, 0.5, 0.6, 0.6) if _confluence_stacks > 0 else CONFLUENCE_DIM

	draw_string(
		font,
		Vector2(center_x - 10.0, y + 6.0),
		count_text,
		HORIZONTAL_ALIGNMENT_CENTER,
		20.0,
		24,
		count_color
	)

	# --- Tier label ---
	var label := "CONFLUENCE"
	var label_color := CONFLUENCE_DIM
	if _confluence_tier == 1:
		label_color = CONFLUENCE_ACTIVE
	elif _confluence_tier >= 2:
		label_color = CONFLUENCE_MAX
		label_color.a = 0.7 + 0.3 * absf(sin(float(Time.get_ticks_msec()) / 300.0))
	elif _confluence_stacks > 0:
		label_color = Color(0.35, 0.5, 0.6, 0.6)

	draw_string(
		font,
		Vector2(center_x + 14.0, y - 6.0),
		label,
		HORIZONTAL_ALIGNMENT_LEFT,
		90.0,
		11,
		label_color
	)

	# --- Tier pips (5 dots for max 5 Confluence stacks) ---
	var pip_w := 10.0
	var pip_h := 3.0
	var pip_gap := 4.0
	var pip_count := 5
	var total_w := pip_w * pip_count + pip_gap * (pip_count - 1)
	var pip_x := center_x - total_w / 2.0
	var pip_y := y + 12.0

	for i in pip_count:
		var px := pip_x + i * (pip_w + pip_gap)
		var pip_rect := Rect2(px, pip_y, pip_w, pip_h)
		var lit: bool = i < _confluence_stacks

		if lit:
			var pip_color: Color
			if _confluence_tier >= 2:
				pip_color = CONFLUENCE_MAX
				pip_color.a = 0.7 + 0.3 * absf(sin(float(Time.get_ticks_msec()) / 300.0))
			elif _confluence_tier == 1:
				pip_color = CONFLUENCE_ACTIVE
			else:
				pip_color = Color(0.3, 0.65, 0.85, 0.8)
			draw_rect(pip_rect, pip_color)
		else:
			draw_rect(pip_rect, CONFLUENCE_DIM)


func update_confluence(tier: int, stacks: int) -> void:
	_confluence_tier = tier
	_confluence_stacks = stacks


func update_spells(spells: Array) -> void:
	ability_bar.update_spells(spells)


func update_gcd(ratio: float) -> void:
	_gcd_ratio = ratio
	ability_bar.update_gcd(ratio)


func show_damage_flash() -> void:
	_damage_flash_timer = DAMAGE_FLASH_DURATION


func show_heal_flash() -> void:
	_heal_flash_timer = HEAL_FLASH_DURATION


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
