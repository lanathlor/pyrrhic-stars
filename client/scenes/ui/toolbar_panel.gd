extends Control

## Micro menu bar — small clickable buttons on the right side, aligned with the spell bar.
## Opens inventory panels and other menus via click.

signal equip_pressed
signal bag_pressed

const HUD_BG := Color(0.02, 0.025, 0.035, 0.82)
const HUD_BORDER := Color(0.28, 0.3, 0.36, 0.85)
const HOVER_BG := Color(0.08, 0.1, 0.14, 0.7)
const ACTIVE_BORDER := Color(0.32, 0.58, 0.92, 0.95)
const TEXT_PRIMARY := Color(0.9, 0.92, 0.96, 0.95)
const TEXT_MUTED := Color(0.63, 0.67, 0.74, 0.92)
const TOOLTIP_BG := Color(0.03, 0.035, 0.05, 0.92)

const BTN_SIZE := 24.0
const BTN_GAP := 3.0
const BAR_PAD := 4.0
const SPELL_SLOT_SIZE := 58.0
const SPELL_BOTTOM_MARGIN := 14.0

const BUTTONS := [
	{"label": "I", "tooltip": "Character [I]"},
	{"label": "B", "tooltip": "Bag [B]"},
]

var _hovered_btn: int = -1
var _btn_rects: Array[Rect2] = []
var _bar_rect := Rect2()
var _active: Array[bool] = [false, false]


func _ready() -> void:
	mouse_filter = Control.MOUSE_FILTER_IGNORE
	_compute_layout()


func _notification(what: int) -> void:
	if what == NOTIFICATION_RESIZED:
		_compute_layout()


func _compute_layout() -> void:
	var vp := get_viewport_rect().size
	var btn_count := BUTTONS.size()
	var total_w := btn_count * BTN_SIZE + (btn_count - 1) * BTN_GAP + BAR_PAD * 2.0
	var bar_x := vp.x * 0.75 - total_w / 2.0
	var bar_y := (
		vp.y - SPELL_SLOT_SIZE - SPELL_BOTTOM_MARGIN + (SPELL_SLOT_SIZE - BTN_SIZE) / 2.0 - BAR_PAD
	)
	var bar_h := BTN_SIZE + BAR_PAD * 2.0
	_bar_rect = Rect2(bar_x, bar_y, total_w, bar_h)

	_btn_rects.clear()
	for i in btn_count:
		var bx := bar_x + BAR_PAD + i * (BTN_SIZE + BTN_GAP)
		var by := bar_y + BAR_PAD
		_btn_rects.append(Rect2(bx, by, BTN_SIZE, BTN_SIZE))


func _process(_delta: float) -> void:
	queue_redraw()


func _draw() -> void:
	if _btn_rects.is_empty():
		return

	var font := ThemeDB.fallback_font

	# Bar background
	draw_rect(_bar_rect, HUD_BG)
	draw_rect(_bar_rect, HUD_BORDER, false, 1.0)

	for i in _btn_rects.size():
		var r := _btn_rects[i]
		var btn: Dictionary = BUTTONS[i]

		# Button background
		if _hovered_btn == i:
			draw_rect(r, HOVER_BG)
		else:
			draw_rect(r, Color(0.04, 0.05, 0.07, 0.45))

		# Border — active (panel open) or default
		if i < _active.size() and _active[i]:
			draw_rect(r, ACTIVE_BORDER, false, 1.5)
		else:
			draw_rect(r, HUD_BORDER, false, 1.0)

		# Label centered in button
		var label: String = btn["label"]
		var text_w := font.get_string_size(label, HORIZONTAL_ALIGNMENT_LEFT, -1, 11).x
		var tx := r.position.x + (BTN_SIZE - text_w) / 2.0
		var ty := r.position.y + BTN_SIZE / 2.0 + 4.0
		draw_string(font, Vector2(tx, ty), label, HORIZONTAL_ALIGNMENT_LEFT, -1, 11, TEXT_PRIMARY)

	# Tooltip for hovered button
	if _hovered_btn >= 0 and _hovered_btn < BUTTONS.size():
		var btn: Dictionary = BUTTONS[_hovered_btn]
		var tip: String = btn["tooltip"]
		var tip_font_size := 10
		var tip_w := (
			font.get_string_size(tip, HORIZONTAL_ALIGNMENT_LEFT, -1, tip_font_size).x + 12.0
		)
		var tip_h := 20.0
		var btn_r := _btn_rects[_hovered_btn]
		var tip_x := btn_r.position.x + BTN_SIZE / 2.0 - tip_w / 2.0
		var tip_y := btn_r.position.y - tip_h - 4.0
		var tip_rect := Rect2(tip_x, tip_y, tip_w, tip_h)
		draw_rect(tip_rect, TOOLTIP_BG)
		draw_rect(tip_rect, HUD_BORDER, false, 1.0)
		draw_string(
			font,
			Vector2(tip_x + 6.0, tip_y + 14.0),
			tip,
			HORIZONTAL_ALIGNMENT_LEFT,
			-1,
			tip_font_size,
			TEXT_MUTED
		)


func _input(event: InputEvent) -> void:
	if Input.get_mouse_mode() != Input.MOUSE_MODE_VISIBLE:
		if _hovered_btn != -1:
			_hovered_btn = -1
		return

	if event is InputEventMouseMotion:
		var prev := _hovered_btn
		_hovered_btn = _hit_test(event.position)
		if _hovered_btn != prev:
			queue_redraw()

	elif (
		event is InputEventMouseButton and event.pressed and event.button_index == MOUSE_BUTTON_LEFT
	):
		var hit := _hit_test(event.position)
		if hit == 0:
			equip_pressed.emit()
			get_viewport().set_input_as_handled()
		elif hit == 1:
			bag_pressed.emit()
			get_viewport().set_input_as_handled()


func _hit_test(pos: Vector2) -> int:
	for i in _btn_rects.size():
		if _btn_rects[i].has_point(pos):
			return i
	return -1


func update_active_state(equip_open: bool, bag_open: bool) -> void:
	_active = [equip_open, bag_open]
