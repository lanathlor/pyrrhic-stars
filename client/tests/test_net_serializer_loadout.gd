class_name TestNetSerializerLoadout
extends GdUnitTestSuite

## Tests for NetSerializer loadout and ability catalog encode/decode.

const NetSer := preload("res://scripts/autoload/net_serializer.gd")

var _ser: NetSer
var _catalog_buf: StreamPeerBuffer


func before_test() -> void:
	_ser = NetSer.new()
	add_child(_ser)


func after_test() -> void:
	_ser.queue_free()


# =============================================================================
# encode_set_loadout / decode_loadout_state roundtrip
# =============================================================================


func test_loadout_roundtrip_full() -> void:
	var slots: Array = [
		"mending_surge",
		"mending_beam",
		"vital_bloom",
		"restoration_matrix",
		"life_swap",
		"transfusion"
	]
	var encoded := NetSerializer.Inv.encode_set_loadout(slots)
	var decoded := NetSerializer.Inv.decode_loadout_state(encoded)
	assert_array(decoded).has_size(6)
	for i in 6:
		assert_str(decoded[i]).is_equal(slots[i])


func test_loadout_roundtrip_empty_slots() -> void:
	var slots: Array = ["", "", "", "", "", ""]
	var encoded := NetSerializer.Inv.encode_set_loadout(slots)
	var decoded := NetSerializer.Inv.decode_loadout_state(encoded)
	assert_array(decoded).has_size(6)
	for i in 6:
		assert_str(decoded[i]).is_equal("")


func test_loadout_roundtrip_partial() -> void:
	var slots: Array = ["mending_surge", "", "frost_ward", "", "", "gust_step"]
	var encoded := NetSerializer.Inv.encode_set_loadout(slots)
	var decoded := NetSerializer.Inv.decode_loadout_state(encoded)
	assert_str(decoded[0]).is_equal("mending_surge")
	assert_str(decoded[1]).is_equal("")
	assert_str(decoded[2]).is_equal("frost_ward")
	assert_str(decoded[5]).is_equal("gust_step")


func test_loadout_encode_size() -> void:
	# Each str8 is 1 byte len + N bytes string. 6 slots.
	var slots: Array = ["ab", "cd", "ef", "gh", "ij", "kl"]
	var encoded := NetSerializer.Inv.encode_set_loadout(slots)
	# 6 * (1 + 2) = 18 bytes
	assert_int(encoded.size()).is_equal(18)


func test_loadout_encode_all_empty_size() -> void:
	var slots: Array = ["", "", "", "", "", ""]
	var encoded := NetSerializer.Inv.encode_set_loadout(slots)
	# 6 * (1 + 0) = 6 bytes (just length bytes)
	assert_int(encoded.size()).is_equal(6)


# =============================================================================
# decode_ability_catalog
# =============================================================================


func _build_catalog_entry(entry: Dictionary) -> void:
	var id: String = entry.get("id", "")
	var name: String = entry.get("name", "")
	var school: String = entry.get("school", "")
	var ability_type: String = entry.get("type", "")
	var delivery: String = entry.get("delivery", "")
	var flux_cost: String = entry.get("flux_cost", "")
	var description: String = entry.get("desc", "")
	var cooldown: float = entry.get("cooldown", 0.0)
	var commit_time: float = entry.get("commit_time", 0.0)
	var stats: Dictionary = entry.get("stats", {})
	# Helper: appends one entry to _catalog_buf
	var affinity: String = stats.get("affinity", "")
	var implemented: bool = stats.get("implemented", false)
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
	# 8 x f32: exact stats
	_catalog_buf.put_float(stats.get("flux_amount", 0.0))
	_catalog_buf.put_float(stats.get("base_heal", 0.0))
	_catalog_buf.put_float(stats.get("base_damage", 0.0))
	_catalog_buf.put_float(stats.get("ability_range", 0.0))
	_catalog_buf.put_float(stats.get("gcd", 0.0))
	_catalog_buf.put_float(stats.get("zone_radius", 0.0))
	_catalog_buf.put_float(stats.get("zone_duration", 0.0))
	_catalog_buf.put_float(stats.get("zone_heal_tick", 0.0))
	_catalog_buf.put_u8(1 if stats.get("sustain", false) else 0)


func _put_str8_to_buf(s: String) -> void:
	var b := s.to_utf8_buffer()
	_catalog_buf.put_u8(b.size())
	if b.size() > 0:
		_catalog_buf.put_data(b)


func test_decode_catalog_single_entry() -> void:
	_catalog_buf = StreamPeerBuffer.new()
	_catalog_buf.put_u8(1)  # count
	_build_catalog_entry(
		{
			id = "mending_surge",
			name = "Mending Surge",
			school = "bioarcanotechnic",
			type = "enhancement",
			delivery = "direct",
			flux_cost = "high",
			desc = "Emergency heal.",
			cooldown = 0.0,
			commit_time = 0.4,
			stats =
			{
				implemented = true,
				affinity = "primary",
				flux_amount = 40.0,
				base_heal = 80.0,
				gcd = 0.8,
			},
		}
	)
	var entries := NetSerializer.Inv.decode_ability_catalog(_catalog_buf.data_array)
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
		{
			id = "mending_surge",
			name = "Mending Surge",
			school = "bioarcanotechnic",
			type = "enhancement",
			delivery = "direct",
			flux_cost = "high",
			desc = "Big heal.",
			cooldown = 0.0,
			commit_time = 0.4,
			stats = {implemented = true, affinity = "primary"},
		}
	)
	_build_catalog_entry(
		{
			id = "fireball",
			name = "Fireball",
			school = "fire",
			type = "destruction",
			delivery = "zone",
			flux_cost = "high",
			desc = "AoE explosion.",
			cooldown = 0.0,
			commit_time = 3.0,
			stats = {affinity = "off"},
		}
	)
	var entries := NetSerializer.Inv.decode_ability_catalog(_catalog_buf.data_array)
	assert_int(entries.size()).is_equal(2)
	assert_str(entries[0]["id"]).is_equal("mending_surge")
	assert_str(entries[1]["id"]).is_equal("fireball")
	assert_bool(entries[0]["implemented"]).is_true()
	assert_bool(entries[1]["implemented"]).is_false()
	assert_str(entries[1]["affinity"]).is_equal("off")


func test_decode_catalog_empty() -> void:
	_catalog_buf = StreamPeerBuffer.new()
	_catalog_buf.put_u8(0)  # count = 0
	var entries := NetSerializer.Inv.decode_ability_catalog(_catalog_buf.data_array)
	assert_array(entries).is_empty()


func test_decode_catalog_empty_buffer() -> void:
	var entries := NetSerializer.Inv.decode_ability_catalog(PackedByteArray())
	assert_array(entries).is_empty()


func test_decode_catalog_long_description() -> void:
	_catalog_buf = StreamPeerBuffer.new()
	_catalog_buf.put_u8(1)
	var long_desc := "A".repeat(300)
	_build_catalog_entry(
		{
			id = "test_ability",
			name = "Test Ability",
			school = "frost",
			type = "destruction",
			delivery = "bolt",
			flux_cost = "medium",
			desc = long_desc,
			cooldown = 8.0,
			commit_time = 1.5,
			stats = {affinity = "secondary"},
		}
	)
	var entries := NetSerializer.Inv.decode_ability_catalog(_catalog_buf.data_array)
	assert_int(entries.size()).is_equal(1)
	assert_int(entries[0]["description"].length()).is_equal(300)
	assert_float(entries[0]["cooldown"]).is_equal_approx(8.0, 0.01)
	assert_float(entries[0]["commit_time"]).is_equal_approx(1.5, 0.01)


func test_decode_catalog_unimplemented() -> void:
	_catalog_buf = StreamPeerBuffer.new()
	_catalog_buf.put_u8(1)
	_build_catalog_entry(
		{
			id = "chain_lightning",
			name = "Chain Lightning",
			school = "electricity",
			type = "destruction",
			delivery = "bolt",
			flux_cost = "medium",
			desc = "Bouncing bolt.",
			cooldown = 6.0,
			commit_time = 0.8,
			stats = {affinity = "secondary"},
		}
	)
	var entries := NetSerializer.Inv.decode_ability_catalog(_catalog_buf.data_array)
	assert_bool(entries[0]["implemented"]).is_false()
	assert_str(entries[0]["affinity"]).is_equal("secondary")
