extends Node3D

## Block shield effect — translucent hex-pattern energy barrier.
## Stays active while blocking; call fade_out() to dismiss.

const SHIELD_SHADER := preload("res://scenes/controllers/vanguard/vfx/block_shield.gdshader")
const FADE_IN_TIME: float = 0.1
const FADE_OUT_TIME: float = 0.1

var _material: ShaderMaterial = null
var _particles: GPUParticles3D = null
var _fade_tween: Tween = null


func _ready() -> void:
	_setup_shield_mesh()
	_setup_edge_particles()
	_fade_in()


func fade_out() -> void:
	if _fade_tween and _fade_tween.is_valid():
		_fade_tween.kill()
	_fade_tween = get_tree().create_tween()
	_fade_tween.tween_method(_set_opacity, 1.0, 0.0, FADE_OUT_TIME)
	_fade_tween.tween_callback(queue_free)


func _fade_in() -> void:
	_set_opacity(0.0)
	_fade_tween = get_tree().create_tween()
	_fade_tween.tween_method(_set_opacity, 0.0, 1.0, FADE_IN_TIME)


func _set_opacity(val: float) -> void:
	if _material:
		_material.set_shader_parameter("opacity", val)


func _setup_shield_mesh() -> void:
	var mesh_inst := MeshInstance3D.new()
	var quad := QuadMesh.new()
	quad.size = Vector2(1.6, 1.8)
	quad.subdivide_width = 8
	quad.subdivide_depth = 8
	mesh_inst.mesh = quad
	mesh_inst.cast_shadow = GeometryInstance3D.SHADOW_CASTING_SETTING_OFF

	_material = ShaderMaterial.new()
	_material.shader = SHIELD_SHADER
	mesh_inst.set_surface_override_material(0, _material)

	add_child(mesh_inst)


func _setup_edge_particles() -> void:
	_particles = GPUParticles3D.new()
	_particles.amount = 12
	_particles.lifetime = 0.6
	_particles.cast_shadow = GeometryInstance3D.SHADOW_CASTING_SETTING_OFF

	var mat := ParticleProcessMaterial.new()
	mat.emission_shape = ParticleProcessMaterial.EMISSION_SHAPE_BOX
	mat.emission_box_extents = Vector3(0.6, 0.75, 0.02)
	mat.initial_velocity_min = 0.2
	mat.initial_velocity_max = 0.5
	mat.gravity = Vector3.ZERO
	mat.scale_min = 0.5
	mat.scale_max = 1.0
	_particles.process_material = mat

	var quad := QuadMesh.new()
	quad.size = Vector2(0.08, 0.08)
	var draw_mat := StandardMaterial3D.new()
	draw_mat.albedo_color = Color(0.9, 0.15, 0.1, 0.9)
	draw_mat.emission_enabled = true
	draw_mat.emission = Color(0.9, 0.15, 0.1)
	draw_mat.emission_energy_multiplier = 4.0
	draw_mat.shading_mode = BaseMaterial3D.SHADING_MODE_UNSHADED
	draw_mat.billboard_mode = BaseMaterial3D.BILLBOARD_PARTICLES
	draw_mat.transparency = BaseMaterial3D.TRANSPARENCY_ALPHA
	quad.material = draw_mat
	_particles.draw_pass_1 = quad

	add_child(_particles)
