extends Node

## Binary message serialization for WebSocket protocol.
## Wire format: [opcode:2][sender_id:2][payload...]
## All multi-byte values are big-endian.

# Opcodes — must match server/internal/message/message.go
const OP_PLAYER_SYNC := 0x0001
const OP_ENEMY_SYNC := 0x0002
const OP_DAMAGE := 0x0003
const OP_NET_FLASH := 0x0004
const OP_PROJECTILE_SPAWN := 0x0005

const OP_CLASS_SELECT := 0x0010
const OP_READY_STATE := 0x0011
const OP_PLAYER_INFO := 0x0012

const OP_SPAWN_PLAYERS := 0x0020
const OP_START_FIGHT := 0x0021
const OP_SHOW_RESULT := 0x0022
const OP_RESET_READY := 0x0023

const OP_JOIN_ZONE := 0xFF00
const OP_ZONE_JOINED := 0xFF01
const OP_PEER_CONNECTED := 0xFF02
const OP_PEER_DISCONNECTED := 0xFF03

const HEADER_SIZE := 4


# =============================================================================
# Header encode / decode
# =============================================================================

func encode_header(opcode: int, sender_id: int) -> PackedByteArray:
	var buf := PackedByteArray()
	buf.resize(HEADER_SIZE)
	buf.encode_u16(0, _swap16(opcode))
	buf.encode_u16(2, _swap16(sender_id))
	return buf


func decode_header(data: PackedByteArray) -> Dictionary:
	if data.size() < HEADER_SIZE:
		return {}
	return {
		"opcode": _swap16(data.decode_u16(0)),
		"sender_id": _swap16(data.decode_u16(2)),
		"payload": data.slice(HEADER_SIZE),
	}


# =============================================================================
# Player sync
# =============================================================================

func encode_player_sync(pos: Vector3, rot_y: float, anim: String, anim_speed: float, hp: float) -> PackedByteArray:
	var buf := StreamPeerBuffer.new()
	_put_vec3(buf, pos)
	buf.put_float(rot_y)
	_put_str8(buf, anim)
	buf.put_float(anim_speed)
	buf.put_float(hp)
	return buf.data_array


func decode_player_sync(data: PackedByteArray) -> Dictionary:
	var buf := StreamPeerBuffer.new()
	buf.data_array = data
	return {
		"pos": _get_vec3(buf),
		"rot_y": buf.get_float(),
		"anim": _get_str8(buf),
		"anim_speed": buf.get_float(),
		"hp": buf.get_float(),
	}


# =============================================================================
# Enemy sync
# =============================================================================

func encode_enemy_sync(pos: Vector3, rot_y: float, hp: float, net_state: int,
		phase: int, ranged_pos: Vector3, charge_dir: Vector3) -> PackedByteArray:
	var buf := StreamPeerBuffer.new()
	_put_vec3(buf, pos)
	buf.put_float(rot_y)
	buf.put_float(hp)
	buf.put_32(net_state)
	buf.put_32(phase)
	_put_vec3(buf, ranged_pos)
	_put_vec3(buf, charge_dir)
	return buf.data_array


func decode_enemy_sync(data: PackedByteArray) -> Dictionary:
	var buf := StreamPeerBuffer.new()
	buf.data_array = data
	return {
		"pos": _get_vec3(buf),
		"rot_y": buf.get_float(),
		"hp": buf.get_float(),
		"state": buf.get_32(),
		"phase": buf.get_32(),
		"ranged_pos": _get_vec3(buf),
		"charge_dir": _get_vec3(buf),
	}


# =============================================================================
# Damage
# =============================================================================

func encode_damage(target_peer: int, amount: float, hit_pos: Vector3) -> PackedByteArray:
	var buf := StreamPeerBuffer.new()
	buf.put_u16(target_peer)
	buf.put_float(amount)
	_put_vec3(buf, hit_pos)
	return buf.data_array


func decode_damage(data: PackedByteArray) -> Dictionary:
	var buf := StreamPeerBuffer.new()
	buf.data_array = data
	return {
		"target_peer": buf.get_u16(),
		"amount": buf.get_float(),
		"hit_pos": _get_vec3(buf),
	}


# =============================================================================
# Net flash (no payload needed, just the peer ID from header)
# =============================================================================

func encode_net_flash() -> PackedByteArray:
	return PackedByteArray()


# =============================================================================
# Projectile spawn
# =============================================================================

func encode_projectile_spawn(spawn_pos: Vector3, direction: Vector3, dmg: float) -> PackedByteArray:
	var buf := StreamPeerBuffer.new()
	_put_vec3(buf, spawn_pos)
	_put_vec3(buf, direction)
	buf.put_float(dmg)
	return buf.data_array


func decode_projectile_spawn(data: PackedByteArray) -> Dictionary:
	var buf := StreamPeerBuffer.new()
	buf.data_array = data
	return {
		"spawn_pos": _get_vec3(buf),
		"direction": _get_vec3(buf),
		"dmg": buf.get_float(),
	}


# =============================================================================
# Lobby messages
# =============================================================================

func encode_class_select(class_name_str: String) -> PackedByteArray:
	var buf := StreamPeerBuffer.new()
	_put_str8(buf, class_name_str)
	return buf.data_array


func decode_class_select(data: PackedByteArray) -> Dictionary:
	var buf := StreamPeerBuffer.new()
	buf.data_array = data
	return {"class_name": _get_str8(buf)}


func encode_ready_state(is_ready: bool) -> PackedByteArray:
	return PackedByteArray([1 if is_ready else 0])


func decode_ready_state(data: PackedByteArray) -> Dictionary:
	return {"is_ready": data[0] == 1 if data.size() > 0 else false}


func encode_player_info(peer_id: int, class_name_str: String, is_ready: bool) -> PackedByteArray:
	var buf := StreamPeerBuffer.new()
	buf.put_u16(peer_id)
	_put_str8(buf, class_name_str)
	buf.put_u8(1 if is_ready else 0)
	return buf.data_array


func decode_player_info(data: PackedByteArray) -> Dictionary:
	var buf := StreamPeerBuffer.new()
	buf.data_array = data
	return {
		"peer_id": buf.get_u16(),
		"class_name": _get_str8(buf),
		"is_ready": buf.get_u8() == 1,
	}


# =============================================================================
# Game flow
# =============================================================================

func encode_show_result(text: String, color: Color) -> PackedByteArray:
	var buf := StreamPeerBuffer.new()
	_put_str8(buf, text)
	buf.put_float(color.r)
	buf.put_float(color.g)
	buf.put_float(color.b)
	return buf.data_array


func decode_show_result(data: PackedByteArray) -> Dictionary:
	var buf := StreamPeerBuffer.new()
	buf.data_array = data
	var text := _get_str8(buf)
	return {
		"text": text,
		"color": Color(buf.get_float(), buf.get_float(), buf.get_float()),
	}


# =============================================================================
# Zone management
# =============================================================================

func encode_join_zone(zone_id: String) -> PackedByteArray:
	return zone_id.to_utf8_buffer()


func decode_zone_joined(data: PackedByteArray) -> Dictionary:
	if data.size() < 3:
		return {}
	var buf := StreamPeerBuffer.new()
	buf.data_array = data
	buf.big_endian = true
	return {
		"peer_id": buf.get_u16(),
		"is_host": buf.get_u8() == 1,
	}


func decode_peer_id(data: PackedByteArray) -> int:
	if data.size() < 2:
		return 0
	var buf := StreamPeerBuffer.new()
	buf.data_array = data
	buf.big_endian = true
	return buf.get_u16()


# =============================================================================
# Helpers
# =============================================================================

func _put_vec3(buf: StreamPeerBuffer, v: Vector3) -> void:
	buf.put_float(v.x)
	buf.put_float(v.y)
	buf.put_float(v.z)


func _get_vec3(buf: StreamPeerBuffer) -> Vector3:
	return Vector3(buf.get_float(), buf.get_float(), buf.get_float())


func _put_str8(buf: StreamPeerBuffer, s: String) -> void:
	var bytes := s.to_utf8_buffer()
	buf.put_u8(bytes.size())
	if bytes.size() > 0:
		buf.put_data(bytes)


func _get_str8(buf: StreamPeerBuffer) -> String:
	var length := buf.get_u8()
	if length == 0:
		return ""
	var bytes := buf.get_data(length)
	if bytes[0] != OK:
		return ""
	return (bytes[1] as PackedByteArray).get_string_from_utf8()


## Swap bytes of a 16-bit value (native endian <-> big endian).
func _swap16(v: int) -> int:
	return ((v & 0xFF) << 8) | ((v >> 8) & 0xFF)
