class_name CodexPanelDrawChrome
extends RefCounted

## Draw helpers for Codex Panel chrome: title bar, tabs, loadout slots, presets, commitment.

const T := preload("res://scenes/shared/hud/codex_panel_theme.gd")


static func draw_title(
	ci: CanvasItem,
	panel_rect: Rect2,
	close_rect: Rect2,
	hovered_close: bool,
) -> void:
	var font := ThemeDB.fallback_font
	var py := panel_rect.position.y

	ci.draw_string(
		font,
		Vector2(panel_rect.position.x + 16, py + 28),
		"HARMONIST CODEX",
		HORIZONTAL_ALIGNMENT_LEFT,
		panel_rect.size.x - 60,
		18,
		T.ACCENT
	)

	# Separator
	var sep_y := py + T.TITLE_H
	ci.draw_line(
		Vector2(panel_rect.position.x + 8, sep_y),
		Vector2(panel_rect.end.x - 8, sep_y),
		T.PANEL_BORDER,
		1.0
	)

	# Close button
	var cc := Color(1.0, 0.4, 0.4) if hovered_close else T.CLOSE_COLOR
	if hovered_close:
		ci.draw_rect(close_rect, Color(cc, 0.2))
	ci.draw_string(
		font,
		Vector2(close_rect.position.x + 6, close_rect.position.y + 18),
		"X",
		HORIZONTAL_ALIGNMENT_CENTER,
		close_rect.size.x - 12,
		14,
		cc
	)


static func draw_tabs(
	ci: CanvasItem,
	tab_rects: Array[Rect2],
	schools: Array[String],
	active_tab: int,
	hovered_tab: int,
) -> void:
	var font := ThemeDB.fallback_font
	for i in tab_rects.size():
		var rect := tab_rects[i]
		var is_active := i == active_tab
		var is_hovered := i == hovered_tab

		if is_active:
			ci.draw_rect(rect, T.TAB_ACTIVE_BG)
			ci.draw_rect(rect, T.ACCENT, false, 1.0)
		elif is_hovered:
			ci.draw_rect(rect, T.TAB_HOVER_BG)

		var label: String = "All" if i == 0 else T.SCHOOL_LABELS.get(schools[i], schools[i])
		var tc: Color
		if i == 0:
			tc = T.TEXT_PRIMARY if is_active else T.TEXT_MUTED
		else:
			var sc := T.get_school_color(schools[i])
			tc = sc if is_active else Color(sc, 0.6)

		ci.draw_string(
			font,
			Vector2(rect.position.x + 8, rect.position.y + 18),
			label,
			HORIZONTAL_ALIGNMENT_LEFT,
			rect.size.x - 16,
			10,
			tc
		)


static func draw_loadout(ci: CanvasItem, s: Dictionary) -> void:
	var panel_rect: Rect2 = s.panel
	var loadout_rect: Rect2 = s.loadout
	var loadout_slot_rects: Array = s.slots
	var pending_loadout: Array = s.pending
	var hovered_loadout_slot: int = s.hovered
	var dragging: bool = s.dragging
	var font := ThemeDB.fallback_font

	ci.draw_line(
		Vector2(panel_rect.position.x + 8, loadout_rect.position.y),
		Vector2(panel_rect.end.x - 8, loadout_rect.position.y),
		T.PANEL_BORDER,
		1.0
	)
	ci.draw_string(
		font,
		Vector2(loadout_rect.position.x + 16, loadout_rect.position.y + 14),
		"LOADOUT",
		HORIZONTAL_ALIGNMENT_LEFT,
		80,
		10,
		T.TEXT_MUTED
	)

	for i in 6:
		_draw_loadout_slot(
			ci, font, loadout_slot_rects[i], i, pending_loadout, hovered_loadout_slot, dragging
		)

	_draw_apply_button(ci, font, s.apply, s.hovered_apply, s.dirty)


static func draw_presets(ci: CanvasItem, s: Dictionary) -> void:
	var preset_rects: Array = s.rects
	var preset_delete_rects: Array = s.delete_rects
	var hovered_preset: int = s.hovered
	var hovered_preset_delete: int = s.hovered_delete
	var font := ThemeDB.fallback_font

	for i in preset_rects.size():
		if i >= AbilityCatalog.presets.size():
			break
		_draw_single_preset(
			ci,
			font,
			preset_rects[i],
			preset_delete_rects[i],
			AbilityCatalog.presets[i],
			i == hovered_preset,
			i == hovered_preset_delete
		)

	if s.naming:
		_draw_preset_name_input(ci, font, s.name_rect, s.name_input)

	_draw_preset_save_button(ci, font, s.save_rect, s.hovered_save, s.naming)


static func draw_commitment(ci: CanvasItem, s: Dictionary) -> void:
	var panel_rect: Rect2 = s.panel
	var commit_total_rect: Rect2 = s.total_rect
	var commit_rects: Array = s.rects
	var commitment_schools: Array = s.schools
	var pending_commitment: Dictionary = s.pending
	var focused_commit_field: int = s.focused
	var commit_input_text: String = s.input_text
	var font := ThemeDB.fallback_font

	ci.draw_line(
		Vector2(panel_rect.position.x + 8, commit_total_rect.position.y),
		Vector2(panel_rect.end.x - 8, commit_total_rect.position.y),
		T.PANEL_BORDER,
		1.0
	)

	var label_y := commit_total_rect.position.y + (commit_total_rect.size.y + 12.0) / 2.0
	ci.draw_string(
		font,
		Vector2(commit_total_rect.position.x + 16, label_y),
		"FLUX COMMITMENT",
		HORIZONTAL_ALIGNMENT_LEFT,
		180,
		12,
		T.TEXT_MUTED
	)

	var total: int = 0
	for school in commitment_schools:
		total += pending_commitment.get(school, 0)

	var commit_entry_w := 100.0 + T.COMMIT_INPUT_W + 24.0
	var commit_total_w := commitment_schools.size() * commit_entry_w
	var cx := panel_rect.position.x + (panel_rect.size.x - commit_total_w) / 2.0
	for i in commitment_schools.size():
		_draw_commit_field(
			ci,
			font,
			cx,
			commit_total_rect.position.y,
			commit_rects[i],
			commitment_schools[i],
			pending_commitment,
			i == focused_commit_field,
			commit_input_text
		)
		cx += commit_entry_w

	_draw_commit_total(ci, font, cx + 12.0, commit_total_rect, total)


static func draw_drag_ghost(
	ci: CanvasItem, drag_ability: Dictionary, drag_pos: Vector2, hovered_slot: int
) -> void:
	if drag_ability.is_empty():
		return
	var font := ThemeDB.fallback_font
	var gw := 120.0
	var gh := 40.0
	var gx := drag_pos.x - gw / 2.0
	var gy := drag_pos.y - gh / 2.0
	var school: String = drag_ability.get("school", "")
	var sc := T.get_school_color(school)
	ci.draw_rect(Rect2(gx, gy, gw, gh), Color(T.PANEL_BG, 0.85))
	ci.draw_rect(Rect2(gx, gy, 3, gh), sc)
	ci.draw_rect(Rect2(gx, gy, gw, gh), Color(sc, 0.6), false, 1.5)
	ci.draw_string(
		font,
		Vector2(gx + 8, gy + 16),
		drag_ability.get("name", "???"),
		HORIZONTAL_ALIGNMENT_LEFT,
		gw - 12,
		11,
		T.TEXT_PRIMARY
	)
	if hovered_slot >= 0:
		ci.draw_string(
			font,
			Vector2(gx + 8, gy + 32),
			"-> Slot %s" % T.SLOT_KEYBINDS[hovered_slot],
			HORIZONTAL_ALIGNMENT_LEFT,
			gw - 12,
			9,
			T.ACCENT
		)


static func _draw_loadout_slot(
	ci: CanvasItem,
	font: Font,
	rect: Rect2,
	i: int,
	pending_loadout: Array,
	hovered_loadout_slot: int,
	dragging: bool,
) -> void:
	var ability_id: String = pending_loadout[i] if i < pending_loadout.size() else ""
	var is_hovered := i == hovered_loadout_slot
	var is_drop := dragging and is_hovered

	var bg := T.SLOT_FILLED_BG if ability_id != "" else T.SLOT_EMPTY_BG
	if is_drop:
		bg = Color(T.ACCENT.r, T.ACCENT.g, T.ACCENT.b, 0.25)
	elif is_hovered:
		bg = Color(0.08, 0.1, 0.15, 0.8)

	ci.draw_rect(rect, bg)
	ci.draw_rect(rect, T.ACCENT if is_drop else T.PANEL_BORDER, false, 1.5 if is_drop else 1.0)
	ci.draw_string(
		font,
		Vector2(rect.position.x + 4, rect.position.y + 12),
		T.SLOT_KEYBINDS[i],
		HORIZONTAL_ALIGNMENT_LEFT,
		20,
		9,
		T.TEXT_MUTED
	)

	if ability_id != "":
		_draw_loadout_slot_filled(ci, font, rect, ability_id)
	else:
		ci.draw_string(
			font,
			Vector2(rect.position.x + 8, rect.position.y + 34),
			"Empty",
			HORIZONTAL_ALIGNMENT_LEFT,
			rect.size.x - 12,
			10,
			T.TEXT_DIM
		)


static func _draw_loadout_slot_filled(
	ci: CanvasItem, font: Font, rect: Rect2, ability_id: String
) -> void:
	var ability := AbilityCatalog.get_ability(ability_id)
	var sn: String = ability.get("name", ability_id)
	var sc: String = ability.get("school", "")
	ci.draw_rect(Rect2(rect.position.x, rect.position.y, 3, rect.size.y), T.get_school_color(sc))
	var parts := sn.split(" ", true, 1)
	ci.draw_string(
		font,
		Vector2(rect.position.x + 8, rect.position.y + 28),
		parts[0],
		HORIZONTAL_ALIGNMENT_LEFT,
		rect.size.x - 12,
		10,
		T.TEXT_PRIMARY
	)
	if parts.size() > 1:
		ci.draw_string(
			font,
			Vector2(rect.position.x + 8, rect.position.y + 42),
			parts[1],
			HORIZONTAL_ALIGNMENT_LEFT,
			rect.size.x - 12,
			10,
			T.TEXT_PRIMARY
		)


static func _draw_apply_button(
	ci: CanvasItem, font: Font, apply_rect: Rect2, hovered_apply: bool, is_dirty: bool
) -> void:
	var abg := (
		Color(T.APPLY_ACTIVE, 0.5 if hovered_apply else 0.3)
		if is_dirty
		else Color(T.APPLY_INACTIVE, 0.15)
	)
	ci.draw_rect(apply_rect, abg)
	ci.draw_rect(apply_rect, T.APPLY_ACTIVE if is_dirty else T.APPLY_INACTIVE, false, 1.0)
	ci.draw_string(
		font,
		Vector2(apply_rect.position.x + 6, apply_rect.position.y + 21),
		"APPLY",
		HORIZONTAL_ALIGNMENT_CENTER,
		apply_rect.size.x - 12,
		12,
		T.APPLY_ACTIVE if is_dirty else T.APPLY_INACTIVE
	)


static func _draw_single_preset(
	ci: CanvasItem,
	font: Font,
	rect: Rect2,
	del_rect: Rect2,
	preset: Dictionary,
	is_hovered: bool,
	is_del_hovered: bool,
) -> void:
	var pname: String = preset.get("name", "?")
	var bg := T.PRESET_HOVER_BG if is_hovered else T.PRESET_BG
	ci.draw_rect(rect, bg)
	ci.draw_rect(rect, T.ACCENT if is_hovered else T.PANEL_BORDER, false, 1.0)
	ci.draw_string(
		font,
		Vector2(rect.position.x + 6, rect.position.y + 16),
		pname,
		HORIZONTAL_ALIGNMENT_LEFT,
		rect.size.x - T.PRESET_DELETE_SIZE - 10,
		10,
		T.TEXT_PRIMARY if is_hovered else T.TEXT_MUTED
	)
	if is_hovered or is_del_hovered:
		var dc := T.PRESET_DELETE_COLOR if is_del_hovered else T.TEXT_DIM
		ci.draw_string(
			font,
			Vector2(del_rect.position.x + 2, del_rect.position.y + 11),
			"x",
			HORIZONTAL_ALIGNMENT_CENTER,
			T.PRESET_DELETE_SIZE - 4,
			9,
			dc
		)


static func _draw_preset_name_input(
	ci: CanvasItem, font: Font, rect: Rect2, name_input: String
) -> void:
	ci.draw_rect(rect, Color(0.08, 0.1, 0.15, 0.9))
	ci.draw_rect(rect, T.ACCENT, false, 1.5)
	ci.draw_string(
		font,
		Vector2(rect.position.x + 6, rect.position.y + 16),
		name_input + "_",
		HORIZONTAL_ALIGNMENT_LEFT,
		rect.size.x - 12,
		10,
		T.TEXT_PRIMARY
	)


static func _draw_preset_save_button(
	ci: CanvasItem, font: Font, save_rect: Rect2, hovered: bool, naming: bool
) -> void:
	var save_active := not naming
	var save_alpha := 0.4 if hovered else 0.2
	var sbg := (
		Color(T.PRESET_SAVE_COLOR, save_alpha) if save_active else Color(T.APPLY_INACTIVE, 0.15)
	)
	var border_c := T.PRESET_SAVE_COLOR if save_active else T.APPLY_INACTIVE
	ci.draw_rect(save_rect, sbg)
	ci.draw_rect(save_rect, border_c, false, 1.0)
	ci.draw_string(
		font,
		Vector2(save_rect.position.x + 4, save_rect.position.y + 16),
		"SAVE",
		HORIZONTAL_ALIGNMENT_CENTER,
		save_rect.size.x - 8,
		10,
		border_c
	)


static func _draw_commit_field(
	ci: CanvasItem,
	font: Font,
	cx: float,
	top_y: float,
	input_rect: Rect2,
	school: String,
	pending: Dictionary,
	is_focused: bool,
	input_text: String,
) -> void:
	var label: String = T.SCHOOL_LABELS.get(school, school)
	var sc_color: Color = T.get_school_color(school)
	ci.draw_string(
		font, Vector2(cx, top_y + 18), label, HORIZONTAL_ALIGNMENT_LEFT, 96, 11, sc_color
	)
	var ibg := Color(0.08, 0.1, 0.15, 0.9) if is_focused else Color(0.04, 0.05, 0.07, 0.6)
	ci.draw_rect(input_rect, ibg)
	ci.draw_rect(
		input_rect, T.ACCENT if is_focused else T.PANEL_BORDER, false, 1.5 if is_focused else 1.0
	)
	var display_text: String = input_text if is_focused else str(pending.get(school, 0))
	ci.draw_string(
		font,
		Vector2(input_rect.position.x + 6, input_rect.position.y + 20),
		display_text,
		HORIZONTAL_ALIGNMENT_LEFT,
		T.COMMIT_INPUT_W - 12,
		14,
		T.TEXT_PRIMARY
	)
	ci.draw_string(
		font,
		Vector2(input_rect.end.x + 4, input_rect.position.y + 20),
		"%",
		HORIZONTAL_ALIGNMENT_LEFT,
		16,
		12,
		T.TEXT_MUTED
	)


static func _draw_commit_total(
	ci: CanvasItem, font: Font, total_x: float, total_rect: Rect2, total: int
) -> void:
	var total_color := T.APPLY_ACTIVE if total == 100 else T.CLOSE_COLOR
	ci.draw_string(
		font,
		Vector2(total_x, total_rect.position.y + total_rect.size.y / 2.0 + 6),
		"= %d%%" % total,
		HORIZONTAL_ALIGNMENT_LEFT,
		80,
		14,
		total_color
	)
