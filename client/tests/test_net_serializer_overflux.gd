class_name TestNetSerializerOverflux
extends GdUnitTestSuite

## Tests for NetSerializer overflux codec: encode/decode roundtrips.

const NetSer := preload("res://scripts/autoload/net_serializer.gd")
const H := preload("res://scripts/autoload/net_serialize_helpers.gd")

var _ser: NetSer


func before_test() -> void:
	_ser = NetSer.new()
	add_child(_ser)


func after_test() -> void:
	_ser.queue_free()


# =============================================================================
# encode_overflux_conditions
# =============================================================================


func test_encode_overflux_conditions_empty() -> void:
	var buf := NetSerializer.Char.encode_overflux_conditions([])
	assert_int(buf.size()).is_equal(1)
	assert_int(buf[0]).is_equal(0x00)


func test_encode_overflux_conditions_single() -> void:
	var conditions := [{"id": "enemy_hp", "rank": 3}]
	var buf := NetSerializer.Char.encode_overflux_conditions(conditions)
	var reader := StreamPeerBuffer.new()
	reader.data_array = buf
	# count
	assert_int(reader.get_u8()).is_equal(1)
	# id as str8: length=8 then "enemy_hp"
	var id_len := reader.get_u8()
	assert_int(id_len).is_equal(8)
	var id_data := reader.get_data(id_len)
	var id_str := (id_data[1] as PackedByteArray).get_string_from_utf8()
	assert_str(id_str).is_equal("enemy_hp")
	# rank
	assert_int(reader.get_u8()).is_equal(3)


func test_encode_overflux_conditions_multiple() -> void:
	var conditions := [
		{"id": "enemy_hp", "rank": 2},
		{"id": "damage", "rank": 5},
	]
	var buf := NetSerializer.Char.encode_overflux_conditions(conditions)
	var reader := StreamPeerBuffer.new()
	reader.data_array = buf
	# count
	assert_int(reader.get_u8()).is_equal(2)
	# first condition
	var id1_len := reader.get_u8()
	assert_int(id1_len).is_equal(8)
	var id1_data := reader.get_data(id1_len)
	var id1_str := (id1_data[1] as PackedByteArray).get_string_from_utf8()
	assert_str(id1_str).is_equal("enemy_hp")
	assert_int(reader.get_u8()).is_equal(2)
	# second condition
	var id2_len := reader.get_u8()
	assert_int(id2_len).is_equal(6)
	var id2_data := reader.get_data(id2_len)
	var id2_str := (id2_data[1] as PackedByteArray).get_string_from_utf8()
	assert_str(id2_str).is_equal("damage")
	assert_int(reader.get_u8()).is_equal(5)


# =============================================================================
# decode_overflux_state
# =============================================================================


func test_decode_overflux_state_empty() -> void:
	var buf := StreamPeerBuffer.new()
	buf.put_u16(0)  # total_score = 0
	buf.put_u8(0)  # count = 0
	var result: Dictionary = NetSerializer.Char.decode_overflux_state(buf.data_array)
	assert_int(result["total_score"]).is_equal(0)
	assert_array(result["conditions"]).is_empty()


func test_decode_overflux_state_with_conditions() -> void:
	var buf := StreamPeerBuffer.new()
	buf.put_u16(12)  # total_score = 12
	buf.put_u8(2)  # count = 2
	# condition 1: "enemy_hp" rank 3
	H.put_str8(buf, "enemy_hp")
	buf.put_u8(3)
	# condition 2: "damage" rank 1
	H.put_str8(buf, "damage")
	buf.put_u8(1)

	var result: Dictionary = NetSerializer.Char.decode_overflux_state(buf.data_array)
	assert_int(result["total_score"]).is_equal(12)
	assert_int(result["conditions"].size()).is_equal(2)
	assert_str(result["conditions"][0]["id"]).is_equal("enemy_hp")
	assert_int(result["conditions"][0]["rank"]).is_equal(3)
	assert_str(result["conditions"][1]["id"]).is_equal("damage")
	assert_int(result["conditions"][1]["rank"]).is_equal(1)


# =============================================================================
# decode_instance_join_prompt
# =============================================================================


func test_decode_instance_join_prompt() -> void:
	var buf := StreamPeerBuffer.new()
	H.put_str8(buf, "arena_boss")  # zone name
	H.put_str8(buf, "PlayerOne")  # leader name
	buf.put_u16(20)  # total_score
	buf.put_u8(1)  # 1 condition
	H.put_str8(buf, "enemy_hp")
	buf.put_u8(5)

	var result: Dictionary = NetSerializer.Char.decode_instance_join_prompt(buf.data_array)
	assert_str(result["zone_name"]).is_equal("arena_boss")
	assert_str(result["leader_name"]).is_equal("PlayerOne")
	assert_int(result["total_score"]).is_equal(20)
	assert_int(result["conditions"].size()).is_equal(1)
	assert_str(result["conditions"][0]["id"]).is_equal("enemy_hp")
	assert_int(result["conditions"][0]["rank"]).is_equal(5)


# =============================================================================
# encode_instance_join_reply
# =============================================================================


func test_encode_instance_join_reply_accept() -> void:
	var buf := NetSerializer.Char.encode_instance_join_reply(true)
	assert_int(buf.size()).is_equal(1)
	assert_int(buf[0]).is_equal(0x01)


func test_encode_instance_join_reply_decline() -> void:
	var buf := NetSerializer.Char.encode_instance_join_reply(false)
	assert_int(buf.size()).is_equal(1)
	assert_int(buf[0]).is_equal(0x00)


# =============================================================================
# Roundtrip: encode conditions -> wrap in state payload -> decode
# =============================================================================


func test_encode_decode_roundtrip() -> void:
	var original := [
		{"id": "enemy_hp", "rank": 3},
		{"id": "damage", "rank": 2},
	]
	# Encode the conditions
	var encoded := NetSerializer.Char.encode_overflux_conditions(original)

	# Build a state payload: [total_score:u16 LE] + encoded conditions bytes
	# The encoded bytes already contain [count:u8][per: str8 + rank:u8]
	var state_buf := StreamPeerBuffer.new()
	state_buf.put_u16(99)  # arbitrary total_score
	state_buf.put_data(encoded)

	# Decode as overflux state
	var result: Dictionary = NetSerializer.Char.decode_overflux_state(state_buf.data_array)
	assert_int(result["total_score"]).is_equal(99)
	assert_int(result["conditions"].size()).is_equal(2)
	assert_str(result["conditions"][0]["id"]).is_equal("enemy_hp")
	assert_int(result["conditions"][0]["rank"]).is_equal(3)
	assert_str(result["conditions"][1]["id"]).is_equal("damage")
	assert_int(result["conditions"][1]["rank"]).is_equal(2)
