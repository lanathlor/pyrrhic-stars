class_name TestCameraCurrentAfterTopLevel
extends GdUnitTestSuite

## Diagnostic test for grey screen bug on arena zone transfer.
## Checks camera.current and _is_local() state after spawn.
## NOTE: in headless mode, camera behavior may differ from windowed.

const VANGUARD_SCENE := "res://scenes/controllers/vanguard/vanguard.tscn"


## After spawning a vanguard, camera.current must be true and the
## player must report _is_local() = true.
func test_vanguard_is_local_and_camera_current() -> void:
	var scene: PackedScene = load(VANGUARD_SCENE)
	var player: CharacterBody3D = auto_free(scene.instantiate()) as CharacterBody3D
	player.name = "Player_1"
	player.peer_id = 1
	add_child(player)
	player.global_position = Vector3(0.0, 0.1, 48.0)

	var cam: Camera3D = player.get_node("Camera3D")
	# Camera must be current for the local player
	assert_bool(cam.current).is_true()
	# _is_local must return true (NetworkManager.is_active=false in test)
	assert_bool(player._is_local()).is_true()
	# Camera must have top_level set
	assert_bool(cam.top_level).is_true()


## After a physics frame, the camera must have moved from origin.
func test_vanguard_camera_updates_after_physics() -> void:
	var scene: PackedScene = load(VANGUARD_SCENE)
	var player: CharacterBody3D = auto_free(scene.instantiate()) as CharacterBody3D
	player.name = "Player_1"
	player.peer_id = 1
	add_child(player)
	player.global_position = Vector3(0.0, 0.1, 48.0)
	player._net_position = Vector3(0.0, 0.1, 48.0)

	# Wait for _physics_process to run
	await get_tree().physics_frame
	await get_tree().physics_frame

	var cam: Camera3D = player.get_node("Camera3D")
	# Camera must NOT be at origin
	assert_float(cam.global_position.length()).is_greater(1.0)
