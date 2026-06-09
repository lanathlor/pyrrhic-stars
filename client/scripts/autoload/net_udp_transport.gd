## Encapsulates UDP transport for high-frequency game state.
## Used by NetworkManager to send/receive world state and player input
## over UDP alongside the reliable WebSocket channel.

var _udp := PacketPeerUDP.new()
var _connected := false
var _confirmed := false
var _last_tick: int = 0
var _attempt_time: float = 0.0
var _token: PackedByteArray = PackedByteArray()
var _host: String = ""
var _on_message: Callable
var _on_failed: Callable


func _init(on_message: Callable, on_failed: Callable) -> void:
	_on_message = on_message
	_on_failed = on_failed


func is_confirmed() -> bool:
	return _confirmed


func reset_tick() -> void:
	_last_tick = 0


func set_host(host: String) -> void:
	_host = host


func close() -> void:
	if _connected:
		_udp.close()
		_connected = false
		_confirmed = false
		_last_tick = 0
		_attempt_time = 0.0
		_token = PackedByteArray()


func poll() -> void:
	if not _connected:
		return
	if not _confirmed and _attempt_time > 0.0:
		var elapsed := Time.get_ticks_msec() / 1000.0 - _attempt_time
		if elapsed > 2.0:
			print("[Net] UDP association timed out, disconnecting")
			_udp.close()
			_connected = false
			_attempt_time = 0.0
			_on_failed.call()
			return
	while _udp.get_available_packet_count() > 0:
		var pkt := _udp.get_packet()
		_handle_packet(pkt)


func handle_associate(payload: PackedByteArray, peer_id: int) -> void:
	if payload.size() < 18:
		return
	_token = payload.slice(0, 16)
	var port: int = (payload[16] << 8) | payload[17]
	if _connected:
		_udp.close()
	var err := _udp.connect_to_host(_host, port)
	if err != OK:
		print("[Net] UDP connect failed: %s" % error_string(err))
		return
	var assoc := PackedByteArray()
	assoc.resize(20)
	var op := NetSerializer.OP_UDP_ASSOCIATE_ACK
	assoc[0] = (op >> 8) & 0xFF
	assoc[1] = op & 0xFF
	assoc[2] = (peer_id >> 8) & 0xFF
	assoc[3] = peer_id & 0xFF
	for i in 16:
		assoc[4 + i] = _token[i]
	var send_err := _udp.put_packet(assoc)
	if send_err != OK:
		print("[Net] UDP ack send failed: %s" % error_string(send_err))
		return
	_connected = true
	_confirmed = false
	_last_tick = 0
	_attempt_time = Time.get_ticks_msec() / 1000.0
	print("[Net] UDP association sent to %s:%d" % [_host, port])


func send(opcode: int, peer_id: int, payload: PackedByteArray) -> void:
	var header := NetSerializer.encode_header(opcode, peer_id)
	var msg := PackedByteArray()
	msg.append_array(header)
	msg.append_array(payload)
	_udp.put_packet(msg)


func _handle_packet(data: PackedByteArray) -> void:
	if data.size() < 4:
		return
	var opcode: int = (data[0] << 8) | data[1]
	var payload := data.slice(4)
	if opcode == NetSerializer.OP_WORLD_STATE and payload.size() >= 4:
		var tick: int = payload[0] | (payload[1] << 8) | (payload[2] << 16) | (payload[3] << 24)
		if tick <= _last_tick:
			return
		_last_tick = tick
	if not _confirmed:
		_confirmed = true
		print("[Net] UDP confirmed (first packet received)")
	_on_message.call(data)
