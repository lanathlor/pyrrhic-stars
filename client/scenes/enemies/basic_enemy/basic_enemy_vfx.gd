extends Node

## Telegraph visuals, health bar, shaders, and particle effects for BasicEnemy.
## Attached as a child node. References the parent enemy via `enemy`.

const _FIRE_SHADER_CODE := """
shader_type spatial;
render_mode unshaded, blend_add, cull_disabled, depth_draw_never;

varying flat float v_seed;

void vertex() {
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
	v_seed = COLOR.r * 7.3 + COLOR.g * 13.1 + float(INSTANCE_ID) * 1.37;
}

float hash(vec2 p) {
	return fract(sin(dot(p, vec2(127.1, 311.7))) * 43758.5453);
}

float noise(vec2 p) {
	vec2 i = floor(p);
	vec2 f = fract(p);
	f = f * f * (3.0 - 2.0 * f);
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
	vec2 uv = UV * 2.0 - 1.0;
	float dist = length(uv);
	float circle = 1.0 - smoothstep(0.3, 1.0, dist);
	float t = TIME * 3.0 + v_seed;
	vec2 flame_uv = uv * 2.5;
	flame_uv.y -= t;
	float distort = fbm(flame_uv * 1.5 + vec2(v_seed * 0.3, t * 0.5)) * 0.6;
	flame_uv += distort;
	float flame = fbm(flame_uv);
	float shape = smoothstep(1.0, -0.5, uv.y);
	flame *= shape;
	float alpha = flame * circle;
	alpha = smoothstep(0.05, 0.5, alpha);
	vec3 col_hot = vec3(1.0, 0.95, 0.8);
	vec3 col_mid = vec3(1.0, 0.45, 0.05);
	vec3 col_cool = vec3(0.6, 0.08, 0.01);
	vec3 col_smoke = vec3(0.15, 0.02, 0.0);
	float intensity = alpha;
	vec3 fire_color;
	if (intensity > 0.7) {
		fire_color = mix(col_mid, col_hot, (intensity - 0.7) / 0.3);
	} else if (intensity > 0.4) {
		fire_color = mix(col_cool, col_mid, (intensity - 0.4) / 0.3);
	} else {
		fire_color = mix(col_smoke, col_cool, intensity / 0.4);
	}
	fire_color *= COLOR.rgb;
	ALBEDO = fire_color * (2.0 + intensity * 4.0);
	ALPHA = alpha * COLOR.a;
}
"""

var enemy: CharacterBody3D

# Dynamic visual nodes
var melee_telegraph_mesh: MeshInstance3D
var laser_warning_mesh: MeshInstance3D
var aoe_telegraph_mesh: MeshInstance3D
var charge_telegraph_mesh: MeshInstance3D
var health_bar_pivot: Node3D
var health_bar_fg: MeshInstance3D
var aoe_particles: GPUParticles3D
var aoe_slam_particles: GPUParticles3D


func _ready() -> void:
	enemy = get_parent()


# =============================================================================
# Health bar (billboard quad above head)
# =============================================================================


func create_health_bar() -> void:
	health_bar_pivot = Node3D.new()
	health_bar_pivot.top_level = true
	enemy.add_child(health_bar_pivot)

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
	health_bar_pivot.add_child(bg)

	# Foreground bar (green fill)
	health_bar_fg = MeshInstance3D.new()
	var fg_mesh := QuadMesh.new()
	fg_mesh.size = Vector2(1.5, 0.12)
	health_bar_fg.mesh = fg_mesh
	var fg_mat := StandardMaterial3D.new()
	fg_mat.albedo_color = Color(0.15, 0.85, 0.15)
	fg_mat.shading_mode = BaseMaterial3D.SHADING_MODE_UNSHADED
	fg_mat.billboard_mode = BaseMaterial3D.BILLBOARD_ENABLED
	fg_mat.no_depth_test = true
	fg_mat.render_priority = 1
	health_bar_fg.set_surface_override_material(0, fg_mat)
	health_bar_pivot.add_child(health_bar_fg)


func update_health_bar() -> void:
	if not health_bar_fg:
		return
	var ratio := enemy.health / enemy.max_health
	(health_bar_fg.mesh as QuadMesh).size.x = 1.5 * maxf(ratio, 0.01)


func update_health_bar_color(current_phase: int) -> void:
	if not health_bar_fg:
		return
	var mat := health_bar_fg.get_surface_override_material(0) as StandardMaterial3D
	if not mat:
		return
	match current_phase:
		2:
			mat.albedo_color = Color(1.0, 0.6, 0.1)
		3:
			mat.albedo_color = Color(0.9, 0.15, 0.15)
			mat.emission_enabled = true
			mat.emission = Color(0.9, 0.1, 0.1)
			mat.emission_energy_multiplier = 1.0


func face_health_bar_to_camera() -> void:
	health_bar_pivot.global_position = enemy.global_position + Vector3(0.0, 3.0, 0.0)


# =============================================================================
# Melee telegraph
# =============================================================================


func create_melee_telegraph(melee_range: float) -> void:
	melee_telegraph_mesh = MeshInstance3D.new()
	var mesh := PlaneMesh.new()
	mesh.size = Vector2(melee_range * 2.0, melee_range * 2.0)
	mesh.subdivide_width = 32
	mesh.subdivide_depth = 32
	melee_telegraph_mesh.mesh = mesh
	var mat := ShaderMaterial.new()
	mat.shader = _create_cone_shader()
	mat.set_shader_parameter("color", Color(1.0, 0.1, 0.1, 0.45))
	mat.set_shader_parameter("edge_color", Color(1.0, 0.2, 0.1, 0.9))
	mat.set_shader_parameter("edge_width", 0.08)
	mat.set_shader_parameter("half_angle", deg_to_rad(90.0))  # 180 cone = 90 half
	melee_telegraph_mesh.set_surface_override_material(0, mat)
	melee_telegraph_mesh.visible = false
	melee_telegraph_mesh.position = Vector3(0.0, 0.02, 0.0)
	enemy.add_child(melee_telegraph_mesh)


func update_melee_telegraph_params(melee_range: float, melee_cone_angle: float) -> void:
	var mesh := melee_telegraph_mesh.mesh as PlaneMesh
	if mesh:
		mesh.size = Vector2(melee_range * 2.0, melee_range * 2.0)
	var mat := melee_telegraph_mesh.get_surface_override_material(0) as ShaderMaterial
	if mat:
		mat.set_shader_parameter("half_angle", melee_cone_angle / 2.0)


# =============================================================================
# AoE telegraph
# =============================================================================


func create_aoe_telegraph() -> void:
	aoe_telegraph_mesh = MeshInstance3D.new()
	var mesh := PlaneMesh.new()
	mesh.size = Vector2(10.0, 10.0)
	mesh.subdivide_width = 32
	mesh.subdivide_depth = 32
	aoe_telegraph_mesh.mesh = mesh
	var mat := ShaderMaterial.new()
	mat.shader = _create_circle_shader()
	mat.set_shader_parameter("color", Color(1.0, 0.3, 0.0, 0.35))
	mat.set_shader_parameter("edge_color", Color(1.0, 0.5, 0.0, 0.9))
	mat.set_shader_parameter("edge_width", 0.06)
	aoe_telegraph_mesh.set_surface_override_material(0, mat)
	aoe_telegraph_mesh.visible = false
	aoe_telegraph_mesh.position = Vector3(0.0, 0.03, 0.0)
	enemy.add_child(aoe_telegraph_mesh)


# =============================================================================
# Laser warning
# =============================================================================


func create_laser_warning() -> void:
	laser_warning_mesh = MeshInstance3D.new()
	var mesh := BoxMesh.new()
	mesh.size = Vector3(0.15, 0.15, 1.0)
	laser_warning_mesh.mesh = mesh
	var mat := StandardMaterial3D.new()
	mat.albedo_color = Color(1.0, 0.0, 0.0, 0.9)
	mat.emission_enabled = true
	mat.emission = Color(1.0, 0.1, 0.1)
	mat.emission_energy_multiplier = 5.0
	mat.shading_mode = BaseMaterial3D.SHADING_MODE_UNSHADED
	laser_warning_mesh.set_surface_override_material(0, mat)
	laser_warning_mesh.visible = false
	laser_warning_mesh.top_level = true
	enemy.add_child(laser_warning_mesh)


func update_laser_warning(ranged_target_position: Vector3) -> void:
	var start := enemy.global_position + Vector3(0.0, 1.0, 0.0)
	var end := ranged_target_position
	var mid := (start + end) / 2.0
	var dist := start.distance_to(end)

	laser_warning_mesh.global_position = mid
	laser_warning_mesh.scale = Vector3(1.0, 1.0, dist)
	if dist > 0.1:
		laser_warning_mesh.look_at(end, Vector3.UP)


# =============================================================================
# Charge telegraph
# =============================================================================


func create_charge_telegraph() -> void:
	charge_telegraph_mesh = MeshInstance3D.new()
	var mesh := BoxMesh.new()
	mesh.size = Vector3(0.6, 0.02, 1.0)
	charge_telegraph_mesh.mesh = mesh
	var mat := StandardMaterial3D.new()
	mat.albedo_color = Color(1.0, 0.5, 0.0, 0.7)
	mat.emission_enabled = true
	mat.emission = Color(1.0, 0.4, 0.0)
	mat.emission_energy_multiplier = 2.0
	mat.shading_mode = BaseMaterial3D.SHADING_MODE_UNSHADED
	charge_telegraph_mesh.set_surface_override_material(0, mat)
	charge_telegraph_mesh.visible = false
	charge_telegraph_mesh.top_level = true
	enemy.add_child(charge_telegraph_mesh)


# =============================================================================
# AoE fire particles
# =============================================================================


func create_aoe_particles() -> void:
	var fire_shader := _create_fire_shader()
	aoe_particles = _build_charging_particles(fire_shader)
	enemy.add_child(aoe_particles)
	aoe_slam_particles = _build_slam_particles(fire_shader)
	enemy.add_child(aoe_slam_particles)


func _build_charging_particles(fire_shader: Shader) -> GPUParticles3D:
	var particles := GPUParticles3D.new()
	particles.amount = 150
	particles.lifetime = 1.0
	particles.emitting = false
	particles.position = Vector3(0.0, 0.3, 0.0)
	particles.process_material = _build_charging_process_mat()

	var draw_mesh := QuadMesh.new()
	draw_mesh.size = Vector2(0.6, 0.8)
	particles.draw_pass_1 = draw_mesh
	var fire_mat := ShaderMaterial.new()
	fire_mat.shader = fire_shader
	draw_mesh.material = fire_mat
	return particles


func _build_charging_process_mat() -> ParticleProcessMaterial:
	var mat := ParticleProcessMaterial.new()
	mat.emission_shape = ParticleProcessMaterial.EMISSION_SHAPE_SPHERE
	mat.emission_sphere_radius = 0.3
	mat.direction = Vector3(0.0, 1.0, 0.0)
	mat.spread = 45.0
	mat.initial_velocity_min = 1.5
	mat.initial_velocity_max = 3.0
	mat.gravity = Vector3(0.0, 3.0, 0.0)
	mat.scale_min = 0.3
	mat.scale_max = 0.8
	mat.angular_velocity_min = -90.0
	mat.angular_velocity_max = 90.0

	var color_ramp := Gradient.new()
	color_ramp.set_color(0, Color(1.0, 1.0, 0.9, 1.0))
	color_ramp.add_point(0.3, Color(1.0, 0.7, 0.3, 0.9))
	color_ramp.add_point(0.7, Color(0.8, 0.2, 0.05, 0.6))
	color_ramp.set_color(1, Color(0.2, 0.02, 0.0, 0.0))
	var color_texture := GradientTexture1D.new()
	color_texture.gradient = color_ramp
	mat.color_ramp = color_texture

	var scale_curve := CurveTexture.new()
	var curve := Curve.new()
	curve.add_point(Vector2(0.0, 0.3))
	curve.add_point(Vector2(0.3, 1.0))
	curve.add_point(Vector2(0.7, 0.8))
	curve.add_point(Vector2(1.0, 0.1))
	scale_curve.curve = curve
	mat.scale_curve = scale_curve
	return mat


func _build_slam_particles(fire_shader: Shader) -> GPUParticles3D:
	var particles := GPUParticles3D.new()
	particles.amount = 512
	particles.lifetime = 1.0
	particles.one_shot = true
	particles.explosiveness = 1.0
	particles.emitting = false
	particles.position = Vector3(0.0, 0.5, 0.0)
	particles.process_material = _build_slam_process_mat()

	var slam_mesh := QuadMesh.new()
	slam_mesh.size = Vector2(1.5, 1.5)
	particles.draw_pass_1 = slam_mesh
	var slam_fire_mat := ShaderMaterial.new()
	slam_fire_mat.shader = fire_shader
	slam_mesh.material = slam_fire_mat
	return particles


func _build_slam_process_mat() -> ParticleProcessMaterial:
	var mat := ParticleProcessMaterial.new()
	mat.emission_shape = ParticleProcessMaterial.EMISSION_SHAPE_SPHERE
	mat.emission_sphere_radius = 1.0
	mat.direction = Vector3(0.0, 0.0, 0.0)
	mat.spread = 180.0
	mat.initial_velocity_min = 4.0
	mat.initial_velocity_max = 10.0
	mat.gravity = Vector3(0.0, 0.5, 0.0)
	mat.damping_min = 2.0
	mat.damping_max = 4.0
	mat.scale_min = 2.0
	mat.scale_max = 4.0
	mat.angular_velocity_min = -120.0
	mat.angular_velocity_max = 120.0

	var slam_ramp := Gradient.new()
	slam_ramp.set_color(0, Color(1.0, 1.0, 0.95, 1.0))
	slam_ramp.add_point(0.08, Color(1.0, 0.9, 0.4, 1.0))
	slam_ramp.add_point(0.25, Color(1.0, 0.5, 0.08, 0.95))
	slam_ramp.add_point(0.5, Color(0.8, 0.2, 0.03, 0.7))
	slam_ramp.add_point(0.75, Color(0.4, 0.06, 0.01, 0.35))
	slam_ramp.set_color(1, Color(0.08, 0.01, 0.0, 0.0))
	var slam_color_tex := GradientTexture1D.new()
	slam_color_tex.gradient = slam_ramp
	mat.color_ramp = slam_color_tex

	var slam_scale_curve := CurveTexture.new()
	var slam_curve := Curve.new()
	slam_curve.add_point(Vector2(0.0, 0.6))
	slam_curve.add_point(Vector2(0.1, 1.0))
	slam_curve.add_point(Vector2(0.4, 0.9))
	slam_curve.add_point(Vector2(0.7, 0.5))
	slam_curve.add_point(Vector2(1.0, 0.05))
	slam_scale_curve.curve = slam_curve
	mat.scale_curve = slam_scale_curve
	return mat


# =============================================================================
# Shaders
# =============================================================================


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
	shader.code = _FIRE_SHADER_CODE
	return shader
