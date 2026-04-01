extends CharacterBody3D

## FPS player controller for the Gunner class.
## WASD movement, mouse look, hitscan raycast gun.

signal died

# Movement — tuned toward Halo 3 feel
# H3: no sprint, 7.69 m/s measured base, weighty inertia
@export var walk_speed: float = 5.5
@export var sprint_speed: float = 7.7
@export var jump_velocity: float = 4.0
@export var mouse_sensitivity: float = 0.002
@export var ground_accel: float = 25.0  # ~0.22s to full speed
@export var ground_decel: float = 18.0  # ~0.31s to stop — visible slide
@export var air_accel: float = 2.5      # nearly committed to jump trajectory
@export var air_decel: float = 1.0      # almost nothing — momentum carries

# Gun
@export var fire_rate: float = 0.18
@export var gun_damage: float = 10.0

# Weapon attachment (tweak in inspector while running)
@export var _weapon_offset_pos := Vector3(0.0, 0.1, 0.0)
@export var _weapon_offset_rot_deg := Vector3(180.0, 90.0, 0.0)

# Dodge roll
@export var roll_speed: float = 14.0
@export var roll_duration: float = 0.3
@export var roll_cooldown: float = 2.5

# Health
var health: float = 100.0
var max_health: float = 150.0

# Network identity (set by main.gd before add_child)
var peer_id: int = 0

var _fire_cooldown: float = 0.0
var _gravity: float = 8.5

# Roll state
var _is_rolling: bool = false
var _roll_timer: float = 0.0
var _roll_cooldown_timer: float = 0.0
var _roll_direction: Vector3 = Vector3.ZERO

# Network sync
var _net_anim: String = ""
var _net_anim_speed: float = 1.0
var _net_position: Vector3 = Vector3.ZERO
var _net_rotation_y: float = 0.0
const NET_INTERP_SPEED := 15.0

const WEAPON_SCENE := "res://assets/models/weapons/weapon_rifle.glb"

@onready var head: Node3D = $Head
@onready var camera: Camera3D = $Head/Camera3D
@onready var gun_ray: RayCast3D = $Head/GunRay
@onready var muzzle_light: OmniLight3D = $Head/MuzzleLight
@onready var hud: Control = $HUDLayer/GunnerHUD
@onready var character_model: Node3D = $CharacterModel

var _muzzle_flash_timer: float = 0.0


func _ready() -> void:
	GameManager.register_player(self)
	_net_position = global_position
	_net_rotation_y = rotation.y
	NetworkManager.message_received.connect(_on_network_message)

	if _is_local():
		Input.set_mouse_mode(Input.MOUSE_MODE_CAPTURED)
		hud.update_health(health, max_health)
		# FPS: hide own body so it doesn't clip into the camera
		character_model.hide_model()
		_attach_weapon.call_deferred()
	else:
		# Remote player: show model, hide HUD, disable camera
		$HUDLayer.visible = false
		camera.current = false
		set_process_unhandled_input(false)
		_attach_weapon.call_deferred()



func _attach_weapon() -> void:
	var offset_pos := _weapon_offset_pos
	var offset_rot := Vector3(deg_to_rad(_weapon_offset_rot_deg.x), deg_to_rad(_weapon_offset_rot_deg.y), deg_to_rad(_weapon_offset_rot_deg.z))
	character_model.attach_weapon(WEAPON_SCENE, "mixamorig_RightHand", offset_pos, offset_rot)


func _process(_delta: float) -> void:
	# Live-update weapon offset from inspector while game runs
	if character_model.weapon_node:
		character_model.weapon_node.position = _weapon_offset_pos
		character_model.weapon_node.rotation = Vector3(
			deg_to_rad(_weapon_offset_rot_deg.x),
			deg_to_rad(_weapon_offset_rot_deg.y),
			deg_to_rad(_weapon_offset_rot_deg.z))


func _exit_tree() -> void:
	GameManager.unregister_player(self)


func _is_local() -> bool:
	if not NetworkManager.is_active:
		return true
	return peer_id == NetworkManager.get_my_id()


func _sync_state_to_peers() -> void:
	var payload := NetSerializer.encode_player_sync(
		_net_position, _net_rotation_y, _net_anim, _net_anim_speed, health)
	NetworkManager.send_msg(NetSerializer.OP_PLAYER_SYNC, payload)


func _on_network_message(opcode: int, sender_id: int, payload: PackedByteArray) -> void:
	if opcode == NetSerializer.OP_PLAYER_SYNC and sender_id == peer_id:
		var data := NetSerializer.decode_player_sync(payload)
		_net_position = data.pos
		_net_rotation_y = data.rot_y
		_net_anim = data.anim
		_net_anim_speed = data.anim_speed
		health = data.hp
	elif opcode == NetSerializer.OP_DAMAGE:
		var data := NetSerializer.decode_damage(payload)
		if data.target_peer == peer_id and _is_local():
			_apply_damage(data.amount, data.hit_pos)
	elif opcode == NetSerializer.OP_NET_FLASH and sender_id == peer_id:
		if not _is_local():
			character_model.flash_damage()


func _unhandled_input(event: InputEvent) -> void:
	if not _is_local():
		return
	if event is InputEventMouseMotion and Input.get_mouse_mode() == Input.MOUSE_MODE_CAPTURED:
		rotate_y(-event.relative.x * mouse_sensitivity)
		head.rotate_x(-event.relative.y * mouse_sensitivity)
		head.rotation.x = clampf(head.rotation.x, deg_to_rad(-89.0), deg_to_rad(89.0))


func _physics_process(delta: float) -> void:
	if not _is_local():
		# Remote: interpolate toward synced position/rotation
		global_position = global_position.lerp(_net_position, NET_INTERP_SPEED * delta)
		rotation.y = lerp_angle(rotation.y, _net_rotation_y, NET_INTERP_SPEED * delta)
		if _net_anim != "":
			character_model.play_anim(_net_anim, _net_anim_speed)
		return

	_roll_cooldown_timer = maxf(_roll_cooldown_timer - delta, 0.0)
	_apply_gravity(delta)

	if _is_rolling:
		_process_roll(delta)
	else:
		_handle_jump()
		_handle_dodge()
		_handle_movement(delta)

	move_and_slide()

	if not _is_rolling and not Input.is_action_pressed("sprint"):
		_handle_shooting(delta)

	_update_muzzle_flash(delta)
	_update_animation()
	hud.update_roll_cooldown(_roll_cooldown_timer, roll_cooldown)

	# Sync to remote peers
	_net_position = global_position
	_net_rotation_y = rotation.y
	if NetworkManager.is_active:
		_sync_state_to_peers()


func _apply_gravity(delta: float) -> void:
	if not is_on_floor():
		velocity.y -= _gravity * delta


func _handle_jump() -> void:
	if Input.is_action_just_pressed("jump") and is_on_floor():
		velocity.y = jump_velocity


func _handle_movement(delta: float) -> void:
	var on_floor := is_on_floor()
	var speed: float = sprint_speed if Input.is_action_pressed("sprint") else walk_speed
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


func _handle_dodge() -> void:
	if Input.is_action_just_pressed("dodge") and _roll_cooldown_timer <= 0.0 and is_on_floor():
		_start_roll()


func _start_roll() -> void:
	var input_dir := Input.get_vector("move_left", "move_right", "move_forward", "move_backward")
	if input_dir.length() > 0.1:
		_roll_direction = (transform.basis * Vector3(input_dir.x, 0.0, input_dir.y)).normalized()
	else:
		# Default: roll backward (away from where you're looking)
		_roll_direction = (transform.basis * Vector3(0.0, 0.0, 1.0)).normalized()
	_is_rolling = true
	_roll_timer = roll_duration
	_roll_cooldown_timer = roll_cooldown


func _process_roll(delta: float) -> void:
	_roll_timer -= delta
	velocity.x = _roll_direction.x * roll_speed
	velocity.z = _roll_direction.z * roll_speed
	if _roll_timer <= 0.0:
		_is_rolling = false
		# Bleed off some speed coming out of roll
		velocity.x *= 0.4
		velocity.z *= 0.4


func _handle_shooting(delta: float) -> void:
	_fire_cooldown -= delta
	if Input.is_action_pressed("shoot") and _fire_cooldown <= 0.0:
		_shoot()
		_fire_cooldown = fire_rate


func _shoot() -> void:
	gun_ray.force_raycast_update()
	_muzzle_flash_timer = 0.05
	muzzle_light.visible = true
	hud.on_shoot()

	if gun_ray.is_colliding():
		var collider := gun_ray.get_collider()
		if collider.has_method("take_damage"):
			var hit_pos := gun_ray.get_collision_point()
			_deal_damage(collider, gun_damage, hit_pos)
			hud.show_hit_marker()


func _update_muzzle_flash(delta: float) -> void:
	if _muzzle_flash_timer > 0.0:
		_muzzle_flash_timer -= delta
		if _muzzle_flash_timer <= 0.0:
			muzzle_light.visible = false


func take_damage(amount: float, hit_position: Vector3 = Vector3.ZERO) -> void:
	_apply_damage(amount, hit_position)


func _apply_damage(amount: float, _hit_position: Vector3 = Vector3.ZERO) -> void:
	health -= amount
	health = maxf(health, 0.0)
	hud.update_health(health, max_health)
	hud.show_damage_flash()
	character_model.flash_damage()
	if health <= 0.0:
		died.emit()
	elif NetworkManager.is_active:
		NetworkManager.send_msg(NetSerializer.OP_NET_FLASH, NetSerializer.encode_net_flash())


func _deal_damage(target: Node, amount: float, hit_pos: Vector3) -> void:
	if not target.has_method("take_damage"):
		return
	if NetworkManager.is_active:
		var target_peer: int = target.peer_id if "peer_id" in target else 0
		NetworkManager.send_msg(NetSerializer.OP_DAMAGE,
			NetSerializer.encode_damage(target_peer, amount, hit_pos))
	else:
		target.take_damage(amount, hit_pos)


func _update_animation() -> void:
	if _is_rolling:
		_net_anim = "roll"
		_net_anim_speed = 1.0
		character_model.play_anim_timed("roll", roll_duration)
		return
	if not is_on_floor():
		_net_anim = "rifle_jump"
		_net_anim_speed = 2.0
		character_model.play_anim("rifle_jump", 2.0)
		return
	var flat_vel := Vector3(velocity.x, 0.0, velocity.z)
	if flat_vel.length() > 0.5:
		var speed_ratio := flat_vel.length() / sprint_speed
		_net_anim_speed = clampf(speed_ratio, 0.5, 1.5)
		_net_anim = "rifle_run"
		character_model.play_anim("rifle_run", _net_anim_speed)
	else:
		_net_anim = "rifle_idle"
		_net_anim_speed = 1.0
		character_model.play_anim("rifle_idle")
