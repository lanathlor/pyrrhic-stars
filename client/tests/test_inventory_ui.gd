class_name TestInventoryUI
extends GdUnitTestSuite

## Tests for inventory UI components — equipment panel, bag panel, mouse filter, layout.

const EquipScript := preload("res://scenes/ui/inventory_panel.gd")
const BagScript := preload("res://scenes/ui/bag_panel.gd")

var _equip: Control
var _bag: Control


func _cleanup() -> void:
	InventoryManager.equipped.clear()
	InventoryManager.bag.clear()
	InventoryManager.computed_stats = {
		"hull": 0.0,
		"output": 0.0,
		"plating": 0.0,
		"tempo": 0.0,
		"identity": 0.0,
		"mastery": 0.0,
	}


# =============================================================================
# Equipment panel
# =============================================================================


func _make_equip() -> void:
	_equip = auto_free(Control.new())
	_equip.set_script(EquipScript)
	_equip.size = Vector2(1920, 1080)
	add_child(_equip)


func test_equip_mouse_filter_is_ignore() -> void:
	_make_equip()
	assert_int(_equip.mouse_filter).is_equal(Control.MOUSE_FILTER_IGNORE)
	_cleanup()


func test_equip_has_6_slot_rects() -> void:
	_make_equip()
	_equip._compute_layout()
	assert_int(_equip._slot_rects.size()).is_equal(6)
	_cleanup()


func test_equip_slot_rects_are_58x58() -> void:
	_make_equip()
	_equip._compute_layout()
	for i in range(6):
		var r: Rect2 = _equip._slot_rects[i]
		assert_float(r.size.x).is_equal_approx(58.0, 0.1)
		assert_float(r.size.y).is_equal_approx(58.0, 0.1)
	_cleanup()


func test_equip_toggle_visibility() -> void:
	_make_equip()
	_equip.visible = false
	_equip.toggle()
	assert_bool(_equip.visible).is_true()
	_equip.toggle()
	assert_bool(_equip.visible).is_false()
	_cleanup()


func test_equip_initial_hover_state() -> void:
	_make_equip()
	assert_int(_equip._hovered_slot).is_equal(-1)
	_cleanup()


func test_equip_hover_detection() -> void:
	_make_equip()
	_equip.visible = true
	_equip._compute_layout()
	var r: Rect2 = _equip._slot_rects[0]
	_equip._update_hover(r.get_center())
	assert_int(_equip._hovered_slot).is_equal(0)
	_cleanup()


func test_equip_hover_slot_3() -> void:
	_make_equip()
	_equip.visible = true
	_equip._compute_layout()
	var r: Rect2 = _equip._slot_rects[3]
	_equip._update_hover(r.get_center())
	assert_int(_equip._hovered_slot).is_equal(3)
	_cleanup()


func test_equip_hover_outside_resets() -> void:
	_make_equip()
	_equip.visible = true
	_equip._compute_layout()
	var r: Rect2 = _equip._slot_rects[0]
	_equip._update_hover(r.get_center())
	assert_int(_equip._hovered_slot).is_equal(0)
	_equip._update_hover(Vector2(0, 0))
	assert_int(_equip._hovered_slot).is_equal(-1)
	_cleanup()


func test_equip_hidden_ignores_input() -> void:
	_make_equip()
	_equip.visible = false
	var event := InputEventMouseMotion.new()
	event.position = Vector2(500, 500)
	_equip._input(event)
	assert_int(_equip._hovered_slot).is_equal(-1)
	_cleanup()


func test_equip_layout_column_major() -> void:
	_make_equip()
	_equip._compute_layout()
	# Slots 0,1,2 in left column; 3,4,5 in right column
	var r0: Rect2 = _equip._slot_rects[0]
	var r1: Rect2 = _equip._slot_rects[1]
	var r3: Rect2 = _equip._slot_rects[3]
	# Slot 1 is below slot 0
	assert_float(r1.position.y).is_greater(r0.position.y)
	# Slot 3 is right of slot 0
	assert_float(r3.position.x).is_greater(r0.position.x)
	# Slots 0 and 3 at same Y
	assert_float(r0.position.y).is_equal_approx(r3.position.y, 0.1)
	_cleanup()


# =============================================================================
# Bag panel
# =============================================================================


func _make_bag() -> void:
	_bag = auto_free(Control.new())
	_bag.set_script(BagScript)
	_bag.size = Vector2(1920, 1080)
	add_child(_bag)


func test_bag_mouse_filter_is_ignore() -> void:
	_make_bag()
	assert_int(_bag.mouse_filter).is_equal(Control.MOUSE_FILTER_IGNORE)
	_cleanup()


func test_bag_toggle_visibility() -> void:
	_make_bag()
	_bag.visible = false
	_bag.toggle()
	assert_bool(_bag.visible).is_true()
	_bag.toggle()
	assert_bool(_bag.visible).is_false()
	_cleanup()


func test_bag_empty_has_grid() -> void:
	_make_bag()
	_bag._compute_layout()
	# At least 1 row x 4 cols = 4 slot rects
	assert_int(_bag._slot_rects.size()).is_equal(4)
	_cleanup()


func test_bag_3_items_has_4_rects() -> void:
	_make_bag()
	(
		InventoryManager
		. _on_inventory_state(
			{
				"equipped": [],
				"bag":
				[
					{
						"slot_id": 0,
						"item_id": 1,
						"def_id": "a",
						"ilvl": 1,
						"name": "A",
						"stat_lines": []
					},
					{
						"slot_id": 1,
						"item_id": 2,
						"def_id": "b",
						"ilvl": 1,
						"name": "B",
						"stat_lines": []
					},
					{
						"slot_id": 2,
						"item_id": 3,
						"def_id": "c",
						"ilvl": 1,
						"name": "C",
						"stat_lines": []
					},
				],
				"stats":
				{
					"hull": 0.0,
					"output": 0.0,
					"plating": 0.0,
					"tempo": 0.0,
					"identity": 0.0,
					"mastery": 0.0
				},
			}
		)
	)
	_bag._compute_layout()
	# 3 items → 1 row of 4 slots
	assert_int(_bag._slot_rects.size()).is_equal(4)
	_cleanup()


func test_bag_5_items_has_8_rects() -> void:
	_make_bag()
	var bag_items: Array = []
	for i in range(5):
		bag_items.append(
			{"slot_id": 0, "item_id": i, "def_id": "x", "ilvl": 1, "name": "X", "stat_lines": []}
		)
	(
		InventoryManager
		. _on_inventory_state(
			{
				"equipped": [],
				"bag": bag_items,
				"stats":
				{
					"hull": 0.0,
					"output": 0.0,
					"plating": 0.0,
					"tempo": 0.0,
					"identity": 0.0,
					"mastery": 0.0
				},
			}
		)
	)
	_bag._compute_layout()
	# 5 items → 2 rows x 4 cols = 8 rects
	assert_int(_bag._slot_rects.size()).is_equal(8)
	_cleanup()


func test_bag_hover_detection() -> void:
	_make_bag()
	_bag.visible = true
	_bag._compute_layout()
	var r: Rect2 = _bag._slot_rects[0]
	_bag._update_hover(r.get_center())
	assert_int(_bag._hovered_index).is_equal(0)
	_cleanup()


func test_bag_hover_outside_resets() -> void:
	_make_bag()
	_bag.visible = true
	_bag._compute_layout()
	var r: Rect2 = _bag._slot_rects[0]
	_bag._update_hover(r.get_center())
	_bag._update_hover(Vector2(0, 0))
	assert_int(_bag._hovered_index).is_equal(-1)
	_cleanup()


func test_bag_hidden_ignores_input() -> void:
	_make_bag()
	_bag.visible = false
	var event := InputEventMouseMotion.new()
	event.position = Vector2(500, 500)
	_bag._input(event)
	assert_int(_bag._hovered_index).is_equal(-1)
	_cleanup()


func test_bag_slot_rects_are_58x58() -> void:
	_make_bag()
	_bag._compute_layout()
	var r: Rect2 = _bag._slot_rects[0]
	assert_float(r.size.x).is_equal_approx(58.0, 0.1)
	assert_float(r.size.y).is_equal_approx(58.0, 0.1)
	_cleanup()


func test_bag_max_6_rows() -> void:
	_make_bag()
	var bag_items: Array = []
	for i in range(30):
		bag_items.append(
			{"slot_id": 0, "item_id": i, "def_id": "x", "ilvl": 1, "name": "X", "stat_lines": []}
		)
	(
		InventoryManager
		. _on_inventory_state(
			{
				"equipped": [],
				"bag": bag_items,
				"stats":
				{
					"hull": 0.0,
					"output": 0.0,
					"plating": 0.0,
					"tempo": 0.0,
					"identity": 0.0,
					"mastery": 0.0
				},
			}
		)
	)
	_bag._compute_layout()
	# 30 items → ceil(30/4) = 8, but capped at 6 rows → 24 rects
	assert_int(_bag._slot_rects.size()).is_equal(24)
	_cleanup()


# =============================================================================
# Both panels positioned correctly
# =============================================================================


func test_equip_panel_left_of_center() -> void:
	_make_equip()
	_equip._compute_layout()
	var pr: Rect2 = _equip._panel_rect
	var panel_center_x: float = pr.position.x + pr.size.x / 2.0
	assert_float(panel_center_x).is_less(960.0)
	_cleanup()


func test_bag_panel_right_of_center() -> void:
	_make_bag()
	_bag._compute_layout()
	var pr: Rect2 = _bag._panel_rect
	var panel_center_x: float = pr.position.x + pr.size.x / 2.0
	assert_float(panel_center_x).is_greater(960.0)
	_cleanup()
