extends Node

## Manages arena/hub environment loading, geometry, atmosphere, boss gate, exit portal.

const BOSS_ROOM_ENTRY_Z := 12.0
const EXIT_PORTAL_POS := Vector3(0.0, 0.1, 0.0)

var ctrl: Node

var current_env: Node3D = null
var boss_gate: CSGBox3D
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
	# Grab boss gate reference from arena scene
	if current_env.has_node("BossGate"):
		boss_gate = current_env.get_node("BossGate") as CSGBox3D
	else:
		boss_gate = null
	if ctrl._shared_hud:
		ctrl._shared_hud.set_environment(current_env)
	print("[Main] Loaded environment: %s" % scene_path)


func unload_environment() -> void:
	if current_env and is_instance_valid(current_env):
		current_env.queue_free()
		current_env = null
	ctrl.entity_mgr.clear_all_enemies()
	ctrl.entity_mgr.clear_all_npcs()
	boss_gate = null
	remove_portal_trail()


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
	if current_env and current_env.has_node("PortalTrail"):
		portal_trail = current_env.get_node("PortalTrail")
		portal_trail.visible = true
		portal_trail.set_process(true)
	else:
		portal_trail = null


func remove_portal_trail() -> void:
	if portal_trail and is_instance_valid(portal_trail):
		portal_trail.visible = false
		portal_trail.set_process(false)
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
