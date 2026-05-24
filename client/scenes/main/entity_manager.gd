extends Node

## Entity lifecycle: spawning/despawning players, projectiles, enemies, NPCs.

const CLASS_SCENES := {
	"gunner": "res://scenes/controllers/gunner/gunner.tscn",
	"vanguard": "res://scenes/controllers/vanguard/vanguard.tscn",
	"blade_dancer": "res://scenes/controllers/blade_dancer/blade_dancer.tscn",
	"arcanotechnicien": "res://scenes/controllers/arcanotechnicien/arcanotechnicien.tscn",
}
const NPC_MODEL_SCENE := "res://scenes/shared/character_model/character_model.tscn"
const NPC_PUPPET_SCRIPT := "res://scenes/shared/npc_puppet/npc_puppet.gd"

var ctrl: Node

var spawned_players: Dictionary = {}  # peer_id -> CharacterBody3D
var spawned_projectiles: Dictionary = {}  # proj_id -> Node3D
var enemy_nodes: Dictionary = {}  # enemy_id -> CharacterBody3D
var npc_nodes: Dictionary = {}  # npc_id -> Node3D

@onready var _players_node: Node3D = get_parent().get_node("Players")
@onready var _projectiles_node: Node3D = get_parent().get_node("Projectiles")


func _ready() -> void:
	ctrl = get_parent()


func spawn_player(
	peer_id: int, class_name_str: String, spawn_pos: Vector3, spec_id: String = ""
) -> void:
	if peer_id in spawned_players:
		return
	if not CLASS_SCENES.has(class_name_str):
		class_name_str = "gunner"
	var scene: PackedScene = load(CLASS_SCENES[class_name_str]) as PackedScene
	var player: CharacterBody3D = scene.instantiate() as CharacterBody3D
	player.name = "Player_%d" % peer_id
	player.peer_id = peer_id
	# Set spec before add_child so _ready() creates the correct combat subsystem
	if spec_id != "" and "spec_id" in player:
		player.spec_id = spec_id
	_players_node.add_child(player)
	player.add_to_group("players")
	player.global_position = spawn_pos
	# Initialize net sync targets so remote interpolation starts at the correct position
	player._net_position = spawn_pos
	player._net_rotation_y = player.rotation.y
	# Apply hub spawn facing direction for local player
	if ctrl.state == ctrl.GameState.HUB and peer_id == NetworkManager.get_my_id():
		player.rotation.y = ctrl.HUB_SPAWN_YAW
		player._net_rotation_y = ctrl.HUB_SPAWN_YAW
		if "_camera_yaw" in player:
			player._camera_yaw = ctrl.HUB_SPAWN_YAW
	spawned_players[peer_id] = player

	# Feed local player to shared HUD and connect death signal
	if peer_id == NetworkManager.get_my_id():
		if ctrl._shared_hud:
			ctrl._shared_hud.set_local_player(player, class_name_str, peer_id)
		if ctrl._map_overlay:
			ctrl._map_overlay.set_local_info(peer_id, ctrl._player_names)
		if player.has_signal("died"):
			player.died.connect(ctrl._on_local_player_died)

	# Add overhead name for remote players in hub
	if ctrl.state == ctrl.GameState.HUB and peer_id != NetworkManager.get_my_id():
		update_overhead_name(player, peer_id)


func despawn_all_players() -> void:
	for pid in spawned_players:
		var player = spawned_players[pid]
		if is_instance_valid(player):
			player.queue_free()
	spawned_players.clear()
	despawn_all_projectiles()
	if ctrl._shared_hud:
		ctrl._shared_hud.clear_local_player()


func despawn_all_projectiles() -> void:
	for pid in spawned_projectiles:
		var proj = spawned_projectiles[pid]
		if is_instance_valid(proj):
			proj.queue_free()
	spawned_projectiles.clear()


func spawn_projectile(
	proj_id: int,
	pos: Vector3,
	dir: Vector3,
	spd: float = 22.0,
	ang_vel: float = 0.0,
	tag: String = ""
) -> void:
	var scene: PackedScene = (
		load("res://scenes/enemies/basic_enemy/enemy_projectile.tscn") as PackedScene
	)
	if not scene:
		return
	var proj: Node3D = scene.instantiate() as Node3D
	proj.name = "Proj_%d" % proj_id
	_projectiles_node.add_child(proj)
	proj.global_position = pos
	if proj.has_method("setup"):
		proj.setup(dir, spd, ang_vel, tag)
	spawned_projectiles[proj_id] = proj


func update_enemies(enemies_data: Array) -> void:
	var seen_ids: Dictionary = {}
	for edata in enemies_data:
		var eid: int = edata["enemy_id"]
		seen_ids[eid] = true
		var alive: bool = edata["alive"]
		if alive and eid not in enemy_nodes:
			# Spawn new enemy node
			var scene: PackedScene = (
				load("res://scenes/enemies/basic_enemy/basic_enemy.tscn") as PackedScene
			)
			if scene:
				var node: CharacterBody3D = scene.instantiate() as CharacterBody3D
				node.name = "Enemy_%d" % eid
				node.peer_id = eid
				ctrl.add_child(node)
				enemy_nodes[eid] = node
		if eid in enemy_nodes:
			var node: CharacterBody3D = enemy_nodes[eid]
			if is_instance_valid(node):
				if alive:
					node.visible = true
					node.collision_layer = 4
					node.set_physics_process(true)
					if node.has_method("apply_server_state"):
						node.apply_server_state(edata)
				else:
					node.visible = false
					node.collision_layer = 0
					node.set_physics_process(false)
	# Remove enemies no longer in state
	var to_remove: Array = []
	for eid in enemy_nodes:
		if eid not in seen_ids:
			to_remove.append(eid)
	for eid in to_remove:
		var node = enemy_nodes[eid]
		if is_instance_valid(node):
			node.queue_free()
		enemy_nodes.erase(eid)


func clear_all_enemies() -> void:
	for eid in enemy_nodes:
		var node = enemy_nodes[eid]
		if is_instance_valid(node):
			node.queue_free()
	enemy_nodes.clear()


func update_npcs(npcs_data: Array) -> void:
	var seen_ids: Dictionary = {}
	for ndata in npcs_data:
		var nid: int = ndata["npc_id"]
		seen_ids[nid] = true
		if nid not in npc_nodes:
			var node: Node3D = _create_npc_node(ndata)
			ctrl.add_child(node)
			npc_nodes[nid] = node
		var node: Node3D = npc_nodes[nid]
		if is_instance_valid(node):
			var target_pos: Vector3 = ndata["pos"]
			node.global_position = node.global_position.lerp(target_pos, 0.15)
			node.rotation.y = ndata["rot_y"]
			var model: Node3D = node.get_node_or_null("CharacterModel")
			if model:
				model.position.y = 0.0
				if model.has_method("play_anim"):
					var npc_state: int = ndata["state"]
					var anim_name := "idle" if npc_state == 0 else "run"
					model.play_anim(anim_name)
	# Remove NPCs no longer in state
	var to_remove: Array = []
	for nid in npc_nodes:
		if nid not in seen_ids:
			to_remove.append(nid)
	for nid in to_remove:
		var node = npc_nodes[nid]
		if is_instance_valid(node):
			node.queue_free()
		npc_nodes.erase(nid)


func clear_all_npcs() -> void:
	for nid in npc_nodes:
		var node = npc_nodes[nid]
		if is_instance_valid(node):
			node.queue_free()
	npc_nodes.clear()


func update_overhead_name(player: CharacterBody3D, peer_id: int) -> void:
	var label: Label3D = player.get_node_or_null("OverheadName")
	if label == null:
		label = Label3D.new()
		label.name = "OverheadName"
		label.position = Vector3(0, 2.5, 0)
		label.billboard = BaseMaterial3D.BILLBOARD_ENABLED
		label.font_size = 48
		label.outline_size = 8
		label.modulate = Color(1, 1, 1, 0.9)
		label.no_depth_test = true
		player.add_child(label)
	var uname: String = ctrl._player_names.get(peer_id, "Player_%d" % peer_id)
	if label.text != uname:
		label.text = uname


func _create_npc_node(ndata: Dictionary) -> Node3D:
	var root := Node3D.new()
	root.name = "NPC_%d" % ndata["npc_id"]
	root.position = ndata["pos"]
	root.rotation.y = ndata["rot_y"]
	root.set_script(load(NPC_PUPPET_SCRIPT))

	var def_name: String = ndata.get("def_name", "citizen")

	# Character model (Mixamo) — same as enemies/players
	var model_scene: PackedScene = load(NPC_MODEL_SCENE) as PackedScene
	if model_scene:
		var model: Node3D = model_scene.instantiate()
		model.name = "CharacterModel"
		root.add_child(model)

	# Overhead label
	var label := Label3D.new()
	label.name = "NameLabel"
	label.text = def_name.capitalize()
	label.font_size = 32
	label.position.y = 1.9
	label.billboard = BaseMaterial3D.BILLBOARD_ENABLED
	label.no_depth_test = true
	label.modulate = Color(0.9, 0.9, 0.95, 0.8)
	root.add_child(label)

	return root
