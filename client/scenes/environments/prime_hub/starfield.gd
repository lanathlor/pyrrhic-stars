extends Node3D
## Procedural starfield — hundreds of tiny emissive spheres on a sky dome.

const STAR_COUNT := 500
const DOME_RADIUS := 5000.0
const MIN_ELEVATION := 0.15  # radians above horizon — no stars below ~9°


func _ready() -> void:
	var rng := RandomNumberGenerator.new()
	rng.seed = 7777

	var xforms: Array[Transform3D] = []

	for _i in STAR_COUNT:
		# Random point on upper hemisphere
		var azimuth: float = rng.randf() * TAU
		var elevation: float = rng.randf_range(MIN_ELEVATION, PI * 0.48)

		var x: float = cos(azimuth) * cos(elevation) * DOME_RADIUS
		var z: float = sin(azimuth) * cos(elevation) * DOME_RADIUS
		var y: float = sin(elevation) * DOME_RADIUS

		# Varying sizes — most tiny, a few brighter
		var size: float
		var roll: float = rng.randf()
		if roll < 0.7:
			size = rng.randf_range(1.0, 2.5)
		elif roll < 0.92:
			size = rng.randf_range(2.5, 5.0)
		else:
			size = rng.randf_range(5.0, 9.0)

		xforms.append(Transform3D(
			Basis.from_scale(Vector3(size, size, size)),
			Vector3(x, y, z)))

	# Build multimesh
	var mesh := SphereMesh.new()
	mesh.radius = 1.0
	mesh.height = 2.0
	mesh.radial_segments = 4
	mesh.rings = 2

	var mm := MultiMesh.new()
	mm.transform_format = MultiMesh.TRANSFORM_3D
	mm.instance_count = xforms.size()
	mm.mesh = mesh
	for i in xforms.size():
		mm.set_instance_transform(i, xforms[i])

	var mmi := MultiMeshInstance3D.new()
	mmi.multimesh = mm
	mmi.name = "StarField"

	var mat := StandardMaterial3D.new()
	mat.albedo_color = Color(1, 1, 1)
	mat.emission_enabled = true
	mat.emission = Color(1, 0.98, 0.9)
	mat.emission_energy_multiplier = 4.0
	mat.roughness = 0.0
	mmi.material_override = mat

	add_child(mmi)
