extends Node3D
## Procedural cityscape — dense building forest via MultiMesh.
## Metal ground plane at Y=-300 with thousands of buildings on all sides.
## S/W/E sides: taller buildings (surrounded feel).
## N side: shorter buildings (open vista).
## 4 draw calls total (buildings + warm windows + blue windows + trim).

const GROUND_Y: float = -300.0
const EXTENT: float = 50000.0
const SPACING_NEAR: float = 20.0    # within 500m
const SPACING_MID: float = 40.0     # 500-1200m
const SPACING_FAR: float = 80.0     # 1200-3000m
const SPACING_ULTRA: float = 500.0  # 3000-50000m
const PLAZA_X: float = 135.0
const PLAZA_Z_MIN: float = -120.0
const PLAZA_Z_MAX: float = 145.0


func _ready() -> void:
	_generate()


func _generate() -> void:
	var rng := RandomNumberGenerator.new()
	rng.seed = 42

	var bldg: Array[Transform3D] = []
	var win_warm: Array[Transform3D] = []
	var win_blue: Array[Transform3D] = []
	var trims: Array[Transform3D] = []

	# [spacing, inner_radius, outer_radius, skip%, min_w, max_w]
	# Far bands: width > spacing so buildings overlap = no gaps = solid wall
	var bands: Array[Array] = [
		[SPACING_NEAR, 0.0, 500.0, 0.03, 16.0, 24.0],
		[SPACING_MID, 500.0, 1200.0, 0.05, 22.0, 38.0],
		[SPACING_FAR, 1200.0, 3000.0, 0.05, 35.0, 60.0],
		[SPACING_ULTRA, 3000.0, 10000.0, 0.1, 500.0, 700.0],
		[2000.0, 10000.0, EXTENT, 0.1, 2000.0, 3000.0],
	]

	for bi in bands.size():
		var band: Array = bands[bi]
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

				# Skip plaza footprint
				if cx > -PLAZA_X and cx < PLAZA_X and cz > PLAZA_Z_MIN and cz < PLAZA_Z_MAX:
					continue

				# Skip if inside inner radius (already populated by denser band)
				var dist_from_center: float = maxf(absf(cx), absf(cz - 10.0))
				if dist_from_center < inner_r:
					continue

				if rng.randf() < skip_chance:
					continue

				var w: float = rng.randf_range(min_w, max_w)
				var d: float = rng.randf_range(min_w * 0.8, max_w * 0.9)

				# Height — gaussian, mean rises with distance from plaza
				# Close: mean=160m. Far: grows so view eventually boxes in.
				var base_mean: float = 160.0
				var base_sd: float = 55.0
				var h: float = _gaussian(rng, base_mean, base_sd)
				h = maxf(h, 15.0)

				# S/W/E near plaza: boost mean so they tower above
				var near_s: bool = cz > PLAZA_Z_MAX and cz < PLAZA_Z_MAX + 250.0
				var near_w: bool = cx < -PLAZA_X and cx > -PLAZA_X - 250.0
				var near_e: bool = cx > PLAZA_X and cx < PLAZA_X + 250.0
				if near_s or near_w or near_e:
					h = maxf(h, _gaussian(rng, 350.0, 40.0))

				# North side: mean grows with distance so skyline rises toward horizon
				# Near plaza: mean=150. At 1km: 350. At 3km: 750. At 10km: 2150.
				if cz < PLAZA_Z_MIN:
					var north_dist: float = absf(cz - PLAZA_Z_MIN)
					var north_mean: float = 150.0 + north_dist * 0.2
					var north_h: float = _gaussian(rng, north_mean, 45.0)
					north_h = maxf(north_h, 15.0)
					h = minf(h, north_h)

				var y: float = GROUND_Y + h * 0.5
				bldg.append(Transform3D(
					Basis.from_scale(Vector3(w, h, d)),
					Vector3(cx, y, cz)))

				# Window strip — narrow band on plaza-facing side, like a column of lit windows
				if dist_from_center < 1200.0 and h > 40.0 and rng.randf() < 0.3:
					var to_plaza := Vector2(-cx, 10.0 - cz)
					var win_h: float = h * 0.5
					var wt: Transform3D
					if absf(to_plaza.x) > absf(to_plaza.y):
						var fx: float = cx + (w * 0.5 + 0.1) * signf(to_plaza.x)
						wt = Transform3D(
							Basis.from_scale(Vector3(0.2, win_h, d * 0.4)),
							Vector3(fx, GROUND_Y + h * 0.35, cz))
					else:
						var fz: float = cz + (d * 0.5 + 0.1) * signf(to_plaza.y)
						wt = Transform3D(
							Basis.from_scale(Vector3(w * 0.4, win_h, 0.2)),
							Vector3(cx, GROUND_Y + h * 0.35, fz))
					if rng.randf() < 0.7:
						win_warm.append(wt)
					else:
						win_blue.append(wt)

				# Blue trim cap on tallest buildings
				if h > 150.0 and rng.randf() < 0.4:
					trims.append(Transform3D(
						Basis.from_scale(Vector3(w + 0.4, 0.3, d + 0.4)),
						Vector3(cx, GROUND_Y + h + 0.15, cz)))

	# Ground plane — dark metallic city floor
	var ground_mi := MeshInstance3D.new()
	var ground_mesh := BoxMesh.new()
	ground_mesh.size = Vector3(EXTENT * 3.0, 2.0, EXTENT * 3.0)
	ground_mi.mesh = ground_mesh
	ground_mi.position = Vector3(0.0, GROUND_Y - 1.0, 10.0)
	var ground_mat := StandardMaterial3D.new()
	ground_mat.albedo_color = Color(0.06, 0.065, 0.08)
	ground_mat.roughness = 0.5
	ground_mat.metallic = 0.2
	ground_mi.material_override = ground_mat
	ground_mi.name = "CityGround"
	add_child(ground_mi)

	# 20 landmark mega-towers — hardcoded, break all rules
	# Scattered in the north view at various distances, tall enough to pop out
	# [X, Z, width, depth, height]
	# [X, Z, width, depth, height] — big enough to read at distance
	var landmarks := [
		[-400.0, -600.0, 50.0, 45.0, 600.0],
		[300.0, -800.0, 60.0, 50.0, 700.0],
		[-150.0, -1100.0, 55.0, 50.0, 750.0],
		[600.0, -500.0, 45.0, 40.0, 550.0],
		[-700.0, -900.0, 65.0, 55.0, 680.0],
		[100.0, -1400.0, 55.0, 50.0, 800.0],
		[-500.0, -1600.0, 60.0, 55.0, 720.0],
		[450.0, -1200.0, 50.0, 45.0, 650.0],
		[-250.0, -2000.0, 70.0, 65.0, 850.0],
		[700.0, -1800.0, 65.0, 55.0, 780.0],
		[-900.0, -700.0, 55.0, 50.0, 620.0],
		[200.0, -2400.0, 75.0, 70.0, 900.0],
		[-600.0, -2200.0, 60.0, 55.0, 810.0],
		[900.0, -1000.0, 50.0, 45.0, 640.0],
		[-100.0, -400.0, 40.0, 35.0, 500.0],
		[500.0, -2600.0, 70.0, 65.0, 860.0],
		[-800.0, -1400.0, 55.0, 50.0, 700.0],
		[350.0, -350.0, 35.0, 30.0, 480.0],
		[-350.0, -1800.0, 60.0, 55.0, 740.0],
		[800.0, -2200.0, 65.0, 60.0, 820.0],
	]
	# THE MONOLITH — colossal mega-structure, 3km north, off to the left
	# 400m wide, 300m deep, 3000m tall. Narrow tower dominating the skyline.
	var mono_x := -600.0
	var mono_z := -3000.0
	var mono_w := 400.0
	var mono_d := 300.0
	var mono_h := 3000.0
	var mono_y := GROUND_Y + mono_h * 0.5
	bldg.append(Transform3D(Basis.from_scale(Vector3(mono_w, mono_h, mono_d)), Vector3(mono_x, mono_y, mono_z)))

	# Monolith blue trim at the top
	trims.append(Transform3D(
		Basis.from_scale(Vector3(mono_w + 2.0, 1.0, mono_d + 2.0)),
		Vector3(mono_x, GROUND_Y + mono_h + 0.5, mono_z)))

	# Monolith window strips — tall vertical bands on the south face (facing plaza)
	for strip_i in 5:
		var strip_x: float = mono_x + (strip_i - 2) * 80.0
		win_warm.append(Transform3D(
			Basis.from_scale(Vector3(30.0, mono_h * 0.6, 0.5)),
			Vector3(strip_x, GROUND_Y + mono_h * 0.35, mono_z + mono_d * 0.5 + 0.2)))

	# Monolith lights — massive glow visible from the plaza
	var mono_lights := Node3D.new()
	mono_lights.name = "MonolithLights"
	add_child(mono_lights)

	# Lights stacked up the monolith — 6 warm, 2 blue
	var mono_light_defs := [
		[0.15, Color(0.95, 0.8, 0.45), 30.0, 2000.0],
		[0.3, Color(0.95, 0.8, 0.45), 40.0, 2500.0],
		[0.45, Color(0.95, 0.8, 0.45), 40.0, 2500.0],
		[0.55, Color(0.95, 0.8, 0.45), 30.0, 2000.0],
		[0.65, Color(0.4, 0.55, 0.95), 35.0, 2000.0],
		[0.75, Color(0.95, 0.8, 0.45), 40.0, 2500.0],
		[0.85, Color(0.4, 0.55, 0.95), 50.0, 3000.0],
		[0.95, Color(0.4, 0.55, 0.95), 60.0, 3500.0],
	]
	for li in mono_light_defs.size():
		var def: Array = mono_light_defs[li]
		var ml := OmniLight3D.new()
		ml.name = "Mono_%d" % li
		ml.light_color = def[1]
		ml.light_energy = def[2]
		ml.omni_range = def[3]
		ml.light_volumetric_fog_energy = def[2] * 0.8
		ml.omni_attenuation = 0.8
		ml.shadow_enabled = false
		ml.position = Vector3(mono_x, GROUND_Y + mono_h * def[0], mono_z)
		mono_lights.add_child(ml)

	var landmark_lights := Node3D.new()
	landmark_lights.name = "LandmarkLights"
	add_child(landmark_lights)

	for i in landmarks.size():
		var lm: Array = landmarks[i]
		var lx: float = lm[0]
		var lz: float = lm[1]
		var lw: float = lm[2]
		var ld: float = lm[3]
		var lh: float = lm[4]
		var ly: float = GROUND_Y + lh * 0.5
		bldg.append(Transform3D(Basis.from_scale(Vector3(lw, lh, ld)), Vector3(lx, ly, lz)))
		# All landmarks get blue trim + window glow
		trims.append(Transform3D(
			Basis.from_scale(Vector3(lw + 0.5, 0.4, ld + 0.5)),
			Vector3(lx, GROUND_Y + lh + 0.2, lz)))
		var win_face_z: float = lz + ld * 0.5 + 0.15
		win_blue.append(Transform3D(
			Basis.from_scale(Vector3(lw * 0.8, lh * 0.6, 0.3)),
			Vector3(lx, GROUND_Y + lh * 0.4, win_face_z)))

		# Volumetric fog light on each landmark — visible glow through fog
		# Base light (warm, at mid-height)
		var base_light := OmniLight3D.new()
		base_light.name = "LM_Base_%d" % i
		base_light.light_color = Color(0.9, 0.75, 0.45)
		var dist_to_plaza: float = Vector2(lx, lz - 10.0).length()
		base_light.light_energy = 30.0
		base_light.omni_range = dist_to_plaza + 500.0
		base_light.light_volumetric_fog_energy = 25.0
		base_light.omni_attenuation = 1.2
		base_light.shadow_enabled = false
		base_light.position = Vector3(lx, GROUND_Y + lh * 0.5, lz)
		landmark_lights.add_child(base_light)

		# Top light (blue accent, at the peak)
		var top_light := OmniLight3D.new()
		top_light.name = "LM_Top_%d" % i
		top_light.light_color = Color(0.4, 0.55, 0.9)
		top_light.light_energy = 40.0
		top_light.omni_range = dist_to_plaza + 500.0
		top_light.light_volumetric_fog_energy = 30.0
		top_light.omni_attenuation = 1.0
		top_light.shadow_enabled = false
		top_light.position = Vector3(lx, GROUND_Y + lh * 0.85, lz)
		landmark_lights.add_child(top_light)

	# Horizon wall — 4 massive slabs forming a skyline ring at ~8km
	# Each slab is 20km wide, 3km tall, 500m deep. Looks like a distant
	# mega-city wall. Guaranteed to block the sky in every direction.
	var wall_mat := _solid(Color(0.07, 0.08, 0.1), 0.9)
	var wall_dist := 8000.0
	var wall_w := 20000.0
	var wall_h := 3000.0
	var wall_d := 500.0
	var wall_y := GROUND_Y + wall_h * 0.5
	# North
	_add_wall("WallN", Vector3(0, wall_y, -wall_dist + 10), Vector3(wall_w, wall_h, wall_d), wall_mat)
	# South
	_add_wall("WallS", Vector3(0, wall_y, wall_dist + 10), Vector3(wall_w, wall_h, wall_d), wall_mat)
	# West
	_add_wall("WallW", Vector3(-wall_dist, wall_y, 10), Vector3(wall_d, wall_h, wall_w), wall_mat)
	# East
	_add_wall("WallE", Vector3(wall_dist, wall_y, 10), Vector3(wall_d, wall_h, wall_w), wall_mat)

	# City glow lights — OmniLight3D scattered through the city
	# Visible through volumetric fog as distant glowing patches
	_build_city_glow(rng)

	# Hardcoded tall buildings north of shaft, reaching up to plaza bottom (Y=-4.3)
	# Ground at Y=-180, so height ~176m to reach Y=-4
	var lower_tall := [
		# [X, Z, width, depth, height]
		[-60.0, -90.0, 30.0, 25.0, 176.0],
		[-100.0, -60.0, 25.0, 30.0, 174.0],
		[60.0, -100.0, 28.0, 22.0, 175.0],
		[90.0, -70.0, 22.0, 28.0, 173.0],
	]
	for lt in lower_tall:
		var ltx: float = lt[0]
		var ltz: float = lt[1]
		var ltw: float = lt[2]
		var ltd: float = lt[3]
		var lth: float = lt[4]
		bldg.append(Transform3D(
			Basis.from_scale(Vector3(ltw, lth, ltd)),
			Vector3(ltx, -180.0 + lth * 0.5, ltz)))
		# Window strips on these
		win_warm.append(Transform3D(
			Basis.from_scale(Vector3(0.3, lth * 0.5, ltd * 0.35)),
			Vector3(ltx + ltw * 0.5 + 0.15, -180.0 + lth * 0.4, ltz)))
		win_blue.append(Transform3D(
			Basis.from_scale(Vector3(ltw * 0.35, lth * 0.5, 0.3)),
			Vector3(ltx, -180.0 + lth * 0.4, ltz + ltd * 0.5 + 0.15)))

	# Lower cityscape at Y=-180 — buildings barely poking above Y=-150
	# Clears the shaft area (X -15..25, Z -75..-35) and south building (Z > 45)
	var lower_ground_y := -180.0
	for lgx in range(-30, 31):
		for lgz in range(-30, 31):
			var lcx: float = lgx * 22.0 + rng.randf_range(-6, 6)
			var lcz: float = lgz * 22.0 + rng.randf_range(-6, 6)
			# Skip shaft exclusion
			if lcx > -15.0 and lcx < 25.0 and lcz > -75.0 and lcz < -35.0:
				continue
			# Skip south building + plaza above
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
			bldg.append(Transform3D(
				Basis.from_scale(Vector3(lw, lh, ld)),
				Vector3(lcx, lower_ground_y + lh * 0.5, lcz)))
			# Windows on multiple faces for nearby buildings
			if lh > 15.0:
				var win_y: float = lower_ground_y + lh * 0.35
				var win_h: float = lh * 0.5
				# X-facing windows (front and back)
				for side in [-1.0, 1.0]:
					if rng.randf() < 0.6:
						var fx: float = lcx + (lw * 0.5 + 0.1) * side
						var wlist: Array = win_warm if rng.randf() < 0.7 else win_blue
						wlist.append(Transform3D(
							Basis.from_scale(Vector3(0.2, win_h, ld * 0.35)),
							Vector3(fx, win_y, lcz)))
				# Z-facing windows (left and right)
				for side in [-1.0, 1.0]:
					if rng.randf() < 0.6:
						var fz: float = lcz + (ld * 0.5 + 0.1) * side
						var wlist: Array = win_warm if rng.randf() < 0.7 else win_blue
						wlist.append(Transform3D(
							Basis.from_scale(Vector3(lw * 0.35, win_h, 0.2)),
							Vector3(lcx, win_y, fz)))
				# Blue trim cap
				if rng.randf() < 0.3:
					trims.append(Transform3D(
						Basis.from_scale(Vector3(lw + 0.3, 0.2, ld + 0.3)),
						Vector3(lcx, lower_ground_y + lh + 0.1, lcz)))

	# 5 multimeshes = 5 draw calls
	# Buildings: varied shades of dark grey
	_mm_varied("Buildings", bldg, rng)
	# Windows: faint warm yellow glow, not bright blue rectangles
	_mm("WarmWindows", win_warm, _glow(Color(0.4, 0.35, 0.25), Color(0.6, 0.5, 0.3), 0.6))
	_mm("BlueWindows", win_blue, _glow(Color(0.3, 0.32, 0.38), Color(0.4, 0.45, 0.55), 0.4))
	# Trims: subtle blue accent, not neon
	_mm("Trims", trims, _glow(Color(0.3, 0.4, 0.55), Color(0.2, 0.35, 0.6), 0.5))


func _mm_varied(nm: String, xforms: Array[Transform3D], rng: RandomNumberGenerator) -> void:
	if xforms.is_empty():
		return
	var mesh := BoxMesh.new()
	mesh.size = Vector3.ONE
	var mm := MultiMesh.new()
	mm.transform_format = MultiMesh.TRANSFORM_3D
	mm.use_custom_data = true
	mm.instance_count = xforms.size()
	mm.mesh = mesh
	for i in xforms.size():
		mm.set_instance_transform(i, xforms[i])
		# Pack a brightness variation into custom data (0.7 to 1.3 range)
		var shade: float = rng.randf_range(0.7, 1.3)
		# Slight hue shift: some buildings slightly warmer, some cooler
		var warm: float = rng.randf_range(-0.03, 0.03)
		mm.set_instance_custom_data(i, Color(shade, warm, 0.0, 0.0))
	var mmi := MultiMeshInstance3D.new()
	mmi.multimesh = mm
	mmi.name = nm
	# Shader material that reads INSTANCE_CUSTOM for color variation
	var shader := Shader.new()
	shader.code = "shader_type spatial;
render_mode cull_back;
uniform vec3 base_color = vec3(0.35, 0.36, 0.38);
uniform float roughness_val : hint_range(0.0, 1.0) = 0.75;
varying float v_shade;
varying float v_warm;
void vertex() {
	v_shade = INSTANCE_CUSTOM.r;
	v_warm = INSTANCE_CUSTOM.g;
}
void fragment() {
	ALBEDO = base_color * v_shade + vec3(v_warm, v_warm * 0.5, -v_warm);
	ROUGHNESS = roughness_val;
}
"
	var mat := ShaderMaterial.new()
	mat.shader = shader
	mat.set_shader_parameter("base_color", Vector3(0.35, 0.36, 0.38))
	mat.set_shader_parameter("roughness_val", 0.75)
	mmi.material_override = mat
	add_child(mmi)


func _mm(nm: String, xforms: Array[Transform3D], mat: StandardMaterial3D) -> void:
	if xforms.is_empty():
		return
	var mesh := BoxMesh.new()
	mesh.size = Vector3.ONE
	var mm := MultiMesh.new()
	mm.transform_format = MultiMesh.TRANSFORM_3D
	mm.instance_count = xforms.size()
	mm.mesh = mesh
	for i in xforms.size():
		mm.set_instance_transform(i, xforms[i])
	var mmi := MultiMeshInstance3D.new()
	mmi.multimesh = mm
	mmi.name = nm
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
	m.roughness = 0.2
	return m


func _build_city_glow(rng: RandomNumberGenerator) -> void:
	# Scatter omni lights at building-top height throughout the city.
	# These interact with volumetric fog, creating visible glow patches.
	# ~60 lights total — mix of warm yellow (city lights) and cool blue (Empire accent).
	var glow_container := Node3D.new()
	glow_container.name = "CityGlow"
	add_child(glow_container)

	# Rings at increasing distances: [distance, count, range, energy, y_offset]
	# [distance, count, energy, y_offset]
	# Range is auto-set to reach the plaza
	var rings := [
		[400.0, 14, 10.0, -100.0],
		[800.0, 16, 15.0, -80.0],
		[1500.0, 14, 25.0, -50.0],
		[3000.0, 12, 35.0, -20.0],
	]

	var idx := 0
	for ring in rings:
		var dist: float = ring[0]
		var count: int = ring[1]
		var energy: float = ring[2]
		var y_off: float = ring[3]

		for i in count:
			var angle: float = (TAU / count) * i + rng.randf_range(-0.3, 0.3)
			var px: float = cos(angle) * dist + rng.randf_range(-dist * 0.15, dist * 0.15)
			var pz: float = sin(angle) * dist + 10.0 + rng.randf_range(-dist * 0.15, dist * 0.15)

			var light := OmniLight3D.new()
			light.name = "Glow_%d" % idx
			idx += 1

			# 70% warm yellow, 30% cool blue
			if rng.randf() < 0.7:
				light.light_color = Color(0.95, 0.8, 0.5)
			else:
				light.light_color = Color(0.5, 0.65, 0.9)

			# Range must reach back to the plaza so fog between here and player is lit
			var light_dist: float = Vector2(px, pz - 10.0).length()
			light.light_energy = energy
			light.omni_range = light_dist + 300.0
			light.light_volumetric_fog_energy = energy
			light.omni_attenuation = 1.5
			light.shadow_enabled = false
			light.position = Vector3(px, y_off, pz)

			glow_container.add_child(light)


func _add_wall(nm: String, pos: Vector3, size: Vector3, mat: StandardMaterial3D) -> void:
	var mi := MeshInstance3D.new()
	var mesh := BoxMesh.new()
	mesh.size = size
	mi.mesh = mesh
	mi.position = pos
	mi.material_override = mat
	mi.name = nm
	add_child(mi)


## Box-Muller gaussian. Returns a sample with given mean and std_dev.
func _gaussian(rng: RandomNumberGenerator, mean: float, std_dev: float) -> float:
	var u1: float = maxf(rng.randf(), 0.0001)
	var u2: float = rng.randf()
	var z: float = sqrt(-2.0 * log(u1)) * cos(TAU * u2)
	return mean + std_dev * z
