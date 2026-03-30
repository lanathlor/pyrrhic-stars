extends CharacterBody3D

## Arena boss with 4 attack types and 3 health phases.
## Phase 1 (100-60%): teaching — long telegraphs, moderate damage.
## Phase 2 (60-30%): pressure — faster, ranged burst x2.
## Phase 3 (30-0%): enrage — short telegraphs, burst x3, red glow.

signal died

enum State {
	IDLE, CHASE,
	MELEE_TELEGRAPH, MELEE_ATTACK,
	RANGED_TELEGRAPH, RANGED_ATTACK,
	AOE_TELEGRAPH, AOE_SLAM,
	CHARGE_TELEGRAPH, CHARGE,
	COOLDOWN, PHASE_TRANSITION, DEAD,
}

# Base stats (Phase 1 values, overridden by phase getters)
@export var max_health: float = 2000.0
@export var melee_range: float = 3.0

var health: float
var state: State = State.IDLE
var _state_timer: float = 0.0
var _chase_timer: float = 0.0
var _target_player: CharacterBody3D = null
var _ranged_target_position: Vector3
var _gravity: float = ProjectSettings.get_setting("physics/3d/default_gravity")

# Phase tracking
var _current_phase: int = 1
var _phase_transitioned: Array[int] = []
var _last_attack: String = ""

# Charge tracking
var _charge_direction: Vector3 = Vector3.ZERO
var _charge_distance_traveled: float = 0.0
var _charge_hit_players: Array = []

# Navigation
@onready var nav_agent: NavigationAgent3D = $NavigationAgent3D

# Attack weights per phase: [melee, ranged, aoe, charge]
const PHASE_WEIGHTS := {
	1: [30, 30, 20, 20],
	2: [25, 25, 25, 25],
	3: [20, 20, 25, 35],
}

# Dynamic visual nodes
var _melee_telegraph_mesh: MeshInstance3D
var _laser_warning_mesh: MeshInstance3D
var _aoe_telegraph_mesh: MeshInstance3D
var _charge_telegraph_mesh: MeshInstance3D
var _health_bar_pivot: Node3D
var _health_bar_fg: MeshInstance3D

# Scene references
@onready var melee_area: Area3D = $MeleeArea
@onready var melee_collision: CollisionShape3D = $MeleeArea/CollisionShape3D
@onready var body_mesh: MeshInstance3D = $Body

var _flash_timer: float = 0.0
var _default_body_material: StandardMaterial3D
var _phase_body_material: StandardMaterial3D

const PROJECTILE_SCENE_PATH := "res://scenes/enemies/basic_enemy/enemy_projectile.tscn"
var _projectile_scene: PackedScene


func _ready() -> void:
	health = max_health
	_projectile_scene = load(PROJECTILE_SCENE_PATH)
	melee_collision.disabled = true
	GameManager.register_enemy(self)

	# Store default body material for flash reset
	_default_body_material = body_mesh.get_surface_override_material(0)

	_create_health_bar()
	_create_melee_telegraph()
	_create_laser_warning()
	_create_aoe_telegraph()
	_create_charge_telegraph()
	_change_state(State.CHASE)


func _exit_tree() -> void:
	GameManager.unregister_enemy(self)


func _physics_process(delta: float) -> void:
	if not is_on_floor():
		velocity.y -= _gravity * delta

	_state_timer -= delta
	_update_flash(delta)
	_face_health_bar_to_camera()

	match state:
		State.CHASE:
			_process_chase(delta)
		State.MELEE_TELEGRAPH:
			_process_melee_telegraph()
		State.MELEE_ATTACK:
			_process_melee_attack()
		State.RANGED_TELEGRAPH:
			_process_ranged_telegraph()
		State.RANGED_ATTACK:
			_process_ranged_attack()
		State.AOE_TELEGRAPH:
			_process_aoe_telegraph()
		State.AOE_SLAM:
			_process_aoe_slam()
		State.CHARGE_TELEGRAPH:
			_process_charge_telegraph()
		State.CHARGE:
			_process_charge(delta)
		State.COOLDOWN:
			_process_cooldown()
		State.PHASE_TRANSITION:
			_process_phase_transition()
		State.DEAD:
			velocity = Vector3.ZERO

	move_and_slide()


# =============================================================================
# Phase-aware stat getters
# =============================================================================

func _get_move_speed() -> float:
	match _current_phase:
		2: return 5.0
		3: return 6.0
	return 4.0


func _get_melee_damage() -> float:
	if _current_phase == 3:
		return 35.0
	return 30.0


func _get_melee_telegraph_time() -> float:
	match _current_phase:
		2: return 0.9
		3: return 0.7
	return 1.2


func _get_ranged_telegraph_time() -> float:
	match _current_phase:
		2: return 0.8
		3: return 0.6
	return 1.0


func _get_ranged_per_projectile_damage() -> float:
	match _current_phase:
		2: return 15.0
		3: return 12.0
	return 20.0


func _get_ranged_burst_count() -> int:
	match _current_phase:
		2: return 2
		3: return 3
	return 1


func _get_aoe_damage() -> float:
	if _current_phase == 3:
		return 45.0
	return 40.0


func _get_aoe_radius() -> float:
	match _current_phase:
		2: return 6.0
		3: return 7.0
	return 5.0


func _get_aoe_telegraph_time() -> float:
	match _current_phase:
		2: return 1.2
		3: return 1.0
	return 1.5


func _get_charge_damage() -> float:
	if _current_phase == 3:
		return 40.0
	return 35.0


func _get_charge_speed() -> float:
	match _current_phase:
		2: return 14.0
		3: return 16.0
	return 12.0


func _get_charge_telegraph_time() -> float:
	match _current_phase:
		2: return 0.8
		3: return 0.6
	return 1.0


func _get_charge_max_distance() -> float:
	match _current_phase:
		2: return 18.0
		3: return 20.0
	return 15.0


func _get_cooldown_time() -> float:
	match _current_phase:
		2: return 1.2
		3: return 0.9
	return 1.5


# =============================================================================
# Line of sight
# =============================================================================

func _has_line_of_sight(target_pos: Vector3) -> bool:
	var space := get_world_3d().direct_space_state
	if not space:
		return true
	var from := global_position + Vector3(0.0, 1.0, 0.0)
	var to := target_pos + Vector3(0.0, 1.0, 0.0)
	var query := PhysicsRayQueryParameters3D.create(from, to, 1)  # mask 1 = World layer
	query.exclude = [get_rid()]
	var result := space.intersect_ray(query)
	return result.is_empty()


# =============================================================================
# Attack selection (weighted random with context + anti-repeat)
# =============================================================================

func _select_attack() -> State:
	var weights: Array = PHASE_WEIGHTS[_current_phase].duplicate()
	var attack_names := ["melee", "ranged", "aoe", "charge"]

	# Context: distance to nearest player
	var nearest := GameManager.get_nearest_player(global_position)
	var distance := 999.0
	if nearest and is_instance_valid(nearest):
		var to := nearest.global_position - global_position
		to.y = 0.0
		distance = to.length()

	# Line of sight check — no ranged or charge without clear path
	var has_los := _has_line_of_sight(nearest.global_position) if nearest else false
	if not has_los:
		weights[1] = 0  # no ranged without LOS
		weights[3] = 0  # no charge without LOS

	if distance <= melee_range * 2.0:
		weights[0] = int(weights[0] * 1.5)  # melee boost
		weights[1] = 0                       # no ranged at close range
		weights[2] = int(weights[2] * 1.3)  # AoE good up close
		weights[3] = int(weights[3] * 0.3)  # charge useless up close
	elif distance > melee_range * 3.0:
		weights[0] = int(weights[0] * 0.3)  # melee unlikely if far
		weights[1] = int(weights[1] * 1.5)  # ranged boost
		weights[3] = int(weights[3] * 1.5)  # charge to close gap

	# Anti-repeat: halve weight of last attack
	var last_idx := attack_names.find(_last_attack)
	if last_idx >= 0:
		weights[last_idx] = maxi(weights[last_idx] / 2, 1)

	# Weighted random
	var total := 0
	for w in weights:
		total += w
	var roll := randi() % maxi(total, 1)
	var cumulative := 0
	for i in weights.size():
		cumulative += weights[i]
		if roll < cumulative:
			_last_attack = attack_names[i]
			match i:
				0: return State.MELEE_TELEGRAPH
				1: return State.RANGED_TELEGRAPH
				2: return State.AOE_TELEGRAPH
				3: return State.CHARGE_TELEGRAPH

	_last_attack = "melee"
	return State.MELEE_TELEGRAPH


# =============================================================================
# State processors
# =============================================================================

func _process_chase(delta: float) -> void:
	_chase_timer += delta
	var target := GameManager.get_nearest_player(global_position)
	if not target:
		velocity.x = 0.0
		velocity.z = 0.0
		return

	_target_player = target
	var to_target := target.global_position - global_position
	to_target.y = 0.0
	var distance := to_target.length()

	if distance > 0.1:
		look_at(global_position + to_target.normalized(), Vector3.UP)

	# In melee range — attack immediately (never ranged up close)
	if distance <= melee_range:
		var attack := _select_attack()
		if attack == State.RANGED_TELEGRAPH:
			attack = State.MELEE_TELEGRAPH
		if attack == State.CHARGE_TELEGRAPH:
			attack = State.AOE_TELEGRAPH
		_change_state(attack)
		return

	# Out of melee range — attack after short chase (1.5s), or immediately if far
	var chase_threshold := 1.5 if distance <= melee_range * 3.0 else 0.5
	if _chase_timer >= chase_threshold:
		var attack := _select_attack()
		# Can't melee from here — reroll
		if attack == State.MELEE_TELEGRAPH and distance > melee_range:
			if _has_line_of_sight(target.global_position):
				attack = State.CHARGE_TELEGRAPH if distance > melee_range * 2.0 else State.RANGED_TELEGRAPH
			else:
				attack = State.AOE_TELEGRAPH  # AoE doesn't need LOS
		# AoE slam useless at long range — reroll
		if attack == State.AOE_TELEGRAPH and distance > _get_aoe_radius() * 1.5:
			if _has_line_of_sight(target.global_position):
				attack = State.CHARGE_TELEGRAPH
			else:
				# No LOS and too far for AoE — keep chasing
				_chase_timer = 0.0
				# fall through to navigation movement below
				attack = State.CHASE
		if attack != State.CHASE:
			if attack == State.RANGED_TELEGRAPH:
				var ranged_target := GameManager.get_farthest_player(global_position)
				if ranged_target:
					_target_player = ranged_target
			_change_state(attack)
			return

	# --- Navigation-based movement (with direct fallback) ---
	if distance > melee_range * 0.8:
		var dir: Vector3
		var spd := _get_move_speed()

		# Only update nav target when player moved significantly (avoids per-frame pathfinding)
		if nav_agent.target_position.distance_to(target.global_position) > 1.5:
			nav_agent.target_position = target.global_position
		if not nav_agent.is_navigation_finished():
			var next_pos := nav_agent.get_next_path_position()
			dir = (next_pos - global_position)
			dir.y = 0.0
			if dir.length() > 0.1:
				dir = dir.normalized()
			else:
				dir = to_target.normalized()
		else:
			# Fallback: direct movement toward target
			dir = to_target.normalized()

		velocity.x = dir.x * spd
		velocity.z = dir.z * spd
	else:
		velocity.x = 0.0
		velocity.z = 0.0


func _process_melee_telegraph() -> void:
	velocity.x = 0.0
	velocity.z = 0.0

	if _target_player and is_instance_valid(_target_player):
		var to_target := _target_player.global_position - global_position
		to_target.y = 0.0
		if to_target.length() > 0.1:
			look_at(global_position + to_target.normalized(), Vector3.UP)

	if _state_timer <= 0.0:
		_change_state(State.MELEE_ATTACK)


func _process_melee_attack() -> void:
	CombatLog.log_boss_attack("melee", _current_phase, global_position, _target_player.global_position if _target_player and is_instance_valid(_target_player) else global_position)
	var hit_any := false
	for player in GameManager.players:
		if not is_instance_valid(player) or not player.visible:
			continue
		var dist := global_position.distance_to(player.global_position)
		if dist <= melee_range:
			var hit_dir := (player.global_position - global_position).normalized()
			player.take_damage(_get_melee_damage(), global_position + hit_dir)
			CombatLog.log_boss_hit("melee", _get_melee_damage(), player.name, player.global_position)
			hit_any = true
	if not hit_any:
		CombatLog.log_boss_miss("melee")

	_change_state(State.COOLDOWN)


func _process_ranged_telegraph() -> void:
	velocity.x = 0.0
	velocity.z = 0.0

	if _target_player and is_instance_valid(_target_player):
		_ranged_target_position = _target_player.global_position + Vector3(0.0, 1.0, 0.0)
		_update_laser_warning()

	if _state_timer <= 0.0:
		_change_state(State.RANGED_ATTACK)


func _process_ranged_attack() -> void:
	# Cancel if no LOS at fire time (player ducked behind cover during telegraph)
	if _target_player and is_instance_valid(_target_player) and not _has_line_of_sight(_target_player.global_position):
		CombatLog.log_boss_miss("ranged_no_los")
		_change_state(State.COOLDOWN)
		return

	CombatLog.log_boss_attack("ranged_x%d" % _get_ranged_burst_count(), _current_phase, global_position, _ranged_target_position)
	var count := _get_ranged_burst_count()
	var spread_angle := deg_to_rad(5.0)

	for i in count:
		var offset := (i - (count - 1) / 2.0) * spread_angle
		_fire_projectile_with_offset(offset)

	_change_state(State.COOLDOWN)


func _process_aoe_telegraph() -> void:
	velocity.x = 0.0
	velocity.z = 0.0

	# Cosmetic: lift body mesh during telegraph
	var total_time := _get_aoe_telegraph_time()
	var lift_progress := 1.0 - (_state_timer / total_time) if total_time > 0.0 else 1.0
	body_mesh.position.y = 1.0 + lift_progress * 2.0

	# Scale AoE indicator to current phase radius
	var radius := _get_aoe_radius()
	_aoe_telegraph_mesh.mesh.size = Vector2(radius * 2.0, radius * 2.0)

	if _state_timer <= 0.0:
		_change_state(State.AOE_SLAM)


func _process_aoe_slam() -> void:
	body_mesh.position.y = 1.0  # snap back down
	velocity.x = 0.0
	velocity.z = 0.0

	var radius := _get_aoe_radius()
	var damage := _get_aoe_damage()
	CombatLog.log_boss_attack("aoe_slam", _current_phase, global_position, global_position)
	var hit_any := false
	for player in GameManager.players:
		if not is_instance_valid(player) or not player.visible:
			continue
		var dist := global_position.distance_to(player.global_position)
		if dist <= radius:
			player.take_damage(damage, global_position)
			CombatLog.log_boss_hit("aoe_slam", damage, player.name, player.global_position)
			hit_any = true
	if not hit_any:
		CombatLog.log_boss_miss("aoe_slam")

	_change_state(State.COOLDOWN)


func _process_charge_telegraph() -> void:
	velocity.x = 0.0
	velocity.z = 0.0

	if _target_player and is_instance_valid(_target_player):
		var to_target := _target_player.global_position - global_position
		to_target.y = 0.0
		if to_target.length() > 0.1:
			_charge_direction = to_target.normalized()
			look_at(global_position + _charge_direction, Vector3.UP)
	_update_charge_indicator()

	if _state_timer <= 0.0:
		_change_state(State.CHARGE)


func _process_charge(delta: float) -> void:
	# Log on first frame — cancel if no LOS to target player
	if _charge_distance_traveled == 0.0:
		var target_pos := global_position + _charge_direction * _get_charge_max_distance()
		if _target_player and is_instance_valid(_target_player) and not _has_line_of_sight(_target_player.global_position):
			CombatLog.log_boss_miss("charge_no_los")
			velocity.x = 0.0
			velocity.z = 0.0
			_change_state(State.COOLDOWN)
			return
		CombatLog.log_boss_attack("charge", _current_phase, global_position, target_pos)
	var spd := _get_charge_speed()
	velocity.x = _charge_direction.x * spd
	velocity.z = _charge_direction.z * spd
	_charge_distance_traveled += spd * delta

	# Hit players along the path (no double-hit)
	for player in GameManager.players:
		if not is_instance_valid(player) or not player.visible or player in _charge_hit_players:
			continue
		var dist := global_position.distance_to(player.global_position)
		if dist <= 2.0:
			player.take_damage(_get_charge_damage(), global_position)
			CombatLog.log_boss_hit("charge", _get_charge_damage(), player.name, player.global_position)
			_charge_hit_players.append(player)

	# Stop conditions
	if _charge_distance_traveled >= _get_charge_max_distance() or is_on_wall():
		if _charge_hit_players.is_empty():
			CombatLog.log_boss_miss("charge")
		velocity.x = 0.0
		velocity.z = 0.0
		_change_state(State.COOLDOWN)


func _process_cooldown() -> void:
	velocity.x = 0.0
	velocity.z = 0.0
	if _state_timer <= 0.0:
		_chase_timer = 0.0
		_change_state(State.CHASE)


func _process_phase_transition() -> void:
	velocity.x = 0.0
	velocity.z = 0.0
	if _state_timer <= 0.0:
		_change_state(State.CHASE)


# =============================================================================
# State management
# =============================================================================

func _change_state(new_state: State) -> void:
	# Hide all telegraphs
	_melee_telegraph_mesh.visible = false
	_laser_warning_mesh.visible = false
	_aoe_telegraph_mesh.visible = false
	_charge_telegraph_mesh.visible = false

	state = new_state

	match new_state:
		State.CHASE:
			_chase_timer = 0.0
		State.MELEE_TELEGRAPH:
			_state_timer = _get_melee_telegraph_time()
			_melee_telegraph_mesh.visible = true
		State.MELEE_ATTACK:
			_state_timer = 0.1
		State.RANGED_TELEGRAPH:
			_state_timer = _get_ranged_telegraph_time()
			_laser_warning_mesh.visible = true
			if _target_player and is_instance_valid(_target_player):
				_ranged_target_position = _target_player.global_position + Vector3(0.0, 1.0, 0.0)
		State.RANGED_ATTACK:
			_state_timer = 0.1
		State.AOE_TELEGRAPH:
			_state_timer = _get_aoe_telegraph_time()
			_aoe_telegraph_mesh.visible = true
		State.AOE_SLAM:
			_state_timer = 0.1
		State.CHARGE_TELEGRAPH:
			_state_timer = _get_charge_telegraph_time()
			_charge_telegraph_mesh.visible = true
			_charge_direction = Vector3.ZERO
		State.CHARGE:
			_charge_distance_traveled = 0.0
			_charge_hit_players.clear()
			if _charge_direction.length() < 0.1:
				# Fallback: charge forward
				_charge_direction = -global_transform.basis.z.normalized()
		State.COOLDOWN:
			_state_timer = _get_cooldown_time()
			body_mesh.position.y = 1.0  # ensure body is grounded after AoE
		State.PHASE_TRANSITION:
			_state_timer = 1.5


# =============================================================================
# Damage & Phase transitions
# =============================================================================

func take_damage(amount: float, _hit_position: Vector3 = Vector3.ZERO) -> void:
	if state == State.DEAD or state == State.PHASE_TRANSITION:
		return
	health -= amount
	health = maxf(health, 0.0)
	CombatLog.log_player_damage(amount, global_position, State.keys()[state])
	_update_health_bar()
	_show_damage_flash()

	if health <= 0.0:
		_die()
		return

	# Check phase thresholds
	var hp_ratio := health / max_health
	if hp_ratio <= 0.3 and 3 not in _phase_transitioned:
		_enter_phase(3)
	elif hp_ratio <= 0.6 and 2 not in _phase_transitioned:
		_enter_phase(2)


func _enter_phase(phase: int) -> void:
	_current_phase = phase
	_phase_transitioned.append(phase)
	CombatLog.log_phase_transition(phase, health, CombatLog._elapsed())
	_change_state(State.PHASE_TRANSITION)

	# Visual feedback
	_phase_body_material = StandardMaterial3D.new()
	if phase == 2:
		_phase_body_material.albedo_color = Color(0.9, 0.4, 0.1)
		_phase_body_material.emission_enabled = true
		_phase_body_material.emission = Color(0.9, 0.3, 0.05)
		_phase_body_material.emission_energy_multiplier = 1.0
	elif phase == 3:
		_phase_body_material.albedo_color = Color(1.0, 0.1, 0.1)
		_phase_body_material.emission_enabled = true
		_phase_body_material.emission = Color(1.0, 0.1, 0.05)
		_phase_body_material.emission_energy_multiplier = 2.5
	body_mesh.set_surface_override_material(0, _phase_body_material)
	_default_body_material = _phase_body_material

	# Health bar color
	_update_health_bar_color()


func _die() -> void:
	_change_state(State.DEAD)
	visible = false
	collision_layer = 0
	died.emit()


# =============================================================================
# Damage flash
# =============================================================================

func _show_damage_flash() -> void:
	_flash_timer = 0.1
	var flash_mat := StandardMaterial3D.new()
	flash_mat.albedo_color = Color(1.0, 0.3, 0.3)
	flash_mat.emission_enabled = true
	flash_mat.emission = Color(1.0, 0.2, 0.2)
	flash_mat.emission_energy_multiplier = 2.0
	body_mesh.set_surface_override_material(0, flash_mat)


func _update_flash(delta: float) -> void:
	if _flash_timer > 0.0:
		_flash_timer -= delta
		if _flash_timer <= 0.0:
			body_mesh.set_surface_override_material(0, _default_body_material)


# =============================================================================
# Health bar (billboard quad above head)
# =============================================================================

func _create_health_bar() -> void:
	_health_bar_pivot = Node3D.new()
	_health_bar_pivot.top_level = true
	add_child(_health_bar_pivot)

	# Background bar
	var bg := MeshInstance3D.new()
	var bg_mesh := QuadMesh.new()
	bg_mesh.size = Vector2(1.6, 0.18)
	bg.mesh = bg_mesh
	var bg_mat := StandardMaterial3D.new()
	bg_mat.albedo_color = Color(0.1, 0.1, 0.1, 0.9)
	bg_mat.transparency = BaseMaterial3D.TRANSPARENCY_ALPHA
	bg_mat.shading_mode = BaseMaterial3D.SHADING_MODE_UNSHADED
	bg_mat.billboard_mode = BaseMaterial3D.BILLBOARD_ENABLED
	bg_mat.no_depth_test = true
	bg_mat.render_priority = 0
	bg.set_surface_override_material(0, bg_mat)
	_health_bar_pivot.add_child(bg)

	# Foreground bar (green fill)
	_health_bar_fg = MeshInstance3D.new()
	var fg_mesh := QuadMesh.new()
	fg_mesh.size = Vector2(1.5, 0.12)
	_health_bar_fg.mesh = fg_mesh
	var fg_mat := StandardMaterial3D.new()
	fg_mat.albedo_color = Color(0.15, 0.85, 0.15)
	fg_mat.shading_mode = BaseMaterial3D.SHADING_MODE_UNSHADED
	fg_mat.billboard_mode = BaseMaterial3D.BILLBOARD_ENABLED
	fg_mat.no_depth_test = true
	fg_mat.render_priority = 1
	_health_bar_fg.set_surface_override_material(0, fg_mat)
	_health_bar_pivot.add_child(_health_bar_fg)


func _update_health_bar() -> void:
	if not _health_bar_fg:
		return
	var ratio := health / max_health
	(_health_bar_fg.mesh as QuadMesh).size.x = 1.5 * maxf(ratio, 0.01)


func _update_health_bar_color() -> void:
	if not _health_bar_fg:
		return
	var mat := _health_bar_fg.get_surface_override_material(0) as StandardMaterial3D
	if not mat:
		return
	match _current_phase:
		2:
			mat.albedo_color = Color(1.0, 0.6, 0.1)
		3:
			mat.albedo_color = Color(0.9, 0.15, 0.15)
			mat.emission_enabled = true
			mat.emission = Color(0.9, 0.1, 0.1)
			mat.emission_energy_multiplier = 1.0


func _face_health_bar_to_camera() -> void:
	_health_bar_pivot.global_position = global_position + Vector3(0.0, 2.5, 0.0)


# =============================================================================
# Telegraph visuals
# =============================================================================

func _create_melee_telegraph() -> void:
	_melee_telegraph_mesh = MeshInstance3D.new()
	var mesh := PlaneMesh.new()
	mesh.size = Vector2(melee_range * 2.0, melee_range * 2.0)
	mesh.subdivide_width = 32
	mesh.subdivide_depth = 32
	_melee_telegraph_mesh.mesh = mesh
	var mat := ShaderMaterial.new()
	mat.shader = _create_circle_shader()
	mat.set_shader_parameter("color", Color(1.0, 0.1, 0.1, 0.45))
	mat.set_shader_parameter("edge_color", Color(1.0, 0.2, 0.1, 0.9))
	mat.set_shader_parameter("edge_width", 0.08)
	_melee_telegraph_mesh.set_surface_override_material(0, mat)
	_melee_telegraph_mesh.visible = false
	_melee_telegraph_mesh.position = Vector3(0.0, 0.02, 0.0)
	add_child(_melee_telegraph_mesh)


func _create_aoe_telegraph() -> void:
	_aoe_telegraph_mesh = MeshInstance3D.new()
	var mesh := PlaneMesh.new()
	mesh.size = Vector2(10.0, 10.0)  # will be resized per phase
	mesh.subdivide_width = 32
	mesh.subdivide_depth = 32
	_aoe_telegraph_mesh.mesh = mesh
	var mat := ShaderMaterial.new()
	mat.shader = _create_circle_shader()
	mat.set_shader_parameter("color", Color(1.0, 0.3, 0.0, 0.35))
	mat.set_shader_parameter("edge_color", Color(1.0, 0.5, 0.0, 0.9))
	mat.set_shader_parameter("edge_width", 0.06)
	_aoe_telegraph_mesh.set_surface_override_material(0, mat)
	_aoe_telegraph_mesh.visible = false
	_aoe_telegraph_mesh.position = Vector3(0.0, 0.03, 0.0)
	add_child(_aoe_telegraph_mesh)


func _create_circle_shader() -> Shader:
	var shader := Shader.new()
	shader.code = "
shader_type spatial;
render_mode unshaded, cull_disabled;

uniform vec4 color : source_color = vec4(1.0, 0.1, 0.1, 0.45);
uniform vec4 edge_color : source_color = vec4(1.0, 0.2, 0.1, 0.9);
uniform float edge_width : hint_range(0.0, 0.2) = 0.08;

void fragment() {
	vec2 center_uv = UV * 2.0 - 1.0;
	float dist = length(center_uv);
	if (dist > 1.0) {
		discard;
	}
	float edge_inner = 1.0 - edge_width;
	float t = smoothstep(edge_inner - 0.02, edge_inner, dist);
	ALBEDO = mix(color.rgb, edge_color.rgb, t);
	ALPHA = mix(color.a, edge_color.a, t);
}
"
	return shader


func _create_laser_warning() -> void:
	_laser_warning_mesh = MeshInstance3D.new()
	var mesh := BoxMesh.new()
	mesh.size = Vector3(0.15, 0.15, 1.0)  # thicker laser for visibility
	_laser_warning_mesh.mesh = mesh
	var mat := StandardMaterial3D.new()
	mat.albedo_color = Color(1.0, 0.0, 0.0, 0.9)
	mat.emission_enabled = true
	mat.emission = Color(1.0, 0.1, 0.1)
	mat.emission_energy_multiplier = 5.0
	mat.shading_mode = BaseMaterial3D.SHADING_MODE_UNSHADED
	_laser_warning_mesh.set_surface_override_material(0, mat)
	_laser_warning_mesh.visible = false
	_laser_warning_mesh.top_level = true
	add_child(_laser_warning_mesh)


func _create_charge_telegraph() -> void:
	_charge_telegraph_mesh = MeshInstance3D.new()
	var mesh := BoxMesh.new()
	mesh.size = Vector3(0.6, 0.02, 1.0)  # wide flat line on ground
	_charge_telegraph_mesh.mesh = mesh
	var mat := StandardMaterial3D.new()
	mat.albedo_color = Color(1.0, 0.5, 0.0, 0.7)
	mat.emission_enabled = true
	mat.emission = Color(1.0, 0.4, 0.0)
	mat.emission_energy_multiplier = 2.0
	mat.shading_mode = BaseMaterial3D.SHADING_MODE_UNSHADED
	_charge_telegraph_mesh.set_surface_override_material(0, mat)
	_charge_telegraph_mesh.visible = false
	_charge_telegraph_mesh.top_level = true
	add_child(_charge_telegraph_mesh)


func _update_laser_warning() -> void:
	var start := global_position + Vector3(0.0, 1.0, 0.0)
	var end := _ranged_target_position
	var mid := (start + end) / 2.0
	var dist := start.distance_to(end)

	_laser_warning_mesh.global_position = mid
	_laser_warning_mesh.scale = Vector3(1.0, 1.0, dist)
	if dist > 0.1:
		_laser_warning_mesh.look_at(end, Vector3.UP)


func _update_charge_indicator() -> void:
	if _charge_direction.length() < 0.1:
		return
	var start := global_position + Vector3(0.0, 0.05, 0.0)
	var max_dist := _get_charge_max_distance()
	var end := start + _charge_direction * max_dist
	var mid := (start + end) / 2.0
	mid.y = 0.05

	_charge_telegraph_mesh.global_position = mid
	_charge_telegraph_mesh.scale = Vector3(1.0, 1.0, max_dist)
	if max_dist > 0.1:
		_charge_telegraph_mesh.look_at(end, Vector3.UP)


# =============================================================================
# Projectiles
# =============================================================================

func _fire_projectile_with_offset(angle_offset: float) -> void:
	if not _projectile_scene:
		return
	var projectile: Node3D = _projectile_scene.instantiate()
	get_tree().current_scene.add_child(projectile)
	var spawn_pos := global_position + Vector3(0.0, 1.0, 0.0)
	projectile.global_position = spawn_pos
	var base_direction := (_ranged_target_position - spawn_pos).normalized()
	# Apply horizontal rotation offset for spread
	var rotated := base_direction.rotated(Vector3.UP, angle_offset)
	projectile.setup(rotated, _get_ranged_per_projectile_damage())
