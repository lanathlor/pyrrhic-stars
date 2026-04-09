extends Control

## Blade Dancer HUD -- 5-config display, 4 dynamic ability slots, lock-on, damage flash.

@onready var damage_overlay: ColorRect = $DamageOverlay
@onready var lock_on_reticle: Control = $LockOnReticle
@onready var config_display: Control = $ConfigDisplay
@onready var ability_bar: Control = $AbilityBar

var _damage_flash_timer: float = 0.0
var _hit_marker_timer: float = 0.0
var _current_config: int = 0
var _gcd_ratio: float = 0.0
var _current_spells: Array = []
var _hovered_slot: int = -1  # -1 = no hover
var _shield_hp: float = 0.0
const SHIELD_MAX: float = 25.0

const DAMAGE_FLASH_DURATION: float = 0.3
const HIT_MARKER_DURATION: float = 0.15

const CONFIG_NAMES: Array[String] = ["ORBIT", "FAN", "LANCE", "SCATTER", "CROWN"]
const CONFIG_COLORS: Array[Color] = [
	Color(0.2, 0.8, 0.9, 1.0),   # Orbit -- cyan
	Color(1.0, 0.5, 0.1, 1.0),   # Fan -- orange
	Color(0.9, 0.2, 0.1, 1.0),   # Lance -- red
	Color(0.6, 0.2, 0.9, 1.0),   # Scatter -- purple
	Color(1.0, 0.85, 0.3, 1.0),  # Crown -- gold
]

const ABILITY_KEYBINDS: Array[String] = ["LMB", "R", "RMB", "E"]


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
		var arc_color := _get_current_color()
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

	# Shield bar — shown above ability bar when shield > 0
	if _shield_hp > 0.1:
		var bar_w := 120.0
		var bar_h := 8.0
		var bar_x := center.x - bar_w / 2.0
		var bar_y := size.y - 90.0  # above ability bar
		var fill := clampf(_shield_hp / SHIELD_MAX, 0.0, 1.0)
		# Background
		draw_rect(Rect2(bar_x, bar_y, bar_w, bar_h), Color(0.15, 0.15, 0.2, 0.7))
		# Fill — white/cyan
		draw_rect(Rect2(bar_x, bar_y, bar_w * fill, bar_h), Color(0.7, 0.9, 1.0, 0.85))
		# Border
		draw_rect(Rect2(bar_x, bar_y, bar_w, bar_h), Color(0.5, 0.7, 0.8, 0.6), false, 1.0)
		# Label
		var shield_text := "%.0f" % _shield_hp
		draw_string(ThemeDB.fallback_font, Vector2(bar_x + bar_w + 6.0, bar_y + 7.0), shield_text,
			HORIZONTAL_ALIGNMENT_LEFT, 40.0, 10, Color(0.7, 0.9, 1.0, 0.9))

	config_display.queue_redraw()
	ability_bar.queue_redraw()


func update_config(config_value: int) -> void:
	_current_config = clampi(config_value, 0, 4)


func update_spells(spells: Array) -> void:
	_current_spells = spells


func update_gcd(ratio: float) -> void:
	_gcd_ratio = clampf(ratio, 0.0, 1.0)


func update_shield(shield_value: float) -> void:
	_shield_hp = maxf(shield_value, 0.0)


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


# --- Config display (drawn on ConfigDisplay control) ---

func _draw_config() -> void:
	var display := config_display
	var center_x := display.size.x / 2.0
	var config_name := _get_config_name(_current_config)
	var color := _get_current_color()

	var font := ThemeDB.fallback_font

	# Config name -- large centered text
	display.draw_string(font, Vector2(center_x - 50.0, 32.0), config_name,
		HORIZONTAL_ALIGNMENT_CENTER, 100, 24, color)

	# Small colored pip for each config, current one is larger
	var pip_y := 44.0
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


# --- Ability bar -- MMO action bar style (drawn on AbilityBar control) ---

func _draw_abilities() -> void:
	var bar := ability_bar
	var slot_size := 58.0
	var slot_gap := 4.0
	var slot_count := 4
	var total_w := slot_size * slot_count + slot_gap * (slot_count - 1)
	var start_x := (bar.size.x - total_w) / 2.0
	var y := bar.size.y - slot_size - 10.0
	var font := ThemeDB.fallback_font

	var active_color := _get_current_color()
	var bg_color := Color(0.08, 0.08, 0.12, 0.85)
	var border_color := Color(0.3, 0.3, 0.35, 0.9)
	var text_color := Color(0.9, 0.9, 0.9, 0.9)
	var keybind_color := Color(0.7, 0.7, 0.7, 0.6)
	var dest_color := Color(0.7, 0.7, 0.7, 0.5)

	for i in slot_count:
		var x := start_x + i * (slot_size + slot_gap)
		var slot_rect := Rect2(x, y, slot_size, slot_size)

		# Dark background
		bar.draw_rect(slot_rect, bg_color)

		# Outer border
		bar.draw_rect(slot_rect, border_color, false, 1.5)

		# Inner glow border (current config color)
		var inner := Rect2(x + 1.5, y + 1.5, slot_size - 3.0, slot_size - 3.0)
		bar.draw_rect(inner, Color(active_color, 0.35), false, 1.5)

		# Keybind label (top-left)
		bar.draw_string(font, Vector2(x + 4.0, y + 12.0), ABILITY_KEYBINDS[i],
			HORIZONTAL_ALIGNMENT_LEFT, slot_size - 8.0, 10, keybind_color)

		if i < _current_spells.size():
			var spell: Dictionary = _current_spells[i]

			# Destination config indicator -- small colored pip (top-right)
			var dest_cfg: int = spell.get("dest", 0)
			var pip_color := _get_config_color(dest_cfg)
			bar.draw_circle(Vector2(x + slot_size - 8.0, y + 8.0), 4.0, pip_color)

			# Spell name (centered, wraps in lower portion)
			var spell_name: String = spell.get("name", "???")
			# Split into two lines if name has a space
			var parts := spell_name.split(" ", true, 1)
			if parts.size() == 2:
				bar.draw_string(font, Vector2(x + 3.0, y + slot_size - 16.0), parts[0],
					HORIZONTAL_ALIGNMENT_LEFT, slot_size - 6.0, 9, text_color)
				bar.draw_string(font, Vector2(x + 3.0, y + slot_size - 5.0), parts[1],
					HORIZONTAL_ALIGNMENT_LEFT, slot_size - 6.0, 9, text_color)
			else:
				bar.draw_string(font, Vector2(x + 3.0, y + slot_size - 6.0), spell_name,
					HORIZONTAL_ALIGNMENT_LEFT, slot_size - 6.0, 9, text_color)

			# Destination config name (small, below pip)
			var dest_name := _get_config_name(dest_cfg)
			bar.draw_string(font, Vector2(x + slot_size - 42.0, y + 22.0), dest_name,
				HORIZONTAL_ALIGNMENT_RIGHT, 38.0, 8, dest_color)

	# Tooltip — detect hover and draw tooltip above hovered slot
	_hovered_slot = -1
	if Input.get_mouse_mode() == Input.MOUSE_MODE_VISIBLE:
		var mouse_pos := bar.get_local_mouse_position()
		for i in slot_count:
			var sx := start_x + i * (slot_size + slot_gap)
			var slot_rect := Rect2(sx, y, slot_size, slot_size)
			if slot_rect.has_point(mouse_pos) and i < _current_spells.size():
				_hovered_slot = i
				break

	if _hovered_slot >= 0 and _hovered_slot < _current_spells.size():
		var spell: Dictionary = _current_spells[_hovered_slot]
		var spell_name: String = spell.get("name", "???")
		var spell_desc: String = spell.get("desc", "")
		var dest_cfg: int = spell.get("dest", 0)
		var dest_name := _get_config_name(dest_cfg)
		var dest_color_tip := _get_config_color(dest_cfg)
		var cast_time: float = spell.get("dur", 0.0)

		var tip_w := 220.0
		var tip_h := 80.0
		var slot_x := start_x + _hovered_slot * (slot_size + slot_gap)
		var tip_x := slot_x + slot_size / 2.0 - tip_w / 2.0
		var tip_y := y - tip_h - 8.0

		# Background
		bar.draw_rect(Rect2(tip_x, tip_y, tip_w, tip_h), Color(0.05, 0.05, 0.1, 0.95))
		bar.draw_rect(Rect2(tip_x, tip_y, tip_w, tip_h), Color(0.4, 0.4, 0.5, 0.8), false, 1.0)

		# Spell name (colored by current config)
		bar.draw_string(font, Vector2(tip_x + 8.0, tip_y + 16.0), spell_name,
			HORIZONTAL_ALIGNMENT_LEFT, tip_w - 16.0, 14, _get_current_color())

		# Transition arrow: CURRENT -> DEST
		var from_name := _get_config_name(_current_config)
		var transition_text := "%s -> %s" % [from_name, dest_name]
		bar.draw_string(font, Vector2(tip_x + 8.0, tip_y + 32.0), transition_text,
			HORIZONTAL_ALIGNMENT_LEFT, tip_w - 16.0, 10, dest_color_tip)

		# Cast time
		bar.draw_string(font, Vector2(tip_x + tip_w - 60.0, tip_y + 32.0), "%.1fs" % cast_time,
			HORIZONTAL_ALIGNMENT_RIGHT, 52.0, 10, Color(0.7, 0.7, 0.7, 0.8))

		# Description
		if spell_desc != "":
			bar.draw_string(font, Vector2(tip_x + 8.0, tip_y + 52.0), spell_desc,
				HORIZONTAL_ALIGNMENT_LEFT, tip_w - 16.0, 10, Color(0.8, 0.8, 0.8, 0.9))
			# Wrap second line if long
			if spell_desc.length() > 35:
				var split_pos := spell_desc.find(" ", 30)
				if split_pos > 0:
					bar.draw_string(font, Vector2(tip_x + 8.0, tip_y + 52.0), spell_desc.left(split_pos),
						HORIZONTAL_ALIGNMENT_LEFT, tip_w - 16.0, 10, Color(0.8, 0.8, 0.8, 0.9))
					bar.draw_string(font, Vector2(tip_x + 8.0, tip_y + 64.0), spell_desc.substr(split_pos + 1),
						HORIZONTAL_ALIGNMENT_LEFT, tip_w - 16.0, 10, Color(0.8, 0.8, 0.8, 0.9))
