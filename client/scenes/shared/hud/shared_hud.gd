extends Control

## Shared HUD overlay — drawn on top of the game world, below class-specific HUDs.
## Contains: player status, group frames, boss frame, damage meter, minimap.
## Managed by main.gd which feeds it data from network events.
## Drawing delegated to MinimapRenderer and MeterRenderer helper classes.

# --- Constants ---
const HudDraw = preload("res://scenes/shared/hud/hud_draw.gd")
const CLASS_MAX_HP := {
	"gunner": 150.0,
	"vanguard": 200.0,
	"blade_dancer": 150.0,
	"arcanotechnicien": 150.0,
}
const HUD_BG := Color(0.02, 0.025, 0.035, 0.82)
const HUD_PANEL := Color(0.04, 0.05, 0.07, 0.45)
const HUD_BORDER := Color(0.28, 0.3, 0.36, 0.85)
const TEXT_PRIMARY := Color(0.9, 0.92, 0.96, 0.95)
const TEXT_MUTED := Color(0.63, 0.67, 0.74, 0.92)
const HEALTH_GOOD := Color(0.28, 0.78, 0.4, 1.0)
const HEALTH_BAD := Color(0.82, 0.24, 0.24, 1.0)
const POWER_COLOR := Color(0.82, 0.68, 0.24, 1.0)
const BOSS_PHASE_ONE := Color(0.56, 0.22, 0.22, 1.0)
const BOSS_PHASE_TWO := Color(0.74, 0.44, 0.18, 1.0)
const BOSS_PHASE_THREE := Color(0.78, 0.18, 0.18, 1.0)
const OFX_COLOR := Color(0.85, 0.55, 0.2, 0.95)
const OFX_BG := Color(0.04, 0.05, 0.07, 0.75)
const OFX_BORDER := Color(0.85, 0.55, 0.2, 0.4)

# --- Sub-renderers ---
var _minimap: MinimapRenderer = MinimapRenderer.new()

# --- References (set by main.gd) ---
var _local_player: CharacterBody3D = null
var _local_class: String = "gunner"
var _local_peer_id: int = 0
var _player_names: Dictionary = {}  # ref to main.gd's dict

# --- Player Status (bottom center) ---
var _player_health: float = 100.0
var _player_max_health: float = 150.0
var _player_resource: float = 0.0
var _player_max_resource: float = 0.0

# --- Group Frames (left side) ---
var _group_member_pids: Array = []  # peer IDs from group state
var _group_member_names: Dictionary = {}  # pid → username from group state

# --- World State (from server ticks) ---
var _world_players: Dictionary = {}  # pid → {pos: Vector3, health: float, rot_y: float}

# --- Boss Frame (top center) ---
var _boss_visible: bool = false
var _boss_name: String = "Arena Guardian"
var _boss_health: float = 2000.0
var _boss_phase: int = 1
var _fight_over: bool = false  # keep boss frame visible after fight ends

# --- Damage Meter (bottom right) ---
var _damage_totals: Dictionary = {}  # pid → float
var _fight_active: bool = false
var _fight_duration: float = 0.0
var _healing_totals: Dictionary = {}  # pid → float (effective healing)
var _overheal_totals: Dictionary = {}  # pid → float (overheal amount)

# --- Minimap (top right) ---
var _enemy_positions: Array = []  # Array of Vector3 for all alive enemies
var _npc_positions: Array = []  # Array of Vector3 for NPCs
var _enemy_alive: bool = false
var _player_rot_y: float = 0.0
var _boss_max_health: float = 2000.0
var _hub_mode: bool = false

# --- Overflux Widget (top left, below group frames) ---
var _overflux_conditions: Array = []
var _overflux_score: int = 0

# --- Dungeon Timer (top left, with the overflux conditions) ---
# Counts down from _time_limit using _fight_duration. Past the limit it flips to
# a red OVERTIME readout; finishing over-time pays only 10% scrip (server-side).
var _time_limit: float = 300.0


func _process(delta: float) -> void:
	# Read local player state each frame for responsive bars
	if _local_player and is_instance_valid(_local_player):
		if "health" in _local_player:
			_player_health = _local_player.health
			_player_max_health = _local_player.max_health
		_player_rot_y = _local_player.rotation.y
		# Duck-type resource (only Vanguard has stamina)
		if "stamina" in _local_player:
			_player_resource = _local_player.stamina
			_player_max_resource = _local_player.max_stamina
		else:
			_player_resource = 0.0
			_player_max_resource = 0.0

	# Throttled floor detection for minimap geometry
	var player_valid := _local_player and is_instance_valid(_local_player)
	if player_valid:
		_minimap.process_floor_detection(delta, _hub_mode, _local_player)

	if _fight_active:
		_fight_duration += delta

	queue_redraw()


func _draw() -> void:
	_draw_player_status()
	_draw_group_frames()
	if _boss_visible:
		_draw_boss_frame()
	if _fight_active or _boss_visible or _fight_over:
		MeterRenderer.draw_damage_meter(self, _damage_totals, _fight_duration, _player_names)
		MeterRenderer.draw_healing_meter(
			self, _healing_totals, _overheal_totals, _damage_totals, _fight_duration, _player_names
		)
	if _overflux_score > 0 or _fight_active or _fight_over:
		_draw_overflux_widget()
	if _hub_mode or _fight_active or _boss_visible or _fight_over:
		if not _local_player or not is_instance_valid(_local_player):
			_minimap.local_player = null
		else:
			_minimap.local_peer_id = _local_peer_id
			_minimap.world_players = _world_players
			_minimap.local_player = _local_player
			_minimap.enemy_positions = _enemy_positions
			_minimap.npc_positions = _npc_positions
			_minimap.player_rot_y = _player_rot_y
			_minimap.draw(self)


# =============================================================================
# Public API (called by main.gd)
# =============================================================================


func set_local_player(player: CharacterBody3D, class_name_str: String, peer_id: int) -> void:
	_local_player = player
	_local_class = class_name_str
	_local_peer_id = peer_id
	_player_max_health = CLASS_MAX_HP.get(class_name_str, 150.0)


func clear_local_player() -> void:
	_local_player = null
	_boss_visible = false
	_fight_active = false
	_fight_over = false
	_damage_totals.clear()
	_healing_totals.clear()
	_overheal_totals.clear()
	_world_players.clear()


func set_player_names(names: Dictionary) -> void:
	_player_names = names


func update_world_state(data: Dictionary) -> void:
	# Players
	var players_data: Array = data.get("players", [])
	_world_players.clear()
	for p in players_data:
		var pid: int = p["peer_id"]
		_world_players[pid] = {
			"pos": p.get("pos", Vector3.ZERO),
			"health": p.get("health", 0.0),
			"rot_y": p.get("rot_y", 0.0),
		}

	# Enemies — track boss and all alive positions for minimap
	var enemies: Array = data.get("enemies", [])
	_enemy_positions.clear()
	_enemy_alive = false
	# Only update boss visibility if fight is not over (preserve frame on result screen)
	if not _fight_over:
		_boss_visible = false
	for edata in enemies:
		if edata.get("alive", false):
			_enemy_alive = true
			_enemy_positions.append(edata.get("pos", Vector3.ZERO))
			# Boss frame: show only for the guard_captain
			if edata.get("def_name", "") == "guard_captain":
				_boss_health = edata.get("health", 0.0)
				_boss_phase = edata.get("phase", 1)
				_boss_max_health = edata.get("max_health", 2000.0)
				_boss_visible = true

	# NPCs — yellow dots on minimap
	var npcs: Array = data.get("npcs", [])
	_npc_positions.clear()
	for ndata in npcs:
		_npc_positions.append(ndata.get("pos", Vector3.ZERO))


func update_group_members(data: Dictionary) -> void:
	var members: Array = data.get("members", [])
	_group_member_pids.clear()
	_group_member_names.clear()
	for m in members:
		var pid: int = m.get("peer_id", 0)
		_group_member_pids.append(pid)
		var uname: String = m.get("username", "")
		if uname != "":
			_group_member_names[pid] = uname


func on_damage_event(data: Dictionary) -> void:
	if not _fight_active:
		return
	var target: int = data.get("target_peer_id", -1)
	var source: int = data.get("source_peer_id", 0)
	var amount: float = data.get("amount", 0.0)
	var overheal: float = data.get("overheal", 0.0)
	var source_type: int = data.get("source_type", 0)

	# Track healing (source_type 5 = player heal)
	if source_type == 5 and source > 0:
		_healing_totals[source] = _healing_totals.get(source, 0.0) + amount
		if overheal > 0.0:
			_overheal_totals[source] = _overheal_totals.get(source, 0.0) + overheal

	# Only count damage TO enemies (enemy IDs are >= 1000)
	if target >= 1000 and source > 0:
		_damage_totals[source] = _damage_totals.get(source, 0.0) + amount


func set_time_limit(seconds: float) -> void:
	# Dungeon completion timer (seconds), sent by the server at fight start.
	_time_limit = seconds


func on_fight_start() -> void:
	_fight_active = true
	_fight_over = false
	# Boss visibility is driven by update_world_state — guard_captain presence
	_damage_totals.clear()
	_healing_totals.clear()
	_overheal_totals.clear()
	_fight_duration = 0.0


func on_wipe() -> void:
	# A party wipe is NOT the end of the run. On the server the clear timer
	# (FightStartTick) keeps ticking while players respawn and run back, so the
	# run can still roll into OVERTIME. Keep the count-down and damage meter
	# active; only on_fight_end (boss kill) stops the clock.
	_fight_active = true
	_fight_over = false


func on_fight_end() -> void:
	_fight_active = false
	_fight_over = true
	# Keep boss frame and damage meter visible for result screen


func set_environment(env: Node3D) -> void:
	_minimap.environment = env


func on_enter_hub() -> void:
	_hub_mode = true
	_boss_visible = false
	_fight_active = false
	_fight_over = false
	_damage_totals.clear()
	_healing_totals.clear()
	_overheal_totals.clear()
	_fight_duration = 0.0
	_world_players.clear()
	_minimap.on_enter_hub()
	clear_overflux_state()


func on_enter_arena() -> void:
	_hub_mode = false
	_minimap.on_enter_arena()


func set_overflux_state(conditions: Array, total_score: int) -> void:
	_overflux_conditions = conditions
	_overflux_score = total_score


func clear_overflux_state() -> void:
	_overflux_conditions = []
	_overflux_score = 0


# =============================================================================
# Drawing — Player Status (bottom center)
# =============================================================================


func _draw_player_status() -> void:
	if not _local_player or not is_instance_valid(_local_player):
		return

	var font := ThemeDB.fallback_font
	var center_x := size.x / 2.0
	var panel_w := 248.0
	var panel_h := 30.0 if _player_max_resource > 0.0 else 14.0
	var panel_rect := Rect2(center_x - panel_w / 2.0, size.y - 126.0, panel_w, panel_h)
	var health_rect := Rect2(panel_rect.position.x, panel_rect.position.y, panel_rect.size.x, 14.0)

	var hp_ratio := clampf(_player_health / maxf(_player_max_health, 1.0), 0.0, 1.0)
	var hp_color := HEALTH_GOOD if hp_ratio > 0.3 else HEALTH_BAD
	_draw_status_bar(health_rect, hp_ratio, hp_color)
	var hp_text := "%d / %d" % [int(_player_health), int(_player_max_health)]
	draw_string(
		font,
		Vector2(health_rect.position.x + 8.0, health_rect.position.y + 13.0),
		"HEALTH",
		HORIZONTAL_ALIGNMENT_LEFT,
		60.0,
		9,
		TEXT_MUTED
	)
	draw_string(
		font,
		Vector2(health_rect.position.x + health_rect.size.x - 94.0, health_rect.position.y + 13.0),
		hp_text,
		HORIZONTAL_ALIGNMENT_RIGHT,
		88.0,
		10,
		TEXT_PRIMARY
	)

	if _player_max_resource > 0.0:
		var res_rect := Rect2(
			panel_rect.position.x, panel_rect.position.y + 18.0, panel_rect.size.x, 8.0
		)
		var res_ratio := clampf(_player_resource / maxf(_player_max_resource, 1.0), 0.0, 1.0)
		_draw_status_bar(res_rect, res_ratio, POWER_COLOR)
		var res_text := "%d / %d" % [int(_player_resource), int(_player_max_resource)]
		draw_string(
			font,
			Vector2(res_rect.position.x + res_rect.size.x - 86.0, res_rect.position.y + 8.0),
			res_text,
			HORIZONTAL_ALIGNMENT_RIGHT,
			80.0,
			9,
			TEXT_PRIMARY
		)


# =============================================================================
# Drawing — Group Frames (left side)
# =============================================================================


func _draw_group_frames() -> void:
	# In arena: use WorldState players directly (authoritative peer IDs + live health)
	# In hub: use group member list (no health data available)
	var has_world_data := not _world_players.is_empty()

	var pids_to_show: Array = []
	if has_world_data:
		# Arena: show all players from WorldState except self
		for pid in _world_players:
			if pid != _local_peer_id:
				pids_to_show.append(pid)
	else:
		# Hub: show group members except self
		for pid in _group_member_pids:
			if pid != _local_peer_id:
				pids_to_show.append(pid)

	pids_to_show.sort()
	if pids_to_show.is_empty():
		return

	var font := ThemeDB.fallback_font
	var frame_x := 18.0
	var frame_y := size.y * 0.5 - 120.0
	var frame_w := 198.0
	var frame_h := 48.0
	var frame_gap := 6.0

	var drawn := 0
	for pid in pids_to_show:
		if drawn >= 4:
			break
		var y := frame_y + drawn * (frame_h + frame_gap)
		var frame_rect := Rect2(frame_x, y, frame_w, frame_h)
		_draw_single_group_frame(font, pid, frame_rect)
		drawn += 1


func _draw_single_group_frame(font: Font, pid: int, rect: Rect2) -> void:
	draw_rect(rect, Color(HUD_PANEL, 0.15), false, 1.0)

	var uname: String = _group_member_names.get(pid, _player_names.get(pid, "Player_%d" % pid))
	if uname.length() > 14:
		uname = uname.substr(0, 14)
	draw_string(
		font,
		Vector2(rect.position.x + 2.0, rect.position.y + 10.0),
		uname,
		HORIZONTAL_ALIGNMENT_LEFT,
		rect.size.x - 70.0,
		10,
		TEXT_PRIMARY
	)

	var bar_rect := Rect2(rect.position.x + 6.0, rect.position.y + 24.0, rect.size.x - 12.0, 14.0)
	_draw_group_health_bar(font, pid, bar_rect)


func _draw_group_health_bar(font: Font, pid: int, bar_rect: Rect2) -> void:
	var max_health: float = 150.0
	var cls: String = "gunner"
	if NetworkManager.player_info.has(pid):
		cls = NetworkManager.player_info[pid].get("class_name", "gunner")
		max_health = CLASS_MAX_HP.get(cls, 150.0)

	var health: float = max_health
	if _world_players.has(pid):
		health = _world_players[pid]["health"]
		if _world_players[pid].has("max_health") and _world_players[pid]["max_health"] > 0.0:
			max_health = _world_players[pid]["max_health"]

	var ratio := clampf(health / maxf(max_health, 1.0), 0.0, 1.0)
	var bar_color := HEALTH_GOOD if ratio > 0.3 else HEALTH_BAD
	_draw_status_bar(bar_rect, ratio, bar_color)

	draw_string(
		font,
		Vector2(bar_rect.position.x + 6.0, bar_rect.position.y + 11.0),
		cls.replace("_", " ").to_upper(),
		HORIZONTAL_ALIGNMENT_LEFT,
		76.0,
		8,
		TEXT_MUTED
	)
	var hp_text := "%d / %d" % [int(health), int(max_health)]
	draw_string(
		font,
		Vector2(bar_rect.position.x + bar_rect.size.x - 84.0, bar_rect.position.y + 11.0),
		hp_text,
		HORIZONTAL_ALIGNMENT_RIGHT,
		78.0,
		9,
		TEXT_PRIMARY
	)


# =============================================================================
# Drawing — Boss Frame (top center)
# =============================================================================


func _draw_boss_frame() -> void:
	var font := ThemeDB.fallback_font
	var center_x := size.x / 2.0
	var panel_rect := Rect2(center_x - 216.0, 14.0, 432.0, 28.0)
	var bar_rect := Rect2(
		panel_rect.position.x, panel_rect.position.y + 14.0, panel_rect.size.x, 12.0
	)

	draw_string(
		font,
		Vector2(panel_rect.position.x, panel_rect.position.y + 9.0),
		_boss_name,
		HORIZONTAL_ALIGNMENT_LEFT,
		240.0,
		12,
		Color(0.93, 0.9, 0.8, 0.97)
	)

	_draw_boss_phase_label(font, panel_rect)
	_draw_boss_health_bar(font, bar_rect)


func _draw_boss_phase_label(font: Font, panel_rect: Rect2) -> void:
	var phase_text := "P%d" % _boss_phase
	var phase_color: Color
	match _boss_phase:
		1:
			phase_color = Color(0.56, 0.74, 0.28)
		2:
			phase_color = Color(0.93, 0.7, 0.25)
		3:
			phase_color = Color(0.93, 0.34, 0.34)
		_:
			phase_color = Color(0.5, 0.5, 0.5)
	draw_string(
		font,
		Vector2(panel_rect.position.x + panel_rect.size.x - 36.0, panel_rect.position.y + 9.0),
		phase_text,
		HORIZONTAL_ALIGNMENT_RIGHT,
		32.0,
		11,
		phase_color
	)


func _draw_boss_health_bar(font: Font, bar_rect: Rect2) -> void:
	var hp_ratio := clampf(_boss_health / maxf(_boss_max_health, 1.0), 0.0, 1.0)
	var bar_color: Color
	match _boss_phase:
		1:
			bar_color = BOSS_PHASE_ONE
		2:
			bar_color = BOSS_PHASE_TWO
		3:
			bar_color = BOSS_PHASE_THREE
		_:
			bar_color = Color(0.5, 0.5, 0.5)
	_draw_status_bar(bar_rect, hp_ratio, bar_color)

	var hp_text := "%d / %d" % [int(_boss_health), int(_boss_max_health)]
	draw_string(
		font,
		Vector2(bar_rect.position.x + bar_rect.size.x - 118.0, bar_rect.position.y + 12.0),
		hp_text,
		HORIZONTAL_ALIGNMENT_RIGHT,
		110.0,
		10,
		TEXT_PRIMARY
	)


# =============================================================================
# Drawing — Overflux Widget (top left)
# =============================================================================


func _draw_overflux_widget() -> void:
	var font := get_theme_default_font()
	var x := 12.0
	var y := 12.0

	# Pill background (overflux score), shown only when a score is present.
	if _overflux_score > 0:
		var text := "OFX: %d" % _overflux_score
		var text_size := font.get_string_size(text, HORIZONTAL_ALIGNMENT_LEFT, -1, 14)
		var pill_w := text_size.x + 16.0
		var pill_h := text_size.y + 8.0
		var pill_rect := Rect2(x, y, pill_w, pill_h)
		draw_rect(pill_rect, OFX_BG)
		draw_rect(pill_rect, OFX_BORDER, false, 1.0)
		draw_string(
			font,
			Vector2(x + 8.0, y + pill_h - 6.0),
			text,
			HORIZONTAL_ALIGNMENT_LEFT,
			-1,
			14,
			OFX_COLOR
		)
		y += pill_h + 4.0

	# Dungeon timer (count down remaining; red OVERTIME past the limit).
	if _fight_active or _fight_over:
		var remaining := _time_limit - _fight_duration
		var timer_text: String
		var timer_color: Color
		if remaining > 0.0:
			timer_text = "TIME %s" % HudDraw.format_mmss(remaining)
			timer_color = TEXT_PRIMARY if remaining > 30.0 else POWER_COLOR
		else:
			timer_text = "OVERTIME +%s" % HudDraw.format_mmss(absf(remaining))
			timer_color = HEALTH_BAD
		draw_string(
			font,
			Vector2(x + 4.0, y + 12.0),
			timer_text,
			HORIZONTAL_ALIGNMENT_LEFT,
			-1,
			13,
			timer_color
		)
		y += 18.0

	# Condition list below the pill / timer
	var cy := y
	for c in _overflux_conditions:
		var cond_text := "%s: %d" % [c.get("id", "?"), c.get("rank", 0)]
		draw_string(
			font,
			Vector2(x + 4.0, cy + 12.0),
			cond_text,
			HORIZONTAL_ALIGNMENT_LEFT,
			-1,
			11,
			TEXT_MUTED
		)
		cy += 16.0


func _draw_status_bar(rect: Rect2, ratio: float, fill_color: Color) -> void:
	HudDraw.status_bar(self, rect, ratio, fill_color, HUD_BG, HUD_BORDER)
