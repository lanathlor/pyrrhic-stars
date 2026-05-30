@tool
extends EditorNode3DGizmoPlugin
## Draws editor gizmos for all server_* level marker nodes.
##
## Enemy/NPC spawns show the actual FBX model (from metadata/model or default).
## All markers are clickable in the viewport to select the spawn node.

const DEFAULT_ENEMY_MODEL = "res://assets/models/aceras.fbx"
const DEFAULT_NPC_MODEL = "res://assets/models/aceras.fbx"

# Mesh cache: model_path → Array[Mesh]
var _mesh_cache = {}

var _spawn_mesh: SphereMesh


func _init() -> void:
	# Enemy group colors
	create_material("enemy_boss", Color(0.85, 0.05, 0.1))
	for i in range(6):
		create_material("enemy_group_%d" % i, _group_color(i))

	create_material("aggro", Color(1.0, 0.85, 0.0, 0.5))
	create_material("leash", Color(1.0, 0.15, 0.15, 0.35))
	create_material("patrol", Color(0.0, 0.85, 0.85))

	create_material("spawn_default", Color(0.15, 0.85, 0.15))
	create_material("spawn_conditional", Color(0.2, 0.6, 1.0))

	create_material("portal", Color(0.3, 0.45, 1.0))
	create_material("portal_radius", Color(0.3, 0.45, 1.0, 0.3))

	create_material("npc", Color(1.0, 0.8, 0.0))
	create_material("npc_path", Color(1.0, 0.8, 0.0, 0.6))

	create_material("trigger", Color(1.0, 0.5, 0.0))
	create_material("bounds", Color(1.0, 1.0, 1.0, 0.25))
	create_material("zone_config", Color(0.0, 0.8, 0.8))

	_spawn_mesh = SphereMesh.new()
	_spawn_mesh.radius = 0.5
	_spawn_mesh.height = 1.0


static func _group_color(group_id: int) -> Color:
	match group_id:
		1: return Color(1.0, 0.55, 0.15)
		2: return Color(0.7, 0.25, 0.85)
		3: return Color(0.15, 0.75, 0.75)
		4: return Color(0.55, 0.8, 0.2)
		5: return Color(0.9, 0.2, 0.6)
		_: return Color(0.9, 0.25, 0.25)


func _has_gizmo(node: Node3D) -> bool:
	for g in node.get_groups():
		if str(g).begins_with("server_"):
			return true
	return false


func _get_gizmo_name() -> String:
	return "LevelMarker"


func _redraw(gizmo: EditorNode3DGizmo) -> void:
	gizmo.clear()
	var node = gizmo.get_node_3d()
	if node == null:
		return

	if node.is_in_group("server_spawn_enemy"):
		_draw_enemy(gizmo, node)
	if node.is_in_group("server_spawn_player"):
		_draw_player_spawn(gizmo, node)
	if node.is_in_group("server_portal"):
		_draw_portal(gizmo, node)
	if node.is_in_group("server_spawn_npc"):
		_draw_npc(gizmo, node)
	if node.is_in_group("server_zone_trigger"):
		_draw_trigger(gizmo, node)
	if node.is_in_group("server_bounds"):
		_draw_bounds(gizmo, node)
	if node.is_in_group("server_zone_config"):
		_draw_zone_config(gizmo, node)


# =============================================================================
# Model loading
# =============================================================================

func _load_meshes(model_path: String) -> Array:
	if _mesh_cache.has(model_path):
		return _mesh_cache[model_path]

	var meshes = []
	var scene = load(model_path) as PackedScene
	if scene == null:
		_mesh_cache[model_path] = meshes
		return meshes

	var instance = scene.instantiate()
	_collect_meshes(instance, meshes)
	instance.free()

	_mesh_cache[model_path] = meshes
	return meshes


func _collect_meshes(node: Node, meshes: Array) -> void:
	if node is MeshInstance3D:
		var mi = node as MeshInstance3D
		if mi.mesh != null:
			meshes.append(mi.mesh)
	for child in node.get_children():
		_collect_meshes(child, meshes)


func _add_model(gizmo: EditorNode3DGizmo, model_path: String, xform: Transform3D, tint_mat = null) -> void:
	var meshes = _load_meshes(model_path)
	if meshes.is_empty():
		return

	for m in meshes:
		if tint_mat != null:
			gizmo.add_mesh(m, tint_mat, xform)
		else:
			gizmo.add_mesh(m, null, xform)

	# Collision triangles for click-to-select (use first mesh)
	var tri = (meshes[0] as Mesh).generate_triangle_mesh()
	if tri != null:
		gizmo.add_collision_triangles(tri)


# =============================================================================
# Enemy spawn
# =============================================================================

func _draw_enemy(gizmo: EditorNode3DGizmo, node: Node3D) -> void:
	var is_boss = node.get_meta("is_boss", false)
	var group_id = int(node.get_meta("group_id", 0))
	var model_path = str(node.get_meta("model", DEFAULT_ENEMY_MODEL))

	# Tint material based on group
	var mat_name: String
	if is_boss:
		mat_name = "enemy_boss"
	else:
		mat_name = "enemy_group_%d" % clampi(group_id, 0, 5)
	var tint = get_material(mat_name, gizmo)

	# Scale boss up
	var scale_val = 1.3 if is_boss else 1.0
	var xform = Transform3D(Basis.IDENTITY.scaled(Vector3(scale_val, scale_val, scale_val)), Vector3.ZERO)

	_add_model(gizmo, model_path, xform, tint)

	# Aggro radius circle
	var aggro_r = float(node.get_meta("aggro_radius", 10.0))
	gizmo.add_lines(_make_circle(aggro_r), get_material("aggro", gizmo))

	# Leash radius circle
	var leash_r = float(node.get_meta("leash_radius", 30.0))
	gizmo.add_lines(_make_circle(leash_r, 48), get_material("leash", gizmo))

	# Patrol path
	var patrol_lines = _get_patrol_lines(node)
	if patrol_lines.size() >= 2:
		gizmo.add_lines(patrol_lines, get_material("patrol", gizmo))


func _get_patrol_lines(node: Node3D) -> PackedVector3Array:
	var lines = PackedVector3Array()

	for child in node.get_children():
		if child is Path3D and child.curve != null and child.curve.point_count >= 2:
			var curve = child.curve as Curve3D
			for i in range(curve.point_count - 1):
				var p0 = _to_local(node, child.to_global(curve.get_point_position(i)))
				var p1 = _to_local(node, child.to_global(curve.get_point_position(i + 1)))
				lines.append(p0)
				lines.append(p1)
			for i in range(curve.point_count):
				var p = _to_local(node, child.to_global(curve.get_point_position(i)))
				lines = _append_cross(lines, p, 0.3)
			return lines

	var pos = node.global_position
	var patrol_a = node.get_meta("patrol_a", pos) as Vector3
	var patrol_b = node.get_meta("patrol_b", pos) as Vector3
	if patrol_a != pos or patrol_b != pos:
		var a_local = _to_local(node, patrol_a)
		var b_local = _to_local(node, patrol_b)
		lines.append(a_local)
		lines.append(Vector3.ZERO)
		lines.append(Vector3.ZERO)
		lines.append(b_local)
		lines = _append_cross(lines, a_local, 0.4)
		lines = _append_cross(lines, b_local, 0.4)
	return lines


# =============================================================================
# Player spawn
# =============================================================================

func _draw_player_spawn(gizmo: EditorNode3DGizmo, node: Node3D) -> void:
	var cond = str(node.get_meta("condition", ""))
	var mat_name = "spawn_conditional" if cond != "" else "spawn_default"
	var mat = get_material(mat_name, gizmo)

	var offset = Transform3D(Basis.IDENTITY, Vector3(0, 0.3, 0))
	gizmo.add_mesh(_spawn_mesh, mat, offset)

	# Collision for click-to-select
	var tri = _spawn_mesh.generate_triangle_mesh()
	if tri != null:
		gizmo.add_collision_triangles(tri)

	var arrow = PackedVector3Array()
	arrow.append(Vector3(0, 1.0, 0))
	arrow.append(Vector3(0, 2.5, 0))
	arrow.append(Vector3(0, 2.5, 0))
	arrow.append(Vector3(-0.3, 2.1, 0))
	arrow.append(Vector3(0, 2.5, 0))
	arrow.append(Vector3(0.3, 2.1, 0))
	arrow.append(Vector3(0, 2.5, 0))
	arrow.append(Vector3(0, 2.1, -0.3))
	arrow.append(Vector3(0, 2.5, 0))
	arrow.append(Vector3(0, 2.1, 0.3))
	gizmo.add_lines(arrow, mat)


# =============================================================================
# Portal
# =============================================================================

func _draw_portal(gizmo: EditorNode3DGizmo, node: Node3D) -> void:
	var mat = get_material("portal", gizmo)

	var ring = PackedVector3Array()
	var seg = 32
	var r = 2.5
	for i in range(seg):
		var a1 = TAU * float(i) / seg
		var a2 = TAU * float(i + 1) / seg
		ring.append(Vector3(cos(a1) * r, sin(a1) * r + r, 0))
		ring.append(Vector3(cos(a2) * r, sin(a2) * r + r, 0))
	gizmo.add_lines(ring, mat)

	var interaction_r = float(node.get_meta("interaction_radius", 4.0))
	gizmo.add_lines(_make_circle(interaction_r), get_material("portal_radius", gizmo))


# =============================================================================
# NPC spawn
# =============================================================================

func _draw_npc(gizmo: EditorNode3DGizmo, node: Node3D) -> void:
	var model_path = str(node.get_meta("model", DEFAULT_NPC_MODEL))
	var mat = get_material("npc", gizmo)

	_add_model(gizmo, model_path, Transform3D.IDENTITY, mat)

	var path_mat = get_material("npc_path", gizmo)
	for child in node.get_children():
		if child is Path3D and child.curve != null and child.curve.point_count >= 2:
			var lines = PackedVector3Array()
			var curve = child.curve as Curve3D
			for i in range(curve.point_count - 1):
				var p0 = _to_local(node, child.to_global(curve.get_point_position(i)))
				var p1 = _to_local(node, child.to_global(curve.get_point_position(i + 1)))
				lines.append(p0)
				lines.append(p1)
			var first = _to_local(node, child.to_global(curve.get_point_position(0)))
			var last = _to_local(node, child.to_global(curve.get_point_position(curve.point_count - 1)))
			lines.append(last)
			lines.append(first)
			for i in range(curve.point_count):
				var p = _to_local(node, child.to_global(curve.get_point_position(i)))
				lines = _append_cross(lines, p, 0.25)
			gizmo.add_lines(lines, path_mat)
			break


# =============================================================================
# Zone trigger
# =============================================================================

func _draw_trigger(gizmo: EditorNode3DGizmo, node: Node3D) -> void:
	var mat = get_material("trigger", gizmo)
	var axis = str(node.get_meta("axis", "z"))
	var lines = PackedVector3Array()
	var ext = 30.0

	match axis:
		"x":
			lines.append(Vector3(0, 0, -ext))
			lines.append(Vector3(0, 0, ext))
			lines.append(Vector3(0, -1, 0))
			lines.append(Vector3(0, 3, 0))
		"y":
			lines.append(Vector3(-ext, 0, 0))
			lines.append(Vector3(ext, 0, 0))
			lines.append(Vector3(0, 0, -ext))
			lines.append(Vector3(0, 0, ext))
		_:
			lines.append(Vector3(-ext, 0, 0))
			lines.append(Vector3(ext, 0, 0))
			lines.append(Vector3(0, -1, 0))
			lines.append(Vector3(0, 3, 0))
	gizmo.add_lines(lines, mat)


# =============================================================================
# Bounds
# =============================================================================

func _draw_bounds(gizmo: EditorNode3DGizmo, node: Node3D) -> void:
	var mat = get_material("bounds", gizmo)
	var mn_x = float(node.get_meta("min_x", -20.0))
	var mx_x = float(node.get_meta("max_x", 20.0))
	var mn_y = float(node.get_meta("min_y", -1.0))
	var mx_y = float(node.get_meta("max_y", 6.0))
	var mn_z = float(node.get_meta("min_z", -15.0))
	var mx_z = float(node.get_meta("max_z", 52.0))

	var c0 = _to_local(node, Vector3(mn_x, mn_y, mn_z))
	var c1 = _to_local(node, Vector3(mx_x, mn_y, mn_z))
	var c2 = _to_local(node, Vector3(mx_x, mn_y, mx_z))
	var c3 = _to_local(node, Vector3(mn_x, mn_y, mx_z))
	var c4 = _to_local(node, Vector3(mn_x, mx_y, mn_z))
	var c5 = _to_local(node, Vector3(mx_x, mx_y, mn_z))
	var c6 = _to_local(node, Vector3(mx_x, mx_y, mx_z))
	var c7 = _to_local(node, Vector3(mn_x, mx_y, mx_z))

	var lines = PackedVector3Array()
	lines.append(c0); lines.append(c1)
	lines.append(c1); lines.append(c2)
	lines.append(c2); lines.append(c3)
	lines.append(c3); lines.append(c0)
	lines.append(c4); lines.append(c5)
	lines.append(c5); lines.append(c6)
	lines.append(c6); lines.append(c7)
	lines.append(c7); lines.append(c4)
	lines.append(c0); lines.append(c4)
	lines.append(c1); lines.append(c5)
	lines.append(c2); lines.append(c6)
	lines.append(c3); lines.append(c7)
	gizmo.add_lines(lines, mat)


# =============================================================================
# Zone config
# =============================================================================

func _draw_zone_config(gizmo: EditorNode3DGizmo, node: Node3D) -> void:
	var mat = get_material("zone_config", gizmo)

	# Draw a diamond shape to distinguish from other markers
	var s := 0.6
	var lines = PackedVector3Array()
	# Horizontal diamond
	lines.append(Vector3(s, 0, 0))
	lines.append(Vector3(0, 0, s))
	lines.append(Vector3(0, 0, s))
	lines.append(Vector3(-s, 0, 0))
	lines.append(Vector3(-s, 0, 0))
	lines.append(Vector3(0, 0, -s))
	lines.append(Vector3(0, 0, -s))
	lines.append(Vector3(s, 0, 0))
	# Vertical post
	lines.append(Vector3(0, -s, 0))
	lines.append(Vector3(0, s * 2.0, 0))
	# Top cross
	lines.append(Vector3(-s * 0.4, s * 2.0, 0))
	lines.append(Vector3(s * 0.4, s * 2.0, 0))
	lines.append(Vector3(0, s * 2.0, -s * 0.4))
	lines.append(Vector3(0, s * 2.0, s * 0.4))
	gizmo.add_lines(lines, mat)


# =============================================================================
# Helpers
# =============================================================================

func _make_circle(radius: float, segments: int = 32, y: float = 0.05) -> PackedVector3Array:
	var lines = PackedVector3Array()
	for i in range(segments):
		var a1 = TAU * float(i) / segments
		var a2 = TAU * float(i + 1) / segments
		lines.append(Vector3(cos(a1) * radius, y, sin(a1) * radius))
		lines.append(Vector3(cos(a2) * radius, y, sin(a2) * radius))
	return lines


func _to_local(node: Node3D, world_pos: Vector3) -> Vector3:
	return node.global_transform.affine_inverse() * world_pos


func _append_cross(lines: PackedVector3Array, center: Vector3, s: float) -> PackedVector3Array:
	lines.append(center + Vector3(-s, 0, 0))
	lines.append(center + Vector3(s, 0, 0))
	lines.append(center + Vector3(0, 0, -s))
	lines.append(center + Vector3(0, 0, s))
	lines.append(center + Vector3(0, -s, 0))
	lines.append(center + Vector3(0, s, 0))
	return lines
