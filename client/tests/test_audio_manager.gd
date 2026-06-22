class_name TestAudioManager
extends GdUnitTestSuite

## Tests for the SFX AudioManager autoload: the player pool is built, the registry
## resolves lazily, a missing/unknown sound is a silent no-op (the system is inert
## until CC0 assets are dropped in), and play_3d positions the chosen player.
## Audio files are not asserted (none ship yet) - only the dispatch logic is.

const STUB := &"__test_tone__"


func after_test() -> void:
	# Undo any pollution of the shared autoload singleton's state.
	AudioManager._cache.erase(STUB)
	AudioManager._warned.erase(STUB)
	AudioManager._last_play_ms.erase(STUB)
	for p in AudioManager._players_3d:
		p.stop()
		p.stream = null
	for p in AudioManager._players_ui:
		p.stop()
		p.stream = null
	AudioManager.stop_music()
	for t in AudioManager._duck_tweens:
		if is_instance_valid(t):
			t.kill()
	AudioManager._duck_tweens.clear()
	for p in AudioManager._ambiance_players.values():
		p.volume_db = 0.0
	AudioManager.stop_all_ambiance()


func test_pool_is_populated() -> void:
	assert_int(AudioManager._players_3d.size()).is_equal(AudioManager.POOL_SIZE_3D)
	assert_int(AudioManager._players_ui.size()).is_equal(AudioManager.POOL_SIZE_UI)


func test_next_3d_player_prefers_idle() -> void:
	var p := AudioManager._next_3d_player()
	assert_bool(p.playing).is_false()


func test_unknown_sound_is_silent_noop() -> void:
	# Unknown name: no crash, nothing assigned or playing.
	AudioManager.play_3d(&"__does_not_exist__", Vector3(1, 2, 3))
	for p in AudioManager._players_3d:
		assert_bool(p.playing).is_false()


func test_registry_lookup_returns_cached_stream() -> void:
	var stub := AudioStreamGenerator.new()
	AudioManager._cache[STUB] = stub
	assert_object(AudioManager._get_stream(STUB)).is_same(stub)


func test_play_3d_positions_chosen_player() -> void:
	var stub := AudioStreamGenerator.new()
	AudioManager._cache[STUB] = stub
	var pos := Vector3(4, 5, 6)
	AudioManager.play_3d(STUB, pos, 0.0)
	var hit: AudioStreamPlayer3D = null
	for p in AudioManager._players_3d:
		if p.stream == stub:
			hit = p
			break
	assert_object(hit).is_not_null()
	assert_vector(hit.global_position).is_equal(pos)


# --- footstep cadence ---


## Walk in a straight line and count accumulator resets (one per step fired).
func _walk_and_count_steps(step_dist: float, count: int, on_floor: bool) -> int:
	AudioManager._foot_has_pos = false
	AudioManager._foot_accum = 0.0
	var pos := Vector3.ZERO
	AudioManager.tick_footsteps(pos, on_floor, 5.0)  # prime position
	var steps := 0
	for i in count:
		pos.x += step_dist
		var before := AudioManager._foot_accum
		AudioManager.tick_footsteps(pos, on_floor, 5.0)
		if AudioManager._foot_accum < before:
			steps += 1
	return steps


func test_footsteps_fire_on_stride_cadence() -> void:
	# 10 m of travel at stride 2.0 -> ~5 steps.
	var steps := _walk_and_count_steps(0.5, 20, true)
	assert_int(steps).is_equal(5)


func test_footsteps_silent_while_airborne() -> void:
	assert_int(_walk_and_count_steps(0.5, 20, false)).is_equal(0)


func test_footsteps_silent_when_below_min_speed() -> void:
	AudioManager._foot_has_pos = false
	AudioManager._foot_accum = 0.0
	var pos := Vector3.ZERO
	AudioManager.tick_footsteps(pos, true, 5.0)
	var fired := false
	for i in 20:
		pos.x += 0.5
		var before := AudioManager._foot_accum
		AudioManager.tick_footsteps(pos, true, 0.0)  # standing still (speed 0)
		if AudioManager._foot_accum < before:
			fired = true
	assert_bool(fired).is_false()


func test_footsteps_teleport_does_not_burst() -> void:
	AudioManager._foot_has_pos = false
	AudioManager._foot_accum = 0.0
	AudioManager.tick_footsteps(Vector3.ZERO, true, 5.0)
	var before := AudioManager._foot_accum
	# A 50 m jump in one tick (zone transfer) must not accumulate.
	AudioManager.tick_footsteps(Vector3(50, 0, 0), true, 5.0)
	assert_float(AudioManager._foot_accum).is_equal(before)


# --- music (single looping track, Music bus) ---


func test_music_plays_and_loops_on_music_bus() -> void:
	AudioManager.play_music(&"lobby")
	assert_bool(AudioManager._music_player.playing).is_true()
	assert_str(AudioManager._music_player.bus).is_equal("Music")
	# Looping is forced on so the drone never falls silent at the seam.
	assert_bool(AudioManager._music_player.stream.loop).is_true()


func test_music_is_idempotent() -> void:
	AudioManager.play_music(&"lobby")
	var stream_first: AudioStream = AudioManager._music_player.stream
	AudioManager.play_music(&"lobby")  # second request must not restart
	assert_bool(AudioManager._music_player.playing).is_true()
	assert_object(AudioManager._music_player.stream).is_same(stream_first)


func test_intermittent_music_plays_without_looping() -> void:
	AudioManager.play_music_intermittent(&"hub", 60.0)
	assert_bool(AudioManager._music_player.playing).is_true()
	assert_bool(AudioManager._music_intermittent).is_true()
	# Intermittent tracks must NOT loop (they replay after a gap via the finished signal).
	assert_bool(AudioManager._music_player.stream.loop).is_false()


func test_play_music_clears_intermittent_mode() -> void:
	AudioManager.play_music_intermittent(&"hub", 60.0)
	AudioManager.play_music(&"lobby")  # switching to a looping track exits intermittent
	assert_bool(AudioManager._music_intermittent).is_false()
	assert_bool(AudioManager._music_player.stream.loop).is_true()


func test_stop_music_silences() -> void:
	AudioManager.play_music(&"lobby")
	AudioManager.stop_music()
	assert_bool(AudioManager._music_player.playing).is_false()
	assert_str(AudioManager._music_current).is_equal("")


# --- ambiance (looping environmental beds, Ambiance bus) ---


func test_ambiance_plays_and_loops_on_ambiance_bus() -> void:
	AudioManager.play_ambiance(&"arena")
	var p: AudioStreamPlayer = AudioManager._ambiance_players[&"arena"]
	assert_bool(p.playing).is_true()
	# Own bus so players can cut atmosphere while keeping gameplay SFX.
	assert_str(p.bus).is_equal("Ambiance")
	assert_bool(p.stream.loop).is_true()


func test_ambiance_is_idempotent() -> void:
	AudioManager.play_ambiance(&"arena")
	var p: AudioStreamPlayer = AudioManager._ambiance_players[&"arena"]
	var stream_first: AudioStream = p.stream
	AudioManager.play_ambiance(&"arena")  # second request must not restart
	assert_bool(p.playing).is_true()
	assert_object(p.stream).is_same(stream_first)


## Music and ambiance are independent: fight start stops the lobby music but the
## arena ambiance keeps playing.
func test_music_and_ambiance_are_independent() -> void:
	AudioManager.play_ambiance(&"arena")
	AudioManager.play_music(&"lobby")
	assert_bool(AudioManager._ambiance_players[&"arena"].playing).is_true()
	assert_bool(AudioManager._music_player.playing).is_true()
	AudioManager.stop_music()  # fight start
	assert_bool(AudioManager._music_player.playing).is_false()
	assert_bool(AudioManager._ambiance_players[&"arena"].playing).is_true()


## A cue dips the ambient bed (so it stands out) while leaving the cue at full level.
func test_cue_ducks_ambiance() -> void:
	AudioManager.play_ambiance(&"arena")
	var p: AudioStreamPlayer = AudioManager._ambiance_players[&"arena"]
	assert_float(p.volume_db).is_equal(0.0)
	AudioManager.play_cue(&"dungeon_start")
	await get_tree().create_timer(0.4).timeout
	assert_float(p.volume_db).is_less(-1.0)  # bed has been ducked under the cue


func test_stop_all_ambiance_silences_every_layer() -> void:
	AudioManager.play_ambiance(&"arena")
	AudioManager.stop_all_ambiance()
	for p in AudioManager._ambiance_players.values():
		assert_bool(p.playing).is_false()


func test_unknown_ambiance_is_noop() -> void:
	AudioManager.play_ambiance(&"__no_such_bed__")
	assert_bool(AudioManager._ambiance_players.has(&"__no_such_bed__")).is_false()
