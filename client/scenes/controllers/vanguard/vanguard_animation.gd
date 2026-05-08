extends Node

## Vanguard animation state, weapon visuals, and damage flash effects.

var ctrl: Node


func _ready() -> void:
	ctrl = get_parent()


func attach_weapon() -> void:
	var offset_pos := Vector3(0.0, 0.08, 0.0)
	var offset_rot := Vector3(deg_to_rad(20.0), 0.0, deg_to_rad(-90.0))
	ctrl.character_model.attach_weapon(
		ctrl.WEAPON_SCENE, "mixamorig_RightHand", offset_pos, offset_rot
	)


func show_body_flash() -> void:
	ctrl.character_model.flash_damage()


func update_flash(_delta: float) -> void:
	pass


func update_weapon_visual() -> void:
	if not ctrl.character_model.weapon_node:
		return


func update_animation() -> void:
	match ctrl.state:
		ctrl.State.DODGE:
			ctrl._visual_state = NetSerializer.VS_DODGE
			ctrl.character_model.travel_timed("dodge", ctrl.dodge_duration)
			return
		ctrl.State.LIGHT_1:
			ctrl._visual_state = NetSerializer.VS_VG_LIGHT_1
			ctrl.character_model.travel_timed("light_1", ctrl.light_duration_1)
			return
		ctrl.State.LIGHT_2:
			ctrl._visual_state = NetSerializer.VS_VG_LIGHT_2
			ctrl.character_model.travel_timed("light_2", ctrl.light_duration_2)
			return
		ctrl.State.LIGHT_3:
			ctrl._visual_state = NetSerializer.VS_VG_LIGHT_3
			ctrl.character_model.travel_timed("light_3", ctrl.light_duration_3)
			return
		ctrl.State.HEAVY_WINDUP:
			ctrl._visual_state = NetSerializer.VS_VG_HEAVY_WINDUP
			ctrl.character_model.travel_timed(
				"heavy", ctrl.heavy_windup_time + ctrl.heavy_attack_duration
			)
			return
		ctrl.State.HEAVY:
			ctrl._visual_state = NetSerializer.VS_VG_HEAVY
			ctrl.character_model.set_animation_speed(3.0)
			return
		ctrl.State.BLOCK:
			ctrl._visual_state = NetSerializer.VS_VG_BLOCK
			ctrl.character_model.travel("block")
			return
		ctrl.State.STAGGER:
			ctrl._visual_state = NetSerializer.VS_VG_STAGGER
			ctrl.character_model.travel("stagger")
			return
		ctrl.State.BLADE_SWIRL:
			ctrl._visual_state = NetSerializer.VS_VG_BLADE_SWIRL
			ctrl.character_model.travel("blade_swirl", 2.0)
			return
		ctrl.State.GROUND_SLAM_WINDUP:
			ctrl._visual_state = NetSerializer.VS_VG_GROUND_SLAM_WINDUP
			ctrl.character_model.travel_timed(
				"ground_slam", ctrl.GROUND_SLAM_WINDUP_TIME + ctrl.GROUND_SLAM_HIT_TIME
			)
			return
		ctrl.State.GROUND_SLAM:
			ctrl._visual_state = NetSerializer.VS_VG_GROUND_SLAM
			ctrl.character_model.set_animation_speed(3.0)
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
