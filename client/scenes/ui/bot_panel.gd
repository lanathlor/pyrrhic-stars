extends CanvasLayer

## Bot spawn panel — select class/spec to spawn AI bots that fight alongside you.
## Opened with [Ctrl+G] in dev mode. Built entirely in code (no .tscn).

signal bot_spawned(cls_name: String, spec_id: String)
signal bot_dismissed(bot_id: int)
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
const UI_BOT_ACTIVE := Color(0.35, 0.78, 0.35, 0.95)

const ROLE_COLORS := {
	"DPS": Color(0.82, 0.55, 0.25),
	"Tank": Color(0.35, 0.75, 0.45),
	"Healer": Color(0.4, 0.72, 0.85),
}

const MAX_BOTS := 4

# Class/spec data — only implemented specs
const CLASS_DATA := {
	"gunner":
	{
		"display": "Gunner",
		"specs":
		[
			{"id": "assault", "name": "Assault", "role": "DPS"},
		]
	},
	"vanguard":
	{
		"display": "Vanguard",
		"specs":
		[
			{"id": "blade", "name": "Blade", "role": "DPS"},
			{"id": "shield", "name": "Shield", "role": "Tank"},
		]
	},
	"blade_dancer":
	{
		"display": "Blade Dancer",
		"specs":
		[
			{"id": "multi_blade", "name": "Multi Blade", "role": "DPS"},
		]
	},
	"arcanotechnicien":
	{
		"display": "Arcanotechnicien",
		"specs":
		[
			{"id": "harmonist", "name": "Harmonist", "role": "Healer"},
		]
	},
}

var _visible: bool = false
var _bg: ColorRect
var _outer_panel: PanelContainer
var _columns_container: HBoxContainer
var _count_label: Label
var _dismiss_btn: Button
var _bot_configs: Array = []  # Array of {"class": str, "spec": str}


func _ready() -> void:
	process_mode = Node.PROCESS_MODE_ALWAYS
	layer = 18
	visible = false

	# Full-screen dark wash
	_bg = ColorRect.new()
	_bg.color = Color(0.0, 0.0, 0.0, 0.6)
	_bg.set_anchors_preset(Control.PRESET_FULL_RECT)
	_bg.mouse_filter = Control.MOUSE_FILTER_STOP
	add_child(_bg)

	# Margin around the whole thing
	var margin := MarginContainer.new()
	margin.set_anchors_preset(Control.PRESET_FULL_RECT)
	margin.add_theme_constant_override("margin_left", 120)
	margin.add_theme_constant_override("margin_right", 120)
	margin.add_theme_constant_override("margin_top", 80)
	margin.add_theme_constant_override("margin_bottom", 80)
	margin.mouse_filter = Control.MOUSE_FILTER_PASS
	_bg.add_child(margin)

	# Outer frame
	var outer_vbox := VBoxContainer.new()
	outer_vbox.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	outer_vbox.size_flags_vertical = Control.SIZE_EXPAND_FILL
	outer_vbox.add_theme_constant_override("separation", 0)
	margin.add_child(outer_vbox)

	# Title bar
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
	outer_vbox.add_child(title_panel)

	var title := Label.new()
	title.text = "SPAWN BOT"
	title.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	title.add_theme_font_size_override("font_size", 16)
	title.add_theme_color_override("font_color", UI_TEXT_ACCENT)
	title_panel.add_child(title)

	# Main body panel
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
	outer_vbox.add_child(_outer_panel)

	var body_vbox := VBoxContainer.new()
	body_vbox.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	body_vbox.size_flags_vertical = Control.SIZE_EXPAND_FILL
	body_vbox.add_theme_constant_override("separation", 12)
	_outer_panel.add_child(body_vbox)

	# Columns container (classes go side by side)
	_columns_container = HBoxContainer.new()
	_columns_container.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	_columns_container.size_flags_vertical = Control.SIZE_EXPAND_FILL
	_columns_container.add_theme_constant_override("separation", 12)
	_columns_container.alignment = BoxContainer.ALIGNMENT_CENTER
	body_vbox.add_child(_columns_container)

	# Build class columns
	for class_id in CLASS_DATA:
		_build_class_column(class_id, CLASS_DATA[class_id])

	# Bottom bar: count + dismiss all
	var bottom_bar := HBoxContainer.new()
	bottom_bar.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	bottom_bar.add_theme_constant_override("separation", 12)
	bottom_bar.alignment = BoxContainer.ALIGNMENT_CENTER
	body_vbox.add_child(bottom_bar)

	_count_label = Label.new()
	_count_label.text = "Bots: 0 / %d" % MAX_BOTS
	_count_label.add_theme_font_size_override("font_size", 14)
	_count_label.add_theme_color_override("font_color", UI_TEXT_MUTED)
	bottom_bar.add_child(_count_label)

	var spacer := Control.new()
	spacer.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	bottom_bar.add_child(spacer)

	_dismiss_btn = Button.new()
	_dismiss_btn.text = "Dismiss All"
	_dismiss_btn.custom_minimum_size = Vector2(120, 30)
	_style_danger_button(_dismiss_btn)
	_dismiss_btn.pressed.connect(_on_dismiss_all)
	bottom_bar.add_child(_dismiss_btn)


func _build_class_column(class_id: String, data: Dictionary) -> void:
	var column := PanelContainer.new()
	column.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	column.size_flags_vertical = Control.SIZE_EXPAND_FILL
	var style := _make_column_style(UI_SURFACE, UI_BORDER, 1)
	column.add_theme_stylebox_override("panel", style)
	_columns_container.add_child(column)

	var vbox := VBoxContainer.new()
	vbox.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	vbox.size_flags_vertical = Control.SIZE_EXPAND_FILL
	vbox.add_theme_constant_override("separation", 0)
	column.add_child(vbox)

	# Class name
	_add_spacer(vbox, 20)
	var name_label := Label.new()
	name_label.text = data["display"]
	name_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	name_label.add_theme_font_size_override("font_size", 22)
	name_label.add_theme_color_override("font_color", UI_TEXT)
	vbox.add_child(name_label)

	_add_spacer(vbox, 8)
	vbox.add_child(_make_separator())
	_add_spacer(vbox, 16)

	# One button per spec
	for spec in data["specs"]:
		var spec_id: String = spec["id"]
		var spec_name: String = spec["name"]
		var role: String = spec["role"]

		var btn_row := HBoxContainer.new()
		btn_row.size_flags_horizontal = Control.SIZE_EXPAND_FILL
		btn_row.alignment = BoxContainer.ALIGNMENT_CENTER
		vbox.add_child(btn_row)

		var btn := Button.new()
		btn.text = "%s  (%s)" % [spec_name, role]
		btn.custom_minimum_size = Vector2(180, 36)
		_style_spawn_button(btn)
		btn.pressed.connect(_on_spawn_pressed.bind(class_id, spec_id))
		btn_row.add_child(btn)

		_add_spacer(vbox, 8)

	# Fill remaining space
	var flex := Control.new()
	flex.size_flags_vertical = Control.SIZE_EXPAND_FILL
	vbox.add_child(flex)

	_add_spacer(vbox, 16)


func toggle() -> void:
	_visible = not _visible
	visible = _visible
	if _visible:
		_update_count()


func _on_spawn_pressed(class_id: String, spec_id: String) -> void:
	if _bot_configs.size() >= MAX_BOTS:
		return
	_bot_configs.append({"class": class_id, "spec": spec_id})
	NetworkManager.send_debug_spawn_bot(class_id, spec_id)
	bot_spawned.emit(class_id, spec_id)
	_update_count()


func _on_dismiss_all() -> void:
	_bot_configs.clear()
	NetworkManager.send_debug_dismiss_bot(0)
	bot_dismissed.emit(0)
	_update_count()


func _update_count() -> void:
	_count_label.text = "Bots: %d / %d" % [_bot_configs.size(), MAX_BOTS]


## Returns the current bot configs so they can be re-sent on zone transfer.
func get_bot_configs() -> Array:
	return _bot_configs.duplicate()


func _input(event: InputEvent) -> void:
	if not _visible:
		return

	if event is InputEventKey and event.pressed and not event.echo:
		if event.keycode == KEY_ESCAPE or (event.ctrl_pressed and event.keycode == KEY_G):
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


func _style_spawn_button(btn: Button) -> void:
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


func _style_danger_button(btn: Button) -> void:
	var normal := StyleBoxFlat.new()
	normal.bg_color = Color(0.08, 0.03, 0.03, 0.9)
	normal.border_color = UI_DANGER
	normal.border_width_bottom = 1
	normal.border_width_top = 1
	normal.border_width_left = 1
	normal.border_width_right = 1
	normal.corner_radius_top_left = 3
	normal.corner_radius_top_right = 3
	normal.corner_radius_bottom_left = 3
	normal.corner_radius_bottom_right = 3

	var hover := normal.duplicate()
	hover.bg_color = Color(0.14, 0.04, 0.04, 0.95)

	btn.add_theme_stylebox_override("normal", normal)
	btn.add_theme_stylebox_override("hover", hover)
	btn.add_theme_stylebox_override("pressed", normal)
	btn.add_theme_stylebox_override("focus", normal)
	btn.add_theme_font_size_override("font_size", 13)
	btn.add_theme_color_override("font_color", UI_DANGER)
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
