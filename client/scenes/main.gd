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
	"arcanotechnicien": "res://scenes/controllers/arcanotechnicien/arcanotechnicien.tscn",
}
const CLASS_INFO := SpecData.CLASS_INFO
const SPEC_INFO := SpecData.SPEC_INFO
const DEFAULT_SPECS := SpecData.DEFAULT_SPECS
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
var _local_class: String = "gunner":
	set(value):
		_local_class = value
		if is_inside_tree():
			InventoryManager.current_class = value
		_local_spec = DEFAULT_SPECS.get(value, "")
var _local_spec: String = "assault"
var _saved_hub_position: Vector3 = Vector3.ZERO
var _saved_hub_rot_y: float = 0.0
var _has_saved_state: bool = false
var _username: String = ""
var _player_names: Dictionary = {}  # peer_id -> username
var _cursor_toggled: bool = false  # backtick toggle state
var _alt_held: bool = false  # alt hold state
var _local_player_dead: bool = false
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
@onready var game_flow: Node = $GameFlowManager
@onready var dev_mgr: Node = $DevModeManager
@onready var replay_mgr: Node = $ReplayManager
@onready var char_mgr: Node = $CharacterManager
# UI scenes (static instances in main.tscn)
@onready var _pause_layer: CanvasLayer = $PauseMenu
@onready var _menu_layer: CanvasLayer = $MenuUI
@onready var _char_select_layer: CanvasLayer = $CharSelectUI
@onready var _char_create_layer: CanvasLayer = $CharCreateUI
@onready var _hub_layer: CanvasLayer = $HubHUD
@onready var _invite_popup: CanvasLayer = $InvitePopup
@onready var _shared_hud_layer: CanvasLayer = $SharedHUD
@onready var _death_overlay_layer: CanvasLayer = $DeathOverlay
@onready var _inventory_layer: CanvasLayer = $InventoryUI
@onready var _spec_panel: CanvasLayer = $SpecPanel


func _ready() -> void:
	process_mode = Node.PROCESS_MODE_ALWAYS
	InventoryManager.current_class = _local_class

	_init_ui_references()
	_connect_ui_signals()

	# Load saved username
	var saved: String = char_mgr.load_saved_username()
	if saved != "":
		_username = saved
		_username_input.visible = false
		_menu_welcome_label.text = "Welcome back, %s" % saved
		_menu_welcome_label.visible = true

	_shared_hud.set_player_names(_player_names)
	_connect_network_signals()

	# Dev mode: --dev CLI arg enables debug panel + auto-start server
	var user_args := OS.get_cmdline_user_args()
	if "--dev" in OS.get_cmdline_args() or "--dev" in user_args:
		dev_mode = true
		dev_mgr.initialize(user_args)

	if dev_mode:
		dev_mgr.dev_auto_start()
	else:
		_enter_menu()


func _init_ui_references() -> void:
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


func _connect_ui_signals() -> void:
	_pause_layer.resume_btn.pressed.connect(_toggle_pause)
	_pause_layer.menu_btn.pressed.connect(
		func():
			get_tree().paused = false
			paused = false
			entity_mgr.despawn_all_players()
			_enter_menu()
	)
	_pause_layer.quit_btn.pressed.connect(func(): get_tree().quit())
	_menu_layer.play_btn.pressed.connect(char_mgr.on_connect_pressed)
	_menu_layer.replays_btn.pressed.connect(replay_mgr.enter_replay_browser)
	_char_select_layer.back_btn.pressed.connect(
		func():
			NetworkManager.disconnect_game()
			_enter_menu()
	)
	_char_select_layer.create_btn.pressed.connect(char_mgr.enter_create_character)
	_enter_world_btn.pressed.connect(char_mgr.on_enter_world_pressed)
	_char_create_layer.back_btn.pressed.connect(
		func():
			_char_create_layer.visible = false
			char_mgr.enter_character_select()
	)
	_char_create_btn.pressed.connect(char_mgr.on_create_character_pressed)
	_group_leave_btn.pressed.connect(func(): NetworkManager.send_group_leave())
	_invite_popup.accept_btn.pressed.connect(group_mgr.accept_invite)
	_invite_popup.decline_btn.pressed.connect(group_mgr.decline_invite)
	_respawn_btn.pressed.connect(game_flow.on_respawn)
	_respawn_hub_btn.pressed.connect(game_flow.on_respawn_hub)
	_inventory_layer.toolbar_panel.spec_pressed.connect(char_mgr.toggle_spec_panel)
	_inventory_layer.toolbar_panel.equip_pressed.connect(_toggle_equip_panel)
	_inventory_layer.toolbar_panel.bag_pressed.connect(_toggle_bag_panel)
	_spec_panel.spec_selected.connect(char_mgr.on_spec_selected)
	_spec_panel.closed.connect(_update_cursor_mode)
	_spec_panel.closed.connect(_sync_toolbar_active)


func _connect_network_signals() -> void:
	NetworkManager.player_disconnected.connect(char_mgr.on_net_player_disconnected)
	NetworkManager.connection_succeeded.connect(char_mgr.on_net_connected)
	NetworkManager.connection_failed.connect(char_mgr.on_net_connection_failed)
	NetworkManager.game_flow_event.connect(game_flow.on_game_flow_event)
	NetworkManager.world_state_received.connect(world_sync.on_world_state)
	NetworkManager.damage_event_received.connect(world_sync.on_damage_event)
	NetworkManager.zone_transfer_received.connect(game_flow.on_zone_transfer)
	NetworkManager.group_state_updated.connect(group_mgr.on_group_state)
	NetworkManager.group_invite_received.connect(group_mgr.on_group_invite)
	NetworkManager.group_error_received.connect(group_mgr.on_group_error)
	NetworkManager.character_state_received.connect(char_mgr.on_character_state)
	NetworkManager.character_list_received.connect(char_mgr.on_character_list)
	NetworkManager.character_error_received.connect(char_mgr.on_character_error)


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

	_handle_cursor_input(event)
	_handle_debug_input(event)
	_handle_menu_input(event)
	_handle_gameplay_input(event)


func _handle_cursor_input(event: InputEvent) -> void:
	if paused:
		return
	if state not in [GameState.FIGHT, GameState.FIGHT_OVER, GameState.HUB, GameState.ARENA_LOBBY]:
		return
	if not event is InputEventKey:
		return
	if event.physical_keycode == KEY_QUOTELEFT and event.pressed and not event.echo:
		_cursor_toggled = not _cursor_toggled
		_update_cursor_mode()
		get_viewport().set_input_as_handled()
	elif event.keycode == KEY_ALT:
		_alt_held = event.pressed
		_update_cursor_mode()


func _handle_debug_input(event: InputEvent) -> void:
	if not dev_mgr.debug_panel:
		return
	if event is InputEventKey and event.pressed and not event.echo:
		if event.ctrl_pressed and event.keycode == KEY_D:
			dev_mgr.toggle_debug_panel()
			get_viewport().set_input_as_handled()


func _handle_menu_input(event: InputEvent) -> void:
	if paused or state == GameState.MENU:
		return
	if not (event is InputEventKey and event.pressed and not event.echo):
		return

	# Full map toggle
	if event.physical_keycode == KEY_M and _map_overlay:
		_toggle_map_overlay()
		get_viewport().set_input_as_handled()
		return

	# Inventory: I = equipment, B = bag, N = spec
	if state in [GameState.CHARACTER_SELECT, GameState.CREATE_CHARACTER]:
		return
	if event.physical_keycode == KEY_I:
		_toggle_equip_panel()
		get_viewport().set_input_as_handled()
	elif event.physical_keycode == KEY_B:
		_toggle_bag_panel()
		get_viewport().set_input_as_handled()
	elif event.physical_keycode == KEY_N:
		char_mgr.toggle_spec_panel()
		get_viewport().set_input_as_handled()


func _toggle_map_overlay() -> void:
	var my_id: int = NetworkManager.get_my_id()
	if my_id in entity_mgr.spawned_players and is_instance_valid(entity_mgr.spawned_players[my_id]):
		var player: CharacterBody3D = entity_mgr.spawned_players[my_id]
		_map_overlay._player_pos = player.global_position
		_map_overlay._player_rot_y = player.rotation.y
	_map_overlay.toggle()
	if _map_overlay.visible:
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


func _handle_gameplay_input(event: InputEvent) -> void:
	if paused:
		return
	if state not in [GameState.HUB, GameState.ARENA_LOBBY, GameState.FIGHT, GameState.FIGHT_OVER]:
		return
	if not (event is InputEventKey and event.pressed):
		return

	if event.physical_keycode == KEY_E:
		if hub_interact.near_lift:
			hub_interact.interact_lift()
		elif hub_interact.near_portal:
			NetworkManager.send_enter_portal()
		elif hub_interact.aimed_peer_id > 0:
			NetworkManager.send_group_invite(hub_interact.aimed_peer_id)
		elif state == GameState.FIGHT_OVER and env_builder.is_near_exit_portal():
			NetworkManager.send_interact(2)  # InteractExitPortal
	elif event.physical_keycode == KEY_G and event.ctrl_pressed:
		if dev_mode:
			dev_mgr.toggle_bot_panel()
			get_viewport().set_input_as_handled()
	elif event.physical_keycode == KEY_G and not event.ctrl_pressed:
		if group_mgr.group_data.get("group_id", 0) > 0:
			NetworkManager.send_group_leave()
		else:
			NetworkManager.send_group_create()


func _physics_process(_delta: float) -> void:
	if state in [GameState.HUB, GameState.ARENA_LOBBY, GameState.FIGHT, GameState.FIGHT_OVER]:
		hub_interact.check_portal_proximity()
		if state == GameState.HUB:
			hub_interact.check_lift_proximity()
			hub_interact.check_aim_at_player()
		elif state == GameState.FIGHT_OVER:
			env_builder.check_exit_portal_proximity()


# =============================================================================
# Menu / Hub (delegated to game_flow)
# =============================================================================


func _enter_menu() -> void:
	game_flow.enter_menu()


func _show_portal_prompt_only() -> void:
	game_flow.show_portal_prompt_only()


func _enter_hub() -> void:
	game_flow.enter_hub()


# =============================================================================
# Pause / cursor / inventory
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
		if not _is_cursor_always_visible_class():
			Input.set_mouse_mode(Input.MOUSE_MODE_CAPTURED)
		else:
			Input.set_mouse_mode(Input.MOUSE_MODE_VISIBLE)


## Returns true for classes that always keep a visible cursor (WoW-style input).
func _is_cursor_always_visible_class() -> bool:
	return _local_class == "arcanotechnicien"


## Show/hide cursor for UI interaction without pausing.
## Active when Alt is held or backtick is toggled on.
func _update_cursor_mode() -> void:
	if _is_cursor_always_visible_class():
		Input.set_mouse_mode(Input.MOUSE_MODE_VISIBLE)
		return
	var inv_open: bool = _inventory_layer.equip_panel.visible or _inventory_layer.bag_panel.visible
	var spec_open: bool = _spec_panel.visible
	var bot_open: bool = dev_mgr.bot_panel != null and dev_mgr.bot_panel.visible
	var want_cursor: bool = _cursor_toggled or _alt_held or inv_open or spec_open or bot_open
	if want_cursor:
		Input.set_mouse_mode(Input.MOUSE_MODE_VISIBLE)
	else:
		Input.set_mouse_mode(Input.MOUSE_MODE_CAPTURED)


func _toggle_equip_panel() -> void:
	_inventory_layer.equip_panel.toggle()
	_update_cursor_mode()
	_sync_toolbar_active()


func _toggle_bag_panel() -> void:
	_inventory_layer.bag_panel.toggle()
	_update_cursor_mode()
	_sync_toolbar_active()


func _sync_toolbar_active() -> void:
	(
		_inventory_layer
		. toolbar_panel
		. update_active_state(
			_spec_panel.visible,
			_inventory_layer.equip_panel.visible,
			_inventory_layer.bag_panel.visible,
		)
	)


# =============================================================================
# Delegate wrappers (called externally via ctrl.XXX)
# =============================================================================


func _on_local_player_died() -> void:
	game_flow.on_local_player_died()


func _select_character_row(char_id: int, class_name_str: String) -> void:
	char_mgr.select_character_row(char_id, class_name_str)


func _select_create_class(cls: String) -> void:
	char_mgr.select_create_class(cls)


# =============================================================================
# Lifecycle
# =============================================================================


func _notification(what: int) -> void:
	if what == NOTIFICATION_WM_CLOSE_REQUEST:
		dev_mgr.stop_dev_server()


func _exit_tree() -> void:
	dev_mgr.stop_dev_server()
