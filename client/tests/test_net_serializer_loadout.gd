class_name TestNetSerializerLoadout
extends GdUnitTestSuite

## Tests for NetSerializer loadout and ability catalog encode/decode.

const NetSer := preload("res://scripts/autoload/net_serializer.gd")

var _ser: NetSer


func before_test() -> void:
	_ser = NetSer.new()
	add_child(_ser)


func after_test() -> void:
	_ser.queue_free()


# =============================================================================
# encode_set_loadout / decode_loadout_state roundtrip
# =============================================================================


func test_loadout_roundtrip_full() -> void:
	var slots: Array = ["mending_surge", "mending_beam", "vital_bloom", "restoration_matrix", "life_swap", "transfusion"]
	var encoded := _ser.encode_set_loadout(slots)
	var decoded := _ser.decode_loadout_state(encoded)
	assert_array(decoded).has_size(6)
	for i in 6:
		assert_str(decoded[i]).is_equal(slots[i])


func test_loadout_roundtrip_empty_slots() -> void:
	var slots: Array = ["", "", "", "", "", ""]
	var encoded := _ser.encode_set_loadout(slots)
	var decoded := _ser.decode_loadout_state(encoded)
	assert_array(decoded).has_size(6)
	for i in 6:
		assert_str(decoded[i]).is_equal("")


func test_loadout_roundtrip_partial() -> void:
	var slots: Array = ["mending_surge", "", "frost_ward", "", "", "gust_step"]
	var encoded := _ser.encode_set_loadout(slots)
	var decoded := _ser.decode_loadout_state(encoded)
	assert_str(decoded[0]).is_equal("mending_surge")
	assert_str(decoded[1]).is_equal("")
	assert_str(decoded[2]).is_equal("frost_ward")
	assert_str(decoded[5]).is_equal("gust_step")


func test_loadout_encode_size() -> void:
	# Each str8 is 1 byte len + N bytes string. 6 slots.
	var slots: Array = ["ab", "cd", "ef", "gh", "ij", "kl"]
	var encoded := _ser.encode_set_loadout(slots)
	# 6 * (1 + 2) = 18 bytes
	assert_int(encoded.size()).is_equal(18)


func test_loadout_encode_all_empty_size() -> void:
	var slots: Array = ["", "", "", "", "", ""]
	var encoded := _ser.encode_set_loadout(slots)
	# 6 * (1 + 0) = 6 bytes (just length bytes)
	assert_int(encoded.size()).is_equal(6)


# =============================================================================
# decode_ability_catalog
# =============================================================================


func _build_catalog_entry(
	id: String, name: String, school: String, ability_type: String,
	delivery: String, flux_cost: String, description: String,
	cooldown: float, commit_time: float, implemented: bool, affinity: String,
	flux_amount: float = 0.0, base_heal: float = 0.0, base_damage: float = 0.0,
	ability_range: float = 0.0, gcd: float = 0.0, commit_time: float = 0.0,
	zone_radius: float = 0.0, zone_duration: float = 0.0, zone_heal_tick: float = 0.0,
	sustain: bool = false,
) -> void:
	# Helper: appends one entry to _catalog_buf
	_put_str8_to_buf(id)
	_put_str8_to_buf(name)
	_put_str8_to_buf(school)
	_put_str8_to_buf(ability_type)
	_put_str8_to_buf(delivery)
	_put_str8_to_buf(flux_cost)
	# description: str16
	var desc_bytes := description.to_utf8_buffer()
	_catalog_buf.put_u16(desc_bytes.size())
	if desc_bytes.size() > 0:
		_catalog_buf.put_data(desc_bytes)
	_catalog_buf.put_float(cooldown)
	_catalog_buf.put_float(commit_time)
	_catalog_buf.put_u8(1 if implemented else 0)
	_put_str8_to_buf(affinity)
	# 9 x f32: exact stats
	_catalog_buf.put_float(flux_amount)
	_catalog_buf.put_float(base_heal)
	_catalog_buf.put_float(base_damage)
	_catalog_buf.put_float(ability_range)
	_catalog_buf.put_float(gcd)
	_catalog_buf.put_float(commit_time)
	_catalog_buf.put_float(zone_radius)
	_catalog_buf.put_float(zone_duration)
	_catalog_buf.put_float(zone_heal_tick)
	_catalog_buf.put_u8(1 if sustain else 0)


var _catalog_buf: StreamPeerBuffer


func _put_str8_to_buf(s: String) -> void:
	var b := s.to_utf8_buffer()
	_catalog_buf.put_u8(b.size())
	if b.size() > 0:
		_catalog_buf.put_data(b)


func test_decode_catalog_single_entry() -> void:
	_catalog_buf = StreamPeerBuffer.new()
	_catalog_buf.put_u8(1)  # count
	_build_catalog_entry(
		"mending_surge", "Mending Surge", "bioarcanotechnic", "enhancement",
		"direct", "high", "Emergency heal.", 0.0, 0.4, true, "primary",
		40.0, 80.0, 0.0, 0.0, 0.8
	)
	var entries := _ser.decode_ability_catalog(_catalog_buf.data_array)
	assert_int(entries.size()).is_equal(1)
	var e: Dictionary = entries[0]
	assert_str(e["id"]).is_equal("mending_surge")
	assert_str(e["name"]).is_equal("Mending Surge")
	assert_str(e["school"]).is_equal("bioarcanotechnic")
	assert_str(e["ability_type"]).is_equal("enhancement")
	assert_str(e["delivery"]).is_equal("direct")
	assert_str(e["flux_cost"]).is_equal("high")
	assert_str(e["description"]).is_equal("Emergency heal.")
	assert_float(e["cooldown"]).is_equal_approx(0.0, 0.01)
	assert_float(e["commit_time"]).is_equal_approx(0.4, 0.01)
	assert_bool(e["implemented"]).is_true()
	assert_str(e["affinity"]).is_equal("primary")
	assert_float(e["flux_amount"]).is_equal_approx(40.0, 0.01)
	assert_float(e["base_heal"]).is_equal_approx(80.0, 0.01)
	assert_float(e["base_damage"]).is_equal_approx(0.0, 0.01)
	assert_float(e["gcd"]).is_equal_approx(0.8, 0.01)


func test_decode_catalog_multiple_entries() -> void:
	_catalog_buf = StreamPeerBuffer.new()
	_catalog_buf.put_u8(2)  # count
	_build_catalog_entry(
		"mending_surge", "Mending Surge", "bioarcanotechnic", "enhancement",
		"direct", "high", "Big heal.", 0.0, 0.4, true, "primary"
	)
	_build_catalog_entry(
		"fireball", "Fireball", "fire", "destruction",
		"zone", "high", "AoE explosion.", 0.0, 3.0, false, "off"
	)
	var entries := _ser.decode_ability_catalog(_catalog_buf.data_array)
	assert_int(entries.size()).is_equal(2)
	assert_str(entries[0]["id"]).is_equal("mending_surge")
	assert_str(entries[1]["id"]).is_equal("fireball")
	assert_bool(entries[0]["implemented"]).is_true()
	assert_bool(entries[1]["implemented"]).is_false()
	assert_str(entries[1]["affinity"]).is_equal("off")


func test_decode_catalog_empty() -> void:
	_catalog_buf = StreamPeerBuffer.new()
	_catalog_buf.put_u8(0)  # count = 0
	var entries := _ser.decode_ability_catalog(_catalog_buf.data_array)
	assert_array(entries).is_empty()


func test_decode_catalog_empty_buffer() -> void:
	var entries := _ser.decode_ability_catalog(PackedByteArray())
	assert_array(entries).is_empty()


func test_decode_catalog_long_description() -> void:
	_catalog_buf = StreamPeerBuffer.new()
	_catalog_buf.put_u8(1)
	var long_desc := "A" .repeat(300)
	_build_catalog_entry(
		"test_ability", "Test Ability", "frost", "destruction",
		"bolt", "medium", long_desc, 8.0, 1.5, false, "secondary"
	)
	var entries := _ser.decode_ability_catalog(_catalog_buf.data_array)
	assert_int(entries.size()).is_equal(1)
	assert_int(entries[0]["description"].length()).is_equal(300)
	assert_float(entries[0]["cooldown"]).is_equal_approx(8.0, 0.01)
	assert_float(entries[0]["commit_time"]).is_equal_approx(1.5, 0.01)


func test_decode_catalog_unimplemented() -> void:
	_catalog_buf = StreamPeerBuffer.new()
	_catalog_buf.put_u8(1)
	_build_catalog_entry(
		"chain_lightning", "Chain Lightning", "electricity", "destruction",
		"bolt", "medium", "Bouncing bolt.", 6.0, 0.8, false, "secondary"
	)
	var entries := _ser.decode_ability_catalog(_catalog_buf.data_array)
	assert_bool(entries[0]["implemented"]).is_false()
	assert_str(entries[0]["affinity"]).is_equal("secondary")
