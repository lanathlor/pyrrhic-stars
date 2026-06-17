extends CanvasLayer

## Dev mode debug panel for boss ability tuning.
## Toggled with Ctrl+D. Populated from OpDebugInfo.

const UI_SURFACE := Color(0.035, 0.045, 0.065, 0.92)
const UI_BORDER := Color(0.28, 0.31, 0.37, 0.9)
const UI_TEXT := Color(0.9, 0.93, 0.98, 0.96)
const UI_TEXT_MUTED := Color(0.6, 0.66, 0.75, 0.95)
const UI_ACCENT := Color(0.32, 0.58, 0.92, 0.95)

var _abilities: PackedStringArray = []
var _def_name: String = ""
var _god_mode: bool = false  # Opt-in: toggle on in the debug panel when needed
var _repeat_ability: String = ""
var _info_requested: bool = false

var _panel: PanelContainer
var _vbox: VBoxContainer
var _ability_container: VBoxContainer
var _time_scale_slider: HSlider
var _time_scale_label: Label
var _god_mode_check: CheckButton
var _repeat_label: Label


func _ready() -> void:
	layer = 100
	visible = false
	_build_ui()
	NetworkManager.debug_info_received.connect(_on_debug_info)


func toggle() -> void:
	visible = !visible
	if visible and not _info_requested:
		NetworkManager.debug.send_request_info()
		_info_requested = true


func _build_ui() -> void:
	_panel = PanelContainer.new()
	_panel.anchor_left = 1.0
	_panel.anchor_right = 1.0
	_panel.anchor_top = 0.0
	_panel.anchor_bottom = 0.0
	_panel.offset_left = -280
	_panel.offset_right = -10
	_panel.offset_top = 10
	_panel.custom_minimum_size = Vector2(270, 0)

	var style := StyleBoxFlat.new()
	style.bg_color = UI_SURFACE
	style.border_color = UI_BORDER
	style.set_border_width_all(1)
	style.set_corner_radius_all(4)
	style.set_content_margin_all(10)
	_panel.add_theme_stylebox_override("panel", style)
	add_child(_panel)

	_vbox = VBoxContainer.new()
	_vbox.add_theme_constant_override("separation", 6)
	_panel.add_child(_vbox)

	_build_header_section()
	_build_boss_section()
	_build_phase_section()
	_build_god_mode_section()
	_build_time_scale_section()
	_build_action_buttons()


func _build_header_section() -> void:
	var title := Label.new()
	title.text = "DEBUG PANEL"
	title.add_theme_color_override("font_color", UI_ACCENT)
	title.add_theme_font_size_override("font_size", 14)
	_vbox.add_child(title)
	_add_separator()


func _build_boss_section() -> void:
	_repeat_label = Label.new()
	_repeat_label.text = "Waiting for boss info..."
	_repeat_label.add_theme_color_override("font_color", UI_TEXT_MUTED)
	_repeat_label.add_theme_font_size_override("font_size", 11)
	_vbox.add_child(_repeat_label)

	var ability_label := Label.new()
	ability_label.text = "ABILITIES"
	ability_label.add_theme_color_override("font_color", UI_TEXT_MUTED)
	ability_label.add_theme_font_size_override("font_size", 11)
	_vbox.add_child(ability_label)

	_ability_container = VBoxContainer.new()
	_ability_container.add_theme_constant_override("separation", 3)
	_vbox.add_child(_ability_container)
	_add_separator()


func _build_phase_section() -> void:
	var phase_label := Label.new()
	phase_label.text = "PHASE"
	phase_label.add_theme_color_override("font_color", UI_TEXT_MUTED)
	phase_label.add_theme_font_size_override("font_size", 11)
	_vbox.add_child(phase_label)

	var phase_row := HBoxContainer.new()
	phase_row.add_theme_constant_override("separation", 4)
	for i in range(1, 4):
		var btn := Button.new()
		btn.text = str(i)
		btn.custom_minimum_size = Vector2(50, 28)
		btn.pressed.connect(_on_phase_pressed.bind(i))
		_style_button(btn)
		phase_row.add_child(btn)
	_vbox.add_child(phase_row)
	_add_separator()


func _build_god_mode_section() -> void:
	_god_mode_check = CheckButton.new()
	_god_mode_check.text = "God Mode"
	_god_mode_check.button_pressed = false
	_god_mode_check.add_theme_color_override("font_color", UI_TEXT)
	_god_mode_check.add_theme_font_size_override("font_size", 12)
	_god_mode_check.toggled.connect(_on_god_mode_toggled)
	_vbox.add_child(_god_mode_check)
	_add_separator()


func _build_time_scale_section() -> void:
	var ts_label := Label.new()
	ts_label.text = "TIME SCALE"
	ts_label.add_theme_color_override("font_color", UI_TEXT_MUTED)
	ts_label.add_theme_font_size_override("font_size", 11)
	_vbox.add_child(ts_label)

	var ts_row := HBoxContainer.new()
	ts_row.add_theme_constant_override("separation", 8)
	_time_scale_slider = HSlider.new()
	_time_scale_slider.min_value = 0.1
	_time_scale_slider.max_value = 2.0
	_time_scale_slider.step = 0.1
	_time_scale_slider.value = 1.0
	_time_scale_slider.custom_minimum_size = Vector2(160, 20)
	_time_scale_slider.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	_time_scale_slider.value_changed.connect(_on_time_scale_changed)
	ts_row.add_child(_time_scale_slider)

	_time_scale_label = Label.new()
	_time_scale_label.text = "1.0x"
	_time_scale_label.add_theme_color_override("font_color", UI_TEXT)
	_time_scale_label.add_theme_font_size_override("font_size", 12)
	_time_scale_label.custom_minimum_size = Vector2(35, 0)
	ts_row.add_child(_time_scale_label)
	_vbox.add_child(ts_row)
	_add_separator()


func _build_action_buttons() -> void:
	var actions_row := HBoxContainer.new()
	actions_row.add_theme_constant_override("separation", 4)

	var reset_btn := Button.new()
	reset_btn.text = "Reset Boss"
	reset_btn.custom_minimum_size = Vector2(0, 28)
	reset_btn.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	reset_btn.pressed.connect(_on_reset_pressed)
	_style_button(reset_btn)
	actions_row.add_child(reset_btn)

	var reload_btn := Button.new()
	reload_btn.text = "Reload YAML"
	reload_btn.custom_minimum_size = Vector2(0, 28)
	reload_btn.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	reload_btn.pressed.connect(_on_reload_pressed)
	_style_button(reload_btn)
	actions_row.add_child(reload_btn)

	_vbox.add_child(actions_row)


func _on_debug_info(def_name: String, abilities: PackedStringArray) -> void:
	_def_name = def_name
	_abilities = abilities
	_repeat_label.text = "Boss: %s" % def_name
	_rebuild_ability_buttons()


func _rebuild_ability_buttons() -> void:
	for child in _ability_container.get_children():
		child.queue_free()
	for i in range(_abilities.size()):
		var ability_id: String = _abilities[i]
		var row := HBoxContainer.new()
		row.add_theme_constant_override("separation", 4)

		var commit_btn := Button.new()
		commit_btn.text = ability_id
		commit_btn.custom_minimum_size = Vector2(0, 26)
		commit_btn.size_flags_horizontal = Control.SIZE_EXPAND_FILL
		commit_btn.pressed.connect(_on_force_commit.bind(ability_id))
		_style_button(commit_btn)
		row.add_child(commit_btn)

		var repeat_btn := Button.new()
		repeat_btn.text = "R"
		repeat_btn.tooltip_text = "Toggle repeat"
		repeat_btn.custom_minimum_size = Vector2(30, 26)
		repeat_btn.pressed.connect(_on_repeat_toggled.bind(ability_id))
		_style_button(repeat_btn)
		row.add_child(repeat_btn)

		if i < 9:
			var hint := Label.new()
			hint.text = "Ctrl+%d" % (i + 1)
			hint.add_theme_color_override("font_color", UI_TEXT_MUTED)
			hint.add_theme_font_size_override("font_size", 10)
			hint.custom_minimum_size = Vector2(50, 0)
			row.add_child(hint)

		_ability_container.add_child(row)


func _input(event: InputEvent) -> void:
	if not visible:
		return
	if event is InputEventKey and event.pressed and not event.echo:
		# Ctrl+1-9 = force-commit ability by index
		if event.ctrl_pressed:
			var idx: int = event.keycode - KEY_1
			if idx >= 0 and idx < _abilities.size():
				_on_force_commit(_abilities[idx])
				get_viewport().set_input_as_handled()


func _on_force_commit(ability_id: String) -> void:
	NetworkManager.debug.send_force_commit(ability_id)


func _on_repeat_toggled(ability_id: String) -> void:
	if _repeat_ability == ability_id:
		_repeat_ability = ""
		NetworkManager.debug.send_repeat_ability("")
		_repeat_label.text = "Boss: %s" % _def_name
	else:
		_repeat_ability = ability_id
		NetworkManager.debug.send_repeat_ability(ability_id)
		_repeat_label.text = "Boss: %s [REPEAT: %s]" % [_def_name, ability_id]


func _on_phase_pressed(phase: int) -> void:
	NetworkManager.debug.send_set_phase(phase)


func _on_god_mode_toggled(enabled: bool) -> void:
	_god_mode = enabled
	NetworkManager.debug.send_god_mode(enabled)


func _on_time_scale_changed(value: float) -> void:
	_time_scale_label.text = "%.1fx" % value
	NetworkManager.debug.send_time_scale(value)


func _on_reset_pressed() -> void:
	NetworkManager.debug.send_reset_boss()
	_repeat_ability = ""
	_repeat_label.text = "Boss: %s" % _def_name


func _on_reload_pressed() -> void:
	NetworkManager.debug.send_reload_yaml()


func _add_separator() -> void:
	var sep := HSeparator.new()
	sep.add_theme_color_override("separator", UI_BORDER)
	_vbox.add_child(sep)


func _style_button(btn: Button) -> void:
	var normal := StyleBoxFlat.new()
	normal.bg_color = Color(0.08, 0.1, 0.15, 0.95)
	normal.border_color = UI_BORDER
	normal.set_border_width_all(1)
	normal.set_corner_radius_all(3)
	normal.set_content_margin_all(4)
	btn.add_theme_stylebox_override("normal", normal)

	var hover := normal.duplicate()
	hover.bg_color = Color(0.12, 0.15, 0.22, 0.95)
	hover.border_color = UI_ACCENT
	btn.add_theme_stylebox_override("hover", hover)

	var pressed := normal.duplicate()
	pressed.bg_color = Color(0.15, 0.2, 0.3, 0.95)
	btn.add_theme_stylebox_override("pressed", pressed)

	btn.add_theme_color_override("font_color", UI_TEXT)
	btn.add_theme_font_size_override("font_size", 11)
