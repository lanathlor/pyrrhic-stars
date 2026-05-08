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

# Client -> Server inputs (server-authoritative protocol)
const OP_PLAYER_INPUT := 0x0030
const OP_ABILITY_INPUT := 0x0031
const OP_INTERACT_INPUT := 0x0032
const OP_RESPAWN_REQUEST := 0x0033

# Server -> Client authoritative state
const OP_WORLD_STATE := 0x0040
const OP_ENTITY_SPAWN := 0x0041
const OP_ENTITY_DESPAWN := 0x0042
const OP_DAMAGE_EVENT := 0x0043
const OP_GAME_FLOW_EVENT := 0x0044
const OP_LOBBY_STATE := 0x0045
const OP_INPUT_ACK := 0x0046

# Social / group — client → server
const OP_GROUP_CREATE := 0x0050
const OP_GROUP_INVITE := 0x0051
const OP_GROUP_INVITE_REPLY := 0x0052
const OP_GROUP_LEAVE := 0x0053
const OP_GROUP_KICK := 0x0054
const OP_ENTER_PORTAL := 0x0055

# Social / group — server → client
const OP_GROUP_STATE := 0x0060
const OP_GROUP_INVITE_RECV := 0x0061
const OP_GROUP_ERROR := 0x0062
const OP_HUB_STATE := 0x0063
const OP_PLAYER_NAMES := 0x0064

# Game flow event types
const FLOW_SPAWN_PLAYERS := 1
const FLOW_FIGHT_START := 2
const FLOW_SHOW_RESULT := 3
const FLOW_PHASE_TRANSITION := 4
const FLOW_RETURN_LOBBY := 5
const FLOW_RETURN_HUB := 6
const FLOW_BOSS_DEAD := 7
const FLOW_ALL_DEAD := 8
const FLOW_BOSS_ACTIVATED := 9
const FLOW_BOSS_RESET := 10

# Zone type constants
const ZONE_TYPE_HUB := 0
const ZONE_TYPE_ARENA := 1

# VisualState constants — passthrough byte sent by client, echoed by server.
# Shared across classes:
const VS_MOVE := 0
const VS_DODGE := 1
const VS_AIRBORNE := 2
const VS_DEAD := 30
# Vanguard-specific (10-19):
const VS_VG_LIGHT_1 := 10
const VS_VG_LIGHT_2 := 11
const VS_VG_LIGHT_3 := 12
const VS_VG_HEAVY_WINDUP := 13
const VS_VG_HEAVY := 14
const VS_VG_BLOCK := 15
const VS_VG_STAGGER := 16
const VS_VG_BLADE_SWIRL := 17
const VS_VG_GROUND_SLAM_WINDUP := 18
const VS_VG_GROUND_SLAM := 19
# Blade Dancer-specific (20-29):
const VS_BD_CASTING := 20
const VS_BD_DASH := 21
const VS_BD_STAGGER := 22

const OP_JOIN_ZONE := 0xFF00
const OP_ZONE_JOINED := 0xFF01
const OP_PEER_CONNECTED := 0xFF02
const OP_PEER_DISCONNECTED := 0xFF03
const OP_SET_USERNAME := 0xFF04
const OP_REQUEST_ZONE_TRANSFER := 0xFF05
const OP_ZONE_TRANSFER := 0xFF06
const OP_CHARACTER_STATE := 0xFF07  # server → client: character confirmed
const OP_CHARACTER_LIST := 0xFF08  # server → client: all characters after auth
const OP_SELECT_CHARACTER := 0xFF09  # client → server: select character by ID
const OP_CREATE_CHARACTER := 0xFF0A  # client → server: create new character
const OP_CHARACTER_ERROR := 0xFF0B  # server → client: character operation error

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


func encode_player_sync(
	pos: Vector3, rot_y: float, anim: String, anim_speed: float, hp: float
) -> PackedByteArray:
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


func encode_enemy_sync(
	pos: Vector3,
	rot_y: float,
	hp: float,
	net_state: int,
	phase: int,
	ranged_pos: Vector3,
	charge_dir: Vector3
) -> PackedByteArray:
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
# Player input (client -> server, server-authoritative protocol)
# =============================================================================


## Client-authoritative movement: send position + visual state.
## Format: [pos_x:f32][pos_y:f32][pos_z:f32][rot_y:f32][tick:u32][visual_state:u8][aim_pitch:f32]
func encode_player_input(
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
## Format: [action_id:u8][extra...]
## action_id: 0=shoot, 1=melee, 2=heavy, 3=dodge
## For shoot: [action_id:u8][aim_pitch:f32][rot_y:f32]
func encode_ability(action_id: int, aim_pitch: float = 0.0, rot_y: float = 0.0) -> PackedByteArray:
	var buf := StreamPeerBuffer.new()
	buf.put_u8(action_id)
	buf.put_float(aim_pitch)
	buf.put_float(rot_y)
	return buf.data_array


# =============================================================================
# Interact input (client -> server, server-authoritative protocol)
# =============================================================================


## Format: [action:u8][data_len:u8][data:...]
func encode_interact_input(action: int, data: String = "") -> PackedByteArray:
	var buf := StreamPeerBuffer.new()
	buf.put_u8(action)
	var data_bytes := data.to_utf8_buffer()
	buf.put_u8(data_bytes.size())
	if data_bytes.size() > 0:
		buf.put_data(data_bytes)
	return buf.data_array


func decode_interact_input(data: PackedByteArray) -> Dictionary:
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
# World state (server -> client, server-authoritative protocol)
# =============================================================================


## Format: [tick:u32][player_count:u8]
##   per player: [peer_id:u16][x:f32][y:f32][z:f32][rot_y:f32][health:f32]
##               [state:u8][class_len:u8][class:...][username_len:u8][username:...]
##               [visual_state:u8][aim_pitch:f32]
## Then: [enemy_count:u8]
##   per enemy: [alive:u8][enemy_id:u16][ex:f32][ey:f32][ez:f32][erot_y:f32]
##              [ehealth:f32][estate:u8][ephase:u8][emax_health:f32]
##              [def_name_len:u8][def_name:...][ranged_target:3f][charge_dir:3f]
## Then: [proj_count:u8]
##   per projectile: [proj_id:u32][px:f32][py:f32][pz:f32][dx:f32][dy:f32][dz:f32]
func decode_world_state(data: PackedByteArray) -> Dictionary:
	var buf := StreamPeerBuffer.new()
	buf.data_array = data

	var tick := buf.get_u32()

	# Players
	var player_count := buf.get_u8()
	var players: Array[Dictionary] = []
	for i in range(player_count):
		var peer_id := buf.get_u16()
		var pos := Vector3(buf.get_float(), buf.get_float(), buf.get_float())
		var rot_y := buf.get_float()
		var health := buf.get_float()
		var state := buf.get_u8()
		var class_name_str := _get_str8(buf)
		var username := _get_str8(buf)
		var visual_state := buf.get_u8()
		var aim_pitch := buf.get_float()
		# Buff bitflags (1 byte) + config (1 byte) + stamina (4 bytes) — ability system
		var buff_flags := buf.get_u8() if buf.get_position() < buf.get_size() else 0
		var config := buf.get_u8() if buf.get_position() < buf.get_size() else 0
		var server_stamina := buf.get_float() if buf.get_position() + 4 <= buf.get_size() else -1.0
		var shield_hp := buf.get_float() if buf.get_position() + 4 <= buf.get_size() else 0.0
		(
			players
			. append(
				{
					"peer_id": peer_id,
					"pos": pos,
					"rot_y": rot_y,
					"health": health,
					"state": state,
					"class_name": class_name_str,
					"username": username,
					"visual_state": visual_state,
					"aim_pitch": aim_pitch,
					"overclock_active": bool(buff_flags & 0x01),
					"rechamber_buff": bool(buff_flags & 0x02),
					"rechamber_phase": (buff_flags >> 2) & 0x03,
					"blade_swirl": bool(buff_flags & 0x10),
					"guard_active": bool(buff_flags & 0x20),
					"config": config,
					"stamina": server_stamina,
					"shield_hp": shield_hp,
				}
			)
		)

	# Enemies — count-prefixed array
	var enemy_count := buf.get_u8()
	var enemies: Array[Dictionary] = []
	for i in range(enemy_count):
		var enemy_alive := buf.get_u8() == 1
		var enemy_id := buf.get_u16()
		var epos := Vector3(buf.get_float(), buf.get_float(), buf.get_float())
		var erot_y := buf.get_float()
		var ehealth := buf.get_float()
		var estate := buf.get_u8()
		var ephase := buf.get_u8()
		var emax_health := buf.get_float()
		var edef_name := _get_str8(buf)
		var ranged_target := Vector3(buf.get_float(), buf.get_float(), buf.get_float())
		var charge_dir := Vector3(buf.get_float(), buf.get_float(), buf.get_float())
		var melee_cone_angle := buf.get_float()
		var e_melee_range := buf.get_float()
		(
			enemies
			. append(
				{
					"alive": enemy_alive,
					"enemy_id": enemy_id,
					"pos": epos,
					"rot_y": erot_y,
					"health": ehealth,
					"state": estate,
					"phase": ephase,
					"max_health": emax_health,
					"def_name": edef_name,
					"ranged_target": ranged_target,
					"charge_dir": charge_dir,
					"melee_cone_angle": melee_cone_angle,
					"melee_range": e_melee_range,
				}
			)
		)

	# Projectiles
	var proj_count := buf.get_u8()
	var projectiles: Array[Dictionary] = []
	for i in range(proj_count):
		var proj_id := buf.get_u32()
		var ppos := Vector3(buf.get_float(), buf.get_float(), buf.get_float())
		var pdir := Vector3(buf.get_float(), buf.get_float(), buf.get_float())
		var pspeed := buf.get_float()
		var pangular_vel := buf.get_float()
		var ptag_len := buf.get_u8()
		var ptag := ""
		if ptag_len > 0:
			ptag = buf.get_data(ptag_len)[1].get_string_from_utf8()
		(
			projectiles
			. append(
				{
					"proj_id": proj_id,
					"pos": ppos,
					"direction": pdir,
					"speed": pspeed,
					"angular_velocity": pangular_vel,
					"visual_tag": ptag,
				}
			)
		)

	# NPCs (appended after projectiles)
	var npc_list: Array[Dictionary] = []
	if buf.get_position() < buf.get_size():
		var npc_count := buf.get_u8()
		for i in range(npc_count):
			var npc_id := buf.get_u16()
			var npc_pos := Vector3(buf.get_float(), buf.get_float(), buf.get_float())
			var npc_rot_y := buf.get_float()
			var npc_state := buf.get_u8()
			var npc_def_name := _get_str8(buf)
			(
				npc_list
				. append(
					{
						"npc_id": npc_id,
						"pos": npc_pos,
						"rot_y": npc_rot_y,
						"state": npc_state,
						"def_name": npc_def_name,
					}
				)
			)

	return {
		"tick": tick,
		"players": players,
		"enemies": enemies,
		"projectiles": projectiles,
		"npcs": npc_list,
	}


# =============================================================================
# Damage event (server -> client, server-authoritative protocol)
# =============================================================================


## Format: [target_peer_id:u16][source_peer_id:u16][amount:f32][hit_x:f32][hit_y:f32][hit_z:f32][source_type:u8]
func decode_damage_event(data: PackedByteArray) -> Dictionary:
	var buf := StreamPeerBuffer.new()
	buf.data_array = data
	return {
		"target_peer_id": buf.get_u16(),
		"source_peer_id": buf.get_u16(),
		"amount": buf.get_float(),
		"hit_pos": Vector3(buf.get_float(), buf.get_float(), buf.get_float()),
		"source_type": buf.get_u8(),
	}


func encode_damage_event(
	target_peer_id: int, source_peer_id: int, amount: float, hit_pos: Vector3, source_type: int
) -> PackedByteArray:
	var buf := StreamPeerBuffer.new()
	buf.put_u16(target_peer_id)
	buf.put_u16(source_peer_id)
	buf.put_float(amount)
	_put_vec3(buf, hit_pos)
	buf.put_u8(source_type)
	return buf.data_array


# =============================================================================
# Game flow event (server -> client, server-authoritative protocol)
# =============================================================================


## Format: [flow_type:u8][text_len:u8][text:...]
## flow_type: 1=spawn_players, 2=fight_start, 3=show_result, 4=phase_transition, 5=return_lobby
func decode_game_flow_event(data: PackedByteArray) -> Dictionary:
	var buf := StreamPeerBuffer.new()
	buf.data_array = data
	var flow_type := buf.get_u8()
	var text := _get_str8(buf)
	return {
		"flow_type": flow_type,
		"text": text,
	}


func encode_game_flow_event(flow_type: int, text: String = "") -> PackedByteArray:
	var buf := StreamPeerBuffer.new()
	buf.put_u8(flow_type)
	_put_str8(buf, text)
	return buf.data_array


# =============================================================================
# Lobby state (server -> client, server-authoritative protocol)
# =============================================================================


## Format: [player_count:u8] per player: [peer_id:u16][class_len:u8][class:...][ready:u8]
func decode_lobby_state(data: PackedByteArray) -> Dictionary:
	var buf := StreamPeerBuffer.new()
	buf.data_array = data
	var player_count := buf.get_u8()
	var players: Array[Dictionary] = []
	for i in range(player_count):
		var peer_id := buf.get_u16()
		var class_name_str := _get_str8(buf)
		var username := _get_str8(buf)
		var is_ready := buf.get_u8() == 1
		(
			players
			. append(
				{
					"peer_id": peer_id,
					"class_name": class_name_str,
					"username": username,
					"is_ready": is_ready,
				}
			)
		)
	return {"players": players}


func encode_lobby_state(players: Array[Dictionary]) -> PackedByteArray:
	var buf := StreamPeerBuffer.new()
	buf.put_u8(players.size())
	for player in players:
		buf.put_u16(player["peer_id"])
		_put_str8(buf, player["class_name"])
		_put_str8(buf, player.get("username", ""))
		buf.put_u8(1 if player["is_ready"] else 0)
	return buf.data_array


# =============================================================================
# Username (client -> server)
# =============================================================================


## Format: [name_len:u8][name:...]
func encode_username(username: String) -> PackedByteArray:
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
func decode_zone_transfer(data: PackedByteArray) -> Dictionary:
	if data.size() < 3:
		return {}
	var zone_type := data[0]
	var buf := StreamPeerBuffer.new()
	buf.data_array = data.slice(1)
	buf.big_endian = true
	var new_peer_id := buf.get_u16()
	return {"zone_type": zone_type, "new_peer_id": new_peer_id}


# =============================================================================
# Character state (server -> client, confirmation after select/create)
# =============================================================================


## Format: [charID:u32 LE][class_len:u8][class:...][name_len:u8][name:...]
##         [pos_x:f32 LE][pos_y:f32 LE][pos_z:f32 LE][rot_y:f32 LE]
func decode_character_state(data: PackedByteArray) -> Dictionary:
	var empty := {
		"char_id": 0, "class_name": "", "char_name": "", "position": Vector3.ZERO, "rot_y": 0.0
	}
	if data.size() < 5:
		return empty
	var buf := StreamPeerBuffer.new()
	buf.data_array = data
	var char_id := buf.get_u32()
	var class_name_str := _get_str8(buf)
	var char_name := _get_str8(buf)
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
# Character list (server -> client, sent once after auth)
# =============================================================================


## Format: [username_len:u8][username:...]
##         [count:u8]
##           per char: [charID:u32 LE][class_len:u8][class:...][name_len:u8][name:...]
##                     [pos_x:f32][pos_y:f32][pos_z:f32][rot_y:f32]
##         [last_char_id:u32 LE]
func decode_character_list(data: PackedByteArray) -> Dictionary:
	var result := {"username": "", "characters": [], "last_char_id": 0}
	if data.size() < 1:
		return result
	var buf := StreamPeerBuffer.new()
	buf.data_array = data
	result.username = _get_str8(buf)
	var count := buf.get_u8()
	for i in range(count):
		var char_id := buf.get_u32()
		var cls := _get_str8(buf)
		var char_name := _get_str8(buf)
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
func encode_select_character(char_id: int) -> PackedByteArray:
	var buf := StreamPeerBuffer.new()
	buf.put_u32(char_id)
	return buf.data_array


## Encode create character: [class_len:u8][class:...][name_len:u8][name:...]
func encode_create_character(class_name_str: String, char_name: String) -> PackedByteArray:
	var buf := StreamPeerBuffer.new()
	_put_str8(buf, class_name_str)
	_put_str8(buf, char_name)
	return buf.data_array


## Decode character error: [error_code:u8][msg_len:u8][msg:...]
func decode_character_error(data: PackedByteArray) -> Dictionary:
	if data.size() < 2:
		return {"error_code": 0, "message": "Unknown error"}
	var buf := StreamPeerBuffer.new()
	buf.data_array = data
	var error_code := buf.get_u8()
	var msg := _get_str8(buf)
	return {"error_code": error_code, "message": msg}


# =============================================================================
# Group state (server -> client)
# =============================================================================


## Format: [group_id:u32 LE][leader_peer:u16 LE][count:u8]
##   per member: [peer_id:u16 LE][name_len:u8][name:...]
func decode_group_state(data: PackedByteArray) -> Dictionary:
	var buf := StreamPeerBuffer.new()
	buf.data_array = data
	var group_id := buf.get_u32()
	var leader_peer := buf.get_u16()
	var count := buf.get_u8()
	var members: Array[Dictionary] = []
	for i in range(count):
		var peer_id := buf.get_u16()
		var username := _get_str8(buf)
		members.append({"peer_id": peer_id, "username": username})
	return {"group_id": group_id, "leader_peer": leader_peer, "members": members}


## Format: [group_id:u32 LE][leader_name_len:u8][leader_name:...]
func decode_group_invite_received(data: PackedByteArray) -> Dictionary:
	var buf := StreamPeerBuffer.new()
	buf.data_array = data
	var group_id := buf.get_u32()
	var leader_name := _get_str8(buf)
	return {"group_id": group_id, "leader_name": leader_name}


## Format: [error_code:u8][msg_len:u8][msg:...]
func decode_group_error(data: PackedByteArray) -> Dictionary:
	var buf := StreamPeerBuffer.new()
	buf.data_array = data
	var error_code := buf.get_u8()
	var msg := _get_str8(buf)
	return {"error_code": error_code, "message": msg}


## Encode group invite target: [target_peer_id:u16 LE]
func encode_group_invite(target_peer_id: int) -> PackedByteArray:
	var buf := StreamPeerBuffer.new()
	buf.put_u16(target_peer_id)
	return buf.data_array


## Encode group invite reply: [group_id:u32 LE][accept:u8]
func encode_group_invite_reply(group_id: int, accept: bool) -> PackedByteArray:
	var buf := StreamPeerBuffer.new()
	buf.put_u32(group_id)
	buf.put_u8(1 if accept else 0)
	return buf.data_array


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
