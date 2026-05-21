extends Node

## Harmonist combat: spell casting, channeling, dodge, and ally targeting.
## Sends ability inputs to the server; server resolves healing.

var ctrl: Node

var _dodge_direction: Vector3 = Vector3.ZERO


func _ready() -> void:
	ctrl = get_parent()


# --- Spell Casting ---


func start_spell(slot: int) -> void:
	if slot < 0 or slot >= ctrl.HARMONIST_SPELLS.size():
		return

	var spell: Dictionary = ctrl.HARMONIST_SPELLS[slot]

	# Start cooldown for this slot
	if spell.cooldown_max > 0.0:
		ctrl._cooldowns[slot] = spell.cooldown_max

	ctrl._casting_spell = spell
	ctrl._cast_timer = spell.dur
	ctrl._gcd_timer = ctrl.gcd_duration

	# Send ability to server with action_id, including target if available
	if NetworkManager.is_active:
		var target_pid: int = _resolve_target_peer()
		if target_pid > 0:
			NetworkManager.send_ability_targeted(
				spell.action_id, 0.0, ctrl.rotation.y, target_pid
			)
		else:
			NetworkManager.send_ability(spell.action_id, 0.0, ctrl.rotation.y)

	# Displacement spells use dodge movement instead of cast state
	if spell.get("delivery", "") == "displacement":
		_start_gust_step()
		return

	# Determine state: short spells are instant casts, longer ones are channels
	if spell.dur > 0.5:
		ctrl._enter_state(ctrl.State.CHANNELING)
	else:
		ctrl._enter_state(ctrl.State.CASTING)

	# Trigger VFX for specific spells
	if ctrl.vfx:
		match slot:
			0:  # Mending Surge — cast flash (heal pulse on server response)
				ctrl.vfx.spawn_cast_flash()
			1:  # Mending Beam — beam tether to target (skip beam on self-cast)
				var target := _resolve_selected_target_node()
				if target and target != ctrl:
					ctrl.vfx.start_heal_beam(target)
				ctrl.vfx.start_channel_flux()
			2:  # Life Swap — cast flash
				ctrl.vfx.spawn_cast_flash()
			3:  # Transfusion — zone telegraph + channel flux
				ctrl.vfx.start_zone_telegraph(ctrl.global_position, 6.0)
				ctrl.vfx.start_channel_flux()
			4:  # Frost Ward — shield on target + cast flash
				ctrl.vfx.spawn_cast_flash()
				var target := _resolve_selected_target_node()
				if target:
					ctrl.vfx.spawn_frost_ward(target)


func process_casting(delta: float) -> void:
	ctrl.movement.face_attack_direction(delta)

	# Slow movement while casting
	var wish_dir: Vector3 = ctrl.movement.get_camera_wish_dir()
	var speed: float = ctrl.run_speed * ctrl.cast_move_speed_mult
	if wish_dir.length() > 0.1:
		var target_vel: Vector3 = wish_dir * speed
		ctrl.velocity.x = move_toward(ctrl.velocity.x, target_vel.x, ctrl.ground_accel * delta)
		ctrl.velocity.z = move_toward(ctrl.velocity.z, target_vel.z, ctrl.ground_accel * delta)
	else:
		ctrl.velocity.x = move_toward(ctrl.velocity.x, 0.0, ctrl.ground_decel * delta)
		ctrl.velocity.z = move_toward(ctrl.velocity.z, 0.0, ctrl.ground_decel * delta)

	ctrl._cast_timer -= delta
	if ctrl._cast_timer <= 0.0:
		_stop_active_vfx()
		ctrl._casting_spell = {}
		ctrl._enter_state(ctrl.State.MOVE)


func process_channeling(delta: float) -> void:
	ctrl.movement.face_attack_direction(delta)

	# Channeling is slower than casting -- nearly stationary
	var wish_dir: Vector3 = ctrl.movement.get_camera_wish_dir()
	var speed: float = ctrl.run_speed * 0.15
	if wish_dir.length() > 0.1:
		var target_vel: Vector3 = wish_dir * speed
		ctrl.velocity.x = move_toward(ctrl.velocity.x, target_vel.x, ctrl.ground_accel * delta)
		ctrl.velocity.z = move_toward(ctrl.velocity.z, target_vel.z, ctrl.ground_accel * delta)
	else:
		ctrl.velocity.x = move_toward(ctrl.velocity.x, 0.0, ctrl.ground_decel * delta)
		ctrl.velocity.z = move_toward(ctrl.velocity.z, 0.0, ctrl.ground_decel * delta)

	# Update channel flux VFX intensity
	if ctrl.vfx and not ctrl._casting_spell.is_empty():
		var total_dur: float = ctrl._casting_spell.get("dur", 1.0)
		var progress: float = clampf(
			(total_dur - ctrl._cast_timer) / maxf(total_dur, 0.01), 0.0, 1.0
		)
		ctrl.vfx.update_channel_flux(progress)

	ctrl._cast_timer -= delta
	if ctrl._cast_timer <= 0.0:
		_stop_active_vfx()
		ctrl._casting_spell = {}
		ctrl._enter_state(ctrl.State.MOVE)


# --- Dodge ---


## Gust Step: dodge-like displacement triggered as a spell (cooldown/GCD/network already handled).
func _start_gust_step() -> void:
	var wish: Vector3 = ctrl.movement.get_camera_wish_dir()
	if wish.length() > 0.1:
		if ctrl._selected_target and is_instance_valid(ctrl._selected_target):
			var input_dir := Input.get_vector(
				"move_left", "move_right", "move_forward", "move_backward"
			)
			_dodge_direction = (
				(ctrl.transform.basis * Vector3(input_dir.x, 0.0, input_dir.y)).normalized()
			)
		else:
			_dodge_direction = wish
	else:
		_dodge_direction = (ctrl.transform.basis * Vector3(0.0, 0.0, 1.0)).normalized()

	ctrl._enter_state(ctrl.State.DODGE)
	ctrl._state_timer = ctrl.dodge_duration
	ctrl._is_invincible = true
	ctrl._casting_spell = {}

	if ctrl.vfx:
		ctrl.vfx.spawn_gust_trail()


func start_dodge() -> void:
	var wish: Vector3 = ctrl.movement.get_camera_wish_dir()
	if wish.length() > 0.1:
		if ctrl._selected_target and is_instance_valid(ctrl._selected_target):
			var input_dir := Input.get_vector(
				"move_left", "move_right", "move_forward", "move_backward"
			)
			_dodge_direction = (
				(ctrl.transform.basis * Vector3(input_dir.x, 0.0, input_dir.y)).normalized()
			)
		else:
			_dodge_direction = wish
	else:
		_dodge_direction = (ctrl.transform.basis * Vector3(0.0, 0.0, 1.0)).normalized()

	ctrl._enter_state(ctrl.State.DODGE)
	ctrl._state_timer = ctrl.dodge_duration
	ctrl._is_invincible = true

	# Gust Step wind trail
	if ctrl.vfx:
		ctrl.vfx.spawn_gust_trail()

	if NetworkManager.is_active:
		NetworkManager.send_ability(3, 0.0, ctrl.rotation.y)


func process_dodge(_delta: float) -> void:
	ctrl.velocity.x = _dodge_direction.x * ctrl.dodge_speed
	ctrl.velocity.z = _dodge_direction.z * ctrl.dodge_speed

	var elapsed: float = ctrl.dodge_duration - ctrl._state_timer
	if elapsed >= ctrl.dodge_iframe_duration:
		ctrl._is_invincible = false

	if ctrl._state_timer <= 0.0:
		ctrl._is_invincible = false
		ctrl.velocity.x *= 0.3
		ctrl.velocity.z *= 0.3
		ctrl._enter_state(ctrl.State.MOVE)


# --- Stagger ---


func process_stagger() -> void:
	ctrl.velocity.x = 0.0
	ctrl.velocity.z = 0.0
	if ctrl._state_timer <= 0.0:
		ctrl._enter_state(ctrl.State.MOVE)


# --- VFX helpers ---


## Stop all active ability VFX (beam, zone, channel flux).
func _stop_active_vfx() -> void:
	if ctrl.vfx:
		ctrl.vfx.stop_channel_flux()
		ctrl.vfx.stop_heal_beam()
		ctrl.vfx.stop_zone_telegraph()


# --- Target resolution ---


## Returns the peer_id of the current heal target, or -1 if none.
## Priority: click-selected target > HUD mouseover target.
## If an enemy is selected, heals self-cast (WoW healer behavior).
func _resolve_target_peer() -> int:
	# 1. Click-selected target
	if ctrl._selected_target and is_instance_valid(ctrl._selected_target):
		if "peer_id" in ctrl._selected_target:
			# Enemy targeted — heals go to self
			if _is_enemy(ctrl._selected_target):
				return ctrl.peer_id
			return ctrl._selected_target.peer_id

	# 2. HUD party-frame mouseover
	if ctrl.hud and ctrl.hud.has_method("get_mouseover_target"):
		var hovered: int = ctrl.hud.get_mouseover_target()
		if hovered > 0:
			return hovered

	return -1


func _is_enemy(node: Node3D) -> bool:
	for enemy in GameManager.enemies:
		if enemy == node:
			return true
	return false


## Returns the effective heal target Node3D, or null if none.
## If an enemy is selected, returns self (heals self-cast).
func _resolve_selected_target_node() -> Node3D:
	if ctrl._selected_target and is_instance_valid(ctrl._selected_target):
		if _is_enemy(ctrl._selected_target):
			return ctrl
		return ctrl._selected_target
	return null
