class_name TestAbilityCatalog
extends GdUnitTestSuite

## Tests for AbilityCatalog autoload — catalog storage, lookups, and filtering.

const AbilityCatalogScript := preload("res://scripts/autoload/ability_catalog.gd")

var _cat: AbilityCatalogScript


func before_test() -> void:
	_cat = auto_free(AbilityCatalogScript.new())
	add_child(_cat)
	await get_tree().process_frame


func _sample_catalog() -> Array:
	return [
		{id = "mending_surge", name = "Mending Surge", school = "bioarcanotechnic",
			ability_type = "enhancement", delivery = "direct", flux_cost = "high",
			description = "Big heal.", cooldown = 0.0, commit_time = 0.4,
			implemented = true, affinity = "primary"},
		{id = "mending_beam", name = "Mending Beam", school = "bioarcanotechnic",
			ability_type = "enhancement", delivery = "beam", flux_cost = "medium",
			description = "Channel heal.", cooldown = 0.0, commit_time = 2.0,
			implemented = true, affinity = "primary"},
		{id = "frost_ward", name = "Frost Ward", school = "frost",
			ability_type = "protection", delivery = "direct", flux_cost = "medium",
			description = "Shield.", cooldown = 12.0, commit_time = 0.2,
			implemented = true, affinity = "primary"},
		{id = "fireball", name = "Fireball", school = "fire",
			ability_type = "destruction", delivery = "zone", flux_cost = "high",
			description = "AoE explosion.", cooldown = 0.0, commit_time = 3.0,
			implemented = false, affinity = "off"},
		{id = "chain_lightning", name = "Chain Lightning", school = "electricity",
			ability_type = "destruction", delivery = "bolt", flux_cost = "medium",
			description = "Bouncing bolt.", cooldown = 6.0, commit_time = 0.8,
			implemented = false, affinity = "secondary"},
	]


# =============================================================================
# Catalog storage
# =============================================================================


func test_initial_catalog_empty() -> void:
	assert_array(_cat.catalog).is_empty()


func test_initial_loadout_empty_strings() -> void:
	assert_array(_cat.current_loadout).has_size(6)
	for s in _cat.current_loadout:
		assert_str(s).is_equal("")


func test_on_catalog_stores_entries() -> void:
	_cat._on_catalog(_sample_catalog())
	assert_int(_cat.catalog.size()).is_equal(5)


func test_on_loadout_stores_slots() -> void:
	var slots := ["mending_surge", "mending_beam", "frost_ward", "", "", ""]
	_cat._on_loadout(slots)
	assert_str(_cat.current_loadout[0]).is_equal("mending_surge")
	assert_str(_cat.current_loadout[2]).is_equal("frost_ward")
	assert_str(_cat.current_loadout[3]).is_equal("")


# =============================================================================
# get_ability lookup
# =============================================================================


func test_get_ability_found() -> void:
	_cat._on_catalog(_sample_catalog())
	var ability := _cat.get_ability("frost_ward")
	assert_str(ability["name"]).is_equal("Frost Ward")
	assert_str(ability["school"]).is_equal("frost")


func test_get_ability_not_found() -> void:
	_cat._on_catalog(_sample_catalog())
	var ability := _cat.get_ability("nonexistent_ability")
	assert_dict(ability).is_empty()


func test_get_ability_empty_catalog() -> void:
	var ability := _cat.get_ability("mending_surge")
	assert_dict(ability).is_empty()


# =============================================================================
# get_abilities_by_school
# =============================================================================


func test_abilities_by_school_bioarcanotechnic() -> void:
	_cat._on_catalog(_sample_catalog())
	var abilities := _cat.get_abilities_by_school("bioarcanotechnic")
	assert_int(abilities.size()).is_equal(2)
	assert_str(abilities[0]["id"]).is_equal("mending_surge")
	assert_str(abilities[1]["id"]).is_equal("mending_beam")


func test_abilities_by_school_frost() -> void:
	_cat._on_catalog(_sample_catalog())
	var abilities := _cat.get_abilities_by_school("frost")
	assert_int(abilities.size()).is_equal(1)
	assert_str(abilities[0]["id"]).is_equal("frost_ward")


func test_abilities_by_school_empty_returns_all() -> void:
	_cat._on_catalog(_sample_catalog())
	var abilities := _cat.get_abilities_by_school("")
	assert_int(abilities.size()).is_equal(5)


func test_abilities_by_school_unknown_returns_empty() -> void:
	_cat._on_catalog(_sample_catalog())
	var abilities := _cat.get_abilities_by_school("arcane")
	assert_array(abilities).is_empty()


# =============================================================================
# get_schools
# =============================================================================


func test_get_schools() -> void:
	_cat._on_catalog(_sample_catalog())
	var schools := _cat.get_schools()
	assert_array(schools).contains(["bioarcanotechnic", "frost", "fire", "electricity"])
	assert_int(schools.size()).is_equal(4)


func test_get_schools_empty_catalog() -> void:
	var schools := _cat.get_schools()
	assert_array(schools).is_empty()


func test_get_schools_no_duplicates() -> void:
	_cat._on_catalog(_sample_catalog())
	var schools := _cat.get_schools()
	var seen := {}
	for s in schools:
		assert_bool(seen.has(s)).is_false()
		seen[s] = true


# =============================================================================
# is_implemented
# =============================================================================


func test_is_implemented_true() -> void:
	_cat._on_catalog(_sample_catalog())
	assert_bool(_cat.is_implemented("mending_surge")).is_true()
	assert_bool(_cat.is_implemented("frost_ward")).is_true()


func test_is_implemented_false() -> void:
	_cat._on_catalog(_sample_catalog())
	assert_bool(_cat.is_implemented("fireball")).is_false()
	assert_bool(_cat.is_implemented("chain_lightning")).is_false()


func test_is_implemented_unknown() -> void:
	_cat._on_catalog(_sample_catalog())
	assert_bool(_cat.is_implemented("nonexistent")).is_false()


# =============================================================================
# get_affinity
# =============================================================================


func test_get_affinity_primary() -> void:
	_cat._on_catalog(_sample_catalog())
	assert_str(_cat.get_affinity("mending_surge")).is_equal("primary")


func test_get_affinity_secondary() -> void:
	_cat._on_catalog(_sample_catalog())
	assert_str(_cat.get_affinity("chain_lightning")).is_equal("secondary")


func test_get_affinity_off() -> void:
	_cat._on_catalog(_sample_catalog())
	assert_str(_cat.get_affinity("fireball")).is_equal("off")


func test_get_affinity_unknown_defaults_off() -> void:
	_cat._on_catalog(_sample_catalog())
	assert_str(_cat.get_affinity("nonexistent")).is_equal("off")


# =============================================================================
# Catalog update replaces previous
# =============================================================================


func test_catalog_update_replaces_old() -> void:
	_cat._on_catalog(_sample_catalog())
	assert_int(_cat.catalog.size()).is_equal(5)

	_cat._on_catalog([{id = "only_one", name = "Only", school = "pure",
		ability_type = "enhancement", implemented = true, affinity = "primary"}])
	assert_int(_cat.catalog.size()).is_equal(1)
	assert_str(_cat.get_ability("only_one")["name"]).is_equal("Only")
	assert_dict(_cat.get_ability("mending_surge")).is_empty()
