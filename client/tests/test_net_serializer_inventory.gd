class_name TestNetSerializerInventory
extends GdUnitTestSuite

## Tests for NetSerializer inventory codec — encode/decode roundtrips.

const NetSer := preload("res://scripts/autoload/net_serializer.gd")

var _ser: NetSer


func before_test() -> void:
	_ser = NetSer.new()
	add_child(_ser)


func after_test() -> void:
	_ser.queue_free()


# =============================================================================
# encode_equip_item / decode roundtrip
# =============================================================================


func test_encode_equip_item_size() -> void:
	var buf := NetSerializer.Inv.encode_equip_item(42, 3)
	# u32 (4 bytes) + u8 (1 byte) = 5
	assert_int(buf.size()).is_equal(5)


func test_encode_equip_item_values() -> void:
	var buf := NetSerializer.Inv.encode_equip_item(12345, 2)
	var reader := StreamPeerBuffer.new()
	reader.data_array = buf
	assert_int(reader.get_u32()).is_equal(12345)
	assert_int(reader.get_u8()).is_equal(2)


func test_encode_equip_item_zero() -> void:
	var buf := NetSerializer.Inv.encode_equip_item(0, 0)
	var reader := StreamPeerBuffer.new()
	reader.data_array = buf
	assert_int(reader.get_u32()).is_equal(0)
	assert_int(reader.get_u8()).is_equal(0)


# =============================================================================
# encode_unequip_item
# =============================================================================


func test_encode_unequip_item_size() -> void:
	var buf := NetSerializer.Inv.encode_unequip_item(5)
	assert_int(buf.size()).is_equal(1)


func test_encode_unequip_item_value() -> void:
	var buf := NetSerializer.Inv.encode_unequip_item(3)
	assert_int(buf[0]).is_equal(3)


# =============================================================================
# decode_inventory_state — build a binary payload matching server format
# =============================================================================


func _build_inventory_payload(equipped: Array, bag: Array, stats: Array) -> PackedByteArray:
	var buf := StreamPeerBuffer.new()
	# equipped count
	buf.put_u8(equipped.size())
	for item in equipped:
		_write_item(buf, item)
	# bag count
	buf.put_u8(bag.size())
	for item in bag:
		_write_item(buf, item)
	# 6 stat floats
	for s in stats:
		buf.put_float(s)
	return buf.data_array


func _write_item(buf: StreamPeerBuffer, item: Dictionary) -> void:
	buf.put_u8(item.get("slot_id", 0))
	buf.put_u32(item.get("item_id", 0))
	# def_id as str8
	var def_bytes := (item.get("def_id", "") as String).to_utf8_buffer()
	buf.put_u8(def_bytes.size())
	if def_bytes.size() > 0:
		buf.put_data(def_bytes)
	# ilvl as u16
	buf.put_u16(item.get("ilvl", 1))
	# name as str8
	var name_bytes := (item.get("name", "") as String).to_utf8_buffer()
	buf.put_u8(name_bytes.size())
	if name_bytes.size() > 0:
		buf.put_data(name_bytes)
	# stat lines: count + per line (u8 stat + f32 value)
	var slines: Array = item.get("stat_lines", [])
	buf.put_u8(slines.size())
	for sl in slines:
		buf.put_u8(sl["stat"])
		buf.put_float(sl["value"])


# --- Tests ---


func test_decode_empty_inventory() -> void:
	var payload := _build_inventory_payload([], [], [0.0, 0.0, 0.0, 0.0, 0.0, 0.0])
	var result: Dictionary = NetSerializer.Inv.decode_inventory_state(payload)
	assert_array(result["equipped"]).is_empty()
	assert_array(result["bag"]).is_empty()
	for key in ["hull", "output", "plating", "tempo", "identity", "mastery"]:
		assert_float(result["stats"][key]).is_equal(0.0)


func test_decode_single_equipped_item() -> void:
	var item := {
		"slot_id": 0,
		"item_id": 42,
		"def_id": "frame_mk1",
		"ilvl": 1,
		"name": "Frame MK1",
		"stat_lines": [{"stat": 0, "value": 10.0}],
	}
	var payload := _build_inventory_payload([item], [], [10.0, 0.0, 0.0, 0.0, 0.0, 0.0])
	var result: Dictionary = NetSerializer.Inv.decode_inventory_state(payload)
	assert_int(result["equipped"].size()).is_equal(1)
	var decoded: Dictionary = result["equipped"][0]
	assert_int(decoded["slot_id"]).is_equal(0)
	assert_int(decoded["item_id"]).is_equal(42)
	assert_str(decoded["def_id"]).is_equal("frame_mk1")
	assert_int(decoded["ilvl"]).is_equal(1)
	assert_str(decoded["name"]).is_equal("Frame MK1")
	assert_int(decoded["stat_lines"].size()).is_equal(1)
	assert_int(decoded["stat_lines"][0]["stat"]).is_equal(0)
	assert_float(decoded["stat_lines"][0]["value"]).is_equal_approx(10.0, 0.01)


func test_decode_multiple_stat_lines() -> void:
	var item := {
		"slot_id": 2,
		"item_id": 99,
		"def_id": "weapon_alpha",
		"ilvl": 3,
		"name": "Alpha Weapon",
		"stat_lines":
		[
			{"stat": 1, "value": 15.0},
			{"stat": 3, "value": 8.0},
			{"stat": 5, "value": 3.5},
		],
	}
	var payload := _build_inventory_payload([item], [], [0.0, 15.0, 0.0, 8.0, 0.0, 3.5])
	var result: Dictionary = NetSerializer.Inv.decode_inventory_state(payload)
	var decoded: Dictionary = result["equipped"][0]
	assert_int(decoded["stat_lines"].size()).is_equal(3)
	assert_float(decoded["stat_lines"][1]["value"]).is_equal_approx(8.0, 0.01)
	assert_int(decoded["stat_lines"][2]["stat"]).is_equal(5)


func test_decode_bag_items() -> void:
	var bag_item := {
		"slot_id": 4,
		"item_id": 200,
		"def_id": "aug_x",
		"ilvl": 2,
		"name": "Augment X",
		"stat_lines": [{"stat": 4, "value": 5.0}],
	}
	var payload := _build_inventory_payload([], [bag_item], [0.0, 0.0, 0.0, 0.0, 5.0, 0.0])
	var result: Dictionary = NetSerializer.Inv.decode_inventory_state(payload)
	assert_array(result["equipped"]).is_empty()
	assert_int(result["bag"].size()).is_equal(1)
	assert_str(result["bag"][0]["name"]).is_equal("Augment X")


func test_decode_computed_stats() -> void:
	var payload := _build_inventory_payload([], [], [50.0, 12.5, 8.0, 3.0, 1.0, 0.5])
	var result: Dictionary = NetSerializer.Inv.decode_inventory_state(payload)
	assert_float(result["stats"]["hull"]).is_equal_approx(50.0, 0.01)
	assert_float(result["stats"]["output"]).is_equal_approx(12.5, 0.01)
	assert_float(result["stats"]["plating"]).is_equal_approx(8.0, 0.01)
	assert_float(result["stats"]["tempo"]).is_equal_approx(3.0, 0.01)
	assert_float(result["stats"]["identity"]).is_equal_approx(1.0, 0.01)
	assert_float(result["stats"]["mastery"]).is_equal_approx(0.5, 0.01)


func test_decode_mixed_equipped_and_bag() -> void:
	var eq := {
		"slot_id": 1,
		"item_id": 10,
		"def_id": "core_a",
		"ilvl": 1,
		"name": "Core A",
		"stat_lines": [{"stat": 0, "value": 20.0}, {"stat": 2, "value": 5.0}],
	}
	var bag := {
		"slot_id": 1,
		"item_id": 11,
		"def_id": "core_b",
		"ilvl": 2,
		"name": "Core B",
		"stat_lines": [{"stat": 0, "value": 30.0}, {"stat": 2, "value": 8.0}],
	}
	var payload := _build_inventory_payload([eq], [bag], [20.0, 0.0, 5.0, 0.0, 0.0, 0.0])
	var result: Dictionary = NetSerializer.Inv.decode_inventory_state(payload)
	assert_int(result["equipped"].size()).is_equal(1)
	assert_int(result["bag"].size()).is_equal(1)
	assert_str(result["equipped"][0]["def_id"]).is_equal("core_a")
	assert_str(result["bag"][0]["def_id"]).is_equal("core_b")


func test_decode_high_ilvl() -> void:
	var item := {
		"slot_id": 5,
		"item_id": 999,
		"def_id": "module_epic",
		"ilvl": 50,
		"name": "Epic Module",
		"stat_lines": [],
	}
	var payload := _build_inventory_payload([item], [], [0.0, 0.0, 0.0, 0.0, 0.0, 0.0])
	var result: Dictionary = NetSerializer.Inv.decode_inventory_state(payload)
	assert_int(result["equipped"][0]["ilvl"]).is_equal(50)
