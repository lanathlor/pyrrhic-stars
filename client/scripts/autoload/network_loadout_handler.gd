class_name NetworkLoadoutHandler
extends RefCounted

## Handles inventory, loadout, flux commitment, and preset network operations.
## Instantiated by NetworkManager; not an autoload itself.

var _net: Node  # Reference to NetworkManager


func _init(net: Node) -> void:
	_net = net


# =============================================================================
# Inventory
# =============================================================================


func handle_inventory_state(payload: PackedByteArray) -> void:
	var data := NetSerializer.Inv.decode_inventory_state(payload)
	print("[Net] Inventory: %d equipped, %d bag" % [data.equipped.size(), data.bag.size()])
	_net.inventory_state_received.emit(data)


func send_equip_item(item_id: int, slot_id: int) -> void:
	_net.send_msg(
		NetSerializer.OP_EQUIP_ITEM, NetSerializer.Inv.encode_equip_item(item_id, slot_id)
	)


func send_unequip_item(slot_id: int) -> void:
	_net.send_msg(NetSerializer.OP_UNEQUIP_ITEM, NetSerializer.Inv.encode_unequip_item(slot_id))


# =============================================================================
# Ability catalog & loadout
# =============================================================================


func handle_ability_catalog(payload: PackedByteArray) -> void:
	var catalog: Array = NetSerializer.Inv.decode_ability_catalog(payload)
	print("[Net] Received ability catalog: %d abilities" % catalog.size())
	_net.ability_catalog_received.emit(catalog)


func handle_loadout_state(payload: PackedByteArray) -> void:
	var slots: Array = NetSerializer.Inv.decode_loadout_state(payload)
	print("[Net] Received loadout: %s" % [slots])
	_net.loadout_state_received.emit(slots)


func send_set_loadout(slots: Array) -> void:
	_net.send_msg(NetSerializer.OP_SET_LOADOUT, NetSerializer.Inv.encode_set_loadout(slots))


# =============================================================================
# Flux commitment
# =============================================================================


func handle_flux_commit_state(payload: PackedByteArray) -> void:
	var entries: Array = NetSerializer.Inv.decode_flux_commit_state(payload)
	print("[Net] Received flux commitment: %s" % [entries])
	_net.flux_commit_state_received.emit(entries)


func send_set_flux_commitment(entries: Array) -> void:
	_net.send_msg(
		NetSerializer.OP_SET_FLUX_COMMITMENT, NetSerializer.Inv.encode_set_flux_commitment(entries)
	)


# =============================================================================
# Presets
# =============================================================================


func handle_preset_list(payload: PackedByteArray) -> void:
	var presets: Array = NetSerializer.Inv.decode_preset_list(payload)
	_net.preset_list_received.emit(presets)


func send_save_preset(preset_name: String, slots: Array, commitment: String) -> void:
	_net.send_msg(
		NetSerializer.OP_SAVE_PRESET,
		NetSerializer.Inv.encode_save_preset(preset_name, slots, commitment)
	)


func send_delete_preset(preset_id: int) -> void:
	_net.send_msg(NetSerializer.OP_DELETE_PRESET, NetSerializer.Inv.encode_delete_preset(preset_id))
