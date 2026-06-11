class_name TestVanguardCameraSpawn
extends GdUnitTestSuite

## Reproduce: spawn vanguard at hub position, check camera follows.
## The vanguard camera has top_level=true so it positions itself in
## world space via _physics_process -> cam.update_camera(). If the
## camera stays at (0,0,0) after a physics frame, the player sees grey.

const VANGUARD_SCENE := preload("res://scenes/controllers/vanguard/vanguard.tscn")
const HUB_SPAWN := Vector3(14.0, -199.9, -80.0)


func test_camera_follows_player_after_spawn() -> void:
	var player: CharacterBody3D = auto_free(VANGUARD_SCENE.instantiate()) as CharacterBody3D
	player.name = "Player_1"
	player.peer_id = 1
	add_child(player)
	player.global_position = HUB_SPAWN

	# Let _physics_process run so the camera updates
	await get_tree().create_timer(0.1).timeout

	var cam: Camera3D = player.get_node("Camera3D")
	var dist := cam.global_position.distance_to(HUB_SPAWN)
	# Camera should be near the player (within camera_distance + offset)
	assert_float(dist).is_less(20.0)
	# Camera must NOT be at origin
	assert_float(cam.global_position.length()).is_greater(1.0)
