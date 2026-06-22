extends CanvasLayer

## Settings overlay (Graphics / Audio / Controls). Code-built like the merchant
## and overflux panels. Reads from and writes to SettingsManager, which owns
## persistence and applies every change. Open from the pause menu or main menu.

signal closed

var ui_ctrl: Node = null

var _root: Control
var _panel: PanelContainer
var _capturing_action: String = ""
var _capturing_group: String = ""
var _capturing_button: Button = null
# action -> Array[Button] (same action can appear under several class tabs)
var _rebind_buttons: Dictionary = {}

# Control references built in _build_ui.
var _res_option: OptionButton
var _mode_option: OptionButton
var _quality_option: OptionButton
var _vsync_check: CheckButton
var _volume_sliders: Dictionary = {}  # key -> HSlider
var _volume_pct: Dictionary = {}  # key -> Label


func _ready() -> void:
	layer = 30
	process_mode = Node.PROCESS_MODE_ALWAYS
	visible = false
	if ui_ctrl == null:
		ui_ctrl = get_parent().get_node_or_null("UIController")
	_build()
	SettingsManager.settings_changed.connect(_refresh_if_visible)


func open() -> void:
	_refresh_all()
	visible = true


func close() -> void:
	_cancel_capture()
	visible = false
	closed.emit()


# =============================================================================
# Construction
# =============================================================================


func _build() -> void:
	var bg := ColorRect.new()
	bg.color = Color(0, 0, 0, 0.72)
	bg.set_anchors_preset(Control.PRESET_FULL_RECT)
	bg.mouse_filter = Control.MOUSE_FILTER_STOP
	add_child(bg)

	_root = CenterContainer.new()
	_root.set_anchors_preset(Control.PRESET_FULL_RECT)
	add_child(_root)

	_panel = PanelContainer.new()
	_panel.custom_minimum_size = Vector2(580, 500)
	_root.add_child(_panel)
	if ui_ctrl:
		ui_ctrl.apply_panel_style(_panel, false, 16)

	var vbox := VBoxContainer.new()
	vbox.add_theme_constant_override("separation", 12)
	_panel.add_child(vbox)

	var title := Label.new()
	title.text = "Settings"
	title.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	vbox.add_child(title)
	if ui_ctrl:
		ui_ctrl.apply_overlay_label(title, 24, ui_ctrl.UI_TEXT)

	var tabs := TabContainer.new()
	tabs.size_flags_vertical = Control.SIZE_EXPAND_FILL
	tabs.custom_minimum_size = Vector2(0, 380)
	vbox.add_child(tabs)
	tabs.add_child(_build_graphics_tab())
	tabs.add_child(_build_audio_tab())
	tabs.add_child(_build_controls_tab())

	var footer := HBoxContainer.new()
	footer.alignment = BoxContainer.ALIGNMENT_END
	vbox.add_child(footer)
	var close_btn := Button.new()
	close_btn.text = "Close"
	close_btn.custom_minimum_size = Vector2(120, 38)
	close_btn.pressed.connect(close)
	footer.add_child(close_btn)
	if ui_ctrl:
		ui_ctrl.apply_button_style(close_btn)


func _build_graphics_tab() -> Control:
	var page := VBoxContainer.new()
	page.name = "Graphics"
	page.add_theme_constant_override("separation", 10)

	_res_option = OptionButton.new()
	for r in SettingsManager.RESOLUTIONS:
		_res_option.add_item(r)
	_res_option.item_selected.connect(
		func(i): SettingsManager.set_value("graphics", "resolution", SettingsManager.RESOLUTIONS[i])
	)
	page.add_child(_row("Resolution", _res_option))

	_mode_option = OptionButton.new()
	for m in SettingsManager.DISPLAY_MODES:
		_mode_option.add_item(m)
	_mode_option.item_selected.connect(
		func(i): SettingsManager.set_value("graphics", "display_mode", i)
	)
	page.add_child(_row("Display Mode", _mode_option))

	_quality_option = OptionButton.new()
	for q in SettingsManager.QUALITY_LEVELS:
		_quality_option.add_item(q)
	_quality_option.item_selected.connect(
		func(i): SettingsManager.set_value("graphics", "quality", i)
	)
	page.add_child(_row("Quality", _quality_option))

	_vsync_check = CheckButton.new()
	_vsync_check.toggled.connect(func(on): SettingsManager.set_value("graphics", "vsync", on))
	page.add_child(_row("VSync", _vsync_check))

	return page


func _build_audio_tab() -> Control:
	var page := VBoxContainer.new()
	page.name = "Audio"
	page.add_theme_constant_override("separation", 14)
	_add_volume_row(page, "Master", "master")
	_add_volume_row(page, "Music", "music")
	_add_volume_row(page, "SFX", "sfx")
	_add_volume_row(page, "Ambiance", "ambiance")
	return page


func _build_controls_tab() -> Control:
	var page := VBoxContainer.new()
	page.name = "Controls"
	page.add_theme_constant_override("separation", 8)

	# Nested tabs: Core (universal) + one per class, so it is clear what a key does
	# and which class a rebind affects.
	var subtabs := TabContainer.new()
	subtabs.size_flags_vertical = Control.SIZE_EXPAND_FILL
	subtabs.custom_minimum_size = Vector2(0, 320)
	page.add_child(subtabs)
	for group in SettingsManager.keybind_groups():
		subtabs.add_child(_build_keybind_group(group))

	var reset_btn := Button.new()
	reset_btn.text = "Reset All to Defaults"
	reset_btn.custom_minimum_size = Vector2(0, 34)
	reset_btn.pressed.connect(
		func():
			SettingsManager.reset_keybinds()
			_refresh_keybind_labels()
	)
	page.add_child(reset_btn)
	if ui_ctrl:
		ui_ctrl.apply_button_style(reset_btn)

	return page


func _build_keybind_group(group: Dictionary) -> Control:
	var scroll := ScrollContainer.new()
	scroll.name = group.name
	scroll.size_flags_vertical = Control.SIZE_EXPAND_FILL

	var list := VBoxContainer.new()
	list.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	list.add_theme_constant_override("separation", 4)
	scroll.add_child(list)

	for bind in group.binds:
		var action: String = bind.action
		var row := HBoxContainer.new()
		row.size_flags_horizontal = Control.SIZE_EXPAND_FILL
		var label := Label.new()
		label.text = bind.label
		label.size_flags_horizontal = Control.SIZE_EXPAND_FILL
		row.add_child(label)
		if ui_ctrl:
			ui_ctrl.apply_overlay_label(label, 15, ui_ctrl.UI_TEXT_MUTED)
		var btn := Button.new()
		btn.custom_minimum_size = Vector2(140, 32)
		btn.pressed.connect(_begin_capture.bind(action, String(group.name), btn))
		row.add_child(btn)
		if ui_ctrl:
			ui_ctrl.apply_button_style(btn)
		if not _rebind_buttons.has(action):
			_rebind_buttons[action] = []
		_rebind_buttons[action].append(btn)
		list.add_child(row)

	return scroll


func _row(label_text: String, control: Control) -> HBoxContainer:
	var row := HBoxContainer.new()
	row.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	var label := Label.new()
	label.text = label_text
	label.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	row.add_child(label)
	control.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	control.custom_minimum_size = Vector2(220, 32)
	row.add_child(control)
	if ui_ctrl:
		ui_ctrl.apply_overlay_label(label, 15, ui_ctrl.UI_TEXT_MUTED)
		if control is Button:
			ui_ctrl.apply_button_style(control)
	return row


func _add_volume_row(page: VBoxContainer, label_text: String, key: String) -> void:
	var row := HBoxContainer.new()
	row.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	var label := Label.new()
	label.text = label_text
	label.custom_minimum_size = Vector2(110, 0)
	row.add_child(label)
	var slider := HSlider.new()
	slider.min_value = 0.0
	slider.max_value = 1.0
	slider.step = 0.05
	slider.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	slider.custom_minimum_size = Vector2(260, 24)
	row.add_child(slider)
	var pct := Label.new()
	pct.custom_minimum_size = Vector2(48, 0)
	pct.horizontal_alignment = HORIZONTAL_ALIGNMENT_RIGHT
	row.add_child(pct)
	slider.value_changed.connect(
		func(v):
			SettingsManager.set_value("audio", key, v)
			pct.text = "%d%%" % roundi(v * 100.0)
	)
	page.add_child(row)
	if ui_ctrl:
		ui_ctrl.apply_overlay_label(label, 15, ui_ctrl.UI_TEXT_MUTED)
		ui_ctrl.apply_overlay_label(pct, 14, ui_ctrl.UI_TEXT_DIM)
	_volume_sliders[key] = slider
	_volume_pct[key] = pct


# =============================================================================
# Refresh from SettingsManager
# =============================================================================


func _refresh_if_visible() -> void:
	if visible:
		_refresh_all()


func _refresh_all() -> void:
	var cur_res: String = SettingsManager.get_value("graphics", "resolution", "1920x1080")
	_set_option(_res_option, SettingsManager.RESOLUTIONS.find(cur_res))
	_set_option(_mode_option, int(SettingsManager.get_value("graphics", "display_mode", 0)))
	_set_option(_quality_option, int(SettingsManager.get_value("graphics", "quality", 2)))
	_vsync_check.set_block_signals(true)
	_vsync_check.button_pressed = bool(SettingsManager.get_value("graphics", "vsync", true))
	_vsync_check.set_block_signals(false)
	for key in _volume_sliders.keys():
		var v := float(SettingsManager.get_value("audio", key, 1.0))
		var slider: HSlider = _volume_sliders[key]
		slider.set_block_signals(true)
		slider.value = v
		slider.set_block_signals(false)
		_volume_pct[key].text = "%d%%" % roundi(v * 100.0)
	_refresh_keybind_labels()


func _set_option(opt: OptionButton, idx: int) -> void:
	if opt == null or idx < 0:
		return
	opt.set_block_signals(true)
	opt.select(idx)
	opt.set_block_signals(false)


func _refresh_keybind_labels() -> void:
	for action in _rebind_buttons.keys():
		var labeltext := SettingsManager.get_action_label(action)
		for btn in _rebind_buttons[action]:
			if btn == _capturing_button:
				continue
			btn.text = labeltext


# =============================================================================
# Key capture
# =============================================================================


func _begin_capture(action: String, group_name: String, btn: Button) -> void:
	_cancel_capture()
	_capturing_action = action
	_capturing_group = group_name
	_capturing_button = btn
	btn.text = "Press a key..."


func _cancel_capture() -> void:
	_capturing_action = ""
	_capturing_group = ""
	_capturing_button = null
	_refresh_keybind_labels()


func _input(event: InputEvent) -> void:
	if not visible:
		return
	if _capturing_action == "":
		# Esc closes the panel when not capturing.
		var is_esc: bool = (
			event is InputEventKey
			and event.pressed
			and not event.echo
			and event.physical_keycode == KEY_ESCAPE
		)
		if is_esc:
			get_viewport().set_input_as_handled()
			close()
		return
	# Capturing a new bind: accept a key or a mouse button (Esc cancels).
	if event is InputEventKey and event.pressed and not event.echo:
		get_viewport().set_input_as_handled()
		if (event as InputEventKey).physical_keycode == KEY_ESCAPE:
			_cancel_capture()
			return
		_apply_capture((event as InputEventKey).physical_keycode)
	elif event is InputEventMouseButton and event.pressed:
		get_viewport().set_input_as_handled()
		_apply_capture(-(event as InputEventMouseButton).button_index)


## Commits a captured bind for the action being rebound. Prevents a double-bind:
## if another action the same class reads already uses this code, unbind it first
## so no input triggers two actions for one class.
func _apply_capture(code: int) -> void:
	var action := _capturing_action
	var group_name := _capturing_group
	_capturing_action = ""
	_capturing_group = ""
	_capturing_button = null
	for other in SettingsManager.conflict_scope_actions(group_name):
		if other != action and SettingsManager.get_keybind(other) == code:
			SettingsManager.set_keybind(other, 0)
	SettingsManager.set_keybind(action, code)
	_refresh_keybind_labels()
