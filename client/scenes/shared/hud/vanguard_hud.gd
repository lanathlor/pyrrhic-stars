extends Control

## Vanguard HUD — lock-on reticle, damage/parry feedback, hit marker, shared spell bar.

@onready var damage_overlay: ColorRect = $DamageOverlay
@onready var lock_on_reticle: Control = $LockOnReticle
@onready var ability_bar = $AbilityBar

var _damage_flash_timer: float = 0.0
var _parry_flash_timer: float = 0.0
var _hit_marker_timer: float = 0.0

const DAMAGE_FLASH_DURATION: float = 0.3
const PARRY_FLASH_DURATION: float = 0.25
const HIT_MARKER_DURATION: float = 0.15
const VANGUARD_COLOR := Color(0.9, 0.6, 0.3)


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
