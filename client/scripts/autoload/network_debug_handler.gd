class_name NetworkDebugHandler
extends RefCounted

## Handles all debug-mode network sends and message handling.
## Instantiated by NetworkManager; not an autoload itself.

var _net: Node  # Reference to NetworkManager


func _init(net: Node) -> void:
	_net = net


# =============================================================================
# Send helpers (dev mode)
# =============================================================================


func send_force_commit(ability_id: String) -> void:
	var payload := NetSerializer.Inp.encode_debug_str8(ability_id)
	_net.send_msg(NetSerializer.OP_DEBUG_FORCE_COMMIT, payload)


func send_set_phase(phase: int) -> void:
	_net.send_msg(NetSerializer.OP_DEBUG_SET_PHASE, NetSerializer.Inp.encode_debug_phase(phase))


func send_god_mode(enabled: bool) -> void:
	_net.send_msg(NetSerializer.OP_DEBUG_GOD_MODE, NetSerializer.Inp.encode_debug_god_mode(enabled))


func send_time_scale(scale: float) -> void:
	var payload := NetSerializer.Inp.encode_debug_time_scale(scale)
	_net.send_msg(NetSerializer.OP_DEBUG_TIME_SCALE, payload)


func send_reset_boss() -> void:
	_net.send_msg(NetSerializer.OP_DEBUG_RESET_BOSS)


func send_repeat_ability(ability_id: String) -> void:
	_net.send_msg(
		NetSerializer.OP_DEBUG_REPEAT_ABILITY, NetSerializer.Inp.encode_debug_str8(ability_id)
	)


func send_reload_yaml() -> void:
	_net.send_msg(NetSerializer.OP_DEBUG_RELOAD_YAML)


func send_request_info() -> void:
	_net.send_msg(NetSerializer.OP_DEBUG_REQUEST_INFO)


func send_spawn_bot(cls_name: String, spec_id: String) -> void:
	var payload := PackedByteArray()
	var cls := cls_name.to_utf8_buffer()
	payload.append(cls.size())
	payload.append_array(cls)
	var spec := spec_id.to_utf8_buffer()
	payload.append(spec.size())
	payload.append_array(spec)
	_net.send_msg(NetSerializer.OP_DEBUG_SPAWN_BOT, payload)


func send_dismiss_bot(bot_id: int = 0) -> void:
	var payload := PackedByteArray()
	payload.resize(2)
	payload.encode_u16(0, bot_id)
	_net.send_msg(NetSerializer.OP_DEBUG_DISMISS_BOT, payload)


# =============================================================================
# Incoming message handler
# =============================================================================


func handle_debug_info(payload: PackedByteArray) -> void:
	var data := NetSerializer.Inp.decode_debug_info(payload)
	print("[Net] Debug info: boss=%s abilities=%s" % [data.def_name, data.abilities])
	_net.debug_info_received.emit(data.def_name, data.abilities)
