extends GdUnitTestSuite

## Verifies the TelegraphManager renders server telegraph descriptors into the
## correct world-space meshes (shape, size, position) and reconciles per tick.

const TM := preload("res://scenes/main/telegraph_manager.gd")

var _mgr: Node
var _container: Node3D


func before_test() -> void:
	_container = Node3D.new()
	_container.name = "Telegraphs"
	add_child(_container)
	_mgr = TM.new()
	add_child(_mgr)  # _ready() wires ctrl + finds the Telegraphs container


func after_test() -> void:
	_mgr.free()
	_container.free()


func _circle(id: int, cx: float, cz: float, r: float) -> Dictionary:
	return {
		"id": id,
		"shape": 0,
		"category": 0,
		"start_tick": 80,
		"execute_tick": 110,
		"cx": cx,
		"cz": cz,
		"radius": r,
	}


func test_circle_size_and_position() -> void:
	_mgr.update_telegraphs([_circle(1000, 3.0, -4.0, 6.5)], 100)
	assert_int(_container.get_child_count()).is_equal(1)
	var mi := _container.get_child(0) as MeshInstance3D
	assert_object(mi).is_not_null()
	assert_vector((mi.mesh as PlaneMesh).size).is_equal(Vector2(13.0, 13.0))
	assert_vector(mi.position).is_equal(Vector3(3.0, TM.FLOOR_Y, -4.0))


func test_multi_circle_draws_a_ring_per_pillar() -> void:
	var multi := {
		"id": 1001,
		"shape": 3,
		"category": 0,
		"start_tick": 7,
		"execute_tick": 27,
		"radius": 9.75,
		"centers": [Vector2(-8, -6), Vector2(8, -6)],
	}
	_mgr.update_telegraphs([multi], 10)
	assert_int(_container.get_child_count()).is_equal(1)  # one grouping node
	var parent := _container.get_child(0)
	assert_int(parent.get_child_count()).is_equal(2)  # one ring per pillar
	var ring0 := parent.get_child(0) as MeshInstance3D
	assert_vector(ring0.position).is_equal(Vector3(-8.0, TM.FLOOR_Y, -6.0))
	assert_vector((ring0.mesh as PlaneMesh).size).is_equal(Vector2(19.5, 19.5))
	var ring1 := parent.get_child(1) as MeshInstance3D
	assert_vector(ring1.position).is_equal(Vector3(8.0, TM.FLOOR_Y, -6.0))


func test_reconcile_frees_absent_telegraphs() -> void:
	_mgr.update_telegraphs([_circle(1000, 0.0, 0.0, 5.0)], 100)
	assert_int(_container.get_child_count()).is_equal(1)
	_mgr.update_telegraphs([], 101)
	await get_tree().process_frame  # queue_free is deferred
	assert_int(_container.get_child_count()).is_equal(0)


func test_fill_progress_driven_from_tick_window() -> void:
	_mgr.update_telegraphs([_circle(1000, 0.0, 0.0, 5.0)], 100)  # start 80, exec 110
	_mgr._process(0.0)
	var mi := _container.get_child(0) as MeshInstance3D
	var fill: float = (mi.material_override as ShaderMaterial).get_shader_parameter("fill")
	# At snapshot tick 100 within [80,110]: ~0.67, clamped to [0,1].
	assert_float(fill).is_between(0.0, 1.0)
	assert_float(fill).is_greater(0.0)
