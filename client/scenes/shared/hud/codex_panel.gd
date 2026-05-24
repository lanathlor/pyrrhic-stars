extends Control

## Codex Panel -- full-screen _draw()-based ability book overlay for the Arcanotechnicien.
## Shows ability catalog with school filter tabs, drag-and-drop loadout editing.

signal loadout_applied(slots: Array)
signal commitment_applied(entries: Array)

# -- Colors --
const BG_OVERLAY := Color(0.0, 0.0, 0.02, 0.7)
const PANEL_BG := Color(0.02, 0.025, 0.035, 0.92)
const PANEL_BORDER := Color(0.28, 0.3, 0.36, 0.85)
const TEXT_PRIMARY := Color(0.92, 0.94, 0.97, 0.95)
const TEXT_MUTED := Color(0.66, 0.7, 0.77, 0.9)
const TEXT_DIM := Color(0.45, 0.48, 0.55, 0.7)
const ACCENT := Color(0.3, 0.65, 0.85)
const TAB_ACTIVE_BG := Color(0.12, 0.15, 0.22, 0.9)
const TAB_HOVER_BG := Color(0.08, 0.1, 0.15, 0.7)
const CLOSE_COLOR := Color(0.8, 0.3, 0.3)
const APPLY_ACTIVE := Color(0.25, 0.7, 0.4)
const APPLY_INACTIVE := Color(0.3, 0.35, 0.4, 0.5)
const UNIMPL_OVERLAY := Color(0.0, 0.0, 0.0, 0.55)
const SLOT_EMPTY_BG := Color(0.04, 0.05, 0.07, 0.6)
const SLOT_FILLED_BG := Color(0.06, 0.08, 0.12, 0.8)

const SCHOOL_COLORS := {
	"bioarcanotechnic": Color(0.2, 0.7, 0.65),
	"biometabolic": Color(0.3, 0.75, 0.35),
	"frost": Color(0.4, 0.75, 0.95),
	"fire": Color(0.9, 0.4, 0.2),
	"electricity": Color(0.95, 0.85, 0.3),
	"aerokinetic": Color(0.7, 0.9, 0.95),
	"hydrodynamic": Color(0.3, 0.5, 0.9),
	"pure": Color(0.85, 0.85, 0.9),
	"shadow": Color(0.55, 0.3, 0.7),
	"gravitonic": Color(0.45, 0.25, 0.6),
	"illusion": Color(0.8, 0.45, 0.7),
	"martial": Color(0.7, 0.55, 0.35),
}

const SCHOOL_LABELS := {
	"bioarcanotechnic": "Bioarc.",
	"biometabolic": "Biometa.",
	"frost": "Frost",
	"fire": "Fire",
	"electricity": "Electric",
	"aerokinetic": "Aero.",
	"hydrodynamic": "Hydro.",
	"pure": "Pure",
	"shadow": "Shadow",
	"gravitonic": "Gravit.",
	"illusion": "Illusion",
	"martial": "Martial",
}

const TYPE_LETTERS := {
	"destruction": "D",
	"protection": "P",
	"enhancement": "E",
	"affliction": "A",
	"displacement": "X",
}

const FLUX_DOT_COUNT := {
	"low": 1,
	"medium": 2,
	"high": 3,
	"extreme": 4,
}

const SLOT_KEYBINDS: Array[String] = ["1", "2", "R", "T", "F", "C"]

# -- Layout --
const MARGIN_H := 80.0
const MARGIN_TOP := 50.0
const MARGIN_BOTTOM := 60.0
const TITLE_H := 42.0
const TAB_ROW_H := 28.0
const TAB_GAP := 4.0
const CARD_W := 150.0
const CARD_H := 95.0
const CARD_GAP := 8.0
const LOADOUT_H := 80.0
const LOADOUT_SLOT_W := 130.0
const LOADOUT_SLOT_H := 55.0
const LOADOUT_GAP := 8.0
const APPLY_W := 80.0
const APPLY_H := 32.0
const COMMIT_H := 64.0
const COMMIT_INPUT_W := 48.0
const COMMIT_INPUT_H := 28.0

# -- State --
var _active_tab: int = 0
var _schools: Array[String] = []
var _filtered_abilities: Array = []
var _pending_loadout: Array[String] = ["", "", "", "", "", ""]
var _scroll_offset: float = 0.0
var _max_scroll: float = 0.0

# Flux commitment -- pending percentages per school
var _pending_commitment: Dictionary = {}  # school → int (0-100)
var _commitment_schools: Array[String] = ["bioarcanotechnic", "biometabolic", "frost", "aerokinetic"]
var _focused_commit_field: int = -1  # index into _commitment_schools, -1 = none
var _commit_input_text: String = ""   # current text in focused field
var _commit_rects: Array[Rect2] = []  # input field rects
var _commit_total_rect: Rect2 = Rect2()

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


func is_open() -> bool:
	return visible


# =============================================================================
# Frame update
# =============================================================================


func _process(_delta: float) -> void:
	if not visible:
		return
	_compute_layout()
	_update_hover()
	queue_redraw()


func _update_hover() -> void:
	var mouse := get_local_mouse_position()
	_hovered_close = _close_rect.has_point(mouse)
	_hovered_apply = _apply_rect.has_point(mouse)

	_hovered_tab = -1
	for i in _tab_rects.size():
		if _tab_rects[i].has_point(mouse):
			_hovered_tab = i
			break

	if not _dragging:
		_hovered_ability_idx = -1
		for i in _card_rects.size():
			var rect := _card_rects[i]
			if rect.position.y < _grid_rect.position.y or rect.end.y > _grid_rect.end.y:
				continue
			if rect.has_point(mouse):
				_hovered_ability_idx = i
				break

	_hovered_loadout_slot = -1
	for i in _loadout_slot_rects.size():
		if _loadout_slot_rects[i].has_point(mouse):
			_hovered_loadout_slot = i
			break


# =============================================================================
# Input
# =============================================================================


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

	if event is InputEventKey and (event as InputEventKey).pressed and _focused_commit_field >= 0:
		var key := event as InputEventKey
		if key.keycode == KEY_ENTER or key.keycode == KEY_KP_ENTER or key.keycode == KEY_TAB:
			_confirm_commit_input()
			if key.keycode == KEY_TAB:
				_focused_commit_field = (_focused_commit_field + 1) % _commitment_schools.size()
				_commit_input_text = str(_pending_commitment.get(_commitment_schools[_focused_commit_field], 0))
			else:
				_focused_commit_field = -1
			accept_event()
			return
		elif key.keycode == KEY_ESCAPE:
			_focused_commit_field = -1
			accept_event()
			return
		elif key.keycode == KEY_BACKSPACE:
			if _commit_input_text.length() > 0:
				_commit_input_text = _commit_input_text.substr(0, _commit_input_text.length() - 1)
			accept_event()
			return
		elif key.unicode >= 48 and key.unicode <= 57:  # 0-9
			if _commit_input_text.length() < 3:
				_commit_input_text += char(key.unicode)
			accept_event()
			return

	if event is InputEventMouseMotion:
		if _dragging:
			_drag_pos = (event as InputEventMouseMotion).position
		accept_event()


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


# =============================================================================
# Data helpers
# =============================================================================


func _is_loadout_dirty() -> bool:
	for i in 6:
		var current: String = AbilityCatalog.current_loadout[i] if i < AbilityCatalog.current_loadout.size() else ""
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
	_pending_commitment.clear()
	if AbilityCatalog.current_commitment.is_empty():
		# Default split matching server default
		_pending_commitment = {
			"bioarcanotechnic": 50,
			"biometabolic": 30,
			"frost": 10,
			"aerokinetic": 10,
		}
	else:
		for school in _commitment_schools:
			_pending_commitment[school] = AbilityCatalog.current_commitment.get(school, 0)


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


func _get_school_color(school: String) -> Color:
	return SCHOOL_COLORS.get(school, Color(0.5, 0.5, 0.55))


func _get_slot_for_ability(ability_id: String) -> int:
	for i in 6:
		if i < _pending_loadout.size() and _pending_loadout[i] == ability_id:
			return i
	return -1


# =============================================================================
# Layout computation
# =============================================================================


func _compute_layout() -> void:
	var sw := size.x
	var sh := size.y
	var font := ThemeDB.fallback_font

	_panel_rect = Rect2(MARGIN_H, MARGIN_TOP, sw - MARGIN_H * 2, sh - MARGIN_TOP - MARGIN_BOTTOM)
	var px := _panel_rect.position.x
	var py := _panel_rect.position.y
	var pw := _panel_rect.size.x
	var ph := _panel_rect.size.y

	_close_rect = Rect2(px + pw - 40, py + 8, 28, 26)

	# Tabs
	var tab_y := py + TITLE_H + 2
	_tab_rects.clear()
	var tx := px + 12.0
	for i in _schools.size():
		var label: String = "All" if i == 0 else SCHOOL_LABELS.get(_schools[i], _schools[i])
		var tw := font.get_string_size(label, HORIZONTAL_ALIGNMENT_LEFT, -1, 10).x + 18.0
		tw = maxf(tw, 44.0)
		_tab_rects.append(Rect2(tx, tab_y + 2, tw, TAB_ROW_H - 4))
		tx += tw + TAB_GAP

	# Grid area (loadout + commitment at bottom)
	var grid_y := tab_y + TAB_ROW_H + 4
	var bottom_h := LOADOUT_H + COMMIT_H
	var loadout_y := py + ph - bottom_h
	_grid_rect = Rect2(px + 12, grid_y, pw - 24, loadout_y - grid_y - 8)
	_loadout_rect = Rect2(px, loadout_y, pw, LOADOUT_H)

	# Card rects
	_card_rects.clear()
	var grid_w := _grid_rect.size.x
	var cols := maxi(int(grid_w / (CARD_W + CARD_GAP)), 1)
	var grid_start_x := _grid_rect.position.x + (grid_w - cols * (CARD_W + CARD_GAP) + CARD_GAP) / 2.0

	for i in _filtered_abilities.size():
		var col := i % cols
		var row := i / cols
		var cx := grid_start_x + col * (CARD_W + CARD_GAP)
		var cy := _grid_rect.position.y + row * (CARD_H + CARD_GAP) - _scroll_offset
		_card_rects.append(Rect2(cx, cy, CARD_W, CARD_H))

	var total_rows := ceili(float(_filtered_abilities.size()) / float(cols))
	var total_height := total_rows * (CARD_H + CARD_GAP)
	_max_scroll = maxf(total_height - _grid_rect.size.y, 0.0)
	_scroll_offset = clampf(_scroll_offset, 0.0, _max_scroll)

	# Loadout slot rects
	_loadout_slot_rects.clear()
	var loadout_total_w := 6 * LOADOUT_SLOT_W + 5 * LOADOUT_GAP + APPLY_W + 20
	var lx := px + (pw - loadout_total_w) / 2.0
	var ly := loadout_y + (LOADOUT_H - LOADOUT_SLOT_H) / 2.0
	for i in 6:
		_loadout_slot_rects.append(Rect2(lx + i * (LOADOUT_SLOT_W + LOADOUT_GAP), ly, LOADOUT_SLOT_W, LOADOUT_SLOT_H))

	_apply_rect = Rect2(lx + 6 * (LOADOUT_SLOT_W + LOADOUT_GAP) + 12, ly + (LOADOUT_SLOT_H - APPLY_H) / 2.0, APPLY_W, APPLY_H)

	# Commitment input rects
	var commit_y := loadout_y + LOADOUT_H
	_commit_total_rect = Rect2(px, commit_y, pw, COMMIT_H)
	_commit_rects.clear()
	var commit_entry_w := 100.0 + COMMIT_INPUT_W + 24.0  # label + input + gap
	var commit_total_w := _commitment_schools.size() * commit_entry_w
	var cx := px + (pw - commit_total_w) / 2.0
	for i in _commitment_schools.size():
		_commit_rects.append(Rect2(cx + 100.0, commit_y + (COMMIT_H - COMMIT_INPUT_H) / 2.0, COMMIT_INPUT_W, COMMIT_INPUT_H))
		cx += commit_entry_w


# =============================================================================
# Drawing
# =============================================================================


func _draw() -> void:
	if not visible:
		return
	if _loadout_slot_rects.is_empty():
		_compute_layout()

	# Background overlay
	draw_rect(Rect2(Vector2.ZERO, size), BG_OVERLAY)

	# Panel
	draw_rect(_panel_rect, PANEL_BG)
	draw_rect(_panel_rect, PANEL_BORDER, false, 1.5)

	_draw_title()
	_draw_tabs()
	_draw_grid()
	_draw_loadout()
	_draw_commitment()

	if _hovered_ability_idx >= 0 and _hovered_ability_idx < _filtered_abilities.size() and not _dragging:
		_draw_tooltip()
	if _dragging:
		_draw_drag_ghost()


func _draw_title() -> void:
	var font := ThemeDB.fallback_font
	var py := _panel_rect.position.y

	draw_string(font, Vector2(_panel_rect.position.x + 16, py + 28), "HARMONIST CODEX",
		HORIZONTAL_ALIGNMENT_LEFT, _panel_rect.size.x - 60, 18, ACCENT)

	# Separator
	var sep_y := py + TITLE_H
	draw_line(Vector2(_panel_rect.position.x + 8, sep_y), Vector2(_panel_rect.end.x - 8, sep_y), PANEL_BORDER, 1.0)

	# Close button
	var cc := Color(1.0, 0.4, 0.4) if _hovered_close else CLOSE_COLOR
	if _hovered_close:
		draw_rect(_close_rect, Color(cc, 0.2))
	draw_string(font, Vector2(_close_rect.position.x + 6, _close_rect.position.y + 18), "X",
		HORIZONTAL_ALIGNMENT_CENTER, _close_rect.size.x - 12, 14, cc)


func _draw_tabs() -> void:
	var font := ThemeDB.fallback_font
	for i in _tab_rects.size():
		var rect := _tab_rects[i]
		var is_active := i == _active_tab
		var is_hovered := i == _hovered_tab

		if is_active:
			draw_rect(rect, TAB_ACTIVE_BG)
			draw_rect(rect, ACCENT, false, 1.0)
		elif is_hovered:
			draw_rect(rect, TAB_HOVER_BG)

		var label: String = "All" if i == 0 else SCHOOL_LABELS.get(_schools[i], _schools[i])
		var tc: Color
		if i == 0:
			tc = TEXT_PRIMARY if is_active else TEXT_MUTED
		else:
			var sc := _get_school_color(_schools[i])
			tc = sc if is_active else Color(sc, 0.6)

		draw_string(font, Vector2(rect.position.x + 8, rect.position.y + 18), label,
			HORIZONTAL_ALIGNMENT_LEFT, rect.size.x - 16, 10, tc)


func _draw_grid() -> void:
	# Separator
	draw_line(Vector2(_panel_rect.position.x + 8, _grid_rect.position.y - 2),
		Vector2(_panel_rect.end.x - 8, _grid_rect.position.y - 2), PANEL_BORDER, 1.0)

	if _filtered_abilities.is_empty():
		var font := ThemeDB.fallback_font
		var msg := "Waiting for ability catalog..." if AbilityCatalog.catalog.is_empty() else "No abilities in this school."
		draw_string(font, Vector2(_grid_rect.position.x + 20, _grid_rect.position.y + 40), msg,
			HORIZONTAL_ALIGNMENT_LEFT, _grid_rect.size.x - 40, 12, TEXT_DIM)
		return

	for i in _card_rects.size():
		var rect := _card_rects[i]
		if rect.end.y < _grid_rect.position.y or rect.position.y > _grid_rect.end.y:
			continue
		if rect.position.y < _grid_rect.position.y - 1 or rect.end.y > _grid_rect.end.y + 1:
			continue
		if i < _filtered_abilities.size():
			_draw_ability_card(rect, _filtered_abilities[i], i == _hovered_ability_idx)

	# Scroll indicators
	if _max_scroll > 0:
		var font := ThemeDB.fallback_font
		if _scroll_offset > 0:
			draw_string(font, Vector2(_grid_rect.get_center().x - 6, _grid_rect.position.y + 14),
				"^", HORIZONTAL_ALIGNMENT_CENTER, 12, 12, TEXT_DIM)
		if _scroll_offset < _max_scroll:
			draw_string(font, Vector2(_grid_rect.get_center().x - 6, _grid_rect.end.y - 4),
				"v", HORIZONTAL_ALIGNMENT_CENTER, 12, 12, TEXT_DIM)


func _draw_ability_card(rect: Rect2, ability: Dictionary, hovered: bool) -> void:
	var font := ThemeDB.fallback_font
	var school: String = ability.get("school", "")
	var school_color := _get_school_color(school)
	var ability_id: String = ability.get("id", "")
	var implemented: bool = ability.get("implemented", false)
	var affinity: String = ability.get("affinity", "off")
	var slot := _get_slot_for_ability(ability_id)

	# Affinity border color
	var border_color: Color
	match affinity:
		"primary":
			border_color = school_color
		"secondary":
			border_color = Color(school_color, 0.5)
		_:
			border_color = Color(0.5, 0.25, 0.25, 0.5)

	# Background
	var bg := Color(0.04, 0.05, 0.07, 0.7)
	if hovered and implemented:
		bg = Color(0.08, 0.1, 0.14, 0.85)
	if slot >= 0:
		bg = Color(school_color.r * 0.15, school_color.g * 0.15, school_color.b * 0.15, 0.7)

	draw_rect(rect, bg)

	# School-colored left border
	draw_rect(Rect2(rect.position.x, rect.position.y, 3, rect.size.y), border_color)

	# Card border
	draw_rect(rect, border_color if hovered else PANEL_BORDER, false, 1.5 if hovered else 1.0)

	# Ability name (2 lines)
	var ability_name: String = ability.get("name", "???")
	var name_color := TEXT_PRIMARY if implemented else TEXT_DIM
	var name_parts := ability_name.split(" ", true, 1)
	draw_string(font, Vector2(rect.position.x + 8, rect.position.y + 16), name_parts[0],
		HORIZONTAL_ALIGNMENT_LEFT, rect.size.x - 14, 11, name_color)
	if name_parts.size() > 1:
		draw_string(font, Vector2(rect.position.x + 8, rect.position.y + 30), name_parts[1],
			HORIZONTAL_ALIGNMENT_LEFT, rect.size.x - 14, 11, name_color)

	# School abbrev + type letter (bottom-left area)
	var school_abbrev: String = SCHOOL_LABELS.get(school, school).to_upper().left(3)
	var ability_type: String = ability.get("ability_type", "")
	var type_letter: String = TYPE_LETTERS.get(ability_type, "?")
	draw_string(font, Vector2(rect.position.x + 8, rect.end.y - 22),
		"%s  %s" % [school_abbrev, type_letter],
		HORIZONTAL_ALIGNMENT_LEFT, rect.size.x - 14, 9,
		Color(school_color, 0.7) if implemented else TEXT_DIM)

	# Flux cost dots
	var flux_cost: String = ability.get("flux_cost", "")
	var dot_count: int = FLUX_DOT_COUNT.get(flux_cost, 0)
	if dot_count > 0:
		for d in dot_count:
			draw_rect(Rect2(rect.position.x + 8.0 + d * 10.0, rect.end.y - 10.0, 6.0, 4.0),
				ACCENT if implemented else TEXT_DIM)

	# Slot badge (top-right)
	if slot >= 0:
		var badge := Rect2(rect.end.x - 22, rect.position.y + 4, 18, 16)
		draw_rect(badge, Color(ACCENT, 0.8))
		draw_string(font, Vector2(badge.position.x + 3, badge.position.y + 12), SLOT_KEYBINDS[slot],
			HORIZONTAL_ALIGNMENT_CENTER, 12, 10, Color.WHITE)

	# Unimplemented overlay with cross-hatch
	if not implemented:
		draw_rect(rect, UNIMPL_OVERLAY)
		var hatch_step := 18.0
		var hatch_color := Color(0.3, 0.3, 0.35, 0.25)
		var offset := 0.0
		while offset < rect.size.x + rect.size.y:
			var sx := rect.position.x + offset
			var sy := rect.position.y
			var ex := rect.position.x
			var ey := rect.position.y + offset
			if sx > rect.end.x:
				sy += sx - rect.end.x
				sx = rect.end.x
			if ey > rect.end.y:
				ex += ey - rect.end.y
				ey = rect.end.y
			if sy <= rect.end.y and ex <= rect.end.x:
				draw_line(Vector2(sx, sy), Vector2(ex, ey), hatch_color, 1.0)
			offset += hatch_step


func _draw_loadout() -> void:
	var font := ThemeDB.fallback_font

	# Separator
	draw_line(Vector2(_panel_rect.position.x + 8, _loadout_rect.position.y),
		Vector2(_panel_rect.end.x - 8, _loadout_rect.position.y), PANEL_BORDER, 1.0)

	# Label
	draw_string(font, Vector2(_loadout_rect.position.x + 16, _loadout_rect.position.y + 14),
		"LOADOUT", HORIZONTAL_ALIGNMENT_LEFT, 80, 10, TEXT_MUTED)

	for i in 6:
		var rect := _loadout_slot_rects[i]
		var ability_id: String = _pending_loadout[i] if i < _pending_loadout.size() else ""
		var is_hovered := i == _hovered_loadout_slot
		var is_drop := _dragging and is_hovered

		var bg := SLOT_FILLED_BG if ability_id != "" else SLOT_EMPTY_BG
		if is_drop:
			bg = Color(ACCENT.r, ACCENT.g, ACCENT.b, 0.25)
		elif is_hovered:
			bg = Color(0.08, 0.1, 0.15, 0.8)

		draw_rect(rect, bg)
		draw_rect(rect, ACCENT if is_drop else PANEL_BORDER, false, 1.5 if is_drop else 1.0)

		# Keybind
		draw_string(font, Vector2(rect.position.x + 4, rect.position.y + 12), SLOT_KEYBINDS[i],
			HORIZONTAL_ALIGNMENT_LEFT, 20, 9, TEXT_MUTED)

		if ability_id != "":
			var ability := AbilityCatalog.get_ability(ability_id)
			var sn: String = ability.get("name", ability_id)
			var sc: String = ability.get("school", "")
			draw_rect(Rect2(rect.position.x, rect.position.y, 3, rect.size.y), _get_school_color(sc))
			var parts := sn.split(" ", true, 1)
			draw_string(font, Vector2(rect.position.x + 8, rect.position.y + 28), parts[0],
				HORIZONTAL_ALIGNMENT_LEFT, rect.size.x - 12, 10, TEXT_PRIMARY)
			if parts.size() > 1:
				draw_string(font, Vector2(rect.position.x + 8, rect.position.y + 42), parts[1],
					HORIZONTAL_ALIGNMENT_LEFT, rect.size.x - 12, 10, TEXT_PRIMARY)
		else:
			draw_string(font, Vector2(rect.position.x + 8, rect.position.y + 34), "Empty",
				HORIZONTAL_ALIGNMENT_LEFT, rect.size.x - 12, 10, TEXT_DIM)

	# Apply button (active when dirty AND commitment totals 100%)
	var dirty := _is_anything_dirty() and _commitment_total() == 100
	var abg := Color(APPLY_ACTIVE, 0.5 if _hovered_apply else 0.3) if dirty else Color(APPLY_INACTIVE, 0.15)
	draw_rect(_apply_rect, abg)
	draw_rect(_apply_rect, APPLY_ACTIVE if dirty else APPLY_INACTIVE, false, 1.0)
	draw_string(font, Vector2(_apply_rect.position.x + 6, _apply_rect.position.y + 21), "APPLY",
		HORIZONTAL_ALIGNMENT_CENTER, _apply_rect.size.x - 12, 12, APPLY_ACTIVE if dirty else APPLY_INACTIVE)


func _draw_commitment() -> void:
	var font := ThemeDB.fallback_font

	# Separator
	draw_line(Vector2(_panel_rect.position.x + 8, _commit_total_rect.position.y),
		Vector2(_panel_rect.end.x - 8, _commit_total_rect.position.y), PANEL_BORDER, 1.0)

	# Label
	var label_y := _commit_total_rect.position.y + (_commit_total_rect.size.y + 12.0) / 2.0
	draw_string(font, Vector2(_commit_total_rect.position.x + 16, label_y),
		"FLUX COMMITMENT", HORIZONTAL_ALIGNMENT_LEFT, 180, 12, TEXT_MUTED)

	# Compute total for validation
	var total: int = 0
	for school in _commitment_schools:
		total += _pending_commitment.get(school, 0)

	# Per-school inputs
	var commit_entry_w := 100.0 + COMMIT_INPUT_W + 24.0
	var commit_total_w := _commitment_schools.size() * commit_entry_w
	var cx := _panel_rect.position.x + (_panel_rect.size.x - commit_total_w) / 2.0
	for i in _commitment_schools.size():
		var school: String = _commitment_schools[i]
		var label: String = SCHOOL_LABELS.get(school, school)
		var sc_color: Color = _get_school_color(school)
		var pct: int = _pending_commitment.get(school, 0)

		# School label (above input)
		draw_string(font, Vector2(cx, _commit_total_rect.position.y + 18),
			label, HORIZONTAL_ALIGNMENT_LEFT, 96, 11, sc_color)

		# Input box
		var input_rect := _commit_rects[i]
		var is_focused := i == _focused_commit_field
		var ibg := Color(0.08, 0.1, 0.15, 0.9) if is_focused else Color(0.04, 0.05, 0.07, 0.6)
		draw_rect(input_rect, ibg)
		draw_rect(input_rect, ACCENT if is_focused else PANEL_BORDER, false, 1.5 if is_focused else 1.0)

		# Value text (centered in input)
		var display_text: String = _commit_input_text if is_focused else str(pct)
		draw_string(font, Vector2(input_rect.position.x + 6, input_rect.position.y + 20),
			display_text, HORIZONTAL_ALIGNMENT_LEFT, COMMIT_INPUT_W - 12, 14, TEXT_PRIMARY)

		# "%" suffix
		draw_string(font, Vector2(input_rect.end.x + 4, input_rect.position.y + 20),
			"%", HORIZONTAL_ALIGNMENT_LEFT, 16, 12, TEXT_MUTED)

		cx += commit_entry_w

	# Total indicator (right side)
	var total_color := APPLY_ACTIVE if total == 100 else CLOSE_COLOR
	var total_x := cx + 12.0
	draw_string(font, Vector2(total_x, _commit_total_rect.position.y + _commit_total_rect.size.y / 2.0 + 6),
		"= %d%%" % total, HORIZONTAL_ALIGNMENT_LEFT, 80, 14, total_color)


func _draw_tooltip() -> void:
	var font := ThemeDB.fallback_font
	var ability: Dictionary = _filtered_abilities[_hovered_ability_idx]
	var ability_name: String = ability.get("name", "???")
	var school: String = ability.get("school", "")
	var ability_type: String = ability.get("ability_type", "")
	var delivery: String = ability.get("delivery", "")
	var flux_cost: String = ability.get("flux_cost", "")
	var cooldown: float = ability.get("cooldown", 0.0)
	var commit_time: float = ability.get("commit_time", 0.0)
	var description: String = ability.get("description", "")
	var affinity: String = ability.get("affinity", "off")
	var implemented: bool = ability.get("implemented", false)
	var school_color := _get_school_color(school)

	# Exact stats from server
	var flux_amount: float = ability.get("flux_amount", 0.0)
	var base_heal: float = ability.get("base_heal", 0.0)
	var base_damage: float = ability.get("base_damage", 0.0)
	var ability_range: float = ability.get("range", 0.0)
	var gcd: float = ability.get("gcd", 0.0)
	var detail_commit_time: float = ability.get("commit_time", 0.0)
	var zone_radius: float = ability.get("zone_radius", 0.0)
	var zone_duration: float = ability.get("zone_duration", 0.0)
	var zone_heal_tick: float = ability.get("zone_heal_tick", 0.0)

	# Build stat detail lines
	var stat_details: Array[String] = []
	if base_heal > 0.01:
		stat_details.append("Heal: %.0f" % base_heal)
	if base_damage > 0.01:
		stat_details.append("Damage: %.0f" % base_damage)
	if zone_heal_tick > 0.01:
		stat_details.append("Tick Heal: %.0f" % zone_heal_tick)
	if ability_range > 0.01:
		stat_details.append("Range: %.0fm" % ability_range)
	if zone_radius > 0.01:
		stat_details.append("Radius: %.0fm" % zone_radius)
	if zone_duration > 0.01:
		stat_details.append("Duration: %.1fs" % zone_duration)
	if detail_commit_time > 0.01:
		stat_details.append("Channel: %.1fs" % detail_commit_time)
	if gcd > 0.01:
		stat_details.append("GCD: %.1fs" % gcd)

	# Font sizes
	const TIP_NAME_SIZE := 18
	const TIP_INFO_SIZE := 13
	const TIP_STAT_SIZE := 13
	const TIP_DETAIL_SIZE := 12
	const TIP_AFF_SIZE := 11
	const TIP_DESC_SIZE := 13
	const TIP_WARN_SIZE := 12
	const TIP_LINE_H := 18.0
	const TIP_DESC_LINE_H := 17.0

	var desc_lines := _wrap_text(font, description, 360.0, TIP_DESC_SIZE)
	var tip_w := 400.0
	var detail_rows := ceili(stat_details.size() / 2.0) if stat_details.size() > 0 else 0
	var tip_h := 100.0 + detail_rows * TIP_LINE_H + desc_lines.size() * TIP_DESC_LINE_H + (22.0 if not implemented else 0.0)

	var card_rect := _card_rects[_hovered_ability_idx]
	var tip_x := card_rect.end.x + 8.0
	var tip_y := card_rect.position.y

	if tip_x + tip_w > size.x - 10:
		tip_x = card_rect.position.x - tip_w - 8.0
	tip_y = clampf(tip_y, 10.0, size.y - tip_h - 10.0)

	# Background
	draw_rect(Rect2(tip_x, tip_y, tip_w, tip_h), Color(0.02, 0.025, 0.04, 0.95))
	draw_rect(Rect2(tip_x, tip_y, tip_w, tip_h), Color(school_color, 0.4), false, 1.0)
	draw_rect(Rect2(tip_x, tip_y, 4, tip_h), school_color)

	var ty := tip_y + 22

	# Name
	draw_string(font, Vector2(tip_x + 14, ty), ability_name,
		HORIZONTAL_ALIGNMENT_LEFT, tip_w - 28, TIP_NAME_SIZE, school_color)
	ty += 22

	# School | Type | Delivery
	var info: Array[String] = []
	info.append(SCHOOL_LABELS.get(school, school))
	if ability_type != "":
		info.append(ability_type.capitalize())
	if delivery != "":
		info.append(delivery.capitalize())
	draw_string(font, Vector2(tip_x + 14, ty), " \u00b7 ".join(info),
		HORIZONTAL_ALIGNMENT_LEFT, tip_w - 28, TIP_INFO_SIZE, TEXT_MUTED)
	ty += TIP_LINE_H

	# Stats line (flux / commit / cooldown)
	var stats: Array[String] = []
	if flux_amount > 0.01:
		stats.append("Flux: %.0f" % flux_amount)
	elif flux_cost != "":
		stats.append("Flux: %s" % flux_cost.capitalize())
	if commit_time > 0.01:
		stats.append("%.1fs commit" % commit_time)
	if cooldown > 0.01:
		stats.append("%.0fs CD" % cooldown)
	if stats.size() > 0:
		draw_string(font, Vector2(tip_x + 14, ty), " | ".join(stats),
			HORIZONTAL_ALIGNMENT_LEFT, tip_w - 28, TIP_STAT_SIZE, Color(0.85, 0.75, 0.3, 0.8))
		ty += TIP_LINE_H

	# Detailed stat rows (2 per row)
	var col_w := (tip_w - 28) / 2.0
	var detail_i := 0
	while detail_i < stat_details.size():
		var left: String = stat_details[detail_i]
		draw_string(font, Vector2(tip_x + 14, ty), left,
			HORIZONTAL_ALIGNMENT_LEFT, col_w, TIP_DETAIL_SIZE, Color(0.7, 0.85, 0.75, 0.85))
		if detail_i + 1 < stat_details.size():
			var right: String = stat_details[detail_i + 1]
			draw_string(font, Vector2(tip_x + 14 + col_w, ty), right,
				HORIZONTAL_ALIGNMENT_LEFT, col_w, TIP_DETAIL_SIZE, Color(0.7, 0.85, 0.75, 0.85))
		detail_i += 2
		ty += TIP_LINE_H

	# Affinity
	var aff_color: Color
	match affinity:
		"primary": aff_color = Color(0.3, 0.8, 0.4)
		"secondary": aff_color = Color(0.7, 0.7, 0.3)
		_: aff_color = Color(0.7, 0.35, 0.3)
	var aff_labels := {"primary": "PRIMARY", "secondary": "SECONDARY", "off": "OFF-SPEC"}
	draw_string(font, Vector2(tip_x + 14, ty), aff_labels.get(affinity, "OFF-SPEC"),
		HORIZONTAL_ALIGNMENT_LEFT, tip_w - 28, TIP_AFF_SIZE, aff_color)
	ty += 16

	# Separator
	draw_line(Vector2(tip_x + 10, ty - 2), Vector2(tip_x + tip_w - 10, ty - 2), Color(PANEL_BORDER, 0.5), 1.0)
	ty += 6

	# Description
	for line in desc_lines:
		draw_string(font, Vector2(tip_x + 14, ty), line,
			HORIZONTAL_ALIGNMENT_LEFT, tip_w - 28, TIP_DESC_SIZE, Color(0.82, 0.84, 0.88, 0.9))
		ty += TIP_DESC_LINE_H

	if not implemented:
		ty += 6
		draw_string(font, Vector2(tip_x + 14, ty), "NOT YET IMPLEMENTED",
			HORIZONTAL_ALIGNMENT_LEFT, tip_w - 28, TIP_WARN_SIZE, Color(0.8, 0.4, 0.3, 0.9))


func _draw_drag_ghost() -> void:
	if _drag_ability.is_empty():
		return
	var font := ThemeDB.fallback_font
	var gw := 120.0
	var gh := 40.0
	var gx := _drag_pos.x - gw / 2.0
	var gy := _drag_pos.y - gh / 2.0
	var school: String = _drag_ability.get("school", "")
	var sc := _get_school_color(school)

	draw_rect(Rect2(gx, gy, gw, gh), Color(PANEL_BG, 0.85))
	draw_rect(Rect2(gx, gy, 3, gh), sc)
	draw_rect(Rect2(gx, gy, gw, gh), Color(sc, 0.6), false, 1.5)

	draw_string(font, Vector2(gx + 8, gy + 16), _drag_ability.get("name", "???"),
		HORIZONTAL_ALIGNMENT_LEFT, gw - 12, 11, TEXT_PRIMARY)

	if _hovered_loadout_slot >= 0:
		draw_string(font, Vector2(gx + 8, gy + 32),
			"-> Slot %s" % SLOT_KEYBINDS[_hovered_loadout_slot],
			HORIZONTAL_ALIGNMENT_LEFT, gw - 12, 9, ACCENT)


# =============================================================================
# Utility
# =============================================================================


func _wrap_text(font: Font, text: String, max_width: float, font_size: int) -> Array[String]:
	if text == "":
		return []
	var lines: Array[String] = []
	var words := text.split(" ")
	var current_line := ""
	for word in words:
		var test_line := (current_line + " " + word).strip_edges() if current_line != "" else word
		var w := font.get_string_size(test_line, HORIZONTAL_ALIGNMENT_LEFT, -1, font_size).x
		if w > max_width and current_line != "":
			lines.append(current_line)
			current_line = word
		else:
			current_line = test_line
	if current_line != "":
		lines.append(current_line)
	return lines
