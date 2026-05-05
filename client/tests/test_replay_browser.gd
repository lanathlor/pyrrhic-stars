class_name TestReplayBrowser
extends GdUnitTestSuite

## Tests for ReplayBrowser — row creation and instance loading with edge-case
## API responses. Exercises _create_row and _on_instances_loaded paths that
## crash when the server returns null-valued fields.

var _browser: CanvasLayer


func before_test() -> void:
	_browser = auto_free(load("res://scripts/replay/replay_browser.gd").new())
	add_child(_browser)
	await get_tree().process_frame


# =============================================================================
# _create_row — null fields in instance dict
# =============================================================================


func test_create_row_with_null_participants() -> void:
	# API returns "participants": null instead of an array
	var inst := {
		"instance_id": "abc",
		"encounter_id": "boss1",
		"started_at": "2026-01-01T12:00:00Z",
		"duration_ms": 60000,
		"outcome": "victory",
		"participants": null
	}
	var row := _browser._create_row(inst) as Button
	auto_free(row)
	assert_that(row).is_not_null()


func test_create_row_with_missing_participants() -> void:
	# API omits participants entirely
	var inst := {
		"instance_id": "abc",
		"encounter_id": "boss1",
		"started_at": "2026-01-01T12:00:00Z",
		"duration_ms": 60000,
		"outcome": "victory"
	}
	var row := _browser._create_row(inst) as Button
	auto_free(row)
	assert_that(row).is_not_null()


func test_create_row_with_null_outcome() -> void:
	var inst := {
		"instance_id": "abc",
		"encounter_id": "boss1",
		"started_at": "2026-01-01T12:00:00Z",
		"duration_ms": 60000,
		"outcome": null,
		"participants": []
	}
	var row := _browser._create_row(inst) as Button
	auto_free(row)
	assert_that(row).is_not_null()


func test_create_row_with_null_started_at() -> void:
	var inst := {
		"instance_id": "abc",
		"encounter_id": "boss1",
		"started_at": null,
		"duration_ms": 60000,
		"outcome": "victory",
		"participants": []
	}
	var row := _browser._create_row(inst) as Button
	auto_free(row)
	assert_that(row).is_not_null()


func test_create_row_with_null_duration() -> void:
	var inst := {
		"instance_id": "abc",
		"encounter_id": "boss1",
		"started_at": "2026-01-01T12:00:00Z",
		"duration_ms": null,
		"outcome": "victory",
		"participants": []
	}
	var row := _browser._create_row(inst) as Button
	auto_free(row)
	assert_that(row).is_not_null()


func test_create_row_with_all_null_fields() -> void:
	# Worst case: every optional field is null
	var inst := {
		"instance_id": null,
		"encounter_id": null,
		"started_at": null,
		"duration_ms": null,
		"outcome": null,
		"participants": null
	}
	var row := _browser._create_row(inst) as Button
	auto_free(row)
	assert_that(row).is_not_null()


# =============================================================================
# _on_instances_loaded — integration with null-laden data
# =============================================================================


func test_on_instances_loaded_with_null_participants_in_rows() -> void:
	# Simulates API returning multiple instances where participants is null
	var data: Array = [
		{
			"instance_id": "a",
			"encounter_id": "boss1",
			"started_at": "2026-01-01T12:00:00Z",
			"duration_ms": 30000,
			"outcome": "victory",
			"participants": null
		},
		{
			"instance_id": "b",
			"encounter_id": "boss2",
			"started_at": "2026-01-02T12:00:00Z",
			"duration_ms": 45000,
			"outcome": "wipe",
			"participants": null
		},
	]
	_browser._on_instances_loaded(data)
	assert_int(_browser._rows.size()).is_equal(2)


func test_on_instances_loaded_with_mixed_data() -> void:
	# Some rows have participants, some have null, some omit it
	var data: Array = [
		{
			"instance_id": "a",
			"encounter_id": "boss1",
			"started_at": "2026-01-01T12:00:00Z",
			"duration_ms": 30000,
			"outcome": "victory",
			"participants": [{"name": "Alice"}]
		},
		{
			"instance_id": "b",
			"encounter_id": "boss2",
			"started_at": "2026-01-02T12:00:00Z",
			"duration_ms": 45000,
			"outcome": "wipe",
			"participants": null
		},
		{
			"instance_id": "c",
			"encounter_id": "boss1",
			"started_at": "2026-01-03T12:00:00Z",
			"duration_ms": 120000,
			"outcome": "victory"
		},
	]
	_browser._on_instances_loaded(data)
	assert_int(_browser._rows.size()).is_equal(3)


func test_on_instances_loaded_empty() -> void:
	_browser._on_instances_loaded([])
	assert_int(_browser._rows.size()).is_equal(0)
	assert_bool(_browser._loading_label.visible).is_true()


func test_create_row_with_float_duration() -> void:
	# JSON parser may return numbers as float
	var inst := {
		"instance_id": "abc",
		"encounter_id": "boss1",
		"started_at": "2026-01-01T12:00:00Z",
		"duration_ms": 60000.0,
		"outcome": "victory",
		"participants": []
	}
	var row := _browser._create_row(inst) as Button
	auto_free(row)
	assert_that(row).is_not_null()
