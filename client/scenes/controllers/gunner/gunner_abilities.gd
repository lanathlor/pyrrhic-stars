extends Node

## Gunner abilities: overclock, rechamber, dodge roll, reload, load enhanced, mag dump.

const TACTICAL_RELOAD_TIME: float = 1.5
const EMPTY_RELOAD_TIME: float = 2.2

var ctrl: Node


func _ready() -> void:
	ctrl = get_parent()


# --- Dodge Roll ---


func handle_dodge() -> void:
	if (
		Input.is_action_just_pressed("dodge")
		and ctrl._roll_cooldown_timer <= 0.0
		and ctrl.is_on_floor()
	):
		start_roll()


func start_roll() -> void:
	var input_dir := Input.get_vector("move_left", "move_right", "move_forward", "move_backward")
	if input_dir.length() > 0.1:
		ctrl._roll_direction = (
			(ctrl.transform.basis * Vector3(input_dir.x, 0.0, input_dir.y)).normalized()
		)
	else:
		# Default: roll backward (away from where you're looking)
		ctrl._roll_direction = (ctrl.transform.basis * Vector3(0.0, 0.0, 1.0)).normalized()
	ctrl._is_rolling = true
	ctrl._roll_timer = ctrl.roll_duration
	ctrl._roll_cooldown_timer = ctrl.roll_cooldown


func process_roll(delta: float) -> void:
	ctrl._roll_timer -= delta
	var roll_dir: Vector3 = ctrl._roll_direction
	ctrl.velocity.x = roll_dir.x * ctrl.roll_speed
	ctrl.velocity.z = roll_dir.z * ctrl.roll_speed
	if ctrl._roll_timer <= 0.0:
		ctrl._is_rolling = false
		# Bleed off some speed coming out of roll
		ctrl.velocity.x *= 0.4
		ctrl.velocity.z *= 0.4


# --- Overclock ---


func handle_overclock(delta: float) -> void:
	# Tick timers
	if ctrl._overclock_active:
		ctrl._overclock_timer -= delta
		if ctrl._overclock_timer <= 0.0:
			ctrl._overclock_active = false
			ctrl._overclock_timer = 0.0
	if ctrl._overclock_cooldown > 0.0:
		ctrl._overclock_cooldown -= delta
	# Activation
	if (
		Input.is_action_just_pressed("ability_1")
		and not ctrl._overclock_active
		and ctrl._overclock_cooldown <= 0.0
	):
		ctrl._overclock_active = true
		ctrl._overclock_timer = ctrl.OVERCLOCK_DURATION
		ctrl._overclock_cooldown = ctrl.OVERCLOCK_COOLDOWN
		if NetworkManager.is_active:
			NetworkManager.send_ability(10, ctrl.head.rotation.x, ctrl.rotation.y)


# --- Rechamber ---


func get_rechamber_status() -> String:
	match ctrl._rechamber_phase:
		1:
			return "..."
		2:
			return "FIRE!"
	return ""


func handle_rechamber(delta: float) -> void:
	# Tick rechamber buff
	if ctrl._rechamber_buff:
		ctrl._rechamber_buff_timer -= delta
		if ctrl._rechamber_buff_timer <= 0.0:
			ctrl._rechamber_buff = false
			ctrl._rechamber_buff_timer = 0.0
	# Tick rechamber phases
	match ctrl._rechamber_phase:
		1:  # Windup
			ctrl._rechamber_timer -= delta
			if ctrl._rechamber_timer <= 0.0:
				ctrl._rechamber_phase = 2
				ctrl._rechamber_timer = ctrl.RECHAMBER_WINDOW
		2:  # Timing window — handled in weapon.handle_shooting for confirm
			ctrl._rechamber_timer -= delta
			if ctrl._rechamber_timer <= 0.0:
				ctrl._rechamber_phase = 3
				ctrl._rechamber_timer = ctrl.RECHAMBER_LOCKOUT
		3:  # Lockout
			ctrl._rechamber_timer -= delta
			if ctrl._rechamber_timer <= 0.0:
				ctrl._rechamber_phase = 0
	# Activation — only when idle and not shooting
	if (
		Input.is_action_just_pressed("ability_2")
		and ctrl._rechamber_phase == 0
		and ctrl._fire_cooldown <= 0.0
	):
		ctrl._rechamber_phase = 1
		ctrl._rechamber_timer = ctrl.RECHAMBER_WINDUP
		ctrl._fire_cooldown = ctrl.RECHAMBER_WINDUP  # lock out shooting during windup
		if NetworkManager.is_active:
			NetworkManager.send_ability(11, ctrl.head.rotation.x, ctrl.rotation.y)


# --- Reload ---


func handle_reload() -> void:
	if not Input.is_action_just_pressed("reload"):
		return
	if ctrl._reloading or ctrl._mag_dump_active:
		return
	if ctrl._magazine >= ctrl._mag_max:
		return
	_start_reload()


func _start_reload() -> void:
	var raw_dur: float = EMPTY_RELOAD_TIME if ctrl._magazine <= 0 else TACTICAL_RELOAD_TIME
	ctrl._reloading = true
	ctrl._reload_timer = raw_dur
	ctrl._reload_total = raw_dur
	ctrl._reload_server_acked = false
	# _fire_cooldown ticks without Tempo scaling, so pre-scale it to match
	# the Tempo-scaled reload timer drain rate.
	var tempo_mult: float = 1.0 + InventoryManager.get_stat("tempo") / 100.0
	ctrl._fire_cooldown = raw_dur / tempo_mult
	if NetworkManager.is_active:
		NetworkManager.send_ability(13, ctrl.head.rotation.x, ctrl.rotation.y)


# --- Load Enhanced ---


func handle_load_enhanced() -> void:
	if not Input.is_action_just_pressed("load_enhanced"):
		return
	if ctrl._munitions <= 0.0 or ctrl._enhanced_loaded > 0:
		return
	# Client prediction: move reserve to loaded
	ctrl._enhanced_loaded = int(ctrl._munitions)
	ctrl._munitions = 0.0
	if NetworkManager.is_active:
		NetworkManager.send_ability(14, ctrl.head.rotation.x, ctrl.rotation.y)


# --- Mag Dump ---


func handle_mag_dump() -> void:
	if not Input.is_action_just_pressed("mag_dump"):
		return
	if ctrl._reloading or ctrl._mag_dump_active:
		return
	if ctrl._magazine <= 0 and ctrl._enhanced_loaded <= 0:
		return
	if ctrl._mag_dump_cooldown > 0.0:
		return
	# Client prediction
	ctrl._mag_dump_active = true
	ctrl._mag_dump_cooldown = 12.0
	if NetworkManager.is_active:
		NetworkManager.send_ability(15, ctrl.head.rotation.x, ctrl.rotation.y)
