class_name TestOverfluxPanel
extends GdUnitTestSuite

## Tests for the overflux condition selection panel logic.

const OverfluxPanel := preload("res://scenes/ui/overflux_panel.gd")

var _panel: CanvasLayer


func before_test() -> void:
	_panel = auto_free(OverfluxPanel.new())
	add_child(_panel)
	await get_tree().process_frame


func after_test() -> void:
	pass


# =============================================================================
# Visibility
# =============================================================================


func test_panel_starts_hidden() -> void:
	assert_bool(_panel.visible).is_false()


func test_open_makes_visible() -> void:
	_panel.open()
	assert_bool(_panel.visible).is_true()


# =============================================================================
# Rank reset on open
# =============================================================================


func test_open_resets_ranks() -> void:
	# Set a rank, then reopen: should be back to 0
	_panel._on_rank_pressed("enemy_hp", 3)
	_panel.open()
	assert_int(_panel._selected_ranks["enemy_hp"]).is_equal(0)


# =============================================================================
# Score calculation
# =============================================================================


func test_rank_selection_updates_score() -> void:
	_panel.open()
	# enemy_hp has score_per_rank=4, setting rank to 3 -> score = 12
	_panel._on_rank_pressed("enemy_hp", 3)
	assert_str(_panel._score_label.text).is_equal("Overflux: 12")


# =============================================================================
# Signal: confirmed
# =============================================================================


func test_confirm_emits_signal() -> void:
	var signal_data: Array = []
	_panel.confirmed.connect(func(conditions: Array) -> void: signal_data.append_array(conditions))
	_panel.open()
	_panel._on_rank_pressed("enemy_hp", 2)
	_panel._confirm()
	assert_int(signal_data.size()).is_equal(1)
	assert_str(signal_data[0]["id"]).is_equal("enemy_hp")
	assert_int(signal_data[0]["rank"]).is_equal(2)


func test_confirm_with_no_conditions() -> void:
	var signal_data: Array = []
	_panel.confirmed.connect(func(conditions: Array) -> void: signal_data.append_array(conditions))
	_panel.open()
	# All ranks are 0 after open, so confirm should emit empty array
	_panel._confirm()
	assert_array(signal_data).is_empty()


func test_confirm_with_conditions() -> void:
	var signal_data: Array = []
	_panel.confirmed.connect(func(conditions: Array) -> void: signal_data.append_array(conditions))
	_panel.open()
	_panel._on_rank_pressed("enemy_hp", 5)
	_panel._confirm()
	assert_int(signal_data.size()).is_equal(1)
	assert_str(signal_data[0]["id"]).is_equal("enemy_hp")
	assert_int(signal_data[0]["rank"]).is_equal(5)


# =============================================================================
# Signal: cancelled
# =============================================================================


func test_cancel_emits_signal() -> void:
	var cancel_count: Array = []
	_panel.cancelled.connect(func() -> void: cancel_count.append(1))
	_panel.open()
	_panel._cancel()
	assert_int(cancel_count.size()).is_equal(1)
	# Panel should also be hidden after cancel
	assert_bool(_panel.visible).is_false()
