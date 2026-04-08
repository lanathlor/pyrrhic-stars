extends Node3D
## Lower level street network at Y=-150.
## Warm orange palette, spacious streets radiating from the lift landing.
## Lift landing at (5, -150, -55).

const GROUND_Y := -150.0
const LIFT_X := 5.0
const LIFT_Z := -55.0

# Street network covers ~200m x 200m around the lift
const AREA_MIN_X := -95.0
const AREA_MAX_X := 105.0
const AREA_MIN_Z := -155.0
const AREA_MAX_Z := 45.0

# Street widths
const MAIN_STREET_W := 14.0
const SIDE_STREET_W := 10.0

# Building height range
const MIN_HEIGHT := 8.0
const MAX_HEIGHT := 30.0

# Ceiling height above ground (the upper plaza floor sits at Y~-0.3)
const CEILING_Y := -5.0


func _ready() -> void:
	_build_ground()
	_build_ceiling()
	_build_streets()
	_build_buildings()
	_build_lighting()


func _build_ground() -> void:
	# Main ground plane for the lower level
	var floor_mesh := BoxMesh.new()
	floor_mesh.size = Vector3(200, 0.3, 200)
	var floor_mi := MeshInstance3D.new()
	floor_mi.name = "LowerFloor"
	floor_mi.mesh = floor_mesh
	floor_mi.position = Vector3(LIFT_X, GROUND_Y - 0.15, LIFT_Z)
	floor_mi.material_override = _mat(Color(0.3, 0.28, 0.25), 0.8)
	add_child(floor_mi)

	# Floor collision
	var body := StaticBody3D.new()
	body.name = "LowerFloorBody"
	body.collision_layer = 1
	body.collision_mask = 0
	var shape := BoxShape3D.new()
	shape.size = Vector3(200, 0.3, 200)
	var col := CollisionShape3D.new()
	col.shape = shape
	col.position = Vector3(LIFT_X, GROUND_Y - 0.15, LIFT_Z)
	body.add_child(col)
	add_child(body)

	# Central plaza area around lift (40m x 40m, polished surface)
	var plaza_mesh := BoxMesh.new()
	plaza_mesh.size = Vector3(40, 0.05, 40)
	var plaza_mi := MeshInstance3D.new()
	plaza_mi.name = "CentralPlaza"
	plaza_mi.mesh = plaza_mesh
	plaza_mi.position = Vector3(LIFT_X, GROUND_Y + 0.03, LIFT_Z)
	plaza_mi.material_override = _mat(Color(0.38, 0.34, 0.3), 0.6)
	add_child(plaza_mi)


func _build_ceiling() -> void:
	# Ceiling represents the underside of the upper plaza
	var ceil_mesh := BoxMesh.new()
	ceil_mesh.size = Vector3(250, 0.5, 250)
	var ceil_mi := MeshInstance3D.new()
	ceil_mi.name = "LowerCeiling"
	ceil_mi.mesh = ceil_mesh
	ceil_mi.position = Vector3(LIFT_X, CEILING_Y, LIFT_Z)
	ceil_mi.material_override = _mat(Color(0.22, 0.2, 0.18), 0.9)
	add_child(ceil_mi)


func _build_streets() -> void:
	# Street pavement (slightly raised, lighter than ground)
	var street_mat := _mat(Color(0.35, 0.32, 0.28), 0.7)

	# Main north-south boulevard through the lift area
	_add_street("MainNS", Vector3(LIFT_X, GROUND_Y + 0.02, LIFT_Z),
		Vector3(MAIN_STREET_W, 0.04, 180), street_mat)

	# Main east-west boulevard through the lift area
	_add_street("MainEW", Vector3(LIFT_X, GROUND_Y + 0.02, LIFT_Z),
		Vector3(180, 0.04, MAIN_STREET_W), street_mat)

	# Side streets (grid every 50m, offset from center)
	for i in range(-2, 3):
		if i == 0:
			continue  # skip center (main boulevard)
		var offset := i * 50.0
		# N-S side streets
		_add_street("SideNS%d" % i,
			Vector3(LIFT_X + offset, GROUND_Y + 0.02, LIFT_Z),
			Vector3(SIDE_STREET_W, 0.04, 160), street_mat)
		# E-W side streets
		_add_street("SideEW%d" % i,
			Vector3(LIFT_X, GROUND_Y + 0.02, LIFT_Z + offset),
			Vector3(160, 0.04, SIDE_STREET_W), street_mat)


func _add_street(label: String, pos: Vector3, size: Vector3, mat: Material) -> void:
	var mesh := BoxMesh.new()
	mesh.size = size
	var mi := MeshInstance3D.new()
	mi.name = label
	mi.mesh = mesh
	mi.position = pos
	mi.material_override = mat
	add_child(mi)


func _build_buildings() -> void:
	var rng := RandomNumberGenerator.new()
	rng.seed = 42

	var bldg_xforms: Array[Transform3D] = []
	var window_xforms: Array[Transform3D] = []
	var sign_xforms: Array[Transform3D] = []

	# Buildings placed in blocks between streets
	# Street grid: main at center, sides at ±50, ±100
	# Blocks are between streets, buildings cluster within blocks
	var street_positions := [-100.0, -50.0, 0.0, 50.0, 100.0]

	for si in range(street_positions.size() - 1):
		for sj in range(street_positions.size() - 1):
			var block_x: float = LIFT_X + (street_positions[si] + street_positions[si + 1]) * 0.5
			var block_z: float = LIFT_Z + (street_positions[sj] + street_positions[sj + 1]) * 0.5
			var block_w: float = absf(street_positions[si + 1] - street_positions[si]) - SIDE_STREET_W - 4.0

			if block_w < 10.0:
				continue

			# 2-4 buildings per block
			var count := rng.randi_range(2, 4)
			for _b in range(count):
				var bw: float = rng.randf_range(10, min(22, block_w * 0.45))
				var bd: float = rng.randf_range(8, min(18, block_w * 0.45))
				var bh: float = rng.randf_range(MIN_HEIGHT, MAX_HEIGHT)

				# Keep buildings below ceiling
				bh = minf(bh, absf(CEILING_Y - GROUND_Y) - 5.0)

				var ox: float = rng.randf_range(-block_w * 0.3, block_w * 0.3)
				var oz: float = rng.randf_range(-block_w * 0.3, block_w * 0.3)

				var bx: float = block_x + ox
				var bz: float = block_z + oz
				var by: float = GROUND_Y + bh * 0.5

				# Skip if too close to lift shaft
				if absf(bx - LIFT_X) < 25.0 and absf(bz - LIFT_Z) < 25.0:
					continue

				bldg_xforms.append(Transform3D(
					Basis.from_scale(Vector3(bw, bh, bd)),
					Vector3(bx, by, bz)))

				# Window strips on front face
				if bh > 12.0 and rng.randf() < 0.6:
					var win_h: float = bh * 0.6
					var face_dir: float = 1.0 if rng.randf() > 0.5 else -1.0
					var fx: float = bx + (bw * 0.5 + 0.1) * face_dir
					window_xforms.append(Transform3D(
						Basis.from_scale(Vector3(0.15, win_h, bd * 0.4)),
						Vector3(fx, GROUND_Y + bh * 0.4, bz)))

				# Neon sign on some buildings (warm orange)
				if rng.randf() < 0.35:
					var sign_face: float = 1.0 if rng.randf() > 0.5 else -1.0
					var sx: float = bx + (bw * 0.5 + 0.15) * sign_face
					var sy: float = GROUND_Y + rng.randf_range(3, min(bh - 2, 8))
					sign_xforms.append(Transform3D(
						Basis.from_scale(Vector3(0.2, 1.5, rng.randf_range(2, 4))),
						Vector3(sx, sy, bz)))

	# Warm-toned building material (muted orange-brown)
	_mm("LowerBuildings", bldg_xforms,
		_mat(Color(0.42, 0.35, 0.28), 0.75))

	# Window strips (warm amber glow)
	_mm("LowerWindows", window_xforms,
		_glow(Color(0.5, 0.4, 0.25), Color(0.8, 0.6, 0.3), 0.8))

	# Neon signs (orange glow)
	_mm("LowerSigns", sign_xforms,
		_glow(Color(0.9, 0.5, 0.15), Color(1.0, 0.6, 0.2), 2.0))


func _build_lighting() -> void:
	# Central plaza warm overhead lights
	var positions := [
		Vector3(LIFT_X - 12, GROUND_Y + 6, LIFT_Z - 12),
		Vector3(LIFT_X + 12, GROUND_Y + 6, LIFT_Z - 12),
		Vector3(LIFT_X - 12, GROUND_Y + 6, LIFT_Z + 12),
		Vector3(LIFT_X + 12, GROUND_Y + 6, LIFT_Z + 12),
		Vector3(LIFT_X, GROUND_Y + 8, LIFT_Z),
	]

	for i in positions.size():
		var light := OmniLight3D.new()
		light.name = "PlazaLight%d" % i
		light.position = positions[i]
		light.light_color = Color(1.0, 0.75, 0.45)
		light.light_energy = 3.0
		light.omni_range = 25.0
		add_child(light)

	# Street lights along main boulevards
	for offset in [-80.0, -40.0, 0.0, 40.0, 80.0]:
		# N-S boulevard lights
		var ln := OmniLight3D.new()
		ln.name = "StreetNS_%d" % int(offset)
		ln.position = Vector3(LIFT_X + 8, GROUND_Y + 5, LIFT_Z + offset)
		ln.light_color = Color(1.0, 0.7, 0.35)
		ln.light_energy = 2.0
		ln.omni_range = 18.0
		add_child(ln)

		# E-W boulevard lights
		var le := OmniLight3D.new()
		le.name = "StreetEW_%d" % int(offset)
		le.position = Vector3(LIFT_X + offset, GROUND_Y + 5, LIFT_Z + 8)
		le.light_color = Color(1.0, 0.7, 0.35)
		le.light_energy = 2.0
		le.omni_range = 18.0
		add_child(le)

	# Ambient orange fill (large weak light covering the area)
	var ambient := OmniLight3D.new()
	ambient.name = "AmbientOrange"
	ambient.position = Vector3(LIFT_X, GROUND_Y + 40, LIFT_Z)
	ambient.light_color = Color(1.0, 0.7, 0.4)
	ambient.light_energy = 1.0
	ambient.omni_range = 120.0
	add_child(ambient)


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


func _mat(color: Color, rough: float) -> StandardMaterial3D:
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
