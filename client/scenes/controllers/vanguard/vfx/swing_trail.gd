extends Node3D

## Sword swing trail — a curved red arc that sweeps through the attack zone.
## Generates a 120° arc mesh at the player's melee range and animates a reveal/fade.
## One-shot: spawns, sweeps, fades, frees itself.

const TRAIL_SHADER := preload("res://scenes/controllers/vanguard/vfx/swing_trail.gdshader")

## Arc geometry
const ARC_DEGREES: float = 120.0
const ARC_RADIUS: float = 2.8
const ARC_Y_BOTTOM: float = 0.8
const ARC_Y_TOP: float = 1.6
const ARC_SEGMENTS: int = 24

## Timing
const SWEEP_DURATION: float = 0.25
const LINGER_DURATION: float = 0.05
const FADE_DURATION: float = 0.15

## Large AABB — vertices in world space, node at origin
const HUGE_AABB := AABB(Vector3(-500, -500, -500), Vector3(1000, 1000, 1000))

var _mesh_instance: MeshInstance3D = null
var _glow_instance: MeshInstance3D = null
var _material: ShaderMaterial = null
var _glow_material: ShaderMaterial = null
var _light: OmniLight3D = null

var _sweep_progress: float = 0.0
var _phase: int = 0  # 0=sweep, 1=linger, 2=fade
var _timer: float = 0.0
var _origin: Vector3 = Vector3.ZERO
var _facing_angle: float = 0.0


func _ready() -> void:
	top_level = true


## Call start() after adding to scene tree. Pass the player node.
func start(player: Node3D) -> void:
	_origin = player.global_position
	_facing_angle = player.rotation.y

	# Main arc
	_material = ShaderMaterial.new()
	_material.shader = TRAIL_SHADER
	_material.set_shader_parameter("emission_boost", 3.5)

	_mesh_instance = MeshInstance3D.new()
	_mesh_instance.cast_shadow = GeometryInstance3D.SHADOW_CASTING_SETTING_OFF
	_mesh_instance.custom_aabb = HUGE_AABB
	add_child(_mesh_instance)

	# Wider glow arc
	_glow_material = ShaderMaterial.new()
	_glow_material.shader = TRAIL_SHADER
	_glow_material.set_shader_parameter("trail_color", Color(0.9, 0.15, 0.1, 0.5))
	_glow_material.set_shader_parameter("core_color", Color(1.0, 0.4, 0.2, 0.6))
	_glow_material.set_shader_parameter("emission_boost", 1.8)

	_glow_instance = MeshInstance3D.new()
	_glow_instance.cast_shadow = GeometryInstance3D.SHADOW_CASTING_SETTING_OFF
	_glow_instance.custom_aabb = HUGE_AABB
	add_child(_glow_instance)

	# Build static arc meshes
	_mesh_instance.mesh = _build_arc(ARC_RADIUS, ARC_Y_BOTTOM, ARC_Y_TOP, _material)
	_glow_instance.mesh = _build_arc(
		ARC_RADIUS + 0.3, ARC_Y_BOTTOM - 0.1, ARC_Y_TOP + 0.1, _glow_material
	)

	# Position and rotate to match player facing
	_mesh_instance.global_position = _origin
	_mesh_instance.rotation.y = _facing_angle
	_glow_instance.global_position = _origin
	_glow_instance.rotation.y = _facing_angle

	# Light follows the arc center
	_light = OmniLight3D.new()
	_light.light_color = Color(0.9, 0.15, 0.1)
	_light.light_energy = 5.0
	_light.omni_range = 5.0
	_light.shadow_enabled = false
	add_child(_light)
	var mid_angle: float = _facing_angle
	_light.global_position = (
		_origin
		+ Vector3(sin(mid_angle) * ARC_RADIUS * 0.6, 1.2, -cos(mid_angle) * ARC_RADIUS * 0.6)
	)

	_phase = 0
	_timer = SWEEP_DURATION
	_sweep_progress = 0.0
	_material.set_shader_parameter("reveal", 0.0)
	_glow_material.set_shader_parameter("reveal", 0.0)


## Stop early (e.g., interrupted). Starts fade immediately.
func stop() -> void:
	if _phase < 2:
		_phase = 2
		_timer = FADE_DURATION


func _process(delta: float) -> void:
	_timer -= delta

	match _phase:
		0:  # Sweep — progressively reveal the arc
			_sweep_progress = 1.0 - maxf(_timer / SWEEP_DURATION, 0.0)
			_material.set_shader_parameter("reveal", _sweep_progress)
			_glow_material.set_shader_parameter("reveal", _sweep_progress)
			_material.set_shader_parameter("fade", 1.0)
			_glow_material.set_shader_parameter("fade", 0.6)
			if _timer <= 0.0:
				_phase = 1
				_timer = LINGER_DURATION
		1:  # Linger at full brightness
			_material.set_shader_parameter("reveal", 1.0)
			_glow_material.set_shader_parameter("reveal", 1.0)
			if _timer <= 0.0:
				_phase = 2
				_timer = FADE_DURATION
		2:  # Fade out
			var ratio: float = maxf(_timer / FADE_DURATION, 0.0)
			_material.set_shader_parameter("fade", ratio)
			_glow_material.set_shader_parameter("fade", ratio * 0.6)
			if _light:
				_light.light_energy = 5.0 * ratio
			if _timer <= 0.0:
				queue_free()


func _build_arc(radius: float, y_bottom: float, y_top: float, mat: ShaderMaterial) -> ArrayMesh:
	var st := SurfaceTool.new()
	st.begin(Mesh.PRIMITIVE_TRIANGLE_STRIP)
	st.set_material(mat)

	var arc_rad: float = deg_to_rad(ARC_DEGREES)

	for i in ARC_SEGMENTS + 1:
		var t: float = float(i) / float(ARC_SEGMENTS)
		# Arc sweeps from right to left in front of the player (local -Z forward)
		# Centered on forward, spanning ARC_DEGREES
		var angle: float = -arc_rad * 0.5 + arc_rad * t

		# In local space: forward is -Z, right is +X
		var x: float = sin(angle) * radius
		var z: float = -cos(angle) * radius

		var normal := Vector3(sin(angle), 0.0, -cos(angle))

		# UV.y = t → 0=arc start (bright), 1=arc end (fading)
		st.set_normal(normal)
		st.set_uv(Vector2(0.0, 1.0 - t))
		st.add_vertex(Vector3(x, y_bottom, z))

		st.set_normal(normal)
		st.set_uv(Vector2(1.0, 1.0 - t))
		st.add_vertex(Vector3(x, y_top, z))

	return st.commit()
