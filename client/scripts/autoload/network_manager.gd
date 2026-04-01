extends Node

## Manages WebSocket connection to the Go relay server.
## Replaces the old ENet peer-to-peer networking.

signal player_connected(peer_id: int)
signal player_disconnected(peer_id: int)
signal connection_succeeded
signal connection_failed
signal all_players_ready
signal player_info_changed
signal message_received(opcode: int, sender_id: int, payload: PackedByteArray)

const DEFAULT_PORT := 7777

var is_host := false
var is_active := false

# peer_id -> { "class_name": String, "ready": bool }
var player_info: Dictionary = {}

var _ws := WebSocketPeer.new()
var _my_peer_id: int = 0
var _was_connected := false


func connect_to_server(address: String = "127.0.0.1") -> Error:
	disconnect_game()
	var port := DEFAULT_PORT
	var url := "ws://%s:%d/ws" % [address, port]
	print("[Net] Connecting to %s..." % url)
	var err := _ws.connect_to_url(url)
	if err != OK:
		print("[Net] Failed to connect: %s" % error_string(err))
		connection_failed.emit()
		return err
	is_active = true
	_was_connected = false
	return OK


func disconnect_game() -> void:
	if _ws.get_ready_state() != WebSocketPeer.STATE_CLOSED:
		_ws.close()
	player_info.clear()
	is_host = false
	is_active = false
	_my_peer_id = 0
	_was_connected = false


func get_my_id() -> int:
	if not is_active:
		return 1
	return _my_peer_id


func send_msg(opcode: int, payload: PackedByteArray = PackedByteArray()) -> void:
	if _ws.get_ready_state() != WebSocketPeer.STATE_OPEN:
		return
	var header := NetSerializer.encode_header(opcode, _my_peer_id)
	var msg := PackedByteArray()
	msg.append_array(header)
	msg.append_array(payload)
	_ws.send(msg, WebSocketPeer.WRITE_MODE_BINARY)


func _process(_delta: float) -> void:
	if not is_active:
		return

	_ws.poll()
	var ws_state := _ws.get_ready_state()

	match ws_state:
		WebSocketPeer.STATE_OPEN:
			if not _was_connected:
				_was_connected = true
				_on_ws_connected()
			while _ws.get_available_packet_count() > 0:
				var data := _ws.get_packet()
				_on_message(data)

		WebSocketPeer.STATE_CLOSED:
			if _was_connected:
				print("[Net] Disconnected (code %d)" % _ws.get_close_code())
			else:
				print("[Net] Connection failed")
				connection_failed.emit()
			is_active = false
			_was_connected = false


func _on_ws_connected() -> void:
	print("[Net] WebSocket connected, joining zone...")
	send_msg(NetSerializer.OP_JOIN_ZONE, NetSerializer.encode_join_zone("arena"))


func _on_message(data: PackedByteArray) -> void:
	var parsed := NetSerializer.decode_header(data)
	if parsed.is_empty():
		return

	var opcode: int = parsed.opcode
	var sender_id: int = parsed.sender_id
	var payload: PackedByteArray = parsed.payload

	match opcode:
		NetSerializer.OP_ZONE_JOINED:
			_handle_zone_joined(payload)
		NetSerializer.OP_PEER_CONNECTED:
			_handle_peer_connected(payload)
		NetSerializer.OP_PEER_DISCONNECTED:
			_handle_peer_disconnected(payload)
		NetSerializer.OP_CLASS_SELECT:
			_handle_class_select(sender_id, payload)
		NetSerializer.OP_READY_STATE:
			_handle_ready_state(sender_id, payload)
		NetSerializer.OP_PLAYER_INFO:
			_handle_player_info(payload)
		NetSerializer.OP_RESET_READY:
			_reset_ready_local()
		_:
			message_received.emit(opcode, sender_id, payload)


# =============================================================================
# Server message handlers
# =============================================================================

func _handle_zone_joined(payload: PackedByteArray) -> void:
	var info := NetSerializer.decode_zone_joined(payload)
	if info.is_empty():
		return
	_my_peer_id = info.peer_id
	is_host = info.is_host
	print("[Net] Joined zone as peer %d (host=%s)" % [_my_peer_id, is_host])

	# Register self
	player_info[_my_peer_id] = {"class_name": "gunner", "ready": false}
	connection_succeeded.emit()
	player_info_changed.emit()


func _handle_peer_connected(payload: PackedByteArray) -> void:
	var peer_id := NetSerializer.decode_peer_id(payload)
	if peer_id == 0 or peer_id == _my_peer_id:
		return
	print("[Net] Peer %d connected" % peer_id)
	if not player_info.has(peer_id):
		player_info[peer_id] = {"class_name": "gunner", "ready": false}
	player_connected.emit(peer_id)
	player_info_changed.emit()

	# If host, send existing player info to the new peer
	if is_host:
		for pid in player_info:
			send_msg(NetSerializer.OP_PLAYER_INFO,
				NetSerializer.encode_player_info(pid, player_info[pid]["class_name"], player_info[pid]["ready"]))


func _handle_peer_disconnected(payload: PackedByteArray) -> void:
	var peer_id := NetSerializer.decode_peer_id(payload)
	if peer_id == 0:
		return
	print("[Net] Peer %d disconnected" % peer_id)
	player_info.erase(peer_id)
	player_disconnected.emit(peer_id)
	player_info_changed.emit()


# =============================================================================
# Lobby message handlers
# =============================================================================

func _handle_class_select(sender_id: int, payload: PackedByteArray) -> void:
	var info := NetSerializer.decode_class_select(payload)
	if sender_id in player_info:
		player_info[sender_id]["class_name"] = info.class_name
		player_info_changed.emit()
	# If host, relay to all peers via PlayerInfo
	if is_host:
		for pid in player_info:
			send_msg(NetSerializer.OP_PLAYER_INFO,
				NetSerializer.encode_player_info(sender_id, info.class_name, player_info[sender_id]["ready"]))


func _handle_ready_state(sender_id: int, payload: PackedByteArray) -> void:
	var info := NetSerializer.decode_ready_state(payload)
	if sender_id in player_info:
		player_info[sender_id]["ready"] = info.is_ready
		player_info_changed.emit()
	# If host, relay and check
	if is_host:
		for pid in player_info:
			send_msg(NetSerializer.OP_PLAYER_INFO,
				NetSerializer.encode_player_info(sender_id, player_info[sender_id]["class_name"], info.is_ready))
		_check_all_ready()


func _handle_player_info(payload: PackedByteArray) -> void:
	var info := NetSerializer.decode_player_info(payload)
	player_info[info.peer_id] = {"class_name": info.class_name, "ready": info.is_ready}
	player_info_changed.emit()


# =============================================================================
# Ready / lobby helpers
# =============================================================================

func set_player_class(class_name_str: String) -> void:
	if not is_active:
		return
	send_msg(NetSerializer.OP_CLASS_SELECT, NetSerializer.encode_class_select(class_name_str))
	# Update locally immediately for host
	if _my_peer_id in player_info:
		player_info[_my_peer_id]["class_name"] = class_name_str
		player_info_changed.emit()


func set_player_ready(is_ready: bool) -> void:
	if not is_active:
		return
	send_msg(NetSerializer.OP_READY_STATE, NetSerializer.encode_ready_state(is_ready))
	# Update locally immediately
	if _my_peer_id in player_info:
		player_info[_my_peer_id]["ready"] = is_ready
		player_info_changed.emit()
		if is_host:
			_check_all_ready()


func reset_ready_states() -> void:
	if is_host and is_active:
		send_msg(NetSerializer.OP_RESET_READY)
	_reset_ready_local()


func _reset_ready_local() -> void:
	for pid in player_info:
		player_info[pid]["ready"] = false
	player_info_changed.emit()


func _check_all_ready() -> void:
	if player_info.size() < 2:
		return
	for pid in player_info:
		if not player_info[pid]["ready"]:
			return
	all_players_ready.emit()
