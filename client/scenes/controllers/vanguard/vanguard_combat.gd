extends Node

## Vanguard combat: cleave, upheaval, dodge, block/parry, vortex, execution.

const PlayerTelegraph := preload("res://scenes/shared/telegraph/player_telegraph.gd")
const VANGUARD_TELEGRAPH_COLOR := Color(0.9, 0.6, 0.3, 0.4)

var ctrl: Node

var _has_hit_this_attack: bool = false
var _dodge_direction: Vector3 = Vector3.ZERO


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


# --- Cleave (LMB) — single repeatable sweep ---


func start_cleave() -> void:
	_has_hit_this_attack = false
	ctrl.vfx.start_swing_trail()
	if NetworkManager.is_active:
		NetworkManager.send_ability(1, 0.0, ctrl.rotation.y)

	ctrl._enter_state(ctrl.State.CLEAVE)
	ctrl._state_timer = ctrl.CLEAVE_DURATION


func process_cleave(delta: float) -> void:
	ctrl.movement.face_attack_direction(delta)
	ctrl.movement.apply_attack_movement(delta)

	if not _has_hit_this_attack and ctrl._state_timer <= ctrl.CLEAVE_DURATION * 0.6:
		_has_hit_this_attack = true
		perform_melee_hit(ctrl.CLEAVE_DAMAGE)

	if ctrl._state_timer <= 0.0:
		ctrl.vfx.stop_swing_trail()
		ctrl._enter_state(ctrl.State.MOVE)


# --- Upheaval (R) — cone slam ---


func start_upheaval() -> void:
	_has_hit_this_attack = false
	ctrl.vfx.start_swing_trail()
	if NetworkManager.is_active:
		NetworkManager.send_ability(2, 0.0, ctrl.rotation.y)
	ctrl._enter_state(ctrl.State.UPHEAVAL_WINDUP)
	ctrl._state_timer = ctrl.UPHEAVAL_WINDUP_TIME


func process_upheaval_windup(delta: float) -> void:
	ctrl.velocity.x = 0.0
	ctrl.velocity.z = 0.0
	ctrl.movement.face_attack_direction(delta)
	if ctrl._state_timer <= 0.0:
		ctrl._enter_state(ctrl.State.UPHEAVAL)
		ctrl._state_timer = ctrl.UPHEAVAL_HIT_TIME
		# Telegraph cone — width varies by onslaught tier
		var half_angle: float = 30.0 if ctrl._onslaught_tier == 0 else 60.0
		PlayerTelegraph.spawn_cone(
			ctrl.get_tree().current_scene,
			ctrl.global_position,
			ctrl.rotation.y,
			7.0,
			half_angle,
			VANGUARD_TELEGRAPH_COLOR
		)


func process_upheaval(delta: float) -> void:
	ctrl.movement.face_attack_direction(delta)
	ctrl.movement.apply_attack_movement(delta)

	if not _has_hit_this_attack and ctrl._state_timer <= ctrl.UPHEAVAL_HIT_TIME * 0.5:
		_has_hit_this_attack = true
		perform_melee_hit(ctrl.UPHEAVAL_DAMAGE)
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
		NetworkManager.send_ability(5, 0.0, ctrl.rotation.y)
		ctrl._block_cooldown = 3.0
		ctrl._enter_state(ctrl.State.MOVE)


# --- Stagger ---


func process_stagger() -> void:
	ctrl.velocity.x = 0.0
	ctrl.velocity.z = 0.0
	if ctrl._state_timer <= 0.0:
		ctrl._enter_state(ctrl.State.MOVE)


# --- Vortex (F) — forward advancing spin ---


func start_vortex() -> void:
	ctrl._vortex_cooldown = ctrl.VORTEX_COOLDOWN
	ctrl._enter_state(ctrl.State.VORTEX)
	# Duration varies by onslaught tier
	var duration: float = ctrl.VORTEX_DURATIONS[ctrl._onslaught_tier]
	ctrl._state_timer = duration
	ctrl.vfx.start_vortex()
	if NetworkManager.is_active:
		NetworkManager.send_ability(20, 0.0, ctrl.rotation.y)
	# Telegraph: circle at start position
	PlayerTelegraph.spawn_circle(
		ctrl.get_tree().current_scene, ctrl.global_position, 4.0, VANGUARD_TELEGRAPH_COLOR
	)


func process_vortex(_delta: float) -> void:
	# Forward dash in facing direction at high speed
	var forward: Vector3 = -ctrl.transform.basis.z
	forward.y = 0.0
	forward = forward.normalized()
	ctrl.velocity.x = forward.x * ctrl.VORTEX_SPEED
	ctrl.velocity.z = forward.z * ctrl.VORTEX_SPEED

	if ctrl._state_timer <= 0.0:
		ctrl.vfx.stop_vortex()
		ctrl.velocity.x *= 0.2
		ctrl.velocity.z *= 0.2
		ctrl._enter_state(ctrl.State.MOVE)


# --- Execution (T) — slow windup + devastating overhead chop ---


func start_execution() -> void:
	ctrl._enter_state(ctrl.State.EXECUTION_WINDUP)
	ctrl._state_timer = ctrl.EXECUTION_WINDUP_TIME
	if NetworkManager.is_active:
		NetworkManager.send_ability(21, 0.0, ctrl.rotation.y)


func process_execution_windup(delta: float) -> void:
	ctrl.velocity.x = 0.0
	ctrl.velocity.z = 0.0
	ctrl.movement.face_attack_direction(delta)
	if ctrl._state_timer <= 0.0:
		ctrl._enter_state(ctrl.State.EXECUTION)
		ctrl._state_timer = ctrl.EXECUTION_HIT_TIME
		ctrl._execution_cooldown = ctrl.EXECUTION_COOLDOWN
		ctrl.vfx.spawn_execution_shockwave(ctrl.global_position, ctrl.rotation.y)
		# Narrow 15° half-angle cone (30° total) matching server
		PlayerTelegraph.spawn_cone(
			ctrl.get_tree().current_scene,
			ctrl.global_position,
			ctrl.rotation.y,
			7.0,
			15.0,
			VANGUARD_TELEGRAPH_COLOR
		)


func process_execution(_delta: float) -> void:
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
