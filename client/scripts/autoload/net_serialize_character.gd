class_name NetSerializeCharacter
## Character, zone, lobby, and group codecs.
## All methods are static — called via forwarding stubs in NetSerializer.

const H := preload("res://scripts/autoload/net_serialize_helpers.gd")

# =============================================================================
# Zone management
# =============================================================================


static func encode_join_zone(zone_id: String) -> PackedByteArray:
	return zone_id.to_utf8_buffer()


static func decode_zone_joined(data: PackedByteArray) -> Dictionary:
	if data.size() < 3:
		return {}
	var buf := StreamPeerBuffer.new()
	buf.data_array = data
	buf.big_endian = true
	return {
		"peer_id": buf.get_u16(),
		"is_host": buf.get_u8() == 1,
	}


static func decode_peer_id(data: PackedByteArray) -> int:
	if data.size() < 2:
		return 0
	var buf := StreamPeerBuffer.new()
	buf.data_array = data
	buf.big_endian = true
	return buf.get_u16()


# =============================================================================
# Username
# =============================================================================


## Format: [name_len:u8][name:...]
static func encode_username(username: String) -> PackedByteArray:
	var buf := StreamPeerBuffer.new()
	var name_bytes := username.to_utf8_buffer()
	buf.put_u8(name_bytes.size())
	if name_bytes.size() > 0:
		buf.put_data(name_bytes)
	return buf.data_array


# =============================================================================
# Zone transfer (server -> client)
# =============================================================================


## Format: [zone_type:u8][new_peer_id:u16 BE]
static func decode_zone_transfer(data: PackedByteArray) -> Dictionary:
	if data.size() < 3:
		return {}
	var zone_type := data[0]
	var buf := StreamPeerBuffer.new()
	buf.data_array = data.slice(1)
	buf.big_endian = true
	var new_peer_id := buf.get_u16()
	return {"zone_type": zone_type, "new_peer_id": new_peer_id}


# =============================================================================
# Character state (server -> client)
# =============================================================================


## Format: [charID:u32 LE][class_len:u8][class:...][name_len:u8][name:...]
##         [pos_x:f32 LE][pos_y:f32 LE][pos_z:f32 LE][rot_y:f32 LE]
static func decode_character_state(data: PackedByteArray) -> Dictionary:
	var empty := {
		"char_id": 0, "class_name": "", "char_name": "", "position": Vector3.ZERO, "rot_y": 0.0
	}
	if data.size() < 5:
		return empty
	var buf := StreamPeerBuffer.new()
	buf.data_array = data
	var char_id := buf.get_u32()
	var class_name_str := H.get_str8(buf)
	var char_name := H.get_str8(buf)
	var px := buf.get_float()
	var py := buf.get_float()
	var pz := buf.get_float()
	var ry := buf.get_float()
	return {
		"char_id": char_id,
		"class_name": class_name_str,
		"char_name": char_name,
		"position": Vector3(px, py, pz),
		"rot_y": ry
	}


# =============================================================================
# Character list (server -> client)
# =============================================================================


## Format: [username_len:u8][username:...]
##         [count:u8]
##           per char: [charID:u32 LE][class_len:u8][class:...][name_len:u8][name:...]
##                     [pos_x:f32][pos_y:f32][pos_z:f32][rot_y:f32]
##         [last_char_id:u32 LE]
static func decode_character_list(data: PackedByteArray) -> Dictionary:
	var result := {"username": "", "characters": [], "last_char_id": 0}
	if data.size() < 1:
		return result
	var buf := StreamPeerBuffer.new()
	buf.data_array = data
	result.username = H.get_str8(buf)
	var count := buf.get_u8()
	for i in range(count):
		var char_id := buf.get_u32()
		var cls := H.get_str8(buf)
		var char_name := H.get_str8(buf)
		var px := buf.get_float()
		var py := buf.get_float()
		var pz := buf.get_float()
		var ry := buf.get_float()
		(
			result
			. characters
			. append(
				{
					"char_id": char_id,
					"class_name": cls,
					"char_name": char_name,
					"position": Vector3(px, py, pz),
					"rot_y": ry,
				}
			)
		)
	result.last_char_id = buf.get_u32()
	return result


## Encode character selection by ID: [charID:u32 LE]
static func encode_select_character(char_id: int) -> PackedByteArray:
	var buf := StreamPeerBuffer.new()
	buf.put_u32(char_id)
	return buf.data_array


## Encode create character: [class_len:u8][class:...][name_len:u8][name:...]
static func encode_create_character(class_name_str: String, char_name: String) -> PackedByteArray:
	var buf := StreamPeerBuffer.new()
	H.put_str8(buf, class_name_str)
	H.put_str8(buf, char_name)
	return buf.data_array


## Decode character error: [error_code:u8][msg_len:u8][msg:...]
static func decode_character_error(data: PackedByteArray) -> Dictionary:
	if data.size() < 2:
		return {"error_code": 0, "message": "Unknown error"}
	var buf := StreamPeerBuffer.new()
	buf.data_array = data
	var error_code := buf.get_u8()
	var msg := H.get_str8(buf)
	return {"error_code": error_code, "message": msg}


# =============================================================================
# Lobby state (server -> client)
# =============================================================================


## Format: [player_count:u8] per player: [peer_id:u16][class_len:u8][class:...][ready:u8]
static func decode_lobby_state(data: PackedByteArray) -> Dictionary:
	var buf := StreamPeerBuffer.new()
	buf.data_array = data
	var player_count := buf.get_u8()
	var players: Array[Dictionary] = []
	for i in range(player_count):
		var peer_id := buf.get_u16()
		var class_name_str := H.get_str8(buf)
		var spec_name_str := H.get_str8(buf)
		var username := H.get_str8(buf)
		var is_ready := buf.get_u8() == 1
		(
			players
			. append(
				{
					"peer_id": peer_id,
					"class_name": class_name_str,
					"spec_name": spec_name_str,
					"username": username,
					"is_ready": is_ready,
				}
			)
		)
	return {"players": players}


static func encode_lobby_state(players: Array[Dictionary]) -> PackedByteArray:
	var buf := StreamPeerBuffer.new()
	buf.put_u8(players.size())
	for player in players:
		buf.put_u16(player["peer_id"])
		H.put_str8(buf, player["class_name"])
		H.put_str8(buf, player.get("username", ""))
		buf.put_u8(1 if player["is_ready"] else 0)
	return buf.data_array


# =============================================================================
# Group state (server -> client)
# =============================================================================


## Format: [group_id:u32 LE][leader_peer:u16 LE][count:u8]
##   per member: [peer_id:u16 LE][name_len:u8][name:...]
static func decode_group_state(data: PackedByteArray) -> Dictionary:
	var buf := StreamPeerBuffer.new()
	buf.data_array = data
	var group_id := buf.get_u32()
	var leader_peer := buf.get_u16()
	var count := buf.get_u8()
	var members: Array[Dictionary] = []
	for i in range(count):
		var peer_id := buf.get_u16()
		var username := H.get_str8(buf)
		members.append({"peer_id": peer_id, "username": username})
	return {"group_id": group_id, "leader_peer": leader_peer, "members": members}


## Format: [group_id:u32 LE][leader_name_len:u8][leader_name:...]
static func decode_group_invite_received(data: PackedByteArray) -> Dictionary:
	var buf := StreamPeerBuffer.new()
	buf.data_array = data
	var group_id := buf.get_u32()
	var leader_name := H.get_str8(buf)
	return {"group_id": group_id, "leader_name": leader_name}


## Format: [error_code:u8][msg_len:u8][msg:...]
static func decode_group_error(data: PackedByteArray) -> Dictionary:
	var buf := StreamPeerBuffer.new()
	buf.data_array = data
	var error_code := buf.get_u8()
	var msg := H.get_str8(buf)
	return {"error_code": error_code, "message": msg}


## Encode group invite target: [target_peer_id:u16 LE]
static func encode_group_invite(target_peer_id: int) -> PackedByteArray:
	var buf := StreamPeerBuffer.new()
	buf.put_u16(target_peer_id)
	return buf.data_array


## Encode group invite reply: [group_id:u32 LE][accept:u8]
static func encode_group_invite_reply(group_id: int, accept: bool) -> PackedByteArray:
	var buf := StreamPeerBuffer.new()
	buf.put_u32(group_id)
	buf.put_u8(1 if accept else 0)
	return buf.data_array
