extends CharacterBody3D

## Blade Dancer -- Positional state machine controller.
## 5 blade configurations, 20 transition spells (4 per config).
## Third-person target-lock camera, no cooldowns, small GCD.

signal died

enum Config { ORBIT, FAN, LANCE, SCATTER, CROWN }
enum State { MOVE, CASTING, DASH, STAGGER, DEAD }

const PlayerTelegraph := preload("res://scenes/shared/telegraph/player_telegraph.gd")

const NET_INTERP_SPEED := 15.0

const BLADE_SCENE := "res://assets/models/weapons/weapon_floating_blade.glb"
const TELEGRAPH_COLOR := Color(0.2, 0.75, 0.9, 0.4)

# Input actions mapped to spell slots 0-3
const SPELL_SLOT_ACTIONS: Array[StringName] = [
	&"light_attack",  # slot 0 -- LMB
	&"heavy_attack",  # slot 1 -- R
	&"block",  # slot 2 -- RMB
	&"ability_2",  # slot 3 -- T
]

## All 20 transition spells. SPELL_TABLE[origin_config][slot] -> spell dict.
## Each spell transitions from origin_config to dest config.
## action_id = 30 + origin_config * 4 + slot
const SPELL_TABLE := {
	Config.ORBIT:
	[
		{
			name = "Shielded Sweep",
			dest = Config.FAN,
			dur = 0.4,
			action_id = 30,
			telegraph = "circle",
			radius = 4.0,
			desc = "8 dmg AoE (4m). 15% DR for 2s."
		},
		{
			name = "Guarded Thrust",
			dest = Config.LANCE,
			dur = 0.3,
			action_id = 31,
			telegraph = "none",
			desc = "25 dmg single. +8 shield."
		},
		{
			name = "Protected Scatter",
			dest = Config.SCATTER,
			dur = 0.4,
			action_id = 32,
			telegraph = "none",
			desc = "5 dmg x3 nearest. 1.5/tick DoT 12s. 10% DR."
		},
		{
			name = "Fortified Command",
			dest = Config.CROWN,
			dur = 0.5,
			action_id = 33,
			telegraph = "circle_target",
			radius = 5.0,
			desc = "5 dmg AoE at target (5m). 20% DR for 2s."
		},
	],
	Config.FAN:
	[
		{
			name = "Reaping Guard",
			dest = Config.ORBIT,
			dur = 0.4,
			action_id = 34,
			telegraph = "circle",
			radius = 3.0,
			desc = "8 dmg AoE (3m). +12 shield."
		},
		{
			name = "Cleaving Pierce",
			dest = Config.LANCE,
			dur = 0.3,
			action_id = 35,
			telegraph = "none",
			desc = "30 dmg single target."
		},
		{
			name = "Slashing Spread",
			dest = Config.SCATTER,
			dur = 0.4,
			action_id = 36,
			telegraph = "circle_target",
			radius = 5.0,
			desc = "8 dmg AoE at target (5m). 1.5/tick DoT 10s."
		},
		{
			name = "Sweeping Hex",
			dest = Config.CROWN,
			dur = 0.5,
			action_id = 37,
			telegraph = "circle_target",
			radius = 5.0,
			desc = "10 dmg AoE at target (5m)."
		},
	],
	Config.LANCE:
	[
		{
			name = "Piercing Barrier",
			dest = Config.ORBIT,
			dur = 0.4,
			action_id = 38,
			telegraph = "none",
			desc = "18 dmg single. +15 shield."
		},
		{
			name = "Focused Slash",
			dest = Config.FAN,
			dur = 0.3,
			action_id = 39,
			telegraph = "circle_target",
			radius = 4.0,
			desc = "15 dmg AoE at target (4m)."
		},
		{
			name = "Targeted Spread",
			dest = Config.SCATTER,
			dur = 0.4,
			action_id = 40,
			telegraph = "none",
			desc = "12 dmg single. 2.0/tick DoT 15s."
		},
		{
			name = "Pinning Strike",
			dest = Config.CROWN,
			dur = 0.3,
			action_id = 41,
			telegraph = "none",
			desc = "25 dmg single target."
		},
	],
	Config.SCATTER:
	[
		{
			name = "Dispersed Shield",
			dest = Config.ORBIT,
			dur = 0.5,
			action_id = 42,
			telegraph = "none",
			desc = "+18 shield. 15% DR for 2s."
		},
		{
			name = "Rain of Blades",
			dest = Config.FAN,
			dur = 0.4,
			action_id = 43,
			telegraph = "circle_target",
			radius = 5.0,
			desc = "15 dmg AoE at target (5m). 1.0/tick DoT 10s."
		},
		{
			name = "Converging Strike",
			dest = Config.LANCE,
			dur = 0.3,
			action_id = 44,
			telegraph = "none",
			desc = "32 dmg single. 1.5/tick DoT 10s."
		},
		{
			name = "Chaos Bind",
			dest = Config.CROWN,
			dur = 0.5,
			action_id = 45,
			telegraph = "none",
			desc = "8 dmg x4 nearest enemies."
		},
	],
	Config.CROWN:
	[
		{
			name = "Commanding Ward",
			dest = Config.ORBIT,
			dur = 0.5,
			action_id = 46,
			telegraph = "none",
			desc = "+20 shield."
		},
		{
			name = "Royal Cleave",
			dest = Config.FAN,
			dur = 0.3,
			action_id = 47,
			telegraph = "circle",
			radius = 5.0,
			desc = "12 dmg AoE (5m)."
		},
		{
			name = "Decree Strike",
			dest = Config.LANCE,
			dur = 0.3,
			action_id = 48,
			telegraph = "none",
			desc = "28 dmg single target."
		},
		{
			name = "Sovereign Scatter",
			dest = Config.SCATTER,
			dur = 0.4,
			action_id = 49,
			telegraph = "none",
			desc = "5 dmg x3 nearest. 1.5/tick DoT 12s."
		},
	],
}

# Movement
@export var run_speed: float = 6.0
@export var sprint_speed: float = 9.0
@export var mouse_sensitivity: float = 0.003
@export var ground_accel: float = 20.0
@export var ground_decel: float = 15.0
@export var air_accel: float = 2.0
@export var air_decel: float = 1.0
@export var rotation_speed: float = 10.0

# Dash
@export var dash_speed: float = 15.0
@export var dash_duration: float = 0.2
@export var dash_iframe_duration: float = 0.15

# GCD
@export var gcd_duration: float = 0.5

# Cast range (all damage abilities are ranged)
@export var cast_range: float = 20.0

# Camera
@export var camera_distance: float = 6.0
@export var camera_height_offset: float = 2.0

# Health
var health: float = 150.0
var max_health: float = 150.0
var peer_id: int = 0

# State
var state: State = State.MOVE
var config: Config = Config.ORBIT

var _cast_timer: float = 0.0
var _casting_spell: Dictionary = {}
var _state_timer: float = 0.0
var _gcd_timer: float = 0.0
var _is_invincible: bool = false

# Dash
var _dash_direction: Vector3 = Vector3.ZERO

# Camera
var _camera_yaw: float = 0.0
var _camera_pitch: float = -0.3

# Lock-on
var _lock_target: Node3D = null
var _lock_on_active: bool = false

var _gravity: float = 8.5  # must match server gravity
var _flash_timer: float = 0.0
var _facing_angle: float = 0.0
var _alive: bool = true

# Network sync
var _net_anim: String = ""
var _net_anim_speed: float = 1.0
var _net_position: Vector3 = Vector3.ZERO
var _net_rotation_y: float = 0.0

# Blade visuals
var _blade_nodes: Array[Node3D] = []
var _blade_orbit_angle: float = 0.0
var _blade_lerp_speed: float = 12.0
var _blade_spin: float = 0.0

# Materials -- one per config
var _orbit_material: StandardMaterial3D
var _fan_material: StandardMaterial3D
var _lance_material: StandardMaterial3D
var _scatter_material: StandardMaterial3D
var _crown_material: StandardMaterial3D

@onready var camera: Camera3D = $Camera3D
@onready var character_model: Node3D = $CharacterModel
@onready var blade_pivot: Node3D = $BladePivot
@onready var hud: Control = $HUDLayer/BladeDancerHUD


func _ready() -> void:
	GameManager.register_player(self)
	_net_position = global_position
	_net_rotation_y = rotation.y
	camera.top_level = true

	if _is_local():
		Input.set_mouse_mode(Input.MOUSE_MODE_CAPTURED)
		hud.update_config(config)
		hud.update_spells(SPELL_TABLE[config])
	else:
		$HUDLayer.visible = false
		camera.current = false
		set_process_unhandled_input(false)

	_setup_blade_materials()
	_setup_blades()


func _exit_tree() -> void:
	GameManager.unregister_player(self)


func _is_local() -> bool:
	if has_meta("replay_puppet"):
		return false
	if not NetworkManager.is_active:
		return true
	return peer_id == NetworkManager.get_my_id()


## Apply authoritative state from the server's WorldState.
func apply_server_state(data: Dictionary) -> void:
	if _is_local():
		health = data.health
		if data.has("config"):
			var server_config: int = data.config
			if server_config >= 0 and server_config <= 4 and server_config != config:
				config = server_config as Config
				hud.update_config(config)
				hud.update_spells(SPELL_TABLE[config])
		var server_shield: float = data.get("shield_hp", 0.0)
		hud.update_shield(server_shield)
		if health <= 0.0 and _alive:
			_alive = false
			_enter_state(State.DEAD)
			died.emit()
		elif health > 0.0 and not _alive:
			_alive = true
			_enter_state(State.MOVE)
			global_position = data.pos
			_net_position = data.pos
	else:
		_net_position = data.pos
		_net_rotation_y = data.rot_y
		health = data.health
		_net_anim = data.anim_name
		_net_anim_speed = data.anim_speed
		if data.has("config"):
			config = data.config as Config


## Called by main.gd when server confirms this player hit an enemy.
func on_hit_confirmed(_amount: float) -> void:
	hud.show_hit_marker()


## Visual-only damage feedback.
func on_damage_visual(_amount: float, _hit_pos: Vector3) -> void:
	hud.show_damage_flash()
	_show_body_flash()


func _unhandled_input(event: InputEvent) -> void:
	if not _is_local():
		return
	if event is InputEventMouseMotion and Input.get_mouse_mode() == Input.MOUSE_MODE_CAPTURED:
		if not _lock_on_active:
			_camera_yaw -= event.relative.x * mouse_sensitivity
		_camera_pitch -= event.relative.y * mouse_sensitivity
		_camera_pitch = clampf(_camera_pitch, deg_to_rad(-60.0), deg_to_rad(20.0))

	if event.is_action_pressed("lock_on"):
		_toggle_lock_on()


func _physics_process(delta: float) -> void:
	if not _is_local():
		global_position = global_position.move_toward(_net_position, 12.0 * delta)
		rotation.y = lerp_angle(rotation.y, _net_rotation_y, 8.0 * delta)
		if _net_anim != "":
			character_model.play_anim(_net_anim, _net_anim_speed)
		_update_blade_visual(delta)
		return

	if not is_on_floor():
		velocity.y -= _gravity * delta
	else:
		velocity.y = -0.5

	_state_timer -= delta
	_gcd_timer -= delta
	_update_flash(delta)
	_update_camera()

	match state:
		State.MOVE:
			_process_move(delta)
		State.CASTING:
			_process_casting(delta)
		State.DASH:
			_process_dash(delta)
		State.STAGGER:
			_process_stagger()
		State.DEAD:
			velocity.x = 0.0
			velocity.z = 0.0

	move_and_slide()

	_update_animation()
	_update_blade_visual(delta)
	# Clear lock if target is dead, freed, or hidden — use same path as Q toggle
	if _lock_on_active and _lock_target:
		if (
			not is_instance_valid(_lock_target)
			or not _lock_target.visible
			or ("_server_alive" in _lock_target and not _lock_target._server_alive)
		):
			_toggle_lock_on()
	if _lock_on_active and _lock_target:
		hud.update_lock_on(_lock_target, camera)
	hud.update_gcd(_gcd_timer / gcd_duration if _gcd_timer > 0.0 else 0.0)

	if NetworkManager.is_active:
		NetworkManager.send_player_position(global_position, rotation.y, _net_anim, _net_anim_speed)


# --- Movement ---


func _get_camera_wish_dir() -> Vector3:
	var input_dir := Input.get_vector("move_left", "move_right", "move_forward", "move_backward")
	if input_dir.length() < 0.1:
		return Vector3.ZERO

	var cam_xf := camera.global_transform
	var cam_forward := -cam_xf.basis.z
	cam_forward.y = 0.0
	if cam_forward.length() < 0.01:
		return Vector3.ZERO
	cam_forward = cam_forward.normalized()
	var cam_right := cam_xf.basis.x
	cam_right.y = 0.0
	cam_right = cam_right.normalized()
	return (cam_right * input_dir.x + cam_forward * -input_dir.y).normalized()


func _process_move(delta: float) -> void:
	var cursor_active := Input.get_mouse_mode() != Input.MOUSE_MODE_CAPTURED

	# Ability inputs (gated by GCD, disabled when cursor is visible)
	if _gcd_timer <= 0.0 and not cursor_active:
		# Check spell slots 0-3
		for slot in 4:
			if Input.is_action_just_pressed(SPELL_SLOT_ACTIONS[slot]):
				_start_spell(slot)
				return

		# Dash on dodge key (not a spell slot)
		if Input.is_action_just_pressed("dodge") and is_on_floor():
			_start_dash()
			return

	# Jump
	if Input.is_action_just_pressed("jump") and is_on_floor():
		velocity.y = 3.5

	# Movement
	var speed := sprint_speed if Input.is_action_pressed("sprint") else run_speed
	var wish_dir := _get_camera_wish_dir()

	var on_floor := is_on_floor()
	var accel: float = ground_accel if on_floor else air_accel
	var decel: float = ground_decel if on_floor else air_decel

	if wish_dir.length() > 0.1:
		var target_vel := wish_dir * speed
		velocity.x = move_toward(velocity.x, target_vel.x, accel * delta)
		velocity.z = move_toward(velocity.z, target_vel.z, accel * delta)
		if _lock_on_active and _lock_target and is_instance_valid(_lock_target):
			_face_target(delta)
		else:
			_face_direction(wish_dir, delta)
	else:
		velocity.x = move_toward(velocity.x, 0.0, decel * delta)
		velocity.z = move_toward(velocity.z, 0.0, decel * delta)
		if _lock_on_active and _lock_target and is_instance_valid(_lock_target):
			_face_target(delta)


func _get_target_yaw(dir: Vector3) -> float:
	var t := Transform3D()
	t = t.looking_at(dir, Vector3.UP)
	return t.basis.get_euler().y


func _face_direction(dir: Vector3, delta: float) -> void:
	if dir.length() < 0.1:
		return
	var target_angle := _get_target_yaw(dir)
	_facing_angle = lerp_angle(_facing_angle, target_angle, rotation_speed * delta)
	rotation.y = _facing_angle


func _face_target(delta: float) -> void:
	if not _lock_target or not is_instance_valid(_lock_target):
		return
	var to_target := _lock_target.global_position - global_position
	to_target.y = 0.0
	if to_target.length() > 0.1:
		var target_angle := _get_target_yaw(to_target)
		_facing_angle = lerp_angle(_facing_angle, target_angle, rotation_speed * delta)
		rotation.y = _facing_angle


func _face_attack_direction(delta: float) -> void:
	if _lock_on_active and _lock_target and is_instance_valid(_lock_target):
		_face_target(delta)
		return

	var best: Node3D = null
	var best_dist: float = cast_range
	for enemy in GameManager.enemies:
		if not is_instance_valid(enemy) or not enemy.visible:
			continue
		var dist := global_position.distance_to(enemy.global_position)
		if dist < best_dist:
			best_dist = dist
			best = enemy

	if best:
		var to_enemy := best.global_position - global_position
		to_enemy.y = 0.0
		if to_enemy.length() > 0.1:
			var target_angle := _get_target_yaw(to_enemy)
			_facing_angle = lerp_angle(_facing_angle, target_angle, 25.0 * delta)
			rotation.y = _facing_angle
		return

	var cam_fwd := -camera.global_transform.basis.z
	cam_fwd.y = 0.0
	if cam_fwd.length() > 0.01:
		cam_fwd = cam_fwd.normalized()
		var target_angle := _get_target_yaw(cam_fwd)
		_facing_angle = lerp_angle(_facing_angle, target_angle, 15.0 * delta)
		rotation.y = _facing_angle


# --- Spell Casting ---


func _start_spell(slot: int) -> void:
	var spells: Array = SPELL_TABLE[config]
	if slot < 0 or slot >= spells.size():
		return
	var spell: Dictionary = spells[slot]

	_casting_spell = spell
	_cast_timer = spell.dur
	_gcd_timer = gcd_duration

	# Send ability to server
	if NetworkManager.is_active:
		NetworkManager.send_ability(spell.action_id, 0.0, rotation.y)

	# Spawn telegraph if the spell has one
	_spawn_spell_telegraph(spell)

	# Client-side raycast for optimistic hit feedback
	_perform_raycast_hit(cast_range)

	_enter_state(State.CASTING)


func _spawn_spell_telegraph(spell: Dictionary) -> void:
	var telegraph_type: String = spell.get("telegraph", "none")
	if telegraph_type == "none":
		return

	var spell_radius: float = spell.get("radius", 5.0)

	if telegraph_type == "circle":
		PlayerTelegraph.spawn_circle(
			get_tree().root, global_position, spell_radius, TELEGRAPH_COLOR
		)
	elif telegraph_type == "circle_target":
		var target_pos := _get_aim_target_position()
		if target_pos != Vector3.ZERO:
			PlayerTelegraph.spawn_circle(get_tree().root, target_pos, spell_radius, TELEGRAPH_COLOR)


func _get_aim_target_position() -> Vector3:
	var origin := global_position + Vector3(0.0, 1.0, 0.0)
	var direction: Vector3
	if _lock_on_active and _lock_target and is_instance_valid(_lock_target):
		return _lock_target.global_position
	# Raycast to find enemy
	direction = -transform.basis.z
	direction.y = 0.0
	direction = direction.normalized()
	var space := get_world_3d().direct_space_state
	if not space:
		return Vector3.ZERO
	var query := PhysicsRayQueryParameters3D.create(origin, origin + direction * 20.0, 4)  # layer 4 = enemies
	query.exclude = [get_rid()]
	var result := space.intersect_ray(query)
	if result:
		return result.position
	return Vector3.ZERO


func _process_casting(delta: float) -> void:
	_face_attack_direction(delta)

	# Slow movement while casting
	var wish_dir := _get_camera_wish_dir()
	var speed := run_speed * 0.4
	if wish_dir.length() > 0.1:
		var target_vel := wish_dir * speed
		velocity.x = move_toward(velocity.x, target_vel.x, ground_accel * delta)
		velocity.z = move_toward(velocity.z, target_vel.z, ground_accel * delta)
	else:
		velocity.x = move_toward(velocity.x, 0.0, ground_decel * delta)
		velocity.z = move_toward(velocity.z, 0.0, ground_decel * delta)

	_cast_timer -= delta
	if _cast_timer <= 0.0 and not _casting_spell.is_empty():
		# Transition config on cast completion
		config = _casting_spell.dest as Config
		hud.update_config(config)
		hud.update_spells(SPELL_TABLE[config])
		_casting_spell = {}
		_enter_state(State.MOVE)


# --- Dash ---


func _start_dash() -> void:
	_gcd_timer = gcd_duration
	var wish := _get_camera_wish_dir()
	if wish.length() > 0.1:
		_dash_direction = wish
	else:
		_dash_direction = -transform.basis.z.normalized()

	_enter_state(State.DASH)
	_state_timer = dash_duration
	_is_invincible = true


func _process_dash(_delta: float) -> void:
	velocity.x = _dash_direction.x * dash_speed
	velocity.z = _dash_direction.z * dash_speed

	var elapsed := dash_duration - _state_timer
	if elapsed >= dash_iframe_duration:
		_is_invincible = false

	if _state_timer <= 0.0:
		_is_invincible = false
		velocity.x *= 0.3
		velocity.z *= 0.3
		_enter_state(State.MOVE)


# --- Stagger ---


func _process_stagger() -> void:
	velocity.x = 0.0
	velocity.z = 0.0
	if _state_timer <= 0.0:
		_enter_state(State.MOVE)


# --- Ranged Hit Detection ---


func _perform_raycast_hit(max_range: float) -> void:
	# Server resolves hits -- client only shows optimistic hit marker
	var origin := global_position + Vector3(0.0, 1.0, 0.0)
	var direction: Vector3
	if _lock_on_active and _lock_target and is_instance_valid(_lock_target):
		direction = (_lock_target.global_position + Vector3(0.0, 1.0, 0.0) - origin).normalized()
	else:
		direction = -transform.basis.z
		direction.y = 0.0
		direction = direction.normalized()

	var space := get_world_3d().direct_space_state
	if not space:
		return
	var query := PhysicsRayQueryParameters3D.create(origin, origin + direction * max_range, 4 | 1)
	query.exclude = [get_rid()]
	var result := space.intersect_ray(query)
	# Hit marker now driven by server-confirmed damage events (on_hit_confirmed)


# --- Lock-on ---


func _toggle_lock_on() -> void:
	if _lock_on_active:
		# _camera_yaw is already at the auto-computed angle — no snap on unlock
		_lock_on_active = false
		_lock_target = null
		hud.hide_lock_on()
	else:
		var target := _find_lock_target()
		if target:
			_lock_on_active = true
			_lock_target = target
			hud.show_lock_on()


func _find_lock_target() -> Node3D:
	var best: Node3D = null
	var best_dist: float = 30.0
	for enemy in GameManager.enemies:
		if not is_instance_valid(enemy) or not enemy.visible:
			continue
		var dist := global_position.distance_to(enemy.global_position)
		if dist < best_dist:
			best_dist = dist
			best = enemy
	return best


# --- Damage (server-authoritative) ---


func take_damage(_amount: float, _hit_position: Vector3 = Vector3.ZERO) -> void:
	pass  # Server handles all damage


# --- Visual feedback ---


func _show_body_flash() -> void:
	character_model.flash_damage()


func _update_flash(_delta: float) -> void:
	pass


func _update_animation() -> void:
	match state:
		State.DASH:
			_net_anim = "roll"
			_net_anim_speed = 1.0
			character_model.play_anim_timed("roll", dash_duration)
			return
		State.CASTING:
			_net_anim = "slash"
			_net_anim_speed = 1.0
			var dur: float = _casting_spell.get("dur", 0.4)
			character_model.play_anim_timed("slash", dur)
			return
		State.STAGGER:
			_net_anim = "idle"
			_net_anim_speed = 1.0
			character_model.play_anim("idle")
			return
		State.DEAD:
			_net_anim = "idle"
			_net_anim_speed = 1.0
			character_model.play_anim("idle")
			return

	if not is_on_floor():
		_net_anim = "jump"
		_net_anim_speed = 2.0
		character_model.play_anim("jump", 2.0)
		return

	var flat_vel := Vector3(velocity.x, 0.0, velocity.z)
	if flat_vel.length() > 0.5:
		var speed_ratio := flat_vel.length() / sprint_speed
		_net_anim_speed = clampf(speed_ratio, 0.5, 1.5)
		_net_anim = "run"
		character_model.play_anim("run", _net_anim_speed)
	else:
		_net_anim = "idle"
		_net_anim_speed = 1.0
		character_model.play_anim("idle")


# --- Blade Visuals ---


func _setup_blade_materials() -> void:
	_orbit_material = _make_material(Color(0.2, 0.8, 0.9), Color(0.1, 0.6, 0.8))
	_fan_material = _make_material(Color(1.0, 0.5, 0.1), Color(0.9, 0.35, 0.05))
	_lance_material = _make_material(Color(0.9, 0.2, 0.1), Color(0.8, 0.1, 0.05))
	_scatter_material = _make_material(Color(0.6, 0.2, 0.9), Color(0.5, 0.1, 0.8))
	_crown_material = _make_material(Color(1.0, 0.85, 0.3), Color(0.9, 0.75, 0.2))


func _make_material(albedo: Color, emission: Color) -> StandardMaterial3D:
	var mat := StandardMaterial3D.new()
	mat.albedo_color = albedo
	mat.emission_enabled = true
	mat.emission = emission
	mat.emission_energy_multiplier = 2.0
	return mat


func _get_config_material(cfg: Config) -> StandardMaterial3D:
	match cfg:
		Config.ORBIT:
			return _orbit_material
		Config.FAN:
			return _fan_material
		Config.LANCE:
			return _lance_material
		Config.SCATTER:
			return _scatter_material
		Config.CROWN:
			return _crown_material
	return _orbit_material


func _setup_blades() -> void:
	var blade_scene := load(BLADE_SCENE) as PackedScene
	if not blade_scene:
		push_warning("BladeDancer: could not load blade model %s" % BLADE_SCENE)
		return
	for i in 3:
		var blade := blade_scene.instantiate()
		blade_pivot.add_child(blade)
		_blade_nodes.append(blade)
		_apply_blade_material(blade, _orbit_material)


func _update_blade_visual(delta: float) -> void:
	if _blade_nodes.is_empty():
		return

	_blade_spin += delta * 2.0
	_blade_orbit_angle += 120.0 * delta

	var targets: Array[Vector3] = []
	var target_rots: Array[float] = []
	var mat: StandardMaterial3D
	var lerp_speed := _blade_lerp_speed

	match state:
		State.CASTING:
			# During casting, blend toward destination config formation
			var dest_cfg: Config = _casting_spell.get("dest", Config.ORBIT) as Config
			mat = _get_config_material(dest_cfg)
			lerp_speed = 20.0
			var dur: float = _casting_spell.get("dur", 0.4)
			var progress := 1.0 - (_cast_timer / dur) if dur > 0.0 else 1.0
			progress = clampf(progress, 0.0, 1.0)

			# Sweep blades forward during first half, settle into dest formation second half
			if progress < 0.5:
				var sweep := progress * 2.0  # 0->1 in first half
				var sweep_angle := lerpf(-60.0, 60.0, sweep)
				for i in 3:
					var a := deg_to_rad(sweep_angle + (i - 1) * 25.0)
					var r := 1.8
					targets.append(Vector3(sin(a) * r, 0.9, -cos(a) * r))
					target_rots.append(a)
			else:
				var settle := (progress - 0.5) * 2.0  # 0->1 in second half
				var dest_targets := _get_formation_positions(dest_cfg)
				var dest_rots := _get_formation_rotations(dest_cfg)
				for i in 3:
					var sweep_a := deg_to_rad(60.0 + (i - 1) * 25.0)
					var sweep_pos := Vector3(sin(sweep_a) * 1.8, 0.9, -cos(sweep_a) * 1.8)
					targets.append(sweep_pos.lerp(dest_targets[i], settle))
					target_rots.append(lerp_angle(sweep_a, dest_rots[i], settle))

		State.DASH:
			mat = _get_config_material(config)
			lerp_speed = 15.0
			for i in 3:
				var spread := (i - 1) * 0.5
				targets.append(Vector3(spread, 0.9, 1.5))
				target_rots.append(PI)

		_:
			# Idle / Move -- use current config formation
			mat = _get_config_material(config)
			targets = _get_formation_positions(config)
			target_rots = _get_formation_rotations(config)

	for i in 3:
		if i >= targets.size():
			break
		_blade_nodes[i].position = _blade_nodes[i].position.lerp(targets[i], lerp_speed * delta)
		_blade_nodes[i].rotation.y = lerp_angle(
			_blade_nodes[i].rotation.y, target_rots[i], lerp_speed * delta
		)
		_blade_nodes[i].rotation.x = sin(_blade_spin + i * 2.0) * 0.15
		_apply_blade_material(_blade_nodes[i], mat)


## Get idle formation positions for a given config.
func _get_formation_positions(cfg: Config) -> Array[Vector3]:
	var positions: Array[Vector3] = []
	match cfg:
		Config.ORBIT:
			# Circular orbit around player
			for i in 3:
				var angle := deg_to_rad(_blade_orbit_angle + i * 120.0)
				positions.append(Vector3(cos(angle) * 1.0, 0.9, sin(angle) * 1.0))
		Config.FAN:
			# Arc spread in front of player
			for i in 3:
				var angle := deg_to_rad(-30.0 + i * 30.0)
				positions.append(Vector3(sin(angle) * 1.5, 0.9, -cos(angle) * 1.5))
		Config.LANCE:
			# Tight line aimed forward
			var local_dir: Vector3
			if _lock_on_active and _lock_target and is_instance_valid(_lock_target):
				var to_target := _lock_target.global_position - global_position
				to_target.y = 0.0
				local_dir = transform.basis.inverse() * to_target.normalized()
			else:
				local_dir = Vector3(0.0, 0.0, -1.0)
			for i in 3:
				var d := 2.0 + i * 0.4
				positions.append(Vector3(local_dir.x * d, 0.9, local_dir.z * d))
		Config.SCATTER:
			# Blades spread outward in different directions
			for i in 3:
				var angle := deg_to_rad(_blade_orbit_angle * 0.7 + i * 120.0)
				var r := 1.8
				positions.append(Vector3(cos(angle) * r, 0.6 + i * 0.3, sin(angle) * r))
		Config.CROWN:
			# Hover above head in a halo
			for i in 3:
				var angle := deg_to_rad(_blade_orbit_angle * 0.5 + i * 120.0)
				var r := 0.6
				positions.append(Vector3(cos(angle) * r, 1.8, sin(angle) * r))
	return positions


## Get idle formation rotations for a given config.
func _get_formation_rotations(cfg: Config) -> Array[float]:
	var rotations: Array[float] = []
	match cfg:
		Config.ORBIT:
			for i in 3:
				var angle := deg_to_rad(_blade_orbit_angle + i * 120.0)
				rotations.append(angle + PI / 2.0)
		Config.FAN:
			for i in 3:
				var angle := deg_to_rad(-30.0 + i * 30.0)
				rotations.append(angle)
		Config.LANCE:
			var local_dir: Vector3
			if _lock_on_active and _lock_target and is_instance_valid(_lock_target):
				var to_target := _lock_target.global_position - global_position
				to_target.y = 0.0
				local_dir = transform.basis.inverse() * to_target.normalized()
			else:
				local_dir = Vector3(0.0, 0.0, -1.0)
			for i in 3:
				rotations.append(atan2(local_dir.x, local_dir.z))
		Config.SCATTER:
			for i in 3:
				var angle := deg_to_rad(_blade_orbit_angle * 0.7 + i * 120.0)
				rotations.append(angle + PI / 4.0)
		Config.CROWN:
			for i in 3:
				var angle := deg_to_rad(_blade_orbit_angle * 0.5 + i * 120.0)
				rotations.append(angle)
	return rotations


## Apply a material override to all MeshInstance3D children in a GLB instance.
func _apply_blade_material(node: Node3D, mat: StandardMaterial3D) -> void:
	for child in node.get_children():
		if child is MeshInstance3D:
			for s in child.get_surface_override_material_count():
				child.set_surface_override_material(s, mat)
		if child.get_child_count() > 0:
			_apply_blade_material(child, mat)


# --- Camera ---


func _update_camera() -> void:
	var player_pos := global_position + Vector3(0.0, camera_height_offset, 0.0)
	var delta := get_physics_process_delta_time()

	if _lock_on_active and _lock_target and is_instance_valid(_lock_target):
		# Dark Souls lock-on: camera orbits behind the player, looking toward target.
		var target_pos := _lock_target.global_position + Vector3(0.0, 1.0, 0.0)
		var midpoint := player_pos.lerp(target_pos, 0.4)

		# Compute desired yaw: opposite of player-to-target direction (behind the player)
		var to_target := target_pos - player_pos
		var desired_yaw := atan2(to_target.x, to_target.z) + PI

		# Smoothly interpolate camera yaw toward the auto-computed angle
		_camera_yaw = lerp_angle(_camera_yaw, desired_yaw, 6.0 * delta)

		# Auto-adjust pitch based on height difference
		var height_diff := target_pos.y - player_pos.y
		var desired_pitch := clampf(-0.2 - height_diff * 0.05, deg_to_rad(-60.0), deg_to_rad(20.0))
		_camera_pitch = lerp(_camera_pitch, desired_pitch, 4.0 * delta)

		# Position camera behind the player (opposite side from target)
		var cam_offset := Vector3(0.0, 0.0, camera_distance)
		cam_offset = cam_offset.rotated(Vector3.RIGHT, _camera_pitch)
		cam_offset = cam_offset.rotated(Vector3.UP, _camera_yaw)
		var desired_cam_pos := player_pos + cam_offset
		camera.global_position = _apply_camera_collision(player_pos, desired_cam_pos)
		camera.look_at(midpoint, Vector3.UP)
	else:
		var cam_offset := Vector3(0.0, 0.0, camera_distance)
		cam_offset = cam_offset.rotated(Vector3.RIGHT, _camera_pitch)
		cam_offset = cam_offset.rotated(Vector3.UP, _camera_yaw)
		var desired_cam_pos := player_pos + cam_offset
		camera.global_position = _apply_camera_collision(player_pos, desired_cam_pos)
		camera.look_at(player_pos, Vector3.UP)


func _apply_camera_collision(from: Vector3, to: Vector3) -> Vector3:
	var space := get_world_3d().direct_space_state
	if not space:
		return to
	var query := PhysicsRayQueryParameters3D.create(from, to, 1)
	query.exclude = [get_rid()]
	var result := space.intersect_ray(query)
	if result:
		return result.position + (from - to).normalized() * 0.3
	return to


# --- Helpers ---


func _enter_state(new_state: State) -> void:
	match state:
		State.DASH:
			_is_invincible = false
	state = new_state
