extends CharacterBody3D

## Vanguard — Souls-like third-person melee controller (Blade spec).
## Combo chains, dodge rolls with i-frames, block/parry, stamina, lock-on.

signal died

enum State { MOVE, DODGE, LIGHT_1, LIGHT_2, LIGHT_3, HEAVY_WINDUP, HEAVY, BLOCK, STAGGER, DEAD }

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
@export var dodge_stamina_cost: float = 25.0

# Combat — light attacks (3-hit combo, escalating damage)
@export var light_damage_1: float = 30.0
@export var light_damage_2: float = 35.0
@export var light_damage_3: float = 55.0
@export var light_attack_duration: float = 0.35
@export var light_combo_window: float = 0.4
@export var light_stamina_cost: float = 15.0

# Combat — heavy attack
@export var heavy_damage: float = 75.0
@export var heavy_windup_time: float = 0.5
@export var heavy_attack_duration: float = 0.3
@export var heavy_stamina_cost: float = 30.0

# Melee hit detection
@export var melee_range: float = 3.0
@export var melee_arc_degrees: float = 120.0

# Block / parry
@export var block_damage_reduction: float = 0.7
@export var parry_window: float = 0.15
@export var block_stamina_drain: float = 15.0

# Health & Stamina
var health: float = 150.0
var max_health: float = 200.0
var peer_id: int = 0
var stamina: float = 100.0
var max_stamina: float = 100.0
@export var stamina_regen_rate: float = 30.0
@export var stamina_regen_delay: float = 0.6

# State
var state: State = State.MOVE
var _state_timer: float = 0.0
var _combo_window_timer: float = 0.0
var _stamina_cooldown_timer: float = 0.0
var _dodge_direction: Vector3 = Vector3.ZERO
var _is_invincible: bool = false
var _parry_timer: float = 0.0
var _has_hit_this_attack: bool = false
var _queued_light: bool = false
var _stagger_duration: float = 0.3

# Camera
var _camera_yaw: float = 0.0
var _camera_pitch: float = -0.3
@export var camera_distance: float = 6.0
@export var camera_height_offset: float = 2.0

# Lock-on
var _lock_target: Node3D = null
var _lock_on_active: bool = false

var _gravity: float = 9.8
var _flash_timer: float = 0.0
var _facing_angle: float = 0.0

# Network sync
var _net_anim: String = ""
var _net_anim_speed: float = 1.0
var _net_position: Vector3 = Vector3.ZERO
var _net_rotation_y: float = 0.0
const NET_INTERP_SPEED := 15.0

const WEAPON_SCENE := "res://assets/models/weapons/weapon_longsword.glb"

@onready var camera: Camera3D = $Camera3D
@onready var character_model: Node3D = $CharacterModel
@onready var hud: Control = $HUDLayer/VanguardHUD


func _ready() -> void:
	GameManager.register_player(self)
	_net_position = global_position
	_net_rotation_y = rotation.y
	camera.top_level = true
	NetworkManager.message_received.connect(_on_network_message)

	if _is_local():
		Input.set_mouse_mode(Input.MOUSE_MODE_CAPTURED)
		hud.update_health(health, max_health)
		hud.update_stamina(stamina, max_stamina)
	else:
		$HUDLayer.visible = false
		camera.current = false
		set_process_unhandled_input(false)

	_attach_weapon.call_deferred()



func _attach_weapon() -> void:
	var offset_pos := Vector3(0.0, 0.08, 0.0)
	var offset_rot := Vector3(deg_to_rad(20.0), 0.0, deg_to_rad(-90.0))
	character_model.attach_weapon(WEAPON_SCENE, "mixamorig_RightHand", offset_pos, offset_rot)


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


func _unhandled_input(event: InputEvent) -> void:
	if not _is_local():
		return
	if event is InputEventMouseMotion and Input.get_mouse_mode() == Input.MOUSE_MODE_CAPTURED:
		_camera_yaw -= event.relative.x * mouse_sensitivity
		_camera_pitch -= event.relative.y * mouse_sensitivity
		_camera_pitch = clampf(_camera_pitch, deg_to_rad(-60.0), deg_to_rad(20.0))

	if event.is_action_pressed("lock_on"):
		_toggle_lock_on()


func _physics_process(delta: float) -> void:
	if not _is_local():
		global_position = global_position.lerp(_net_position, NET_INTERP_SPEED * delta)
		rotation.y = lerp_angle(rotation.y, _net_rotation_y, NET_INTERP_SPEED * delta)
		if _net_anim != "":
			character_model.play_anim(_net_anim, _net_anim_speed)
		return

	if not is_on_floor():
		velocity.y -= _gravity * delta

	_state_timer -= delta
	_update_flash(delta)
	_update_camera()
	_update_stamina(delta)
	_update_parry(delta)

	match state:
		State.MOVE:
			_process_move(delta)
		State.DODGE:
			_process_dodge(delta)
		State.LIGHT_1, State.LIGHT_2, State.LIGHT_3:
			_process_light_attack(delta)
		State.HEAVY_WINDUP:
			_process_heavy_windup(delta)
		State.HEAVY:
			_process_heavy_attack(delta)
		State.BLOCK:
			_process_block(delta)
		State.STAGGER:
			_process_stagger()
		State.DEAD:
			velocity.x = 0.0
			velocity.z = 0.0

	move_and_slide()
	_update_animation()
	_update_weapon_visual()
	hud.update_lock_on(_lock_target, camera)

	# Sync to remote peers
	_net_position = global_position
	_net_rotation_y = rotation.y
	if NetworkManager.is_active:
		_sync_state_to_peers()


# --- Movement ---

## Get world-space wish direction from input + actual camera transform.
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
	# forward = negative Y in get_vector, so negate input_dir.y
	return (cam_right * input_dir.x + cam_forward * -input_dir.y).normalized()


func _process_move(delta: float) -> void:
	if _combo_window_timer > 0.0:
		_combo_window_timer -= delta
		if _combo_window_timer <= 0.0:
			_queued_light = false

	# Attack inputs
	if Input.is_action_just_pressed("light_attack") and stamina >= light_stamina_cost:
		_start_light_attack(1)
		return
	if Input.is_action_just_pressed("heavy_attack") and stamina >= heavy_stamina_cost:
		_start_heavy_attack()
		return
	if Input.is_action_pressed("block"):
		_enter_state(State.BLOCK)
		_parry_timer = parry_window
		return

	# Jump
	if Input.is_action_just_pressed("jump") and is_on_floor():
		velocity.y = 4.5

	# Dodge
	if Input.is_action_just_pressed("dodge") and is_on_floor() and stamina >= dodge_stamina_cost:
		_start_dodge()
		return

	# Movement — direction derived from actual camera transform
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
	# Compute the Y rotation that makes the node's -Z align with dir
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


## Auto-face during attacks: lock target > nearest enemy > camera forward.
func _face_attack_direction(delta: float) -> void:
	if _lock_on_active and _lock_target and is_instance_valid(_lock_target):
		_face_target(delta)
		return

	# Auto-target nearest visible enemy within engagement range
	var best: Node3D = null
	var best_dist: float = melee_range * 2.5
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
			# Fast snap during attacks
			_facing_angle = lerp_angle(_facing_angle, target_angle, 25.0 * delta)
			rotation.y = _facing_angle
		return

	# No enemy nearby — face camera forward direction
	var cam_fwd := -camera.global_transform.basis.z
	cam_fwd.y = 0.0
	if cam_fwd.length() > 0.01:
		cam_fwd = cam_fwd.normalized()
		var target_angle := _get_target_yaw(cam_fwd)
		_facing_angle = lerp_angle(_facing_angle, target_angle, 15.0 * delta)
		rotation.y = _facing_angle


# --- Dodge ---

func _start_dodge() -> void:
	var wish := _get_camera_wish_dir()
	if wish.length() > 0.1:
		if _lock_on_active and _lock_target and is_instance_valid(_lock_target):
			# Dodge relative to character facing (strafe dodges when locked on)
			var input_dir := Input.get_vector("move_left", "move_right", "move_forward", "move_backward")
			_dodge_direction = (transform.basis * Vector3(input_dir.x, 0.0, input_dir.y)).normalized()
		else:
			_dodge_direction = wish
	else:
		# No input: dodge backward relative to facing
		_dodge_direction = (transform.basis * Vector3(0.0, 0.0, 1.0)).normalized()

	_enter_state(State.DODGE)
	_state_timer = dodge_duration
	_is_invincible = true
	_consume_stamina(dodge_stamina_cost)


func _process_dodge(delta: float) -> void:
	velocity.x = _dodge_direction.x * dodge_speed
	velocity.z = _dodge_direction.z * dodge_speed

	# I-frames end after dodge_iframe_duration
	var elapsed := dodge_duration - _state_timer
	if elapsed >= dodge_iframe_duration:
		_is_invincible = false

	if _state_timer <= 0.0:
		_is_invincible = false
		velocity.x *= 0.3
		velocity.z *= 0.3
		_enter_state(State.MOVE)


# --- Light Attack Combo ---

func _start_light_attack(combo_step: int) -> void:
	_queued_light = false
	_has_hit_this_attack = false
	_consume_stamina(light_stamina_cost)

	match combo_step:
		1: _enter_state(State.LIGHT_1)
		2: _enter_state(State.LIGHT_2)
		3: _enter_state(State.LIGHT_3)
	_state_timer = light_attack_duration


func _process_light_attack(delta: float) -> void:
	_face_attack_direction(delta)

	# Forward lunge — only if not already inside enemy
	var nearest_dist := _nearest_enemy_distance()
	if nearest_dist > 1.2:
		var fwd := -transform.basis.z
		velocity.x = fwd.x * 3.0
		velocity.z = fwd.z * 3.0
	else:
		velocity.x = 0.0
		velocity.z = 0.0

	# Buffer next attack
	if Input.is_action_just_pressed("light_attack"):
		_queued_light = true

	# Hit at 40% through the swing
	if not _has_hit_this_attack and _state_timer <= light_attack_duration * 0.6:
		_has_hit_this_attack = true
		_perform_melee_hit(_get_current_light_damage())

	if _state_timer <= 0.0:
		velocity.x = 0.0
		velocity.z = 0.0
		var next := _get_next_combo_step()
		if _queued_light and next > 0 and stamina >= light_stamina_cost:
			_start_light_attack(next)
		else:
			_combo_window_timer = light_combo_window
			_enter_state(State.MOVE)


func _get_current_light_damage() -> float:
	match state:
		State.LIGHT_1: return light_damage_1
		State.LIGHT_2: return light_damage_2
		State.LIGHT_3: return light_damage_3
	return 0.0


func _get_next_combo_step() -> int:
	match state:
		State.LIGHT_1: return 2
		State.LIGHT_2: return 3
	return 0


# --- Heavy Attack ---

func _start_heavy_attack() -> void:
	_has_hit_this_attack = false
	_consume_stamina(heavy_stamina_cost)
	_enter_state(State.HEAVY_WINDUP)
	_state_timer = heavy_windup_time


func _process_heavy_windup(delta: float) -> void:
	velocity.x = 0.0
	velocity.z = 0.0
	_face_attack_direction(delta)
	if _state_timer <= 0.0:
		_enter_state(State.HEAVY)
		_state_timer = heavy_attack_duration


func _process_heavy_attack(delta: float) -> void:
	_face_attack_direction(delta)

	# Heavier lunge — only if not already inside enemy
	var nearest_dist := _nearest_enemy_distance()
	if nearest_dist > 1.2:
		var fwd := -transform.basis.z
		velocity.x = fwd.x * 4.0
		velocity.z = fwd.z * 4.0
	else:
		velocity.x = 0.0
		velocity.z = 0.0

	if not _has_hit_this_attack and _state_timer <= heavy_attack_duration * 0.5:
		_has_hit_this_attack = true
		_perform_melee_hit(heavy_damage)
	if _state_timer <= 0.0:
		velocity.x = 0.0
		velocity.z = 0.0
		_enter_state(State.MOVE)


# --- Block ---

func _process_block(delta: float) -> void:
	velocity.x = 0.0
	velocity.z = 0.0
	_face_attack_direction(delta)
	_consume_stamina(block_stamina_drain * delta)
	if not Input.is_action_pressed("block") or stamina <= 0.0:
		_enter_state(State.MOVE)


func _update_parry(delta: float) -> void:
	if _parry_timer > 0.0:
		_parry_timer -= delta


# --- Stagger ---

func _process_stagger() -> void:
	velocity.x = 0.0
	velocity.z = 0.0
	if _state_timer <= 0.0:
		_enter_state(State.MOVE)


# --- Melee Hit Detection ---

func _perform_melee_hit(damage: float) -> void:
	var forward := -transform.basis.z
	forward.y = 0.0
	forward = forward.normalized()

	for enemy in GameManager.enemies:
		if not is_instance_valid(enemy):
			continue
		var to_enemy := enemy.global_position - global_position
		to_enemy.y = 0.0
		var dist := to_enemy.length()
		if dist > melee_range:
			continue
		if dist < 0.01:
			# On top of enemy, always hits
			_deal_damage(enemy, damage, global_position)
			hud.show_hit_marker()
			continue
		var angle := rad_to_deg(forward.angle_to(to_enemy.normalized()))
		if angle <= melee_arc_degrees / 2.0:
			_deal_damage(enemy, damage, global_position)
			hud.show_hit_marker()


# --- Stamina ---

func _consume_stamina(amount: float) -> void:
	stamina -= amount
	stamina = maxf(stamina, 0.0)
	_stamina_cooldown_timer = stamina_regen_delay
	hud.update_stamina(stamina, max_stamina)


func _update_stamina(delta: float) -> void:
	if _stamina_cooldown_timer > 0.0:
		_stamina_cooldown_timer -= delta
		return
	if state == State.BLOCK:
		return
	if stamina < max_stamina:
		stamina = minf(stamina + stamina_regen_rate * delta, max_stamina)
		hud.update_stamina(stamina, max_stamina)


# --- Lock-on ---

func _toggle_lock_on() -> void:
	if _lock_on_active:
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


# --- Damage ---

func take_damage(amount: float, hit_position: Vector3 = Vector3.ZERO) -> void:
	_apply_damage(amount, hit_position)


func _apply_damage(amount: float, _hit_position: Vector3 = Vector3.ZERO) -> void:
	if state == State.DEAD:
		return
	if _is_invincible:
		return

	# Parry — negate all damage
	if state == State.BLOCK and _parry_timer > 0.0:
		hud.show_parry_flash()
		return

	# Block — reduce damage
	if state == State.BLOCK:
		amount *= (1.0 - block_damage_reduction)

	health -= amount
	health = maxf(health, 0.0)
	hud.update_health(health, max_health)
	hud.show_damage_flash()
	_show_body_flash()

	if health <= 0.0:
		_die()
	elif state != State.BLOCK and state != State.DODGE:
		_enter_state(State.STAGGER)
		_state_timer = _stagger_duration


func _die() -> void:
	_enter_state(State.DEAD)
	died.emit()


func _deal_damage(target: Node, amount: float, hit_pos: Vector3) -> void:
	if not target.has_method("take_damage"):
		return
	if NetworkManager.is_active:
		var target_peer: int = target.peer_id if "peer_id" in target else 0
		NetworkManager.send_msg(NetSerializer.OP_DAMAGE,
			NetSerializer.encode_damage(target_peer, amount, hit_pos))
	else:
		target.take_damage(amount, hit_pos)


# --- Visual feedback ---

func _show_body_flash() -> void:
	character_model.flash_damage()


func _update_flash(_delta: float) -> void:
	pass


func _update_animation() -> void:
	match state:
		State.DODGE:
			_net_anim = "roll"
			_net_anim_speed = 1.0
			character_model.play_anim_timed("roll", dodge_duration)
			return
		State.LIGHT_1:
			_net_anim = "sword_slash_1"
			_net_anim_speed = 1.0
			character_model.play_anim_timed("sword_slash_1", light_attack_duration)
			return
		State.LIGHT_2:
			_net_anim = "sword_slash_2"
			_net_anim_speed = 1.0
			character_model.play_anim_timed("sword_slash_2", light_attack_duration)
			return
		State.LIGHT_3:
			_net_anim = "sword_slash_3"
			_net_anim_speed = 1.0
			character_model.play_anim_timed("sword_slash_3", light_attack_duration)
			return
		State.HEAVY_WINDUP:
			_net_anim = "sword_heavy"
			_net_anim_speed = 1.0
			character_model.play_anim_timed("sword_heavy", heavy_windup_time + heavy_attack_duration)
			return
		State.HEAVY:
			_net_anim = "sword_heavy"
			_net_anim_speed = 3.0
			character_model.set_animation_speed(3.0)
			return
		State.BLOCK:
			_net_anim = "sword_block"
			_net_anim_speed = 1.0
			character_model.play_anim("sword_block")
			return
		State.STAGGER:
			_net_anim = "sword_impact"
			_net_anim_speed = 1.0
			character_model.play_anim("sword_impact")
			return
		State.DEAD:
			_net_anim = "sword_idle"
			_net_anim_speed = 1.0
			character_model.play_anim("sword_idle")
			return

	if not is_on_floor():
		_net_anim = "sword_jump"
		_net_anim_speed = 2.0
		character_model.play_anim("sword_jump", 2.0)
		return

	var flat_vel := Vector3(velocity.x, 0.0, velocity.z)
	if flat_vel.length() > 0.5:
		var speed_ratio := flat_vel.length() / sprint_speed
		_net_anim_speed = clampf(speed_ratio, 0.5, 1.5)
		_net_anim = "sword_run"
		character_model.play_anim("sword_run", _net_anim_speed)
	else:
		_net_anim = "sword_idle"
		_net_anim_speed = 1.0
		character_model.play_anim("sword_idle")


func _update_weapon_visual() -> void:
	# Weapon is now bone-attached; skip if not ready yet
	if not character_model.weapon_node:
		return


# --- Helpers ---

func _nearest_enemy_distance() -> float:
	var best := INF
	for enemy in GameManager.enemies:
		if not is_instance_valid(enemy) or not enemy.visible:
			continue
		var d := global_position.distance_to(enemy.global_position)
		if d < best:
			best = d
	return best


# --- Camera ---

func _update_camera() -> void:
	var player_pos := global_position + Vector3(0.0, camera_height_offset, 0.0)
	var desired_cam_pos: Vector3

	if _lock_on_active and _lock_target and is_instance_valid(_lock_target):
		var target_pos := _lock_target.global_position + Vector3(0.0, 1.0, 0.0)
		var to_target := target_pos - player_pos
		to_target.y = 0.0
		if to_target.length() > 0.1:
			var behind := -to_target.normalized()
			desired_cam_pos = player_pos + behind * camera_distance + Vector3(0.0, 3.0, 0.0)
			camera.global_position = _apply_camera_collision(player_pos, desired_cam_pos)
			# Look at a point between player and boss, slightly low, for better overview
			var look_target := (player_pos + target_pos) * 0.5
			look_target.y = 0.8
			camera.look_at(look_target, Vector3.UP)
			var offset := desired_cam_pos - player_pos
			_camera_yaw = atan2(offset.x, offset.z)
		else:
			desired_cam_pos = player_pos + Vector3(0.0, 0.0, camera_distance)
			camera.global_position = _apply_camera_collision(player_pos, desired_cam_pos)
			camera.look_at(player_pos, Vector3.UP)
	else:
		var cam_offset := Vector3(0.0, 0.0, camera_distance)
		cam_offset = cam_offset.rotated(Vector3.RIGHT, _camera_pitch)
		cam_offset = cam_offset.rotated(Vector3.UP, _camera_yaw)
		desired_cam_pos = player_pos + cam_offset
		camera.global_position = _apply_camera_collision(player_pos, desired_cam_pos)
		camera.look_at(player_pos, Vector3.UP)


func _apply_camera_collision(from: Vector3, to: Vector3) -> Vector3:
	var space := get_world_3d().direct_space_state
	if not space:
		return to
	var query := PhysicsRayQueryParameters3D.create(from, to, 1)  # mask 1 = World layer
	query.exclude = [get_rid()]
	var result := space.intersect_ray(query)
	if result:
		# Pull camera slightly in front of the hit surface
		return result.position + (from - to).normalized() * 0.3
	return to


# --- State helpers ---

func _enter_state(new_state: State) -> void:
	match state:
		State.DODGE:
			_is_invincible = false
	state = new_state
	_has_hit_this_attack = false
