extends Node3D

## One-shot hit impact effect — sparks, flash, and light at damage point.
## Auto-frees after particles finish.

const LIFETIME: float = 0.5


func _ready() -> void:
	top_level = true
	_setup_sparks()
	_setup_flash()
	_setup_light()
	get_tree().create_timer(LIFETIME).timeout.connect(queue_free)


func _setup_sparks() -> void:
	var particles := GPUParticles3D.new()
	particles.amount = 20
	particles.lifetime = 0.3
	particles.one_shot = true
	particles.explosiveness = 1.0
	particles.cast_shadow = GeometryInstance3D.SHADOW_CASTING_SETTING_OFF

	var mat := ParticleProcessMaterial.new()
	mat.emission_shape = ParticleProcessMaterial.EMISSION_SHAPE_SPHERE
	mat.emission_sphere_radius = 0.1
	mat.spread = 180.0
	mat.initial_velocity_min = 5.0
	mat.initial_velocity_max = 10.0
	mat.gravity = Vector3(0.0, -5.0, 0.0)
	mat.scale_min = 0.8
	mat.scale_max = 1.2
	mat.damping_min = 2.0
	mat.damping_max = 4.0
	particles.process_material = mat

	var quad := QuadMesh.new()
	quad.size = Vector2(0.06, 0.2)
	var draw_mat := StandardMaterial3D.new()
	draw_mat.albedo_color = Color(1.0, 0.4, 0.2, 0.9)
	draw_mat.emission_enabled = true
	draw_mat.emission = Color(1.0, 0.4, 0.2)
	draw_mat.emission_energy_multiplier = 5.0
	draw_mat.shading_mode = BaseMaterial3D.SHADING_MODE_UNSHADED
	draw_mat.billboard_mode = BaseMaterial3D.BILLBOARD_PARTICLES
	draw_mat.transparency = BaseMaterial3D.TRANSPARENCY_ALPHA
	quad.material = draw_mat
	particles.draw_pass_1 = quad

	add_child(particles)


func _setup_flash() -> void:
	var flash := MeshInstance3D.new()
	var quad := QuadMesh.new()
	quad.size = Vector2(0.8, 0.8)
	flash.mesh = quad
	flash.cast_shadow = GeometryInstance3D.SHADOW_CASTING_SETTING_OFF

	var mat := StandardMaterial3D.new()
	mat.albedo_color = Color(1.0, 0.6, 0.4, 0.9)
	mat.emission_enabled = true
	mat.emission = Color(1.0, 0.6, 0.4)
	mat.emission_energy_multiplier = 8.0
	mat.shading_mode = BaseMaterial3D.SHADING_MODE_UNSHADED
	mat.billboard_mode = BaseMaterial3D.BILLBOARD_ENABLED
	mat.transparency = BaseMaterial3D.TRANSPARENCY_ALPHA
	mat.blend_mode = BaseMaterial3D.BLEND_MODE_ADD
	flash.set_surface_override_material(0, mat)
	add_child(flash)

	var tween := get_tree().create_tween()
	tween.tween_property(flash, "scale", Vector3(4.0, 4.0, 4.0), 0.1).from(Vector3.ONE)
	tween.parallel().tween_property(mat, "albedo_color:a", 0.0, 0.1).from(0.9)


func _setup_light() -> void:
	var light := OmniLight3D.new()
	light.light_color = Color(0.9, 0.15, 0.1)
	light.light_energy = 5.0
	light.omni_range = 4.0
	light.shadow_enabled = false
	add_child(light)

	var tween := get_tree().create_tween()
	tween.tween_property(light, "light_energy", 0.0, 0.15)
