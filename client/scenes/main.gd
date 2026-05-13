extends Node3D

## Game flow: Menu -> Hub -> Arena (Lobby -> Fight -> Fight Over) -> Hub.
## Server-authoritative: all game flow is driven by server events.

enum GameState {
	MENU,
	CHARACTER_SELECT,
	CREATE_CHARACTER,
	HUB,
	ARENA_LOBBY,
	FIGHT,
	FIGHT_OVER,
	REPLAY_BROWSER,
	REPLAY,
}

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
	"gunner":
	{
		"name": "Gunner",
		"genre": "FPS",
		"desc": "Fast movement, high fire rate.\nRelentless aggression."
	},
	"vanguard":
	{
		"name": "Vanguard",
		"genre": "Souls-like",
		"desc": "Big AoE swings, punish windows.\nHeavy and deliberate."
	},
	"blade_dancer":
	{
		"name": "Blade Dancer",
		"genre": "State Machine",
		"desc": "5 configurations, 4 spells each.\nHighest skill ceiling."
	},
}
const SERVER_ADDRESS := "90.29.26.144"
const USERNAME_SAVE_PATH := "user://username.txt"
const UI_SURFACE := Color(0.035, 0.045, 0.065, 0.88)
const UI_SURFACE_ALT := Color(0.05, 0.06, 0.085, 0.92)
const UI_SURFACE_ACTIVE := Color(0.08, 0.1, 0.15, 0.96)
const UI_BORDER := Color(0.28, 0.31, 0.37, 0.9)
const UI_BORDER_ACTIVE := Color(0.32, 0.58, 0.92, 0.95)
const UI_TEXT := Color(0.9, 0.93, 0.98, 0.96)
const UI_TEXT_MUTED := Color(0.6, 0.66, 0.75, 0.95)
const UI_TEXT_DIM := Color(0.48, 0.53, 0.6, 0.95)
const UI_DANGER := Color(0.86, 0.28, 0.28, 0.96)

var state: GameState = GameState.MENU
var paused: bool = false
var dev_mode: bool = false
var _debug_panel: CanvasLayer = null
var _server_pid: int = -1
var _dev_class: String = "gunner"
var _dev_zone: String = "arena"
var _dev_connected: bool = false
var _local_class: String = "gunner"
var _saved_hub_position: Vector3 = Vector3.ZERO
var _saved_hub_rot_y: float = 0.0
var _has_saved_state: bool = false
var _username: String = ""
var _player_names: Dictionary = {}  # peer_id -> username
var _cursor_toggled: bool = false  # backtick toggle state
var _alt_held: bool = false  # alt hold state
var _local_player_dead: bool = false
var _replay_browser: CanvasLayer = null
var _replay_scene: Node3D = null
var _char_list_data: Dictionary = {}
var _selected_char_id: int = 0
var _account_username: String = ""
var _username_input: LineEdit
var _menu_welcome_label: Label
var _char_list_container: VBoxContainer
var _enter_world_btn: Button
var _char_select_welcome: Label
var _char_create_cards: Dictionary = {}
var _char_name_input: LineEdit
var _char_create_error_label: Label
var _char_create_btn: Button
var _hub_class_label: Label
var _hub_status_label: Label
var _portal_prompt: Label
var _lift_prompt: Label
var _group_panel: PanelContainer
var _group_label: Label
var _group_leave_btn: Button
var _invite_label: Label
var _shared_hud: Control
var _map_overlay: Control
var _death_overlay_bg: ColorRect
var _respawn_btn: Button
var _respawn_hub_btn: Button

# Group data (alias for group_mgr access pattern)
var _group_data: Dictionary:
	get:
		if group_mgr:
			return group_mgr.group_data
		return {}
	set(value):
		if group_mgr:
			group_mgr.group_data = value
# Portal trail (alias for env_builder)
var _portal_trail: Node3D:
	get:
		if env_builder:
			return env_builder.portal_trail
		return null

# Sub-systems (static children in main.tscn)
@onready var entity_mgr: Node = $EntityManager
@onready var world_sync: Node = $WorldStateSync
@onready var hub_interact: Node = $HubInteraction
@onready var group_mgr: Node = $GroupManager
@onready var env_builder: Node = $EnvironmentBuilder
@onready var ui_ctrl: Node = $UIController
# UI scenes (static instances in main.tscn)
@onready var _pause_layer: CanvasLayer = $PauseMenu
@onready var _menu_layer: CanvasLayer = $MenuUI
@onready var _char_select_layer: CanvasLayer = $CharSelectUI
@onready var _char_create_layer: CanvasLayer = $CharCreateUI
@onready var _hub_layer: CanvasLayer = $HubHUD
@onready var _invite_popup: CanvasLayer = $InvitePopup
@onready var _shared_hud_layer: CanvasLayer = $SharedHUD
@onready var _death_overlay_layer: CanvasLayer = $DeathOverlay


func _ready() -> void:
	process_mode = Node.PROCESS_MODE_ALWAYS

	# Resolve inner UI element references from static scenes
	_username_input = _menu_layer.username_input
	_menu_welcome_label = _menu_layer.welcome_label
	_char_list_container = _char_select_layer.char_list_container
	_enter_world_btn = _char_select_layer.enter_world_btn
	_char_select_welcome = _char_select_layer.welcome_label
	_char_create_cards = _char_create_layer.cards
	_char_name_input = _char_create_layer.name_input
	_char_create_error_label = _char_create_layer.error_label
	_char_create_btn = _char_create_layer.create_btn
	_hub_class_label = _hub_layer.class_label
	_hub_status_label = _hub_layer.status_label
	_portal_prompt = _hub_layer.portal_prompt
	_lift_prompt = _hub_layer.lift_prompt
	_group_panel = _hub_layer.group_panel
	_group_label = _hub_layer.group_label
	_group_leave_btn = _hub_layer.group_leave_btn
	_invite_label = _invite_popup.invite_label
	_shared_hud = _shared_hud_layer.hud
	_map_overlay = _shared_hud_layer.map_overlay
	_death_overlay_bg = _death_overlay_layer.background
	_respawn_btn = _death_overlay_layer.respawn_btn
	_respawn_hub_btn = _death_overlay_layer.respawn_hub_btn

	# Connect UI signals
	_pause_layer.resume_btn.pressed.connect(_toggle_pause)
	_pause_layer.menu_btn.pressed.connect(
		func():
			get_tree().paused = false
			paused = false
			entity_mgr.despawn_all_players()
			_enter_menu()
	)
	_pause_layer.quit_btn.pressed.connect(func(): get_tree().quit())
	_menu_layer.play_btn.pressed.connect(_on_connect_pressed)
	_menu_layer.replays_btn.pressed.connect(_enter_replay_browser)
	_char_select_layer.back_btn.pressed.connect(
		func():
			NetworkManager.disconnect_game()
			_enter_menu()
	)
	_char_select_layer.create_btn.pressed.connect(_enter_create_character)
	_enter_world_btn.pressed.connect(_on_enter_world_pressed)
	_char_create_layer.back_btn.pressed.connect(
		func():
			_char_create_layer.visible = false
			_enter_character_select()
	)
	_char_create_btn.pressed.connect(_on_create_character_pressed)
	_group_leave_btn.pressed.connect(func(): NetworkManager.send_group_leave())
	_invite_popup.accept_btn.pressed.connect(group_mgr.accept_invite)
	_invite_popup.decline_btn.pressed.connect(group_mgr.decline_invite)
	_respawn_btn.pressed.connect(_on_respawn)
	_respawn_hub_btn.pressed.connect(_on_respawn_hub)

	# Load saved username
	var saved: String = _load_saved_username()
	if saved != "":
		_username = saved
		_username_input.visible = false
		_menu_welcome_label.text = "Welcome back, %s" % saved
		_menu_welcome_label.visible = true

	# Set player names on shared HUD
	_shared_hud.set_player_names(_player_names)

	# Connect network signals
	NetworkManager.player_disconnected.connect(_on_net_player_disconnected)
	NetworkManager.connection_succeeded.connect(_on_net_connected)
	NetworkManager.connection_failed.connect(_on_net_connection_failed)
	# Server-authoritative signals
	NetworkManager.game_flow_event.connect(_on_game_flow_event)
	NetworkManager.world_state_received.connect(world_sync.on_world_state)
	NetworkManager.damage_event_received.connect(world_sync.on_damage_event)
	NetworkManager.zone_transfer_received.connect(_on_zone_transfer)
	NetworkManager.group_state_updated.connect(group_mgr.on_group_state)
	NetworkManager.group_invite_received.connect(group_mgr.on_group_invite)
	NetworkManager.group_error_received.connect(group_mgr.on_group_error)
	NetworkManager.character_state_received.connect(_on_character_state)
	NetworkManager.character_list_received.connect(_on_character_list)
	NetworkManager.character_error_received.connect(_on_character_error)

	# Dev mode: --dev CLI arg enables debug panel + auto-start server
	var user_args := OS.get_cmdline_user_args()
	if "--dev" in OS.get_cmdline_args() or "--dev" in user_args:
		dev_mode = true
		# Parse optional --class=X and --zone=X overrides
		for arg in user_args:
			if arg.begins_with("--class="):
				_dev_class = arg.split("=")[1]
			elif arg.begins_with("--zone="):
				_dev_zone = arg.split("=")[1]
		# Load editor config (CLI args take priority, already parsed above)
		_load_dev_config()
		var DebugPanelScript := preload("res://scenes/ui/debug_panel.gd")
		_debug_panel = DebugPanelScript.new()
		add_child(_debug_panel)
		print("[Main] Dev mode enabled — class=%s zone=%s" % [_dev_class, _dev_zone])

	if dev_mode:
		_dev_auto_start()
	else:
		_enter_menu()


func _input(event: InputEvent) -> void:
	if event.is_action_pressed("ui_cancel"):
		if state == GameState.FIGHT_OVER:
			return
		if (
			state == GameState.MENU
			or state == GameState.CHARACTER_SELECT
			or state == GameState.CREATE_CHARACTER
		):
			return
		_toggle_pause()
		get_viewport().set_input_as_handled()

	# Cursor mode: Alt (hold) + backtick/tilde (toggle)
	if (
		not paused
		and (
			state == GameState.FIGHT
			or state == GameState.FIGHT_OVER
			or state == GameState.HUB
			or state == GameState.ARENA_LOBBY
		)
	):
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

	# Debug panel toggle: Ctrl+D
	if _debug_panel and event is InputEventKey and event.pressed and not event.echo:
		if event.ctrl_pressed and event.keycode == KEY_D:
			_debug_panel.toggle()
			get_viewport().set_input_as_handled()

	# Full map toggle
	if not paused and state != GameState.MENU:
		if event is InputEventKey and event.pressed and not event.echo:
			if event.physical_keycode == KEY_M:
				if _map_overlay:
					var my_id: int = NetworkManager.get_my_id()
					if (
						my_id in entity_mgr.spawned_players
						and is_instance_valid(entity_mgr.spawned_players[my_id])
					):
						var player: CharacterBody3D = entity_mgr.spawned_players[my_id]
						_map_overlay._player_pos = player.global_position
						_map_overlay._player_rot_y = player.rotation.y
					_map_overlay.toggle()
					if _map_overlay.visible:
						# Scan collision geometry from the live scene
						_map_overlay.scan_environment(env_builder.current_env)
						_map_overlay._recompute_scale()
						if env_builder.portal_trail:
							if (
								my_id in entity_mgr.spawned_players
								and is_instance_valid(entity_mgr.spawned_players[my_id])
							):
								_map_overlay.set_waypoint_path(
									env_builder.portal_trail.get_path_to_target(
										entity_mgr.spawned_players[my_id].global_position
									)
								)
					get_viewport().set_input_as_handled()

	# Hub interactions
	if state == GameState.HUB and not paused:
		if event is InputEventKey and event.pressed:
			if event.physical_keycode == KEY_E:
				if hub_interact.near_lift:
					hub_interact.interact_lift()
				elif hub_interact.near_portal:
					NetworkManager.send_enter_portal()
				elif hub_interact.aimed_peer_id > 0:
					NetworkManager.send_group_invite(hub_interact.aimed_peer_id)
			elif event.physical_keycode == KEY_G:
				# Toggle group: create if not in group, leave if in group
				if group_mgr.group_data.get("group_id", 0) > 0:
					NetworkManager.send_group_leave()
				else:
					NetworkManager.send_group_create()

	# Arena exit portal interaction
	if state == GameState.FIGHT_OVER and not paused:
		if event is InputEventKey and event.pressed:
			if event.physical_keycode == KEY_E:
				if env_builder.is_near_exit_portal():
					NetworkManager.send_interact(2)  # InteractExitPortal


func _physics_process(_delta: float) -> void:
	if state == GameState.HUB:
		hub_interact.check_portal_proximity()
		hub_interact.check_lift_proximity()
		hub_interact.check_aim_at_player()
	elif state == GameState.FIGHT_OVER:
		env_builder.check_exit_portal_proximity()


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
	env_builder.unload_environment()
	if _enter_world_btn:
		_enter_world_btn.disabled = false
	# Show welcome or username input depending on saved state.
	var saved: String = _load_saved_username()
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
	var err: int = NetworkManager.connect_to_server(SERVER_ADDRESS)
	if err != OK:
		print("[Main] Failed to connect: %s" % error_string(err))
		return
	print("[Main] Connecting to %s:%d..." % [SERVER_ADDRESS, NetworkManager.DEFAULT_PORT])
	_menu_layer.visible = false


func _load_saved_username() -> String:
	if not FileAccess.file_exists(USERNAME_SAVE_PATH):
		return ""
	var f: FileAccess = FileAccess.open(USERNAME_SAVE_PATH, FileAccess.READ)
	if f == null:
		return ""
	var uname: String = f.get_as_text().strip_edges()
	f.close()
	return uname


func _save_username(uname: String) -> void:
	var f: FileAccess = FileAccess.open(USERNAME_SAVE_PATH, FileAccess.WRITE)
	if f == null:
		return
	f.store_string(uname)
	f.close()


# =============================================================================
# Dev Mode: auto-start server + connect
# =============================================================================


## Load .dev_config.json written by the debug_launcher editor plugin.
## Only sets values not already overridden by CLI args.
func _load_dev_config() -> void:
	var config_path: String = ProjectSettings.globalize_path("res://.dev_config.json")
	if not FileAccess.file_exists(config_path):
		return
	var f: FileAccess = FileAccess.open(config_path, FileAccess.READ)
	if f == null:
		return
	var json := JSON.new()
	if json.parse(f.get_as_text()) != OK:
		return
	var data: Dictionary = json.data
	# CLI args (already parsed) take priority — only apply config if still default
	var has_class_arg := false
	var has_zone_arg := false
	for arg in OS.get_cmdline_user_args():
		if arg.begins_with("--class="):
			has_class_arg = true
		elif arg.begins_with("--zone="):
			has_zone_arg = true
	if not has_class_arg and data.has("class"):
		_dev_class = data["class"]
	if not has_zone_arg and data.has("zone"):
		_dev_zone = data["zone"]


## Start the Go gateway server and connect automatically.
func _dev_auto_start() -> void:
	_menu_layer.visible = false
	_local_class = _dev_class

	# Start server subprocess.
	# Uses bash to cd into server dir, build (cached), then exec the binary
	# so the PID points directly to the gateway process.
	var client_dir: String = ProjectSettings.globalize_path("res://").rstrip("/")
	var project_root: String = client_dir.get_base_dir()
	var server_dir: String = project_root + "/server"
	OS.set_environment("CODEX_DEV", "1")
	OS.set_environment("GOPATH", project_root + "/.go")
	print("[Main] Starting dev server from %s..." % server_dir)
	_server_pid = (
		OS
		. create_process(
			"bash",
			[
				"-c",
				"cd '%s' && go build -o bin/gateway ./cmd/gateway && exec bin/gateway" % server_dir,
			]
		)
	)
	if _server_pid <= 0:
		push_error("[Main] Failed to start dev server — falling back to menu")
		_enter_menu()
		return
	print("[Main] Dev server started (PID %d)" % _server_pid)

	# Connect with retries (server needs a moment to build + bind)
	NetworkManager.username = "Dev"
	NetworkManager.dev_params = {"class": _dev_class, "zone": _dev_zone}
	await _dev_connect_with_retry()


## Retry connecting to the dev server until it's ready.
## Waits for zone_transfer_received signal which fires when dev auto-join completes.
func _dev_connect_with_retry() -> void:
	_dev_connected = false
	NetworkManager.zone_transfer_received.connect(_on_dev_zone_transfer, CONNECT_ONE_SHOT)

	var max_attempts: int = 40  # 40 * 0.5s = 20s max wait (build + start)
	for attempt in range(max_attempts):
		if _dev_connected:
			print("[Main] Dev server connected on attempt %d" % (attempt + 1))
			return
		# Only initiate a new connection if the previous attempt failed.
		if not NetworkManager.is_active:
			NetworkManager.connect_to_server("127.0.0.1")
		await get_tree().create_timer(0.5).timeout

	# Cleanup
	if NetworkManager.zone_transfer_received.is_connected(_on_dev_zone_transfer):
		NetworkManager.zone_transfer_received.disconnect(_on_dev_zone_transfer)
	push_error("[Main] Could not connect to dev server after %d attempts" % max_attempts)
	_stop_dev_server()
	_enter_menu()


func _on_dev_zone_transfer(_zone_type: int, _peer_id: int) -> void:
	_dev_connected = true


func _stop_dev_server() -> void:
	if _server_pid > 0:
		OS.kill(_server_pid)
		print("[Main] Dev server stopped (PID %d)" % _server_pid)
		_server_pid = -1


func _notification(what: int) -> void:
	if what == NOTIFICATION_WM_CLOSE_REQUEST:
		_stop_dev_server()


func _exit_tree() -> void:
	_stop_dev_server()


# =============================================================================
# Replay System
# =============================================================================


func _enter_replay_browser() -> void:
	state = GameState.REPLAY_BROWSER
	_menu_layer.visible = false

	var browser_script: GDScript = load("res://scripts/replay/replay_browser.gd")
	_replay_browser = CanvasLayer.new()
	_replay_browser.set_script(browser_script)
	add_child(_replay_browser)
	_replay_browser.replay_selected.connect(_on_replay_selected)
	_replay_browser.browser_closed.connect(_on_browser_closed)


func _on_replay_selected(replay: Variant) -> void:
	# Clean up browser
	if _replay_browser:
		_replay_browser.queue_free()
		_replay_browser = null

	state = GameState.REPLAY

	# Create replay scene
	var scene_script: GDScript = load("res://scripts/replay/replay_scene.gd")
	_replay_scene = Node3D.new()
	_replay_scene.set_script(scene_script)
	add_child(_replay_scene)
	_replay_scene.replay_exited.connect(_on_replay_exited)
	_replay_scene.start_replay(replay)


func _on_browser_closed() -> void:
	if _replay_browser:
		_replay_browser.queue_free()
		_replay_browser = null
	state = GameState.MENU
	_menu_layer.visible = true


func _on_replay_exited() -> void:
	if _replay_scene:
		_replay_scene.queue_free()
		_replay_scene = null
	state = GameState.MENU
	_menu_layer.visible = true
	Input.mouse_mode = Input.MOUSE_MODE_VISIBLE


# =============================================================================
# Character state callbacks
# =============================================================================


func _on_character_state(data: Dictionary) -> void:
	# Server confirmed character selection. Restore position and enter hub.
	_selected_char_id = data.get("char_id", 0)
	if data.class_name != "":
		_local_class = data.class_name
	if data.position != Vector3.ZERO:
		_saved_hub_position = data.position
		_saved_hub_rot_y = data.rot_y
		_has_saved_state = true
	print(
		(
			"[Main] Character confirmed: id=%d class=%s name=%s pos=%s"
			% [_selected_char_id, _local_class, data.get("char_name", ""), _saved_hub_position]
		)
	)


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
	print(
		(
			"[Main] Character list: %d characters, username=%s"
			% [data.characters.size(), _account_username]
		)
	)
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
	if peer_id in entity_mgr.spawned_players:
		var player: CharacterBody3D = entity_mgr.spawned_players[peer_id]
		if is_instance_valid(player):
			player.queue_free()
		entity_mgr.spawned_players.erase(peer_id)


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
	hub_interact.near_portal = false
	_portal_prompt.visible = false
	hub_interact.near_lift = false
	if _lift_prompt:
		_lift_prompt.visible = false

	# Load hub scene if not already loaded
	if env_builder.current_env == null or env_builder.current_env.name != "Hub":
		env_builder.unload_environment()
		env_builder.load_environment(HUB_SCENE)

	# Despawn existing players
	entity_mgr.despawn_all_players()

	# Spawn local player in hub (use saved position if returning player)
	var my_id: int = NetworkManager.get_my_id()
	if my_id > 0:
		var spawn_pos: Vector3 = HUB_SPAWNS[0]
		if _has_saved_state:
			spawn_pos = _saved_hub_position
			_has_saved_state = false
		entity_mgr.spawn_player(my_id, _local_class, spawn_pos)
		if _saved_hub_rot_y != 0.0:
			var player: CharacterBody3D = entity_mgr.spawned_players.get(my_id)
			if player:
				player.rotation.y = _saved_hub_rot_y
			_saved_hub_rot_y = 0.0

	hub_interact.update_hub_display()
	group_mgr.update_group_panel()
	if _shared_hud:
		_shared_hud.on_enter_hub()
	if _map_overlay:
		_map_overlay.reset_floor()
	env_builder.create_portal_trail()


# =============================================================================
# Arena warmup
# =============================================================================


func _enter_arena_warmup() -> void:
	state = GameState.ARENA_LOBBY
	get_tree().paused = false
	paused = false
	_pause_layer.visible = false
	_menu_layer.visible = false
	env_builder.remove_exit_portal()
	_hub_layer.visible = false
	if _shared_hud:
		_shared_hud.on_enter_arena()
	if _map_overlay:
		_map_overlay.set_floor("arena", "Arena")

	# Load arena scene if not already loaded
	if env_builder.current_env == null or env_builder.current_env.name != "Arena":
		env_builder.unload_environment()
		env_builder.load_environment(ARENA_SCENE)


func _select_class(class_name_str: String) -> void:
	_local_class = class_name_str
	if NetworkManager.is_active:
		NetworkManager.set_player_class(class_name_str)
	hub_interact.update_hub_display()


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
		entity_mgr.spawn_player(pid, class_name_str, spawn_pos)
		spawn_idx += 1

	Input.set_mouse_mode(Input.MOUSE_MODE_CAPTURED)


# =============================================================================
# Fight
# =============================================================================


func _start_fight() -> void:
	state = GameState.FIGHT
	_hub_layer.visible = false
	_cursor_toggled = false
	_alt_held = false

	# Enemies are managed dynamically via update_enemies from world state
	CombatLog.start_fight()
	if _shared_hud:
		_shared_hud.on_fight_start()


func _on_boss_dead() -> void:
	state = GameState.FIGHT_OVER
	env_builder.open_boss_gate()
	env_builder.spawn_exit_portal()
	if _local_player_dead and _death_overlay_layer.visible:
		_respawn_btn.disabled = false
	CombatLog.end_fight("VICTORY")
	if _shared_hud:
		_shared_hud.on_fight_end()


func _on_all_dead() -> void:
	state = GameState.FIGHT_OVER
	env_builder.open_boss_gate()
	if _local_player_dead and _death_overlay_layer.visible:
		_respawn_btn.disabled = false
	CombatLog.end_fight("WIPE")
	if _shared_hud:
		_shared_hud.on_fight_end()


# =============================================================================
# Zone transfer
# =============================================================================


func _on_zone_transfer(zone_type: int, _new_peer_id: int) -> void:
	print("[Main] Zone transfer: type=%d, new_peer=%d" % [zone_type, _new_peer_id])
	_hide_death_overlay()
	env_builder.remove_exit_portal()
	entity_mgr.despawn_all_players()
	entity_mgr.clear_all_enemies()
	if _map_overlay:
		_map_overlay.visible = false
	entity_mgr.clear_all_npcs()

	if zone_type == NetSerializer.ZONE_TYPE_ARENA:
		env_builder.unload_environment()
		env_builder.load_environment(ARENA_SCENE)
		# Spawn local player in warmup room immediately
		state = GameState.ARENA_LOBBY
		_hub_layer.visible = false
		_menu_layer.visible = false
		if _shared_hud:
			_shared_hud.on_enter_arena()
		Input.set_mouse_mode(Input.MOUSE_MODE_CAPTURED)
		var my_id: int = NetworkManager.get_my_id()
		if my_id > 0:
			entity_mgr.spawn_player(my_id, _local_class, LOBBY_SPAWN)
	else:
		env_builder.unload_environment()
		env_builder.load_environment(HUB_SCENE)
		_enter_hub()


# =============================================================================
# Server-authoritative signal handlers
# =============================================================================


func _on_game_flow_event(flow_type: int, _text: String) -> void:
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
			entity_mgr.clear_all_enemies()
			entity_mgr.clear_all_npcs()
			_enter_arena_warmup()
		NetSerializer.FLOW_BOSS_ACTIVATED:
			env_builder.close_boss_gate()
		NetSerializer.FLOW_BOSS_RESET:
			env_builder.open_boss_gate()


# =============================================================================
# Pause
# =============================================================================


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
	var want_cursor: bool = _cursor_toggled or _alt_held
	if want_cursor:
		Input.set_mouse_mode(Input.MOUSE_MODE_VISIBLE)
	else:
		Input.set_mouse_mode(Input.MOUSE_MODE_CAPTURED)


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
	ui_ctrl.populate_char_select()


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
	var char_name: String = _char_name_input.text.strip_edges()
	if char_name.length() < 2 or char_name.length() > 20:
		_char_create_error_label.text = "Name must be 2-20 characters."
		_char_create_error_label.visible = true
		return
	_char_create_error_label.visible = false
	_char_create_btn.disabled = true
	NetworkManager.send_create_character(_local_class, char_name)


# =============================================================================
# Death overlay
# =============================================================================


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
