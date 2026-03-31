class_name TestGunner
extends GdUnitTestSuite

## Tests for the Gunner FPS controller — movement, roll, shooting rules.

const GUNNER_SCENE := "res://scenes/controllers/gunner/gunner.tscn"
const DELTA := 1.0 / 60.0  # 60 fps

var _gunner: CharacterBody3D


func before_test() -> void:
	_gunner = auto_free(load(GUNNER_SCENE).instantiate())
	# Place on a floor so is_on_floor() can work
	_gunner.position = Vector3(0.0, 5.0, 0.0)
	add_child(_gunner)
	# Let the scene tree process one frame so @onready vars resolve
	await get_tree().process_frame


func after_test() -> void:
	# Clean up input state between tests
	for action in ["move_forward", "move_backward", "move_left", "move_right", "sprint", "shoot", "dodge", "jump"]:
		if Input.is_action_pressed(action):
			Input.action_release(action)


# --- Health & Damage ---

func test_initial_health() -> void:
	assert_float(_gunner.health).is_equal(100.0)
	assert_float(_gunner.max_health).is_equal(100.0)


func test_take_damage_reduces_health() -> void:
	_gunner.take_damage(30.0)
	assert_float(_gunner.health).is_equal(70.0)


func test_take_damage_clamps_at_zero() -> void:
	_gunner.take_damage(999.0)
	assert_float(_gunner.health).is_equal(0.0)


func test_respawn_restores_health() -> void:
	_gunner.take_damage(999.0)
	# take_damage calls _respawn at 0 HP
	assert_float(_gunner.health).is_equal(100.0)


# --- Roll mechanics ---

func test_roll_sets_cooldown() -> void:
	_gunner._roll_cooldown_timer = 0.0
	_gunner._start_roll()
	assert_float(_gunner._roll_cooldown_timer).is_equal(_gunner.roll_cooldown)


func test_roll_sets_rolling_state() -> void:
	_gunner._start_roll()
	assert_bool(_gunner._is_rolling).is_true()


func test_roll_timer_set_to_duration() -> void:
	_gunner._start_roll()
	assert_float(_gunner._roll_timer).is_equal(_gunner.roll_duration)


func test_roll_ends_after_duration() -> void:
	_gunner._start_roll()
	# Simulate enough frames to exceed roll duration
	var frames := ceili(_gunner.roll_duration / DELTA) + 2
	for i in frames:
		_gunner._process_roll(DELTA)
	assert_bool(_gunner._is_rolling).is_false()


func test_roll_cooldown_prevents_second_roll() -> void:
	_gunner._start_roll()
	_gunner._is_rolling = false  # force end roll
	# Cooldown is still active
	assert_float(_gunner._roll_cooldown_timer).is_greater(0.0)


func test_roll_default_direction_is_backward() -> void:
	# No movement input → roll backward (local +Z)
	_gunner._start_roll()
	# Gunner faces -Z by default, so backward is +Z
	assert_float(_gunner._roll_direction.z).is_greater(0.0)


func test_roll_bleeds_velocity_on_exit() -> void:
	_gunner._start_roll()
	# Fast-forward to end of roll
	_gunner._roll_timer = DELTA * 0.5
	_gunner._process_roll(DELTA)
	# Velocity should be reduced (40% of roll speed)
	var speed := Vector2(_gunner.velocity.x, _gunner.velocity.z).length()
	assert_float(speed).is_less(_gunner.roll_speed * 0.5)


# --- Sprint / Shooting interaction ---

func test_fire_cooldown_decreases() -> void:
	_gunner._fire_cooldown = 1.0
	_gunner._handle_shooting(DELTA)
	assert_float(_gunner._fire_cooldown).is_less(1.0)


# --- Movement tuning ---

func test_walk_speed_value() -> void:
	assert_float(_gunner.walk_speed).is_equal(5.5)


func test_sprint_speed_value() -> void:
	assert_float(_gunner.sprint_speed).is_equal(7.7)


func test_gravity_is_reduced() -> void:
	# Should be less than default 9.8 for Halo-style arc
	assert_float(_gunner._gravity).is_less(9.8)


func test_air_accel_much_lower_than_ground() -> void:
	assert_float(_gunner.air_accel).is_less(_gunner.ground_accel * 0.2)


# --- Weapon Attachment ---

func test_weapon_scene_path_defined() -> void:
	assert_that(_gunner.WEAPON_SCENE).is_not_null()


func test_model_hidden_for_fps() -> void:
	# CharacterModel children should be invisible in FPS mode
	await get_tree().process_frame
	for child in _gunner.character_model.get_children():
		if child is Node3D:
			assert_bool(child.visible).is_false()
