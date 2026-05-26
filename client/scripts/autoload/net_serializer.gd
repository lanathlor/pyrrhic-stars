extends Node

## Binary message serialization for WebSocket protocol.
## Wire format: [opcode:2][sender_id:2][payload...]
## All multi-byte values are big-endian.
##
## Implementation split across codec files. Call static methods via:
##   NetSerializer.World.decode_world_state(data)
##   NetSerializer.Char.decode_lobby_state(data)
##   NetSerializer.Inv.decode_inventory_state(data)
##   NetSerializer.Inp.encode_player_input(...)

## Codec class references — use these for encode/decode calls.
const World := preload("res://scripts/autoload/net_serialize_world.gd")
const Char := preload("res://scripts/autoload/net_serialize_character.gd")
const Inv := preload("res://scripts/autoload/net_serialize_inventory.gd")
const Inp := preload("res://scripts/autoload/net_serialize_input.gd")

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

# Social / group — client -> server
const OP_GROUP_CREATE := 0x0050
const OP_GROUP_INVITE := 0x0051
const OP_GROUP_INVITE_REPLY := 0x0052
const OP_GROUP_LEAVE := 0x0053
const OP_GROUP_KICK := 0x0054
const OP_ENTER_PORTAL := 0x0055

# Social / group — server -> client
const OP_GROUP_STATE := 0x0060
const OP_GROUP_INVITE_RECV := 0x0061
const OP_GROUP_ERROR := 0x0062
const OP_HUB_STATE := 0x0063
const OP_PLAYER_NAMES := 0x0064

# Inventory — client -> server
const OP_EQUIP_ITEM := 0x0070
const OP_UNEQUIP_ITEM := 0x0071

# Inventory — server -> client
const OP_INVENTORY_STATE := 0x0080

# Loadout — client -> server
const OP_SET_LOADOUT := 0x0090
const OP_SET_FLUX_COMMITMENT := 0x0091
const OP_SAVE_PRESET := 0x0092
const OP_DELETE_PRESET := 0x0093

# Loadout — server -> client
const OP_LOADOUT_STATE := 0x00A0
const OP_ABILITY_CATALOG := 0x00A1
const OP_FLUX_COMMIT_STATE := 0x00A2
const OP_PRESET_LIST := 0x00A3

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
const VS_VG_VORTEX := 17
const VS_VG_EXECUTION_WINDUP := 18
const VS_VG_EXECUTION := 19
# Blade Dancer-specific (20-29):
const VS_BD_CASTING := 20
const VS_BD_DASH := 21
const VS_BD_STAGGER := 22
# Arcanotechnicien-specific (40-49):
const VS_AT_CASTING := 40
const VS_AT_CHANNELING := 41
const VS_AT_STAGGER := 42
const VS_AT_CHANNELING_BEAM := 43
const VS_AT_CHANNELING_ZONE := 44
const VS_AT_SUSTAINING := 45

# Debug — client -> server (dev mode only)
const OP_DEBUG_FORCE_COMMIT := 0x00D0
const OP_DEBUG_SET_PHASE := 0x00D1
const OP_DEBUG_GOD_MODE := 0x00D2
const OP_DEBUG_TIME_SCALE := 0x00D3
const OP_DEBUG_RESET_BOSS := 0x00D4
const OP_DEBUG_REPEAT_ABILITY := 0x00D5
const OP_DEBUG_RELOAD_YAML := 0x00D6
const OP_DEBUG_REQUEST_INFO := 0x00D7
const OP_DEBUG_SPAWN_BOT := 0x00D8
const OP_DEBUG_DISMISS_BOT := 0x00D9

# Debug — server -> client
const OP_DEBUG_INFO := 0x00E0

const OP_JOIN_ZONE := 0xFF00
const OP_ZONE_JOINED := 0xFF01
const OP_PEER_CONNECTED := 0xFF02
const OP_PEER_DISCONNECTED := 0xFF03
const OP_SET_USERNAME := 0xFF04
const OP_REQUEST_ZONE_TRANSFER := 0xFF05
const OP_ZONE_TRANSFER := 0xFF06
const OP_CHARACTER_STATE := 0xFF07  # server -> client: character confirmed
const OP_CHARACTER_LIST := 0xFF08  # server -> client: all characters after auth
const OP_SELECT_CHARACTER := 0xFF09  # client -> server: select character by ID
const OP_CREATE_CHARACTER := 0xFF0A  # client -> server: create new character
const OP_CHARACTER_ERROR := 0xFF0B  # server -> client: character operation error

const HEADER_SIZE := 4

# =============================================================================
# Header encode / decode (kept here — used on every message)
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


func _swap16(v: int) -> int:
	return ((v & 0xFF) << 8) | ((v >> 8) & 0xFF)
