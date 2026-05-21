extends Node3D

## Gust Step wind trail — one-shot cyan burst at dodge start position.
## Wind particles + expanding ground ring + light. Auto-frees after 0.8s.

const LIFETIME: float = 0.8
const FLUX_CYAN := Color(0.2, 0.9, 1.0)

static var _circle_shader: Shader


func _ready() -> void:
	top_level = true
	_load_shader()
	_setup_wind_particles()
	_setup_ground_ring()
	_setup_light()
	get_tree().create_timer(LIFETIME).timeout.connect(queue_free)


func _load_shader() -> void:
	if _circle_shader == null:
		_circle_shader = load("res://assets/shaders/telegraph_circle.gdshader")


func _setup_wind_particles() -> void:
	var particles := GPUParticles3D.new()
	particles.amount = 16
	particles.lifetime = 0.5
	particles.one_shot = true
	particles.explosiveness = 0.9
	particles.cast_shadow = GeometryInstance3D.SHADOW_CASTING_SETTING_OFF

	var mat := ParticleProcessMaterial.new()
	mat.emission_shape = ParticleProcessMaterial.EMISSION_SHAPE_SPHERE
	mat.emission_sphere_radius = 0.3
	mat.spread = 60.0
	mat.initial_velocity_min = 3.0
	mat.initial_velocity_max = 5.0
	mat.gravity = Vector3.ZERO
	mat.scale_min = 0.5
	mat.scale_max = 1.2
	mat.damping_min = 2.0
	mat.damping_max = 4.0
	particles.process_material = mat

	var quad := QuadMesh.new()
	quad.size = Vector2(0.1, 0.1)
	var draw_mat := StandardMaterial3D.new()
	draw_mat.albedo_color = Color(FLUX_CYAN, 0.8)
	draw_mat.emission_enabled = true
	draw_mat.emission = FLUX_CYAN
	draw_mat.emission_energy_multiplier = 4.0
	draw_mat.shading_mode = BaseMaterial3D.SHADING_MODE_UNSHADED
	draw_mat.billboard_mode = BaseMaterial3D.BILLBOARD_PARTICLES
	draw_mat.transparency = BaseMaterial3D.TRANSPARENCY_ALPHA
	draw_mat.blend_mode = BaseMaterial3D.BLEND_MODE_ADD
	quad.material = draw_mat
	particles.draw_pass_1 = quad

	particles.position = Vector3(0.0, 0.5, 0.0)
	add_child(particles)


func _setup_ground_ring() -> void:
	var mesh_inst := MeshInstance3D.new()
	var plane := PlaneMesh.new()
	plane.size = Vector2(2.0, 2.0)
	plane.subdivide_width = 8
	plane.subdivide_depth = 8
	mesh_inst.mesh = plane
	mesh_inst.cast_shadow = GeometryInstance3D.SHADOW_CASTING_SETTING_OFF
	mesh_inst.position = Vector3(0.0, 0.03, 0.0)

	var mat := ShaderMaterial.new()
	mat.shader = _circle_shader
	mat.set_shader_parameter("color", Color(FLUX_CYAN, 0.3))
	mat.set_shader_parameter("edge_color", Color(FLUX_CYAN, 0.7))
	mat.set_shader_parameter("edge_width", 0.1)
	mat.set_shader_parameter("fade", 1.0)
	mesh_inst.set_surface_override_material(0, mat)
	add_child(mesh_inst)

	var tween := get_tree().create_tween()
	tween.tween_property(mesh_inst, "scale", Vector3(3.0, 1.0, 3.0), 0.3).from(
		Vector3(0.5, 1.0, 0.5)
	)
	tween.parallel().tween_method(
		func(val: float) -> void: mat.set_shader_parameter("fade", val),
		1.0,
		0.0,
		0.3,
	)


func _setup_light() -> void:
	var light := OmniLight3D.new()
	light.light_color = FLUX_CYAN
	light.light_energy = 3.0
	light.omni_range = 3.0
	light.shadow_enabled = false
	add_child(light)

	var tween := get_tree().create_tween()
	tween.tween_property(light, "light_energy", 0.0, 0.2)
