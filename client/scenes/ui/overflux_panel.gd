extends CanvasLayer

## Overflux condition selection panel. Shown before entering a portal to let the
## player choose difficulty conditions. Built entirely in code (no .tscn).

signal confirmed(conditions: Array)  # [{id: String, rank: int}, ...]
signal cancelled

const UI_SURFACE := Color(0.035, 0.045, 0.065, 0.88)
const UI_SURFACE_ALT := Color(0.05, 0.06, 0.085, 0.92)
const UI_BORDER := Color(0.22, 0.24, 0.30, 0.7)
const UI_BORDER_ACTIVE := Color(0.32, 0.58, 0.92, 0.95)
const UI_TEXT := Color(0.9, 0.93, 0.98, 0.96)
const UI_TEXT_MUTED := Color(0.6, 0.66, 0.75, 0.95)
const UI_TEXT_ACCENT := Color(0.72, 0.68, 0.55, 0.95)
const UI_OVERFLUX := Color(0.85, 0.55, 0.2, 0.95)

const CONDITIONS := [
	{
		"id": "enemy_hp",
		"name": "Fortified",
		"desc": "Increases enemy max health by 20% per rank",
		"max_rank": 5,
		"score_per_rank": 4,
	},
	{
		"id": "tempered",
		"name": "Tempered",
		"desc": "Boss uses a smarter behavior tree with new abilities",
		"max_rank": 1,
		"score_per_rank": 10,
	},
	{
		"id": "frenzied",
		"name": "Frenzied",
		"desc": "Mobs use a more aggressive behavior tree with new abilities",
		"max_rank": 1,
		"score_per_rank": 10,
	},
	{
		"id": "volatile",
		"name": "Volatile",
		"desc": "Boss ability patterns are denser and more complex",
		"max_rank": 1,
		"score_per_rank": 10,
	},
]

var _is_open: bool = false
var _bg: ColorRect
var _outer_panel: PanelContainer
var _score_label: Label
var _rank_buttons: Dictionary = {}  # condition_id -> Array[Button]
var _selected_ranks: Dictionary = {}  # condition_id -> int


func _ready() -> void:
	process_mode = Node.PROCESS_MODE_ALWAYS
	layer = 18
	visible = false

	for c in CONDITIONS:
		_selected_ranks[c.id] = 0

	_bg = ColorRect.new()
	_bg.color = Color(0.0, 0.0, 0.0, 0.6)
	_bg.set_anchors_preset(Control.PRESET_FULL_RECT)
	_bg.mouse_filter = Control.MOUSE_FILTER_STOP
	add_child(_bg)

	var center := CenterContainer.new()
	center.set_anchors_preset(Control.PRESET_FULL_RECT)
	center.mouse_filter = Control.MOUSE_FILTER_PASS
	_bg.add_child(center)

	_outer_panel = PanelContainer.new()
	_outer_panel.custom_minimum_size = Vector2(480, 0)
	var panel_style := StyleBoxFlat.new()
	panel_style.bg_color = UI_SURFACE
	panel_style.border_color = UI_BORDER
	panel_style.set_border_width_all(1)
	panel_style.set_content_margin_all(24)
	panel_style.set_corner_radius_all(4)
	_outer_panel.add_theme_stylebox_override("panel", panel_style)
	center.add_child(_outer_panel)

	var vbox := VBoxContainer.new()
	vbox.add_theme_constant_override("separation", 16)
	_outer_panel.add_child(vbox)

	_build_title(vbox)
	_build_conditions(vbox)
	_build_footer(vbox)


func _build_title(parent: VBoxContainer) -> void:
	var title := Label.new()
	title.text = "OVERFLUX"
	title.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	title.add_theme_font_size_override("font_size", 22)
	title.add_theme_color_override("font_color", UI_OVERFLUX)
	parent.add_child(title)

	var sep := HSeparator.new()
	sep.add_theme_stylebox_override("separator", _make_separator_style())
	parent.add_child(sep)


func _build_conditions(parent: VBoxContainer) -> void:
	for c in CONDITIONS:
		var row := VBoxContainer.new()
		row.add_theme_constant_override("separation", 6)
		parent.add_child(row)

		var header := HBoxContainer.new()
		header.add_theme_constant_override("separation", 8)
		row.add_child(header)

		var name_label := Label.new()
		name_label.text = c.name
		name_label.add_theme_font_size_override("font_size", 16)
		name_label.add_theme_color_override("font_color", UI_TEXT)
		name_label.size_flags_horizontal = Control.SIZE_EXPAND_FILL
		header.add_child(name_label)

		# Rank buttons: 0 (off) through max_rank
		var btn_row := HBoxContainer.new()
		btn_row.add_theme_constant_override("separation", 4)
		header.add_child(btn_row)

		var buttons: Array[Button] = []
		for rank in range(c.max_rank + 1):
			var btn := Button.new()
			btn.text = "Off" if rank == 0 else str(rank)
			btn.custom_minimum_size = Vector2(36, 28)
			btn.add_theme_font_size_override("font_size", 13)
			btn.pressed.connect(_on_rank_pressed.bind(c.id, rank))
			_style_rank_button(btn, rank == 0)
			btn_row.add_child(btn)
			buttons.append(btn)
		_rank_buttons[c.id] = buttons

		var desc_label := Label.new()
		desc_label.text = c.desc
		desc_label.add_theme_font_size_override("font_size", 12)
		desc_label.add_theme_color_override("font_color", UI_TEXT_MUTED)
		desc_label.autowrap_mode = TextServer.AUTOWRAP_WORD
		row.add_child(desc_label)


func _build_footer(parent: VBoxContainer) -> void:
	var sep := HSeparator.new()
	sep.add_theme_stylebox_override("separator", _make_separator_style())
	parent.add_child(sep)

	var footer := HBoxContainer.new()
	footer.add_theme_constant_override("separation", 12)
	footer.alignment = BoxContainer.ALIGNMENT_CENTER
	parent.add_child(footer)

	_score_label = Label.new()
	_score_label.add_theme_font_size_override("font_size", 18)
	_score_label.add_theme_color_override("font_color", UI_OVERFLUX)
	_score_label.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	footer.add_child(_score_label)
	_update_score_label()

	var cancel_btn := Button.new()
	cancel_btn.text = "Cancel"
	cancel_btn.custom_minimum_size = Vector2(100, 36)
	cancel_btn.pressed.connect(_cancel)
	_style_action_button(cancel_btn, false)
	footer.add_child(cancel_btn)

	var enter_btn := Button.new()
	enter_btn.text = "Enter Instance"
	enter_btn.custom_minimum_size = Vector2(140, 36)
	enter_btn.pressed.connect(_confirm)
	_style_action_button(enter_btn, true)
	footer.add_child(enter_btn)


func open() -> void:
	# Reset all ranks to 0
	for c in CONDITIONS:
		_selected_ranks[c.id] = 0
	_refresh_all_buttons()
	_update_score_label()
	_is_open = true
	visible = true
	Input.set_mouse_mode(Input.MOUSE_MODE_VISIBLE)


func close() -> void:
	_is_open = false
	visible = false


func _confirm() -> void:
	var conditions: Array = []
	for c in CONDITIONS:
		var rank: int = _selected_ranks.get(c.id, 0)
		if rank > 0:
			conditions.append({"id": c.id, "rank": rank})
	close()
	confirmed.emit(conditions)


func _cancel() -> void:
	close()
	cancelled.emit()


func _on_rank_pressed(cond_id: String, rank: int) -> void:
	_selected_ranks[cond_id] = rank
	_refresh_buttons(cond_id)
	_update_score_label()


func _refresh_all_buttons() -> void:
	for cond_id in _rank_buttons:
		_refresh_buttons(cond_id)


func _refresh_buttons(cond_id: String) -> void:
	var buttons: Array = _rank_buttons[cond_id]
	var current_rank: int = _selected_ranks.get(cond_id, 0)
	for i in range(buttons.size()):
		_style_rank_button(buttons[i], i == current_rank)


func _update_score_label() -> void:
	var total := 0
	for c in CONDITIONS:
		var rank: int = _selected_ranks.get(c.id, 0)
		total += rank * c.score_per_rank
	_score_label.text = "Overflux: %d" % total


func _input(event: InputEvent) -> void:
	if not _is_open:
		return

	if event is InputEventKey and event.pressed and not event.echo:
		if event.keycode == KEY_ESCAPE:
			_cancel()
			get_viewport().set_input_as_handled()
			return

	if event is InputEventMouseButton and event.pressed and event.button_index == MOUSE_BUTTON_LEFT:
		var panel_rect := _outer_panel.get_global_rect()
		if not panel_rect.has_point(event.position):
			_cancel()
			get_viewport().set_input_as_handled()


# -- Styling helpers --


func _style_rank_button(btn: Button, active: bool) -> void:
	var style := StyleBoxFlat.new()
	style.set_corner_radius_all(3)
	style.set_border_width_all(1)
	if active:
		style.bg_color = Color(0.12, 0.18, 0.32, 0.95)
		style.border_color = UI_BORDER_ACTIVE
		btn.add_theme_color_override("font_color", UI_TEXT)
	else:
		style.bg_color = Color(0.04, 0.05, 0.07, 0.9)
		style.border_color = UI_BORDER
		btn.add_theme_color_override("font_color", UI_TEXT_MUTED)
	btn.add_theme_stylebox_override("normal", style)

	var hover := style.duplicate()
	hover.bg_color = hover.bg_color.lightened(0.1)
	btn.add_theme_stylebox_override("hover", hover)
	btn.add_theme_stylebox_override("pressed", style)
	btn.add_theme_stylebox_override("focus", StyleBoxEmpty.new())


func _style_action_button(btn: Button, primary: bool) -> void:
	var style := StyleBoxFlat.new()
	style.set_corner_radius_all(3)
	style.set_border_width_all(1)
	if primary:
		style.bg_color = Color(0.15, 0.25, 0.45, 0.95)
		style.border_color = UI_BORDER_ACTIVE
		btn.add_theme_color_override("font_color", UI_TEXT)
	else:
		style.bg_color = Color(0.06, 0.07, 0.1, 0.9)
		style.border_color = UI_BORDER
		btn.add_theme_color_override("font_color", UI_TEXT_MUTED)
	btn.add_theme_stylebox_override("normal", style)

	var hover := style.duplicate()
	hover.bg_color = hover.bg_color.lightened(0.1)
	btn.add_theme_stylebox_override("hover", hover)
	btn.add_theme_stylebox_override("pressed", style)
	btn.add_theme_stylebox_override("focus", StyleBoxEmpty.new())


func _make_separator_style() -> StyleBoxLine:
	var s := StyleBoxLine.new()
	s.color = UI_BORDER
	s.thickness = 1
	return s
