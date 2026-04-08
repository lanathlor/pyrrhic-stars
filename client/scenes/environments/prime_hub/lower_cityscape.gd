extends Node3D
## Small cityscape at Y=-150, barely poking above the lower landing level.
## Clears the lift shaft area.

const GROUND_Y := -150.0
const EXTENT := 800.0
const SPACING := 25.0

# Shaft exclusion zone (generous margin around lift at X=5, Z=-55)
const SHAFT_X_MIN := -10.0
const SHAFT_X_MAX := 20.0
const SHAFT_Z_MIN := -70.0
const SHAFT_Z_MAX := -40.0

# Also skip the south building box (Z=50 to 157) and plaza above (X ±125, Z -115 to 157)
const PLAZA_X := 135.0
const PLAZA_Z_MIN := -125.0
const PLAZA_Z_MAX := 165.0


func _ready() -> void:
	var rng := RandomNumberGenerator.new()
	rng.seed = 99

	var bldg: Array[Transform3D] = []
	var wins: Array[Transform3D] = []

	for gx in range(-32, 33):
		for gz in range(-32, 33):
			var cx: float = gx * SPACING + rng.randf_range(-8, 8)
			var cz: float = gz * SPACING + rng.randf_range(-8, 8)

			# Skip plaza / south building footprint
			if cx > -PLAZA_X and cx < PLAZA_X and cz > PLAZA_Z_MIN and cz < PLAZA_Z_MAX:
				continue

			# Skip shaft exclusion zone
			if cx > SHAFT_X_MIN and cx < SHAFT_X_MAX and cz > SHAFT_Z_MIN and cz < SHAFT_Z_MAX:
				continue

			var dist := maxf(absf(cx), absf(cz))
			if dist > EXTENT:
				continue

			if rng.randf() < 0.3:
				continue

			var w: float = rng.randf_range(6, 18)
			var d: float = rng.randf_range(5, 15)
			var h: float = rng.randf_range(8, 45)

			var y: float = GROUND_Y + h * 0.5
			bldg.append(Transform3D(
				Basis.from_scale(Vector3(w, h, d)),
				Vector3(cx, y, cz)))

			# Occasional window strip
			if h > 15.0 and rng.randf() < 0.25:
				var win_h: float = h * 0.5
				var fx: float = cx + (w * 0.5 + 0.1) * (1.0 if rng.randf() > 0.5 else -1.0)
				wins.append(Transform3D(
					Basis.from_scale(Vector3(0.2, win_h, d * 0.35)),
					Vector3(fx, GROUND_Y + h * 0.35, cz)))

	# Buildings (no ground plane — UndergroundBase box already covers the floor)
	_mm("LowerBuildings", bldg, _solid(Color(0.35, 0.36, 0.38), 0.75))
	_mm("LowerWindows", wins, _glow(Color(0.4, 0.35, 0.25), Color(0.6, 0.5, 0.3), 0.6))


func _mm(label: String, xforms: Array[Transform3D], mat: Material) -> void:
	if xforms.is_empty():
		return
	var mesh := BoxMesh.new()
	mesh.size = Vector3.ONE
	var mm := MultiMesh.new()
	mm.transform_format = MultiMesh.TRANSFORM_3D
	mm.mesh = mesh
	mm.instance_count = xforms.size()
	for i in xforms.size():
		mm.set_instance_transform(i, xforms[i])
	var mmi := MultiMeshInstance3D.new()
	mmi.name = label
	mmi.multimesh = mm
	mmi.material_override = mat
	add_child(mmi)


func _solid(color: Color, rough: float) -> StandardMaterial3D:
	var m := StandardMaterial3D.new()
	m.albedo_color = color
	m.roughness = rough
	return m


func _glow(color: Color, emission: Color, energy: float) -> StandardMaterial3D:
	var m := StandardMaterial3D.new()
	m.albedo_color = color
	m.emission_enabled = true
	m.emission = emission
	m.emission_energy_multiplier = energy
	m.roughness = 0.4
	return m
