extends Node3D
## Reusable ground indicator for player AoE abilities.
## Spawns via static helpers, lingers briefly at full opacity, then fades out
## and frees itself. Friendly blue by default; color is caller-configurable.

enum Shape { CIRCLE, CONE }

var shape: Shape = Shape.CIRCLE
var radius: float = 5.0
var half_angle: float = PI / 4.0  # for cone only, radians
var telegraph_color: Color = Color(0.3, 0.7, 1.0, 0.4)
var edge_color: Color = Color(0.5, 0.85, 1.0, 0.9)
var linger_time: float = 0.15
var fade_time: float = 0.35

var _timer: float = 0.0
var _phase: int = 0  # 0=linger, 1=fade
var _material: ShaderMaterial

static var _circle_shader: Shader
static var _cone_shader: Shader
static var _self_script: GDScript


static func _get_self_script() -> GDScript:
	if _self_script == null:
		_self_script = load("res://scenes/shared/telegraph/player_telegraph.gd")
	return _self_script


static func spawn_circle(
	parent: Node, pos: Vector3, p_radius: float, p_color: Color = Color(0.3, 0.7, 1.0, 0.4)
) -> Node3D:
	var t: Node3D = _get_self_script().new()
	t.shape = Shape.CIRCLE
	t.radius = p_radius
	t.telegraph_color = Color(p_color.r, p_color.g, p_color.b, p_color.a)
	t.edge_color = Color(
		minf(p_color.r * 1.3, 1.0), minf(p_color.g * 1.3, 1.0), minf(p_color.b * 1.3, 1.0), 0.9
	)
	parent.add_child(t)
	t.global_position = pos + Vector3(0.0, 0.03, 0.0)
	return t


static func spawn_cone(
	parent: Node,
	pos: Vector3,
	rot_y: float,
	p_radius: float,
	half_angle_deg: float,
	p_color: Color = Color(0.3, 0.7, 1.0, 0.4)
) -> Node3D:
	var t: Node3D = _get_self_script().new()
	t.shape = Shape.CONE
	t.radius = p_radius
	t.half_angle = deg_to_rad(half_angle_deg)
	t.telegraph_color = Color(p_color.r, p_color.g, p_color.b, p_color.a)
	t.edge_color = Color(
		minf(p_color.r * 1.3, 1.0), minf(p_color.g * 1.3, 1.0), minf(p_color.b * 1.3, 1.0), 0.9
	)
	parent.add_child(t)
	t.global_position = pos + Vector3(0.0, 0.03, 0.0)
	t.rotation.y = rot_y
	return t


func _ready() -> void:
	top_level = true
	_load_shaders()
	_create_mesh()
	_timer = linger_time
	_phase = 0


func _process(delta: float) -> void:
	_timer -= delta
	if _phase == 0:
		# Linger phase -- full opacity
		if _timer <= 0.0:
			_phase = 1
			_timer = fade_time
	elif _phase == 1:
		# Fade phase
		var ratio: float = maxf(_timer / fade_time, 0.0)
		if _material:
			_material.set_shader_parameter("fade", ratio)
		if _timer <= 0.0:
			queue_free()


func _load_shaders() -> void:
	if _circle_shader == null:
		_circle_shader = load("res://assets/shaders/telegraph_circle.gdshader")
	if _cone_shader == null:
		_cone_shader = load("res://assets/shaders/telegraph_cone.gdshader")


func _create_mesh() -> void:
	var mesh_instance := MeshInstance3D.new()
	var mesh := PlaneMesh.new()
	mesh.size = Vector2(radius * 2.0, radius * 2.0)
	mesh.subdivide_width = 32
	mesh.subdivide_depth = 32
	mesh_instance.mesh = mesh

	_material = ShaderMaterial.new()
	if shape == Shape.CIRCLE:
		_material.shader = _circle_shader
	else:
		_material.shader = _cone_shader
		_material.set_shader_parameter("half_angle", half_angle)

	_material.set_shader_parameter("color", telegraph_color)
	_material.set_shader_parameter("edge_color", edge_color)
	_material.set_shader_parameter("edge_width", 0.08)
	_material.set_shader_parameter("fade", 1.0)
	mesh_instance.set_surface_override_material(0, _material)

	add_child(mesh_instance)
