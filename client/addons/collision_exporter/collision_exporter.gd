@tool
extends EditorScript
## Exports level collision data to JSON for the Go server.
##
## Usage:
##   1. Open the scene you want to export in the editor.
##   2. File > Run Script > select this script.
##
## Tag scene nodes with groups to control what gets exported:
##   - server_collision    — CSGBox3D or StaticBody3D with BoxShape3D children → obstacles
##   - server_elevator     — Node3D with metadata: bottom_y, top_y, half_x, half_z, speed
##   - server_spawn_player — Node3D → player spawn position (optional meta: condition)
##   - server_spawn_enemy  — Node3D with metadata: def_name, is_boss, patrol_a, patrol_b,
##                           aggro_radius, leash_radius, group_id, condition
##                           Optional Path3D child for patrol_waypoints
##   - server_spawn_npc    — Node3D with metadata: def_name, speed, idle_duration
##                           Optional Path3D child for waypoints (falls back to node position)
##   - server_portal       — Node3D with metadata: target_zone, interaction_radius, condition
##   - server_zone_trigger — Node3D with metadata: trigger_id, axis
##   - server_bounds       — Node3D with metadata: min_x, max_x, min_y, max_y, min_z, max_z
##   - server_navmesh      — Reserved for Option C (navmesh export). Skipped for now.

const VERSION := 3


func _run() -> void:
	var root := get_editor_interface().get_edited_scene_root()
	if root == null:
		printerr("collision_exporter: no scene open in editor")
		return

	var scene_path: String = root.scene_file_path
	var zone_name: String = scene_path.get_file().get_basename()
	# For scenes like military_building.tscn, map to "hub"
	if zone_name != "arena" and zone_name != "hub":
		zone_name = _infer_zone_name(scene_path)

	print("collision_exporter: exporting zone '%s' from %s" % [zone_name, scene_path])

	var obstacles: Array = []
	var elevators: Array = []
	var player_spawns: Array = []
	var enemy_spawns: Array = []
	var npc_spawns: Array = []
	var portals: Array = []
	var zone_triggers: Array = []
	var bounds_override: Dictionary = {}

	_walk_tree(root, obstacles, elevators, player_spawns, enemy_spawns, npc_spawns, portals, zone_triggers, bounds_override)

	# Compute bounds from obstacles if no override
	var bounds: Dictionary
	if bounds_override.size() > 0:
		bounds = bounds_override
	else:
		bounds = _compute_bounds(obstacles)

	var data: Dictionary = {
		"version": VERSION,
		"zone": zone_name,
		"source_scene": scene_path,
		"bounds": bounds,
		"obstacles": obstacles,
		"elevators": elevators,
		"player_spawns": player_spawns,
		"enemy_spawns": enemy_spawns,
		"npc_spawns": npc_spawns,
		"portals": portals,
		"zone_triggers": zone_triggers,
	}

	var output_dir := ProjectSettings.globalize_path("res://") + "../../shared/levels/"
	DirAccess.make_dir_recursive_absolute(output_dir)
	var output_path := output_dir + zone_name + ".json"

	var json_str := JSON.stringify(data, "\t")
	var f := FileAccess.open(output_path, FileAccess.WRITE)
	if f == null:
		printerr("collision_exporter: cannot open %s for writing" % output_path)
		return
	f.store_string(json_str)
	f.store_string("\n")
	f.close()

	print("collision_exporter: wrote %s (%d obstacles, %d elevators, %d player spawns, %d enemy spawns, %d npc spawns, %d portals, %d zone triggers)" % [
		output_path, obstacles.size(), elevators.size(), player_spawns.size(), enemy_spawns.size(),
		npc_spawns.size(), portals.size(), zone_triggers.size(),
	])


func _walk_tree(
	node: Node,
	obstacles: Array,
	elevators: Array,
	player_spawns: Array,
	enemy_spawns: Array,
	npc_spawns: Array,
	portals: Array,
	zone_triggers: Array,
	bounds_override: Dictionary,
) -> void:
	if node.is_in_group("server_collision"):
		_extract_collision(node, obstacles)
	if node.is_in_group("server_elevator"):
		_extract_elevator(node, elevators)
	if node.is_in_group("server_spawn_player"):
		_extract_player_spawn(node, player_spawns)
	if node.is_in_group("server_spawn_enemy"):
		_extract_enemy_spawn(node, enemy_spawns)
	if node.is_in_group("server_spawn_npc"):
		_extract_npc_spawn(node, npc_spawns)
	if node.is_in_group("server_portal"):
		_extract_portal(node, portals)
	if node.is_in_group("server_zone_trigger"):
		_extract_zone_trigger(node, zone_triggers)
	if node.is_in_group("server_bounds"):
		_extract_bounds(node, bounds_override)
	if node.is_in_group("server_navmesh"):
		print("collision_exporter: skipping server_navmesh node '%s' (reserved for Option C)" % node.name)

	for child in node.get_children():
		_walk_tree(child, obstacles, elevators, player_spawns, enemy_spawns, npc_spawns, portals, zone_triggers, bounds_override)


func _extract_collision(node: Node, obstacles: Array) -> void:
	if node is CSGBox3D:
		var box := node as CSGBox3D
		var pos := box.global_position
		var half := box.size / 2.0
		obstacles.append({
			"name": box.name,
			"center": [snapf(pos.x, 0.01), snapf(pos.y, 0.01), snapf(pos.z, 0.01)],
			"half_extents": [snapf(half.x, 0.01), snapf(half.y, 0.01), snapf(half.z, 0.01)],
		})
	elif node is StaticBody3D:
		for child in node.get_children():
			if child is CollisionShape3D:
				var col := child as CollisionShape3D
				var shape := col.shape
				if shape is BoxShape3D:
					var box_shape := shape as BoxShape3D
					var pos := col.global_position
					var half := box_shape.size / 2.0
					obstacles.append({
						"name": node.name + "/" + col.name,
						"center": [snapf(pos.x, 0.01), snapf(pos.y, 0.01), snapf(pos.z, 0.01)],
						"half_extents": [snapf(half.x, 0.01), snapf(half.y, 0.01), snapf(half.z, 0.01)],
					})
				else:
					print("collision_exporter: skipping non-box shape in '%s/%s'" % [node.name, col.name])


func _extract_elevator(node: Node, elevators: Array) -> void:
	var n := node as Node3D
	var pos := n.global_position
	elevators.append({
		"name": str(n.name),
		"center_x": snapf(pos.x + float(n.get_meta("offset_x", 0.0)), 0.01),
		"center_z": snapf(pos.z + float(n.get_meta("offset_z", 0.0)), 0.01),
		"half_x": float(n.get_meta("half_x", 4.0)),
		"half_z": float(n.get_meta("half_z", 4.0)),
		"bottom_y": float(n.get_meta("bottom_y", -200.0)),
		"top_y": float(n.get_meta("top_y", 0.0)),
		"speed": float(n.get_meta("speed", 10.0)),
	})


func _extract_player_spawn(node: Node, player_spawns: Array) -> void:
	var n := node as Node3D
	var pos := n.global_position
	var spawn: Dictionary = {
		"x": snapf(pos.x, 0.01),
		"y": snapf(pos.y, 0.01),
		"z": snapf(pos.z, 0.01),
	}
	var cond: String = str(n.get_meta("condition", ""))
	if cond != "":
		spawn["condition"] = cond
	player_spawns.append(spawn)


func _extract_enemy_spawn(node: Node, enemy_spawns: Array) -> void:
	var n := node as Node3D
	var pos := n.global_position
	var patrol_a: Vector3 = n.get_meta("patrol_a", pos)
	var patrol_b: Vector3 = n.get_meta("patrol_b", pos)
	var spawn: Dictionary = {
		"x": snapf(pos.x, 0.01),
		"y": snapf(pos.y, 0.01),
		"z": snapf(pos.z, 0.01),
		"def_name": str(n.get_meta("def_name", "")),
		"patrol_a": {"x": snapf(patrol_a.x, 0.01), "y": snapf(patrol_a.y, 0.01), "z": snapf(patrol_a.z, 0.01)},
		"patrol_b": {"x": snapf(patrol_b.x, 0.01), "y": snapf(patrol_b.y, 0.01), "z": snapf(patrol_b.z, 0.01)},
		"aggro_radius": float(n.get_meta("aggro_radius", 10.0)),
		"leash_radius": float(n.get_meta("leash_radius", 30.0)),
	}
	if n.get_meta("is_boss", false):
		spawn["is_boss"] = true
	var gid: int = n.get_meta("group_id", 0)
	if gid > 0:
		spawn["group_id"] = gid
	var cond: String = str(n.get_meta("condition", ""))
	if cond != "":
		spawn["condition"] = cond
	# Check for Path3D child for waypoint-based patrol
	var path_child: Path3D = null
	for child in n.get_children():
		if child is Path3D:
			path_child = child
			break
	if path_child:
		var waypoints: Array = []
		var curve: Curve3D = path_child.curve
		for i in range(curve.point_count):
			var p: Vector3 = path_child.global_transform * curve.get_point_position(i)
			waypoints.append({"x": snapf(p.x, 0.01), "y": snapf(p.y, 0.01), "z": snapf(p.z, 0.01)})
		if waypoints.size() >= 2:
			spawn["patrol_waypoints"] = waypoints
	enemy_spawns.append(spawn)


func _extract_npc_spawn(node: Node, npc_spawns: Array) -> void:
	var n := node as Node3D
	var pos := n.global_position
	var spawn: Dictionary = {
		"def_name": str(n.get_meta("def_name", "")),
		"speed": float(n.get_meta("speed", 1.5)),
		"idle_duration": float(n.get_meta("idle_duration", 4.0)),
	}
	# Check for Path3D child for waypoints
	var path_child: Path3D = null
	for child in n.get_children():
		if child is Path3D:
			path_child = child
			break
	var waypoints: Array = []
	if path_child:
		var curve: Curve3D = path_child.curve
		for i in range(curve.point_count):
			var p: Vector3 = path_child.global_transform * curve.get_point_position(i)
			waypoints.append({"x": snapf(p.x, 0.01), "y": snapf(p.y, 0.01), "z": snapf(p.z, 0.01)})
	if waypoints.is_empty():
		# Fall back to node position as single waypoint
		waypoints.append({"x": snapf(pos.x, 0.01), "y": snapf(pos.y, 0.01), "z": snapf(pos.z, 0.01)})
	spawn["waypoints"] = waypoints
	npc_spawns.append(spawn)


func _extract_portal(node: Node, portals: Array) -> void:
	var n := node as Node3D
	var pos := n.global_position
	var portal: Dictionary = {
		"name": str(n.name),
		"x": snapf(pos.x, 0.01),
		"y": snapf(pos.y, 0.01),
		"z": snapf(pos.z, 0.01),
		"target_zone": str(n.get_meta("target_zone", "")),
		"interaction_radius": float(n.get_meta("interaction_radius", 4.0)),
	}
	var cond: String = str(n.get_meta("condition", ""))
	if cond != "":
		portal["condition"] = cond
	portals.append(portal)


func _extract_zone_trigger(node: Node, zone_triggers: Array) -> void:
	var n := node as Node3D
	var pos := n.global_position
	var axis: String = str(n.get_meta("axis", "z"))
	var threshold: float = 0.0
	match axis:
		"x":
			threshold = pos.x
		"y":
			threshold = pos.y
		_:
			threshold = pos.z
	zone_triggers.append({
		"name": str(n.name),
		"trigger_id": str(n.get_meta("trigger_id", "")),
		"axis": axis,
		"threshold": snapf(threshold, 0.01),
	})


func _extract_bounds(node: Node, bounds: Dictionary) -> void:
	var n := node as Node3D
	bounds["min_x"] = float(n.get_meta("min_x", -20.0))
	bounds["max_x"] = float(n.get_meta("max_x", 20.0))
	bounds["min_y"] = float(n.get_meta("min_y", -1.0))
	bounds["max_y"] = float(n.get_meta("max_y", 6.0))
	bounds["min_z"] = float(n.get_meta("min_z", -15.0))
	bounds["max_z"] = float(n.get_meta("max_z", 52.0))


func _compute_bounds(obstacles: Array) -> Dictionary:
	if obstacles.is_empty():
		return {"min_x": -10.0, "max_x": 10.0, "min_y": -1.0, "max_y": 5.0, "min_z": -10.0, "max_z": 10.0}
	var min_x := INF
	var max_x := -INF
	var min_y := INF
	var max_y := -INF
	var min_z := INF
	var max_z := -INF
	for obs in obstacles:
		var c: Array = obs["center"]
		var h: Array = obs["half_extents"]
		min_x = minf(min_x, c[0] - h[0])
		max_x = maxf(max_x, c[0] + h[0])
		min_y = minf(min_y, c[1] - h[1])
		max_y = maxf(max_y, c[1] + h[1])
		min_z = minf(min_z, c[2] - h[2])
		max_z = maxf(max_z, c[2] + h[2])
	return {
		"min_x": snapf(min_x - 1.0, 0.01),
		"max_x": snapf(max_x + 1.0, 0.01),
		"min_y": snapf(min_y - 1.0, 0.01),
		"max_y": snapf(max_y + 1.0, 0.01),
		"min_z": snapf(min_z - 1.0, 0.01),
		"max_z": snapf(max_z + 1.0, 0.01),
	}


func _infer_zone_name(scene_path: String) -> String:
	if "hub" in scene_path or "prime_hub" in scene_path:
		return "hub"
	if "arena" in scene_path:
		return "arena"
	# Fall back to directory name
	return scene_path.get_base_dir().get_file()
