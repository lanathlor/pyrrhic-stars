extends SceneTree
## Headless adversarial tests for portal trail navmesh baking.
## Run: godot4 --headless --script res://tests/test_portal_trail_nav.gd
##
## Verifies that navmesh baking carves out buildings and paths route around them.

# Exact building data from portal_trail.gd
const BUILDINGS := [
	{"center": Vector3(-65, 0, -125), "size": Vector3(50, 0, 45)},  # 0 A1
	{"center": Vector3(-20, 0, -130), "size": Vector3(30, 0, 30)},  # 1 A2
	{"center": Vector3(30, 0, -125), "size": Vector3(40, 0, 40)},  # 2 A3
	{"center": Vector3(75, 0, -130), "size": Vector3(25, 0, 35)},  # 3 A4
	{"center": Vector3(-65, 0, -80), "size": Vector3(40, 0, 15)},  # 4 B1
	{"center": Vector3(-68, 0, -55), "size": Vector3(35, 0, 14)},  # 5 B2
	{"center": Vector3(-62, 0, -30), "size": Vector3(42, 0, 16)},  # 6 B3
	{"center": Vector3(30, 0, -82), "size": Vector3(30, 0, 12)},  # 7 C1
	{"center": Vector3(65, 0, -80), "size": Vector3(30, 0, 14)},  # 8 C2
	{"center": Vector3(35, 0, -42), "size": Vector3(28, 0, 18)},  # 9 C3
	{"center": Vector3(70, 0, -38), "size": Vector3(35, 0, 15)},  # 10 C4
	{"center": Vector3(-60, 0, 15), "size": Vector3(45, 0, 40)},  # 11 D1
	{"center": Vector3(-20, 0, 20), "size": Vector3(30, 0, 30)},  # 12 D2
	{"center": Vector3(30, 0, 18), "size": Vector3(35, 0, 35)},  # 13 D3
	{"center": Vector3(75, 0, 15), "size": Vector3(28, 0, 35)},  # 14 D4
	{"center": Vector3(-8, 0, -55), "size": Vector3(2, 0, 2)},  # 15 Monument
	{"center": Vector3(5, 0, -55), "size": Vector3(10, 0, 10)},  # 16 Lift shaft
]

const FLOOR_Y := -200.0
var _nav_map_rid: RID
var _region_rid: RID
var _pass := 0
var _fail := 0
var _errors: Array[String] = []


func _init() -> void:
	_bake_navmesh()
	call_deferred("_run_tests")


func _add_box_faces(sg: NavigationMeshSourceGeometryData3D, center: Vector3, size: Vector3) -> void:
	var hx := size.x / 2.0
	var hy := size.y / 2.0
	var hz := size.z / 2.0
	var cx := center.x
	var cy := center.y
	var cz := center.z
	(
		sg
		. add_faces(
			PackedVector3Array(
				[
					# Bottom
					Vector3(cx - hx, cy - hy, cz - hz),
					Vector3(cx + hx, cy - hy, cz - hz),
					Vector3(cx + hx, cy - hy, cz + hz),
					Vector3(cx - hx, cy - hy, cz - hz),
					Vector3(cx + hx, cy - hy, cz + hz),
					Vector3(cx - hx, cy - hy, cz + hz),
					# Top
					Vector3(cx - hx, cy + hy, cz - hz),
					Vector3(cx + hx, cy + hy, cz + hz),
					Vector3(cx + hx, cy + hy, cz - hz),
					Vector3(cx - hx, cy + hy, cz - hz),
					Vector3(cx - hx, cy + hy, cz + hz),
					Vector3(cx + hx, cy + hy, cz + hz),
					# Front (Z-)
					Vector3(cx - hx, cy - hy, cz - hz),
					Vector3(cx + hx, cy + hy, cz - hz),
					Vector3(cx + hx, cy - hy, cz - hz),
					Vector3(cx - hx, cy - hy, cz - hz),
					Vector3(cx - hx, cy + hy, cz - hz),
					Vector3(cx + hx, cy + hy, cz - hz),
					# Back (Z+)
					Vector3(cx - hx, cy - hy, cz + hz),
					Vector3(cx + hx, cy - hy, cz + hz),
					Vector3(cx + hx, cy + hy, cz + hz),
					Vector3(cx - hx, cy - hy, cz + hz),
					Vector3(cx + hx, cy + hy, cz + hz),
					Vector3(cx - hx, cy + hy, cz + hz),
					# Left (X-)
					Vector3(cx - hx, cy - hy, cz - hz),
					Vector3(cx - hx, cy - hy, cz + hz),
					Vector3(cx - hx, cy + hy, cz + hz),
					Vector3(cx - hx, cy - hy, cz - hz),
					Vector3(cx - hx, cy + hy, cz + hz),
					Vector3(cx - hx, cy + hy, cz - hz),
					# Right (X+)
					Vector3(cx + hx, cy - hy, cz - hz),
					Vector3(cx + hx, cy + hy, cz + hz),
					Vector3(cx + hx, cy - hy, cz + hz),
					Vector3(cx + hx, cy - hy, cz - hz),
					Vector3(cx + hx, cy + hy, cz - hz),
					Vector3(cx + hx, cy + hy, cz + hz),
				]
			),
			Transform3D.IDENTITY
		)
	)


func _bake_navmesh() -> void:
	_nav_map_rid = NavigationServer3D.map_create()
	NavigationServer3D.map_set_active(_nav_map_rid, true)
	NavigationServer3D.map_set_cell_size(_nav_map_rid, 0.5)
	NavigationServer3D.map_set_cell_height(_nav_map_rid, 0.5)

	var nav_mesh := NavigationMesh.new()
	nav_mesh.agent_radius = 1.0
	nav_mesh.agent_height = 2.0
	nav_mesh.agent_max_climb = 0.5
	nav_mesh.agent_max_slope = 50.0
	nav_mesh.cell_size = 0.5
	nav_mesh.cell_height = 0.5

	var source_geo := NavigationMeshSourceGeometryData3D.new()

	# Floor
	var hx := 100.0
	var hz := 100.0
	var cx := 5.0
	var cz := -55.0
	(
		source_geo
		. add_faces(
			PackedVector3Array(
				[
					Vector3(cx - hx, FLOOR_Y, cz - hz),
					Vector3(cx + hx, FLOOR_Y, cz - hz),
					Vector3(cx + hx, FLOOR_Y, cz + hz),
					Vector3(cx - hx, FLOOR_Y, cz - hz),
					Vector3(cx + hx, FLOOR_Y, cz + hz),
					Vector3(cx - hx, FLOOR_Y, cz + hz),
				]
			),
			Transform3D.IDENTITY
		)
	)

	# Add buildings as solid wall boxes — navmesh baker will carve around them
	for obs in BUILDINGS:
		var ocx: float = obs["center"].x
		var ocz: float = obs["center"].z
		var osx: float = obs["size"].x
		var osz: float = obs["size"].z
		# Building walls extend from floor up 5m — baker will see them as obstacles
		_add_box_faces(source_geo, Vector3(ocx, FLOOR_Y + 2.5, ocz), Vector3(osx, 5.0, osz))

	NavigationServer3D.bake_from_source_geometry_data(nav_mesh, source_geo)
	print(
		(
			"\nReal navmesh: %d polygons, %d vertices"
			% [nav_mesh.get_polygon_count(), nav_mesh.get_vertices().size()]
		)
	)

	_region_rid = NavigationServer3D.region_create()
	NavigationServer3D.region_set_map(_region_rid, _nav_map_rid)
	NavigationServer3D.region_set_navigation_mesh(_region_rid, nav_mesh)
	NavigationServer3D.map_force_update(_nav_map_rid)


func _run_tests() -> void:
	# NavigationServer needs physics frames to sync the map
	for i in 5:
		await root.get_tree().physics_frame
	NavigationServer3D.map_force_update(_nav_map_rid)

	_test_navmesh_has_polygons()
	_test_path_on_open_street()
	_test_spawn_to_lift_is_near()
	_test_path_avoids_building_a1()
	_test_path_avoids_building_d2()
	_test_path_avoids_building_d3()
	_test_cross_district_avoids_all()
	_test_canyon_to_lift()
	_test_row_c_uses_street()
	_test_path_from_inside_building()
	_test_path_longer_than_straight_line()
	_test_no_path_point_inside_any_building()

	print("\n========================================")
	print("RESULTS: %d passed, %d failed" % [_pass, _fail])
	if _errors.size() > 0:
		print("FAILURES:")
		for e in _errors:
			print("  ✗ %s" % e)
	print("========================================")

	NavigationServer3D.free_rid(_region_rid)
	NavigationServer3D.free_rid(_nav_map_rid)
	quit(1 if _fail > 0 else 0)


# =============================================================================
# Helpers
# =============================================================================


func _get_path(from: Vector3, to: Vector3) -> PackedVector3Array:
	return NavigationServer3D.map_get_path(_nav_map_rid, from, to, true)


func _check(name: String, condition: bool, msg: String = "") -> void:
	if condition:
		_pass += 1
		print("  ✓ %s" % name)
	else:
		_fail += 1
		var full := "%s: %s" % [name, msg] if msg != "" else name
		_errors.append(full)
		print("  ✗ %s — %s" % [name, msg])


func _point_inside_building(point: Vector3, bld: Dictionary) -> bool:
	var bcx: float = bld["center"].x
	var bcz: float = bld["center"].z
	var bsx: float = bld["size"].x / 2.0
	var bsz: float = bld["size"].z / 2.0
	return (
		point.x >= bcx - bsx
		and point.x <= bcx + bsx
		and point.z >= bcz - bsz
		and point.z <= bcz + bsz
	)


func _segment_intersects_aabb(
	a: Vector3, b: Vector3, min_x: float, max_x: float, min_z: float, max_z: float
) -> bool:
	var dx := b.x - a.x
	var dz := b.z - a.z
	var p := [-dx, dx, -dz, dz]
	var q := [a.x - min_x, max_x - a.x, a.z - min_z, max_z - a.z]

	var t_min := 0.0
	var t_max := 1.0
	for i in 4:
		if absf(p[i]) < 1e-8:
			if q[i] < 0.0:
				return false
		else:
			var t: float = q[i] / p[i]
			if p[i] < 0.0:
				t_min = maxf(t_min, t)
			else:
				t_max = minf(t_max, t)
			if t_min > t_max:
				return false
	return true


func _segment_intersects_building(a: Vector3, b: Vector3, bld: Dictionary) -> bool:
	var bcx: float = bld["center"].x
	var bcz: float = bld["center"].z
	var bsx: float = bld["size"].x / 2.0
	var bsz: float = bld["size"].z / 2.0
	return _segment_intersects_aabb(a, b, bcx - bsx, bcx + bsx, bcz - bsz, bcz + bsz)


func _path_violated_buildings(path: PackedVector3Array) -> Array[int]:
	var violated: Array[int] = []
	for bi in BUILDINGS.size():
		var bld: Dictionary = BUILDINGS[bi]
		for i in range(path.size() - 1):
			if _segment_intersects_building(path[i], path[i + 1], bld):
				violated.append(bi)
				break
	return violated


func _path_length(path: PackedVector3Array) -> float:
	var d := 0.0
	for i in range(1, path.size()):
		d += path[i - 1].distance_to(path[i])
	return d


# =============================================================================
# Tests
# =============================================================================


func _test_navmesh_has_polygons() -> void:
	var path := _get_path(Vector3(3.5, FLOOR_Y, -45.0), Vector3(5.0, FLOOR_Y, -45.0))
	_check("navmesh_has_polygons", path.size() > 0, "Navmesh returned zero points — baking failed")


func _test_path_on_open_street() -> void:
	var path := _get_path(Vector3(5.0, FLOOR_Y, -100.0), Vector3(5.0, FLOOR_Y, -10.0))
	_check(
		"path_on_open_street",
		path.size() >= 2,
		"No path found on open central avenue (got %d points)" % path.size()
	)


func _test_spawn_to_lift_is_near() -> void:
	var path := _get_path(Vector3(3.5, FLOOR_Y, -45.0), Vector3(5.0, FLOOR_Y, -45.0))
	if path.size() < 2:
		_check("spawn_to_lift_near", false, "No path from spawn to lift")
		return
	var d := _path_length(path)
	_check("spawn_to_lift_near", d < 10.0, "Path length %0.1fm — expected < 10m" % d)


func _test_path_avoids_building_a1() -> void:
	var path := _get_path(Vector3(-90.0, FLOOR_Y, -90.0), Vector3(-40.0, FLOOR_Y, -140.0))
	if path.size() < 2:
		_check("avoids_A1", false, "No path returned")
		return
	var a1 := BUILDINGS[0]
	var hit := false
	for i in range(path.size() - 1):
		if _segment_intersects_building(path[i], path[i + 1], a1):
			hit = true
			break
	_check(
		"avoids_A1",
		not hit,
		"Path cuts through building A1 (center %s size %s)" % [a1["center"], a1["size"]]
	)


func _test_path_avoids_building_d2() -> void:
	var path := _get_path(Vector3(-35.0, FLOOR_Y, 0.0), Vector3(-5.0, FLOOR_Y, 35.0))
	if path.size() < 2:
		_check("avoids_D2", false, "No path returned")
		return
	var d2 := BUILDINGS[12]
	var hit := false
	for i in range(path.size() - 1):
		if _segment_intersects_building(path[i], path[i + 1], d2):
			hit = true
			break
	_check(
		"avoids_D2",
		not hit,
		"Path cuts through building D2 (center %s size %s)" % [d2["center"], d2["size"]]
	)


func _test_path_avoids_building_d3() -> void:
	var path := _get_path(Vector3(10.0, FLOOR_Y, 0.0), Vector3(50.0, FLOOR_Y, 36.0))
	if path.size() < 2:
		_check("avoids_D3", false, "No path returned")
		return
	var d3 := BUILDINGS[13]
	var hit := false
	for i in range(path.size() - 1):
		if _segment_intersects_building(path[i], path[i + 1], d3):
			hit = true
			break
	_check(
		"avoids_D3",
		not hit,
		"Path cuts through building D3 (center %s size %s)" % [d3["center"], d3["size"]]
	)


func _test_cross_district_avoids_all() -> void:
	var path := _get_path(Vector3(-85.0, FLOOR_Y, 30.0), Vector3(85.0, FLOOR_Y, -140.0))
	if path.size() < 2:
		_check("cross_district_avoids_all", false, "No path returned")
		return
	var violated := _path_violated_buildings(path)
	_check(
		"cross_district_avoids_all",
		violated.is_empty(),
		"Path cuts through %d buildings: %s" % [violated.size(), str(violated)]
	)


func _test_canyon_to_lift() -> void:
	# B2 spans X[-85.5,-50.5] Z[-62,-48]. Start west of B2 in the street.
	var path := _get_path(Vector3(-90.0, FLOOR_Y, -55.0), Vector3(5.0, FLOOR_Y, -45.0))
	if path.size() < 2:
		_check("canyon_to_lift", false, "No path returned")
		return
	var violated := _path_violated_buildings(path)
	_check("canyon_to_lift", violated.is_empty(), "Path cuts through buildings: %s" % str(violated))


func _test_row_c_uses_street() -> void:
	var path := _get_path(Vector3(20.0, FLOOR_Y, -70.0), Vector3(75.0, FLOOR_Y, -70.0))
	if path.size() < 2:
		_check("row_c_uses_street", false, "No path returned")
		return
	var violated := _path_violated_buildings(path)
	_check(
		"row_c_uses_street", violated.is_empty(), "Path cuts through buildings: %s" % str(violated)
	)


func _test_path_from_inside_building() -> void:
	var path := _get_path(Vector3(30.0, FLOOR_Y, -125.0), Vector3(5.0, FLOOR_Y, -45.0))
	_check("path_from_inside_building", path.size() >= 2, "No path from inside building A3 to lift")


func _test_path_longer_than_straight_line() -> void:
	var from := Vector3(-82.5, FLOOR_Y, -5.0)
	var to := Vector3(-37.5, FLOOR_Y, 35.0)
	var straight := from.distance_to(to)
	var path := _get_path(from, to)
	if path.size() < 2:
		_check("path_longer_than_straight", false, "No path returned")
		return
	var d := _path_length(path)
	_check(
		"path_longer_than_straight",
		d > straight,
		"Path %0.1fm should exceed straight-line %0.1fm" % [d, straight]
	)


func _test_no_path_point_inside_any_building() -> void:
	var routes := [
		[Vector3(-80.0, FLOOR_Y, -90.0), Vector3(80.0, FLOOR_Y, -90.0)],
		[Vector3(-90.0, FLOOR_Y, -55.0), Vector3(90.0, FLOOR_Y, -55.0)],
		[Vector3(-90.0, FLOOR_Y, 0.0), Vector3(90.0, FLOOR_Y, 0.0)],
		[Vector3(5.0, FLOOR_Y, -140.0), Vector3(5.0, FLOOR_Y, 30.0)],
		[Vector3(-85.0, FLOOR_Y, 30.0), Vector3(85.0, FLOOR_Y, -140.0)],
	]
	var found_inside := ""
	for route in routes:
		var path := _get_path(route[0], route[1])
		for pi in path.size():
			for bi in BUILDINGS.size():
				if _point_inside_building(path[pi], BUILDINGS[bi]):
					found_inside = (
						"Point %d (%s) route %s→%s inside building %d"
						% [pi, path[pi], route[0], route[1], bi]
					)
					break
			if found_inside != "":
				break
		if found_inside != "":
			break
	_check("no_point_inside_any_building", found_inside == "", found_inside)
