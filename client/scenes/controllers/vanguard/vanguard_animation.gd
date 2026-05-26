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
		ctrl.State.CLEAVE:
			_update_blade_animation()
		ctrl.State.UPHEAVAL_WINDUP, ctrl.State.UPHEAVAL:
			_update_blade_animation()
		ctrl.State.BLOCK, ctrl.State.STAGGER:
			_update_blade_animation()
		ctrl.State.VORTEX:
			_update_blade_animation()
		ctrl.State.EXECUTION_WINDUP, ctrl.State.EXECUTION:
			_update_blade_animation()
		ctrl.State.SHIELD_BLOCK, ctrl.State.SHIELD_BASH:
			_update_shield_animation()
		ctrl.State.BULL_RUSH, ctrl.State.BRACE:
			_update_shield_animation()
		ctrl.State.RETALIATE_WINDUP, ctrl.State.RETALIATE:
			_update_shield_animation()
		ctrl.State.GUARD_BREAK:
			_update_shield_animation()
		ctrl.State.DEAD:
			ctrl._visual_state = NetSerializer.VS_DEAD
			ctrl.character_model.travel("dead")
		_:
			_update_movement_animation()


func _update_blade_animation() -> void:
	match ctrl.state:
		ctrl.State.CLEAVE:
			ctrl._visual_state = NetSerializer.VS_VG_LIGHT_1
			ctrl.character_model.travel_timed("cleave", ctrl.CLEAVE_DURATION)
		ctrl.State.UPHEAVAL_WINDUP:
			ctrl._visual_state = NetSerializer.VS_VG_HEAVY_WINDUP
			ctrl.character_model.travel_timed(
				"upheaval", ctrl.UPHEAVAL_WINDUP_TIME + ctrl.UPHEAVAL_HIT_TIME
			)
		ctrl.State.UPHEAVAL:
			ctrl._visual_state = NetSerializer.VS_VG_HEAVY
			ctrl.character_model.set_animation_speed(3.0)
		ctrl.State.BLOCK:
			ctrl._visual_state = NetSerializer.VS_VG_BLOCK
			ctrl.character_model.travel("block")
		ctrl.State.STAGGER:
			ctrl._visual_state = NetSerializer.VS_VG_STAGGER
			ctrl.character_model.travel("stagger")
		ctrl.State.VORTEX:
			ctrl._visual_state = NetSerializer.VS_VG_VORTEX
			ctrl.character_model.travel("vortex", 2.0)
		ctrl.State.EXECUTION_WINDUP:
			ctrl._visual_state = NetSerializer.VS_VG_EXECUTION_WINDUP
			ctrl.character_model.travel_timed(
				"execution", ctrl.EXECUTION_WINDUP_TIME + ctrl.EXECUTION_HIT_TIME
			)
		ctrl.State.EXECUTION:
			ctrl._visual_state = NetSerializer.VS_VG_EXECUTION
			ctrl.character_model.set_animation_speed(3.0)


func _update_shield_animation() -> void:
	# Shield states -- reuse Blade animations for Phase 0
	match ctrl.state:
		ctrl.State.SHIELD_BLOCK:
			ctrl._visual_state = NetSerializer.VS_VG_BLOCK
			ctrl.character_model.travel("block")
		ctrl.State.SHIELD_BASH:
			ctrl._visual_state = NetSerializer.VS_VG_LIGHT_1
			ctrl.character_model.travel_timed("cleave", ctrl.SHIELD_BASH_DURATION)
		ctrl.State.BULL_RUSH:
			ctrl._visual_state = NetSerializer.VS_VG_VORTEX
			ctrl.character_model.travel("run", 2.0)
		ctrl.State.BRACE:
			ctrl._visual_state = NetSerializer.VS_VG_BLOCK
			ctrl.character_model.travel("block")
		ctrl.State.RETALIATE_WINDUP:
			ctrl._visual_state = NetSerializer.VS_VG_EXECUTION_WINDUP
			ctrl.character_model.travel_timed(
				"execution", ctrl.RETALIATE_WINDUP_TIME + ctrl.RETALIATE_HIT_TIME
			)
		ctrl.State.RETALIATE:
			ctrl._visual_state = NetSerializer.VS_VG_EXECUTION
			ctrl.character_model.set_animation_speed(3.0)
		ctrl.State.GUARD_BREAK:
			ctrl._visual_state = NetSerializer.VS_VG_STAGGER
			ctrl.character_model.travel("stagger")


func _update_movement_animation() -> void:
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
