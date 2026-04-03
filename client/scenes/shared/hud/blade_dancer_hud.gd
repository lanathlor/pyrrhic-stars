extends Control

## Blade Dancer HUD — configuration state display, ability bar, lock-on, damage flash.

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

# Ability short names per config: [orbit_name, lance_name]
const ABILITY_NAMES := [
	["Slash", "Pierce"],
	["Launch", "Impale"],
	["Barrier", "Recall"],
	["Dash", "Retreat"],
]
const ABILITY_KEYBINDS := ["LMB", "R", "RMB", "C"]


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
	queue_redraw()


func _draw() -> void:
	var center := size / 2.0

	# GCD arc on crosshair
	if _gcd_ratio > 0.01:
		var radius := 22.0
		var thickness := 3.0
		var arc_color := ORBIT_COLOR if _current_config == 0 else LANCE_COLOR
		arc_color.a = 0.7
		var start_angle := -PI / 2.0
		var sweep_angle := _gcd_ratio * TAU
		var segments := 32
		for i in range(segments):
			var a1 := start_angle + sweep_angle * (float(i) / float(segments))
			var a2 := start_angle + sweep_angle * (float(i + 1) / float(segments))
			draw_line(center + Vector2(cos(a1), sin(a1)) * radius,
				center + Vector2(cos(a2), sin(a2)) * radius, arc_color, thickness, true)

	# Hit marker
	if _hit_marker_timer > 0.0:
		var t: float = _hit_marker_timer / HIT_MARKER_DURATION
		var color := Color(1.0, 0.2, 0.2, t)
		var gap: float = 5.0
		var x_len: float = 10.0
		var thick: float = 2.5
		draw_line(center + Vector2(-gap - x_len, -gap - x_len), center + Vector2(-gap, -gap), color, thick, true)
		draw_line(center + Vector2(gap + x_len, -gap - x_len), center + Vector2(gap, -gap), color, thick, true)
		draw_line(center + Vector2(-gap - x_len, gap + x_len), center + Vector2(-gap, gap), color, thick, true)
		draw_line(center + Vector2(gap + x_len, gap + x_len), center + Vector2(gap, gap), color, thick, true)

	config_display.queue_redraw()
	ability_bar.queue_redraw()


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

	var font := ThemeDB.fallback_font
	display.draw_string(font, Vector2(center_x - 40.0, 32.0), config_name,
		HORIZONTAL_ALIGNMENT_CENTER, 80, 24, color)


# --- Ability bar — MMO action bar style (drawn on AbilityBar control) ---

func _draw_abilities() -> void:
	var bar := ability_bar
	var slot_size := 52.0
	var slot_gap := 4.0
	var slot_count := 4
	var total_w := slot_size * slot_count + slot_gap * (slot_count - 1)
	var start_x := (bar.size.x - total_w) / 2.0
	var y := bar.size.y - slot_size - 10.0
	var font := ThemeDB.fallback_font

	var active_color := ORBIT_COLOR if _current_config == 0 else LANCE_COLOR
	var bg_color := Color(0.08, 0.08, 0.12, 0.85)
	var border_color := Color(0.3, 0.3, 0.35, 0.9)
	var text_color := Color(0.9, 0.9, 0.9, 0.9)
	var keybind_color := Color(0.7, 0.7, 0.7, 0.6)

	for i in slot_count:
		var x := start_x + i * (slot_size + slot_gap)
		var slot_rect := Rect2(x, y, slot_size, slot_size)

		# Dark background
		bar.draw_rect(slot_rect, bg_color)

		# Outer border (subtle gray)
		bar.draw_rect(slot_rect, border_color, false, 1.5)
		# Inner glow border (config-colored)
		var inner := Rect2(x + 1.5, y + 1.5, slot_size - 3.0, slot_size - 3.0)
		bar.draw_rect(inner, Color(active_color, 0.35), false, 1.5)

		# Keybind label (top-left corner)
		bar.draw_string(font, Vector2(x + 4.0, y + 12.0), ABILITY_KEYBINDS[i],
			HORIZONTAL_ALIGNMENT_LEFT, slot_size - 8.0, 10, keybind_color)

		# Ability name (centered in lower portion)
		var ability_name: String = ABILITY_NAMES[i][_current_config]
		bar.draw_string(font, Vector2(x + 4.0, y + slot_size - 6.0), ability_name,
			HORIZONTAL_ALIGNMENT_LEFT, slot_size - 8.0, 11, text_color)

