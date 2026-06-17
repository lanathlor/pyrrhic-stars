extends CanvasLayer

## Spec selection panel — full-screen overlay with side-by-side spec columns.
## Inspired by WoW's specialization screen. Opened with [N].
## Built entirely in code (no .tscn).

signal spec_selected(spec_id: String)
signal closed

const UI_SURFACE := Color(0.035, 0.045, 0.065, 0.88)
const UI_SURFACE_ALT := Color(0.05, 0.06, 0.085, 0.92)
const UI_SURFACE_ACTIVE := Color(0.06, 0.075, 0.12, 0.94)
const UI_BORDER := Color(0.22, 0.24, 0.30, 0.7)
const UI_BORDER_ACTIVE := Color(0.32, 0.58, 0.92, 0.95)
const UI_TEXT := Color(0.9, 0.93, 0.98, 0.96)
const UI_TEXT_MUTED := Color(0.6, 0.66, 0.75, 0.95)
const UI_TEXT_DIM := Color(0.48, 0.53, 0.6, 0.95)
const UI_TEXT_ACCENT := Color(0.72, 0.68, 0.55, 0.95)
const UI_DANGER := Color(0.86, 0.28, 0.28, 0.96)
const UI_ACTIVE_GREEN := Color(0.35, 0.78, 0.35, 0.95)

const ROLE_COLORS := {
	"DPS": Color(0.82, 0.55, 0.25),
	"Tank": Color(0.35, 0.75, 0.45),
	"Healer": Color(0.4, 0.72, 0.85),
}

var _visible: bool = false
var _bg: ColorRect
var _outer_panel: PanelContainer
var _columns_container: HBoxContainer
var _current_spec: String = ""


func _ready() -> void:
	process_mode = Node.PROCESS_MODE_ALWAYS
	layer = 17
	visible = false

	_bg = ColorRect.new()
	_bg.color = Color(0.0, 0.0, 0.0, 0.6)
	_bg.set_anchors_preset(Control.PRESET_FULL_RECT)
	_bg.mouse_filter = Control.MOUSE_FILTER_STOP
	add_child(_bg)

	var margin := MarginContainer.new()
	margin.set_anchors_preset(Control.PRESET_FULL_RECT)
	margin.add_theme_constant_override("margin_left", 80)
	margin.add_theme_constant_override("margin_right", 80)
	margin.add_theme_constant_override("margin_top", 50)
	margin.add_theme_constant_override("margin_bottom", 50)
	margin.mouse_filter = Control.MOUSE_FILTER_PASS
	_bg.add_child(margin)

	var outer_vbox := _build_outer_frame(margin)
	_build_title_bar(outer_vbox)
	_build_body_panel(outer_vbox)


func _build_outer_frame(parent: Control) -> VBoxContainer:
	var outer_vbox := VBoxContainer.new()
	outer_vbox.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	outer_vbox.size_flags_vertical = Control.SIZE_EXPAND_FILL
	outer_vbox.add_theme_constant_override("separation", 0)
	parent.add_child(outer_vbox)
	return outer_vbox


func _build_title_bar(parent: VBoxContainer) -> void:
	var title_panel := PanelContainer.new()
	title_panel.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	var title_style := StyleBoxFlat.new()
	title_style.bg_color = Color(0.025, 0.03, 0.05, 0.95)
	title_style.border_color = UI_BORDER
	title_style.border_width_bottom = 1
	title_style.border_width_top = 1
	title_style.border_width_left = 1
	title_style.border_width_right = 1
	title_style.content_margin_top = 10
	title_style.content_margin_bottom = 10
	title_panel.add_theme_stylebox_override("panel", title_style)
	parent.add_child(title_panel)

	var title := Label.new()
	title.text = "SPECIALIZATION"
	title.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	title.add_theme_font_size_override("font_size", 16)
	title.add_theme_color_override("font_color", UI_TEXT_ACCENT)
	title_panel.add_child(title)


func _build_body_panel(parent: VBoxContainer) -> void:
	_outer_panel = PanelContainer.new()
	_outer_panel.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	_outer_panel.size_flags_vertical = Control.SIZE_EXPAND_FILL
	_outer_panel.mouse_filter = Control.MOUSE_FILTER_STOP
	var body_style := StyleBoxFlat.new()
	body_style.bg_color = Color(0.02, 0.025, 0.04, 0.92)
	body_style.border_color = UI_BORDER
	body_style.border_width_bottom = 1
	body_style.border_width_left = 1
	body_style.border_width_right = 1
	body_style.content_margin_left = 16
	body_style.content_margin_right = 16
	body_style.content_margin_top = 16
	body_style.content_margin_bottom = 16
	_outer_panel.add_theme_stylebox_override("panel", body_style)
	parent.add_child(_outer_panel)

	_columns_container = HBoxContainer.new()
	_columns_container.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	_columns_container.size_flags_vertical = Control.SIZE_EXPAND_FILL
	_columns_container.add_theme_constant_override("separation", 12)
	_columns_container.alignment = BoxContainer.ALIGNMENT_CENTER
	_outer_panel.add_child(_columns_container)


func toggle() -> void:
	_visible = not _visible
	visible = _visible
	if _visible:
		Input.mouse_mode = Input.MOUSE_MODE_VISIBLE


func set_specs(specs: Array, current_spec: String) -> void:
	_current_spec = current_spec

	# Clear existing columns
	for child in _columns_container.get_children():
		child.queue_free()

	for spec in specs:
		_build_column(spec, spec.get("id", "") == current_spec)


func _build_column(spec: Dictionary, is_current: bool) -> void:
	var spec_id: String = spec.get("id", "")
	var implemented: bool = spec.get("implemented", false)

	var border_width: int = 2 if is_current else 1
	var bg_color: Color = UI_SURFACE_ACTIVE if is_current else UI_SURFACE
	var border_color: Color = UI_BORDER_ACTIVE if is_current else UI_BORDER

	var column := PanelContainer.new()
	column.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	column.size_flags_vertical = Control.SIZE_EXPAND_FILL
	var col_style := _make_column_style(bg_color, border_color, border_width)
	column.add_theme_stylebox_override("panel", col_style)
	if not implemented and not is_current:
		column.modulate.a = 0.55
	_columns_container.add_child(column)

	var vbox := VBoxContainer.new()
	vbox.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	vbox.size_flags_vertical = Control.SIZE_EXPAND_FILL
	vbox.add_theme_constant_override("separation", 0)
	column.add_child(vbox)

	_build_column_header(vbox, spec)
	_build_column_description(vbox, spec)
	_build_column_mastery(vbox, spec.get("mastery", ""))
	_build_column_bottom(vbox, spec_id, is_current, implemented)
	_add_spacer(vbox, 16)

	if implemented and not is_current:
		_build_column_hover_overlay(column, spec_id, bg_color, border_color, border_width)


func _build_column_header(vbox: VBoxContainer, spec: Dictionary) -> void:
	_add_spacer(vbox, 30)

	var name_label := Label.new()
	name_label.text = spec.get("name", "???")
	name_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	name_label.add_theme_font_size_override("font_size", 24)
	name_label.add_theme_color_override("font_color", UI_TEXT)
	vbox.add_child(name_label)

	_add_spacer(vbox, 6)

	var role: String = spec.get("role", "")
	var role_label := Label.new()
	role_label.text = role
	role_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	role_label.add_theme_font_size_override("font_size", 14)
	role_label.add_theme_color_override("font_color", ROLE_COLORS.get(role, UI_TEXT_MUTED))
	vbox.add_child(role_label)

	_add_spacer(vbox, 12)
	vbox.add_child(_make_separator())
	_add_spacer(vbox, 12)


func _build_column_description(vbox: VBoxContainer, spec: Dictionary) -> void:
	var desc_label := Label.new()
	desc_label.text = spec.get("desc", "")
	desc_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	desc_label.add_theme_font_size_override("font_size", 13)
	desc_label.add_theme_color_override("font_color", UI_TEXT_ACCENT)
	desc_label.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	desc_label.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	vbox.add_child(desc_label)

	var target: String = spec.get("target", "")
	var damage: String = spec.get("damage", "")
	if target != "" and damage != "":
		_add_spacer(vbox, 8)
		var info_label := Label.new()
		info_label.text = "%s  ·  %s" % [target, damage]
		info_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
		info_label.add_theme_font_size_override("font_size", 12)
		info_label.add_theme_color_override("font_color", UI_TEXT_MUTED)
		vbox.add_child(info_label)

	_add_spacer(vbox, 14)
	vbox.add_child(_make_separator())
	_add_spacer(vbox, 14)


func _build_column_mastery(vbox: VBoxContainer, mastery: String) -> void:
	var mastery_header := Label.new()
	mastery_header.text = "Mastery"
	mastery_header.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	mastery_header.add_theme_font_size_override("font_size", 14)
	mastery_header.add_theme_color_override("font_color", UI_TEXT_MUTED)
	vbox.add_child(mastery_header)

	_add_spacer(vbox, 4)

	var mastery_label := Label.new()
	mastery_label.text = mastery
	mastery_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	mastery_label.add_theme_font_size_override("font_size", 12)
	mastery_label.add_theme_color_override("font_color", UI_TEXT_DIM)
	mastery_label.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	mastery_label.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	vbox.add_child(mastery_label)

	var flex_spacer := Control.new()
	flex_spacer.size_flags_vertical = Control.SIZE_EXPAND_FILL
	vbox.add_child(flex_spacer)


func _build_column_bottom(
	vbox: VBoxContainer, spec_id: String, is_current: bool, implemented: bool
) -> void:
	var bottom_container := CenterContainer.new()
	bottom_container.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	bottom_container.custom_minimum_size = Vector2(0, 40)
	vbox.add_child(bottom_container)

	if is_current:
		var active_label := Label.new()
		active_label.text = "Active"
		active_label.add_theme_font_size_override("font_size", 16)
		active_label.add_theme_color_override("font_color", UI_ACTIVE_GREEN)
		bottom_container.add_child(active_label)
	elif not implemented:
		var soon_label := Label.new()
		soon_label.text = "COMING SOON"
		soon_label.add_theme_font_size_override("font_size", 14)
		soon_label.add_theme_color_override("font_color", UI_DANGER)
		bottom_container.add_child(soon_label)
	else:
		var activate_btn := Button.new()
		activate_btn.text = "Activate"
		activate_btn.custom_minimum_size = Vector2(140, 32)
		_style_activate_button(activate_btn)
		activate_btn.pressed.connect(_on_card_pressed.bind(spec_id))
		bottom_container.add_child(activate_btn)


func _build_column_hover_overlay(
	column: PanelContainer, spec_id: String, bg_color: Color, border_color: Color, border_width: int
) -> void:
	var hover_btn := Button.new()
	hover_btn.flat = true
	hover_btn.set_anchors_preset(Control.PRESET_FULL_RECT)
	hover_btn.mouse_filter = Control.MOUSE_FILTER_PASS
	var empty_style := StyleBoxEmpty.new()
	hover_btn.add_theme_stylebox_override("normal", empty_style)
	hover_btn.add_theme_stylebox_override("hover", empty_style)
	hover_btn.add_theme_stylebox_override("pressed", empty_style)
	hover_btn.add_theme_stylebox_override("focus", empty_style)
	hover_btn.pressed.connect(_on_card_pressed.bind(spec_id))
	hover_btn.mouse_entered.connect(_on_column_hover_enter.bind(column))
	hover_btn.mouse_exited.connect(
		_on_column_hover_exit.bind(column, bg_color, border_color, border_width)
	)
	column.add_child(hover_btn)


func _on_card_pressed(spec_id: String) -> void:
	spec_selected.emit(spec_id)


func _on_column_hover_enter(column: PanelContainer) -> void:
	var hover_style := _make_column_style(UI_SURFACE_ALT, Color(0.3, 0.4, 0.55, 0.8), 1)
	column.add_theme_stylebox_override("panel", hover_style)


func _on_column_hover_exit(column: PanelContainer, bg: Color, border: Color, border_w: int) -> void:
	var style := _make_column_style(bg, border, border_w)
	column.add_theme_stylebox_override("panel", style)


func _input(event: InputEvent) -> void:
	if not _visible:
		return

	# Esc is handled centrally (main._input → ui_controller.close_open_overlay).
	if event is InputEventKey and event.pressed and not event.echo:
		if event.keycode == KEY_N:
			_close()
			get_viewport().set_input_as_handled()
			return

	if event is InputEventMouseButton and event.pressed and event.button_index == MOUSE_BUTTON_LEFT:
		var panel_rect := _outer_panel.get_global_rect()
		if not panel_rect.has_point(event.position):
			_close()
			get_viewport().set_input_as_handled()


func _close() -> void:
	_visible = false
	visible = false
	closed.emit()


# =============================================================================
# Style helpers
# =============================================================================


func _make_column_style(bg: Color, border: Color, border_w: int) -> StyleBoxFlat:
	var sb := StyleBoxFlat.new()
	sb.bg_color = bg
	sb.border_color = border
	sb.border_width_bottom = border_w
	sb.border_width_top = border_w
	sb.border_width_left = border_w
	sb.border_width_right = border_w
	sb.corner_radius_top_left = 4
	sb.corner_radius_top_right = 4
	sb.corner_radius_bottom_left = 4
	sb.corner_radius_bottom_right = 4
	sb.content_margin_left = 20
	sb.content_margin_right = 20
	sb.content_margin_top = 0
	sb.content_margin_bottom = 0
	return sb


func _style_activate_button(btn: Button) -> void:
	var normal := StyleBoxFlat.new()
	normal.bg_color = Color(0.04, 0.05, 0.08, 0.9)
	normal.border_color = UI_BORDER_ACTIVE
	normal.border_width_bottom = 1
	normal.border_width_top = 1
	normal.border_width_left = 1
	normal.border_width_right = 1
	normal.corner_radius_top_left = 3
	normal.corner_radius_top_right = 3
	normal.corner_radius_bottom_left = 3
	normal.corner_radius_bottom_right = 3

	var hover := normal.duplicate()
	hover.bg_color = Color(0.06, 0.08, 0.14, 0.95)

	var pressed := normal.duplicate()
	pressed.bg_color = Color(0.08, 0.1, 0.18, 0.95)

	btn.add_theme_stylebox_override("normal", normal)
	btn.add_theme_stylebox_override("hover", hover)
	btn.add_theme_stylebox_override("pressed", pressed)
	btn.add_theme_stylebox_override("focus", normal)
	btn.add_theme_font_size_override("font_size", 14)
	btn.add_theme_color_override("font_color", UI_BORDER_ACTIVE)
	btn.add_theme_color_override("font_hover_color", UI_TEXT)


func _make_separator() -> HSeparator:
	var sep := HSeparator.new()
	var sep_style := StyleBoxLine.new()
	sep_style.color = Color(0.2, 0.22, 0.28, 0.5)
	sep_style.thickness = 1
	sep.add_theme_stylebox_override("separator", sep_style)
	return sep


func _add_spacer(parent: Control, height: float) -> void:
	var spacer := Control.new()
	spacer.custom_minimum_size = Vector2(0, height)
	parent.add_child(spacer)
