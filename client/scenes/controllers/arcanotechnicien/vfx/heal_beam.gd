extends Node3D

## Healing beam tether — energy stream from caster to target during Mending Beam.
## Dynamically orients a quad mesh each frame. Uses heal_beam.gdshader.
## Call start() to create, stop() to fade out and free.

const BEAM_SHADER := preload("res://assets/shaders/heal_beam.gdshader")
const BEAM_WIDTH: float = 0.6
const CHEST_OFFSET := Vector3(0.0, 1.2, 0.0)
const FADE_OUT_TIME: float = 0.2

var _caster: Node3D = null
var _target: Node3D = null
var _mesh_inst: MeshInstance3D = null
var _material: ShaderMaterial = null
var _particles: GPUParticles3D = null
var _light: OmniLight3D = null
var _stopping: bool = false


func _ready() -> void:
	top_level = true


func start(caster: Node3D, target: Node3D) -> void:
	_caster = caster
	_target = target
	_setup_beam_mesh()
	_setup_particles()
	_setup_light()
	_update_orientation()


func stop() -> void:
	if _stopping:
		return
	_stopping = true
	if _particles:
		_particles.emitting = false
	var tween := get_tree().create_tween()
	tween.tween_method(_set_fade, 1.0, 0.0, FADE_OUT_TIME)
	tween.tween_callback(queue_free)


func _set_fade(val: float) -> void:
	if _material:
		_material.set_shader_parameter("fade", val)
	if _light:
		_light.light_energy = 2.0 * val


func _process(_delta: float) -> void:
	if _stopping:
		return
	if not _caster or not is_instance_valid(_caster):
		queue_free()
		return
	if not _target or not is_instance_valid(_target):
		queue_free()
		return
	_update_orientation()


func _update_orientation() -> void:
	var start_pos: Vector3 = _caster.global_position + CHEST_OFFSET
	var end_pos: Vector3 = _target.global_position + CHEST_OFFSET
	var dist: float = start_pos.distance_to(end_pos)
	if dist < 0.1:
		dist = 0.1

	global_position = (start_pos + end_pos) * 0.5
	look_at(end_pos)

	# Stretch the quad along the beam direction (local -Z after look_at)
	if _mesh_inst and _mesh_inst.mesh:
		_mesh_inst.mesh.size = Vector2(BEAM_WIDTH, dist)
	# Rotate mesh to face beam direction — quad's Y maps along local -Z
	_mesh_inst.rotation = Vector3(-PI * 0.5, 0.0, 0.0)

	if _light:
		_light.position = Vector3.ZERO


func _setup_beam_mesh() -> void:
	_mesh_inst = MeshInstance3D.new()
	var quad := QuadMesh.new()
	quad.size = Vector2(BEAM_WIDTH, 1.0)
	quad.subdivide_width = 4
	quad.subdivide_depth = 8
	_mesh_inst.mesh = quad
	_mesh_inst.cast_shadow = GeometryInstance3D.SHADOW_CASTING_SETTING_OFF

	_material = ShaderMaterial.new()
	_material.shader = BEAM_SHADER
	_mesh_inst.set_surface_override_material(0, _material)

	add_child(_mesh_inst)


func _setup_particles() -> void:
	_particles = GPUParticles3D.new()
	_particles.amount = 8
	_particles.lifetime = 0.4
	_particles.cast_shadow = GeometryInstance3D.SHADOW_CASTING_SETTING_OFF

	var mat := ParticleProcessMaterial.new()
	mat.emission_shape = ParticleProcessMaterial.EMISSION_SHAPE_BOX
	mat.emission_box_extents = Vector3(0.2, 0.2, 0.5)
	mat.initial_velocity_min = 0.3
	mat.initial_velocity_max = 0.8
	mat.gravity = Vector3.ZERO
	mat.scale_min = 0.5
	mat.scale_max = 1.0
	_particles.process_material = mat

	var quad := QuadMesh.new()
	quad.size = Vector2(0.08, 0.08)
	var draw_mat := StandardMaterial3D.new()
	draw_mat.albedo_color = Color(0.3, 1.0, 0.5, 0.8)
	draw_mat.emission_enabled = true
	draw_mat.emission = Color(0.3, 1.0, 0.5)
	draw_mat.emission_energy_multiplier = 4.0
	draw_mat.shading_mode = BaseMaterial3D.SHADING_MODE_UNSHADED
	draw_mat.billboard_mode = BaseMaterial3D.BILLBOARD_PARTICLES
	draw_mat.transparency = BaseMaterial3D.TRANSPARENCY_ALPHA
	quad.material = draw_mat
	_particles.draw_pass_1 = quad

	add_child(_particles)


func _setup_light() -> void:
	_light = OmniLight3D.new()
	_light.light_color = Color(0.2, 0.9, 0.5)
	_light.light_energy = 2.0
	_light.omni_range = 4.0
	_light.shadow_enabled = false
	add_child(_light)
