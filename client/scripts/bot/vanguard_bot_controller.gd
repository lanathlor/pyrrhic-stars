extends Node

## Combat bot AI for the Vanguard. Closes to melee, combos, dodges telegraphs, blocks.
## Attach as a child of a Vanguard CharacterBody3D.

var _player: CharacterBody3D
var _strafe_timer: float = 0.0
var _strafe_dir: float = 1.0
var _attack_timer: float = 0.0


func _ready() -> void:
	_player = get_parent() as CharacterBody3D
	if not _player:
		push_error("[VanguardBot] Must be a child of CharacterBody3D")
		queue_free()
		return
	process_physics_priority = -1  # Run before player so input reads work
	Input.set_mouse_mode(Input.MOUSE_MODE_CAPTURED)
	print("[VanguardBot] Ready, controlling %s" % _player.name)


func _physics_process(delta: float) -> void:
	if not _player or not is_instance_valid(_player):
		return

	if _player.state == _player.State.DEAD:
		_release_movement()
		return

	var target := _find_target()
	if not target:
		_release_movement()
		# No enemy visible — walk toward arena to trigger fight
		Input.action_press("move_forward")
		return

	var to_target := target.global_position - _player.global_position
	to_target.y = 0.0
	var distance := to_target.length()
	var dir := to_target.normalized() if distance > 0.1 else Vector3.FORWARD

	_attack_timer = maxf(_attack_timer - delta, 0.0)
	_update_strafe(delta)
	_release_movement()

	# Let committed states (dodge, stagger) play out
	if _player.state in [_player.State.DODGE, _player.State.STAGGER]:
		return

	# Queue combo continuation during light attacks
	if _player.state in [_player.State.LIGHT_1, _player.State.LIGHT_2]:
		if _player.stamina >= _player.light_stamina_cost:
			_player._queued_light = true
		return

	# Let heavy windup / heavy / light_3 play out
	if _player.state in [_player.State.HEAVY_WINDUP, _player.State.HEAVY, _player.State.LIGHT_3]:
		return

	var melee_range: float = _player.melee_range

	# --- Priority 1: Dodge AoE slam telegraph ---
	if _is_enemy_state(target, "AOE_TELEGRAPH") and distance < 8.0:
		if (
			_player.stamina >= _player.dodge_stamina_cost
			and _player.is_on_floor()
			and _player.state == _player.State.MOVE
		):
			_move_away(dir)
			_player._start_dodge()
			return
		elif _player.state == _player.State.MOVE:
			_move_away(dir)
			Input.action_press("sprint")
			return

	# --- Priority 2: Dodge charge telegraph ---
	if _is_enemy_state(target, "CHARGE_TELEGRAPH"):
		if (
			_player.stamina >= _player.dodge_stamina_cost
			and _player.is_on_floor()
			and _player.state == _player.State.MOVE
		):
			_move_strafe(dir)
			_player._start_dodge()
			return
		elif _player.state == _player.State.MOVE:
			Input.action_press("block")
			return

	# --- Priority 3: Dodge melee telegraph ---
	if _is_enemy_state(target, "MELEE_TELEGRAPH") and distance < melee_range * 1.8:
		if (
			_player.stamina >= _player.dodge_stamina_cost
			and _player.is_on_floor()
			and _player.state == _player.State.MOVE
		):
			_move_strafe(dir)
			_player._start_dodge()
			return
		elif _player.state == _player.State.MOVE:
			Input.action_press("block")
			return

	# --- Priority 4: Strafe during ranged telegraph ---
	if _is_enemy_state(target, "RANGED_TELEGRAPH"):
		_move_strafe(dir)
		if distance > melee_range:
			_move_toward(dir)
		return

	# --- Priority 5: Block active melee swing or charge ---
	if (
		(_is_enemy_state(target, "MELEE_ATTACK") or _is_enemy_state(target, "CHARGE"))
		and distance < melee_range * 2.0
	):
		if _player.state == _player.State.MOVE:
			Input.action_press("block")
		return

	# --- Priority 6: Recover stamina when low ---
	if _player.stamina < 20.0:
		if distance < melee_range * 1.5:
			_move_away(dir)
		return

	# --- Priority 7: Attack when in range (during cooldown, chase, or aoe telegraph) ---
	var is_punish_window := (
		_is_enemy_state(target, "COOLDOWN")
		or _is_enemy_state(target, "CHASE")
		or _is_enemy_state(target, "RANGED_TELEGRAPH")
		or _is_enemy_state(target, "AOE_TELEGRAPH")
	)
	if distance <= melee_range * 0.95 and _player.state == _player.State.MOVE and is_punish_window:
		if _attack_timer <= 0.0:
			# Heavy during enemy cooldown (big punish window)
			if (
				_is_enemy_state(target, "COOLDOWN")
				and _player.stamina >= _player.heavy_stamina_cost
			):
				_player._start_heavy_attack()
				_attack_timer = 1.0
			elif _player.stamina >= _player.light_stamina_cost:
				_player._start_light_attack(1)
				_attack_timer = 0.15
		return

	# --- Priority 8: Block during enemy ranged attack ---
	if _is_enemy_state(target, "RANGED_ATTACK") and _player.state == _player.State.MOVE:
		Input.action_press("block")
		return

	# --- Priority 9: Close distance (always rush in) ---
	if distance > melee_range * 0.7:
		_move_toward(dir)
		if distance > melee_range * 1.5:
			Input.action_press("sprint")


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
	for action in ["move_forward", "move_backward", "move_left", "move_right", "sprint", "block"]:
		Input.action_release(action)
