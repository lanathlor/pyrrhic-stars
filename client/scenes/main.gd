extends Node3D

## Game flow: Menu → Lobby → Fight → Result → Lobby.
## Supports solo play and local ENet multiplayer.
## Solo: single player, no networking.
## Host/Join: ENet multiplayer, dynamic player spawning.

enum GameState { MENU, LOBBY, FIGHT, RESULT }

var state: GameState = GameState.MENU
var paused: bool = false

const LOBBY_SPAWN := Vector3(0.0, 0.1, 20.0)
const ENEMY_SPAWN := Vector3(0.0, 0.1, 0.0)
const PLAYER_SPAWNS := [
	Vector3(-2.0, 0.1, 20.0),
	Vector3(0.0, 0.1, 20.0),
	Vector3(2.0, 0.1, 20.0),
	Vector3(-1.0, 0.1, 21.0),
]

const CLASS_SCENES := {
	"gunner": "res://scenes/controllers/gunner/gunner.tscn",
	"vanguard": "res://scenes/controllers/vanguard/vanguard.tscn",
	"blade_dancer": "res://scenes/controllers/blade_dancer/blade_dancer.tscn",
}

@onready var enemy: CharacterBody3D = $BasicEnemy

var _spawned_players: Dictionary = {}  # peer_id -> CharacterBody3D
var _local_class: String = "gunner"
var _is_solo: bool = false
var _players_node: Node3D

# Dynamic nodes
var _gate: CSGBox3D
var _trigger: Area3D
var _pause_layer: CanvasLayer
var _result_layer: CanvasLayer
var _result_label: Label
var _menu_layer: CanvasLayer
var _lobby_layer: CanvasLayer
var _lobby_status_label: Label
var _lobby_players_label: Label
var _lobby_class_label: Label
var _address_input: LineEdit


func _ready() -> void:
	process_mode = Node.PROCESS_MODE_ALWAYS

	# Free pre-placed player nodes — we use dynamic spawning
	for node_name in ["Gunner", "Vanguard", "BladeDancer"]:
		var node := get_node_or_null(node_name)
		if node:
			node.queue_free()

	# Create players container
	_players_node = Node3D.new()
	_players_node.name = "Players"
	add_child(_players_node)

	_create_gate()
	_create_trigger()
	_create_pause_menu()
	_create_result_overlay()
	_create_menu_ui()
	_create_lobby_ui()
	_bake_navigation()

	# Hide enemy until fight
	enemy.visible = false
	enemy.collision_layer = 0
	enemy.set_physics_process(false)
	enemy.died.connect(_on_enemy_died)

	# Connect network signals
	NetworkManager.player_connected.connect(_on_net_player_connected)
	NetworkManager.player_disconnected.connect(_on_net_player_disconnected)
	NetworkManager.connection_succeeded.connect(_on_net_connected)
	NetworkManager.connection_failed.connect(_on_net_connection_failed)
	NetworkManager.all_players_ready.connect(_on_all_players_ready)
	NetworkManager.player_info_changed.connect(_update_lobby_display)

	_enter_menu()


func _input(event: InputEvent) -> void:
	if event.is_action_pressed("ui_cancel"):
		if state == GameState.RESULT:
			return
		if state == GameState.MENU:
			return
		_toggle_pause()
		get_viewport().set_input_as_handled()

	# Class selection in lobby
	if state == GameState.LOBBY and not paused:
		if event is InputEventKey and event.pressed:
			if event.physical_keycode == KEY_1:
				_select_class("gunner")
			elif event.physical_keycode == KEY_2:
				_select_class("vanguard")
			elif event.physical_keycode == KEY_3:
				_select_class("blade_dancer")
			elif event.physical_keycode == KEY_ENTER or event.physical_keycode == KEY_KP_ENTER:
				_toggle_ready()


# =============================================================================
# Menu
# =============================================================================

func _enter_menu() -> void:
	state = GameState.MENU
	_is_solo = false
	NetworkManager.disconnect_game()
	_menu_layer.visible = true
	_lobby_layer.visible = false
	_result_layer.visible = false
	_pause_layer.visible = false
	Input.set_mouse_mode(Input.MOUSE_MODE_VISIBLE)


func _on_solo_pressed() -> void:
	_is_solo = true
	_menu_layer.visible = false
	_enter_lobby()


func _on_host_pressed() -> void:
	_is_solo = false
	NetworkManager.disconnect_game()
	var err := NetworkManager.host_game()
	if err != OK:
		print("[Main] Failed to host: %s" % error_string(err))
		return
	print("[Main] Hosting on port %d" % NetworkManager.PORT)
	_menu_layer.visible = false
	_enter_lobby()


func _on_join_pressed() -> void:
	_is_solo = false
	NetworkManager.disconnect_game()
	var address := _address_input.text.strip_edges()
	if address == "":
		address = "127.0.0.1"
	var err := NetworkManager.join_game(address)
	if err != OK:
		print("[Main] Failed to join: %s" % error_string(err))
		return
	print("[Main] Connecting to %s:%d..." % [address, NetworkManager.PORT])
	_menu_layer.visible = false
	_lobby_layer.visible = true
	_lobby_status_label.text = "Connecting..."


func _on_net_connected() -> void:
	print("[Main] Connected as peer %d" % NetworkManager.get_my_id())
	_enter_lobby()


func _on_net_connection_failed() -> void:
	print("[Main] Connection failed")
	_enter_menu()


func _on_net_player_connected(peer_id: int) -> void:
	print("[Main] Peer %d connected" % peer_id)
	_update_lobby_display()


func _on_net_player_disconnected(peer_id: int) -> void:
	print("[Main] Peer %d disconnected" % peer_id)
	# Remove their player if spawned
	if peer_id in _spawned_players:
		var player: CharacterBody3D = _spawned_players[peer_id]
		if is_instance_valid(player):
			player.queue_free()
		_spawned_players.erase(peer_id)
	_update_lobby_display()


# =============================================================================
# Lobby
# =============================================================================

func _enter_lobby() -> void:
	state = GameState.LOBBY
	get_tree().paused = false
	paused = false
	_pause_layer.visible = false
	_result_layer.visible = false
	_menu_layer.visible = false
	_lobby_layer.visible = true
	Input.set_mouse_mode(Input.MOUSE_MODE_VISIBLE)

	# Despawn any existing players
	for pid in _spawned_players:
		var player = _spawned_players[pid]
		if is_instance_valid(player):
			player.queue_free()
	_spawned_players.clear()

	# Hide enemy
	enemy.visible = false
	enemy.collision_layer = 0
	enemy.set_physics_process(false)

	# Open gate
	_gate.visible = false
	_gate.use_collision = false
	_trigger.monitoring = true

	# Reset ready states
	if NetworkManager.is_active:
		NetworkManager.reset_ready_states()

	_update_lobby_display()


func _select_class(class_name_str: String) -> void:
	_local_class = class_name_str
	if NetworkManager.is_active:
		NetworkManager.set_player_class.rpc(class_name_str)
	_update_lobby_display()


func _toggle_ready() -> void:
	if _is_solo:
		# Solo: spawn and go directly
		_spawn_solo_player()
		return

	if not NetworkManager.is_active:
		return

	var my_id := NetworkManager.get_my_id()
	if my_id not in NetworkManager.player_info:
		return
	var currently_ready: bool = NetworkManager.player_info[my_id]["ready"]
	NetworkManager.set_player_ready.rpc(not currently_ready)


func _on_all_players_ready() -> void:
	# Host spawns all players in the lobby
	if NetworkManager.is_host:
		_spawn_multiplayer_players.rpc()


func _update_lobby_display() -> void:
	if not _lobby_layer or not _lobby_layer.visible:
		return

	_lobby_class_label.text = "[1] Gunner   [2] Vanguard   [3] Blade Dancer\nSelected: %s" % _local_class.to_upper()

	if _is_solo:
		_lobby_status_label.text = "SOLO MODE\nPress ENTER to start"
		_lobby_players_label.text = ""
		return

	if not NetworkManager.is_active:
		_lobby_status_label.text = "Not connected"
		_lobby_players_label.text = ""
		return

	var text := ""
	if NetworkManager.is_host:
		var ip := _get_lan_ip()
		text += "LAN Address: %s:%d\n\n" % [ip, NetworkManager.PORT]
	text += "Players:\n"
	for pid in NetworkManager.player_info:
		var info: Dictionary = NetworkManager.player_info[pid]
		var ready_str := " [READY]" if info["ready"] else ""
		var you_str := " (you)" if pid == NetworkManager.get_my_id() else ""
		text += "  Peer %d: %s%s%s\n" % [pid, info["class_name"].to_upper(), ready_str, you_str]

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

func _spawn_solo_player() -> void:
	var scene := load(CLASS_SCENES[_local_class]) as PackedScene
	var player := scene.instantiate() as CharacterBody3D
	player.name = "Player_solo"
	_players_node.add_child(player)
	player.global_position = LOBBY_SPAWN
	_spawned_players[1] = player
	player.died.connect(_on_player_died)

	_lobby_layer.visible = false
	Input.set_mouse_mode(Input.MOUSE_MODE_CAPTURED)

	# Enable trigger for solo
	_trigger.monitoring = true
	state = GameState.LOBBY  # still in lobby, waiting to enter arena


@rpc("authority", "call_local", "reliable")
func _spawn_multiplayer_players() -> void:
	_lobby_layer.visible = false

	# Spawn all players in the lobby room
	var spawn_idx := 0
	for pid in NetworkManager.player_info:
		var info: Dictionary = NetworkManager.player_info[pid]
		var class_name_str: String = info["class_name"]
		if not CLASS_SCENES.has(class_name_str):
			push_error("[Main] Unknown class: %s" % class_name_str)
			continue
		var scene := load(CLASS_SCENES[class_name_str]) as PackedScene
		var player := scene.instantiate() as CharacterBody3D
		player.name = "Player_%d" % pid
		var spawn_pos: Vector3 = PLAYER_SPAWNS[spawn_idx % PLAYER_SPAWNS.size()]
		# Authority MUST be set before add_child so _ready() sees the correct authority
		player.set_multiplayer_authority(pid)
		_players_node.add_child(player)
		player.global_position = spawn_pos
		_spawned_players[pid] = player
		player.died.connect(_on_player_died)
		spawn_idx += 1

	Input.set_mouse_mode(Input.MOUSE_MODE_CAPTURED)

	# Stay in lobby — trigger zone will start the fight when a player enters the arena
	_trigger.monitoring = true


# =============================================================================
# Fight
# =============================================================================

func _start_fight() -> void:
	state = GameState.FIGHT
	_trigger.set_deferred("monitoring", false)
	_lobby_layer.visible = false

	# Close gate behind the players
	_gate.visible = true
	_gate.use_collision = true

	# Enemy reset — only host manages enemy state
	var is_host := _is_solo or not NetworkManager.is_active or NetworkManager.is_host
	if is_host:
		enemy.global_position = ENEMY_SPAWN
		enemy.health = enemy.max_health
		enemy._current_phase = 1
		enemy._phase_transitioned.clear()
		enemy._last_attack = ""
		var fg_mat := enemy._health_bar_fg.get_surface_override_material(0) as StandardMaterial3D
		if fg_mat:
			fg_mat.albedo_color = Color(0.15, 0.85, 0.15)
			fg_mat.emission_enabled = false
		enemy._update_health_bar()
		enemy._change_state(enemy.State.CHASE)

	# All peers: make enemy visible and enable physics
	enemy.visible = true
	enemy.collision_layer = 4
	enemy.set_physics_process(true)
	CombatLog.start_fight()


func _on_trigger_entered(body: Node3D) -> void:
	if state != GameState.LOBBY:
		return
	if body not in _spawned_players.values():
		return
	# Any player entering the arena triggers the fight
	if _is_solo:
		_start_fight()
	elif NetworkManager.is_active and NetworkManager.is_host:
		_start_fight_rpc.rpc()


@rpc("authority", "call_local", "reliable")
func _start_fight_rpc() -> void:
	_start_fight()


func _on_player_died() -> void:
	if state != GameState.FIGHT:
		return
	# Check if ALL players are dead
	var all_dead := true
	for pid in _spawned_players:
		var player: CharacterBody3D = _spawned_players[pid]
		if is_instance_valid(player) and player.health > 0.0:
			all_dead = false
			break
	if all_dead:
		CombatLog.end_fight("PLAYER_DIED")
		if _is_solo or (NetworkManager.is_active and NetworkManager.is_host):
			_show_result_rpc.rpc("YOU DIED", Color(0.8, 0.15, 0.15)) if NetworkManager.is_active else _show_result("YOU DIED", Color(0.8, 0.15, 0.15))


func _on_enemy_died() -> void:
	if state != GameState.FIGHT:
		return
	CombatLog.end_fight("VICTORY")
	if _is_solo or (NetworkManager.is_active and NetworkManager.is_host):
		_show_result_rpc.rpc("VICTORY", Color(0.15, 0.8, 0.3)) if NetworkManager.is_active else _show_result("VICTORY", Color(0.15, 0.8, 0.3))


@rpc("authority", "call_local", "reliable")
func _show_result_rpc(text: String, color: Color) -> void:
	_show_result(text, color)


func _show_result(text: String, color: Color) -> void:
	if state == GameState.RESULT:
		return  # Prevent double-trigger
	state = GameState.RESULT
	_result_label.text = text
	_result_label.add_theme_color_override("font_color", color)
	_result_layer.visible = true
	Input.set_mouse_mode(Input.MOUSE_MODE_VISIBLE)
	get_tree().create_timer(3.0).timeout.connect(_on_result_timeout, CONNECT_ONE_SHOT)


func _on_result_timeout() -> void:
	_enter_lobby()


func _toggle_pause() -> void:
	paused = not paused
	get_tree().paused = paused
	_pause_layer.visible = paused
	if paused:
		Input.set_mouse_mode(Input.MOUSE_MODE_VISIBLE)
	else:
		Input.set_mouse_mode(Input.MOUSE_MODE_CAPTURED)


# =============================================================================
# UI builders
# =============================================================================

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


func _create_trigger() -> void:
	_trigger = Area3D.new()
	_trigger.transform.origin = Vector3(0.0, 1.5, 11.0)
	_trigger.collision_layer = 0
	_trigger.collision_mask = 2  # detect players
	var shape := CollisionShape3D.new()
	var box := BoxShape3D.new()
	box.size = Vector3(5.0, 3.0, 1.0)
	shape.shape = box
	_trigger.add_child(shape)
	_trigger.body_entered.connect(_on_trigger_entered)
	add_child(_trigger)


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
		# Despawn players
		for pid in _spawned_players:
			var p = _spawned_players[pid]
			if is_instance_valid(p):
				p.queue_free()
		_spawned_players.clear()
		_enter_menu()
	)
	vbox.add_child(menu_btn)

	var quit_btn := Button.new()
	quit_btn.text = "Quit"
	quit_btn.pressed.connect(func(): get_tree().quit())
	vbox.add_child(quit_btn)


func _create_result_overlay() -> void:
	_result_layer = CanvasLayer.new()
	_result_layer.layer = 19
	_result_layer.visible = false
	add_child(_result_layer)

	var bg := ColorRect.new()
	bg.color = Color(0.0, 0.0, 0.0, 0.5)
	bg.anchor_right = 1.0
	bg.anchor_bottom = 1.0
	bg.mouse_filter = Control.MOUSE_FILTER_IGNORE
	_result_layer.add_child(bg)

	_result_label = Label.new()
	_result_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	_result_label.vertical_alignment = VERTICAL_ALIGNMENT_CENTER
	_result_label.anchor_left = 0.0
	_result_label.anchor_right = 1.0
	_result_label.anchor_top = 0.3
	_result_label.anchor_bottom = 0.5
	_result_label.add_theme_font_size_override("font_size", 64)
	_result_layer.add_child(_result_label)

	var sub := Label.new()
	sub.text = "Returning to lobby..."
	sub.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	sub.anchor_left = 0.0
	sub.anchor_right = 1.0
	sub.anchor_top = 0.5
	sub.anchor_bottom = 0.6
	sub.add_theme_font_size_override("font_size", 20)
	sub.add_theme_color_override("font_color", Color(0.7, 0.7, 0.7))
	_result_layer.add_child(sub)


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
	subtitle.text = "Phase 0 — Local Co-op"
	subtitle.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	subtitle.add_theme_font_size_override("font_size", 18)
	subtitle.add_theme_color_override("font_color", Color(0.5, 0.5, 0.6))
	vbox.add_child(subtitle)

	var spacer := Control.new()
	spacer.custom_minimum_size.y = 30.0
	vbox.add_child(spacer)

	var solo_btn := Button.new()
	solo_btn.text = "Solo"
	solo_btn.custom_minimum_size.y = 50.0
	solo_btn.pressed.connect(_on_solo_pressed)
	vbox.add_child(solo_btn)

	var host_btn := Button.new()
	host_btn.text = "Host Game"
	host_btn.custom_minimum_size.y = 50.0
	host_btn.pressed.connect(_on_host_pressed)
	vbox.add_child(host_btn)

	var join_hbox := HBoxContainer.new()
	join_hbox.add_theme_constant_override("separation", 8)
	vbox.add_child(join_hbox)

	_address_input = LineEdit.new()
	_address_input.placeholder_text = "127.0.0.1"
	_address_input.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	_address_input.custom_minimum_size.y = 50.0
	join_hbox.add_child(_address_input)

	var join_btn := Button.new()
	join_btn.text = "Join"
	join_btn.custom_minimum_size = Vector2(80.0, 50.0)
	join_btn.pressed.connect(_on_join_pressed)
	join_hbox.add_child(join_btn)


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
	title.text = "LOBBY"
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


# =============================================================================
# Navigation
# =============================================================================

func _bake_navigation() -> void:
	var nav_region := NavigationRegion3D.new()
	var nav_mesh := NavigationMesh.new()
	nav_mesh.agent_radius = 0.8
	nav_mesh.agent_height = 2.0
	nav_mesh.cell_size = 0.25
	nav_mesh.cell_height = 0.25
	nav_region.navigation_mesh = nav_mesh
	add_child(nav_region)

	var source_geo := NavigationMeshSourceGeometryData3D.new()

	# Arena floor: 40x30 centered at origin
	source_geo.add_faces(PackedVector3Array([
		Vector3(-20, 0, -15), Vector3(20, 0, -15), Vector3(20, 0, 15),
		Vector3(-20, 0, -15), Vector3(20, 0, 15), Vector3(-20, 0, 15),
	]), Transform3D.IDENTITY)

	# Lobby floor
	source_geo.add_faces(PackedVector3Array([
		Vector3(-5, 0, 15), Vector3(5, 0, 15), Vector3(5, 0, 25),
		Vector3(-5, 0, 15), Vector3(5, 0, 25), Vector3(-5, 0, 25),
	]), Transform3D.IDENTITY)

	# Carve out obstacles using projected obstructions
	for pos in [Vector3(-8,0,-6), Vector3(8,0,-6), Vector3(-8,0,6), Vector3(8,0,6), Vector3(0,0,-10), Vector3(0,0,10)]:
		_add_obstruction(source_geo, pos, Vector2(1.5, 1.5), 4.0)
	_add_obstruction(source_geo, Vector3(-5, 0, -2), Vector2(3.0, 1.0), 1.2)
	_add_obstruction(source_geo, Vector3(5, 0, 2), Vector2(3.0, 1.0), 1.2)
	_add_obstruction(source_geo, Vector3(-12, 0, 0), Vector2(1.0, 3.0), 1.2)
	_add_obstruction(source_geo, Vector3(12, 0, 0), Vector2(1.0, 3.0), 1.2)
	_add_obstruction(source_geo, Vector3(0, 0, -15), Vector2(40.0, 0.5), 5.0)
	_add_obstruction(source_geo, Vector3(20, 0, 0), Vector2(0.5, 30.0), 5.0)
	_add_obstruction(source_geo, Vector3(-20, 0, 0), Vector2(0.5, 30.0), 5.0)
	_add_obstruction(source_geo, Vector3(-11.25, 0, 15), Vector2(17.5, 0.5), 5.0)
	_add_obstruction(source_geo, Vector3(11.25, 0, 15), Vector2(17.5, 0.5), 5.0)
	_add_obstruction(source_geo, Vector3(-5, 0, 20), Vector2(0.5, 10.0), 5.0)
	_add_obstruction(source_geo, Vector3(5, 0, 20), Vector2(0.5, 10.0), 5.0)
	_add_obstruction(source_geo, Vector3(0, 0, 25), Vector2(10.0, 0.5), 5.0)

	NavigationServer3D.bake_from_source_geometry_data(nav_mesh, source_geo)

	var verts := nav_mesh.get_vertices()
	var polys := nav_mesh.get_polygon_count()
	print("[Main] NavMesh baked: %d vertices, %d polygons" % [verts.size(), polys])


func _add_obstruction(source_geo: NavigationMeshSourceGeometryData3D, center: Vector3, size: Vector2, height: float) -> void:
	var hx := size.x / 2.0
	var hz := size.y / 2.0
	var outline := PackedVector3Array([
		Vector3(center.x - hx, 0, center.z - hz),
		Vector3(center.x + hx, 0, center.z - hz),
		Vector3(center.x + hx, 0, center.z + hz),
		Vector3(center.x - hx, 0, center.z + hz),
	])
	source_geo.add_projected_obstruction(outline, 0.0, height, true)


func _get_lan_ip() -> String:
	var fallback := "127.0.0.1"
	# Prefer real LAN (192.168.x, 10.x) over Docker/bridge (172.x)
	for prefix in ["192.168.", "10."]:
		for addr in IP.get_local_addresses():
			if addr.begins_with(prefix):
				return addr
	# Fall back to 172.16-31.x (could be bridge/Docker but better than nothing)
	for addr in IP.get_local_addresses():
		if addr.begins_with("172.") and ":" not in addr:
			return addr
	return fallback
