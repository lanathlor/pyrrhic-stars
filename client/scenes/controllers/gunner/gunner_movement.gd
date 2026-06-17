extends Node

## Gunner movement, gravity, jump, animation, and remote animation driving.

var ctrl: Node


func _ready() -> void:
	ctrl = get_parent()


func apply_gravity(delta: float) -> void:
	if not ctrl.is_on_floor():
		ctrl.velocity.y -= ctrl._gravity * delta
	else:
		ctrl.velocity.y = -0.5  # keep pressed to floor so is_on_floor() stays reliable


func handle_jump() -> void:
	if (
		Input.is_action_just_pressed("jump")
		and ctrl.is_on_floor()
		and not GameManager.text_input_active()
	):
		ctrl.velocity.y = ctrl.jump_velocity


func handle_movement(delta: float) -> void:
	var on_floor: bool = ctrl.is_on_floor()
	var speed: float = ctrl.sprint_speed if Input.is_action_pressed("sprint") else ctrl.walk_speed
	if ctrl._overclock_active:
		speed *= ctrl.OVERCLOCK_SPEED_MULT
	var accel: float = ctrl.ground_accel if on_floor else ctrl.air_accel
	var decel: float = ctrl.ground_decel if on_floor else ctrl.air_decel

	var input_dir := GameManager.move_vector()
	var raw: Vector3 = ctrl.transform.basis * Vector3(input_dir.x, 0.0, input_dir.y)
	var wish_dir: Vector3 = raw.normalized()

	var flat_vel: Vector3 = Vector3(ctrl.velocity.x, 0.0, ctrl.velocity.z)

	if wish_dir.length() > 0.1:
		# Accelerate toward desired direction
		var target_vel: Vector3 = wish_dir * speed
		flat_vel.x = move_toward(flat_vel.x, target_vel.x, accel * delta)
		flat_vel.z = move_toward(flat_vel.z, target_vel.z, accel * delta)
	else:
		# Decelerate to stop
		flat_vel.x = move_toward(flat_vel.x, 0.0, decel * delta)
		flat_vel.z = move_toward(flat_vel.z, 0.0, decel * delta)

	ctrl.velocity.x = flat_vel.x
	ctrl.velocity.z = flat_vel.z


func update_animation() -> void:
	if ctrl._is_rolling:
		ctrl._visual_state = NetSerializer.VS_DODGE
		ctrl.character_model.travel_timed("roll", ctrl.roll_duration)
		return
	if not ctrl.is_on_floor():
		ctrl._visual_state = NetSerializer.VS_AIRBORNE
		ctrl.character_model.travel("jump", 2.0)
		return
	ctrl._visual_state = NetSerializer.VS_MOVE
	var flat_vel: Vector3 = Vector3(ctrl.velocity.x, 0.0, ctrl.velocity.z)
	if flat_vel.length() > 0.5:
		var speed_ratio: float = flat_vel.length() / ctrl.sprint_speed
		ctrl.character_model.travel("run", clampf(speed_ratio, 0.5, 1.5))
	else:
		ctrl.character_model.travel("idle")


func drive_remote_animation(prev_pos: Vector3, delta: float) -> void:
	match ctrl._visual_state:
		NetSerializer.VS_DODGE:
			ctrl.character_model.travel("roll")
		NetSerializer.VS_AIRBORNE:
			ctrl.character_model.travel("jump", 2.0)
		NetSerializer.VS_DEAD:
			ctrl.character_model.travel("idle")
		_:  # VS_MOVE or unknown -- derive from velocity
			var vel: Vector3 = (
				(ctrl.global_position - prev_pos) / delta if delta > 0 else Vector3.ZERO
			)
			var speed: float = Vector2(vel.x, vel.z).length()
			if speed > 0.5:
				ctrl.character_model.travel("run", clampf(speed / ctrl.sprint_speed, 0.5, 1.5))
			else:
				ctrl.character_model.travel("idle")
