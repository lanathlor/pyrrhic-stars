extends Node

## Vanguard camera positioning, collision, and lock-on targeting.

var ctrl: Node


func _ready() -> void:
	ctrl = get_parent()


func update_camera() -> void:
	var player_pos: Vector3 = ctrl.global_position + Vector3(0.0, ctrl.camera_height_offset, 0.0)
	var delta: float = ctrl.get_physics_process_delta_time()

	if ctrl._lock_on_active and ctrl._lock_target and is_instance_valid(ctrl._lock_target):
		var target_pos: Vector3 = ctrl._lock_target.global_position + Vector3(0.0, 1.0, 0.0)
		var midpoint: Vector3 = player_pos.lerp(target_pos, 0.4)

		var to_target: Vector3 = target_pos - player_pos
		var desired_yaw: float = atan2(to_target.x, to_target.z) + PI

		ctrl._camera_yaw = lerp_angle(ctrl._camera_yaw, desired_yaw, 6.0 * delta)

		var height_diff: float = target_pos.y - player_pos.y
		var desired_pitch: float = clampf(
			-0.2 - height_diff * 0.05, deg_to_rad(-60.0), deg_to_rad(20.0)
		)
		ctrl._camera_pitch = lerp(ctrl._camera_pitch, desired_pitch, 4.0 * delta)

		var cam_offset := Vector3(0.0, 0.0, ctrl.camera_distance)
		cam_offset = cam_offset.rotated(Vector3.RIGHT, ctrl._camera_pitch)
		cam_offset = cam_offset.rotated(Vector3.UP, ctrl._camera_yaw)
		var desired_cam_pos: Vector3 = player_pos + cam_offset
		ctrl.camera.global_position = apply_camera_collision(player_pos, desired_cam_pos)
		ctrl.camera.look_at(midpoint, Vector3.UP)
	else:
		var cam_offset := Vector3(0.0, 0.0, ctrl.camera_distance)
		cam_offset = cam_offset.rotated(Vector3.RIGHT, ctrl._camera_pitch)
		cam_offset = cam_offset.rotated(Vector3.UP, ctrl._camera_yaw)
		var desired_cam_pos: Vector3 = player_pos + cam_offset
		ctrl.camera.global_position = apply_camera_collision(player_pos, desired_cam_pos)
		ctrl.camera.look_at(player_pos, Vector3.UP)


func apply_camera_collision(from: Vector3, to: Vector3) -> Vector3:
	var space: PhysicsDirectSpaceState3D = ctrl.get_world_3d().direct_space_state
	if not space:
		return to
	var query := PhysicsRayQueryParameters3D.create(from, to, 1)
	query.exclude = [ctrl.get_rid()]
	var result: Dictionary = space.intersect_ray(query)
	if result:
		return result.position + (from - to).normalized() * 0.3
	return to


func toggle_lock_on() -> void:
	if ctrl._lock_on_active:
		ctrl._lock_on_active = false
		ctrl._lock_target = null
		ctrl.hud.hide_lock_on()
	else:
		var target := find_lock_target()
		if target:
			ctrl._lock_on_active = true
			ctrl._lock_target = target
			ctrl.hud.show_lock_on()


func find_lock_target() -> Node3D:
	var best: Node3D = null
	var best_dist: float = 30.0
	for enemy in GameManager.enemies:
		if not is_instance_valid(enemy) or not enemy.visible:
			continue
		var dist: float = ctrl.global_position.distance_to(enemy.global_position)
		if dist < best_dist:
			best_dist = dist
			best = enemy
	return best


func nearest_enemy_distance() -> float:
	var best := INF
	for enemy in GameManager.enemies:
		if not is_instance_valid(enemy) or not enemy.visible:
			continue
		var d: float = ctrl.global_position.distance_to(enemy.global_position)
		if d < best:
			best = d
	return best
