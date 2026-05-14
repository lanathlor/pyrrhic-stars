extends CharacterBody3D

## Vanguard — Souls-like third-person melee controller (Blade spec).
## Combo chains, dodge rolls with i-frames, block/parry, stamina, lock-on.
## Sub-systems: combat, movement, cam, anim (child nodes).

signal died

enum State {
	MOVE,
	DODGE,
	LIGHT_1,
	LIGHT_2,
	LIGHT_3,
	HEAVY_WINDUP,
	HEAVY,
	BLOCK,
	STAGGER,
	DEAD,
	BLADE_SWIRL,
	GROUND_SLAM_WINDUP,
	GROUND_SLAM
}

const PlayerTelegraph := preload("res://scenes/shared/telegraph/player_telegraph.gd")

# Blade Swirl (Q)
const BLADE_SWIRL_DURATION: float = 1.5
const BLADE_SWIRL_COOLDOWN: float = 10.0
const BLADE_SWIRL_STAMINA: float = 25.0
const BLADE_SWIRL_SPEED_MULT: float = 0.8

# Ground Slam (E)
const GROUND_SLAM_COOLDOWN: float = 8.0
const GROUND_SLAM_STAMINA: float = 20.0
const GROUND_SLAM_WINDUP_TIME: float = 0.3
const GROUND_SLAM_HIT_TIME: float = 0.1

# Telegraph color
const VANGUARD_TELEGRAPH_COLOR := Color(0.9, 0.6, 0.3, 0.4)

# Network
const NET_INTERP_SPEED := 15.0

const WEAPON_SCENE := "res://assets/models/weapons/weapon_longsword.glb"

const CombatScript := preload("res://scenes/controllers/vanguard/vanguard_combat.gd")
const MovementScript := preload("res://scenes/controllers/vanguard/vanguard_movement.gd")
const CameraScript := preload("res://scenes/controllers/vanguard/vanguard_camera.gd")
const AnimScript := preload("res://scenes/controllers/vanguard/vanguard_animation.gd")
const VfxScript := preload("res://scenes/controllers/vanguard/vfx/vanguard_vfx.gd")

# Movement
@export var run_speed: float = 5.0
@export var sprint_speed: float = 7.0
@export var mouse_sensitivity: float = 0.003
@export var ground_accel: float = 20.0
@export var ground_decel: float = 15.0
@export var air_accel: float = 2.0
@export var air_decel: float = 1.0
@export var rotation_speed: float = 10.0

# Dodge
@export var dodge_speed: float = 12.0
@export var dodge_duration: float = 0.4
@export var dodge_iframe_duration: float = 0.15
@export var dodge_stamina_cost: float = 20.0

# Combat — light attacks (3-hit combo, escalating damage)
@export var light_damage_1: float = 30.0
@export var light_damage_2: float = 35.0
@export var light_damage_3: float = 55.0
@export var light_duration_1: float = 0.55
@export var light_duration_2: float = 0.65
@export var light_duration_3: float = 0.75
@export var light_combo_window: float = 0.4
@export var light_stamina_cost: float = 10.0

# Combat — heavy attack
@export var heavy_damage: float = 75.0
@export var heavy_windup_time: float = 0.5
@export var heavy_attack_duration: float = 0.3
@export var heavy_stamina_cost: float = 20.0

# Melee hit detection
@export var melee_range: float = 3.0
@export var melee_arc_degrees: float = 120.0
@export var attack_move_speed_mult: float = 0.55

# Block / parry
@export var block_damage_reduction: float = 0.8
@export var parry_window: float = 0.15
@export var block_stamina_drain: float = 15.0

# Stamina
@export var stamina_regen_rate: float = 30.0
@export var stamina_regen_delay: float = 0.6

# Camera
@export var camera_distance: float = 6.0
@export var camera_height_offset: float = 2.0

# Health & Stamina
var health: float = 200.0
var max_health: float = 200.0
var peer_id: int = 0
var stamina: float = 100.0
var max_stamina: float = 100.0

# State
var state: State = State.MOVE

# Sub-systems
var combat: Node
var movement: Node
var cam: Node
var anim: Node
var vfx: Node

var _block_cooldown: float = 0.0
var _state_timer: float = 0.0
var _combo_window_timer: float = 0.0
var _is_invincible: bool = false
var _parry_timer: float = 0.0
var _queued_light: bool = false
var _stagger_duration: float = 0.3

# Blade Swirl (Q)
var _blade_swirl_timer: float = 0.0
var _blade_swirl_cooldown: float = 0.0

# Ground Slam (E)
var _ground_slam_cooldown: float = 0.0

# Camera
var _camera_yaw: float = 0.0
var _camera_pitch: float = -0.3

# Lock-on
var _lock_target: Node3D = null
var _lock_on_active: bool = false

var _gravity: float = 8.5
var _alive: bool = true

# Network sync
var _visual_state: int = 0
var _net_position: Vector3 = Vector3.ZERO
var _net_rotation_y: float = 0.0
var _prev_remote_vs: int = -1

@onready var camera: Camera3D = $Camera3D
@onready var character_model: Node3D = $CharacterModel
@onready var hud: Control = $HUDLayer/VanguardHUD


func _ready() -> void:
	# Create sub-systems
	combat = _add_subsystem("Combat", CombatScript)
	movement = _add_subsystem("Movement", MovementScript)
	cam = _add_subsystem("Cam", CameraScript)
	anim = _add_subsystem("Anim", AnimScript)
	vfx = _add_subsystem("Vfx", VfxScript)

	GameManager.register_player(self)
	_net_position = global_position
	_net_rotation_y = rotation.y
	camera.top_level = true

	# Set up animation state machine
	(
		character_model
		. setup_state_machine(
			{
				"idle": "sword_idle",
				"run": "sword_run",
				"jump": "sword_jump",
				"dodge": "roll",
				"light_1": "sword_slash_1",
				"light_2": "sword_slash_2",
				"light_3": "sword_slash_3",
				"heavy": "sword_heavy",
				"block": "sword_block",
				"stagger": "sword_impact",
				"blade_swirl": "sword_heavy",
				"ground_slam": "sword_heavy",
				"dead": "sword_idle",
			}
		)
	)

	if _is_local():
		Input.set_mouse_mode(Input.MOUSE_MODE_CAPTURED)
	else:
		$HUDLayer.visible = false
		camera.current = false
		set_process_unhandled_input(false)

	anim.attach_weapon.call_deferred()


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
		var server_stamina: float = data.get("stamina", -1.0)
		if server_stamina >= 0.0:
			stamina = server_stamina
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
func on_hit_confirmed(_amount: float, hit_pos: Vector3 = Vector3.ZERO) -> void:
	hud.show_hit_marker()
	if hit_pos != Vector3.ZERO:
		vfx.spawn_hit_impact(hit_pos)


## Visual-only damage feedback (called from main.gd on DamageEvent).
func on_damage_visual(_amount: float, _hit_pos: Vector3) -> void:
	if _parry_timer > 0.0 and state == State.BLOCK:
		vfx.spawn_parry_flash()
	hud.show_damage_flash()
	anim.show_body_flash()


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
	anim.update_flash(delta)
	cam.update_camera()
	movement.update_stamina(delta)
	_update_parry(delta)
	_block_cooldown = maxf(_block_cooldown - delta, 0.0)
	_blade_swirl_cooldown = maxf(_blade_swirl_cooldown - delta, 0.0)
	_ground_slam_cooldown = maxf(_ground_slam_cooldown - delta, 0.0)

	match state:
		State.MOVE:
			movement.process_move(delta)
		State.DODGE:
			combat.process_dodge(delta)
		State.LIGHT_1, State.LIGHT_2, State.LIGHT_3:
			combat.process_light_attack(delta)
		State.HEAVY_WINDUP:
			combat.process_heavy_windup(delta)
		State.HEAVY:
			combat.process_heavy_attack(delta)
		State.BLOCK:
			combat.process_block(delta)
		State.STAGGER:
			combat.process_stagger()
		State.BLADE_SWIRL:
			combat.process_blade_swirl(delta)
		State.GROUND_SLAM_WINDUP:
			combat.process_ground_slam_windup(delta)
		State.GROUND_SLAM:
			combat.process_ground_slam(delta)
		State.DEAD:
			velocity.x = 0.0
			velocity.z = 0.0

	move_and_slide()
	if global_position.y < -250.0:
		global_position.y = -199.0

	anim.update_animation()
	anim.update_weapon_visual()
	# Clear lock if target is dead, freed, or hidden
	if _lock_on_active and _lock_target:
		if (
			not is_instance_valid(_lock_target)
			or not _lock_target.visible
			or ("_server_alive" in _lock_target and not _lock_target._server_alive)
		):
			_toggle_lock_on()
	if _lock_on_active and _lock_target:
		hud.update_lock_on(_lock_target, camera)
	(
		hud
		. update_spells(
			[
				{
					name = "Light Attack",
					keybind = "LMB",
					desc = "3-hit combo. 30/35/55 dmg.",
					cooldown = 0.0,
					cooldown_max = 0.0,
					stamina_cost = light_stamina_cost
				},
				{
					name = "Heavy Attack",
					keybind = "R",
					desc = "75 dmg. 0.5s windup.",
					cooldown = 0.0,
					cooldown_max = 0.0,
					stamina_cost = heavy_stamina_cost
				},
				{
					name = "Block",
					keybind = "RMB",
					desc = "80→50% DR. 0.15s parry. 3s CD.",
					cooldown = _block_cooldown,
					cooldown_max = 3.0
				},
				{
					name = "Dodge",
					keybind = "Space",
					desc = "I-frame dodge.",
					cooldown = 0.0,
					cooldown_max = 0.0,
					stamina_cost = dodge_stamina_cost
				},
				{
					name = "Blade Swirl",
					keybind = "F",
					desc = "AoE spinning attack. 10s CD.",
					cooldown = _blade_swirl_cooldown,
					cooldown_max = BLADE_SWIRL_COOLDOWN,
					stamina_cost = BLADE_SWIRL_STAMINA
				},
				{
					name = "Ground Slam",
					keybind = "T",
					desc = "Cone AoE slam. 8s CD.",
					cooldown = _ground_slam_cooldown,
					cooldown_max = GROUND_SLAM_COOLDOWN,
					stamina_cost = GROUND_SLAM_STAMINA
				},
			]
		)
	)

	# Send position + visual state to server
	if NetworkManager.is_active:
		NetworkManager.send_player_position(global_position, rotation.y, _visual_state)


# --- Parry timer (shared between combat states) ---


func _update_parry(delta: float) -> void:
	if _parry_timer > 0.0:
		_parry_timer -= delta


# --- Damage (server-authoritative) ---


func take_damage(_amount: float, _hit_position: Vector3 = Vector3.ZERO) -> void:
	pass  # Server handles all damage


func _drive_remote_animation(prev_pos: Vector3, delta: float) -> void:
	var vs_changed: bool = _visual_state != _prev_remote_vs

	# VFX transitions on state change
	if vs_changed:
		_drive_remote_vfx(_prev_remote_vs, _visual_state)
		_prev_remote_vs = _visual_state

	match _visual_state:
		NetSerializer.VS_DODGE:
			character_model.travel("dodge")
		NetSerializer.VS_VG_LIGHT_1:
			character_model.travel("light_1")
		NetSerializer.VS_VG_LIGHT_2:
			character_model.travel("light_2")
		NetSerializer.VS_VG_LIGHT_3:
			character_model.travel("light_3")
		NetSerializer.VS_VG_HEAVY_WINDUP, NetSerializer.VS_VG_HEAVY:
			character_model.travel("heavy")
		NetSerializer.VS_VG_BLOCK:
			character_model.travel("block")
		NetSerializer.VS_VG_STAGGER:
			character_model.travel("stagger")
		NetSerializer.VS_VG_BLADE_SWIRL:
			character_model.travel("blade_swirl", 2.0)
		NetSerializer.VS_VG_GROUND_SLAM_WINDUP, NetSerializer.VS_VG_GROUND_SLAM:
			character_model.travel("ground_slam")
		NetSerializer.VS_DEAD:
			character_model.travel("dead")
		_:  # VS_MOVE or unknown — derive from velocity
			var vel := (global_position - prev_pos) / delta if delta > 0 else Vector3.ZERO
			var speed := Vector2(vel.x, vel.z).length()
			if speed > 0.5:
				character_model.travel("run", clampf(speed / sprint_speed, 0.5, 1.5))
			else:
				character_model.travel("idle")


func _drive_remote_vfx(old_vs: int, new_vs: int) -> void:
	# Stop effects from previous state
	var attack_states := [
		NetSerializer.VS_VG_LIGHT_1,
		NetSerializer.VS_VG_LIGHT_2,
		NetSerializer.VS_VG_LIGHT_3,
		NetSerializer.VS_VG_HEAVY_WINDUP,
		NetSerializer.VS_VG_HEAVY,
	]
	if old_vs in attack_states and new_vs not in attack_states:
		vfx.stop_swing_trail()
	if old_vs == NetSerializer.VS_VG_BLOCK and new_vs != NetSerializer.VS_VG_BLOCK:
		vfx.hide_block_shield()
	if old_vs == NetSerializer.VS_VG_BLADE_SWIRL and new_vs != NetSerializer.VS_VG_BLADE_SWIRL:
		vfx.stop_blade_swirl()

	# Start effects for new state
	if new_vs in attack_states and old_vs not in attack_states:
		vfx.start_swing_trail()
	if new_vs == NetSerializer.VS_VG_BLOCK and old_vs != NetSerializer.VS_VG_BLOCK:
		vfx.show_block_shield()
	if new_vs == NetSerializer.VS_VG_BLADE_SWIRL and old_vs != NetSerializer.VS_VG_BLADE_SWIRL:
		vfx.start_blade_swirl()
	if (
		new_vs == NetSerializer.VS_VG_GROUND_SLAM
		and old_vs == NetSerializer.VS_VG_GROUND_SLAM_WINDUP
	):
		vfx.spawn_ground_slam_shockwave(global_position, rotation.y)


# --- State helpers ---


func _enter_state(new_state: State) -> void:
	match state:
		State.DODGE:
			_is_invincible = false
	state = new_state
	combat._has_hit_this_attack = false


# --- Delegate wrappers for test/bot compatibility ---


func _consume_stamina(amount: float) -> void:
	movement.consume_stamina(amount)


func _start_dodge() -> void:
	combat.start_dodge()


func _process_dodge(delta: float) -> void:
	combat.process_dodge(delta)


func _start_light_attack(combo_step: int) -> void:
	combat.start_light_attack(combo_step)


func _process_light_attack(delta: float) -> void:
	combat.process_light_attack(delta)


func _get_current_light_damage() -> float:
	return combat.get_current_light_damage()


func _get_current_light_duration() -> float:
	return combat.get_current_light_duration()


func _get_next_combo_step() -> int:
	return combat.get_next_combo_step()


func _start_heavy_attack() -> void:
	combat.start_heavy_attack()


func _process_heavy_windup(delta: float) -> void:
	combat.process_heavy_windup(delta)


func _process_heavy_attack(delta: float) -> void:
	combat.process_heavy_attack(delta)


func _process_block(delta: float) -> void:
	combat.process_block(delta)


func _process_stagger() -> void:
	combat.process_stagger()


func _start_blade_swirl() -> void:
	combat.start_blade_swirl()


func _process_blade_swirl(delta: float) -> void:
	combat.process_blade_swirl(delta)


func _start_ground_slam() -> void:
	combat.start_ground_slam()


func _process_ground_slam_windup(delta: float) -> void:
	combat.process_ground_slam_windup(delta)


func _process_ground_slam(delta: float) -> void:
	combat.process_ground_slam(delta)


func _perform_melee_hit(damage: float) -> void:
	combat.perform_melee_hit(damage)


func _toggle_lock_on() -> void:
	cam.toggle_lock_on()


func _find_lock_target() -> Node3D:
	return cam.find_lock_target()


func _update_camera() -> void:
	cam.update_camera()


func _apply_camera_collision(from: Vector3, to: Vector3) -> Vector3:
	return cam.apply_camera_collision(from, to)


func _nearest_enemy_distance() -> float:
	return cam.nearest_enemy_distance()


func _update_animation() -> void:
	anim.update_animation()


func _update_weapon_visual() -> void:
	anim.update_weapon_visual()


func _show_body_flash() -> void:
	anim.show_body_flash()


func _update_flash(delta: float) -> void:
	anim.update_flash(delta)


func _get_camera_wish_dir() -> Vector3:
	return movement.get_camera_wish_dir()


func _process_move(delta: float) -> void:
	movement.process_move(delta)


func _update_stamina(delta: float) -> void:
	movement.update_stamina(delta)


func _face_direction(dir: Vector3, delta: float) -> void:
	movement.face_direction(dir, delta)


func _face_target(delta: float) -> void:
	movement.face_target(delta)


func _face_attack_direction(delta: float) -> void:
	movement.face_attack_direction(delta)


func _apply_attack_movement(delta: float) -> void:
	movement.apply_attack_movement(delta)
