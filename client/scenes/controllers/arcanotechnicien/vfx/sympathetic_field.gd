extends Node3D

## Sympathetic Field — persistent soft ground circle showing healing amplification radius.
## Very translucent, ambient glow. Always visible to all players.
## Uses telegraph_circle.gdshader at low opacity.

const HEAL_GREEN := Color(0.3, 1.0, 0.4)
const FADE_TIME: float = 0.3
const DEFAULT_RADIUS: float = 8.0

var _target: Node3D = null
var _material: ShaderMaterial = null
var _mesh_inst: MeshInstance3D = null

static var _circle_shader: Shader


func _ready() -> void:
	top_level = true


func start(target: Node3D, radius: float = DEFAULT_RADIUS) -> void:
	_target = target
	global_position = target.global_position + Vector3(0.0, 0.02, 0.0)
	_load_shader()
	_setup_ground_plane(radius)
	# Fade in
	_set_fade(0.0)
	var tween := get_tree().create_tween()
	tween.tween_method(_set_fade, 0.0, 1.0, FADE_TIME)


func stop() -> void:
	var tween := get_tree().create_tween()
	tween.tween_method(_set_fade, 1.0, 0.0, FADE_TIME)
	tween.tween_callback(queue_free)


func update_radius(radius: float) -> void:
	if _mesh_inst and _mesh_inst.mesh:
		_mesh_inst.mesh.size = Vector2(radius * 2.0, radius * 2.0)


func _set_fade(val: float) -> void:
	if _material:
		_material.set_shader_parameter("fade", val)


func _process(_delta: float) -> void:
	if _target and is_instance_valid(_target):
		global_position = _target.global_position + Vector3(0.0, 0.02, 0.0)


func _load_shader() -> void:
	if _circle_shader == null:
		_circle_shader = load("res://assets/shaders/telegraph_circle.gdshader")


func _setup_ground_plane(radius: float) -> void:
	_mesh_inst = MeshInstance3D.new()
	var plane := PlaneMesh.new()
	plane.size = Vector2(radius * 2.0, radius * 2.0)
	plane.subdivide_width = 16
	plane.subdivide_depth = 16
	_mesh_inst.mesh = plane
	_mesh_inst.cast_shadow = GeometryInstance3D.SHADOW_CASTING_SETTING_OFF

	_material = ShaderMaterial.new()
	_material.shader = _circle_shader
	_material.set_shader_parameter("color", Color(HEAL_GREEN, 0.0))
	_material.set_shader_parameter("edge_color", Color(HEAL_GREEN, 0.2))
	_material.set_shader_parameter("edge_width", 0.015)
	_material.set_shader_parameter("fade", 1.0)
	_mesh_inst.set_surface_override_material(0, _material)

	add_child(_mesh_inst)


