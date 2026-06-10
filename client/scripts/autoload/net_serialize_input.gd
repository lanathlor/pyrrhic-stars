class_name NetSerializeInput
## Player input, ability, interact, legacy sync, lobby messages, and debug codecs.
## All methods are static — called via forwarding stubs in NetSerializer.

const H := preload("res://scripts/autoload/net_serialize_helpers.gd")

# =============================================================================
# Player input (client -> server, server-authoritative protocol)
# =============================================================================


## Client-authoritative movement: send position + visual state.
## Format: [pos_x:f32][pos_y:f32][pos_z:f32][rot_y:f32][tick:u32][visual_state:u8][aim_pitch:f32]
static func encode_player_input(
	pos: Vector3, rot_y: float, tick: int, visual_state: int = 0, aim_pitch: float = 0.0
) -> PackedByteArray:
	var buf := StreamPeerBuffer.new()
	buf.put_float(pos.x)
	buf.put_float(pos.y)
	buf.put_float(pos.z)
	buf.put_float(rot_y)
	buf.put_u32(tick)
	buf.put_u8(visual_state)
	buf.put_float(aim_pitch)
	return buf.data_array


## Encode an ability/action request.
## Format: [action_id:u8][aim_pitch:f32][rot_y:f32]
static func encode_ability(
	action_id: int, aim_pitch: float = 0.0, rot_y: float = 0.0
) -> PackedByteArray:
	var buf := StreamPeerBuffer.new()
	buf.put_u8(action_id)
	buf.put_float(aim_pitch)
	buf.put_float(rot_y)
	return buf.data_array


## Encode an ability/action request with a target peer ID.
## Format: [action_id:u8][aim_pitch:f32][rot_y:f32][target_peer_id:u16]
static func encode_ability_targeted(
	action_id: int, aim_pitch: float, rot_y: float, target_peer_id: int
) -> PackedByteArray:
	var buf := StreamPeerBuffer.new()
	buf.put_u8(action_id)
	buf.put_float(aim_pitch)
	buf.put_float(rot_y)
	buf.put_u16(target_peer_id)
	return buf.data_array


# =============================================================================
# Interact input (client -> server)
# =============================================================================


## Format: [action:u8][data_len:u8][data:...]
static func encode_interact_input(action: int, data: String = "") -> PackedByteArray:
	var buf := StreamPeerBuffer.new()
	buf.put_u8(action)
	var data_bytes := data.to_utf8_buffer()
	buf.put_u8(data_bytes.size())
	if data_bytes.size() > 0:
		buf.put_data(data_bytes)
	return buf.data_array


static func decode_interact_input(data: PackedByteArray) -> Dictionary:
	var buf := StreamPeerBuffer.new()
	buf.data_array = data
	var action := buf.get_u8()
	var data_len := buf.get_u8()
	var text := ""
	if data_len > 0:
		var result := buf.get_data(data_len)
		if result[0] == OK:
			text = (result[1] as PackedByteArray).get_string_from_utf8()
	return {
		"action": action,
		"data": text,
	}


# =============================================================================
# Merchant input (client -> server)
# =============================================================================


static func encode_merchant_interact(tier: int) -> PackedByteArray:
	var buf := PackedByteArray()
	buf.append(tier)
	return buf


static func encode_merchant_buy(tier: int, def_id: String) -> PackedByteArray:
	var buf := PackedByteArray()
	buf.append(tier)
	var id_bytes := def_id.to_utf8_buffer()
	buf.append(id_bytes.size())
	buf.append_array(id_bytes)
	return buf


# =============================================================================
# Legacy sync (kept for backwards compat / tests)
# =============================================================================


static func encode_player_sync(
	pos: Vector3, rot_y: float, anim: String, anim_speed: float, hp: float
) -> PackedByteArray:
	var buf := StreamPeerBuffer.new()
	H.put_vec3(buf, pos)
	buf.put_float(rot_y)
	H.put_str8(buf, anim)
	buf.put_float(anim_speed)
	buf.put_float(hp)
	return buf.data_array


static func decode_player_sync(data: PackedByteArray) -> Dictionary:
	var buf := StreamPeerBuffer.new()
	buf.data_array = data
	return {
		"pos": H.get_vec3(buf),
		"rot_y": buf.get_float(),
		"anim": H.get_str8(buf),
		"anim_speed": buf.get_float(),
		"hp": buf.get_float(),
	}


static func encode_enemy_sync(data: Dictionary) -> PackedByteArray:
	var buf := StreamPeerBuffer.new()
	H.put_vec3(buf, data.get("pos", Vector3.ZERO))
	buf.put_float(data.get("rot_y", 0.0))
	buf.put_float(data.get("hp", 0.0))
	buf.put_32(data.get("net_state", 0))
	buf.put_32(data.get("phase", 0))
	H.put_vec3(buf, data.get("ranged_pos", Vector3.ZERO))
	H.put_vec3(buf, data.get("charge_dir", Vector3.ZERO))
	return buf.data_array


static func decode_enemy_sync(data: PackedByteArray) -> Dictionary:
	var buf := StreamPeerBuffer.new()
	buf.data_array = data
	return {
		"pos": H.get_vec3(buf),
		"rot_y": buf.get_float(),
		"hp": buf.get_float(),
		"state": buf.get_32(),
		"phase": buf.get_32(),
		"ranged_pos": H.get_vec3(buf),
		"charge_dir": H.get_vec3(buf),
	}


static func encode_damage(target_peer: int, amount: float, hit_pos: Vector3) -> PackedByteArray:
	var buf := StreamPeerBuffer.new()
	buf.put_u16(target_peer)
	buf.put_float(amount)
	H.put_vec3(buf, hit_pos)
	return buf.data_array


static func decode_damage(data: PackedByteArray) -> Dictionary:
	var buf := StreamPeerBuffer.new()
	buf.data_array = data
	return {
		"target_peer": buf.get_u16(),
		"amount": buf.get_float(),
		"hit_pos": H.get_vec3(buf),
	}


static func encode_net_flash() -> PackedByteArray:
	return PackedByteArray()


static func encode_projectile_spawn(
	spawn_pos: Vector3, direction: Vector3, dmg: float
) -> PackedByteArray:
	var buf := StreamPeerBuffer.new()
	H.put_vec3(buf, spawn_pos)
	H.put_vec3(buf, direction)
	buf.put_float(dmg)
	return buf.data_array


static func decode_projectile_spawn(data: PackedByteArray) -> Dictionary:
	var buf := StreamPeerBuffer.new()
	buf.data_array = data
	return {
		"spawn_pos": H.get_vec3(buf),
		"direction": H.get_vec3(buf),
		"dmg": buf.get_float(),
	}


# =============================================================================
# Lobby messages
# =============================================================================


static func encode_class_select(class_name_str: String) -> PackedByteArray:
	var buf := StreamPeerBuffer.new()
	H.put_str8(buf, class_name_str)
	return buf.data_array


static func decode_class_select(data: PackedByteArray) -> Dictionary:
	var buf := StreamPeerBuffer.new()
	buf.data_array = data
	return {"class_name": H.get_str8(buf)}


static func encode_ready_state(is_ready: bool) -> PackedByteArray:
	return PackedByteArray([1 if is_ready else 0])


static func decode_ready_state(data: PackedByteArray) -> Dictionary:
	return {"is_ready": data[0] == 1 if data.size() > 0 else false}


static func encode_player_info(
	peer_id: int, class_name_str: String, is_ready: bool
) -> PackedByteArray:
	var buf := StreamPeerBuffer.new()
	buf.put_u16(peer_id)
	H.put_str8(buf, class_name_str)
	buf.put_u8(1 if is_ready else 0)
	return buf.data_array


static func decode_player_info(data: PackedByteArray) -> Dictionary:
	var buf := StreamPeerBuffer.new()
	buf.data_array = data
	return {
		"peer_id": buf.get_u16(),
		"class_name": H.get_str8(buf),
		"is_ready": buf.get_u8() == 1,
	}


static func encode_show_result(text: String, color: Color) -> PackedByteArray:
	var buf := StreamPeerBuffer.new()
	H.put_str8(buf, text)
	buf.put_float(color.r)
	buf.put_float(color.g)
	buf.put_float(color.b)
	return buf.data_array


static func decode_show_result(data: PackedByteArray) -> Dictionary:
	var buf := StreamPeerBuffer.new()
	buf.data_array = data
	var text := H.get_str8(buf)
	return {
		"text": text,
		"color": Color(buf.get_float(), buf.get_float(), buf.get_float()),
	}


# =============================================================================
# Debug (dev mode)
# =============================================================================


## Encode force-commit: [str8: ability_id]
static func encode_debug_str8(s: String) -> PackedByteArray:
	var buf := StreamPeerBuffer.new()
	H.put_str8(buf, s)
	return buf.data_array


## Encode set-phase: [u8: phase]
static func encode_debug_phase(phase: int) -> PackedByteArray:
	return PackedByteArray([phase])


## Encode god-mode: [u8: 0=off, 1=on]
static func encode_debug_god_mode(enabled: bool) -> PackedByteArray:
	return PackedByteArray([1 if enabled else 0])


## Encode time-scale: [f32 LE: scale]
static func encode_debug_time_scale(scale: float) -> PackedByteArray:
	var buf := StreamPeerBuffer.new()
	buf.put_float(scale)
	return buf.data_array


## Decode debug info: [str8: def_name][u8: count][str8: ability_id]...
static func decode_debug_info(data: PackedByteArray) -> Dictionary:
	var buf := StreamPeerBuffer.new()
	buf.data_array = data
	var def_name := H.get_str8(buf)
	var count := buf.get_u8()
	var abilities: PackedStringArray = []
	for i in range(count):
		abilities.append(H.get_str8(buf))
	return {"def_name": def_name, "abilities": abilities}
