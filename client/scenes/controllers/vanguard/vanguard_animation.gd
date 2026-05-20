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
		ctrl.State.CLEAVE:
			ctrl._visual_state = NetSerializer.VS_VG_LIGHT_1
			ctrl.character_model.travel_timed("cleave", ctrl.CLEAVE_DURATION)
			return
		ctrl.State.UPHEAVAL_WINDUP:
			ctrl._visual_state = NetSerializer.VS_VG_HEAVY_WINDUP
			ctrl.character_model.travel_timed(
				"upheaval", ctrl.UPHEAVAL_WINDUP_TIME + ctrl.UPHEAVAL_HIT_TIME
			)
			return
		ctrl.State.UPHEAVAL:
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
		ctrl.State.VORTEX:
			ctrl._visual_state = NetSerializer.VS_VG_VORTEX
			ctrl.character_model.travel("vortex", 2.0)
			return
		ctrl.State.EXECUTION_WINDUP:
			ctrl._visual_state = NetSerializer.VS_VG_EXECUTION_WINDUP
			ctrl.character_model.travel_timed(
				"execution", ctrl.EXECUTION_WINDUP_TIME + ctrl.EXECUTION_HIT_TIME
			)
			return
		ctrl.State.EXECUTION:
			ctrl._visual_state = NetSerializer.VS_VG_EXECUTION
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
