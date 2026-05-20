extends Node3D

## Vortex aura — spinning red arc ribbons around the player during forward dash.
## Multiple curved mesh strips at different heights/radii orbit the character,
## creating the WoW Whirlwind / Wukong spiral look. Radius matches 4.0 hit area.

const TRAIL_SHADER := preload("res://scenes/controllers/vanguard/vfx/swing_trail.gdshader")
const FADE_OUT_TIME: float = 0.3
const SPIN_SPEED: float = 4.0

## Large AABB — arcs are in world space, node is at origin
const HUGE_AABB := AABB(Vector3(-500, -500, -500), Vector3(1000, 1000, 1000))

var _target: Node3D = null
var _arcs: Array[Dictionary] = []
var _light: OmniLight3D = null
var _stopping: bool = false
var _fade_timer: float = 0.0
var _time: float = 0.0


func _ready() -> void:
	top_level = true


func start(target: Node3D) -> void:
	_target = target
	global_position = target.global_position

	# Define arc layers: each has a radius, height, arc span, width, speed multiplier
	var arc_defs: Array[Dictionary] = [
		# Outer ground ring — radius matches 4.0 server hit area
		{radius = 3.8, y = 0.15, arc_deg = 300.0, height = 0.25, speed_mult = 1.0, phase = 0.0},
		{radius = 3.5, y = 0.15, arc_deg = 240.0, height = 0.2, speed_mult = -0.8, phase = 2.0},
		# Mid-height arcs
		{radius = 2.8, y = 0.7, arc_deg = 270.0, height = 0.4, speed_mult = 1.3, phase = 1.0},
		{radius = 2.4, y = 1.0, arc_deg = 200.0, height = 0.35, speed_mult = -1.1, phase = 3.5},
		# Upper arcs — smaller, faster
		{radius = 1.8, y = 1.5, arc_deg = 220.0, height = 0.3, speed_mult = 1.6, phase = 0.5},
		{radius = 2.0, y = 1.8, arc_deg = 180.0, height = 0.3, speed_mult = -1.4, phase = 4.0},
		# Tight inner accent
		{radius = 1.2, y = 1.2, arc_deg = 160.0, height = 0.3, speed_mult = 2.0, phase = 2.5},
	]

	for def in arc_defs:
		var mesh_inst := MeshInstance3D.new()
		mesh_inst.cast_shadow = GeometryInstance3D.SHADOW_CASTING_SETTING_OFF
		mesh_inst.custom_aabb = HUGE_AABB
		add_child(mesh_inst)

		var mat := ShaderMaterial.new()
		mat.shader = TRAIL_SHADER
		# Inner arcs are brighter, outer arcs deeper red
		var brightness: float = 1.0 - (def.radius / 4.0) * 0.4
		mat.set_shader_parameter("core_color", Color(1.0, 0.4 * brightness, 0.2 * brightness, 1.0))
		mat.set_shader_parameter(
			"trail_color", Color(0.9, 0.15 * brightness, 0.1 * brightness, 0.9)
		)
		mat.set_shader_parameter("emission_boost", 2.5 + brightness)

		var arc_mesh := _build_arc_mesh(
			def.radius, def.y, def.height, deg_to_rad(def.arc_deg), 32, mat
		)
		mesh_inst.mesh = arc_mesh

		(
			_arcs
			. append(
				{
					mesh = mesh_inst,
					material = mat,
					speed_mult = def.speed_mult,
					phase = def.phase,
					angle = def.phase,
				}
			)
		)

	_setup_light()


func stop() -> void:
	_stopping = true
	_fade_timer = FADE_OUT_TIME


func _process(delta: float) -> void:
	_time += delta

	# Follow target
	if _target and is_instance_valid(_target):
		global_position = _target.global_position

	# Spin each arc independently
	for arc in _arcs:
		arc.angle += SPIN_SPEED * arc.speed_mult * delta
		arc.mesh.rotation.y = arc.angle

	# Pulse the light
	if _light:
		_light.light_energy = 3.5 + sin(_time * 8.0) * 1.0

	# Fade out
	if _stopping:
		_fade_timer -= delta
		var ratio: float = maxf(_fade_timer / FADE_OUT_TIME, 0.0)
		for arc in _arcs:
			arc.material.set_shader_parameter("fade", ratio)
		if _light:
			_light.light_energy *= ratio
		if _fade_timer <= 0.0:
			queue_free()


func _build_arc_mesh(
	radius: float,
	y_center: float,
	height: float,
	arc_span: float,
	segments: int,
	mat: ShaderMaterial
) -> ArrayMesh:
	var st := SurfaceTool.new()
	st.begin(Mesh.PRIMITIVE_TRIANGLE_STRIP)
	st.set_material(mat)

	var y_bottom: float = y_center - height * 0.5
	var y_top: float = y_center + height * 0.5

	for i in segments + 1:
		var t: float = float(i) / float(segments)
		var angle: float = -arc_span * 0.5 + arc_span * t

		var x: float = cos(angle) * radius
		var z: float = sin(angle) * radius

		# UV.y = t means 0=arc start (bright head), 1=arc end (fading tail)
		# UV.x = 0 bottom, 1 top
		st.set_normal(Vector3(cos(angle), 0.0, sin(angle)))
		st.set_uv(Vector2(0.0, t))
		st.add_vertex(Vector3(x, y_bottom, z))

		st.set_normal(Vector3(cos(angle), 0.0, sin(angle)))
		st.set_uv(Vector2(1.0, t))
		st.add_vertex(Vector3(x, y_top, z))

	return st.commit()


func _setup_light() -> void:
	_light = OmniLight3D.new()
	_light.light_color = Color(0.9, 0.15, 0.1)
	_light.light_energy = 3.5
	_light.omni_range = 10.0
	_light.shadow_enabled = false
	add_child(_light)
