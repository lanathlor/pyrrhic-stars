extends GdUnitTestSuite

## WorldState decode roundtrip tests for NetSerializer.
## Builds binary buffers matching the Go server's EncodeWorldState format
## and verifies decode_world_state() produces the correct field values.
##
## The wire format per player is:
##   peer_id:u16  pos:3×f32  rot_y:f32  health:f32  max_health:f32
##   state:u8  class:str8  spec:str8  username:str8
##   visual_state:u8  aim_pitch:f32
##   buff_flags:u8  config:u8  stamina:f32  shield:f32  munitions:f32
##   resonance:f32  flux:f32  mastery_stacks:u8
##   gunner_assault:7×u8  speed_mult:u8


## Build a minimal single-player WorldState buffer with explicit resource values.
func _build_player_worldstate(
	peer_id: int,
	health: float,
	max_health: float,
	cls: String,
	spec_name: String,
	username: String,
	stamina: float,
	shield_hp: float,
	munitions: float,
	resonance: float,
	flux: float,
	mastery_stacks: int,
	buff_flags: int = 0,
	visual_state: int = 0,
) -> PackedByteArray:
	var buf := StreamPeerBuffer.new()
	# tick
	buf.put_u32(1)
	# player count
	buf.put_u8(1)
	# peer_id
	buf.put_u16(peer_id)
	# pos (0, 0, 0)
	buf.put_float(0.0)
	buf.put_float(0.0)
	buf.put_float(0.0)
	# rot_y
	buf.put_float(0.0)
	# health, max_health
	buf.put_float(health)
	buf.put_float(max_health)
	# state
	buf.put_u8(0)
	# class: str8
	var class_bytes := cls.to_utf8_buffer()
	buf.put_u8(class_bytes.size())
	if class_bytes.size() > 0:
		buf.put_data(class_bytes)
	# spec: str8
	var spec_bytes := spec_name.to_utf8_buffer()
	buf.put_u8(spec_bytes.size())
	if spec_bytes.size() > 0:
		buf.put_data(spec_bytes)
	# username: str8
	var name_bytes := username.to_utf8_buffer()
	buf.put_u8(name_bytes.size())
	if name_bytes.size() > 0:
		buf.put_data(name_bytes)
	# visual_state
	buf.put_u8(visual_state)
	# aim_pitch
	buf.put_float(0.0)
	# buff_flags
	buf.put_u8(buff_flags)
	# config
	buf.put_u8(0)
	# stamina
	buf.put_float(stamina)
	# shield
	buf.put_float(shield_hp)
	# munitions
	buf.put_float(munitions)
	# resonance
	buf.put_float(resonance)
	# flux  (MUST come before mastery_stacks — this is the field order bug check)
	buf.put_float(flux)
	# mastery_stacks (u8)
	buf.put_u8(mastery_stacks)
	# gunner assault state (7 bytes, all zero)
	for i in 7:
		buf.put_u8(0)
	# speed_mult (255 = 1.0)
	buf.put_u8(255)

	# enemy count = 0
	buf.put_u8(0)
	# projectile count = 0
	buf.put_u8(0)
	# npc count = 0
	buf.put_u8(0)

	return buf.data_array


## Arcanotechnicien: resonance, flux, and confluence stacks must decode correctly.
## This catches the exact bug where flux(f32) and mastery_stacks(u8) are swapped,
## which corrupts the float by reading a u8 mid-stream.
func test_arcanotechnicien_field_order() -> void:
	var data := _build_player_worldstate(
		1,            # peer_id
		120.0,        # health
		170.0,        # max_health
		"arcanotechnicien",  # class
		"harmonist",  # spec
		"TestHealer", # username
		0.0,          # stamina
		0.0,          # shield
		0.0,          # munitions
		42.0,         # resonance
		87.5,         # flux
		3,            # mastery_stacks (confluence)
	)

	var ws: Dictionary = NetSerializer.decode_world_state(data)
	assert_int(ws["tick"]).is_equal(1)
	assert_int(ws["players"].size()).is_equal(1)

	var p: Dictionary = ws["players"][0]
	assert_int(p["peer_id"]).is_equal(1)
	assert_float(p["health"]).is_equal(120.0)
	assert_float(p["max_health"]).is_equal(170.0)
	assert_str(p["class_name"]).is_equal("arcanotechnicien")
	assert_str(p["spec_name"]).is_equal("harmonist")

	# Critical field-order checks — these three are adjacent on the wire.
	assert_float(p["resonance"]).is_equal(42.0)
	assert_float(p["flux"]).is_equal(87.5)
	assert_int(p["onslaught_stacks"]).is_equal(3)


## Gunner: verify all resource fields decode in the right positions.
func test_gunner_resource_fields() -> void:
	var data := _build_player_worldstate(
		7,            # peer_id
		100.0,        # health
		150.0,        # max_health
		"gunner",     # class
		"assault",    # spec
		"TestGunner", # username
		50.0,         # stamina
		25.0,         # shield
		80.0,         # munitions
		0.0,          # resonance
		0.0,          # flux
		0,            # mastery_stacks
	)

	var ws: Dictionary = NetSerializer.decode_world_state(data)
	var p: Dictionary = ws["players"][0]

	assert_float(p["health"]).is_equal(100.0)
	assert_float(p["max_health"]).is_equal(150.0)
	assert_float(p["stamina"]).is_equal(50.0)
	assert_float(p["shield_hp"]).is_equal(25.0)
	assert_float(p["munitions"]).is_equal(80.0)
	assert_float(p["resonance"]).is_equal(0.0)
	assert_float(p["flux"]).is_equal(0.0)
	assert_int(p["onslaught_stacks"]).is_equal(0)


## Vanguard: mastery stacks (onslaught/devotion) decode after flux.
func test_vanguard_mastery_stacks() -> void:
	var data := _build_player_worldstate(
		2,            # peer_id
		200.0,        # health
		200.0,        # max_health
		"vanguard",   # class
		"blade",      # spec
		"TestTank",   # username
		100.0,        # stamina
		50.0,         # shield
		0.0,          # munitions
		30.0,         # resonance
		0.0,          # flux
		5,            # mastery_stacks (onslaught)
		(2 << 6),     # buff_flags: onslaught tier 2 in bits 6-7
	)

	var ws: Dictionary = NetSerializer.decode_world_state(data)
	var p: Dictionary = ws["players"][0]

	assert_float(p["stamina"]).is_equal(100.0)
	assert_float(p["shield_hp"]).is_equal(50.0)
	assert_float(p["resonance"]).is_equal(30.0)
	assert_int(p["onslaught_stacks"]).is_equal(5)
	assert_int(p["onslaught_tier"]).is_equal(2)


## Multiple players: verify the second player decodes correctly (offset not drifting).
func test_two_players_no_offset_drift() -> void:
	var buf := StreamPeerBuffer.new()
	# tick
	buf.put_u32(42)
	# player count = 2
	buf.put_u8(2)

	# --- Player 1 (gunner) ---
	_append_player(buf, 1, 100.0, 150.0, "gunner", "", "Alice", 50.0, 0.0, 10.0, 0.0, 0.0, 0)
	# --- Player 2 (arcanotechnicien) ---
	_append_player(buf, 2, 120.0, 170.0, "arcanotechnicien", "harmonist", "Bob", 0.0, 0.0, 0.0, 42.0, 87.5, 3)

	# enemy count = 0, proj count = 0, npc count = 0
	buf.put_u8(0)
	buf.put_u8(0)
	buf.put_u8(0)

	var ws: Dictionary = NetSerializer.decode_world_state(buf.data_array)
	assert_int(ws["tick"]).is_equal(42)
	assert_int(ws["players"].size()).is_equal(2)

	var p1: Dictionary = ws["players"][0]
	assert_int(p1["peer_id"]).is_equal(1)
	assert_float(p1["health"]).is_equal(100.0)
	assert_float(p1["stamina"]).is_equal(50.0)
	assert_float(p1["munitions"]).is_equal(10.0)
	assert_float(p1["flux"]).is_equal(0.0)

	var p2: Dictionary = ws["players"][1]
	assert_int(p2["peer_id"]).is_equal(2)
	assert_float(p2["health"]).is_equal(120.0)
	assert_str(p2["class_name"]).is_equal("arcanotechnicien")
	assert_float(p2["resonance"]).is_equal(42.0)
	assert_float(p2["flux"]).is_equal(87.5)
	assert_int(p2["onslaught_stacks"]).is_equal(3)


## Player + enemy: verify enemy fields aren't corrupted by player decode.
func test_player_plus_enemy() -> void:
	var buf := StreamPeerBuffer.new()
	buf.put_u32(10)
	# 1 player
	buf.put_u8(1)
	_append_player(buf, 1, 120.0, 170.0, "arcanotechnicien", "harmonist", "Healer", 0.0, 0.0, 0.0, 42.0, 87.5, 3)
	# 1 enemy
	buf.put_u8(1)
	_append_enemy(buf, 500, 2000.0, 2000.0, "arena_boss")
	# 0 projectiles, 0 npcs
	buf.put_u8(0)
	buf.put_u8(0)

	var ws: Dictionary = NetSerializer.decode_world_state(buf.data_array)
	assert_int(ws["players"].size()).is_equal(1)
	assert_int(ws["enemies"].size()).is_equal(1)

	var p: Dictionary = ws["players"][0]
	assert_float(p["flux"]).is_equal(87.5)
	assert_float(p["resonance"]).is_equal(42.0)

	var e: Dictionary = ws["enemies"][0]
	assert_int(e["enemy_id"]).is_equal(500)
	assert_float(e["health"]).is_equal(2000.0)
	assert_float(e["max_health"]).is_equal(2000.0)
	assert_str(e["def_name"]).is_equal("arena_boss")


# ---- Helpers ----

func _append_player(
	buf: StreamPeerBuffer,
	peer_id: int,
	health: float,
	max_health: float,
	cls: String,
	spec: String,
	username: String,
	stamina: float,
	shield_hp: float,
	munitions: float,
	resonance: float,
	flux: float,
	mastery_stacks: int,
	buff_flags: int = 0,
	visual_state: int = 0,
) -> void:
	buf.put_u16(peer_id)
	buf.put_float(0.0); buf.put_float(0.0); buf.put_float(0.0)  # pos
	buf.put_float(0.0)  # rot_y
	buf.put_float(health)
	buf.put_float(max_health)
	buf.put_u8(0)  # state
	var cb := cls.to_utf8_buffer()
	buf.put_u8(cb.size())
	if cb.size() > 0: buf.put_data(cb)
	var sb := spec.to_utf8_buffer()
	buf.put_u8(sb.size())
	if sb.size() > 0: buf.put_data(sb)
	var nb := username.to_utf8_buffer()
	buf.put_u8(nb.size())
	if nb.size() > 0: buf.put_data(nb)
	buf.put_u8(visual_state)
	buf.put_float(0.0)  # aim_pitch
	buf.put_u8(buff_flags)
	buf.put_u8(0)  # config
	buf.put_float(stamina)
	buf.put_float(shield_hp)
	buf.put_float(munitions)
	buf.put_float(resonance)
	buf.put_float(flux)
	buf.put_u8(mastery_stacks)
	for i in 7: buf.put_u8(0)  # gunner assault
	buf.put_u8(255)  # speed_mult


func _append_enemy(
	buf: StreamPeerBuffer,
	enemy_id: int,
	health: float,
	max_health: float,
	def_name: String,
) -> void:
	buf.put_u8(1)  # alive
	buf.put_u16(enemy_id)
	buf.put_float(0.0); buf.put_float(0.0); buf.put_float(0.0)  # pos
	buf.put_float(0.0)  # rot_y
	buf.put_float(health)
	buf.put_u8(0)  # state
	buf.put_u8(0)  # phase
	buf.put_float(max_health)
	var db := def_name.to_utf8_buffer()
	buf.put_u8(db.size())
	if db.size() > 0: buf.put_data(db)
	# ranged_target (3 floats) + charge_dir (3 floats) + melee_cone_angle + melee_range
	for i in 8: buf.put_float(0.0)


## Verify that the entire buffer is consumed after decoding a single arcanotechnicien player.
## If any field is misaligned, the trailing sections (enemies, projectiles, npcs) will fail.
func test_arcanotechnicien_buffer_fully_consumed() -> void:
	var buf := StreamPeerBuffer.new()
	buf.put_u32(1)  # tick
	buf.put_u8(1)   # 1 player
	_append_player(buf, 1, 120.0, 170.0, "arcanotechnicien", "harmonist", "Healer", 0.0, 0.0, 0.0, 42.0, 87.5, 3)
	# 1 enemy to prove we can read past the player
	buf.put_u8(1)
	_append_enemy(buf, 500, 2000.0, 2000.0, "arena_boss")
	# 1 projectile
	buf.put_u8(1)
	buf.put_u32(99)  # proj_id
	buf.put_float(1.0); buf.put_float(2.0); buf.put_float(3.0)  # pos
	buf.put_float(0.0); buf.put_float(0.0); buf.put_float(1.0)  # dir
	buf.put_float(22.0)  # speed
	buf.put_float(0.0)   # angular_vel
	buf.put_u8(0)        # tag len = 0
	# 0 npcs
	buf.put_u8(0)

	var ws: Dictionary = NetSerializer.decode_world_state(buf.data_array)

	assert_int(ws["players"].size()).is_equal(1)
	assert_int(ws["enemies"].size()).is_equal(1)
	assert_int(ws["projectiles"].size()).is_equal(1)

	var p: Dictionary = ws["players"][0]
	assert_float(p["flux"]).is_equal(87.5)
	assert_float(p["resonance"]).is_equal(42.0)
	assert_int(p["onslaught_stacks"]).is_equal(3)

	var e: Dictionary = ws["enemies"][0]
	assert_int(e["enemy_id"]).is_equal(500)
	assert_float(e["health"]).is_equal(2000.0)

	var proj: Dictionary = ws["projectiles"][0]
	assert_int(proj["proj_id"]).is_equal(99)
	assert_float(proj["speed"]).is_equal(22.0)
