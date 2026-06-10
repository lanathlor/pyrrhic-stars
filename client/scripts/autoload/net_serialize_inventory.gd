class_name NetSerializeInventory
## Inventory, loadout, flux commitment, preset, and ability catalog codecs.
## All methods are static — called via forwarding stubs in NetSerializer.

const H := preload("res://scripts/autoload/net_serialize_helpers.gd")

# =============================================================================
# Inventory codec
# =============================================================================


## Decode an OpInventoryState payload into a Dictionary.
## Format: [equip_count:u8] per: item... [bag_count:u8] per: item... [6x stat:f32]
static func decode_inventory_state(data: PackedByteArray) -> Dictionary:
	var buf := StreamPeerBuffer.new()
	buf.data_array = data

	# Equipped items
	var equip_count := buf.get_u8()
	var equipped: Array[Dictionary] = []
	for i in range(equip_count):
		equipped.append(_decode_inventory_item(buf))

	# Bag items
	var bag_count := buf.get_u8()
	var bag: Array[Dictionary] = []
	for i in range(bag_count):
		bag.append(_decode_inventory_item(buf))

	# Computed stats (6 floats)
	var stats := {
		"hull": buf.get_float(),
		"output": buf.get_float(),
		"plating": buf.get_float(),
		"tempo": buf.get_float(),
		"identity": buf.get_float(),
		"mastery": buf.get_float(),
	}

	return {"equipped": equipped, "bag": bag, "stats": stats}


static func _decode_inventory_item(buf: StreamPeerBuffer) -> Dictionary:
	var slot_id := buf.get_u8()
	var item_id := buf.get_u32()
	var def_id := H.get_str8(buf)
	var ilvl := buf.get_u16()
	var item_name := H.get_str8(buf)
	var stat_count := buf.get_u8()
	var stat_lines: Array[Dictionary] = []
	for j in range(stat_count):
		var stat := buf.get_u8()
		var value := buf.get_float()
		stat_lines.append({"stat": stat, "value": value})
	return {
		"slot_id": slot_id,
		"item_id": item_id,
		"def_id": def_id,
		"ilvl": ilvl,
		"name": item_name,
		"stat_lines": stat_lines,
	}


## Encode OpEquipItem payload: [item_id:u32 LE][slot_id:u8]
static func encode_equip_item(item_id: int, slot_id: int) -> PackedByteArray:
	var buf := StreamPeerBuffer.new()
	buf.put_u32(item_id)
	buf.put_u8(slot_id)
	return buf.data_array


## Encode OpUnequipItem payload: [slot_id:u8]
static func encode_unequip_item(slot_id: int) -> PackedByteArray:
	return PackedByteArray([slot_id])


# =============================================================================
# Loadout / Ability catalog
# =============================================================================


## Encode a loadout change: 6 ability ID strings as str8.
static func encode_set_loadout(slots: Array) -> PackedByteArray:
	var buf := StreamPeerBuffer.new()
	for i in 6:
		var id: String = slots[i] if i < slots.size() else ""
		H.put_str8(buf, id)
	return buf.data_array


## Decode loadout state: 6 str8 ability IDs.
static func decode_loadout_state(data: PackedByteArray) -> Array:
	var buf := StreamPeerBuffer.new()
	buf.data_array = data
	var slots: Array = []
	for i in 6:
		if buf.get_position() >= buf.get_size():
			slots.append("")
			continue
		slots.append(H.get_str8(buf))
	return slots


## Encode flux commitment: [count:u8][per: school:str8 + pct:u8]
static func encode_set_flux_commitment(entries: Array) -> PackedByteArray:
	var buf := StreamPeerBuffer.new()
	buf.put_u8(entries.size())
	for entry in entries:
		H.put_str8(buf, entry["school"])
		buf.put_u8(entry["percentage"])
	return buf.data_array


## Decode flux commitment state: [count:u8][per: school:str8 + pct:u8]
static func decode_flux_commit_state(data: PackedByteArray) -> Array:
	var buf := StreamPeerBuffer.new()
	buf.data_array = data
	var entries: Array = []
	if buf.get_size() < 1:
		return entries
	var count: int = buf.get_u8()
	for _i in count:
		if buf.get_position() >= buf.get_size():
			break
		var school: String = H.get_str8(buf)
		var pct: int = buf.get_u8() if buf.get_position() < buf.get_size() else 0
		entries.append({"school": school, "percentage": pct})
	return entries


## Encode save-preset: [name:str8][6x ability_id:str8][commitment:str8]
static func encode_save_preset(
	preset_name: String, slots: Array, commitment: String
) -> PackedByteArray:
	var buf := StreamPeerBuffer.new()
	H.put_str8(buf, preset_name)
	for i in 6:
		var id: String = slots[i] if i < slots.size() else ""
		H.put_str8(buf, id)
	H.put_str8(buf, commitment)
	return buf.data_array


## Encode delete-preset: [preset_id:u32 LE]
static func encode_delete_preset(preset_id: int) -> PackedByteArray:
	var buf := StreamPeerBuffer.new()
	buf.put_u32(preset_id)
	return buf.data_array


## Decode preset list: [count:u8][per: id:u32, name:str8, 6x slot:str8, commitment:str8]
static func decode_preset_list(data: PackedByteArray) -> Array:
	var buf := StreamPeerBuffer.new()
	buf.data_array = data
	var presets: Array = []
	if buf.get_size() < 1:
		return presets
	var count: int = buf.get_u8()
	for _i in count:
		if buf.get_position() >= buf.get_size():
			break
		var preset: Dictionary = {}
		preset["id"] = buf.get_u32()
		preset["name"] = H.get_str8(buf)
		var slots: Array = []
		for _j in 6:
			slots.append(H.get_str8(buf))
		preset["slots"] = slots
		preset["commitment"] = H.get_str8(buf)
		presets.append(preset)
	return presets


## Decode ability catalog: [count:u8][per: id:str8, name:str8, school:str8,
## ability_type:str8, delivery:str8, flux_cost:str8, description:str16,
## cooldown:f32, commit_time:f32, implemented:u8, affinity:str8,
## flux_amount:f32, base_heal:f32, base_damage:f32, range:f32, gcd:f32,
## zone_radius:f32, zone_duration:f32, zone_heal_tick:f32]
static func decode_ability_catalog(data: PackedByteArray) -> Array:
	var entries: Array = []
	if data.size() < 1:
		return entries
	var buf := StreamPeerBuffer.new()
	buf.data_array = data
	var count: int = buf.get_u8()
	for _i in count:
		entries.append(_decode_ability_entry(buf))
	return entries


static func _decode_ability_entry(buf: StreamPeerBuffer) -> Dictionary:
	var entry: Dictionary = {}
	for key in ["id", "name", "school", "ability_type", "delivery", "flux_cost"]:
		if buf.get_position() >= buf.get_size():
			break
		entry[key] = H.get_str8(buf)
	entry["description"] = _decode_str16(buf)
	if buf.get_position() + 4 <= buf.get_size():
		entry["cooldown"] = buf.get_float()
	if buf.get_position() + 4 <= buf.get_size():
		entry["commit_time"] = buf.get_float()
	if buf.get_position() < buf.get_size():
		entry["implemented"] = buf.get_u8() != 0
	if buf.get_position() < buf.get_size():
		entry["affinity"] = H.get_str8(buf)
	else:
		entry["affinity"] = "off"
	for key in [
		"flux_amount",
		"base_heal",
		"base_damage",
		"range",
		"gcd",
		"zone_radius",
		"zone_duration",
		"zone_heal_tick"
	]:
		entry[key] = buf.get_float() if buf.get_position() + 4 <= buf.get_size() else 0.0
	entry["sustain"] = buf.get_u8() != 0 if buf.get_position() < buf.get_size() else false
	return entry


static func _decode_str16(buf: StreamPeerBuffer) -> String:
	if buf.get_position() + 2 > buf.get_size():
		return ""
	var dlen: int = buf.get_u16()
	if dlen == 0 or buf.get_position() + dlen > buf.get_size():
		return ""
	var result := buf.get_data(dlen)
	if result[0] == OK:
		return (result[1] as PackedByteArray).get_string_from_utf8()
	return ""


# =============================================================================
# Merchant codecs (server -> client)
# =============================================================================


static func decode_merchant_state(payload: PackedByteArray) -> Dictionary:
	var off := 0
	var balance := payload.decode_u32(off)
	off += 4
	var watermark := payload.decode_u16(off)
	off += 2
	var season := payload.decode_u16(off)
	off += 2
	var max_score := payload.decode_u16(off)
	off += 2
	var tier_count := payload[off]
	off += 1
	var tiers := []
	for _i in tier_count:
		var ilvl := payload[off]
		off += 1
		var unlocked := payload[off] == 1
		off += 1
		var price := payload.decode_u32(off)
		off += 4
		var item_count := payload[off]
		off += 1
		var items := []
		for _j in item_count:
			var def_len := payload[off]
			off += 1
			var def_id := payload.slice(off, off + def_len).get_string_from_utf8()
			off += def_len
			var name_len := payload[off]
			off += 1
			var item_name := payload.slice(off, off + name_len).get_string_from_utf8()
			off += name_len
			var slot_id := payload[off]
			off += 1
			var stat_count := payload[off]
			off += 1
			var stats := []
			for _k in stat_count:
				var stat_id := payload[off]
				off += 1
				var value := payload.decode_float(off)
				off += 4
				stats.append({"stat": stat_id, "value": value})
			items.append({"def_id": def_id, "name": item_name, "slot": slot_id, "stats": stats})
		tiers.append({"ilvl": ilvl, "unlocked": unlocked, "price": price, "items": items})
	return {
		"balance": balance,
		"watermark": watermark,
		"season": season,
		"max_score": max_score,
		"tiers": tiers,
	}


static func decode_merchant_buy_result(payload: PackedByteArray) -> Dictionary:
	var success := payload[0] == 1
	var new_balance := payload.decode_u32(1)
	var item_id := payload.decode_u32(5)
	var err_len := payload[9]
	var err_msg := ""
	if err_len > 0:
		err_msg = payload.slice(10, 10 + err_len).get_string_from_utf8()
	return {"success": success, "new_balance": new_balance, "item_id": item_id, "error": err_msg}


static func decode_scrip_award(payload: PackedByteArray) -> Dictionary:
	var amount := payload.decode_u16(0)
	var new_balance := payload.decode_u32(2)
	return {"amount": amount, "new_balance": new_balance}
