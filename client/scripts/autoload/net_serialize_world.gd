class_name NetSerializeWorld
## World state, damage events, and game flow event codecs.
## All methods are static — called via forwarding stubs in NetSerializer.

const H := preload("res://scripts/autoload/net_serialize_helpers.gd")


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
static func decode_world_state(data: PackedByteArray) -> Dictionary:
	var buf := StreamPeerBuffer.new()
	buf.data_array = data

	var tick := buf.get_u32()

	var player_count := buf.get_u8()
	var players: Array[Dictionary] = []
	for i in range(player_count):
		players.append(_decode_player_entry(buf))

	var enemy_count := buf.get_u8()
	var enemies: Array[Dictionary] = []
	for i in range(enemy_count):
		enemies.append(_decode_enemy_entry(buf))

	var proj_count := buf.get_u8()
	var projectiles: Array[Dictionary] = []
	for i in range(proj_count):
		projectiles.append(_decode_projectile_entry(buf))

	# NPCs (appended after projectiles)
	var npc_list: Array[Dictionary] = []
	if buf.get_position() < buf.get_size():
		var npc_count := buf.get_u8()
		for i in range(npc_count):
			var npc_id := buf.get_u16()
			var npc_pos := Vector3(buf.get_float(), buf.get_float(), buf.get_float())
			var npc_rot_y := buf.get_float()
			var npc_state := buf.get_u8()
			var npc_def_name := H.get_str8(buf)
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

	# Telegraphs (appended after NPCs). Server-authoritative danger/heal zones.
	var telegraphs: Array[Dictionary] = []
	if buf.get_position() < buf.get_size():
		var tg_count := buf.get_u8()
		for i in range(tg_count):
			telegraphs.append(_decode_telegraph_entry(buf))

	return {
		"tick": tick,
		"players": players,
		"enemies": enemies,
		"projectiles": projectiles,
		"npcs": npc_list,
		"telegraphs": telegraphs,
	}


## Telegraph shapes: 0=circle 1=cone 2=line 3=multi_circle.
## Categories: 0=unavoidable 1=parryable 2=blockable 3=heal.
static func _decode_telegraph_entry(buf: StreamPeerBuffer) -> Dictionary:
	var d := {
		"id": buf.get_u32(),
		"shape": buf.get_u8(),
		"category": buf.get_u8(),
		"start_tick": buf.get_u32(),
		"execute_tick": buf.get_u32(),
	}
	match d["shape"]:
		0:  # circle
			d["cx"] = buf.get_float()
			d["cz"] = buf.get_float()
			d["radius"] = buf.get_float()
		1:  # cone
			d["cx"] = buf.get_float()
			d["cz"] = buf.get_float()
			d["facing"] = buf.get_float()
			d["half_angle"] = buf.get_float()
			d["range"] = buf.get_float()
		2:  # line
			d["cx"] = buf.get_float()
			d["cz"] = buf.get_float()
			d["dir_x"] = buf.get_float()
			d["dir_z"] = buf.get_float()
			d["length"] = buf.get_float()
			d["width"] = buf.get_float()
		3:  # multi_circle
			d["radius"] = buf.get_float()
			var n := buf.get_u8()
			var centers: Array[Vector2] = []
			for j in range(n):
				centers.append(Vector2(buf.get_float(), buf.get_float()))
			d["centers"] = centers
	return d


## encode_telegraphs mirrors the server's AppendTelegraphs (used by tests).
static func encode_telegraphs(buf: StreamPeerBuffer, telegraphs: Array) -> void:
	buf.put_u8(telegraphs.size())
	for t: Dictionary in telegraphs:
		buf.put_u32(t["id"])
		buf.put_u8(t["shape"])
		buf.put_u8(t["category"])
		buf.put_u32(t["start_tick"])
		buf.put_u32(t["execute_tick"])
		match t["shape"]:
			0:
				buf.put_float(t["cx"])
				buf.put_float(t["cz"])
				buf.put_float(t["radius"])
			1:
				buf.put_float(t["cx"])
				buf.put_float(t["cz"])
				buf.put_float(t["facing"])
				buf.put_float(t["half_angle"])
				buf.put_float(t["range"])
			2:
				buf.put_float(t["cx"])
				buf.put_float(t["cz"])
				buf.put_float(t["dir_x"])
				buf.put_float(t["dir_z"])
				buf.put_float(t["length"])
				buf.put_float(t["width"])
			3:
				buf.put_float(t["radius"])
				var centers: Array = t["centers"]
				buf.put_u8(centers.size())
				for c: Vector2 in centers:
					buf.put_float(c.x)
					buf.put_float(c.y)


static func _decode_player_entry(buf: StreamPeerBuffer) -> Dictionary:
	var peer_id := buf.get_u16()
	var pos := Vector3(buf.get_float(), buf.get_float(), buf.get_float())
	var rot_y := buf.get_float()
	var health := buf.get_float()
	var max_health := buf.get_float()
	var state := buf.get_u8()
	var class_name_str := H.get_str8(buf)
	var spec_name_str := H.get_str8(buf)
	var username := H.get_str8(buf)
	var visual_state := buf.get_u8()
	var aim_pitch := buf.get_float()

	var ext := _decode_player_extended(buf, class_name_str)

	var result := {
		"peer_id": peer_id,
		"pos": pos,
		"rot_y": rot_y,
		"health": health,
		"max_health": max_health,
		"state": state,
		"class_name": class_name_str,
		"spec_name": spec_name_str,
		"username": username,
		"visual_state": visual_state,
		"aim_pitch": aim_pitch,
	}
	result.merge(ext)
	return result


static func _decode_player_extended(buf: StreamPeerBuffer, cls: String) -> Dictionary:
	var bf := buf.get_u8() if buf.get_position() < buf.get_size() else 0
	var config := buf.get_u8() if buf.get_position() < buf.get_size() else 0
	var stamina := buf.get_float() if buf.get_position() + 4 <= buf.get_size() else -1.0
	var shield_hp := buf.get_float() if buf.get_position() + 4 <= buf.get_size() else 0.0
	var munitions := buf.get_float() if buf.get_position() + 4 <= buf.get_size() else 0.0
	var resonance := buf.get_float() if buf.get_position() + 4 <= buf.get_size() else 0.0
	var player_flux := buf.get_float() if buf.get_position() + 4 <= buf.get_size() else 0.0
	var max_flux := buf.get_float() if buf.get_position() + 4 <= buf.get_size() else 0.0
	var onslaught_stacks := buf.get_u8() if buf.get_position() < buf.get_size() else 0

	var assault := _decode_assault_state(buf)
	var flux_pools := _decode_flux_pools(buf)

	return {
		"overclock_active": bool(bf & 0x01),
		"rechamber_buff": bool(bf & 0x02),
		"rechamber_phase": (bf >> 2) & 0x03,
		"vortex": bool(bf & 0x10),
		"guard_active": bool(bf & 0x20),
		"onslaught_tier": (bf >> 6) & 0x03,
		"onslaught_stacks": onslaught_stacks,
		"flow_tier": (bf >> 6) & 0x03 if cls == "blade_dancer" else 0,
		"flow_stacks": onslaught_stacks if cls == "blade_dancer" else 0,
		"channel_phase": (bf >> 6) & 0x03 if cls == "arcanotechnicien" else 0,
		"config": config,
		"stamina": stamina,
		"shield_hp": shield_hp,
		"munitions": munitions,
		"resonance": resonance,
		"magazine": assault["magazine"],
		"mag_max": assault["mag_max"],
		"stability": assault["stability"],
		"steadiness": assault["steadiness"],
		"pressure_stacks": assault["pressure_stacks"],
		"enhanced_loaded": assault["enhanced_loaded"],
		"reloading": assault["reloading"],
		"mag_dump_active": assault["mag_dump_active"],
		"speed_mult": assault["speed_mult"],
		"flux": player_flux,
		"max_flux": max_flux,
		"flux_pools": flux_pools,
	}


static func _decode_assault_state(buf: StreamPeerBuffer) -> Dictionary:
	var magazine := buf.get_u8() if buf.get_position() < buf.get_size() else 0
	var mag_max := buf.get_u8() if buf.get_position() < buf.get_size() else 0
	var stability_q := buf.get_u8() if buf.get_position() < buf.get_size() else 255
	var steadiness_q := buf.get_u8() if buf.get_position() < buf.get_size() else 255
	var pressure_stacks := buf.get_u8() if buf.get_position() < buf.get_size() else 0
	var enhanced_loaded := buf.get_u8() if buf.get_position() < buf.get_size() else 0
	var assault_flags := buf.get_u8() if buf.get_position() < buf.get_size() else 0
	var speed_mult_q := buf.get_u8() if buf.get_position() < buf.get_size() else 255
	return {
		"magazine": magazine,
		"mag_max": mag_max,
		"stability": float(stability_q) / 255.0,
		"steadiness": float(steadiness_q) / 255.0,
		"pressure_stacks": pressure_stacks,
		"enhanced_loaded": enhanced_loaded,
		"reloading": bool(assault_flags & 0x01),
		"mag_dump_active": bool(assault_flags & 0x02),
		"speed_mult": float(speed_mult_q) / 255.0,
	}


static func _decode_flux_pools(buf: StreamPeerBuffer) -> Array:
	var pool_count := buf.get_u8() if buf.get_position() < buf.get_size() else 0
	var pools: Array = []
	var school_order: Array[String] = ["bioarcanotechnic", "biometabolic", "frost", "aerokinetic"]
	for pool_i in range(pool_count):
		var current := buf.get_float() if buf.get_position() + 4 <= buf.get_size() else 0.0
		var mx := buf.get_float() if buf.get_position() + 4 <= buf.get_size() else 0.0
		if pool_i < school_order.size():
			pools.append({"school": school_order[pool_i], "current": current, "max": mx})
	return pools


static func _decode_enemy_entry(buf: StreamPeerBuffer) -> Dictionary:
	var enemy_alive := buf.get_u8() == 1
	var enemy_id := buf.get_u16()
	var epos := Vector3(buf.get_float(), buf.get_float(), buf.get_float())
	var erot_y := buf.get_float()
	var ehealth := buf.get_float()
	var estate := buf.get_u8()
	var ephase := buf.get_u8()
	var emax_health := buf.get_float()
	var edef_name := H.get_str8(buf)
	var ranged_target := Vector3(buf.get_float(), buf.get_float(), buf.get_float())
	var charge_dir := Vector3(buf.get_float(), buf.get_float(), buf.get_float())
	var melee_cone_angle := buf.get_float()
	var e_melee_range := buf.get_float()
	return {
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


static func _decode_projectile_entry(buf: StreamPeerBuffer) -> Dictionary:
	var proj_id := buf.get_u32()
	var ppos := Vector3(buf.get_float(), buf.get_float(), buf.get_float())
	var pdir := Vector3(buf.get_float(), buf.get_float(), buf.get_float())
	var pspeed := buf.get_float()
	var pangular_vel := buf.get_float()
	var ptag_len := buf.get_u8()
	var ptag := ""
	if ptag_len > 0:
		ptag = buf.get_data(ptag_len)[1].get_string_from_utf8()
	return {
		"proj_id": proj_id,
		"pos": ppos,
		"direction": pdir,
		"speed": pspeed,
		"angular_velocity": pangular_vel,
		"visual_tag": ptag,
	}


## Format: [target_peer_id:u16][source_peer_id:u16][amount:f32]
## [hit_x:f32][hit_y:f32][hit_z:f32][source_type:u8][overheal:f32]
static func decode_damage_event(data: PackedByteArray) -> Dictionary:
	var buf := StreamPeerBuffer.new()
	buf.data_array = data
	var result := {
		"target_peer_id": buf.get_u16(),
		"source_peer_id": buf.get_u16(),
		"amount": buf.get_float(),
		"hit_pos": Vector3(buf.get_float(), buf.get_float(), buf.get_float()),
		"source_type": buf.get_u8(),
		"overheal": 0.0,
	}
	if buf.get_position() < data.size():
		result["overheal"] = buf.get_float()
	return result


static func encode_damage_event(data: Dictionary) -> PackedByteArray:
	var buf := StreamPeerBuffer.new()
	buf.put_u16(data.get("target_peer_id", 0))
	buf.put_u16(data.get("source_peer_id", 0))
	buf.put_float(data.get("amount", 0.0))
	H.put_vec3(buf, data.get("hit_pos", Vector3.ZERO))
	buf.put_u8(data.get("source_type", 0))
	buf.put_float(data.get("overheal", 0.0))
	return buf.data_array


## Format: [flow_type:u8][text_len:u8][text:...]
static func decode_game_flow_event(data: PackedByteArray) -> Dictionary:
	var buf := StreamPeerBuffer.new()
	buf.data_array = data
	var flow_type := buf.get_u8()
	var text := H.get_str8(buf)
	return {
		"flow_type": flow_type,
		"text": text,
	}


static func encode_game_flow_event(flow_type: int, text: String = "") -> PackedByteArray:
	var buf := StreamPeerBuffer.new()
	buf.put_u8(flow_type)
	H.put_str8(buf, text)
	return buf.data_array
