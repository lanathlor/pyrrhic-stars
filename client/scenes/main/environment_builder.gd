extends Node

## Manages arena/hub environment loading, geometry, atmosphere, boss gate, exit portal.

const BOSS_ROOM_ENTRY_Z := 12.0
const EXIT_PORTAL_POS := Vector3(0.0, 0.1, 0.0)

var ctrl: Node

var current_env: Node3D = null
var boss_gate: CSGBox3D
var atmosphere: Node3D
var arena_buildings: Node3D
var exit_portal: CSGCylinder3D = null
var portal_trail: Node3D

var _exit_portal_prompt: Label = null
var _near_exit_portal: bool = false


func _ready() -> void:
	ctrl = get_parent()


func load_environment(scene_path: String) -> void:
	var scene: PackedScene = load(scene_path) as PackedScene
	current_env = scene.instantiate()
	ctrl.add_child(current_env)
	if ctrl._shared_hud:
		ctrl._shared_hud.set_environment(current_env)
	print("[Main] Loaded environment: %s" % scene_path)


func unload_environment() -> void:
	if current_env and is_instance_valid(current_env):
		current_env.queue_free()
		current_env = null
	ctrl.entity_mgr.clear_all_enemies()
	ctrl.entity_mgr.clear_all_npcs()
	if boss_gate and is_instance_valid(boss_gate):
		boss_gate.queue_free()
	boss_gate = null
	atmosphere = null
	if arena_buildings and is_instance_valid(arena_buildings):
		arena_buildings.queue_free()
	arena_buildings = null
	remove_portal_trail()


func create_hallway_geometry() -> void:
	# Hallway connects warmup room (Z 40-52) to boss room (Z -14.5 to 12).
	# Hallway spans Z 12 to 40, X -8 to 8 (narrower corridor).
	var mat_floor := StandardMaterial3D.new()
	mat_floor.albedo_color = Color(0.12, 0.12, 0.15)
	mat_floor.roughness = 0.3
	mat_floor.metallic = 0.25
	var mat_wall := StandardMaterial3D.new()
	mat_wall.albedo_color = Color(0.18, 0.18, 0.22)
	mat_wall.roughness = 0.75
	var mat_cover := StandardMaterial3D.new()
	mat_cover.albedo_color = Color(0.16, 0.18, 0.22)
	mat_cover.roughness = 0.7

	# Hallway floor (16 wide x 28 deep)
	var hall_floor := CSGBox3D.new()
	hall_floor.name = "HallwayFloor"
	hall_floor.size = Vector3(16.0, 0.5, 28.0)
	hall_floor.transform.origin = Vector3(0.0, -0.25, 26.0)
	hall_floor.use_collision = true
	hall_floor.collision_layer = 1
	hall_floor.collision_mask = 0
	hall_floor.material = mat_floor
	current_env.add_child(hall_floor)

	# Hallway left wall
	var hall_wall_l := CSGBox3D.new()
	hall_wall_l.name = "HallwayWallLeft"
	hall_wall_l.size = Vector3(0.5, 5.0, 28.0)
	hall_wall_l.transform.origin = Vector3(-8.0, 2.5, 26.0)
	hall_wall_l.use_collision = true
	hall_wall_l.collision_layer = 1
	hall_wall_l.collision_mask = 0
	hall_wall_l.material = mat_wall
	current_env.add_child(hall_wall_l)

	# Hallway right wall
	var hall_wall_r := CSGBox3D.new()
	hall_wall_r.name = "HallwayWallRight"
	hall_wall_r.size = Vector3(0.5, 5.0, 28.0)
	hall_wall_r.transform.origin = Vector3(8.0, 2.5, 26.0)
	hall_wall_r.use_collision = true
	hall_wall_r.collision_layer = 1
	hall_wall_r.collision_mask = 0
	hall_wall_r.material = mat_wall
	current_env.add_child(hall_wall_r)

	# Connector walls: fill the gap between hallway (X 8) and boss room (X 20)
	# Left connector at Z=12
	var conn_wall_l := CSGBox3D.new()
	conn_wall_l.name = "ConnectorWallLeft"
	conn_wall_l.size = Vector3(12.0, 5.0, 0.5)
	conn_wall_l.transform.origin = Vector3(-14.0, 2.5, 11.6)
	conn_wall_l.use_collision = true
	conn_wall_l.collision_layer = 1
	conn_wall_l.collision_mask = 0
	conn_wall_l.material = mat_wall
	current_env.add_child(conn_wall_l)

	# Right connector at Z=12
	var conn_wall_r := CSGBox3D.new()
	conn_wall_r.name = "ConnectorWallRight"
	conn_wall_r.size = Vector3(12.0, 5.0, 0.5)
	conn_wall_r.transform.origin = Vector3(14.0, 2.5, 11.6)
	conn_wall_r.use_collision = true
	conn_wall_r.collision_layer = 1
	conn_wall_r.collision_mask = 0
	conn_wall_r.material = mat_wall
	current_env.add_child(conn_wall_r)

	# Hallway cover obstacles (matching server hallway obstacles)
	var cover_positions := [
		Vector3(-4.0, 0.6, 27.0),
		Vector3(4.0, 0.6, 27.0),
		Vector3(-4.0, 0.6, 17.0),
		Vector3(4.0, 0.6, 17.0),
	]
	for i in cover_positions.size():
		var cover := CSGBox3D.new()
		cover.name = "HallwayCover%d" % i
		cover.size = Vector3(2.0, 1.2, 1.0)
		cover.transform.origin = cover_positions[i]
		cover.use_collision = true
		cover.collision_layer = 1
		cover.collision_mask = 0
		cover.material = mat_cover
		current_env.add_child(cover)

	# Boss room gate at Z=12 (hidden by default, closes when boss aggros)
	boss_gate = CSGBox3D.new()
	boss_gate.size = Vector3(40.0, 5.0, 0.5)
	boss_gate.transform.origin = Vector3(0.0, 2.5, BOSS_ROOM_ENTRY_Z)
	boss_gate.use_collision = true
	boss_gate.collision_layer = 1
	boss_gate.collision_mask = 0
	var gate_mat := StandardMaterial3D.new()
	gate_mat.albedo_color = Color(0.6, 0.15, 0.15)
	gate_mat.emission_enabled = true
	gate_mat.emission = Color(0.5, 0.1, 0.1)
	gate_mat.emission_energy_multiplier = 0.5
	boss_gate.material = gate_mat
	boss_gate.visible = false
	boss_gate.use_collision = false
	current_env.add_child(boss_gate)


func create_arena_buildings() -> void:
	if arena_buildings and is_instance_valid(arena_buildings):
		arena_buildings.queue_free()
	var buildings_script: GDScript = load("res://scenes/environments/arena/arena_buildings.gd")
	arena_buildings = Node3D.new()
	arena_buildings.name = "ArenaBuildings"
	arena_buildings.set_script(buildings_script)
	current_env.add_child(arena_buildings)


func create_atmosphere() -> void:
	if atmosphere and is_instance_valid(atmosphere):
		atmosphere.queue_free()
	var atmosphere_script: GDScript = load("res://scenes/environments/arena/dungeon_atmosphere.gd")
	atmosphere = Node3D.new()
	atmosphere.name = "DungeonAtmosphere"
	atmosphere.set_script(atmosphere_script)
	current_env.add_child(atmosphere)


func close_boss_gate() -> void:
	if boss_gate:
		boss_gate.visible = true
		boss_gate.use_collision = true
	# Push local player into the boss room if near the gate
	var entity_mgr: Node = ctrl.entity_mgr
	var my_id: int = NetworkManager.get_my_id()
	if my_id in entity_mgr.spawned_players:
		var player: CharacterBody3D = entity_mgr.spawned_players[my_id]
		if (
			is_instance_valid(player)
			and player.global_position.z > BOSS_ROOM_ENTRY_Z - 2.0
			and player.global_position.z < BOSS_ROOM_ENTRY_Z + 2.0
		):
			player.global_position.z = BOSS_ROOM_ENTRY_Z - 3.0


func open_boss_gate() -> void:
	if boss_gate:
		boss_gate.visible = false
		boss_gate.use_collision = false


func spawn_exit_portal() -> void:
	if exit_portal:
		return
	exit_portal = CSGCylinder3D.new()
	exit_portal.radius = 1.5
	exit_portal.height = 0.1
	exit_portal.transform.origin = EXIT_PORTAL_POS
	var mat := StandardMaterial3D.new()
	mat.albedo_color = Color(0.2, 0.5, 1.0, 0.7)
	mat.transparency = BaseMaterial3D.TRANSPARENCY_ALPHA
	mat.emission_enabled = true
	mat.emission = Color(0.3, 0.6, 1.0)
	mat.emission_energy_multiplier = 2.0
	exit_portal.material = mat
	if current_env:
		current_env.add_child(exit_portal)
	else:
		ctrl.add_child(exit_portal)

	# Create prompt label
	_exit_portal_prompt = Label.new()
	_exit_portal_prompt.text = "[E] Return to Hub"
	_exit_portal_prompt.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	_exit_portal_prompt.anchor_left = 0.3
	_exit_portal_prompt.anchor_right = 0.7
	_exit_portal_prompt.anchor_top = 0.6
	_exit_portal_prompt.anchor_bottom = 0.65
	ctrl.ui_ctrl.apply_overlay_label(_exit_portal_prompt, 18, ctrl.UI_BORDER_ACTIVE)
	_exit_portal_prompt.visible = false
	if ctrl._shared_hud_layer:
		ctrl._shared_hud_layer.add_child(_exit_portal_prompt)


func remove_exit_portal() -> void:
	if exit_portal and is_instance_valid(exit_portal):
		exit_portal.queue_free()
	exit_portal = null
	if _exit_portal_prompt and is_instance_valid(_exit_portal_prompt):
		_exit_portal_prompt.queue_free()
	_exit_portal_prompt = null
	_near_exit_portal = false


func check_exit_portal_proximity() -> void:
	if not exit_portal or not is_instance_valid(exit_portal):
		_near_exit_portal = false
		if _exit_portal_prompt and is_instance_valid(_exit_portal_prompt):
			_exit_portal_prompt.visible = false
		return
	var entity_mgr: Node = ctrl.entity_mgr
	var my_id: int = NetworkManager.get_my_id()
	if my_id not in entity_mgr.spawned_players:
		_near_exit_portal = false
		return
	var player: CharacterBody3D = entity_mgr.spawned_players[my_id]
	if not is_instance_valid(player):
		return
	var dist: float = player.global_position.distance_to(EXIT_PORTAL_POS)
	_near_exit_portal = dist < 3.0
	if _exit_portal_prompt and is_instance_valid(_exit_portal_prompt):
		_exit_portal_prompt.visible = _near_exit_portal


func is_near_exit_portal() -> bool:
	return _near_exit_portal


func create_portal_trail() -> void:
	remove_portal_trail()
	var trail_script: GDScript = load("res://scenes/environments/prime_hub/portal_trail.gd")
	portal_trail = Node3D.new()
	portal_trail.name = "PortalTrail"
	portal_trail.set_script(trail_script)
	ctrl.add_child(portal_trail)


func remove_portal_trail() -> void:
	if portal_trail and is_instance_valid(portal_trail):
		portal_trail.queue_free()
	portal_trail = null


func bake_hub_navigation() -> void:
	var nav_region := NavigationRegion3D.new()
	var nav_mesh := NavigationMesh.new()
	nav_mesh.agent_radius = 0.8
	nav_mesh.agent_height = 2.0
	nav_mesh.cell_size = 0.25
	nav_mesh.cell_height = 0.25
	nav_region.navigation_mesh = nav_mesh
	ctrl.add_child(nav_region)

	var source_geo := NavigationMeshSourceGeometryData3D.new()
	# Hub floor: 30x25 centered at (0, 0, 2.5)
	(
		source_geo
		. add_faces(
			PackedVector3Array(
				[
					Vector3(-15, 0, -10),
					Vector3(15, 0, -10),
					Vector3(15, 0, 15),
					Vector3(-15, 0, -10),
					Vector3(15, 0, 15),
					Vector3(-15, 0, 15),
				]
			),
			Transform3D.IDENTITY
		)
	)

	NavigationServer3D.bake_from_source_geometry_data(nav_mesh, source_geo)
