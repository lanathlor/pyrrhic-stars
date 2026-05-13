extends Node

## Vanguard combat: light/heavy attacks, dodge, block/parry, blade swirl, ground slam.

const PlayerTelegraph := preload("res://scenes/shared/telegraph/player_telegraph.gd")
const VANGUARD_TELEGRAPH_COLOR := Color(0.9, 0.6, 0.3, 0.4)

var ctrl: Node

var _has_hit_this_attack: bool = false
var _queued_light: bool = false
var _dodge_direction: Vector3 = Vector3.ZERO
var _blade_swirl_tick_timer: float = 0.0


func _ready() -> void:
	ctrl = get_parent()


# --- Dodge ---


func start_dodge() -> void:
	var wish: Vector3 = ctrl.movement.get_camera_wish_dir()
	if wish.length() > 0.1:
		if ctrl._lock_on_active and ctrl._lock_target and is_instance_valid(ctrl._lock_target):
			var input_dir := Input.get_vector(
				"move_left", "move_right", "move_forward", "move_backward"
			)
			_dodge_direction = (
				(ctrl.transform.basis * Vector3(input_dir.x, 0.0, input_dir.y)).normalized()
			)
		else:
			_dodge_direction = wish
	else:
		_dodge_direction = (ctrl.transform.basis * Vector3(0.0, 0.0, 1.0)).normalized()

	ctrl._enter_state(ctrl.State.DODGE)
	ctrl._state_timer = ctrl.dodge_duration
	ctrl._is_invincible = true
	if NetworkManager.is_active:
		NetworkManager.send_ability(3, 0.0, ctrl.rotation.y)


func process_dodge(_delta: float) -> void:
	ctrl.velocity.x = _dodge_direction.x * ctrl.dodge_speed
	ctrl.velocity.z = _dodge_direction.z * ctrl.dodge_speed

	var elapsed: float = ctrl.dodge_duration - ctrl._state_timer
	if elapsed >= ctrl.dodge_iframe_duration:
		ctrl._is_invincible = false

	if ctrl._state_timer <= 0.0:
		ctrl._is_invincible = false
		ctrl.velocity.x *= 0.3
		ctrl.velocity.z *= 0.3
		ctrl._enter_state(ctrl.State.MOVE)


# --- Light Attack Combo ---


func start_light_attack(combo_step: int) -> void:
	_queued_light = false
	_has_hit_this_attack = false
	ctrl.vfx.start_swing_trail()
	if NetworkManager.is_active:
		NetworkManager.send_ability(1, 0.0, ctrl.rotation.y)

	match combo_step:
		1:
			ctrl._enter_state(ctrl.State.LIGHT_1)
			ctrl._state_timer = ctrl.light_duration_1
		2:
			ctrl._enter_state(ctrl.State.LIGHT_2)
			ctrl._state_timer = ctrl.light_duration_2
		3:
			ctrl._enter_state(ctrl.State.LIGHT_3)
			ctrl._state_timer = ctrl.light_duration_3


func process_light_attack(delta: float) -> void:
	ctrl.movement.face_attack_direction(delta)

	var dur: float = get_current_light_duration()

	ctrl.movement.apply_attack_movement(delta)

	if Input.is_action_just_pressed("light_attack"):
		_queued_light = true

	if not _has_hit_this_attack and ctrl._state_timer <= dur * 0.6:
		_has_hit_this_attack = true
		perform_melee_hit(get_current_light_damage())

	if ctrl._state_timer <= 0.0:
		var next: int = get_next_combo_step()
		if _queued_light and next > 0 and ctrl.stamina >= ctrl.light_stamina_cost:
			start_light_attack(next)
		else:
			ctrl.vfx.stop_swing_trail()
			ctrl._combo_window_timer = ctrl.light_combo_window
			ctrl._enter_state(ctrl.State.MOVE)


func get_current_light_damage() -> float:
	match ctrl.state:
		ctrl.State.LIGHT_1:
			return ctrl.light_damage_1
		ctrl.State.LIGHT_2:
			return ctrl.light_damage_2
		ctrl.State.LIGHT_3:
			return ctrl.light_damage_3
	return 0.0


func get_current_light_duration() -> float:
	match ctrl.state:
		ctrl.State.LIGHT_1:
			return ctrl.light_duration_1
		ctrl.State.LIGHT_2:
			return ctrl.light_duration_2
		ctrl.State.LIGHT_3:
			return ctrl.light_duration_3
	return ctrl.light_duration_1


func get_next_combo_step() -> int:
	match ctrl.state:
		ctrl.State.LIGHT_1:
			return 2
		ctrl.State.LIGHT_2:
			return 3
	return 0


# --- Heavy Attack ---


func start_heavy_attack() -> void:
	_has_hit_this_attack = false
	ctrl.vfx.start_swing_trail()
	if NetworkManager.is_active:
		NetworkManager.send_ability(2, 0.0, ctrl.rotation.y)
	ctrl._enter_state(ctrl.State.HEAVY_WINDUP)
	ctrl._state_timer = ctrl.heavy_windup_time


func process_heavy_windup(delta: float) -> void:
	ctrl.velocity.x = 0.0
	ctrl.velocity.z = 0.0
	ctrl.movement.face_attack_direction(delta)
	if ctrl._state_timer <= 0.0:
		ctrl._enter_state(ctrl.State.HEAVY)
		ctrl._state_timer = ctrl.heavy_attack_duration


func process_heavy_attack(delta: float) -> void:
	ctrl.movement.face_attack_direction(delta)
	ctrl.movement.apply_attack_movement(delta)

	if not _has_hit_this_attack and ctrl._state_timer <= ctrl.heavy_attack_duration * 0.5:
		_has_hit_this_attack = true
		perform_melee_hit(ctrl.heavy_damage)
	if ctrl._state_timer <= 0.0:
		ctrl.vfx.stop_swing_trail()
		ctrl._enter_state(ctrl.State.MOVE)


# --- Block ---


func process_block(delta: float) -> void:
	ctrl.velocity.x = 0.0
	ctrl.velocity.z = 0.0
	ctrl.movement.face_attack_direction(delta)
	ctrl.movement.consume_stamina(ctrl.block_stamina_drain * delta)
	if not Input.is_action_pressed("block") or ctrl.stamina <= 0.0:
		ctrl.vfx.hide_block_shield()
		ctrl._enter_state(ctrl.State.MOVE)


# --- Stagger ---


func process_stagger() -> void:
	ctrl.velocity.x = 0.0
	ctrl.velocity.z = 0.0
	if ctrl._state_timer <= 0.0:
		ctrl._enter_state(ctrl.State.MOVE)


# --- Blade Swirl (Q) ---


func start_blade_swirl() -> void:
	ctrl._blade_swirl_cooldown = ctrl.BLADE_SWIRL_COOLDOWN
	ctrl._blade_swirl_timer = ctrl.BLADE_SWIRL_DURATION
	_blade_swirl_tick_timer = 0.0
	ctrl._enter_state(ctrl.State.BLADE_SWIRL)
	ctrl._state_timer = ctrl.BLADE_SWIRL_DURATION
	ctrl.vfx.start_blade_swirl()
	if NetworkManager.is_active:
		NetworkManager.send_ability(20, 0.0, ctrl.rotation.y)
	PlayerTelegraph.spawn_circle(
		ctrl.get_tree().current_scene, ctrl.global_position, 6.0, VANGUARD_TELEGRAPH_COLOR
	)


func process_blade_swirl(delta: float) -> void:
	ctrl.movement.face_attack_direction(delta)

	var wish_dir: Vector3 = ctrl.movement.get_camera_wish_dir()
	var speed: float = ctrl.run_speed * ctrl.BLADE_SWIRL_SPEED_MULT
	if wish_dir.length() > 0.1:
		var target_vel: Vector3 = wish_dir * speed
		ctrl.velocity.x = move_toward(ctrl.velocity.x, target_vel.x, ctrl.ground_accel * delta)
		ctrl.velocity.z = move_toward(ctrl.velocity.z, target_vel.z, ctrl.ground_accel * delta)
	else:
		ctrl.velocity.x = move_toward(ctrl.velocity.x, 0.0, ctrl.ground_decel * delta)
		ctrl.velocity.z = move_toward(ctrl.velocity.z, 0.0, ctrl.ground_decel * delta)

	_blade_swirl_tick_timer += delta
	if _blade_swirl_tick_timer >= 0.5:
		_blade_swirl_tick_timer -= 0.5
		PlayerTelegraph.spawn_circle(
			ctrl.get_tree().current_scene, ctrl.global_position, 6.0, VANGUARD_TELEGRAPH_COLOR
		)

	if ctrl._state_timer <= 0.0:
		ctrl.vfx.stop_blade_swirl()
		ctrl._enter_state(ctrl.State.MOVE)


# --- Ground Slam (E) ---


func start_ground_slam() -> void:
	ctrl._enter_state(ctrl.State.GROUND_SLAM_WINDUP)
	ctrl._state_timer = ctrl.GROUND_SLAM_WINDUP_TIME
	if NetworkManager.is_active:
		NetworkManager.send_ability(21, 0.0, ctrl.rotation.y)


func process_ground_slam_windup(delta: float) -> void:
	ctrl.velocity.x = 0.0
	ctrl.velocity.z = 0.0
	ctrl.movement.face_attack_direction(delta)
	if ctrl._state_timer <= 0.0:
		ctrl._enter_state(ctrl.State.GROUND_SLAM)
		ctrl._state_timer = ctrl.GROUND_SLAM_HIT_TIME
		ctrl._ground_slam_cooldown = ctrl.GROUND_SLAM_COOLDOWN
		ctrl.vfx.spawn_ground_slam_shockwave(ctrl.global_position, ctrl.rotation.y)
		PlayerTelegraph.spawn_cone(
			ctrl.get_tree().current_scene,
			ctrl.global_position,
			ctrl.rotation.y,
			7.0,
			45.0,
			VANGUARD_TELEGRAPH_COLOR
		)


func process_ground_slam(_delta: float) -> void:
	ctrl.velocity.x = 0.0
	ctrl.velocity.z = 0.0
	if ctrl._state_timer <= 0.0:
		ctrl._enter_state(ctrl.State.MOVE)


# --- Melee Hit Detection ---


func perform_melee_hit(_damage: float) -> void:
	var forward: Vector3 = -ctrl.transform.basis.z
	forward.y = 0.0
	forward = forward.normalized()

	for enemy in GameManager.enemies:
		if not is_instance_valid(enemy):
			continue
		var to_enemy: Vector3 = enemy.global_position - ctrl.global_position
		to_enemy.y = 0.0
		var dist: float = to_enemy.length()
		if dist > ctrl.melee_range:
			continue
		if dist < 0.01:
			continue
		var angle: float = rad_to_deg(forward.angle_to(to_enemy.normalized()))
		if angle <= ctrl.melee_arc_degrees / 2.0:
			pass  # Hit marker now driven by server-confirmed damage events
