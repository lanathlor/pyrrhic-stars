extends Node3D
## Procedural cityscape — dense building forest via MultiMesh.
## Metal ground plane at Y=-300 with thousands of buildings on all sides.
## S/W/E sides: taller buildings (surrounded feel).
## N side: shorter buildings (open vista).
## 4 draw calls total (buildings + warm windows + blue windows + trim).

const Helpers := preload("res://scenes/environments/prime_hub/cityscape_helpers.gd")

const GROUND_Y: float = -300.0
const EXTENT: float = 50000.0
const SPACING_NEAR: float = 20.0  # within 500m
const SPACING_MID: float = 40.0  # 500-1200m
const SPACING_FAR: float = 80.0  # 1200-3000m
const SPACING_ULTRA: float = 500.0  # 3000-50000m
const PLAZA_X: float = 135.0
const PLAZA_Z_MIN: float = -120.0
const PLAZA_Z_MAX: float = 145.0

# Lower street district exclusion (hand-crafted area inside UndergroundBase)
const DISTRICT_X_MIN: float = -100.0
const DISTRICT_X_MAX: float = 110.0
const DISTRICT_Z_MIN: float = -160.0
const DISTRICT_Z_MAX: float = 50.0

# Accumulator arrays used during generation (avoids passing 4+ arrays).
var _bldg: Array[Transform3D] = []
var _win_warm: Array[Transform3D] = []
var _win_blue: Array[Transform3D] = []
var _trims: Array[Transform3D] = []


func _ready() -> void:
	_generate()


func _generate() -> void:
	var rng := RandomNumberGenerator.new()
	rng.seed = 42

	_bldg = []
	_win_warm = []
	_win_blue = []
	_trims = []

	# [spacing, inner_radius, outer_radius, skip%, min_w, max_w]
	var bands: Array[Array] = [
		[SPACING_NEAR, 0.0, 500.0, 0.03, 16.0, 24.0],
		[SPACING_MID, 500.0, 1200.0, 0.05, 22.0, 38.0],
		[SPACING_FAR, 1200.0, 3000.0, 0.05, 35.0, 60.0],
		[SPACING_ULTRA, 3000.0, 10000.0, 0.1, 500.0, 700.0],
		[2000.0, 10000.0, EXTENT, 0.1, 2000.0, 3000.0],
	]

	_populate_bands(rng, bands)

	# Ground plane
	add_child(Helpers.build_ground(GROUND_Y, EXTENT))

	# Landmarks + monolith
	_place_landmarks(rng)

	# Horizon walls
	var wall_mat := Helpers.solid_mat(Color(0.07, 0.08, 0.1), 0.9)
	_add_horizon_walls(wall_mat)

	# City glow lights
	add_child(Helpers.build_city_glow(rng))

	# Tall buildings on the UndergroundBase roof
	_place_lower_tall()

	# Lower cityscape grid
	_populate_lower_grid(rng)

	# Build all MultiMeshes (5 draw calls)
	_finalize_multimeshes(rng)

	# Free accumulator references
	_bldg = []
	_win_warm = []
	_win_blue = []
	_trims = []


func _finalize_multimeshes(rng: RandomNumberGenerator) -> void:
	var mm_bldg := Helpers.build_mm_varied("Buildings", _bldg, rng)
	if mm_bldg:
		add_child(mm_bldg)
	var mm_warm := Helpers.build_mm(
		"WarmWindows",
		_win_warm,
		Helpers.glow_mat(Color(0.4, 0.35, 0.25), Color(0.6, 0.5, 0.3), 0.6)
	)
	if mm_warm:
		add_child(mm_warm)
	var mm_blue := Helpers.build_mm(
		"BlueWindows",
		_win_blue,
		Helpers.glow_mat(Color(0.3, 0.32, 0.38), Color(0.4, 0.45, 0.55), 0.4)
	)
	if mm_blue:
		add_child(mm_blue)
	var mm_trims := Helpers.build_mm(
		"Trims", _trims, Helpers.glow_mat(Color(0.3, 0.4, 0.55), Color(0.2, 0.35, 0.6), 0.5)
	)
	if mm_trims:
		add_child(mm_trims)


func _add_horizon_walls(wall_mat: StandardMaterial3D) -> void:
	var wall_dist := 8000.0
	var wall_w := 20000.0
	var wall_h := 3000.0
	var wall_d := 500.0
	var wall_y := GROUND_Y + wall_h * 0.5
	add_child(
		Helpers.build_wall(
			"WallN", Vector3(0, wall_y, -wall_dist + 10), Vector3(wall_w, wall_h, wall_d), wall_mat
		)
	)
	add_child(
		Helpers.build_wall(
			"WallS", Vector3(0, wall_y, wall_dist + 10), Vector3(wall_w, wall_h, wall_d), wall_mat
		)
	)
	add_child(
		Helpers.build_wall(
			"WallW", Vector3(-wall_dist, wall_y, 10), Vector3(wall_d, wall_h, wall_w), wall_mat
		)
	)
	add_child(
		Helpers.build_wall(
			"WallE", Vector3(wall_dist, wall_y, 10), Vector3(wall_d, wall_h, wall_w), wall_mat
		)
	)


func _populate_bands(rng: RandomNumberGenerator, bands: Array[Array]) -> void:
	for bi in bands.size():
		_populate_single_band(rng, bands[bi])


func _populate_single_band(rng: RandomNumberGenerator, band: Array) -> void:
	var spacing: float = band[0]
	var inner_r: float = band[1]
	var outer_r: float = band[2]
	var skip_chance: float = band[3]
	var min_w: float = band[4]
	var max_w: float = band[5]
	var half := int(outer_r / spacing)

	for gx in range(-half, half + 1):
		for gz in range(-half, half + 1):
			var cx: float = gx * spacing + rng.randf_range(-spacing * 0.3, spacing * 0.3)
			var cz: float = gz * spacing + rng.randf_range(-spacing * 0.3, spacing * 0.3) + 10.0

			if _is_excluded(cx, cz):
				continue

			var dist_from_center: float = maxf(absf(cx), absf(cz - 10.0))
			if dist_from_center < inner_r:
				continue
			if rng.randf() < skip_chance:
				continue

			var w: float = rng.randf_range(min_w, max_w)
			var d: float = rng.randf_range(min_w * 0.8, max_w * 0.9)
			var h: float = _compute_height(rng, cx, cz)
			var y: float = GROUND_Y + h * 0.5
			_bldg.append(Transform3D(Basis.from_scale(Vector3(w, h, d)), Vector3(cx, y, cz)))

			if dist_from_center < 1200.0 and h > 40.0 and rng.randf() < 0.3:
				_add_window_strip(rng, Vector3(cx, h, cz), Vector2(w, d))

			if h > 150.0 and rng.randf() < 0.4:
				_trims.append(
					Transform3D(
						Basis.from_scale(Vector3(w + 0.4, 0.3, d + 0.4)),
						Vector3(cx, GROUND_Y + h + 0.15, cz)
					)
				)


func _is_excluded(cx: float, cz: float) -> bool:
	if cx > -PLAZA_X and cx < PLAZA_X and cz > PLAZA_Z_MIN and cz < PLAZA_Z_MAX:
		return true
	if cx > DISTRICT_X_MIN and cx < DISTRICT_X_MAX and cz > DISTRICT_Z_MIN and cz < DISTRICT_Z_MAX:
		return true
	return false


func _compute_height(rng: RandomNumberGenerator, cx: float, cz: float) -> float:
	var h: float = Helpers.gaussian(rng, 160.0, 55.0)
	h = maxf(h, 15.0)

	# S/W/E near plaza: boost mean so they tower above
	var near_s: bool = cz > PLAZA_Z_MAX and cz < PLAZA_Z_MAX + 250.0
	var near_w: bool = cx < -PLAZA_X and cx > -PLAZA_X - 250.0
	var near_e: bool = cx > PLAZA_X and cx < PLAZA_X + 250.0
	if near_s or near_w or near_e:
		h = maxf(h, Helpers.gaussian(rng, 350.0, 40.0))

	# North side: shorter skyline rising toward horizon
	if cz < PLAZA_Z_MIN:
		var north_dist: float = absf(cz - PLAZA_Z_MIN)
		var north_mean: float = 150.0 + north_dist * 0.2
		var north_h: float = Helpers.gaussian(rng, north_mean, 45.0)
		north_h = maxf(north_h, 15.0)
		h = minf(h, north_h)
	return h


## pos: (cx, h, cz), dims: (w, d)
func _add_window_strip(rng: RandomNumberGenerator, pos: Vector3, dims: Vector2) -> void:
	var cx: float = pos.x
	var h: float = pos.y
	var cz: float = pos.z
	var w: float = dims.x
	var d: float = dims.y
	var to_plaza := Vector2(-cx, 10.0 - cz)
	var win_h: float = h * 0.5
	var wt: Transform3D
	if absf(to_plaza.x) > absf(to_plaza.y):
		var fx: float = cx + (w * 0.5 + 0.1) * signf(to_plaza.x)
		wt = Transform3D(
			Basis.from_scale(Vector3(0.2, win_h, d * 0.4)), Vector3(fx, GROUND_Y + h * 0.35, cz)
		)
	else:
		var fz: float = cz + (d * 0.5 + 0.1) * signf(to_plaza.y)
		wt = Transform3D(
			Basis.from_scale(Vector3(w * 0.4, win_h, 0.2)), Vector3(cx, GROUND_Y + h * 0.35, fz)
		)
	if rng.randf() < 0.7:
		_win_warm.append(wt)
	else:
		_win_blue.append(wt)


func _place_landmarks(rng: RandomNumberGenerator) -> void:
	_place_monolith(rng)

	var landmark_lights := Node3D.new()
	landmark_lights.name = "LandmarkLights"
	add_child(landmark_lights)

	for i in Helpers.LANDMARKS.size():
		var lm: Array = Helpers.LANDMARKS[i]
		var lx: float = lm[0]
		var lz: float = lm[1]
		var lw: float = lm[2]
		var ld: float = lm[3]
		var lh: float = lm[4]
		var ly: float = GROUND_Y + lh * 0.5
		_bldg.append(Transform3D(Basis.from_scale(Vector3(lw, lh, ld)), Vector3(lx, ly, lz)))
		_trims.append(
			Transform3D(
				Basis.from_scale(Vector3(lw + 0.5, 0.4, ld + 0.5)),
				Vector3(lx, GROUND_Y + lh + 0.2, lz)
			)
		)
		var win_face_z: float = lz + ld * 0.5 + 0.15
		_win_blue.append(
			Transform3D(
				Basis.from_scale(Vector3(lw * 0.8, lh * 0.6, 0.3)),
				Vector3(lx, GROUND_Y + lh * 0.4, win_face_z)
			)
		)
		_add_landmark_lights(landmark_lights, i, Vector3(lx, lh, lz))


func _place_monolith(rng: RandomNumberGenerator) -> void:
	var mono_x := -600.0
	var mono_z := -3000.0
	var mono_w := 400.0
	var mono_d := 300.0
	var mono_h := 3000.0
	var mono_y := GROUND_Y + mono_h * 0.5
	_bldg.append(
		Transform3D(
			Basis.from_scale(Vector3(mono_w, mono_h, mono_d)), Vector3(mono_x, mono_y, mono_z)
		)
	)
	_trims.append(
		Transform3D(
			Basis.from_scale(Vector3(mono_w + 2.0, 1.0, mono_d + 2.0)),
			Vector3(mono_x, GROUND_Y + mono_h + 0.5, mono_z)
		)
	)

	# Window strips on south face
	var mono_wins: Array[Transform3D] = []
	for strip_i in 5:
		var strip_x: float = mono_x + (strip_i - 2) * 80.0
		mono_wins.append(
			Transform3D(
				Basis.from_scale(Vector3(30.0, mono_h * 0.6, 0.5)),
				Vector3(strip_x, GROUND_Y + mono_h * 0.35, mono_z + mono_d * 0.5 + 0.2)
			)
		)
	var mm_mono := Helpers.build_mm(
		"MonolithWindows",
		mono_wins,
		Helpers.glow_mat(Color(0.4, 0.35, 0.25), Color(0.6, 0.5, 0.3), 0.6)
	)
	if mm_mono:
		add_child(mm_mono)

	# Monolith lights
	add_child(Helpers.build_monolith_lights(GROUND_Y, mono_x, mono_z, mono_h))
	# Consume rng to keep determinism with original
	rng.randf()


## pos: (x, height, z)
func _add_landmark_lights(container: Node3D, idx: int, pos: Vector3) -> void:
	var lx: float = pos.x
	var lh: float = pos.y
	var lz: float = pos.z
	var dist_to_plaza: float = Vector2(lx, lz - 10.0).length()

	var base_light := OmniLight3D.new()
	base_light.name = "LM_Base_%d" % idx
	base_light.light_color = Color(0.9, 0.75, 0.45)
	base_light.light_energy = 30.0
	base_light.omni_range = dist_to_plaza + 500.0
	base_light.light_volumetric_fog_energy = 25.0
	base_light.omni_attenuation = 1.2
	base_light.shadow_enabled = false
	base_light.position = Vector3(lx, GROUND_Y + lh * 0.5, lz)
	container.add_child(base_light)

	var top_light := OmniLight3D.new()
	top_light.name = "LM_Top_%d" % idx
	top_light.light_color = Color(0.4, 0.55, 0.9)
	top_light.light_energy = 40.0
	top_light.omni_range = dist_to_plaza + 500.0
	top_light.light_volumetric_fog_energy = 30.0
	top_light.omni_attenuation = 1.0
	top_light.shadow_enabled = false
	top_light.position = Vector3(lx, GROUND_Y + lh * 0.85, lz)
	container.add_child(top_light)


func _place_lower_tall() -> void:
	var lower_roof_y := -155.0
	for lt in Helpers.LOWER_TALL:
		var ltx: float = lt[0]
		var ltz: float = lt[1]
		var ltw: float = lt[2]
		var ltd: float = lt[3]
		var lth: float = lt[4]
		_bldg.append(
			Transform3D(
				Basis.from_scale(Vector3(ltw, lth, ltd)),
				Vector3(ltx, lower_roof_y + lth * 0.5, ltz)
			)
		)
		_win_warm.append(
			Transform3D(
				Basis.from_scale(Vector3(0.3, lth * 0.5, ltd * 0.35)),
				Vector3(ltx + ltw * 0.5 + 0.15, lower_roof_y + lth * 0.4, ltz)
			)
		)
		_win_blue.append(
			Transform3D(
				Basis.from_scale(Vector3(ltw * 0.35, lth * 0.5, 0.3)),
				Vector3(ltx, lower_roof_y + lth * 0.4, ltz + ltd * 0.5 + 0.15)
			)
		)


func _populate_lower_grid(rng: RandomNumberGenerator) -> void:
	var lower_roof_y := -155.0
	for lgx in range(-30, 31):
		for lgz in range(-30, 31):
			var lcx: float = lgx * 22.0 + rng.randf_range(-6, 6)
			var lcz: float = lgz * 22.0 + rng.randf_range(-6, 6)
			if (
				lcx > DISTRICT_X_MIN
				and lcx < DISTRICT_X_MAX
				and lcz > DISTRICT_Z_MIN
				and lcz < DISTRICT_Z_MAX
			):
				continue
			if lcx > -PLAZA_X and lcx < PLAZA_X and lcz > 45.0 and lcz < PLAZA_Z_MAX + 10.0:
				continue
			var ldist := maxf(absf(lcx), absf(lcz))
			if ldist > 650.0:
				continue
			if rng.randf() < 0.25:
				continue
			var lw: float = rng.randf_range(12, 32)
			var ld: float = rng.randf_range(10, 28)
			var lh: float = rng.randf_range(40, 160)
			_bldg.append(
				Transform3D(
					Basis.from_scale(Vector3(lw, lh, ld)),
					Vector3(lcx, lower_roof_y + lh * 0.5, lcz)
				)
			)
			if lh > 15.0:
				_add_lower_windows(rng, Vector3(lcx, lh, lcz), Vector2(lw, ld), lower_roof_y)


func _add_lower_windows(
	rng: RandomNumberGenerator, pos: Vector3, dims: Vector2, roof_y: float
) -> void:
	var lcx: float = pos.x
	var lh: float = pos.y
	var lcz: float = pos.z
	var lw: float = dims.x
	var ld: float = dims.y
	var win_y: float = roof_y + lh * 0.35
	var win_h: float = lh * 0.5
	for side in [-1.0, 1.0]:
		if rng.randf() < 0.6:
			var fx: float = lcx + (lw * 0.5 + 0.1) * side
			var wlist := _win_warm if rng.randf() < 0.7 else _win_blue
			wlist.append(
				Transform3D(
					Basis.from_scale(Vector3(0.2, win_h, ld * 0.35)), Vector3(fx, win_y, lcz)
				)
			)
	for side in [-1.0, 1.0]:
		if rng.randf() < 0.6:
			var fz: float = lcz + (ld * 0.5 + 0.1) * side
			var wlist := _win_warm if rng.randf() < 0.7 else _win_blue
			wlist.append(
				Transform3D(
					Basis.from_scale(Vector3(lw * 0.35, win_h, 0.2)), Vector3(lcx, win_y, fz)
				)
			)
	if rng.randf() < 0.3:
		_trims.append(
			Transform3D(
				Basis.from_scale(Vector3(lw + 0.3, 0.2, ld + 0.3)),
				Vector3(lcx, roof_y + lh + 0.1, lcz)
			)
		)
