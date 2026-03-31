class_name TestBladeDancer
extends GdUnitTestSuite

## Tests for the Blade Dancer state machine controller — config transitions,
## blade formations, abilities, dash, and guard.

const BD_SCENE := "res://scenes/controllers/blade_dancer/blade_dancer.tscn"
const DELTA := 1.0 / 60.0

var _bd: CharacterBody3D


func before_test() -> void:
	_bd = auto_free(load(BD_SCENE).instantiate())
	_bd.position = Vector3(0.0, 5.0, 0.0)
	add_child(_bd)
	await get_tree().process_frame


func after_test() -> void:
	for action in ["move_forward", "move_backward", "move_left", "move_right",
			"sprint", "dodge", "light_attack", "heavy_attack", "block", "lock_on", "jump"]:
		if Input.is_action_pressed(action):
			Input.action_release(action)


# --- Health ---

func test_initial_health() -> void:
	assert_float(_bd.health).is_equal(100.0)
	assert_float(_bd.max_health).is_equal(150.0)


func test_take_damage_reduces_health() -> void:
	_bd.take_damage(40.0)
	assert_float(_bd.health).is_equal(60.0)


func test_take_damage_clamps_at_zero() -> void:
	_bd.take_damage(999.0)
	assert_float(_bd.health).is_equal(0.0)


func test_death_emits_signal() -> void:
	var died_emitted := false
	_bd.died.connect(func(): died_emitted = true)
	_bd.take_damage(999.0)
	assert_bool(died_emitted).is_true()


func test_dead_state_on_death() -> void:
	_bd.take_damage(999.0)
	assert_int(_bd.state).is_equal(_bd.State.DEAD)


func test_invincible_during_dash_blocks_damage() -> void:
	_bd._is_invincible = true
	_bd.take_damage(50.0)
	assert_float(_bd.health).is_equal(100.0)


# --- Config ---

func test_initial_config_is_orbit() -> void:
	assert_int(_bd.config).is_equal(_bd.Config.ORBIT)


func test_config_transition_orbit_to_lance() -> void:
	_bd._config_at_cast = _bd.Config.ORBIT
	_bd._transition_config_for_ability()
	assert_int(_bd.config).is_equal(_bd.Config.LANCE)


func test_config_transition_lance_to_orbit() -> void:
	_bd.config = _bd.Config.LANCE
	_bd._config_at_cast = _bd.Config.LANCE
	_bd._transition_config_for_ability()
	assert_int(_bd.config).is_equal(_bd.Config.ORBIT)


func test_config_round_trip() -> void:
	# Orbit -> Lance -> Orbit
	_bd._config_at_cast = _bd.Config.ORBIT
	_bd._transition_config_for_ability()
	assert_int(_bd.config).is_equal(_bd.Config.LANCE)
	_bd._config_at_cast = _bd.Config.LANCE
	_bd._transition_config_for_ability()
	assert_int(_bd.config).is_equal(_bd.Config.ORBIT)


# --- GCD ---

func test_initial_gcd_is_zero() -> void:
	assert_float(_bd._gcd_timer).is_less_equal(0.0)


func test_edge_triggers_gcd() -> void:
	_bd._start_edge()
	assert_float(_bd._gcd_timer).is_equal(_bd.gcd_duration)


func test_surge_triggers_gcd() -> void:
	_bd._start_surge()
	assert_float(_bd._gcd_timer).is_equal(_bd.gcd_duration)


func test_guard_triggers_gcd() -> void:
	_bd._start_guard()
	assert_float(_bd._gcd_timer).is_equal(_bd.gcd_duration)


# --- Edge (Ability 1) ---

func test_edge_enters_edge_state() -> void:
	_bd._start_edge()
	assert_int(_bd.state).is_equal(_bd.State.EDGE)


func test_edge_stores_config_at_cast() -> void:
	_bd.config = _bd.Config.LANCE
	_bd._start_edge()
	assert_int(_bd._config_at_cast).is_equal(_bd.Config.LANCE)


func test_edge_completes_to_move() -> void:
	_bd._start_edge()
	# Run past edge duration (0.3s)
	var frames := ceili(0.35 / DELTA)
	for i in frames:
		_bd._process_edge(DELTA)
	assert_int(_bd.state).is_equal(_bd.State.MOVE)


# --- Surge (Ability 2) ---

func test_orbit_surge_skips_windup() -> void:
	_bd.config = _bd.Config.ORBIT
	_bd._start_surge()
	assert_int(_bd.state).is_equal(_bd.State.SURGE)


func test_lance_surge_has_windup() -> void:
	_bd.config = _bd.Config.LANCE
	_bd._start_surge()
	assert_int(_bd.state).is_equal(_bd.State.SURGE_WINDUP)


func test_lance_surge_windup_transitions_to_surge() -> void:
	_bd.config = _bd.Config.LANCE
	_bd._start_surge()
	var frames := ceili((_bd.lance_surge_windup + 0.05) / DELTA)
	for i in frames:
		_bd._process_surge_windup(DELTA)
	assert_int(_bd.state).is_equal(_bd.State.SURGE)


# --- Guard (Ability 3) ---

func test_orbit_guard_activates_guard() -> void:
	_bd.config = _bd.Config.ORBIT
	_bd._start_guard()
	assert_int(_bd.state).is_equal(_bd.State.GUARD)
	assert_bool(_bd._guard_active).is_true()


func test_orbit_guard_reduces_damage() -> void:
	_bd.config = _bd.Config.ORBIT
	_bd._start_guard()
	_bd.take_damage(100.0)
	# 50% reduction: 100 * 0.5 = 50 damage taken
	assert_float(_bd.health).is_equal(50.0)


func test_orbit_guard_stays_in_orbit() -> void:
	_bd.config = _bd.Config.ORBIT
	_bd._start_guard()
	# Guard doesn't transition config
	assert_int(_bd.config).is_equal(_bd.Config.ORBIT)


func test_lance_guard_becomes_recall() -> void:
	_bd.config = _bd.Config.LANCE
	_bd._start_guard()
	assert_int(_bd.state).is_equal(_bd.State.RECALL)


func test_recall_transitions_to_orbit() -> void:
	_bd.config = _bd.Config.LANCE
	_bd._start_guard()
	# Run past recall duration
	var frames := ceili(0.35 / DELTA)
	for i in frames:
		_bd._process_recall(DELTA)
	assert_int(_bd.config).is_equal(_bd.Config.ORBIT)


# --- Dash ---

func test_orbit_dash_enters_dash_state() -> void:
	_bd.config = _bd.Config.ORBIT
	_bd._start_dash()
	assert_int(_bd.state).is_equal(_bd.State.DASH)


func test_dash_grants_invincibility() -> void:
	_bd._start_dash()
	assert_bool(_bd._is_invincible).is_true()


func test_dash_invincibility_expires() -> void:
	_bd._start_dash()
	var frames := ceili((_bd.dash_iframe_duration + 0.02) / DELTA)
	for i in frames:
		_bd._process_dash(DELTA)
	assert_bool(_bd._is_invincible).is_false()


func test_dash_completes_to_move() -> void:
	_bd._start_dash()
	var frames := ceili((_bd.dash_duration + 0.05) / DELTA)
	for i in frames:
		_bd._process_dash(DELTA)
	assert_int(_bd.state).is_equal(_bd.State.MOVE)


func test_dash_transitions_config() -> void:
	_bd.config = _bd.Config.ORBIT
	_bd._start_dash()
	var frames := ceili((_bd.dash_duration + 0.05) / DELTA)
	for i in frames:
		_bd._process_dash(DELTA)
	assert_int(_bd.config).is_equal(_bd.Config.LANCE)


func test_dash_bleeds_velocity() -> void:
	_bd._start_dash()
	var frames := ceili((_bd.dash_duration + 0.05) / DELTA)
	for i in frames:
		_bd._process_dash(DELTA)
	# After dash, velocity should be reduced (multiplied by 0.3)
	var flat_speed := Vector2(_bd.velocity.x, _bd.velocity.z).length()
	assert_float(flat_speed).is_less(_bd.dash_speed * 0.5)


# --- Stagger ---

func test_damage_causes_stagger_in_move() -> void:
	_bd.state = _bd.State.MOVE
	_bd.take_damage(10.0)
	assert_int(_bd.state).is_equal(_bd.State.STAGGER)


func test_no_stagger_during_dash() -> void:
	_bd._start_dash()
	_bd._is_invincible = false  # disable iframes to test stagger immunity
	_bd.take_damage(10.0)
	assert_int(_bd.state).is_equal(_bd.State.DASH)


func test_no_stagger_during_guard() -> void:
	_bd.config = _bd.Config.ORBIT
	_bd._start_guard()
	_bd.take_damage(10.0)
	assert_int(_bd.state).is_equal(_bd.State.GUARD)


# --- Blade Visuals ---

func test_three_blades_spawned() -> void:
	assert_int(_bd._blade_nodes.size()).is_equal(3)


func test_blades_are_children_of_pivot() -> void:
	for blade in _bd._blade_nodes:
		assert_that(blade.get_parent()).is_same(_bd.blade_pivot)


func test_orbit_blades_stay_within_radius() -> void:
	_bd.config = _bd.Config.ORBIT
	_bd.state = _bd.State.MOVE
	# Simulate several frames to let blades settle
	for i in 60:
		_bd._update_blade_visual(DELTA)
	for blade in _bd._blade_nodes:
		var flat_dist := Vector2(blade.position.x, blade.position.z).length()
		assert_float(flat_dist).is_less(2.0)


func test_lance_blades_stay_within_range() -> void:
	_bd.config = _bd.Config.LANCE
	_bd.state = _bd.State.MOVE
	for i in 60:
		_bd._update_blade_visual(DELTA)
	for blade in _bd._blade_nodes:
		var flat_dist := Vector2(blade.position.x, blade.position.z).length()
		# Lance blades: 2.0 + 0.8 max = 2.8, with lerp tolerance
		assert_float(flat_dist).is_less(4.0)


func test_guard_blades_tighter_than_orbit() -> void:
	# First get orbit radius
	_bd.config = _bd.Config.ORBIT
	_bd.state = _bd.State.MOVE
	for i in 60:
		_bd._update_blade_visual(DELTA)
	var orbit_dist := Vector2(_bd._blade_nodes[0].position.x, _bd._blade_nodes[0].position.z).length()

	# Now guard
	_bd.config = _bd.Config.ORBIT
	_bd._start_guard()
	for i in 60:
		_bd._update_blade_visual(DELTA)
	var guard_dist := Vector2(_bd._blade_nodes[0].position.x, _bd._blade_nodes[0].position.z).length()

	assert_float(guard_dist).is_less(orbit_dist)


func test_surge_blades_move_forward() -> void:
	_bd.config = _bd.Config.ORBIT
	_bd._start_surge()
	for i in 30:
		_bd._update_blade_visual(DELTA)
	# All blades should be in front (negative local Z)
	for blade in _bd._blade_nodes:
		assert_float(blade.position.z).is_less(0.0)


func test_dash_blades_trail_behind() -> void:
	_bd._start_dash()
	for i in 30:
		_bd._update_blade_visual(DELTA)
	# All blades should be behind (positive local Z)
	for blade in _bd._blade_nodes:
		assert_float(blade.position.z).is_greater(0.0)


func test_blade_visual_produces_three_targets_per_state() -> void:
	# Verify no index-out-of-bounds across all combat states
	for s in [_bd.State.EDGE, _bd.State.SURGE_WINDUP, _bd.State.SURGE,
			_bd.State.GUARD, _bd.State.RECALL, _bd.State.DASH, _bd.State.MOVE]:
		_bd.state = s
		_bd._state_timer = 0.15
		# Should not crash — each state must produce 3 target positions
		_bd._update_blade_visual(DELTA)
	# If we get here without error, all states produce valid targets
	assert_bool(true).is_true()


# --- Tuning values ---

func test_cast_range_is_reasonable() -> void:
	assert_float(_bd.cast_range).is_greater(5.0)
	assert_float(_bd.cast_range).is_less(50.0)


func test_gcd_is_short() -> void:
	assert_float(_bd.gcd_duration).is_greater(0.1)
	assert_float(_bd.gcd_duration).is_less(2.0)


func test_dash_duration_is_snappy() -> void:
	assert_float(_bd.dash_duration).is_less(0.5)


func test_dash_speed_exceeds_sprint() -> void:
	assert_float(_bd.dash_speed).is_greater(_bd.sprint_speed)
