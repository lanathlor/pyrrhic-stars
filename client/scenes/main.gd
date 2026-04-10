extends Node3D

## Game flow: Menu -> Hub -> Arena (Lobby -> Fight -> Fight Over) -> Hub.
## Server-authoritative: all game flow is driven by server events.

enum GameState { MENU, CHARACTER_SELECT, CREATE_CHARACTER, HUB, ARENA_LOBBY, FIGHT, FIGHT_OVER }

var state: GameState = GameState.MENU
var paused: bool = false

const HUB_SCENE := "res://scenes/environments/prime_hub/military_building.tscn"
const ARENA_SCENE := "res://scenes/environments/arena/arena.tscn"
const EXIT_PORTAL_POS := Vector3(0.0, 0.1, 0.0)

const LOBBY_SPAWN := Vector3(0.0, 0.1, 48.0)
const PLAYER_SPAWNS := [
	Vector3(-2.0, 0.1, 48.0),
	Vector3(0.0, 0.1, 48.0),
	Vector3(2.0, 0.1, 48.0),
	Vector3(-1.0, 0.1, 49.0),
	Vector3(1.0, 0.1, 49.0),
]
const ARENA_ENTRY_Z := 40.0
const BOSS_ROOM_ENTRY_Z := 12.0
const HUB_SPAWNS := [
	Vector3(14.0, -199.9, -80.0),
	Vector3(14.0, -199.9, -78.0),
	Vector3(14.0, -199.9, -82.0),
	Vector3(12.5, -199.9, -79.0),
	Vector3(12.5, -199.9, -81.0),
]
const HUB_SPAWN_YAW := PI / 2.0  # face west

const CLASS_SCENES := {
	"gunner": "res://scenes/controllers/gunner/gunner.tscn",
	"vanguard": "res://scenes/controllers/vanguard/vanguard.tscn",
	"blade_dancer": "res://scenes/controllers/blade_dancer/blade_dancer.tscn",
}

const CLASS_INFO := {
	"gunner": {"name": "Gunner", "genre": "FPS", "desc": "Fast movement, high fire rate.\nRelentless aggression."},
	"vanguard": {"name": "Vanguard", "genre": "Souls-like", "desc": "Big AoE swings, punish windows.\nHeavy and deliberate."},
	"blade_dancer": {"name": "Blade Dancer", "genre": "State Machine", "desc": "5 configurations, 4 spells each.\nHighest skill ceiling."},
}

var _spawned_players: Dictionary = {}  # peer_id -> CharacterBody3D
var _spawned_projectiles: Dictionary = {}  # proj_id -> Node3D
var _local_class: String = "gunner"
var _saved_hub_position: Vector3 = Vector3.ZERO
var _saved_hub_rot_y: float = 0.0
var _has_saved_state: bool = false
var _class_button_group: ButtonGroup = ButtonGroup.new()
var _players_node: Node3D
var _projectiles_node: Node3D
var _username: String = ""
const SERVER_ADDRESS := "109.222.207.243"
const USERNAME_SAVE_PATH := "user://username.txt"

# Scene management
var _current_env: Node3D = null
var _enemy_nodes: Dictionary = {}  # enemy_id -> CharacterBody3D
var _npc_nodes: Dictionary = {}  # npc_id -> Node3D

# Dynamic nodes
var _boss_gate: CSGBox3D
var _atmosphere: Node3D
var _arena_buildings: Node3D
var _pause_layer: CanvasLayer
var _menu_layer: CanvasLayer
var _username_input: LineEdit
var _menu_welcome_label: Label

# Character select UI
var _char_select_layer: CanvasLayer
var _char_list_container: VBoxContainer  # holds character row buttons
var _char_list_data: Dictionary = {}
var _selected_char_id: int = 0
var _enter_world_btn: Button
var _char_select_welcome: Label
var _account_username: String = ""

# Create character UI
var _char_create_layer: CanvasLayer
var _char_create_cards: Dictionary = {}  # class_name -> PanelContainer
var _char_name_input: LineEdit
var _char_create_error_label: Label
var _char_create_btn: Button

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
var _near_lift: bool = false
var _lift_prompt: Label
var _player_names: Dictionary = {}  # peer_id -> username
var _group_data: Dictionary = {}  # current group state
var _aimed_peer_id: int = 0  # peer id under crosshair for invite
var _cursor_toggled: bool = false  # backtick toggle state
var _alt_held: bool = false        # alt hold state

# Shared HUD (boss frame, group frames, damage meter, minimap, player status)
var _shared_hud_layer: CanvasLayer
var _shared_hud: Control
var _map_overlay: Control

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

# Portal trail (hub guide)
var _portal_trail: Node3D


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
	_create_char_select_ui()
	_create_char_create_ui()
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
	NetworkManager.world_state_received.connect(_on_world_state)
	NetworkManager.damage_event_received.connect(_on_damage_event)
	NetworkManager.zone_transfer_received.connect(_on_zone_transfer)
	NetworkManager.group_state_updated.connect(_on_group_state)
	NetworkManager.group_invite_received.connect(_on_group_invite)
	NetworkManager.group_error_received.connect(_on_group_error)
	NetworkManager.character_state_received.connect(_on_character_state)
	NetworkManager.character_list_received.connect(_on_character_list)
	NetworkManager.character_error_received.connect(_on_character_error)

	_enter_menu()


func _input(event: InputEvent) -> void:
	if event.is_action_pressed("ui_cancel"):
		if state == GameState.FIGHT_OVER:
			return
		if state == GameState.MENU or state == GameState.CHARACTER_SELECT or state == GameState.CREATE_CHARACTER:
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

	# Full map toggle
	if not paused and state != GameState.MENU:
		if event is InputEventKey and event.pressed and not event.echo:
			if event.physical_keycode == KEY_M:
				if _map_overlay:
					var my_id := NetworkManager.get_my_id()
					if my_id in _spawned_players and is_instance_valid(_spawned_players[my_id]):
						var player: CharacterBody3D = _spawned_players[my_id]
						_map_overlay._player_pos = player.global_position
						_map_overlay._player_rot_y = player.rotation.y
					_map_overlay.toggle()
					if _map_overlay.visible:
						# Scan collision geometry from the live scene
						_map_overlay.scan_environment(_current_env)
						_map_overlay._recompute_scale()
						if _portal_trail:
							if my_id in _spawned_players and is_instance_valid(_spawned_players[my_id]):
								_map_overlay.set_waypoint_path(
									_portal_trail.get_path_to_target(
										_spawned_players[my_id].global_position))
					get_viewport().set_input_as_handled()

	# Hub interactions
	if state == GameState.HUB and not paused:
		if event is InputEventKey and event.pressed:
			if event.physical_keycode == KEY_E:
				if _near_lift:
					_interact_lift()
				elif _near_portal:
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
		_check_lift_proximity()
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
	_hub_layer.visible = false
	_char_select_layer.visible = false
	_char_create_layer.visible = false
	_pause_layer.visible = false
	Input.set_mouse_mode(Input.MOUSE_MODE_VISIBLE)
	_unload_environment()
	if _enter_world_btn:
		_enter_world_btn.disabled = false
	# Show welcome or username input depending on saved state.
	var saved := _load_saved_username()
	if saved != "":
		_username = saved
		_username_input.visible = false
		_menu_welcome_label.text = "Welcome back, %s" % saved
		_menu_welcome_label.visible = true
	else:
		_username_input.visible = true
		_menu_welcome_label.visible = false


func _on_connect_pressed() -> void:
	# If no saved username, require input.
	if _username == "":
		_username = _username_input.text.strip_edges()
		if _username == "":
			_username_input.grab_focus()
			return
		_save_username(_username)

	NetworkManager.username = _username
	NetworkManager.disconnect_game()
	var err := NetworkManager.connect_to_server(SERVER_ADDRESS)
	if err != OK:
		print("[Main] Failed to connect: %s" % error_string(err))
		return
	print("[Main] Connecting to %s:%d..." % [SERVER_ADDRESS, NetworkManager.DEFAULT_PORT])
	_menu_layer.visible = false


func _load_saved_username() -> String:
	if not FileAccess.file_exists(USERNAME_SAVE_PATH):
		return ""
	var f := FileAccess.open(USERNAME_SAVE_PATH, FileAccess.READ)
	if f == null:
		return ""
	var name := f.get_as_text().strip_edges()
	f.close()
	return name


func _save_username(name: String) -> void:
	var f := FileAccess.open(USERNAME_SAVE_PATH, FileAccess.WRITE)
	if f == null:
		return
	f.store_string(name)
	f.close()


func _on_character_state(data: Dictionary) -> void:
	# Server confirmed character selection. Restore position and enter hub.
	_selected_char_id = data.get("char_id", 0)
	if data.class_name != "":
		_local_class = data.class_name
	if data.position != Vector3.ZERO:
		_saved_hub_position = data.position
		_saved_hub_rot_y = data.rot_y
		_has_saved_state = true
	print("[Main] Character confirmed: id=%d class=%s name=%s pos=%s" % [
		_selected_char_id, _local_class, data.get("char_name", ""), _saved_hub_position])


func _on_character_list(data: Dictionary) -> void:
	_char_list_data = data
	_account_username = data.get("username", "")
	var last_id: int = data.get("last_char_id", 0)
	_selected_char_id = last_id
	# Set local class from the last played character.
	for ch in data.get("characters", []):
		if ch.char_id == last_id:
			_local_class = ch.class_name
			break
	print("[Main] Character list: %d characters, username=%s" % [data.characters.size(), _account_username])
	_enter_character_select()


func _on_character_error(data: Dictionary) -> void:
	print("[Main] Character error: %s" % data.message)
	if _char_create_error_label:
		_char_create_error_label.text = data.message
		_char_create_error_label.visible = true
	if _char_create_btn:
		_char_create_btn.disabled = false


func _on_net_connected() -> void:
	if state == GameState.CHARACTER_SELECT or state == GameState.CREATE_CHARACTER:
		# ZoneJoined after character selection/creation — enter hub
		print("[Main] Joined hub as peer %d" % NetworkManager.get_my_id())
		_enter_hub()
	else:
		print("[Main] Connected, waiting for character list...")


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
	_char_select_layer.visible = false
	_char_create_layer.visible = false
	_hub_layer.visible = true
	Input.set_mouse_mode(Input.MOUSE_MODE_CAPTURED)
	_near_portal = false
	_portal_prompt.visible = false
	_near_lift = false
	if _lift_prompt:
		_lift_prompt.visible = false

	# Load hub scene if not already loaded
	if _current_env == null or _current_env.name != "Hub":
		_unload_environment()
		_load_environment(HUB_SCENE)

	# Despawn existing players
	_despawn_all_players()

	# Spawn local player in hub (use saved position if returning player)
	var my_id := NetworkManager.get_my_id()
	if my_id > 0:
		var spawn_pos: Vector3 = HUB_SPAWNS[0]
		if _has_saved_state:
			spawn_pos = _saved_hub_position
			_has_saved_state = false
		_spawn_player(my_id, _local_class, spawn_pos)
		if _saved_hub_rot_y != 0.0:
			var player: CharacterBody3D = _spawned_players.get(my_id)
			if player:
				player.rotation.y = _saved_hub_rot_y
			_saved_hub_rot_y = 0.0

	_update_hub_display()
	_update_group_panel()
	if _shared_hud:
		_shared_hud.on_enter_hub()
	if _map_overlay:
		_map_overlay.reset_floor()
	_create_portal_trail()


func _check_portal_proximity() -> void:
	var my_id := NetworkManager.get_my_id()
	if my_id not in _spawned_players:
		_near_portal = false
		_portal_prompt.visible = false
		return
	var player: CharacterBody3D = _spawned_players[my_id]
	if not is_instance_valid(player):
		return
	# Use the actual PortalArea node position
	var portal_area := _current_env.get_node_or_null("PortalArea") if _current_env else null
	if not portal_area:
		_near_portal = false
		_portal_prompt.visible = false
		return
	var dist := player.global_position.distance_to(portal_area.global_position)
	_near_portal = dist < 4.0
	_portal_prompt.visible = _near_portal


func _check_lift_proximity() -> void:
	var my_id := NetworkManager.get_my_id()
	if my_id not in _spawned_players:
		_near_lift = false
		if _lift_prompt:
			_lift_prompt.visible = false
		return
	var player: CharacterBody3D = _spawned_players[my_id]
	if not is_instance_valid(player):
		return

	var pos: Vector3 = player.global_position
	var found_lift: Node = null

	# Check interior elevator (ElevatorCab)
	var elev := _current_env.get_node_or_null("ElevatorCab") if _current_env else null
	if elev and elev.is_idle():
		var elev_pos: Vector3 = (elev as Node3D).global_position
		var door_z: float = elev_pos.z - 2.0
		var in_x := absf(pos.x - elev_pos.x) < 2.5
		var near_door := in_x and absf(pos.z - door_z) < 3.0
		var bottom_y: float = elev.get("BOTTOM_Y")
		var top_y: float = elev.get("TOP_Y")
		if near_door and (pos.y < bottom_y + 5.0 or pos.y > top_y - 5.0):
			found_lift = elev

	# Check public lift — detect at BOTH stations (top and bottom)
	# Top station: (5, 0, -55), Bottom station: (5, -200, -55)
	if not found_lift:
		var plift := _current_env.get_node_or_null("Plaza/PublicLift") if _current_env else null
		if plift and plift.is_idle():
			var station_x := 5.0
			var station_z := -55.0
			var dist_xz := Vector2(pos.x - station_x, pos.z - station_z).length()
			if dist_xz < 6.0:
				# Near top station (Y around 0)
				var near_top := pos.y > -5.0 and pos.y < 5.0
				# Near bottom station (Y around -200)
				var near_bottom := pos.y > -205.0 and pos.y < -195.0
				if near_top or near_bottom:
					found_lift = plift

	_near_lift = found_lift != null
	if _lift_prompt:
		if found_lift:
			var plift_pos_y: float = (found_lift as Node3D).global_position.y
			var lift_here := absf(pos.y - plift_pos_y) < 5.0
			if lift_here:
				_lift_prompt.text = "Press [E] — %s" % found_lift.get_floor_label()
			else:
				_lift_prompt.text = "Press [E] — Call lift"
		_lift_prompt.visible = _near_lift


func _interact_lift() -> void:
	var pos: Vector3 = Vector3.ZERO
	var my_id := NetworkManager.get_my_id()
	if my_id in _spawned_players:
		pos = _spawned_players[my_id].global_position

	# Try interior elevator first
	var elev := _current_env.get_node_or_null("ElevatorCab") if _current_env else null
	if elev and elev.is_idle():
		var elev_pos: Vector3 = (elev as Node3D).global_position
		var door_z: float = elev_pos.z - 2.0
		if absf(pos.x - elev_pos.x) < 2.5 and absf(pos.z - door_z) < 3.0:
			elev.activate()
			return

	# Try public lift — fixed stations at top (Y=0) and bottom (Y=-200)
	var plift := _current_env.get_node_or_null("Plaza/PublicLift") if _current_env else null
	if plift and plift.is_idle():
		var dist_xz := Vector2(pos.x - 5.0, pos.z - (-55.0)).length()
		var near_top := pos.y > -5.0 and pos.y < 5.0
		var near_bottom := pos.y > -205.0 and pos.y < -195.0
		if dist_xz < 6.0 and (near_top or near_bottom):
			plift.activate()
			return


func _update_hub_display() -> void:
	if _hub_class_label:
		_hub_class_label.text = "Class: %s" % _local_class.to_upper()


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
# Arena warmup
# =============================================================================

func _enter_arena_warmup() -> void:
	state = GameState.ARENA_LOBBY
	get_tree().paused = false
	paused = false
	_pause_layer.visible = false
	_menu_layer.visible = false
	_remove_exit_portal()
	_hub_layer.visible = false
	if _shared_hud:
		_shared_hud.on_enter_arena()
	if _map_overlay:
		_map_overlay.set_floor("arena", "Arena")

	# Load arena scene if not already loaded
	if _current_env == null or _current_env.name != "Arena":
		_unload_environment()
		_load_environment(ARENA_SCENE)
		_create_hallway_geometry()
		_create_atmosphere()
		_create_arena_buildings()


func _select_class(class_name_str: String) -> void:
	_local_class = class_name_str
	if NetworkManager.is_active:
		NetworkManager.set_player_class(class_name_str)
	_update_hub_display()


# =============================================================================
# Spawning
# =============================================================================

func _spawn_multiplayer_players() -> void:
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
	player.add_to_group("players")
	player.global_position = spawn_pos
	# Initialize net sync targets so remote interpolation starts at the correct position
	player._net_position = spawn_pos
	player._net_rotation_y = player.rotation.y
	# Apply hub spawn facing direction for local player
	if state == GameState.HUB and peer_id == NetworkManager.get_my_id():
		player.rotation.y = HUB_SPAWN_YAW
		player._net_rotation_y = HUB_SPAWN_YAW
		if "_camera_yaw" in player:
			player._camera_yaw = HUB_SPAWN_YAW
	_spawned_players[peer_id] = player

	# Feed local player to shared HUD and connect death signal
	if peer_id == NetworkManager.get_my_id():
		if _shared_hud:
			_shared_hud.set_local_player(player, class_name_str, peer_id)
		if _map_overlay:
			_map_overlay.set_local_info(peer_id, _player_names)
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
	_hub_layer.visible = false
	_cursor_toggled = false
	_alt_held = false

	# Enemies are managed dynamically via _update_enemies from world state
	CombatLog.start_fight()
	if _shared_hud:
		_shared_hud.on_fight_start()


func _on_boss_dead() -> void:
	state = GameState.FIGHT_OVER
	_open_boss_gate()
	_spawn_exit_portal()
	if _local_player_dead and _death_overlay_layer.visible:
		_respawn_btn.disabled = false
	CombatLog.end_fight("VICTORY")
	if _shared_hud:
		_shared_hud.on_fight_end()


func _on_all_dead() -> void:
	state = GameState.FIGHT_OVER
	_open_boss_gate()
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
	_clear_all_enemies()
	if _map_overlay:
		_map_overlay.visible = false
	_clear_all_npcs()

	if zone_type == NetSerializer.ZONE_TYPE_ARENA:
		_unload_environment()
		_load_environment(ARENA_SCENE)
		_create_hallway_geometry()
		_create_atmosphere()
		_create_arena_buildings()
		# Spawn local player in warmup room immediately
		state = GameState.ARENA_LOBBY
		_hub_layer.visible = false
		_menu_layer.visible = false
		if _shared_hud:
			_shared_hud.on_enter_arena()
		Input.set_mouse_mode(Input.MOUSE_MODE_CAPTURED)
		var my_id := NetworkManager.get_my_id()
		if my_id > 0:
			_spawn_player(my_id, _local_class, LOBBY_SPAWN)
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
			_clear_all_enemies()
			_clear_all_npcs()
			_enter_arena_warmup()
		NetSerializer.FLOW_BOSS_ACTIVATED:
			_close_boss_gate()
		NetSerializer.FLOW_BOSS_RESET:
			_open_boss_gate()


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

	# Enemies — dynamically spawn/update/despawn from server state
	var enemies_data: Array = data.get("enemies", [])
	_update_enemies(enemies_data)

	# NPCs — hub ambient characters
	var npcs_data: Array = data.get("npcs", [])
	_update_npcs(npcs_data)

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

	# Feed map overlay (only when visible to avoid unnecessary work)
	if _map_overlay and _map_overlay.visible:
		var local_pos := Vector3.ZERO
		var local_rot := 0.0
		if my_id in _spawned_players and is_instance_valid(_spawned_players[my_id]):
			local_pos = _spawned_players[my_id].global_position
			local_rot = _spawned_players[my_id].rotation.y
		_map_overlay.update_state({
			"player_pos": local_pos,
			"player_rot_y": local_rot,
			"players": _shared_hud._world_players,
			"npcs": _shared_hud._npc_positions,
			"enemies": _shared_hud._enemy_positions,
		})
		if _portal_trail and local_pos != Vector3.ZERO:
			_map_overlay.set_waypoint_path(
				_portal_trail.get_path_to_target(local_pos))


func _on_damage_event(data: Dictionary) -> void:
	var target_peer: int = data.get("target_peer_id", -1)
	var source_peer: int = data.get("source_peer_id", 0)
	var amount: float = data.get("amount", 0.0)
	var hit_pos: Vector3 = data.get("hit_pos", Vector3.ZERO)
	var source_type: int = data.get("source_type", 0)
	if target_peer >= 1000:
		# Player hit an enemy (enemy IDs are >= 1000)
		if target_peer in _enemy_nodes:
			var enode: CharacterBody3D = _enemy_nodes[target_peer]
			if is_instance_valid(enode) and enode.has_method("on_damage_visual"):
				enode.on_damage_visual(amount, hit_pos)
		# Server-confirmed hit marker on the attacker's HUD
		if source_peer == NetworkManager.get_my_id():
			var local_player: CharacterBody3D = _spawned_players.get(source_peer)
			if is_instance_valid(local_player) and local_player.has_method("on_hit_confirmed"):
				local_player.on_hit_confirmed(amount)
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
	if _shared_hud:
		_shared_hud.set_environment(_current_env)
	print("[Main] Loaded environment: %s" % scene_path)


func _unload_environment() -> void:
	if _current_env and is_instance_valid(_current_env):
		_current_env.queue_free()
		_current_env = null
	_clear_all_enemies()
	_clear_all_npcs()
	if _boss_gate and is_instance_valid(_boss_gate):
		_boss_gate.queue_free()
	_boss_gate = null
	_atmosphere = null
	if _arena_buildings and is_instance_valid(_arena_buildings):
		_arena_buildings.queue_free()
	_arena_buildings = null
	_remove_portal_trail()


func _update_enemies(enemies_data: Array) -> void:
	var seen_ids: Dictionary = {}
	for edata in enemies_data:
		var eid: int = edata["enemy_id"]
		seen_ids[eid] = true
		var alive: bool = edata["alive"]
		if alive and eid not in _enemy_nodes:
			# Spawn new enemy node
			var scene := load("res://scenes/enemies/basic_enemy/basic_enemy.tscn") as PackedScene
			if scene:
				var node := scene.instantiate() as CharacterBody3D
				node.name = "Enemy_%d" % eid
				node.peer_id = eid
				add_child(node)
				_enemy_nodes[eid] = node
		if eid in _enemy_nodes:
			var node: CharacterBody3D = _enemy_nodes[eid]
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
	for eid in _enemy_nodes:
		if eid not in seen_ids:
			to_remove.append(eid)
	for eid in to_remove:
		var node = _enemy_nodes[eid]
		if is_instance_valid(node):
			node.queue_free()
		_enemy_nodes.erase(eid)


func _clear_all_enemies() -> void:
	for eid in _enemy_nodes:
		var node = _enemy_nodes[eid]
		if is_instance_valid(node):
			node.queue_free()
	_enemy_nodes.clear()


func _update_npcs(npcs_data: Array) -> void:
	var seen_ids: Dictionary = {}
	for ndata in npcs_data:
		var nid: int = ndata["npc_id"]
		seen_ids[nid] = true
		if nid not in _npc_nodes:
			var node := _create_npc_node(ndata)
			add_child(node)
			_npc_nodes[nid] = node
		var node: Node3D = _npc_nodes[nid]
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
	for nid in _npc_nodes:
		if nid not in seen_ids:
			to_remove.append(nid)
	for nid in to_remove:
		var node = _npc_nodes[nid]
		if is_instance_valid(node):
			node.queue_free()
		_npc_nodes.erase(nid)


const NPC_MODEL_SCENE := "res://scenes/shared/character_model/character_model.tscn"
const NPC_PUPPET_SCRIPT := "res://scenes/shared/npc_puppet/npc_puppet.gd"

func _create_npc_node(ndata: Dictionary) -> Node3D:
	var root := Node3D.new()
	root.name = "NPC_%d" % ndata["npc_id"]
	root.position = ndata["pos"]
	root.rotation.y = ndata["rot_y"]
	root.set_script(load(NPC_PUPPET_SCRIPT))

	var def_name: String = ndata.get("def_name", "citizen")

	# Character model (Mixamo) — same as enemies/players
	var model_scene := load(NPC_MODEL_SCENE) as PackedScene
	if model_scene:
		var model := model_scene.instantiate()
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


func _clear_all_npcs() -> void:
	for nid in _npc_nodes:
		var node = _npc_nodes[nid]
		if is_instance_valid(node):
			node.queue_free()
	_npc_nodes.clear()


func _create_hallway_geometry() -> void:
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
	_current_env.add_child(hall_floor)

	# Hallway left wall
	var hall_wall_l := CSGBox3D.new()
	hall_wall_l.name = "HallwayWallLeft"
	hall_wall_l.size = Vector3(0.5, 5.0, 28.0)
	hall_wall_l.transform.origin = Vector3(-8.0, 2.5, 26.0)
	hall_wall_l.use_collision = true
	hall_wall_l.collision_layer = 1
	hall_wall_l.collision_mask = 0
	hall_wall_l.material = mat_wall
	_current_env.add_child(hall_wall_l)

	# Hallway right wall
	var hall_wall_r := CSGBox3D.new()
	hall_wall_r.name = "HallwayWallRight"
	hall_wall_r.size = Vector3(0.5, 5.0, 28.0)
	hall_wall_r.transform.origin = Vector3(8.0, 2.5, 26.0)
	hall_wall_r.use_collision = true
	hall_wall_r.collision_layer = 1
	hall_wall_r.collision_mask = 0
	hall_wall_r.material = mat_wall
	_current_env.add_child(hall_wall_r)

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
	_current_env.add_child(conn_wall_l)

	# Right connector at Z=12
	var conn_wall_r := CSGBox3D.new()
	conn_wall_r.name = "ConnectorWallRight"
	conn_wall_r.size = Vector3(12.0, 5.0, 0.5)
	conn_wall_r.transform.origin = Vector3(14.0, 2.5, 11.6)
	conn_wall_r.use_collision = true
	conn_wall_r.collision_layer = 1
	conn_wall_r.collision_mask = 0
	conn_wall_r.material = mat_wall
	_current_env.add_child(conn_wall_r)

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
		_current_env.add_child(cover)

	# Boss room gate at Z=12 (hidden by default, closes when boss aggros)
	_boss_gate = CSGBox3D.new()
	_boss_gate.size = Vector3(40.0, 5.0, 0.5)
	_boss_gate.transform.origin = Vector3(0.0, 2.5, BOSS_ROOM_ENTRY_Z)
	_boss_gate.use_collision = true
	_boss_gate.collision_layer = 1
	_boss_gate.collision_mask = 0
	var gate_mat := StandardMaterial3D.new()
	gate_mat.albedo_color = Color(0.6, 0.15, 0.15)
	gate_mat.emission_enabled = true
	gate_mat.emission = Color(0.5, 0.1, 0.1)
	gate_mat.emission_energy_multiplier = 0.5
	_boss_gate.material = gate_mat
	_boss_gate.visible = false
	_boss_gate.use_collision = false
	_current_env.add_child(_boss_gate)


func _create_arena_buildings() -> void:
	if _arena_buildings and is_instance_valid(_arena_buildings):
		_arena_buildings.queue_free()
	var BuildingsScript := load("res://scenes/environments/arena/arena_buildings.gd")
	_arena_buildings = Node3D.new()
	_arena_buildings.name = "ArenaBuildings"
	_arena_buildings.set_script(BuildingsScript)
	_current_env.add_child(_arena_buildings)


func _create_atmosphere() -> void:
	if _atmosphere and is_instance_valid(_atmosphere):
		_atmosphere.queue_free()
	var AtmosphereScript := load("res://scenes/environments/arena/dungeon_atmosphere.gd")
	_atmosphere = Node3D.new()
	_atmosphere.name = "DungeonAtmosphere"
	_atmosphere.set_script(AtmosphereScript)
	_current_env.add_child(_atmosphere)


func _close_boss_gate() -> void:
	if _boss_gate:
		_boss_gate.visible = true
		_boss_gate.use_collision = true
	# Push local player into the boss room if near the gate
	var my_id := NetworkManager.get_my_id()
	if my_id in _spawned_players:
		var player: CharacterBody3D = _spawned_players[my_id]
		if is_instance_valid(player) and player.global_position.z > BOSS_ROOM_ENTRY_Z - 2.0 and player.global_position.z < BOSS_ROOM_ENTRY_Z + 2.0:
			player.global_position.z = BOSS_ROOM_ENTRY_Z - 3.0


func _open_boss_gate() -> void:
	if _boss_gate:
		_boss_gate.visible = false
		_boss_gate.use_collision = false


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

	# Welcome label — shown for returning players.
	_menu_welcome_label = Label.new()
	_menu_welcome_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	_menu_welcome_label.add_theme_font_size_override("font_size", 22)
	_menu_welcome_label.add_theme_color_override("font_color", Color(0.7, 0.75, 0.9))
	_menu_welcome_label.visible = false
	vbox.add_child(_menu_welcome_label)

	# Username input — only shown for new players (no saved username).
	_username_input = LineEdit.new()
	_username_input.placeholder_text = "Choose a username..."
	_username_input.custom_minimum_size.y = 50.0
	_username_input.max_length = 20
	_username_input.alignment = HORIZONTAL_ALIGNMENT_CENTER
	vbox.add_child(_username_input)

	# Load saved username — if exists, show welcome instead of input.
	var saved := _load_saved_username()
	if saved != "":
		_username = saved
		_username_input.visible = false
		_menu_welcome_label.text = "Welcome back, %s" % saved
		_menu_welcome_label.visible = true

	var play_btn := Button.new()
	play_btn.text = "Play"
	play_btn.custom_minimum_size = Vector2(200.0, 55.0)
	play_btn.pressed.connect(_on_connect_pressed)
	vbox.add_child(play_btn)


func _create_char_select_ui() -> void:
	_char_select_layer = CanvasLayer.new()
	_char_select_layer.layer = 18
	_char_select_layer.visible = false
	add_child(_char_select_layer)

	var bg := ColorRect.new()
	bg.color = Color(0.05, 0.05, 0.1, 0.95)
	bg.anchor_right = 1.0
	bg.anchor_bottom = 1.0
	bg.mouse_filter = Control.MOUSE_FILTER_STOP
	_char_select_layer.add_child(bg)

	var outer := VBoxContainer.new()
	outer.anchor_left = 0.5
	outer.anchor_right = 0.5
	outer.anchor_top = 0.1
	outer.anchor_bottom = 0.9
	outer.offset_left = -300.0
	outer.offset_right = 300.0
	outer.add_theme_constant_override("separation", 20)
	_char_select_layer.add_child(outer)

	# Title
	var title := Label.new()
	title.text = "SELECT CHARACTER"
	title.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	title.add_theme_font_size_override("font_size", 36)
	title.add_theme_color_override("font_color", Color(0.8, 0.85, 1.0))
	outer.add_child(title)

	# Welcome label
	_char_select_welcome = Label.new()
	_char_select_welcome.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	_char_select_welcome.add_theme_font_size_override("font_size", 16)
	_char_select_welcome.add_theme_color_override("font_color", Color(0.5, 0.5, 0.6))
	outer.add_child(_char_select_welcome)

	# Scrollable character list
	var scroll := ScrollContainer.new()
	scroll.size_flags_vertical = Control.SIZE_EXPAND_FILL
	scroll.custom_minimum_size.y = 200.0
	outer.add_child(scroll)

	_char_list_container = VBoxContainer.new()
	_char_list_container.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	_char_list_container.add_theme_constant_override("separation", 4)
	scroll.add_child(_char_list_container)

	# Buttons
	var btn_hbox := HBoxContainer.new()
	btn_hbox.alignment = BoxContainer.ALIGNMENT_CENTER
	btn_hbox.add_theme_constant_override("separation", 20)
	outer.add_child(btn_hbox)

	var back_btn := Button.new()
	back_btn.text = "Back"
	back_btn.custom_minimum_size = Vector2(100.0, 45.0)
	back_btn.pressed.connect(func():
		NetworkManager.disconnect_game()
		_enter_menu()
	)
	btn_hbox.add_child(back_btn)

	var create_btn := Button.new()
	create_btn.text = "Create New Character"
	create_btn.custom_minimum_size = Vector2(200.0, 45.0)
	create_btn.pressed.connect(_enter_create_character)
	btn_hbox.add_child(create_btn)

	_enter_world_btn = Button.new()
	_enter_world_btn.text = "Enter World"
	_enter_world_btn.custom_minimum_size = Vector2(160.0, 45.0)
	_enter_world_btn.pressed.connect(_on_enter_world_pressed)
	btn_hbox.add_child(_enter_world_btn)


func _create_char_create_ui() -> void:
	_char_create_layer = CanvasLayer.new()
	_char_create_layer.layer = 18
	_char_create_layer.visible = false
	add_child(_char_create_layer)

	var bg := ColorRect.new()
	bg.color = Color(0.05, 0.05, 0.1, 0.95)
	bg.anchor_right = 1.0
	bg.anchor_bottom = 1.0
	bg.mouse_filter = Control.MOUSE_FILTER_STOP
	_char_create_layer.add_child(bg)

	var outer := VBoxContainer.new()
	outer.anchor_left = 0.5
	outer.anchor_right = 0.5
	outer.anchor_top = 0.1
	outer.anchor_bottom = 0.9
	outer.offset_left = -350.0
	outer.offset_right = 350.0
	outer.add_theme_constant_override("separation", 20)
	_char_create_layer.add_child(outer)

	var title := Label.new()
	title.text = "CREATE CHARACTER"
	title.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	title.add_theme_font_size_override("font_size", 36)
	title.add_theme_color_override("font_color", Color(0.8, 0.85, 1.0))
	outer.add_child(title)

	# Class cards
	var cards_hbox := HBoxContainer.new()
	cards_hbox.add_theme_constant_override("separation", 20)
	cards_hbox.alignment = BoxContainer.ALIGNMENT_CENTER
	cards_hbox.size_flags_vertical = Control.SIZE_EXPAND_FILL
	outer.add_child(cards_hbox)

	var normal_style := StyleBoxFlat.new()
	normal_style.bg_color = Color(0.12, 0.12, 0.18, 0.9)
	normal_style.border_color = Color(0.3, 0.3, 0.4, 0.6)
	normal_style.set_border_width_all(2)
	normal_style.set_corner_radius_all(6)
	normal_style.set_content_margin_all(16)

	var selected_style := StyleBoxFlat.new()
	selected_style.bg_color = Color(0.1, 0.12, 0.22, 0.95)
	selected_style.border_color = Color(0.3, 0.6, 1.0, 0.9)
	selected_style.set_border_width_all(3)
	selected_style.set_corner_radius_all(6)
	selected_style.set_content_margin_all(16)

	for cls in CLASS_INFO:
		var info: Dictionary = CLASS_INFO[cls]
		var card := PanelContainer.new()
		card.custom_minimum_size = Vector2(200.0, 250.0)
		card.size_flags_horizontal = Control.SIZE_EXPAND_FILL
		card.add_theme_stylebox_override("panel", normal_style)
		card.set_meta("normal_style", normal_style)
		card.set_meta("selected_style", selected_style)
		cards_hbox.add_child(card)
		_char_create_cards[cls] = card

		var vbox := VBoxContainer.new()
		vbox.add_theme_constant_override("separation", 10)
		card.add_child(vbox)

		var name_label := Label.new()
		name_label.text = info.name
		name_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
		name_label.add_theme_font_size_override("font_size", 24)
		name_label.add_theme_color_override("font_color", Color(0.9, 0.9, 0.95))
		vbox.add_child(name_label)

		var genre_label := Label.new()
		genre_label.text = info.genre
		genre_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
		genre_label.add_theme_font_size_override("font_size", 14)
		genre_label.add_theme_color_override("font_color", Color(0.4, 0.65, 1.0))
		vbox.add_child(genre_label)

		var sep := HSeparator.new()
		sep.add_theme_color_override("separator", Color(0.3, 0.3, 0.4, 0.4))
		vbox.add_child(sep)

		var desc_label := Label.new()
		desc_label.text = info.desc
		desc_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
		desc_label.add_theme_font_size_override("font_size", 14)
		desc_label.add_theme_color_override("font_color", Color(0.6, 0.6, 0.65))
		desc_label.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
		vbox.add_child(desc_label)

		# Click detection
		var click_btn := Button.new()
		click_btn.flat = true
		click_btn.anchor_right = 1.0
		click_btn.anchor_bottom = 1.0
		click_btn.mouse_filter = Control.MOUSE_FILTER_STOP
		var cls_capture: String = cls
		click_btn.pressed.connect(func(): _select_create_class(cls_capture))
		card.add_child(click_btn)

	# Name input
	var name_label := Label.new()
	name_label.text = "Character Name"
	name_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	name_label.add_theme_font_size_override("font_size", 18)
	name_label.add_theme_color_override("font_color", Color(0.7, 0.7, 0.8))
	outer.add_child(name_label)

	_char_name_input = LineEdit.new()
	_char_name_input.placeholder_text = "Enter a name (2-20 characters)..."
	_char_name_input.custom_minimum_size.y = 45.0
	_char_name_input.max_length = 20
	_char_name_input.alignment = HORIZONTAL_ALIGNMENT_CENTER
	outer.add_child(_char_name_input)

	# Error label
	_char_create_error_label = Label.new()
	_char_create_error_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	_char_create_error_label.add_theme_font_size_override("font_size", 14)
	_char_create_error_label.add_theme_color_override("font_color", Color(1.0, 0.3, 0.3))
	_char_create_error_label.visible = false
	outer.add_child(_char_create_error_label)

	# Buttons
	var btn_hbox := HBoxContainer.new()
	btn_hbox.alignment = BoxContainer.ALIGNMENT_CENTER
	btn_hbox.add_theme_constant_override("separation", 20)
	outer.add_child(btn_hbox)

	var back_btn := Button.new()
	back_btn.text = "Back"
	back_btn.custom_minimum_size = Vector2(100.0, 45.0)
	back_btn.pressed.connect(func():
		_char_create_layer.visible = false
		_enter_character_select()
	)
	btn_hbox.add_child(back_btn)

	_char_create_btn = Button.new()
	_char_create_btn.text = "Create"
	_char_create_btn.custom_minimum_size = Vector2(160.0, 45.0)
	_char_create_btn.pressed.connect(_on_create_character_pressed)
	btn_hbox.add_child(_char_create_btn)


# =============================================================================
# Character Select logic
# =============================================================================

func _enter_character_select() -> void:
	state = GameState.CHARACTER_SELECT
	_menu_layer.visible = false
	_hub_layer.visible = false
	_char_create_layer.visible = false
	_char_select_layer.visible = true
	Input.set_mouse_mode(Input.MOUSE_MODE_VISIBLE)
	_populate_char_select()


func _populate_char_select() -> void:
	# Update welcome label.
	if _account_username != "":
		_char_select_welcome.text = "Welcome, %s" % _account_username
	else:
		_char_select_welcome.text = ""

	# Clear existing rows.
	for child in _char_list_container.get_children():
		child.queue_free()

	var characters: Array = _char_list_data.get("characters", [])
	var last_id: int = _char_list_data.get("last_char_id", 0)

	var normal_style := StyleBoxFlat.new()
	normal_style.bg_color = Color(0.12, 0.12, 0.18, 0.8)
	normal_style.border_color = Color(0.3, 0.3, 0.4, 0.4)
	normal_style.set_border_width_all(1)
	normal_style.set_corner_radius_all(4)
	normal_style.set_content_margin_all(12)

	var selected_style := StyleBoxFlat.new()
	selected_style.bg_color = Color(0.1, 0.12, 0.22, 0.95)
	selected_style.border_color = Color(0.3, 0.6, 1.0, 0.9)
	selected_style.set_border_width_all(2)
	selected_style.set_corner_radius_all(4)
	selected_style.set_content_margin_all(12)

	if characters.is_empty():
		var empty_label := Label.new()
		empty_label.text = "No characters yet. Create one to get started!"
		empty_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
		empty_label.add_theme_font_size_override("font_size", 16)
		empty_label.add_theme_color_override("font_color", Color(0.5, 0.5, 0.6))
		_char_list_container.add_child(empty_label)
		_enter_world_btn.disabled = true
		return

	_enter_world_btn.disabled = false

	for ch in characters:
		var char_id: int = ch.char_id
		var class_display: String = CLASS_INFO.get(ch.class_name, {}).get("name", ch.class_name)

		var row := PanelContainer.new()
		row.custom_minimum_size.y = 50.0
		row.size_flags_horizontal = Control.SIZE_EXPAND_FILL
		row.set_meta("char_id", char_id)
		row.set_meta("normal_style", normal_style)
		row.set_meta("selected_style", selected_style)
		if char_id == _selected_char_id:
			row.add_theme_stylebox_override("panel", selected_style)
		else:
			row.add_theme_stylebox_override("panel", normal_style)
		_char_list_container.add_child(row)

		var hbox := HBoxContainer.new()
		hbox.add_theme_constant_override("separation", 20)
		row.add_child(hbox)

		var name_lbl := Label.new()
		name_lbl.text = ch.char_name
		name_lbl.add_theme_font_size_override("font_size", 18)
		name_lbl.add_theme_color_override("font_color", Color(0.9, 0.9, 0.95))
		name_lbl.size_flags_horizontal = Control.SIZE_EXPAND_FILL
		hbox.add_child(name_lbl)

		var class_lbl := Label.new()
		class_lbl.text = class_display
		class_lbl.add_theme_font_size_override("font_size", 16)
		class_lbl.add_theme_color_override("font_color", Color(0.4, 0.65, 1.0))
		hbox.add_child(class_lbl)

		# Click detection
		var btn := Button.new()
		btn.flat = true
		btn.anchor_right = 1.0
		btn.anchor_bottom = 1.0
		btn.mouse_filter = Control.MOUSE_FILTER_STOP
		var id_capture: int = char_id
		var cls_capture: String = ch.class_name
		btn.pressed.connect(func(): _select_character_row(id_capture, cls_capture))
		row.add_child(btn)

	# Auto-select last played if none selected.
	if _selected_char_id == 0 and not characters.is_empty():
		_select_character_row(characters[0].char_id, characters[0].class_name)


func _select_character_row(char_id: int, class_name_str: String) -> void:
	_selected_char_id = char_id
	_local_class = class_name_str
	# Update row highlights.
	for row in _char_list_container.get_children():
		if row is PanelContainer and row.has_meta("char_id"):
			if row.get_meta("char_id") == char_id:
				row.add_theme_stylebox_override("panel", row.get_meta("selected_style"))
			else:
				row.add_theme_stylebox_override("panel", row.get_meta("normal_style"))


func _on_enter_world_pressed() -> void:
	if _selected_char_id == 0:
		return
	_enter_world_btn.disabled = true
	NetworkManager.send_select_character(_selected_char_id)


# =============================================================================
# Create Character logic
# =============================================================================

func _enter_create_character() -> void:
	state = GameState.CREATE_CHARACTER
	_char_select_layer.visible = false
	_char_create_layer.visible = true
	_char_create_error_label.visible = false
	_char_name_input.text = ""
	_char_create_btn.disabled = false
	Input.set_mouse_mode(Input.MOUSE_MODE_VISIBLE)
	# Default select gunner.
	_select_create_class("gunner")


func _select_create_class(cls: String) -> void:
	_local_class = cls
	for c_name in _char_create_cards:
		var card: PanelContainer = _char_create_cards[c_name]
		if c_name == cls:
			card.add_theme_stylebox_override("panel", card.get_meta("selected_style"))
		else:
			card.add_theme_stylebox_override("panel", card.get_meta("normal_style"))


func _on_create_character_pressed() -> void:
	var char_name := _char_name_input.text.strip_edges()
	if char_name.length() < 2 or char_name.length() > 20:
		_char_create_error_label.text = "Name must be 2-20 characters."
		_char_create_error_label.visible = true
		return
	_char_create_error_label.visible = false
	_char_create_btn.disabled = true
	NetworkManager.send_create_character(_local_class, char_name)


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

	# Lift prompt
	_lift_prompt = Label.new()
	_lift_prompt.text = "Press [E] — Go up"
	_lift_prompt.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	_lift_prompt.anchor_left = 0.0
	_lift_prompt.anchor_right = 1.0
	_lift_prompt.anchor_top = 0.65
	_lift_prompt.anchor_bottom = 0.7
	_lift_prompt.add_theme_font_size_override("font_size", 24)
	_lift_prompt.add_theme_color_override("font_color", Color(0.5, 0.7, 1.0))
	_lift_prompt.visible = false
	_hub_layer.add_child(_lift_prompt)


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

	_map_overlay = preload("res://scenes/shared/hud/map_overlay.gd").new()
	_map_overlay.name = "MapOverlay"
	_map_overlay.anchor_right = 1.0
	_map_overlay.anchor_bottom = 1.0
	_map_overlay.mouse_filter = Control.MOUSE_FILTER_IGNORE
	_map_overlay.visible = false
	_shared_hud_layer.add_child(_map_overlay)


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


func _create_portal_trail() -> void:
	_remove_portal_trail()
	var TrailScript := load("res://scenes/environments/prime_hub/portal_trail.gd")
	_portal_trail = Node3D.new()
	_portal_trail.name = "PortalTrail"
	_portal_trail.set_script(TrailScript)
	add_child(_portal_trail)


func _remove_portal_trail() -> void:
	if _portal_trail and is_instance_valid(_portal_trail):
		_portal_trail.queue_free()
	_portal_trail = null
