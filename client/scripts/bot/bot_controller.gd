extends Node

## Combat bot AI for the Gunner. Maintains range, shoots, dodges telegraphs.
## Attach as a child of a Gunner CharacterBody3D.

@export var engage_range: float = 10.0  # ideal combat distance
@export var too_close: float = 5.0      # back off below this
@export var strafe_change_interval: float = 1.5  # seconds between strafe direction swaps

var _player: CharacterBody3D
var _strafe_timer: float = 0.0
var _strafe_dir: float = 1.0  # 1.0 = right, -1.0 = left


func _ready() -> void:
	_player = get_parent() as CharacterBody3D
	if not _player:
		push_error("[Bot] Must be a child of CharacterBody3D")
		queue_free()
		return
	# Capture mouse so the gunner script doesn't fight us
	Input.set_mouse_mode(Input.MOUSE_MODE_CAPTURED)
	print("[Bot] Ready, controlling %s" % _player.name)


func _physics_process(delta: float) -> void:
	if not _player or not is_instance_valid(_player):
		return

	var target := _find_target()
	if not target:
		_release_all()
		return

	var to_target := target.global_position - _player.global_position
	to_target.y = 0.0
	var distance := to_target.length()
	var dir := to_target.normalized() if distance > 0.1 else Vector3.FORWARD

	_aim_at(target)
	_update_strafe(delta)

	# Dodge melee telegraphs
	if _should_dodge(target, distance):
		Input.action_press("dodge")
	else:
		Input.action_release("dodge")

	# Movement: maintain range
	if distance > engage_range:
		_move_toward(dir)
	elif distance < too_close:
		_move_away(dir)
	else:
		_strafe(dir)

	# Shoot if we have line of sight and aren't rolling
	var is_rolling: bool = _player._is_rolling if "_is_rolling" in _player else false
	var is_sprinting := Input.is_action_pressed("sprint")
	if not is_rolling and not is_sprinting:
		Input.action_press("shoot")
	else:
		Input.action_release("shoot")


func _find_target() -> CharacterBody3D:
	return GameManager.get_nearest_player(_player.global_position) if GameManager.enemies.is_empty() \
		else GameManager.enemies[0] if is_instance_valid(GameManager.enemies[0]) \
		else null


func _aim_at(target: CharacterBody3D) -> void:
	# Aim at enemy center mass
	var aim_pos := target.global_position + Vector3(0.0, 1.0, 0.0)
	var to_aim := aim_pos - _player.global_position

	# Yaw
	_player.rotation.y = atan2(-to_aim.x, -to_aim.z)

	# Pitch on head
	var head: Node3D = _player.get_node_or_null("Head")
	if head:
		var flat_dist := Vector2(to_aim.x, to_aim.z).length()
		var pitch := atan2(to_aim.y - 1.6, flat_dist)
		head.rotation.x = clampf(pitch, deg_to_rad(-89.0), deg_to_rad(89.0))


func _should_dodge(target: CharacterBody3D, distance: float) -> bool:
	if not "state" in target:
		return false
	# Dodge when enemy is in melee telegraph and we're close
	if target.state == target.State.MELEE_TELEGRAPH and distance < target.melee_range * 1.5:
		return true
	# Dodge AoE slam telegraph — get out of radius
	if target.state == target.State.AOE_TELEGRAPH and distance < 8.0:
		return true
	# Dodge charge telegraph — strafe to evade
	if target.state == target.State.CHARGE_TELEGRAPH:
		return true
	return false


func _move_toward(dir: Vector3) -> void:
	# Convert world direction to local input
	var local := _player.transform.basis.inverse() * dir
	_set_movement(local.x, local.z)


func _move_away(dir: Vector3) -> void:
	var local := _player.transform.basis.inverse() * (-dir)
	_set_movement(local.x, local.z)


func _strafe(dir: Vector3) -> void:
	# Strafe perpendicular to enemy direction
	var right := dir.cross(Vector3.UP).normalized()
	var strafe := right * _strafe_dir
	var local := _player.transform.basis.inverse() * strafe
	_set_movement(local.x, local.z)


func _update_strafe(delta: float) -> void:
	_strafe_timer -= delta
	if _strafe_timer <= 0.0:
		_strafe_dir *= -1.0
		_strafe_timer = strafe_change_interval


func _set_movement(local_x: float, local_z: float) -> void:
	# Map local direction to input actions
	if local_z < -0.3:
		Input.action_press("move_forward")
		Input.action_release("move_backward")
	elif local_z > 0.3:
		Input.action_press("move_backward")
		Input.action_release("move_forward")
	else:
		Input.action_release("move_forward")
		Input.action_release("move_backward")

	if local_x > 0.3:
		Input.action_press("move_right")
		Input.action_release("move_left")
	elif local_x < -0.3:
		Input.action_press("move_left")
		Input.action_release("move_right")
	else:
		Input.action_release("move_left")
		Input.action_release("move_right")


func _release_all() -> void:
	for action in ["move_forward", "move_backward", "move_left", "move_right", "shoot", "sprint", "dodge"]:
		Input.action_release(action)
