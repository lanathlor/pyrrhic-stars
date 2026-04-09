extends Node3D
## Dungeon atmospheric effects: rain, lightning, and flickering accent lights.
## Attach to the Arena root after loading the scene.

@export var rain_enabled: bool = true
@export var lightning_enabled: bool = true
@export var flicker_enabled: bool = true

# Rain config
const RAIN_AREA := Vector3(50.0, 1.0, 80.0)  # covers full dungeon length
const RAIN_CENTER := Vector3(0.0, 15.0, 18.0)  # centered over dungeon
const RAIN_AMOUNT := 4000
const RAIN_LIFETIME := 1.2

# Lightning config
var _lightning_timer: float = 0.0
var _lightning_cooldown: float = 8.0
var _lightning_flash_time: float = 0.0
var _env: Environment

# Flicker config
var _flicker_lights: Array[OmniLight3D] = []
var _flicker_base_energy: Array[float] = []
var _flicker_timers: Array[float] = []

# Fire glow lights (orange accent — alien occupation fires)
var _fire_positions := [
	Vector3(-15.0, 1.5, -8.0),   # boss room left corner
	Vector3(14.0, 1.5, 5.0),     # boss room right
	Vector3(-5.0, 1.5, 22.0),    # hallway left
	Vector3(5.0, 1.5, 32.0),     # hallway right
	Vector3(-20.0, 2.0, -10.0),  # west building base
	Vector3(20.0, 2.0, 4.0),     # east building base
	Vector3(0.0, 2.0, 15.0),     # hallway entrance, rain shaft
]


func _ready() -> void:
	_env = _find_environment()

	if rain_enabled:
		_setup_rain()
	if flicker_enabled:
		_setup_fire_lights()
		_setup_hallway_flicker()
	if lightning_enabled:
		_lightning_cooldown = randf_range(5.0, 15.0)


func _process(delta: float) -> void:
	if lightning_enabled and _env:
		_process_lightning(delta)
	if flicker_enabled:
		_process_flicker(delta)


func _find_environment() -> Environment:
	var we := get_parent().get_node_or_null("WorldEnvironment") as WorldEnvironment
	if we:
		return we.environment
	return null


# === RAIN ===

func _setup_rain() -> void:
	var rain := GPUParticles3D.new()
	rain.name = "Rain"
	rain.amount = RAIN_AMOUNT
	rain.lifetime = RAIN_LIFETIME
	rain.visibility_aabb = AABB(-RAIN_AREA / 2.0, RAIN_AREA)
	rain.transform.origin = RAIN_CENTER

	var mat := ParticleProcessMaterial.new()
	# Gravity pulls rain down
	mat.gravity = Vector3(0.0, -20.0, 0.0)
	# Slight wind drift
	mat.direction = Vector3(0.2, -1.0, 0.1)
	mat.spread = 5.0
	mat.initial_velocity_min = 12.0
	mat.initial_velocity_max = 18.0
	# Emission shape: box covering the dungeon
	mat.emission_shape = ParticleProcessMaterial.EMISSION_SHAPE_BOX
	mat.emission_box_extents = RAIN_AREA / 2.0
	# Tiny scale for raindrops
	mat.scale_min = 0.8
	mat.scale_max = 1.2
	rain.process_material = mat

	# Mesh: thin stretched quad for raindrop streaks
	var mesh := QuadMesh.new()
	mesh.size = Vector2(0.015, 0.4)
	rain.draw_pass_1 = mesh

	# Translucent white-blue material for droplets
	var draw_mat := StandardMaterial3D.new()
	draw_mat.transparency = BaseMaterial3D.TRANSPARENCY_ALPHA
	draw_mat.albedo_color = Color(0.6, 0.65, 0.8, 0.3)
	draw_mat.emission_enabled = true
	draw_mat.emission = Color(0.1, 0.12, 0.2)
	draw_mat.emission_energy_multiplier = 0.3
	draw_mat.billboard_mode = BaseMaterial3D.BILLBOARD_ENABLED
	draw_mat.shading_mode = BaseMaterial3D.SHADING_MODE_UNSHADED
	mesh.material = draw_mat

	add_child(rain)

	# Rain splash particles on the floor
	_setup_rain_splashes()


func _setup_rain_splashes() -> void:
	var splash := GPUParticles3D.new()
	splash.name = "RainSplashes"
	splash.amount = 800
	splash.lifetime = 0.3
	splash.transform.origin = Vector3(0.0, 0.05, 18.0)
	splash.visibility_aabb = AABB(Vector3(-25, -1, -20), Vector3(50, 3, 75))

	var mat := ParticleProcessMaterial.new()
	mat.gravity = Vector3(0.0, -8.0, 0.0)
	mat.direction = Vector3(0.0, 1.0, 0.0)
	mat.spread = 60.0
	mat.initial_velocity_min = 1.0
	mat.initial_velocity_max = 2.5
	mat.emission_shape = ParticleProcessMaterial.EMISSION_SHAPE_BOX
	mat.emission_box_extents = Vector3(20.0, 0.1, 35.0)
	mat.scale_min = 0.5
	mat.scale_max = 1.0
	splash.process_material = mat

	var mesh := SphereMesh.new()
	mesh.radius = 0.02
	mesh.height = 0.04

	var draw_mat := StandardMaterial3D.new()
	draw_mat.transparency = BaseMaterial3D.TRANSPARENCY_ALPHA
	draw_mat.albedo_color = Color(0.5, 0.55, 0.7, 0.2)
	draw_mat.shading_mode = BaseMaterial3D.SHADING_MODE_UNSHADED
	mesh.material = draw_mat

	splash.draw_pass_1 = mesh
	add_child(splash)


# === LIGHTNING ===

func _process_lightning(delta: float) -> void:
	_lightning_timer += delta

	if _lightning_flash_time > 0.0:
		_lightning_flash_time -= delta
		# Fade flash out
		var intensity: float = clampf(_lightning_flash_time / 0.15, 0.0, 1.0)
		_env.ambient_light_energy = lerpf(1.2, 3.5, intensity)
		if _lightning_flash_time <= 0.0:
			_env.ambient_light_energy = 1.2
		return

	if _lightning_timer >= _lightning_cooldown:
		_lightning_timer = 0.0
		_lightning_cooldown = randf_range(6.0, 20.0)
		_trigger_lightning()


func _trigger_lightning() -> void:
	_lightning_flash_time = 0.15 + randf() * 0.1
	_env.ambient_light_energy = 2.5
	# Double flash sometimes
	if randf() < 0.3:
		var timer := get_tree().create_timer(0.25)
		timer.timeout.connect(_second_flash)


func _second_flash() -> void:
	if _env:
		_lightning_flash_time = 0.1
		_env.ambient_light_energy = 1.8


# === FIRE GLOW / FLICKERING LIGHTS ===

func _setup_fire_lights() -> void:
	for pos in _fire_positions:
		var light := OmniLight3D.new()
		light.name = "FireGlow"
		light.transform.origin = pos
		light.light_color = Color(0.9, 0.45, 0.1)
		light.light_energy = 2.5
		light.omni_range = 8.0
		add_child(light)

		_flicker_lights.append(light)
		_flicker_base_energy.append(2.5)
		_flicker_timers.append(randf() * TAU)


func _setup_hallway_flicker() -> void:
	# Hallway overhead lights that flicker like damaged fixtures
	var hallway_light_positions := [
		Vector3(0.0, 3.5, 32.0),
		Vector3(0.0, 3.5, 22.0),
		Vector3(0.0, 4.5, 12.5),   # transition building underpass
	]
	for pos in hallway_light_positions:
		var light := OmniLight3D.new()
		light.name = "HallwayFlicker"
		light.transform.origin = pos
		light.light_color = Color(0.4, 0.45, 0.7)
		light.light_energy = 2.0
		light.omni_range = 12.0
		add_child(light)

		_flicker_lights.append(light)
		_flicker_base_energy.append(2.0)
		_flicker_timers.append(randf() * TAU)


func _process_flicker(delta: float) -> void:
	for i in _flicker_lights.size():
		_flicker_timers[i] += delta * (3.0 + sin(Time.get_ticks_msec() * 0.001 + i) * 2.0)
		var flicker: float = sin(_flicker_timers[i] * 4.0) * 0.3 + sin(_flicker_timers[i] * 7.3) * 0.15
		# Fire lights: warm flicker. Hallway lights: occasional dropout
		if _flicker_lights[i].light_color.r > 0.6:
			# Fire glow — gentle flicker
			_flicker_lights[i].light_energy = _flicker_base_energy[i] + flicker
		else:
			# Hallway light — harsh flicker with occasional dropout
			var dropout: float = 1.0
			if sin(_flicker_timers[i] * 11.0) > 0.85:
				dropout = 0.1
			_flicker_lights[i].light_energy = (_flicker_base_energy[i] + flicker) * dropout
