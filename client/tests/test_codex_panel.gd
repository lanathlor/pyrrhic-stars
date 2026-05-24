class_name TestCodexPanel
extends GdUnitTestSuite

## Tests for CodexPanel state management — open/close, filtering, loadout editing,
## dirty detection, and text wrapping.

const CodexPanelScript := preload("res://scenes/shared/hud/codex_panel.gd")

var _panel: Control
var _saved_catalog: Array
var _saved_loadout: Array


func before_test() -> void:
	# Save the real autoload state so we can restore it after test.
	_saved_catalog = AbilityCatalog.catalog.duplicate()
	_saved_loadout = AbilityCatalog.current_loadout.duplicate()
	_load_sample_catalog()

	_panel = auto_free(CodexPanelScript.new())
	_panel.name = "CodexPanel"
	_panel.size = Vector2(1920, 1080)
	add_child(_panel)
	await get_tree().process_frame


func after_test() -> void:
	# Restore autoload state.
	AbilityCatalog._on_catalog(_saved_catalog)
	AbilityCatalog._on_loadout(_saved_loadout)


func _load_sample_catalog() -> void:
	var catalog := [
		{id = "mending_surge", name = "Mending Surge", school = "bioarcanotechnic",
			ability_type = "enhancement", delivery = "direct", flux_cost = "high",
			description = "Emergency heal.", cooldown = 0.0, commit_time = 0.4,
			implemented = true, affinity = "primary"},
		{id = "mending_beam", name = "Mending Beam", school = "bioarcanotechnic",
			ability_type = "enhancement", delivery = "beam", flux_cost = "medium",
			description = "Channel heal.", cooldown = 0.0, commit_time = 2.0,
			implemented = true, affinity = "primary"},
		{id = "frost_ward", name = "Frost Ward", school = "frost",
			ability_type = "protection", delivery = "direct", flux_cost = "medium",
			description = "Shield.", cooldown = 12.0, commit_time = 0.2,
			implemented = true, affinity = "primary"},
		{id = "fireball", name = "Fireball", school = "fire",
			ability_type = "destruction", delivery = "zone", flux_cost = "high",
			description = "AoE explosion.", cooldown = 0.0, commit_time = 3.0,
			implemented = false, affinity = "off"},
	]
	AbilityCatalog._on_catalog(catalog)
	AbilityCatalog._on_loadout(["mending_surge", "mending_beam", "frost_ward", "", "", ""])


# =============================================================================
# Open / Close
# =============================================================================


func test_initial_state_hidden() -> void:
	assert_bool(_panel.visible).is_false()
	assert_bool(_panel.is_open()).is_false()


func test_open_shows_panel() -> void:
	_panel.open()
	assert_bool(_panel.visible).is_true()
	assert_bool(_panel.is_open()).is_true()


func test_close_hides_panel() -> void:
	_panel.open()
	_panel.close()
	assert_bool(_panel.visible).is_false()
	assert_bool(_panel.is_open()).is_false()


func test_open_copies_current_loadout() -> void:
	_panel.open()
	assert_bool(_panel._pending_loadout[0] == "mending_surge").is_true()
	assert_bool(_panel._pending_loadout[1] == "mending_beam").is_true()
	assert_bool(_panel._pending_loadout[2] == "frost_ward").is_true()
	assert_bool(_panel._pending_loadout[3] == "").is_true()


func test_open_resets_scroll() -> void:
	_panel.open()
	_panel._scroll_offset = 100.0
	_panel.close()
	_panel.open()
	assert_float(_panel._scroll_offset).is_equal(0.0)


func test_open_resets_drag() -> void:
	_panel.open()
	_panel._dragging = true
	_panel.close()
	_panel.open()
	assert_bool(_panel._dragging).is_false()


func test_close_stops_drag() -> void:
	_panel.open()
	_panel._dragging = true
	_panel.close()
	assert_bool(_panel._dragging).is_false()


# =============================================================================
# School tabs / filtering
# =============================================================================


func test_schools_built_on_open() -> void:
	_panel.open()
	# First tab is "" (All), then schools from catalog order
	assert_str(_panel._schools[0]).is_equal("")
	assert_bool(_panel._schools.has("bioarcanotechnic")).is_true()
	assert_bool(_panel._schools.has("frost")).is_true()
	assert_bool(_panel._schools.has("fire")).is_true()


func test_all_tab_shows_all_abilities() -> void:
	_panel.open()
	# _active_tab = 0 is "All"
	assert_int(_panel._filtered_abilities.size()).is_equal(4)


func test_filter_by_school() -> void:
	_panel.open()
	# Find the bioarcanotechnic tab index
	var tab_idx: int = _panel._schools.find("bioarcanotechnic")
	assert_int(tab_idx).is_greater(0)
	_panel._active_tab = tab_idx
	_panel._filter_abilities()
	assert_int(_panel._filtered_abilities.size()).is_equal(2)
	assert_str(_panel._filtered_abilities[0]["id"]).is_equal("mending_surge")
	assert_str(_panel._filtered_abilities[1]["id"]).is_equal("mending_beam")


func test_filter_frost_school() -> void:
	_panel.open()
	var tab_idx: int = _panel._schools.find("frost")
	_panel._active_tab = tab_idx
	_panel._filter_abilities()
	assert_int(_panel._filtered_abilities.size()).is_equal(1)
	assert_str(_panel._filtered_abilities[0]["id"]).is_equal("frost_ward")


# =============================================================================
# Loadout editing
# =============================================================================


func test_slot_for_ability_found() -> void:
	_panel.open()
	assert_int(_panel._get_slot_for_ability("mending_surge")).is_equal(0)
	assert_int(_panel._get_slot_for_ability("frost_ward")).is_equal(2)


func test_slot_for_ability_not_in_loadout() -> void:
	_panel.open()
	assert_int(_panel._get_slot_for_ability("fireball")).is_equal(-1)
	assert_int(_panel._get_slot_for_ability("nonexistent")).is_equal(-1)


func test_pending_loadout_edit() -> void:
	_panel.open()
	_panel._pending_loadout[3] = "fireball"
	assert_str(_panel._pending_loadout[3]).is_equal("fireball")


func test_remove_from_loadout() -> void:
	_panel.open()
	assert_str(_panel._pending_loadout[0]).is_equal("mending_surge")
	_panel._pending_loadout[0] = ""
	assert_str(_panel._pending_loadout[0]).is_equal("")


# =============================================================================
# Dirty detection
# =============================================================================


func test_not_dirty_when_unchanged() -> void:
	_panel.open()
	assert_bool(_panel._is_loadout_dirty()).is_false()


func test_dirty_when_slot_added() -> void:
	_panel.open()
	_panel._pending_loadout[3] = "fireball"
	assert_bool(_panel._is_loadout_dirty()).is_true()


func test_dirty_when_slot_removed() -> void:
	_panel.open()
	_panel._pending_loadout[0] = ""
	assert_bool(_panel._is_loadout_dirty()).is_true()


func test_dirty_when_slot_swapped() -> void:
	_panel.open()
	_panel._pending_loadout[0] = "frost_ward"
	_panel._pending_loadout[2] = "mending_surge"
	assert_bool(_panel._is_loadout_dirty()).is_true()


func test_not_dirty_after_revert() -> void:
	_panel.open()
	_panel._pending_loadout[0] = "fireball"
	assert_bool(_panel._is_loadout_dirty()).is_true()
	_panel._pending_loadout[0] = "mending_surge"
	assert_bool(_panel._is_loadout_dirty()).is_false()


# =============================================================================
# Text wrapping
# =============================================================================


func test_wrap_empty_string() -> void:
	var font: Font = ThemeDB.fallback_font
	var lines: Array[String] = _panel._wrap_text(font, "", 200.0, 10)
	assert_array(lines).is_empty()


func test_wrap_short_string_single_line() -> void:
	var font: Font = ThemeDB.fallback_font
	var lines: Array[String] = _panel._wrap_text(font, "Short text.", 500.0, 10)
	assert_int(lines.size()).is_equal(1)
	assert_str(lines[0]).is_equal("Short text.")


func test_wrap_long_string_multiple_lines() -> void:
	var font: Font = ThemeDB.fallback_font
	var long_text := "Massive single-target emergency heal. The biggest heal per commit in the game. Burns Flux fast but nothing else saves a dying ally this quickly."
	var lines: Array[String] = _panel._wrap_text(font, long_text, 200.0, 10)
	assert_int(lines.size()).is_greater(1)
	# All words should be preserved across lines
	var rejoined := " ".join(lines)
	assert_str(rejoined).is_equal(long_text)


func test_wrap_preserves_all_words() -> void:
	var font: Font = ThemeDB.fallback_font
	var text := "one two three four five six seven eight nine ten"
	var lines: Array[String] = _panel._wrap_text(font, text, 80.0, 10)
	var rejoined := " ".join(lines)
	assert_str(rejoined).is_equal(text)


# =============================================================================
# Signal emission
# =============================================================================


func test_loadout_applied_signal_emitted() -> void:
	_panel.open()
	_panel._pending_loadout[3] = "fireball"
	var result: Dictionary = {slots = []}
	_panel.loadout_applied.connect(func(slots: Array) -> void: result.slots = slots)
	# Simulate Apply click by emitting the signal directly
	_panel.loadout_applied.emit(_panel._pending_loadout.duplicate())
	assert_int(result.slots.size()).is_equal(6)
	assert_str(result.slots[3]).is_equal("fireball")


# =============================================================================
# Mouse filter state
# =============================================================================


func test_mouse_filter_ignore_when_closed() -> void:
	assert_int(_panel.mouse_filter).is_equal(Control.MOUSE_FILTER_IGNORE)


func test_mouse_filter_stop_when_open() -> void:
	_panel.open()
	assert_int(_panel.mouse_filter).is_equal(Control.MOUSE_FILTER_STOP)


func test_mouse_filter_ignore_after_close() -> void:
	_panel.open()
	_panel.close()
	assert_int(_panel.mouse_filter).is_equal(Control.MOUSE_FILTER_IGNORE)
