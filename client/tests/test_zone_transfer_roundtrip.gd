class_name TestZoneTransferRoundtrip
extends GdUnitTestSuite

## E2E-lite: instantiates the real main scene and drives zone transfers
## programmatically to catch freed-reference bugs, camera state issues,
## and double-load problems on hub -> arena -> hub -> ... cycles.
##
## NetworkManager.is_active is false, so get_my_id() returns 1.
## We call game_flow.on_zone_transfer() directly to simulate server messages.

const MAIN_SCENE := "res://scenes/main.tscn"

var _main: Node3D


func before_test() -> void:
	_main = auto_free(load(MAIN_SCENE).instantiate())
	# Prevent network connection and menu flow
	_main.dev_mode = false
	add_child(_main)
	# Let _ready() and one frame complete
	await get_tree().process_frame
	await get_tree().process_frame


## Simulate entering hub state directly (skip menu/char select).
func _enter_open_world() -> void:
	_main.game_flow.enter_hub()
	await get_tree().process_frame
	await get_tree().process_frame


## Simulate a zone transfer to arena (type=1) as if the server sent it.
func _transfer_to_arena() -> void:
	_main.game_flow.on_zone_transfer(1, 1)  # type=instanced, peer=1
	# Let queue_free and scene loading settle
	await get_tree().process_frame
	await get_tree().process_frame


## Simulate a zone transfer to hub (type=0) as if the server sent it.
func _transfer_to_hub() -> void:
	_main.game_flow.on_zone_transfer(0, 1)  # type=open_world, peer=1
	await get_tree().process_frame
	await get_tree().process_frame


## Assert that the local player exists and has a valid camera.
func _assert_player_alive(_label: String) -> void:
	var my_id: int = NetworkManager.get_my_id()
	var players: Dictionary = _main.entity_mgr.spawned_players
	assert_bool(players.has(my_id)).is_true()
	var player: CharacterBody3D = players[my_id]
	assert_bool(is_instance_valid(player)).is_true()
	# Player must be in the scene tree
	assert_bool(player.is_inside_tree()).is_true()
	# Camera must exist and be current
	var camera: Camera3D = player.get_node_or_null("Head/Camera3D")
	if camera:
		assert_bool(camera.current).is_true()


## Assert the environment is loaded and valid.
func _assert_env_valid(_label: String) -> void:
	var env: Node3D = _main.env_builder.current_env
	assert_bool(env != null).is_true()
	assert_bool(is_instance_valid(env)).is_true()
	assert_bool(env.is_inside_tree()).is_true()


## Assert shared_hud._local_player is not a freed reference.
func _assert_hud_player_valid(_label: String) -> void:
	var hud = _main._shared_hud
	if hud == null:
		return
	# Access the internal _local_player field
	var lp = hud.get("_local_player")
	if lp != null:
		assert_bool(is_instance_valid(lp)).is_true()


# =========================================================================
# Tests
# =========================================================================


func test_hub_arena_hub_single_cycle() -> void:
	await _enter_open_world()
	_assert_env_valid("hub initial")
	_assert_player_alive("hub initial")
	_assert_hud_player_valid("hub initial")

	await _transfer_to_arena()
	_assert_env_valid("arena")
	_assert_player_alive("arena")
	_assert_hud_player_valid("arena")

	await _transfer_to_hub()
	_assert_env_valid("hub return")
	_assert_player_alive("hub return")
	_assert_hud_player_valid("hub return")


func test_hub_arena_hub_three_cycles() -> void:
	await _enter_open_world()

	for i in 3:
		await _transfer_to_arena()
		_assert_env_valid("arena cycle %d" % i)
		_assert_player_alive("arena cycle %d" % i)
		_assert_hud_player_valid("arena cycle %d" % i)

		await _transfer_to_hub()
		_assert_env_valid("hub cycle %d" % i)
		_assert_player_alive("hub cycle %d" % i)
		_assert_hud_player_valid("hub cycle %d" % i)


## Specifically targets the freed-reference bug at shared_hud.gd:120.
## After each transfer, forces a redraw and checks that _local_player
## is never a freed reference. Also checks between the queue_free and
## the actual free (one frame gap) by not awaiting before the check.
func test_local_player_never_freed_reference() -> void:
	await _enter_open_world()

	for i in 5:
		# Transfer to arena
		_main.game_flow.on_zone_transfer(1, 1)
		# Immediately force shared_hud to redraw (simulates what the engine does)
		_force_hud_draw("arena sync %d" % i)
		await get_tree().process_frame
		_force_hud_draw("arena frame %d" % i)
		await get_tree().process_frame

		# Transfer to hub
		_main.game_flow.on_zone_transfer(0, 1)
		_force_hud_draw("hub sync %d" % i)
		await get_tree().process_frame
		_force_hud_draw("hub frame %d" % i)
		await get_tree().process_frame


## Reproduce exactly what shared_hud._draw line 120 does and assert no crash.
func _force_hud_draw(_label: String) -> void:
	var hud = _main._shared_hud
	if hud == null:
		return
	var lp = hud.get("_local_player")
	# This is the exact check the bug requires: if lp is a freed ref,
	# is_instance_valid returns false but accessing properties crashes.
	if lp != null:
		assert_bool(is_instance_valid(lp)).is_true()
	# Also force the actual _draw path
	hud.queue_redraw()


## After a zone transfer, there must be exactly one WorldEnvironment in the
## tree. queue_free defers deletion, so the old env's WorldEnvironment can
## coexist with the new one for a frame. Godot only supports one active
## WorldEnvironment; the duplicate breaks the viewport.
func test_single_world_environment_after_transfer() -> void:
	await _enter_open_world()

	for i in 3:
		# Transfer to arena, then check IMMEDIATELY (no await) for duplicates.
		_main.game_flow.on_zone_transfer(1, 1)
		var we_count := _count_world_environments(_main)
		assert_int(we_count).is_less_equal(1)
		# Now let the frame settle for next iteration.
		await get_tree().process_frame
		await get_tree().process_frame

		# Transfer to hub.
		_main.game_flow.on_zone_transfer(0, 1)
		we_count = _count_world_environments(_main)
		assert_int(we_count).is_less_equal(1)
		await get_tree().process_frame
		await get_tree().process_frame


## Count all WorldEnvironment nodes in the full scene subtree.
func _count_world_environments(root: Node) -> int:
	var count := 0
	if root is WorldEnvironment:
		count += 1
	for child in root.get_children():
		count += _count_world_environments(child)
	return count
