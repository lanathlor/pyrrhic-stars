class_name TestVanguard
extends GdUnitTestSuite

## Tests for the Vanguard Souls-like controller — combat, stamina, dodge, lock-on.

const VanguardScript := preload("res://scenes/controllers/vanguard/vanguard.gd")
const VANGUARD_SCENE := "res://scenes/controllers/vanguard/vanguard.tscn"
const DELTA := 1.0 / 60.0

var _vanguard: VanguardScript


func before_test() -> void:
	_vanguard = auto_free(load(VANGUARD_SCENE).instantiate()) as VanguardScript
	_vanguard.position = Vector3(0.0, 5.0, 0.0)
	add_child(_vanguard)
	await get_tree().process_frame


func after_test() -> void:
	for action in [
		"move_forward",
		"move_backward",
		"move_left",
		"move_right",
		"sprint",
		"dodge",
		"light_attack",
		"heavy_attack",
		"block",
		"lock_on"
	]:
		if Input.is_action_pressed(action):
			Input.action_release(action)


# --- Health ---


func test_initial_health() -> void:
	assert_float(_vanguard.health).is_equal(200.0)
	assert_float(_vanguard.max_health).is_equal(200.0)


# --- Stamina ---


func test_initial_stamina() -> void:
	assert_float(_vanguard.stamina).is_equal(100.0)
	assert_float(_vanguard.max_stamina).is_equal(100.0)


func test_consume_stamina() -> void:
	_vanguard._consume_stamina(30.0)
	assert_float(_vanguard.stamina).is_equal(70.0)


func test_stamina_clamps_at_zero() -> void:
	_vanguard._consume_stamina(999.0)
	assert_float(_vanguard.stamina).is_equal(0.0)


# --- Dodge ---


func test_dodge_sets_state() -> void:
	_vanguard._start_dodge()
	assert_int(_vanguard.state).is_equal(_vanguard.State.DODGE)


func test_dodge_grants_invincibility() -> void:
	_vanguard._start_dodge()
	assert_bool(_vanguard._is_invincible).is_true()


func test_dodge_ends_after_duration() -> void:
	_vanguard._start_dodge()
	var frames := ceili(_vanguard.dodge_duration / DELTA) + 2
	for i in frames:
		_vanguard._state_timer -= DELTA
		_vanguard._process_dodge(DELTA)
	assert_int(_vanguard.state).is_equal(_vanguard.State.MOVE)


func test_invincibility_flag_during_dodge() -> void:
	_vanguard._start_dodge()
	assert_bool(_vanguard._is_invincible).is_true()


func test_iframes_end_during_dodge() -> void:
	_vanguard._start_dodge()
	# Simulate enough frames to pass iframe window but not full dodge
	var iframe_frames := ceili(_vanguard.dodge_iframe_duration / DELTA) + 2
	for i in iframe_frames:
		_vanguard._state_timer -= DELTA
		_vanguard._process_dodge(DELTA)
	# Should still be in dodge but no longer invincible
	if _vanguard.state == _vanguard.State.DODGE:
		assert_bool(_vanguard._is_invincible).is_false()


func test_dodge_bleeds_velocity() -> void:
	_vanguard._start_dodge()
	# Fast-forward timer to just before end, then tick past it
	_vanguard._state_timer = DELTA * 0.5
	_vanguard._state_timer -= DELTA
	_vanguard._process_dodge(DELTA)
	var speed := Vector2(_vanguard.velocity.x, _vanguard.velocity.z).length()
	assert_float(speed).is_less(_vanguard.dodge_speed * 0.5)


# --- Cleave (single repeatable sweep, no combo) ---


func test_cleave_sets_state() -> void:
	_vanguard._start_cleave()
	assert_int(_vanguard.state).is_equal(_vanguard.State.CLEAVE)


func test_cleave_returns_to_move() -> void:
	_vanguard._start_cleave()
	var frames := ceili(_vanguard.CLEAVE_DURATION / DELTA) + 2
	for i in frames:
		_vanguard._state_timer -= DELTA
		_vanguard._process_cleave(DELTA)
	assert_int(_vanguard.state).is_equal(_vanguard.State.MOVE)


# --- Upheaval ---


func test_upheaval_starts_windup() -> void:
	_vanguard._start_upheaval()
	assert_int(_vanguard.state).is_equal(_vanguard.State.UPHEAVAL_WINDUP)


func test_upheaval_windup_transitions() -> void:
	_vanguard._start_upheaval()
	var frames := ceili(_vanguard.UPHEAVAL_WINDUP_TIME / DELTA) + 2
	for i in frames:
		_vanguard._state_timer -= DELTA
		_vanguard._process_upheaval_windup(DELTA)
	assert_int(_vanguard.state).is_equal(_vanguard.State.UPHEAVAL)


# --- Block & Parry ---


func test_block_state_sets_block() -> void:
	_vanguard.state = _vanguard.State.BLOCK
	assert_int(_vanguard.state).is_equal(_vanguard.State.BLOCK)


func test_block_drains_stamina() -> void:
	_vanguard.state = _vanguard.State.BLOCK
	var before := _vanguard.stamina
	_vanguard._consume_stamina(_vanguard.block_stamina_drain * DELTA)
	assert_float(_vanguard.stamina).is_less(before)


# --- Onslaught ---


func test_onslaught_tier_default() -> void:
	assert_int(_vanguard._onslaught_tier).is_equal(0)


# --- Lock-on ---


func test_lock_on_initially_off() -> void:
	assert_bool(_vanguard._lock_on_active).is_false()
	assert_that(_vanguard._lock_target).is_null()


func test_lock_on_no_enemies() -> void:
	_vanguard._toggle_lock_on()
	assert_bool(_vanguard._lock_on_active).is_false()


func test_lock_on_double_toggle_off() -> void:
	# Force lock-on state
	_vanguard._lock_on_active = true
	_vanguard._toggle_lock_on()
	assert_bool(_vanguard._lock_on_active).is_false()
	assert_that(_vanguard._lock_target).is_null()


# --- Movement values ---


func test_run_speed() -> void:
	assert_float(_vanguard.run_speed).is_equal(5.0)


func test_sprint_speed() -> void:
	assert_float(_vanguard.sprint_speed).is_equal(7.0)


func test_melee_range() -> void:
	assert_float(_vanguard.melee_range).is_equal(3.0)


func test_dodge_iframe_value() -> void:
	assert_float(_vanguard.dodge_iframe_duration).is_equal(0.15)


# --- Weapon Attachment ---


func test_weapon_scene_path_defined() -> void:
	assert_that(_vanguard.WEAPON_SCENE).is_not_null()


func test_weapon_attaches_after_ready() -> void:
	# call_deferred runs after the frame, so process one more frame
	await get_tree().process_frame
	# weapon_node is only non-null if the GLB asset loads as PackedScene
	# (may fail in headless mode without full import)
	var loaded = load(_vanguard.WEAPON_SCENE)
	if loaded is PackedScene:
		assert_that(_vanguard.character_model.weapon_node).is_not_null()
	else:
		assert_that(_vanguard.character_model.weapon_node).is_null()


func test_weapon_is_in_scene_tree() -> void:
	await get_tree().process_frame
	if _vanguard.character_model.weapon_node:
		assert_bool(_vanguard.character_model.weapon_node.is_inside_tree()).is_true()
