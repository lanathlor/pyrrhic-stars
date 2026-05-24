extends Node

## AbilityCatalog — caches the ability catalog and current loadout from the server.
## Autoloaded; other scripts access it as AbilityCatalog.

## Full ability catalog received from server. Array of Dictionary.
var catalog: Array = []

## Current loadout: 6 ability IDs.
var current_loadout: Array = ["", "", "", "", "", ""]

## Current flux commitment: school → percentage (0-100).
var current_commitment: Dictionary = {}

## Saved loadout presets from server. Array[Dictionary] with keys: id, name, slots, commitment.
var presets: Array = []

## Index for fast lookup.
var _by_id: Dictionary = {}


func _ready() -> void:
	NetworkManager.ability_catalog_received.connect(_on_catalog)
	NetworkManager.loadout_state_received.connect(_on_loadout)
	NetworkManager.flux_commit_state_received.connect(_on_flux_commit)
	NetworkManager.preset_list_received.connect(_on_presets)


func _on_catalog(entries: Array) -> void:
	catalog = entries
	_by_id.clear()
	for entry in entries:
		_by_id[entry.get("id", "")] = entry


func _on_loadout(slots: Array) -> void:
	current_loadout = slots


func _on_flux_commit(entries: Array) -> void:
	current_commitment.clear()
	for entry in entries:
		current_commitment[entry.get("school", "")] = entry.get("percentage", 0)


func _on_presets(list: Array) -> void:
	presets = list


## Get an ability entry by ID, or empty dict if not found.
func get_ability(id: String) -> Dictionary:
	return _by_id.get(id, {})


## Get all abilities for a given school, or all abilities if school is empty.
func get_abilities_by_school(school: String) -> Array:
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


## Check if an ability is implemented.
func is_implemented(id: String) -> bool:
	var entry: Dictionary = get_ability(id)
	return entry.get("implemented", false)


## Get the affinity tier for an ability: "primary", "secondary", or "off".
func get_affinity(id: String) -> String:
	var entry: Dictionary = get_ability(id)
	return entry.get("affinity", "off")


## Parse a commitment CSV string ("school:pct,school:pct,...") into a Dictionary.
func parse_commitment(csv: String) -> Dictionary:
	var result: Dictionary = {}
	if csv == "":
		return result
	for pair in csv.split(","):
		var parts := pair.split(":")
		if parts.size() == 2:
			result[parts[0]] = parts[1].to_int()
	return result


## Encode a commitment Dictionary (school -> int) as a CSV string.
func encode_commitment(commitment: Dictionary) -> String:
	var parts: PackedStringArray = PackedStringArray()
	for school in commitment:
		parts.append("%s:%d" % [school, commitment[school]])
	return ",".join(parts)
