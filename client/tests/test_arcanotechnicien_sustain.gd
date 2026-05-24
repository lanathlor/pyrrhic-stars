class_name TestArcanotechnicienSustain
extends GdUnitTestSuite

## Tests for the Arcanotechnicien sustain mechanic — harmonist_combat sustain
## state, ability table flags, HUD sustain display, and arcanotechnicien controller
## integration.

const ArcanoScript := preload("res://scenes/controllers/arcanotechnicien/arcanotechnicien.gd")
const CombatScript := preload("res://scenes/controllers/arcanotechnicien/harmonist_combat.gd")
const HudScript := preload("res://scenes/shared/hud/arcanotechnicien_hud.gd")
const ARCANO_SCENE := "res://scenes/controllers/arcanotechnicien/arcanotechnicien.tscn"
const DELTA := 1.0 / 60.0

var _arcano: ArcanoScript
var _hud: HudScript


func before_test() -> void:
	_arcano = auto_free(load(ARCANO_SCENE).instantiate()) as ArcanoScript
	_arcano.position = Vector3(0.0, 5.0, 0.0)
	add_child(_arcano)
	await get_tree().process_frame
	_hud = _arcano.hud


func after_test() -> void:
	for action in [
		"move_forward",
		"move_backward",
		"move_left",
		"move_right",
		"dodge",
		"harmonist_slot_0",
		"harmonist_slot_1",
		"heavy_attack",
		"ability_1",
		"ability_2",
		"ui_cancel"
	]:
		if Input.is_action_pressed(action):
			Input.action_release(action)


# =============================================================================
# Ability table — sustain flags
# =============================================================================


func test_ability_table_mending_beam_has_sustain_flag() -> void:
	var ability: Dictionary = ArcanoScript.HARMONIST_ABILITIES[1]
	assert_str(ability.name).is_equal("Mending Beam")
	assert_bool(ability.get("sustain", false)).is_true()


func test_ability_table_transfusion_has_sustain_flag() -> void:
	var ability: Dictionary = ArcanoScript.HARMONIST_ABILITIES[3]
	assert_str(ability.name).is_equal("Transfusion")
	assert_bool(ability.get("sustain", false)).is_true()


func test_ability_table_mending_surge_no_sustain() -> void:
	var ability: Dictionary = ArcanoScript.HARMONIST_ABILITIES[0]
	assert_str(ability.name).is_equal("Mending Surge")
	assert_bool(ability.get("sustain", false)).is_false()


func test_ability_table_life_swap_no_sustain() -> void:
	var ability: Dictionary = ArcanoScript.HARMONIST_ABILITIES[2]
	assert_str(ability.name).is_equal("Life Swap")
	assert_bool(ability.get("sustain", false)).is_false()


func test_ability_table_frost_ward_no_sustain() -> void:
	var ability: Dictionary = ArcanoScript.HARMONIST_ABILITIES[4]
	assert_str(ability.name).is_equal("Frost Ward")
	assert_bool(ability.get("sustain", false)).is_false()


func test_ability_table_slot5_is_transfusion_with_sustain() -> void:
	var ability: Dictionary = ArcanoScript.HARMONIST_ABILITIES[5]
	assert_str(ability.name).is_equal("Transfusion")
	assert_bool(ability.get("sustain", false)).is_true()


func test_ability_table_sustain_count() -> void:
	# Exactly 3 out of 6 abilities should have sustain (Mending Beam, Transfusion x2)
	var count: int = 0
	for ability in ArcanoScript.HARMONIST_ABILITIES:
		if ability.get("sustain", false):
			count += 1
	assert_int(count).is_equal(3)


# =============================================================================
# Ability table — sustained abilities are channels (dur > 0.5)
# =============================================================================


func test_mending_beam_is_channel_duration() -> void:
	var ability: Dictionary = ArcanoScript.HARMONIST_ABILITIES[1]
	# Sustained abilities must be channels (dur > 0.5 routes to CHANNELING)
	assert_float(ability.dur).is_greater(0.5)


func test_transfusion_is_channel_duration() -> void:
	var ability: Dictionary = ArcanoScript.HARMONIST_ABILITIES[3]
	assert_float(ability.dur).is_greater(0.5)


# =============================================================================
# Combat subsystem — sustain initial state
# =============================================================================


func test_combat_sustain_initially_false() -> void:
	assert_bool(_arcano.combat._sustaining).is_false()


func test_combat_sustain_elapsed_initially_zero() -> void:
	assert_float(_arcano.combat._sustain_elapsed).is_equal(0.0)


# =============================================================================
# Combat subsystem — _ability_has_sustain helper
# =============================================================================


func test_ability_has_sustain_true() -> void:
	assert_bool(_arcano.combat._ability_has_sustain({sustain = true})).is_true()


func test_ability_has_sustain_false_when_missing() -> void:
	assert_bool(_arcano.combat._ability_has_sustain({})).is_false()


func test_ability_has_sustain_false_when_explicit_false() -> void:
	assert_bool(_arcano.combat._ability_has_sustain({sustain = false})).is_false()


# =============================================================================
# Combat subsystem — cancel_sustain
# =============================================================================


func test_cancel_sustain_resets_flag() -> void:
	_arcano.combat._sustaining = true
	_arcano.combat._sustain_elapsed = 5.0
	_arcano.state = ArcanoScript.State.CHANNELING
	_arcano._committing_ability = {name = "test"}
	_arcano.combat.cancel_sustain()
	assert_bool(_arcano.combat._sustaining).is_false()


func test_cancel_sustain_resets_elapsed() -> void:
	_arcano.combat._sustaining = true
	_arcano.combat._sustain_elapsed = 5.0
	_arcano.state = ArcanoScript.State.CHANNELING
	_arcano._committing_ability = {name = "test"}
	_arcano.combat.cancel_sustain()
	assert_float(_arcano.combat._sustain_elapsed).is_equal(0.0)


func test_cancel_sustain_enters_move_state() -> void:
	_arcano.combat._sustaining = true
	_arcano.combat._sustain_elapsed = 1.0
	_arcano.state = ArcanoScript.State.CHANNELING
	_arcano._committing_ability = {name = "test"}
	_arcano.combat.cancel_sustain()
	assert_int(_arcano.state).is_equal(ArcanoScript.State.MOVE)


func test_cancel_sustain_clears_committing_ability() -> void:
	_arcano.combat._sustaining = true
	_arcano.combat._sustain_elapsed = 1.0
	_arcano.state = ArcanoScript.State.CHANNELING
	_arcano._committing_ability = {name = "test", sustain = true}
	_arcano.combat.cancel_sustain()
	assert_dict(_arcano._committing_ability).is_empty()


func test_cancel_sustain_noop_when_not_sustaining() -> void:
	# Calling cancel_sustain when not sustaining should be safe (no crash, no state change)
	_arcano.state = ArcanoScript.State.MOVE
	_arcano.combat._sustaining = false
	_arcano.combat.cancel_sustain()
	assert_int(_arcano.state).is_equal(ArcanoScript.State.MOVE)
	assert_bool(_arcano.combat._sustaining).is_false()


# =============================================================================
# Combat subsystem — sustain entry via process_channeling
# =============================================================================


func test_channeling_enters_sustain_when_ability_has_sustain() -> void:
	# Set up a sustain ability that has finished its channel
	_arcano.state = ArcanoScript.State.CHANNELING
	_arcano._committing_ability = {name = "Mending Beam", dur = 2.0, sustain = true}
	_arcano._cast_timer = 0.01  # About to expire
	_arcano.combat._sustaining = false

	# Tick past the timer
	_arcano.combat.process_channeling(0.02)
	assert_bool(_arcano.combat._sustaining).is_true()


func test_channeling_stays_channeling_state_during_sustain() -> void:
	_arcano.state = ArcanoScript.State.CHANNELING
	_arcano._committing_ability = {name = "Mending Beam", dur = 2.0, sustain = true}
	_arcano._cast_timer = 0.01
	_arcano.combat._sustaining = false
	_arcano.combat.process_channeling(0.02)
	# Should remain in CHANNELING (not MOVE)
	assert_int(_arcano.state).is_equal(ArcanoScript.State.CHANNELING)


func test_channeling_exits_to_move_when_no_sustain() -> void:
	_arcano.state = ArcanoScript.State.CHANNELING
	_arcano._committing_ability = {name = "Life Swap", dur = 0.3}
	_arcano._cast_timer = 0.01
	_arcano.combat._sustaining = false
	_arcano.combat.process_channeling(0.02)
	# Non-sustain ability: should exit to MOVE
	assert_int(_arcano.state).is_equal(ArcanoScript.State.MOVE)


func test_sustain_elapsed_increments_during_sustain() -> void:
	_arcano.state = ArcanoScript.State.CHANNELING
	_arcano._committing_ability = {name = "Mending Beam", dur = 2.0, sustain = true}
	_arcano._cast_timer = 0.0
	_arcano.combat._sustaining = true
	_arcano.combat._sustain_elapsed = 0.0
	# Tick in sustain mode
	_arcano.combat.process_channeling(DELTA)
	assert_float(_arcano.combat._sustain_elapsed).is_greater(0.0)


func test_sustain_elapsed_accumulates() -> void:
	_arcano.state = ArcanoScript.State.CHANNELING
	_arcano._committing_ability = {name = "Mending Beam", dur = 2.0, sustain = true}
	_arcano._cast_timer = 0.0
	_arcano.combat._sustaining = true
	_arcano.combat._sustain_elapsed = 0.0
	for i in 10:
		_arcano.combat.process_channeling(DELTA)
	assert_float(_arcano.combat._sustain_elapsed).is_equal_approx(DELTA * 10.0, 0.001)


# =============================================================================
# Combat subsystem — start_ability cleans up sustain
# =============================================================================


func test_start_ability_cancels_active_sustain() -> void:
	_arcano.combat._sustaining = true
	_arcano.combat._sustain_elapsed = 3.0
	_arcano._gcd_timer = 0.0
	_arcano._cooldowns = [0.0, 0.0, 0.0, 0.0, 0.0, 0.0]
	# _resolve_ability requires catalog to be loaded (no offline fallback)
	AbilityCatalog.catalog = [{id = "mending_surge", name = "Mending Surge", delivery = "direct"}]
	AbilityCatalog._by_id = {"mending_surge": AbilityCatalog.catalog[0]}
	AbilityCatalog.current_loadout = ["mending_surge", "", "", "", "", ""]
	_arcano.combat.start_ability(0)
	AbilityCatalog.catalog = []
	AbilityCatalog._by_id = {}
	AbilityCatalog.current_loadout = ["", "", "", "", "", ""]
	assert_bool(_arcano.combat._sustaining).is_false()
	assert_float(_arcano.combat._sustain_elapsed).is_equal(0.0)


# =============================================================================
# Combat subsystem — dodge cleans up sustain
# =============================================================================


func test_start_dodge_cancels_active_sustain() -> void:
	_arcano.combat._sustaining = true
	_arcano.combat._sustain_elapsed = 2.0
	_arcano.state = ArcanoScript.State.CHANNELING
	_arcano.combat.start_dodge()
	assert_bool(_arcano.combat._sustaining).is_false()
	assert_float(_arcano.combat._sustain_elapsed).is_equal(0.0)


# =============================================================================
# apply_server_state — sustain cancel from server
# =============================================================================


func test_server_state_cancels_sustain_when_phase_not_3() -> void:
	_arcano.combat._sustaining = true
	_arcano.combat._sustain_elapsed = 1.0
	_arcano.state = ArcanoScript.State.CHANNELING
	_arcano._committing_ability = {name = "test"}
	_arcano._alive = true
	# Server sends channel_phase = 0 (idle / cooldown)
	_arcano.apply_server_state({pos = Vector3.ZERO, rot_y = 0.0, health = 100.0, channel_phase = 0})
	assert_bool(_arcano.combat._sustaining).is_false()


func test_server_state_keeps_sustain_when_phase_is_3() -> void:
	_arcano.combat._sustaining = true
	_arcano.combat._sustain_elapsed = 1.0
	_arcano.state = ArcanoScript.State.CHANNELING
	_arcano._committing_ability = {name = "test"}
	_arcano._alive = true
	# Server sends channel_phase = 3 (sustain)
	_arcano.apply_server_state({pos = Vector3.ZERO, rot_y = 0.0, health = 100.0, channel_phase = 3})
	assert_bool(_arcano.combat._sustaining).is_true()


# =============================================================================
# HUD — update_sustain / update_channel / hide_channel
# =============================================================================


func test_hud_update_sustain_activates_sustain() -> void:
	_hud.update_sustain("Mending Beam", 1.5)
	assert_bool(_hud._sustain_active).is_true()
	assert_bool(_hud._channel_active).is_true()


func test_hud_update_sustain_sets_ability_name() -> void:
	_hud.update_sustain("Mending Beam", 1.5)
	assert_str(_hud._channel_ability_name).is_equal("Mending Beam")


func test_hud_update_sustain_sets_elapsed() -> void:
	_hud.update_sustain("Mending Beam", 2.3)
	assert_float(_hud._sustain_elapsed).is_equal(2.3)


func test_hud_update_channel_clears_sustain() -> void:
	_hud.update_sustain("Mending Beam", 1.0)
	assert_bool(_hud._sustain_active).is_true()
	# Switch to normal channel
	_hud.update_channel(0.5, "Life Swap")
	assert_bool(_hud._sustain_active).is_false()


func test_hud_hide_channel_clears_sustain() -> void:
	_hud.update_sustain("Mending Beam", 1.0)
	_hud.hide_channel()
	assert_bool(_hud._sustain_active).is_false()
	assert_bool(_hud._channel_active).is_false()


func test_hud_hide_channel_clears_ability_name() -> void:
	_hud.update_sustain("Mending Beam", 1.0)
	_hud.hide_channel()
	assert_str(_hud._channel_ability_name).is_equal("")


func test_hud_update_channel_progress_sets_value() -> void:
	_hud.update_channel(0.75, "Transfusion")
	assert_float(_hud._channel_progress).is_equal_approx(0.75, 0.01)
	assert_bool(_hud._sustain_active).is_false()


func test_hud_update_channel_progress_clamps() -> void:
	_hud.update_channel(1.5, "test")
	assert_float(_hud._channel_progress).is_equal(1.0)


# =============================================================================
# HUD — _update_hud_channel integration (sustain vs channel bar)
# =============================================================================


func test_hud_channel_shows_sustain_during_sustain() -> void:
	_arcano.state = ArcanoScript.State.CHANNELING
	_arcano._committing_ability = {name = "Mending Beam", dur = 2.0, sustain = true}
	_arcano.combat._sustaining = true
	_arcano.combat._sustain_elapsed = 1.5
	_arcano._update_hud_channel()
	assert_bool(_hud._sustain_active).is_true()
	assert_str(_hud._channel_ability_name).is_equal("Mending Beam")


func test_hud_channel_shows_progress_during_normal_channel() -> void:
	_arcano.state = ArcanoScript.State.CHANNELING
	_arcano._committing_ability = {name = "Transfusion", dur = 1.5}
	_arcano._cast_timer = 0.75
	_arcano.combat._sustaining = false
	_arcano._update_hud_channel()
	assert_bool(_hud._sustain_active).is_false()
	assert_bool(_hud._channel_active).is_true()
	# Progress should be ~50% (0.75 elapsed out of 1.5 total)
	assert_float(_hud._channel_progress).is_equal_approx(0.5, 0.05)


func test_hud_channel_hidden_in_move_state() -> void:
	_arcano.state = ArcanoScript.State.MOVE
	_arcano._committing_ability = {}
	_arcano._update_hud_channel()
	assert_bool(_hud._channel_active).is_false()


# =============================================================================
# NetSerializer — VS_AT_SUSTAINING constant
# =============================================================================


func test_vs_at_sustaining_constant_exists() -> void:
	assert_int(NetSerializer.VS_AT_SUSTAINING).is_equal(45)


func test_vs_at_sustaining_distinct_from_channeling() -> void:
	assert_int(NetSerializer.VS_AT_SUSTAINING).is_not_equal(NetSerializer.VS_AT_CHANNELING)
	assert_int(NetSerializer.VS_AT_SUSTAINING).is_not_equal(NetSerializer.VS_AT_CHANNELING_BEAM)
	assert_int(NetSerializer.VS_AT_SUSTAINING).is_not_equal(NetSerializer.VS_AT_CHANNELING_ZONE)


# =============================================================================
# State enum — CHANNELING is reused for sustain
# =============================================================================


func test_channeling_state_exists() -> void:
	# Sustain reuses CHANNELING state -- no separate SUSTAINING state needed
	assert_int(ArcanoScript.State.CHANNELING).is_equal(3)


func test_no_separate_sustain_state() -> void:
	# Confirm there is no SUSTAINING state enum value
	var state_names: Array = ArcanoScript.State.keys()
	assert_bool("SUSTAINING" in state_names).is_false()
