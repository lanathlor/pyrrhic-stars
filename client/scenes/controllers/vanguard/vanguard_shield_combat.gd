extends Node

## Vanguard Shield combat: shield bash, bull rush, shield block, brace, retaliate, guard break.

const PlayerTelegraph := preload("res://scenes/shared/telegraph/player_telegraph.gd")
const SHIELD_TELEGRAPH_COLOR := Color(0.3, 0.6, 0.9, 0.4)

var ctrl: Node

var _has_hit_this_attack: bool = false
var _dodge_direction: Vector3 = Vector3.ZERO
var _bull_rush_direction: Vector3 = Vector3.ZERO


func _ready() -> void:
	ctrl = get_parent()


# --- Dodge (same as Blade) ---


func start_dodge() -> void:
	var wish: Vector3 = ctrl.movement.get_camera_wish_dir()
	if wish.length() > 0.1:
		if ctrl._lock_on_active and ctrl._lock_target and is_instance_valid(ctrl._lock_target):
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


# --- Shield Bash (LMB) — quick bash that works during block ---


func start_shield_bash() -> void:
	_has_hit_this_attack = false
	if NetworkManager.is_active:
		NetworkManager.send_ability(1, 0.0, ctrl.rotation.y)
	ctrl._enter_state(ctrl.State.SHIELD_BASH)
	ctrl._state_timer = ctrl.SHIELD_BASH_DURATION


func process_shield_bash(delta: float) -> void:
	ctrl.movement.face_attack_direction(delta)
	ctrl.movement.apply_attack_movement(delta)

	if not _has_hit_this_attack and ctrl._state_timer <= ctrl.SHIELD_BASH_DURATION * 0.5:
		_has_hit_this_attack = true
		perform_melee_hit(ctrl.SHIELD_BASH_DAMAGE)

	if ctrl._state_timer <= 0.0:
		# Return to block if RMB still held, otherwise move
		if Input.is_action_pressed("block") and ctrl.stamina > 0.0:
			ctrl._enter_state(ctrl.State.SHIELD_BLOCK)
		else:
			ctrl._enter_state(ctrl.State.MOVE)


# --- Bull Rush (R) — charge forward, AoE at end ---


func start_bull_rush() -> void:
	_has_hit_this_attack = false
	ctrl._bull_rush_cooldown = ctrl.BULL_RUSH_COOLDOWN
	# Drop guard if blocking
	if ctrl.state == ctrl.State.SHIELD_BLOCK:
		_end_shield_block_visual()

	var forward: Vector3 = -ctrl.transform.basis.z
	forward.y = 0.0
	_bull_rush_direction = forward.normalized()

	if NetworkManager.is_active:
		NetworkManager.send_ability(2, 0.0, ctrl.rotation.y)
	ctrl._enter_state(ctrl.State.BULL_RUSH)
	ctrl._state_timer = ctrl.BULL_RUSH_DURATION

	# Telegraph: circle at destination
	var end_pos: Vector3 = ctrl.global_position + _bull_rush_direction * ctrl.BULL_RUSH_DISTANCE
	PlayerTelegraph.spawn_circle(
		ctrl.get_tree().current_scene, end_pos, 5.0, SHIELD_TELEGRAPH_COLOR
	)


func process_bull_rush(delta: float) -> void:
	ctrl.velocity.x = _bull_rush_direction.x * ctrl.BULL_RUSH_SPEED
	ctrl.velocity.z = _bull_rush_direction.z * ctrl.BULL_RUSH_SPEED

	if not _has_hit_this_attack and ctrl._state_timer <= ctrl.BULL_RUSH_DURATION * 0.3:
		_has_hit_this_attack = true
		perform_melee_hit(ctrl.BULL_RUSH_DAMAGE)

	if ctrl._state_timer <= 0.0:
		ctrl.velocity.x *= 0.2
		ctrl.velocity.z *= 0.2
		ctrl._enter_state(ctrl.State.MOVE)


# --- Shield Block (RMB) — sustained block with guard parry window ---


func start_shield_block() -> void:
	ctrl._enter_state(ctrl.State.SHIELD_BLOCK)
	ctrl._parry_timer = ctrl.SHIELD_PARRY_WINDOW
	ctrl.vfx.show_tower_shield()
	if NetworkManager.is_active:
		NetworkManager.send_ability(4, 0.0, ctrl.rotation.y)


func process_shield_block(delta: float) -> void:
	# Move at server-authoritative reduced speed while blocking
	var wish_dir: Vector3 = ctrl.movement.get_camera_wish_dir()
	var block_speed: float = ctrl.run_speed * ctrl._server_speed_mult
	if wish_dir.length() > 0.1:
		var target_vel: Vector3 = wish_dir * block_speed
		ctrl.velocity.x = move_toward(ctrl.velocity.x, target_vel.x, ctrl.ground_accel * delta)
		ctrl.velocity.z = move_toward(ctrl.velocity.z, target_vel.z, ctrl.ground_accel * delta)
	else:
		ctrl.velocity.x = move_toward(ctrl.velocity.x, 0.0, ctrl.ground_decel * delta)
		ctrl.velocity.z = move_toward(ctrl.velocity.z, 0.0, ctrl.ground_decel * delta)
	ctrl.movement.face_attack_direction(delta)

	# Shield Bash during block (LMB)
	var cursor_active := Input.get_mouse_mode() != Input.MOUSE_MODE_CAPTURED
	if (
		not cursor_active
		and Input.is_action_just_pressed("light_attack")
		and ctrl.stamina >= ctrl.SHIELD_BASH_STAMINA
	):
		start_shield_bash()
		return

	# Brace during block (F)
	if (
		not cursor_active
		and Input.is_action_just_pressed("ability_1")
		and ctrl._brace_cooldown <= 0.0
	):
		start_brace()
		return

	# No client-side stamina drain — server handles it proportionally to damage
	if not Input.is_action_pressed("block") or ctrl.stamina <= 0.0:
		_end_shield_block_visual()
		if NetworkManager.is_active:
			NetworkManager.send_ability(5, 0.0, ctrl.rotation.y)
		ctrl._shield_block_cooldown = ctrl.SHIELD_BLOCK_COOLDOWN
		ctrl._enter_state(ctrl.State.MOVE)


func _end_shield_block_visual() -> void:
	ctrl.vfx.hide_tower_shield()


# --- Brace (F) — plant feet during block, reduce stamina drain ---


func start_brace() -> void:
	ctrl._brace_cooldown = ctrl.BRACE_COOLDOWN
	if NetworkManager.is_active:
		NetworkManager.send_ability(20, 0.0, ctrl.rotation.y)
	ctrl._enter_state(ctrl.State.BRACE)
	ctrl._state_timer = ctrl.BRACE_DURATION


func process_brace(delta: float) -> void:
	ctrl.velocity.x = 0.0
	ctrl.velocity.z = 0.0
	ctrl.movement.face_attack_direction(delta)

	if ctrl._state_timer <= 0.0:
		# Return to block if RMB still held, otherwise move
		if Input.is_action_pressed("block") and ctrl.stamina > 0.0:
			ctrl._enter_state(ctrl.State.SHIELD_BLOCK)
		else:
			_end_shield_block_visual()
			ctrl._enter_state(ctrl.State.MOVE)


# --- Retaliate (T) — consume Devotion, massive frontal slam ---


func start_retaliate() -> void:
	_has_hit_this_attack = false
	ctrl._retaliate_cooldown = ctrl.RETALIATE_COOLDOWN

	# Drop guard if blocking
	if ctrl.state == ctrl.State.SHIELD_BLOCK or ctrl.state == ctrl.State.BRACE:
		_end_shield_block_visual()

	if NetworkManager.is_active:
		NetworkManager.send_ability(21, 0.0, ctrl.rotation.y)
	ctrl._enter_state(ctrl.State.RETALIATE_WINDUP)
	ctrl._state_timer = ctrl.RETALIATE_WINDUP_TIME


func process_retaliate_windup(delta: float) -> void:
	ctrl.velocity.x = 0.0
	ctrl.velocity.z = 0.0
	ctrl.movement.face_attack_direction(delta)
	if ctrl._state_timer <= 0.0:
		ctrl._enter_state(ctrl.State.RETALIATE)
		ctrl._state_timer = ctrl.RETALIATE_HIT_TIME
		ctrl.vfx.spawn_retaliate_slam(ctrl.global_position, ctrl.rotation.y)
		# Wide 90° half-angle cone (180° total)
		PlayerTelegraph.spawn_cone(
			ctrl.get_tree().current_scene,
			ctrl.global_position,
			ctrl.rotation.y,
			6.0,
			90.0,
			SHIELD_TELEGRAPH_COLOR
		)


func process_retaliate(delta: float) -> void:
	ctrl.velocity.x = 0.0
	ctrl.velocity.z = 0.0
	if not _has_hit_this_attack and ctrl._state_timer <= ctrl.RETALIATE_HIT_TIME * 0.5:
		_has_hit_this_attack = true
		perform_melee_hit(ctrl.RETALIATE_DAMAGE)
	if ctrl._state_timer <= 0.0:
		ctrl._enter_state(ctrl.State.MOVE)


# --- Guard Break (server-driven stagger) ---


func process_guard_break() -> void:
	ctrl.velocity.x = 0.0
	ctrl.velocity.z = 0.0
	if ctrl._state_timer <= 0.0:
		ctrl._enter_state(ctrl.State.MOVE)


# --- Stagger ---


func process_stagger() -> void:
	ctrl.velocity.x = 0.0
	ctrl.velocity.z = 0.0
	if ctrl._state_timer <= 0.0:
		ctrl._enter_state(ctrl.State.MOVE)


# --- Melee Hit Detection (visual-only, server confirms damage) ---


func perform_melee_hit(_damage: float) -> void:
	var forward: Vector3 = -ctrl.transform.basis.z
	forward.y = 0.0
	forward = forward.normalized()

	for enemy in GameManager.enemies:
		if not is_instance_valid(enemy):
			continue
		var to_enemy: Vector3 = enemy.global_position - ctrl.global_position
		to_enemy.y = 0.0
		var dist: float = to_enemy.length()
		if dist > ctrl.melee_range:
			continue
		if dist < 0.01:
			continue
		var angle: float = rad_to_deg(forward.angle_to(to_enemy.normalized()))
		if angle <= ctrl.melee_arc_degrees / 2.0:
			pass  # Hit marker driven by server-confirmed damage events
