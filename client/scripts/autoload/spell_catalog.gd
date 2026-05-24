extends Node

## SpellCatalog — caches the spell catalog and current loadout from the server.
## Autoloaded; other scripts access it as SpellCatalog.

## Full spell catalog received from server. Array of Dictionary.
var catalog: Array = []

## Current loadout: 6 spell IDs.
var current_loadout: Array = ["", "", "", "", "", ""]

## Index for fast lookup.
var _by_id: Dictionary = {}


func _ready() -> void:
	NetworkManager.spell_catalog_received.connect(_on_catalog)
	NetworkManager.loadout_state_received.connect(_on_loadout)


func _on_catalog(entries: Array) -> void:
	catalog = entries
	_by_id.clear()
	for entry in entries:
		_by_id[entry.get("id", "")] = entry


func _on_loadout(slots: Array) -> void:
	current_loadout = slots


## Get a spell entry by ID, or empty dict if not found.
func get_spell(id: String) -> Dictionary:
	return _by_id.get(id, {})


## Get all spells for a given school, or all spells if school is empty.
func get_spells_by_school(school: String) -> Array:
	if school == "":
		return catalog
	var result: Array = []
	for entry in catalog:
		if entry.get("school", "") == school:
			result.append(entry)
	return result


## Get unique school names in catalog order.
func get_schools() -> Array:
	var seen: Dictionary = {}
	var schools: Array = []
	for entry in catalog:
		var s: String = entry.get("school", "")
		if s != "" and not seen.has(s):
			seen[s] = true
			schools.append(s)
	return schools


## Check if a spell is implemented.
func is_implemented(id: String) -> bool:
	var entry: Dictionary = get_spell(id)
	return entry.get("implemented", false)


## Get the affinity tier for a spell: "primary", "secondary", or "off".
func get_affinity(id: String) -> String:
	var entry: Dictionary = get_spell(id)
	return entry.get("affinity", "off")
