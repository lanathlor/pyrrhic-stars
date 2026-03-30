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

# Dodge roll
@export var roll_speed: float = 14.0
@export var roll_duration: float = 0.3
@export var roll_cooldown: float = 2.5

# Health
var health: float = 100.0
var max_health: float = 150.0

var _fire_cooldown: float = 0.0
var _gravity: float = 8.5

# Roll state
var _is_rolling: bool = false
var _roll_timer: float = 0.0
var _roll_cooldown_timer: float = 0.0
var _roll_direction: Vector3 = Vector3.ZERO

@onready var head: Node3D = $Head
@onready var camera: Camera3D = $Head/Camera3D
@onready var gun_ray: RayCast3D = $Head/GunRay
@onready var muzzle_light: OmniLight3D = $Head/MuzzleLight
@onready var hud: Control = $HUDLayer/GunnerHUD

var _muzzle_flash_timer: float = 0.0


func _ready() -> void:
	Input.set_mouse_mode(Input.MOUSE_MODE_CAPTURED)
	GameManager.register_player(self)
	hud.update_health(health, max_health)


func _exit_tree() -> void:
	GameManager.unregister_player(self)


func _unhandled_input(event: InputEvent) -> void:
	if event is InputEventMouseMotion and Input.get_mouse_mode() == Input.MOUSE_MODE_CAPTURED:
		rotate_y(-event.relative.x * mouse_sensitivity)
		head.rotate_x(-event.relative.y * mouse_sensitivity)
		head.rotation.x = clampf(head.rotation.x, deg_to_rad(-89.0), deg_to_rad(89.0))

	# ESC handled by main.gd pause menu


func _physics_process(delta: float) -> void:
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
	hud.update_roll_cooldown(_roll_cooldown_timer, roll_cooldown)


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
			collider.take_damage(gun_damage, hit_pos)
			hud.show_hit_marker()


func _update_muzzle_flash(delta: float) -> void:
	if _muzzle_flash_timer > 0.0:
		_muzzle_flash_timer -= delta
		if _muzzle_flash_timer <= 0.0:
			muzzle_light.visible = false


func take_damage(amount: float, _hit_position: Vector3 = Vector3.ZERO) -> void:
	health -= amount
	health = maxf(health, 0.0)
	hud.update_health(health, max_health)
	hud.show_damage_flash()
	if health <= 0.0:
		died.emit()
