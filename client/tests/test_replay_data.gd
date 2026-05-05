class_name TestReplayData
extends GdUnitTestSuite

## Tests for ReplayData.from_json — verifies resilience to null/missing fields
## that occur when the server omits or nullifies optional JSON values.

const _ReplayDataScript := preload("res://scripts/replay/replay_data.gd")

# =============================================================================
# from_json — null collection fields
# =============================================================================


func test_from_json_with_null_participants() -> void:
	var data := {
		"instance_id": "abc",
		"encounter_id": "boss1",
		"participants": null,
		"frames": [],
		"events": []
	}
	var rd = _ReplayDataScript.from_json(data)
	assert_that(rd).is_not_null()
	assert_array(rd.participants).is_empty()


func test_from_json_with_null_frames() -> void:
	var data := {
		"instance_id": "abc",
		"encounter_id": "boss1",
		"participants": [],
		"frames": null,
		"events": []
	}
	var rd = _ReplayDataScript.from_json(data)
	assert_that(rd).is_not_null()
	assert_array(rd.frames).is_empty()


func test_from_json_with_null_events() -> void:
	var data := {
		"instance_id": "abc",
		"encounter_id": "boss1",
		"participants": [],
		"frames": [],
		"events": null
	}
	var rd = _ReplayDataScript.from_json(data)
	assert_that(rd).is_not_null()
	assert_array(rd.events).is_empty()


func test_from_json_with_all_null_collections() -> void:
	var data := {"instance_id": "abc", "participants": null, "frames": null, "events": null}
	var rd = _ReplayDataScript.from_json(data)
	assert_that(rd).is_not_null()
	assert_array(rd.participants).is_empty()
	assert_array(rd.frames).is_empty()
	assert_array(rd.events).is_empty()


func test_from_json_with_missing_collections() -> void:
	# Keys entirely absent — the default [] should work
	var data := {"instance_id": "abc", "encounter_id": "boss1"}
	var rd = _ReplayDataScript.from_json(data)
	assert_that(rd).is_not_null()
	assert_array(rd.participants).is_empty()
	assert_array(rd.frames).is_empty()
	assert_array(rd.events).is_empty()


# =============================================================================
# from_json — null scalar fields
# =============================================================================


func test_from_json_with_null_scalars() -> void:
	var data := {
		"instance_id": null,
		"encounter_id": null,
		"zone_id": null,
		"tick_rate": null,
		"frame_count": null,
		"duration_ms": null,
		"outcome": null,
		"participants": [],
		"frames": [],
		"events": []
	}
	var rd = _ReplayDataScript.from_json(data)
	assert_that(rd).is_not_null()


func test_from_json_with_float_numerics() -> void:
	# JSON parsers may return numbers as float — must still work
	var data := {
		"instance_id": "abc",
		"tick_rate": 20.0,
		"duration_ms": 60000.0,
		"participants": [],
		"frames": [],
		"events": []
	}
	var rd = _ReplayDataScript.from_json(data)
	assert_int(rd.tick_rate).is_equal(20)
	assert_int(rd.duration_ms).is_equal(60000)


# =============================================================================
# Helpers — should survive empty data
# =============================================================================


func test_get_events_at_missing_tick() -> void:
	var data := {"instance_id": "abc", "participants": [], "frames": [], "events": []}
	var rd = _ReplayDataScript.from_json(data)
	assert_array(rd.get_events_at_tick(999)).is_empty()


func test_get_participant_name_unknown() -> void:
	var data := {"instance_id": "abc", "participants": [], "frames": [], "events": []}
	var rd = _ReplayDataScript.from_json(data)
	assert_str(rd.get_participant_name("unknown_id")).is_equal("unknown_id")


func test_total_time_str_zero() -> void:
	var data := {"instance_id": "abc", "participants": [], "frames": [], "events": []}
	var rd = _ReplayDataScript.from_json(data)
	assert_str(rd.total_time_str()).is_equal("0:00")
