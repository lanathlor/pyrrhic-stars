extends Control

## Vanguard HUD — lock-on reticle, damage/parry feedback, hit marker, shared ability bar.

const DAMAGE_FLASH_DURATION: float = 0.3
const PARRY_FLASH_DURATION: float = 0.25
const HIT_MARKER_DURATION: float = 0.15
const VANGUARD_COLOR := Color(0.82, 0.44, 0.24)
const ONSLAUGHT_DIM := Color(0.25, 0.18, 0.12, 0.4)
const ONSLAUGHT_EMPOWERED := Color(0.82, 0.44, 0.24, 0.95)
const ONSLAUGHT_MAXIMUM := Color(1.0, 0.35, 0.15, 1.0)

const DEVOTION_DIM := Color(0.15, 0.22, 0.30, 0.4)
const DEVOTION_EMPOWERED := Color(0.3, 0.6, 0.9, 0.95)
const DEVOTION_MAXIMUM := Color(0.4, 0.8, 1.0, 1.0)

var _damage_flash_timer: float = 0.0
var _parry_flash_timer: float = 0.0
var _hit_marker_timer: float = 0.0
var _onslaught_tier: int = 0
var _onslaught_stacks: int = 0
var _devotion_tier: int = 0
var _devotion_stacks: int = 0
var _showing_devotion: bool = false

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
	if _showing_devotion:
		_draw_devotion()
	else:
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
	var y := size.y - 160.0
	var palette := {
		dim = ONSLAUGHT_DIM,
		neutral = Color(0.5, 0.45, 0.4, 0.6),
		empowered = ONSLAUGHT_EMPOWERED,
		maximum = ONSLAUGHT_MAXIMUM,
		speed = 250.0,
	}
	_draw_resource_counter(_onslaught_stacks, _onslaught_tier, "ONSLAUGHT", y, palette)


func update_onslaught(tier: int, stacks: int) -> void:
	_showing_devotion = false
	_onslaught_tier = tier
	_onslaught_stacks = stacks


func update_devotion(tier: int, stacks: int) -> void:
	_showing_devotion = true
	_devotion_tier = tier
	_devotion_stacks = stacks


func _draw_devotion() -> void:
	var y := size.y - 160.0
	var palette := {
		dim = DEVOTION_DIM,
		neutral = Color(0.3, 0.45, 0.55, 0.6),
		empowered = DEVOTION_EMPOWERED,
		maximum = DEVOTION_MAXIMUM,
		speed = 300.0,
	}
	_draw_resource_counter(_devotion_stacks, _devotion_tier, "DEVOTION", y, palette)


func _draw_resource_counter(
	stacks: int, tier: int, label: String, y: float, palette: Dictionary
) -> void:
	var font := ThemeDB.fallback_font
	var center_x := size.x / 2.0
	var active_color := _get_tier_color(tier, stacks, palette)

	draw_string(
		font,
		Vector2(center_x - 10.0, y + 6.0),
		"%d" % stacks,
		HORIZONTAL_ALIGNMENT_CENTER,
		20.0,
		24,
		active_color
	)
	draw_string(
		font,
		Vector2(center_x + 14.0, y - 6.0),
		label,
		HORIZONTAL_ALIGNMENT_LEFT,
		90.0,
		11,
		active_color
	)

	_draw_tier_pips(center_x, y + 12.0, tier, palette)


func _get_tier_color(tier: int, stacks: int, palette: Dictionary) -> Color:
	if tier >= 2:
		var c: Color = palette.maximum
		c.a = 0.7 + 0.3 * absf(sin(float(Time.get_ticks_msec()) / palette.speed))
		return c
	if tier == 1:
		return palette.empowered
	if stacks > 0:
		return palette.neutral
	return palette.dim


func _draw_tier_pips(center_x: float, pip_y: float, tier: int, palette: Dictionary) -> void:
	var pip_w := 14.0
	var pip_h := 3.0
	var pip_gap := 4.0
	var total_w := pip_w * 3.0 + pip_gap * 2.0
	var pip_x := center_x - total_w / 2.0

	for i in 3:
		var px := pip_x + i * (pip_w + pip_gap)
		var pip_rect := Rect2(px, pip_y, pip_w, pip_h)
		var lit := tier >= 2 or (tier == 1 and i < 2)

		if lit:
			draw_rect(pip_rect, _get_tier_color(tier, 1, palette))
		else:
			draw_rect(pip_rect, palette.dim)


func update_abilities(abilities: Array) -> void:
	ability_bar.update_abilities(abilities)


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
