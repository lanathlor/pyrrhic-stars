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
signal lobby_state_updated(data: Dictionary)
signal world_state_received(state: Dictionary)
signal damage_event_received(event: Dictionary)
signal game_flow_event(flow_type: int, text: String)
signal zone_transfer_received(zone_type: int, new_peer_id: int)
signal group_state_updated(data: Dictionary)
signal group_invite_received(group_id: int, leader_name: String)
signal group_error_received(code: int, msg: String)
signal friend_list_received(friends: Array)
signal friend_request_received(user_id: String, name: String)
signal friend_status_updated(data: Dictionary)
signal friend_error_received(msg: String)
signal player_names_received(names: Dictionary)
signal character_state_received(data: Dictionary)
signal character_list_received(data: Dictionary)
signal character_error_received(data: Dictionary)
signal debug_info_received(def_name: String, abilities: PackedStringArray)
signal inventory_state_received(data: Dictionary)
signal ability_catalog_received(catalog: Array)
signal loadout_state_received(slots: Array)
signal flux_commit_state_received(entries: Array)
signal preset_list_received(presets: Array)
signal instance_join_prompt_received(data: Dictionary)
signal overflux_state_received(data: Dictionary)
signal merchant_state_received(data: Dictionary)
signal merchant_buy_result(data: Dictionary)
signal scrip_awarded(data: Dictionary)

const UDPTransport := preload("res://scripts/autoload/net_udp_transport.gd")

## Kept for migration compatibility. Always false -- server is the authority.
var is_host := false
var is_active := false
var username: String = ""
var current_zone_type: int = NetSerializer.ZONE_TYPE_HUB
# Server-authoritative spawn for the local player in the current zone (from level data).
var spawn_yaw: float = 0.0  # facing direction, radians
var spawn_pos: Vector3 = Vector3.ZERO
# peer_id -> { "class_name": String, "spec_name": String, "ready": bool }
var player_info: Dictionary = {}
var lobby_phase: int = 0  # 0=waiting, 1=countdown
var lobby_countdown: int = 0  # seconds remaining
var dev_params: Dictionary = {}  # Set by main.gd in dev mode: {class, zone}

## Sub-handlers for debug and loadout/inventory operations.
var debug: NetworkDebugHandler
var loadout: NetworkLoadoutHandler
var social: NetworkSocialHandler

var previous_peer_id: int = 0  # peer_id from previous zone (for stale data rejection)

var _ws := WebSocketPeer.new()
var _my_peer_id: int = 0
var _was_connected := false
var _input_tick: int = 0
var _udp: RefCounted
var _ws_host: String = ""


func _ready() -> void:
	_udp = UDPTransport.new(_on_message, _on_udp_failed)
	debug = NetworkDebugHandler.new(self)
	loadout = NetworkLoadoutHandler.new(self)
	social = NetworkSocialHandler.new(self)


# =============================================================================
# Connection
# =============================================================================


func connect_to_server(address: String = "127.0.0.1") -> Error:
	disconnect_game()
	var ws_base := ServerConfig.gateway_ws_base(address)
	var url: String
	if dev_params.size() > 0:
		# Dev path: the gateway's CODEX_DEV bypass accepts a client UUID directly,
		# so local iteration and the MCP harness work without Kratos running.
		var uuid := IdentityManager.get_player_id()
		var encoded_name := username.uri_encode()
		url = "%s/ws?uuid=%s&username=%s" % [ws_base, uuid, encoded_name]
		url += "&dev_auto=1"
		url += "&dev_class=%s" % dev_params.get("class", "gunner")
		url += "&dev_zone=%s" % dev_params.get("zone", "arena")
	else:
		# Normal path: authenticate with the Kratos session token.
		url = "%s/ws?token=%s" % [ws_base, AuthManager.get_token().uri_encode()]
	_ws_host = address
	_udp.set_host(address)
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
	_udp.close()
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


## Send position + rotation + visual state for one simulation tick.
## Requires confirmed UDP association; no-op until UDP is ready.
func send_player_position(
	pos: Vector3, rot_y: float, visual_state: int = 0, aim_pitch: float = 0.0
) -> void:
	if not _udp.is_confirmed():
		return
	_input_tick += 1
	var payload := NetSerializer.Inp.encode_player_input(
		pos, rot_y, _input_tick, visual_state, aim_pitch
	)
	_udp.send(NetSerializer.OP_PLAYER_INPUT, _my_peer_id, payload)


## Send a combat action to the server.
func send_ability(action_id: int, aim_pitch: float = 0.0, rot_y: float = 0.0) -> void:
	send_msg(
		NetSerializer.OP_ABILITY_INPUT,
		NetSerializer.Inp.encode_ability(action_id, aim_pitch, rot_y)
	)


## Send a combat action with a target peer ID (e.g. heals targeting an ally).
func send_ability_targeted(
	action_id: int, aim_pitch: float, rot_y: float, target_peer_id: int
) -> void:
	send_msg(
		NetSerializer.OP_ABILITY_INPUT,
		NetSerializer.Inp.encode_ability_targeted(action_id, aim_pitch, rot_y, target_peer_id)
	)


## Send a generic interaction to the server (class select, ready toggle, etc.).
func send_interact(action: int, data: String = "") -> void:
	send_msg(NetSerializer.OP_INTERACT_INPUT, NetSerializer.Inp.encode_interact_input(action, data))


# =============================================================================
# Lobby helpers (use InteractInput under the hood)
# =============================================================================


func set_player_class(class_name_str: String) -> void:
	if not is_active:
		return
	send_interact(0, class_name_str)  # InteractClassSelect = 0


func set_player_spec(spec_name: String) -> void:
	if not is_active:
		return
	send_interact(3, spec_name)  # InteractSpecSelect = 3


func set_player_ready(_is_ready: bool) -> void:
	if not is_active:
		return
	send_interact(1)  # InteractReadyToggle = 1


func reset_ready_states() -> void:
	if not is_active:
		return
	send_interact(2)  # InteractResetReady = 2


# =============================================================================
# Group / social send helpers
# =============================================================================


func send_group_create() -> void:
	send_msg(NetSerializer.OP_GROUP_CREATE)


func send_group_invite(target_peer_id: int) -> void:
	send_msg(NetSerializer.OP_GROUP_INVITE, NetSerializer.Char.encode_group_invite(target_peer_id))


func send_group_invite_reply(group_id: int, accept: bool) -> void:
	send_msg(
		NetSerializer.OP_GROUP_INVITE_REPLY,
		NetSerializer.Char.encode_group_invite_reply(group_id, accept)
	)


func send_group_leave() -> void:
	send_msg(NetSerializer.OP_GROUP_LEAVE)


func send_enter_portal() -> void:
	send_msg(NetSerializer.OP_ENTER_PORTAL)


func send_enter_portal_with_conditions(conditions: Array) -> void:
	send_msg(
		NetSerializer.OP_ENTER_PORTAL, NetSerializer.Char.encode_overflux_conditions(conditions)
	)


func send_instance_join_reply(accept: bool) -> void:
	var payload := NetSerializer.Char.encode_instance_join_reply(accept)
	send_msg(NetSerializer.OP_INSTANCE_JOIN_REPLY, payload)


func send_instance_reset() -> void:
	send_msg(NetSerializer.OP_INSTANCE_RESET)


## Send a respawn request. type: 0 = arena (death respawn), 1 = hub (return to
## open world), 2 = unstuck (teleport an alive player to the hub spawn).
func send_respawn_request(type: int) -> void:
	send_msg(NetSerializer.OP_RESPAWN_REQUEST, PackedByteArray([type]))


# =============================================================================
# Merchant
# =============================================================================


func send_merchant_interact(tier: int) -> void:
	send_msg(NetSerializer.OP_MERCHANT_INTERACT, NetSerializer.Inp.encode_merchant_interact(tier))


func send_merchant_buy(tier: int, def_id: String) -> void:
	send_msg(NetSerializer.OP_MERCHANT_BUY, NetSerializer.Inp.encode_merchant_buy(tier, def_id))


# =============================================================================
# Character management
# =============================================================================


func send_select_character(char_id: int) -> void:
	send_msg(NetSerializer.OP_SELECT_CHARACTER, NetSerializer.Char.encode_select_character(char_id))


func send_create_character(class_name_str: String, char_name: String) -> void:
	send_msg(
		NetSerializer.OP_CREATE_CHARACTER,
		NetSerializer.Char.encode_create_character(class_name_str, char_name)
	)


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

	_udp.poll()


func _on_ws_connected() -> void:
	print("[Net] WebSocket connected, waiting for character list...")
	connection_succeeded.emit()


func _on_udp_failed() -> void:
	print("[Net] UDP required but association failed, disconnecting")
	disconnect_game()
	connection_failed.emit()


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

	if _handle_zone_opcodes(opcode, payload):
		return
	if _handle_state_opcodes(opcode, payload):
		return
	if _handle_group_opcodes(opcode, payload):
		return
	if _handle_delegated_opcodes(opcode, payload):
		return
	message_received.emit(opcode, sender_id, payload)


func _handle_zone_opcodes(opcode: int, payload: PackedByteArray) -> bool:
	match opcode:
		NetSerializer.OP_CHARACTER_STATE:
			_handle_character_state(payload)
		NetSerializer.OP_CHARACTER_LIST:
			_handle_character_list(payload)
		NetSerializer.OP_CHARACTER_ERROR:
			_handle_character_error(payload)
		NetSerializer.OP_ZONE_JOINED:
			_handle_zone_joined(payload)
		NetSerializer.OP_PEER_CONNECTED:
			_handle_peer_connected(payload)
		NetSerializer.OP_PEER_DISCONNECTED:
			_handle_peer_disconnected(payload)
		NetSerializer.OP_ZONE_TRANSFER:
			_handle_zone_transfer(payload)
		NetSerializer.OP_UDP_ASSOCIATE:
			_udp.handle_associate(payload, _my_peer_id)
		_:
			return false
	return true


func _handle_state_opcodes(opcode: int, payload: PackedByteArray) -> bool:
	match opcode:
		NetSerializer.OP_LOBBY_STATE:
			_handle_lobby_state(payload)
		NetSerializer.OP_WORLD_STATE:
			_handle_world_state(payload)
		NetSerializer.OP_DAMAGE_EVENT:
			_handle_damage_event(payload)
		NetSerializer.OP_GAME_FLOW_EVENT:
			_handle_game_flow_event(payload)
		_:
			return false
	return true


func _handle_group_opcodes(opcode: int, payload: PackedByteArray) -> bool:
	match opcode:
		NetSerializer.OP_GROUP_STATE:
			_handle_group_state(payload)
		NetSerializer.OP_GROUP_INVITE_RECV:
			_handle_group_invite(payload)
		NetSerializer.OP_GROUP_ERROR:
			_handle_group_error(payload)
		NetSerializer.OP_INSTANCE_JOIN_PROMPT:
			_handle_instance_join_prompt(payload)
		NetSerializer.OP_OVERFLUX_STATE:
			_handle_overflux_state(payload)
		NetSerializer.OP_FRIEND_LIST:
			social.handle_friend_list(payload)
		NetSerializer.OP_FRIEND_REQUEST_RECV:
			social.handle_friend_request_recv(payload)
		NetSerializer.OP_FRIEND_STATUS:
			social.handle_friend_status(payload)
		NetSerializer.OP_FRIEND_ERROR:
			social.handle_friend_error(payload)
		_:
			return false
	return true


func _handle_delegated_opcodes(opcode: int, payload: PackedByteArray) -> bool:
	match opcode:
		NetSerializer.OP_INVENTORY_STATE:
			loadout.handle_inventory_state(payload)
		NetSerializer.OP_ABILITY_CATALOG:
			loadout.handle_ability_catalog(payload)
		NetSerializer.OP_LOADOUT_STATE:
			loadout.handle_loadout_state(payload)
		NetSerializer.OP_FLUX_COMMIT_STATE:
			loadout.handle_flux_commit_state(payload)
		NetSerializer.OP_PRESET_LIST:
			loadout.handle_preset_list(payload)
		NetSerializer.OP_MERCHANT_STATE:
			var data := NetSerializer.Inv.decode_merchant_state(payload)
			merchant_state_received.emit(data)
		NetSerializer.OP_MERCHANT_BUY_RESULT:
			var data := NetSerializer.Inv.decode_merchant_buy_result(payload)
			merchant_buy_result.emit(data)
		NetSerializer.OP_SCRIP_AWARD:
			var data := NetSerializer.Inv.decode_scrip_award(payload)
			scrip_awarded.emit(data)
		NetSerializer.OP_DEBUG_INFO:
			debug.handle_debug_info(payload)
		_:
			return false
	return true


# =============================================================================
# Zone handlers
# =============================================================================


func _handle_zone_joined(payload: PackedByteArray) -> void:
	var info := NetSerializer.Char.decode_zone_joined(payload)
	if info.is_empty():
		return
	_my_peer_id = info.peer_id
	spawn_yaw = info.get("spawn_yaw", 0.0)
	spawn_pos = info.get("spawn_pos", Vector3.ZERO)
	# is_host stays false -- server is always authority
	print("[Net] Joined zone as peer %d" % _my_peer_id)

	# Register self with a default entry until the first LobbyState arrives
	player_info[_my_peer_id] = {"class_name": "gunner", "ready": false}
	connection_succeeded.emit()
	player_info_changed.emit()


func _handle_peer_connected(payload: PackedByteArray) -> void:
	var peer_id := NetSerializer.Char.decode_peer_id(payload)
	if peer_id == 0 or peer_id == _my_peer_id:
		return
	print("[Net] Peer %d connected" % peer_id)
	if not player_info.has(peer_id):
		player_info[peer_id] = {"class_name": "gunner", "ready": false}
	player_connected.emit(peer_id)
	player_info_changed.emit()


func _handle_peer_disconnected(payload: PackedByteArray) -> void:
	var peer_id := NetSerializer.Char.decode_peer_id(payload)
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
	var data := NetSerializer.Char.decode_lobby_state(payload)
	lobby_phase = data.phase
	lobby_countdown = data.countdown
	player_info.clear()
	for p in data.players:
		player_info[p.peer_id] = {
			"class_name": p.class_name,
			"spec_name": p.spec_name,
			"username": p.username,
			"ready": p.is_ready,
		}
	lobby_state_updated.emit(data)
	player_info_changed.emit()


func _handle_world_state(payload: PackedByteArray) -> void:
	var data := NetSerializer.World.decode_world_state(payload)
	world_state_received.emit(data)


func _handle_damage_event(payload: PackedByteArray) -> void:
	var data := NetSerializer.World.decode_damage_event(payload)
	damage_event_received.emit(data)


func _handle_game_flow_event(payload: PackedByteArray) -> void:
	var data := NetSerializer.World.decode_game_flow_event(payload)
	game_flow_event.emit(data.flow_type, data.text)
	# Emit all_players_ready for legacy compatibility when server sends FLOW_SPAWN_PLAYERS
	if data.flow_type == NetSerializer.FLOW_SPAWN_PLAYERS:
		all_players_ready.emit()


func _handle_character_state(payload: PackedByteArray) -> void:
	var data := NetSerializer.Char.decode_character_state(payload)
	print("[Net] Character state: class=%s pos=%s" % [data.class_name, data.position])
	character_state_received.emit(data)


func _handle_character_list(payload: PackedByteArray) -> void:
	var data := NetSerializer.Char.decode_character_list(payload)
	print(
		(
			"[Net] Character list: %d characters, last_char_id=%d"
			% [data.characters.size(), data.last_char_id]
		)
	)
	character_list_received.emit(data)


func _handle_character_error(payload: PackedByteArray) -> void:
	var data := NetSerializer.Char.decode_character_error(payload)
	print("[Net] Character error: code=%d msg=%s" % [data.error_code, data.message])
	character_error_received.emit(data)


func _handle_zone_transfer(payload: PackedByteArray) -> void:
	var data := NetSerializer.Char.decode_zone_transfer(payload)
	if data.is_empty():
		return
	previous_peer_id = _my_peer_id
	_my_peer_id = data.new_peer_id
	current_zone_type = data.zone_type
	spawn_yaw = data.get("spawn_yaw", 0.0)
	spawn_pos = data.get("spawn_pos", Vector3.ZERO)
	player_info.clear()
	_udp.reset_tick()
	print("[Net] Zone transfer: type=%d, new peer_id=%d" % [data.zone_type, data.new_peer_id])
	zone_transfer_received.emit(data.zone_type, data.new_peer_id)


func _handle_group_state(payload: PackedByteArray) -> void:
	var data := NetSerializer.Char.decode_group_state(payload)
	group_state_updated.emit(data)


func _handle_group_invite(payload: PackedByteArray) -> void:
	var data := NetSerializer.Char.decode_group_invite_received(payload)
	group_invite_received.emit(data.group_id, data.leader_name)


func _handle_group_error(payload: PackedByteArray) -> void:
	var data := NetSerializer.Char.decode_group_error(payload)
	group_error_received.emit(data.error_code, data.message)


func _handle_instance_join_prompt(payload: PackedByteArray) -> void:
	var data := NetSerializer.Char.decode_instance_join_prompt(payload)
	instance_join_prompt_received.emit(data)


func _handle_overflux_state(payload: PackedByteArray) -> void:
	var data := NetSerializer.Char.decode_overflux_state(payload)
	overflux_state_received.emit(data)
