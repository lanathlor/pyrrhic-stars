extends Node

## Arcanotechnicien camera: positioning, lock-on, collision, animation, and visual flash.
## Pulled back further than melee classes for spatial awareness of healing targets.

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


## Find lock target: prioritize allies for healing, fallback to enemies.
func find_lock_target() -> Node3D:
	# First check allies (for heal targeting)
	var best_ally: Node3D = null
	var best_ally_dist: float = 30.0
	for player in GameManager.players:
		if not is_instance_valid(player) or not player.visible:
			continue
		if player == ctrl:
			continue
		var dist: float = ctrl.global_position.distance_to(player.global_position)
		if dist < best_ally_dist:
			best_ally_dist = dist
			best_ally = player
	if best_ally:
		return best_ally

	# Fallback to nearest enemy
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


func show_body_flash() -> void:
	ctrl.character_model.flash_damage()


func update_flash(_delta: float) -> void:
	pass


func update_animation() -> void:
	match ctrl.state:
		ctrl.State.DODGE:
			ctrl._visual_state = NetSerializer.VS_DODGE
			ctrl.character_model.travel_timed("dodge", ctrl.dodge_duration)
			return
		ctrl.State.CASTING:
			ctrl._visual_state = NetSerializer.VS_AT_CASTING
			var dur: float = ctrl._casting_spell.get("dur", 0.4)
			ctrl.character_model.travel_timed("casting", dur)
			return
		ctrl.State.CHANNELING:
			ctrl._visual_state = NetSerializer.VS_AT_CHANNELING
			ctrl.character_model.travel("channeling")
			return
		ctrl.State.STAGGER:
			ctrl._visual_state = NetSerializer.VS_AT_STAGGER
			ctrl.character_model.travel("stagger")
			return
		ctrl.State.DEAD:
			ctrl._visual_state = NetSerializer.VS_DEAD
			ctrl.character_model.travel("dead")
			return

	if not ctrl.is_on_floor():
		ctrl._visual_state = NetSerializer.VS_AIRBORNE
		ctrl.character_model.travel("jump", 2.0)
		return

	ctrl._visual_state = NetSerializer.VS_MOVE
	var flat_vel := Vector3(ctrl.velocity.x, 0.0, ctrl.velocity.z)
	if flat_vel.length() > 0.5:
		var speed_ratio: float = flat_vel.length() / ctrl.sprint_speed
		ctrl.character_model.travel("run", clampf(speed_ratio, 0.5, 1.5))
	else:
		ctrl.character_model.travel("idle")
