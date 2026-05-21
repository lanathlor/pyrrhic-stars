extends Node

## Arcanotechnicien locomotion, facing, and movement processing.
## Same pattern as vanguard_movement.gd / blade_dancer_movement.gd.

var ctrl: Node

var _facing_angle: float = 0.0


func _ready() -> void:
	ctrl = get_parent()


## Get world-space wish direction from input + actual camera transform.
func get_camera_wish_dir() -> Vector3:
	var input_dir := Input.get_vector("move_left", "move_right", "move_forward", "move_backward")
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
	if not ctrl._selected_target or not is_instance_valid(ctrl._selected_target):
		return
	var to_target: Vector3 = ctrl._selected_target.global_position - ctrl.global_position
	to_target.y = 0.0
	if to_target.length() > 0.1:
		var target_angle := get_target_yaw(to_target)
		_facing_angle = lerp_angle(_facing_angle, target_angle, ctrl.rotation_speed * delta)
		ctrl.rotation.y = _facing_angle


## Auto-face during casts: selected target > nearest ally (for heals) > camera forward.
func face_attack_direction(delta: float) -> void:
	if ctrl._selected_target and is_instance_valid(ctrl._selected_target):
		face_target(delta)
		return

	# For a healer, look toward nearest ally if no lock
	var best: Node3D = null
	var best_dist: float = ctrl.cast_range
	for player in GameManager.players:
		if not is_instance_valid(player) or not player.visible:
			continue
		if player == ctrl:
			continue
		var dist: float = ctrl.global_position.distance_to(player.global_position)
		if dist < best_dist:
			best_dist = dist
			best = player

	# Fallback to enemy if no allies nearby
	if not best:
		for enemy in GameManager.enemies:
			if not is_instance_valid(enemy) or not enemy.visible:
				continue
			var dist: float = ctrl.global_position.distance_to(enemy.global_position)
			if dist < best_dist:
				best_dist = dist
				best = enemy

	if best:
		var to_target: Vector3 = best.global_position - ctrl.global_position
		to_target.y = 0.0
		if to_target.length() > 0.1:
			var target_angle := get_target_yaw(to_target)
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


func process_move(delta: float) -> void:
	# Spell inputs (gated by GCD only -- cursor is always visible for this class)
	if ctrl._gcd_timer <= 0.0:
		for slot in 5:
			if Input.is_action_just_pressed(ctrl.SPELL_SLOT_ACTIONS[slot]):
				if ctrl._cooldowns[slot] <= 0.0:
					ctrl.combat.start_spell(slot)
					return

		# Slot 5 (C) -- Gust Step if available, otherwise dodge
		if Input.is_action_just_pressed("dodge") and ctrl.is_on_floor():
			if ctrl._cooldowns[5] <= 0.0:
				ctrl.combat.start_spell(5)
			else:
				ctrl.combat.start_dodge()
			return
	else:
		# GCD is active but still allow dodge
		if Input.is_action_just_pressed("dodge") and ctrl.is_on_floor():
			ctrl.combat.start_dodge()
			return

	# Jump
	if Input.is_action_just_pressed("jump") and ctrl.is_on_floor():
		ctrl.velocity.y = 3.5

	_apply_movement(delta)


func _apply_movement(delta: float) -> void:
	var speed: float = ctrl.sprint_speed if Input.is_action_pressed("sprint") else ctrl.run_speed
	var wish_dir: Vector3 = get_camera_wish_dir()

	var on_floor: bool = ctrl.is_on_floor()
	var accel: float = ctrl.ground_accel if on_floor else ctrl.air_accel
	var decel: float = ctrl.ground_decel if on_floor else ctrl.air_decel

	if wish_dir.length() > 0.1:
		var target_vel: Vector3 = wish_dir * speed
		ctrl.velocity.x = move_toward(ctrl.velocity.x, target_vel.x, accel * delta)
		ctrl.velocity.z = move_toward(ctrl.velocity.z, target_vel.z, accel * delta)
		# WoW behavior: while right-click held, character faces camera direction
		if ctrl._right_mouse_held:
			var cam_fwd: Vector3 = -ctrl.camera.global_transform.basis.z
			cam_fwd.y = 0.0
			if cam_fwd.length() > 0.01:
				face_direction(cam_fwd.normalized(), delta)
		else:
			face_direction(wish_dir, delta)
	else:
		ctrl.velocity.x = move_toward(ctrl.velocity.x, 0.0, decel * delta)
		ctrl.velocity.z = move_toward(ctrl.velocity.z, 0.0, decel * delta)
