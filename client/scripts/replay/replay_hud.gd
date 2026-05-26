extends CanvasLayer
## Playback controls overlay for the replay scene.

signal play_toggled(playing: bool)
signal speed_changed(speed: float)
signal frame_seeked(frame: int)
signal back_pressed

const Draw := preload("res://scripts/replay/replay_hud_draw.gd")
const SPEEDS := [0.25, 0.5, 1.0, 2.0, 4.0]

var _speed_index: int = 2  # start at 1.0x
var _playing: bool = true
var _frame_count: int = 0
var _tick_rate: int = 20
var _dragging: bool = false
var _replay: Variant = null
var _damage_totals: Dictionary = {}  # entity_id(String) -> float
var _current_tick: int = 0
var _boss_health: float = 0.0
var _boss_max_health: float = 2000.0
var _boss_phase: int = 1
var _boss_name: String = ""
var _boss_visible: bool = false

var _time_label: Label
var _total_label: Label
var _slider: HSlider
var _play_btn: Button
var _speed_label: Label
var _encounter_label: Label
var _status_overlay: Control
var _current_players: Array = []


func _ready() -> void:
	_build_ui()


func init(replay: Variant) -> void:
	_replay = replay
	_frame_count = replay.frame_count
	_tick_rate = replay.tick_rate
	_slider.max_value = maxf(replay.frame_count - 1, 0)
	_total_label.text = "/ " + replay.total_time_str()
	_encounter_label.text = replay.encounter_id + " - " + replay.outcome
	_boss_name = replay.encounter_id.replace("_", " ").capitalize()


func update_frame(frame: int) -> void:
	_current_tick = frame
	if not _dragging:
		_slider.value = frame
	var seconds: float = float(frame) / float(_tick_rate)
	var mins: int = int(seconds) / 60
	var secs: int = int(seconds) % 60
	_time_label.text = "%d:%02d" % [mins, secs]


func update_players(players: Array) -> void:
	_current_players = players.duplicate()
	_current_players.sort_custom(
		func(a: Dictionary, b: Dictionary) -> bool: return a["peer_id"] < b["peer_id"]
	)
	_status_overlay.queue_redraw()


func update_enemies(enemies: Array) -> void:
	_boss_visible = false
	for edata in enemies:
		if not edata.get("alive", false):
			continue
		_boss_visible = true
		_boss_health = edata.get("health", 0.0)
		_boss_max_health = edata.get("max_health", 2000.0)
		_boss_phase = edata.get("phase", 1) + 1  # server phase is 0-indexed
		var def_name: String = edata.get("def_name", "")
		if def_name != "":
			_boss_name = def_name.replace("_", " ").capitalize()
		break  # use first alive enemy as the boss
	_status_overlay.queue_redraw()


func _build_ui() -> void:
	layer = 10

	_status_overlay = Control.new()
	_status_overlay.name = "StatusOverlay"
	_status_overlay.set_anchors_preset(Control.PRESET_FULL_RECT)
	_status_overlay.mouse_filter = Control.MOUSE_FILTER_IGNORE
	_status_overlay.draw.connect(_draw_player_status)
	add_child(_status_overlay)

	var panel := PanelContainer.new()
	panel.set_anchors_preset(Control.PRESET_BOTTOM_WIDE)
	panel.offset_top = -80
	add_child(panel)

	var vbox := VBoxContainer.new()
	vbox.add_theme_constant_override("separation", 4)
	panel.add_child(vbox)

	_encounter_label = Label.new()
	_encounter_label.text = ""
	_encounter_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	vbox.add_child(_encounter_label)

	_build_timeline(vbox)
	_build_controls(vbox)


func _build_timeline(vbox: VBoxContainer) -> void:
	var timeline := HBoxContainer.new()
	timeline.add_theme_constant_override("separation", 8)
	vbox.add_child(timeline)

	_time_label = Label.new()
	_time_label.text = "0:00"
	_time_label.custom_minimum_size.x = 50
	timeline.add_child(_time_label)

	_slider = HSlider.new()
	_slider.min_value = 0
	_slider.max_value = 100
	_slider.step = 1
	_slider.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	_slider.drag_started.connect(func() -> void: _dragging = true)
	_slider.drag_ended.connect(_on_slider_drag_ended)
	_slider.value_changed.connect(_on_slider_changed)
	timeline.add_child(_slider)

	_total_label = Label.new()
	_total_label.text = "/ 0:00"
	_total_label.custom_minimum_size.x = 60
	timeline.add_child(_total_label)


func _build_controls(vbox: VBoxContainer) -> void:
	var controls := HBoxContainer.new()
	controls.alignment = BoxContainer.ALIGNMENT_CENTER
	controls.add_theme_constant_override("separation", 6)
	vbox.add_child(controls)

	var step_back_1s := Button.new()
	step_back_1s.text = "<<"
	step_back_1s.pressed.connect(func() -> void: _seek_relative(-_tick_rate))
	controls.add_child(step_back_1s)

	var step_back := Button.new()
	step_back.text = "<"
	step_back.pressed.connect(func() -> void: _seek_relative(-1))
	controls.add_child(step_back)

	_play_btn = Button.new()
	_play_btn.text = "Pause"
	_play_btn.custom_minimum_size.x = 70
	_play_btn.pressed.connect(_toggle_play)
	controls.add_child(_play_btn)

	var step_fwd := Button.new()
	step_fwd.text = ">"
	step_fwd.pressed.connect(func() -> void: _seek_relative(1))
	controls.add_child(step_fwd)

	var step_fwd_1s := Button.new()
	step_fwd_1s.text = ">>"
	step_fwd_1s.pressed.connect(func() -> void: _seek_relative(_tick_rate))
	controls.add_child(step_fwd_1s)

	_speed_label = Label.new()
	_speed_label.text = "1.0x"
	_speed_label.custom_minimum_size.x = 40
	controls.add_child(_speed_label)

	var slower := Button.new()
	slower.text = "-"
	slower.pressed.connect(func() -> void: _change_speed(-1))
	controls.add_child(slower)

	var faster := Button.new()
	faster.text = "+"
	faster.pressed.connect(func() -> void: _change_speed(1))
	controls.add_child(faster)

	var back := Button.new()
	back.text = "Back"
	back.pressed.connect(func() -> void: back_pressed.emit())
	controls.add_child(back)


func _unhandled_key_input(event: InputEvent) -> void:
	if not event.is_pressed() or event.is_echo():
		return
	match (event as InputEventKey).keycode:
		KEY_SPACE:
			_toggle_play()
			get_viewport().set_input_as_handled()
		KEY_LEFT:
			_seek_relative(-1)
			get_viewport().set_input_as_handled()
		KEY_RIGHT:
			_seek_relative(1)
			get_viewport().set_input_as_handled()
		KEY_UP:
			_change_speed(1)
			get_viewport().set_input_as_handled()
		KEY_DOWN:
			_change_speed(-1)
			get_viewport().set_input_as_handled()
		KEY_HOME:
			frame_seeked.emit(0)
			get_viewport().set_input_as_handled()
		KEY_END:
			frame_seeked.emit(_frame_count - 1)
			get_viewport().set_input_as_handled()


func _toggle_play() -> void:
	_playing = not _playing
	_play_btn.text = "Pause" if _playing else "Play"
	play_toggled.emit(_playing)


func _change_speed(direction: int) -> void:
	_speed_index = clampi(_speed_index + direction, 0, SPEEDS.size() - 1)
	var spd: float = SPEEDS[_speed_index]
	_speed_label.text = "%sx" % str(spd)
	speed_changed.emit(spd)


func _seek_relative(delta: int) -> void:
	var target: int = clampi(int(_slider.value) + delta, 0, _frame_count - 1)
	frame_seeked.emit(target)


func _on_slider_changed(value: float) -> void:
	if _dragging:
		frame_seeked.emit(int(value))


func _on_slider_drag_ended(_value_changed: bool) -> void:
	_dragging = false
	frame_seeked.emit(int(_slider.value))


func record_event(ev: Dictionary) -> void:
	var event_type: int = ev.get("event_type", 0)
	if event_type != 1 and event_type != 5:  # Damage, BuffTick
		return
	var amount: float = ev.get("amount", 0.0)
	if amount <= 0.0:
		return
	var target: String = ev.get("target", "")
	if not target.begins_with("enemy_"):
		return
	var source: String = ev.get("source", "")
	_damage_totals[source] = _damage_totals.get(source, 0.0) + amount
	_status_overlay.queue_redraw()


func rebuild_damage(up_to_frame: int) -> void:
	_damage_totals.clear()
	if _replay == null:
		_status_overlay.queue_redraw()
		return
	for ev in _replay.events:
		var tick: int = ReplayUtils.safe_int(ev, "tick", 0)
		if tick > up_to_frame:
			continue
		var event_type: int = ev.get("event_type", 0)
		if event_type != 1 and event_type != 5:
			continue
		var amount: float = ev.get("amount", 0.0)
		if amount <= 0.0:
			continue
		var target: String = ev.get("target", "")
		if not target.begins_with("enemy_"):
			continue
		var source: String = ev.get("source", "")
		_damage_totals[source] = _damage_totals.get(source, 0.0) + amount
	_status_overlay.queue_redraw()


func _draw_player_status() -> void:
	Draw.draw_player_status(_status_overlay, _current_players)
	Draw.draw_damage_meter(_status_overlay, _damage_totals, _current_tick, _tick_rate, _replay)
	if _boss_visible:
		Draw.draw_boss_frame(
			_status_overlay, _boss_name, _boss_health, _boss_max_health, _boss_phase
		)
