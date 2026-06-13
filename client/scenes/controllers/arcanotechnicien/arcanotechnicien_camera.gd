extends Node

## Arcanotechnicien camera: positioning, collision, animation, and visual flash.
## WoW-style: player controls rotation via right-click drag. No auto lock-on.

var ctrl: Node


func _ready() -> void:
	ctrl = get_parent()


func update_camera() -> void:
	var player_pos: Vector3 = ctrl.global_position + Vector3(0.0, ctrl.camera_height_offset, 0.0)
	var cam_offset: Vector3 = Vector3(0.0, 0.0, ctrl.camera_distance)
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
			var dur: float = ctrl._committing_ability.get("dur", 0.4)
			ctrl.character_model.travel_timed("casting", dur)
			return
		ctrl.State.CHANNELING:
			if ctrl.combat._sustaining:
				ctrl._visual_state = NetSerializer.VS_AT_SUSTAINING
			else:
				var delivery: String = ctrl._committing_ability.get("delivery", "")
				match delivery:
					"beam":
						ctrl._visual_state = NetSerializer.VS_AT_CHANNELING_BEAM
					"zone":
						ctrl._visual_state = NetSerializer.VS_AT_CHANNELING_ZONE
					_:
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
	var flat_vel: Vector3 = Vector3(ctrl.velocity.x, 0.0, ctrl.velocity.z)
	ctrl.character_model.travel_locomotion(flat_vel.length(), ctrl.run_speed, ctrl.sprint_speed)


func drive_remote_animation(prev_pos: Vector3, delta: float) -> void:
	# Drive remote VFX on visual state change
	if ctrl._visual_state != ctrl._prev_remote_vs:
		if ctrl.vfx:
			ctrl.vfx.drive_remote_vfx(ctrl._prev_remote_vs, ctrl._visual_state)
		ctrl._prev_remote_vs = ctrl._visual_state

	match ctrl._visual_state:
		NetSerializer.VS_DODGE:
			ctrl.character_model.travel("dodge")
		NetSerializer.VS_AT_CASTING:
			ctrl.character_model.travel("casting")
		NetSerializer.VS_AT_CHANNELING:
			ctrl.character_model.travel("channeling")
		NetSerializer.VS_AT_CHANNELING_BEAM:
			ctrl.character_model.travel("channeling")
		NetSerializer.VS_AT_CHANNELING_ZONE:
			ctrl.character_model.travel("channeling")
		NetSerializer.VS_AT_STAGGER:
			ctrl.character_model.travel("stagger")
		NetSerializer.VS_DEAD:
			ctrl.character_model.travel("dead")
		_:  # VS_MOVE or unknown -- derive from velocity
			var vel: Vector3 = (
				(ctrl.global_position - prev_pos) / delta if delta > 0 else Vector3.ZERO
			)
			var speed: float = Vector2(vel.x, vel.z).length()
			ctrl.character_model.travel_locomotion(speed, ctrl.run_speed, ctrl.sprint_speed)
