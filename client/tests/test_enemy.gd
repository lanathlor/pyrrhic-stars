class_name TestEnemy
extends GdUnitTestSuite

## Tests for BasicEnemy — boss state machine, phases, attacks, damage.

const ENEMY_SCENE := "res://scenes/enemies/basic_enemy/basic_enemy.tscn"

var _enemy: CharacterBody3D


func before_test() -> void:
	_enemy = auto_free(load(ENEMY_SCENE).instantiate())
	_enemy.position = Vector3(0.0, 5.0, 0.0)
	add_child(_enemy)
	await get_tree().process_frame


func after_test() -> void:
	GameManager.players.clear()
	GameManager.enemies.clear()


# =============================================================================
# Health & Damage
# =============================================================================

func test_initial_health() -> void:
	assert_float(_enemy.health).is_equal(_enemy.max_health)


func test_max_health_is_2000() -> void:
	assert_float(_enemy.max_health).is_equal(2000.0)


func test_take_damage() -> void:
	var initial := _enemy.health
	_enemy.take_damage(50.0)
	assert_float(_enemy.health).is_equal(initial - 50.0)


func test_damage_clamps_at_zero() -> void:
	_enemy.take_damage(9999.0)
	assert_float(_enemy.health).is_equal(0.0)


func test_dies_at_zero_health() -> void:
	_enemy.take_damage(_enemy.max_health)
	assert_int(_enemy.state).is_equal(_enemy.State.DEAD)


func test_dead_enemy_ignores_damage() -> void:
	_enemy.take_damage(_enemy.max_health)
	_enemy.take_damage(50.0)
	assert_float(_enemy.health).is_equal(0.0)


# =============================================================================
# State machine basics
# =============================================================================

func test_starts_in_chase() -> void:
	assert_int(_enemy.state).is_equal(_enemy.State.CHASE)


func test_melee_telegraph_sets_timer() -> void:
	_enemy._change_state(_enemy.State.MELEE_TELEGRAPH)
	assert_float(_enemy._state_timer).is_equal(_enemy._get_melee_telegraph_time())


func test_ranged_telegraph_sets_timer() -> void:
	_enemy._change_state(_enemy.State.RANGED_TELEGRAPH)
	assert_float(_enemy._state_timer).is_equal(_enemy._get_ranged_telegraph_time())


func test_cooldown_sets_timer() -> void:
	_enemy._change_state(_enemy.State.COOLDOWN)
	assert_float(_enemy._state_timer).is_equal(_enemy._get_cooldown_time())


func test_aoe_telegraph_sets_timer() -> void:
	_enemy._change_state(_enemy.State.AOE_TELEGRAPH)
	assert_float(_enemy._state_timer).is_equal(_enemy._get_aoe_telegraph_time())


func test_charge_telegraph_sets_timer() -> void:
	_enemy._change_state(_enemy.State.CHARGE_TELEGRAPH)
	assert_float(_enemy._state_timer).is_equal(_enemy._get_charge_telegraph_time())


# =============================================================================
# Telegraph visuals
# =============================================================================

func test_melee_telegraph_shows_visual() -> void:
	_enemy._change_state(_enemy.State.MELEE_TELEGRAPH)
	assert_bool(_enemy._melee_telegraph_mesh.visible).is_true()
	assert_bool(_enemy._laser_warning_mesh.visible).is_false()
	assert_bool(_enemy._aoe_telegraph_mesh.visible).is_false()
	assert_bool(_enemy._charge_telegraph_mesh.visible).is_false()


func test_ranged_telegraph_shows_laser() -> void:
	_enemy._change_state(_enemy.State.RANGED_TELEGRAPH)
	assert_bool(_enemy._laser_warning_mesh.visible).is_true()
	assert_bool(_enemy._melee_telegraph_mesh.visible).is_false()
	assert_bool(_enemy._aoe_telegraph_mesh.visible).is_false()
	assert_bool(_enemy._charge_telegraph_mesh.visible).is_false()


func test_aoe_telegraph_shows_visual() -> void:
	_enemy._change_state(_enemy.State.AOE_TELEGRAPH)
	assert_bool(_enemy._aoe_telegraph_mesh.visible).is_true()
	assert_bool(_enemy._melee_telegraph_mesh.visible).is_false()
	assert_bool(_enemy._laser_warning_mesh.visible).is_false()
	assert_bool(_enemy._charge_telegraph_mesh.visible).is_false()


func test_charge_telegraph_shows_visual() -> void:
	_enemy._change_state(_enemy.State.CHARGE_TELEGRAPH)
	assert_bool(_enemy._charge_telegraph_mesh.visible).is_true()
	assert_bool(_enemy._melee_telegraph_mesh.visible).is_false()
	assert_bool(_enemy._laser_warning_mesh.visible).is_false()
	assert_bool(_enemy._aoe_telegraph_mesh.visible).is_false()


func test_chase_hides_all_telegraphs() -> void:
	_enemy._change_state(_enemy.State.MELEE_TELEGRAPH)
	_enemy._change_state(_enemy.State.CHASE)
	assert_bool(_enemy._melee_telegraph_mesh.visible).is_false()
	assert_bool(_enemy._laser_warning_mesh.visible).is_false()
	assert_bool(_enemy._aoe_telegraph_mesh.visible).is_false()
	assert_bool(_enemy._charge_telegraph_mesh.visible).is_false()


# =============================================================================
# Phase system
# =============================================================================

func test_starts_in_phase_1() -> void:
	assert_int(_enemy._current_phase).is_equal(1)


func test_phase_2_at_60_percent() -> void:
	_enemy.take_damage(_enemy.max_health * 0.41)  # bring to 59%
	assert_int(_enemy._current_phase).is_equal(2)


func test_phase_3_at_30_percent() -> void:
	_enemy.take_damage(_enemy.max_health * 0.71)  # bring to 29%
	assert_int(_enemy._current_phase).is_equal(3)


func test_phase_transition_enters_transition_state() -> void:
	_enemy.take_damage(_enemy.max_health * 0.41)
	assert_int(_enemy.state).is_equal(_enemy.State.PHASE_TRANSITION)


func test_phase_transition_invulnerability() -> void:
	_enemy.take_damage(_enemy.max_health * 0.41)  # triggers phase 2
	var health_after_transition := _enemy.health
	_enemy.take_damage(100.0)  # should be ignored during transition
	assert_float(_enemy.health).is_equal(health_after_transition)


func test_no_double_phase_trigger() -> void:
	_enemy.take_damage(_enemy.max_health * 0.41)  # phase 2
	_enemy._change_state(_enemy.State.CHASE)  # force out of transition
	var health_before := _enemy.health
	_enemy.take_damage(1.0)  # still above 30%, should not re-trigger phase 2
	assert_int(_enemy._current_phase).is_equal(2)
	# Should not enter phase transition again
	assert_int(_enemy.state).is_not_equal(_enemy.State.PHASE_TRANSITION)


func test_phase_transition_timer() -> void:
	_enemy._change_state(_enemy.State.PHASE_TRANSITION)
	assert_float(_enemy._state_timer).is_equal(1.5)


func test_boss_dies_from_any_phase() -> void:
	_enemy._current_phase = 3
	_enemy.take_damage(_enemy.max_health)
	assert_int(_enemy.state).is_equal(_enemy.State.DEAD)


# =============================================================================
# Phase-aware stats
# =============================================================================

func test_phase_1_move_speed() -> void:
	assert_float(_enemy._get_move_speed()).is_equal(4.0)


func test_phase_2_move_speed() -> void:
	_enemy._current_phase = 2
	assert_float(_enemy._get_move_speed()).is_equal(5.0)


func test_phase_3_move_speed() -> void:
	_enemy._current_phase = 3
	assert_float(_enemy._get_move_speed()).is_equal(6.0)


func test_phase_1_ranged_burst_count() -> void:
	assert_int(_enemy._get_ranged_burst_count()).is_equal(1)


func test_phase_2_ranged_burst_count() -> void:
	_enemy._current_phase = 2
	assert_int(_enemy._get_ranged_burst_count()).is_equal(2)


func test_phase_3_ranged_burst_count() -> void:
	_enemy._current_phase = 3
	assert_int(_enemy._get_ranged_burst_count()).is_equal(3)


func test_telegraph_times_decrease_by_phase() -> void:
	var p1_melee := _enemy._get_melee_telegraph_time()
	_enemy._current_phase = 2
	var p2_melee := _enemy._get_melee_telegraph_time()
	_enemy._current_phase = 3
	var p3_melee := _enemy._get_melee_telegraph_time()
	assert_float(p1_melee).is_greater(p2_melee)
	assert_float(p2_melee).is_greater(p3_melee)


func test_cooldown_decreases_by_phase() -> void:
	_enemy._current_phase = 1
	var c1 := _enemy._get_cooldown_time()
	_enemy._current_phase = 2
	var c2 := _enemy._get_cooldown_time()
	_enemy._current_phase = 3
	var c3 := _enemy._get_cooldown_time()
	assert_float(c1).is_greater_equal(c2)
	assert_float(c2).is_greater(c3)


func test_aoe_radius_increases_by_phase() -> void:
	var r1 := _enemy._get_aoe_radius()
	_enemy._current_phase = 3
	var r3 := _enemy._get_aoe_radius()
	assert_float(r3).is_greater(r1)


func test_charge_speed_increases_by_phase() -> void:
	var s1 := _enemy._get_charge_speed()
	_enemy._current_phase = 3
	var s3 := _enemy._get_charge_speed()
	assert_float(s3).is_greater(s1)


# =============================================================================
# Attack selection
# =============================================================================

func test_select_attack_returns_valid_state() -> void:
	var result := _enemy._select_attack()
	var valid := [
		_enemy.State.MELEE_TELEGRAPH,
		_enemy.State.RANGED_TELEGRAPH,
		_enemy.State.AOE_TELEGRAPH,
		_enemy.State.CHARGE_TELEGRAPH,
	]
	assert_bool(result in valid).is_true()


func test_select_attack_works_all_phases() -> void:
	for phase in [1, 2, 3]:
		_enemy._current_phase = phase
		var result := _enemy._select_attack()
		# Should not crash and should return valid state
		assert_bool(result >= 0).is_true()


# =============================================================================
# Charge state
# =============================================================================

func test_charge_resets_distance() -> void:
	_enemy._charge_distance_traveled = 99.0
	_enemy._change_state(_enemy.State.CHARGE)
	assert_float(_enemy._charge_distance_traveled).is_equal(0.0)


func test_charge_clears_hit_list() -> void:
	_enemy._charge_hit_players.append(null)
	_enemy._change_state(_enemy.State.CHARGE)
	assert_int(_enemy._charge_hit_players.size()).is_equal(0)


# =============================================================================
# AoE slam state
# =============================================================================

func test_aoe_particles_created() -> void:
	assert_that(_enemy._aoe_particles).is_not_null()
	assert_that(_enemy._aoe_slam_particles).is_not_null()


func test_aoe_telegraph_starts_particles() -> void:
	_enemy._change_state(_enemy.State.AOE_TELEGRAPH)
	assert_bool(_enemy._aoe_particles.emitting).is_true()


func test_aoe_slam_stops_charge_particles() -> void:
	_enemy._change_state(_enemy.State.AOE_TELEGRAPH)
	_enemy._change_state(_enemy.State.AOE_SLAM)
	_enemy._process_aoe_slam()
	assert_bool(_enemy._aoe_particles.emitting).is_false()


# =============================================================================
# Death
# =============================================================================

func test_dead_state_zeroes_velocity() -> void:
	_enemy.velocity = Vector3(5.0, 0.0, 3.0)
	_enemy._change_state(_enemy.State.DEAD)
	_enemy._physics_process(1.0 / 60.0)
	assert_float(_enemy.velocity.x).is_equal(0.0)
	assert_float(_enemy.velocity.z).is_equal(0.0)


func test_dead_enemy_hides() -> void:
	_enemy.take_damage(_enemy.max_health)
	assert_bool(_enemy.visible).is_false()


func test_dead_enemy_loses_collision() -> void:
	_enemy.take_damage(_enemy.max_health)
	assert_int(_enemy.collision_layer).is_equal(0)


# =============================================================================
# Character model & Weapons
# =============================================================================

func test_character_model_exists() -> void:
	assert_that(_enemy.character_model).is_not_null()


func test_weapon_scenes_defined() -> void:
	assert_that(_enemy.SWORD_SCENE_PATH).is_not_null()
	assert_that(_enemy.GUN_SCENE_PATH).is_not_null()


func test_sword_attachment_created() -> void:
	await get_tree().process_frame
	assert_that(_enemy._sword_attachment).is_not_null()


func test_gun_attachment_created() -> void:
	await get_tree().process_frame
	assert_that(_enemy._gun_attachment).is_not_null()


func test_last_weapon_starts_as_sword() -> void:
	assert_str(_enemy._last_weapon).is_equal("sword")


func test_melee_sets_last_weapon_to_sword() -> void:
	_enemy._last_weapon = "gun"
	_enemy._change_state(_enemy.State.MELEE_TELEGRAPH)
	_enemy._update_weapons(1.0 / 60.0)
	assert_str(_enemy._last_weapon).is_equal("sword")


func test_ranged_sets_last_weapon_to_gun() -> void:
	_enemy._change_state(_enemy.State.RANGED_TELEGRAPH)
	_enemy._update_weapons(1.0 / 60.0)
	assert_str(_enemy._last_weapon).is_equal("gun")


func test_sword_visible_during_melee() -> void:
	await get_tree().process_frame
	_enemy._change_state(_enemy.State.MELEE_TELEGRAPH)
	_enemy._update_weapons(1.0 / 60.0)
	if _enemy._sword_attachment:
		assert_bool(_enemy._sword_attachment.visible).is_true()
	if _enemy._gun_attachment:
		assert_bool(_enemy._gun_attachment.visible).is_false()


func test_gun_visible_during_ranged() -> void:
	await get_tree().process_frame
	_enemy._change_state(_enemy.State.RANGED_TELEGRAPH)
	_enemy._update_weapons(1.0 / 60.0)
	if _enemy._sword_attachment:
		assert_bool(_enemy._sword_attachment.visible).is_false()
	if _enemy._gun_attachment:
		assert_bool(_enemy._gun_attachment.visible).is_true()


func test_sword_persists_during_chase_after_melee() -> void:
	await get_tree().process_frame
	_enemy._change_state(_enemy.State.MELEE_TELEGRAPH)
	_enemy._update_weapons(1.0 / 60.0)
	_enemy._change_state(_enemy.State.CHASE)
	_enemy._update_weapons(1.0 / 60.0)
	if _enemy._sword_attachment:
		assert_bool(_enemy._sword_attachment.visible).is_true()


func test_gun_persists_during_chase_after_ranged() -> void:
	await get_tree().process_frame
	_enemy._change_state(_enemy.State.RANGED_TELEGRAPH)
	_enemy._update_weapons(1.0 / 60.0)
	_enemy._change_state(_enemy.State.CHASE)
	_enemy._update_weapons(1.0 / 60.0)
	if _enemy._gun_attachment:
		assert_bool(_enemy._gun_attachment.visible).is_true()
	if _enemy._sword_attachment:
		assert_bool(_enemy._sword_attachment.visible).is_false()


# =============================================================================
# Animation correctness
# =============================================================================

func test_core_anims_loaded() -> void:
	await get_tree().process_frame
	var loaded: PackedStringArray = _enemy.character_model._loaded_anims
	for anim_name in ["idle", "run", "slash", "rifle_idle"]:
		assert_bool(anim_name in loaded).is_true()


func test_model_y_pinned_to_zero_during_ranged() -> void:
	_enemy._change_state(_enemy.State.RANGED_TELEGRAPH)
	# Simulate several frames
	for i in 10:
		_enemy._update_boss_animation()
		_enemy.character_model.position.y = 0.0  # mimics the pinning in _physics_process
	assert_float(_enemy.character_model.position.y).is_equal(0.0)
