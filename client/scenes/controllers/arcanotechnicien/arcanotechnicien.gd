extends CharacterBody3D

## Arcanotechnicien -- Tactical Flux channeling controller.
## Harmonist spec: positional healer with Zone, Beam, and Direct delivery methods.
## Confluence mechanic: stacking ability power from consecutive commits.
## Sub-systems: combat, movement, cam (child nodes).

signal died

enum State {
	MOVE,
	DODGE,
	CASTING,
	CHANNELING,
	STAGGER,
	DEAD,
}

const NET_INTERP_SPEED := 15.0

# Input actions mapped to ability slots 0-5
const ABILITY_SLOT_ACTIONS: Array[StringName] = [
	&"harmonist_slot_0",  # slot 0 -- 1 key
	&"harmonist_slot_1",  # slot 1 -- 2 key
	&"heavy_attack",  # slot 2 -- R
	&"ability_2",  # slot 3 -- T
	&"ability_1",  # slot 4 -- F
	&"dodge",  # slot 5 -- C (overloaded: dodge when no target, ability when targeting)
]

const SLOT_KEYBINDS: Array[String] = ["1", "2", "R", "T", "F", "C"]

## Harmonist ability table. Populated from server via AbilityCatalog.
## Empty: the server loadout is the source of truth.
const HARMONIST_ABILITIES: Array[Dictionary] = []

const MovementScript := preload(
	"res://scenes/controllers/arcanotechnicien/arcanotechnicien_movement.gd"
)
const CombatScript := preload("res://scenes/controllers/arcanotechnicien/harmonist_combat.gd")
const CameraScript := preload(
	"res://scenes/controllers/arcanotechnicien/arcanotechnicien_camera.gd"
)
const VfxScript := preload("res://scenes/controllers/arcanotechnicien/vfx/harmonist_vfx.gd")
const HudUpdaterScript := preload(
	"res://scenes/controllers/arcanotechnicien/arcanotechnicien_hud_updater.gd"
)
const TargetingScript := preload(
	"res://scenes/controllers/arcanotechnicien/arcanotechnicien_targeting.gd"
)

# Movement
@export var run_speed: float = 5.5
@export var sprint_speed: float = 8.0
@export var mouse_sensitivity: float = 0.003
@export var ground_accel: float = 20.0
@export var ground_decel: float = 15.0
@export var air_accel: float = 2.0
@export var air_decel: float = 1.0
@export var rotation_speed: float = 10.0

# Dodge
@export var dodge_speed: float = 11.0
@export var dodge_duration: float = 0.35
@export var dodge_iframe_duration: float = 0.12

# Casting
@export var ability_range: float = 25.0
@export var commit_move_speed_mult: float = 0.35

# GCD
@export var gcd_duration: float = 0.8

# Camera -- pulled back for spatial awareness
@export var camera_distance: float = 8.0
@export var camera_height_offset: float = 2.5

# Health & Flux
var health: float = 150.0
var max_health: float = 150.0
var flux: float = 100.0
var max_flux: float = 100.0
var flux_pools: Array = []  # per-school: [{school, current, max}, ...]
var peer_id: int = 0

# State
var state: State = State.MOVE

# Sub-systems
var combat: Node
var movement: Node
var cam: Node
var vfx: Node
var hud_updater: Node
var targeting: Node

var spec_id: String = "harmonist"

var _state_timer: float = 0.0
var _gcd_timer: float = 0.0
var _cast_timer: float = 0.0
var _committing_ability: Dictionary = {}
var _is_invincible: bool = false

# Confluence (shared Arcanotechnicien mechanic) -- read from server state
var _confluence_tier: int = 0
var _confluence_stacks: int = 0

# Cooldowns per ability slot
var _cooldowns: Array[float] = [0.0, 0.0, 0.0, 0.0, 0.0, 0.0]

# Camera
var _camera_yaw: float = 0.0
var _camera_pitch: float = -0.3

# WoW-style click targeting
var _selected_target: Node3D = null
var _right_mouse_held: bool = false
var _rmb_cursor_pos: Vector2 = Vector2.ZERO  # cursor position before RMB drag
var _rmb_last_mouse: Vector2 = Vector2.ZERO  # last mouse pos for manual delta

var _gravity: float = 8.5
var _alive: bool = true
var _prev_remote_vs: int = -1

# Network sync
var _visual_state: int = 0
var _net_position: Vector3 = Vector3.ZERO
var _net_rotation_y: float = 0.0

@onready var camera: Camera3D = $Camera3D
@onready var character_model: Node3D = $CharacterModel
@onready var hud: Control = $HUDLayer/ArcanotechnicienHUD


func _ready() -> void:
	# Create sub-systems
	combat = _add_subsystem("Combat", CombatScript)
	movement = _add_subsystem("Movement", MovementScript)
	cam = _add_subsystem("Cam", CameraScript)
	vfx = _add_subsystem("Vfx", VfxScript)
	hud_updater = _add_subsystem("HudUpdater", HudUpdaterScript)
	targeting = _add_subsystem("Targeting", TargetingScript)

	GameManager.register_player(self)
	_net_position = global_position
	_net_rotation_y = rotation.y
	camera.top_level = true

	# Set up animation state machine
	(
		character_model
		. setup_state_machine(
			{
				"idle": "idle",
				"run": "run",
				"jump": "jump",
				"dodge": "roll",
				"casting": "ual_spell_shoot",
				"channeling": "ual_spell_idle",
				"stagger": "ual_hit_chest",
				"dead": "ual_death",
			}
		)
	)

	# Sympathetic Field is visible to all players (local and remote)
	vfx.show_sympathetic_field()

	if _is_local():
		Input.set_mouse_mode(Input.MOUSE_MODE_VISIBLE)
		hud_updater.update_abilities()
	else:
		$HUDLayer.visible = false
		camera.current = false
		set_process_unhandled_input(false)
		set_process_input(false)


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
	# Confluence uses the same wire bytes as onslaught (reused field).
	_confluence_tier = data.get("onslaught_tier", 0)
	_confluence_stacks = data.get("onslaught_stacks", 0)
	flux = data.get("flux", flux)
	if data.get("max_flux", 0.0) > 0.0:
		max_flux = data["max_flux"]
	flux_pools = data.get("flux_pools", flux_pools)
	var server_channel_phase: int = data.get("channel_phase", 0)
	if _is_local():
		health = data.health
		if health <= 0.0 and _alive:
			_alive = false
			_enter_state(State.DEAD)
			died.emit()
		elif health > 0.0 and not _alive:
			_alive = true
			_enter_state(State.MOVE)
			global_position = data.pos
			_net_position = data.pos
		# Server cancelled sustain (damage, movement, flux depleted).
		# Only cancel when server is idle (phase 0) — phases 1/2 (commit/execute)
		# mean the server hasn't entered sustain yet, not that it cancelled.
		if combat._sustaining and server_channel_phase == 0:
			combat.cancel_sustain()
	else:
		_net_position = data.pos
		_net_rotation_y = data.rot_y
		health = data.health
		_visual_state = data.get("visual_state", 0)


## Called by main.gd when server confirms this player hit an enemy.
func on_hit_confirmed(_amount: float, _hit_pos: Vector3 = Vector3.ZERO) -> void:
	hud.show_hit_marker()


## Visual-only damage feedback (called from main.gd on DamageEvent).
func on_damage_visual(_amount: float, _hit_pos: Vector3) -> void:
	hud.show_damage_flash()
	cam.show_body_flash()


## Visual-only heal feedback.
func on_heal_visual(_amount: float, hit_pos: Vector3) -> void:
	hud.show_heal_flash()
	if vfx:
		vfx.spawn_heal_pulse(hit_pos)


## Escape intercept: close codex first, then clear selection, then pause.
func _input(event: InputEvent) -> void:
	if not _is_local():
		return
	if event.is_action_pressed("ui_cancel"):
		if combat._sustaining:
			combat.cancel_sustain()
			if NetworkManager.is_active:
				NetworkManager.send_ability(255, 0.0, rotation.y)
			get_viewport().set_input_as_handled()
			return
		if state == State.CASTING or state == State.CHANNELING:
			combat.cancel_commit()
			if NetworkManager.is_active:
				NetworkManager.send_ability(255, 0.0, rotation.y)
			get_viewport().set_input_as_handled()
			return
		if hud.is_codex_open():
			hud.close_codex()
			get_viewport().set_input_as_handled()
			return
		if _selected_target != null:
			targeting.clear_selection()
			get_viewport().set_input_as_handled()


func _unhandled_input(event: InputEvent) -> void:
	if not _is_local():
		return

	# Right mouse button: hide cursor while held for camera drag
	if event is InputEventMouseButton and event.button_index == MOUSE_BUTTON_RIGHT:
		_right_mouse_held = event.pressed
		if _right_mouse_held:
			_rmb_cursor_pos = get_viewport().get_mouse_position()
			_rmb_last_mouse = _rmb_cursor_pos
			Input.set_mouse_mode(Input.MOUSE_MODE_CONFINED_HIDDEN)
		else:
			get_viewport().warp_mouse(_rmb_cursor_pos)
			Input.set_mouse_mode(Input.MOUSE_MODE_VISIBLE)
		get_viewport().set_input_as_handled()

	# Camera rotation: compute delta manually (CONFINED_HIDDEN has no relative)
	if event is InputEventMouseMotion and _right_mouse_held:
		var current_mouse: Vector2 = get_viewport().get_mouse_position()
		var delta_mouse: Vector2 = current_mouse - _rmb_last_mouse
		_rmb_last_mouse = current_mouse
		_camera_yaw -= delta_mouse.x * mouse_sensitivity
		_camera_pitch -= delta_mouse.y * mouse_sensitivity
		_camera_pitch = clampf(_camera_pitch, deg_to_rad(-60.0), deg_to_rad(20.0))

	# Toggle codex with P
	if event.is_action_pressed("toggle_codex"):
		if hud.is_codex_open():
			hud.close_codex()
		elif state == State.MOVE and _alive:
			hud.toggle_codex()
		get_viewport().set_input_as_handled()
		return

	# Left-click targeting
	if event is InputEventMouseButton and event.button_index == MOUSE_BUTTON_LEFT and event.pressed:
		targeting.try_click_target(event.position)


func _physics_process(delta: float) -> void:
	if not _is_local():
		var prev_pos := global_position
		global_position = global_position.move_toward(_net_position, 12.0 * delta)
		rotation.y = lerp_angle(rotation.y, _net_rotation_y, 8.0 * delta)
		cam.drive_remote_animation(prev_pos, delta)
		return

	if not is_on_floor():
		velocity.y -= _gravity * delta
	else:
		velocity.y = -0.5

	_state_timer -= delta
	_gcd_timer -= delta
	cam.update_camera()

	# Tick cooldowns
	for i in _cooldowns.size():
		_cooldowns[i] = maxf(_cooldowns[i] - delta, 0.0)

	match state:
		State.MOVE:
			movement.process_move(delta)
		State.DODGE:
			combat.process_dodge(delta)
		State.CASTING:
			combat.process_casting(delta)
		State.CHANNELING:
			combat.process_channeling(delta)
		State.STAGGER:
			combat.process_stagger()
		State.DEAD:
			velocity.x = 0.0
			velocity.z = 0.0

	move_and_slide()
	if global_position.y < -250.0:
		global_position.y = -199.0

	cam.update_animation()
	# Clear selection if target is dead, freed, or hidden
	if _selected_target:
		if not is_instance_valid(_selected_target) or not _selected_target.visible:
			targeting.clear_selection()
		else:
			hud.update_selected_target(_selected_target, camera)

	hud_updater.update_hud()

	# Send position + visual state to server
	if NetworkManager.is_active:
		NetworkManager.send_player_position(global_position, rotation.y, _visual_state)


# --- Damage (server-authoritative) ---


func take_damage(_amount: float, _hit_position: Vector3 = Vector3.ZERO) -> void:
	pass  # Server handles all damage


# --- State helpers ---


func _enter_state(new_state: State) -> void:
	# Close codex on any non-MOVE state (combat gating)
	if new_state != State.MOVE:
		hud.close_codex()
	# Clean up VFX when leaving casting/channeling states (interruption, stagger)
	match state:
		State.DODGE:
			_is_invincible = false
		State.CASTING, State.CHANNELING:
			if new_state != State.CASTING and new_state != State.CHANNELING:
				if vfx:
					vfx.stop_channel_flux()
					vfx.stop_heal_beam()
					vfx.stop_zone_telegraph()
	state = new_state


# --- Delegate wrappers for test/bot compatibility ---


func _start_ability(slot: int) -> void:
	combat.start_ability(slot)


func _start_dodge() -> void:
	combat.start_dodge()


func _process_dodge(delta: float) -> void:
	combat.process_dodge(delta)


func _process_casting(delta: float) -> void:
	combat.process_casting(delta)


func _process_channeling(delta: float) -> void:
	combat.process_channeling(delta)


func _process_stagger() -> void:
	combat.process_stagger()


func _update_hud_channel() -> void:
	hud_updater.update_channel()


## Stub for spec switching (only harmonist implemented for now).
func _switch_spec(new_spec: String, _from_client: bool = false) -> void:
	if new_spec == spec_id:
		return
	spec_id = new_spec
	# Future: swap combat subsystem when Destroyer/Battlemage are implemented
