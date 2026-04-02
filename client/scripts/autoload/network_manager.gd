extends Node

## Manages WebSocket connection to the Go game server.
## Server-authoritative model: client sends inputs, server sends world state.

# -- Legacy signals (kept for migration compatibility) --
signal player_connected(peer_id: int)
signal player_disconnected(peer_id: int)
signal connection_succeeded
signal connection_failed
signal all_players_ready
signal player_info_changed
signal message_received(opcode: int, sender_id: int, payload: PackedByteArray)

# -- Server-authoritative signals --
signal lobby_state_updated(players: Array)
signal world_state_received(state: Dictionary)
signal damage_event_received(event: Dictionary)
signal game_flow_event(flow_type: int, text: String)
signal hub_state_received(state: Dictionary)
signal zone_transfer_received(zone_type: int, new_peer_id: int)
signal group_state_updated(data: Dictionary)
signal group_invite_received(group_id: int, leader_name: String)
signal group_error_received(code: int, msg: String)
signal player_names_received(names: Dictionary)

const DEFAULT_PORT := 7777

## Kept for migration compatibility. Always false -- server is the authority.
var is_host := false
var is_active := false
var username: String = ""
var current_zone_type: int = NetSerializer.ZONE_TYPE_HUB

# peer_id -> { "class_name": String, "ready": bool }
# Populated from OP_LOBBY_STATE; used by lobby UI.
var player_info: Dictionary = {}

var _ws := WebSocketPeer.new()
var _my_peer_id: int = 0
var _was_connected := false
var _input_tick: int = 0


# =============================================================================
# Connection
# =============================================================================

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
	_input_tick = 0


func get_my_id() -> int:
	if not is_active:
		return 1
	return _my_peer_id


# =============================================================================
# Sending
# =============================================================================

func send_msg(opcode: int, payload: PackedByteArray = PackedByteArray()) -> void:
	if _ws.get_ready_state() != WebSocketPeer.STATE_OPEN:
		return
	var header := NetSerializer.encode_header(opcode, _my_peer_id)
	var msg := PackedByteArray()
	msg.append_array(header)
	msg.append_array(payload)
	_ws.send(msg, WebSocketPeer.WRITE_MODE_BINARY)


## Send position + rotation + animation for one simulation tick.
func send_player_position(pos: Vector3, rot_y: float, anim_name: String = "", anim_speed: float = 1.0, aim_pitch: float = 0.0) -> void:
	_input_tick += 1
	send_msg(NetSerializer.OP_PLAYER_INPUT,
		NetSerializer.encode_player_input(pos, rot_y, _input_tick, anim_name, anim_speed, aim_pitch))


## Send a combat action to the server.
func send_ability(action_id: int, aim_pitch: float = 0.0) -> void:
	send_msg(NetSerializer.OP_ABILITY_INPUT,
		NetSerializer.encode_ability(action_id, aim_pitch))


## Send a generic interaction to the server (class select, ready toggle, etc.).
func send_interact(action: int, data: String = "") -> void:
	send_msg(NetSerializer.OP_INTERACT_INPUT, NetSerializer.encode_interact_input(action, data))


# =============================================================================
# Lobby helpers (use InteractInput under the hood)
# =============================================================================

func set_player_class(class_name_str: String) -> void:
	if not is_active:
		return
	send_interact(0, class_name_str)  # InteractClassSelect = 0


func set_player_ready(_is_ready: bool) -> void:
	if not is_active:
		return
	send_interact(1)  # InteractReadyToggle = 1


func reset_ready_states() -> void:
	if not is_active:
		return
	send_interact(2)  # InteractResetReady = 2


# =============================================================================
# Poll loop
# =============================================================================

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
	print("[Net] WebSocket connected, sending username and joining hub...")
	# Send username first, then join the hub zone
	if username != "":
		send_msg(NetSerializer.OP_SET_USERNAME, NetSerializer.encode_username(username))
	send_msg(NetSerializer.OP_JOIN_ZONE, NetSerializer.encode_join_zone("hub"))
	current_zone_type = NetSerializer.ZONE_TYPE_HUB


# =============================================================================
# Message dispatch
# =============================================================================

func _on_message(data: PackedByteArray) -> void:
	var parsed := NetSerializer.decode_header(data)
	if parsed.is_empty():
		return

	var opcode: int = parsed.opcode
	var sender_id: int = parsed.sender_id
	var payload: PackedByteArray = parsed.payload

	match opcode:
		# -- Zone management --
		NetSerializer.OP_ZONE_JOINED:
			_handle_zone_joined(payload)
		NetSerializer.OP_PEER_CONNECTED:
			_handle_peer_connected(payload)
		NetSerializer.OP_PEER_DISCONNECTED:
			_handle_peer_disconnected(payload)
		NetSerializer.OP_ZONE_TRANSFER:
			_handle_zone_transfer(payload)

		# -- Server-authoritative state --
		NetSerializer.OP_LOBBY_STATE:
			_handle_lobby_state(payload)
		NetSerializer.OP_WORLD_STATE:
			_handle_world_state(payload)
		NetSerializer.OP_DAMAGE_EVENT:
			_handle_damage_event(payload)
		NetSerializer.OP_GAME_FLOW_EVENT:
			_handle_game_flow_event(payload)
		NetSerializer.OP_HUB_STATE:
			_handle_hub_state(payload)

		# -- Group --
		NetSerializer.OP_GROUP_STATE:
			_handle_group_state(payload)
		NetSerializer.OP_GROUP_INVITE_RECV:
			_handle_group_invite(payload)
		NetSerializer.OP_GROUP_ERROR:
			_handle_group_error(payload)

		# -- Anything else (legacy or unknown) --
		_:
			message_received.emit(opcode, sender_id, payload)


# =============================================================================
# Zone handlers
# =============================================================================

func _handle_zone_joined(payload: PackedByteArray) -> void:
	var info := NetSerializer.decode_zone_joined(payload)
	if info.is_empty():
		return
	_my_peer_id = info.peer_id
	# is_host stays false -- server is always authority
	print("[Net] Joined zone as peer %d" % _my_peer_id)

	# Register self with a default entry until the first LobbyState arrives
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


func _handle_peer_disconnected(payload: PackedByteArray) -> void:
	var peer_id := NetSerializer.decode_peer_id(payload)
	if peer_id == 0:
		return
	print("[Net] Peer %d disconnected" % peer_id)
	player_info.erase(peer_id)
	player_disconnected.emit(peer_id)
	player_info_changed.emit()


# =============================================================================
# Server-authoritative handlers
# =============================================================================

func _handle_lobby_state(payload: PackedByteArray) -> void:
	var data := NetSerializer.decode_lobby_state(payload)
	player_info.clear()
	for p in data.players:
		player_info[p.peer_id] = {"class_name": p.class_name, "ready": p.is_ready}
	lobby_state_updated.emit(data.players)
	player_info_changed.emit()


func _handle_world_state(payload: PackedByteArray) -> void:
	var data := NetSerializer.decode_world_state(payload)
	world_state_received.emit(data)


func _handle_damage_event(payload: PackedByteArray) -> void:
	var data := NetSerializer.decode_damage_event(payload)
	damage_event_received.emit(data)


func _handle_game_flow_event(payload: PackedByteArray) -> void:
	var data := NetSerializer.decode_game_flow_event(payload)
	game_flow_event.emit(data.flow_type, data.text)
	# Emit all_players_ready for legacy compatibility when server sends FLOW_SPAWN_PLAYERS
	if data.flow_type == NetSerializer.FLOW_SPAWN_PLAYERS:
		all_players_ready.emit()


func _handle_hub_state(payload: PackedByteArray) -> void:
	var data := NetSerializer.decode_hub_state(payload)
	hub_state_received.emit(data)


func _handle_zone_transfer(payload: PackedByteArray) -> void:
	var data := NetSerializer.decode_zone_transfer(payload)
	if data.is_empty():
		return
	_my_peer_id = data.new_peer_id
	current_zone_type = data.zone_type
	player_info.clear()
	print("[Net] Zone transfer: type=%d, new peer_id=%d" % [data.zone_type, data.new_peer_id])
	zone_transfer_received.emit(data.zone_type, data.new_peer_id)


func _handle_group_state(payload: PackedByteArray) -> void:
	var data := NetSerializer.decode_group_state(payload)
	group_state_updated.emit(data)


func _handle_group_invite(payload: PackedByteArray) -> void:
	var data := NetSerializer.decode_group_invite_received(payload)
	group_invite_received.emit(data.group_id, data.leader_name)


func _handle_group_error(payload: PackedByteArray) -> void:
	var data := NetSerializer.decode_group_error(payload)
	group_error_received.emit(data.error_code, data.message)


# =============================================================================
# Group / social send helpers
# =============================================================================

func send_group_create() -> void:
	send_msg(NetSerializer.OP_GROUP_CREATE)


func send_group_invite(target_peer_id: int) -> void:
	send_msg(NetSerializer.OP_GROUP_INVITE, NetSerializer.encode_group_invite(target_peer_id))


func send_group_invite_reply(group_id: int, accept: bool) -> void:
	send_msg(NetSerializer.OP_GROUP_INVITE_REPLY, NetSerializer.encode_group_invite_reply(group_id, accept))


func send_group_leave() -> void:
	send_msg(NetSerializer.OP_GROUP_LEAVE)


func send_enter_portal() -> void:
	send_msg(NetSerializer.OP_ENTER_PORTAL)


## Send a respawn request. type: 0 = arena, 1 = hub.
func send_respawn_request(type: int) -> void:
	send_msg(NetSerializer.OP_RESPAWN_REQUEST, PackedByteArray([type]))
