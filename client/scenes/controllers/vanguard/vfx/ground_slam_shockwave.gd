extends Node3D

## Ground slam shockwave — expanding ring in a cone + debris particles + dust.
## One-shot, auto-frees after effect completes.

const SHOCKWAVE_SHADER := preload(
	"res://scenes/controllers/vanguard/vfx/ground_slam_shockwave.gdshader"
)
const EXPAND_DURATION: float = 0.4
const LIFETIME: float = 0.8

var _ring_material: ShaderMaterial = null


func _ready() -> void:
	top_level = true
	_setup_shockwave_ring()
	_setup_debris_particles()
	_setup_dust_particles()
	_setup_impact_flash()
	_setup_light()
	_animate_ring()
	get_tree().create_timer(LIFETIME).timeout.connect(queue_free)


func _setup_shockwave_ring() -> void:
	var mesh_inst := MeshInstance3D.new()
	var plane := PlaneMesh.new()
	plane.size = Vector2(14.0, 14.0)
	plane.subdivide_width = 32
	plane.subdivide_depth = 32
	mesh_inst.mesh = plane
	mesh_inst.cast_shadow = GeometryInstance3D.SHADOW_CASTING_SETTING_OFF

	_ring_material = ShaderMaterial.new()
	_ring_material.shader = SHOCKWAVE_SHADER
	_ring_material.set_shader_parameter("expand_progress", 0.0)
	mesh_inst.set_surface_override_material(0, _ring_material)

	add_child(mesh_inst)


func _animate_ring() -> void:
	var tween := get_tree().create_tween()
	tween.tween_method(_set_expand, 0.0, 1.0, EXPAND_DURATION)


func _set_expand(val: float) -> void:
	if _ring_material:
		_ring_material.set_shader_parameter("expand_progress", val)


func _setup_debris_particles() -> void:
	var particles := GPUParticles3D.new()
	particles.amount = 24
	particles.lifetime = 0.5
	particles.one_shot = true
	particles.explosiveness = 0.9
	particles.cast_shadow = GeometryInstance3D.SHADOW_CASTING_SETTING_OFF

	var mat := ParticleProcessMaterial.new()
	mat.emission_shape = ParticleProcessMaterial.EMISSION_SHAPE_SPHERE
	mat.emission_sphere_radius = 0.5
	mat.direction = Vector3(0.0, 0.5, -1.0)
	mat.spread = 45.0
	mat.initial_velocity_min = 6.0
	mat.initial_velocity_max = 12.0
	mat.gravity = Vector3(0.0, -8.0, 0.0)
	mat.scale_min = 0.5
	mat.scale_max = 1.5
	mat.damping_min = 1.0
	mat.damping_max = 3.0
	particles.process_material = mat

	var sphere := SphereMesh.new()
	sphere.radius = 0.06
	sphere.height = 0.12
	var draw_mat := StandardMaterial3D.new()
	draw_mat.albedo_color = Color(0.6, 0.5, 0.35, 0.9)
	draw_mat.shading_mode = BaseMaterial3D.SHADING_MODE_UNSHADED
	draw_mat.transparency = BaseMaterial3D.TRANSPARENCY_ALPHA
	sphere.material = draw_mat
	particles.draw_pass_1 = sphere

	add_child(particles)


func _setup_dust_particles() -> void:
	var particles := GPUParticles3D.new()
	particles.amount = 16
	particles.lifetime = 0.6
	particles.one_shot = true
	particles.explosiveness = 0.8
	particles.cast_shadow = GeometryInstance3D.SHADOW_CASTING_SETTING_OFF

	var mat := ParticleProcessMaterial.new()
	mat.emission_shape = ParticleProcessMaterial.EMISSION_SHAPE_SPHERE
	mat.emission_sphere_radius = 0.3
	mat.direction = Vector3(0.0, 0.2, -1.0)
	mat.spread = 50.0
	mat.initial_velocity_min = 2.0
	mat.initial_velocity_max = 5.0
	mat.gravity = Vector3(0.0, -1.0, 0.0)
	mat.scale_min = 1.0
	mat.scale_max = 2.0
	mat.damping_min = 3.0
	mat.damping_max = 5.0
	particles.process_material = mat

	var quad := QuadMesh.new()
	quad.size = Vector2(0.6, 0.6)
	var draw_mat := StandardMaterial3D.new()
	draw_mat.albedo_color = Color(0.7, 0.55, 0.3, 0.5)
	draw_mat.shading_mode = BaseMaterial3D.SHADING_MODE_UNSHADED
	draw_mat.billboard_mode = BaseMaterial3D.BILLBOARD_PARTICLES
	draw_mat.transparency = BaseMaterial3D.TRANSPARENCY_ALPHA
	quad.material = draw_mat
	particles.draw_pass_1 = quad

	add_child(particles)


func _setup_impact_flash() -> void:
	var flash := MeshInstance3D.new()
	var quad := QuadMesh.new()
	quad.size = Vector2(2.5, 2.5)
	flash.mesh = quad
	flash.position.y = 0.1
	flash.cast_shadow = GeometryInstance3D.SHADOW_CASTING_SETTING_OFF

	var mat := StandardMaterial3D.new()
	mat.albedo_color = Color(1.0, 0.4, 0.2, 0.8)
	mat.emission_enabled = true
	mat.emission = Color(1.0, 0.4, 0.2)
	mat.emission_energy_multiplier = 8.0
	mat.shading_mode = BaseMaterial3D.SHADING_MODE_UNSHADED
	mat.billboard_mode = BaseMaterial3D.BILLBOARD_ENABLED
	mat.transparency = BaseMaterial3D.TRANSPARENCY_ALPHA
	mat.blend_mode = BaseMaterial3D.BLEND_MODE_ADD
	flash.set_surface_override_material(0, mat)
	add_child(flash)

	var tween := get_tree().create_tween()
	tween.tween_property(flash, "scale", Vector3(4.0, 4.0, 4.0), 0.12).from(Vector3(0.5, 0.5, 0.5))
	tween.parallel().tween_property(mat, "albedo_color:a", 0.0, 0.12).from(0.8)


func _setup_light() -> void:
	var light := OmniLight3D.new()
	light.light_color = Color(0.9, 0.15, 0.1)
	light.light_energy = 6.0
	light.omni_range = 8.0
	light.shadow_enabled = false
	add_child(light)

	var tween := get_tree().create_tween()
	tween.tween_property(light, "light_energy", 0.0, 0.2)
