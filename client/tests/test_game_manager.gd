class_name TestGameManager
extends GdUnitTestSuite

## Tests for the GameManager autoload — player/enemy registry and queries.

var _manager: Node


func before_test() -> void:
	# Use a fresh instance instead of the autoload to avoid cross-test pollution
	_manager = load("res://scripts/autoload/game_manager.gd").new()
	add_child(_manager)


func after_test() -> void:
	_manager.queue_free()


# --- Registration ---


func test_register_player() -> void:
	var player := auto_free(CharacterBody3D.new())
	_manager.register_player(player)
	assert_array(_manager.players).has_size(1)
	assert_array(_manager.players).contains([player])


func test_unregister_player() -> void:
	var player := auto_free(CharacterBody3D.new())
	_manager.register_player(player)
	_manager.unregister_player(player)
	assert_array(_manager.players).is_empty()


func test_register_enemy() -> void:
	var enemy := auto_free(CharacterBody3D.new())
	_manager.register_enemy(enemy)
	assert_array(_manager.enemies).has_size(1)


func test_unregister_enemy() -> void:
	var enemy := auto_free(CharacterBody3D.new())
	_manager.register_enemy(enemy)
	_manager.unregister_enemy(enemy)
	assert_array(_manager.enemies).is_empty()


# --- Nearest / Farthest queries ---


func test_nearest_player_single() -> void:
	var player := auto_free(CharacterBody3D.new())
	player.position = Vector3(5.0, 0.0, 0.0)
	_manager.register_player(player)

	var nearest := _manager.get_nearest_player(Vector3.ZERO)
	assert_that(nearest).is_same(player)


func test_nearest_player_multiple() -> void:
	var close := auto_free(CharacterBody3D.new())
	close.position = Vector3(2.0, 0.0, 0.0)
	var far := auto_free(CharacterBody3D.new())
	far.position = Vector3(10.0, 0.0, 0.0)

	_manager.register_player(far)
	_manager.register_player(close)

	var nearest := _manager.get_nearest_player(Vector3.ZERO)
	assert_that(nearest).is_same(close)


func test_farthest_player_multiple() -> void:
	var close := auto_free(CharacterBody3D.new())
	close.position = Vector3(2.0, 0.0, 0.0)
	var far := auto_free(CharacterBody3D.new())
	far.position = Vector3(10.0, 0.0, 0.0)

	_manager.register_player(close)
	_manager.register_player(far)

	var farthest := _manager.get_farthest_player(Vector3.ZERO)
	assert_that(farthest).is_same(far)


func test_nearest_player_no_players() -> void:
	var nearest := _manager.get_nearest_player(Vector3.ZERO)
	assert_that(nearest).is_null()


func test_farthest_player_no_players() -> void:
	var farthest := _manager.get_farthest_player(Vector3.ZERO)
	assert_that(farthest).is_null()


func test_queries_ignore_freed_players() -> void:
	var valid := auto_free(CharacterBody3D.new())
	valid.position = Vector3(5.0, 0.0, 0.0)
	var doomed := CharacterBody3D.new()
	doomed.position = Vector3(1.0, 0.0, 0.0)

	_manager.register_player(doomed)
	_manager.register_player(valid)
	doomed.free()

	var nearest := _manager.get_nearest_player(Vector3.ZERO)
	assert_that(nearest).is_same(valid)
