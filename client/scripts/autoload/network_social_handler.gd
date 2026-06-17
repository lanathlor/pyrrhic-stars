class_name NetworkSocialHandler
extends RefCounted

## Handles friends and name-based group operations (invite/kick by name).
## Instantiated by NetworkManager; not an autoload itself.

var _net: Node  # Reference to NetworkManager


func _init(net: Node) -> void:
	_net = net


# =============================================================================
# Group (name-based)
# =============================================================================


## Invite a player by account name (type_flag 0) or character name (type_flag 1).
func send_group_invite_by_name(type_flag: int, name: String) -> void:
	_net.send_msg(
		NetSerializer.OP_GROUP_INVITE_BY_NAME,
		NetSerializer.Char.encode_group_invite_by_name(type_flag, name)
	)


func send_group_kick(target_peer_id: int) -> void:
	_net.send_msg(NetSerializer.OP_GROUP_KICK, NetSerializer.Char.encode_group_kick(target_peer_id))


# =============================================================================
# Friends send helpers
# =============================================================================


## Send a friend request by account name (type_flag 0) or character name (type_flag 1).
func send_friend_request(type_flag: int, name: String) -> void:
	_net.send_msg(
		NetSerializer.OP_FRIEND_REQUEST,
		NetSerializer.Friends.encode_friend_request(type_flag, name)
	)


func send_friend_respond(accept: bool, requester_uid: String) -> void:
	_net.send_msg(
		NetSerializer.OP_FRIEND_RESPOND,
		NetSerializer.Friends.encode_friend_respond(accept, requester_uid)
	)


func send_friend_remove(friend_uid: String) -> void:
	_net.send_msg(
		NetSerializer.OP_FRIEND_REMOVE, NetSerializer.Friends.encode_friend_remove(friend_uid)
	)


func send_friend_list_request() -> void:
	_net.send_msg(NetSerializer.OP_FRIEND_LIST_REQUEST)


# =============================================================================
# Friends receive handlers (decode + emit on NetworkManager signals)
# =============================================================================


func handle_friend_list(payload: PackedByteArray) -> void:
	_net.friend_list_received.emit(NetSerializer.Friends.decode_friend_list(payload))


func handle_friend_request_recv(payload: PackedByteArray) -> void:
	var req := NetSerializer.Friends.decode_friend_request_recv(payload)
	_net.friend_request_received.emit(req.user_id, req.name)


func handle_friend_status(payload: PackedByteArray) -> void:
	_net.friend_status_updated.emit(NetSerializer.Friends.decode_friend_status(payload))


func handle_friend_error(payload: PackedByteArray) -> void:
	var err := NetSerializer.Friends.decode_friend_error(payload)
	_net.friend_error_received.emit(err.message)
