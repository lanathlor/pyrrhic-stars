extends CanvasLayer
## Fight replay browser — lists recorded encounters, lets the user select one
## and launch the replay scene.

signal replay_selected(replay: Variant)
signal browser_closed

const _COL_WIDTHS: Array[float] = [160.0, 150.0, 80.0, 100.0, 200.0]
const _COL_NAMES: Array[String] = ["Encounter", "Date", "Duration", "Outcome", "Players"]

var _api: Node
var _list_container: VBoxContainer
var _watch_btn: Button
var _loading_label: Label
var _selected_instance_id: String = ""
var _rows: Array[Control] = []
var _all_data: Array = []

# Filter controls
var _encounter_filter: LineEdit
var _outcome_filter: OptionButton
var _player_filter: LineEdit


func _ready() -> void:
	layer = 5

	var api_script := load("res://scripts/replay/replay_api.gd")
	_api = Node.new()
	_api.set_script(api_script)
	add_child(_api)
	_api.instances_loaded.connect(_on_instances_loaded)
	_api.replay_loaded.connect(_on_replay_loaded)

	_build_ui()
	_api.fetch_instances()


func _build_ui() -> void:
	var bg := ColorRect.new()
	bg.color = Color(0.0, 0.0, 0.0, 0.6)
	bg.set_anchors_preset(Control.PRESET_FULL_RECT)
	add_child(bg)

	var panel := PanelContainer.new()
	panel.set_anchors_preset(Control.PRESET_CENTER)
	panel.custom_minimum_size = Vector2(850, 550)
	panel.offset_left = -425
	panel.offset_right = 425
	panel.offset_top = -275
	panel.offset_bottom = 275
	add_child(panel)

	var vbox := VBoxContainer.new()
	vbox.add_theme_constant_override("separation", 8)
	panel.add_child(vbox)

	var title := Label.new()
	title.text = "Fight Replays"
	title.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	title.add_theme_font_size_override("font_size", 22)
	vbox.add_child(title)

	_build_filter_bar(vbox)
	_build_header_row(vbox)
	_build_scroll_area(vbox)
	_build_bottom_buttons(vbox)


func _build_filter_bar(vbox: VBoxContainer) -> void:
	var filter_bar := HBoxContainer.new()
	filter_bar.add_theme_constant_override("separation", 12)
	vbox.add_child(filter_bar)

	var enc_label := Label.new()
	enc_label.text = "Encounter:"
	enc_label.add_theme_font_size_override("font_size", 13)
	filter_bar.add_child(enc_label)

	_encounter_filter = LineEdit.new()
	_encounter_filter.placeholder_text = "Search..."
	_encounter_filter.custom_minimum_size.x = 150
	_encounter_filter.text_changed.connect(func(_t: String) -> void: _apply_filters())
	filter_bar.add_child(_encounter_filter)

	var out_label := Label.new()
	out_label.text = "Outcome:"
	out_label.add_theme_font_size_override("font_size", 13)
	filter_bar.add_child(out_label)

	_outcome_filter = OptionButton.new()
	_outcome_filter.add_item("All", 0)
	_outcome_filter.add_item("Player Win", 1)
	_outcome_filter.add_item("Boss Win", 2)
	_outcome_filter.add_item("Timeout", 3)
	_outcome_filter.custom_minimum_size.x = 120
	_outcome_filter.item_selected.connect(func(_i: int) -> void: _apply_filters())
	filter_bar.add_child(_outcome_filter)

	var player_label := Label.new()
	player_label.text = "Player:"
	player_label.add_theme_font_size_override("font_size", 13)
	filter_bar.add_child(player_label)

	_player_filter = LineEdit.new()
	_player_filter.placeholder_text = "Name..."
	_player_filter.custom_minimum_size.x = 120
	_player_filter.text_changed.connect(func(_t: String) -> void: _apply_filters())
	filter_bar.add_child(_player_filter)


func _build_header_row(vbox: VBoxContainer) -> void:
	var header := _make_row_container()
	vbox.add_child(header)
	for i in _COL_NAMES.size():
		var lbl := Label.new()
		lbl.text = _COL_NAMES[i]
		lbl.custom_minimum_size.x = _COL_WIDTHS[i]
		lbl.size_flags_horizontal = Control.SIZE_EXPAND_FILL if i == _COL_NAMES.size() - 1 else 0
		lbl.add_theme_font_size_override("font_size", 14)
		lbl.modulate = Color(0.7, 0.7, 0.7)
		header.add_child(lbl)


func _build_scroll_area(vbox: VBoxContainer) -> void:
	var scroll := ScrollContainer.new()
	scroll.size_flags_vertical = Control.SIZE_EXPAND_FILL
	vbox.add_child(scroll)

	_list_container = VBoxContainer.new()
	_list_container.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	_list_container.add_theme_constant_override("separation", 2)
	scroll.add_child(_list_container)

	_loading_label = Label.new()
	_loading_label.text = "Loading..."
	_loading_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	_list_container.add_child(_loading_label)


func _build_bottom_buttons(vbox: VBoxContainer) -> void:
	var bottom := HBoxContainer.new()
	bottom.alignment = BoxContainer.ALIGNMENT_CENTER
	bottom.add_theme_constant_override("separation", 12)
	vbox.add_child(bottom)

	var back_btn := Button.new()
	back_btn.text = "Back"
	back_btn.custom_minimum_size.x = 100
	back_btn.pressed.connect(func() -> void: browser_closed.emit())
	bottom.add_child(back_btn)

	var refresh_btn := Button.new()
	refresh_btn.text = "Refresh"
	refresh_btn.custom_minimum_size.x = 100
	refresh_btn.pressed.connect(
		func() -> void:
			_loading_label.visible = true
			_loading_label.text = "Loading..."
			_api.fetch_instances()
	)
	bottom.add_child(refresh_btn)

	_watch_btn = Button.new()
	_watch_btn.text = "Watch Replay"
	_watch_btn.custom_minimum_size.x = 140
	_watch_btn.disabled = true
	_watch_btn.pressed.connect(_on_watch_pressed)
	bottom.add_child(_watch_btn)


func _make_row_container() -> HBoxContainer:
	var hbox := HBoxContainer.new()
	hbox.add_theme_constant_override("separation", 8)
	return hbox


func _on_instances_loaded(data: Array) -> void:
	_all_data = data
	_apply_filters()


func _apply_filters() -> void:
	# Clear existing rows
	for row in _rows:
		if is_instance_valid(row):
			row.queue_free()
	_rows.clear()
	_selected_instance_id = ""
	_watch_btn.disabled = true

	var enc_query: String = _encounter_filter.text.strip_edges().to_lower()
	var outcome_idx: int = _outcome_filter.selected
	var outcome_values: Array[String] = ["", "player_win", "boss_win", "timeout"]
	var outcome_query: String = (
		outcome_values[outcome_idx] if outcome_idx < outcome_values.size() else ""
	)
	var player_query: String = _player_filter.text.strip_edges().to_lower()

	var filtered: Array = []
	for inst in _all_data:
		if not inst is Dictionary:
			continue
		if enc_query != "":
			var enc: String = ReplayUtils.safe_str(inst, "encounter_id", "").to_lower()
			if enc.find(enc_query) == -1:
				continue
		if outcome_query != "":
			var out: String = ReplayUtils.safe_str(inst, "outcome", "")
			if out != outcome_query:
				continue
		if player_query != "":
			if not _has_matching_player(inst, player_query):
				continue
		filtered.append(inst)

	_loading_label.visible = filtered.is_empty()
	if filtered.is_empty():
		if _all_data.is_empty():
			_loading_label.text = "No recorded fights found."
		else:
			_loading_label.text = "No fights match the current filters."
		return

	for inst in filtered:
		var row := _create_row(inst)
		_list_container.add_child(row)
		_rows.append(row)


func _create_row(inst: Dictionary) -> Button:
	var btn := Button.new()
	btn.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	btn.custom_minimum_size.y = 32

	var hbox := _make_row_container()
	hbox.set_anchors_preset(Control.PRESET_FULL_RECT)
	btn.add_child(hbox)

	var encounter: String = ReplayUtils.safe_str(inst, "encounter_id", "unknown")
	var started: String = ReplayUtils.safe_str(inst, "started_at", "")
	if started.length() > 16:
		started = started.substr(0, 16).replace("T", " ")

	var dur_ms: int = ReplayUtils.safe_int(inst, "duration_ms", 0)
	var dur_secs: int = dur_ms / 1000
	var duration: String = "%d:%02d" % [dur_secs / 60, dur_secs % 60]

	var outcome: String = ReplayUtils.safe_str(inst, "outcome", "")
	var outcome_display: String = outcome.replace("_", " ")

	var player_text: String = _format_players(inst)

	var values: Array[String] = [encounter, started, duration, outcome_display, player_text]
	for i in values.size():
		var lbl := Label.new()
		lbl.text = values[i]
		lbl.custom_minimum_size.x = _COL_WIDTHS[i]
		lbl.size_flags_horizontal = Control.SIZE_EXPAND_FILL if i == values.size() - 1 else 0
		lbl.clip_text = true
		lbl.add_theme_font_size_override("font_size", 14)
		lbl.vertical_alignment = VERTICAL_ALIGNMENT_CENTER
		hbox.add_child(lbl)

	var iid: String = ReplayUtils.safe_str(inst, "instance_id", "")
	btn.pressed.connect(func() -> void: _select_instance(iid, btn))

	return btn


func _select_instance(instance_id: String, btn: Button) -> void:
	_selected_instance_id = instance_id
	_watch_btn.disabled = false
	for row in _rows:
		if is_instance_valid(row):
			row.modulate = Color.WHITE
	btn.modulate = Color(0.5, 0.8, 1.0)


func _on_watch_pressed() -> void:
	if _selected_instance_id == "":
		return
	_watch_btn.disabled = true
	_watch_btn.text = "Loading..."
	_api.fetch_replay(_selected_instance_id)


func _on_replay_loaded(replay: Variant) -> void:
	_watch_btn.text = "Watch Replay"
	_watch_btn.disabled = false
	if replay == null:
		_loading_label.visible = true
		_loading_label.text = "Failed to load replay data."
		return
	replay_selected.emit(replay)


static func _get_players(inst: Dictionary) -> Array:
	var participants: Array = inst.get("participants") if inst.get("participants") is Array else []
	var players: Array = []
	for p in participants:
		if p is Dictionary:
			var pclass: String = ReplayUtils.safe_str(p, "class", "")
			if pclass == "enemy":
				continue
			players.append(p)
	return players


static func _format_players(inst: Dictionary) -> String:
	var players := _get_players(inst)
	if players.is_empty():
		return "-"
	var names: PackedStringArray = []
	for p in players:
		var pname: String = ReplayUtils.safe_str(p, "name", "?")
		var pclass: String = ReplayUtils.safe_str(p, "class", "")
		if pclass != "":
			names.append("%s (%s)" % [pname, pclass])
		else:
			names.append(pname)
	return ", ".join(names) if names.size() > 0 else "-"


static func _has_matching_player(inst: Dictionary, query: String) -> bool:
	var players := _get_players(inst)
	for p in players:
		var pname: String = ReplayUtils.safe_str(p, "name", "").to_lower()
		if pname.find(query) != -1:
			return true
	return false
