extends Node3D
## Air traffic — flying vehicles on east-west highway lanes at various depths.
## Lanes run perpendicular to north axis (along X).
## Multiple altitude layers, from between buildings up to near plaza level.

const SPEED_MIN := 60.0
const SPEED_MAX := 180.0

## East-west highway lanes: [z_position, y_altitude, vehicles_per_lane, lane_half_length]
const _EW_LANES := [
	[-200.0, -50.0, 6, 800.0],
	[-200.0, -30.0, 5, 900.0],
	[-350.0, -80.0, 7, 1200.0],
	[-350.0, -40.0, 5, 1000.0],
	[-500.0, -100.0, 8, 1500.0],
	[-500.0, -60.0, 6, 1200.0],
	[-500.0, -20.0, 4, 1000.0],
	[-800.0, -120.0, 8, 2000.0],
	[-800.0, -70.0, 6, 1800.0],
	[-800.0, -30.0, 5, 1500.0],
	[-1200.0, -150.0, 10, 2500.0],
	[-1200.0, -80.0, 7, 2200.0],
	[-1200.0, -20.0, 5, 1800.0],
	[-1800.0, -120.0, 10, 3000.0],
	[-1800.0, -50.0, 8, 2500.0],
	[-2500.0, -100.0, 12, 3500.0],
	[-2500.0, -30.0, 8, 3000.0],
	[-3500.0, -80.0, 12, 4000.0],
	[-3500.0, -20.0, 8, 3500.0],
	[200.0, -60.0, 5, 800.0],
	[400.0, -80.0, 6, 1000.0],
]

## North-south cross lanes: [x_position, y, count, half_length]
const _NS_LANES := [
	[-300.0, -50.0, 6, 1000.0],
	[300.0, -70.0, 6, 1200.0],
	[-600.0, -40.0, 5, 1500.0],
	[600.0, -60.0, 5, 1500.0],
]

var _vehicles: Array[Dictionary] = []
var _mesh: BoxMesh
var _mat_warm: StandardMaterial3D
var _mat_blue: StandardMaterial3D
var _idx: int = 0


func _ready() -> void:
	var rng := RandomNumberGenerator.new()
	rng.seed = 9999

	_mesh = BoxMesh.new()
	_mesh.size = Vector3(4.0, 1.5, 8.0)
	_mat_warm = _make_vehicle_mat(Color(0.9, 0.8, 0.5), Color(0.95, 0.85, 0.5))
	_mat_blue = _make_vehicle_mat(Color(0.4, 0.55, 0.9), Color(0.3, 0.5, 0.95))

	_spawn_ew_vehicles(rng)
	_spawn_ns_vehicles(rng)


func _make_vehicle_mat(albedo: Color, emission: Color) -> StandardMaterial3D:
	var mat := StandardMaterial3D.new()
	mat.albedo_color = albedo
	mat.emission_enabled = true
	mat.emission = emission
	mat.emission_energy_multiplier = 4.0
	return mat


func _spawn_ew_vehicles(rng: RandomNumberGenerator) -> void:
	for lane in _EW_LANES:
		var lane_z: float = lane[0]
		var lane_y: float = lane[1]
		var count: int = lane[2]
		var half_len: float = lane[3]
		for i in count:
			var z_jitter: float = rng.randf_range(-15.0, 15.0)
			var y_jitter: float = rng.randf_range(-5.0, 5.0)
			var a := Vector3(-half_len, lane_y + y_jitter, lane_z + z_jitter + 10.0)
			var b := Vector3(half_len, lane_y + y_jitter, lane_z + z_jitter + 10.0)
			_add_vehicle(rng, a, b)


func _spawn_ns_vehicles(rng: RandomNumberGenerator) -> void:
	for lane in _NS_LANES:
		var lane_x: float = lane[0]
		var lane_y: float = lane[1]
		var count: int = lane[2]
		var half_len: float = lane[3]
		for i in count:
			var x_jitter: float = rng.randf_range(-15.0, 15.0)
			var y_jitter: float = rng.randf_range(-5.0, 5.0)
			var a := Vector3(lane_x + x_jitter, lane_y + y_jitter, -half_len + 10.0)
			var b := Vector3(lane_x + x_jitter, lane_y + y_jitter, half_len + 10.0)
			_add_vehicle(rng, a, b)


func _add_vehicle(rng: RandomNumberGenerator, a: Vector3, b: Vector3) -> void:
	var mi := MeshInstance3D.new()
	mi.mesh = _mesh
	mi.material_override = _mat_warm if rng.randf() < 0.7 else _mat_blue
	mi.name = "V_%d" % _idx
	add_child(mi)
	var forward: bool = rng.randf() < 0.5
	(
		_vehicles
		. append(
			{
				"node": mi,
				"a": a if forward else b,
				"b": b if forward else a,
				"speed": rng.randf_range(SPEED_MIN, SPEED_MAX),
				"t": rng.randf(),
			}
		)
	)
	_idx += 1


func _process(delta: float) -> void:
	for v in _vehicles:
		var node: MeshInstance3D = v["node"]
		var a: Vector3 = v["a"]
		var b: Vector3 = v["b"]
		var speed: float = v["speed"]
		var lane_len: float = a.distance_to(b)
		if lane_len < 0.001:
			continue
		var dt: float = (speed * delta) / lane_len

		v["t"] += dt
		if v["t"] >= 1.0:
			v["t"] -= 1.0  # loop seamlessly

		node.position = a.lerp(b, v["t"])
		if node.position.distance_squared_to(b) > 0.001:
			node.look_at(b)
