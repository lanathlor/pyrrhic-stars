extends Control

## Shared spell bar — MMO-style action bar with dynamic slots, tooltips, cooldowns.
## Used by all class HUDs. Each class provides an Array[Dictionary] of spells.
##
## Spell dict keys (required): name, desc, keybind, cooldown, cooldown_max
## Optional keys: active, active_remaining, active_max, status_text, color, dur

var accent_color: Color = Color.WHITE
var custom_tooltip_draw := Callable()

var _spells: Array = []
var _gcd_ratio: float = 0.0
var _hovered_slot: int = -1

const SLOT_SIZE: float = 58.0
const SLOT_GAP: float = 4.0


func update_spells(spells: Array) -> void:
	_spells = spells


func update_gcd(ratio: float) -> void:
	_gcd_ratio = clampf(ratio, 0.0, 1.0)


func _process(_delta: float) -> void:
	queue_redraw()


func _draw() -> void:
	var slot_count := _spells.size()
	if slot_count == 0:
		return

	var total_w := SLOT_SIZE * slot_count + SLOT_GAP * (slot_count - 1)
	var start_x := (size.x - total_w) / 2.0
	var y := size.y - SLOT_SIZE - 10.0
	var font := ThemeDB.fallback_font

	var bg_color := Color(0.08, 0.08, 0.12, 0.85)
	var border_color := Color(0.3, 0.3, 0.35, 0.9)
	var text_color := Color(0.9, 0.9, 0.9, 0.9)
	var keybind_color := Color(0.7, 0.7, 0.7, 0.6)

	for i in slot_count:
		var x := start_x + i * (SLOT_SIZE + SLOT_GAP)
		var slot_rect := Rect2(x, y, SLOT_SIZE, SLOT_SIZE)
		var spell: Dictionary = _spells[i]
		var slot_color: Color = spell.get("color", accent_color)

		# Dark background
		draw_rect(slot_rect, bg_color)

		# Outer border
		draw_rect(slot_rect, border_color, false, 1.5)

		# Inner glow border (accent color)
		var inner := Rect2(x + 1.5, y + 1.5, SLOT_SIZE - 3.0, SLOT_SIZE - 3.0)
		var is_active: bool = spell.get("active", false)
		if is_active:
			# Bright pulsing border for active abilities
			var pulse := 0.6 + 0.4 * absf(sin(float(Time.get_ticks_msec()) / 300.0))
			draw_rect(inner, Color(slot_color, pulse), false, 2.0)
		else:
			draw_rect(inner, Color(slot_color, 0.35), false, 1.5)

		# Keybind label (top-left)
		var keybind: String = spell.get("keybind", "?")
		draw_string(font, Vector2(x + 4.0, y + 12.0), keybind,
			HORIZONTAL_ALIGNMENT_LEFT, SLOT_SIZE - 8.0, 10, keybind_color)

		# Spell name (lower portion of slot)
		var spell_name: String = spell.get("name", "???")
		var status_text: String = spell.get("status_text", "")
		if status_text != "":
			# Status text overrides spell name display (e.g. "FIRE!")
			draw_string(font, Vector2(x + 3.0, y + SLOT_SIZE / 2.0 + 5.0), status_text,
				HORIZONTAL_ALIGNMENT_CENTER, SLOT_SIZE - 6.0, 14, slot_color)
		else:
			# Split name into two lines if it has a space
			var parts := spell_name.split(" ", true, 1)
			if parts.size() == 2:
				draw_string(font, Vector2(x + 3.0, y + SLOT_SIZE - 16.0), parts[0],
					HORIZONTAL_ALIGNMENT_LEFT, SLOT_SIZE - 6.0, 9, text_color)
				draw_string(font, Vector2(x + 3.0, y + SLOT_SIZE - 5.0), parts[1],
					HORIZONTAL_ALIGNMENT_LEFT, SLOT_SIZE - 6.0, 9, text_color)
			else:
				draw_string(font, Vector2(x + 3.0, y + SLOT_SIZE - 6.0), spell_name,
					HORIZONTAL_ALIGNMENT_LEFT, SLOT_SIZE - 6.0, 9, text_color)

		# Cooldown overlay + number
		var cd: float = spell.get("cooldown", 0.0)
		var cd_max: float = spell.get("cooldown_max", 0.0)
		if cd > 0.01 and cd_max > 0.0:
			_draw_cooldown_overlay(x, y, cd, cd_max)

		# Active buff remaining indicator (small arc at top-right)
		if is_active:
			var active_rem: float = spell.get("active_remaining", 0.0)
			var active_max: float = spell.get("active_max", 0.0)
			if active_max > 0.0 and active_rem > 0.0:
				var ratio := active_rem / active_max
				var arc_center := Vector2(x + SLOT_SIZE - 10.0, y + 10.0)
				var arc_radius := 6.0
				draw_arc(arc_center, arc_radius, 0.0, TAU, 16, Color(0.2, 0.2, 0.25, 0.5), 2.0, true)
				var start_angle := -PI / 2.0
				var end_angle := start_angle + ratio * TAU
				draw_arc(arc_center, arc_radius, start_angle, end_angle, 16, Color(slot_color, 0.9), 2.0, true)

	# GCD overlay (Blade Dancer) — subtle sweep on all slots
	if _gcd_ratio > 0.01:
		for gi in slot_count:
			var gx := start_x + gi * (SLOT_SIZE + SLOT_GAP)
			var sweep_h := SLOT_SIZE * _gcd_ratio
			draw_rect(Rect2(gx, y, SLOT_SIZE, sweep_h), Color(0.0, 0.0, 0.0, 0.35))

	# Tooltip — detect hover and draw above hovered slot
	_hovered_slot = -1
	if Input.get_mouse_mode() == Input.MOUSE_MODE_VISIBLE:
		var mouse_pos := get_local_mouse_position()
		for hi in slot_count:
			var hx := start_x + hi * (SLOT_SIZE + SLOT_GAP)
			var hover_rect := Rect2(hx, y, SLOT_SIZE, SLOT_SIZE)
			if hover_rect.has_point(mouse_pos) and hi < _spells.size():
				_hovered_slot = hi
				break

	if _hovered_slot >= 0 and _hovered_slot < _spells.size():
		_draw_tooltip(start_x, y)


func _draw_cooldown_overlay(x: float, y: float, cd: float, cd_max: float) -> void:
	var font := ThemeDB.fallback_font
	var ratio := cd / cd_max

	# Dark overlay
	draw_rect(Rect2(x, y, SLOT_SIZE, SLOT_SIZE), Color(0.0, 0.0, 0.0, 0.55))

	# Clock sweep from 12 o'clock
	var center := Vector2(x + SLOT_SIZE / 2.0, y + SLOT_SIZE / 2.0)
	var sweep_radius := SLOT_SIZE / 2.0 - 2.0
	var start_angle := -PI / 2.0
	var end_angle := start_angle + (1.0 - ratio) * TAU
	if (1.0 - ratio) > 0.01:
		draw_arc(center, sweep_radius, start_angle, end_angle, 24, Color(accent_color, 0.4), 2.5, true)

	# Cooldown number — centered
	var cd_text: String
	if cd >= 10.0:
		cd_text = "%d" % ceili(cd)
	else:
		cd_text = "%.1f" % cd
	draw_string(font, Vector2(x + 2.0, y + SLOT_SIZE / 2.0 + 6.0), cd_text,
		HORIZONTAL_ALIGNMENT_CENTER, SLOT_SIZE - 4.0, 16, Color(1.0, 1.0, 1.0, 0.95))


func _draw_tooltip(start_x: float, slot_y: float) -> void:
	var font := ThemeDB.fallback_font
	var spell: Dictionary = _spells[_hovered_slot]
	var spell_name: String = spell.get("name", "???")
	var spell_desc: String = spell.get("desc", "")
	var cd_max: float = spell.get("cooldown_max", 0.0)
	var cast_time: float = spell.get("dur", 0.0)

	var tip_w := 220.0
	var tip_h := 80.0

	# Check if custom tooltip needs more space
	var has_custom_tooltip := custom_tooltip_draw.is_valid()
	if has_custom_tooltip:
		tip_h = 95.0

	var slot_x := start_x + _hovered_slot * (SLOT_SIZE + SLOT_GAP)
	var tip_x := slot_x + SLOT_SIZE / 2.0 - tip_w / 2.0
	var tip_y := slot_y - tip_h - 8.0

	# Clamp to screen edges
	tip_x = clampf(tip_x, 4.0, size.x - tip_w - 4.0)

	var tip_rect := Rect2(tip_x, tip_y, tip_w, tip_h)

	# Background
	draw_rect(tip_rect, Color(0.05, 0.05, 0.1, 0.95))
	draw_rect(tip_rect, Color(0.4, 0.4, 0.5, 0.8), false, 1.0)

	# Spell name (colored by accent)
	var name_color: Color = spell.get("color", accent_color)
	draw_string(font, Vector2(tip_x + 8.0, tip_y + 16.0), spell_name,
		HORIZONTAL_ALIGNMENT_LEFT, tip_w - 16.0, 14, name_color)

	# Cast time (right-aligned, if present)
	if cast_time > 0.01:
		draw_string(font, Vector2(tip_x + tip_w - 60.0, tip_y + 16.0), "%.1fs" % cast_time,
			HORIZONTAL_ALIGNMENT_RIGHT, 52.0, 10, Color(0.7, 0.7, 0.7, 0.8))

	# Cooldown info
	if cd_max > 0.01:
		draw_string(font, Vector2(tip_x + tip_w - 60.0, tip_y + 30.0), "CD: %ds" % ceili(cd_max),
			HORIZONTAL_ALIGNMENT_RIGHT, 52.0, 10, Color(0.7, 0.7, 0.7, 0.6))

	# Custom tooltip content (Blade Dancer config transitions, etc.)
	var desc_y := tip_y + 32.0
	if has_custom_tooltip:
		custom_tooltip_draw.call(self, spell, tip_rect)
		desc_y = tip_y + 46.0

	# Description
	if spell_desc != "":
		var desc_color := Color(0.8, 0.8, 0.8, 0.9)
		if spell_desc.length() > 35:
			var split_pos := spell_desc.find(" ", 30)
			if split_pos > 0:
				draw_string(font, Vector2(tip_x + 8.0, desc_y + 14.0), spell_desc.left(split_pos),
					HORIZONTAL_ALIGNMENT_LEFT, tip_w - 16.0, 10, desc_color)
				draw_string(font, Vector2(tip_x + 8.0, desc_y + 26.0), spell_desc.substr(split_pos + 1),
					HORIZONTAL_ALIGNMENT_LEFT, tip_w - 16.0, 10, desc_color)
			else:
				draw_string(font, Vector2(tip_x + 8.0, desc_y + 14.0), spell_desc,
					HORIZONTAL_ALIGNMENT_LEFT, tip_w - 16.0, 10, desc_color)
		else:
			draw_string(font, Vector2(tip_x + 8.0, desc_y + 14.0), spell_desc,
				HORIZONTAL_ALIGNMENT_LEFT, tip_w - 16.0, 10, desc_color)
