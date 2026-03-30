extends Control

## Vanguard HUD — health bar, stamina bar, lock-on reticle, damage/parry feedback.

@onready var health_bar: ProgressBar = $HealthBar
@onready var stamina_bar: ProgressBar = $StaminaBar
@onready var damage_overlay: ColorRect = $DamageOverlay
@onready var lock_on_reticle: Control = $LockOnReticle

var _damage_flash_timer: float = 0.0
var _parry_flash_timer: float = 0.0
var _hit_marker_timer: float = 0.0

const DAMAGE_FLASH_DURATION: float = 0.3
const PARRY_FLASH_DURATION: float = 0.25
const HIT_MARKER_DURATION: float = 0.15


func _ready() -> void:
	damage_overlay.modulate.a = 0.0
	# Style stamina bar gold
	var stamina_fill := StyleBoxFlat.new()
	stamina_fill.bg_color = Color(0.85, 0.75, 0.2)
	stamina_bar.add_theme_stylebox_override("fill", stamina_fill)
	var stamina_bg := StyleBoxFlat.new()
	stamina_bg.bg_color = Color(0.15, 0.15, 0.1)
	stamina_bar.add_theme_stylebox_override("background", stamina_bg)


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


func update_health(current: float, maximum: float) -> void:
	health_bar.max_value = maximum
	health_bar.value = current


func update_stamina(current: float, maximum: float) -> void:
	stamina_bar.max_value = maximum
	stamina_bar.value = current


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


func update_lock_on(target: Node3D, cam: Camera3D) -> void:
	lock_on_reticle.set_meta("lock_target", target)
	lock_on_reticle.set_meta("lock_camera", cam)
