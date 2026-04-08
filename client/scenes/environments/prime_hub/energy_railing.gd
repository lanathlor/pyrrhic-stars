@tool
extends Node3D
## Sci-fi energy beam railing.
## Creates a glowing blue force-field plane with drifting particles.
## Attach to a Node3D positioned where the railing center should be.

const BEAM_SHADER := preload("res://scenes/environments/prime_hub/energy_beam.gdshader")

## Full size of the beam plane (width, height, depth is ignored — plane is flat).
## For an X-aligned railing: Vector3(0.1, 1.2, 6) means the plane faces X, is 1.2m tall, 6m wide.
## For a Z-aligned railing: Vector3(6, 1.2, 0.1) means the plane faces Z, is 1.2m tall, 6m wide.
@export var beam_size: Vector3 = Vector3(0.1, 1.2, 6.0)

## Color uniforms exposed to inspector
@export var beam_color: Color = Color(0.2, 0.5, 1.0, 0.6)
@export var core_color: Color = Color(0.8, 0.9, 1.0, 0.9)

var _mesh_instance: MeshInstance3D
var _particles: GPUParticles3D


func _ready() -> void:
	_build_beam()
	_build_particles()


func _build_beam() -> void:
	_mesh_instance = MeshInstance3D.new()
	_mesh_instance.name = "BeamMesh"

	var quad := QuadMesh.new()

	# Determine if the railing faces X or Z based on which dimension is thin
	var is_x_facing := beam_size.x < beam_size.z
	if is_x_facing:
		# Plane faces X axis — size is (Z width, Y height)
		quad.size = Vector2(beam_size.z, beam_size.y)
		# QuadMesh faces +Z by default; rotate to face +X
		_mesh_instance.rotation_degrees.y = 90.0
	else:
		# Plane faces Z axis — size is (X width, Y height)
		quad.size = Vector2(beam_size.x, beam_size.y)
		# QuadMesh already faces +Z, no rotation needed

	var mat := ShaderMaterial.new()
	mat.shader = BEAM_SHADER
	mat.set_shader_parameter("beam_color", beam_color)
	mat.set_shader_parameter("core_color", core_color)
	mat.set_shader_parameter("scroll_speed", 0.8)
	mat.set_shader_parameter("pulse_speed", 1.5)
	mat.set_shader_parameter("pulse_strength", 0.3)
	mat.set_shader_parameter("fresnel_power", 2.5)
	mat.set_shader_parameter("fresnel_strength", 1.5)
	mat.set_shader_parameter("pattern_scale", 12.0)
	mat.set_shader_parameter("pattern_intensity", 0.4)
	mat.set_shader_parameter("core_width", 0.15)

	_mesh_instance.mesh = quad
	_mesh_instance.material_override = mat
	_mesh_instance.cast_shadow = GeometryInstance3D.SHADOW_CASTING_SETTING_OFF

	add_child(_mesh_instance)


func _build_particles() -> void:
	_particles = GPUParticles3D.new()
	_particles.name = "BeamParticles"
	_particles.emitting = true
	_particles.amount = 40
	_particles.lifetime = 2.0
	_particles.explosiveness = 0.0
	_particles.randomness = 0.5
	_particles.fixed_fps = 30
	_particles.visibility_aabb = AABB(Vector3(-4, -1, -4), Vector3(8, 3, 8))

	# Determine plane dimensions
	var is_x_facing := beam_size.x < beam_size.z
	var plane_width: float
	var plane_height: float = beam_size.y

	if is_x_facing:
		plane_width = beam_size.z
	else:
		plane_width = beam_size.x

	# Process material — controls particle behavior
	var proc_mat := ParticleProcessMaterial.new()
	proc_mat.direction = Vector3(0.0, 1.0, 0.0)
	proc_mat.spread = 15.0
	proc_mat.initial_velocity_min = 0.2
	proc_mat.initial_velocity_max = 0.5
	proc_mat.gravity = Vector3.ZERO
	proc_mat.damping_min = 0.5
	proc_mat.damping_max = 1.0

	# Emission shape: box covering the beam plane
	proc_mat.emission_shape = ParticleProcessMaterial.EMISSION_SHAPE_BOX
	if is_x_facing:
		proc_mat.emission_box_extents = Vector3(0.05, plane_height * 0.4, plane_width * 0.45)
	else:
		proc_mat.emission_box_extents = Vector3(plane_width * 0.45, plane_height * 0.4, 0.05)

	# Color: blue to white, fading out
	proc_mat.color = Color(0.4, 0.7, 1.0, 0.8)

	# Scale: small sparkles that shrink over lifetime
	proc_mat.scale_min = 0.6
	proc_mat.scale_max = 1.0

	_particles.process_material = proc_mat

	# Draw pass: tiny quad billboard for each particle
	var particle_mesh := QuadMesh.new()
	particle_mesh.size = Vector2(0.04, 0.04)

	var particle_mat := StandardMaterial3D.new()
	particle_mat.transparency = BaseMaterial3D.TRANSPARENCY_ALPHA
	particle_mat.shading_mode = BaseMaterial3D.SHADING_MODE_UNSHADED
	particle_mat.billboard_mode = BaseMaterial3D.BILLBOARD_ENABLED
	particle_mat.albedo_color = Color(0.5, 0.8, 1.0, 0.9)
	particle_mat.emission_enabled = true
	particle_mat.emission = Color(0.3, 0.6, 1.0)
	particle_mat.emission_energy_multiplier = 2.0

	particle_mesh.material = particle_mat
	_particles.draw_pass_1 = particle_mesh

	add_child(_particles)
