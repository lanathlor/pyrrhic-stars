extends Node3D
## Particle trail guiding the player toward the dungeon portal in the hub.
## Bakes per-floor navigation meshes from manually defined geometry (CSG nodes
## are invisible to Godot's navmesh parser), then queries NavigationServer3D
## for collision-aware paths.  Wispy Flux-blue particles drift along the result.

const UPDATE_INTERVAL := 0.3
const PARTICLE_SPACING := 2.0
const MAX_PARTICLES_PER_EMITTER := 8
const MAX_EMITTERS := 30
const PARTICLE_LIFETIME := 1.4
const FADE_NEAR_DIST := 5.0
const HIDE_DIST := 3.0
const TRAIL_Y_OFFSET := -0.8  # pull particles down to ground level (navmesh sits ~1m above floor)

# Per-floor waypoint targets.
# Checked in order — first match wins. Lobby must be before plaza outdoor.
const FLOORS := [
	{
		# Lower District
		"target": Vector3(5.0, -199.8, -55.0),
		"y_min": -210.0, "y_max": -150.0,
		"arrival_radius": 5.0,
	},
	{
		# Tower lobby interior — guide to elevator door at back
		"target": Vector3(0.0, 0.2, 43.0),
		"y_min": -5.0, "y_max": 10.0,
		"arrival_radius": 3.0,
		"bounds_min": Vector3(-24, 0, -1),
		"bounds_max": Vector3(24, 0, 43),
	},
	{
		# Plaza outdoor — guide to tower front entrance
		"target": Vector3(0.0, 0.2, -1.0),
		"y_min": -5.0, "y_max": 10.0,
		"arrival_radius": 4.0,
	},
	{
		# Ops level — guide to portal on landing pad
		"target": Vector3(33.0, 100.2, 5.5),
		"y_min": 95.0, "y_max": 110.0,
		"arrival_radius": 4.0,
	},
]

var _timer: float = 0.0
var _emitters: Array[GPUParticles3D] = []
var _nav_ready: bool = false
var _nav_map_rid: RID
var _region_rids: Array[RID] = []

var _process_mat: ParticleProcessMaterial
var _mesh: QuadMesh


func _ready() -> void:
	_setup_particles()
	call_deferred("_bake_navigation")


func _process(delta: float) -> void:
	_timer += delta
	if _timer < UPDATE_INTERVAL:
		return
	_timer = 0.0
	_update_trail()


# =============================================================================
# Navigation baking — manual geometry since CSG is invisible to navmesh parser
# =============================================================================

func _bake_navigation() -> void:
	_nav_map_rid = NavigationServer3D.map_create()
	NavigationServer3D.map_set_active(_nav_map_rid, true)
	NavigationServer3D.map_set_cell_size(_nav_map_rid, 0.5)
	NavigationServer3D.map_set_cell_height(_nav_map_rid, 0.5)

	_bake_lower_district()
	_bake_plaza()
	_bake_ops()

	NavigationServer3D.map_force_update(_nav_map_rid)
	# Nav server needs physics frames to fully sync the map before queries work
	await get_tree().physics_frame
	await get_tree().physics_frame
	_nav_ready = true
	print("[PortalTrail] Navigation baked for 3 floors")


func _bake_floor(floor_y: float, walkable_faces: PackedVector3Array,
		obstructions: Array) -> void:
	var nav_mesh := NavigationMesh.new()
	nav_mesh.agent_radius = 1.0
	nav_mesh.agent_height = 2.0
	nav_mesh.agent_max_climb = 0.5
	nav_mesh.agent_max_slope = 50.0
	nav_mesh.cell_size = 0.5
	nav_mesh.cell_height = 0.5

	var source_geo := NavigationMeshSourceGeometryData3D.new()
	source_geo.add_faces(walkable_faces, Transform3D.IDENTITY)

	# Add buildings as solid wall boxes — the navmesh baker carves walkable
	# surface around vertical walls it encounters.
	for obs in obstructions:
		var cx: float = obs["center"].x
		var cz: float = obs["center"].z
		var sx: float = obs["size"].x
		var sz: float = obs["size"].z
		_add_box_faces(source_geo, Vector3(cx, floor_y + 2.5, cz), Vector3(sx, 5.0, sz))

	NavigationServer3D.bake_from_source_geometry_data(nav_mesh, source_geo)

	var region_rid := NavigationServer3D.region_create()
	NavigationServer3D.region_set_map(region_rid, _nav_map_rid)
	NavigationServer3D.region_set_navigation_mesh(region_rid, nav_mesh)
	_region_rids.append(region_rid)


func _add_box_faces(sg: NavigationMeshSourceGeometryData3D,
		center: Vector3, size: Vector3) -> void:
	## Add the 6 faces of a box to the source geometry.
	var hx := size.x / 2.0
	var hy := size.y / 2.0
	var hz := size.z / 2.0
	var cx := center.x
	var cy := center.y
	var cz := center.z
	# Bottom
	sg.add_faces(PackedVector3Array([
		Vector3(cx-hx, cy-hy, cz-hz), Vector3(cx+hx, cy-hy, cz-hz), Vector3(cx+hx, cy-hy, cz+hz),
		Vector3(cx-hx, cy-hy, cz-hz), Vector3(cx+hx, cy-hy, cz+hz), Vector3(cx-hx, cy-hy, cz+hz),
	]), Transform3D.IDENTITY)
	# Top
	sg.add_faces(PackedVector3Array([
		Vector3(cx-hx, cy+hy, cz-hz), Vector3(cx+hx, cy+hy, cz+hz), Vector3(cx+hx, cy+hy, cz-hz),
		Vector3(cx-hx, cy+hy, cz-hz), Vector3(cx-hx, cy+hy, cz+hz), Vector3(cx+hx, cy+hy, cz+hz),
	]), Transform3D.IDENTITY)
	# Front (Z-)
	sg.add_faces(PackedVector3Array([
		Vector3(cx-hx, cy-hy, cz-hz), Vector3(cx+hx, cy+hy, cz-hz), Vector3(cx+hx, cy-hy, cz-hz),
		Vector3(cx-hx, cy-hy, cz-hz), Vector3(cx-hx, cy+hy, cz-hz), Vector3(cx+hx, cy+hy, cz-hz),
	]), Transform3D.IDENTITY)
	# Back (Z+)
	sg.add_faces(PackedVector3Array([
		Vector3(cx-hx, cy-hy, cz+hz), Vector3(cx+hx, cy-hy, cz+hz), Vector3(cx+hx, cy+hy, cz+hz),
		Vector3(cx-hx, cy-hy, cz+hz), Vector3(cx+hx, cy+hy, cz+hz), Vector3(cx-hx, cy+hy, cz+hz),
	]), Transform3D.IDENTITY)
	# Left (X-)
	sg.add_faces(PackedVector3Array([
		Vector3(cx-hx, cy-hy, cz-hz), Vector3(cx-hx, cy-hy, cz+hz), Vector3(cx-hx, cy+hy, cz+hz),
		Vector3(cx-hx, cy-hy, cz-hz), Vector3(cx-hx, cy+hy, cz+hz), Vector3(cx-hx, cy+hy, cz-hz),
	]), Transform3D.IDENTITY)
	# Right (X+)
	sg.add_faces(PackedVector3Array([
		Vector3(cx+hx, cy-hy, cz-hz), Vector3(cx+hx, cy+hy, cz+hz), Vector3(cx+hx, cy-hy, cz+hz),
		Vector3(cx+hx, cy-hy, cz-hz), Vector3(cx+hx, cy+hy, cz-hz), Vector3(cx+hx, cy+hy, cz+hz),
	]), Transform3D.IDENTITY)


func _make_floor_quad(cx: float, cz: float, sx: float, sz: float,
		y: float) -> PackedVector3Array:
	## Returns two triangles forming a floor quad.
	var hx := sx / 2.0
	var hz := sz / 2.0
	return PackedVector3Array([
		Vector3(cx - hx, y, cz - hz), Vector3(cx + hx, y, cz - hz), Vector3(cx + hx, y, cz + hz),
		Vector3(cx - hx, y, cz - hz), Vector3(cx + hx, y, cz + hz), Vector3(cx - hx, y, cz + hz),
	])


func _bake_lower_district() -> void:
	## Floor at Y=-200, buildings as obstructions.
	## Ground: 200x200 centered at (5, -55).
	var y := -200.0
	var faces := _make_floor_quad(5.0, -55.0, 200.0, 200.0, y)

	# All CSG buildings with use_collision = true
	var buildings := [
		# Row A (north, Z -100 to -150)
		{ "center": Vector3(-65, 0, -125), "size": Vector3(50, 0, 45) },  # A1
		{ "center": Vector3(-20, 0, -130), "size": Vector3(30, 0, 30) },  # A2
		{ "center": Vector3(30, 0, -125),  "size": Vector3(40, 0, 40) },  # A3
		{ "center": Vector3(75, 0, -130),  "size": Vector3(25, 0, 35) },  # A4
		# Row B (west of plaza)
		{ "center": Vector3(-65, 0, -80),  "size": Vector3(40, 0, 15) },  # B1
		{ "center": Vector3(-68, 0, -55),  "size": Vector3(35, 0, 14) },  # B2
		{ "center": Vector3(-62, 0, -30),  "size": Vector3(42, 0, 16) },  # B3
		# Row C (east of plaza)
		{ "center": Vector3(30, 0, -82),   "size": Vector3(30, 0, 12) },  # C1
		{ "center": Vector3(65, 0, -80),   "size": Vector3(30, 0, 14) },  # C2
		{ "center": Vector3(35, 0, -42),   "size": Vector3(28, 0, 18) },  # C3
		{ "center": Vector3(70, 0, -38),   "size": Vector3(35, 0, 15) },  # C4
		# Row D (south)
		{ "center": Vector3(-60, 0, 15),   "size": Vector3(45, 0, 40) },  # D1
		{ "center": Vector3(-20, 0, 20),   "size": Vector3(30, 0, 30) },  # D2
		{ "center": Vector3(30, 0, 18),    "size": Vector3(35, 0, 35) },  # D3
		{ "center": Vector3(75, 0, 15),    "size": Vector3(28, 0, 35) },  # D4
		# Monument
		{ "center": Vector3(-8, 0, -55),   "size": Vector3(2, 0, 2) },
		# Lift shaft area
		{ "center": Vector3(5, 0, -55),    "size": Vector3(10, 0, 10) },
	]

	_bake_floor(y, faces, buildings)


func _bake_plaza() -> void:
	## Floor at Y=0, mostly open. Tower walls as obstructions but NOT the
	## north entrance (X[-9,9] Z=-1) — the player enters through the front door.
	var y := 0.0
	var faces := _make_floor_quad(0.0, 21.0, 250.0, 272.0, y)

	var obstructions := [
		# Lift shaft hole
		{ "center": Vector3(5, 0, -55),  "size": Vector3(10, 0, 10) },
		# Tower walls — split into pieces leaving the front entrance open
		# West wall (X=-25)
		{ "center": Vector3(-25, 0, 21),  "size": Vector3(1, 0, 44) },
		# East wall (X=25)
		{ "center": Vector3(25, 0, 21),   "size": Vector3(1, 0, 44) },
		# South wall (Z=43)
		{ "center": Vector3(0, 0, 43),    "size": Vector3(50, 0, 1) },
		# North wall left (NL: center X=-17, 16m wide → X[-25,-9])
		{ "center": Vector3(-17, 0, -1),  "size": Vector3(16, 0, 1) },
		# North wall right (NR: center X=17, 16m wide → X[9,25])
		{ "center": Vector3(17, 0, -1),   "size": Vector3(16, 0, 1) },
	]

	_bake_floor(y, faces, obstructions)


func _bake_ops() -> void:
	## Ops floor at Y=100, tower interior 50x44 + landing pad ramp.
	var y := 100.0
	# Main ops floor
	var faces := _make_floor_quad(0.0, 21.0, 50.0, 44.0, y)
	# Landing pad ramp (from tower east wall to portal)
	faces.append_array(_make_floor_quad(31.0, 5.5, 16.0, 12.0, y))

	var obstructions := [
		# Elevator shaft
		{ "center": Vector3(0, 0, 45), "size": Vector3(5, 0, 5) },
		# Ops partitions
		{ "center": Vector3(-18, 0, 10), "size": Vector3(12, 0, 0.5) },
		{ "center": Vector3(18, 0, 10),  "size": Vector3(12, 0, 0.5) },
		{ "center": Vector3(0, 0, 21),   "size": Vector3(2, 0, 0.5) },
		{ "center": Vector3(0, 0, 21),   "size": Vector3(0.5, 0, 11) },
	]

	_bake_floor(y, faces, obstructions)


# =============================================================================
# Trail update
# =============================================================================

func _update_trail() -> void:
	var player := _get_local_player()
	if not player:
		_clear_emitters()
		return

	var player_pos := player.global_position
	var floor_def := _get_current_floor(player_pos)
	if floor_def.is_empty():
		_clear_emitters()
		return

	var target: Vector3 = floor_def["target"]
	var arrival_radius: float = floor_def["arrival_radius"]
	var dist_xz := Vector2(player_pos.x - target.x, player_pos.z - target.z).length()
	if dist_xz < arrival_radius:
		_clear_emitters()
		return

	var path := _get_nav_path(player_pos, target)
	var total_dist := _path_length(path)
	if total_dist < HIDE_DIST:
		_clear_emitters()
		return
	var fade := clampf((total_dist - HIDE_DIST) / (FADE_NEAR_DIST - HIDE_DIST), 0.0, 1.0)

	var positions := _sample_path(path, PARTICLE_SPACING, MAX_EMITTERS)

	# Resize emitter pool
	while _emitters.size() > positions.size():
		var e: GPUParticles3D = _emitters.pop_back()
		e.queue_free()
	while _emitters.size() < positions.size():
		var e := _create_emitter()
		add_child(e)
		_emitters.append(e)

	for i in positions.size():
		_emitters[i].global_position = positions[i] + Vector3(0.0, TRAIL_Y_OFFSET, 0.0)
		_emitters[i].emitting = true
		var m: StandardMaterial3D = _emitters[i].draw_pass_1.material
		m.albedo_color.a = 0.6 * fade


func _get_nav_path(from: Vector3, to: Vector3) -> PackedVector3Array:
	if _nav_ready:
		var path := NavigationServer3D.map_get_path(_nav_map_rid, from, to, true)
		if path.size() >= 2:
			return path
	return PackedVector3Array([from, to])


func _get_current_floor(player_pos: Vector3) -> Dictionary:
	for floor_def in FLOORS:
		if player_pos.y < floor_def["y_min"] or player_pos.y > floor_def["y_max"]:
			continue
		# Optional XZ bounds check (for sub-zones like the tower lobby)
		if floor_def.has("bounds_min"):
			var bmin: Vector3 = floor_def["bounds_min"]
			var bmax: Vector3 = floor_def["bounds_max"]
			if player_pos.x < bmin.x or player_pos.x > bmax.x \
					or player_pos.z < bmin.z or player_pos.z > bmax.z:
				continue
		return floor_def
	return {}


func _path_length(path: PackedVector3Array) -> float:
	var total := 0.0
	for i in range(1, path.size()):
		total += path[i - 1].distance_to(path[i])
	return total


func _sample_path(path: PackedVector3Array, spacing: float, max_count: int) -> Array[Vector3]:
	var result: Array[Vector3] = []
	if path.size() < 2:
		return result

	var distances: Array[float] = [0.0]
	for i in range(1, path.size()):
		distances.append(distances[i - 1] + path[i - 1].distance_to(path[i]))
	var total := distances[distances.size() - 1]

	var d := spacing
	while d < total and result.size() < max_count:
		var seg := 0
		for j in range(1, distances.size()):
			if distances[j] >= d:
				seg = j - 1
				break
		var seg_len := distances[seg + 1] - distances[seg]
		if seg_len < 0.001:
			d += spacing
			continue
		var t := (d - distances[seg]) / seg_len
		result.append(path[seg].lerp(path[seg + 1], t))
		d += spacing

	return result


# =============================================================================
# Particle setup
# =============================================================================

func _setup_particles() -> void:
	_process_mat = ParticleProcessMaterial.new()
	_process_mat.gravity = Vector3(0.0, 0.8, 0.0)
	_process_mat.direction = Vector3(0.0, 1.0, 0.0)
	_process_mat.spread = 40.0
	_process_mat.initial_velocity_min = 0.2
	_process_mat.initial_velocity_max = 0.6
	_process_mat.emission_shape = ParticleProcessMaterial.EMISSION_SHAPE_SPHERE
	_process_mat.emission_sphere_radius = 0.3
	_process_mat.scale_min = 0.6
	_process_mat.scale_max = 1.0
	_process_mat.damping_min = 1.0
	_process_mat.damping_max = 2.0

	var alpha_curve := CurveTexture.new()
	var curve := Curve.new()
	curve.add_point(Vector2(0.0, 0.0))
	curve.add_point(Vector2(0.15, 0.7))
	curve.add_point(Vector2(0.7, 0.5))
	curve.add_point(Vector2(1.0, 0.0))
	alpha_curve.curve = curve
	_process_mat.alpha_curve = alpha_curve

	_mesh = QuadMesh.new()
	_mesh.size = Vector2(0.15, 0.15)
	var mat := StandardMaterial3D.new()
	mat.transparency = BaseMaterial3D.TRANSPARENCY_ALPHA
	mat.shading_mode = BaseMaterial3D.SHADING_MODE_UNSHADED
	mat.billboard_mode = BaseMaterial3D.BILLBOARD_ENABLED
	mat.albedo_color = Color(0.3, 0.55, 1.0, 0.6)
	mat.emission_enabled = true
	mat.emission = Color(0.2, 0.4, 0.9)
	mat.emission_energy_multiplier = 1.5
	mat.no_depth_test = false
	mat.render_priority = -1
	_mesh.material = mat


func _create_emitter() -> GPUParticles3D:
	var e := GPUParticles3D.new()
	e.amount = MAX_PARTICLES_PER_EMITTER
	e.lifetime = PARTICLE_LIFETIME
	e.process_material = _process_mat
	e.draw_pass_1 = _mesh.duplicate()
	e.emitting = true
	e.visibility_aabb = AABB(Vector3(-2, -2, -2), Vector3(4, 4, 4))
	return e


func _clear_emitters() -> void:
	for e in _emitters:
		if is_instance_valid(e):
			e.queue_free()
	_emitters.clear()


func _get_local_player() -> CharacterBody3D:
	var my_id := NetworkManager.get_my_id()
	for node in get_tree().get_nodes_in_group("players"):
		if node is CharacterBody3D and node.peer_id == my_id:
			return node
	return null


func _exit_tree() -> void:
	for rid in _region_rids:
		NavigationServer3D.free_rid(rid)
	if _nav_map_rid.is_valid():
		NavigationServer3D.free_rid(_nav_map_rid)
