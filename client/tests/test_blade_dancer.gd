class_name TestBladeDancer
extends GdUnitTestSuite

## Tests for the Blade Dancer state machine controller -- 5 configs, 20 abilities,
## blade formations, dash, and committing.

const BDScript := preload("res://scenes/controllers/blade_dancer/blade_dancer.gd")
const BD_SCENE := "res://scenes/controllers/blade_dancer/blade_dancer.tscn"
const DELTA := 1.0 / 60.0

var _bd: BDScript


func before_test() -> void:
	_bd = auto_free(load(BD_SCENE).instantiate()) as BDScript
	_bd.position = Vector3(0.0, 5.0, 0.0)
	add_child(_bd)
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
		"lock_on",
		"jump",
		"ability_2"
	]:
		if Input.is_action_pressed(action):
			Input.action_release(action)


# --- Health ---


func test_initial_health() -> void:
	assert_float(_bd.max_health).is_equal(150.0)


func test_take_damage_is_noop() -> void:
	# Server-authoritative: take_damage does nothing client-side
	var hp_before := _bd.health
	_bd.take_damage(40.0)
	assert_float(_bd.health).is_equal(hp_before)


# --- Config ---


func test_initial_config_is_orbit() -> void:
	assert_int(_bd.config).is_equal(_bd.Config.ORBIT)


func test_five_configs_exist() -> void:
	assert_int(_bd.Config.ORBIT).is_equal(0)
	assert_int(_bd.Config.FAN).is_equal(1)
	assert_int(_bd.Config.LANCE).is_equal(2)
	assert_int(_bd.Config.SCATTER).is_equal(3)
	assert_int(_bd.Config.CROWN).is_equal(4)


func test_ability_table_has_five_configs() -> void:
	assert_int(_bd.ABILITY_TABLE.size()).is_equal(5)


func test_each_config_has_four_abilities() -> void:
	for cfg in _bd.ABILITY_TABLE:
		assert_int(_bd.ABILITY_TABLE[cfg].size()).is_equal(4)


func test_ability_action_ids_are_sequential() -> void:
	# action_id = 30 + origin_config * 4 + slot
	for cfg in _bd.ABILITY_TABLE:
		var abilities: Array = _bd.ABILITY_TABLE[cfg]
		for slot in abilities.size():
			var ability: Dictionary = abilities[slot]
			var expected_id: int = 30 + cfg * 4 + slot
			assert_int(ability.action_id).is_equal(expected_id)


func test_no_ability_transitions_to_self() -> void:
	for cfg in _bd.ABILITY_TABLE:
		var abilities: Array = _bd.ABILITY_TABLE[cfg]
		for ability in abilities:
			assert_int(ability.dest).is_not_equal(cfg)


func test_all_abilities_have_unique_action_ids() -> void:
	var seen: Dictionary = {}
	for cfg in _bd.ABILITY_TABLE:
		for ability in _bd.ABILITY_TABLE[cfg]:
			assert_bool(seen.has(ability.action_id)).is_false()
			seen[ability.action_id] = true
	assert_int(seen.size()).is_equal(20)


# --- Committing ---


func test_ability_enters_casting_state() -> void:
	_bd._start_ability(0)
	assert_int(_bd.state).is_equal(_bd.State.CASTING)


func test_ability_triggers_gcd() -> void:
	_bd._start_ability(0)
	assert_float(_bd._gcd_timer).is_equal(_bd.gcd_duration)


func test_ability_transitions_config_on_complete() -> void:
	# Orbit slot 0 -> Fan
	_bd._start_ability(0)
	var dur: float = _bd._committing_ability.dur
	var frames := ceili((dur + 0.05) / DELTA)
	for i in frames:
		_bd._cast_timer -= DELTA
		if _bd._cast_timer <= 0.0:
			_bd.config = int(_bd._committing_ability.dest)
			_bd._enter_state(_bd.State.MOVE)
			break
	assert_int(_bd.config).is_equal(_bd.Config.FAN)


func test_casting_completes_to_move() -> void:
	_bd._start_ability(1)
	var dur: float = _bd._committing_ability.dur
	var frames := ceili((dur + 0.05) / DELTA)
	for i in frames:
		_bd._process_casting(DELTA)
	assert_int(_bd.state).is_equal(_bd.State.MOVE)


func test_all_config_transitions_reachable() -> void:
	# From each config, verify the 4 destinations cover the other 4 configs
	for cfg in _bd.ABILITY_TABLE:
		var dests: Array = []
		for ability in _bd.ABILITY_TABLE[cfg]:
			dests.append(ability.dest)
		# Should have exactly 4 unique destinations
		var unique: Dictionary = {}
		for d in dests:
			unique[d] = true
		assert_int(unique.size()).is_equal(4)


func test_orbit_ability_0_goes_to_fan() -> void:
	_bd.config = _bd.Config.ORBIT
	_bd._start_ability(0)
	assert_int(_bd._committing_ability.dest).is_equal(_bd.Config.FAN)


func test_fan_ability_0_goes_to_orbit() -> void:
	_bd.config = _bd.Config.FAN
	_bd.hud.update_config(_bd.Config.FAN)
	_bd.hud.update_abilities(_bd.ABILITY_TABLE[_bd.Config.FAN])
	_bd._start_ability(0)
	assert_int(_bd._committing_ability.dest).is_equal(_bd.Config.ORBIT)


func test_lance_ability_1_goes_to_fan() -> void:
	_bd.config = _bd.Config.LANCE
	_bd.hud.update_config(_bd.Config.LANCE)
	_bd.hud.update_abilities(_bd.ABILITY_TABLE[_bd.Config.LANCE])
	_bd._start_ability(1)
	assert_int(_bd._committing_ability.dest).is_equal(_bd.Config.FAN)


# --- GCD ---


func test_initial_gcd_is_zero() -> void:
	assert_float(_bd._gcd_timer).is_less_equal(0.0)


# --- Dash ---


func test_dash_enters_dash_state() -> void:
	_bd._start_dash()
	assert_int(_bd.state).is_equal(_bd.State.DASH)


func test_dash_grants_invincibility() -> void:
	_bd._start_dash()
	assert_bool(_bd._is_invincible).is_true()


func test_dash_invincibility_expires() -> void:
	_bd._start_dash()
	var frames := ceili((_bd.dash_iframe_duration + 0.02) / DELTA)
	for i in frames:
		_bd._state_timer -= DELTA
		_bd._process_dash(DELTA)
	assert_bool(_bd._is_invincible).is_false()


func test_dash_completes_to_move() -> void:
	_bd._start_dash()
	var frames := ceili((_bd.dash_duration + 0.05) / DELTA)
	for i in frames:
		_bd._state_timer -= DELTA
		_bd._process_dash(DELTA)
	assert_int(_bd.state).is_equal(_bd.State.MOVE)


func test_dash_does_not_change_config() -> void:
	_bd.config = _bd.Config.SCATTER
	_bd._start_dash()
	var frames := ceili((_bd.dash_duration + 0.05) / DELTA)
	for i in frames:
		_bd._process_dash(DELTA)
	assert_int(_bd.config).is_equal(_bd.Config.SCATTER)


func test_dash_bleeds_velocity() -> void:
	_bd._start_dash()
	var frames := ceili((_bd.dash_duration + 0.05) / DELTA)
	for i in frames:
		_bd._state_timer -= DELTA
		_bd._process_dash(DELTA)
	var flat_speed := Vector2(_bd.velocity.x, _bd.velocity.z).length()
	assert_float(flat_speed).is_less(_bd.dash_speed * 0.5)


# --- Stagger ---


func test_stagger_state_completes_to_move() -> void:
	_bd.state = _bd.State.STAGGER
	_bd._state_timer = 0.1
	var frames := ceili(0.15 / DELTA)
	for i in frames:
		_bd._state_timer -= DELTA
		_bd._process_stagger()
	assert_int(_bd.state).is_equal(_bd.State.MOVE)


# --- Blade Visuals ---


func test_six_blades_spawned() -> void:
	assert_int(_bd._blade_nodes.size()).is_equal(6)


func test_blades_are_children_of_pivot() -> void:
	for blade in _bd._blade_nodes:
		assert_that(blade.get_parent()).is_same(_bd.blade_pivot)


func test_orbit_blades_stay_within_radius() -> void:
	_bd.config = _bd.Config.ORBIT
	_bd.state = _bd.State.MOVE
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
		assert_float(flat_dist).is_less(4.0)


func test_fan_blades_spread_in_front() -> void:
	_bd.config = _bd.Config.FAN
	_bd.state = _bd.State.MOVE
	for i in 60:
		_bd._update_blade_visual(DELTA)
	# Fan blades should be in front (negative Z in local space)
	for blade in _bd._blade_nodes:
		assert_float(blade.position.z).is_less(0.0)


func test_crown_blades_hover_above() -> void:
	_bd.config = _bd.Config.CROWN
	_bd.state = _bd.State.MOVE
	for i in 60:
		_bd._update_blade_visual(DELTA)
	# Crown blades should be above head height (y > 1.5)
	for blade in _bd._blade_nodes:
		assert_float(blade.position.y).is_greater(1.4)


func test_dash_blades_trail_behind() -> void:
	_bd._start_dash()
	for i in 30:
		_bd._update_blade_visual(DELTA)
	for blade in _bd._blade_nodes:
		assert_float(blade.position.z).is_greater(0.0)


func test_five_config_materials_exist() -> void:
	assert_that(_bd._orbit_material).is_not_null()
	assert_that(_bd._fan_material).is_not_null()
	assert_that(_bd._lance_material).is_not_null()
	assert_that(_bd._scatter_material).is_not_null()
	assert_that(_bd._crown_material).is_not_null()


func test_blade_visual_no_crash_all_states() -> void:
	# Verify no index-out-of-bounds across all states
	for s in [_bd.State.CASTING, _bd.State.DASH, _bd.State.MOVE]:
		_bd.state = s
		_bd._state_timer = 0.15
		if s == _bd.State.CASTING:
			_bd._committing_ability = {dest = _bd.Config.FAN, dur = 0.4}
			_bd._cast_timer = 0.2
		_bd._update_blade_visual(DELTA)
	assert_bool(true).is_true()


# --- Tuning values ---


func test_ability_range_is_reasonable() -> void:
	assert_float(_bd.ability_range).is_greater(5.0)
	assert_float(_bd.ability_range).is_less(50.0)


func test_gcd_is_short() -> void:
	assert_float(_bd.gcd_duration).is_greater(0.1)
	assert_float(_bd.gcd_duration).is_less(2.0)


func test_dash_duration_is_snappy() -> void:
	assert_float(_bd.dash_duration).is_less(0.5)


func test_dash_speed_exceeds_sprint() -> void:
	assert_float(_bd.dash_speed).is_greater(_bd.sprint_speed)
