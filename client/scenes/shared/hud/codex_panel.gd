extends Control
## Codex Panel -- ability book overlay. Drawing delegated to Chrome/Cards/Layout helpers.

signal loadout_applied(slots: Array)
signal commitment_applied(entries: Array)
signal preset_saved(preset_name: String, slots: Array, commitment: String)
signal preset_deleted(preset_id: int)

const T := preload("res://scenes/shared/hud/codex_panel_theme.gd")

# -- State --
var _active_tab: int = 0
var _schools: Array[String] = []
var _filtered_abilities: Array = []
var _pending_loadout: Array[String] = ["", "", "", "", "", ""]
var _scroll_offset: float = 0.0
var _max_scroll: float = 0.0

# Flux commitment -- pending percentages per school
var _pending_commitment: Dictionary = {}
var _commitment_schools: Array[String] = [
	"bioarcanotechnic", "biometabolic", "frost", "aerokinetic"
]
var _focused_commit_field: int = -1
var _commit_input_text: String = ""
var _commit_rects: Array[Rect2] = []
var _commit_total_rect: Rect2 = Rect2()

# Presets
var _preset_rects: Array[Rect2] = []
var _preset_delete_rects: Array[Rect2] = []
var _preset_save_rect: Rect2 = Rect2()
var _preset_name_rect: Rect2 = Rect2()
var _hovered_preset: int = -1
var _hovered_preset_delete: int = -1
var _hovered_preset_save: bool = false
var _naming_preset: bool = false
var _preset_name_input: String = ""

# Hover
var _hovered_ability_idx: int = -1
var _hovered_loadout_slot: int = -1
var _hovered_tab: int = -1
var _hovered_close: bool = false
var _hovered_apply: bool = false

# Drag
var _dragging: bool = false
var _drag_ability_id: String = ""
var _drag_ability: Dictionary = {}
var _drag_pos: Vector2 = Vector2.ZERO

# Cached layout rects
var _panel_rect: Rect2 = Rect2()
var _close_rect: Rect2 = Rect2()
var _apply_rect: Rect2 = Rect2()
var _grid_rect: Rect2 = Rect2()
var _loadout_rect: Rect2 = Rect2()
var _tab_rects: Array[Rect2] = []
var _card_rects: Array[Rect2] = []
var _loadout_slot_rects: Array[Rect2] = []


func _ready() -> void:
	mouse_filter = Control.MOUSE_FILTER_IGNORE
	focus_mode = Control.FOCUS_ALL
	visible = false


func open() -> void:
	_pending_loadout.clear()
	for s in AbilityCatalog.current_loadout:
		_pending_loadout.append(String(s))
	while _pending_loadout.size() < 6:
		_pending_loadout.append("")
	_build_schools()
	_active_tab = 0
	_filter_abilities()
	_scroll_offset = 0.0
	_hovered_ability_idx = -1
	_hovered_loadout_slot = -1
	_dragging = false
	_focused_commit_field = -1
	_commit_input_text = ""
	_init_pending_commitment()
	visible = true
	mouse_filter = Control.MOUSE_FILTER_STOP


func close() -> void:
	visible = false
	mouse_filter = Control.MOUSE_FILTER_IGNORE
	_dragging = false
	_naming_preset = false
	_preset_name_input = ""


func is_open() -> bool:
	return visible


func _process(_delta: float) -> void:
	if not visible:
		return
	_compute_layout()
	_update_hover()
	queue_redraw()


func _update_hover() -> void:
	var mouse := get_local_mouse_position()
	var hover := (
		CodexPanelLayout
		. compute_hover(
			mouse,
			_close_rect,
			_apply_rect,
			_tab_rects,
			_card_rects,
			_grid_rect,
			_loadout_slot_rects,
			_preset_rects,
			_preset_delete_rects,
			_preset_save_rect,
			_dragging,
		)
	)
	_hovered_close = hover["close"]
	_hovered_apply = hover["apply"]
	_hovered_tab = hover["tab"]
	if not _dragging:
		_hovered_ability_idx = hover["ability_idx"]
	_hovered_loadout_slot = hover["loadout_slot"]
	_hovered_preset = hover["preset"]
	_hovered_preset_delete = hover["preset_delete"]
	_hovered_preset_save = hover["preset_save"]


func _gui_input(event: InputEvent) -> void:
	if not visible:
		return

	if event is InputEventMouseButton:
		var mb := event as InputEventMouseButton
		if mb.button_index == MOUSE_BUTTON_LEFT:
			if mb.pressed:
				_on_left_press(mb.position)
			else:
				_on_left_release(mb.position)
			accept_event()
		elif mb.button_index == MOUSE_BUTTON_RIGHT and mb.pressed:
			_on_right_click(mb.position)
			accept_event()
		elif mb.button_index == MOUSE_BUTTON_WHEEL_UP and mb.pressed:
			_scroll_offset = maxf(_scroll_offset - 40.0, 0.0)
			accept_event()
		elif mb.button_index == MOUSE_BUTTON_WHEEL_DOWN and mb.pressed:
			_scroll_offset = minf(_scroll_offset + 40.0, _max_scroll)
			accept_event()

	if event is InputEventKey and (event as InputEventKey).pressed and _naming_preset:
		if _handle_preset_name_key(event as InputEventKey):
			accept_event()
			return

	if event is InputEventKey and (event as InputEventKey).pressed and _focused_commit_field >= 0:
		if _handle_commit_field_key(event as InputEventKey):
			accept_event()
			return

	if event is InputEventMouseMotion:
		if _dragging:
			_drag_pos = (event as InputEventMouseMotion).position
		accept_event()


func _handle_preset_name_key(key: InputEventKey) -> bool:
	if key.keycode == KEY_ENTER or key.keycode == KEY_KP_ENTER:
		if _preset_name_input.strip_edges() != "":
			var commitment_csv := AbilityCatalog.encode_commitment(_pending_commitment)
			preset_saved.emit(
				_preset_name_input.strip_edges(), _pending_loadout.duplicate(), commitment_csv
			)
		_naming_preset = false
		_preset_name_input = ""
		return true
	if key.keycode == KEY_ESCAPE:
		_naming_preset = false
		_preset_name_input = ""
		return true
	if key.keycode == KEY_BACKSPACE:
		if _preset_name_input.length() > 0:
			_preset_name_input = _preset_name_input.substr(0, _preset_name_input.length() - 1)
		return true
	if key.unicode >= 32 and key.unicode < 127:  # printable ASCII
		if _preset_name_input.length() < 30:
			_preset_name_input += char(key.unicode)
		return true
	return false


func _handle_commit_field_key(key: InputEventKey) -> bool:
	if key.keycode == KEY_ENTER or key.keycode == KEY_KP_ENTER or key.keycode == KEY_TAB:
		_confirm_commit_input()
		if key.keycode == KEY_TAB:
			_focused_commit_field = (_focused_commit_field + 1) % _commitment_schools.size()
			_commit_input_text = str(
				_pending_commitment.get(_commitment_schools[_focused_commit_field], 0)
			)
		else:
			_focused_commit_field = -1
		return true
	if key.keycode == KEY_ESCAPE:
		_focused_commit_field = -1
		return true
	if key.keycode == KEY_BACKSPACE:
		if _commit_input_text.length() > 0:
			_commit_input_text = _commit_input_text.substr(0, _commit_input_text.length() - 1)
		return true
	if key.unicode >= 48 and key.unicode <= 57:  # 0-9
		if _commit_input_text.length() < 3:
			_commit_input_text += char(key.unicode)
		return true
	return false


func _on_left_press(pos: Vector2) -> void:
	if _close_rect.has_point(pos):
		close()
		return

	for i in _tab_rects.size():
		if _tab_rects[i].has_point(pos):
			_active_tab = i
			_filter_abilities()
			_scroll_offset = 0.0
			return

	if _apply_rect.has_point(pos) and _is_anything_dirty() and _commitment_total() == 100:
		if _is_loadout_dirty():
			loadout_applied.emit(_pending_loadout.duplicate())
		if _is_commitment_dirty():
			var entries: Array = []
			for school in _commitment_schools:
				entries.append({"school": school, "percentage": _pending_commitment.get(school, 0)})
			commitment_applied.emit(entries)
		return

	if _handle_preset_click(pos):
		return

	# Commitment input field focus
	_confirm_commit_input()
	var clicked_field := false
	for i in _commit_rects.size():
		if _commit_rects[i].has_point(pos):
			_focused_commit_field = i
			_commit_input_text = str(_pending_commitment.get(_commitment_schools[i], 0))
			clicked_field = true
			grab_focus()
			break
	if not clicked_field:
		_focused_commit_field = -1

	# Start drag from ability card (only implemented abilities)
	for i in _card_rects.size():
		var rect := _card_rects[i]
		if rect.position.y < _grid_rect.position.y or rect.end.y > _grid_rect.end.y:
			continue
		if rect.has_point(pos) and i < _filtered_abilities.size():
			var ability: Dictionary = _filtered_abilities[i]
			if ability.get("implemented", false):
				_dragging = true
				_drag_ability_id = ability.get("id", "")
				_drag_ability = ability
				_drag_pos = pos
				_hovered_ability_idx = -1
			return


func _handle_preset_click(pos: Vector2) -> bool:
	for i in _preset_rects.size():
		if i >= AbilityCatalog.presets.size():
			break
		if not _preset_rects[i].has_point(pos):
			continue
		var preset: Dictionary = AbilityCatalog.presets[i]
		# Check delete button first
		if i < _preset_delete_rects.size() and _preset_delete_rects[i].has_point(pos):
			preset_deleted.emit(preset.get("id", 0))
			return true
		# Load preset into pending state
		var slots: Array = preset.get("slots", [])
		_pending_loadout.clear()
		for j in 6:
			_pending_loadout.append(slots[j] if j < slots.size() else "")
		var commit_csv: String = preset.get("commitment", "")
		if commit_csv != "":
			var parsed := AbilityCatalog.parse_commitment(commit_csv)
			for school in _commitment_schools:
				_pending_commitment[school] = parsed.get(school, 0)
		return true

	# Save button
	if _preset_save_rect.has_point(pos) and not _naming_preset:
		_naming_preset = true
		_preset_name_input = ""
		grab_focus()
		return true

	return false


func _on_left_release(_pos: Vector2) -> void:
	if not _dragging:
		return
	_dragging = false

	# Drop on loadout slot
	if _hovered_loadout_slot >= 0 and _hovered_loadout_slot < 6:
		# Remove from any other slot first
		for j in 6:
			if _pending_loadout[j] == _drag_ability_id:
				_pending_loadout[j] = ""
		_pending_loadout[_hovered_loadout_slot] = _drag_ability_id

	_drag_ability_id = ""
	_drag_ability = {}


func _on_right_click(pos: Vector2) -> void:
	for i in _loadout_slot_rects.size():
		if _loadout_slot_rects[i].has_point(pos):
			_pending_loadout[i] = ""
			return


func _is_loadout_dirty() -> bool:
	for i in 6:
		var current: String = (
			AbilityCatalog.current_loadout[i] if i < AbilityCatalog.current_loadout.size() else ""
		)
		var pending: String = _pending_loadout[i] if i < _pending_loadout.size() else ""
		if current != pending:
			return true
	return false


func _is_commitment_dirty() -> bool:
	for school in _commitment_schools:
		var current: int = AbilityCatalog.current_commitment.get(school, 0)
		var pending: int = _pending_commitment.get(school, 0)
		if current != pending:
			return true
	return false


func _is_anything_dirty() -> bool:
	return _is_loadout_dirty() or _is_commitment_dirty()


func _commitment_total() -> int:
	var total: int = 0
	for school in _commitment_schools:
		total += _pending_commitment.get(school, 0)
	return total


func _confirm_commit_input() -> void:
	if _focused_commit_field < 0 or _focused_commit_field >= _commitment_schools.size():
		return
	var val: int = _commit_input_text.to_int()
	val = clampi(val, 0, 100)
	_pending_commitment[_commitment_schools[_focused_commit_field]] = val


func _init_pending_commitment() -> void:
	_pending_commitment = CodexPanelLayout.init_commitment(_commitment_schools)


func _build_schools() -> void:
	_schools = [""]
	var seen: Dictionary = {}
	for entry in AbilityCatalog.catalog:
		var school: String = entry.get("school", "")
		if school != "" and not seen.has(school):
			seen[school] = true
			_schools.append(school)


func _filter_abilities() -> void:
	if _active_tab == 0 or _active_tab >= _schools.size():
		_filtered_abilities = AbilityCatalog.catalog.duplicate()
	else:
		_filtered_abilities = AbilityCatalog.get_abilities_by_school(_schools[_active_tab])


func _compute_layout() -> void:
	var r := (
		CodexPanelLayout
		. compute(
			{
				size = size,
				schools = _schools,
				abilities = _filtered_abilities,
				presets = AbilityCatalog.presets,
				commit_schools = _commitment_schools,
				scroll = _scroll_offset,
				naming = _naming_preset,
			}
		)
	)
	_panel_rect = r.panel_rect
	_close_rect = r.close_rect
	_grid_rect = r.grid_rect
	_loadout_rect = r.loadout_rect
	_apply_rect = r.apply_rect
	_commit_total_rect = r.commit_total_rect
	_preset_save_rect = r.preset_save_rect
	_preset_name_rect = r.preset_name_rect
	_tab_rects = r.tab_rects
	_card_rects = r.card_rects
	_loadout_slot_rects = r.loadout_slot_rects
	_commit_rects = r.commit_rects
	_preset_rects = r.preset_rects
	_preset_delete_rects = r.preset_delete_rects
	_max_scroll = r.max_scroll
	_scroll_offset = r.scroll_offset


func _draw() -> void:
	if not visible:
		return
	if _loadout_slot_rects.is_empty():
		_compute_layout()
	draw_rect(Rect2(Vector2.ZERO, size), T.BG_OVERLAY)
	draw_rect(_panel_rect, T.PANEL_BG)
	draw_rect(_panel_rect, T.PANEL_BORDER, false, 1.5)
	CodexPanelDrawChrome.draw_title(self, _panel_rect, _close_rect, _hovered_close)
	CodexPanelDrawChrome.draw_tabs(self, _tab_rects, _schools, _active_tab, _hovered_tab)
	_draw_grid_and_chrome()
	var show_tip := (
		_hovered_ability_idx >= 0
		and _hovered_ability_idx < _filtered_abilities.size()
		and not _dragging
	)
	if show_tip:
		CodexPanelDrawCards.draw_tooltip(
			self, size, _card_rects, _filtered_abilities, _hovered_ability_idx
		)
	if _dragging:
		CodexPanelDrawChrome.draw_drag_ghost(self, _drag_ability, _drag_pos, _hovered_loadout_slot)


func _draw_grid_and_chrome() -> void:
	var gs := {
		panel = _panel_rect,
		grid = _grid_rect,
		cards = _card_rects,
		abilities = _filtered_abilities,
		hovered = _hovered_ability_idx,
		loadout = _pending_loadout,
		scroll = _scroll_offset,
		max_scroll = _max_scroll
	}
	CodexPanelDrawCards.draw_grid(self, gs)
	var ls := {
		panel = _panel_rect,
		loadout = _loadout_rect,
		slots = _loadout_slot_rects,
		apply = _apply_rect,
		pending = _pending_loadout,
		hovered = _hovered_loadout_slot,
		hovered_apply = _hovered_apply,
		dragging = _dragging,
		dirty = _is_anything_dirty() and _commitment_total() == 100
	}
	CodexPanelDrawChrome.draw_loadout(self, ls)
	var ps := {
		rects = _preset_rects,
		delete_rects = _preset_delete_rects,
		save_rect = _preset_save_rect,
		name_rect = _preset_name_rect,
		hovered = _hovered_preset,
		hovered_delete = _hovered_preset_delete,
		hovered_save = _hovered_preset_save,
		naming = _naming_preset,
		name_input = _preset_name_input
	}
	CodexPanelDrawChrome.draw_presets(self, ps)
	var cs := {
		panel = _panel_rect,
		total_rect = _commit_total_rect,
		rects = _commit_rects,
		schools = _commitment_schools,
		pending = _pending_commitment,
		focused = _focused_commit_field,
		input_text = _commit_input_text
	}
	CodexPanelDrawChrome.draw_commitment(self, cs)
