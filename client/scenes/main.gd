extends Node3D

## Game flow: Lobby → Fight → Result → Lobby.
## Pause menu on ESC. Class selection with 1/2 keys in lobby.

enum GameState { LOBBY, FIGHT, RESULT }

var state: GameState = GameState.LOBBY
var paused: bool = false

const LOBBY_SPAWN := Vector3(0.0, 0.1, 20.0)
const ENEMY_SPAWN := Vector3(0.0, 0.1, 0.0)

@onready var gunner: CharacterBody3D = $Gunner
@onready var vanguard: CharacterBody3D = $Vanguard
@onready var blade_dancer: CharacterBody3D = $BladeDancer
@onready var enemy: CharacterBody3D = $BasicEnemy

var player: CharacterBody3D  # Active player reference
var _all_controllers: Array[CharacterBody3D] = []

# Dynamic nodes
var _gate: CSGBox3D
var _trigger: Area3D
var _pause_layer: CanvasLayer
var _result_layer: CanvasLayer
var _result_label: Label
var _class_select_layer: CanvasLayer


func _ready() -> void:
	process_mode = Node.PROCESS_MODE_ALWAYS
	_create_gate()
	_create_trigger()
	_create_pause_menu()
	_create_result_overlay()
	_create_class_select_ui()
	_bake_navigation()

	_all_controllers = [gunner, vanguard, blade_dancer]
	gunner.died.connect(_on_player_died)
	vanguard.died.connect(_on_player_died)
	blade_dancer.died.connect(_on_player_died)
	enemy.died.connect(_on_enemy_died)

	_select_player(gunner)
	_enter_lobby()


func _input(event: InputEvent) -> void:
	# _input instead of _unhandled_input so it fires even when paused
	if event.is_action_pressed("ui_cancel"):
		if state == GameState.RESULT:
			return
		_toggle_pause()
		get_viewport().set_input_as_handled()

	# Class selection in lobby
	if state == GameState.LOBBY and not paused:
		if event is InputEventKey and event.pressed:
			if event.physical_keycode == KEY_1:
				_select_player(gunner)
			elif event.physical_keycode == KEY_2:
				_select_player(vanguard)
			elif event.physical_keycode == KEY_3:
				_select_player(blade_dancer)


# --- Player selection ---

func _select_player(new_player: CharacterBody3D) -> void:
	# Deactivate all controllers
	for ctrl in _all_controllers:
		ctrl.visible = false
		ctrl.collision_layer = 0
		ctrl.set_physics_process(false)
		ctrl.set_process_unhandled_input(false)
		ctrl.camera.current = false
		ctrl.get_node("HUDLayer").visible = false

	player = new_player

	# Activate new player
	player.health = player.max_health
	player.global_position = LOBBY_SPAWN
	player.rotation.y = 0.0
	player.velocity = Vector3.ZERO
	player.hud.update_health(player.health, player.max_health)
	player.visible = true
	player.collision_layer = 2
	player.set_physics_process(true)
	player.set_process_unhandled_input(true)
	player.camera.current = true
	player.get_node("HUDLayer").visible = true

	# Class-specific reset
	if player == vanguard:
		vanguard.hud.update_stamina(vanguard.stamina, vanguard.max_stamina)
	elif player == blade_dancer:
		blade_dancer.hud.update_config(blade_dancer.config)

	Input.set_mouse_mode(Input.MOUSE_MODE_CAPTURED)


# --- State transitions ---

func _enter_lobby() -> void:
	state = GameState.LOBBY
	get_tree().paused = false
	paused = false
	_pause_layer.visible = false
	_result_layer.visible = false
	_class_select_layer.visible = true

	# Reset active player
	player.health = player.max_health
	player.global_position = LOBBY_SPAWN
	player.rotation.y = 0.0
	player.velocity = Vector3.ZERO
	player.hud.update_health(player.health, player.max_health)
	player.visible = true
	player.collision_layer = 2
	player.set_physics_process(true)
	Input.set_mouse_mode(Input.MOUSE_MODE_CAPTURED)

	# Reset class-specific state
	if player == vanguard:
		vanguard.state = vanguard.State.MOVE
		vanguard.stamina = vanguard.max_stamina
		vanguard.hud.update_stamina(vanguard.stamina, vanguard.max_stamina)
		vanguard._is_invincible = false
		vanguard._lock_on_active = false
		vanguard._lock_target = null
	elif player == blade_dancer:
		blade_dancer.state = blade_dancer.State.MOVE
		blade_dancer.config = blade_dancer.Config.ORBIT
		blade_dancer._is_invincible = false
		blade_dancer._guard_active = false
		blade_dancer._lock_on_active = false
		blade_dancer._lock_target = null
		blade_dancer.hud.update_config(blade_dancer.config)

	# Hide enemy
	enemy.visible = false
	enemy.collision_layer = 0
	enemy.set_physics_process(false)

	# Open gate
	_gate.visible = false
	_gate.use_collision = false

	# Enable trigger
	_trigger.monitoring = true


func _start_fight() -> void:
	state = GameState.FIGHT
	_trigger.set_deferred("monitoring", false)
	_class_select_layer.visible = false

	# Close gate
	_gate.visible = true
	_gate.use_collision = true

	# Spawn enemy (full boss reset)
	enemy.global_position = ENEMY_SPAWN
	enemy.health = enemy.max_health
	enemy._current_phase = 1
	enemy._phase_transitioned.clear()
	enemy._last_attack = ""
	# Reset health bar color to green
	var fg_mat := enemy._health_bar_fg.get_surface_override_material(0) as StandardMaterial3D
	if fg_mat:
		fg_mat.albedo_color = Color(0.15, 0.85, 0.15)
		fg_mat.emission_enabled = false
	enemy._update_health_bar()
	enemy.visible = true
	enemy.collision_layer = 4
	enemy.set_physics_process(true)
	enemy._change_state(enemy.State.CHASE)
	CombatLog.start_fight()


func _on_player_died() -> void:
	if state != GameState.FIGHT:
		return
	CombatLog.end_fight("PLAYER_DIED")
	_show_result("YOU DIED", Color(0.8, 0.15, 0.15))


func _on_enemy_died() -> void:
	if state != GameState.FIGHT:
		return
	CombatLog.end_fight("VICTORY")
	_show_result("VICTORY", Color(0.15, 0.8, 0.3))


func _show_result(text: String, color: Color) -> void:
	state = GameState.RESULT
	_result_label.text = text
	_result_label.add_theme_color_override("font_color", color)
	_result_layer.visible = true
	Input.set_mouse_mode(Input.MOUSE_MODE_VISIBLE)
	get_tree().create_timer(3.0).timeout.connect(_enter_lobby)


func _toggle_pause() -> void:
	paused = not paused
	get_tree().paused = paused
	_pause_layer.visible = paused
	if paused:
		Input.set_mouse_mode(Input.MOUSE_MODE_VISIBLE)
	else:
		Input.set_mouse_mode(Input.MOUSE_MODE_CAPTURED)


# --- Trigger ---

func _on_trigger_entered(body: Node3D) -> void:
	if state == GameState.LOBBY and body == player:
		_start_fight()


# --- Build dynamic nodes ---

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

	var restart_btn := Button.new()
	restart_btn.text = "Restart"
	restart_btn.pressed.connect(func(): get_tree().paused = false; get_tree().reload_current_scene())
	vbox.add_child(restart_btn)

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


func _create_class_select_ui() -> void:
	_class_select_layer = CanvasLayer.new()
	_class_select_layer.layer = 15
	add_child(_class_select_layer)

	var label := Label.new()
	label.text = "[1] Gunner   [2] Vanguard   [3] Blade Dancer"
	label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	label.anchor_left = 0.0
	label.anchor_right = 1.0
	label.anchor_top = 0.0
	label.anchor_bottom = 0.0
	label.offset_top = 20.0
	label.offset_bottom = 50.0
	label.add_theme_font_size_override("font_size", 22)
	label.add_theme_color_override("font_color", Color(0.8, 0.8, 0.9, 0.8))
	label.mouse_filter = Control.MOUSE_FILTER_IGNORE
	_class_select_layer.add_child(label)


func _bake_navigation() -> void:
	# Build navmesh manually from known arena geometry (CSG baking doesn't work)
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
	# Pillars (1.5x1.5)
	for pos in [Vector3(-8,0,-6), Vector3(8,0,-6), Vector3(-8,0,6), Vector3(8,0,6), Vector3(0,0,-10), Vector3(0,0,10)]:
		_add_obstruction(source_geo, pos, Vector2(1.5, 1.5), 4.0)
	# Cover blocks
	_add_obstruction(source_geo, Vector3(-5, 0, -2), Vector2(3.0, 1.0), 1.2)
	_add_obstruction(source_geo, Vector3(5, 0, 2), Vector2(3.0, 1.0), 1.2)
	_add_obstruction(source_geo, Vector3(-12, 0, 0), Vector2(1.0, 3.0), 1.2)
	_add_obstruction(source_geo, Vector3(12, 0, 0), Vector2(1.0, 3.0), 1.2)
	# Walls
	_add_obstruction(source_geo, Vector3(0, 0, -15), Vector2(40.0, 0.5), 5.0)
	_add_obstruction(source_geo, Vector3(20, 0, 0), Vector2(0.5, 30.0), 5.0)
	_add_obstruction(source_geo, Vector3(-20, 0, 0), Vector2(0.5, 30.0), 5.0)
	_add_obstruction(source_geo, Vector3(-11.25, 0, 15), Vector2(17.5, 0.5), 5.0)
	_add_obstruction(source_geo, Vector3(11.25, 0, 15), Vector2(17.5, 0.5), 5.0)
	# Lobby walls
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
