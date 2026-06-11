extends Node

## Manages arena/hub environment loading, geometry, atmosphere, gates, exit portal.

const EXIT_PORTAL_POS := Vector3(0.0, 0.1, 0.0)

var ctrl: Node

var current_env: Node3D = null
var gates: Dictionary = {}  # gate_id → Node3D (CSGBox3D or similar)
var exit_portal: CSGCylinder3D = null
var portal_trail: Node3D

var _exit_portal_prompt: Label = null
var _near_exit_portal: bool = false


func _ready() -> void:
	ctrl = get_parent()


func load_environment(scene_path: String) -> void:
	if current_env and is_instance_valid(current_env):
		push_error(
			(
				(
					"[EnvironmentBuilder] BUG: loading %s while %s is still in tree. "
					+ "Call unload_environment() first."
				)
				% [scene_path, current_env.name]
			)
		)
		var msg := "dual env: %s loaded when loading %s" % [current_env.name, scene_path]
		assert(false, msg)
	var we_before := _count_world_environments(ctrl.get_tree().root)
	var scene: PackedScene = load(scene_path) as PackedScene
	current_env = scene.instantiate()
	ctrl.add_child(current_env)
	var we_after := _count_world_environments(ctrl.get_tree().root)
	# Scan for gate nodes tagged with server_gate group
	_discover_gates(current_env)
	if ctrl._shared_hud:
		ctrl._shared_hud.set_environment(current_env)
	print("[EnvBuilder] loaded %s (WorldEnvs: %d -> %d)" % [scene_path, we_before, we_after])


func _discover_gates(root: Node) -> void:
	gates.clear()
	_walk_for_gates(root)
	# Apply default state: gates start invisible unless default_closed
	for gate_id in gates:
		var node: Node3D = gates[gate_id]
		var default_closed: bool = node.get_meta("default_closed", false)
		if default_closed:
			node.visible = true
			if node is CSGBox3D:
				(node as CSGBox3D).use_collision = true
		else:
			node.visible = false
			if node is CSGBox3D:
				(node as CSGBox3D).use_collision = false


func _walk_for_gates(node: Node) -> void:
	if node.is_in_group("server_gate"):
		var gate_id: String = str(node.get_meta("gate_id", ""))
		if gate_id != "":
			gates[gate_id] = node
	for child in node.get_children():
		_walk_for_gates(child)


func unload_environment() -> void:
	if current_env and is_instance_valid(current_env):
		var env_name: String = current_env.name
		# Count WorldEnvironment nodes before unload
		var we_count := _count_world_environments(ctrl.get_tree().root)
		print("[EnvBuilder] unload %s (WorldEnvs before: %d)" % [env_name, we_count])
		# Immediate free: both hub and arena have a WorldEnvironment node.
		# queue_free defers deletion, so both coexist for a frame. Godot
		# only supports one active WorldEnvironment per viewport; the
		# overlap breaks rendering on the next zone entry.
		current_env.get_parent().remove_child(current_env)
		current_env.free()
		current_env = null
		var we_after := _count_world_environments(ctrl.get_tree().root)
		print("[EnvBuilder] unloaded %s (WorldEnvs after: %d)" % [env_name, we_after])
	else:
		print("[EnvBuilder] unload: nothing to unload (current_env=%s)" % str(current_env))
	ctrl.entity_mgr.clear_all_enemies()
	ctrl.entity_mgr.clear_all_npcs()
	gates.clear()
	remove_portal_trail()


func close_gate(gate_id: String) -> void:
	if gate_id not in gates:
		return
	var node: Node3D = gates[gate_id]
	node.visible = true
	if node is CSGBox3D:
		(node as CSGBox3D).use_collision = true
	# Push local player through the gate if overlapping
	var push_axis: String = str(node.get_meta("push_axis", ""))
	var push_offset: float = float(node.get_meta("push_offset", 0.0))
	if push_axis == "":
		return
	var entity_mgr: Node = ctrl.entity_mgr
	var my_id: int = NetworkManager.get_my_id()
	if my_id in entity_mgr.spawned_players:
		var player: CharacterBody3D = entity_mgr.spawned_players[my_id]
		if not is_instance_valid(player):
			return
		var gate_pos: float = 0.0
		var player_pos: float = 0.0
		match push_axis:
			"x":
				gate_pos = node.global_position.x
				player_pos = player.global_position.x
			"z":
				gate_pos = node.global_position.z
				player_pos = player.global_position.z
			_:
				return
		if abs(player_pos - gate_pos) < 2.0:
			match push_axis:
				"x":
					player.global_position.x = gate_pos + push_offset
				"z":
					player.global_position.z = gate_pos + push_offset


func open_gate(gate_id: String) -> void:
	if gate_id not in gates:
		return
	var node: Node3D = gates[gate_id]
	node.visible = false
	if node is CSGBox3D:
		(node as CSGBox3D).use_collision = false


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


func _count_world_environments(node: Node) -> int:
	var count := 0
	if node is WorldEnvironment:
		count += 1
	for child in node.get_children():
		count += _count_world_environments(child)
	return count
