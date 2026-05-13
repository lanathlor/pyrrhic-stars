extends Node3D

## One-shot parry flash — bright burst on successful parry timing.
## Flash disc + expanding ring + directional sparks + point light.

const LIFETIME: float = 0.4


func _ready() -> void:
	top_level = true
	_setup_flash_disc()
	_setup_expanding_ring()
	_setup_sparks()
	_setup_light()
	get_tree().create_timer(LIFETIME).timeout.connect(queue_free)


func _setup_flash_disc() -> void:
	var flash := MeshInstance3D.new()
	var quad := QuadMesh.new()
	quad.size = Vector2(2.5, 2.5)
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
	tween.tween_property(flash, "scale", Vector3(1.5, 1.5, 1.5), 0.12).from(Vector3(0.5, 0.5, 0.5))
	tween.parallel().tween_property(mat, "albedo_color:a", 0.0, 0.12).from(0.9)


func _setup_expanding_ring() -> void:
	var ring := MeshInstance3D.new()
	var torus := TorusMesh.new()
	torus.inner_radius = 0.3
	torus.outer_radius = 0.5
	ring.mesh = torus
	ring.rotation.x = -PI / 2.0
	ring.cast_shadow = GeometryInstance3D.SHADOW_CASTING_SETTING_OFF

	var mat := StandardMaterial3D.new()
	mat.albedo_color = Color(1.0, 0.4, 0.2, 0.8)
	mat.emission_enabled = true
	mat.emission = Color(1.0, 0.4, 0.2)
	mat.emission_energy_multiplier = 4.0
	mat.shading_mode = BaseMaterial3D.SHADING_MODE_UNSHADED
	mat.transparency = BaseMaterial3D.TRANSPARENCY_ALPHA
	mat.blend_mode = BaseMaterial3D.BLEND_MODE_ADD
	ring.set_surface_override_material(0, mat)
	add_child(ring)

	var tween := get_tree().create_tween()
	tween.tween_property(ring, "scale", Vector3(8.0, 8.0, 8.0), 0.2).from(Vector3.ONE)
	tween.parallel().tween_property(mat, "albedo_color:a", 0.0, 0.2).from(0.8)


func _setup_sparks() -> void:
	var particles := GPUParticles3D.new()
	particles.amount = 16
	particles.lifetime = 0.25
	particles.one_shot = true
	particles.explosiveness = 1.0
	particles.cast_shadow = GeometryInstance3D.SHADOW_CASTING_SETTING_OFF

	var mat := ParticleProcessMaterial.new()
	mat.emission_shape = ParticleProcessMaterial.EMISSION_SHAPE_SPHERE
	mat.emission_sphere_radius = 0.15
	mat.direction = Vector3(0.0, 0.0, -1.0)
	mat.spread = 60.0
	mat.initial_velocity_min = 8.0
	mat.initial_velocity_max = 12.0
	mat.gravity = Vector3(0.0, -3.0, 0.0)
	mat.damping_min = 4.0
	mat.damping_max = 6.0
	mat.scale_min = 0.8
	mat.scale_max = 1.5
	particles.process_material = mat

	var quad := QuadMesh.new()
	quad.size = Vector2(0.08, 0.25)
	var draw_mat := StandardMaterial3D.new()
	draw_mat.albedo_color = Color(1.0, 0.6, 0.4, 0.9)
	draw_mat.emission_enabled = true
	draw_mat.emission = Color(1.0, 0.4, 0.2)
	draw_mat.emission_energy_multiplier = 6.0
	draw_mat.shading_mode = BaseMaterial3D.SHADING_MODE_UNSHADED
	draw_mat.billboard_mode = BaseMaterial3D.BILLBOARD_PARTICLES
	draw_mat.transparency = BaseMaterial3D.TRANSPARENCY_ALPHA
	quad.material = draw_mat
	particles.draw_pass_1 = quad

	add_child(particles)


func _setup_light() -> void:
	var light := OmniLight3D.new()
	light.light_color = Color(0.9, 0.15, 0.1)
	light.light_energy = 8.0
	light.omni_range = 6.0
	light.shadow_enabled = false
	add_child(light)

	var tween := get_tree().create_tween()
	tween.tween_property(light, "light_energy", 0.0, 0.15)
