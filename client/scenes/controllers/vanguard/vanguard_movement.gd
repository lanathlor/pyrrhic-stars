extends Node

## Vanguard locomotion, facing, and stamina management.

var ctrl: Node

var _facing_angle: float = 0.0
var _stamina_cooldown_timer: float = 0.0


func _ready() -> void:
	ctrl = get_parent()


## Get world-space wish direction from input + actual camera transform.
func get_camera_wish_dir() -> Vector3:
	var input_dir := GameManager.move_vector()
	if input_dir.length() < 0.1:
		return Vector3.ZERO
	var cam_xf: Transform3D = ctrl.camera.global_transform
	var cam_forward: Vector3 = -cam_xf.basis.z
	cam_forward.y = 0.0
	if cam_forward.length() < 0.01:
		return Vector3.ZERO
	cam_forward = cam_forward.normalized()
	var cam_right: Vector3 = cam_xf.basis.x
	cam_right.y = 0.0
	cam_right = cam_right.normalized()
	return (cam_right * input_dir.x + cam_forward * -input_dir.y).normalized()


func get_target_yaw(dir: Vector3) -> float:
	var t := Transform3D()
	t = t.looking_at(dir, Vector3.UP)
	return t.basis.get_euler().y


func face_direction(dir: Vector3, delta: float) -> void:
	if dir.length() < 0.1:
		return
	var target_angle := get_target_yaw(dir)
	_facing_angle = lerp_angle(_facing_angle, target_angle, ctrl.rotation_speed * delta)
	ctrl.rotation.y = _facing_angle


func face_target(delta: float) -> void:
	if not ctrl._lock_target or not is_instance_valid(ctrl._lock_target):
		return
	var to_target: Vector3 = ctrl._lock_target.global_position - ctrl.global_position
	to_target.y = 0.0
	if to_target.length() > 0.1:
		var target_angle := get_target_yaw(to_target)
		_facing_angle = lerp_angle(_facing_angle, target_angle, ctrl.rotation_speed * delta)
		ctrl.rotation.y = _facing_angle


## Auto-face during attacks: lock target > nearest enemy > camera forward.
func face_attack_direction(delta: float) -> void:
	if ctrl._lock_on_active and ctrl._lock_target and is_instance_valid(ctrl._lock_target):
		face_target(delta)
		return

	var best: Node3D = null
	var best_dist: float = ctrl.melee_range * 2.5
	for enemy in GameManager.enemies:
		if not is_instance_valid(enemy) or not enemy.visible:
			continue
		var dist: float = ctrl.global_position.distance_to(enemy.global_position)
		if dist < best_dist:
			best_dist = dist
			best = enemy

	if best:
		var to_enemy: Vector3 = best.global_position - ctrl.global_position
		to_enemy.y = 0.0
		if to_enemy.length() > 0.1:
			var target_angle := get_target_yaw(to_enemy)
			_facing_angle = lerp_angle(_facing_angle, target_angle, 25.0 * delta)
			ctrl.rotation.y = _facing_angle
		return

	var cam_fwd: Vector3 = -ctrl.camera.global_transform.basis.z
	cam_fwd.y = 0.0
	if cam_fwd.length() > 0.01:
		cam_fwd = cam_fwd.normalized()
		var target_angle := get_target_yaw(cam_fwd)
		_facing_angle = lerp_angle(_facing_angle, target_angle, 15.0 * delta)
		ctrl.rotation.y = _facing_angle


## Apply reduced movement during attack states.
func apply_attack_movement(delta: float) -> void:
	var wish_dir := get_camera_wish_dir()
	var speed: float = ctrl.run_speed * ctrl.attack_move_speed_mult
	if wish_dir.length() > 0.1:
		var target_vel: Vector3 = wish_dir * speed
		ctrl.velocity.x = move_toward(ctrl.velocity.x, target_vel.x, ctrl.ground_accel * delta)
		ctrl.velocity.z = move_toward(ctrl.velocity.z, target_vel.z, ctrl.ground_accel * delta)
	else:
		ctrl.velocity.x = move_toward(ctrl.velocity.x, 0.0, ctrl.ground_decel * delta)
		ctrl.velocity.z = move_toward(ctrl.velocity.z, 0.0, ctrl.ground_decel * delta)


func process_move(delta: float) -> void:
	var cursor_active := Input.get_mouse_mode() != Input.MOUSE_MODE_CAPTURED

	if ctrl.spec_id == "shield":
		_process_move_shield(delta, cursor_active)
	else:
		_process_move_blade(delta, cursor_active)


func _process_move_blade(delta: float, cursor_active: bool) -> void:
	# Attack inputs (disabled when cursor is visible)
	if (
		not cursor_active
		and Input.is_action_just_pressed("light_attack")
		and ctrl.stamina >= ctrl.CLEAVE_STAMINA
	):
		ctrl.combat.start_cleave()
		return
	if Input.is_action_just_pressed("heavy_attack") and ctrl.stamina >= ctrl.UPHEAVAL_STAMINA:
		ctrl.combat.start_upheaval()
		return
	if (
		not cursor_active
		and Input.is_action_just_pressed("block")
		and ctrl._block_cooldown <= 0.0
		and ctrl.stamina > 0.0
	):
		ctrl._enter_state(ctrl.State.BLOCK)
		ctrl._parry_timer = ctrl.parry_window
		ctrl.vfx.show_block_shield()
		AudioManager.play_3d(&"vanguard_block", ctrl.global_position)
		NetworkManager.send_ability(4, 0.0, ctrl.rotation.y)
		return

	# Jump
	if (
		Input.is_action_just_pressed("jump")
		and ctrl.is_on_floor()
		and not GameManager.text_input_active()
	):
		ctrl.velocity.y = 3.5

	# Dodge
	if (
		Input.is_action_just_pressed("dodge")
		and ctrl.is_on_floor()
		and ctrl.stamina >= ctrl.dodge_stamina_cost
	):
		ctrl.combat.start_dodge()
		return

	# Vortex (F)
	if (
		not cursor_active
		and Input.is_action_just_pressed("ability_1")
		and ctrl.stamina >= ctrl.VORTEX_STAMINA
		and ctrl._vortex_cooldown <= 0.0
	):
		ctrl.combat.start_vortex()
		return

	# Execution (T)
	if (
		not cursor_active
		and Input.is_action_just_pressed("ability_2")
		and ctrl.stamina >= ctrl.EXECUTION_STAMINA
		and ctrl._execution_cooldown <= 0.0
	):
		ctrl.combat.start_execution()
		return

	_apply_movement(delta)


func _process_move_shield(delta: float, cursor_active: bool) -> void:
	if _handle_shield_combat_input(cursor_active):
		return

	# Jump
	if (
		Input.is_action_just_pressed("jump")
		and ctrl.is_on_floor()
		and not GameManager.text_input_active()
	):
		ctrl.velocity.y = 3.5

	# Brace (F) -- only while blocking (server validates, but feels better with client check)
	if (
		not cursor_active
		and Input.is_action_just_pressed("ability_1")
		and ctrl._brace_cooldown <= 0.0
		and ctrl.state == ctrl.State.SHIELD_BLOCK
	):
		ctrl.combat.start_brace()
		return

	# Retaliate (T)
	if (
		not cursor_active
		and Input.is_action_just_pressed("ability_2")
		and ctrl._retaliate_cooldown <= 0.0
	):
		ctrl.combat.start_retaliate()
		return

	_apply_movement(delta)


func _handle_shield_combat_input(cursor_active: bool) -> bool:
	# Shield Bash (LMB)
	if (
		not cursor_active
		and Input.is_action_just_pressed("light_attack")
		and ctrl.stamina >= ctrl.SHIELD_BASH_STAMINA
	):
		ctrl.combat.start_shield_bash()
		return true

	# Bull Rush (R)
	if (
		Input.is_action_just_pressed("heavy_attack")
		and ctrl.stamina >= ctrl.BULL_RUSH_STAMINA
		and ctrl._bull_rush_cooldown <= 0.0
	):
		ctrl.combat.start_bull_rush()
		return true

	# Shield Block (RMB)
	if (
		not cursor_active
		and Input.is_action_just_pressed("block")
		and ctrl._shield_block_cooldown <= 0.0
		and ctrl.stamina > 0.0
	):
		ctrl.combat.start_shield_block()
		return true

	# Dodge
	if (
		Input.is_action_just_pressed("dodge")
		and ctrl.is_on_floor()
		and ctrl.stamina >= ctrl.dodge_stamina_cost
	):
		ctrl.combat.start_dodge()
		return true

	return false


func _apply_movement(delta: float) -> void:
	# Movement
	var speed: float = ctrl.sprint_speed if Input.is_action_pressed("sprint") else ctrl.run_speed
	var wish_dir := get_camera_wish_dir()

	var on_floor: bool = ctrl.is_on_floor()
	var accel: float = ctrl.ground_accel if on_floor else ctrl.air_accel
	var decel: float = ctrl.ground_decel if on_floor else ctrl.air_decel

	if wish_dir.length() > 0.1:
		var target_vel: Vector3 = wish_dir * speed
		ctrl.velocity.x = move_toward(ctrl.velocity.x, target_vel.x, accel * delta)
		ctrl.velocity.z = move_toward(ctrl.velocity.z, target_vel.z, accel * delta)
		if ctrl._lock_on_active and ctrl._lock_target and is_instance_valid(ctrl._lock_target):
			face_target(delta)
		else:
			face_direction(wish_dir, delta)
	else:
		ctrl.velocity.x = move_toward(ctrl.velocity.x, 0.0, decel * delta)
		ctrl.velocity.z = move_toward(ctrl.velocity.z, 0.0, decel * delta)
		if ctrl._lock_on_active and ctrl._lock_target and is_instance_valid(ctrl._lock_target):
			face_target(delta)


func consume_stamina(amount: float) -> void:
	ctrl.stamina -= amount
	ctrl.stamina = maxf(ctrl.stamina, 0.0)
	_stamina_cooldown_timer = ctrl.stamina_regen_delay


func update_stamina(_delta: float) -> void:
	# Stamina is server-authoritative — no client-side prediction.
	pass
