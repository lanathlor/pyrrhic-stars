class_name TestArcanotechnicienLoadoutSecurity
extends GdUnitTestSuite

## RED TESTS — Arcanotechnicien loadout security and dodge-key interaction.
##
## These tests prove client-side security bugs where the player can execute
## abilities not in their loadout. They express DESIRED behavior and FAIL
## until the corresponding bugs are fixed.
##
## Related server tests: TestSetLoadout_Rejects* in combat_test.go

const ArcanoScript := preload("res://scenes/controllers/arcanotechnicien/arcanotechnicien.gd")
const CombatScript := preload("res://scenes/controllers/arcanotechnicien/harmonist_combat.gd")
const ARCANO_SCENE := "res://scenes/controllers/arcanotechnicien/arcanotechnicien.tscn"

var _arcano: ArcanoScript


func before_test() -> void:
	_arcano = auto_free(load(ARCANO_SCENE).instantiate()) as ArcanoScript
	_arcano.position = Vector3(0.0, 5.0, 0.0)
	add_child(_arcano)
	await get_tree().process_frame


func after_test() -> void:
	for action in [
		"dodge", "harmonist_slot_0", "harmonist_slot_1", "heavy_attack", "ability_1", "ability_2"
	]:
		if Input.is_action_pressed(action):
			Input.action_release(action)


func _cleanup_catalog() -> void:
	AbilityCatalog.catalog = []
	AbilityCatalog._by_id = {}
	AbilityCatalog.current_loadout = ["", "", "", "", "", ""]


# =============================================================================
# Hardcoded fallback table mismatch
#
# HARMONIST_ABILITIES was authored before the loadout system existed.
# Slot 5 is "Gust Step" in the hardcoded table but "Transfusion" on the server.
# When the catalog hasn't been received yet (brief window at login, or offline
# play), the client falls through to hardcoded data and commits the wrong ability.
# =============================================================================


func test_hardcoded_slot5_matches_server_default() -> void:
	## The hardcoded HARMONIST_ABILITIES[5] is Gust Step, but the server's
	## default loadout slot 5 is Transfusion. This mismatch means the
	## offline fallback commits a completely different ability than the
	## server expects.
	var ability: Dictionary = ArcanoScript.HARMONIST_ABILITIES[5]
	assert_str(ability.name).is_equal("Transfusion")


func test_hardcoded_fallback_has_no_displacement() -> void:
	## The hardcoded fallback table should not contain displacement abilities.
	## Displacement triggers client-side movement without server authorization.
	var ability: Dictionary = ArcanoScript.HARMONIST_ABILITIES[5]
	assert_str(ability.get("delivery", "")).is_not_equal("displacement")


# =============================================================================
# Offline resolve exposes unauthorized displacement
#
# When the server catalog hasn't been received yet (AbilityCatalog.catalog empty),
# _resolve_ability() falls through to the hardcoded HARMONIST_ABILITIES table.
# This returns Gust Step for slot 5 even if the player's actual server-side
# loadout doesn't include it. The client then executes displacement movement
# without server authorization.
# =============================================================================


func test_resolve_ability_offline_rejects_displacement() -> void:
	## When catalog is not loaded (offline/pre-catalog), _resolve_ability(5)
	## should not return a displacement ability from the hardcoded table.
	AbilityCatalog.catalog = []
	AbilityCatalog._by_id = {}
	AbilityCatalog.current_loadout = ["", "", "", "", "", ""]

	var result: Dictionary = _arcano.combat._resolve_ability(5)
	_cleanup_catalog()

	if not result.is_empty():
		assert_str(result.get("delivery", "")).is_not_equal("displacement")


func test_start_ability_offline_does_not_displace() -> void:
	## When catalog is not loaded and slot 5 fallback returns Gust Step,
	## start_ability(5) enters DODGE state via _start_gust_step().
	## Expected: no state change (MOVE) — can't commit displacement without server.
	AbilityCatalog.catalog = []
	AbilityCatalog._by_id = {}
	AbilityCatalog.current_loadout = ["", "", "", "", "", ""]
	_arcano._gcd_timer = 0.0
	_arcano._cooldowns = [0.0, 0.0, 0.0, 0.0, 0.0, 0.0]
	_arcano.state = ArcanoScript.State.MOVE

	var saved_vfx: Node = _arcano.vfx
	_arcano.vfx = null
	_arcano.combat.start_ability(5)
	_arcano.vfx = saved_vfx
	_cleanup_catalog()

	assert_int(_arcano.state).is_equal(ArcanoScript.State.MOVE)


# =============================================================================
# Empty slot 5 does nothing
#
# Harmonist has no dodge. Gust step is mobility through the ability book only.
# When slot 5 is empty and C is pressed, nothing should happen.
# =============================================================================


func test_empty_slot5_does_nothing() -> void:
	## With slot 5 empty (catalog loaded, no ability bound), pressing C should
	## do nothing — harmonist has no dodge fallback.
	AbilityCatalog.catalog = [{id = "mending_surge", name = "Mending Surge"}]
	AbilityCatalog._by_id = {"mending_surge": AbilityCatalog.catalog[0]}
	AbilityCatalog.current_loadout = ["mending_surge", "", "", "", "", ""]
	_arcano._gcd_timer = 0.0
	_arcano._cooldowns = [0.0, 0.0, 0.0, 0.0, 0.0, 0.0]
	_arcano.state = ArcanoScript.State.MOVE
	_arcano._is_invincible = false

	var saved_vfx: Node = _arcano.vfx
	_arcano.vfx = null
	_arcano.combat.start_ability(5)
	_arcano.vfx = saved_vfx
	_cleanup_catalog()

	assert_int(_arcano.state).is_equal(ArcanoScript.State.MOVE)
