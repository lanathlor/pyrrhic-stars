extends Control

## Bag panel — grid of unequipped items. Opened with [B].
## Tooltip shows item stats + comparison with currently equipped item in that slot.

const HUD_BG := Color(0.02, 0.025, 0.035, 0.92)
const HUD_PANEL := Color(0.04, 0.05, 0.07, 0.55)
const HUD_BORDER := Color(0.28, 0.3, 0.36, 0.85)
const HUD_ACCENT := Color(0.32, 0.58, 0.92, 0.95)
const EMPTY_SLOT := Color(0.06, 0.07, 0.09, 0.6)
const TEXT_PRIMARY := Color(0.9, 0.92, 0.96, 0.95)
const TEXT_MUTED := Color(0.63, 0.67, 0.74, 0.92)
const TEXT_DIM := Color(0.48, 0.53, 0.6, 0.95)

const ICON_SIZE := 58.0
const ICON_GAP := 4.0
const COLS := 4
const MAX_ROWS := 6
const PAD := 12.0
const TITLE_H := 24.0
const STAT_LINE_H := 14.0

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

const DIFF_UP := Color(0.3, 0.85, 0.3)
const DIFF_DOWN := Color(0.85, 0.3, 0.3)
const DIFF_NEUTRAL := Color(0.5, 0.5, 0.5)

var merchant_open: bool = false
var _hovered_index: int = -1
var _panel_rect := Rect2()
var _slot_rects: Array[Rect2] = []


func _ready() -> void:
	mouse_filter = Control.MOUSE_FILTER_IGNORE
	InventoryManager.inventory_changed.connect(func(): queue_redraw())
	InventoryManager.scrip_changed.connect(func(_b): queue_redraw())


func _notification(what: int) -> void:
	if what == NOTIFICATION_RESIZED:
		_compute_layout()
		queue_redraw()


func _compute_layout() -> void:
	var vp := get_viewport_rect().size
	var item_count := InventoryManager.bag.size()
	var rows := clampi(ceili(float(max(item_count, 1)) / COLS), 1, MAX_ROWS)
	var grid_w := COLS * ICON_SIZE + (COLS - 1) * ICON_GAP
	var grid_h := rows * ICON_SIZE + (rows - 1) * ICON_GAP
	var panel_w := grid_w + PAD * 2.0
	var panel_h := TITLE_H + grid_h + PAD * 2.0
	var panel_x: float
	var panel_y: float
	if merchant_open:
		panel_x = vp.x / 2.0 + 300.0 + 16.0
		panel_y = vp.y / 2.0 + 40.0
		panel_x = minf(panel_x, vp.x - panel_w - 8.0)
		panel_y = minf(panel_y, vp.y - panel_h - 8.0)
	else:
		panel_x = vp.x / 2.0 + 120.0 - panel_w / 2.0
		panel_y = vp.y / 2.0 - panel_h / 2.0
	_panel_rect = Rect2(panel_x, panel_y, panel_w, panel_h)

	_slot_rects.clear()
	var grid_x := panel_x + PAD
	var grid_y := panel_y + TITLE_H + PAD
	var max_slots := rows * COLS
	for idx in range(max_slots):
		var col := idx % COLS
		var row := idx / COLS
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
		"BAG",
		HORIZONTAL_ALIGNMENT_LEFT,
		_panel_rect.size.x - PAD * 2.0,
		11,
		TEXT_MUTED
	)

	# Scrip balance (right-aligned in the title row)
	draw_string(
		font,
		Vector2(_panel_rect.position.x + PAD, _panel_rect.position.y + 17.0),
		"Scrip: %d" % InventoryManager.scrip_balance,
		HORIZONTAL_ALIGNMENT_RIGHT,
		_panel_rect.size.x - PAD * 2.0,
		11,
		HUD_ACCENT
	)

	# Draw slots
	for i in range(_slot_rects.size()):
		var item: Variant = InventoryManager.bag[i] if i < InventoryManager.bag.size() else null
		_draw_slot(i, item, font)

	# Tooltip
	if _hovered_index >= 0 and _hovered_index < InventoryManager.bag.size():
		_draw_bag_tooltip(InventoryManager.bag[_hovered_index], _slot_rects[_hovered_index])


func _draw_slot(idx: int, item: Variant, font: Font) -> void:
	var r := _slot_rects[idx]
	var hovered := _hovered_index == idx

	if item != null:
		var bg := Color(0.08, 0.1, 0.14, 0.7) if not hovered else Color(0.12, 0.15, 0.22, 0.8)
		draw_rect(r, bg)
	else:
		draw_rect(r, EMPTY_SLOT)

	draw_rect(r, HUD_BORDER, false, 1.0)

	var inner := Rect2(r.position + Vector2(1.5, 1.5), r.size - Vector2(3.0, 3.0))
	if item != null:
		_draw_filled_bag_slot(font, r, inner, item, hovered)
	else:
		draw_rect(inner, Color(HUD_BORDER, 0.15), false, 1.0)


func _draw_filled_bag_slot(
	font: Font, r: Rect2, inner: Rect2, item: Dictionary, hovered: bool
) -> void:
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
	var slot_id: int = item.get("slot_id", 0)
	var label: String = SLOT_LABELS[slot_id] if slot_id < 6 else "?"
	draw_string(
		font,
		Vector2(r.position.x + 4.0, r.end.y - 4.0),
		label,
		HORIZONTAL_ALIGNMENT_LEFT,
		30.0,
		8,
		TEXT_DIM
	)
	_draw_bag_stat_dots(r, item.get("stat_lines", []))


func _draw_bag_stat_dots(r: Rect2, stat_lines: Array) -> void:
	if stat_lines.size() == 0:
		return
	var sid: int = stat_lines[0].get("stat", 0)
	var skey: String = STAT_NAMES_ORDERED[sid] if sid < 6 else "hull"
	draw_rect(
		Rect2(r.position.x + 4.0, r.position.y + 4.0, 6.0, 6.0), STAT_COLORS.get(skey, TEXT_MUTED)
	)
	if stat_lines.size() > 1:
		var sid2: int = stat_lines[1].get("stat", 0)
		var skey2: String = STAT_NAMES_ORDERED[sid2] if sid2 < 6 else "hull"
		draw_rect(
			Rect2(r.position.x + 12.0, r.position.y + 4.0, 6.0, 6.0),
			STAT_COLORS.get(skey2, TEXT_MUTED)
		)


func _draw_bag_tooltip(item: Dictionary, slot_rect: Rect2) -> void:
	var font := ThemeDB.fallback_font
	var stat_lines: Array = item.get("stat_lines", [])
	var slot_id: int = item.get("slot_id", 0)
	var equipped: Variant = InventoryManager.get_equipped(slot_id)
	var eq_lines: Array = equipped.get("stat_lines", []) if equipped != null else []

	var split := ItemData.merge_and_split(stat_lines)
	var p_stats: Array = split[0]
	var s_stats: Array = split[1]
	var has_divider := p_stats.size() > 0 and s_stats.size() > 0
	var merged_count := p_stats.size() + s_stats.size()
	var compare_data := _compute_compare(stat_lines, eq_lines) if equipped != null else []

	var tip_w := 220.0
	var stats_h := merged_count * STAT_LINE_H + (8.0 if has_divider else 0.0)
	var compare_h := (20.0 + compare_data.size() * STAT_LINE_H) if equipped != null else 0.0
	var tip_h := 48.0 + stats_h + compare_h
	var tr := _position_bag_tooltip(slot_rect, tip_w, tip_h)

	_draw_bag_tooltip_frame(tr, item)
	_draw_bag_tooltip_header(font, tr, item, slot_id)
	var sy := _draw_bag_tooltip_stat_sections(font, tr.position, tip_w, p_stats, s_stats)
	if equipped != null:
		_draw_bag_tooltip_compare(font, Rect2(tr.position.x, sy, tip_w, 0), equipped, compare_data)


func _position_bag_tooltip(slot_rect: Rect2, tip_w: float, tip_h: float) -> Rect2:
	var tip_x := slot_rect.position.x + ICON_SIZE / 2.0 - tip_w / 2.0
	var tip_y := slot_rect.position.y - tip_h - 6.0
	tip_x = clampf(tip_x, 4.0, get_viewport_rect().size.x - tip_w - 4.0)
	if tip_y < 4.0:
		tip_y = slot_rect.end.y + 6.0
	return Rect2(tip_x, tip_y, tip_w, tip_h)


func _draw_bag_tooltip_frame(tr: Rect2, item: Dictionary) -> void:
	draw_rect(tr, HUD_PANEL)
	draw_rect(tr, HUD_BORDER, false, 1.0)
	var ic := ItemData.ilvl_color(item.get("ilvl", 1))
	draw_rect(
		Rect2(tr.position + Vector2(1.0, 1.0), tr.size - Vector2(2.0, 2.0)),
		Color(ic, 0.2),
		false,
		1.0
	)


func _draw_bag_tooltip_header(font: Font, tr: Rect2, item: Dictionary, slot_id: int) -> void:
	var tip_x := tr.position.x
	var tip_y := tr.position.y
	var tip_w := tr.size.x
	var ilvl: int = item.get("ilvl", 1)
	var ic := ItemData.ilvl_color(ilvl)
	var name_str: String = item.get("name", item.get("def_id", "???"))
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


func _draw_bag_tooltip_stat_sections(
	font: Font, origin: Vector2, tip_w: float, p_stats: Array, s_stats: Array
) -> float:
	var has_divider := p_stats.size() > 0 and s_stats.size() > 0
	var tip_x := origin.x
	var sy := origin.y + 42.0
	for sl in p_stats:
		_draw_stat_line(font, Rect2(tip_x, sy, tip_w, 0), sl, 11)
		sy += STAT_LINE_H
	if has_divider:
		sy += 2.0
		draw_line(
			Vector2(tip_x + 8.0, sy), Vector2(tip_x + tip_w - 8.0, sy), Color(HUD_BORDER, 0.5), 1.0
		)
		sy += 6.0
	for sl in s_stats:
		_draw_stat_line(font, Rect2(tip_x, sy, tip_w, 0), sl, 9)
		sy += STAT_LINE_H
	return sy


func _draw_bag_tooltip_compare(
	font: Font, area: Rect2, equipped: Dictionary, compare_data: Array
) -> void:
	var tip_x := area.position.x
	var tip_w := area.size.x
	var sy := area.position.y + 4.0
	var eq_ilvl: int = equipped.get("ilvl", 1)
	var eq_name: String = equipped.get("name", "???")
	draw_string(
		font,
		Vector2(tip_x + 8.0, sy),
		"Equipped: %s (iLvl %d)" % [eq_name, eq_ilvl],
		HORIZONTAL_ALIGNMENT_LEFT,
		tip_w - 16.0,
		8,
		TEXT_DIM
	)
	sy += STAT_LINE_H
	for cd in compare_data:
		var sid: int = cd["stat"]
		var diff: float = cd["diff"]
		var dc: Color = DIFF_UP if diff > 0.0 else DIFF_DOWN
		var prefix: String = "+" if diff > 0.0 else ""
		draw_string(
			font,
			Vector2(tip_x + 8.0, sy),
			"%s%.0f %s" % [prefix, diff, STAT_DISPLAY[sid] if sid < 6 else "?"],
			HORIZONTAL_ALIGNMENT_LEFT,
			tip_w - 16.0,
			9,
			dc
		)
		sy += STAT_LINE_H


func _draw_stat_line(font: Font, area: Rect2, sl: Dictionary, fs: int) -> void:
	var tx := area.position.x
	var sy := area.position.y
	var tw := area.size.x
	var sid: int = sl.get("stat", 0)
	var sval: float = sl.get("value", 0.0)
	var cls := InventoryManager.current_class
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


func _compute_compare(bag_lines: Array, eq_lines: Array) -> Array:
	var bag_m := {}
	for sl in bag_lines:
		var sid: int = sl.get("stat", 0)
		bag_m[sid] = bag_m.get(sid, 0.0) + sl.get("value", 0.0)
	var eq_m := {}
	for sl in eq_lines:
		var sid: int = sl.get("stat", 0)
		eq_m[sid] = eq_m.get(sid, 0.0) + sl.get("value", 0.0)
	var result := []
	for sid in range(6):
		var bv: float = bag_m.get(sid, 0.0)
		var ev: float = eq_m.get(sid, 0.0)
		if bv == 0.0 and ev == 0.0:
			continue
		var diff := bv - ev
		if absf(diff) < 0.5:
			continue
		result.append({"stat": sid, "diff": diff})
	return result


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
	var old := _hovered_index
	_hovered_index = -1
	for i in range(_slot_rects.size()):
		if _slot_rects[i].has_point(pos):
			_hovered_index = i
			break
	if old != _hovered_index:
		queue_redraw()


func _handle_click(pos: Vector2) -> bool:
	for i in range(_slot_rects.size()):
		if _slot_rects[i].has_point(pos):
			if i < InventoryManager.bag.size():
				var item: Dictionary = InventoryManager.bag[i]
				InventoryManager.equip_item(item["item_id"], item["slot_id"])
			return true
	return false


func toggle() -> void:
	visible = not visible
	if visible:
		_compute_layout()
		queue_redraw()
