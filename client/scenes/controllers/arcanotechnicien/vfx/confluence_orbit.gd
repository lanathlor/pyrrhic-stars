extends Node3D

## Confluence orbit particles — spinning arc meshes around the caster.
## Density and brightness scale with Confluence tier (0-3).
## Uses swing_trail.gdshader with green/cyan colors. Pattern from blade_swirl_aura.gd.

const TRAIL_SHADER := preload("res://scenes/controllers/vanguard/vfx/swing_trail.gdshader")
const SPIN_SPEED: float = 2.5
const HUGE_AABB := AABB(Vector3(-500, -500, -500), Vector3(1000, 1000, 1000))
const HEAL_GREEN := Color(0.3, 1.0, 0.4)
const FLUX_CYAN := Color(0.2, 0.9, 1.0)

## Arc definitions per tier. Each tier includes all arcs from lower tiers.
const TIER_ARCS: Array[Array] = [
	[],  # tier 0: nothing
	[  # tier 1: 2 subtle green arcs
		{radius = 1.2, y = 0.8, arc_deg = 120.0, height = 0.15, speed_mult = 1.0, phase = 0.0},
		{radius = 1.0, y = 1.2, arc_deg = 100.0, height = 0.15, speed_mult = -0.8, phase = 2.0},
	],
	[  # tier 2: 4 mixed arcs
		{radius = 1.3, y = 0.5, arc_deg = 140.0, height = 0.2, speed_mult = 1.2, phase = 0.0},
		{radius = 1.1, y = 0.9, arc_deg = 110.0, height = 0.2, speed_mult = -1.0, phase = 1.5},
		{radius = 0.9, y = 1.3, arc_deg = 130.0, height = 0.18, speed_mult = 1.3, phase = 3.0},
		{radius = 1.2, y = 1.6, arc_deg = 90.0, height = 0.15, speed_mult = -1.1, phase = 4.5},
	],
	[  # tier 3: 6 bright fast arcs
		{radius = 1.4, y = 0.3, arc_deg = 160.0, height = 0.25, speed_mult = 1.4, phase = 0.0},
		{radius = 1.2, y = 0.7, arc_deg = 130.0, height = 0.22, speed_mult = -1.2, phase = 1.0},
		{radius = 1.0, y = 1.0, arc_deg = 150.0, height = 0.22, speed_mult = 1.5, phase = 2.0},
		{radius = 0.8, y = 1.3, arc_deg = 120.0, height = 0.2, speed_mult = -1.3, phase = 3.5},
		{radius = 1.1, y = 1.6, arc_deg = 140.0, height = 0.2, speed_mult = 1.6, phase = 5.0},
		{radius = 0.9, y = 1.9, arc_deg = 100.0, height = 0.18, speed_mult = -1.5, phase = 4.0},
	],
]

var _target: Node3D = null
var _arcs: Array[Dictionary] = []
var _light: OmniLight3D = null
var _current_tier: int = 0
var _time: float = 0.0


func _ready() -> void:
	top_level = true


func start(target: Node3D) -> void:
	_target = target
	global_position = target.global_position


func update(tier: int, _stacks: int) -> void:
	tier = clampi(tier, 0, 3)
	if tier == _current_tier:
		return
	_current_tier = tier
	_rebuild_arcs()


func _rebuild_arcs() -> void:
	# Clear existing arcs
	for arc in _arcs:
		if is_instance_valid(arc.mesh):
			arc.mesh.queue_free()
	_arcs.clear()

	if _light:
		_light.queue_free()
		_light = null

	if _current_tier <= 0:
		return

	var arc_defs: Array = TIER_ARCS[_current_tier]
	var brightness_mult: float = 0.6 + 0.4 * (_current_tier / 3.0)

	for def in arc_defs:
		var mesh_inst := MeshInstance3D.new()
		mesh_inst.cast_shadow = GeometryInstance3D.SHADOW_CASTING_SETTING_OFF
		mesh_inst.custom_aabb = HUGE_AABB
		add_child(mesh_inst)

		var mat := ShaderMaterial.new()
		mat.shader = TRAIL_SHADER
		# Mix green and cyan based on height — lower arcs greener, upper arcs more cyan
		var cyan_blend: float = def.y / 2.0
		var core_col := HEAL_GREEN.lerp(FLUX_CYAN, cyan_blend)
		var trail_col := HEAL_GREEN.lerp(FLUX_CYAN, cyan_blend * 0.6)
		mat.set_shader_parameter("core_color", Color(core_col, 1.0))
		mat.set_shader_parameter("trail_color", Color(trail_col, 0.8))
		mat.set_shader_parameter("emission_boost", 2.0 + brightness_mult)

		var arc_mesh := _build_arc_mesh(
			{
				radius = def.radius,
				y_center = def.y,
				height = def.height,
				arc_span = deg_to_rad(def.arc_deg),
				segments = 24,
			},
			mat
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


func _process(delta: float) -> void:
	_time += delta

	if _target and is_instance_valid(_target):
		global_position = _target.global_position

	for arc in _arcs:
		arc.angle += SPIN_SPEED * arc.speed_mult * delta
		arc.mesh.rotation.y = arc.angle

	if _light:
		_light.light_energy = (1.0 + _current_tier) * (1.0 + sin(_time * 4.0) * 0.2)


func _build_arc_mesh(arc: Dictionary, mat: ShaderMaterial) -> ArrayMesh:
	var radius: float = arc.radius
	var y_center: float = arc.y_center
	var height: float = arc.height
	var arc_span: float = arc.arc_span
	var segments: int = arc.segments
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

		st.set_normal(Vector3(cos(angle), 0.0, sin(angle)))
		st.set_uv(Vector2(0.0, t))
		st.add_vertex(Vector3(x, y_bottom, z))

		st.set_normal(Vector3(cos(angle), 0.0, sin(angle)))
		st.set_uv(Vector2(1.0, t))
		st.add_vertex(Vector3(x, y_top, z))

	return st.commit()


func _setup_light() -> void:
	_light = OmniLight3D.new()
	_light.light_color = HEAL_GREEN.lerp(FLUX_CYAN, 0.3)
	_light.light_energy = 1.0 + _current_tier
	_light.omni_range = 5.0
	_light.shadow_enabled = false
	_light.position = Vector3(0.0, 1.0, 0.0)
	add_child(_light)
