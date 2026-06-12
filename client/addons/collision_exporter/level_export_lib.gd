class_name LevelExportLib
## Shared level export logic used by both the editor plugin (on-save) and the
## headless CI script. Uses manual parent-chain transform walks so it works
## correctly without editor transform propagation.

const VERSION := 5


# --- Public API ---

## Extracts all level data from a scene root and returns the full dictionary.
static func extract_level(root: Node, scene_path: String) -> Dictionary:
	var obstacles: Array = []
	var elevators: Array = []
	var player_spawns: Array = []
	var enemy_spawns: Array = []
	var npc_spawns: Array = []
	var portals: Array = []
	var zone_triggers: Array = []
	var gates: Array = []
	var bounds_override: Dictionary = {}
	var zone_config: Dictionary = {}

	_walk_tree(root, obstacles, elevators, player_spawns, enemy_spawns,
		npc_spawns, portals, zone_triggers, gates, bounds_override, zone_config)

	var zone_name: String = ""
	if zone_config.has("zone_name"):
		zone_name = zone_config["zone_name"]
	if zone_name == "":
		zone_name = scene_path.get_file().get_basename()

	var bounds: Dictionary
	if bounds_override.size() > 0:
		bounds = bounds_override
	else:
		bounds = _compute_bounds(obstacles)

	var navmesh_data: Dictionary = _bake_navmesh(root)

	var data: Dictionary = {
		"version": VERSION,
		"zone": zone_name,
		"zone_type": zone_config.get("zone_type", "open_world"),
		"enemy_radius": zone_config.get("enemy_radius", 0.0),
		"source_scene": scene_path,
		"bounds": bounds,
		"obstacles": obstacles,
		"elevators": elevators,
		"player_spawns": player_spawns,
		"enemy_spawns": enemy_spawns,
		"npc_spawns": npc_spawns,
		"portals": portals,
		"zone_triggers": zone_triggers,
		"gates": gates,
	}
	var spawn_yaw: float = float(zone_config.get("spawn_yaw", 0.0))
	if spawn_yaw != 0.0:
		data["spawn_yaw"] = spawn_yaw
	if navmesh_data.size() > 0:
		data["navmesh"] = navmesh_data

	return data


## Serializes level data to a JSON string, preserving int types for Go compat.
static func serialize_level(data: Dictionary) -> String:
	# JSON.stringify handles int/float correctly when values are properly typed
	# in GDScript (int stays as 1, float stays as 1.0). Our extractors type
	# everything explicitly, so this is safe.
	return JSON.stringify(data, "\t") + "\n"


## Writes level data to the shared/levels/ directory.
static func write_level(data: Dictionary, output_dir: String) -> String:
	var zone_name: String = data.get("zone", "unknown")
	DirAccess.make_dir_recursive_absolute(output_dir)
	var output_path := output_dir + zone_name + ".json"

	var json_str := serialize_level(data)
	var f := FileAccess.open(output_path, FileAccess.WRITE)
	if f == null:
		printerr("LevelExportLib: cannot open %s for writing" % output_path)
		return ""
	f.store_string(json_str)
	f.close()
	return output_path


# --- Transform helpers ---

static func _global_pos(node: Node3D) -> Vector3:
	return _global_xform(node).origin


static func _global_xform(node: Node3D) -> Transform3D:
	var xform := node.transform
	var parent := node.get_parent()
	while parent is Node3D:
		xform = parent.transform * xform
		parent = parent.get_parent()
	return xform


# --- Tree walker ---

static func _walk_tree(
	node: Node,
	obstacles: Array,
	elevators: Array,
	player_spawns: Array,
	enemy_spawns: Array,
	npc_spawns: Array,
	portals: Array,
	zone_triggers: Array,
	gates: Array,
	bounds_override: Dictionary,
	zone_config: Dictionary,
) -> void:
	if node.is_in_group("server_ignore"):
		return

	# Gates are dynamic server-side obstacles; they must not leak into the
	# static obstacle list or they block movement even while open.
	if node.is_in_group("server_gate"):
		_extract_gate(node, gates)
	elif node is CSGBox3D:
		_extract_collision(node, obstacles)
	elif node.is_in_group("server_collision"):
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
	if node.is_in_group("server_zone_config"):
		_extract_zone_config(node, zone_config)

	for child in node.get_children():
		_walk_tree(child, obstacles, elevators, player_spawns, enemy_spawns,
			npc_spawns, portals, zone_triggers, gates, bounds_override, zone_config)


# --- Extractors ---

static func _extract_collision(node: Node, obstacles: Array) -> void:
	if node is CSGBox3D:
		var box := node as CSGBox3D
		var pos := _global_pos(box)
		var half := box.size / 2.0
		obstacles.append({
			"name": box.name,
			"center": [snapped(pos.x, 0.01), snapped(pos.y, 0.01), snapped(pos.z, 0.01)],
			"half_extents": [snapped(half.x, 0.01), snapped(half.y, 0.01), snapped(half.z, 0.01)],
		})
	elif node is StaticBody3D:
		for child in node.get_children():
			if child is CollisionShape3D:
				var col := child as CollisionShape3D
				var shape := col.shape
				if shape is BoxShape3D:
					var box_shape := shape as BoxShape3D
					var pos := _global_pos(col)
					var half := box_shape.size / 2.0
					obstacles.append({
						"name": node.name + "/" + col.name,
						"center": [snapped(pos.x, 0.01), snapped(pos.y, 0.01), snapped(pos.z, 0.01)],
						"half_extents": [snapped(half.x, 0.01), snapped(half.y, 0.01), snapped(half.z, 0.01)],
					})


static func _extract_elevator(node: Node, elevators: Array) -> void:
	var n := node as Node3D
	var pos := _global_pos(n)
	elevators.append({
		"name": str(n.name),
		"center_x": snapped(pos.x + float(n.get_meta("offset_x", 0.0)), 0.01),
		"center_z": snapped(pos.z + float(n.get_meta("offset_z", 0.0)), 0.01),
		"half_x": float(n.get_meta("half_x", 4.0)),
		"half_z": float(n.get_meta("half_z", 4.0)),
		"bottom_y": float(n.get_meta("bottom_y", -200.0)),
		"top_y": float(n.get_meta("top_y", 0.0)),
		"speed": float(n.get_meta("speed", 10.0)),
	})


static func _extract_player_spawn(node: Node, player_spawns: Array) -> void:
	var n := node as Node3D
	var pos := _global_pos(n)
	var spawn: Dictionary = {
		"x": snapped(pos.x, 0.01),
		"y": snapped(pos.y, 0.01),
		"z": snapped(pos.z, 0.01),
	}
	var cond: String = str(n.get_meta("condition", ""))
	if cond != "":
		spawn["condition"] = cond
	player_spawns.append(spawn)


static func _extract_enemy_spawn(node: Node, enemy_spawns: Array) -> void:
	var n := node as Node3D
	var pos := _global_pos(n)
	var patrol_a: Vector3 = n.get_meta("patrol_a", pos)
	var patrol_b: Vector3 = n.get_meta("patrol_b", pos)
	var spawn: Dictionary = {
		"x": snapped(pos.x, 0.01),
		"y": snapped(pos.y, 0.01),
		"z": snapped(pos.z, 0.01),
		"def_name": str(n.get_meta("def_name", "")),
		"patrol_a": {"x": snapped(patrol_a.x, 0.01), "y": snapped(patrol_a.y, 0.01), "z": snapped(patrol_a.z, 0.01)},
		"patrol_b": {"x": snapped(patrol_b.x, 0.01), "y": snapped(patrol_b.y, 0.01), "z": snapped(patrol_b.z, 0.01)},
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
	var path_child: Path3D = null
	for child in n.get_children():
		if child is Path3D:
			path_child = child
			break
	if path_child:
		var waypoints: Array = []
		var curve: Curve3D = path_child.curve
		var path_xform := _global_xform(path_child)
		for i in range(curve.point_count):
			var p: Vector3 = path_xform * curve.get_point_position(i)
			waypoints.append({"x": snapped(p.x, 0.01), "y": snapped(p.y, 0.01), "z": snapped(p.z, 0.01)})
		if waypoints.size() >= 2:
			spawn["patrol_waypoints"] = waypoints
	enemy_spawns.append(spawn)


static func _extract_npc_spawn(node: Node, npc_spawns: Array) -> void:
	var n := node as Node3D
	var pos := _global_pos(n)
	var spawn: Dictionary = {
		"def_name": str(n.get_meta("def_name", "")),
		"speed": float(n.get_meta("speed", 1.5)),
		"idle_duration": float(n.get_meta("idle_duration", 4.0)),
	}
	var path_child: Path3D = null
	for child in n.get_children():
		if child is Path3D:
			path_child = child
			break
	var waypoints: Array = []
	if path_child:
		var curve: Curve3D = path_child.curve
		var path_xform := _global_xform(path_child)
		for i in range(curve.point_count):
			var p: Vector3 = path_xform * curve.get_point_position(i)
			waypoints.append({"x": snapped(p.x, 0.01), "y": snapped(p.y, 0.01), "z": snapped(p.z, 0.01)})
	if waypoints.is_empty():
		waypoints.append({"x": snapped(pos.x, 0.01), "y": snapped(pos.y, 0.01), "z": snapped(pos.z, 0.01)})
	spawn["waypoints"] = waypoints
	npc_spawns.append(spawn)


static func _extract_portal(node: Node, portals: Array) -> void:
	var n := node as Node3D
	var pos := _global_pos(n)
	var portal: Dictionary = {
		"name": str(n.name),
		"x": snapped(pos.x, 0.01),
		"y": snapped(pos.y, 0.01),
		"z": snapped(pos.z, 0.01),
		"target_zone": str(n.get_meta("target_zone", "")),
		"interaction_radius": float(n.get_meta("interaction_radius", 4.0)),
	}
	var cond: String = str(n.get_meta("condition", ""))
	if cond != "":
		portal["condition"] = cond
	portals.append(portal)


static func _extract_zone_trigger(node: Node, zone_triggers: Array) -> void:
	var n := node as Node3D
	var pos := _global_pos(n)
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
		"threshold": snapped(threshold, 0.01),
	})


static func _extract_gate(node: Node, gates: Array) -> void:
	if not node is CSGBox3D:
		printerr("LevelExportLib: server_gate node '%s' is not CSGBox3D, skipping" % node.name)
		return
	var box := node as CSGBox3D
	var pos := _global_pos(box)
	var half := box.size / 2.0
	var gate_id: String = str(box.get_meta("gate_id", ""))
	if gate_id == "":
		printerr("LevelExportLib: server_gate node '%s' has no gate_id, skipping" % node.name)
		return
	var close_on := _split_meta_list(box, "close_on")
	var open_on := _split_meta_list(box, "open_on")
	var gate: Dictionary = {
		"name": str(box.name),
		"gate_id": gate_id,
		"center": [snapped(pos.x, 0.01), snapped(pos.y, 0.01), snapped(pos.z, 0.01)],
		"half_extents": [snapped(half.x, 0.01), snapped(half.y, 0.01), snapped(half.z, 0.01)],
		"close_on": close_on,
		"open_on": open_on,
	}
	if box.get_meta("default_closed", false):
		gate["default_closed"] = true
	var push_axis: String = str(box.get_meta("push_axis", ""))
	if push_axis != "":
		gate["push_axis"] = push_axis
		gate["push_offset"] = float(box.get_meta("push_offset", 0.0))
	gates.append(gate)


static func _split_meta_list(node: Node, meta_name: String) -> Array:
	var out: Array = []
	for s in str(node.get_meta(meta_name, "")).split(","):
		var trimmed: String = s.strip_edges()
		if trimmed != "":
			out.append(trimmed)
	return out


static func _extract_bounds(node: Node, bounds: Dictionary) -> void:
	var n := node as Node3D
	bounds["min_x"] = float(n.get_meta("min_x", -20.0))
	bounds["max_x"] = float(n.get_meta("max_x", 20.0))
	bounds["min_y"] = float(n.get_meta("min_y", -1.0))
	bounds["max_y"] = float(n.get_meta("max_y", 6.0))
	bounds["min_z"] = float(n.get_meta("min_z", -15.0))
	bounds["max_z"] = float(n.get_meta("max_z", 52.0))


static func _extract_zone_config(node: Node, config: Dictionary) -> void:
	var n := node as Node3D
	var zn: String = str(n.get_meta("zone_name", ""))
	if zn != "":
		config["zone_name"] = zn
	config["zone_type"] = str(n.get_meta("zone_type", "open_world"))
	config["enemy_radius"] = float(n.get_meta("enemy_radius", 0.0))
	config["spawn_yaw"] = float(n.get_meta("spawn_yaw", 0.0))


# --- Navmesh baking ---

static func _bake_navmesh(root: Node) -> Dictionary:
	var boxes: Array[Dictionary] = []
	_collect_all_csg_boxes(root, boxes)
	if boxes.is_empty():
		return {}

	var nav_mesh := NavigationMesh.new()
	nav_mesh.agent_radius = 0.5
	nav_mesh.agent_height = 2.0
	nav_mesh.agent_max_climb = 0.5
	nav_mesh.agent_max_slope = 45.0
	nav_mesh.cell_size = 0.25
	nav_mesh.cell_height = 0.25

	var source_geo := NavigationMeshSourceGeometryData3D.new()
	for box in boxes:
		_add_box_faces(source_geo, box["center"], box["size"], nav_mesh.agent_max_climb)

	NavigationServer3D.bake_from_source_geometry_data(nav_mesh, source_geo)

	var verts: PackedVector3Array = nav_mesh.get_vertices()
	if verts.is_empty():
		return {}

	var verts_out: Array = []
	for v in verts:
		verts_out.append([snapped(v.x, 0.001), snapped(v.y, 0.001), snapped(v.z, 0.001)])

	var polys_out: Array = []
	for i in range(nav_mesh.get_polygon_count()):
		var indices: PackedInt32Array = nav_mesh.get_polygon(i)
		var idx_arr: Array[int] = []
		for idx in indices:
			idx_arr.append(idx)
		polys_out.append(idx_arr)

	return {"vertices": verts_out, "polygons": polys_out}


static func _collect_all_csg_boxes(node: Node, boxes: Array[Dictionary]) -> void:
	## Collects ALL CSGBox3D nodes including server_ignore (floors are geometry).
	if node is CSGBox3D:
		var box := node as CSGBox3D
		boxes.append({"center": _global_pos(box), "size": box.size})
	for child in node.get_children():
		_collect_all_csg_boxes(child, boxes)


static func _add_box_faces(sg: NavigationMeshSourceGeometryData3D, center: Vector3, size: Vector3, max_walkable_top_y: float = 0.5) -> void:
	var h := size / 2.0
	# Only add the top face for floor-level geometry. Walls and obstacles have
	# their top surface above max_walkable_top_y — skipping it prevents the
	# navmesh baker from creating walkable polygons on wall/pillar tops.
	if center.y + h.y <= max_walkable_top_y:
		_add_quad(sg, center, h, Vector3.UP, true)
	_add_quad(sg, center, h, Vector3.UP, false)
	_add_quad(sg, center, h, Vector3.BACK, true)
	_add_quad(sg, center, h, Vector3.BACK, false)
	_add_quad(sg, center, h, Vector3.LEFT, true)
	_add_quad(sg, center, h, Vector3.LEFT, false)


static func _add_quad(
	sg: NavigationMeshSourceGeometryData3D,
	c: Vector3, h: Vector3, axis: Vector3, positive: bool,
) -> void:
	var verts: PackedVector3Array
	if axis == Vector3.UP:
		var y := c.y + h.y if positive else c.y - h.y
		if positive:
			verts = PackedVector3Array([
				Vector3(c.x - h.x, y, c.z - h.z), Vector3(c.x + h.x, y, c.z + h.z), Vector3(c.x + h.x, y, c.z - h.z),
				Vector3(c.x - h.x, y, c.z - h.z), Vector3(c.x - h.x, y, c.z + h.z), Vector3(c.x + h.x, y, c.z + h.z),
			])
		else:
			verts = PackedVector3Array([
				Vector3(c.x - h.x, y, c.z - h.z), Vector3(c.x + h.x, y, c.z - h.z), Vector3(c.x + h.x, y, c.z + h.z),
				Vector3(c.x - h.x, y, c.z - h.z), Vector3(c.x + h.x, y, c.z + h.z), Vector3(c.x - h.x, y, c.z + h.z),
			])
	elif axis == Vector3.BACK:
		var z := c.z + h.z if positive else c.z - h.z
		if positive:
			verts = PackedVector3Array([
				Vector3(c.x - h.x, c.y - h.y, z), Vector3(c.x + h.x, c.y + h.y, z), Vector3(c.x + h.x, c.y - h.y, z),
				Vector3(c.x - h.x, c.y - h.y, z), Vector3(c.x - h.x, c.y + h.y, z), Vector3(c.x + h.x, c.y + h.y, z),
			])
		else:
			verts = PackedVector3Array([
				Vector3(c.x - h.x, c.y - h.y, z), Vector3(c.x + h.x, c.y - h.y, z), Vector3(c.x + h.x, c.y + h.y, z),
				Vector3(c.x - h.x, c.y - h.y, z), Vector3(c.x + h.x, c.y + h.y, z), Vector3(c.x - h.x, c.y + h.y, z),
			])
	else:
		var x := c.x + h.x if positive else c.x - h.x
		if positive:
			verts = PackedVector3Array([
				Vector3(x, c.y - h.y, c.z - h.z), Vector3(x, c.y + h.y, c.z + h.z), Vector3(x, c.y + h.y, c.z - h.z),
				Vector3(x, c.y - h.y, c.z - h.z), Vector3(x, c.y - h.y, c.z + h.z), Vector3(x, c.y + h.y, c.z + h.z),
			])
		else:
			verts = PackedVector3Array([
				Vector3(x, c.y - h.y, c.z - h.z), Vector3(x, c.y + h.y, c.z - h.z), Vector3(x, c.y + h.y, c.z + h.z),
				Vector3(x, c.y - h.y, c.z - h.z), Vector3(x, c.y - h.y, c.z + h.z), Vector3(x, c.y + h.y, c.z + h.z),
			])
	sg.add_faces(verts, Transform3D.IDENTITY)


static func _compute_bounds(obstacles: Array) -> Dictionary:
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
		"min_x": snapped(min_x - 1.0, 0.01),
		"max_x": snapped(max_x + 1.0, 0.01),
		"min_y": snapped(min_y - 1.0, 0.01),
		"max_y": snapped(max_y + 1.0, 0.01),
		"min_z": snapped(min_z - 1.0, 0.01),
		"max_z": snapped(max_z + 1.0, 0.01),
	}
