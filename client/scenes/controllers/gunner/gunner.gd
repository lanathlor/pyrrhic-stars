extends CharacterBody3D

## FPS player controller for the Gunner class.
## WASD movement, mouse look, hitscan raycast gun.
## Sub-systems: weapon, abilities (child nodes).

signal died

# Overclock
const OVERCLOCK_DURATION: float = 7.0
const OVERCLOCK_COOLDOWN: float = 15.0
const OVERCLOCK_FIRE_RATE: float = 0.10
const OVERCLOCK_SPEED_MULT: float = 1.3

# Rechamber
const RECHAMBER_WINDUP: float = 0.6
const RECHAMBER_WINDOW: float = 0.35
const RECHAMBER_LOCKOUT: float = 0.8
const RECHAMBER_BUFF_DURATION: float = 4.0

# Network
const NET_INTERP_SPEED := 15.0

const WEAPON_SCENE := "res://assets/models/weapons/weapon_rifle.glb"

const WeaponScript := preload("res://scenes/controllers/gunner/gunner_weapon.gd")
const AbilitiesScript := preload("res://scenes/controllers/gunner/gunner_abilities.gd")

# Movement — tuned toward Halo 3 feel
# H3: no sprint, 7.69 m/s measured base, weighty inertia
@export var walk_speed: float = 5.5
@export var sprint_speed: float = 7.7
@export var jump_velocity: float = 4.0
@export var mouse_sensitivity: float = 0.002
@export var ground_accel: float = 25.0  # ~0.22s to full speed
@export var ground_decel: float = 18.0  # ~0.31s to stop — visible slide
@export var air_accel: float = 2.5  # nearly committed to jump trajectory
@export var air_decel: float = 1.0  # almost nothing — momentum carries

# Gun
@export var fire_rate: float = 0.18
@export var gun_damage: float = 10.0

# Weapon attachment (tweak in inspector while running)
@export var _weapon_offset_pos := Vector3(0.0, 0.1, 0.0)
@export var _weapon_offset_rot_deg := Vector3(180.0, 90.0, 0.0)

# Viewmodel (FPS weapon view)
@export var _vm_pos := Vector3(0.2, -0.2, -0.25)
@export var _vm_rot_deg := Vector3(90.0, 180.0, 180.0)
@export var _vm_scale := Vector3(0.8, 0.8, 0.8)

# Dodge roll
@export var roll_speed: float = 14.0
@export var roll_duration: float = 0.3
@export var roll_cooldown: float = 2.5

# Health
var health: float = 150.0
var max_health: float = 150.0
# Network identity (set by main.gd before add_child)
var peer_id: int = 0

# Sub-systems
var weapon: Node
var abilities: Node

var _alive: bool = true
var _fire_cooldown: float = 0.0
var _gravity: float = 8.5

# Roll state
var _is_rolling: bool = false
var _roll_timer: float = 0.0
var _roll_cooldown_timer: float = 0.0
var _roll_direction: Vector3 = Vector3.ZERO

# Overclock state
var _overclock_active: bool = false
var _overclock_timer: float = 0.0
var _overclock_cooldown: float = 0.0

# Rechamber state
var _rechamber_phase: int = 0  # 0=idle, 1=windup, 2=timing_window, 3=lockout
var _rechamber_timer: float = 0.0
var _rechamber_buff: bool = false
var _rechamber_buff_timer: float = 0.0

# Munitions (enhanced rounds)
var _munitions: float = 5.0
var _max_munitions: float = 5.0

# Network sync
var _visual_state: int = 0
var _net_position: Vector3 = Vector3.ZERO
var _net_rotation_y: float = 0.0

# Remote fire detection
var _net_aim_pitch: float = 0.0
var _net_state: int = 0  # track remote state for attack transition detection

@onready var head: Node3D = $Head
@onready var camera: Camera3D = $Head/Camera3D
@onready var gun_ray: RayCast3D = $Head/GunRay
@onready var muzzle_light: OmniLight3D = $Head/MuzzleLight
@onready var hud: Control = $HUDLayer/GunnerHUD
@onready var character_model: Node3D = $CharacterModel


func _ready() -> void:
	# Create sub-systems
	weapon = _add_subsystem("Weapon", WeaponScript)
	abilities = _add_subsystem("Abilities", AbilitiesScript)

	GameManager.register_player(self)
	_net_position = global_position
	_net_rotation_y = rotation.y

	# Set up animation state machine
	(
		character_model
		. setup_state_machine(
			{
				"idle": "rifle_idle",
				"run": "rifle_run",
				"jump": "rifle_jump",
				"roll": "roll",
			}
		)
	)

	if _is_local():
		Input.set_mouse_mode(Input.MOUSE_MODE_CAPTURED)
		# FPS: hide own body so it doesn't clip into the camera
		character_model.hide_model()
		weapon.attach_weapon.call_deferred()
		weapon.setup_viewmodel.call_deferred()
	else:
		# Remote player: show model, hide HUD, disable camera
		$HUDLayer.visible = false
		camera.current = false
		set_process_unhandled_input(false)
		weapon.attach_weapon.call_deferred()


func _add_subsystem(node_name: String, script: GDScript) -> Node:
	var node: Node = script.new()
	node.name = node_name
	add_child(node)
	return node


func _process(_delta: float) -> void:
	weapon.update_weapon_live()


func _exit_tree() -> void:
	GameManager.unregister_player(self)


func _is_local() -> bool:
	if has_meta("replay_puppet"):
		return false
	if not NetworkManager.is_active:
		return true
	return peer_id == NetworkManager.get_my_id()


func _unhandled_input(event: InputEvent) -> void:
	if not _is_local():
		return
	if event is InputEventMouseMotion and Input.get_mouse_mode() == Input.MOUSE_MODE_CAPTURED:
		rotate_y(-event.relative.x * mouse_sensitivity)
		head.rotate_x(-event.relative.y * mouse_sensitivity)
		head.rotation.x = clampf(head.rotation.x, deg_to_rad(-89.0), deg_to_rad(89.0))


func _physics_process(delta: float) -> void:
	if not _is_local():
		var prev_pos := global_position
		global_position = global_position.move_toward(_net_position, 12.0 * delta)
		rotation.y = lerp_angle(rotation.y, _net_rotation_y, 8.0 * delta)
		_drive_remote_animation(prev_pos, delta)
		return

	# Dead: freeze movement and abilities, but keep sending position
	if not _alive:
		velocity = Vector3.ZERO
		if NetworkManager.is_active:
			NetworkManager.send_player_position(global_position, rotation.y, _visual_state)
		return

	_roll_cooldown_timer = maxf(_roll_cooldown_timer - delta, 0.0)
	_apply_gravity(delta)

	if _is_rolling:
		abilities.process_roll(delta)
	else:
		_handle_jump()
		abilities.handle_dodge()
		_handle_movement(delta)

	move_and_slide()

	if not _is_rolling and not Input.is_action_pressed("sprint"):
		weapon.handle_shooting(delta)
		abilities.handle_overclock(delta)
		abilities.handle_rechamber(delta)

	weapon.update_muzzle_flash(delta)
	weapon.update_viewmodel(delta)
	_update_animation()
	(
		hud
		. update_spells(
			[
				{
					name = "Shoot",
					keybind = "LMB",
					desc = "10 dmg hitscan. 0.18s fire rate.",
					cooldown = 0.0,
					cooldown_max = 0.0
				},
				{
					name = "Roll",
					keybind = "C",
					desc = "Dodge roll with i-frames.",
					cooldown = _roll_cooldown_timer,
					cooldown_max = roll_cooldown
				},
				{
					name = "Overclock",
					keybind = "F",
					desc = "7s fire rate + speed boost.",
					cooldown = _overclock_cooldown if not _overclock_active else 0.0,
					cooldown_max = OVERCLOCK_COOLDOWN,
					active = _overclock_active,
					active_remaining = _overclock_timer,
					active_max = OVERCLOCK_DURATION
				},
				{
					name = "Rechamber",
					keybind = "T",
					desc = "Timed reload. Perfect timing = dmg buff.",
					cooldown = 0.0,
					cooldown_max = 0.0,
					active = _rechamber_buff,
					active_remaining = _rechamber_buff_timer,
					active_max = RECHAMBER_BUFF_DURATION,
					status_text = _get_rechamber_status()
				},
			]
		)
	)
	hud.update_munitions(_munitions, _max_munitions)

	# Send position + visual state to server
	if NetworkManager.is_active:
		NetworkManager.send_player_position(
			global_position, rotation.y, _visual_state, head.rotation.x
		)


func _apply_gravity(delta: float) -> void:
	if not is_on_floor():
		velocity.y -= _gravity * delta
	else:
		velocity.y = -0.5  # keep pressed to floor so is_on_floor() stays reliable


func _handle_jump() -> void:
	if Input.is_action_just_pressed("jump") and is_on_floor():
		velocity.y = jump_velocity


func _handle_movement(delta: float) -> void:
	var on_floor := is_on_floor()
	var speed: float = sprint_speed if Input.is_action_pressed("sprint") else walk_speed
	if _overclock_active:
		speed *= OVERCLOCK_SPEED_MULT
	var accel: float = ground_accel if on_floor else air_accel
	var decel: float = ground_decel if on_floor else air_decel

	var input_dir := Input.get_vector("move_left", "move_right", "move_forward", "move_backward")
	var wish_dir := (transform.basis * Vector3(input_dir.x, 0.0, input_dir.y)).normalized()

	var flat_vel := Vector3(velocity.x, 0.0, velocity.z)

	if wish_dir.length() > 0.1:
		# Accelerate toward desired direction
		var target_vel := wish_dir * speed
		flat_vel.x = move_toward(flat_vel.x, target_vel.x, accel * delta)
		flat_vel.z = move_toward(flat_vel.z, target_vel.z, accel * delta)
	else:
		# Decelerate to stop
		flat_vel.x = move_toward(flat_vel.x, 0.0, decel * delta)
		flat_vel.z = move_toward(flat_vel.z, 0.0, decel * delta)

	velocity.x = flat_vel.x
	velocity.z = flat_vel.z


## Called by main.gd when server confirms this player hit an enemy.
func on_hit_confirmed(_amount: float, _hit_pos: Vector3 = Vector3.ZERO) -> void:
	hud.show_hit_marker()


## Called by main.gd on remote gunners when a damage event confirms they hit something.
func on_hit_tracer(hit_pos: Vector3) -> void:
	weapon.spawn_tracer(weapon.get_muzzle_pos(), hit_pos)


## Called by main.gd when the server sends a DAMAGE_EVENT targeting this player.
## Health is already updated via apply_server_state -- this is visuals only.
func on_damage_visual(_amount: float, _hit_pos: Vector3) -> void:
	hud.show_damage_flash()
	character_model.flash_damage()


## Called by main.gd each tick with the authoritative world state for this player.
func apply_server_state(data: Dictionary) -> void:
	# data has: pos (Vector3), rot_y (float), health (float), state (int),
	#           visual_state (int)
	if data.has("max_health") and data["max_health"] > 0.0:
		max_health = data["max_health"]
	if _is_local():
		health = data.health
		# Sync buff states from server truth
		_overclock_active = data.get("overclock_active", false)
		_rechamber_buff = data.get("rechamber_buff", false)
		var server_phase: int = data.get("rechamber_phase", 0)
		if server_phase != _rechamber_phase and server_phase == 0:
			_rechamber_phase = 0  # server reset overrides client
		_munitions = data.get("munitions", 0.0)
		if health <= 0.0 and _alive:
			_alive = false
			died.emit()
		elif health > 0.0 and not _alive:
			_alive = true
			# Snap to server position on respawn
			global_position = data.pos
			_net_position = data.pos
	else:
		# Remote player: apply all state
		_net_position = data.pos
		_net_rotation_y = data.rot_y
		health = data.health
		_visual_state = data.get("visual_state", 0)
		_net_aim_pitch = data.get("aim_pitch", 0.0)
		var new_state: int = data.get("state", 0)
		if new_state == 2 and _net_state != 2:  # transition into attack
			weapon.fire_remote_tracer()
		_net_state = new_state


func _update_animation() -> void:
	if _is_rolling:
		_visual_state = NetSerializer.VS_DODGE
		character_model.travel_timed("roll", roll_duration)
		return
	if not is_on_floor():
		_visual_state = NetSerializer.VS_AIRBORNE
		character_model.travel("jump", 2.0)
		return
	_visual_state = NetSerializer.VS_MOVE
	var flat_vel := Vector3(velocity.x, 0.0, velocity.z)
	if flat_vel.length() > 0.5:
		var speed_ratio := flat_vel.length() / sprint_speed
		character_model.travel("run", clampf(speed_ratio, 0.5, 1.5))
	else:
		character_model.travel("idle")


func _drive_remote_animation(prev_pos: Vector3, delta: float) -> void:
	match _visual_state:
		NetSerializer.VS_DODGE:
			character_model.travel("roll")
		NetSerializer.VS_AIRBORNE:
			character_model.travel("jump", 2.0)
		NetSerializer.VS_DEAD:
			character_model.travel("idle")
		_:  # VS_MOVE or unknown — derive from velocity
			var vel := (global_position - prev_pos) / delta if delta > 0 else Vector3.ZERO
			var speed := Vector2(vel.x, vel.z).length()
			if speed > 0.5:
				character_model.travel("run", clampf(speed / sprint_speed, 0.5, 1.5))
			else:
				character_model.travel("idle")


# --- Delegate wrappers for test/bot compatibility ---


func _start_roll() -> void:
	abilities.start_roll()


func _process_roll(delta: float) -> void:
	abilities.process_roll(delta)


func _handle_shooting(delta: float) -> void:
	weapon.handle_shooting(delta)


func _handle_overclock(delta: float) -> void:
	abilities.handle_overclock(delta)


func _handle_rechamber(delta: float) -> void:
	abilities.handle_rechamber(delta)


func _get_rechamber_status() -> String:
	return abilities.get_rechamber_status()


func _spawn_tracer(from_pos: Vector3, to_pos: Vector3) -> void:
	weapon.spawn_tracer(from_pos, to_pos)
