extends Control

## Character sheet — equipment grid + derived stats breakdown.
## Opened with [I]. Shows actual gameplay effects, not just raw numbers.

const HUD_BG := Color(0.02, 0.025, 0.035, 0.92)
const HUD_PANEL := Color(0.04, 0.05, 0.07, 0.55)
const HUD_BORDER := Color(0.28, 0.3, 0.36, 0.85)
const HUD_ACCENT := Color(0.32, 0.58, 0.92, 0.95)
const HUD_INSET := Color(0.08, 0.09, 0.12, 0.5)
const TEXT_PRIMARY := Color(0.9, 0.92, 0.96, 0.95)
const TEXT_MUTED := Color(0.63, 0.67, 0.74, 0.92)
const TEXT_DIM := Color(0.48, 0.53, 0.6, 0.95)
const EMPTY_SLOT := Color(0.06, 0.07, 0.09, 0.6)

const ICON_SIZE := 58.0
const ICON_GAP := 4.0
const COLS := 2
const ROWS := 3
const PAD := 12.0
const TITLE_H := 24.0
const STAT_LINE_H := 14.0
const SHEET_LINE_H := 16.0
const SECTION_H := 18.0
const SECTION_GAP := 6.0
const PANEL_W := 280.0

const SLOT_LABELS := ["FRM", "PWR", "WPN", "SEC", "AUG", "MOD"]
const SLOT_FULL := ["Frame", "Power Core", "Primary Weapon", "Secondary Tool", "Augment", "Module"]
const STAT_NAMES_ORDERED := ["hull", "output", "plating", "tempo", "identity", "mastery"]
const STAT_DISPLAY := ["Hull", "Output", "Plating", "Tempo", "Identity", "Mastery"]
const STAT_COLORS := {
	"hull": Color(0.3, 0.8, 0.3),
	"output": Color(0.9, 0.3, 0.2),
	"plating": Color(0.5, 0.6, 0.8),
	"tempo": Color(0.9, 0.8, 0.2),
	"identity": Color(0.6, 0.3, 0.9),
	"mastery": Color(0.2, 0.7, 0.9),
}

var _hovered_slot: int = -1
var _panel_rect := Rect2()
var _slot_rects: Array[Rect2] = []


func _ready() -> void:
	mouse_filter = Control.MOUSE_FILTER_IGNORE
	InventoryManager.equipment_changed.connect(func(): queue_redraw())


func _notification(what: int) -> void:
	if what == NOTIFICATION_RESIZED:
		_compute_layout()
		queue_redraw()


func _compute_layout() -> void:
	var vp := get_viewport_rect().size
	var grid_w := COLS * ICON_SIZE + (COLS - 1) * ICON_GAP
	var grid_h := ROWS * ICON_SIZE + (ROWS - 1) * ICON_GAP

	# Count non-zero stats for sheet height
	var n_core := 0
	var n_class := 0
	for si in range(3):
		if InventoryManager.computed_stats.get(STAT_NAMES_ORDERED[si], 0.0) > 0.0:
			n_core += 1
	for si in range(3, 6):
		if InventoryManager.computed_stats.get(STAT_NAMES_ORDERED[si], 0.0) > 0.0:
			n_class += 1

	var sheet_h := 0.0
	if n_core > 0:
		sheet_h += SECTION_GAP + SECTION_H + n_core * SHEET_LINE_H
	if n_class > 0:
		sheet_h += SECTION_GAP + SECTION_H + n_class * SHEET_LINE_H

	var panel_h := TITLE_H + PAD + grid_h + sheet_h + PAD
	var panel_x := vp.x / 2.0 - PANEL_W / 2.0 - 120.0
	var panel_y := vp.y / 2.0 - panel_h / 2.0
	_panel_rect = Rect2(panel_x, panel_y, PANEL_W, panel_h)

	_slot_rects.clear()
	var grid_x := panel_x + (PANEL_W - grid_w) / 2.0
	var grid_y := panel_y + TITLE_H + PAD
	for slot_id in range(6):
		var col := slot_id / 3
		var row := slot_id % 3
		var sx := grid_x + col * (ICON_SIZE + ICON_GAP)
		var sy := grid_y + row * (ICON_SIZE + ICON_GAP)
		_slot_rects.append(Rect2(sx, sy, ICON_SIZE, ICON_SIZE))


func _draw() -> void:
	if not visible:
		return
	_compute_layout()
	var font := ThemeDB.fallback_font

	# Panel background
	draw_rect(_panel_rect, HUD_BG)
	draw_rect(_panel_rect, HUD_BORDER, false, 1.0)

	# Title
	draw_string(
		font,
		Vector2(_panel_rect.position.x + PAD, _panel_rect.position.y + 17.0),
		"CHARACTER SHEET",
		HORIZONTAL_ALIGNMENT_LEFT,
		_panel_rect.size.x - PAD * 2.0,
		11,
		TEXT_MUTED
	)

	# Draw 6 equipment slots
	for i in range(6):
		_draw_slot(i, font)

	# Character sheet below grid
	var lx := _panel_rect.position.x + PAD
	var rw := _panel_rect.size.x - PAD * 2.0
	var sy := _slot_rects[2].end.y  # bottom of left column
	sy = _draw_core_section(font, lx, rw, sy)
	_draw_class_section(font, lx, rw, sy)

	# Tooltip
	if _hovered_slot >= 0:
		var item: Variant = InventoryManager.get_equipped(_hovered_slot)
		if item != null:
			_draw_item_tooltip(item, _slot_rects[_hovered_slot])


func _draw_core_section(font: Font, lx: float, rw: float, sy: float) -> float:
	var stats := InventoryManager.computed_stats
	var cls := InventoryManager.current_class
	var hull: float = stats.get("hull", 0.0)
	var output: float = stats.get("output", 0.0)
	var plating: float = stats.get("plating", 0.0)
	if hull <= 0.0 and output <= 0.0 and plating <= 0.0:
		return sy
	sy += SECTION_GAP
	_draw_section_header(font, lx, sy, rw, "Core")
	sy += SECTION_H
	if hull > 0.0:
		var base_hp: float = ItemData.CLASS_BASE_HP.get(cls, 150.0)
		_draw_sheet_line(
			font, Rect2(lx, sy, rw, 0), "Health", "%.0f HP" % (base_hp + hull), STAT_COLORS["hull"]
		)
		sy += SHEET_LINE_H
	if output > 0.0:
		_draw_sheet_line(
			font, Rect2(lx, sy, rw, 0), "Damage", "+%.0f%%" % output, STAT_COLORS["output"]
		)
		sy += SHEET_LINE_H
	if plating > 0.0:
		_draw_sheet_line(
			font,
			Rect2(lx, sy, rw, 0),
			"Mitigation",
			"-%.0f per hit" % plating,
			STAT_COLORS["plating"]
		)
		sy += SHEET_LINE_H
	return sy


func _draw_class_section(font: Font, lx: float, rw: float, sy: float) -> void:
	var stats := InventoryManager.computed_stats
	var cls := InventoryManager.current_class
	var tempo: float = stats.get("tempo", 0.0)
	var identity: float = stats.get("identity", 0.0)
	var mastery_val: float = stats.get("mastery", 0.0)
	if tempo <= 0.0 and identity <= 0.0 and mastery_val <= 0.0:
		return
	var class_label: String = cls.replace("_", " ").capitalize()
	sy += SECTION_GAP
	_draw_section_header(font, lx, sy, rw, class_label)
	sy += SECTION_H
	if tempo > 0.0:
		_draw_class_stat_line(font, Rect2(lx, sy, rw, 0), 3, tempo, cls)
		sy += SHEET_LINE_H
	if identity > 0.0:
		_draw_class_stat_line(font, Rect2(lx, sy, rw, 0), 4, identity, cls)
		sy += SHEET_LINE_H
	if mastery_val > 0.0:
		_draw_class_stat_line(font, Rect2(lx, sy, rw, 0), 5, mastery_val, cls)
		sy += SHEET_LINE_H


## Section header: "── Label ──────────"
func _draw_section_header(font: Font, x: float, y: float, w: float, label: String) -> void:
	var ly := y + SECTION_H / 2.0
	draw_line(Vector2(x, ly), Vector2(x + 12.0, ly), Color(HUD_BORDER, 0.5), 1.0)
	draw_string(
		font, Vector2(x + 16.0, y + 13.0), label, HORIZONTAL_ALIGNMENT_LEFT, w - 20.0, 9, TEXT_MUTED
	)
	var label_end := x + 18.0 + font.get_string_size(label, HORIZONTAL_ALIGNMENT_LEFT, -1, 9).x
	draw_line(Vector2(label_end, ly), Vector2(x + w, ly), Color(HUD_BORDER, 0.5), 1.0)


## Core stat line: "Label          Value" (label muted left, derived value colored right)
func _draw_sheet_line(
	font: Font, area: Rect2, label: String, value_str: String, color: Color
) -> void:
	var x := area.position.x
	var y := area.position.y
	var w := area.size.x
	draw_string(
		font, Vector2(x, y + 12.0), label, HORIZONTAL_ALIGNMENT_LEFT, w * 0.5, 10, TEXT_MUTED
	)
	draw_string(font, Vector2(x, y + 12.0), value_str, HORIZONTAL_ALIGNMENT_RIGHT, w, 10, color)


## Class stat line: "ClassName  value  description"
func _draw_class_stat_line(
	font: Font, area: Rect2, stat_id: int, value: float, cls: String
) -> void:
	var x := area.position.x
	var y := area.position.y
	var w := area.size.x
	var skey: String = STAT_NAMES_ORDERED[stat_id]
	var color: Color = STAT_COLORS.get(skey, TEXT_MUTED)
	var stat_name := ItemData.class_stat_name(stat_id, cls)
	var desc := ItemData.stat_effect_desc(stat_id, value, cls)
	draw_string(font, Vector2(x, y + 12.0), stat_name, HORIZONTAL_ALIGNMENT_LEFT, 100.0, 10, color)
	draw_string(
		font,
		Vector2(x + 100.0, y + 12.0),
		"%.0f" % value,
		HORIZONTAL_ALIGNMENT_RIGHT,
		36.0,
		10,
		color
	)
	if desc != "":
		draw_string(
			font,
			Vector2(x + 142.0, y + 12.0),
			desc,
			HORIZONTAL_ALIGNMENT_LEFT,
			w - 142.0,
			8,
			TEXT_DIM
		)


func _draw_slot(slot_id: int, font: Font) -> void:
	var r := _slot_rects[slot_id]
	var item: Variant = InventoryManager.get_equipped(slot_id)
	var hovered := _hovered_slot == slot_id

	if item != null:
		var bg := Color(0.08, 0.1, 0.14, 0.7) if not hovered else Color(0.12, 0.15, 0.22, 0.8)
		draw_rect(r, bg)
	else:
		draw_rect(r, EMPTY_SLOT)

	draw_rect(r, HUD_BORDER, false, 1.0)

	var inner := Rect2(r.position + Vector2(1.5, 1.5), r.size - Vector2(3.0, 3.0))
	if item != null:
		_draw_filled_slot(font, r, slot_id, item, hovered)
	else:
		draw_rect(inner, Color(HUD_BORDER, 0.2), false, 1.0)
		draw_string(
			font,
			Vector2(r.position.x + 2.0, r.position.y + ICON_SIZE / 2.0 + 4.0),
			SLOT_LABELS[slot_id],
			HORIZONTAL_ALIGNMENT_CENTER,
			ICON_SIZE - 4.0,
			9,
			TEXT_DIM
		)


func _draw_filled_slot(font: Font, r: Rect2, slot_id: int, item: Dictionary, hovered: bool) -> void:
	var inner := Rect2(r.position + Vector2(1.5, 1.5), r.size - Vector2(3.0, 3.0))
	var ilvl: int = item.get("ilvl", 1)
	var ic := ItemData.ilvl_color(ilvl)
	if hovered:
		draw_rect(inner, Color(ic, 0.8), false, 2.0)
	else:
		draw_rect(inner, Color(ic, 0.35), false, 1.5)
	draw_string(
		font,
		Vector2(r.end.x - 20.0, r.position.y + 12.0),
		"%d" % ilvl,
		HORIZONTAL_ALIGNMENT_RIGHT,
		16.0,
		9,
		ic
	)
	draw_string(
		font,
		Vector2(r.position.x + 4.0, r.end.y - 4.0),
		SLOT_LABELS[slot_id],
		HORIZONTAL_ALIGNMENT_LEFT,
		30.0,
		8,
		TEXT_DIM
	)
	_draw_slot_stat_dots(r, item.get("stat_lines", []))


func _draw_slot_stat_dots(r: Rect2, stat_lines: Array) -> void:
	if stat_lines.size() == 0:
		return
	var sid: int = stat_lines[0].get("stat", 0)
	var skey: String = STAT_NAMES_ORDERED[sid] if sid < 6 else "hull"
	var dot_r := Rect2(r.position.x + 4.0, r.position.y + 4.0, 6.0, 6.0)
	draw_rect(dot_r, STAT_COLORS.get(skey, TEXT_MUTED))
	if stat_lines.size() > 1:
		var sid2: int = stat_lines[1].get("stat", 0)
		var skey2: String = STAT_NAMES_ORDERED[sid2] if sid2 < 6 else "hull"
		draw_rect(
			Rect2(r.position.x + 12.0, r.position.y + 4.0, 6.0, 6.0),
			STAT_COLORS.get(skey2, TEXT_MUTED)
		)


func _draw_item_tooltip(item: Dictionary, slot_rect: Rect2) -> void:
	var font := ThemeDB.fallback_font
	var stat_lines: Array = item.get("stat_lines", [])
	var cls := InventoryManager.current_class

	var split := ItemData.merge_and_split(stat_lines)
	var p_stats: Array = split[0]
	var s_stats: Array = split[1]
	var has_divider := p_stats.size() > 0 and s_stats.size() > 0
	var merged_count := p_stats.size() + s_stats.size()

	var tip_w := 230.0
	var tip_h := 48.0 + merged_count * STAT_LINE_H + (8.0 if has_divider else 0.0)
	var tr := _position_tooltip(slot_rect, tip_w, tip_h)

	_draw_tooltip_frame(tr, item)
	_draw_tooltip_header(font, tr, item)
	_draw_tooltip_stats(font, tr.position, tip_w, p_stats, s_stats)


func _position_tooltip(slot_rect: Rect2, tip_w: float, tip_h: float) -> Rect2:
	var tip_x := slot_rect.position.x + ICON_SIZE / 2.0 - tip_w / 2.0
	var tip_y := slot_rect.position.y - tip_h - 6.0
	tip_x = clampf(tip_x, 4.0, get_viewport_rect().size.x - tip_w - 4.0)
	if tip_y < 4.0:
		tip_y = slot_rect.end.y + 6.0
	return Rect2(tip_x, tip_y, tip_w, tip_h)


func _draw_tooltip_frame(tr: Rect2, item: Dictionary) -> void:
	draw_rect(tr, HUD_PANEL)
	draw_rect(tr, HUD_BORDER, false, 1.0)
	var ic := ItemData.ilvl_color(item.get("ilvl", 1))
	draw_rect(
		Rect2(tr.position + Vector2(1.0, 1.0), tr.size - Vector2(2.0, 2.0)),
		Color(ic, 0.2),
		false,
		1.0
	)


func _draw_tooltip_header(font: Font, tr: Rect2, item: Dictionary) -> void:
	var tip_x := tr.position.x
	var tip_y := tr.position.y
	var tip_w := tr.size.x
	var ilvl: int = item.get("ilvl", 1)
	var ic := ItemData.ilvl_color(ilvl)
	var name_str: String = item.get("name", item.get("def_id", "???"))
	var slot_id: int = item.get("slot_id", 0)
	var slot_name: String = SLOT_FULL[slot_id] if slot_id < 6 else "?"
	draw_string(
		font,
		Vector2(tip_x + 8.0, tip_y + 14.0),
		name_str,
		HORIZONTAL_ALIGNMENT_LEFT,
		tip_w - 50.0,
		11,
		TEXT_PRIMARY
	)
	draw_string(
		font,
		Vector2(tip_x + tip_w - 40.0, tip_y + 14.0),
		"iLvl %d" % ilvl,
		HORIZONTAL_ALIGNMENT_RIGHT,
		32.0,
		9,
		ic
	)
	draw_string(
		font,
		Vector2(tip_x + 8.0, tip_y + 28.0),
		slot_name,
		HORIZONTAL_ALIGNMENT_LEFT,
		tip_w - 16.0,
		9,
		TEXT_DIM
	)


func _draw_tooltip_stats(
	font: Font, origin: Vector2, tip_w: float, p_stats: Array, s_stats: Array
) -> float:
	var cls := InventoryManager.current_class
	var has_divider := p_stats.size() > 0 and s_stats.size() > 0
	var tip_x := origin.x
	var sy := origin.y + 42.0
	for sl in p_stats:
		_draw_tooltip_stat(font, Rect2(tip_x, sy, tip_w, 0), sl, cls, 11)
		sy += STAT_LINE_H
	if has_divider:
		sy += 2.0
		draw_line(
			Vector2(tip_x + 8.0, sy), Vector2(tip_x + tip_w - 8.0, sy), Color(HUD_BORDER, 0.5), 1.0
		)
		sy += 6.0
	for sl in s_stats:
		_draw_tooltip_stat(font, Rect2(tip_x, sy, tip_w, 0), sl, cls, 9)
		sy += STAT_LINE_H
	return sy


## Draw a single stat line in a tooltip with class-aware name and effect description.
func _draw_tooltip_stat(font: Font, area: Rect2, sl: Dictionary, cls: String, fs: int) -> void:
	var tx := area.position.x
	var sy := area.position.y
	var tw := area.size.x
	var sid: int = sl.get("stat", 0)
	var sval: float = sl.get("value", 0.0)
	var skey: String = STAT_NAMES_ORDERED[sid] if sid < 6 else "hull"
	var scolor: Color = STAT_COLORS.get(skey, TEXT_MUTED)
	var display_name := ItemData.class_stat_name(sid, cls)
	draw_string(
		font,
		Vector2(tx + 8.0, sy),
		"+%.0f %s" % [sval, display_name],
		HORIZONTAL_ALIGNMENT_LEFT,
		tw * 0.5,
		fs,
		scolor
	)
	var desc := ItemData.stat_effect_desc(sid, sval, cls)
	if desc != "":
		draw_string(
			font,
			Vector2(tx + tw * 0.5, sy),
			desc,
			HORIZONTAL_ALIGNMENT_RIGHT,
			tw * 0.5 - 8.0,
			8,
			TEXT_DIM
		)


func _input(event: InputEvent) -> void:
	if not visible:
		return
	if event is InputEventMouseMotion:
		_update_hover(event.position)
	elif (
		event is InputEventMouseButton and event.pressed and event.button_index == MOUSE_BUTTON_LEFT
	):
		if _handle_click(event.position):
			get_viewport().set_input_as_handled()
		elif not _panel_rect.has_point(event.position):
			visible = false
			get_viewport().set_input_as_handled()


func _update_hover(pos: Vector2) -> void:
	var old := _hovered_slot
	_hovered_slot = -1
	for i in range(_slot_rects.size()):
		if _slot_rects[i].has_point(pos):
			_hovered_slot = i
			break
	if old != _hovered_slot:
		queue_redraw()


func _handle_click(pos: Vector2) -> bool:
	for i in range(_slot_rects.size()):
		if _slot_rects[i].has_point(pos):
			var item: Variant = InventoryManager.get_equipped(i)
			if item != null:
				InventoryManager.unequip_item(i)
			return true
	return false


func toggle() -> void:
	visible = not visible
	if visible:
		_compute_layout()
		queue_redraw()
