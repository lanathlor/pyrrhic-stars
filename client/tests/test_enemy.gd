extends GdUnitTestSuite

## Tests for BasicEnemy scene — verifies VFX child has its script and methods.

const ENEMY_SCENE := "res://scenes/enemies/basic_enemy/basic_enemy.tscn"


func test_vfx_node_has_face_health_bar_method() -> void:
	var scene: PackedScene = load(ENEMY_SCENE)
	var enemy: CharacterBody3D = scene.instantiate() as CharacterBody3D
	add_child(enemy)

	var vfx_node = enemy.get_node("BasicEnemyVfx")
	assert_that(vfx_node).is_not_null()
	assert_bool(vfx_node.has_method("face_health_bar_to_camera")).is_true()

	enemy.queue_free()


func test_vfx_node_is_not_plain_node() -> void:
	var scene: PackedScene = load(ENEMY_SCENE)
	var enemy: CharacterBody3D = scene.instantiate() as CharacterBody3D
	add_child(enemy)

	var vfx_node = enemy.get_node("BasicEnemyVfx")
	assert_that(vfx_node).is_not_null()
	# Script must be attached — a plain Node would lack the VFX methods
	assert_that(vfx_node.get_script()).is_not_null()

	enemy.queue_free()
