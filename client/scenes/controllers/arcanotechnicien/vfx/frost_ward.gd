extends Node3D

## Frost Ward shield — hex-pattern energy barrier on target ally.
## Reuses block_shield.gdshader with frost blue color.
## Follows target, auto-frees after duration or on fade_out().

const SHIELD_SHADER := preload("res://scenes/controllers/vanguard/vfx/block_shield.gdshader")
const FROST_BLUE := Color(0.4, 0.7, 1.0)
const FADE_IN_TIME: float = 0.12
const FADE_OUT_TIME: float = 0.15
const DURATION: float = 6.0

var _target: Node3D = null
var _material: ShaderMaterial = null
var _fade_tween: Tween = null
var _duration_timer: float = 0.0


func _ready() -> void:
	top_level = true


func start(target: Node3D) -> void:
	_target = target
	_duration_timer = DURATION
	global_position = target.global_position + Vector3(0.0, 1.0, 0.0)
	_setup_shield_mesh()
	_setup_edge_particles()
	_setup_light()
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


func _process(delta: float) -> void:
	# Follow target
	if _target and is_instance_valid(_target):
		global_position = _target.global_position + Vector3(0.0, 1.0, 0.0)
	else:
		fade_out()
		return

	# Auto-expire
	_duration_timer -= delta
	if _duration_timer <= 0.0:
		fade_out()


func _setup_shield_mesh() -> void:
	var mesh_inst := MeshInstance3D.new()
	var quad := QuadMesh.new()
	quad.size = Vector2(2.0, 2.0)
	quad.subdivide_width = 8
	quad.subdivide_depth = 8
	mesh_inst.mesh = quad
	mesh_inst.cast_shadow = GeometryInstance3D.SHADOW_CASTING_SETTING_OFF

	_material = ShaderMaterial.new()
	_material.shader = SHIELD_SHADER
	_material.set_shader_parameter("shield_color", Color(FROST_BLUE, 0.6))
	_material.set_shader_parameter("pulse_speed", 1.5)
	mesh_inst.set_surface_override_material(0, _material)

	# Billboard so it always faces camera
	mesh_inst.billboard = GeometryInstance3D.VISIBILITY_RANGE_FADE_DISABLED
	add_child(mesh_inst)


func _setup_edge_particles() -> void:
	var particles := GPUParticles3D.new()
	particles.amount = 12
	particles.lifetime = 0.6
	particles.cast_shadow = GeometryInstance3D.SHADOW_CASTING_SETTING_OFF

	var mat := ParticleProcessMaterial.new()
	mat.emission_shape = ParticleProcessMaterial.EMISSION_SHAPE_BOX
	mat.emission_box_extents = Vector3(0.8, 0.8, 0.02)
	mat.initial_velocity_min = 0.2
	mat.initial_velocity_max = 0.5
	mat.gravity = Vector3.ZERO
	mat.scale_min = 0.5
	mat.scale_max = 1.0
	particles.process_material = mat

	var quad := QuadMesh.new()
	quad.size = Vector2(0.08, 0.08)
	var draw_mat := StandardMaterial3D.new()
	draw_mat.albedo_color = Color(FROST_BLUE, 0.8)
	draw_mat.emission_enabled = true
	draw_mat.emission = FROST_BLUE
	draw_mat.emission_energy_multiplier = 4.0
	draw_mat.shading_mode = BaseMaterial3D.SHADING_MODE_UNSHADED
	draw_mat.billboard_mode = BaseMaterial3D.BILLBOARD_PARTICLES
	draw_mat.transparency = BaseMaterial3D.TRANSPARENCY_ALPHA
	quad.material = draw_mat
	particles.draw_pass_1 = quad

	add_child(particles)


func _setup_light() -> void:
	var light := OmniLight3D.new()
	light.light_color = FROST_BLUE
	light.light_energy = 2.0
	light.omni_range = 3.0
	light.shadow_enabled = false
	add_child(light)
