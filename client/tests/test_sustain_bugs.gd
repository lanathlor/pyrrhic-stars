class_name TestSustainBugs
extends GdUnitTestSuite

## Tests proving sustain mechanic bugs. Each test expresses DESIRED behavior
## and FAILS until the corresponding bug is fixed.

const ArcanoScript := preload("res://scenes/controllers/arcanotechnicien/arcanotechnicien.gd")
const CombatScript := preload("res://scenes/controllers/arcanotechnicien/harmonist_combat.gd")
const ARCANO_SCENE := "res://scenes/controllers/arcanotechnicien/arcanotechnicien.tscn"
const DELTA := 1.0 / 60.0

var _arcano: ArcanoScript
var _saved_catalog: Array
var _saved_loadout: Array
var _test_abilities: Array[Dictionary] = [
	{
		name = "Mending Surge",
		id = "mending_surge",
		dur = 0.4,
		delivery = "direct",
		cooldown_max = 0.0
	},
	{
		name = "Mending Beam",
		id = "mending_beam",
		dur = 2.0,
		delivery = "beam",
		cooldown_max = 0.0,
		sustain = true
	},
	{name = "Life Swap", id = "life_swap", dur = 0.3, delivery = "direct", cooldown_max = 6.0},
	{
		name = "Transfusion",
		id = "transfusion",
		dur = 1.5,
		delivery = "zone",
		cooldown_max = 8.0,
		sustain = true
	},
	{name = "Frost Ward", id = "frost_ward", dur = 0.2, delivery = "direct", cooldown_max = 12.0},
	{
		name = "Transfusion",
		id = "transfusion_c",
		dur = 1.5,
		delivery = "zone",
		cooldown_max = 8.0,
		sustain = true
	},
]


func before_test() -> void:
	_saved_catalog = AbilityCatalog.catalog.duplicate()
	_saved_loadout = AbilityCatalog.current_loadout.duplicate()
	AbilityCatalog._on_catalog(_test_abilities)
	(
		AbilityCatalog
		. _on_loadout(
			[
				"mending_surge",
				"mending_beam",
				"life_swap",
				"transfusion",
				"frost_ward",
				"transfusion_c",
			]
		)
	)
	_arcano = auto_free(load(ARCANO_SCENE).instantiate()) as ArcanoScript
	_arcano.position = Vector3(0.0, 5.0, 0.0)
	add_child(_arcano)
	await get_tree().process_frame


func after_test() -> void:
	AbilityCatalog._on_catalog(_saved_catalog)
	AbilityCatalog._on_loadout(_saved_loadout)
	for action in [
		"dodge", "harmonist_slot_0", "harmonist_slot_1", "heavy_attack", "ability_1", "ability_2"
	]:
		if Input.is_action_pressed(action):
			Input.action_release(action)


# =============================================================================
# BUG 1: Cooldown starts at commit time, not on sustain release
#
# Root cause: start_ability() line 60-62 sets cooldown immediately for ALL abilities.
# Sustain abilities should defer cooldown until cancel_sustain().
# =============================================================================


func test_bug1a_transfusion_cooldown_set_at_commit_time() -> void:
	## Verify: sustain abilities do NOT set cooldown at commit time.
	## Transfusion (slot 3, cooldown_max=8.0) cooldown should remain ZERO
	## after committing because the sustain hasn't ended yet.
	var ability: Dictionary = _test_abilities[3]  # Transfusion
	assert_bool(ability.get("sustain", false)).is_true()
	var cd_max: float = ability.get("cooldown_max", 0.0)
	assert_float(cd_max).is_equal(8.0)
	# Call the real start_ability — cooldown should be deferred for sustain abilities
	_arcano._cooldowns[3] = 0.0
	_arcano.combat.start_ability(3)
	assert_float(_arcano._cooldowns[3]).is_equal(0.0)


func test_bug1b_cancel_sustain_does_not_set_cooldown() -> void:
	## Verify: cancel_sustain() never sets any cooldown.
	## After sustain ends, the ability's cooldown should begin.
	_arcano._cooldowns = [0.0, 0.0, 0.0, 0.0, 0.0, 0.0]
	_arcano.combat._sustaining = true
	_arcano.combat._sustain_elapsed = 2.0
	_arcano.state = ArcanoScript.State.CHANNELING
	_arcano._committing_ability = {name = "Transfusion", cooldown_max = 8.0, sustain = true}
	_arcano.combat.cancel_sustain()
	# Expected: cooldown starts on cancel (8.0 for Transfusion)
	assert_float(_arcano._cooldowns[3]).is_greater(0.0)


# =============================================================================
# BUG 2: Server commit/execute phase kills client sustain (no channel bar)
#
# Root cause: apply_server_state cancels sustain whenever channel_phase != 3.
# But server CommitTime (3.0s) > client dur (2.0s), so the server is still in
# commit/execute when the client enters sustain. The check should only cancel
# when the server transitions OUT of sustain, not when it hasn't arrived yet.
# =============================================================================


func test_bug2a_server_commit_kills_client_sustain() -> void:
	## Client sustain is active. Server sends channel_phase=1 (still in commit).
	## Expected: client sustain survives (server just hasn't caught up).
	_arcano.combat._sustaining = true
	_arcano.combat._sustain_elapsed = 0.1
	_arcano.state = ArcanoScript.State.CHANNELING
	_arcano._committing_ability = {name = "Mending Beam", dur = 2.0, sustain = true}
	_arcano._alive = true
	_arcano.apply_server_state({pos = Vector3.ZERO, rot_y = 0.0, health = 100.0, channel_phase = 1})
	assert_bool(_arcano.combat._sustaining).is_true()


func test_bug2b_server_execute_kills_client_sustain() -> void:
	## Same but server is in execute phase (channel_phase=2).
	_arcano.combat._sustaining = true
	_arcano.combat._sustain_elapsed = 0.1
	_arcano.state = ArcanoScript.State.CHANNELING
	_arcano._committing_ability = {name = "Mending Beam", dur = 2.0, sustain = true}
	_arcano._alive = true
	_arcano.apply_server_state({pos = Vector3.ZERO, rot_y = 0.0, health = 100.0, channel_phase = 2})
	assert_bool(_arcano.combat._sustaining).is_true()


func test_bug2c_server_idle_after_sustain_should_cancel() -> void:
	## Regression guard: server sends channel_phase=0 AFTER having been in sustain.
	## This IS a valid cancel (server-side interrupt). Should still work.
	_arcano.combat._sustaining = true
	_arcano.combat._sustain_elapsed = 5.0
	_arcano.state = ArcanoScript.State.CHANNELING
	_arcano._committing_ability = {name = "Mending Beam", dur = 2.0, sustain = true}
	_arcano._alive = true
	# idle — server cancelled
	_arcano.apply_server_state({pos = Vector3.ZERO, rot_y = 0.0, health = 100.0, channel_phase = 0})
	assert_bool(_arcano.combat._sustaining).is_false()


# =============================================================================
# BUG 3: Pressing C during sustain fires Gust Step instead of just cancelling
#
# Root cause: process_channeling calls _check_ability_input() at the top with no
# sustain guard. During sustain, ability input should only cancel (like ESC).
# =============================================================================


func test_ability_during_sustain_cancels_only() -> void:
	## During sustain, calling start_ability should cancel sustain only, not commit.
	## _resolve_ability requires catalog to be loaded (no offline fallback).
	_arcano.state = ArcanoScript.State.CHANNELING
	_arcano._committing_ability = {name = "Transfusion", dur = 1.5, sustain = true}
	_arcano._cast_timer = 0.0
	_arcano.combat._sustaining = true
	_arcano.combat._sustain_elapsed = 1.0
	_arcano._gcd_timer = 0.0
	_arcano._cooldowns = [0.0, 0.0, 0.0, 0.0, 0.0, 0.0]
	# Set up catalog so _resolve_ability returns a real ability
	AbilityCatalog.catalog = [{id = "mending_surge", name = "Mending Surge", delivery = "direct"}]
	AbilityCatalog._by_id = {"mending_surge": AbilityCatalog.catalog[0]}
	AbilityCatalog.current_loadout = ["mending_surge", "", "", "", "", ""]
	var saved_vfx: Node = _arcano.vfx
	_arcano.vfx = null
	_arcano.combat.start_ability(0)
	_arcano.vfx = saved_vfx
	AbilityCatalog.catalog = []
	AbilityCatalog._by_id = {}
	AbilityCatalog.current_loadout = ["", "", "", "", "", ""]
	# Expected: state should be MOVE (cancel only, no new ability)
	assert_int(_arcano.state).is_equal(ArcanoScript.State.MOVE)


# =============================================================================
# BUG 4: Client executes unbound abilities from hardcoded fallback table
#
# Root cause: _resolve_ability() falls through to HARMONIST_ABILITIES even when
# AbilityCatalog is loaded and the slot is empty. This lets the client execute
# displacement (Gust Step) locally for an ability the player hasn't bound.
# A modified client can exploit this for unauthorized movement.
# =============================================================================


func test_bug4a_resolve_ability_uses_hardcoded_when_slot_empty() -> void:
	## When server catalog is loaded but slot 5 is empty, _resolve_ability should
	## return {} (no ability). Instead it falls through to HARMONIST_ABILITIES[5].
	# Populate AbilityCatalog with a minimal catalog so catalog.size() > 0
	AbilityCatalog.catalog = [{id = "mending_surge", name = "Mending Surge"}]
	AbilityCatalog._by_id = {"mending_surge": AbilityCatalog.catalog[0]}
	# Set loadout with slot 5 explicitly empty (gust_step not equipped)
	AbilityCatalog.current_loadout = ["mending_surge", "", "", "", "", ""]
	var result: Dictionary = _arcano.combat._resolve_ability(5)
	# Clean up
	AbilityCatalog.catalog = []
	AbilityCatalog._by_id = {}
	AbilityCatalog.current_loadout = ["", "", "", "", "", ""]
	# Expected: empty dict (slot 5 is unbound)
	# Actual: returns Gust Step from HARMONIST_ABILITIES[5]
	assert_dict(result).is_empty()


func test_bug4b_unbound_displacement_still_moves_player() -> void:
	## Even when Gust Step is not in the loadout, start_ability(5) executes
	## the displacement locally because _resolve_ability falls through.
	## This is a client-side trust vulnerability: the client moves the player
	## without server authorization.
	AbilityCatalog.catalog = [{id = "mending_surge", name = "Mending Surge"}]
	AbilityCatalog._by_id = {"mending_surge": AbilityCatalog.catalog[0]}
	AbilityCatalog.current_loadout = ["mending_surge", "", "", "", "", ""]
	_arcano._gcd_timer = 0.0
	_arcano._cooldowns = [0.0, 0.0, 0.0, 0.0, 0.0, 0.0]
	var saved_vfx: Node = _arcano.vfx
	_arcano.vfx = null
	_arcano.combat.start_ability(5)
	_arcano.vfx = saved_vfx
	# Clean up
	AbilityCatalog.catalog = []
	AbilityCatalog._by_id = {}
	AbilityCatalog.current_loadout = ["", "", "", "", "", ""]
	# Expected: MOVE (ability not in loadout, should be blocked)
	# Actual: DODGE (Gust Step displacement executed from hardcoded table)
	assert_int(_arcano.state).is_equal(ArcanoScript.State.MOVE)
