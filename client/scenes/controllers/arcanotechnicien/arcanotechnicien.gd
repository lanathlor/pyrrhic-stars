extends CharacterBody3D

## Arcanotechnicien -- Tactical Flux channeling controller.
## Harmonist spec: positional healer with Zone, Beam, and Direct delivery methods.
## Confluence mechanic: stacking spell power from consecutive casts.
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

# Input actions mapped to spell slots 0-5
const SPELL_SLOT_ACTIONS: Array[StringName] = [
	&"harmonist_slot_0",  # slot 0 -- 1 key
	&"harmonist_slot_1",  # slot 1 -- 2 key
	&"heavy_attack",      # slot 2 -- R
	&"ability_2",         # slot 3 -- T
	&"ability_1",         # slot 4 -- F
	&"dodge",             # slot 5 -- C (overloaded: dodge when no target, spell when targeting)
]

## Harmonist spell table. action_id = 50 + slot_index.
const HARMONIST_SPELLS: Array[Dictionary] = [
	{
		name = "Mending Surge",
		keybind = "1",
		desc = "Direct. Massive single-target emergency heal. High Flux cost.",
		action_id = 50,
		dur = 0.4,
		delivery = "direct",
		cooldown_max = 0.0,
	},
	{
		name = "Mending Beam",
		keybind = "2",
		desc = "Beam. High sustained single-target throughput. Channel.",
		action_id = 51,
		dur = 2.0,
		delivery = "beam",
		cooldown_max = 0.0,
	},
	{
		name = "Life Swap",
		keybind = "R",
		desc = "Direct. Drain healthy ally to empower next heal. Low Flux.",
		action_id = 52,
		dur = 0.3,
		delivery = "direct",
		cooldown_max = 6.0,
	},
	{
		name = "Transfusion",
		keybind = "T",
		desc = "Beam to Zone. Drain one ally, AoE heal everyone else.",
		action_id = 53,
		dur = 1.5,
		delivery = "zone",
		cooldown_max = 8.0,
	},
	{
		name = "Frost Ward",
		keybind = "F",
		desc = "Instant. Frost barrier on ally. Absorbs damage.",
		action_id = 54,
		dur = 0.2,
		delivery = "direct",
		cooldown_max = 12.0,
	},
	{
		name = "Gust Step",
		keybind = "C",
		desc = "Instant. Wind-propelled repositioning.",
		action_id = 55,
		dur = 0.3,
		delivery = "displacement",
		cooldown_max = 10.0,
	},
]

const MovementScript := preload("res://scenes/controllers/arcanotechnicien/arcanotechnicien_movement.gd")
const CombatScript := preload("res://scenes/controllers/arcanotechnicien/harmonist_combat.gd")
const CameraScript := preload("res://scenes/controllers/arcanotechnicien/arcanotechnicien_camera.gd")
const VfxScript := preload("res://scenes/controllers/arcanotechnicien/vfx/harmonist_vfx.gd")

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
@export var cast_range: float = 25.0
@export var cast_move_speed_mult: float = 0.35

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
var peer_id: int = 0

# State
var state: State = State.MOVE

# Sub-systems
var combat: Node
var movement: Node
var cam: Node
var vfx: Node

var spec_id: String = "harmonist"

var _state_timer: float = 0.0
var _gcd_timer: float = 0.0
var _cast_timer: float = 0.0
var _casting_spell: Dictionary = {}
var _is_invincible: bool = false

# Confluence (shared Arcanotechnicien mechanic) -- read from server state
var _confluence_tier: int = 0
var _confluence_stacks: int = 0

# Cooldowns per spell slot
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
				"casting": "slash",
				"channeling": "slash",
				"stagger": "idle",
				"dead": "idle",
			}
		)
	)

	# Sympathetic Field is visible to all players (local and remote)
	vfx.show_sympathetic_field()

	if _is_local():
		Input.set_mouse_mode(Input.MOUSE_MODE_VISIBLE)
		_update_hud_spells()
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


## Escape intercept: clear selection first, only pause if no selection.
func _input(event: InputEvent) -> void:
	if not _is_local():
		return
	if event.is_action_pressed("ui_cancel") and _selected_target != null:
		_clear_selection()
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

	# Left-click targeting
	if event is InputEventMouseButton and event.button_index == MOUSE_BUTTON_LEFT and event.pressed:
		_try_click_target(event.position)


func _physics_process(delta: float) -> void:
	if not _is_local():
		var prev_pos := global_position
		global_position = global_position.move_toward(_net_position, 12.0 * delta)
		rotation.y = lerp_angle(rotation.y, _net_rotation_y, 8.0 * delta)
		_drive_remote_animation(prev_pos, delta)
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
		if (
			not is_instance_valid(_selected_target)
			or not _selected_target.visible
		):
			_clear_selection()
		else:
			hud.update_selected_target(_selected_target, camera)

	_update_hud_spells()
	hud.update_gcd(_gcd_timer / gcd_duration if _gcd_timer > 0.0 else 0.0)
	hud.update_confluence(_confluence_tier, _confluence_stacks)
	if vfx:
		vfx.update_confluence(_confluence_tier, _confluence_stacks)
	hud.update_flux(flux, max_flux)
	_update_hud_channel()
	_update_hud_party()

	# Send position + visual state to server
	if NetworkManager.is_active:
		NetworkManager.send_player_position(global_position, rotation.y, _visual_state)


func _update_hud_spells() -> void:
	var spell_data: Array = []
	for i in HARMONIST_SPELLS.size():
		var spell: Dictionary = HARMONIST_SPELLS[i]
		spell_data.append({
			name = spell.name,
			keybind = spell.keybind,
			desc = spell.desc,
			cooldown = _cooldowns[i],
			cooldown_max = spell.cooldown_max,
		})
	hud.update_spells(spell_data)


func _update_hud_channel() -> void:
	if state == State.CHANNELING and not _casting_spell.is_empty():
		var total_dur: float = _casting_spell.get("dur", 1.0)
		var elapsed: float = total_dur - _cast_timer
		var progress: float = clampf(elapsed / maxf(total_dur, 0.01), 0.0, 1.0)
		hud.update_channel(progress, _casting_spell.get("name", ""))
	else:
		hud.hide_channel()


func _update_hud_party() -> void:
	var party: Array = []
	for p in GameManager.players:
		if not is_instance_valid(p) or not p.visible:
			continue
		if p == self:
			continue
		var pid: int = p.peer_id if "peer_id" in p else 0
		var p_health: float = p.health if "health" in p else 0.0
		var p_max_health: float = p.max_health if "max_health" in p else 150.0
		var cls: String = "unknown"
		var uname: String = "Player_%d" % pid
		if NetworkManager.player_info.has(pid):
			cls = NetworkManager.player_info[pid].get("class_name", "unknown")
			var info_name: String = NetworkManager.player_info[pid].get("username", "")
			if info_name != "":
				uname = info_name
		party.append({
			"peer_id": pid,
			"name": uname,
			"health": p_health,
			"max_health": p_max_health,
			"class_name": cls,
		})
	hud.update_party(party)


# --- Damage (server-authoritative) ---


func take_damage(_amount: float, _hit_position: Vector3 = Vector3.ZERO) -> void:
	pass  # Server handles all damage


func _drive_remote_animation(prev_pos: Vector3, delta: float) -> void:
	# Drive remote VFX on visual state change
	if _visual_state != _prev_remote_vs:
		if vfx:
			vfx.drive_remote_vfx(_prev_remote_vs, _visual_state)
		_prev_remote_vs = _visual_state

	match _visual_state:
		NetSerializer.VS_DODGE:
			character_model.travel("dodge")
		NetSerializer.VS_AT_CASTING:
			character_model.travel("casting")
		NetSerializer.VS_AT_CHANNELING, \
		NetSerializer.VS_AT_CHANNELING_BEAM, \
		NetSerializer.VS_AT_CHANNELING_ZONE:
			character_model.travel("channeling")
		NetSerializer.VS_AT_STAGGER:
			character_model.travel("stagger")
		NetSerializer.VS_DEAD:
			character_model.travel("dead")
		_:  # VS_MOVE or unknown -- derive from velocity
			var vel := (global_position - prev_pos) / delta if delta > 0 else Vector3.ZERO
			var speed := Vector2(vel.x, vel.z).length()
			if speed > 0.5:
				character_model.travel("run", clampf(speed / sprint_speed, 0.5, 1.5))
			else:
				character_model.travel("idle")


# --- State helpers ---


func _enter_state(new_state: State) -> void:
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


func _start_spell(slot: int) -> void:
	combat.start_spell(slot)


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


func _update_camera() -> void:
	cam.update_camera()


func _apply_camera_collision(from: Vector3, to: Vector3) -> Vector3:
	return cam.apply_camera_collision(from, to)


func _get_camera_wish_dir() -> Vector3:
	return movement.get_camera_wish_dir()


func _process_move(delta: float) -> void:
	movement.process_move(delta)


func _face_direction(dir: Vector3, delta: float) -> void:
	movement.face_direction(dir, delta)


func _face_target(delta: float) -> void:
	movement.face_target(delta)


func _face_attack_direction(delta: float) -> void:
	movement.face_attack_direction(delta)


func _show_body_flash() -> void:
	cam.show_body_flash()


func _update_flash(delta: float) -> void:
	cam.update_flash(delta)


func _update_animation() -> void:
	cam.update_animation()


# --- WoW-style click targeting ---


func _try_click_target(screen_pos: Vector2) -> void:
	# Check HUD party frames first (UI priority over world)
	if hud and hud.has_method("get_clicked_target"):
		var party_pid: int = hud.get_clicked_target(screen_pos)
		if party_pid > 0:
			_select_target_by_peer_id(party_pid)
			return

	# Raycast into 3D world: mask 6 = layer 2 (Player) | layer 3 (Enemy)
	var from: Vector3 = camera.project_ray_origin(screen_pos)
	var dir: Vector3 = camera.project_ray_normal(screen_pos)
	var to: Vector3 = from + dir * 100.0
	var space: PhysicsDirectSpaceState3D = get_world_3d().direct_space_state
	if not space:
		return
	var query := PhysicsRayQueryParameters3D.create(from, to, 6)
	query.exclude = [get_rid()]
	var result: Dictionary = space.intersect_ray(query)
	if result:
		var hit_node: Node3D = result.collider
		# Walk up to find a node with peer_id (player or enemy)
		while hit_node and not ("peer_id" in hit_node):
			hit_node = hit_node.get_parent()
		if hit_node and "peer_id" in hit_node and hit_node != self:
			_select_target(hit_node)
			return
	# Clicked empty space — clear selection
	_clear_selection()


func _select_target(target: Node3D) -> void:
	_selected_target = target
	hud.show_selected_target(target, camera)


func _select_target_by_peer_id(pid: int) -> void:
	for player in GameManager.players:
		if is_instance_valid(player) and player.visible and "peer_id" in player:
			if player.peer_id == pid and player != self:
				_select_target(player)
				return
	for enemy in GameManager.enemies:
		if is_instance_valid(enemy) and enemy.visible and "peer_id" in enemy:
			if enemy.peer_id == pid:
				_select_target(enemy)
				return


func _clear_selection() -> void:
	_selected_target = null
	hud.hide_selected_target()


## Stub for spec switching (only harmonist implemented for now).
func _switch_spec(new_spec: String, _from_client: bool = false) -> void:
	if new_spec == spec_id:
		return
	spec_id = new_spec
	# Future: swap combat subsystem when Destroyer/Battlemage are implemented
