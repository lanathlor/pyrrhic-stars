extends CharacterBody3D

## Arcanotechnicien -- Tactical Flux channeling controller.
## Harmonist spec: positional healer with Zone, Beam, and Direct delivery methods.
## Confluence mechanic: stacking spell power from consecutive casts.
## Sub-systems: combat, movement, cam (child nodes).

signal died

enum State {
	MOVE,
	DODGE,
	CASTING,
	CHANNELING,
	STAGGER,
	DEAD,
}

const NET_INTERP_SPEED := 15.0

# Input actions mapped to spell slots 0-5
const SPELL_SLOT_ACTIONS: Array[StringName] = [
	&"light_attack",   # slot 0 -- LMB
	&"block",          # slot 1 -- RMB
	&"heavy_attack",   # slot 2 -- R
	&"ability_2",      # slot 3 -- T
	&"ability_1",      # slot 4 -- F
	&"dodge",          # slot 5 -- C (overloaded: dodge when no target, spell when targeting)
]

## Harmonist spell table. action_id = 50 + slot_index.
const HARMONIST_SPELLS: Array[Dictionary] = [
	{
		name = "Mending Surge",
		keybind = "LMB",
		desc = "Direct. Massive single-target emergency heal. High Flux cost.",
		action_id = 50,
		dur = 0.4,
		delivery = "direct",
		cooldown_max = 0.0,
	},
	{
		name = "Mending Beam",
		keybind = "RMB",
		desc = "Beam. High sustained single-target throughput. Channel.",
		action_id = 51,
		dur = 2.0,
		delivery = "beam",
		cooldown_max = 0.0,
	},
	{
		name = "Life Swap",
		keybind = "R",
		desc = "Direct. Drain healthy ally to empower next heal. Low Flux.",
		action_id = 52,
		dur = 0.3,
		delivery = "direct",
		cooldown_max = 6.0,
	},
	{
		name = "Transfusion",
		keybind = "T",
		desc = "Beam to Zone. Drain one ally, AoE heal everyone else.",
		action_id = 53,
		dur = 1.5,
		delivery = "zone",
		cooldown_max = 8.0,
	},
	{
		name = "Frost Ward",
		keybind = "F",
		desc = "Instant. Frost barrier on ally. Absorbs damage.",
		action_id = 54,
		dur = 0.2,
		delivery = "direct",
		cooldown_max = 12.0,
	},
	{
		name = "Gust Step",
		keybind = "C",
		desc = "Instant. Wind-propelled repositioning.",
		action_id = 55,
		dur = 0.3,
		delivery = "displacement",
		cooldown_max = 10.0,
	},
]

const MovementScript := preload("res://scenes/controllers/arcanotechnicien/arcanotechnicien_movement.gd")
const CombatScript := preload("res://scenes/controllers/arcanotechnicien/harmonist_combat.gd")
const CameraScript := preload("res://scenes/controllers/arcanotechnicien/arcanotechnicien_camera.gd")

# Movement
@export var run_speed: float = 5.5
@export var sprint_speed: float = 8.0
@export var mouse_sensitivity: float = 0.003
@export var ground_accel: float = 20.0
@export var ground_decel: float = 15.0
@export var air_accel: float = 2.0
@export var air_decel: float = 1.0
@export var rotation_speed: float = 10.0

# Dodge
@export var dodge_speed: float = 11.0
@export var dodge_duration: float = 0.35
@export var dodge_iframe_duration: float = 0.12

# Casting
@export var cast_range: float = 25.0
@export var cast_move_speed_mult: float = 0.35

# GCD
@export var gcd_duration: float = 0.8

# Camera -- pulled back for spatial awareness
@export var camera_distance: float = 8.0
@export var camera_height_offset: float = 2.5

# Health & Flux
var health: float = 150.0
var max_health: float = 150.0
var flux: float = 100.0
var max_flux: float = 100.0
var peer_id: int = 0

# State
var state: State = State.MOVE

# Sub-systems
var combat: Node
var movement: Node
var cam: Node

var spec_id: String = "harmonist"

var _state_timer: float = 0.0
var _gcd_timer: float = 0.0
var _cast_timer: float = 0.0
var _casting_spell: Dictionary = {}
var _is_invincible: bool = false

# Confluence (shared Arcanotechnicien mechanic) -- read from server state
var _confluence_tier: int = 0
var _confluence_stacks: int = 0

# Cooldowns per spell slot
var _cooldowns: Array[float] = [0.0, 0.0, 0.0, 0.0, 0.0, 0.0]

# Camera
var _camera_yaw: float = 0.0
var _camera_pitch: float = -0.3

# Lock-on (for ally targeting)
var _lock_target: Node3D = null
var _lock_on_active: bool = false

var _gravity: float = 8.5
var _alive: bool = true

# Network sync
var _visual_state: int = 0
var _net_position: Vector3 = Vector3.ZERO
var _net_rotation_y: float = 0.0

@onready var camera: Camera3D = $Camera3D
@onready var character_model: Node3D = $CharacterModel
@onready var hud: Control = $HUDLayer/ArcanotechnicienHUD


func _ready() -> void:
	# Create sub-systems
	combat = _add_subsystem("Combat", CombatScript)
	movement = _add_subsystem("Movement", MovementScript)
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
				"dodge": "roll",
				"casting": "slash",
				"channeling": "slash",
				"stagger": "idle",
				"dead": "idle",
			}
		)
	)

	if _is_local():
		Input.set_mouse_mode(Input.MOUSE_MODE_CAPTURED)
		_update_hud_spells()
	else:
		$HUDLayer.visible = false
		camera.current = false
		set_process_unhandled_input(false)


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
	# Confluence uses the same wire bytes as onslaught (reused field).
	_confluence_tier = data.get("onslaught_tier", 0)
	_confluence_stacks = data.get("onslaught_stacks", 0)
	if _is_local():
		health = data.health
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


## Called by main.gd when server confirms this player hit an enemy.
func on_hit_confirmed(_amount: float, _hit_pos: Vector3 = Vector3.ZERO) -> void:
	hud.show_hit_marker()


## Visual-only damage feedback (called from main.gd on DamageEvent).
func on_damage_visual(_amount: float, _hit_pos: Vector3) -> void:
	hud.show_damage_flash()
	cam.show_body_flash()


## Visual-only heal feedback.
func on_heal_visual(_amount: float, _hit_pos: Vector3) -> void:
	hud.show_heal_flash()


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
		return

	if not is_on_floor():
		velocity.y -= _gravity * delta
	else:
		velocity.y = -0.5

	_state_timer -= delta
	_gcd_timer -= delta
	cam.update_camera()

	# Tick cooldowns
	for i in _cooldowns.size():
		_cooldowns[i] = maxf(_cooldowns[i] - delta, 0.0)

	match state:
		State.MOVE:
			movement.process_move(delta)
		State.DODGE:
			combat.process_dodge(delta)
		State.CASTING:
			combat.process_casting(delta)
		State.CHANNELING:
			combat.process_channeling(delta)
		State.STAGGER:
			combat.process_stagger()
		State.DEAD:
			velocity.x = 0.0
			velocity.z = 0.0

	move_and_slide()
	if global_position.y < -250.0:
		global_position.y = -199.0

	cam.update_animation()
	# Clear lock if target is dead, freed, or hidden
	if _lock_on_active and _lock_target:
		if (
			not is_instance_valid(_lock_target)
			or not _lock_target.visible
		):
			cam.toggle_lock_on()
	if _lock_on_active and _lock_target:
		hud.update_lock_on(_lock_target, camera)

	_update_hud_spells()
	hud.update_gcd(_gcd_timer / gcd_duration if _gcd_timer > 0.0 else 0.0)
	hud.update_confluence(_confluence_tier, _confluence_stacks)

	# Send position + visual state to server
	if NetworkManager.is_active:
		NetworkManager.send_player_position(global_position, rotation.y, _visual_state)


func _update_hud_spells() -> void:
	var spell_data: Array = []
	for i in HARMONIST_SPELLS.size():
		var spell: Dictionary = HARMONIST_SPELLS[i]
		spell_data.append({
			name = spell.name,
			keybind = spell.keybind,
			desc = spell.desc,
			cooldown = _cooldowns[i],
			cooldown_max = spell.cooldown_max,
		})
	hud.update_spells(spell_data)


# --- Damage (server-authoritative) ---


func take_damage(_amount: float, _hit_position: Vector3 = Vector3.ZERO) -> void:
	pass  # Server handles all damage


func _drive_remote_animation(prev_pos: Vector3, delta: float) -> void:
	match _visual_state:
		NetSerializer.VS_DODGE:
			character_model.travel("dodge")
		NetSerializer.VS_AT_CASTING:
			character_model.travel("casting")
		NetSerializer.VS_AT_CHANNELING:
			character_model.travel("channeling")
		NetSerializer.VS_AT_STAGGER:
			character_model.travel("stagger")
		NetSerializer.VS_DEAD:
			character_model.travel("dead")
		_:  # VS_MOVE or unknown -- derive from velocity
			var vel := (global_position - prev_pos) / delta if delta > 0 else Vector3.ZERO
			var speed := Vector2(vel.x, vel.z).length()
			if speed > 0.5:
				character_model.travel("run", clampf(speed / sprint_speed, 0.5, 1.5))
			else:
				character_model.travel("idle")


# --- State helpers ---


func _enter_state(new_state: State) -> void:
	match state:
		State.DODGE:
			_is_invincible = false
	state = new_state


# --- Delegate wrappers for test/bot compatibility ---


func _start_spell(slot: int) -> void:
	combat.start_spell(slot)


func _start_dodge() -> void:
	combat.start_dodge()


func _process_dodge(delta: float) -> void:
	combat.process_dodge(delta)


func _process_casting(delta: float) -> void:
	combat.process_casting(delta)


func _process_channeling(delta: float) -> void:
	combat.process_channeling(delta)


func _process_stagger() -> void:
	combat.process_stagger()


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


## Stub for spec switching (only harmonist implemented for now).
func _switch_spec(new_spec: String, _from_client: bool = false) -> void:
	if new_spec == spec_id:
		return
	spec_id = new_spec
	# Future: swap combat subsystem when Destroyer/Battlemage are implemented
