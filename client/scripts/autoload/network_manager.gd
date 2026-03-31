extends Node

## Manages ENet multiplayer connections for local co-op.
## Tracks connected peers and their class selections.

signal player_connected(peer_id: int)
signal player_disconnected(peer_id: int)
signal connection_succeeded
signal connection_failed
signal all_players_ready
signal player_info_changed

const PORT := 7777
const MAX_CLIENTS := 3

var is_host := false
var is_active := false

# peer_id -> { "class_name": String, "ready": bool }
var player_info: Dictionary = {}


func host_game() -> Error:
	var peer := ENetMultiplayerPeer.new()
	var err := peer.create_server(PORT, MAX_CLIENTS)
	if err != OK:
		return err
	multiplayer.multiplayer_peer = peer
	is_host = true
	is_active = true
	multiplayer.peer_connected.connect(_on_peer_connected)
	multiplayer.peer_disconnected.connect(_on_peer_disconnected)
	# Register host
	player_info[1] = {"class_name": "gunner", "ready": false}
	player_connected.emit(1)
	return OK


func join_game(address: String = "127.0.0.1") -> Error:
	var peer := ENetMultiplayerPeer.new()
	var err := peer.create_client(address, PORT)
	if err != OK:
		return err
	multiplayer.multiplayer_peer = peer
	is_active = true
	multiplayer.connected_to_server.connect(_on_connected_to_server)
	multiplayer.connection_failed.connect(_on_connection_failed)
	multiplayer.peer_connected.connect(_on_peer_connected)
	multiplayer.peer_disconnected.connect(_on_peer_disconnected)
	return OK


func disconnect_game() -> void:
	if multiplayer.multiplayer_peer:
		multiplayer.multiplayer_peer.close()
		multiplayer.multiplayer_peer = null
	player_info.clear()
	is_host = false
	is_active = false


func get_my_id() -> int:
	if not is_active:
		return 1
	return multiplayer.get_unique_id()


func _on_peer_connected(id: int) -> void:
	if not player_info.has(id):
		player_info[id] = {"class_name": "gunner", "ready": false}
	player_connected.emit(id)
	# If host, send existing player info to the new peer
	if is_host:
		for pid in player_info:
			_sync_player_info.rpc_id(id, pid, player_info[pid]["class_name"], player_info[pid]["ready"])


func _on_peer_disconnected(id: int) -> void:
	player_info.erase(id)
	player_disconnected.emit(id)


func _on_connected_to_server() -> void:
	var my_id := multiplayer.get_unique_id()
	player_info[my_id] = {"class_name": "gunner", "ready": false}
	connection_succeeded.emit()


func _on_connection_failed() -> void:
	is_active = false
	connection_failed.emit()


@rpc("any_peer", "call_local", "reliable")
func set_player_class(class_name_str: String) -> void:
	var sender := multiplayer.get_remote_sender_id()
	if sender == 0:
		sender = multiplayer.get_unique_id()
	if sender in player_info:
		player_info[sender]["class_name"] = class_name_str
		player_info_changed.emit()
		# If host, relay to all peers
		if is_host:
			for pid in player_info:
				if pid != sender:
					_sync_player_info.rpc_id(pid, sender, class_name_str, player_info[sender]["ready"])


@rpc("any_peer", "call_local", "reliable")
func set_player_ready(is_ready: bool) -> void:
	var sender := multiplayer.get_remote_sender_id()
	if sender == 0:
		sender = multiplayer.get_unique_id()
	if sender in player_info:
		player_info[sender]["ready"] = is_ready
		player_info_changed.emit()
		# If host, relay and check if all ready
		if is_host:
			for pid in player_info:
				if pid != sender:
					_sync_player_info.rpc_id(pid, sender, player_info[sender]["class_name"], is_ready)
			_check_all_ready()


@rpc("authority", "call_local", "reliable")
func _sync_player_info(peer_id: int, class_name_str: String, is_ready: bool) -> void:
	player_info[peer_id] = {"class_name": class_name_str, "ready": is_ready}
	player_info_changed.emit()


func _check_all_ready() -> void:
	if player_info.size() < 2:
		return
	for pid in player_info:
		if not player_info[pid]["ready"]:
			return
	all_players_ready.emit()


func reset_ready_states() -> void:
	if is_host and is_active:
		_reset_ready.rpc()
	else:
		_reset_ready_local()


@rpc("authority", "call_local", "reliable")
func _reset_ready() -> void:
	_reset_ready_local()


func _reset_ready_local() -> void:
	for pid in player_info:
		player_info[pid]["ready"] = false
	player_info_changed.emit()
