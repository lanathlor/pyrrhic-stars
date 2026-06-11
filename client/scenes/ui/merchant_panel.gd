extends Control

## Mercenary Scrip Exchange panel. Opens when the player interacts with a
## merchant NPC in the hub. Extends Control for draw-based rendering,
## following inventory_panel.gd style. Hosted inside a CanvasLayer by main.gd.

signal closed

const UI_SURFACE := Color(0.035, 0.045, 0.065, 0.92)
const UI_BORDER := Color(0.22, 0.24, 0.30, 0.7)
const UI_BORDER_ACTIVE := Color(0.32, 0.58, 0.92, 0.95)
const UI_TEXT := Color(0.9, 0.93, 0.98, 0.96)
const UI_TEXT_MUTED := Color(0.6, 0.66, 0.75, 0.95)
const UI_TEXT_DIM := Color(0.48, 0.53, 0.6, 0.95)
const UI_LOCKED := Color(0.55, 0.35, 0.25, 0.95)
const UI_SUCCESS := Color(0.35, 0.78, 0.35, 0.95)
const UI_ERROR := Color(0.86, 0.28, 0.28, 0.96)
const UI_BUY_BG := Color(0.12, 0.18, 0.32, 0.85)
const UI_BUY_HOVER := Color(0.18, 0.26, 0.44, 0.95)
const UI_ITEM_BG := Color(0.04, 0.05, 0.07, 0.55)
const UI_ITEM_HOVER := Color(0.06, 0.08, 0.12, 0.7)
const UI_LOCKED_ITEM := Color(0.04, 0.05, 0.07, 0.4)
const UI_LOCKED_TEXT := Color(0.4, 0.43, 0.48, 0.6)
const UI_LOCKED_STAT := Color(0.35, 0.38, 0.42, 0.5)
const DIFF_UP := Color(0.3, 0.85, 0.3)
const DIFF_DOWN := Color(0.85, 0.3, 0.3)
const TIP_W := 240.0
const TIP_STAT_H := 14.0

const TIER_NAMES := ["Standard Issue", "Veteran", "Elite", "Commander"]
const STAT_NAMES := ["Hull", "Output", "Plating", "Tempo", "Identity", "Mastery"]
const SLOT_NAMES := [
	"Frame",
	"Power Core",
	"Primary Weapon",
	"Secondary / Tool",
	"Augment",
	"Module",
]
const STAT_COLORS := {
	0: Color(0.3, 0.8, 0.3),
	1: Color(0.9, 0.3, 0.2),
	2: Color(0.5, 0.6, 0.8),
	3: Color(0.9, 0.8, 0.2),
	4: Color(0.6, 0.3, 0.9),
	5: Color(0.2, 0.7, 0.9),
}

const PANEL_W := 600.0
const PANEL_H := 500.0
const PAD := 16.0
const TITLE_H := 36.0
const ITEM_ROW_H := 60.0
const BUY_BTN_W := 90.0
const FEEDBACK_DURATION := 2.5

var merchant_data: Dictionary = {}
var current_tier: int = -1
var hovered_item: int = -1
var feedback_text: String = ""
var feedback_timer: float = 0.0

var _is_open: bool = false
var _panel_rect := Rect2()
var _item_rects: Array[Rect2] = []
var _buy_rects: Array[Rect2] = []


func _ready() -> void:
	mouse_filter = Control.MOUSE_FILTER_IGNORE
	visible = false


func open_shop(tier: int) -> void:
	current_tier = tier
	merchant_data = {}
	hovered_item = -1
	feedback_text = ""
	feedback_timer = 0.0
	_is_open = true
	visible = true
	mouse_filter = Control.MOUSE_FILTER_STOP
	Input.set_mouse_mode(Input.MOUSE_MODE_VISIBLE)
	NetworkManager.send_merchant_interact(tier)
	queue_redraw()


func close_shop() -> void:
	_is_open = false
	visible = false
	mouse_filter = Control.MOUSE_FILTER_IGNORE
	current_tier = -1
	merchant_data = {}
	hovered_item = -1
	feedback_text = ""
	feedback_timer = 0.0
	closed.emit()


func _on_merchant_state(data: Dictionary) -> void:
	merchant_data = data
	queue_redraw()


func _on_buy_result(data: Dictionary) -> void:
	if not _is_open:
		return
	if data.get("success", false):
		feedback_text = "Purchased!"
		if merchant_data.size() > 0:
			merchant_data["balance"] = data.get("new_balance", merchant_data.get("balance", 0))
		# Re-request updated shop state so the item list refreshes
		NetworkManager.send_merchant_interact(current_tier)
	else:
		var err: String = data.get("error", "Purchase failed")
		feedback_text = err
	feedback_timer = FEEDBACK_DURATION
	queue_redraw()


func _on_scrip_awarded(data: Dictionary) -> void:
	var amount: int = data.get("amount", 0)
	feedback_text = "+%d Mercenary Scrip" % amount
	feedback_timer = FEEDBACK_DURATION
	if merchant_data.size() > 0:
		merchant_data["balance"] = data.get("new_balance", merchant_data.get("balance", 0))
	queue_redraw()


func _process(delta: float) -> void:
	if feedback_timer > 0.0:
		feedback_timer -= delta
		if feedback_timer <= 0.0:
			feedback_timer = 0.0
			feedback_text = ""
			queue_redraw()


func _input(event: InputEvent) -> void:
	if not _is_open:
		return

	if event is InputEventKey and event.pressed and not event.echo:
		if event.keycode == KEY_ESCAPE:
			close_shop()
			get_viewport().set_input_as_handled()
			return

	if event is InputEventMouseMotion:
		_update_hover(event.position)

	if event is InputEventMouseButton and event.pressed and event.button_index == MOUSE_BUTTON_LEFT:
		if _handle_click(event.position):
			get_viewport().set_input_as_handled()
		elif not _panel_rect.has_point(event.position):
			close_shop()
			get_viewport().set_input_as_handled()


func _update_hover(pos: Vector2) -> void:
	var old := hovered_item
	hovered_item = -1
	for i in range(_item_rects.size()):
		if _item_rects[i].has_point(pos):
			hovered_item = i
			break
	if old != hovered_item:
		queue_redraw()


func _handle_click(pos: Vector2) -> bool:
	var tier_data := _get_current_tier()
	if tier_data.size() == 0 or not tier_data.get("unlocked", false):
		return _panel_rect.has_point(pos)

	var items: Array = tier_data.get("items", [])
	for i in range(_buy_rects.size()):
		if _buy_rects[i].has_point(pos) and i < items.size():
			var item: Dictionary = items[i]
			NetworkManager.send_merchant_buy(current_tier, item.get("def_id", ""))
			return true

	return _panel_rect.has_point(pos)


func _get_current_tier() -> Dictionary:
	var tiers: Array = merchant_data.get("tiers", [])
	if current_tier >= 0 and current_tier < tiers.size():
		return tiers[current_tier]
	return {}


# =============================================================================
# Layout
# =============================================================================


func _compute_layout() -> void:
	var vp := get_viewport_rect().size
	var px := vp.x / 2.0 - PANEL_W / 2.0
	var py := vp.y / 2.0 - PANEL_H / 2.0
	_panel_rect = Rect2(px, py, PANEL_W, PANEL_H)

	_item_rects.clear()
	_buy_rects.clear()

	var tier_data := _get_current_tier()
	var items: Array = tier_data.get("items", [])
	var items_y := py + TITLE_H + PAD + 20.0
	for i in range(items.size()):
		var ry := items_y + i * (ITEM_ROW_H + 4.0)
		var item_r := Rect2(px + PAD, ry, PANEL_W - PAD * 2.0, ITEM_ROW_H)
		_item_rects.append(item_r)
		var buy_r := Rect2(
			item_r.end.x - BUY_BTN_W, item_r.position.y + 8.0, BUY_BTN_W, ITEM_ROW_H - 16.0
		)
		_buy_rects.append(buy_r)


# =============================================================================
# Drawing
# =============================================================================


func _draw() -> void:
	if not _is_open:
		return

	_compute_layout()
	var font := ThemeDB.fallback_font
	var pr := _panel_rect

	draw_rect(pr, UI_SURFACE)
	draw_rect(pr, UI_BORDER, false, 1.0)
	var w := pr.size.x - PAD * 2.0
	var lx := pr.position.x + PAD
	var ty := pr.position.y + 24.0
	var tier_name: String = (
		TIER_NAMES[current_tier]
		if current_tier >= 0 and current_tier < TIER_NAMES.size()
		else "Tier %d" % current_tier
	)
	draw_string(
		font,
		Vector2(lx, ty),
		"Mercenary Scrip Exchange : %s" % tier_name,
		HORIZONTAL_ALIGNMENT_LEFT,
		w - 120.0,
		14,
		UI_TEXT
	)
	draw_string(
		font,
		Vector2(lx, ty),
		"Scrip: %d" % merchant_data.get("balance", 0),
		HORIZONTAL_ALIGNMENT_RIGHT,
		w,
		13,
		UI_BORDER_ACTIVE
	)
	var div_y := pr.position.y + TITLE_H
	draw_line(Vector2(lx, div_y), Vector2(pr.end.x - PAD, div_y), UI_BORDER, 1.0)
	var tier_data := _get_current_tier()
	if merchant_data.size() == 0:
		draw_string(
			font,
			Vector2(lx, div_y + 40.0),
			"Loading...",
			HORIZONTAL_ALIGNMENT_CENTER,
			w,
			13,
			UI_TEXT_DIM
		)
	elif tier_data.size() == 0:
		draw_string(
			font,
			Vector2(lx, div_y + 40.0),
			"No data for this tier.",
			HORIZONTAL_ALIGNMENT_CENTER,
			w,
			13,
			UI_TEXT_DIM
		)
	else:
		_draw_tier(font, tier_data)
	if hovered_item >= 0:
		var items_tip: Array = tier_data.get("items", [])
		if hovered_item < items_tip.size():
			_draw_item_tooltip(font, items_tip[hovered_item])
	_draw_feedback(font)
	draw_string(
		font,
		Vector2(lx, pr.end.y - 4.0),
		"[Esc] Close",
		HORIZONTAL_ALIGNMENT_RIGHT,
		w,
		9,
		UI_TEXT_DIM
	)


func _draw_tier(font: Font, tier_data: Dictionary) -> void:
	var pr := _panel_rect
	var locked: bool = not tier_data.get("unlocked", false)
	var items: Array = tier_data.get("items", [])
	var price: int = tier_data.get("price", 0)
	var sub_y := pr.position.y + TITLE_H + PAD + 14.0
	var w := pr.size.x - PAD * 2.0
	if locked:
		draw_string(
			font,
			Vector2(pr.position.x + PAD, sub_y),
			"LOCKED",
			HORIZONTAL_ALIGNMENT_LEFT,
			100.0,
			11,
			UI_LOCKED
		)
		var req_str := "Requires higher overflux score"
		var max_score: int = merchant_data.get("max_score", 0)
		if max_score > 0:
			var pct: int = clampi(merchant_data.get("watermark", 0) * 100 / max_score, 0, 100)
			req_str = "Progress: %d%%" % pct
		draw_string(
			font,
			Vector2(pr.position.x + PAD, sub_y),
			req_str,
			HORIZONTAL_ALIGNMENT_RIGHT,
			w,
			11,
			UI_TEXT_MUTED
		)
	else:
		var ilvl: int = tier_data.get("ilvl", 1)
		draw_string(
			font,
			Vector2(pr.position.x + PAD, sub_y),
			"Item Level: %d" % ilvl,
			HORIZONTAL_ALIGNMENT_LEFT,
			200.0,
			11,
			UI_TEXT_MUTED
		)
		draw_string(
			font,
			Vector2(pr.position.x + PAD, sub_y),
			"Cost per item: %d Scrip" % price,
			HORIZONTAL_ALIGNMENT_RIGHT,
			w,
			11,
			UI_TEXT_MUTED
		)
	for i in range(items.size()):
		_draw_item_row(font, i, items[i], price, locked)


func _draw_item_row(
	font: Font, index: int, item: Dictionary, price: int, locked: bool = false
) -> void:
	if index >= _item_rects.size():
		return

	var r: Rect2 = _item_rects[index]
	var hovered := hovered_item == index
	var c_text: Color = UI_LOCKED_TEXT if locked else UI_TEXT
	var c_dim: Color = UI_LOCKED_STAT if locked else UI_TEXT_DIM
	var c_price: Color = UI_LOCKED_STAT if locked else UI_BORDER_ACTIVE

	# Row background
	if locked:
		draw_rect(r, Color(0.06, 0.07, 0.1, 0.5) if hovered else UI_LOCKED_ITEM)
		draw_rect(r, Color(UI_BORDER, 0.4), false, 1.0)
	else:
		draw_rect(r, UI_ITEM_HOVER if hovered else UI_ITEM_BG)
		draw_rect(r, UI_BORDER, false, 1.0)

	var x := r.position.x + 8.0
	var y := r.position.y

	# Item name
	var item_name: String = item.get("name", item.get("def_id", "???"))
	draw_string(
		font,
		Vector2(x, y + 16.0),
		item_name,
		HORIZONTAL_ALIGNMENT_LEFT,
		r.size.x - BUY_BTN_W - 20.0,
		12,
		c_text
	)

	# Slot name + price
	var slot_id: int = item.get("slot", 0)
	var slot_name: String = SLOT_NAMES[slot_id] if slot_id < SLOT_NAMES.size() else "?"
	draw_string(font, Vector2(x, y + 30.0), slot_name, HORIZONTAL_ALIGNMENT_LEFT, 200.0, 10, c_dim)
	draw_string(
		font,
		Vector2(x, y + 30.0),
		"%d Scrip" % price,
		HORIZONTAL_ALIGNMENT_RIGHT,
		r.size.x - BUY_BTN_W - 24.0,
		10,
		c_price
	)

	# Stats (compact, horizontal)
	var stats: Array = item.get("stats", [])
	var stat_x := x
	var stat_y := y + 46.0
	for s in stats:
		var sid: int = s.get("stat", 0)
		var sval: float = s.get("value", 0.0)
		var sname: String = STAT_NAMES[sid] if sid < STAT_NAMES.size() else "?"
		var scolor: Color = UI_LOCKED_STAT if locked else STAT_COLORS.get(sid, UI_TEXT_MUTED)
		var stat_str := "+%.0f %s" % [sval, sname]
		draw_string(
			font, Vector2(stat_x, stat_y), stat_str, HORIZONTAL_ALIGNMENT_LEFT, 120.0, 9, scolor
		)
		stat_x += font.get_string_size(stat_str, HORIZONTAL_ALIGNMENT_LEFT, -1, 9).x + 12.0

	# Buy button
	if index < _buy_rects.size():
		var br: Rect2 = _buy_rects[index]
		if locked:
			draw_rect(br, Color(UI_BUY_BG, 0.4))
			draw_rect(br, Color(UI_BORDER, 0.3), false, 1.0)
			draw_string(
				font,
				Vector2(br.position.x, br.position.y + br.size.y / 2.0 + 5.0),
				"Locked",
				HORIZONTAL_ALIGNMENT_CENTER,
				br.size.x,
				11,
				UI_LOCKED_TEXT
			)
		else:
			var mouse_pos := get_viewport().get_mouse_position()
			var btn_hover := hovered and br.has_point(mouse_pos)
			draw_rect(br, UI_BUY_HOVER if btn_hover else UI_BUY_BG)
			draw_rect(br, UI_BORDER_ACTIVE, false, 1.0)
			draw_string(
				font,
				Vector2(br.position.x, br.position.y + br.size.y / 2.0 + 5.0),
				"Buy : %d" % price,
				HORIZONTAL_ALIGNMENT_CENTER,
				br.size.x,
				11,
				UI_TEXT
			)


func _draw_feedback(font: Font) -> void:
	if feedback_text == "":
		return

	var fb_alpha := 1.0 if feedback_timer >= 0.5 else maxf(feedback_timer / 0.5, 0.0)
	var is_good := feedback_text.begins_with("+") or feedback_text == "Purchased!"
	var fb_color := Color(UI_SUCCESS, fb_alpha) if is_good else Color(UI_ERROR, fb_alpha)
	var pr := _panel_rect
	draw_string(
		font,
		Vector2(pr.position.x + PAD, pr.end.y - 16.0),
		feedback_text,
		HORIZONTAL_ALIGNMENT_CENTER,
		pr.size.x - PAD * 2.0,
		13,
		fb_color
	)


func _draw_item_tooltip(font: Font, item: Dictionary) -> void:
	var stats: Array = item.get("stats", [])
	var slot_id: int = item.get("slot", 0)
	var equipped: Variant = InventoryManager.get_equipped(slot_id)
	var eq_lines: Array = equipped.get("stat_lines", []) if equipped != null else []
	var compare_data := _compute_compare(stats, eq_lines) if equipped != null else []
	var split := ItemData.merge_and_split(stats)
	var p_stats: Array = split[0]
	var s_stats: Array = split[1]
	var has_div := p_stats.size() > 0 and s_stats.size() > 0
	var n := p_stats.size() + s_stats.size()
	var stats_h := n * TIP_STAT_H + (8.0 if has_div else 0.0)
	var cmp_h := (20.0 + compare_data.size() * TIP_STAT_H) if equipped != null else 0.0
	var tr := _position_tooltip(TIP_W, 48.0 + stats_h + cmp_h)
	draw_rect(tr, UI_SURFACE)
	draw_rect(tr, UI_BORDER, false, 1.0)
	var tx := tr.position.x + 8.0
	var item_name: String = item.get("name", item.get("def_id", "???"))
	var slot_name: String = SLOT_NAMES[slot_id] if slot_id < SLOT_NAMES.size() else "?"
	draw_string(
		font,
		Vector2(tx, tr.position.y + 14.0),
		item_name,
		HORIZONTAL_ALIGNMENT_LEFT,
		TIP_W - 16.0,
		11,
		UI_TEXT
	)
	draw_string(
		font,
		Vector2(tx, tr.position.y + 28.0),
		slot_name,
		HORIZONTAL_ALIGNMENT_LEFT,
		TIP_W - 16.0,
		9,
		UI_TEXT_DIM
	)
	var sy := tr.position.y + 42.0
	for sl in p_stats:
		_draw_tip_stat(font, tr.position.x, sy, sl, 11)
		sy += TIP_STAT_H
	if has_div:
		sy += 2.0
		draw_line(
			Vector2(tx, sy), Vector2(tr.position.x + TIP_W - 8.0, sy), Color(UI_BORDER, 0.5), 1.0
		)
		sy += 6.0
	for sl in s_stats:
		_draw_tip_stat(font, tr.position.x, sy, sl, 9)
		sy += TIP_STAT_H
	if equipped != null:
		sy += 4.0
		var eq_name: String = equipped.get("name", "???")
		var eq_ilvl: int = equipped.get("ilvl", 1)
		draw_string(
			font,
			Vector2(tx, sy),
			"Equipped: %s (iLvl %d)" % [eq_name, eq_ilvl],
			HORIZONTAL_ALIGNMENT_LEFT,
			TIP_W - 16.0,
			8,
			UI_TEXT_DIM
		)
		sy += TIP_STAT_H
		for cd in compare_data:
			var sid: int = cd["stat"]
			var diff: float = cd["diff"]
			var dc: Color = DIFF_UP if diff > 0.0 else DIFF_DOWN
			var prefix: String = "+" if diff > 0.0 else ""
			var sname: String = STAT_NAMES[sid] if sid < STAT_NAMES.size() else "?"
			draw_string(
				font,
				Vector2(tx, sy),
				"%s%.0f %s" % [prefix, diff, sname],
				HORIZONTAL_ALIGNMENT_LEFT,
				TIP_W - 16.0,
				9,
				dc
			)
			sy += TIP_STAT_H


func _position_tooltip(tip_w: float, tip_h: float) -> Rect2:
	var r: Rect2 = _item_rects[hovered_item]
	var vp := get_viewport_rect().size
	var tip_x := r.end.x + 8.0
	var tip_y := r.position.y
	if tip_x + tip_w > vp.x - 4.0:
		tip_x = r.position.x - tip_w - 8.0
	return Rect2(tip_x, clampf(tip_y, 4.0, vp.y - tip_h - 4.0), tip_w, tip_h)


func _draw_tip_stat(font: Font, tx: float, sy: float, sl: Dictionary, fs: int) -> void:
	var sid: int = sl.get("stat", 0)
	var sval: float = sl.get("value", 0.0)
	var dname := ItemData.class_stat_name(sid, InventoryManager.current_class)
	draw_string(
		font,
		Vector2(tx + 8.0, sy),
		"+%.0f %s" % [sval, dname],
		HORIZONTAL_ALIGNMENT_LEFT,
		TIP_W - 16.0,
		fs,
		STAT_COLORS.get(sid, UI_TEXT_MUTED)
	)


func _compute_compare(item_stats: Array, eq_lines: Array) -> Array:
	var item_m := {}
	for sl in item_stats:
		var sid: int = sl.get("stat", 0)
		item_m[sid] = item_m.get(sid, 0.0) + sl.get("value", 0.0)
	var eq_m := {}
	for sl in eq_lines:
		var sid: int = sl.get("stat", 0)
		eq_m[sid] = eq_m.get(sid, 0.0) + sl.get("value", 0.0)
	var result := []
	for sid in range(6):
		var bv: float = item_m.get(sid, 0.0)
		var ev: float = eq_m.get(sid, 0.0)
		if bv == 0.0 and ev == 0.0:
			continue
		var diff := bv - ev
		if absf(diff) < 0.5:
			continue
		result.append({"stat": sid, "diff": diff})
	return result
