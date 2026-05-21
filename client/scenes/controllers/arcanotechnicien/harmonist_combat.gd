extends Node

## Harmonist combat: spell casting, channeling, dodge, and ally targeting.
## Sends ability inputs to the server; server resolves healing.

var ctrl: Node

var _dodge_direction: Vector3 = Vector3.ZERO


func _ready() -> void:
	ctrl = get_parent()


# --- Spell Casting ---


func start_spell(slot: int) -> void:
	if slot < 0 or slot >= ctrl.HARMONIST_SPELLS.size():
		return

	var spell: Dictionary = ctrl.HARMONIST_SPELLS[slot]

	# Start cooldown for this slot
	if spell.cooldown_max > 0.0:
		ctrl._cooldowns[slot] = spell.cooldown_max

	ctrl._casting_spell = spell
	ctrl._cast_timer = spell.dur
	ctrl._gcd_timer = ctrl.gcd_duration

	# Send ability to server with action_id
	if NetworkManager.is_active:
		NetworkManager.send_ability(spell.action_id, 0.0, ctrl.rotation.y)

	# Determine state: short spells are instant casts, longer ones are channels
	if spell.dur > 0.5:
		ctrl._enter_state(ctrl.State.CHANNELING)
	else:
		ctrl._enter_state(ctrl.State.CASTING)


func process_casting(delta: float) -> void:
	ctrl.movement.face_attack_direction(delta)

	# Slow movement while casting
	var wish_dir: Vector3 = ctrl.movement.get_camera_wish_dir()
	var speed: float = ctrl.run_speed * ctrl.cast_move_speed_mult
	if wish_dir.length() > 0.1:
		var target_vel: Vector3 = wish_dir * speed
		ctrl.velocity.x = move_toward(ctrl.velocity.x, target_vel.x, ctrl.ground_accel * delta)
		ctrl.velocity.z = move_toward(ctrl.velocity.z, target_vel.z, ctrl.ground_accel * delta)
	else:
		ctrl.velocity.x = move_toward(ctrl.velocity.x, 0.0, ctrl.ground_decel * delta)
		ctrl.velocity.z = move_toward(ctrl.velocity.z, 0.0, ctrl.ground_decel * delta)

	ctrl._cast_timer -= delta
	if ctrl._cast_timer <= 0.0:
		ctrl._casting_spell = {}
		ctrl._enter_state(ctrl.State.MOVE)


func process_channeling(delta: float) -> void:
	ctrl.movement.face_attack_direction(delta)

	# Channeling is slower than casting -- nearly stationary
	var wish_dir: Vector3 = ctrl.movement.get_camera_wish_dir()
	var speed: float = ctrl.run_speed * 0.15
	if wish_dir.length() > 0.1:
		var target_vel: Vector3 = wish_dir * speed
		ctrl.velocity.x = move_toward(ctrl.velocity.x, target_vel.x, ctrl.ground_accel * delta)
		ctrl.velocity.z = move_toward(ctrl.velocity.z, target_vel.z, ctrl.ground_accel * delta)
	else:
		ctrl.velocity.x = move_toward(ctrl.velocity.x, 0.0, ctrl.ground_decel * delta)
		ctrl.velocity.z = move_toward(ctrl.velocity.z, 0.0, ctrl.ground_decel * delta)

	ctrl._cast_timer -= delta
	if ctrl._cast_timer <= 0.0:
		ctrl._casting_spell = {}
		ctrl._enter_state(ctrl.State.MOVE)


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


# --- Stagger ---


func process_stagger() -> void:
	ctrl.velocity.x = 0.0
	ctrl.velocity.z = 0.0
	if ctrl._state_timer <= 0.0:
		ctrl._enter_state(ctrl.State.MOVE)
