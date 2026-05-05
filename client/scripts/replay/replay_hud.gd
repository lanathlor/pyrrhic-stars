extends CanvasLayer
## Playback controls overlay for the replay scene.

signal play_toggled(playing: bool)
signal speed_changed(speed: float)
signal frame_seeked(frame: int)
signal back_pressed

const SPEEDS := [0.25, 0.5, 1.0, 2.0, 4.0]
var _speed_index: int = 2  # start at 1.0x
var _playing: bool = true
var _frame_count: int = 0
var _tick_rate: int = 20
var _dragging: bool = false

var _time_label: Label
var _total_label: Label
var _slider: HSlider
var _play_btn: Button
var _speed_label: Label
var _encounter_label: Label


func _ready() -> void:
	_build_ui()


func init(replay: Variant) -> void:
	_frame_count = replay.frame_count
	_tick_rate = replay.tick_rate
	_slider.max_value = maxf(replay.frame_count - 1, 0)
	_total_label.text = "/ " + replay.total_time_str()
	_encounter_label.text = replay.encounter_id + " - " + replay.outcome


func update_frame(frame: int) -> void:
	if not _dragging:
		_slider.value = frame
	var seconds: float = float(frame) / float(_tick_rate)
	var mins: int = int(seconds) / 60
	var secs: int = int(seconds) % 60
	_time_label.text = "%d:%02d" % [mins, secs]


func _build_ui() -> void:
	layer = 10

	var panel := PanelContainer.new()
	panel.set_anchors_preset(Control.PRESET_BOTTOM_WIDE)
	panel.offset_top = -80
	add_child(panel)

	var vbox := VBoxContainer.new()
	vbox.add_theme_constant_override("separation", 4)
	panel.add_child(vbox)

	# Top row: encounter info
	_encounter_label = Label.new()
	_encounter_label.text = ""
	_encounter_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	vbox.add_child(_encounter_label)

	# Timeline row
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

	# Controls row
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
