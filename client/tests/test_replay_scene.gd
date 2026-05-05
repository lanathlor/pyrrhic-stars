class_name TestReplayScene
extends GdUnitTestSuite

## Tests for ReplayScene — verifies that spawned player controllers behave as
## passive puppets during replay, not as player-controlled characters.
##
## BUG: Controllers use _is_local() which returns true when NetworkManager is
## inactive. In replay mode the network is off, so every spawned player thinks
## it's the local player — capturing input, activating its camera, and hiding
## the free-fly spectator camera.

var _scene: Node3D


func before_test() -> void:
	var script := load("res://scripts/replay/replay_scene.gd")
	_scene = auto_free(Node3D.new())
	_scene.set_script(script)
	add_child(_scene)
	await get_tree().process_frame


func _make_replay() -> RefCounted:
	var rd_script: GDScript = load("res://scripts/replay/replay_data.gd")
	var rd: RefCounted = rd_script.new()
	rd.instance_id = "test"
	rd.encounter_id = "test_boss"
	rd.zone_id = "arena"
	rd.tick_rate = 20
	rd.frame_count = 0
	rd.duration_ms = 0
	rd.outcome = "boss_win"
	return rd


# =============================================================================
# Spawned players must not take over the camera or capture input
# =============================================================================


func test_spawned_gunner_camera_not_current() -> void:
	var replay: RefCounted = _make_replay()
	_scene.start_replay(replay)
	await get_tree().process_frame

	var pdata := {"peer_id": 1, "class_name": "gunner", "pos": Vector3.ZERO, "rot_y": 0.0}
	_scene._spawn_replay_player(pdata)
	await get_tree().process_frame

	var player: CharacterBody3D = _scene._spawned_players[1]
	var player_cam: Camera3D = player.get_node("Head/Camera3D")
	assert_bool(player_cam.current).is_false()
	assert_bool(_scene._camera.current).is_true()


func test_spawned_vanguard_camera_not_current() -> void:
	var replay: RefCounted = _make_replay()
	_scene.start_replay(replay)
	await get_tree().process_frame

	var pdata := {"peer_id": 2, "class_name": "vanguard", "pos": Vector3.ZERO, "rot_y": 0.0}
	_scene._spawn_replay_player(pdata)
	await get_tree().process_frame

	var player: CharacterBody3D = _scene._spawned_players[2]
	var player_cam: Camera3D = player.camera
	assert_bool(player_cam.current).is_false()
	assert_bool(_scene._camera.current).is_true()


func test_spawned_blade_dancer_camera_not_current() -> void:
	var replay: RefCounted = _make_replay()
	_scene.start_replay(replay)
	await get_tree().process_frame

	var pdata := {"peer_id": 3, "class_name": "blade_dancer", "pos": Vector3.ZERO, "rot_y": 0.0}
	_scene._spawn_replay_player(pdata)
	await get_tree().process_frame

	var player: CharacterBody3D = _scene._spawned_players[3]
	var player_cam: Camera3D = player.camera
	assert_bool(player_cam.current).is_false()
	assert_bool(_scene._camera.current).is_true()


func test_spawned_player_input_disabled() -> void:
	var replay: RefCounted = _make_replay()
	_scene.start_replay(replay)
	await get_tree().process_frame

	var pdata := {"peer_id": 1, "class_name": "gunner", "pos": Vector3.ZERO, "rot_y": 0.0}
	_scene._spawn_replay_player(pdata)
	await get_tree().process_frame

	var player: CharacterBody3D = _scene._spawned_players[1]
	assert_bool(player.is_processing_unhandled_input()).is_false()
	assert_bool(player.is_processing_input()).is_false()


func test_mouse_mode_not_captured_after_spawn() -> void:
	var replay: RefCounted = _make_replay()
	_scene.start_replay(replay)
	await get_tree().process_frame

	# Ensure mouse is visible before spawn
	Input.set_mouse_mode(Input.MOUSE_MODE_VISIBLE)

	var pdata := {"peer_id": 1, "class_name": "gunner", "pos": Vector3(5, 0, 5), "rot_y": 0.0}
	_scene._spawn_replay_player(pdata)
	await get_tree().process_frame

	assert_int(Input.get_mouse_mode()).is_equal(Input.MOUSE_MODE_VISIBLE)


func test_spawned_player_interpolates_position() -> void:
	var replay: RefCounted = _make_replay()
	_scene.start_replay(replay)
	await get_tree().process_frame

	var start_pos := Vector3(5, 0, 5)
	var pdata := {"peer_id": 1, "class_name": "gunner", "pos": start_pos, "rot_y": 0.0}
	_scene._spawn_replay_player(pdata)
	await get_tree().process_frame

	# Simulate frame update with new position
	var new_pos := Vector3(10, 0, 10)
	var state := {
		"peer_id": 1,
		"pos": new_pos,
		"rot_y": 0.5,
		"health": 100.0,
		"state": 0,
		"class_name": "gunner",
		"username": "test",
		"anim_name": "rifle_run",
		"anim_speed": 1.0,
		"aim_pitch": 0.0
	}
	var player: CharacterBody3D = _scene._spawned_players[1]
	player.apply_server_state(state)

	# Let physics interpolate for several frames
	for i in 15:
		await get_tree().physics_frame

	# Player should have moved toward the new position (not stuck at start)
	assert_bool(player.global_position.distance_to(start_pos) > 0.1).is_true()
