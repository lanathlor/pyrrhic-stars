class_name TestLocalPlayerStaleDespawn
extends GdUnitTestSuite

## Regression: after hub->arena zone transfer, a stale world state from the
## previous zone can still arrive (UDP in flight / queued ticks). Its player
## list does not contain the new local peer_id, so _despawn_stale_players
## freed the local player - taking the only active Camera3D with it and
## leaving a permanent grey screen. _sync_players never respawns self
## (the local player is owned by the game flow manager), so the game never
## recovered.
##
## The fix: world-state reconciliation only manages remote puppets and must
## never despawn the local player.

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

	_entity_mgr = Node.new()
	_entity_mgr.name = "EntityManager"
	_entity_mgr.set_script(EntityMgrScript)
	_main.add_child(_entity_mgr)

	_world_sync = Node.new()
	_world_sync.name = "WorldStateSync"
	_world_sync.set_script(WorldSyncScript)
	_main.add_child(_world_sync)

	add_child(_main)
	await get_tree().process_frame


func _add_player(pid: int) -> CharacterBody3D:
	var player := CharacterBody3D.new()
	player.name = "Player_%d" % pid
	player.set("peer_id", pid)
	_players_node.add_child(player)
	_entity_mgr.spawned_players[pid] = player
	return player


## Stale hub world state arrives after the arena spawn: its player list has
## only old-zone peers (no local peer_id=1). The local player must survive.
func test_local_player_survives_stale_world_state() -> void:
	var local_player := _add_player(1)  # NetworkManager.get_my_id() == 1 when inactive

	# seen_peers from a stale hub state: old peer_id 64 only, no peer 1
	_world_sync._despawn_stale_players(_entity_mgr, {64: true})
	await get_tree().process_frame

	assert_bool(is_instance_valid(local_player)).is_true()
	assert_bool(1 in _entity_mgr.spawned_players).is_true()


## Remote puppets missing from the state must still be despawned.
func test_remote_player_still_despawned_when_stale() -> void:
	_add_player(1)
	var remote_player := _add_player(7)

	_world_sync._despawn_stale_players(_entity_mgr, {64: true})
	await get_tree().process_frame

	assert_bool(is_instance_valid(remote_player)).is_false()
	assert_bool(7 in _entity_mgr.spawned_players).is_false()
	assert_bool(1 in _entity_mgr.spawned_players).is_true()
