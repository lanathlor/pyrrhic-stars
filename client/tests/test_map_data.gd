class_name TestMapData
extends GdUnitTestSuite

## Tests for MapData geometry constants, shared_hud minimap enhancements,
## and map_overlay — covering floor detection, geometry loading, waypoints,
## NPC parsing, toggle behavior, and coordinate transforms.

# =============================================================================
# MapData — Floor definitions
# =============================================================================


func test_floors_array_has_four_entries() -> void:
	assert_int(MapData.FLOORS.size()).is_equal(4)


func test_floors_have_required_fields() -> void:
	for f in MapData.FLOORS:
		for key in ["id", "name", "target", "y_min", "y_max", "arrival_radius"]:
			assert_bool(f.has(key)).is_true()


func test_floors_have_unique_ids() -> void:
	var ids: Array[String] = []
	for f in MapData.FLOORS:
		var fid: String = f["id"]
		assert_bool(fid in ids).is_false()
		ids.append(fid)


# =============================================================================
# MapData — get_floor_for_position
# =============================================================================


func test_get_floor_lower_district() -> void:
	var f := MapData.get_floor_for_position(Vector3(5.0, -199.0, -55.0))
	assert_str(f.get("id", "")).is_equal("lower_district")


func test_get_floor_plaza() -> void:
	# Outside tower bounds -> plaza
	var f := MapData.get_floor_for_position(Vector3(50.0, 0.0, -30.0))
	assert_str(f.get("id", "")).is_equal("plaza")


func test_get_floor_tower_lobby() -> void:
	# Inside tower bounds X[-24,24] Z[-1,43]
	var f := MapData.get_floor_for_position(Vector3(0.0, 0.0, 20.0))
	assert_str(f.get("id", "")).is_equal("tower_lobby")


func test_get_floor_ops() -> void:
	var f := MapData.get_floor_for_position(Vector3(10.0, 100.0, 20.0))
	assert_str(f.get("id", "")).is_equal("ops")


func test_get_floor_unknown_y_returns_empty() -> void:
	var f := MapData.get_floor_for_position(Vector3(0.0, 50.0, 0.0))
	assert_dict(f).is_empty()


func test_tower_lobby_bounds_excludes_x25() -> void:
	# X=25 is outside tower bounds
	var f := MapData.get_floor_for_position(Vector3(25.0, 0.0, 20.0))
	assert_str(f.get("id", "")).is_not_equal("tower_lobby")


# =============================================================================
# MapData — get_geometry_for_floor
# =============================================================================


func test_get_geometry_lower_district() -> void:
	var geo := MapData.get_geometry_for_floor("lower_district")
	assert_bool(geo.has("buildings")).is_true()


func test_get_geometry_plaza() -> void:
	var geo := MapData.get_geometry_for_floor("plaza")
	assert_bool(geo.has("buildings")).is_true()


func test_get_geometry_ops() -> void:
	var geo := MapData.get_geometry_for_floor("ops")
	assert_bool(geo.has("buildings")).is_true()


func test_get_geometry_arena() -> void:
	var geo := MapData.get_geometry_for_floor("arena")
	assert_bool(geo.has("buildings")).is_true()


func test_tower_lobby_returns_plaza_geometry() -> void:
	var geo_lobby := MapData.get_geometry_for_floor("tower_lobby")
	var geo_plaza := MapData.get_geometry_for_floor("plaza")
	assert_that(geo_lobby.get("center")).is_equal(geo_plaza.get("center"))


func test_unknown_floor_returns_empty() -> void:
	var geo := MapData.get_geometry_for_floor("nonexistent")
	assert_dict(geo).is_empty()


# =============================================================================
# MapData — Building counts
# =============================================================================


func test_lower_district_has_17_buildings() -> void:
	var buildings: Array = MapData.LOWER_DISTRICT["buildings"]
	assert_int(buildings.size()).is_equal(17)


func test_arena_has_17_elements() -> void:
	var buildings: Array = MapData.ARENA["buildings"]
	assert_int(buildings.size()).is_equal(17)


func test_ops_has_extra_floors() -> void:
	var extras: Array = MapData.OPS.get("extra_floors", [])
	assert_int(extras.size()).is_equal(1)


# =============================================================================
# SharedHUD — hub/arena mode transitions
# =============================================================================

var _hud: Control


func _make_hud() -> void:
	_hud = auto_free(Control.new())
	_hud.set_script(load("res://scenes/shared/hud/shared_hud.gd"))
	add_child(_hud)


func test_initial_hub_mode_false() -> void:
	_make_hud()
	assert_bool(_hud._hub_mode).is_false()


func test_on_enter_hub_sets_hub_mode() -> void:
	_make_hud()
	_hud.on_enter_hub()
	assert_bool(_hud._hub_mode).is_true()


func test_on_enter_arena_clears_hub_mode() -> void:
	_make_hud()
	_hud.on_enter_hub()
	_hud.on_enter_arena()
	assert_bool(_hud._hub_mode).is_false()


func test_on_enter_arena_sets_floor_id() -> void:
	_make_hud()
	_hud.on_enter_arena()
	assert_str(_hud._current_floor_id).is_equal("arena")


func test_on_enter_hub_resets_floor_id() -> void:
	_make_hud()
	_hud.on_enter_arena()
	_hud.on_enter_hub()
	assert_str(_hud._current_floor_id).is_equal("")


# =============================================================================
# SharedHUD — NPC positions from world state
# =============================================================================


func test_npc_positions_parsed() -> void:
	_make_hud()
	(
		_hud
		. update_world_state(
			{
				"players": [],
				"enemies": [],
				"npcs":
				[
					{
						"npc_id": 1,
						"pos": Vector3(10.0, 0.0, 20.0),
						"rot_y": 0.0,
						"anim_name": "idle"
					},
					{
						"npc_id": 2,
						"pos": Vector3(30.0, 0.0, 40.0),
						"rot_y": 1.0,
						"anim_name": "walk"
					},
				],
			}
		)
	)
	assert_int(_hud._npc_positions.size()).is_equal(2)


func test_npc_positions_cleared_on_update() -> void:
	_make_hud()
	(
		_hud
		. update_world_state(
			{
				"players": [],
				"enemies": [],
				"npcs": [{"npc_id": 1, "pos": Vector3.ZERO, "rot_y": 0.0, "anim_name": "idle"}],
			}
		)
	)
	_hud.update_world_state({"players": [], "enemies": [], "npcs": []})
	assert_int(_hud._npc_positions.size()).is_equal(0)


# =============================================================================
# SharedHUD — Floor detection and geometry
# =============================================================================


func test_detect_floor_lower_district() -> void:
	_make_hud()
	_hud._hub_mode = true
	_hud._detect_floor(Vector3(5.0, -199.0, -55.0))
	assert_str(_hud._current_floor_id).is_equal("lower_district")


func test_detect_floor_ops() -> void:
	_make_hud()
	_hud._hub_mode = true
	_hud._detect_floor(Vector3(10.0, 100.0, 20.0))
	assert_str(_hud._current_floor_id).is_equal("ops")


func test_detect_floor_sets_waypoint() -> void:
	_make_hud()
	_hud._hub_mode = true
	_hud._detect_floor(Vector3(5.0, -199.0, -55.0))
	assert_bool(_hud._has_waypoint).is_true()
	assert_that(_hud._waypoint_target).is_equal(Vector3(5.0, -199.8, -55.0))


func test_floor_geometry_fallback_without_env() -> void:
	_make_hud()
	_hud._hub_mode = true
	# No environment set — should fall back to MapData
	_hud._detect_floor(Vector3(5.0, -199.0, -55.0))
	assert_int(_hud._floor_rects.size()).is_equal(17)


func test_floor_geometry_fallback_first_rect_correct() -> void:
	_make_hud()
	_hud._hub_mode = true
	_hud._detect_floor(Vector3(5.0, -199.0, -55.0))
	# A1: center=(-65,-125), size=(50,45) -> Rect2(-90, -147.5, 50, 45)
	var r: Rect2 = _hud._floor_rects[0]
	assert_float(r.position.x).is_equal_approx(-90.0, 0.1)
	assert_float(r.position.y).is_equal_approx(-147.5, 0.1)
	assert_float(r.size.x).is_equal_approx(50.0, 0.1)
	assert_float(r.size.y).is_equal_approx(45.0, 0.1)


# =============================================================================
# MapOverlay — basic state
# =============================================================================

var _overlay: Control


func _make_overlay() -> void:
	_overlay = auto_free(Control.new())
	_overlay.set_script(load("res://scenes/shared/hud/map_overlay.gd"))
	_overlay.size = Vector2(1920, 1080)
	_overlay.visible = false
	add_child(_overlay)


func test_overlay_initial_state() -> void:
	_make_overlay()
	assert_str(_overlay._current_floor_id).is_equal("")
	assert_bool(_overlay.visible).is_false()


func test_overlay_toggle_on_off() -> void:
	_make_overlay()
	_overlay._player_pos = Vector3(5.0, -199.0, -55.0)
	_overlay.toggle()
	assert_bool(_overlay.visible).is_true()
	_overlay.toggle()
	assert_bool(_overlay.visible).is_false()


func test_overlay_set_floor_arena() -> void:
	_make_overlay()
	_overlay.set_floor("arena", "Arena")
	assert_str(_overlay._current_floor_id).is_equal("arena")
	assert_str(_overlay._floor_name).is_equal("Arena")


func test_overlay_set_floor_sets_name() -> void:
	_make_overlay()
	_overlay.set_floor("lower_district", "Lower District")
	assert_str(_overlay._floor_name).is_equal("Lower District")


func test_overlay_reset_floor() -> void:
	_make_overlay()
	_overlay.set_floor("arena", "Arena")
	_overlay.reset_floor()
	assert_str(_overlay._current_floor_id).is_equal("")


# =============================================================================
# MapOverlay — waypoints
# =============================================================================


func test_overlay_arena_no_waypoint() -> void:
	_make_overlay()
	_overlay.set_floor("arena", "Arena")
	assert_bool(_overlay._has_waypoint).is_false()


func test_overlay_hub_floor_has_waypoint() -> void:
	_make_overlay()
	_overlay._player_pos = Vector3(5.0, -199.0, -55.0)
	_overlay.set_floor("lower_district", "Lower District")
	assert_bool(_overlay._has_waypoint).is_true()


func test_overlay_set_waypoint_path() -> void:
	_make_overlay()
	var path := PackedVector3Array(
		[Vector3.ZERO, Vector3(10.0, 0.0, 10.0), Vector3(20.0, 0.0, 0.0)]
	)
	_overlay.set_waypoint_path(path)
	assert_int(_overlay._waypoint_path.size()).is_equal(3)


# =============================================================================
# MapOverlay — state updates
# =============================================================================


func test_overlay_update_state() -> void:
	_make_overlay()
	(
		_overlay
		. update_state(
			{
				"player_pos": Vector3(10.0, 0.0, 20.0),
				"player_rot_y": 1.5,
				"players": {1: {"pos": Vector3.ZERO, "health": 100.0, "rot_y": 0.0}},
				"npcs": [Vector3(5.0, 0.0, 5.0)],
				"enemies": [Vector3(20.0, 0.0, 20.0)],
			}
		)
	)
	assert_that(_overlay._player_pos).is_equal(Vector3(10.0, 0.0, 20.0))
	assert_float(_overlay._player_rot_y).is_equal_approx(1.5, 0.01)
	assert_int(_overlay._npc_positions.size()).is_equal(1)
	assert_int(_overlay._enemy_positions.size()).is_equal(1)


# =============================================================================
# MapOverlay — coordinate transforms
# =============================================================================


func test_overlay_world_to_screen_center() -> void:
	_make_overlay()
	# Manually set floor geometry as scan_environment would
	_overlay._floor_center = Vector2(5.0, -55.0)
	_overlay._floor_size = Vector2(200.0, 200.0)
	_overlay.visible = true
	_overlay._recompute_scale()
	# Floor center (5, -55) should map to screen center
	var center_screen: Vector2 = _overlay._world_to_screen(Vector3(5.0, 0.0, -55.0))
	var expected: Vector2 = _overlay.size / 2.0
	assert_float(center_screen.distance_to(expected)).is_less(1.0)


func test_overlay_scale_computation() -> void:
	_make_overlay()
	_overlay._floor_center = Vector2(5.0, -55.0)
	_overlay._floor_size = Vector2(200.0, 200.0)
	_overlay.visible = true
	_overlay._recompute_scale()
	# 200x200 floor, 1920x1080 viewport, 70% usable
	# scale = min(1920*0.7/200, 1080*0.7/200) = min(6.72, 3.78) = 3.78
	var expected_scale := 1080.0 * 0.7 / 200.0
	assert_float(_overlay._map_scale).is_equal_approx(expected_scale, 0.01)
