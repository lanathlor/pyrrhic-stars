extends CanvasLayer

## Instance lobby ready-up panel. Shows player list with ready states, a ready
## button, spec change button, and countdown display. Code-built (no .tscn).

signal ready_toggled
signal spec_change_requested

const UI_SURFACE := Color(0.035, 0.045, 0.065, 0.88)
const UI_BORDER := Color(0.22, 0.24, 0.30, 0.7)
const UI_TEXT := Color(0.9, 0.93, 0.98, 0.96)
const UI_TEXT_MUTED := Color(0.6, 0.66, 0.75, 0.95)
const UI_READY_GREEN := Color(0.35, 0.78, 0.35, 0.95)
const UI_NOT_READY := Color(0.5, 0.5, 0.55, 0.8)
const UI_COUNTDOWN := Color(0.92, 0.82, 0.35, 0.98)

var _panel: PanelContainer
var _player_rows: VBoxContainer
var _ready_btn: Button
var _spec_btn: Button
var _status_label: Label
var _countdown_label: Label
var _local_ready: bool = false


func _ready() -> void:
	process_mode = Node.PROCESS_MODE_ALWAYS
	layer = 15
	visible = false
	_build_ui()


func _build_ui() -> void:
	# Anchor panel to right side of screen, vertically centered
	var anchor := Control.new()
	anchor.set_anchors_preset(Control.PRESET_CENTER_RIGHT)
	anchor.anchor_left = 1.0
	anchor.anchor_right = 1.0
	anchor.anchor_top = 0.5
	anchor.anchor_bottom = 0.5
	anchor.offset_left = -320
	anchor.offset_right = -20
	anchor.offset_top = -220
	anchor.offset_bottom = 220
	anchor.mouse_filter = Control.MOUSE_FILTER_IGNORE
	add_child(anchor)

	_panel = PanelContainer.new()
	var style := StyleBoxFlat.new()
	style.bg_color = UI_SURFACE
	style.border_color = UI_BORDER
	style.border_width_top = 1
	style.border_width_bottom = 1
	style.border_width_left = 1
	style.border_width_right = 1
	style.corner_radius_top_left = 4
	style.corner_radius_top_right = 4
	style.corner_radius_bottom_left = 4
	style.corner_radius_bottom_right = 4
	style.content_margin_left = 16
	style.content_margin_right = 16
	style.content_margin_top = 14
	style.content_margin_bottom = 14
	_panel.add_theme_stylebox_override("panel", style)
	_panel.set_anchors_preset(Control.PRESET_FULL_RECT)
	anchor.add_child(_panel)

	var vbox := VBoxContainer.new()
	vbox.add_theme_constant_override("separation", 10)
	_panel.add_child(vbox)

	# Title
	var title := Label.new()
	title.text = "INSTANCE LOBBY"
	title.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	title.add_theme_font_size_override("font_size", 16)
	title.add_theme_color_override("font_color", UI_TEXT)
	vbox.add_child(title)

	# Separator
	var sep := HSeparator.new()
	sep.add_theme_constant_override("separation", 4)
	vbox.add_child(sep)

	# Player list container
	_player_rows = VBoxContainer.new()
	_player_rows.add_theme_constant_override("separation", 6)
	_player_rows.size_flags_vertical = Control.SIZE_EXPAND_FILL
	vbox.add_child(_player_rows)

	# Status label
	_status_label = Label.new()
	_status_label.text = "Waiting for players..."
	_status_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	_status_label.add_theme_font_size_override("font_size", 13)
	_status_label.add_theme_color_override("font_color", UI_TEXT_MUTED)
	vbox.add_child(_status_label)

	# Countdown label (hidden by default)
	_countdown_label = Label.new()
	_countdown_label.text = ""
	_countdown_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	_countdown_label.add_theme_font_size_override("font_size", 22)
	_countdown_label.add_theme_color_override("font_color", UI_COUNTDOWN)
	_countdown_label.visible = false
	vbox.add_child(_countdown_label)

	# Separator
	var sep2 := HSeparator.new()
	sep2.add_theme_constant_override("separation", 4)
	vbox.add_child(sep2)

	# Button row
	var btn_row := HBoxContainer.new()
	btn_row.add_theme_constant_override("separation", 8)
	btn_row.alignment = BoxContainer.ALIGNMENT_CENTER
	vbox.add_child(btn_row)

	# Spec button
	_spec_btn = Button.new()
	_spec_btn.text = "SPEC [N]"
	_spec_btn.custom_minimum_size = Vector2(100, 34)
	_spec_btn.pressed.connect(func(): spec_change_requested.emit())
	btn_row.add_child(_spec_btn)

	# Ready button
	_ready_btn = Button.new()
	_ready_btn.text = "READY [R]"
	_ready_btn.custom_minimum_size = Vector2(140, 34)
	_ready_btn.pressed.connect(_on_ready_pressed)
	btn_row.add_child(_ready_btn)


func _on_ready_pressed() -> void:
	ready_toggled.emit()


func update_lobby_state(data: Dictionary) -> void:
	var phase: int = data.get("phase", 0)
	var countdown: int = data.get("countdown", 0)
	var players: Array = data.get("players", [])

	var my_id: int = NetworkManager.get_my_id()

	# Update local ready state from server
	for p in players:
		if p.peer_id == my_id:
			_local_ready = p.is_ready
			break

	# Update ready button text
	_ready_btn.text = "CANCEL [R]" if _local_ready else "READY [R]"

	# Rebuild player rows
	for child in _player_rows.get_children():
		child.queue_free()

	var ready_count := 0
	for p in players:
		if p.is_ready:
			ready_count += 1
		_add_player_row(p, p.peer_id == my_id)

	# Status label
	if phase == 0:
		_status_label.visible = true
		_status_label.text = "%d / %d ready" % [ready_count, players.size()]
		_countdown_label.visible = false
	else:
		_status_label.visible = false
		_countdown_label.visible = true
		_countdown_label.text = "Starting in %d..." % countdown


func _add_player_row(player: Dictionary, is_local: bool) -> void:
	var row := HBoxContainer.new()
	row.add_theme_constant_override("separation", 8)
	_player_rows.add_child(row)

	# Ready indicator
	var indicator := Label.new()
	indicator.custom_minimum_size = Vector2(20, 0)
	indicator.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	if player.is_ready:
		indicator.text = "+"
		indicator.add_theme_color_override("font_color", UI_READY_GREEN)
	else:
		indicator.text = "-"
		indicator.add_theme_color_override("font_color", UI_NOT_READY)
	indicator.add_theme_font_size_override("font_size", 16)
	row.add_child(indicator)

	# Username
	var name_label := Label.new()
	var display_name: String = player.username
	if is_local:
		display_name += " (you)"
	name_label.text = display_name
	name_label.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	name_label.add_theme_font_size_override("font_size", 14)
	name_label.add_theme_color_override("font_color", UI_TEXT if not is_local else UI_COUNTDOWN)
	row.add_child(name_label)

	# Class / Spec
	var spec_label := Label.new()
	var spec_text: String = player.class_name
	if player.spec_name != "":
		spec_text += " / %s" % player.spec_name
	spec_label.text = spec_text
	spec_label.add_theme_font_size_override("font_size", 12)
	spec_label.add_theme_color_override("font_color", UI_TEXT_MUTED)
	row.add_child(spec_label)
