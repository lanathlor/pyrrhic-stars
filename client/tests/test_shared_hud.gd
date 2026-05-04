class_name TestSharedHUD
extends GdUnitTestSuite

## Tests for the shared HUD overlay — player status, group frames, boss frame,
## damage meter, and minimap data layer.

var _hud: Control


func before_test() -> void:
	_hud = auto_free(Control.new())
	_hud.set_script(load("res://scenes/shared/hud/shared_hud.gd"))
	add_child(_hud)
	await get_tree().process_frame


# =============================================================================
# Initial values
# =============================================================================


func test_initial_health() -> void:
	assert_float(_hud._player_health).is_equal(100.0)


func test_initial_max_health() -> void:
	assert_float(_hud._player_max_health).is_equal(150.0)


func test_initial_resource_zero() -> void:
	assert_float(_hud._player_resource).is_equal(0.0)
	assert_float(_hud._player_max_resource).is_equal(0.0)


func test_initial_fight_inactive() -> void:
	assert_bool(_hud._fight_active).is_false()


func test_initial_boss_invisible() -> void:
	assert_bool(_hud._boss_visible).is_false()


func test_initial_damage_totals_empty() -> void:
	assert_dict(_hud._damage_totals).is_empty()


func test_initial_world_players_empty() -> void:
	assert_dict(_hud._world_players).is_empty()


# =============================================================================
# set_local_player
# =============================================================================


func test_set_local_player_gunner_max_hp() -> void:
	var player := auto_free(CharacterBody3D.new())
	_hud.set_local_player(player, "gunner", 1)
	assert_float(_hud._player_max_health).is_equal(150.0)


func test_set_local_player_vanguard_max_hp() -> void:
	var player := auto_free(CharacterBody3D.new())
	_hud.set_local_player(player, "vanguard", 2)
	assert_float(_hud._player_max_health).is_equal(200.0)


func test_set_local_player_blade_dancer_max_hp() -> void:
	var player := auto_free(CharacterBody3D.new())
	_hud.set_local_player(player, "blade_dancer", 3)
	assert_float(_hud._player_max_health).is_equal(150.0)


func test_set_local_player_unknown_class_defaults() -> void:
	var player := auto_free(CharacterBody3D.new())
	_hud.set_local_player(player, "unknown_class", 99)
	assert_float(_hud._player_max_health).is_equal(150.0)


func test_set_local_player_stores_peer_id() -> void:
	var player := auto_free(CharacterBody3D.new())
	_hud.set_local_player(player, "gunner", 42)
	assert_int(_hud._local_peer_id).is_equal(42)


func test_set_local_player_stores_class() -> void:
	var player := auto_free(CharacterBody3D.new())
	_hud.set_local_player(player, "vanguard", 1)
	assert_str(_hud._local_class).is_equal("vanguard")


# =============================================================================
# update_world_state
# =============================================================================


func test_update_world_state_populates_players() -> void:
	var data := {
		"players":
		[
			{"peer_id": 1, "pos": Vector3(1.0, 0.0, 2.0), "health": 100.0, "rot_y": 0.5},
			{"peer_id": 2, "pos": Vector3(3.0, 0.0, 4.0), "health": 80.0, "rot_y": 1.0},
		],
		"enemies": [],
	}
	_hud.update_world_state(data)
	assert_dict(_hud._world_players).has_size(2)
	assert_float(_hud._world_players[1]["health"]).is_equal(100.0)
	assert_float(_hud._world_players[2]["health"]).is_equal(80.0)


func test_update_world_state_populates_boss() -> void:
	var data := {
		"players": [],
		"enemies":
		[
			{
				"alive": true,
				"pos": Vector3(10.0, 0.0, 10.0),
				"health": 1500.0,
				"phase": 2,
				"def_name": "guard_captain",
				"max_health": 2000.0,
			},
		],
	}
	_hud.update_world_state(data)
	assert_bool(_hud._enemy_alive).is_true()
	assert_bool(_hud._boss_visible).is_true()
	assert_float(_hud._boss_health).is_equal(1500.0)
	assert_int(_hud._boss_phase).is_equal(2)


func test_update_world_state_trash_no_boss_frame() -> void:
	var data := {
		"players": [],
		"enemies":
		[
			{
				"alive": true,
				"pos": Vector3(5.0, 0.0, 25.0),
				"health": 150.0,
				"phase": 1,
				"def_name": "hallway_melee",
				"max_health": 200.0,
			},
		],
	}
	_hud.update_world_state(data)
	assert_bool(_hud._enemy_alive).is_true()
	assert_bool(_hud._boss_visible).is_false()


func test_update_world_state_clears_old_players() -> void:
	# First update with 2 players
	(
		_hud
		. update_world_state(
			{
				"players":
				[
					{"peer_id": 1, "pos": Vector3.ZERO, "health": 100.0, "rot_y": 0.0},
					{"peer_id": 2, "pos": Vector3.ZERO, "health": 80.0, "rot_y": 0.0},
				],
				"enemies": [],
			}
		)
	)
	assert_dict(_hud._world_players).has_size(2)
	# Second update with 1 player
	(
		_hud
		. update_world_state(
			{
				"players":
				[
					{"peer_id": 1, "pos": Vector3.ZERO, "health": 100.0, "rot_y": 0.0},
				],
				"enemies": [],
			}
		)
	)
	assert_dict(_hud._world_players).has_size(1)


# =============================================================================
# on_fight_start
# =============================================================================


func test_on_fight_start_activates_fight() -> void:
	_hud.on_fight_start()
	assert_bool(_hud._fight_active).is_true()


func test_on_fight_start_boss_driven_by_world_state() -> void:
	# Boss visibility is now driven by update_world_state, not fight_start
	_hud.on_fight_start()
	assert_bool(_hud._boss_visible).is_false()
	# Feed boss data to make it visible
	(
		_hud
		. update_world_state(
			{
				"players": [],
				"enemies":
				[
					{
						"alive": true,
						"pos": Vector3.ZERO,
						"health": 2000.0,
						"phase": 1,
						"def_name": "guard_captain",
						"max_health": 2000.0,
					}
				],
			}
		)
	)
	assert_bool(_hud._boss_visible).is_true()


func test_on_fight_start_clears_damage() -> void:
	_hud._damage_totals[1] = 500.0
	_hud.on_fight_start()
	assert_dict(_hud._damage_totals).is_empty()


func test_on_fight_start_resets_duration() -> void:
	_hud._fight_duration = 30.0
	_hud.on_fight_start()
	assert_float(_hud._fight_duration).is_equal(0.0)


# =============================================================================
# on_fight_end
# =============================================================================


func test_on_fight_end_deactivates_fight() -> void:
	_hud.on_fight_start()
	_hud.on_fight_end()
	assert_bool(_hud._fight_active).is_false()


func test_on_fight_end_keeps_boss_if_world_state() -> void:
	_hud.on_fight_start()
	# Feed boss via world state
	(
		_hud
		. update_world_state(
			{
				"players": [],
				"enemies":
				[
					{
						"alive": true,
						"pos": Vector3.ZERO,
						"health": 2000.0,
						"phase": 1,
						"def_name": "guard_captain",
						"max_health": 2000.0,
					}
				],
			}
		)
	)
	_hud.on_fight_end()
	# Boss frame stays visible as long as world state says so
	assert_bool(_hud._boss_visible).is_true()


# =============================================================================
# on_enter_hub
# =============================================================================


func test_on_enter_hub_clears_boss() -> void:
	_hud.on_fight_start()
	_hud.on_enter_hub()
	assert_bool(_hud._boss_visible).is_false()


func test_on_enter_hub_clears_fight() -> void:
	_hud.on_fight_start()
	_hud.on_enter_hub()
	assert_bool(_hud._fight_active).is_false()


func test_on_enter_hub_clears_damage() -> void:
	_hud._damage_totals[1] = 999.0
	_hud.on_enter_hub()
	assert_dict(_hud._damage_totals).is_empty()


func test_on_enter_hub_clears_world_players() -> void:
	_hud._world_players[1] = {"pos": Vector3.ZERO, "health": 100.0, "rot_y": 0.0}
	_hud.on_enter_hub()
	assert_dict(_hud._world_players).is_empty()


func test_on_enter_hub_resets_duration() -> void:
	_hud._fight_duration = 45.0
	_hud.on_enter_hub()
	assert_float(_hud._fight_duration).is_equal(0.0)


# =============================================================================
# on_damage_event
# =============================================================================


func test_on_damage_event_accumulates_damage() -> void:
	_hud.on_fight_start()
	_hud.on_damage_event({"target_peer_id": 1000, "source_peer_id": 1, "amount": 50.0})
	_hud.on_damage_event({"target_peer_id": 1000, "source_peer_id": 1, "amount": 30.0})
	assert_float(_hud._damage_totals[1]).is_equal(80.0)


func test_on_damage_event_tracks_multiple_sources() -> void:
	_hud.on_fight_start()
	_hud.on_damage_event({"target_peer_id": 1000, "source_peer_id": 1, "amount": 50.0})
	_hud.on_damage_event({"target_peer_id": 1000, "source_peer_id": 2, "amount": 70.0})
	assert_float(_hud._damage_totals[1]).is_equal(50.0)
	assert_float(_hud._damage_totals[2]).is_equal(70.0)


func test_on_damage_event_ignores_player_target() -> void:
	_hud.on_fight_start()
	# target_peer_id < 1000 means damage to a player, not an enemy
	_hud.on_damage_event({"target_peer_id": 1, "source_peer_id": 1000, "amount": 100.0})
	assert_dict(_hud._damage_totals).is_empty()


func test_on_damage_event_ignores_when_fight_inactive() -> void:
	# Fight not started
	_hud.on_damage_event({"target_peer_id": 1000, "source_peer_id": 1, "amount": 50.0})
	assert_dict(_hud._damage_totals).is_empty()


func test_on_damage_event_ignores_zero_source() -> void:
	_hud.on_fight_start()
	# source_peer_id 0 = no source (should not count)
	_hud.on_damage_event({"target_peer_id": 1000, "source_peer_id": 0, "amount": 50.0})
	assert_dict(_hud._damage_totals).is_empty()


# =============================================================================
# update_group_members
# =============================================================================


func test_update_group_members_populates_pids() -> void:
	var data := {
		"members":
		[
			{"peer_id": 1, "username": "Alice"},
			{"peer_id": 2, "username": "Bob"},
			{"peer_id": 3, "username": "Carol"},
		],
	}
	_hud.update_group_members(data)
	assert_array(_hud._group_member_pids).has_size(3)
	assert_array(_hud._group_member_pids).contains([1, 2, 3])


func test_update_group_members_stores_names() -> void:
	var data := {
		"members":
		[
			{"peer_id": 1, "username": "Alice"},
			{"peer_id": 2, "username": "Bob"},
		],
	}
	_hud.update_group_members(data)
	assert_str(_hud._group_member_names[1]).is_equal("Alice")
	assert_str(_hud._group_member_names[2]).is_equal("Bob")


func test_update_group_members_clears_previous() -> void:
	(
		_hud
		. update_group_members(
			{
				"members":
				[
					{"peer_id": 1, "username": "Alice"},
					{"peer_id": 2, "username": "Bob"},
				],
			}
		)
	)
	assert_array(_hud._group_member_pids).has_size(2)
	(
		_hud
		. update_group_members(
			{
				"members":
				[
					{"peer_id": 3, "username": "Carol"},
				],
			}
		)
	)
	assert_array(_hud._group_member_pids).has_size(1)
	assert_array(_hud._group_member_pids).contains([3])


func test_update_group_members_empty() -> void:
	_hud.update_group_members({"members": []})
	assert_array(_hud._group_member_pids).is_empty()
	assert_dict(_hud._group_member_names).is_empty()


# =============================================================================
# clear_local_player
# =============================================================================


func test_clear_local_player_nulls_ref() -> void:
	var player := auto_free(CharacterBody3D.new())
	_hud.set_local_player(player, "gunner", 1)
	_hud.clear_local_player()
	assert_that(_hud._local_player).is_null()


func test_clear_local_player_clears_state() -> void:
	_hud.on_fight_start()
	_hud._damage_totals[1] = 100.0
	_hud._world_players[1] = {"pos": Vector3.ZERO, "health": 100.0, "rot_y": 0.0}
	_hud.clear_local_player()
	assert_bool(_hud._boss_visible).is_false()
	assert_bool(_hud._fight_active).is_false()
	assert_dict(_hud._damage_totals).is_empty()
	assert_dict(_hud._world_players).is_empty()


# =============================================================================
# CLASS_MAX_HP constant
# =============================================================================


func test_class_max_hp_gunner() -> void:
	assert_float(_hud.CLASS_MAX_HP["gunner"]).is_equal(150.0)


func test_class_max_hp_vanguard() -> void:
	assert_float(_hud.CLASS_MAX_HP["vanguard"]).is_equal(200.0)


func test_class_max_hp_blade_dancer() -> void:
	assert_float(_hud.CLASS_MAX_HP["blade_dancer"]).is_equal(150.0)
