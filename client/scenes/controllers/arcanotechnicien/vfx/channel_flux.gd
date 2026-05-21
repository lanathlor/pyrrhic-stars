extends Node3D

## Channel Flux accumulation — particles orbiting the caster during channeled spells.
## Intensity scales with channel progress (0.0 to 1.0).
## Call start() to begin, update_progress() each frame, stop() to fade and free.

const FADE_OUT_TIME: float = 0.2
const FLUX_CYAN := Color(0.2, 0.9, 1.0)
const HEAL_GREEN := Color(0.3, 1.0, 0.4)

var _target: Node3D = null
var _outer_particles: GPUParticles3D = null
var _inner_particles: GPUParticles3D = null
var _light: OmniLight3D = null
var _stopping: bool = false
var _outer_draw_mat: StandardMaterial3D = null
var _inner_draw_mat: StandardMaterial3D = null


func _ready() -> void:
	top_level = true


func start(target: Node3D) -> void:
	_target = target
	global_position = target.global_position + Vector3(0.0, 1.0, 0.0)
	_setup_outer_particles()
	_setup_inner_particles()
	_setup_light()


func update_progress(progress: float) -> void:
	if _stopping:
		return
	# Scale intensity with channel progress
	if _outer_particles:
		_outer_particles.speed_scale = 0.3 + 0.7 * progress
	if _inner_particles:
		_inner_particles.speed_scale = 0.3 + 0.7 * progress
	if _outer_draw_mat:
		_outer_draw_mat.albedo_color.a = 0.4 + 0.5 * progress
	if _inner_draw_mat:
		_inner_draw_mat.albedo_color.a = 0.4 + 0.5 * progress
	if _light:
		_light.light_energy = 0.5 + 2.0 * progress


func stop() -> void:
	if _stopping:
		return
	_stopping = true
	if _outer_particles:
		_outer_particles.emitting = false
	if _inner_particles:
		_inner_particles.emitting = false
	var tween := get_tree().create_tween()
	if _light:
		tween.tween_property(_light, "light_energy", 0.0, FADE_OUT_TIME)
	tween.tween_callback(queue_free)


func _process(_delta: float) -> void:
	if _target and is_instance_valid(_target):
		global_position = _target.global_position + Vector3(0.0, 1.0, 0.0)
	elif not _stopping:
		stop()


func _setup_outer_particles() -> void:
	_outer_particles = GPUParticles3D.new()
	_outer_particles.amount = 16
	_outer_particles.lifetime = 0.8
	_outer_particles.cast_shadow = GeometryInstance3D.SHADOW_CASTING_SETTING_OFF

	var mat := ParticleProcessMaterial.new()
	mat.emission_shape = ParticleProcessMaterial.EMISSION_SHAPE_RING
	mat.emission_ring_radius = 1.0
	mat.emission_ring_inner_radius = 0.8
	mat.emission_ring_height = 0.1
	mat.emission_ring_axis = Vector3(0.0, 1.0, 0.0)
	mat.direction = Vector3(0.0, 1.0, 0.0)
	mat.spread = 30.0
	mat.initial_velocity_min = 0.5
	mat.initial_velocity_max = 1.5
	mat.gravity = Vector3.ZERO
	mat.scale_min = 0.4
	mat.scale_max = 0.8
	mat.angular_velocity_min = 90.0
	mat.angular_velocity_max = 180.0
	_outer_particles.process_material = mat

	var quad := QuadMesh.new()
	quad.size = Vector2(0.06, 0.06)
	_outer_draw_mat = StandardMaterial3D.new()
	_outer_draw_mat.albedo_color = Color(FLUX_CYAN, 0.4)
	_outer_draw_mat.emission_enabled = true
	_outer_draw_mat.emission = FLUX_CYAN
	_outer_draw_mat.emission_energy_multiplier = 4.0
	_outer_draw_mat.shading_mode = BaseMaterial3D.SHADING_MODE_UNSHADED
	_outer_draw_mat.billboard_mode = BaseMaterial3D.BILLBOARD_PARTICLES
	_outer_draw_mat.transparency = BaseMaterial3D.TRANSPARENCY_ALPHA
	quad.material = _outer_draw_mat
	_outer_particles.draw_pass_1 = quad

	add_child(_outer_particles)


func _setup_inner_particles() -> void:
	_inner_particles = GPUParticles3D.new()
	_inner_particles.amount = 8
	_inner_particles.lifetime = 0.6
	_inner_particles.cast_shadow = GeometryInstance3D.SHADOW_CASTING_SETTING_OFF

	var mat := ParticleProcessMaterial.new()
	mat.emission_shape = ParticleProcessMaterial.EMISSION_SHAPE_SPHERE
	mat.emission_sphere_radius = 0.3
	mat.direction = Vector3(0.0, 1.0, 0.0)
	mat.spread = 40.0
	mat.initial_velocity_min = 0.3
	mat.initial_velocity_max = 0.8
	mat.gravity = Vector3.ZERO
	mat.scale_min = 0.3
	mat.scale_max = 0.6
	_inner_particles.process_material = mat

	var quad := QuadMesh.new()
	quad.size = Vector2(0.06, 0.06)
	_inner_draw_mat = StandardMaterial3D.new()
	_inner_draw_mat.albedo_color = Color(HEAL_GREEN, 0.4)
	_inner_draw_mat.emission_enabled = true
	_inner_draw_mat.emission = HEAL_GREEN
	_inner_draw_mat.emission_energy_multiplier = 3.0
	_inner_draw_mat.shading_mode = BaseMaterial3D.SHADING_MODE_UNSHADED
	_inner_draw_mat.billboard_mode = BaseMaterial3D.BILLBOARD_PARTICLES
	_inner_draw_mat.transparency = BaseMaterial3D.TRANSPARENCY_ALPHA
	quad.material = _inner_draw_mat
	_inner_particles.draw_pass_1 = quad

	add_child(_inner_particles)


func _setup_light() -> void:
	_light = OmniLight3D.new()
	_light.light_color = FLUX_CYAN
	_light.light_energy = 0.5
	_light.omni_range = 4.0
	_light.shadow_enabled = false
	add_child(_light)
