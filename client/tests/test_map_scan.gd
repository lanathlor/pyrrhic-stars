extends SceneTree
## Adversarial tests for the map scanner: loads the REAL hub and arena scenes,
## runs scan_environment at each floor's Y level, and asserts that a minimum
## number of geometry shapes are found. If the scanner misses geometry, these
## tests MUST fail.
##
## Run: godot4 --headless --script res://tests/test_map_scan.gd

var _pass := 0
var _fail := 0
var _errors: Array[String] = []


func _init() -> void:
	call_deferred("_run_tests")


func _run_tests() -> void:
	await root.get_tree().process_frame
	await root.get_tree().process_frame

	print("\n=== Map Scanner — Real Scene Tests ===")

	# Load the REAL hub scene
	var hub_scene := load("res://scenes/environments/prime_hub/military_building.tscn") as PackedScene
	var hub := hub_scene.instantiate()
	root.add_child(hub)
	await root.get_tree().process_frame  # let transforms propagate

	var overlay := Control.new()
	overlay.set_script(load("res://scenes/shared/hud/map_overlay.gd"))
	overlay.size = Vector2(1920, 1080)
	overlay.visible = false
	root.add_child(overlay)

	# --- Lower District (Y = -200) ---
	print("\n-- Lower District (Y=-200) --")
	overlay._player_pos = Vector3(5.0, -200.0, -55.0)
	overlay.scan_environment(hub)

	_check("lower_district_has_rects",
		overlay._scanned_rects.size() > 0,
		"Expected scanned rects > 0, got %d" % overlay._scanned_rects.size())

	# 15 buildings with use_collision + ground floor + other CSG = many shapes
	# Lower district has at minimum 15 building blocks (A1-D4 + monument + lift shaft area)
	_check("lower_district_min_15_rects",
		overlay._scanned_rects.size() >= 15,
		"Expected >= 15 building rects, got %d" % overlay._scanned_rects.size())

	# Verify specific building A1 is present: center=(-65,-125) size=(50,45)
	# Expected rect: Rect2(-90, -147.5, 50, 45)
	var found_a1 := _find_rect_near(overlay._scanned_rects, -65.0, -125.0, 5.0)
	_check("lower_district_building_a1_found", found_a1,
		"Building A1 (center -65,-125) not found in scanned rects")

	# Verify building D3 is present: center=(30,18) size=(35,35)
	var found_d3 := _find_rect_near(overlay._scanned_rects, 30.0, 18.0, 5.0)
	_check("lower_district_building_d3_found", found_d3,
		"Building D3 (center 30,18) not found in scanned rects")

	# Verify bounds are reasonable (should cover ~200x200 area)
	_check("lower_district_bounds_width",
		overlay._floor_size.x > 100.0,
		"Floor width %.1f too small, expected > 100" % overlay._floor_size.x)

	_check("lower_district_bounds_height",
		overlay._floor_size.y > 100.0,
		"Floor height %.1f too small, expected > 100" % overlay._floor_size.y)

	# --- Plaza level (Y = 0) ---
	print("\n-- Plaza level (Y=0, outside tower) --")
	overlay._player_pos = Vector3(50.0, 0.0, -30.0)
	overlay.scan_environment(hub)

	_check("plaza_has_rects",
		overlay._scanned_rects.size() > 0,
		"Expected scanned rects > 0 at plaza, got %d" % overlay._scanned_rects.size())

	# Tower walls should be visible at plaza level
	# FrontWallLeft at X=-17 Z=-1, FrontWallRight at X=17 Z=-1, etc.
	var found_front_wall := _find_rect_near(overlay._scanned_rects, -17.0, -1.0, 3.0)
	_check("plaza_tower_front_wall_found", found_front_wall,
		"Tower front wall (near -17,-1) not found in scanned rects")

	# --- Tower lobby (Y = 0, inside tower) ---
	print("\n-- Tower lobby (Y=0, inside tower) --")
	overlay._player_pos = Vector3(0.0, 0.0, 20.0)
	overlay.scan_environment(hub)

	_check("lobby_has_rects",
		overlay._scanned_rects.size() > 0,
		"Expected scanned rects > 0 in lobby, got %d" % overlay._scanned_rects.size())

	# Should see lobby floor, walls, etc.
	_check("lobby_min_3_rects",
		overlay._scanned_rects.size() >= 3,
		"Expected >= 3 rects in lobby, got %d" % overlay._scanned_rects.size())

	# --- Ops level (Y = 100) ---
	print("\n-- Ops level (Y=100) --")
	overlay._player_pos = Vector3(10.0, 100.0, 20.0)
	overlay.scan_environment(hub)

	_check("ops_has_rects",
		overlay._scanned_rects.size() > 0,
		"Expected scanned rects > 0 at ops, got %d" % overlay._scanned_rects.size())

	# Should see ops floor, partitions, landing pad
	_check("ops_min_3_rects",
		overlay._scanned_rects.size() >= 3,
		"Expected >= 3 rects at ops, got %d" % overlay._scanned_rects.size())

	# Landing pad area should be visible (around X=33, Z=5.5)
	var found_landing := _find_rect_near(overlay._scanned_rects, 33.0, 5.5, 10.0)
	_check("ops_landing_pad_found", found_landing,
		"Landing pad area (near 33, 5.5) not found in scanned rects")

	# --- Floors should NOT bleed into each other ---
	print("\n-- Floor isolation --")

	# At lower district Y=-200, should NOT see ops-level partitions (Y=100)
	overlay._player_pos = Vector3(5.0, -200.0, -55.0)
	overlay.scan_environment(hub)
	# Ops partitions are at Y=100 — they must not appear at Y=-200
	var found_ops_partition := _find_rect_near(overlay._scanned_rects, -18.0, 10.0, 2.0)
	_check("no_ops_partition_at_lower_district", not found_ops_partition,
		"Ops partition (near -18,10) should not appear when scanning at Y=-200")

	hub.queue_free()

	# --- Arena scene ---
	print("\n-- Arena --")
	var arena_scene := load("res://scenes/environments/arena/arena.tscn") as PackedScene
	var arena := arena_scene.instantiate()
	root.add_child(arena)
	await root.get_tree().process_frame

	overlay._player_pos = Vector3(0.0, 0.0, 0.0)
	overlay.scan_environment(arena)

	_check("arena_has_rects",
		overlay._scanned_rects.size() > 0,
		"Expected scanned rects > 0 in arena, got %d" % overlay._scanned_rects.size())

	# Arena has: floor, 3 walls, 6 pillars, 4 covers = at minimum 10+
	_check("arena_min_10_rects",
		overlay._scanned_rects.size() >= 10,
		"Expected >= 10 rects in arena (walls+pillars+floor), got %d" % overlay._scanned_rects.size())

	# Verify a pillar is found (Pillar1 at -8, -6)
	var found_pillar := _find_rect_near(overlay._scanned_rects, -8.0, -6.0, 2.0)
	_check("arena_pillar_found", found_pillar,
		"Pillar1 (near -8,-6) not found in scanned rects")

	arena.queue_free()
	overlay.queue_free()

	# --- Minimap scanner (shared_hud) ---
	print("\n-- Minimap scanner (shared_hud) --")
	var hub2 := hub_scene.instantiate()
	root.add_child(hub2)
	await root.get_tree().process_frame

	var hud := Control.new()
	hud.set_script(load("res://scenes/shared/hud/shared_hud.gd"))
	root.add_child(hud)
	hud.set_environment(hub2)
	hud._hub_mode = true

	# Simulate a player at lower district
	var fake_player := CharacterBody3D.new()
	fake_player.global_position = Vector3(5.0, -200.0, -55.0)
	root.add_child(fake_player)
	hud.set_local_player(fake_player, "gunner", 1)

	hud._detect_floor(Vector3(5.0, -200.0, -55.0))

	_check("minimap_lower_district_has_rects",
		hud._floor_rects.size() > 0,
		"Minimap should have rects for lower district, got %d" % hud._floor_rects.size())

	_check("minimap_lower_district_min_15",
		hud._floor_rects.size() >= 15,
		"Minimap expected >= 15 rects, got %d" % hud._floor_rects.size())

	fake_player.queue_free()
	hud.queue_free()
	hub2.queue_free()

	# --- Classification tests ---
	print("\n-- Node classification --")

	var hub3 := hub_scene.instantiate()
	root.add_child(hub3)
	await root.get_tree().process_frame

	var overlay2 := Control.new()
	overlay2.set_script(load("res://scenes/shared/hud/map_overlay.gd"))
	overlay2.size = Vector2(1920, 1080)
	overlay2.visible = false
	root.add_child(overlay2)

	# Scan plaza and check types
	overlay2._player_pos = Vector3(0.0, 0.0, -20.0)
	overlay2.scan_environment(hub3)

	# GrassTop (250x272, h=0.1) should be classified as "garden", not "wall"
	var grass_type := ""
	for entry in overlay2._scanned_rects:
		var r: Rect2 = entry["rect"]
		if r.size.x > 200.0 and r.size.y > 200.0:
			grass_type = entry["type"]
			break
	_check("plaza_grass_is_garden", grass_type == "garden",
		"GrassTop should be 'garden' type, got '%s'" % grass_type)

	# Sidewalk paths should be present as "ground"
	var found_path := false
	for entry in overlay2._scanned_rects:
		var r: Rect2 = entry["rect"]
		if r.size.x < 5.0 and r.size.y > 50.0 and entry["type"] == "ground":
			found_path = true
			break
	_check("plaza_sidewalk_path_shown", found_path,
		"Narrow sidewalk paths should appear as ground type")

	# Buildings A1 at lower district should be "wall" type
	overlay2._player_pos = Vector3(5.0, -200.0, -55.0)
	overlay2.scan_environment(hub3)
	var a1_is_wall := false
	for entry in overlay2._scanned_rects:
		var r: Rect2 = entry["rect"]
		var cx: float = r.position.x + r.size.x / 2.0
		var cz: float = r.position.y + r.size.y / 2.0
		if absf(cx - (-65.0)) < 5.0 and absf(cz - (-125.0)) < 5.0:
			a1_is_wall = entry["type"] == "wall"
			break
	_check("lower_district_a1_is_wall", a1_is_wall,
		"Building A1 should be classified as wall")

	# Greenery should be tagged green
	overlay2._player_pos = Vector3(0.0, 0.0, -20.0)
	overlay2.scan_environment(hub3)
	var found_green := false
	for entry in overlay2._scanned_rects:
		if entry["type"] == "green":
			found_green = true
			break
	for circ in overlay2._scanned_circles:
		if circ.get("green", false):
			found_green = true
			break
	_check("plaza_has_greenery", found_green,
		"Plaza should have green-tagged shapes (hedges, planters)")

	overlay2.queue_free()
	hub3.queue_free()

	# --- Minimap must rescan when called again on same floor ---
	print("\n-- Minimap rescan on same floor --")

	var hub_rescan := hub_scene.instantiate()
	root.add_child(hub_rescan)
	await root.get_tree().process_frame

	var hud_rescan := Control.new()
	hud_rescan.set_script(load("res://scenes/shared/hud/shared_hud.gd"))
	root.add_child(hud_rescan)
	hud_rescan.set_environment(hub_rescan)
	hud_rescan._hub_mode = true

	var fp := CharacterBody3D.new()
	fp.global_position = Vector3(0.0, 0.0, -20.0)
	root.add_child(fp)
	hud_rescan.set_local_player(fp, "gunner", 1)

	# First detect — populates floor rects
	hud_rescan._detect_floor(Vector3(0.0, 0.0, -20.0))
	var first_count: int = hud_rescan._floor_rects.size()
	_check("rescan_first_detect_has_rects", first_count > 0,
		"First detect should populate rects, got %d" % first_count)

	# Manually corrupt the cached data to simulate stale state
	hud_rescan._floor_rects.clear()

	# Second detect on the SAME floor — should rescan, not skip
	hud_rescan._detect_floor(Vector3(0.0, 0.0, -20.0))
	var second_count: int = hud_rescan._floor_rects.size()
	_check("rescan_second_detect_repopulates", second_count == first_count,
		"Second detect on same floor should rescan, got %d (expected %d)" % [second_count, first_count])

	fp.queue_free()
	hud_rescan.queue_free()
	hub_rescan.queue_free()

	# --- GroundPlaneN must not appear ---
	print("\n-- GroundPlaneN exclusion --")

	var hub4 := hub_scene.instantiate()
	root.add_child(hub4)
	await root.get_tree().process_frame

	var overlay3 := Control.new()
	overlay3.set_script(load("res://scenes/shared/hud/map_overlay.gd"))
	overlay3.size = Vector2(1920, 1080)
	overlay3.visible = false
	root.add_child(overlay3)

	# Player standing on the plaza at Y=0
	overlay3._player_pos = Vector3(0.0, 0.0, -20.0)
	overlay3.scan_environment(hub4)

	# GroundPlaneN: pos=(2, -2, -77.5) size=(6, 4, 57) -> rect Rect2(-1, -106, 6, 57)
	# It's a sub-floor fill (Y=-4 to 0), should NOT appear.
	# Note: PathNorth (3x75) is at a similar position and IS expected — match by width=6.
	var found_ground_plane_n := false
	for entry in overlay3._scanned_rects:
		var r: Rect2 = entry["rect"]
		if absf(r.size.x - 6.0) < 0.5 and absf(r.size.y - 57.0) < 1.0:
			found_ground_plane_n = true
			break
	_check("no_ground_plane_n_at_plaza", not found_ground_plane_n,
		"GroundPlaneN (6x57 sub-floor fill) should not appear on plaza map")

	# Also check the minimap scanner
	var hud2 := Control.new()
	hud2.set_script(load("res://scenes/shared/hud/shared_hud.gd"))
	root.add_child(hud2)
	hud2.set_environment(hub4)
	hud2._hub_mode = true
	var fake_player2 := CharacterBody3D.new()
	fake_player2.global_position = Vector3(0.0, 0.0, -20.0)
	root.add_child(fake_player2)
	hud2.set_local_player(fake_player2, "gunner", 1)
	hud2._detect_floor(Vector3(0.0, 0.0, -20.0))

	var found_in_minimap := false
	for entry in hud2._floor_rects:
		var r: Rect2 = entry["rect"] if entry is Dictionary else entry
		if absf(r.size.x - 6.0) < 0.5 and absf(r.size.y - 57.0) < 1.0:
			found_in_minimap = true
			break
	_check("no_ground_plane_n_in_minimap", not found_in_minimap,
		"GroundPlaneN (6x57 sub-floor fill) should not appear in minimap")

	fake_player2.queue_free()
	hud2.queue_free()
	overlay3.queue_free()
	hub4.queue_free()

	# --- Results ---
	print("\n========================================")
	print("RESULTS: %d passed, %d failed" % [_pass, _fail])
	if _errors.size() > 0:
		print("FAILURES:")
		for e in _errors:
			print("  x %s" % e)
	print("========================================")
	quit(1 if _fail > 0 else 0)


# =============================================================================
# Helpers
# =============================================================================

func _check(name: String, condition: bool, msg: String = "") -> void:
	if condition:
		_pass += 1
		print("  OK %s" % name)
	else:
		_fail += 1
		var full := "%s: %s" % [name, msg] if msg != "" else name
		_errors.append(full)
		print("  FAIL %s — %s" % [name, msg])


func _find_rect_near(rects: Array, world_x: float, world_z: float, tolerance: float) -> bool:
	## Check if any scanned rect has its center near (world_x, world_z).
	## Supports both plain Rect2 arrays and {rect: Rect2, ...} dict arrays.
	for entry in rects:
		var r: Rect2
		if entry is Rect2:
			r = entry
		else:
			r = entry["rect"]
		var cx: float = r.position.x + r.size.x / 2.0
		var cz: float = r.position.y + r.size.y / 2.0
		if absf(cx - world_x) <= tolerance and absf(cz - world_z) <= tolerance:
			return true
	return false
