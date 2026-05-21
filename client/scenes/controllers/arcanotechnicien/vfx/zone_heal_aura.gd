extends Node3D

## Persistent ground circle for Transfusion AoE heal zone.
## Stays alive during channel — call stop() to dismiss.
## Uses telegraph_circle.gdshader with heal-green color and gentle pulse.

const FADE_OUT_TIME: float = 0.3
const HEAL_GREEN := Color(0.3, 1.0, 0.4)

var _target: Node3D = null
var _material: ShaderMaterial = null
var _particles: GPUParticles3D = null
var _stopping: bool = false
var _time: float = 0.0

static var _circle_shader: Shader


func _ready() -> void:
	top_level = true


func start(target: Node3D, radius: float) -> void:
	_target = target
	global_position = target.global_position + Vector3(0.0, 0.05, 0.0)
	_load_shader()
	_setup_ground_plane(radius)
	_setup_edge_particles(radius)


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


func _process(delta: float) -> void:
	_time += delta

	# Follow caster
	if _target and is_instance_valid(_target):
		global_position = _target.global_position + Vector3(0.0, 0.05, 0.0)
	elif not _stopping:
		stop()
		return

	# Gentle pulse
	if not _stopping and _material:
		var pulse: float = 0.85 + 0.15 * sin(_time * 3.0)
		_material.set_shader_parameter("fade", pulse)


func _load_shader() -> void:
	if _circle_shader == null:
		_circle_shader = load("res://assets/shaders/telegraph_circle.gdshader")


func _setup_ground_plane(radius: float) -> void:
	var mesh_inst := MeshInstance3D.new()
	var plane := PlaneMesh.new()
	plane.size = Vector2(radius * 2.0, radius * 2.0)
	plane.subdivide_width = 16
	plane.subdivide_depth = 16
	mesh_inst.mesh = plane
	mesh_inst.cast_shadow = GeometryInstance3D.SHADOW_CASTING_SETTING_OFF

	_material = ShaderMaterial.new()
	_material.shader = _circle_shader
	_material.set_shader_parameter("color", Color(HEAL_GREEN, 0.3))
	_material.set_shader_parameter("edge_color", Color(0.5, 1.0, 0.6, 0.8))
	_material.set_shader_parameter("edge_width", 0.06)
	_material.set_shader_parameter("fade", 1.0)
	mesh_inst.set_surface_override_material(0, _material)

	add_child(mesh_inst)


func _setup_edge_particles(radius: float) -> void:
	_particles = GPUParticles3D.new()
	_particles.amount = 16
	_particles.lifetime = 1.2
	_particles.cast_shadow = GeometryInstance3D.SHADOW_CASTING_SETTING_OFF

	var mat := ParticleProcessMaterial.new()
	mat.emission_shape = ParticleProcessMaterial.EMISSION_SHAPE_RING
	mat.emission_ring_radius = radius * 0.95
	mat.emission_ring_inner_radius = radius * 0.85
	mat.emission_ring_height = 0.1
	mat.emission_ring_axis = Vector3(0.0, 1.0, 0.0)
	mat.direction = Vector3(0.0, 1.0, 0.0)
	mat.spread = 20.0
	mat.initial_velocity_min = 0.2
	mat.initial_velocity_max = 0.6
	mat.gravity = Vector3.ZERO
	mat.scale_min = 0.4
	mat.scale_max = 0.8
	_particles.process_material = mat

	var quad := QuadMesh.new()
	quad.size = Vector2(0.06, 0.06)
	var draw_mat := StandardMaterial3D.new()
	draw_mat.albedo_color = Color(HEAL_GREEN, 0.7)
	draw_mat.emission_enabled = true
	draw_mat.emission = HEAL_GREEN
	draw_mat.emission_energy_multiplier = 3.0
	draw_mat.shading_mode = BaseMaterial3D.SHADING_MODE_UNSHADED
	draw_mat.billboard_mode = BaseMaterial3D.BILLBOARD_PARTICLES
	draw_mat.transparency = BaseMaterial3D.TRANSPARENCY_ALPHA
	quad.material = draw_mat
	_particles.draw_pass_1 = quad

	add_child(_particles)
