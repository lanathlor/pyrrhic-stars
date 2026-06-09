extends Node

## Harmonist combat: ability committing, channeling, dodge, and ally targeting.
## Sends ability inputs to the server; server resolves healing.

var ctrl: Node

var _dodge_direction: Vector3 = Vector3.ZERO
var _sustaining: bool = false
var _sustain_elapsed: float = 0.0
var _sustain_slot: int = -1


func _ready() -> void:
	ctrl = get_parent()


# --- Ability Input ---


## Check for ability/dodge input and start committing if available.
## Returns true if an ability or dodge was started (caller should return).
func _check_ability_input() -> bool:
	if ctrl._gcd_timer > 0.0:
		return false
	for slot in 5:
		if Input.is_action_just_pressed(ctrl.ABILITY_SLOT_ACTIONS[slot]):
			if ctrl._cooldowns[slot] <= 0.0:
				start_ability(slot)
				return true
	# Slot 5 (C key): commit if ability equipped, no dodge fallback for harmonist
	if Input.is_action_just_pressed("dodge") and ctrl.is_on_floor():
		if ctrl._cooldowns[5] <= 0.0:
			var ability := _resolve_ability(5)
			if not ability.is_empty():
				start_ability(5)
				return true
		return false
	return false


# --- Ability Committing ---


func start_ability(slot: int) -> void:
	if slot < 0 or slot >= 6:
		return

	var ability := _resolve_ability(slot)
	if ability.is_empty():
		return

	# During sustain, new ability input only cancels -- does not fire the new ability
	if _sustaining:
		cancel_sustain()
		if NetworkManager.is_active:
			NetworkManager.send_ability(255, 0.0, ctrl.rotation.y)
		return

	_commit_ability(slot, ability)


func _commit_ability(slot: int, ability: Dictionary) -> void:
	var action_id: int = 50 + slot

	# Start cooldown for this slot (sustain abilities defer cooldown to cancel)
	var cd_max: float = ability.get("cooldown_max", ability.get("cooldown", 0.0))
	if cd_max > 0.0 and not _ability_has_sustain(ability):
		ctrl._cooldowns[slot] = cd_max

	# Track slot for deferred sustain cooldown
	if _ability_has_sustain(ability):
		_sustain_slot = slot

	ctrl._committing_ability = ability
	ctrl._cast_timer = ability.get("dur", ability.get("commit_time", 0.3))
	ctrl._gcd_timer = ctrl.gcd_duration

	# Send ability to server with action_id, including target if available
	if NetworkManager.is_active:
		var target_pid: int = _resolve_target_peer()
		if target_pid > 0:
			NetworkManager.send_ability_targeted(action_id, 0.0, ctrl.rotation.y, target_pid)
		else:
			NetworkManager.send_ability(action_id, 0.0, ctrl.rotation.y)

	var delivery: String = ability.get("delivery", "")

	# Displacement abilities use dodge movement instead of commit state
	if delivery == "displacement":
		_start_gust_step()
		return

	# Determine state: short abilities are instant commits, longer ones are channels
	if ctrl._cast_timer > 0.5:
		ctrl._enter_state(ctrl.State.CHANNELING)
	else:
		ctrl._enter_state(ctrl.State.CASTING)

	_trigger_delivery_vfx(delivery, ability)


func _trigger_delivery_vfx(delivery: String, ability: Dictionary) -> void:
	if not ctrl.vfx:
		return
	match delivery:
		"direct":
			ctrl.vfx.spawn_cast_flash()
		"beam":
			var target := _resolve_selected_target_node()
			if target and target != ctrl:
				ctrl.vfx.start_heal_beam(target)
			ctrl.vfx.start_channel_flux()
		"zone":
			var radius: float = ability.get("zone_radius", 6.0)
			ctrl.vfx.start_zone_telegraph(ctrl.global_position, radius)
			ctrl.vfx.start_channel_flux()


## Resolve ability data for a slot from AbilityCatalog. No committing without server catalog.
func _resolve_ability(slot: int) -> Dictionary:
	if AbilityCatalog.catalog.size() > 0:
		if slot < AbilityCatalog.current_loadout.size():
			var ability_id: String = AbilityCatalog.current_loadout[slot]
			if ability_id != "":
				var entry: Dictionary = AbilityCatalog.get_ability(ability_id)
				if not entry.is_empty():
					return entry
	return {}


func process_casting(delta: float) -> void:
	ctrl.movement.face_attack_direction(delta)

	# Slow movement while committing
	var wish_dir: Vector3 = ctrl.movement.get_camera_wish_dir()
	var speed: float = ctrl.run_speed * ctrl.commit_move_speed_mult
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
		ctrl._committing_ability = {}
		ctrl._enter_state(ctrl.State.MOVE)


func process_channeling(delta: float) -> void:
	# Allow breaking a channel by committing a new ability or dodging
	if not ctrl.hud.is_codex_open() and _check_ability_input():
		return

	ctrl.movement.face_attack_direction(delta)

	# Channeling is slower than committing -- nearly stationary
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
	if ctrl.vfx and not ctrl._committing_ability.is_empty():
		var total_dur: float = ctrl._committing_ability.get(
			"dur", ctrl._committing_ability.get("commit_time", 1.0)
		)
		var progress: float = clampf(
			(total_dur - ctrl._cast_timer) / maxf(total_dur, 0.01), 0.0, 1.0
		)
		ctrl.vfx.update_channel_flux(progress)

	ctrl._cast_timer -= delta
	if ctrl._cast_timer <= 0.0:
		if not _sustaining and _ability_has_sustain(ctrl._committing_ability):
			# Channel complete — enter sustain phase (infinite hold)
			_sustaining = true
			_sustain_elapsed = 0.0
			ctrl._cast_timer = 0.0
		elif _sustaining:
			# In sustain — keep ticking (server drives effects and cancel)
			_sustain_elapsed += delta
			if ctrl.vfx:
				ctrl.vfx.update_channel_flux(clampf(0.8 + _sustain_elapsed * 0.02, 0.8, 1.0))
		else:
			_stop_active_vfx()
			ctrl._committing_ability = {}
			ctrl._enter_state(ctrl.State.MOVE)


func _ability_has_sustain(ability: Dictionary) -> bool:
	return ability.get("sustain", false)


## Cancel a commit (channel/cast) before it fires. No cooldown penalty.
func cancel_commit() -> void:
	_stop_active_vfx()
	ctrl._committing_ability = {}
	ctrl._cast_timer = 0.0
	ctrl._enter_state(ctrl.State.MOVE)


## Cancel sustain and return to MOVE. Called by ESC, movement, or new input.
func cancel_sustain() -> void:
	if not _sustaining:
		return
	_apply_sustain_cooldown()
	_sustaining = false
	_sustain_elapsed = 0.0
	_sustain_slot = -1
	_stop_active_vfx()
	ctrl._committing_ability = {}
	ctrl._enter_state(ctrl.State.MOVE)


func _apply_sustain_cooldown() -> void:
	if ctrl._committing_ability.is_empty():
		return
	var slot: int = _sustain_slot
	if slot < 0:
		# Look up slot from ability name in server loadout
		var ability_name: String = ctrl._committing_ability.get("name", "")
		for i in AbilityCatalog.current_loadout.size():
			var entry: Dictionary = AbilityCatalog.get_ability(AbilityCatalog.current_loadout[i])
			if entry.get("name", "") == ability_name:
				slot = i
				break
	if slot < 0 or slot >= ctrl._cooldowns.size():
		return
	var cd_max: float = ctrl._committing_ability.get(
		"cooldown_max", ctrl._committing_ability.get("cooldown", 0.0)
	)
	if cd_max > 0.0:
		ctrl._cooldowns[slot] = cd_max


# --- Dodge ---


## Gust Step: dodge-like displacement triggered as an ability
## (cooldown/GCD/network already handled).
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
	ctrl._committing_ability = {}

	if ctrl.vfx:
		ctrl.vfx.spawn_gust_trail()


func start_dodge() -> void:
	if _sustaining:
		_apply_sustain_cooldown()
		_sustaining = false
		_sustain_elapsed = 0.0
		_sustain_slot = -1
		_stop_active_vfx()
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
## If an enemy is selected, heals self-targeted (WoW healer behavior).
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
## If an enemy is selected, returns self (heals self-targeted).
func _resolve_selected_target_node() -> Node3D:
	if ctrl._selected_target and is_instance_valid(ctrl._selected_target):
		if _is_enemy(ctrl._selected_target):
			return ctrl
		return ctrl._selected_target
	return null
