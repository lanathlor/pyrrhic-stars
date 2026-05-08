extends CharacterBody3D

## Visual puppet for the arena boss.
## Receives authoritative state from the server via apply_server_state().
## Contains NO game logic — only interpolation, telegraph visuals, animations.

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

var _melee_cone_angle: float = PI  # full cone angle in radians (default 180°)
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
# Dynamic visual nodes
var _melee_telegraph_mesh: MeshInstance3D
var _laser_warning_mesh: MeshInstance3D
var _aoe_telegraph_mesh: MeshInstance3D
var _charge_telegraph_mesh: MeshInstance3D
var _health_bar_pivot: Node3D
var _health_bar_fg: MeshInstance3D
# AoE fire particles
var _aoe_particles: GPUParticles3D
var _aoe_slam_particles: GPUParticles3D
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


func _ready() -> void:
	health = max_health
	_server_position = global_position
	_server_rotation_y = rotation.y
	GameManager.register_enemy(self)

	_create_health_bar()
	_create_melee_telegraph()
	_create_laser_warning()
	_create_aoe_telegraph()
	_create_aoe_particles()
	_create_charge_telegraph()
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
	_face_health_bar_to_camera()
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
	_update_health_bar()
	_update_health_bar_color()
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
		_melee_telegraph_mesh.visible = false
		_laser_warning_mesh.visible = false
		_aoe_telegraph_mesh.visible = false
		_charge_telegraph_mesh.visible = false
		if _aoe_particles:
			_aoe_particles.emitting = false
		# 2=MeleeTelegraph, 4=RangedTelegraph, 6=AoETelegraph,
		# 7=AoESlam, 8=ChargeTelegraph, 12=Dead
		match synced_state:
			2:  # MELEE_TELEGRAPH
				_update_melee_telegraph_params()
				_melee_telegraph_mesh.visible = true
			4:  # RANGED_TELEGRAPH
				_laser_warning_mesh.visible = true
			6:  # AOE_TELEGRAPH
				_aoe_telegraph_mesh.visible = true
				if _aoe_particles:
					_aoe_particles.emitting = true
			7:  # AOE_SLAM
				if _aoe_slam_particles:
					_aoe_slam_particles.emitting = true
			8:  # CHARGE_TELEGRAPH
				_charge_telegraph_mesh.visible = true
			12:  # DEAD
				visible = false
		state = synced_state as State
	# Update telegraph positions for synced data
	if _laser_warning_mesh.visible:
		_update_laser_warning()
	if _charge_telegraph_mesh.visible:
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
	_health_bar_pivot.global_position = global_position + Vector3(0.0, 3.0, 0.0)


# =============================================================================
# Telegraph visuals
# =============================================================================


func _update_melee_telegraph_params() -> void:
	# Update mesh size to match active melee range
	var mesh := _melee_telegraph_mesh.mesh as PlaneMesh
	if mesh:
		mesh.size = Vector2(melee_range * 2.0, melee_range * 2.0)
	# Update cone half-angle
	var mat := _melee_telegraph_mesh.get_surface_override_material(0) as ShaderMaterial
	if mat:
		mat.set_shader_parameter("half_angle", _melee_cone_angle / 2.0)


func _create_melee_telegraph() -> void:
	_melee_telegraph_mesh = MeshInstance3D.new()
	var mesh := PlaneMesh.new()
	mesh.size = Vector2(melee_range * 2.0, melee_range * 2.0)
	mesh.subdivide_width = 32
	mesh.subdivide_depth = 32
	_melee_telegraph_mesh.mesh = mesh
	var mat := ShaderMaterial.new()
	mat.shader = _create_cone_shader()
	mat.set_shader_parameter("color", Color(1.0, 0.1, 0.1, 0.45))
	mat.set_shader_parameter("edge_color", Color(1.0, 0.2, 0.1, 0.9))
	mat.set_shader_parameter("edge_width", 0.08)
	mat.set_shader_parameter("half_angle", deg_to_rad(90.0))  # 180° cone = 90° half
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
	shader.code = """
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
"""
	return shader


func _create_cone_shader() -> Shader:
	var shader := Shader.new()
	shader.code = """
shader_type spatial;
render_mode unshaded, cull_disabled;

uniform vec4 color : source_color = vec4(1.0, 0.1, 0.1, 0.45);
uniform vec4 edge_color : source_color = vec4(1.0, 0.2, 0.1, 0.9);
uniform float edge_width : hint_range(0.0, 0.2) = 0.08;
uniform float half_angle : hint_range(0.0, 3.14159) = 1.5708; // radians

void fragment() {
	vec2 uv = UV * 2.0 - 1.0;
	float dist = length(uv);
	if (dist > 1.0) {
		discard;
	}
	// Cone check: forward is -Z in local space = -Y in centered UV
	float angle = acos(clamp(-uv.y / max(dist, 0.001), -1.0, 1.0));
	if (angle > half_angle) {
		discard;
	}
	// Edge glow at outer radius
	float edge_inner = 1.0 - edge_width;
	float t = smoothstep(edge_inner - 0.02, edge_inner, dist);
	// Edge glow at cone boundary
	float cone_edge = smoothstep(half_angle - 0.12, half_angle, angle);
	t = max(t, cone_edge);
	ALBEDO = mix(color.rgb, edge_color.rgb, t);
	ALPHA = mix(color.a, edge_color.a, t);
}
"""
	return shader


func _create_fire_shader() -> Shader:
	var shader := Shader.new()
	shader.code = """
shader_type spatial;
render_mode unshaded, blend_add, cull_disabled, depth_draw_never;

// Procedural flame particle shader
// Noise-based alpha mask, UV distortion, fire color ramp, soft edges

varying flat float v_seed;

void vertex() {
	// Billboard: extract scale, rebuild modelview facing camera
	float s_x = length(MODEL_MATRIX[0].xyz);
	float s_y = length(MODEL_MATRIX[1].xyz);
	float s_z = length(MODEL_MATRIX[2].xyz);
	mat4 bill = mat4(
		vec4(VIEW_MATRIX[0][0], VIEW_MATRIX[1][0], VIEW_MATRIX[2][0], 0.0) * s_x,
		vec4(VIEW_MATRIX[0][1], VIEW_MATRIX[1][1], VIEW_MATRIX[2][1], 0.0) * s_y,
		vec4(VIEW_MATRIX[0][2], VIEW_MATRIX[1][2], VIEW_MATRIX[2][2], 0.0) * s_z,
		MODEL_MATRIX[3]
	);
	MODELVIEW_MATRIX = VIEW_MATRIX * bill;
	MODELVIEW_NORMAL_MATRIX = mat3(MODELVIEW_MATRIX);

	// Per-instance seed for variation between particles
	v_seed = COLOR.r * 7.3 + COLOR.g * 13.1 + float(INSTANCE_ID) * 1.37;
}

// Hash-based noise
float hash(vec2 p) {
	return fract(sin(dot(p, vec2(127.1, 311.7))) * 43758.5453);
}

float noise(vec2 p) {
	vec2 i = floor(p);
	vec2 f = fract(p);
	f = f * f * (3.0 - 2.0 * f); // smoothstep
	float a = hash(i);
	float b = hash(i + vec2(1.0, 0.0));
	float c = hash(i + vec2(0.0, 1.0));
	float d = hash(i + vec2(1.0, 1.0));
	return mix(mix(a, b, f.x), mix(c, d, f.x), f.y);
}

float fbm(vec2 p) {
	float v = 0.0;
	float a = 0.5;
	for (int i = 0; i < 4; i++) {
		v += a * noise(p);
		p *= 2.2;
		a *= 0.5;
	}
	return v;
}

void fragment() {
	vec2 uv = UV * 2.0 - 1.0; // center UV [-1, 1]

	// Radial distance from center
	float dist = length(uv);

	// Soft circular mask
	float circle = 1.0 - smoothstep(0.3, 1.0, dist);

	// Scroll UVs upward for flame lick effect
	float t = TIME * 3.0 + v_seed;
	vec2 flame_uv = uv * 2.5;
	flame_uv.y -= t;  // scroll up

	// Distort UVs with noise for organic movement
	float distort = fbm(flame_uv * 1.5 + vec2(v_seed * 0.3, t * 0.5)) * 0.6;
	flame_uv += distort;

	// Main flame noise
	float flame = fbm(flame_uv);

	// Shape: stronger at bottom, tapers at top
	float shape = smoothstep(1.0, -0.5, uv.y); // bright at bottom, fades at top
	flame *= shape;

	// Combine with circle mask
	float alpha = flame * circle;
	alpha = smoothstep(0.05, 0.5, alpha);

	// Fire color ramp based on intensity
	// Hot core (white-yellow) -> mid (orange) -> cool (red-black)
	vec3 col_hot = vec3(1.0, 0.95, 0.8);   // white-yellow core
	vec3 col_mid = vec3(1.0, 0.45, 0.05);   // orange
	vec3 col_cool = vec3(0.6, 0.08, 0.01);  // deep red
	vec3 col_smoke = vec3(0.15, 0.02, 0.0); // almost black

	float intensity = alpha;
	vec3 fire_color;
	if (intensity > 0.7) {
		fire_color = mix(col_mid, col_hot, (intensity - 0.7) / 0.3);
	} else if (intensity > 0.4) {
		fire_color = mix(col_cool, col_mid, (intensity - 0.4) / 0.3);
	} else {
		fire_color = mix(col_smoke, col_cool, intensity / 0.4);
	}

	// Multiply by vertex color (particle color ramp over lifetime)
	fire_color *= COLOR.rgb;

	// HDR emission for bloom
	ALBEDO = fire_color * (2.0 + intensity * 4.0);
	ALPHA = alpha * COLOR.a;
}
"""
	return shader


func _create_aoe_particles() -> void:
	var fire_shader := _create_fire_shader()

	# --- Charging fire particles (emitted during telegraph, ramp up) ---
	_aoe_particles = GPUParticles3D.new()
	_aoe_particles.amount = 150
	_aoe_particles.lifetime = 1.0
	_aoe_particles.emitting = false
	_aoe_particles.position = Vector3(0.0, 0.3, 0.0)

	var mat := ParticleProcessMaterial.new()
	mat.emission_shape = ParticleProcessMaterial.EMISSION_SHAPE_SPHERE
	mat.emission_sphere_radius = 0.3
	mat.direction = Vector3(0.0, 1.0, 0.0)
	mat.spread = 45.0
	mat.initial_velocity_min = 1.5
	mat.initial_velocity_max = 3.0
	mat.gravity = Vector3(0.0, 3.0, 0.0)  # fire rises
	mat.scale_min = 0.3
	mat.scale_max = 0.8
	mat.angular_velocity_min = -90.0
	mat.angular_velocity_max = 90.0
	# Lifetime color: fade out alpha over life
	var color_ramp := Gradient.new()
	color_ramp.set_color(0, Color(1.0, 1.0, 0.9, 1.0))
	color_ramp.add_point(0.3, Color(1.0, 0.7, 0.3, 0.9))
	color_ramp.add_point(0.7, Color(0.8, 0.2, 0.05, 0.6))
	color_ramp.set_color(1, Color(0.2, 0.02, 0.0, 0.0))
	var color_texture := GradientTexture1D.new()
	color_texture.gradient = color_ramp
	mat.color_ramp = color_texture
	# Scale curve: grow then shrink
	var scale_curve := CurveTexture.new()
	var curve := Curve.new()
	curve.add_point(Vector2(0.0, 0.3))
	curve.add_point(Vector2(0.3, 1.0))
	curve.add_point(Vector2(0.7, 0.8))
	curve.add_point(Vector2(1.0, 0.1))
	scale_curve.curve = curve
	mat.scale_curve = scale_curve
	_aoe_particles.process_material = mat

	var draw_mesh := QuadMesh.new()
	draw_mesh.size = Vector2(0.6, 0.8)  # taller than wide for flame shape
	_aoe_particles.draw_pass_1 = draw_mesh
	var fire_mat := ShaderMaterial.new()
	fire_mat.shader = fire_shader
	draw_mesh.material = fire_mat

	add_child(_aoe_particles)

	# --- Slam burst: dense fireball expanding outward ---
	_aoe_slam_particles = GPUParticles3D.new()
	_aoe_slam_particles.amount = 512
	_aoe_slam_particles.lifetime = 1.0
	_aoe_slam_particles.one_shot = true
	_aoe_slam_particles.explosiveness = 1.0  # all at once
	_aoe_slam_particles.emitting = false
	_aoe_slam_particles.position = Vector3(0.0, 0.5, 0.0)

	var slam_mat := ParticleProcessMaterial.new()
	slam_mat.emission_shape = ParticleProcessMaterial.EMISSION_SHAPE_SPHERE
	slam_mat.emission_sphere_radius = 1.0  # start from a wider core
	slam_mat.direction = Vector3(0.0, 0.0, 0.0)  # no bias — pure radial
	slam_mat.spread = 180.0
	slam_mat.initial_velocity_min = 4.0  # slow — keeps the ball dense
	slam_mat.initial_velocity_max = 10.0  # some faster ones at the front edge
	slam_mat.gravity = Vector3(0.0, 0.5, 0.0)  # slight upward mushroom drift
	slam_mat.damping_min = 2.0
	slam_mat.damping_max = 4.0
	slam_mat.scale_min = 2.0  # huge overlapping quads = solid mass
	slam_mat.scale_max = 4.0
	slam_mat.angular_velocity_min = -120.0
	slam_mat.angular_velocity_max = 120.0
	# Color: white-hot flash -> yellow -> orange -> red -> black
	var slam_ramp := Gradient.new()
	slam_ramp.set_color(0, Color(1.0, 1.0, 0.95, 1.0))  # white flash
	slam_ramp.add_point(0.08, Color(1.0, 0.9, 0.4, 1.0))  # bright yellow
	slam_ramp.add_point(0.25, Color(1.0, 0.5, 0.08, 0.95))  # orange
	slam_ramp.add_point(0.5, Color(0.8, 0.2, 0.03, 0.7))  # red-orange
	slam_ramp.add_point(0.75, Color(0.4, 0.06, 0.01, 0.35))  # dark red
	slam_ramp.set_color(1, Color(0.08, 0.01, 0.0, 0.0))  # fade to nothing
	var slam_color_tex := GradientTexture1D.new()
	slam_color_tex.gradient = slam_ramp
	slam_mat.color_ramp = slam_color_tex
	# Scale curve: start big, hold, then shrink — keeps ball solid longer
	var slam_scale_curve := CurveTexture.new()
	var slam_curve := Curve.new()
	slam_curve.add_point(Vector2(0.0, 0.6))
	slam_curve.add_point(Vector2(0.1, 1.0))
	slam_curve.add_point(Vector2(0.4, 0.9))
	slam_curve.add_point(Vector2(0.7, 0.5))
	slam_curve.add_point(Vector2(1.0, 0.05))
	slam_scale_curve.curve = slam_curve
	slam_mat.scale_curve = slam_scale_curve
	_aoe_slam_particles.process_material = slam_mat

	var slam_mesh := QuadMesh.new()
	slam_mesh.size = Vector2(1.5, 1.5)  # large base quad
	_aoe_slam_particles.draw_pass_1 = slam_mesh
	var slam_fire_mat := ShaderMaterial.new()
	slam_fire_mat.shader = fire_shader
	slam_mesh.material = slam_fire_mat

	add_child(_aoe_slam_particles)


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


# Charge max distance per phase (needed for telegraph visual only)
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
		State.MELEE_TELEGRAPH, State.MELEE_ATTACK, State.CHARGE_TELEGRAPH, State.CHARGE, State.AOE_TELEGRAPH, State.AOE_SLAM:
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
