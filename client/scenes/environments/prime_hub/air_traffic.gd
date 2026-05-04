extends Node3D
## Air traffic — flying vehicles on east-west highway lanes at various depths.
## Lanes run perpendicular to north axis (along X).
## Multiple altitude layers, from between buildings up to near plaza level.

const SPEED_MIN := 60.0
const SPEED_MAX := 180.0

var _vehicles: Array[Dictionary] = []


func _ready() -> void:
	var rng := RandomNumberGenerator.new()
	rng.seed = 9999

	var mesh := BoxMesh.new()
	mesh.size = Vector3(4.0, 1.5, 8.0)

	var mat_warm := StandardMaterial3D.new()
	mat_warm.albedo_color = Color(0.9, 0.8, 0.5)
	mat_warm.emission_enabled = true
	mat_warm.emission = Color(0.95, 0.85, 0.5)
	mat_warm.emission_energy_multiplier = 4.0

	var mat_blue := StandardMaterial3D.new()
	mat_blue.albedo_color = Color(0.4, 0.55, 0.9)
	mat_blue.emission_enabled = true
	mat_blue.emission = Color(0.3, 0.5, 0.95)
	mat_blue.emission_energy_multiplier = 4.0

	# Define highway lanes — east-west lines at various Z depths and Y altitudes
	# [z_position, y_altitude, vehicles_per_lane, lane_half_length]
	var lanes := [
		# Close lanes — visible detail
		[-200.0, -50.0, 6, 800.0],
		[-200.0, -30.0, 5, 900.0],
		[-350.0, -80.0, 7, 1200.0],
		[-350.0, -40.0, 5, 1000.0],
		[-500.0, -100.0, 8, 1500.0],
		[-500.0, -60.0, 6, 1200.0],
		[-500.0, -20.0, 4, 1000.0],
		# Mid-distance lanes
		[-800.0, -120.0, 8, 2000.0],
		[-800.0, -70.0, 6, 1800.0],
		[-800.0, -30.0, 5, 1500.0],
		[-1200.0, -150.0, 10, 2500.0],
		[-1200.0, -80.0, 7, 2200.0],
		[-1200.0, -20.0, 5, 1800.0],
		# Far lanes — streaks of light in the distance
		[-1800.0, -120.0, 10, 3000.0],
		[-1800.0, -50.0, 8, 2500.0],
		[-2500.0, -100.0, 12, 3500.0],
		[-2500.0, -30.0, 8, 3000.0],
		[-3500.0, -80.0, 12, 4000.0],
		[-3500.0, -20.0, 8, 3500.0],
		# Also some east-west lanes on S/W/E sides
		[200.0, -60.0, 5, 800.0],
		[400.0, -80.0, 6, 1000.0],
	]

	# Also add a few north-south cross lanes for variety
	var cross_lanes := [
		[-300.0, -50.0, 6, 1000.0],  # [x_position, y, count, half_length]
		[300.0, -70.0, 6, 1200.0],
		[-600.0, -40.0, 5, 1500.0],
		[600.0, -60.0, 5, 1500.0],
	]

	var idx := 0

	# East-west lanes
	for lane in lanes:
		var lane_z: float = lane[0]
		var lane_y: float = lane[1]
		var count: int = lane[2]
		var half_len: float = lane[3]

		for i in count:
			var z_jitter: float = rng.randf_range(-15.0, 15.0)
			var y_jitter: float = rng.randf_range(-5.0, 5.0)
			var a := Vector3(-half_len, lane_y + y_jitter, lane_z + z_jitter + 10.0)
			var b := Vector3(half_len, lane_y + y_jitter, lane_z + z_jitter + 10.0)

			var mi := MeshInstance3D.new()
			mi.mesh = mesh
			mi.material_override = mat_warm if rng.randf() < 0.7 else mat_blue
			mi.name = "V_%d" % idx
			add_child(mi)

			var going_right: bool = rng.randf() < 0.5
			(
				_vehicles
				. append(
					{
						"node": mi,
						"a": a if going_right else b,
						"b": b if going_right else a,
						"speed": rng.randf_range(SPEED_MIN, SPEED_MAX),
						"t": rng.randf(),
					}
				)
			)
			idx += 1

	# North-south cross lanes
	for lane in cross_lanes:
		var lane_x: float = lane[0]
		var lane_y: float = lane[1]
		var count: int = lane[2]
		var half_len: float = lane[3]

		for i in count:
			var x_jitter: float = rng.randf_range(-15.0, 15.0)
			var y_jitter: float = rng.randf_range(-5.0, 5.0)
			var a := Vector3(lane_x + x_jitter, lane_y + y_jitter, -half_len + 10.0)
			var b := Vector3(lane_x + x_jitter, lane_y + y_jitter, half_len + 10.0)

			var mi := MeshInstance3D.new()
			mi.mesh = mesh
			mi.material_override = mat_warm if rng.randf() < 0.7 else mat_blue
			mi.name = "V_%d" % idx
			add_child(mi)

			var going_south: bool = rng.randf() < 0.5
			(
				_vehicles
				. append(
					{
						"node": mi,
						"a": a if going_south else b,
						"b": b if going_south else a,
						"speed": rng.randf_range(SPEED_MIN, SPEED_MAX),
						"t": rng.randf(),
					}
				)
			)
			idx += 1


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
