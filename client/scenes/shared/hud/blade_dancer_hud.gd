extends Control

## Blade Dancer HUD — health bar, configuration state display, ability bar, lock-on, damage flash.

@onready var health_bar: ProgressBar = $HealthBar
@onready var damage_overlay: ColorRect = $DamageOverlay
@onready var lock_on_reticle: Control = $LockOnReticle
@onready var config_display: Control = $ConfigDisplay
@onready var ability_bar: Control = $AbilityBar

var _damage_flash_timer: float = 0.0
var _hit_marker_timer: float = 0.0
var _current_config: int = 0  # 0 = ORBIT, 1 = LANCE
var _gcd_ratio: float = 0.0   # 0.0 = ready, 1.0 = just used

const DAMAGE_FLASH_DURATION: float = 0.3
const HIT_MARKER_DURATION: float = 0.15

const ORBIT_COLOR := Color(0.2, 0.8, 0.9, 1.0)
const LANCE_COLOR := Color(1.0, 0.6, 0.1, 1.0)

# Ability names per config: [orbit_name, lance_name]
const ABILITY_LABELS := [
	["LMB: Slash", "LMB: Pierce"],
	["R: Launch", "R: Impale"],
	["RMB: Barrier", "RMB: Recall"],
	["C: Dash Fwd", "C: Retreat"],
]


func _ready() -> void:
	damage_overlay.modulate.a = 0.0
	config_display.draw.connect(_draw_config)
	ability_bar.draw.connect(_draw_abilities)


func _process(delta: float) -> void:
	if _damage_flash_timer > 0.0:
		_damage_flash_timer -= delta
		damage_overlay.modulate.a = _damage_flash_timer / DAMAGE_FLASH_DURATION * 0.4
	if _hit_marker_timer > 0.0:
		_hit_marker_timer -= delta
	lock_on_reticle.queue_redraw()
	config_display.queue_redraw()
	ability_bar.queue_redraw()


func update_health(current: float, maximum: float) -> void:
	health_bar.max_value = maximum
	health_bar.value = current


func update_config(config_value: int) -> void:
	_current_config = config_value


func update_gcd(ratio: float) -> void:
	_gcd_ratio = clampf(ratio, 0.0, 1.0)


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


func update_lock_on(target: Node3D, cam: Camera3D) -> void:
	lock_on_reticle.set_meta("lock_target", target)
	lock_on_reticle.set_meta("lock_camera", cam)


# --- Config display (drawn on ConfigDisplay control) ---

func _draw_config() -> void:
	var display := config_display
	var center_x := display.size.x / 2.0
	var config_name := "ORBIT" if _current_config == 0 else "LANCE"
	var color := ORBIT_COLOR if _current_config == 0 else LANCE_COLOR

	# Config name
	var font := ThemeDB.fallback_font
	display.draw_string(font, Vector2(center_x - 40.0, 28.0), config_name, HORIZONTAL_ALIGNMENT_CENTER, 80, 24, color)

	# GCD arc (small circle, fills as GCD ticks down)
	if _gcd_ratio > 0.01:
		var arc_center := Vector2(center_x, 45.0)
		var arc_radius: float = 10.0
		var bg_color := Color(0.3, 0.3, 0.3, 0.4)
		display.draw_arc(arc_center, arc_radius, 0.0, TAU, 32, bg_color, 2.0)
		var fill_angle := _gcd_ratio * TAU
		display.draw_arc(arc_center, arc_radius, -PI / 2.0, -PI / 2.0 + fill_angle, 16, color, 3.0)


# --- Ability bar (drawn on AbilityBar control) ---

func _draw_abilities() -> void:
	var bar := ability_bar
	var y := bar.size.y - 30.0
	var total_width := bar.size.x
	var slot_width := 140.0
	var gap := 10.0
	var total := slot_width * 4.0 + gap * 3.0
	var start_x := (total_width - total) / 2.0
	var font := ThemeDB.fallback_font

	var active_color := ORBIT_COLOR if _current_config == 0 else LANCE_COLOR
	var bg_color := Color(0.1, 0.1, 0.15, 0.6)
	var text_color := Color(0.9, 0.9, 0.9, 0.9)

	for i in 4:
		var x := start_x + i * (slot_width + gap)
		var rect := Rect2(x, y, slot_width, 24.0)

		# Background
		bar.draw_rect(rect, bg_color)
		# Border
		bar.draw_rect(rect, active_color, false, 1.5)

		# Label
		var label: String = ABILITY_LABELS[i][_current_config]
		bar.draw_string(font, Vector2(x + 8.0, y + 17.0), label, HORIZONTAL_ALIGNMENT_LEFT, slot_width - 16.0, 13, text_color)
