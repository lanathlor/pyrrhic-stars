extends CharacterBody3D

## Vanguard — Souls-like third-person melee controller.
## Blade spec: Onslaught momentum — build stacks on hits, lose on damage.
## Shield spec: Devotion mastery — build charges from blocked damage, burst with Retaliate.
## Sub-systems: combat, movement, cam, anim (child nodes).

signal died

enum State {
	MOVE,
	DODGE,
	# Blade states
	CLEAVE,
	UPHEAVAL_WINDUP,
	UPHEAVAL,
	BLOCK,
	STAGGER,
	DEAD,
	VORTEX,
	EXECUTION_WINDUP,
	EXECUTION,
	# Shield states
	SHIELD_BLOCK,
	SHIELD_BASH,
	BULL_RUSH,
	BRACE,
	RETALIATE_WINDUP,
	RETALIATE,
	GUARD_BREAK,
}

const PlayerTelegraph := preload("res://scenes/shared/telegraph/player_telegraph.gd")

# Cleave (LMB) — fast repeatable sweep
const CLEAVE_DAMAGE: float = 30.0
const CLEAVE_DURATION: float = 0.45
const CLEAVE_STAMINA: float = 10.0

# Upheaval (R) — cone slam
const UPHEAVAL_DAMAGE: float = 55.0
const UPHEAVAL_WINDUP_TIME: float = 0.3
const UPHEAVAL_HIT_TIME: float = 0.3
const UPHEAVAL_STAMINA: float = 20.0

# Vortex (F) — forward advancing spin
const VORTEX_COOLDOWN: float = 10.0
const VORTEX_STAMINA: float = 25.0
const VORTEX_SPEED: float = 10.0
const VORTEX_DURATIONS: Array[float] = [0.6, 0.8, 1.0]  # by onslaught tier

# Execution (T) — slow windup + devastating overhead chop
const EXECUTION_COOLDOWN: float = 8.0
const EXECUTION_STAMINA: float = 30.0
const EXECUTION_WINDUP_TIME: float = 0.8
const EXECUTION_HIT_TIME: float = 0.15

# Shield Bash (LMB) — quick bash, works during block
const SHIELD_BASH_DAMAGE: float = 15.0
const SHIELD_BASH_DURATION: float = 0.35
const SHIELD_BASH_STAMINA: float = 8.0

# Bull Rush (R) — charge forward + AoE
const BULL_RUSH_DAMAGE: float = 60.0
const BULL_RUSH_SPEED: float = 12.0
const BULL_RUSH_DISTANCE: float = 12.0
const BULL_RUSH_DURATION: float = 0.6
const BULL_RUSH_COOLDOWN: float = 8.0
const BULL_RUSH_STAMINA: float = 20.0

# Shield Block (RMB) — sustained block, stamina drain on damage
const SHIELD_BLOCK_COOLDOWN: float = 1.5
const SHIELD_PARRY_WINDOW: float = 0.12

# Brace (F) — plant feet during block, reduced stamina drain
const BRACE_DURATION: float = 3.5
const BRACE_COOLDOWN: float = 12.0

# Retaliate (T) — consume Devotion, frontal slam
const RETALIATE_DAMAGE: float = 30.0
const RETALIATE_WINDUP_TIME: float = 0.5
const RETALIATE_HIT_TIME: float = 0.2
const RETALIATE_COOLDOWN: float = 1.5

# Telegraph color
const VANGUARD_TELEGRAPH_COLOR := Color(0.9, 0.6, 0.3, 0.4)

# Network
const NET_INTERP_SPEED := 15.0

const WEAPON_SCENE := "res://assets/models/weapons/weapon_longsword.glb"

const CombatScript := preload("res://scenes/controllers/vanguard/vanguard_combat.gd")
const ShieldCombatScript := preload("res://scenes/controllers/vanguard/vanguard_shield_combat.gd")
const MovementScript := preload("res://scenes/controllers/vanguard/vanguard_movement.gd")
const CameraScript := preload("res://scenes/controllers/vanguard/vanguard_camera.gd")
const AnimScript := preload("res://scenes/controllers/vanguard/vanguard_animation.gd")
const VfxScript := preload("res://scenes/controllers/vanguard/vfx/vanguard_vfx.gd")
const RemoteScript := preload("res://scenes/controllers/vanguard/vanguard_remote.gd")
const HudUpdaterScript := preload("res://scenes/controllers/vanguard/vanguard_hud_updater.gd")

# Movement
@export var run_speed: float = 5.0
@export var sprint_speed: float = 7.0
@export var mouse_sensitivity: float = 0.003
@export var ground_accel: float = 20.0
@export var ground_decel: float = 15.0
@export var air_accel: float = 2.0
@export var air_decel: float = 1.0
@export var rotation_speed: float = 10.0

# Dodge
@export var dodge_speed: float = 12.0
@export var dodge_duration: float = 0.4
@export var dodge_iframe_duration: float = 0.15
@export var dodge_stamina_cost: float = 20.0

# Melee hit detection
@export var melee_range: float = 3.0
@export var melee_arc_degrees: float = 120.0
@export var attack_move_speed_mult: float = 0.55

# Block / parry
@export var block_damage_reduction: float = 0.8
@export var parry_window: float = 0.15
@export var block_stamina_drain: float = 15.0

# Stamina
@export var stamina_regen_rate: float = 30.0
@export var stamina_regen_delay: float = 0.6

# Camera
@export var camera_distance: float = 6.0
@export var camera_height_offset: float = 2.0

# Health & Stamina
var health: float = 200.0
var max_health: float = 200.0
var peer_id: int = 0
var stamina: float = 100.0
var max_stamina: float = 100.0

# State
var state: State = State.MOVE

# Sub-systems
var combat: Node
var movement: Node
var cam: Node
var anim: Node
var vfx: Node
var remote: Node
var hud_updater: Node

var spec_id: String = "blade"
var _stuck_recovery := StuckRecovery.new()
var _spec_grace: float = 0.0  # prevents server revert during client-initiated spec change
var _server_speed_mult: float = 1.0  # server-authoritative movement speed multiplier

var _block_cooldown: float = 0.0
var _state_timer: float = 0.0
var _is_invincible: bool = false
var _parry_timer: float = 0.0
var _stagger_duration: float = 0.3

# Onslaught (Blade) / Devotion (Shield) — read from server state
var _onslaught_tier: int = 0
var _onslaught_stacks: int = 0
var _devotion_tier: int = 0
var _devotion_stacks: int = 0

# Ability cooldowns
var _vortex_cooldown: float = 0.0
var _execution_cooldown: float = 0.0
var _bull_rush_cooldown: float = 0.0
var _brace_cooldown: float = 0.0
var _retaliate_cooldown: float = 0.0
var _shield_block_cooldown: float = 0.0

# Camera / Lock-on
var _camera_yaw: float = 0.0
var _camera_pitch: float = -0.3
var _lock_target: Node3D = null
var _lock_on_active: bool = false

var _gravity: float = 8.5
var _alive: bool = true

# Network sync
var _visual_state: int = 0
var _net_position: Vector3 = Vector3.ZERO
var _net_rotation_y: float = 0.0
var _prev_remote_vs: int = -1

@onready var camera: Camera3D = $Camera3D
@onready var character_model: Node3D = $CharacterModel
@onready var hud: Control = $HUDLayer/VanguardHUD


func _ready() -> void:
	# Create sub-systems — spec determines which combat script
	if spec_id == "shield":
		combat = _add_subsystem("Combat", ShieldCombatScript)
	else:
		combat = _add_subsystem("Combat", CombatScript)
	movement = _add_subsystem("Movement", MovementScript)
	cam = _add_subsystem("Cam", CameraScript)
	anim = _add_subsystem("Anim", AnimScript)
	vfx = _add_subsystem("Vfx", VfxScript)
	remote = _add_subsystem("Remote", RemoteScript)
	hud_updater = _add_subsystem("HudUpdater", HudUpdaterScript)

	GameManager.register_player(self)
	_net_position = global_position
	_net_rotation_y = rotation.y
	camera.top_level = true

	# Set up animation state machine
	(
		character_model
		. setup_state_machine(
			{
				"idle": "sword_idle",
				"run": "sword_run",
				"jump": "sword_jump",
				"dodge": "ual_roll",
				"cleave": "sword_slash_1",
				"upheaval": "sword_heavy",
				"block": "sword_block",
				"stagger": "sword_impact",
				"vortex": "sword_spin",
				"execution": "sword_slash_3",
				"dead": "ual_death",
			}
		)
	)

	if _is_local():
		Input.set_mouse_mode(Input.MOUSE_MODE_CAPTURED)
	else:
		$HUDLayer.visible = false
		camera.current = false
		set_process_unhandled_input(false)

	anim.attach_weapon.call_deferred()


func _add_subsystem(node_name: String, script: GDScript) -> Node:
	var node: Node = script.new()
	node.name = node_name
	add_child(node)
	return node


func _exit_tree() -> void:
	GameManager.unregister_player(self)


func _is_local() -> bool:
	if has_meta("replay_puppet"):
		return false
	if not NetworkManager.is_active:
		return true
	return peer_id == NetworkManager.get_my_id()


## Apply authoritative state from the server's WorldState.
func apply_server_state(data: Dictionary) -> void:
	if data.has("max_health") and data["max_health"] > 0.0:
		max_health = data["max_health"]
	# Detect spec from server (same wire bytes carry Onslaught or Devotion).
	# Grace timer prevents stale world state from reverting a client-initiated change.
	var server_spec: String = data.get("spec_name", "")
	if server_spec != "" and server_spec != spec_id and _spec_grace <= 0.0:
		_switch_spec(server_spec)
	if spec_id == "shield":
		_devotion_tier = data.get("onslaught_tier", 0)
		_devotion_stacks = data.get("onslaught_stacks", 0)
	else:
		_onslaught_tier = data.get("onslaught_tier", 0)
		_onslaught_stacks = data.get("onslaught_stacks", 0)
	_server_speed_mult = data.get("speed_mult", 1.0)
	if _is_local():
		health = data.health
		var server_stamina: float = data.get("stamina", -1.0)
		if server_stamina >= 0.0:
			stamina = server_stamina
		if health <= 0.0 and _alive:
			_alive = false
			_enter_state(State.DEAD)
			died.emit()
		elif health > 0.0 and not _alive:
			_alive = true
			_enter_state(State.MOVE)
			global_position = data.pos
			_net_position = data.pos
	else:
		_net_position = data.pos
		_net_rotation_y = data.rot_y
		health = data.health
		_visual_state = data.get("visual_state", 0)


## Hot-swap combat subsystem when spec changes.
## Set from_client=true when the switch is initiated by the user (spec panel),
## which sets a grace timer to prevent stale server state from reverting it.
func _switch_spec(new_spec: String, from_client: bool = false) -> void:
	if new_spec == spec_id:
		return
	spec_id = new_spec
	if from_client:
		_spec_grace = 0.5  # ignore server spec for 500ms
	# Reset to safe state — old combat node's states aren't valid for new spec
	if state != State.MOVE and state != State.DEAD:
		state = State.MOVE
		_state_timer = 0.0
	# Replace combat subsystem
	if combat:
		combat.queue_free()
	if spec_id == "shield":
		combat = _add_subsystem("Combat", ShieldCombatScript)
	else:
		combat = _add_subsystem("Combat", CombatScript)


## Called by main.gd when server confirms this player hit an enemy.
func on_hit_confirmed(_amount: float, hit_pos: Vector3 = Vector3.ZERO) -> void:
	hud.show_hit_marker()
	if hit_pos != Vector3.ZERO:
		vfx.spawn_hit_impact(hit_pos)


## Visual-only damage feedback (called from main.gd on DamageEvent).
func on_damage_visual(_amount: float, _hit_pos: Vector3) -> void:
	if _parry_timer > 0.0 and (state == State.BLOCK or state == State.SHIELD_BLOCK):
		vfx.spawn_parry_flash()
	hud.show_damage_flash()
	anim.show_body_flash()


func _unhandled_input(event: InputEvent) -> void:
	if not _is_local():
		return
	if event is InputEventMouseMotion and Input.get_mouse_mode() == Input.MOUSE_MODE_CAPTURED:
		if not _lock_on_active:
			_camera_yaw -= event.relative.x * mouse_sensitivity
		_camera_pitch -= event.relative.y * mouse_sensitivity
		_camera_pitch = clampf(_camera_pitch, deg_to_rad(-60.0), deg_to_rad(20.0))

	if event.is_action_pressed("lock_on"):
		_toggle_lock_on()


func _physics_process(delta: float) -> void:
	var spawn_frame: int = get_meta("_spawn_frame", -1)
	var age: int = Engine.get_physics_frames() - spawn_frame
	if age <= 30 or Engine.get_physics_frames() % 600 == 0:
		print(
			(
				"[Vanguard] tick#%d peer=%d my_id=%d local=%s current=%s pos=%s cam=%s"
				% [
					age,
					peer_id,
					NetworkManager.get_my_id(),
					_is_local(),
					camera.current,
					global_position,
					camera.global_position
				]
			)
		)
	if not _is_local():
		var prev_pos := global_position
		global_position = global_position.move_toward(_net_position, 12.0 * delta)
		rotation.y = lerp_angle(rotation.y, _net_rotation_y, 8.0 * delta)
		remote.drive_remote_animation(prev_pos, delta)
		return

	if not is_on_floor():
		velocity.y -= _gravity * delta
	else:
		velocity.y = -0.5

	_state_timer -= delta
	anim.update_flash(delta)
	cam.update_camera()
	movement.update_stamina(delta)
	_tick_cooldowns(delta)
	_process_combat_state(delta)

	move_and_slide()
	var commanding := GameManager.move_vector().length() > 0.1
	if _stuck_recovery.apply(self, commanding, delta) and NetworkManager.is_active:
		NetworkManager.send_respawn_request(2)
	if global_position.y < -250.0:
		global_position.y = -199.0

	AudioManager.tick_footsteps(
		global_position, is_on_floor(), Vector2(velocity.x, velocity.z).length()
	)

	anim.update_animation()
	anim.update_weapon_visual()
	_update_lock_on_state()

	hud_updater.update_hud()

	# Send position + visual state to server
	if NetworkManager.is_active:
		NetworkManager.send_player_position(global_position, rotation.y, _visual_state)


func _tick_cooldowns(delta: float) -> void:
	if _parry_timer > 0.0:
		_parry_timer -= delta
	_block_cooldown = maxf(_block_cooldown - delta, 0.0)
	_vortex_cooldown = maxf(_vortex_cooldown - delta, 0.0)
	_execution_cooldown = maxf(_execution_cooldown - delta, 0.0)
	_bull_rush_cooldown = maxf(_bull_rush_cooldown - delta, 0.0)
	_brace_cooldown = maxf(_brace_cooldown - delta, 0.0)
	_spec_grace = maxf(_spec_grace - delta, 0.0)
	_retaliate_cooldown = maxf(_retaliate_cooldown - delta, 0.0)
	_shield_block_cooldown = maxf(_shield_block_cooldown - delta, 0.0)


func _process_combat_state(delta: float) -> void:
	match state:
		State.MOVE:
			movement.process_move(delta)
		State.DODGE:
			combat.process_dodge(delta)
		State.CLEAVE:
			combat.process_cleave(delta)
		State.UPHEAVAL_WINDUP:
			combat.process_upheaval_windup(delta)
		State.UPHEAVAL:
			combat.process_upheaval(delta)
		State.BLOCK:
			combat.process_block(delta)
		State.STAGGER:
			combat.process_stagger()
		State.VORTEX:
			combat.process_vortex(delta)
		State.EXECUTION_WINDUP:
			combat.process_execution_windup(delta)
		State.EXECUTION:
			combat.process_execution(delta)
		State.SHIELD_BLOCK:
			combat.process_shield_block(delta)
		State.SHIELD_BASH:
			combat.process_shield_bash(delta)
		State.BULL_RUSH:
			combat.process_bull_rush(delta)
		State.BRACE:
			combat.process_brace(delta)
		State.RETALIATE_WINDUP:
			combat.process_retaliate_windup(delta)
		State.RETALIATE:
			combat.process_retaliate(delta)
		State.GUARD_BREAK:
			combat.process_guard_break()
		State.DEAD:
			velocity.x = 0.0
			velocity.z = 0.0


func _update_lock_on_state() -> void:
	# Clear lock if target is dead, freed, or hidden
	if _lock_on_active and _lock_target:
		if (
			not is_instance_valid(_lock_target)
			or not _lock_target.visible
			or ("_server_alive" in _lock_target and not _lock_target._server_alive)
		):
			_toggle_lock_on()
	if _lock_on_active and _lock_target:
		hud.update_lock_on(_lock_target, camera)


func take_damage(_amount: float, _hit_position: Vector3 = Vector3.ZERO) -> void:
	pass  # Server handles all damage


func _enter_state(new_state: State) -> void:
	match state:
		State.DODGE:
			_is_invincible = false
	state = new_state
	if "_has_hit_this_attack" in combat:
		combat._has_hit_this_attack = false


func _consume_stamina(amount: float) -> void:
	movement.consume_stamina(amount)


func _start_dodge() -> void:
	combat.start_dodge()


func _process_dodge(delta: float) -> void:
	combat.process_dodge(delta)


func _start_cleave() -> void:
	combat.start_cleave()


func _process_cleave(delta: float) -> void:
	combat.process_cleave(delta)


func _start_upheaval() -> void:
	combat.start_upheaval()


func _process_upheaval_windup(delta: float) -> void:
	combat.process_upheaval_windup(delta)


func _toggle_lock_on() -> void:
	cam.toggle_lock_on()
