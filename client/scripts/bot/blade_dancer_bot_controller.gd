extends Node

## Combat bot AI for the Blade Dancer. Chains Edge/Surge rotation, uses Guard and Dash defensively.
## Attach as a child of a BladeDancer CharacterBody3D.

var _player: CharacterBody3D
var _strafe_timer: float = 0.0
var _strafe_dir: float = 1.0


func _ready() -> void:
	_player = get_parent() as CharacterBody3D
	if not _player:
		push_error("[BladeDancerBot] Must be a child of CharacterBody3D")
		queue_free()
		return
	process_physics_priority = -1
	Input.set_mouse_mode(Input.MOUSE_MODE_CAPTURED)
	print("[BladeDancerBot] Ready, controlling %s" % _player.name)


func _physics_process(delta: float) -> void:
	if not _player or not is_instance_valid(_player):
		return

	if _player.state == _player.State.DEAD:
		_release_movement()
		return

	var target := _find_target()
	if not target:
		_release_movement()
		Input.action_press("move_forward")
		return

	# Auto lock-on
	if not _player._lock_on_active:
		_player._toggle_lock_on()

	var to_target := target.global_position - _player.global_position
	to_target.y = 0.0
	var distance := to_target.length()
	var dir := to_target.normalized() if distance > 0.1 else Vector3.FORWARD

	_update_strafe(delta)
	_release_movement()

	# Let committed states play out
	if _player.state in [_player.State.DASH, _player.State.STAGGER]:
		return
	if _player.state in [_player.State.EDGE, _player.State.SURGE, _player.State.SURGE_WINDUP]:
		return
	if _player.state == _player.State.RECALL:
		return

	# Let Guard play out (but can interrupt with dodge if needed)
	if _player.state == _player.State.GUARD:
		# Emergency dodge out of guard for AoE
		if _is_enemy_state(target, "AOE_TELEGRAPH") and distance < 8.0:
			if _player._gcd_timer <= 0.0 and _player.is_on_floor():
				_move_away(dir)
				_player._start_dash()
			return
		return

	# --- Priority 1: Dodge AoE slam telegraph ---
	if _is_enemy_state(target, "AOE_TELEGRAPH") and distance < 8.0:
		if _player._gcd_timer <= 0.0 and _player.is_on_floor() and _player.state == _player.State.MOVE:
			_move_away(dir)
			_player._start_dash()
			return
		elif _player.state == _player.State.MOVE:
			_move_away(dir)
			Input.action_press("sprint")
			return

	# --- Priority 2: Dodge charge telegraph ---
	if _is_enemy_state(target, "CHARGE_TELEGRAPH"):
		if _player._gcd_timer <= 0.0 and _player.is_on_floor() and _player.state == _player.State.MOVE:
			_move_strafe(dir)
			_player._start_dash()
			return
		elif _player.state == _player.State.MOVE:
			# Use Guard as fallback
			if _player._gcd_timer <= 0.0 and _player.config == _player.Config.ORBIT:
				_player._start_guard()
			else:
				_move_strafe(dir)
			return

	# --- Priority 3: Guard during melee telegraph ---
	if _is_enemy_state(target, "MELEE_TELEGRAPH") and distance < 5.0:
		if _player._gcd_timer <= 0.0 and _player.state == _player.State.MOVE:
			if _player.config == _player.Config.ORBIT:
				_player._start_guard()
			else:
				# In Lance, dash away
				_player._start_dash()
			return

	# --- Priority 4: Strafe during ranged telegraph ---
	if _is_enemy_state(target, "RANGED_TELEGRAPH"):
		_move_strafe(dir)
		if distance > 3.0:
			_move_toward(dir)
		return

	# --- Priority 5: Guard during active melee/charge ---
	if (_is_enemy_state(target, "MELEE_ATTACK") or _is_enemy_state(target, "CHARGE")) and distance < 5.0:
		if _player._gcd_timer <= 0.0 and _player.state == _player.State.MOVE:
			if _player.config == _player.Config.ORBIT:
				_player._start_guard()
			else:
				_player._start_dash()
		return

	# --- Priority 6: DPS rotation (ranged caster — fire from distance) ---
	if _player.state == _player.State.MOVE and _player._gcd_timer <= 0.0 and distance <= 18.0:
		_do_dps_rotation()
		return

	# --- Priority 7: Maintain optimal range (8-12m) ---
	if distance > 14.0:
		_move_toward(dir)
		if distance > 18.0:
			Input.action_press("sprint")
	elif distance < 6.0:
		_move_away(dir)
		if distance < 3.0:
			Input.action_press("sprint")


# --- DPS rotation ---

func _do_dps_rotation() -> void:
	# Optimal: Orbit Edge (->Lance) -> Lance Surge (->Orbit) -> repeat
	# Gives 25 + 50 = 75 damage per 2 GCDs
	match _player.config:
		_player.Config.ORBIT:
			_player._start_edge()
		_player.Config.LANCE:
			_player._start_surge()


# --- Targeting ---

func _find_target() -> CharacterBody3D:
	for enemy in GameManager.enemies:
		if is_instance_valid(enemy) and enemy.visible:
			return enemy
	return null


func _is_enemy_state(target: Node3D, state_name: String) -> bool:
	if not "state" in target or not "State" in target:
		return false
	var state_enum = target.State
	if not state_name in state_enum:
		return false
	return target.state == state_enum[state_name]


# --- Camera-relative movement ---

func _world_to_camera_input(world_dir: Vector3) -> Vector2:
	var cam: Camera3D = _player.camera
	if not cam:
		return Vector2.ZERO
	var cam_forward := -cam.global_transform.basis.z
	cam_forward.y = 0.0
	if cam_forward.length() < 0.01:
		return Vector2.ZERO
	cam_forward = cam_forward.normalized()
	var cam_right := cam.global_transform.basis.x
	cam_right.y = 0.0
	cam_right = cam_right.normalized()
	return Vector2(world_dir.dot(cam_right), world_dir.dot(cam_forward))


func _apply_movement(cam_input: Vector2) -> void:
	if cam_input.y > 0.3:
		Input.action_press("move_forward")
	elif cam_input.y < -0.3:
		Input.action_press("move_backward")
	if cam_input.x > 0.3:
		Input.action_press("move_right")
	elif cam_input.x < -0.3:
		Input.action_press("move_left")


func _move_toward(dir: Vector3) -> void:
	_apply_movement(_world_to_camera_input(dir))


func _move_away(dir: Vector3) -> void:
	_apply_movement(_world_to_camera_input(-dir))


func _move_strafe(dir: Vector3) -> void:
	var right := dir.cross(Vector3.UP).normalized()
	_apply_movement(_world_to_camera_input(right * _strafe_dir))


# --- Strafe timing ---

func _update_strafe(delta: float) -> void:
	_strafe_timer -= delta
	if _strafe_timer <= 0.0:
		_strafe_dir *= -1.0
		_strafe_timer = 1.5 + randf() * 1.0


# --- Cleanup ---

func _release_movement() -> void:
	for action in ["move_forward", "move_backward", "move_left", "move_right", "sprint"]:
		Input.action_release(action)
