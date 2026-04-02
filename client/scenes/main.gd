extends Node3D

## Game flow: Menu -> Hub -> Arena (Lobby -> Fight -> Fight Over) -> Hub.
## Server-authoritative: all game flow is driven by server events.

enum GameState { MENU, HUB, ARENA_LOBBY, FIGHT, FIGHT_OVER }

var state: GameState = GameState.MENU
var paused: bool = false

const HUB_SCENE := "res://scenes/environments/hub/hub.tscn"
const ARENA_SCENE := "res://scenes/environments/arena/arena.tscn"
const EXIT_PORTAL_POS := Vector3(0.0, 0.1, 0.0)

const LOBBY_SPAWN := Vector3(0.0, 0.1, 20.0)
const ENEMY_SPAWN := Vector3(0.0, 0.1, 0.0)
const PLAYER_SPAWNS := [
	Vector3(-2.0, 0.1, 20.0),
	Vector3(0.0, 0.1, 20.0),
	Vector3(2.0, 0.1, 20.0),
	Vector3(-1.0, 0.1, 21.0),
	Vector3(1.0, 0.1, 21.0),
]
const HUB_SPAWNS := [
	Vector3(-2.0, 0.1, -5.0),
	Vector3(0.0, 0.1, -5.0),
	Vector3(2.0, 0.1, -5.0),
	Vector3(-1.0, 0.1, -3.0),
	Vector3(1.0, 0.1, -3.0),
]

const CLASS_SCENES := {
	"gunner": "res://scenes/controllers/gunner/gunner.tscn",
	"vanguard": "res://scenes/controllers/vanguard/vanguard.tscn",
	"blade_dancer": "res://scenes/controllers/blade_dancer/blade_dancer.tscn",
}

var _spawned_players: Dictionary = {}  # peer_id -> CharacterBody3D
var _spawned_projectiles: Dictionary = {}  # proj_id -> Node3D
var _local_class: String = "gunner"
var _players_node: Node3D
var _projectiles_node: Node3D
var _username: String = ""

# Scene management
var _current_env: Node3D = null
var _enemy_node: CharacterBody3D = null

# Dynamic nodes
var _gate: CSGBox3D
var _pause_layer: CanvasLayer
var _menu_layer: CanvasLayer
var _lobby_layer: CanvasLayer
var _lobby_status_label: Label
var _lobby_players_label: Label
var _lobby_class_label: Label
var _address_input: LineEdit
var _username_input: LineEdit

# Hub UI
var _hub_layer: CanvasLayer
var _hub_class_label: Label
var _hub_status_label: Label
var _portal_prompt: Label

# Group UI
var _group_panel: PanelContainer
var _group_label: Label
var _group_create_btn: Button
var _group_leave_btn: Button
var _invite_popup: CanvasLayer
var _invite_label: Label
var _pending_invite_group_id: int = 0

# Hub state
var _near_portal: bool = false
var _player_names: Dictionary = {}  # peer_id -> username
var _group_data: Dictionary = {}  # current group state
var _aimed_peer_id: int = 0  # peer id under crosshair for invite
var _cursor_toggled: bool = false  # backtick toggle state
var _alt_held: bool = false        # alt hold state

# Shared HUD (boss frame, group frames, damage meter, minimap, player status)
var _shared_hud_layer: CanvasLayer
var _shared_hud: Control

# Death overlay
var _death_overlay_layer: CanvasLayer
var _death_overlay_bg: ColorRect
var _death_label: Label
var _respawn_btn: Button
var _respawn_hub_btn: Button
var _local_player_dead: bool = false

# Exit portal (spawns after boss kill)
var _exit_portal: CSGCylinder3D = null
var _near_exit_portal: bool = false
var _exit_portal_prompt: Label = null


func _ready() -> void:
	process_mode = Node.PROCESS_MODE_ALWAYS

	# Create players container
	_players_node = Node3D.new()
	_players_node.name = "Players"
	add_child(_players_node)

	# Create projectiles container
	_projectiles_node = Node3D.new()
	_projectiles_node.name = "Projectiles"
	add_child(_projectiles_node)

	_create_pause_menu()
	_create_menu_ui()
	_create_lobby_ui()
	_create_hub_ui()
	_create_group_panel()
	_create_invite_popup()
	_create_shared_hud()
	_create_death_overlay()

	# Connect network signals
	NetworkManager.player_disconnected.connect(_on_net_player_disconnected)
	NetworkManager.connection_succeeded.connect(_on_net_connected)
	NetworkManager.connection_failed.connect(_on_net_connection_failed)
	# Server-authoritative signals
	NetworkManager.game_flow_event.connect(_on_game_flow_event)
	NetworkManager.player_info_changed.connect(_update_lobby_display)
	NetworkManager.world_state_received.connect(_on_world_state)
	NetworkManager.damage_event_received.connect(_on_damage_event)
	NetworkManager.zone_transfer_received.connect(_on_zone_transfer)
	NetworkManager.group_state_updated.connect(_on_group_state)
	NetworkManager.group_invite_received.connect(_on_group_invite)
	NetworkManager.group_error_received.connect(_on_group_error)

	_enter_menu()


func _input(event: InputEvent) -> void:
	if event.is_action_pressed("ui_cancel"):
		if state == GameState.FIGHT_OVER:
			return
		if state == GameState.MENU:
			return
		_toggle_pause()
		get_viewport().set_input_as_handled()

	# Cursor mode: Alt (hold) + backtick/tilde (toggle)
	if not paused and (state == GameState.FIGHT or state == GameState.FIGHT_OVER or state == GameState.HUB or state == GameState.ARENA_LOBBY):
		if event is InputEventKey:
			# Backtick toggle
			if event.physical_keycode == KEY_QUOTELEFT and event.pressed and not event.echo:
				_cursor_toggled = not _cursor_toggled
				_update_cursor_mode()
				get_viewport().set_input_as_handled()
			# Alt hold
			elif event.keycode == KEY_ALT:
				_alt_held = event.pressed
				_update_cursor_mode()

	# Class selection in hub or arena lobby
	if (state == GameState.HUB or state == GameState.ARENA_LOBBY) and not paused:
		if event is InputEventKey and event.pressed:
			if event.physical_keycode == KEY_1:
				_select_class("gunner")
			elif event.physical_keycode == KEY_2:
				_select_class("vanguard")
			elif event.physical_keycode == KEY_3:
				_select_class("blade_dancer")

	# Ready toggle in arena lobby
	if state == GameState.ARENA_LOBBY and not paused:
		if event is InputEventKey and event.pressed:
			if event.physical_keycode == KEY_ENTER or event.physical_keycode == KEY_KP_ENTER:
				_toggle_ready()

	# Hub interactions
	if state == GameState.HUB and not paused:
		if event is InputEventKey and event.pressed:
			if event.physical_keycode == KEY_E:
				if _near_portal:
					NetworkManager.send_enter_portal()
				elif _aimed_peer_id > 0:
					NetworkManager.send_group_invite(_aimed_peer_id)
			elif event.physical_keycode == KEY_G:
				# Toggle group: create if not in group, leave if in group
				if _group_data.get("group_id", 0) > 0:
					NetworkManager.send_group_leave()
				else:
					NetworkManager.send_group_create()

	# Arena exit portal interaction
	if state == GameState.FIGHT_OVER and not paused:
		if event is InputEventKey and event.pressed:
			if event.physical_keycode == KEY_E:
				if _near_exit_portal:
					NetworkManager.send_interact(2)  # InteractExitPortal


func _physics_process(_delta: float) -> void:
	if state == GameState.HUB:
		_check_portal_proximity()
		_check_aim_at_player()
	elif state == GameState.FIGHT_OVER:
		_check_exit_portal_proximity()


# =============================================================================
# Menu
# =============================================================================

func _enter_menu() -> void:
	state = GameState.MENU
	NetworkManager.disconnect_game()
	_menu_layer.visible = true
	_lobby_layer.visible = false
	_hub_layer.visible = false
	_pause_layer.visible = false
	Input.set_mouse_mode(Input.MOUSE_MODE_VISIBLE)
	_unload_environment()


func _on_connect_pressed() -> void:
	var address := _address_input.text.strip_edges()
	if address == "":
		address = "109.222.207.243"
	_connect_to_address(address)


func _connect_to_address(address: String) -> void:
	_username = _username_input.text.strip_edges()
	if _username == "":
		_username_input.grab_focus()
		return
	NetworkManager.username = _username

	NetworkManager.disconnect_game()
	var err := NetworkManager.connect_to_server(address)
	if err != OK:
		print("[Main] Failed to connect: %s" % error_string(err))
		return
	print("[Main] Connecting to %s:%d..." % [address, NetworkManager.DEFAULT_PORT])
	_menu_layer.visible = false


func _on_net_connected() -> void:
	print("[Main] Connected as peer %d" % NetworkManager.get_my_id())
	_enter_hub()


func _on_net_connection_failed() -> void:
	print("[Main] Connection failed")
	_enter_menu()


func _on_net_player_disconnected(peer_id: int) -> void:
	print("[Main] Peer %d disconnected" % peer_id)
	if peer_id in _spawned_players:
		var player: CharacterBody3D = _spawned_players[peer_id]
		if is_instance_valid(player):
			player.queue_free()
		_spawned_players.erase(peer_id)


# =============================================================================
# Hub
# =============================================================================

func _enter_hub() -> void:
	state = GameState.HUB
	get_tree().paused = false
	paused = false
	_pause_layer.visible = false
	_menu_layer.visible = false
	_lobby_layer.visible = false
	_hub_layer.visible = true
	Input.set_mouse_mode(Input.MOUSE_MODE_CAPTURED)
	_near_portal = false
	_portal_prompt.visible = false

	# Load hub scene if not already loaded
	if _current_env == null or _current_env.name != "Hub":
		_unload_environment()
		_load_environment(HUB_SCENE)

	# Despawn existing players
	_despawn_all_players()

	# Spawn local player in hub
	var my_id := NetworkManager.get_my_id()
	if my_id > 0:
		_spawn_player(my_id, _local_class, HUB_SPAWNS[0])

	_update_hub_display()
	_update_group_panel()
	if _shared_hud:
		_shared_hud.on_enter_hub()


func _check_portal_proximity() -> void:
	var my_id := NetworkManager.get_my_id()
	if my_id not in _spawned_players:
		_near_portal = false
		_portal_prompt.visible = false
		return
	var player: CharacterBody3D = _spawned_players[my_id]
	if not is_instance_valid(player):
		return
	# Portal is at (0, 2, 12) in hub
	var portal_pos := Vector3(0.0, 0.1, 12.0)
	var dist := player.global_position.distance_to(portal_pos)
	_near_portal = dist < 4.0
	_portal_prompt.visible = _near_portal


func _update_hub_display() -> void:
	if _hub_class_label:
		_hub_class_label.text = "[1] Gunner  [2] Vanguard  [3] Blade Dancer\nSelected: %s" % _local_class.to_upper()


func _update_overhead_name(player: CharacterBody3D, peer_id: int) -> void:
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
	var uname: String = _player_names.get(peer_id, "Player_%d" % peer_id)
	if label.text != uname:
		label.text = uname


func _check_aim_at_player() -> void:
	_aimed_peer_id = 0
	var my_id := NetworkManager.get_my_id()
	if my_id not in _spawned_players:
		return
	var local_player: CharacterBody3D = _spawned_players[my_id]
	if not is_instance_valid(local_player):
		return
	# Get camera
	var camera := get_viewport().get_camera_3d()
	if not camera:
		return
	var from := camera.global_position
	var forward := -camera.global_transform.basis.z
	var to := from + forward * 15.0

	# Simple distance-based check against remote players
	var best_dist := 3.0  # max aim distance
	for pid in _spawned_players:
		if pid == my_id:
			continue
		var p: CharacterBody3D = _spawned_players[pid]
		if not is_instance_valid(p):
			continue
		# Project point onto ray
		var to_player := p.global_position - from
		var dot := to_player.dot(forward)
		if dot < 0 or dot > 15.0:
			continue
		var closest_on_ray := from + forward * dot
		var dist := closest_on_ray.distance_to(p.global_position + Vector3(0, 1, 0))
		if dist < best_dist:
			best_dist = dist
			_aimed_peer_id = pid

	# Update hub status with aim info
	if _hub_status_label:
		if _aimed_peer_id > 0 and not _near_portal:
			var uname: String = _player_names.get(_aimed_peer_id, "Player_%d" % _aimed_peer_id)
			_hub_status_label.text = "Press [E] to invite %s" % uname
		elif not _near_portal:
			if _group_data.get("group_id", 0) > 0:
				_hub_status_label.text = "In group - Walk to portal | [G] Leave group"
			else:
				_hub_status_label.text = "[G] Create group | Aim at player + [E] to invite"


# =============================================================================
# Group handlers
# =============================================================================

func _on_group_state(data: Dictionary) -> void:
	_group_data = data
	_update_group_panel()
	if _shared_hud:
		_shared_hud.update_group_members(data)


func _on_group_invite(group_id: int, leader_name: String) -> void:
	_pending_invite_group_id = group_id
	_invite_label.text = "%s invites you to a group\n[Accept]  [Decline]" % leader_name
	_invite_popup.visible = true
	Input.set_mouse_mode(Input.MOUSE_MODE_VISIBLE)
	# Auto-decline after 30 seconds
	get_tree().create_timer(30.0).timeout.connect(func():
		if _invite_popup.visible and _pending_invite_group_id == group_id:
			_decline_invite()
	)


func _on_group_error(code: int, msg: String) -> void:
	print("[Main] Group error: %s" % msg)
	if _hub_status_label:
		_hub_status_label.text = "Error: %s" % msg


func _accept_invite() -> void:
	if _pending_invite_group_id > 0:
		NetworkManager.send_group_invite_reply(_pending_invite_group_id, true)
		_pending_invite_group_id = 0
	_invite_popup.visible = false
	if state == GameState.HUB:
		Input.set_mouse_mode(Input.MOUSE_MODE_CAPTURED)


func _decline_invite() -> void:
	if _pending_invite_group_id > 0:
		NetworkManager.send_group_invite_reply(_pending_invite_group_id, false)
		_pending_invite_group_id = 0
	_invite_popup.visible = false
	if state == GameState.HUB:
		Input.set_mouse_mode(Input.MOUSE_MODE_CAPTURED)


func _update_group_panel() -> void:
	if not _group_panel:
		return
	var group_id: int = _group_data.get("group_id", 0)
	if group_id == 0:
		_group_label.text = "No group\n[G] Create group"
		_group_leave_btn.visible = false
		_group_panel.visible = state == GameState.HUB
		return

	var leader_peer: int = _group_data.get("leader_peer", 0)
	var members: Array = _group_data.get("members", [])
	var text := "Group:\n"
	for m in members:
		var uname: String = m.get("username", "???")
		var pid: int = m.get("peer_id", 0)
		var leader_str := " *" if pid == leader_peer else ""
		var you_str := " (you)" if pid == NetworkManager.get_my_id() else ""
		text += "  %s%s%s\n" % [uname, leader_str, you_str]
	_group_label.text = text
	_group_leave_btn.visible = true
	_group_panel.visible = state == GameState.HUB


# =============================================================================
# Arena Lobby (warmup room)
# =============================================================================

func _enter_arena_lobby() -> void:
	state = GameState.ARENA_LOBBY
	get_tree().paused = false
	paused = false
	_pause_layer.visible = false
	_menu_layer.visible = false
	_remove_exit_portal()
	_hub_layer.visible = false
	_lobby_layer.visible = true
	Input.set_mouse_mode(Input.MOUSE_MODE_VISIBLE)

	# Load arena scene if not already loaded
	if _current_env == null or _current_env.name != "Arena":
		_unload_environment()
		_load_environment(ARENA_SCENE)
		_create_gate()
		_create_enemy()

	# Hide enemy
	if _enemy_node:
		_enemy_node.visible = false
		_enemy_node.collision_layer = 0
		_enemy_node.set_physics_process(false)

	# Open gate
	if _gate:
		_gate.visible = false
		_gate.use_collision = false

	_update_lobby_display()


func _select_class(class_name_str: String) -> void:
	_local_class = class_name_str
	if NetworkManager.is_active:
		NetworkManager.set_player_class(class_name_str)
	_update_lobby_display()
	_update_hub_display()


func _toggle_ready() -> void:
	if not NetworkManager.is_active:
		return
	var my_id := NetworkManager.get_my_id()
	if my_id not in NetworkManager.player_info:
		return
	var currently_ready: bool = NetworkManager.player_info[my_id]["ready"]
	NetworkManager.set_player_ready(not currently_ready)


func _update_lobby_display() -> void:
	if not _lobby_layer or not _lobby_layer.visible:
		return

	_lobby_class_label.text = "[1] Gunner   [2] Vanguard   [3] Blade Dancer\nSelected: %s" % _local_class.to_upper()

	if not NetworkManager.is_active:
		_lobby_status_label.text = "Connecting..."
		_lobby_players_label.text = ""
		return

	var text := "Players:\n"
	for pid in NetworkManager.player_info:
		var info: Dictionary = NetworkManager.player_info[pid]
		var ready_str := " [READY]" if info["ready"] else ""
		var you_str := " (you)" if pid == NetworkManager.get_my_id() else ""
		var uname: String = _player_names.get(pid, "Peer %d" % pid)
		text += "  %s: %s%s%s\n" % [uname, info["class_name"].to_upper(), ready_str, you_str]

	_lobby_players_label.text = text

	var my_id := NetworkManager.get_my_id()
	var am_ready: bool = NetworkManager.player_info.get(my_id, {}).get("ready", false)
	if am_ready:
		_lobby_status_label.text = "Waiting for other players..."
	else:
		_lobby_status_label.text = "Press ENTER when ready"


# =============================================================================
# Spawning
# =============================================================================

func _spawn_multiplayer_players() -> void:
	_lobby_layer.visible = false

	var spawn_idx := 0
	for pid in NetworkManager.player_info:
		var info: Dictionary = NetworkManager.player_info[pid]
		var class_name_str: String = info["class_name"]
		if not CLASS_SCENES.has(class_name_str):
			push_error("[Main] Unknown class: %s" % class_name_str)
			continue
		var spawn_pos: Vector3 = PLAYER_SPAWNS[spawn_idx % PLAYER_SPAWNS.size()]
		_spawn_player(pid, class_name_str, spawn_pos)
		spawn_idx += 1

	Input.set_mouse_mode(Input.MOUSE_MODE_CAPTURED)


func _spawn_player(peer_id: int, class_name_str: String, spawn_pos: Vector3) -> void:
	if peer_id in _spawned_players:
		return
	if not CLASS_SCENES.has(class_name_str):
		class_name_str = "gunner"
	var scene := load(CLASS_SCENES[class_name_str]) as PackedScene
	var player := scene.instantiate() as CharacterBody3D
	player.name = "Player_%d" % peer_id
	player.peer_id = peer_id
	_players_node.add_child(player)
	player.global_position = spawn_pos
	# Initialize net sync targets so remote interpolation starts at the correct position
	player._net_position = spawn_pos
	player._net_rotation_y = player.rotation.y
	_spawned_players[peer_id] = player

	# Feed local player to shared HUD and connect death signal
	if peer_id == NetworkManager.get_my_id():
		if _shared_hud:
			_shared_hud.set_local_player(player, class_name_str, peer_id)
		if player.has_signal("died"):
			player.died.connect(_on_local_player_died)

	# Add overhead name for remote players in hub
	if state == GameState.HUB and peer_id != NetworkManager.get_my_id():
		_update_overhead_name(player, peer_id)


func _despawn_all_players() -> void:
	for pid in _spawned_players:
		var player = _spawned_players[pid]
		if is_instance_valid(player):
			player.queue_free()
	_spawned_players.clear()
	_despawn_all_projectiles()
	if _shared_hud:
		_shared_hud.clear_local_player()


func _despawn_all_projectiles() -> void:
	for pid in _spawned_projectiles:
		var proj = _spawned_projectiles[pid]
		if is_instance_valid(proj):
			proj.queue_free()
	_spawned_projectiles.clear()


func _spawn_projectile(proj_id: int, pos: Vector3, dir: Vector3) -> void:
	var scene := load("res://scenes/enemies/basic_enemy/enemy_projectile.tscn") as PackedScene
	if not scene:
		return
	var proj := scene.instantiate() as Node3D
	proj.name = "Proj_%d" % proj_id
	_projectiles_node.add_child(proj)
	proj.global_position = pos
	if proj.has_method("setup"):
		proj.setup(dir, 0.0)
	_spawned_projectiles[proj_id] = proj


# =============================================================================
# Fight
# =============================================================================

func _start_fight() -> void:
	state = GameState.FIGHT
	_lobby_layer.visible = false
	_hub_layer.visible = false
	_cursor_toggled = false
	_alt_held = false

	if _gate:
		_gate.visible = true
		_gate.use_collision = true

	if _enemy_node:
		_enemy_node.visible = true
		_enemy_node.collision_layer = 4
		_enemy_node.set_physics_process(true)
	CombatLog.start_fight()
	if _shared_hud:
		_shared_hud.on_fight_start()


func _on_boss_dead() -> void:
	state = GameState.FIGHT_OVER
	if _gate:
		_gate.visible = false
		_gate.use_collision = false
	_spawn_exit_portal()
	if _local_player_dead and _death_overlay_layer.visible:
		_respawn_btn.disabled = false
	CombatLog.end_fight("VICTORY")
	if _shared_hud:
		_shared_hud.on_fight_end()


func _on_all_dead() -> void:
	state = GameState.FIGHT_OVER
	if _gate:
		_gate.visible = false
		_gate.use_collision = false
	if _local_player_dead and _death_overlay_layer.visible:
		_respawn_btn.disabled = false
	CombatLog.end_fight("WIPE")
	if _shared_hud:
		_shared_hud.on_fight_end()


# =============================================================================
# Zone transfer
# =============================================================================

func _on_zone_transfer(zone_type: int, new_peer_id: int) -> void:
	print("[Main] Zone transfer: type=%d, new_peer=%d" % [zone_type, new_peer_id])
	_hide_death_overlay()
	_remove_exit_portal()
	_despawn_all_players()

	if zone_type == NetSerializer.ZONE_TYPE_ARENA:
		_unload_environment()
		_load_environment(ARENA_SCENE)
		_create_gate()
		_create_enemy()
		_enter_arena_lobby()
	else:
		_unload_environment()
		_load_environment(HUB_SCENE)
		_enter_hub()


# =============================================================================
# Server-authoritative signal handlers
# =============================================================================

func _on_game_flow_event(flow_type: int, text: String) -> void:
	match flow_type:
		NetSerializer.FLOW_SPAWN_PLAYERS:
			_spawn_multiplayer_players()
		NetSerializer.FLOW_FIGHT_START:
			_start_fight()
		NetSerializer.FLOW_BOSS_DEAD:
			_on_boss_dead()
		NetSerializer.FLOW_ALL_DEAD:
			_on_all_dead()
		NetSerializer.FLOW_RETURN_LOBBY:
			_hide_death_overlay()
			_enter_arena_lobby()


func _on_world_state(data: Dictionary) -> void:
	if state == GameState.MENU:
		return

	var players_data: Array = data.get("players", [])
	var seen_peers: Dictionary = {}
	var my_id := NetworkManager.get_my_id()

	for p_data in players_data:
		var pid: int = p_data["peer_id"]
		seen_peers[pid] = true

		# Store player names
		var uname: String = p_data.get("username", "")
		if uname != "":
			_player_names[pid] = uname

		# Spawn remote player if needed
		if pid != my_id and pid not in _spawned_players:
			var cls: String = p_data.get("class_name", "gunner")
			_spawn_player(pid, cls, p_data["pos"])

		if pid not in _spawned_players:
			continue

		var player: CharacterBody3D = _spawned_players[pid]
		if not is_instance_valid(player):
			continue

		if pid == my_id and state == GameState.HUB:
			# Hub: client-authoritative movement, only snap on extreme desync
			var server_pos: Vector3 = p_data["pos"]
			if player.global_position.distance_to(server_pos) > 8.0:
				player.global_position = server_pos
		elif player.has_method("apply_server_state"):
			player.apply_server_state(p_data)

		# Update overhead name for remote players in hub
		if state == GameState.HUB and pid != my_id:
			_update_overhead_name(player, pid)

	# Despawn players no longer in state
	var to_remove: Array = []
	for pid in _spawned_players:
		if pid not in seen_peers:
			to_remove.append(pid)
	for pid in to_remove:
		var player = _spawned_players[pid]
		if is_instance_valid(player):
			player.queue_free()
		_spawned_players.erase(pid)

	# Enemy (inert in hub — dead at origin, _enemy_node is null in hub)
	var enemy_data: Dictionary = data.get("enemy", {})
	if not enemy_data.is_empty() and is_instance_valid(_enemy_node) and _enemy_node.has_method("apply_server_state"):
		_enemy_node.apply_server_state(enemy_data)

	# Projectiles: spawn/update/remove
	var stale: Array = []
	for pid in _spawned_projectiles:
		if not is_instance_valid(_spawned_projectiles[pid]):
			stale.append(pid)
	for pid in stale:
		_spawned_projectiles.erase(pid)

	var proj_data: Array = data.get("projectiles", [])
	var active_ids: Dictionary = {}
	for p in proj_data:
		var pid: int = p["proj_id"]
		active_ids[pid] = true
		if pid not in _spawned_projectiles:
			_spawn_projectile(pid, p["pos"], p["direction"])
		else:
			_spawned_projectiles[pid].global_position = p["pos"]
	var proj_to_remove: Array = []
	for pid in _spawned_projectiles:
		if pid not in active_ids:
			proj_to_remove.append(pid)
	for pid in proj_to_remove:
		var proj = _spawned_projectiles[pid]
		if is_instance_valid(proj):
			proj.queue_free()
		_spawned_projectiles.erase(pid)

	# Feed shared HUD
	if _shared_hud:
		_shared_hud.update_world_state(data)


func _on_damage_event(data: Dictionary) -> void:
	var target_peer: int = data.get("target_peer_id", -1)
	var source_peer: int = data.get("source_peer_id", 0)
	var amount: float = data.get("amount", 0.0)
	var hit_pos: Vector3 = data.get("hit_pos", Vector3.ZERO)
	var source_type: int = data.get("source_type", 0)
	if target_peer == 0:
		# Player hit the enemy
		if is_instance_valid(_enemy_node) and _enemy_node.has_method("on_damage_visual"):
			_enemy_node.on_damage_visual(amount, hit_pos)
		# Server-confirmed hit marker on the attacker's HUD
		if source_peer == NetworkManager.get_my_id():
			var local_player: CharacterBody3D = _spawned_players.get(source_peer)
			if is_instance_valid(local_player) and local_player.has_method("on_hit_confirmed"):
				local_player.on_hit_confirmed(amount)
		# Show tracer from remote gunner to hit point
		if source_peer != NetworkManager.get_my_id() and source_peer in _spawned_players:
			var source_player: CharacterBody3D = _spawned_players[source_peer]
			if is_instance_valid(source_player) and source_player.has_method("on_hit_tracer"):
				source_player.on_hit_tracer(hit_pos)
		# Floating damage number
		_spawn_damage_number(amount, hit_pos)
	elif target_peer in _spawned_players:
		var player: CharacterBody3D = _spawned_players[target_peer]
		if is_instance_valid(player) and player.has_method("on_damage_visual"):
			player.on_damage_visual(amount, hit_pos)

	# Feed shared HUD damage meter
	if _shared_hud:
		_shared_hud.on_damage_event(data)


func _spawn_damage_number(amount: float, world_pos: Vector3) -> void:
	var label := Label3D.new()
	label.text = str(int(amount))
	label.font_size = 48
	label.outline_size = 8
	label.modulate = Color(1.0, 0.95, 0.3, 1.0)
	label.outline_modulate = Color(0.0, 0.0, 0.0, 0.8)
	label.billboard = BaseMaterial3D.BILLBOARD_ENABLED
	label.no_depth_test = true
	label.pixel_size = 0.005
	# Slight random offset so stacked hits don't overlap
	var offset := Vector3(randf_range(-0.3, 0.3), randf_range(0.0, 0.3), randf_range(-0.3, 0.3))
	label.position = world_pos + offset + Vector3(0.0, 0.5, 0.0)
	add_child(label)

	var tween := create_tween()
	tween.set_parallel(true)
	tween.tween_property(label, "position:y", label.position.y + 1.5, 0.8).set_ease(Tween.EASE_OUT).set_trans(Tween.TRANS_QUAD)
	tween.tween_property(label, "modulate:a", 0.0, 0.8).set_delay(0.3)
	tween.tween_property(label, "outline_modulate:a", 0.0, 0.8).set_delay(0.3)
	tween.chain().tween_callback(label.queue_free)


func _toggle_pause() -> void:
	paused = not paused
	get_tree().paused = paused
	_pause_layer.visible = paused
	_cursor_toggled = false
	_alt_held = false
	if paused:
		Input.set_mouse_mode(Input.MOUSE_MODE_VISIBLE)
	else:
		Input.set_mouse_mode(Input.MOUSE_MODE_CAPTURED)


## Show/hide cursor for UI interaction without pausing.
## Active when Alt is held or backtick is toggled on.
func _update_cursor_mode() -> void:
	var want_cursor := _cursor_toggled or _alt_held
	if want_cursor:
		Input.set_mouse_mode(Input.MOUSE_MODE_VISIBLE)
	else:
		Input.set_mouse_mode(Input.MOUSE_MODE_CAPTURED)


# =============================================================================
# Scene management
# =============================================================================

func _load_environment(scene_path: String) -> void:
	var scene := load(scene_path) as PackedScene
	_current_env = scene.instantiate()
	add_child(_current_env)
	print("[Main] Loaded environment: %s" % scene_path)


func _unload_environment() -> void:
	if _current_env and is_instance_valid(_current_env):
		_current_env.queue_free()
		_current_env = null
	if _enemy_node and is_instance_valid(_enemy_node):
		_enemy_node.queue_free()
	_enemy_node = null
	if _gate and is_instance_valid(_gate):
		_gate.queue_free()
	_gate = null


func _create_gate() -> void:
	_gate = CSGBox3D.new()
	_gate.size = Vector3(5.0, 5.0, 0.5)
	_gate.transform.origin = Vector3(0.0, 2.5, 15.0)
	_gate.use_collision = true
	_gate.collision_layer = 1
	_gate.collision_mask = 0
	var mat := StandardMaterial3D.new()
	mat.albedo_color = Color(0.5, 0.2, 0.2)
	_gate.material = mat
	_gate.visible = false
	_gate.use_collision = false
	add_child(_gate)


func _create_enemy() -> void:
	var enemy_scene := load("res://scenes/enemies/basic_enemy/basic_enemy.tscn") as PackedScene
	if enemy_scene:
		_enemy_node = enemy_scene.instantiate() as CharacterBody3D
		_enemy_node.name = "BasicEnemy"
		_enemy_node.global_position = ENEMY_SPAWN
		_enemy_node.visible = false
		_enemy_node.collision_layer = 0
		_enemy_node.set_physics_process(false)
		add_child(_enemy_node)


# =============================================================================
# UI builders
# =============================================================================

func _create_pause_menu() -> void:
	_pause_layer = CanvasLayer.new()
	_pause_layer.layer = 20
	_pause_layer.process_mode = Node.PROCESS_MODE_ALWAYS
	_pause_layer.visible = false
	add_child(_pause_layer)

	var bg := ColorRect.new()
	bg.color = Color(0.0, 0.0, 0.0, 0.6)
	bg.anchor_right = 1.0
	bg.anchor_bottom = 1.0
	bg.mouse_filter = Control.MOUSE_FILTER_STOP
	_pause_layer.add_child(bg)

	var vbox := VBoxContainer.new()
	vbox.anchor_left = 0.5
	vbox.anchor_right = 0.5
	vbox.anchor_top = 0.35
	vbox.anchor_bottom = 0.65
	vbox.offset_left = -120.0
	vbox.offset_right = 120.0
	vbox.add_theme_constant_override("separation", 16)
	_pause_layer.add_child(vbox)

	var title := Label.new()
	title.text = "PAUSED"
	title.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	title.add_theme_font_size_override("font_size", 36)
	vbox.add_child(title)

	var resume_btn := Button.new()
	resume_btn.text = "Resume"
	resume_btn.pressed.connect(_toggle_pause)
	vbox.add_child(resume_btn)

	var menu_btn := Button.new()
	menu_btn.text = "Back to Menu"
	menu_btn.pressed.connect(func():
		get_tree().paused = false
		paused = false
		_despawn_all_players()
		_enter_menu()
	)
	vbox.add_child(menu_btn)

	var quit_btn := Button.new()
	quit_btn.text = "Quit"
	quit_btn.pressed.connect(func(): get_tree().quit())
	vbox.add_child(quit_btn)


func _create_menu_ui() -> void:
	_menu_layer = CanvasLayer.new()
	_menu_layer.layer = 18
	_menu_layer.visible = false
	add_child(_menu_layer)

	var bg := ColorRect.new()
	bg.color = Color(0.05, 0.05, 0.1, 0.95)
	bg.anchor_right = 1.0
	bg.anchor_bottom = 1.0
	bg.mouse_filter = Control.MOUSE_FILTER_STOP
	_menu_layer.add_child(bg)

	var vbox := VBoxContainer.new()
	vbox.anchor_left = 0.5
	vbox.anchor_right = 0.5
	vbox.anchor_top = 0.25
	vbox.anchor_bottom = 0.75
	vbox.offset_left = -150.0
	vbox.offset_right = 150.0
	vbox.add_theme_constant_override("separation", 20)
	_menu_layer.add_child(vbox)

	var title := Label.new()
	title.text = "CODEX ONLINE"
	title.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	title.add_theme_font_size_override("font_size", 48)
	title.add_theme_color_override("font_color", Color(0.8, 0.85, 1.0))
	vbox.add_child(title)

	var subtitle := Label.new()
	subtitle.text = "Phase 0 -- Server Authoritative"
	subtitle.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	subtitle.add_theme_font_size_override("font_size", 18)
	subtitle.add_theme_color_override("font_color", Color(0.5, 0.5, 0.6))
	vbox.add_child(subtitle)

	var spacer := Control.new()
	spacer.custom_minimum_size.y = 10.0
	vbox.add_child(spacer)

	# Username input
	_username_input = LineEdit.new()
	_username_input.placeholder_text = "Enter username..."
	_username_input.custom_minimum_size.y = 50.0
	_username_input.max_length = 20
	vbox.add_child(_username_input)

	var spacer2 := Control.new()
	spacer2.custom_minimum_size.y = 10.0
	vbox.add_child(spacer2)

	var connect_hbox := HBoxContainer.new()
	connect_hbox.add_theme_constant_override("separation", 8)
	vbox.add_child(connect_hbox)

	_address_input = LineEdit.new()
	_address_input.placeholder_text = "109.222.207.243"
	_address_input.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	_address_input.custom_minimum_size.y = 50.0
	connect_hbox.add_child(_address_input)

	var connect_btn := Button.new()
	connect_btn.text = "Connect"
	connect_btn.custom_minimum_size = Vector2(100.0, 50.0)
	connect_btn.pressed.connect(_on_connect_pressed)
	connect_hbox.add_child(connect_btn)


func _create_lobby_ui() -> void:
	_lobby_layer = CanvasLayer.new()
	_lobby_layer.layer = 15
	_lobby_layer.visible = false
	add_child(_lobby_layer)

	var bg := ColorRect.new()
	bg.color = Color(0.05, 0.05, 0.1, 0.85)
	bg.anchor_right = 1.0
	bg.anchor_bottom = 1.0
	bg.mouse_filter = Control.MOUSE_FILTER_IGNORE
	_lobby_layer.add_child(bg)

	var vbox := VBoxContainer.new()
	vbox.anchor_left = 0.5
	vbox.anchor_right = 0.5
	vbox.anchor_top = 0.15
	vbox.anchor_bottom = 0.85
	vbox.offset_left = -200.0
	vbox.offset_right = 200.0
	vbox.add_theme_constant_override("separation", 16)
	_lobby_layer.add_child(vbox)

	var title := Label.new()
	title.text = "ARENA LOBBY"
	title.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	title.add_theme_font_size_override("font_size", 36)
	vbox.add_child(title)

	_lobby_class_label = Label.new()
	_lobby_class_label.text = "[1] Gunner   [2] Vanguard   [3] Blade Dancer"
	_lobby_class_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	_lobby_class_label.add_theme_font_size_override("font_size", 20)
	_lobby_class_label.add_theme_color_override("font_color", Color(0.8, 0.8, 0.9))
	vbox.add_child(_lobby_class_label)

	var spacer := Control.new()
	spacer.custom_minimum_size.y = 20.0
	vbox.add_child(spacer)

	_lobby_players_label = Label.new()
	_lobby_players_label.text = ""
	_lobby_players_label.add_theme_font_size_override("font_size", 18)
	_lobby_players_label.add_theme_color_override("font_color", Color(0.7, 0.8, 0.7))
	vbox.add_child(_lobby_players_label)

	var spacer2 := Control.new()
	spacer2.custom_minimum_size.y = 20.0
	vbox.add_child(spacer2)

	_lobby_status_label = Label.new()
	_lobby_status_label.text = "Press ENTER when ready"
	_lobby_status_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	_lobby_status_label.add_theme_font_size_override("font_size", 22)
	_lobby_status_label.add_theme_color_override("font_color", Color(0.9, 0.9, 0.5))
	vbox.add_child(_lobby_status_label)


func _create_hub_ui() -> void:
	_hub_layer = CanvasLayer.new()
	_hub_layer.layer = 14
	_hub_layer.visible = false
	add_child(_hub_layer)

	# Class selection (top-center)
	_hub_class_label = Label.new()
	_hub_class_label.text = "[1] Gunner  [2] Vanguard  [3] Blade Dancer"
	_hub_class_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	_hub_class_label.anchor_left = 0.0
	_hub_class_label.anchor_right = 1.0
	_hub_class_label.anchor_top = 0.0
	_hub_class_label.anchor_bottom = 0.0
	_hub_class_label.offset_top = 10.0
	_hub_class_label.offset_bottom = 60.0
	_hub_class_label.add_theme_font_size_override("font_size", 18)
	_hub_class_label.add_theme_color_override("font_color", Color(0.8, 0.8, 0.9))
	_hub_layer.add_child(_hub_class_label)

	# Status (top-left)
	_hub_status_label = Label.new()
	_hub_status_label.text = "Hub - Walk to the portal to enter the arena"
	_hub_status_label.anchor_left = 0.0
	_hub_status_label.anchor_right = 1.0
	_hub_status_label.anchor_top = 0.0
	_hub_status_label.anchor_bottom = 0.0
	_hub_status_label.offset_top = 65.0
	_hub_status_label.offset_bottom = 90.0
	_hub_status_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	_hub_status_label.add_theme_font_size_override("font_size", 14)
	_hub_status_label.add_theme_color_override("font_color", Color(0.6, 0.6, 0.7))
	_hub_layer.add_child(_hub_status_label)

	# Portal prompt (center-bottom)
	_portal_prompt = Label.new()
	_portal_prompt.text = "Press [E] to enter Arena"
	_portal_prompt.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	_portal_prompt.anchor_left = 0.0
	_portal_prompt.anchor_right = 1.0
	_portal_prompt.anchor_top = 0.7
	_portal_prompt.anchor_bottom = 0.75
	_portal_prompt.add_theme_font_size_override("font_size", 28)
	_portal_prompt.add_theme_color_override("font_color", Color(0.3, 0.6, 1.0))
	_portal_prompt.visible = false
	_hub_layer.add_child(_portal_prompt)


func _create_group_panel() -> void:
	_group_panel = PanelContainer.new()
	_group_panel.anchor_left = 0.0
	_group_panel.anchor_right = 0.0
	_group_panel.anchor_top = 0.0
	_group_panel.anchor_bottom = 0.0
	_group_panel.offset_left = 10.0
	_group_panel.offset_top = 80.0
	_group_panel.offset_right = 220.0
	_group_panel.offset_bottom = 250.0
	_group_panel.visible = false
	_hub_layer.add_child(_group_panel)

	var vbox := VBoxContainer.new()
	vbox.add_theme_constant_override("separation", 8)
	_group_panel.add_child(vbox)

	_group_label = Label.new()
	_group_label.text = "No group\n[G] Create group"
	_group_label.add_theme_font_size_override("font_size", 14)
	_group_label.add_theme_color_override("font_color", Color(0.8, 0.9, 0.8))
	vbox.add_child(_group_label)

	_group_leave_btn = Button.new()
	_group_leave_btn.text = "Leave Group [G]"
	_group_leave_btn.visible = false
	_group_leave_btn.pressed.connect(func(): NetworkManager.send_group_leave())
	vbox.add_child(_group_leave_btn)


func _create_invite_popup() -> void:
	_invite_popup = CanvasLayer.new()
	_invite_popup.layer = 21
	_invite_popup.visible = false
	add_child(_invite_popup)

	var bg := ColorRect.new()
	bg.color = Color(0.0, 0.0, 0.0, 0.6)
	bg.anchor_right = 1.0
	bg.anchor_bottom = 1.0
	bg.mouse_filter = Control.MOUSE_FILTER_STOP
	_invite_popup.add_child(bg)

	var panel := PanelContainer.new()
	panel.anchor_left = 0.5
	panel.anchor_right = 0.5
	panel.anchor_top = 0.35
	panel.anchor_bottom = 0.5
	panel.offset_left = -180.0
	panel.offset_right = 180.0
	_invite_popup.add_child(panel)

	var vbox := VBoxContainer.new()
	vbox.add_theme_constant_override("separation", 16)
	panel.add_child(vbox)

	_invite_label = Label.new()
	_invite_label.text = "Group invitation"
	_invite_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	_invite_label.add_theme_font_size_override("font_size", 20)
	vbox.add_child(_invite_label)

	var hbox := HBoxContainer.new()
	hbox.add_theme_constant_override("separation", 16)
	hbox.alignment = BoxContainer.ALIGNMENT_CENTER
	vbox.add_child(hbox)

	var accept_btn := Button.new()
	accept_btn.text = "Accept"
	accept_btn.custom_minimum_size = Vector2(100, 40)
	accept_btn.pressed.connect(_accept_invite)
	hbox.add_child(accept_btn)

	var decline_btn := Button.new()
	decline_btn.text = "Decline"
	decline_btn.custom_minimum_size = Vector2(100, 40)
	decline_btn.pressed.connect(_decline_invite)
	hbox.add_child(decline_btn)


func _create_shared_hud() -> void:
	_shared_hud_layer = CanvasLayer.new()
	_shared_hud_layer.layer = 9  # below class HUDs (10), below damage overlay
	add_child(_shared_hud_layer)

	_shared_hud = preload("res://scenes/shared/hud/shared_hud.gd").new()
	_shared_hud.name = "SharedHUD"
	_shared_hud.anchor_right = 1.0
	_shared_hud.anchor_bottom = 1.0
	_shared_hud.mouse_filter = Control.MOUSE_FILTER_IGNORE
	_shared_hud_layer.add_child(_shared_hud)
	_shared_hud.set_player_names(_player_names)


# =============================================================================
# Death overlay
# =============================================================================

func _create_death_overlay() -> void:
	_death_overlay_layer = CanvasLayer.new()
	_death_overlay_layer.layer = 12
	_death_overlay_layer.process_mode = Node.PROCESS_MODE_ALWAYS
	_death_overlay_layer.visible = false
	add_child(_death_overlay_layer)

	_death_overlay_bg = ColorRect.new()
	_death_overlay_bg.color = Color(0.0, 0.0, 0.0, 0.5)
	_death_overlay_bg.anchor_right = 1.0
	_death_overlay_bg.anchor_bottom = 1.0
	_death_overlay_bg.mouse_filter = Control.MOUSE_FILTER_STOP
	_death_overlay_layer.add_child(_death_overlay_bg)

	_death_label = Label.new()
	_death_label.text = "YOU DIED"
	_death_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	_death_label.vertical_alignment = VERTICAL_ALIGNMENT_CENTER
	_death_label.anchor_left = 0.0
	_death_label.anchor_right = 1.0
	_death_label.anchor_top = 0.3
	_death_label.anchor_bottom = 0.45
	_death_label.add_theme_font_size_override("font_size", 64)
	_death_label.add_theme_color_override("font_color", Color(0.8, 0.1, 0.1))
	_death_overlay_layer.add_child(_death_label)

	var btn_container := VBoxContainer.new()
	btn_container.anchor_left = 0.5
	btn_container.anchor_right = 0.5
	btn_container.anchor_top = 0.55
	btn_container.anchor_bottom = 0.7
	btn_container.offset_left = -120.0
	btn_container.offset_right = 120.0
	btn_container.add_theme_constant_override("separation", 12)
	_death_overlay_layer.add_child(btn_container)

	_respawn_btn = Button.new()
	_respawn_btn.text = "Respawn"
	_respawn_btn.custom_minimum_size.y = 45.0
	_respawn_btn.disabled = true
	_respawn_btn.pressed.connect(_on_respawn)
	btn_container.add_child(_respawn_btn)

	_respawn_hub_btn = Button.new()
	_respawn_hub_btn.text = "Return to Hub"
	_respawn_hub_btn.custom_minimum_size.y = 45.0
	_respawn_hub_btn.pressed.connect(_on_respawn_hub)
	btn_container.add_child(_respawn_hub_btn)


func _on_local_player_died() -> void:
	_local_player_dead = true
	_death_overlay_layer.visible = true
	_respawn_btn.disabled = (state == GameState.FIGHT)
	Input.set_mouse_mode(Input.MOUSE_MODE_VISIBLE)


func _on_respawn() -> void:
	NetworkManager.send_respawn_request(0)
	_hide_death_overlay()
	Input.set_mouse_mode(Input.MOUSE_MODE_CAPTURED)


func _on_respawn_hub() -> void:
	NetworkManager.send_respawn_request(1)
	_hide_death_overlay()


func _hide_death_overlay() -> void:
	_local_player_dead = false
	_death_overlay_layer.visible = false
	_respawn_btn.disabled = true


# =============================================================================
# Exit portal
# =============================================================================

func _spawn_exit_portal() -> void:
	if _exit_portal:
		return
	_exit_portal = CSGCylinder3D.new()
	_exit_portal.radius = 1.5
	_exit_portal.height = 0.1
	_exit_portal.transform.origin = EXIT_PORTAL_POS
	var mat := StandardMaterial3D.new()
	mat.albedo_color = Color(0.2, 0.5, 1.0, 0.7)
	mat.transparency = BaseMaterial3D.TRANSPARENCY_ALPHA
	mat.emission_enabled = true
	mat.emission = Color(0.3, 0.6, 1.0)
	mat.emission_energy_multiplier = 2.0
	_exit_portal.material = mat
	if _current_env:
		_current_env.add_child(_exit_portal)
	else:
		add_child(_exit_portal)

	# Create prompt label
	_exit_portal_prompt = Label.new()
	_exit_portal_prompt.text = "[E] Return to Hub"
	_exit_portal_prompt.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	_exit_portal_prompt.anchor_left = 0.3
	_exit_portal_prompt.anchor_right = 0.7
	_exit_portal_prompt.anchor_top = 0.6
	_exit_portal_prompt.anchor_bottom = 0.65
	_exit_portal_prompt.add_theme_font_size_override("font_size", 20)
	_exit_portal_prompt.visible = false
	if _shared_hud_layer:
		_shared_hud_layer.add_child(_exit_portal_prompt)


func _remove_exit_portal() -> void:
	if _exit_portal and is_instance_valid(_exit_portal):
		_exit_portal.queue_free()
	_exit_portal = null
	if _exit_portal_prompt and is_instance_valid(_exit_portal_prompt):
		_exit_portal_prompt.queue_free()
	_exit_portal_prompt = null
	_near_exit_portal = false


func _check_exit_portal_proximity() -> void:
	if not _exit_portal or not is_instance_valid(_exit_portal):
		_near_exit_portal = false
		if _exit_portal_prompt and is_instance_valid(_exit_portal_prompt):
			_exit_portal_prompt.visible = false
		return
	var my_id := NetworkManager.get_my_id()
	if my_id not in _spawned_players:
		_near_exit_portal = false
		return
	var player: CharacterBody3D = _spawned_players[my_id]
	if not is_instance_valid(player):
		return
	var dist := player.global_position.distance_to(EXIT_PORTAL_POS)
	_near_exit_portal = dist < 3.0
	if _exit_portal_prompt and is_instance_valid(_exit_portal_prompt):
		_exit_portal_prompt.visible = _near_exit_portal


# =============================================================================
# Navigation
# =============================================================================

func _bake_hub_navigation() -> void:
	var nav_region := NavigationRegion3D.new()
	var nav_mesh := NavigationMesh.new()
	nav_mesh.agent_radius = 0.8
	nav_mesh.agent_height = 2.0
	nav_mesh.cell_size = 0.25
	nav_mesh.cell_height = 0.25
	nav_region.navigation_mesh = nav_mesh
	add_child(nav_region)

	var source_geo := NavigationMeshSourceGeometryData3D.new()
	# Hub floor: 30x25 centered at (0, 0, 2.5)
	source_geo.add_faces(PackedVector3Array([
		Vector3(-15, 0, -10), Vector3(15, 0, -10), Vector3(15, 0, 15),
		Vector3(-15, 0, -10), Vector3(15, 0, 15), Vector3(-15, 0, 15),
	]), Transform3D.IDENTITY)

	NavigationServer3D.bake_from_source_geometry_data(nav_mesh, source_geo)
