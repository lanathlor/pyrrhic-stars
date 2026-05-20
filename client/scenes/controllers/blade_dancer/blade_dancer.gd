extends CharacterBody3D

## Blade Dancer -- Positional state machine controller.
## 5 blade configurations, 20 transition spells (4 per config).
## Third-person target-lock camera, no cooldowns, small GCD.
## Sub-systems: spells, movement, blades, cam (child nodes).

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
	&"block",  # slot 1 -- RMB
	&"heavy_attack",  # slot 2 -- R
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
			name = "Protected Scatter",
			dest = Config.SCATTER,
			dur = 0.4,
			action_id = 32,
			telegraph = "none",
			desc = "5 dmg x3 nearest. 1.5/tick DoT 12s. 10% DR."
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
			name = "Slashing Spread",
			dest = Config.SCATTER,
			dur = 0.4,
			action_id = 36,
			telegraph = "circle_target",
			radius = 5.0,
			desc = "8 dmg AoE at target (5m). 1.5/tick DoT 10s."
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
			name = "Targeted Spread",
			dest = Config.SCATTER,
			dur = 0.4,
			action_id = 40,
			telegraph = "none",
			desc = "12 dmg single. 2.0/tick DoT 15s."
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
			name = "Converging Strike",
			dest = Config.LANCE,
			dur = 0.3,
			action_id = 44,
			telegraph = "none",
			desc = "32 dmg single. 1.5/tick DoT 10s."
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
			name = "Decree Strike",
			dest = Config.LANCE,
			dur = 0.3,
			action_id = 48,
			telegraph = "none",
			desc = "28 dmg single target."
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
			name = "Sovereign Scatter",
			dest = Config.SCATTER,
			dur = 0.4,
			action_id = 49,
			telegraph = "none",
			desc = "5 dmg x3 nearest. 1.5/tick DoT 12s."
		},
	],
}

const SpellsScript := preload("res://scenes/controllers/blade_dancer/blade_dancer_spells.gd")
const MovementScript := preload("res://scenes/controllers/blade_dancer/blade_dancer_movement.gd")
const BladesScript := preload("res://scenes/controllers/blade_dancer/blade_dancer_blades.gd")
const CameraScript := preload("res://scenes/controllers/blade_dancer/blade_dancer_camera.gd")

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

# Sub-systems
var spells: Node
var movement: Node
var blades: Node
var cam: Node

var _cast_timer: float = 0.0
var _casting_spell: Dictionary = {}
var _state_timer: float = 0.0
var _gcd_timer: float = 0.0
var _is_invincible: bool = false

# Flow mastery (server-authoritative)
var _flow_tier: int = 0
var _flow_stacks: int = 0

# Camera
var _camera_yaw: float = 0.0
var _camera_pitch: float = -0.3

# Lock-on
var _lock_target: Node3D = null
var _lock_on_active: bool = false

var _gravity: float = 8.5  # must match server gravity
var _alive: bool = true

# Network sync
var _visual_state: int = 0
var _net_position: Vector3 = Vector3.ZERO
var _net_rotation_y: float = 0.0

# Blade visual proxies (delegated to blades sub-system, exposed for tests)
var _blade_nodes: Array[Node3D]:
	get:
		if blades:
			return blades._blade_nodes
		return []
var _orbit_material: StandardMaterial3D:
	get:
		if blades:
			return blades._orbit_material
		return null
var _fan_material: StandardMaterial3D:
	get:
		if blades:
			return blades._fan_material
		return null
var _lance_material: StandardMaterial3D:
	get:
		if blades:
			return blades._lance_material
		return null
var _scatter_material: StandardMaterial3D:
	get:
		if blades:
			return blades._scatter_material
		return null
var _crown_material: StandardMaterial3D:
	get:
		if blades:
			return blades._crown_material
		return null

@onready var camera: Camera3D = $Camera3D
@onready var character_model: Node3D = $CharacterModel
@onready var blade_pivot: Node3D = $BladePivot
@onready var hud: Control = $HUDLayer/BladeDancerHUD


func _ready() -> void:
	# Create sub-systems
	spells = _add_subsystem("Spells", SpellsScript)
	movement = _add_subsystem("Movement", MovementScript)
	blades = _add_subsystem("Blades", BladesScript)
	cam = _add_subsystem("Cam", CameraScript)

	GameManager.register_player(self)
	_net_position = global_position
	_net_rotation_y = rotation.y
	camera.top_level = true

	# Set up animation state machine
	(
		character_model
		. setup_state_machine(
			{
				"idle": "idle",
				"run": "run",
				"jump": "jump",
				"dash": "roll",
				"casting": "slash",
				"stagger": "idle",
				"dead": "idle",
			}
		)
	)

	if _is_local():
		Input.set_mouse_mode(Input.MOUSE_MODE_CAPTURED)
		hud.update_config(config)
		hud.update_spells(SPELL_TABLE[config])
	else:
		$HUDLayer.visible = false
		camera.current = false
		set_process_unhandled_input(false)

	blades.setup()


func _add_subsystem(node_name: String, script: GDScript) -> Node:
	var node: Node = script.new()
	node.name = node_name
	add_child(node)
	return node


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
	if data.has("max_health") and data["max_health"] > 0.0:
		max_health = data["max_health"]
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
		_flow_tier = data.get("flow_tier", 0)
		_flow_stacks = data.get("flow_stacks", 0)
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
		_visual_state = data.get("visual_state", 0)
		if data.has("config"):
			config = data.config as Config


## Called by main.gd when server confirms this player hit an enemy.
func on_hit_confirmed(_amount: float, _hit_pos: Vector3 = Vector3.ZERO) -> void:
	hud.show_hit_marker()


## Visual-only damage feedback.
func on_damage_visual(_amount: float, _hit_pos: Vector3) -> void:
	hud.show_damage_flash()
	cam.show_body_flash()


func _unhandled_input(event: InputEvent) -> void:
	if not _is_local():
		return
	if event is InputEventMouseMotion and Input.get_mouse_mode() == Input.MOUSE_MODE_CAPTURED:
		if not _lock_on_active:
			_camera_yaw -= event.relative.x * mouse_sensitivity
		_camera_pitch -= event.relative.y * mouse_sensitivity
		_camera_pitch = clampf(_camera_pitch, deg_to_rad(-60.0), deg_to_rad(20.0))

	if event.is_action_pressed("lock_on"):
		cam.toggle_lock_on()


func _physics_process(delta: float) -> void:
	if not _is_local():
		var prev_pos := global_position
		global_position = global_position.move_toward(_net_position, 12.0 * delta)
		rotation.y = lerp_angle(rotation.y, _net_rotation_y, 8.0 * delta)
		_drive_remote_animation(prev_pos, delta)
		blades.update_blade_visual(delta)
		return

	if not is_on_floor():
		velocity.y -= _gravity * delta
	else:
		velocity.y = -0.5

	_state_timer -= delta
	_gcd_timer -= delta
	cam.update_flash(delta)
	cam.update_camera()

	match state:
		State.MOVE:
			movement.process_move(delta)
		State.CASTING:
			spells.process_casting(delta)
		State.DASH:
			movement.process_dash(delta)
		State.STAGGER:
			movement.process_stagger()
		State.DEAD:
			velocity.x = 0.0
			velocity.z = 0.0

	move_and_slide()

	cam.update_animation()
	blades.update_blade_visual(delta)
	# Clear lock if target is dead, freed, or hidden — use same path as Q toggle
	if _lock_on_active and _lock_target:
		if (
			not is_instance_valid(_lock_target)
			or not _lock_target.visible
			or ("_server_alive" in _lock_target and not _lock_target._server_alive)
		):
			cam.toggle_lock_on()
	if _lock_on_active and _lock_target:
		hud.update_lock_on(_lock_target, camera)
	hud.update_gcd(_gcd_timer / gcd_duration if _gcd_timer > 0.0 else 0.0)
	hud.update_flow(_flow_tier, _flow_stacks)

	if NetworkManager.is_active:
		NetworkManager.send_player_position(global_position, rotation.y, _visual_state)


# --- Damage (server-authoritative) ---


func take_damage(_amount: float, _hit_position: Vector3 = Vector3.ZERO) -> void:
	pass  # Server handles all damage


func _drive_remote_animation(prev_pos: Vector3, delta: float) -> void:
	match _visual_state:
		NetSerializer.VS_DODGE, NetSerializer.VS_BD_DASH:
			character_model.travel("dash")
		NetSerializer.VS_BD_CASTING:
			character_model.travel("casting")
		NetSerializer.VS_BD_STAGGER:
			character_model.travel("stagger")
		NetSerializer.VS_DEAD:
			character_model.travel("dead")
		_:  # VS_MOVE or unknown — derive from velocity
			var vel := (global_position - prev_pos) / delta if delta > 0 else Vector3.ZERO
			var speed := Vector2(vel.x, vel.z).length()
			if speed > 0.5:
				character_model.travel("run", clampf(speed / sprint_speed, 0.5, 1.5))
			else:
				character_model.travel("idle")


# --- State helpers ---


func _enter_state(new_state: State) -> void:
	match state:
		State.DASH:
			_is_invincible = false
	state = new_state


# --- Delegate wrappers for test compatibility ---


func _start_spell(slot: int) -> void:
	spells.start_spell(slot)


func _start_dash() -> void:
	movement.start_dash()


func _process_casting(delta: float) -> void:
	spells.process_casting(delta)


func _process_dash(delta: float) -> void:
	movement.process_dash(delta)


func _process_stagger() -> void:
	movement.process_stagger()


func _update_blade_visual(delta: float) -> void:
	blades.update_blade_visual(delta)


func _toggle_lock_on() -> void:
	cam.toggle_lock_on()


func _find_lock_target() -> Node3D:
	return cam.find_lock_target()


func _update_camera() -> void:
	cam.update_camera()


func _apply_camera_collision(from: Vector3, to: Vector3) -> Vector3:
	return cam.apply_camera_collision(from, to)


func _get_camera_wish_dir() -> Vector3:
	return movement.get_camera_wish_dir()


func _process_move(delta: float) -> void:
	movement.process_move(delta)


func _face_direction(dir: Vector3, delta: float) -> void:
	movement.face_direction(dir, delta)


func _face_target(delta: float) -> void:
	movement.face_target(delta)


func _face_attack_direction(delta: float) -> void:
	movement.face_attack_direction(delta)


func _show_body_flash() -> void:
	cam.show_body_flash()


func _update_flash(delta: float) -> void:
	cam.update_flash(delta)


func _update_animation() -> void:
	cam.update_animation()
