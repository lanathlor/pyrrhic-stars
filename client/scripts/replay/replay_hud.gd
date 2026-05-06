extends CanvasLayer
## Playback controls overlay for the replay scene.

signal play_toggled(playing: bool)
signal speed_changed(speed: float)
signal frame_seeked(frame: int)
signal back_pressed

const SPEEDS := [0.25, 0.5, 1.0, 2.0, 4.0]
const CLASS_MAX_HP := {
	"gunner": 150.0,
	"vanguard": 200.0,
	"blade_dancer": 150.0,
}
const CLASS_COLORS := {
	"gunner": Color(0.24, 0.62, 0.95),
	"vanguard": Color(0.82, 0.44, 0.24),
	"blade_dancer": Color(0.36, 0.82, 0.66),
}

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

	# Player status overlay (top-left HP bars)
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
	var font := ThemeDB.fallback_font
	var x := 12.0
	var y := 12.0
	var panel_w := 200.0
	var name_h := 18.0
	var bar_h := 16.0
	var panel_h := name_h + bar_h + 4.0
	var gap := 4.0

	for pdata in _current_players:
		var hp: float = pdata.get("health", 0.0)
		var cls: String = pdata.get("class_name", "gunner")
		var max_hp: float = CLASS_MAX_HP.get(cls, 150.0)
		var username: String = pdata.get("username", "Player")
		var ratio := clampf(hp / maxf(max_hp, 1.0), 0.0, 1.0)
		var dead := hp <= 0.0

		# Panel background
		_status_overlay.draw_rect(Rect2(x, y, panel_w, panel_h), Color(0.05, 0.05, 0.08, 0.85))

		# Player name
		var name_color := Color(0.4, 0.4, 0.4) if dead else Color.WHITE
		_status_overlay.draw_string(
			font,
			Vector2(x + 6, y + 14),
			username,
			HORIZONTAL_ALIGNMENT_LEFT,
			panel_w - 12,
			12,
			name_color
		)

		# HP bar background
		var bar_y := y + name_h
		_status_overlay.draw_rect(
			Rect2(x + 4, bar_y, panel_w - 8, bar_h), Color(0.15, 0.15, 0.15, 0.9)
		)

		# HP fill
		if not dead:
			var bar_color := Color(0.8, 0.2, 0.2) if ratio <= 0.3 else Color(0.2, 0.75, 0.3)
			_status_overlay.draw_rect(Rect2(x + 4, bar_y, (panel_w - 8) * ratio, bar_h), bar_color)

		# Class label (left)
		var class_label := cls.replace("_", " ").to_upper()
		var text_color := Color(0.5, 0.5, 0.5) if dead else Color(0.8, 0.8, 0.8, 0.7)
		_status_overlay.draw_string(
			font,
			Vector2(x + 8, bar_y + 12),
			class_label,
			HORIZONTAL_ALIGNMENT_LEFT,
			80,
			9,
			text_color
		)

		# HP text (right) or DEAD
		var hp_text := "DEAD" if dead else "%d / %d" % [int(hp), int(max_hp)]
		var hp_color := Color(0.6, 0.2, 0.2) if dead else Color.WHITE
		_status_overlay.draw_string(
			font,
			Vector2(x + panel_w - 100, bar_y + 12),
			hp_text,
			HORIZONTAL_ALIGNMENT_RIGHT,
			88,
			9,
			hp_color
		)

		y += panel_h + gap

	_draw_damage_meter()
	_draw_boss_frame()


func _draw_boss_frame() -> void:
	if not _boss_visible:
		return

	var font := ThemeDB.fallback_font
	var vp_size := _status_overlay.get_viewport_rect().size
	var center_x := vp_size.x / 2.0
	var panel_x := center_x - 216.0
	var panel_w := 432.0
	var panel_y := 14.0

	# Boss name (left)
	_status_overlay.draw_string(
		font,
		Vector2(panel_x, panel_y + 9.0),
		_boss_name,
		HORIZONTAL_ALIGNMENT_LEFT,
		240.0,
		12,
		Color(0.93, 0.9, 0.8, 0.97)
	)

	# Phase label (right)
	var phase_text := "P%d" % _boss_phase
	var phase_color: Color
	match _boss_phase:
		1:
			phase_color = Color(0.56, 0.74, 0.28)
		2:
			phase_color = Color(0.93, 0.7, 0.25)
		3:
			phase_color = Color(0.93, 0.34, 0.34)
		_:
			phase_color = Color(0.5, 0.5, 0.5)
	_status_overlay.draw_string(
		font,
		Vector2(panel_x + panel_w - 36.0, panel_y + 9.0),
		phase_text,
		HORIZONTAL_ALIGNMENT_RIGHT,
		32.0,
		11,
		phase_color
	)

	# HP bar
	var bar_rect := Rect2(panel_x, panel_y + 14.0, panel_w, 12.0)
	var hp_ratio := clampf(_boss_health / maxf(_boss_max_health, 1.0), 0.0, 1.0)
	var bar_color: Color
	match _boss_phase:
		1:
			bar_color = Color(0.56, 0.22, 0.22)
		2:
			bar_color = Color(0.74, 0.44, 0.18)
		3:
			bar_color = Color(0.78, 0.18, 0.18)
		_:
			bar_color = Color(0.5, 0.5, 0.5)

	# Bar background
	_status_overlay.draw_rect(bar_rect, Color(0.08, 0.08, 0.1, 0.9))
	# Bar fill
	if hp_ratio > 0.0:
		_status_overlay.draw_rect(
			Rect2(bar_rect.position, Vector2(bar_rect.size.x * hp_ratio, bar_rect.size.y)),
			bar_color
		)
	# Bar border
	_status_overlay.draw_rect(bar_rect, Color(0.25, 0.25, 0.3, 0.8), false, 1.0)

	# HP text
	var hp_text := "%d / %d" % [int(_boss_health), int(_boss_max_health)]
	_status_overlay.draw_string(
		font,
		Vector2(bar_rect.position.x + bar_rect.size.x - 118.0, bar_rect.position.y + 12.0),
		hp_text,
		HORIZONTAL_ALIGNMENT_RIGHT,
		110.0,
		10,
		Color(0.9, 0.92, 0.96, 0.95)
	)


func _draw_damage_meter() -> void:
	if _damage_totals.is_empty():
		return

	var font := ThemeDB.fallback_font
	var vp_size := _status_overlay.get_viewport_rect().size
	var meter_w := 200.0
	var meter_x := vp_size.x - meter_w - 12.0
	var entry_h := 20.0
	var title_h := 22.0
	var y := 12.0

	# Sort by damage descending
	var sorted_ids: Array = _damage_totals.keys()
	sorted_ids.sort_custom(func(a, b): return _damage_totals[a] > _damage_totals[b])
	var max_damage: float = _damage_totals.get(sorted_ids[0], 1.0) if sorted_ids.size() > 0 else 1.0
	if max_damage <= 0.0:
		max_damage = 1.0

	var entry_count := mini(sorted_ids.size(), 5)

	# Background panel
	var panel_h := title_h + entry_count * entry_h + 4.0
	_status_overlay.draw_rect(Rect2(meter_x, y, meter_w, panel_h), Color(0.05, 0.05, 0.08, 0.85))

	# Title with DPS
	var title := "Damage"
	var elapsed: float = float(_current_tick) / float(_tick_rate)
	if elapsed > 0.0:
		var total_dmg: float = 0.0
		for eid in _damage_totals:
			total_dmg += _damage_totals[eid]
		var dps := total_dmg / maxf(elapsed, 1.0)
		title = "Damage (%.0f DPS)" % dps

	_status_overlay.draw_string(
		font,
		Vector2(meter_x + 6, y + 15),
		title,
		HORIZONTAL_ALIGNMENT_LEFT,
		meter_w - 12,
		11,
		Color(0.7, 0.7, 0.7)
	)

	# Player entries
	for i in entry_count:
		var eid: String = sorted_ids[i]
		var dmg: float = _damage_totals[eid]
		var ey := y + title_h + i * entry_h

		# Class-colored bar
		var cls: String = _get_participant_class(eid)
		var bar_color: Color = CLASS_COLORS.get(cls, Color(0.5, 0.5, 0.5))
		var ratio := dmg / max_damage
		_status_overlay.draw_rect(
			Rect2(meter_x + 4, ey + 2, (meter_w - 8) * ratio, entry_h - 4), Color(bar_color, 0.85)
		)

		# Player name
		var pname: String = _replay.get_participant_name(eid) if _replay else eid
		if pname.length() > 12:
			pname = pname.substr(0, 12)
		_status_overlay.draw_string(
			font,
			Vector2(meter_x + 8, ey + 15),
			pname,
			HORIZONTAL_ALIGNMENT_LEFT,
			meter_w * 0.5,
			10,
			Color.WHITE
		)

		# Damage amount + per-player DPS
		var dmg_text: String
		if dmg >= 1000.0:
			dmg_text = "%.1fk" % (dmg / 1000.0)
		else:
			dmg_text = "%d" % int(dmg)
		if elapsed > 0.0:
			var pdps := dmg / maxf(elapsed, 1.0)
			dmg_text += " (%.0f)" % pdps
		_status_overlay.draw_string(
			font,
			Vector2(meter_x + meter_w - 94, ey + 15),
			dmg_text,
			HORIZONTAL_ALIGNMENT_RIGHT,
			86,
			10,
			Color.WHITE
		)


func _get_participant_class(entity_id: String) -> String:
	if _replay == null:
		return "gunner"
	for p in _replay.participants:
		if p.get("entity_id", "") == entity_id:
			return p.get("class", "gunner")
	return "gunner"
