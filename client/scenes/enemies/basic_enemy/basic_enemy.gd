extends CharacterBody3D

## Visual puppet for the arena boss.
## Receives authoritative state from the server via apply_server_state().
## Contains NO game logic — only interpolation, telegraph visuals, animations.
## VFX/telegraph/health bar creation delegated to BasicEnemyVfx child node.

signal died

# State enum kept for animation mapping only.
# Values match server EnemyState:
# 0=Idle, 1=Chase, 2=MeleeTelegraph, 3=MeleeAttack, 4=RangedTelegraph,
# 5=RangedAttack, 6=AoETelegraph, 7=AoESlam, 8=ChargeTelegraph, 9=Charge,
# 10=Cooldown, 11=PhaseTransition, 12=Dead, 13=Patrol
enum State {
	IDLE,
	CHASE,
	MELEE_TELEGRAPH,
	MELEE_ATTACK,
	RANGED_TELEGRAPH,
	RANGED_ATTACK,
	AOE_TELEGRAPH,
	AOE_SLAM,
	CHARGE_TELEGRAPH,
	CHARGE,
	COOLDOWN,
	PHASE_TRANSITION,
	DEAD,
	PATROL,
}

# Interpolation
const NET_INTERP_SPEED := 15.0
const SWORD_SCENE_PATH := "res://assets/models/weapons/weapon_longsword.glb"
const GUN_SCENE_PATH := "res://assets/models/weapons/weapon_rifle.glb"

# Stats needed for health bar display
@export var max_health: float = 2000.0
@export var melee_range: float = 3.0

var health: float
var state: State = State.IDLE
# Enemy network identity (server assigns IDs >= 1000 to avoid player peer ID collision)
var peer_id: int = 0

var _melee_cone_angle: float = PI  # full cone angle in radians (default 180)
# Phase tracking (for health bar color and charge distance)
var _current_phase: int = 1
# Server state (set by main.gd from WorldState)
var _server_position: Vector3 = Vector3.ZERO
var _server_rotation_y: float = 0.0
var _server_health: float = 2000.0
var _server_state: int = 0
var _server_phase: int = 1
var _server_ranged_target: Vector3 = Vector3.ZERO
var _server_charge_dir: Vector3 = Vector3.ZERO
var _server_alive: bool = true
var _last_synced_state: int = -1
var _prev_position: Vector3 = Vector3.ZERO
var _visual_velocity: Vector3 = Vector3.ZERO
# Weapon nodes (bone-attached via CharacterModel)
var _sword_node: Node3D
var _gun_node: Node3D
var _sword_attachment: BoneAttachment3D
var _gun_attachment: BoneAttachment3D
var _last_weapon: String = "sword"  # which weapon to show between attacks
var _def_name: String = ""  # enemy definition name from server
# Ranged target position for laser warning visual
var _ranged_target_position: Vector3
# Charge direction for charge telegraph visual
var _charge_direction: Vector3 = Vector3.ZERO

# Scene references
@onready var character_model: Node3D = $CharacterModel
@onready var vfx = $BasicEnemyVfx


func _ready() -> void:
	health = max_health
	_server_position = global_position
	_server_rotation_y = rotation.y
	GameManager.register_enemy(self)

	vfx.create_health_bar()
	vfx.create_melee_telegraph(melee_range)
	vfx.create_laser_warning()
	vfx.create_aoe_telegraph()
	vfx.create_aoe_particles()
	vfx.create_charge_telegraph()
	_attach_weapons.call_deferred()

	# Set up animation state machine for enemy
	(
		character_model
		. setup_state_machine(
			{
				"sword_idle": "sword_idle",
				"sword_run": "sword_run",
				"gun_idle": "rifle_aim_idle",
				"gun_run": "rifle_aim_run",
				"melee_windup": "sword_heavy",
				"melee_attack": "sword_slash_1",
				"gun_shoot": "rifle_shoot",
			}
		)
	)


func _exit_tree() -> void:
	GameManager.unregister_enemy(self)


# =============================================================================
# Server state application
# =============================================================================


func apply_server_state(data: Dictionary) -> void:
	_server_position = data.pos
	_server_rotation_y = data.rot_y
	_server_health = data.health
	_server_state = data.state
	_server_phase = data.phase
	_server_ranged_target = data.ranged_target
	_server_charge_dir = data.charge_dir
	_server_alive = data.alive
	health = _server_health
	_current_phase = _server_phase
	_ranged_target_position = _server_ranged_target
	_charge_direction = _server_charge_dir
	# Dynamic max_health from server (varies per enemy def)
	if data.has("max_health") and data["max_health"] > 0.0:
		max_health = data["max_health"]
	if data.has("melee_cone_angle") and data["melee_cone_angle"] > 0.0:
		_melee_cone_angle = data["melee_cone_angle"]
	if data.has("melee_range") and data["melee_range"] > 0.0:
		melee_range = data["melee_range"]
	if data.has("def_name") and data["def_name"] != "":
		if _def_name == "" and data["def_name"] != _def_name:
			_def_name = data["def_name"]
			# Ranged enemies default to gun weapon
			if _def_name == "hallway_ranged":
				_last_weapon = "gun"


func _physics_process(delta: float) -> void:
	if not vfx:
		return
	vfx.face_health_bar_to_camera()
	_update_weapons(delta)
	character_model.position.y = 0.0

	# Interpolate position/rotation from server
	var old_pos := global_position
	global_position = global_position.lerp(_server_position, NET_INTERP_SPEED * delta)
	rotation.y = lerp_angle(rotation.y, _server_rotation_y, NET_INTERP_SPEED * delta)
	# Compute visual velocity for animation (run vs idle)
	if delta > 0.001:
		_visual_velocity = (global_position - old_pos) / delta

	# Update visuals based on server state
	_update_state_visuals()
	vfx.update_health_bar()
	vfx.update_health_bar_color(_current_phase)
	_update_boss_animation()

	# Handle death
	if not _server_alive and visible:
		visible = false
		collision_layer = 0
		died.emit()


# =============================================================================
# State visual sync
# =============================================================================


func _update_state_visuals() -> void:
	# Map server state int to telegraph visibility
	var synced_state: int = _server_state
	if synced_state != _last_synced_state:
		_last_synced_state = synced_state
		vfx.melee_telegraph_mesh.visible = false
		vfx.laser_warning_mesh.visible = false
		vfx.aoe_telegraph_mesh.visible = false
		vfx.charge_telegraph_mesh.visible = false
		if vfx.aoe_particles:
			vfx.aoe_particles.emitting = false
		# 2=MeleeTelegraph, 4=RangedTelegraph, 6=AoETelegraph,
		# 7=AoESlam, 8=ChargeTelegraph, 12=Dead
		match synced_state:
			2:  # MELEE_TELEGRAPH
				vfx.update_melee_telegraph_params(melee_range, _melee_cone_angle)
				vfx.melee_telegraph_mesh.visible = true
			4:  # RANGED_TELEGRAPH
				vfx.laser_warning_mesh.visible = true
			6:  # AOE_TELEGRAPH
				vfx.aoe_telegraph_mesh.visible = true
				if vfx.aoe_particles:
					vfx.aoe_particles.emitting = true
			7:  # AOE_SLAM
				if vfx.aoe_slam_particles:
					vfx.aoe_slam_particles.emitting = true
			8:  # CHARGE_TELEGRAPH
				vfx.charge_telegraph_mesh.visible = true
			12:  # DEAD
				visible = false
		state = synced_state as State
	# Update telegraph positions for synced data
	if vfx.laser_warning_mesh.visible:
		vfx.update_laser_warning(_ranged_target_position)
	if vfx.charge_telegraph_mesh.visible:
		_update_charge_indicator()


# =============================================================================
# Damage visual (called externally for hit flash)
# =============================================================================


func on_damage_visual(_amount: float, _hit_pos: Vector3) -> void:
	character_model.flash_damage()


# =============================================================================
# Character model animations
# =============================================================================


func _update_boss_animation() -> void:
	match state:
		State.PATROL:
			if _last_weapon == "gun":
				character_model.travel("gun_run", 0.5)
			else:
				character_model.travel("sword_run", 0.5)
		State.CHASE:
			var flat_speed := Vector2(_visual_velocity.x, _visual_velocity.z).length()
			if _last_weapon == "gun":
				if flat_speed > 0.5:
					character_model.travel("gun_run")
				else:
					character_model.travel("gun_idle")
			else:
				if flat_speed > 0.5:
					character_model.travel("sword_run")
				else:
					character_model.travel("sword_idle")
		State.MELEE_TELEGRAPH:
			character_model.travel("melee_windup", 0.3)
		State.MELEE_ATTACK:
			character_model.travel("melee_attack")
		State.RANGED_TELEGRAPH:
			character_model.travel("gun_idle")
		State.RANGED_ATTACK:
			character_model.travel("gun_shoot")
		State.AOE_TELEGRAPH, State.AOE_SLAM:
			character_model.travel("sword_idle")
		State.CHARGE_TELEGRAPH:
			character_model.travel("sword_idle")
		State.CHARGE:
			character_model.travel("sword_run", 1.5)
		State.COOLDOWN, State.PHASE_TRANSITION, State.DEAD:
			if _last_weapon == "gun":
				character_model.travel("gun_idle")
			else:
				character_model.travel("sword_idle")


# =============================================================================
# Charge telegraph update (uses vfx mesh directly)
# =============================================================================


func _update_charge_indicator() -> void:
	if _charge_direction.length() < 0.1:
		return
	var start := global_position + Vector3(0.0, 0.05, 0.0)
	var max_dist := _get_charge_max_distance()
	var end := start + _charge_direction * max_dist
	var mid := (start + end) / 2.0
	mid.y = 0.05

	vfx.charge_telegraph_mesh.global_position = mid
	vfx.charge_telegraph_mesh.scale = Vector3(1.0, 1.0, max_dist)
	if max_dist > 0.1:
		vfx.charge_telegraph_mesh.look_at(end, Vector3.UP)


func _get_charge_max_distance() -> float:
	match _current_phase:
		2:
			return 18.0
		3:
			return 20.0
	return 15.0


# =============================================================================
# Weapons (bone-attached via CharacterModel)
# =============================================================================


func _attach_weapons() -> void:
	# Sword in right hand — used for melee, charge, AoE
	_sword_node = character_model.attach_weapon(
		SWORD_SCENE_PATH,
		"mixamorig_RightHand",
		Vector3(0.0, 0.08, 0.0),
		Vector3(deg_to_rad(20.0), 0.0, deg_to_rad(-90.0))
	)
	if _sword_node:
		_sword_node.scale = Vector3(1.3, 1.3, 1.3)  # boss-sized
		# Store attachment for show/hide
		_sword_attachment = _sword_node.get_parent() as BoneAttachment3D

	# Gun in left hand — used for ranged
	var skel: Skeleton3D = character_model._skeleton
	if skel:
		var bone_idx: int = skel.find_bone("mixamorig_RightHand")
		if bone_idx >= 0:
			_gun_attachment = BoneAttachment3D.new()
			_gun_attachment.bone_name = "mixamorig_RightHand"
			skel.add_child(_gun_attachment)

			var gun_scene := load(GUN_SCENE_PATH) as PackedScene
			if gun_scene:
				_gun_node = gun_scene.instantiate()
				_gun_node.position = Vector3(0.0, 0.1, 0.0)
				_gun_node.rotation = Vector3(deg_to_rad(180.0), deg_to_rad(90.0), 0.0)
				_gun_node.scale = Vector3(1.5, 1.5, 1.5)  # boss-sized
				_gun_attachment.add_child(_gun_node)

	# Show weapon based on _last_weapon (set by apply_server_state before deferred call)
	if _sword_attachment:
		_sword_attachment.visible = (_last_weapon == "sword")
	if _gun_attachment:
		_gun_attachment.visible = (_last_weapon == "gun")


func _update_weapons(_delta: float) -> void:
	# Track which weapon was last actively used
	match state:
		State.MELEE_TELEGRAPH, State.MELEE_ATTACK:
			_last_weapon = "sword"
		State.CHARGE_TELEGRAPH, State.CHARGE:
			_last_weapon = "sword"
		State.AOE_TELEGRAPH, State.AOE_SLAM:
			_last_weapon = "sword"
		State.RANGED_TELEGRAPH, State.RANGED_ATTACK:
			_last_weapon = "gun"

	# Show last used weapon during idle states
	var show_sword := _last_weapon == "sword"
	var show_gun := _last_weapon == "gun"

	if _sword_attachment:
		_sword_attachment.visible = show_sword
	if _gun_attachment:
		_gun_attachment.visible = show_gun
