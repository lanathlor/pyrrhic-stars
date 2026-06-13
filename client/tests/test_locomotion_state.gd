class_name TestLocomotionState
extends GdUnitTestSuite

## Guards CharacterModel.travel_locomotion: planar speed must select
## idle / run / sprint at the right thresholds, so the dedicated sprint
## state engages near sprint_speed and not during a normal jog.

const MODEL_SCENE := "res://scenes/shared/character_model/character_model.tscn"

const RUN_SPEED := 6.0
const SPRINT_SPEED := 9.0

var _model: Node3D


func before_test() -> void:
	var scene := load(MODEL_SCENE) as PackedScene
	_model = auto_free(scene.instantiate()) as Node3D
	add_child(_model)
	await get_tree().process_frame
	_model.setup_state_machine({"idle": "ual_idle", "run": "ual_jog", "sprint": "ual_sprint"})
	await get_tree().process_frame


func _state_after(speed: float) -> String:
	_model.travel_locomotion(speed, RUN_SPEED, SPRINT_SPEED)
	# travel() only switches when the target differs; force a couple frames
	# so the state machine settles on the requested node.
	await get_tree().process_frame
	await get_tree().process_frame
	return _model.state_playback.get_current_node()


func test_below_deadzone_is_idle() -> void:
	assert_str(await _state_after(0.2)).is_equal("idle")


func test_jog_speed_is_run() -> void:
	# run_speed (6.0) is 0.67 * sprint_speed, below the 0.85 sprint cutoff
	assert_str(await _state_after(RUN_SPEED)).is_equal("run")


func test_sprint_speed_is_sprint() -> void:
	assert_str(await _state_after(SPRINT_SPEED)).is_equal("sprint")


func test_threshold_boundary_is_sprint() -> void:
	# exactly at sprint_speed * 0.85 should already be sprinting
	assert_str(await _state_after(SPRINT_SPEED * 0.85)).is_equal("sprint")


func test_just_below_threshold_is_run() -> void:
	assert_str(await _state_after(SPRINT_SPEED * 0.85 - 0.1)).is_equal("run")
