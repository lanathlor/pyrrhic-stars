extends Node

## Vanguard remote player animation and VFX driving.

var ctrl: Node


func _ready() -> void:
	ctrl = get_parent()


func drive_remote_animation(prev_pos: Vector3, delta: float) -> void:
	var vs_changed: bool = ctrl._visual_state != ctrl._prev_remote_vs

	# VFX transitions on state change
	if vs_changed:
		_drive_remote_vfx(ctrl._prev_remote_vs, ctrl._visual_state)
		ctrl._prev_remote_vs = ctrl._visual_state

	match ctrl._visual_state:
		NetSerializer.VS_DODGE:
			ctrl.character_model.travel("dodge")
		NetSerializer.VS_VG_LIGHT_1:
			ctrl.character_model.travel("cleave")
		NetSerializer.VS_VG_LIGHT_2, NetSerializer.VS_VG_LIGHT_3:
			ctrl.character_model.travel("cleave")
		NetSerializer.VS_VG_HEAVY_WINDUP, NetSerializer.VS_VG_HEAVY:
			ctrl.character_model.travel("upheaval")
		NetSerializer.VS_VG_BLOCK:
			ctrl.character_model.travel("block")
		NetSerializer.VS_VG_STAGGER:
			ctrl.character_model.travel("stagger")
		NetSerializer.VS_VG_VORTEX:
			ctrl.character_model.travel("vortex", 2.0)
		NetSerializer.VS_VG_EXECUTION_WINDUP, NetSerializer.VS_VG_EXECUTION:
			ctrl.character_model.travel("execution")
		NetSerializer.VS_DEAD:
			ctrl.character_model.travel("dead")
		_:  # VS_MOVE or unknown -- derive from velocity
			var vel: Vector3 = (
				(ctrl.global_position - prev_pos) / delta if delta > 0 else Vector3.ZERO
			)
			var speed: float = Vector2(vel.x, vel.z).length()
			if speed > 0.5:
				ctrl.character_model.travel("run", clampf(speed / ctrl.sprint_speed, 0.5, 1.5))
			else:
				ctrl.character_model.travel("idle")


func _drive_remote_vfx(old_vs: int, new_vs: int) -> void:
	# Stop effects from previous state
	var attack_states := [
		NetSerializer.VS_VG_LIGHT_1,
		NetSerializer.VS_VG_LIGHT_2,
		NetSerializer.VS_VG_LIGHT_3,
		NetSerializer.VS_VG_HEAVY_WINDUP,
		NetSerializer.VS_VG_HEAVY,
	]
	if old_vs in attack_states and new_vs not in attack_states:
		ctrl.vfx.stop_swing_trail()
	if old_vs == NetSerializer.VS_VG_BLOCK and new_vs != NetSerializer.VS_VG_BLOCK:
		if ctrl.spec_id == "shield":
			ctrl.vfx.hide_tower_shield()
		else:
			ctrl.vfx.hide_block_shield()
	if old_vs == NetSerializer.VS_VG_VORTEX and new_vs != NetSerializer.VS_VG_VORTEX:
		ctrl.vfx.stop_vortex()

	# Start effects for new state
	if new_vs in attack_states and old_vs not in attack_states:
		ctrl.vfx.start_swing_trail()
	if new_vs == NetSerializer.VS_VG_BLOCK and old_vs != NetSerializer.VS_VG_BLOCK:
		if ctrl.spec_id == "shield":
			ctrl.vfx.show_tower_shield()
		else:
			ctrl.vfx.show_block_shield()
	if new_vs == NetSerializer.VS_VG_VORTEX and old_vs != NetSerializer.VS_VG_VORTEX:
		ctrl.vfx.start_vortex()
	if new_vs == NetSerializer.VS_VG_EXECUTION and old_vs == NetSerializer.VS_VG_EXECUTION_WINDUP:
		if ctrl.spec_id == "shield":
			ctrl.vfx.spawn_retaliate_slam(ctrl.global_position, ctrl.rotation.y)
		else:
			ctrl.vfx.spawn_execution_shockwave(ctrl.global_position, ctrl.rotation.y)
