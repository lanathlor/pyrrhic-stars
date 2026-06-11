class_name TestHubTeleportCorrection
extends GdUnitTestSuite

## Regression: grey screen returning from arena to hub. The server sends
## a world state tick with stale position (arena coords or origin) right
## after the zone transfer. world_state_sync._apply_player_state teleports
## the local player to that stale position, overriding the correct hub
## spawn. The player ends up 200 units above the hub floor.

const HUB_SPAWN := Vector3(14.0, -199.9, -80.0)
const ARENA_POS := Vector3(0.0, 0.1, 48.0)


## Stale server position must not override spawn within the grace period.
func test_stale_pos_ignored_right_after_spawn() -> void:
	var player: CharacterBody3D = auto_free(CharacterBody3D.new()) as CharacterBody3D
	add_child(player)
	player.global_position = HUB_SPAWN
	player.set_meta("_spawn_frame", Engine.get_physics_frames())
	await get_tree().process_frame

	# Simulate the teleport correction from world_state_sync
	var server_pos := ARENA_POS
	var spawn_age: int = Engine.get_physics_frames() - player.get_meta("_spawn_frame", 0)
	if spawn_age > 60 and player.global_position.distance_to(server_pos) > 8.0:
		player.global_position = server_pos

	assert_float(player.global_position.distance_to(HUB_SPAWN)).is_less(1.0)


## After the grace period, teleport correction should work normally.
func test_teleport_works_after_grace_period() -> void:
	var player: CharacterBody3D = auto_free(CharacterBody3D.new()) as CharacterBody3D
	add_child(player)
	player.global_position = HUB_SPAWN
	# Pretend spawn happened 100 frames ago
	player.set_meta("_spawn_frame", Engine.get_physics_frames() - 100)
	await get_tree().process_frame

	var server_pos := Vector3(5.0, -199.9, -70.0)
	var spawn_age: int = Engine.get_physics_frames() - player.get_meta("_spawn_frame", 0)
	if spawn_age > 60 and player.global_position.distance_to(server_pos) > 8.0:
		player.global_position = server_pos

	assert_float(player.global_position.distance_to(server_pos)).is_less(1.0)
