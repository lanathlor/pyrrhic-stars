class_name NetSerializeFriends
## Binary codecs for the friends subsystem.
## Wire formats must match server/internal/codec/encode.go and friend_handler.go.
## All multi-byte values are little-endian (StreamPeerBuffer default).

const H := preload("res://scripts/autoload/net_serialize_helpers.gd")

# =============================================================================
# Client -> Server
# =============================================================================


## Encode friend request: [type:u8 (0=account,1=char)][name:str8]
static func encode_friend_request(type_flag: int, name: String) -> PackedByteArray:
	var buf := StreamPeerBuffer.new()
	buf.put_u8(type_flag)
	H.put_str8(buf, name)
	return buf.data_array


## Encode friend respond: [accept:u8][requester_user_id:str8]
static func encode_friend_respond(accept: bool, requester_uid: String) -> PackedByteArray:
	var buf := StreamPeerBuffer.new()
	buf.put_u8(1 if accept else 0)
	H.put_str8(buf, requester_uid)
	return buf.data_array


## Encode friend remove: [friend_user_id:str8]
static func encode_friend_remove(friend_uid: String) -> PackedByteArray:
	var buf := StreamPeerBuffer.new()
	H.put_str8(buf, friend_uid)
	return buf.data_array


# =============================================================================
# Server -> Client
# =============================================================================


## Format: [count:u8] per: [user_id:str8][name:str8][online:u8]
static func decode_friend_list(data: PackedByteArray) -> Array:
	var buf := StreamPeerBuffer.new()
	buf.data_array = data
	var count := buf.get_u8()
	var friends: Array = []
	for i in range(count):
		var user_id := H.get_str8(buf)
		var name := H.get_str8(buf)
		var online := buf.get_u8() == 1
		friends.append({"user_id": user_id, "name": name, "online": online})
	return friends


## Format: [requester_user_id:str8][requester_name:str8]
static func decode_friend_request_recv(data: PackedByteArray) -> Dictionary:
	var buf := StreamPeerBuffer.new()
	buf.data_array = data
	var user_id := H.get_str8(buf)
	var name := H.get_str8(buf)
	return {"user_id": user_id, "name": name}


## Format: [user_id:str8][online:u8]
static func decode_friend_status(data: PackedByteArray) -> Dictionary:
	var buf := StreamPeerBuffer.new()
	buf.data_array = data
	var user_id := H.get_str8(buf)
	var online := buf.get_u8() == 1
	return {"user_id": user_id, "online": online}


## Format: [code:u8][msg:str8]
static func decode_friend_error(data: PackedByteArray) -> Dictionary:
	var buf := StreamPeerBuffer.new()
	buf.data_array = data
	var code := buf.get_u8()
	var msg := H.get_str8(buf)
	return {"code": code, "message": msg}
