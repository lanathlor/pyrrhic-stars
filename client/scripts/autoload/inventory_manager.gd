extends Node

## Client-side inventory state manager.
## Receives inventory snapshots from the server and provides query access.

signal inventory_changed
signal equipment_changed

## Equipped items keyed by slot_id (int) → item Dictionary.
var equipped: Dictionary = {}

## Bag items as a flat array of item Dictionaries.
var bag: Array = []

## Current class name — set by main.gd when class changes.
var current_class: String = "gunner"

## Aggregated gear stats from the server.
var computed_stats: Dictionary = {
	"hull": 0.0,
	"output": 0.0,
	"plating": 0.0,
	"tempo": 0.0,
	"identity": 0.0,
	"mastery": 0.0,
}


func _ready() -> void:
	NetworkManager.inventory_state_received.connect(_on_inventory_state)


func _on_inventory_state(data: Dictionary) -> void:
	equipped.clear()
	for item_info: Dictionary in data.equipped:
		equipped[item_info.slot_id] = item_info

	bag = []
	for item_info: Dictionary in data.bag:
		bag.append(item_info)

	computed_stats = data.stats

	equipment_changed.emit()
	inventory_changed.emit()


## Request to equip an item. Server validates and sends back new state.
func equip_item(item_id: int, slot_id: int) -> void:
	NetworkManager.send_equip_item(item_id, slot_id)


## Request to unequip an item from a slot. Server validates and sends back new state.
func unequip_item(slot_id: int) -> void:
	NetworkManager.send_unequip_item(slot_id)


## Returns the equipped item for a slot, or null.
func get_equipped(slot_id: int) -> Variant:
	return equipped.get(slot_id)


## Returns the total value of a stat from computed_stats.
func get_stat(stat_name: String) -> float:
	return computed_stats.get(stat_name, 0.0)
