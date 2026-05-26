extends Node

## Style helpers and dynamic UI population (character list rows).
## Static UI scenes are in scenes/ui/ and instanced in main.tscn.

const CLASS_INFO := {
	"gunner":
	{
		"name": "Gunner",
		"genre": "FPS",
		"desc": "Fast movement, high fire rate.\nRelentless aggression."
	},
	"vanguard":
	{
		"name": "Vanguard",
		"genre": "Souls-like",
		"desc": "Big AoE swings, punish windows.\nHeavy and deliberate."
	},
	"blade_dancer":
	{
		"name": "Blade Dancer",
		"genre": "State Machine",
		"desc": "5 configurations, 4 abilities each.\nHighest skill ceiling."
	},
}
const UI_SURFACE := Color(0.035, 0.045, 0.065, 0.88)
const UI_SURFACE_ALT := Color(0.05, 0.06, 0.085, 0.92)
const UI_SURFACE_ACTIVE := Color(0.08, 0.1, 0.15, 0.96)
const UI_BORDER := Color(0.28, 0.31, 0.37, 0.9)
const UI_BORDER_ACTIVE := Color(0.32, 0.58, 0.92, 0.95)
const UI_TEXT := Color(0.9, 0.93, 0.98, 0.96)
const UI_TEXT_MUTED := Color(0.6, 0.66, 0.75, 0.95)
const UI_TEXT_DIM := Color(0.48, 0.53, 0.6, 0.95)
const UI_DANGER := Color(0.86, 0.28, 0.28, 0.96)

var ctrl: Node


func _ready() -> void:
	ctrl = get_parent()


# =============================================================================
# Style helpers (public — used by UI scene scripts and other sub-systems)
# =============================================================================


func style_box(
	fill: Color, border: Color, border_width: int = 1, padding: int = 10
) -> StyleBoxFlat:
	var style := StyleBoxFlat.new()
	style.bg_color = fill
	style.border_color = border
	style.set_border_width_all(border_width)
	style.set_corner_radius_all(0)
	style.set_content_margin_all(padding)
	return style


func apply_button_style(btn: Button, accent: Color = UI_BORDER_ACTIVE) -> void:
	btn.add_theme_stylebox_override("normal", style_box(UI_SURFACE, UI_BORDER, 1, 10))
	btn.add_theme_stylebox_override("hover", style_box(UI_SURFACE_ALT, accent, 1, 10))
	btn.add_theme_stylebox_override("pressed", style_box(UI_SURFACE_ACTIVE, accent, 1, 10))
	btn.add_theme_stylebox_override("focus", style_box(UI_SURFACE_ACTIVE, accent, 1, 10))
	btn.add_theme_stylebox_override(
		"disabled", style_box(Color(UI_SURFACE, 0.45), Color(UI_BORDER, 0.4), 1, 10)
	)
	btn.add_theme_color_override("font_color", UI_TEXT)
	btn.add_theme_color_override("font_hover_color", UI_TEXT)
	btn.add_theme_color_override("font_pressed_color", UI_TEXT)
	btn.add_theme_color_override("font_focus_color", UI_TEXT)
	btn.add_theme_color_override("font_disabled_color", UI_TEXT_DIM)
	btn.add_theme_constant_override("h_separation", 8)


func apply_line_edit_style(input: LineEdit) -> void:
	input.add_theme_stylebox_override("normal", style_box(UI_SURFACE, UI_BORDER, 1, 10))
	input.add_theme_stylebox_override("focus", style_box(UI_SURFACE_ALT, UI_BORDER_ACTIVE, 1, 10))
	input.add_theme_stylebox_override("read_only", style_box(UI_SURFACE, UI_BORDER, 1, 10))
	input.add_theme_color_override("font_color", UI_TEXT)
	input.add_theme_color_override("font_placeholder_color", UI_TEXT_DIM)
	input.add_theme_color_override("caret_color", UI_TEXT)
	input.add_theme_color_override("selection_color", Color(UI_BORDER_ACTIVE, 0.35))


func apply_panel_style(panel: PanelContainer, active: bool = false, padding: int = 10) -> void:
	var fill: Color = UI_SURFACE_ACTIVE if active else UI_SURFACE
	var border: Color = UI_BORDER_ACTIVE if active else UI_BORDER
	panel.add_theme_stylebox_override("panel", style_box(fill, border, 1, padding))


func apply_overlay_label(label: Label, font_size: int, color: Color = UI_TEXT) -> void:
	label.add_theme_font_size_override("font_size", font_size)
	label.add_theme_color_override("font_color", color)


# =============================================================================
# Character select logic (dynamic population from server data)
# =============================================================================


func populate_char_select() -> void:
	if ctrl._account_username != "":
		ctrl._char_select_welcome.text = "Welcome, %s" % ctrl._account_username
	else:
		ctrl._char_select_welcome.text = ""

	for child in ctrl._char_list_container.get_children():
		child.queue_free()

	var characters: Array = ctrl._char_list_data.get("characters", [])
	var normal_style := _make_char_row_style(UI_SURFACE, UI_BORDER)
	var selected_style := _make_char_row_style(UI_SURFACE_ACTIVE, UI_BORDER_ACTIVE)

	if characters.is_empty():
		var empty_label := Label.new()
		empty_label.text = "No characters yet. Create one to get started!"
		empty_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
		apply_overlay_label(empty_label, 14, UI_TEXT_DIM)
		ctrl._char_list_container.add_child(empty_label)
		ctrl._enter_world_btn.disabled = true
		return

	ctrl._enter_world_btn.disabled = false
	for ch in characters:
		_build_char_row(ch, normal_style, selected_style)

	if ctrl._selected_char_id == 0 and not characters.is_empty():
		ctrl._select_character_row(characters[0].char_id, characters[0].class_name)


func _make_char_row_style(bg: Color, border: Color) -> StyleBoxFlat:
	var s := StyleBoxFlat.new()
	s.bg_color = bg
	s.border_color = border
	s.set_border_width_all(1)
	s.set_corner_radius_all(0)
	s.set_content_margin_all(10)
	return s


func _build_char_row(
	ch: Dictionary, normal_style: StyleBoxFlat, selected_style: StyleBoxFlat
) -> void:
	var char_id: int = ch.char_id
	var class_display: String = CLASS_INFO.get(ch.class_name, {}).get("name", ch.class_name)

	var row := PanelContainer.new()
	row.custom_minimum_size.y = 44.0
	row.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	row.set_meta("char_id", char_id)
	row.set_meta("normal_style", normal_style)
	row.set_meta("selected_style", selected_style)
	if char_id == ctrl._selected_char_id:
		row.add_theme_stylebox_override("panel", selected_style)
	else:
		row.add_theme_stylebox_override("panel", normal_style)
	ctrl._char_list_container.add_child(row)

	var hbox := HBoxContainer.new()
	hbox.add_theme_constant_override("separation", 14)
	row.add_child(hbox)

	var name_lbl := Label.new()
	name_lbl.text = ch.char_name
	apply_overlay_label(name_lbl, 16, UI_TEXT)
	name_lbl.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	hbox.add_child(name_lbl)

	var class_lbl := Label.new()
	class_lbl.text = class_display
	apply_overlay_label(class_lbl, 14, UI_BORDER_ACTIVE)
	hbox.add_child(class_lbl)

	var btn := Button.new()
	btn.flat = true
	btn.anchor_right = 1.0
	btn.anchor_bottom = 1.0
	btn.mouse_filter = Control.MOUSE_FILTER_STOP
	var id_capture: int = char_id
	var cls_capture: String = ch.class_name
	btn.pressed.connect(func(): ctrl._select_character_row(id_capture, cls_capture))
	row.add_child(btn)
