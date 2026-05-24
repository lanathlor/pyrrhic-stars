extends Node3D

## Quick commit flash — brief green glow at caster's hands when committing a direct ability.
## Gives immediate visual feedback that an ability was committed.
## One-shot, auto-frees after 0.35s.

const LIFETIME: float = 0.35
const HEAL_GREEN := Color(0.3, 1.0, 0.4)
const FLUX_CYAN := Color(0.2, 0.9, 1.0)


func _ready() -> void:
	top_level = true
	_setup_flash()
	_setup_particles()
	_setup_light()
	get_tree().create_timer(LIFETIME).timeout.connect(queue_free)


func _setup_flash() -> void:
	var flash := MeshInstance3D.new()
	var quad := QuadMesh.new()
	quad.size = Vector2(0.6, 0.6)
	flash.mesh = quad
	flash.cast_shadow = GeometryInstance3D.SHADOW_CASTING_SETTING_OFF

	var mat := StandardMaterial3D.new()
	mat.albedo_color = Color(HEAL_GREEN, 0.9)
	mat.emission_enabled = true
	mat.emission = HEAL_GREEN
	mat.emission_energy_multiplier = 6.0
	mat.shading_mode = BaseMaterial3D.SHADING_MODE_UNSHADED
	mat.billboard_mode = BaseMaterial3D.BILLBOARD_ENABLED
	mat.transparency = BaseMaterial3D.TRANSPARENCY_ALPHA
	mat.blend_mode = BaseMaterial3D.BLEND_MODE_ADD
	flash.set_surface_override_material(0, mat)
	add_child(flash)

	var tween := get_tree().create_tween()
	tween.tween_property(flash, "scale", Vector3(2.0, 2.0, 2.0), 0.12).from(Vector3(0.3, 0.3, 0.3))
	tween.parallel().tween_property(mat, "albedo_color:a", 0.0, 0.2).from(0.9)


func _setup_particles() -> void:
	var particles := GPUParticles3D.new()
	particles.amount = 8
	particles.lifetime = 0.25
	particles.one_shot = true
	particles.explosiveness = 0.9
	particles.cast_shadow = GeometryInstance3D.SHADOW_CASTING_SETTING_OFF

	var mat := ParticleProcessMaterial.new()
	mat.emission_shape = ParticleProcessMaterial.EMISSION_SHAPE_SPHERE
	mat.emission_sphere_radius = 0.15
	mat.spread = 120.0
	mat.initial_velocity_min = 1.5
	mat.initial_velocity_max = 3.0
	mat.gravity = Vector3(0.0, -1.0, 0.0)
	mat.scale_min = 0.4
	mat.scale_max = 0.8
	mat.damping_min = 2.0
	mat.damping_max = 4.0
	particles.process_material = mat

	var quad := QuadMesh.new()
	quad.size = Vector2(0.05, 0.05)
	var draw_mat := StandardMaterial3D.new()
	draw_mat.albedo_color = Color(FLUX_CYAN, 0.8)
	draw_mat.emission_enabled = true
	draw_mat.emission = FLUX_CYAN
	draw_mat.emission_energy_multiplier = 4.0
	draw_mat.shading_mode = BaseMaterial3D.SHADING_MODE_UNSHADED
	draw_mat.billboard_mode = BaseMaterial3D.BILLBOARD_PARTICLES
	draw_mat.transparency = BaseMaterial3D.TRANSPARENCY_ALPHA
	quad.material = draw_mat
	particles.draw_pass_1 = quad

	add_child(particles)


func _setup_light() -> void:
	var light := OmniLight3D.new()
	light.light_color = HEAL_GREEN
	light.light_energy = 3.0
	light.omni_range = 2.5
	light.shadow_enabled = false
	add_child(light)

	var tween := get_tree().create_tween()
	tween.tween_property(light, "light_energy", 0.0, 0.15)
