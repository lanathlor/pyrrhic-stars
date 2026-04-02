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

# Viewmodel (FPS weapon view)
@export var _vm_pos := Vector3(0.2, -0.2, -0.25)
@export var _vm_rot_deg := Vector3(90.0, 180.0, 180.0)
@export var _vm_scale := Vector3(0.8, 0.8, 0.8)

# Dodge roll
@export var roll_speed: float = 14.0
@export var roll_duration: float = 0.3
@export var roll_cooldown: float = 2.5

# Health
var health: float = 100.0
var max_health: float = 150.0
var _alive: bool = true

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

# Viewmodel state
var _viewmodel: Node3D
var _viewmodel_weapon: Node3D
var _recoil_offset: float = 0.0
var _bob_time: float = 0.0


# Remote fire detection
var _net_aim_pitch: float = 0.0
var _net_state: int = 0  # track remote state for attack transition detection


func _ready() -> void:
	GameManager.register_player(self)
	_net_position = global_position
	_net_rotation_y = rotation.y
	if _is_local():
		Input.set_mouse_mode(Input.MOUSE_MODE_CAPTURED)
		# FPS: hide own body so it doesn't clip into the camera
		character_model.hide_model()
		_attach_weapon.call_deferred()
		_setup_viewmodel.call_deferred()
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


func _setup_viewmodel() -> void:
	_viewmodel = Node3D.new()
	_viewmodel.name = "Viewmodel"
	camera.add_child(_viewmodel)
	_viewmodel.position = _vm_pos
	_viewmodel.rotation = Vector3(
		deg_to_rad(_vm_rot_deg.x),
		deg_to_rad(_vm_rot_deg.y),
		deg_to_rad(_vm_rot_deg.z))
	_viewmodel.scale = _vm_scale

	var weapon_scene := load(WEAPON_SCENE) as PackedScene
	if weapon_scene:
		_viewmodel_weapon = weapon_scene.instantiate()
		_viewmodel.add_child(_viewmodel_weapon)


## Spawn a bullet tracer line in world space from origin to end point.
func _spawn_tracer(from_pos: Vector3, to_pos: Vector3) -> void:
	var diff := to_pos - from_pos
	var length := diff.length()
	if length < 0.1:
		return

	var dir := diff.normalized()
	var mid := (from_pos + to_pos) / 2.0

	# Build transform manually — no need for look_at or being in tree
	var tracer := MeshInstance3D.new()
	var box := BoxMesh.new()
	box.size = Vector3(0.03, 0.03, length)
	tracer.mesh = box
	tracer.cast_shadow = GeometryInstance3D.SHADOW_CASTING_SETTING_OFF

	var mat := StandardMaterial3D.new()
	mat.albedo_color = Color(1.0, 0.95, 0.6, 0.7)
	mat.emission_enabled = true
	mat.emission = Color(1.0, 0.85, 0.3)
	mat.emission_energy_multiplier = 6.0
	mat.shading_mode = BaseMaterial3D.SHADING_MODE_UNSHADED
	mat.transparency = BaseMaterial3D.TRANSPARENCY_ALPHA
	tracer.material_override = mat

	# Orient along the line using a manual basis
	var up := Vector3.UP
	if absf(dir.dot(up)) > 0.99:
		up = Vector3.RIGHT
	var z_axis := -dir
	var x_axis := up.cross(z_axis).normalized()
	var y_axis := z_axis.cross(x_axis).normalized()
	tracer.transform = Transform3D(Basis(x_axis, y_axis, z_axis), mid)

	get_tree().current_scene.add_child(tracer)

	# Fade out and free
	var tween := get_tree().create_tween()
	tween.tween_property(mat, "albedo_color:a", 0.0, 0.12)
	tween.parallel().tween_property(mat, "emission_energy_multiplier", 0.0, 0.12)
	tween.tween_callback(tracer.queue_free)


func _process(_delta: float) -> void:
	# Live-update weapon offset from inspector while game runs
	if character_model.weapon_node:
		character_model.weapon_node.position = _weapon_offset_pos
		character_model.weapon_node.rotation = Vector3(
			deg_to_rad(_weapon_offset_rot_deg.x),
			deg_to_rad(_weapon_offset_rot_deg.y),
			deg_to_rad(_weapon_offset_rot_deg.z))
	# Live-update viewmodel from inspector
	if _viewmodel and _is_local():
		if _recoil_offset <= 0.001:
			_viewmodel.rotation = Vector3(
				deg_to_rad(_vm_rot_deg.x),
				deg_to_rad(_vm_rot_deg.y),
				deg_to_rad(_vm_rot_deg.z))
		_viewmodel.scale = _vm_scale


func _exit_tree() -> void:
	GameManager.unregister_player(self)


func _is_local() -> bool:
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
		# Remote: interpolate toward synced position/rotation
		global_position = global_position.move_toward(_net_position, 12.0 * delta)
		rotation.y = lerp_angle(rotation.y, _net_rotation_y, 8.0 * delta)
		if _net_anim != "":
			character_model.play_anim(_net_anim, _net_anim_speed)
		return

	# Dead: freeze movement and abilities, but keep sending position
	if not _alive:
		velocity = Vector3.ZERO
		if NetworkManager.is_active:
			NetworkManager.send_player_position(global_position, rotation.y, _net_anim, _net_anim_speed)
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
	_update_viewmodel(delta)
	_update_animation()
	hud.update_roll_cooldown(_roll_cooldown_timer, roll_cooldown)

	# Send position + animation to server
	if NetworkManager.is_active:
		NetworkManager.send_player_position(global_position, rotation.y, _net_anim, _net_anim_speed, head.rotation.x)


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
	if Input.is_action_pressed("shoot") and _fire_cooldown <= 0.0 and Input.get_mouse_mode() == Input.MOUSE_MODE_CAPTURED:
		_shoot()
		_fire_cooldown = fire_rate


func _shoot() -> void:
	gun_ray.force_raycast_update()
	_muzzle_flash_timer = 0.05
	muzzle_light.visible = true
	hud.on_shoot()
	_recoil_offset = 0.06

	# Tracer line from weapon muzzle to hit (or max range)
	var tracer_from := _get_muzzle_pos()
	var tracer_to: Vector3
	if gun_ray.is_colliding():
		tracer_to = gun_ray.get_collision_point()
	else:
		tracer_to = head.global_position + head.global_transform.basis * Vector3(0, 0, -100)
	_spawn_tracer(tracer_from, tracer_to)

	# Tell server we fired
	if NetworkManager.is_active:
		NetworkManager.send_ability(0, head.rotation.x)  # 0 = ActionShoot


func _update_muzzle_flash(delta: float) -> void:
	if _muzzle_flash_timer > 0.0:
		_muzzle_flash_timer -= delta
		if _muzzle_flash_timer <= 0.0:
			muzzle_light.visible = false


func _update_viewmodel(delta: float) -> void:
	if not _viewmodel:
		return
	var flat_vel := Vector3(velocity.x, 0.0, velocity.z)
	var speed := flat_vel.length()

	# Walk bob
	if speed > 0.5 and is_on_floor():
		_bob_time += delta * speed * 1.2
		var bob_y := sin(_bob_time * 2.0) * 0.006
		var bob_x := cos(_bob_time) * 0.003
		_viewmodel.position = _vm_pos + Vector3(bob_x, bob_y, 0.0)
	else:
		_bob_time = 0.0
		_viewmodel.position = _viewmodel.position.lerp(_vm_pos, delta * 10.0)

	# Recoil recovery
	if _recoil_offset > 0.001:
		_recoil_offset = lerpf(_recoil_offset, 0.0, delta * 18.0)
	else:
		_recoil_offset = 0.0
	_viewmodel.rotation.x = deg_to_rad(_vm_rot_deg.x) - _recoil_offset


## Get the muzzle position — from viewmodel for local, from bone weapon for remote.
func _get_muzzle_pos() -> Vector3:
	if _is_local() and _viewmodel:
		return _viewmodel.global_position
	if character_model.weapon_node and is_instance_valid(character_model.weapon_node):
		return character_model.weapon_node.global_position
	return global_position + Vector3(0.0, 1.4, 0.0)


## Spawn a tracer for a remote gunner using their synced aim data.
func _fire_remote_tracer() -> void:
	var from_pos := _get_muzzle_pos()
	var dir := Vector3(0, 0, -1)
	dir = dir.rotated(Vector3(1, 0, 0), _net_aim_pitch)
	dir = dir.rotated(Vector3(0, 1, 0), _net_rotation_y)
	var to_pos := from_pos + dir * 100.0
	_spawn_tracer(from_pos, to_pos)


## Called by main.gd when server confirms this player hit an enemy.
func on_hit_confirmed(amount: float) -> void:
	hud.show_hit_marker()


## Called by main.gd on remote gunners when a damage event confirms they hit something.
func on_hit_tracer(hit_pos: Vector3) -> void:
	_spawn_tracer(_get_muzzle_pos(), hit_pos)


## Called by main.gd when the server sends a DAMAGE_EVENT targeting this player.
## Health is already updated via apply_server_state -- this is visuals only.
func on_damage_visual(amount: float, hit_pos: Vector3) -> void:
	hud.show_damage_flash()
	character_model.flash_damage()


## Called by main.gd each tick with the authoritative world state for this player.
func apply_server_state(data: Dictionary) -> void:
	# data has: pos (Vector3), rot_y (float), health (float), state (int),
	#           anim_name (String), anim_speed (float)
	if _is_local():
		health = data.health
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
		_net_anim = data.anim_name
		_net_anim_speed = data.anim_speed
		_net_aim_pitch = data.get("aim_pitch", 0.0)
		var new_state: int = data.get("state", 0)
		if new_state == 2 and _net_state != 2:  # transition into attack
			_fire_remote_tracer()
		_net_state = new_state


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
