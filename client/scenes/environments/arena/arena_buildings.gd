extends Node3D
## Procedural CSG buildings around the arena dungeon — futuristic megacity scale.
## Layout: lobby (enclosed) → open-air street → covered building break → boss plaza
## Attach to the Arena root after loading the scene.

# Materials
var mat_facade: StandardMaterial3D
var mat_facade_dark: StandardMaterial3D
var mat_ledge: StandardMaterial3D
var mat_ceiling: StandardMaterial3D
var mat_window_glow: StandardMaterial3D


func _ready() -> void:
	_init_materials()
	_build_street_east()
	_build_street_west()
	_build_boss_room_east()
	_build_boss_room_west()
	_build_boss_room_north()
	_build_transition_building()
	_build_lobby_building()
	_build_burning_buildings()


func _init_materials() -> void:
	mat_facade = StandardMaterial3D.new()
	mat_facade.albedo_color = Color(0.22, 0.22, 0.26)
	mat_facade.roughness = 0.8

	mat_facade_dark = StandardMaterial3D.new()
	mat_facade_dark.albedo_color = Color(0.08, 0.08, 0.10)
	mat_facade_dark.roughness = 0.9

	mat_ledge = StandardMaterial3D.new()
	mat_ledge.albedo_color = Color(0.28, 0.28, 0.32)
	mat_ledge.roughness = 0.7

	mat_ceiling = StandardMaterial3D.new()
	mat_ceiling.albedo_color = Color(0.14, 0.14, 0.17)
	mat_ceiling.roughness = 0.85

	mat_window_glow = StandardMaterial3D.new()
	mat_window_glow.albedo_color = Color(0.05, 0.06, 0.12)
	mat_window_glow.emission_enabled = true
	mat_window_glow.emission = Color(0.08, 0.10, 0.25)
	mat_window_glow.emission_energy_multiplier = 0.4


# =============================================================================
# Street — East Buildings (X > 8, Z=15 to Z=40) — open air, towering facades
# =============================================================================

func _build_street_east() -> void:
	var parent := Node3D.new()
	parent.name = "StreetEast"
	add_child(parent)

	# Tower SE1: Z=15 to Z=24, height 45 — sleek main tower
	var se1 := _csg_box("SE1", Vector3(10.0, 45.0, 9.0), Vector3(13.0, 22.5, 19.5), mat_facade)
	parent.add_child(se1)
	_add_window_strip(parent, "SE1", Vector3(8.0, 0.0, 15.0), 9.0, 45.0, Vector3.LEFT, 4)
	_add_ledges(parent, "SE1", Vector3(13.0, 0.0, 19.5), 9.0, 45.0, Vector3.LEFT)

	# Tower SE2: Z=25 to Z=32, height 55 — tallest on the east, setback 2 units
	var se2 := _csg_box("SE2", Vector3(12.0, 55.0, 7.0), Vector3(14.0, 27.5, 28.5), mat_facade)
	parent.add_child(se2)
	_add_window_strip(parent, "SE2", Vector3(8.0, 0.0, 25.0), 7.0, 55.0, Vector3.LEFT, 3)
	_add_ledges(parent, "SE2", Vector3(14.0, 0.0, 28.5), 7.0, 55.0, Vector3.LEFT)

	# Tower SE3: Z=33 to Z=40, height 35 — shorter, creates skyline variation
	var se3 := _csg_box("SE3", Vector3(8.0, 35.0, 7.0), Vector3(12.0, 17.5, 36.5), mat_facade)
	parent.add_child(se3)
	_add_window_strip(parent, "SE3", Vector3(8.0, 0.0, 33.0), 7.0, 35.0, Vector3.LEFT, 3)
	_add_ledges(parent, "SE3", Vector3(12.0, 0.0, 36.5), 7.0, 35.0, Vector3.LEFT)

	# Second-row tower behind SE1 (deeper, even taller — visible above)
	var se1_back := _csg_box("SE1B", Vector3(8.0, 60.0, 9.0), Vector3(22.0, 30.0, 19.5), mat_facade)
	parent.add_child(se1_back)


# =============================================================================
# Street — West Buildings (X < -8, Z=15 to Z=40) — asymmetric heights
# =============================================================================

func _build_street_west() -> void:
	var parent := Node3D.new()
	parent.name = "StreetWest"
	add_child(parent)

	# Tower SW1: Z=15 to Z=22, height 50
	var sw1 := _csg_box("SW1", Vector3(10.0, 50.0, 7.0), Vector3(-13.0, 25.0, 18.5), mat_facade)
	parent.add_child(sw1)
	_add_window_strip(parent, "SW1", Vector3(-8.0, 0.0, 15.0), 7.0, 50.0, Vector3.RIGHT, 3)
	_add_ledges(parent, "SW1", Vector3(-13.0, 0.0, 18.5), 7.0, 50.0, Vector3.RIGHT)

	# Tower SW2: Z=23 to Z=33, height 40
	var sw2 := _csg_box("SW2", Vector3(12.0, 40.0, 10.0), Vector3(-14.0, 20.0, 28.0), mat_facade)
	parent.add_child(sw2)
	_add_window_strip(parent, "SW2", Vector3(-8.0, 0.0, 23.0), 10.0, 40.0, Vector3.RIGHT, 4)
	_add_ledges(parent, "SW2", Vector3(-14.0, 0.0, 28.0), 10.0, 40.0, Vector3.RIGHT)

	# Tower SW3: Z=34 to Z=40, height 60 — tallest on the west, dramatic
	var sw3 := _csg_box("SW3", Vector3(10.0, 60.0, 6.0), Vector3(-13.0, 30.0, 37.0), mat_facade)
	parent.add_child(sw3)
	_add_window_strip(parent, "SW3", Vector3(-8.0, 0.0, 34.0), 6.0, 60.0, Vector3.RIGHT, 3)
	_add_ledges(parent, "SW3", Vector3(-13.0, 0.0, 37.0), 6.0, 60.0, Vector3.RIGHT)

	# Second-row tower behind SW2
	var sw2_back := _csg_box("SW2B", Vector3(8.0, 55.0, 10.0), Vector3(-24.0, 27.5, 28.0), mat_facade)
	parent.add_child(sw2_back)


# =============================================================================
# Boss Room — East Flanking Buildings (X > 20)
# =============================================================================

func _build_boss_room_east() -> void:
	var parent := Node3D.new()
	parent.name = "BossEast"
	add_child(parent)

	# Tower BE1: Z=-14 to Z=-6, height 40
	var be1 := _csg_box("BE1", Vector3(8.0, 40.0, 8.0), Vector3(24.25, 20.0, -10.0), mat_facade)
	parent.add_child(be1)
	_add_window_strip(parent, "BE1", Vector3(20.25, 0.0, -14.0), 8.0, 40.0, Vector3.LEFT, 3)
	_add_ledges(parent, "BE1", Vector3(24.25, 0.0, -10.0), 8.0, 40.0, Vector3.LEFT)

	# Tower BE2: Z=-5 to Z=5, height 50 — tallest boss-side east
	var be2 := _csg_box("BE2", Vector3(10.0, 50.0, 10.0), Vector3(25.25, 25.0, 0.0), mat_facade)
	parent.add_child(be2)
	_add_window_strip(parent, "BE2", Vector3(20.25, 0.0, -5.0), 10.0, 50.0, Vector3.LEFT, 4)
	_add_ledges(parent, "BE2", Vector3(25.25, 0.0, 0.0), 10.0, 50.0, Vector3.LEFT)

	# Tower BE3: Z=6 to Z=12, height 35
	var be3 := _csg_box("BE3", Vector3(7.0, 35.0, 6.0), Vector3(23.75, 17.5, 9.0), mat_facade)
	parent.add_child(be3)
	_add_window_strip(parent, "BE3", Vector3(20.25, 0.0, 6.0), 6.0, 35.0, Vector3.LEFT, 3)
	_add_ledges(parent, "BE3", Vector3(23.75, 0.0, 9.0), 6.0, 35.0, Vector3.LEFT)


# =============================================================================
# Boss Room — West Flanking Buildings (X < -20)
# =============================================================================

func _build_boss_room_west() -> void:
	var parent := Node3D.new()
	parent.name = "BossWest"
	add_child(parent)

	# Tower BW1: Z=-14 to Z=-7, height 45
	var bw1 := _csg_box("BW1", Vector3(9.0, 45.0, 7.0), Vector3(-24.75, 22.5, -10.5), mat_facade)
	parent.add_child(bw1)
	_add_window_strip(parent, "BW1", Vector3(-20.25, 0.0, -14.0), 7.0, 45.0, Vector3.RIGHT, 3)
	_add_ledges(parent, "BW1", Vector3(-24.75, 0.0, -10.5), 7.0, 45.0, Vector3.RIGHT)

	# Tower BW2: Z=-6 to Z=4, height 35
	var bw2 := _csg_box("BW2", Vector3(8.0, 35.0, 10.0), Vector3(-24.25, 17.5, -1.0), mat_facade)
	parent.add_child(bw2)
	_add_window_strip(parent, "BW2", Vector3(-20.25, 0.0, -6.0), 10.0, 35.0, Vector3.RIGHT, 4)
	_add_ledges(parent, "BW2", Vector3(-24.25, 0.0, -1.0), 10.0, 35.0, Vector3.RIGHT)

	# Tower BW3: Z=5 to Z=12, height 55 — tallest boss-side west, dramatic asymmetry
	var bw3 := _csg_box("BW3", Vector3(10.0, 55.0, 7.0), Vector3(-25.25, 27.5, 8.5), mat_facade)
	parent.add_child(bw3)
	_add_window_strip(parent, "BW3", Vector3(-20.25, 0.0, 5.0), 7.0, 55.0, Vector3.RIGHT, 3)
	_add_ledges(parent, "BW3", Vector3(-25.25, 0.0, 8.5), 7.0, 55.0, Vector3.RIGHT)


# =============================================================================
# Boss Room — North Backdrop (Z < -15) — massive dead-end wall of towers
# =============================================================================

func _build_boss_room_north() -> void:
	var parent := Node3D.new()
	parent.name = "NorthBuildings"
	add_child(parent)

	# Tower N1: X=-19 to X=-7, height 55 — dominant left
	var n1 := _csg_box("N1", Vector3(12.0, 55.0, 8.0), Vector3(-13.0, 27.5, -19.25), mat_facade)
	parent.add_child(n1)
	_add_window_strip(parent, "N1", Vector3(-19.0, 0.0, -15.25), 12.0, 55.0, Vector3.FORWARD, 5)
	_add_ledges(parent, "N1", Vector3(-13.0, 0.0, -19.25), 12.0, 55.0, Vector3.FORWARD)

	# Tower N2: X=-5 to X=5, height 45 — center, slightly recessed
	var n2 := _csg_box("N2", Vector3(10.0, 45.0, 10.0), Vector3(0.0, 22.5, -20.25), mat_facade)
	parent.add_child(n2)
	_add_window_strip(parent, "N2", Vector3(-5.0, 0.0, -15.25), 10.0, 45.0, Vector3.FORWARD, 4)
	_add_ledges(parent, "N2", Vector3(0.0, 0.0, -20.25), 10.0, 45.0, Vector3.FORWARD)

	# Tower N3: X=7 to X=19, height 50 — right
	var n3 := _csg_box("N3", Vector3(12.0, 50.0, 8.0), Vector3(13.0, 25.0, -19.25), mat_facade)
	parent.add_child(n3)
	_add_window_strip(parent, "N3", Vector3(7.0, 0.0, -15.25), 12.0, 50.0, Vector3.FORWARD, 5)
	_add_ledges(parent, "N3", Vector3(13.0, 0.0, -19.25), 12.0, 50.0, Vector3.FORWARD)

	# Mega-tower behind N1 — visible silhouette above everything
	var n1_back := _csg_box("N1B", Vector3(14.0, 80.0, 10.0), Vector3(-13.0, 40.0, -28.0), mat_facade)
	parent.add_child(n1_back)


# =============================================================================
# Transition Building — short covered break between street and boss room
# Sits at the Z=12 boundary where the street narrows into the boss arena.
# Players walk through the ground floor of a building straddling the street.
# =============================================================================

func _build_transition_building() -> void:
	var parent := Node3D.new()
	parent.name = "TransitionBuilding"
	add_child(parent)

	# The transition spans Z=10 to Z=15, bridging the street (16 wide) to the
	# boss room (40 wide). The building sits ABOVE the walkable corridor.

	# Ceiling slab over the walkable path (collision to contain players)
	var ceil := _csg_box("TransCeil", Vector3(16.0, 1.5, 5.0), Vector3(0.0, 5.5, 12.5), mat_ceiling, true)
	parent.add_child(ceil)

	# Building mass above — the actual building that spans the street
	# Wide enough to connect to the flanking buildings on both sides
	var upper := _csg_box("TransUpper", Vector3(40.0, 30.0, 5.0), Vector3(0.0, 21.75, 12.5), mat_facade)
	parent.add_child(upper)

	# Structural supports on the sides (like columns where the building meets the ground)
	# East pillar — between hallway wall (X=8) and boss room wall (X=20)
	var pillar_e := _csg_box("TransPillarE", Vector3(12.0, 6.25, 5.0), Vector3(14.0, 3.125, 12.5), mat_facade)
	parent.add_child(pillar_e)
	# West pillar
	var pillar_w := _csg_box("TransPillarW", Vector3(12.0, 6.25, 5.0), Vector3(-14.0, 3.125, 12.5), mat_facade)
	parent.add_child(pillar_w)

	# Window strip on the south face (boss-room-facing side)
	_add_window_strip(parent, "TransS", Vector3(-20.0, 6.25, 10.0), 40.0, 24.0, Vector3.FORWARD, 12)

	# Window strip on the north face (street-facing side)
	_add_window_strip(parent, "TransN", Vector3(-20.0, 6.25, 15.0), 40.0, 24.0, Vector3.BACK, 12)

	# Ledges
	_add_ledges(parent, "TransS", Vector3(0.0, 6.25, 12.5), 40.0, 24.0, Vector3.FORWARD)
	_add_ledges(parent, "TransN", Vector3(0.0, 6.25, 12.5), 40.0, 24.0, Vector3.BACK)


# =============================================================================
# Warmup Lobby Ceiling
# =============================================================================

func _build_lobby_building() -> void:
	## The warmup lobby is the ground floor of a building. Ceiling with collision,
	## then the building mass rises above it.
	var parent := Node3D.new()
	parent.name = "LobbyBuilding"
	add_child(parent)

	# Ceiling slab (collision — keeps players inside)
	var ceil := _csg_box("LobbyCeiling", Vector3(16.0, 1.0, 12.0), Vector3(0.0, 5.25, 46.0), mat_ceiling, true)
	parent.add_child(ceil)

	# Building mass above — lobby X range is -8 to 8, Z is 40 to 52
	var upper := _csg_box("LobbyUpper", Vector3(20.0, 40.0, 16.0), Vector3(0.0, 25.75, 46.0), mat_facade)
	parent.add_child(upper)

	# Window strips on the street-facing side (south, Z=40)
	_add_window_strip(parent, "LobbySouth", Vector3(-10.0, 5.75, 38.0), 20.0, 34.0, Vector3.FORWARD, 7)
	_add_ledges(parent, "LobbySouth", Vector3(0.0, 5.75, 46.0), 20.0, 34.0, Vector3.FORWARD)


# =============================================================================
# Burning Buildings — fire particles + strong orange lights on select towers
# =============================================================================

func _build_burning_buildings() -> void:
	var parent := Node3D.new()
	parent.name = "BurningBuildings"
	add_child(parent)

	# Fire sources: position, height of the fire on the building, intensity scale
	var fires: Array[Dictionary] = [
		# Street east — SE2 upper floors on fire
		{"pos": Vector3(9.0, 18.0, 28.0), "height": 12.0, "energy": 5.0, "range": 18.0},
		# Street west — SW1 mid-section burning
		{"pos": Vector3(-9.0, 14.0, 18.0), "height": 8.0, "energy": 4.0, "range": 15.0},
		# Boss room — BW3 lower floors ablaze (near the fight)
		{"pos": Vector3(-21.0, 8.0, 7.0), "height": 10.0, "energy": 5.5, "range": 20.0},
		# Boss room north — N1 burning high up (dramatic backdrop)
		{"pos": Vector3(-14.0, 22.0, -16.0), "height": 10.0, "energy": 4.0, "range": 16.0},
		# Street east — SE3 ground level fire (near lobby end)
		{"pos": Vector3(9.0, 4.0, 36.0), "height": 6.0, "energy": 3.5, "range": 12.0},
	]

	for i in fires.size():
		var fire: Dictionary = fires[i]
		var pos: Vector3 = fire["pos"]
		var fire_height: float = fire["height"]
		var energy: float = fire["energy"]
		var light_range: float = fire["range"]

		# Main fire light — strong warm orange
		var light := OmniLight3D.new()
		light.name = "FireLight%d" % i
		light.transform.origin = pos
		light.light_color = Color(1.0, 0.55, 0.15)
		light.light_energy = energy
		light.omni_range = light_range
		light.shadow_enabled = i < 2  # shadows on the two biggest fires only
		parent.add_child(light)

		# Secondary light higher up — dimmer, redder (upper flames)
		var upper_light := OmniLight3D.new()
		upper_light.name = "FireUpperLight%d" % i
		upper_light.transform.origin = pos + Vector3(0.0, fire_height * 0.6, 0.0)
		upper_light.light_color = Color(0.9, 0.3, 0.05)
		upper_light.light_energy = energy * 0.4
		upper_light.omni_range = light_range * 0.6
		parent.add_child(upper_light)

		# Fire particles
		_add_fire_particles(parent, "Fire%d" % i, pos, fire_height)

		# Emissive glow patch on the building face (hot surface)
		var glow_mat := StandardMaterial3D.new()
		glow_mat.albedo_color = Color(0.15, 0.05, 0.02)
		glow_mat.emission_enabled = true
		glow_mat.emission = Color(0.8, 0.35, 0.05)
		glow_mat.emission_energy_multiplier = 1.5
		var glow_size := Vector3(3.0, fire_height, 0.1)
		var glow := _csg_box("FireGlow%d" % i, glow_size, pos, glow_mat)
		parent.add_child(glow)


func _add_fire_particles(parent: Node3D, fire_name: String, pos: Vector3, height: float) -> void:
	var fire := GPUParticles3D.new()
	fire.name = fire_name
	fire.amount = 200
	fire.lifetime = 1.5
	fire.transform.origin = pos
	fire.visibility_aabb = AABB(Vector3(-4, -2, -4), Vector3(8, height + 8, 8))

	var mat := ParticleProcessMaterial.new()
	mat.gravity = Vector3(0.0, 3.0, 0.0)  # fire rises
	mat.direction = Vector3(0.0, 1.0, 0.0)
	mat.spread = 25.0
	mat.initial_velocity_min = 2.0
	mat.initial_velocity_max = 5.0
	mat.emission_shape = ParticleProcessMaterial.EMISSION_SHAPE_BOX
	mat.emission_box_extents = Vector3(1.5, height * 0.3, 1.0)
	mat.scale_min = 0.6
	mat.scale_max = 1.4
	mat.damping_min = 1.0
	mat.damping_max = 3.0
	# Color: bright orange-yellow → dark red → black smoke
	var gradient := Gradient.new()
	gradient.set_color(0, Color(1.0, 0.8, 0.3, 0.9))
	gradient.add_point(0.3, Color(1.0, 0.4, 0.05, 0.7))
	gradient.add_point(0.6, Color(0.6, 0.15, 0.02, 0.5))
	gradient.set_color(1, Color(0.1, 0.1, 0.1, 0.0))
	var color_ramp := GradientTexture1D.new()
	color_ramp.gradient = gradient
	mat.color_ramp = color_ramp
	fire.process_material = mat

	# Billboard quad mesh for fire particles
	var mesh := QuadMesh.new()
	mesh.size = Vector2(0.8, 0.8)
	var draw_mat := StandardMaterial3D.new()
	draw_mat.transparency = BaseMaterial3D.TRANSPARENCY_ALPHA
	draw_mat.shading_mode = BaseMaterial3D.SHADING_MODE_UNSHADED
	draw_mat.billboard_mode = BaseMaterial3D.BILLBOARD_ENABLED
	draw_mat.vertex_color_use_as_albedo = true
	mesh.material = draw_mat
	fire.draw_pass_1 = mesh

	parent.add_child(fire)

	# Smoke rising above the fire
	var smoke := GPUParticles3D.new()
	smoke.name = fire_name + "Smoke"
	smoke.amount = 80
	smoke.lifetime = 3.0
	smoke.transform.origin = pos + Vector3(0.0, height * 0.5, 0.0)
	smoke.visibility_aabb = AABB(Vector3(-6, -2, -6), Vector3(12, height + 20, 12))

	var smoke_mat := ParticleProcessMaterial.new()
	smoke_mat.gravity = Vector3(0.3, 1.5, 0.1)  # slight wind drift
	smoke_mat.direction = Vector3(0.1, 1.0, 0.0)
	smoke_mat.spread = 20.0
	smoke_mat.initial_velocity_min = 1.0
	smoke_mat.initial_velocity_max = 3.0
	smoke_mat.emission_shape = ParticleProcessMaterial.EMISSION_SHAPE_BOX
	smoke_mat.emission_box_extents = Vector3(1.5, 1.0, 1.0)
	smoke_mat.scale_min = 1.5
	smoke_mat.scale_max = 3.0
	var smoke_gradient := Gradient.new()
	smoke_gradient.set_color(0, Color(0.2, 0.18, 0.15, 0.4))
	smoke_gradient.set_color(1, Color(0.1, 0.1, 0.1, 0.0))
	var smoke_ramp := GradientTexture1D.new()
	smoke_ramp.gradient = smoke_gradient
	smoke_mat.color_ramp = smoke_ramp
	smoke.process_material = smoke_mat

	var smoke_mesh := QuadMesh.new()
	smoke_mesh.size = Vector2(2.0, 2.0)
	var smoke_draw := StandardMaterial3D.new()
	smoke_draw.transparency = BaseMaterial3D.TRANSPARENCY_ALPHA
	smoke_draw.shading_mode = BaseMaterial3D.SHADING_MODE_UNSHADED
	smoke_draw.billboard_mode = BaseMaterial3D.BILLBOARD_ENABLED
	smoke_draw.vertex_color_use_as_albedo = true
	smoke_mesh.material = smoke_draw
	smoke.draw_pass_1 = smoke_mesh

	parent.add_child(smoke)


# =============================================================================
# Helpers
# =============================================================================

func _csg_box(box_name: String, size: Vector3, pos: Vector3, mat: StandardMaterial3D, collision: bool = false) -> CSGBox3D:
	var box := CSGBox3D.new()
	box.name = box_name
	box.size = size
	box.transform.origin = pos
	box.material = mat
	box.use_collision = collision
	if collision:
		box.collision_layer = 1
		box.collision_mask = 0
	return box


func _add_ledges(parent: Node3D, prefix: String, center: Vector3, span: float, height: float, face_dir: Vector3) -> void:
	## Horizontal ledge strips at floor-division intervals — futuristic floor plates.
	var ledge_interval := 4.0
	var y := ledge_interval
	var idx := 0
	while y < height:
		var ledge_size: Vector3
		var ledge_pos: Vector3
		if face_dir == Vector3.LEFT or face_dir == Vector3.RIGHT:
			ledge_size = Vector3(0.4, 0.15, span)
			ledge_pos = Vector3(center.x + face_dir.x * -0.18, y, center.z)
		elif face_dir == Vector3.FORWARD or face_dir == Vector3.BACK:
			ledge_size = Vector3(span, 0.15, 0.4)
			ledge_pos = Vector3(center.x, y, center.z + face_dir.z * -0.18)
		else:
			break
		var ledge := _csg_box("%sLedge%d" % [prefix, idx], ledge_size, ledge_pos, mat_ledge)
		parent.add_child(ledge)
		y += ledge_interval
		idx += 1


func _add_window_strip(parent: Node3D, prefix: String, corner: Vector3, span: float, height: float, face_dir: Vector3, count_along: int) -> void:
	## Vertical window strips running up the facade — futuristic glass columns.
	## Each strip is a tall, narrow dark recess with faint blue glow.
	var strip_w := 0.8
	var spacing := span / float(count_along)
	# Window strips run from ~3m up to near the top
	var strip_bottom := 3.0
	var strip_height := height - 5.0
	if strip_height < 4.0:
		return
	for i in count_along:
		var along := spacing * (float(i) + 0.5)
		var pos: Vector3
		var size: Vector3
		if face_dir == Vector3.LEFT or face_dir == Vector3.RIGHT:
			pos = Vector3(corner.x + face_dir.x * 0.05, strip_bottom + strip_height * 0.5, corner.z + along)
			size = Vector3(0.15, strip_height, strip_w)
		elif face_dir == Vector3.FORWARD or face_dir == Vector3.BACK:
			pos = Vector3(corner.x + along, strip_bottom + strip_height * 0.5, corner.z + face_dir.z * 0.05)
			size = Vector3(strip_w, strip_height, 0.15)
		else:
			continue
		# Alternate between dark and faintly glowing strips
		var mat: StandardMaterial3D = mat_window_glow if i % 3 == 0 else mat_facade_dark
		var win := _csg_box("%sStrip%d" % [prefix, i], size, pos, mat)
		parent.add_child(win)
