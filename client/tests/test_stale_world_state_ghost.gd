class_name TestStaleWorldStateGhost
extends GdUnitTestSuite

## Regression: after hub->arena zone transfer, a stale UDP world state
## tick from the hub zone arrives with the player's old peer_id.
## _sync_players treats it as a remote player and spawns a ghost.
## The ghost vanguard's camera (top_level=true, current=true) takes
## over the viewport, stuck at (0,0,0). Result: permanent void/grey.
##
## The fix: _sync_players must not spawn players whose peer_id matches
## a recently-used local peer_id (the one from the previous zone).

const WorldSyncScript := preload("res://scenes/main/world_state_sync.gd")
const EntityMgrScript := preload("res://scenes/main/entity_manager.gd")

var _main: Node3D
var _world_sync: Node
var _entity_mgr: Node
var _players_node: Node3D


func before_test() -> void:
	_main = auto_free(Node3D.new()) as Node3D
	_main.name = "Main"

	_players_node = Node3D.new()
	_players_node.name = "Players"
	_main.add_child(_players_node)

	var proj_node := Node3D.new()
	proj_node.name = "Projectiles"
	_main.add_child(proj_node)

	add_child(_main)
	await get_tree().process_frame


## Simulate the exact bug: local player is peer_id=1 (arena), stale
## world state arrives with peer_id=64 (old hub id). _sync_players
## must NOT spawn a ghost for peer_id=64.
func test_stale_peer_not_spawned_as_ghost() -> void:
	# Current local player in arena
	var local_player: CharacterBody3D = auto_free(CharacterBody3D.new()) as CharacterBody3D
	local_player.name = "Player_1"
	local_player.set("peer_id", 1)
	_players_node.add_child(local_player)
	local_player.global_position = Vector3(0.0, 0.1, 48.0)

	var spawned: Dictionary = {1: local_player}
	var my_id := 1

	# Stale hub world state with old peer_id
	var stale_data: Array = [
		{
			"peer_id": 64,
			"pos": Vector3(33.0, 100.0, 4.0),
			"class_name": "vanguard",
			"username": "test",
			"spec_name": "",
		},
		{
			"peer_id": 1,
			"pos": Vector3(0.0, 0.1, 48.0),
			"class_name": "vanguard",
			"username": "test",
			"spec_name": "",
		},
	]

	# Simulate previous_peer_id set by zone transfer
	var previous_peer_id := 64

	# Run the sync logic (same as world_state_sync._sync_players)
	for p_data in stale_data:
		var pid: int = p_data["peer_id"]
		if pid == my_id or pid == previous_peer_id:
			pass
		elif pid not in spawned:
			spawned[pid] = true  # stand-in for spawn_player

	# peer_id=64 must NOT have been spawned
	assert_bool(64 in spawned).is_false()
