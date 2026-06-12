class_name TestUalRetarget
extends GdUnitTestSuite

## Guards the UAL -> Mixamo retarget bake: the library must contain the
## retargeted clips with tracks that target Mixamo bones, so they play on
## CharacterModel exactly like native Mixamo clips.

const LIBRARY_PATH := "res://assets/animations/mixamo_anims.tres"
const MODEL_SCENE := "res://scenes/shared/character_model/character_model.tscn"

const EXPECTED_LOOPING := ["ual_idle", "ual_jog", "ual_sprint", "ual_spell_idle"]
const EXPECTED_ONESHOT := ["ual_death", "ual_hit_chest", "ual_sword_attack", "ual_spell_shoot"]


func test_retargeted_clips_present() -> void:
	var library := load(LIBRARY_PATH) as AnimationLibrary
	assert_object(library).is_not_null()
	for clip_name in EXPECTED_LOOPING + EXPECTED_ONESHOT:
		(
			assert_bool(library.has_animation(clip_name))
			. override_failure_message("missing retargeted clip: %s" % clip_name)
			. is_true()
		)


func test_loop_modes() -> void:
	var library := load(LIBRARY_PATH) as AnimationLibrary
	for clip_name in EXPECTED_LOOPING:
		var anim := library.get_animation(clip_name)
		assert_int(anim.loop_mode).is_equal(Animation.LOOP_LINEAR)
	for clip_name in EXPECTED_ONESHOT:
		var anim := library.get_animation(clip_name)
		assert_int(anim.loop_mode).is_equal(Animation.LOOP_NONE)


func test_tracks_target_mixamo_bones() -> void:
	var library := load(LIBRARY_PATH) as AnimationLibrary
	var anim := library.get_animation("ual_jog")
	assert_object(anim).is_not_null()
	assert_float(anim.length).is_greater(0.1)
	assert_int(anim.get_track_count()).is_greater(20)

	var has_hips_pos := false
	for t in anim.get_track_count():
		var path := str(anim.track_get_path(t))
		(
			assert_bool(path.begins_with("Skeleton3D:mixamorig_"))
			. override_failure_message(
				"track %d targets '%s', expected a mixamorig_ bone" % [t, path]
			)
			. is_true()
		)
		assert_int(anim.track_get_key_count(t)).is_greater(2)
		if anim.track_get_type(t) == Animation.TYPE_POSITION_3D and "Hips" in path:
			has_hips_pos = true
	assert_bool(has_hips_pos).override_failure_message("no hips position track").is_true()


func test_clips_play_on_character_model() -> void:
	var scene := load(MODEL_SCENE) as PackedScene
	var model: Node3D = auto_free(scene.instantiate()) as Node3D
	add_child(model)
	await get_tree().process_frame

	model.play_anim("ual_jog")
	await get_tree().process_frame
	var player: AnimationPlayer = model._anim_player
	assert_object(player).is_not_null()
	assert_str(player.current_animation).is_equal("ual_jog")
