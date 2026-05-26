extends SceneTree

## Bake Mixamo FBX animations into a single AnimationLibrary resource.
##
## Reads animation_manifest.yaml, imports each FBX, strips root motion,
## sets loop mode, and saves the result as mixamo_anims.tres.
##
## Run: godot4 --headless --path client --script res://scripts/tools/bake_animations.gd

const MANIFEST_PATH := "res://assets/animations/animation_manifest.yaml"
const OUTPUT_PATH := "res://assets/animations/mixamo_anims.tres"


func _init() -> void:
	_run_bake()
	quit()


func _run_bake() -> void:
	print("[BakeAnimations] Starting animation bake...")

	var manifest := _load_manifest()
	if manifest.is_empty():
		push_error("[BakeAnimations] Failed to load manifest from %s" % MANIFEST_PATH)
		return

	var base_model_path: String = manifest.get("base_model", "")
	if base_model_path == "":
		push_error("[BakeAnimations] No base_model in manifest")
		return

	var animations: Dictionary = manifest.get("animations", {})
	if animations.is_empty():
		push_error("[BakeAnimations] No animations in manifest")
		return

	var base_scene := load(base_model_path) as PackedScene
	if not base_scene:
		push_error("[BakeAnimations] Could not load base model: %s" % base_model_path)
		return

	var library := AnimationLibrary.new()
	var count := 0
	for anim_name: String in animations:
		if _bake_single_anim(library, anim_name, animations[anim_name]):
			count += 1

	var err := ResourceSaver.save(library, OUTPUT_PATH)
	if err != OK:
		push_error("[BakeAnimations] Failed to save library: error %d" % err)
		return
	print("[BakeAnimations] Done! Saved %d animations to %s" % [count, OUTPUT_PATH])


func _bake_single_anim(library: AnimationLibrary, anim_name: String, entry: Dictionary) -> bool:
	var fbx_path: String = entry.get("fbx", "")
	var should_loop: bool = entry.get("loop", false)

	if fbx_path == "":
		push_warning("[BakeAnimations] Skipping '%s' — no fbx path" % anim_name)
		return false

	var anim := _extract_animation(fbx_path)
	if not anim:
		push_warning("[BakeAnimations] Could not extract animation from '%s'" % fbx_path)
		return false

	var anim_copy := anim.duplicate() as Animation
	_strip_root_motion(anim_copy)
	anim_copy.loop_mode = Animation.LOOP_LINEAR if should_loop else Animation.LOOP_NONE

	var err := library.add_animation(anim_name, anim_copy)
	if err != OK:
		push_warning("[BakeAnimations] Failed to add '%s': error %d" % [anim_name, err])
		return false

	print(
		(
			"[BakeAnimations]   Baked: %s (%.2fs, %s)"
			% [anim_name, anim_copy.length, "loop" if should_loop else "once"]
		)
	)
	return true


## Extract the first real animation from an FBX file.
func _extract_animation(fbx_path: String) -> Animation:
	var fbx_scene := load(fbx_path) as PackedScene
	if not fbx_scene:
		return null

	var temp := fbx_scene.instantiate()
	var anim_player := _find_child_of_type(temp, "AnimationPlayer") as AnimationPlayer
	if not anim_player:
		temp.queue_free()
		return null

	# Find the actual animation — skip boilerplate names
	var source_name := ""
	for name: String in anim_player.get_animation_list():
		if name != "Take 001" and name != "RESET":
			source_name = name
			break
	if source_name == "":
		var list := anim_player.get_animation_list()
		if list.size() > 0:
			source_name = list[0]

	var anim: Animation = null
	if source_name != "":
		anim = anim_player.get_animation(source_name)

	temp.queue_free()
	return anim


## Strip root motion: zero X/Z position on hips/root bone tracks, preserve Y.
func _strip_root_motion(anim: Animation) -> void:
	for track_idx in anim.get_track_count():
		if anim.track_get_type(track_idx) != Animation.TYPE_POSITION_3D:
			continue
		var path := anim.track_get_path(track_idx)
		var path_str := str(path).to_lower()
		if "hips" not in path_str and "root" not in path_str:
			continue
		for key_idx in anim.track_get_key_count(track_idx):
			var pos: Vector3 = anim.track_get_key_value(track_idx, key_idx)
			pos.x = 0.0
			pos.z = 0.0
			anim.track_set_key_value(track_idx, key_idx, pos)


## Parse the animation manifest YAML (simple flat format only).
func _load_manifest() -> Dictionary:
	var file := FileAccess.open(MANIFEST_PATH, FileAccess.READ)
	if not file:
		return {}

	var result := {}
	var animations := {}
	var current_anim := ""
	var current_entry := {}
	var in_animations := false

	while not file.eof_reached():
		var line := file.get_line()

		# Skip comments and empty lines
		var stripped := line.strip_edges()
		if stripped == "" or stripped.begins_with("#"):
			continue

		var indent := line.length() - line.lstrip(" ").length()

		if indent == 0:
			# Top-level key
			_flush_anim(animations, current_anim, current_entry)
			current_anim = ""
			current_entry = {}

			if stripped == "animations:":
				in_animations = true
				continue

			var kv := _parse_kv(stripped)
			if kv.size() == 2:
				result[kv[0]] = _unquote(kv[1])
				in_animations = false

		elif indent == 2 and in_animations:
			# Animation name (e.g., "  idle:")
			_flush_anim(animations, current_anim, current_entry)
			current_entry = {}
			current_anim = stripped.trim_suffix(":")

		elif indent == 4 and in_animations and current_anim != "":
			# Animation property (e.g., "    fbx: ...")
			var kv := _parse_kv(stripped)
			if kv.size() == 2:
				var val: String = _unquote(kv[1])
				if val == "true":
					current_entry[kv[0]] = true
				elif val == "false":
					current_entry[kv[0]] = false
				else:
					current_entry[kv[0]] = val

	_flush_anim(animations, current_anim, current_entry)
	result["animations"] = animations
	return result


func _flush_anim(animations: Dictionary, name: String, entry: Dictionary) -> void:
	if name != "" and not entry.is_empty():
		animations[name] = entry


func _parse_kv(line: String) -> PackedStringArray:
	var idx := line.find(":")
	if idx == -1:
		return PackedStringArray()
	var key := line.substr(0, idx).strip_edges()
	var val := line.substr(idx + 1).strip_edges()
	return PackedStringArray([key, val])


func _unquote(s: String) -> String:
	if s.begins_with('"') and s.ends_with('"'):
		return s.substr(1, s.length() - 2)
	return s


func _find_child_of_type(node: Node, type_name: String) -> Node:
	if node.get_class() == type_name:
		return node
	for child in node.get_children():
		var found := _find_child_of_type(child, type_name)
		if found:
			return found
	return null
