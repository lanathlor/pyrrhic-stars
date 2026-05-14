class_name TestInventoryManager
extends GdUnitTestSuite

## Tests for InventoryManager autoload — state management and signal emission.

const InvManager := preload("res://scripts/autoload/inventory_manager.gd")

var _mgr: Node


func before_test() -> void:
	_mgr = InvManager.new()
	add_child(_mgr)


func after_test() -> void:
	if is_instance_valid(_mgr):
		_mgr.queue_free()


# =============================================================================
# Initial state
# =============================================================================


func test_initial_equipped_empty() -> void:
	assert_dict(_mgr.equipped).is_empty()


func test_initial_bag_empty() -> void:
	assert_array(_mgr.bag).is_empty()


func test_initial_computed_stats_zeroed() -> void:
	for key in ["hull", "output", "plating", "tempo", "identity", "mastery"]:
		assert_float(_mgr.computed_stats[key]).is_equal(0.0)


# =============================================================================
# _on_inventory_state — simulates receiving server data
# =============================================================================


func _make_item(slot_id: int, item_id: int, ilvl: int = 1) -> Dictionary:
	return {
		"slot_id": slot_id,
		"item_id": item_id,
		"def_id": "test_item_%d" % item_id,
		"ilvl": ilvl,
		"name": "Test Item %d" % item_id,
		"stat_lines": [{"stat": 0, "value": 10.0}],
	}


func _make_state(equipped: Array = [], bag: Array = [], stats: Dictionary = {}) -> Dictionary:
	var default_stats := {
		"hull": 0.0,
		"output": 0.0,
		"plating": 0.0,
		"tempo": 0.0,
		"identity": 0.0,
		"mastery": 0.0,
	}
	default_stats.merge(stats, true)
	return {"equipped": equipped, "bag": bag, "stats": default_stats}


func test_on_inventory_state_populates_equipped() -> void:
	var item := _make_item(0, 1)
	_mgr._on_inventory_state(_make_state([item]))
	assert_int(_mgr.equipped.size()).is_equal(1)
	assert_that(_mgr.equipped[0]["item_id"]).is_equal(1)


func test_on_inventory_state_populates_bag() -> void:
	var bag_item := _make_item(2, 5, 3)
	_mgr._on_inventory_state(_make_state([], [bag_item]))
	assert_int(_mgr.bag.size()).is_equal(1)
	assert_that(_mgr.bag[0]["item_id"]).is_equal(5)


func test_on_inventory_state_sets_computed_stats() -> void:
	_mgr._on_inventory_state(_make_state([], [], {"hull": 42.0, "output": 15.0}))
	assert_float(_mgr.computed_stats["hull"]).is_equal(42.0)
	assert_float(_mgr.computed_stats["output"]).is_equal(15.0)


func test_on_inventory_state_clears_previous_equipped() -> void:
	_mgr._on_inventory_state(_make_state([_make_item(0, 1), _make_item(1, 2)]))
	assert_int(_mgr.equipped.size()).is_equal(2)
	# Second update with only one item
	_mgr._on_inventory_state(_make_state([_make_item(0, 3)]))
	assert_int(_mgr.equipped.size()).is_equal(1)
	assert_that(_mgr.equipped[0]["item_id"]).is_equal(3)


func test_on_inventory_state_clears_previous_bag() -> void:
	_mgr._on_inventory_state(_make_state([], [_make_item(2, 5), _make_item(3, 6)]))
	assert_int(_mgr.bag.size()).is_equal(2)
	_mgr._on_inventory_state(_make_state())
	assert_int(_mgr.bag.size()).is_equal(0)


# =============================================================================
# get_equipped / get_stat
# =============================================================================


func test_get_equipped_returns_item() -> void:
	_mgr._on_inventory_state(_make_state([_make_item(2, 10)]))
	var item: Variant = _mgr.get_equipped(2)
	assert_that(item).is_not_null()
	assert_that(item["item_id"]).is_equal(10)


func test_get_equipped_returns_null_for_empty_slot() -> void:
	_mgr._on_inventory_state(_make_state([_make_item(0, 1)]))
	assert_that(_mgr.get_equipped(3)).is_null()


func test_get_stat_returns_value() -> void:
	_mgr._on_inventory_state(_make_state([], [], {"plating": 25.0}))
	assert_float(_mgr.get_stat("plating")).is_equal(25.0)


func test_get_stat_returns_zero_for_missing() -> void:
	assert_float(_mgr.get_stat("nonexistent")).is_equal(0.0)


# =============================================================================
# Signals
# =============================================================================


func test_inventory_changed_signal_emitted() -> void:
	var monitor := monitor_signals(_mgr)
	_mgr._on_inventory_state(_make_state())
	await assert_signal(monitor).is_emitted("inventory_changed")


func test_equipment_changed_signal_emitted() -> void:
	var monitor := monitor_signals(_mgr)
	_mgr._on_inventory_state(_make_state())
	await assert_signal(monitor).is_emitted("equipment_changed")


# =============================================================================
# Multiple equipped items keyed by slot
# =============================================================================


func test_all_six_slots_populated() -> void:
	var items: Array = []
	for slot_id in range(6):
		items.append(_make_item(slot_id, 100 + slot_id))
	_mgr._on_inventory_state(_make_state(items))
	assert_int(_mgr.equipped.size()).is_equal(6)
	for slot_id in range(6):
		assert_that(_mgr.get_equipped(slot_id)).is_not_null()
		assert_that(_mgr.get_equipped(slot_id)["item_id"]).is_equal(100 + slot_id)
