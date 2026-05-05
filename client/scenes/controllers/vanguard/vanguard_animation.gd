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
			ctrl._net_anim = "roll"
			ctrl._net_anim_speed = 1.0
			ctrl.character_model.play_anim_timed("roll", ctrl.dodge_duration)
			return
		ctrl.State.LIGHT_1:
			ctrl._net_anim = "sword_slash_1"
			ctrl._net_anim_speed = 1.0
			ctrl.character_model.play_anim_timed("sword_slash_1", ctrl.light_duration_1)
			return
		ctrl.State.LIGHT_2:
			ctrl._net_anim = "sword_slash_2"
			ctrl._net_anim_speed = 1.0
			ctrl.character_model.play_anim_timed("sword_slash_2", ctrl.light_duration_2)
			return
		ctrl.State.LIGHT_3:
			ctrl._net_anim = "sword_slash_3"
			ctrl._net_anim_speed = 1.0
			ctrl.character_model.play_anim_timed("sword_slash_3", ctrl.light_duration_3)
			return
		ctrl.State.HEAVY_WINDUP:
			ctrl._net_anim = "sword_heavy"
			ctrl._net_anim_speed = 1.0
			ctrl.character_model.play_anim_timed(
				"sword_heavy", ctrl.heavy_windup_time + ctrl.heavy_attack_duration
			)
			return
		ctrl.State.HEAVY:
			ctrl._net_anim = "sword_heavy"
			ctrl._net_anim_speed = 3.0
			ctrl.character_model.set_animation_speed(3.0)
			return
		ctrl.State.BLOCK:
			ctrl._net_anim = "sword_block"
			ctrl._net_anim_speed = 1.0
			ctrl.character_model.play_anim("sword_block")
			return
		ctrl.State.STAGGER:
			ctrl._net_anim = "sword_impact"
			ctrl._net_anim_speed = 1.0
			ctrl.character_model.play_anim("sword_impact")
			return
		ctrl.State.BLADE_SWIRL:
			ctrl._net_anim = "sword_heavy"
			ctrl._net_anim_speed = 2.0
			ctrl.character_model.play_anim("sword_heavy", 2.0)
			return
		ctrl.State.GROUND_SLAM_WINDUP:
			ctrl._net_anim = "sword_heavy"
			ctrl._net_anim_speed = 0.5
			ctrl.character_model.play_anim_timed(
				"sword_heavy", ctrl.GROUND_SLAM_WINDUP_TIME + ctrl.GROUND_SLAM_HIT_TIME
			)
			return
		ctrl.State.GROUND_SLAM:
			ctrl._net_anim = "sword_heavy"
			ctrl._net_anim_speed = 3.0
			ctrl.character_model.set_animation_speed(3.0)
			return
		ctrl.State.DEAD:
			ctrl._net_anim = "sword_idle"
			ctrl._net_anim_speed = 1.0
			ctrl.character_model.play_anim("sword_idle")
			return

	if not ctrl.is_on_floor():
		ctrl._net_anim = "sword_jump"
		ctrl._net_anim_speed = 2.0
		ctrl.character_model.play_anim("sword_jump", 2.0)
		return

	var flat_vel := Vector3(ctrl.velocity.x, 0.0, ctrl.velocity.z)
	if flat_vel.length() > 0.5:
		var speed_ratio: float = flat_vel.length() / ctrl.sprint_speed
		ctrl._net_anim_speed = clampf(speed_ratio, 0.5, 1.5)
		ctrl._net_anim = "sword_run"
		ctrl.character_model.play_anim("sword_run", ctrl._net_anim_speed)
	else:
		ctrl._net_anim = "sword_idle"
		ctrl._net_anim_speed = 1.0
		ctrl.character_model.play_anim("sword_idle")
