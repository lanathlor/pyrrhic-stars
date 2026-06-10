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

	# Panel background
	draw_rect(pr, UI_SURFACE)
	draw_rect(pr, UI_BORDER, false, 1.0)

	# Title
	var tier_name: String = ""
	if current_tier >= 0 and current_tier < TIER_NAMES.size():
		tier_name = TIER_NAMES[current_tier]
	else:
		tier_name = "Tier %d" % current_tier
	var title_str := "Mercenary Scrip Exchange : %s" % tier_name
	draw_string(
		font,
		Vector2(pr.position.x + PAD, pr.position.y + 24.0),
		title_str,
		HORIZONTAL_ALIGNMENT_LEFT,
		pr.size.x - PAD * 2.0 - 120.0,
		14,
		UI_TEXT
	)

	# Balance (top-right)
	var balance: int = merchant_data.get("balance", 0)
	draw_string(
		font,
		Vector2(pr.position.x + PAD, pr.position.y + 24.0),
		"Scrip: %d" % balance,
		HORIZONTAL_ALIGNMENT_RIGHT,
		pr.size.x - PAD * 2.0,
		13,
		UI_BORDER_ACTIVE
	)

	# Divider below title
	var div_y := pr.position.y + TITLE_H
	draw_line(Vector2(pr.position.x + PAD, div_y), Vector2(pr.end.x - PAD, div_y), UI_BORDER, 1.0)

	# Content
	var tier_data := _get_current_tier()

	if merchant_data.size() == 0:
		_draw_centered_text(font, "Loading...", pr.position.y + TITLE_H + 40.0, 13, UI_TEXT_DIM)
	elif tier_data.size() == 0:
		_draw_centered_text(
			font, "No data for this tier.", pr.position.y + TITLE_H + 40.0, 13, UI_TEXT_DIM
		)
	elif not tier_data.get("unlocked", false):
		_draw_locked_tier(font, tier_data)
	else:
		_draw_unlocked_tier(font, tier_data)

	# Feedback text at bottom
	_draw_feedback(font)

	# Close hint
	draw_string(
		font,
		Vector2(pr.position.x + PAD, pr.end.y - 4.0),
		"[Esc] Close",
		HORIZONTAL_ALIGNMENT_RIGHT,
		pr.size.x - PAD * 2.0,
		9,
		UI_TEXT_DIM
	)


func _draw_centered_text(font: Font, text: String, y: float, fs: int, color: Color) -> void:
	draw_string(
		font,
		Vector2(_panel_rect.position.x + PAD, y),
		text,
		HORIZONTAL_ALIGNMENT_CENTER,
		_panel_rect.size.x - PAD * 2.0,
		fs,
		color
	)


func _draw_locked_tier(font: Font, _tier_data: Dictionary) -> void:
	var pr := _panel_rect
	var max_score: int = merchant_data.get("max_score", 0)
	var watermark: int = merchant_data.get("watermark", 0)
	var lock_y := pr.position.y + TITLE_H + PAD + 60.0

	draw_string(
		font,
		Vector2(pr.position.x + PAD, lock_y),
		"LOCKED",
		HORIZONTAL_ALIGNMENT_CENTER,
		pr.size.x - PAD * 2.0,
		18,
		UI_LOCKED
	)

	var req_str := "Requires higher overflux score to unlock."
	if max_score > 0:
		var pct: int = clampi(watermark * 100 / max_score, 0, 100)
		req_str = "Progress: %d%% overflux score (watermark: %d)" % [pct, watermark]

	draw_string(
		font,
		Vector2(pr.position.x + PAD, lock_y + 28.0),
		req_str,
		HORIZONTAL_ALIGNMENT_CENTER,
		pr.size.x - PAD * 2.0,
		12,
		UI_TEXT_MUTED
	)


func _draw_unlocked_tier(font: Font, tier_data: Dictionary) -> void:
	var pr := _panel_rect
	var items: Array = tier_data.get("items", [])
	var price: int = tier_data.get("price", 0)
	var ilvl: int = tier_data.get("ilvl", 1)

	# Sub-header
	var sub_y := pr.position.y + TITLE_H + PAD + 2.0
	draw_string(
		font,
		Vector2(pr.position.x + PAD, sub_y + 12.0),
		"Item Level: %d" % ilvl,
		HORIZONTAL_ALIGNMENT_LEFT,
		200.0,
		11,
		UI_TEXT_MUTED
	)
	draw_string(
		font,
		Vector2(pr.position.x + PAD, sub_y + 12.0),
		"Cost per item: %d Scrip" % price,
		HORIZONTAL_ALIGNMENT_RIGHT,
		pr.size.x - PAD * 2.0,
		11,
		UI_TEXT_MUTED
	)

	for i in range(items.size()):
		_draw_item_row(font, i, items[i], price)


func _draw_item_row(font: Font, index: int, item: Dictionary, price: int) -> void:
	if index >= _item_rects.size():
		return

	var r: Rect2 = _item_rects[index]
	var hovered := hovered_item == index

	# Row background
	var bg := UI_ITEM_HOVER if hovered else UI_ITEM_BG
	draw_rect(r, bg)
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
		UI_TEXT
	)

	# Slot name
	var slot_id: int = item.get("slot", 0)
	var slot_name: String = SLOT_NAMES[slot_id] if slot_id < SLOT_NAMES.size() else "?"
	draw_string(
		font, Vector2(x, y + 30.0), slot_name, HORIZONTAL_ALIGNMENT_LEFT, 200.0, 10, UI_TEXT_DIM
	)

	# Stats (compact, horizontal)
	var stats: Array = item.get("stats", [])
	var stat_x := x
	var stat_y := y + 46.0
	for s in stats:
		var sid: int = s.get("stat", 0)
		var sval: float = s.get("value", 0.0)
		var sname: String = STAT_NAMES[sid] if sid < STAT_NAMES.size() else "?"
		var scolor: Color = STAT_COLORS.get(sid, UI_TEXT_MUTED)
		var stat_str := "+%.0f %s" % [sval, sname]
		draw_string(
			font, Vector2(stat_x, stat_y), stat_str, HORIZONTAL_ALIGNMENT_LEFT, 120.0, 9, scolor
		)
		stat_x += font.get_string_size(stat_str, HORIZONTAL_ALIGNMENT_LEFT, -1, 9).x + 12.0

	# Buy button
	if index < _buy_rects.size():
		var br: Rect2 = _buy_rects[index]
		var mouse_pos := get_viewport().get_mouse_position()
		var btn_hover := hovered and br.has_point(mouse_pos)
		var btn_bg := UI_BUY_HOVER if btn_hover else UI_BUY_BG
		draw_rect(br, btn_bg)
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

	var pr := _panel_rect
	var fb_alpha := 1.0
	if feedback_timer < 0.5:
		fb_alpha = maxf(feedback_timer / 0.5, 0.0)

	var fb_color: Color
	if feedback_text.begins_with("+") or feedback_text == "Purchased!":
		fb_color = Color(UI_SUCCESS, fb_alpha)
	else:
		fb_color = Color(UI_ERROR, fb_alpha)

	draw_string(
		font,
		Vector2(pr.position.x + PAD, pr.end.y - 16.0),
		feedback_text,
		HORIZONTAL_ALIGNMENT_CENTER,
		pr.size.x - PAD * 2.0,
		13,
		fb_color
	)
