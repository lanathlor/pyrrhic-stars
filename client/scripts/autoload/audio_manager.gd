extends Node

## Pooled sound-effects player. Spatial one-shots (combat impacts, ability casts)
## go through play_3d(); flat UI sounds through play_ui(). Logical sound names map
## to files under res://assets/audio/sfx/; a missing file warns once and no-ops, so
## the system is inert until real CC0 assets are dropped in (see assets/audio/LICENSES.md).
##
## All players route through the "SFX" bus (see default_bus_layout.tres), so the
## existing SFX settings slider controls their loudness.

const POOL_SIZE_3D := 16
const POOL_SIZE_UI := 3

## Tiny per-sound debounce so identical impacts in a dense fight don't stack into a buzz.
const SPAM_COOLDOWN_MS := 30

## 3D attenuation tuned for arena-scale distances (40x30 arena, tall buildings).
const UNIT_SIZE := 8.0
const MAX_DISTANCE := 80.0

## Ducking: while a one-shot cue plays, dip the ambient beds so the cue stands out
## ("prepare yourself" moment), then fade them back as the cue decays.
const CUE_DUCK_DB := -12.0
const CUE_DUCK_ATTACK := 0.3
const CUE_DUCK_RELEASE := 1.5

## Silence between hub-music plays (track ~96s -> ~5 min cycle). Tunable.
const HUB_MUSIC_GAP := 210.0

## Logical sound name -> file path. Files need not exist yet; drop CC0 .ogg/.wav in.
const SOUNDS := {
	# combat impacts (played at hit_pos by world_state_sync)
	&"impact_enemy": "res://assets/audio/sfx/combat/impact_enemy.ogg",
	&"impact_player": "res://assets/audio/sfx/combat/impact_player.ogg",
	&"heal": "res://assets/audio/sfx/combat/heal.ogg",
	# vanguard attack swings (all share the swing whoosh; the hit thud layers on top)
	&"vanguard_cleave": "res://assets/audio/sfx/abilities/vanguard_swing.ogg",
	&"vanguard_upheaval": "res://assets/audio/sfx/abilities/vanguard_swing.ogg",
	&"vanguard_vortex": "res://assets/audio/sfx/abilities/vanguard_swing.ogg",
	&"vanguard_execution": "res://assets/audio/sfx/abilities/vanguard_swing.ogg",
	&"vanguard_block": "res://assets/audio/sfx/abilities/vanguard_block.ogg",
	&"vanguard_dodge": "res://assets/audio/sfx/abilities/vanguard_dodge.ogg",
	# harmonist ability casts
	&"harmonist_cast": "res://assets/audio/sfx/abilities/harmonist_cast.ogg",
	&"harmonist_beam": "res://assets/audio/sfx/abilities/harmonist_beam.ogg",
	&"harmonist_zone": "res://assets/audio/sfx/abilities/harmonist_zone.ogg",
	&"gust_step": "res://assets/audio/sfx/abilities/gust_step.ogg",
	# gunner
	&"gunner_fire": "res://assets/audio/sfx/abilities/gunner_fire.ogg",
	# movement
	&"footstep": "res://assets/audio/sfx/movement/footstep.ogg",
	# non-spatial one-shot cues (event stingers, on the Ambiance bus)
	&"dungeon_start": "res://assets/audio/sfx/cue/dungeon_start.ogg",
	# music (single track at a time, on the Music bus)
	&"lobby": "res://assets/audio/music/lobby.ogg",  # looping drone (lobby)
	&"hub": "res://assets/audio/music/hub.ogg",  # play-with-gaps ambience (hub)
	# ambiance (looping environmental beds; multiple can layer, on the SFX bus)
	&"arena": "res://assets/audio/sfx/ambiance/arena.ogg",
	# ui
	&"ui_click": "res://assets/audio/sfx/ui/ui_click.ogg",
	&"ui_confirm": "res://assets/audio/sfx/ui/ui_confirm.ogg",
}

## Footstep cadence: one step per STRIDE_LENGTH metres of horizontal travel while
## grounded and moving faster than FOOTSTEP_MIN_SPEED.
const STRIDE_LENGTH := 2.0
const FOOTSTEP_MIN_SPEED := 0.6
## Single-tick moves larger than this are treated as teleports (zone transfer) and
## skipped, so we don't fire a burst of steps on respawn.
const FOOTSTEP_TELEPORT_DIST := 3.0

var _players_3d: Array[AudioStreamPlayer3D] = []
var _players_ui: Array[AudioStreamPlayer] = []
var _ambiance_players: Dictionary = {}  # StringName -> AudioStreamPlayer (one per layer)
var _music_player: AudioStreamPlayer = null
var _music_current: StringName = &""
var _music_intermittent: bool = false  # replay after a gap instead of looping
var _music_gap: float = 0.0
var _music_gap_timer: Timer = null
var _cue_player: AudioStreamPlayer = null
var _duck_tweens: Array[Tween] = []  # active ambiance-duck tweens (one per bed)
var _next_3d_idx: int = 0
var _next_ui_idx: int = 0

var _cache: Dictionary = {}  # StringName -> AudioStream (or null when missing)
var _warned: Dictionary = {}  # StringName -> true (warn-once guard)
var _last_play_ms: Dictionary = {}  # StringName -> int (spam debounce)

var _foot_last_pos: Vector3 = Vector3.ZERO
var _foot_has_pos: bool = false
var _foot_accum: float = 0.0


func _ready() -> void:
	for i in POOL_SIZE_3D:
		var p := AudioStreamPlayer3D.new()
		p.bus = &"SFX"
		p.unit_size = UNIT_SIZE
		p.max_distance = MAX_DISTANCE
		p.attenuation_model = AudioStreamPlayer3D.ATTENUATION_INVERSE_DISTANCE
		add_child(p)
		_players_3d.append(p)
	for i in POOL_SIZE_UI:
		var p := AudioStreamPlayer.new()
		p.bus = &"SFX"
		add_child(p)
		_players_ui.append(p)
	# Single music track at a time, on the Music bus.
	_music_player = AudioStreamPlayer.new()
	_music_player.bus = &"Music"
	add_child(_music_player)
	_music_player.finished.connect(_on_music_finished)
	# One-shot timer for the silence gap between intermittent music plays.
	_music_gap_timer = Timer.new()
	_music_gap_timer.one_shot = true
	add_child(_music_gap_timer)
	_music_gap_timer.timeout.connect(_on_music_gap_timeout)
	# Non-spatial one-shot event cues, on the Ambiance bus.
	_cue_player = AudioStreamPlayer.new()
	_cue_player.bus = &"Ambiance"
	add_child(_cue_player)


## Play a spatial one-shot at a world position. pitch_jitter randomizes pitch to
## avoid repeated identical hits sounding mechanical.
func play_3d(sound: StringName, pos: Vector3, pitch_jitter: float = 0.08) -> void:
	var stream := _get_stream(sound)
	if stream == null:
		return
	if _on_cooldown(sound):
		return
	var player := _next_3d_player()
	player.stream = stream
	player.global_position = pos
	player.pitch_scale = 1.0 + randf_range(-pitch_jitter, pitch_jitter)
	player.play()


## Play a non-spatial one-shot event cue (e.g. the "dungeon is starting" gate stinger).
## Heard everywhere at the same level, not positioned in the world.
func play_cue(sound: StringName) -> void:
	var stream := _get_stream(sound)
	if stream == null:
		return
	if "loop" in stream:
		stream.loop = false
	_cue_player.stream = stream
	_cue_player.play()
	_duck_ambiance_under_cue(stream.get_length())


## Dip the ambient beds while a cue plays, then fade them back. The cue is on the same
## bus but a different player, so dipping the bed players leaves the cue at full level.
func _duck_ambiance_under_cue(cue_len: float) -> void:
	for t in _duck_tweens:
		if is_instance_valid(t):
			t.kill()
	_duck_tweens.clear()
	var hold: float = maxf(cue_len * 0.5 - CUE_DUCK_ATTACK, 0.0)
	for player in _ambiance_players.values():
		if not player.playing:
			continue
		var tween := create_tween()
		tween.tween_property(player, "volume_db", CUE_DUCK_DB, CUE_DUCK_ATTACK)
		tween.tween_interval(hold)
		tween.tween_property(player, "volume_db", 0.0, CUE_DUCK_RELEASE)
		_duck_tweens.append(tween)


## Play a flat (non-positional) UI sound.
func play_ui(sound: StringName) -> void:
	var stream := _get_stream(sound)
	if stream == null:
		return
	if _on_cooldown(sound):
		return
	var player := _next_ui_player()
	player.stream = stream
	player.play()


## Start the looping background music track (single track at a time, on the Music bus).
## Idempotent: re-requesting the playing track is a no-op. Switching tracks replaces it.
func play_music(sound: StringName) -> void:
	if _music_current == sound and _music_player.playing:
		return
	var stream := _get_stream(sound)
	if stream == null:
		return
	if "loop" in stream:
		stream.loop = true
	_music_intermittent = false
	_music_gap_timer.stop()
	_music_current = sound
	_music_player.stream = stream
	_music_player.play()


## Play a music track once, then replay after a silence gap, repeating while it stays
## the current track. Suits tranquil tracks that fade to an ending (no loop seam, and
## the gaps avoid fatigue). Idempotent on the same track.
func play_music_intermittent(sound: StringName, gap: float) -> void:
	if _music_current == sound:
		return
	var stream := _get_stream(sound)
	if stream == null:
		return
	if "loop" in stream:
		stream.loop = false
	_music_intermittent = true
	_music_gap = gap
	_music_gap_timer.stop()
	_music_current = sound
	_music_player.stream = stream
	_music_player.play()


## Stop the background music track (and cancel any pending intermittent replay).
func stop_music() -> void:
	_music_current = &""
	_music_intermittent = false
	if _music_gap_timer:
		_music_gap_timer.stop()
	if _music_player and _music_player.playing:
		_music_player.stop()


func _on_music_finished() -> void:
	# Only intermittent tracks end (looping ones never fire finished); wait, then replay.
	if _music_intermittent and _music_current != &"":
		_music_gap_timer.start(_music_gap)


func _on_music_gap_timeout() -> void:
	if _music_intermittent and _music_current != &"" and _music_player.stream != null:
		_music_player.play()


## Start a looping environmental ambient bed as its own layer, on the Ambiance bus (a
## separate slider so players can cut atmosphere while keeping gameplay SFX). Multiple
## beds can play at once. Idempotent: re-requesting a playing layer is a no-op, so it
## survives repeated state entries without restarting. Forces the stream to loop (the
## .ogg is authored as a seamless loop).
func play_ambiance(sound: StringName) -> void:
	var existing: AudioStreamPlayer = _ambiance_players.get(sound)
	if existing != null and existing.playing:
		return
	var stream := _get_stream(sound)
	if stream == null:
		return
	if "loop" in stream:
		stream.loop = true
	var player: AudioStreamPlayer = existing
	if player == null:
		player = AudioStreamPlayer.new()
		player.bus = &"Ambiance"
		add_child(player)
		_ambiance_players[sound] = player
	player.stream = stream
	player.play()


## Stop one ambient bed layer.
func stop_ambiance(sound: StringName) -> void:
	var player: AudioStreamPlayer = _ambiance_players.get(sound)
	if player != null and player.playing:
		player.stop()


## Stop every ambient bed (e.g. when leaving the zone for the hub or menu).
func stop_all_ambiance() -> void:
	for player in _ambiance_players.values():
		if player.playing:
			player.stop()


## Drive footstep cadence from a controller's _physics_process. Call once per physics
## frame for the LOCAL player only, after move_and_slide(). Plays a spatial step every
## STRIDE_LENGTH metres travelled while grounded and moving. Only one player feeds this
## (the local one), so a single accumulator suffices.
func tick_footsteps(pos: Vector3, on_floor: bool, planar_speed: float) -> void:
	if not on_floor or planar_speed < FOOTSTEP_MIN_SPEED:
		# Idle/airborne: arm the accumulator so a step lands as soon as we move again.
		_foot_accum = STRIDE_LENGTH
		_foot_last_pos = pos
		_foot_has_pos = true
		return
	if not _foot_has_pos:
		_foot_last_pos = pos
		_foot_has_pos = true
		return
	var moved := pos.distance_to(_foot_last_pos)
	_foot_last_pos = pos
	if moved > FOOTSTEP_TELEPORT_DIST:
		return  # teleport / zone transfer
	_foot_accum += moved
	if _foot_accum >= STRIDE_LENGTH:
		_foot_accum -= STRIDE_LENGTH
		play_3d(&"footstep", pos, 0.12)


## Resolve a logical name to its AudioStream, loading lazily and caching the result
## (including null for missing files, so we neither reload nor re-warn).
func _get_stream(sound: StringName) -> AudioStream:
	if _cache.has(sound):
		return _cache[sound]
	var path: String = SOUNDS.get(sound, "")
	var stream: AudioStream = null
	if path != "" and ResourceLoader.exists(path):
		stream = load(path) as AudioStream
	if stream == null and not _warned.has(sound):
		_warned[sound] = true
		push_warning("AudioManager: no audio registered for '%s' (expected %s)" % [sound, path])
	_cache[sound] = stream
	return stream


## Prefer an idle player; otherwise round-robin (oldest is most likely to be reused).
func _next_3d_player() -> AudioStreamPlayer3D:
	for p in _players_3d:
		if not p.playing:
			return p
	var p := _players_3d[_next_3d_idx]
	_next_3d_idx = (_next_3d_idx + 1) % _players_3d.size()
	return p


func _next_ui_player() -> AudioStreamPlayer:
	for p in _players_ui:
		if not p.playing:
			return p
	var p := _players_ui[_next_ui_idx]
	_next_ui_idx = (_next_ui_idx + 1) % _players_ui.size()
	return p


func _on_cooldown(sound: StringName) -> bool:
	var now := Time.get_ticks_msec()
	var last: int = _last_play_ms.get(sound, -SPAM_COOLDOWN_MS - 1)
	if now - last < SPAM_COOLDOWN_MS:
		return true
	_last_play_ms[sound] = now
	return false
