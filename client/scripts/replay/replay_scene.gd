extends Node3D
## Replay playback scene. Loads an environment, spawns entity puppets from
## decoded WorldState frames, and drives playback via the ReplayHUD.

signal replay_exited

const ARENA_SCENE := "res://scenes/environments/arena/arena.tscn"
const ENEMY_SCENE := "res://scenes/enemies/basic_enemy/basic_enemy.tscn"
const PROJECTILE_SCENE := "res://scenes/enemies/basic_enemy/enemy_projectile.tscn"
const CLASS_SCENES := {
	"gunner": "res://scenes/controllers/gunner/gunner.tscn",
	"vanguard": "res://scenes/controllers/vanguard/vanguard.tscn",
	"blade_dancer": "res://scenes/controllers/blade_dancer/blade_dancer.tscn",
}

const ZONE_SCENES := {
	"arena": "res://scenes/environments/arena/arena.tscn",
}

var _replay: Variant  # ReplayData
var _current_frame: int = 0
var _playing: bool = true
var _speed: float = 1.0
var _accumulator: float = 0.0

var _players_node: Node3D
var _enemies_node: Node3D
var _projectiles_node: Node3D
var _camera: Camera3D
var _hud: Node  # replay_hud.gd instance

var _spawned_players: Dictionary = {}  # peer_id -> CharacterBody3D
var _spawned_enemies: Dictionary = {}  # enemy_id -> CharacterBody3D
var _spawned_projectiles: Dictionary = {}  # proj_id -> Node3D


func _ready() -> void:
	_players_node = Node3D.new()
	_players_node.name = "Players"
	add_child(_players_node)

	_enemies_node = Node3D.new()
	_enemies_node.name = "Enemies"
	add_child(_enemies_node)

	_projectiles_node = Node3D.new()
	_projectiles_node.name = "Projectiles"
	add_child(_projectiles_node)


func start_replay(replay: Variant) -> void:
	_replay = replay

	# Load environment scene
	var zone_scene_path: String = ZONE_SCENES.get(replay.zone_id, ARENA_SCENE)
	var env_scene := load(zone_scene_path) as PackedScene
	if env_scene:
		var env := env_scene.instantiate()
		env.name = "Environment"
		add_child(env)
		# Move it to first child so it renders behind entities
		move_child(env, 0)

		# Add atmospheric effects (rain, lightning, fire lights)
		var atmosphere := Node3D.new()
		atmosphere.name = "DungeonAtmosphere"
		atmosphere.set_script(load("res://scenes/environments/arena/dungeon_atmosphere.gd"))
		env.add_child(atmosphere)

	# Create free-fly camera
	var cam_script := load("res://scripts/replay/free_fly_camera.gd")
	_camera = Camera3D.new()
	_camera.name = "FreeFlyCamera"
	_camera.set_script(cam_script)
	_camera.position = Vector3(0.0, 8.0, 20.0)
	add_child(_camera)

	# Create HUD
	var hud_script := load("res://scripts/replay/replay_hud.gd")
	_hud = CanvasLayer.new()
	_hud.set_script(hud_script)
	add_child(_hud)
	_hud.init(replay)
	_hud.play_toggled.connect(_on_play_toggled)
	_hud.speed_changed.connect(_on_speed_changed)
	_hud.frame_seeked.connect(_on_frame_seeked)
	_hud.back_pressed.connect(_on_back_pressed)

	# Apply first frame immediately
	if _replay.frame_count > 0:
		_apply_frame(0)


func _process(delta: float) -> void:
	if _replay == null or _replay.frame_count == 0:
		return

	if _playing:
		_accumulator += delta * _speed
		var frame_duration: float = 1.0 / float(_replay.tick_rate)
		var ticks: int = int(_accumulator / frame_duration)
		_accumulator -= float(ticks) * frame_duration
		if ticks > 0:
			var old_frame := _current_frame
			_current_frame = mini(_current_frame + ticks, _replay.frame_count - 1)
			# Emit events for skipped frames
			for f in range(old_frame + 1, _current_frame + 1):
				_check_events(f)
			if _current_frame >= _replay.frame_count - 1:
				_playing = false

	_apply_frame(_current_frame)
	_hud.update_frame(_current_frame)


func _apply_frame(index: int) -> void:
	var frame_data: PackedByteArray = _replay.get_frame(index)
	if frame_data.is_empty():
		return

	var world_state: Dictionary = NetSerializer.World.decode_world_state(frame_data)
	var players_data: Array = world_state.get("players", [])
	var enemies_data: Array = world_state.get("enemies", [])
	_sync_players(players_data)
	_sync_enemies(enemies_data)
	_sync_projectiles(world_state.get("projectiles", []))
	_hud.update_players(players_data)
	_hud.update_enemies(enemies_data)


func _sync_players(players_data: Array) -> void:
	var seen: Dictionary = {}
	for pdata in players_data:
		var pid: int = pdata["peer_id"]
		seen[pid] = true
		if pid not in _spawned_players:
			_spawn_replay_player(pdata)
		var player: CharacterBody3D = _spawned_players[pid]
		if is_instance_valid(player) and player.has_method("apply_server_state"):
			player.apply_server_state(pdata)
		# Hide dead players, show alive ones
		var hp: float = pdata.get("health", 1.0)
		if is_instance_valid(player):
			player.visible = hp > 0.0
		# Update overhead name
		_update_overhead_name(player, pdata)

	# Remove absent players
	var to_remove: Array = []
	for pid in _spawned_players:
		if pid not in seen:
			to_remove.append(pid)
	for pid in to_remove:
		if is_instance_valid(_spawned_players[pid]):
			_spawned_players[pid].queue_free()
		_spawned_players.erase(pid)


func _spawn_replay_player(pdata: Dictionary) -> void:
	var pid: int = pdata["peer_id"]
	var cls: String = pdata.get("class_name", "gunner")
	if not CLASS_SCENES.has(cls):
		cls = "gunner"
	var scene := load(CLASS_SCENES[cls]) as PackedScene
	if not scene:
		return
	var player := scene.instantiate() as CharacterBody3D
	player.name = "ReplayPlayer_%d" % pid
	player.peer_id = pid
	# Mark as puppet BEFORE add_child so _ready() sees it via _is_local().
	player.set_meta("replay_puppet", true)
	# Set initial position via public setter to avoid interpolation from origin.
	player.set("_net_position", pdata["pos"])
	player.set("_net_rotation_y", pdata.get("rot_y", 0.0))
	_players_node.add_child(player)
	player.global_position = pdata["pos"]
	_spawned_players[pid] = player
	# Ensure free-fly camera stays active
	_camera.current = true
	Input.set_mouse_mode(Input.MOUSE_MODE_VISIBLE)


func _update_overhead_name(player: CharacterBody3D, pdata: Dictionary) -> void:
	if not is_instance_valid(player):
		return
	var label: Label3D = player.get_node_or_null("OverheadName")
	if label == null:
		label = Label3D.new()
		label.name = "OverheadName"
		label.font_size = 32
		label.pixel_size = 0.005
		label.billboard = BaseMaterial3D.BILLBOARD_ENABLED
		label.no_depth_test = true
		label.position = Vector3(0, 2.2, 0)
		player.add_child(label)
	var entity_id: String = "player_%d" % pdata["peer_id"]
	label.text = _replay.get_participant_name(entity_id)


func _sync_enemies(enemies_data: Array) -> void:
	var seen: Dictionary = {}
	for edata in enemies_data:
		var eid: int = edata["enemy_id"]
		seen[eid] = true
		var alive: bool = edata["alive"]
		if alive and eid not in _spawned_enemies:
			var scene := load(ENEMY_SCENE) as PackedScene
			if scene:
				var node := scene.instantiate() as CharacterBody3D
				node.name = "ReplayEnemy_%d" % eid
				node.peer_id = eid
				_enemies_node.add_child(node)
				_spawned_enemies[eid] = node
		if eid in _spawned_enemies:
			var node: CharacterBody3D = _spawned_enemies[eid]
			if is_instance_valid(node):
				if alive:
					node.visible = true
					if node.has_method("apply_server_state"):
						node.apply_server_state(edata)
				else:
					node.visible = false

	var to_remove: Array = []
	for eid in _spawned_enemies:
		if eid not in seen:
			to_remove.append(eid)
	for eid in to_remove:
		if is_instance_valid(_spawned_enemies[eid]):
			_spawned_enemies[eid].queue_free()
		_spawned_enemies.erase(eid)


func _sync_projectiles(proj_data: Array) -> void:
	var active_ids: Dictionary = {}
	for p in proj_data:
		var pid: int = p["proj_id"]
		active_ids[pid] = true
		if pid not in _spawned_projectiles:
			var scene := load(PROJECTILE_SCENE) as PackedScene
			if scene:
				var proj := scene.instantiate() as Node3D
				proj.name = "ReplayProj_%d" % pid
				_projectiles_node.add_child(proj)
				proj.global_position = p["pos"]
				if proj.has_method("setup"):
					(
						proj
						. setup(
							p.get("direction", Vector3.FORWARD),
							p.get("speed", 22.0),
							p.get("angular_velocity", 0.0),
							p.get("visual_tag", ""),
						)
					)
				# Disable self-movement and auto-destruction — replay drives position
				proj.set_physics_process(false)
				if proj is Area3D:
					proj.monitoring = false
				_spawned_projectiles[pid] = proj
		elif is_instance_valid(_spawned_projectiles[pid]):
			_spawned_projectiles[pid].global_position = p["pos"]

	var to_remove: Array = []
	for pid in _spawned_projectiles:
		if pid not in active_ids:
			to_remove.append(pid)
	for pid in to_remove:
		if is_instance_valid(_spawned_projectiles[pid]):
			_spawned_projectiles[pid].queue_free()
		_spawned_projectiles.erase(pid)


func _check_events(frame_tick: int) -> void:
	# The tick stored in replay frames is the absolute server tick.
	# Events are keyed by absolute tick as well.
	var events: Array = _replay.get_events_at_tick(frame_tick)
	for ev in events:
		_hud.record_event(ev)
		var event_type: int = ev.get("event_type", 0)
		match event_type:
			1, 5:  # Damage, BuffTick
				var amount: float = ev.get("amount", 0.0)
				var pos := Vector3(ev.get("pos_x", 0.0), ev.get("pos_y", 0.0), ev.get("pos_z", 0.0))
				if amount > 0 and pos != Vector3.ZERO:
					_spawn_damage_number(amount, pos)


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
	var offset := Vector3(randf_range(-0.3, 0.3), randf_range(0.0, 0.3), randf_range(-0.3, 0.3))
	label.position = world_pos + offset + Vector3(0.0, 0.5, 0.0)
	add_child(label)
	var tween := create_tween()
	tween.set_parallel(true)
	(
		tween
		. tween_property(label, "position:y", label.position.y + 1.5, 0.8)
		. set_ease(Tween.EASE_OUT)
		. set_trans(Tween.TRANS_QUAD)
	)
	tween.tween_property(label, "modulate:a", 0.0, 0.8).set_delay(0.3)
	tween.tween_property(label, "outline_modulate:a", 0.0, 0.8).set_delay(0.3)
	tween.chain().tween_callback(label.queue_free)


func _on_play_toggled(playing: bool) -> void:
	_playing = playing


func _on_speed_changed(speed: float) -> void:
	_speed = speed


func _on_frame_seeked(frame: int) -> void:
	_current_frame = clampi(frame, 0, _replay.frame_count - 1)
	_accumulator = 0.0
	_apply_frame(_current_frame)
	_hud.update_frame(_current_frame)
	_hud.rebuild_damage(_current_frame)


func _on_back_pressed() -> void:
	Input.mouse_mode = Input.MOUSE_MODE_VISIBLE
	replay_exited.emit()
