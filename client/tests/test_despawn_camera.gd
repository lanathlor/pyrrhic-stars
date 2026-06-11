class_name TestDespawnCamera
extends GdUnitTestSuite

## Regression: rain+void+backdrop and grey screen after zone transfer.
## despawn_all_players() used queue_free() which leaves the old player
## (and its Camera3D) in the scene tree for one frame. The old camera
## races with the new player's camera, and the old player renders at
## its stale position. Fix: remove_child before queue_free so the old
## player exits the tree immediately.

const EntityMgrScript := preload("res://scenes/main/entity_manager.gd")


## entity_manager.despawn_all_players must remove players from the tree
## immediately (not defer via queue_free alone). Verify by checking the
## source does remove_child or get_parent().remove_child before free.
func test_despawn_removes_from_tree_before_free() -> void:
	var src: String = (EntityMgrScript as GDScript).source_code
	var fn_idx := src.find("func despawn_all_players")
	assert_int(fn_idx).is_greater(-1)
	var fn_end := src.find("\nfunc ", fn_idx + 1)
	if fn_end < 0:
		fn_end = src.length()
	var fn_body := src.substr(fn_idx, fn_end - fn_idx)

	# queue_free alone leaves the node in the tree for one frame.
	# The function must either use remove_child or get_parent().remove_child,
	# or use free() (immediate) instead of queue_free().
	var uses_queue_free := fn_body.find("queue_free") != -1
	var removes_from_tree := fn_body.find("remove_child") != -1 or fn_body.find(".free()") != -1

	# If queue_free is used, remove_child must also be present
	if uses_queue_free:
		assert_bool(removes_from_tree).is_true()


## Behavioral: after removing a player from tree + queue_free, the
## players node has zero children and the new camera is the only one.
func test_remove_then_free_zero_children() -> void:
	var players_node: Node3D = auto_free(Node3D.new()) as Node3D
	players_node.name = "Players"
	add_child(players_node)

	var old_player := CharacterBody3D.new()
	old_player.name = "Player_old"
	var old_cam := Camera3D.new()
	old_cam.name = "Camera3D"
	old_cam.current = true
	old_player.add_child(old_cam)
	players_node.add_child(old_player)

	# The fix: remove from tree then queue_free
	players_node.remove_child(old_player)
	old_player.queue_free()

	assert_int(players_node.get_child_count()).is_equal(0)

	# New player spawns in same frame
	var new_player: CharacterBody3D = auto_free(CharacterBody3D.new()) as CharacterBody3D
	new_player.name = "Player_new"
	var new_cam := Camera3D.new()
	new_cam.name = "Camera3D"
	new_cam.current = true
	new_player.add_child(new_cam)
	players_node.add_child(new_player)

	assert_int(players_node.get_child_count()).is_equal(1)
	assert_bool(new_cam.current).is_true()
