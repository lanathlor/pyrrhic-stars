class_name CodexPanelLayout
extends RefCounted

## Computes all layout rectangles for the Codex Panel based on viewport size and state.

const T := preload("res://scenes/shared/hud/codex_panel_theme.gd")


## Compute all layout rects and return them in a dictionary.
## Parameters:
##   panel_size -- the Control's size (Vector2)
##   schools -- the filtered school list (Array[String])
##   filtered_abilities -- the currently displayed ability list (Array)
##   presets -- the preset list from AbilityCatalog
##   commitment_schools -- the schools used for commitment
##   scroll_offset -- current scroll position
##   naming_preset -- whether user is typing a preset name
## Returns dictionary with all rect data and updated scroll/max_scroll values.
static func compute(input: Dictionary) -> Dictionary:
	var panel_size: Vector2 = input.size
	var schools: Array = input.schools
	var scroll_offset: float = input.scroll
	var font := ThemeDB.fallback_font

	var panel_rect := Rect2(
		T.MARGIN_H,
		T.MARGIN_TOP,
		panel_size.x - T.MARGIN_H * 2,
		panel_size.y - T.MARGIN_TOP - T.MARGIN_BOTTOM
	)
	var px := panel_rect.position.x
	var py := panel_rect.position.y
	var pw := panel_rect.size.x
	var ph := panel_rect.size.y

	var close_rect := Rect2(px + pw - 40, py + 8, 28, 26)
	var tab_rects := _compute_tab_rects(px, py, schools, font)

	var tab_y := py + T.TITLE_H + 2
	var grid_y := tab_y + T.TAB_ROW_H + 4
	var bottom_h := T.LOADOUT_H + T.COMMIT_H + T.PRESET_ROW_H
	var loadout_y := py + ph - bottom_h
	var grid_rect := Rect2(px + 12, grid_y, pw - 24, loadout_y - grid_y - 8)
	var loadout_rect := Rect2(px, loadout_y, pw, T.LOADOUT_H)

	var grid_data := _compute_grid_rects(grid_rect, input.abilities, scroll_offset)
	var preset_data := _compute_preset_rects(px, loadout_y, input.presets, input.naming, font)
	var loadout_data := _compute_loadout_rects(px, pw, loadout_y)
	var commit_data := _compute_commit_rects(px, pw, loadout_y, input.commit_schools)

	return {
		"panel_rect": panel_rect,
		"close_rect": close_rect,
		"tab_rects": tab_rects,
		"grid_rect": grid_rect,
		"loadout_rect": loadout_rect,
		"card_rects": grid_data.cards,
		"max_scroll": grid_data.max_scroll,
		"scroll_offset": grid_data.scroll_offset,
		"loadout_slot_rects": loadout_data.slots,
		"apply_rect": loadout_data.apply,
		"commit_total_rect": commit_data.total_rect,
		"commit_rects": commit_data.rects,
		"preset_rects": preset_data.rects,
		"preset_delete_rects": preset_data.delete_rects,
		"preset_name_rect": preset_data.name_rect,
		"preset_save_rect": preset_data.save_rect,
	}


## Compute hover state from mouse position and cached rects.
## Returns a dictionary with all hover indices.
static func compute_hover(
	mouse: Vector2,
	close_rect: Rect2,
	apply_rect: Rect2,
	tab_rects: Array[Rect2],
	card_rects: Array[Rect2],
	grid_rect: Rect2,
	loadout_slot_rects: Array[Rect2],
	preset_rects: Array[Rect2],
	preset_delete_rects: Array[Rect2],
	preset_save_rect: Rect2,
	dragging: bool,
) -> Dictionary:
	var hovered_close := close_rect.has_point(mouse)
	var hovered_apply := apply_rect.has_point(mouse)

	var hovered_tab: int = -1
	for i in tab_rects.size():
		if tab_rects[i].has_point(mouse):
			hovered_tab = i
			break

	var hovered_ability_idx: int = -1
	if not dragging:
		for i in card_rects.size():
			var rect := card_rects[i]
			if rect.position.y < grid_rect.position.y or rect.end.y > grid_rect.end.y:
				continue
			if rect.has_point(mouse):
				hovered_ability_idx = i
				break

	var hovered_loadout_slot: int = -1
	for i in loadout_slot_rects.size():
		if loadout_slot_rects[i].has_point(mouse):
			hovered_loadout_slot = i
			break

	var ph := _compute_preset_hover(mouse, preset_rects, preset_delete_rects, preset_save_rect)

	return {
		"close": hovered_close,
		"apply": hovered_apply,
		"tab": hovered_tab,
		"ability_idx": hovered_ability_idx,
		"loadout_slot": hovered_loadout_slot,
		"preset": ph.preset,
		"preset_delete": ph.preset_delete,
		"preset_save": ph.preset_save,
	}


static func init_commitment(schools: Array[String]) -> Dictionary:
	if AbilityCatalog.current_commitment.is_empty():
		return {
			"bioarcanotechnic": 50,
			"biometabolic": 30,
			"frost": 10,
			"aerokinetic": 10,
		}
	var result := {}
	for school in schools:
		result[school] = AbilityCatalog.current_commitment.get(school, 0)
	return result


static func _compute_tab_rects(px: float, py: float, schools: Array, font: Font) -> Array[Rect2]:
	var tab_y := py + T.TITLE_H + 2
	var tab_rects: Array[Rect2] = []
	var tx := px + 12.0
	for i in schools.size():
		var label: String = "All" if i == 0 else T.SCHOOL_LABELS.get(schools[i], schools[i])
		var tw := font.get_string_size(label, HORIZONTAL_ALIGNMENT_LEFT, -1, 10).x + 18.0
		tw = maxf(tw, 44.0)
		tab_rects.append(Rect2(tx, tab_y + 2, tw, T.TAB_ROW_H - 4))
		tx += tw + T.TAB_GAP
	return tab_rects


static func _compute_grid_rects(
	grid_rect: Rect2, abilities: Array, scroll_offset: float
) -> Dictionary:
	var card_rects: Array[Rect2] = []
	var grid_w := grid_rect.size.x
	var cols := maxi(int(grid_w / (T.CARD_W + T.CARD_GAP)), 1)
	var grid_start_x := (
		grid_rect.position.x + (grid_w - cols * (T.CARD_W + T.CARD_GAP) + T.CARD_GAP) / 2.0
	)
	for i in abilities.size():
		var col := i % cols
		var row := i / cols
		var cx := grid_start_x + col * (T.CARD_W + T.CARD_GAP)
		var cy := grid_rect.position.y + row * (T.CARD_H + T.CARD_GAP) - scroll_offset
		card_rects.append(Rect2(cx, cy, T.CARD_W, T.CARD_H))
	var total_rows := ceili(float(abilities.size()) / float(cols))
	var total_height := total_rows * (T.CARD_H + T.CARD_GAP)
	var max_scroll := maxf(total_height - grid_rect.size.y, 0.0)
	var clamped := clampf(scroll_offset, 0.0, max_scroll)
	return {cards = card_rects, max_scroll = max_scroll, scroll_offset = clamped}


static func _compute_preset_rects(
	px: float, loadout_y: float, presets: Array, naming: bool, font: Font
) -> Dictionary:
	var rects: Array[Rect2] = []
	var delete_rects: Array[Rect2] = []
	var preset_y := loadout_y + 16.0
	var preset_x := px + 80.0
	for i in presets.size():
		var pname: String = presets[i].get("name", "?")
		var btn_w := font.get_string_size(pname, HORIZONTAL_ALIGNMENT_LEFT, -1, 10).x + 24.0
		btn_w = maxf(btn_w, 50.0)
		rects.append(Rect2(preset_x, preset_y, btn_w, T.PRESET_BTN_H))
		var del_x := preset_x + btn_w - T.PRESET_DELETE_SIZE - 2
		delete_rects.append(Rect2(del_x, preset_y + 2, T.PRESET_DELETE_SIZE, T.PRESET_DELETE_SIZE))
		preset_x += btn_w + T.PRESET_GAP
	var name_rect := Rect2()
	if naming:
		name_rect = Rect2(preset_x, preset_y, T.PRESET_NAME_INPUT_W, T.PRESET_BTN_H)
		preset_x += T.PRESET_NAME_INPUT_W + T.PRESET_GAP
	var save_rect := Rect2(preset_x, preset_y, T.PRESET_SAVE_W, T.PRESET_BTN_H)
	return {
		rects = rects, delete_rects = delete_rects, name_rect = name_rect, save_rect = save_rect
	}


static func _compute_loadout_rects(px: float, pw: float, loadout_y: float) -> Dictionary:
	var slots: Array[Rect2] = []
	var total_w := 6 * T.LOADOUT_SLOT_W + 5 * T.LOADOUT_GAP + T.APPLY_W + 20
	var lx := px + (pw - total_w) / 2.0
	var ly := loadout_y + T.PRESET_ROW_H + (T.LOADOUT_H - T.LOADOUT_SLOT_H) / 2.0
	for i in 6:
		slots.append(
			Rect2(
				lx + i * (T.LOADOUT_SLOT_W + T.LOADOUT_GAP), ly, T.LOADOUT_SLOT_W, T.LOADOUT_SLOT_H
			)
		)
	var apply := Rect2(
		lx + 6 * (T.LOADOUT_SLOT_W + T.LOADOUT_GAP) + 12,
		ly + (T.LOADOUT_SLOT_H - T.APPLY_H) / 2.0,
		T.APPLY_W,
		T.APPLY_H
	)
	return {slots = slots, apply = apply}


static func _compute_commit_rects(
	px: float, pw: float, loadout_y: float, schools: Array
) -> Dictionary:
	var commit_y := loadout_y + T.LOADOUT_H + T.PRESET_ROW_H
	var total_rect := Rect2(px, commit_y, pw, T.COMMIT_H)
	var rects: Array[Rect2] = []
	var entry_w := 100.0 + T.COMMIT_INPUT_W + 24.0
	var total_w := schools.size() * entry_w
	var cx := px + (pw - total_w) / 2.0
	for i in schools.size():
		var iy := commit_y + (T.COMMIT_H - T.COMMIT_INPUT_H) / 2.0
		rects.append(Rect2(cx + 100.0, iy, T.COMMIT_INPUT_W, T.COMMIT_INPUT_H))
		cx += entry_w
	return {total_rect = total_rect, rects = rects}


static func _compute_preset_hover(
	mouse: Vector2, rects: Array[Rect2], del_rects: Array[Rect2], save_rect: Rect2
) -> Dictionary:
	var preset: int = -1
	var preset_delete: int = -1
	for i in rects.size():
		if rects[i].has_point(mouse):
			preset = i
			if i < del_rects.size() and del_rects[i].has_point(mouse):
				preset_delete = i
			break
	return {
		preset = preset, preset_delete = preset_delete, preset_save = save_rect.has_point(mouse)
	}
