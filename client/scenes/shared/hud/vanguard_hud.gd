extends Control

## Vanguard HUD — lock-on reticle, damage/parry feedback, hit marker.

@onready var damage_overlay: ColorRect = $DamageOverlay
@onready var lock_on_reticle: Control = $LockOnReticle

var _damage_flash_timer: float = 0.0
var _parry_flash_timer: float = 0.0
var _hit_marker_timer: float = 0.0

# Ability cooldown state
var _swirl_cd_ratio: float = 0.0
var _slam_cd_ratio: float = 0.0

const DAMAGE_FLASH_DURATION: float = 0.3
const PARRY_FLASH_DURATION: float = 0.25
const HIT_MARKER_DURATION: float = 0.15
const VANGUARD_COLOR := Color(0.9, 0.6, 0.3)
const VANGUARD_DIM := Color(0.9, 0.6, 0.3, 0.4)


func _ready() -> void:
	damage_overlay.modulate.a = 0.0


func _process(delta: float) -> void:
	if _damage_flash_timer > 0.0:
		_damage_flash_timer -= delta
		damage_overlay.modulate.a = _damage_flash_timer / DAMAGE_FLASH_DURATION * 0.4
	if _parry_flash_timer > 0.0:
		_parry_flash_timer -= delta
		# White flash for parry
		damage_overlay.color = Color(1.0, 1.0, 1.0, 1.0) if _parry_flash_timer > 0.0 else Color(0.8, 0.0, 0.0, 1.0)
		damage_overlay.modulate.a = _parry_flash_timer / PARRY_FLASH_DURATION * 0.3
		if _parry_flash_timer <= 0.0:
			damage_overlay.color = Color(0.8, 0.0, 0.0, 1.0)
	if _hit_marker_timer > 0.0:
		_hit_marker_timer -= delta
	lock_on_reticle.queue_redraw()
	queue_redraw()


func _draw() -> void:
	_draw_hit_marker()
	_draw_ability_cooldowns()


func _draw_hit_marker() -> void:
	if _hit_marker_timer <= 0.0:
		return
	var center := size / 2.0
	var t: float = _hit_marker_timer / HIT_MARKER_DURATION
	var color := Color(1.0, 0.2, 0.2, t)
	var gap: float = 5.0
	var x_len: float = 10.0
	var thick: float = 2.5
	draw_line(center + Vector2(-gap - x_len, -gap - x_len), center + Vector2(-gap, -gap), color, thick, true)
	draw_line(center + Vector2(gap + x_len, -gap - x_len), center + Vector2(gap, -gap), color, thick, true)
	draw_line(center + Vector2(-gap - x_len, gap + x_len), center + Vector2(-gap, gap), color, thick, true)
	draw_line(center + Vector2(gap + x_len, gap + x_len), center + Vector2(gap, gap), color, thick, true)


func _draw_ability_cooldowns() -> void:
	var radius: float = 22.0
	var arc_width: float = 3.0
	var point_count: int = 32

	# Blade Swirl (Q) — bottom-left area
	var q_center := Vector2(size.x / 2.0 - 40.0, size.y - 35.0)
	_draw_cooldown_arc(q_center, radius, arc_width, point_count, "Q", _swirl_cd_ratio)

	# Ground Slam (E) — bottom-right area
	var e_center := Vector2(size.x / 2.0 + 40.0, size.y - 35.0)
	_draw_cooldown_arc(e_center, radius, arc_width, point_count, "E", _slam_cd_ratio)


func _draw_cooldown_arc(center: Vector2, radius: float, arc_width: float, point_count: int, label: String, cd_ratio: float) -> void:
	var label_offset := Vector2(-4.0, 5.0)

	if cd_ratio <= 0.0:
		# Ready — full subtle ring
		draw_arc(center, radius, 0.0, TAU, point_count, Color(VANGUARD_COLOR, 0.5), arc_width, true)
		draw_string(ThemeDB.fallback_font, center + label_offset, label, HORIZONTAL_ALIGNMENT_CENTER, -1, 12, Color(VANGUARD_COLOR, 0.6))
	else:
		# On cooldown — background ring + fill arc
		var fill := 1.0 - cd_ratio
		draw_arc(center, radius, 0.0, TAU, point_count, Color(0.4, 0.4, 0.4, 0.3), arc_width, true)
		if fill > 0.01:
			var start_angle: float = -PI / 2.0
			var end_angle: float = start_angle + fill * TAU
			draw_arc(center, radius, start_angle, end_angle, point_count, Color(VANGUARD_COLOR, 0.8), arc_width, true)
		# Dimmed label
		draw_string(ThemeDB.fallback_font, center + label_offset, label, HORIZONTAL_ALIGNMENT_CENTER, -1, 12, Color(0.5, 0.5, 0.5, 0.4))


func update_ability_cooldowns(swirl_cd: float, swirl_max: float, slam_cd: float, slam_max: float) -> void:
	_swirl_cd_ratio = swirl_cd / swirl_max if swirl_max > 0.0 else 0.0
	_slam_cd_ratio = slam_cd / slam_max if slam_max > 0.0 else 0.0


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
