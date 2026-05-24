extends Node

## Blade Dancer abilities: committing, telegraphs, GCD, and hit detection.

const PlayerTelegraph := preload("res://scenes/shared/telegraph/player_telegraph.gd")
const TELEGRAPH_COLOR := Color(0.2, 0.75, 0.9, 0.4)

var ctrl: Node


func _ready() -> void:
	ctrl = get_parent()


func start_ability(slot: int) -> void:
	var slot_abilities: Array = ctrl.ABILITY_TABLE[ctrl.config]
	if slot < 0 or slot >= slot_abilities.size():
		return
	var ability: Dictionary = slot_abilities[slot]

	ctrl._committing_ability = ability
	ctrl._cast_timer = ability.dur
	ctrl._gcd_timer = ctrl.gcd_duration

	# Send ability to server
	if NetworkManager.is_active:
		NetworkManager.send_ability(ability.action_id, 0.0, ctrl.rotation.y)

	# Spawn telegraph if the ability has one
	_spawn_ability_telegraph(ability)

	# Client-side raycast for optimistic hit feedback
	perform_raycast_hit(ctrl.ability_range)

	ctrl._enter_state(ctrl.State.CASTING)


func _spawn_ability_telegraph(ability: Dictionary) -> void:
	var telegraph_type: String = ability.get("telegraph", "none")
	if telegraph_type == "none":
		return

	var ability_radius: float = ability.get("radius", 5.0)

	if telegraph_type == "circle":
		PlayerTelegraph.spawn_circle(
			ctrl.get_tree().root, ctrl.global_position, ability_radius, TELEGRAPH_COLOR
		)
	elif telegraph_type == "circle_target":
		var target_pos: Vector3 = get_aim_target_position()
		if target_pos != Vector3.ZERO:
			PlayerTelegraph.spawn_circle(
				ctrl.get_tree().root, target_pos, ability_radius, TELEGRAPH_COLOR
			)


func get_aim_target_position() -> Vector3:
	var origin: Vector3 = ctrl.global_position + Vector3(0.0, 1.0, 0.0)
	var direction: Vector3
	if ctrl._lock_on_active and ctrl._lock_target and is_instance_valid(ctrl._lock_target):
		return ctrl._lock_target.global_position
	# Raycast to find enemy
	direction = -ctrl.transform.basis.z
	direction.y = 0.0
	direction = direction.normalized()
	var space: PhysicsDirectSpaceState3D = ctrl.get_world_3d().direct_space_state
	if not space:
		return Vector3.ZERO
	var query := PhysicsRayQueryParameters3D.create(origin, origin + direction * 20.0, 4)
	query.exclude = [ctrl.get_rid()]
	var result: Dictionary = space.intersect_ray(query)
	if result:
		return result.position
	return Vector3.ZERO


func process_casting(delta: float) -> void:
	ctrl.movement.face_attack_direction(delta)

	# Slow movement while committing
	var wish_dir: Vector3 = ctrl.movement.get_camera_wish_dir()
	var speed: float = ctrl.run_speed * 0.4
	if wish_dir.length() > 0.1:
		var target_vel: Vector3 = wish_dir * speed
		ctrl.velocity.x = move_toward(ctrl.velocity.x, target_vel.x, ctrl.ground_accel * delta)
		ctrl.velocity.z = move_toward(ctrl.velocity.z, target_vel.z, ctrl.ground_accel * delta)
	else:
		ctrl.velocity.x = move_toward(ctrl.velocity.x, 0.0, ctrl.ground_decel * delta)
		ctrl.velocity.z = move_toward(ctrl.velocity.z, 0.0, ctrl.ground_decel * delta)

	ctrl._cast_timer -= delta
	if ctrl._cast_timer <= 0.0 and not ctrl._committing_ability.is_empty():
		# Transition config on commit completion
		var dest_config: int = ctrl._committing_ability.dest
		ctrl.config = dest_config
		ctrl.hud.update_config(ctrl.config)
		ctrl.hud.update_abilities(ctrl.ABILITY_TABLE[ctrl.config])
		ctrl._committing_ability = {}
		ctrl._enter_state(ctrl.State.MOVE)


func perform_raycast_hit(max_range: float) -> void:
	# Server resolves hits -- client only shows optimistic hit marker
	var origin: Vector3 = ctrl.global_position + Vector3(0.0, 1.0, 0.0)
	var direction: Vector3
	if ctrl._lock_on_active and ctrl._lock_target and is_instance_valid(ctrl._lock_target):
		direction = (
			(ctrl._lock_target.global_position + Vector3(0.0, 1.0, 0.0) - origin).normalized()
		)
	else:
		direction = -ctrl.transform.basis.z
		direction.y = 0.0
		direction = direction.normalized()

	var space: PhysicsDirectSpaceState3D = ctrl.get_world_3d().direct_space_state
	if not space:
		return
	var query := PhysicsRayQueryParameters3D.create(origin, origin + direction * max_range, 4 | 1)
	query.exclude = [ctrl.get_rid()]
	space.intersect_ray(query)
	# Hit marker now driven by server-confirmed damage events (on_hit_confirmed)
