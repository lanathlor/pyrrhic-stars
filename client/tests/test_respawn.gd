class_name TestRespawn
extends GdUnitTestSuite

## Tests for respawn position snapping in apply_server_state() across all controllers.
## When NetworkManager.is_active is false, _is_local() returns true for all controllers,
## so the local player path (health tracking, death, respawn snap) is exercised directly.

const GUNNER_SCENE := "res://scenes/controllers/gunner/gunner.tscn"
const VANGUARD_SCENE := "res://scenes/controllers/vanguard/vanguard.tscn"
const BD_SCENE := "res://scenes/controllers/blade_dancer/blade_dancer.tscn"
const DELTA := 1.0 / 60.0

var _gunner: CharacterBody3D
var _vanguard: CharacterBody3D
var _bd: CharacterBody3D


func before_test() -> void:
	_gunner = auto_free(load(GUNNER_SCENE).instantiate())
	_gunner.position = Vector3(5.0, 1.0, 5.0)
	add_child(_gunner)

	_vanguard = auto_free(load(VANGUARD_SCENE).instantiate())
	_vanguard.position = Vector3(5.0, 1.0, 5.0)
	add_child(_vanguard)

	_bd = auto_free(load(BD_SCENE).instantiate())
	_bd.position = Vector3(5.0, 1.0, 5.0)
	add_child(_bd)

	await get_tree().process_frame


func _make_state(hp: float, pos: Vector3) -> Dictionary:
	return {
		"health": hp,
		"pos": pos,
		"rot_y": 0.0,
		"visual_state": 0,
	}


# =============================================================================
# Gunner — death via server state
# =============================================================================


func test_gunner_death_via_server_state() -> void:
	var result := [false]
	_gunner.died.connect(func(): result[0] = true)
	# Ensure alive first
	_gunner.apply_server_state(_make_state(100.0, Vector3(5.0, 1.0, 5.0)))
	assert_bool(_gunner._alive).is_true()
	# Kill via server state
	_gunner.apply_server_state(_make_state(0.0, Vector3(5.0, 0.0, 5.0)))
	assert_bool(_gunner._alive).is_false()
	assert_bool(result[0]).is_true()


func test_gunner_respawn_snaps_position() -> void:
	# Kill the player
	_gunner.apply_server_state(_make_state(100.0, Vector3(5.0, 1.0, 5.0)))
	_gunner.apply_server_state(_make_state(0.0, Vector3(5.0, 0.0, 5.0)))
	assert_bool(_gunner._alive).is_false()
	# Respawn at warmup position
	var respawn_pos := Vector3(0.0, 0.1, 20.0)
	_gunner.apply_server_state(_make_state(100.0, respawn_pos))
	assert_bool(_gunner._alive).is_true()
	assert_vector(_gunner.global_position).is_equal(respawn_pos)


func test_gunner_respawn_restores_alive() -> void:
	_gunner.apply_server_state(_make_state(100.0, Vector3(5.0, 1.0, 5.0)))
	_gunner.apply_server_state(_make_state(0.0, Vector3(5.0, 0.0, 5.0)))
	assert_bool(_gunner._alive).is_false()
	_gunner.apply_server_state(_make_state(100.0, Vector3(0.0, 0.1, 20.0)))
	assert_bool(_gunner._alive).is_true()
	assert_float(_gunner.health).is_equal(100.0)


# =============================================================================
# Vanguard — death via server state
# =============================================================================


func test_vanguard_death_via_server_state() -> void:
	var result := [false]
	_vanguard.died.connect(func(): result[0] = true)
	_vanguard.apply_server_state(_make_state(150.0, Vector3(5.0, 1.0, 5.0)))
	assert_bool(_vanguard._alive).is_true()
	_vanguard.apply_server_state(_make_state(0.0, Vector3(5.0, 0.0, 5.0)))
	assert_bool(_vanguard._alive).is_false()
	assert_bool(result[0]).is_true()
	assert_int(_vanguard.state).is_equal(_vanguard.State.DEAD)


func test_vanguard_respawn_snaps_position() -> void:
	_vanguard.apply_server_state(_make_state(150.0, Vector3(5.0, 1.0, 5.0)))
	_vanguard.apply_server_state(_make_state(0.0, Vector3(5.0, 0.0, 5.0)))
	assert_bool(_vanguard._alive).is_false()
	var respawn_pos := Vector3(0.0, 0.1, 20.0)
	_vanguard.apply_server_state(_make_state(150.0, respawn_pos))
	assert_bool(_vanguard._alive).is_true()
	assert_vector(_vanguard.global_position).is_equal(respawn_pos)


func test_vanguard_respawn_restores_move_state() -> void:
	_vanguard.apply_server_state(_make_state(150.0, Vector3(5.0, 1.0, 5.0)))
	_vanguard.apply_server_state(_make_state(0.0, Vector3(5.0, 0.0, 5.0)))
	assert_int(_vanguard.state).is_equal(_vanguard.State.DEAD)
	_vanguard.apply_server_state(_make_state(150.0, Vector3(0.0, 0.1, 20.0)))
	assert_bool(_vanguard._alive).is_true()
	assert_int(_vanguard.state).is_equal(_vanguard.State.MOVE)


# =============================================================================
# Blade Dancer — death via server state
# =============================================================================


func test_blade_dancer_death_via_server_state() -> void:
	var result := [false]
	_bd.died.connect(func(): result[0] = true)
	_bd.apply_server_state(_make_state(100.0, Vector3(5.0, 1.0, 5.0)))
	assert_bool(_bd._alive).is_true()
	_bd.apply_server_state(_make_state(0.0, Vector3(5.0, 0.0, 5.0)))
	assert_bool(_bd._alive).is_false()
	assert_bool(result[0]).is_true()
	assert_int(_bd.state).is_equal(_bd.State.DEAD)


func test_blade_dancer_respawn_snaps_position() -> void:
	_bd.apply_server_state(_make_state(100.0, Vector3(5.0, 1.0, 5.0)))
	_bd.apply_server_state(_make_state(0.0, Vector3(5.0, 0.0, 5.0)))
	assert_bool(_bd._alive).is_false()
	var respawn_pos := Vector3(0.0, 0.1, 20.0)
	_bd.apply_server_state(_make_state(100.0, respawn_pos))
	assert_bool(_bd._alive).is_true()
	assert_vector(_bd.global_position).is_equal(respawn_pos)


func test_blade_dancer_respawn_restores_move_state() -> void:
	_bd.apply_server_state(_make_state(100.0, Vector3(5.0, 1.0, 5.0)))
	_bd.apply_server_state(_make_state(0.0, Vector3(5.0, 0.0, 5.0)))
	assert_int(_bd.state).is_equal(_bd.State.DEAD)
	_bd.apply_server_state(_make_state(100.0, Vector3(0.0, 0.1, 20.0)))
	assert_bool(_bd._alive).is_true()
	assert_int(_bd.state).is_equal(_bd.State.MOVE)


# =============================================================================
# Cross-controller: net_position also snapped
# =============================================================================


func test_gunner_respawn_updates_net_position() -> void:
	_gunner.apply_server_state(_make_state(100.0, Vector3(5.0, 1.0, 5.0)))
	_gunner.apply_server_state(_make_state(0.0, Vector3(5.0, 0.0, 5.0)))
	var respawn_pos := Vector3(0.0, 0.1, 20.0)
	_gunner.apply_server_state(_make_state(100.0, respawn_pos))
	assert_vector(_gunner._net_position).is_equal(respawn_pos)


func test_vanguard_respawn_updates_net_position() -> void:
	_vanguard.apply_server_state(_make_state(150.0, Vector3(5.0, 1.0, 5.0)))
	_vanguard.apply_server_state(_make_state(0.0, Vector3(5.0, 0.0, 5.0)))
	var respawn_pos := Vector3(0.0, 0.1, 20.0)
	_vanguard.apply_server_state(_make_state(150.0, respawn_pos))
	assert_vector(_vanguard._net_position).is_equal(respawn_pos)


func test_blade_dancer_respawn_updates_net_position() -> void:
	_bd.apply_server_state(_make_state(100.0, Vector3(5.0, 1.0, 5.0)))
	_bd.apply_server_state(_make_state(0.0, Vector3(5.0, 0.0, 5.0)))
	var respawn_pos := Vector3(0.0, 0.1, 20.0)
	_bd.apply_server_state(_make_state(100.0, respawn_pos))
	assert_vector(_bd._net_position).is_equal(respawn_pos)


# =============================================================================
# Edge case: already alive player does not snap position
# =============================================================================


func test_gunner_alive_no_position_snap() -> void:
	# Start alive at a known position
	_gunner.global_position = Vector3(5.0, 1.0, 5.0)
	_gunner.apply_server_state(_make_state(80.0, Vector3(10.0, 0.0, 10.0)))
	# Local player only updates health; position is NOT snapped when already alive
	assert_float(_gunner.health).is_equal(80.0)
	assert_vector(_gunner.global_position).is_not_equal(Vector3(10.0, 0.0, 10.0))


func test_vanguard_alive_no_position_snap() -> void:
	_vanguard.global_position = Vector3(5.0, 1.0, 5.0)
	_vanguard.apply_server_state(_make_state(120.0, Vector3(10.0, 0.0, 10.0)))
	assert_float(_vanguard.health).is_equal(120.0)
	assert_vector(_vanguard.global_position).is_not_equal(Vector3(10.0, 0.0, 10.0))


func test_blade_dancer_alive_no_position_snap() -> void:
	_bd.global_position = Vector3(5.0, 1.0, 5.0)
	_bd.apply_server_state(_make_state(80.0, Vector3(10.0, 0.0, 10.0)))
	assert_float(_bd.health).is_equal(80.0)
	assert_vector(_bd.global_position).is_not_equal(Vector3(10.0, 0.0, 10.0))
