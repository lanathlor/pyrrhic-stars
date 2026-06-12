extends SceneTree

## Bake Mixamo FBX animations into a single AnimationLibrary resource.
##
## Reads animation_manifest.yaml, imports each FBX, strips root motion,
## sets loop mode, and saves the result as mixamo_anims.tres.
##
## Manifest entries with `glb:` + `clip:` instead of `fbx:` are retargeted:
## the clip is sampled from its source rig (Quaternius universal rig) and
## converted onto the Mixamo skeleton via world-space rotation deltas, so
## the output plays directly on CharacterModel like any Mixamo clip.
##
## Run: godot4 --headless --path client --script res://scripts/tools/bake_animations.gd

const MANIFEST_PATH := "res://assets/animations/animation_manifest.yaml"
const OUTPUT_PATH := "res://assets/animations/mixamo_anims.tres"

const RETARGET_FPS := 30.0

## Quaternius universal rig (Rigify DEF- bones) -> Mixamo bone names.
const UAL_TO_MIXAMO := {
	"DEF-hips": "mixamorig_Hips",
	"DEF-spine.001": "mixamorig_Spine",
	"DEF-spine.002": "mixamorig_Spine1",
	"DEF-spine.003": "mixamorig_Spine2",
	"DEF-neck": "mixamorig_Neck",
	"DEF-head": "mixamorig_Head",
	"DEF-shoulder.L": "mixamorig_LeftShoulder",
	"DEF-upper_arm.L": "mixamorig_LeftArm",
	"DEF-forearm.L": "mixamorig_LeftForeArm",
	"DEF-hand.L": "mixamorig_LeftHand",
	"DEF-shoulder.R": "mixamorig_RightShoulder",
	"DEF-upper_arm.R": "mixamorig_RightArm",
	"DEF-forearm.R": "mixamorig_RightForeArm",
	"DEF-hand.R": "mixamorig_RightHand",
	"DEF-thigh.L": "mixamorig_LeftUpLeg",
	"DEF-shin.L": "mixamorig_LeftLeg",
	"DEF-foot.L": "mixamorig_LeftFoot",
	"DEF-toe.L": "mixamorig_LeftToeBase",
	"DEF-thigh.R": "mixamorig_RightUpLeg",
	"DEF-shin.R": "mixamorig_RightLeg",
	"DEF-foot.R": "mixamorig_RightFoot",
	"DEF-toe.R": "mixamorig_RightToeBase",
	"DEF-thumb.01.L": "mixamorig_LeftHandThumb1",
	"DEF-thumb.02.L": "mixamorig_LeftHandThumb2",
	"DEF-thumb.03.L": "mixamorig_LeftHandThumb3",
	"DEF-f_index.01.L": "mixamorig_LeftHandIndex1",
	"DEF-f_index.02.L": "mixamorig_LeftHandIndex2",
	"DEF-f_index.03.L": "mixamorig_LeftHandIndex3",
	"DEF-f_middle.01.L": "mixamorig_LeftHandMiddle1",
	"DEF-f_middle.02.L": "mixamorig_LeftHandMiddle2",
	"DEF-f_middle.03.L": "mixamorig_LeftHandMiddle3",
	"DEF-f_ring.01.L": "mixamorig_LeftHandRing1",
	"DEF-f_ring.02.L": "mixamorig_LeftHandRing2",
	"DEF-f_ring.03.L": "mixamorig_LeftHandRing3",
	"DEF-f_pinky.01.L": "mixamorig_LeftHandPinky1",
	"DEF-f_pinky.02.L": "mixamorig_LeftHandPinky2",
	"DEF-f_pinky.03.L": "mixamorig_LeftHandPinky3",
	"DEF-thumb.01.R": "mixamorig_RightHandThumb1",
	"DEF-thumb.02.R": "mixamorig_RightHandThumb2",
	"DEF-thumb.03.R": "mixamorig_RightHandThumb3",
	"DEF-f_index.01.R": "mixamorig_RightHandIndex1",
	"DEF-f_index.02.R": "mixamorig_RightHandIndex2",
	"DEF-f_index.03.R": "mixamorig_RightHandIndex3",
	"DEF-f_middle.01.R": "mixamorig_RightHandMiddle1",
	"DEF-f_middle.02.R": "mixamorig_RightHandMiddle2",
	"DEF-f_middle.03.R": "mixamorig_RightHandMiddle3",
	"DEF-f_ring.01.R": "mixamorig_RightHandRing1",
	"DEF-f_ring.02.R": "mixamorig_RightHandRing2",
	"DEF-f_ring.03.R": "mixamorig_RightHandRing3",
	"DEF-f_pinky.01.R": "mixamorig_RightHandPinky1",
	"DEF-f_pinky.02.R": "mixamorig_RightHandPinky2",
	"DEF-f_pinky.03.R": "mixamorig_RightHandPinky3",
}

# Retarget caches: source rig scenes by glb path, target rig from base_model.
var _src_rigs: Dictionary = {}  # glb path -> {"player": AnimationPlayer, "skeleton": Skeleton3D}
var _target_skeleton: Skeleton3D = null
var _keepalive: Array[Node] = []
var _base_model_path: String = ""


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
	_base_model_path = base_model_path

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

	var anim: Animation = null
	if entry.has("glb") and entry.has("clip"):
		anim = _extract_retargeted(entry["glb"], entry["clip"])
	elif fbx_path != "":
		anim = _extract_animation(fbx_path)
	else:
		push_warning("[BakeAnimations] Skipping '%s' — no fbx or glb/clip source" % anim_name)
		return false

	if not anim:
		push_warning("[BakeAnimations] Could not extract animation for '%s'" % anim_name)
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


# =============================================================================
# Retargeting (universal rig -> Mixamo)
# =============================================================================


func _extract_retargeted(glb_path: String, clip_name: String) -> Animation:
	var rig := _get_source_rig(glb_path)
	var target := _get_target_skeleton()
	if rig.is_empty() or not target:
		return null
	var player: AnimationPlayer = rig["player"]
	if not player.has_animation(clip_name):
		push_warning("[BakeAnimations] Clip '%s' not found in %s" % [clip_name, glb_path])
		return null
	return _retarget_clip(player.get_animation(clip_name), rig["skeleton"], target)


func _get_source_rig(glb_path: String) -> Dictionary:
	if glb_path in _src_rigs:
		return _src_rigs[glb_path]
	var scene := load(glb_path) as PackedScene
	if not scene:
		push_warning("[BakeAnimations] Could not load source rig %s" % glb_path)
		return {}
	var inst := scene.instantiate()
	_keepalive.append(inst)
	var player := _find_child_of_type(inst, "AnimationPlayer") as AnimationPlayer
	var skeleton := _find_child_of_type(inst, "Skeleton3D") as Skeleton3D
	if not player or not skeleton:
		push_warning("[BakeAnimations] No AnimationPlayer/Skeleton3D in %s" % glb_path)
		return {}
	var rig := {"player": player, "skeleton": skeleton}
	_src_rigs[glb_path] = rig
	return rig


func _get_target_skeleton() -> Skeleton3D:
	if _target_skeleton:
		return _target_skeleton
	var scene := load(_base_model_path) as PackedScene
	if not scene:
		return null
	var inst := scene.instantiate()
	_keepalive.append(inst)
	var skeleton := _find_child_of_type(inst, "Skeleton3D") as Skeleton3D
	if not skeleton:
		push_warning("[BakeAnimations] No Skeleton3D in base model %s" % _base_model_path)
		return null
	# Normalize Mixamo bone name variants to the library's mixamorig_ prefix
	for i in skeleton.get_bone_count():
		var bname := skeleton.get_bone_name(i)
		if bname.begins_with("mixamorig1_"):
			skeleton.set_bone_name(i, "mixamorig_" + bname.substr(len("mixamorig1_")))
		elif bname.begins_with("mixamorig:"):
			skeleton.set_bone_name(i, "mixamorig_" + bname.substr(len("mixamorig:")))
	_target_skeleton = skeleton
	return skeleton


## Compute global rest transforms for every bone of a skeleton.
func _global_rests(skel: Skeleton3D) -> Array[Transform3D]:
	var rests: Array[Transform3D] = []
	rests.resize(skel.get_bone_count())
	for i in skel.get_bone_count():
		var parent := skel.get_bone_parent(i)
		var local := skel.get_bone_rest(i)
		rests[i] = local if parent < 0 else rests[parent] * local
	return rests


## Map each source bone name to its rotation/position track index in the clip.
func _index_tracks(clip: Animation) -> Dictionary:
	var out := {"rot": {}, "pos": {}}
	for t in clip.get_track_count():
		var path := clip.track_get_path(t)
		if path.get_subname_count() == 0:
			continue
		var bone := str(path.get_subname(0))
		match clip.track_get_type(t):
			Animation.TYPE_ROTATION_3D:
				out["rot"][bone] = t
			Animation.TYPE_POSITION_3D:
				out["pos"][bone] = t
	return out


## Sample the source skeleton's global pose at time t (local tracks + FK).
func _sample_src_globals(
	clip: Animation, tracks: Dictionary, skel: Skeleton3D, t: float
) -> Array[Transform3D]:
	var globals: Array[Transform3D] = []
	globals.resize(skel.get_bone_count())
	for i in skel.get_bone_count():
		var bname := skel.get_bone_name(i)
		var rest := skel.get_bone_rest(i)
		var rot := rest.basis.get_rotation_quaternion()
		var pos := rest.origin
		if bname in tracks["rot"]:
			rot = clip.rotation_track_interpolate(tracks["rot"][bname], t)
		if bname in tracks["pos"]:
			pos = clip.position_track_interpolate(tracks["pos"][bname], t)
		var local := Transform3D(Basis(rot), pos)
		var parent := skel.get_bone_parent(i)
		globals[i] = local if parent < 0 else globals[parent] * local
	return globals


## Resolve the bone mapping into index pairs ordered parent-before-child.
func _resolve_mapping(src: Skeleton3D, dst: Skeleton3D) -> Array[Dictionary]:
	var pairs: Array[Dictionary] = []
	for src_name: String in UAL_TO_MIXAMO:
		var si := src.find_bone(src_name)
		var di := dst.find_bone(UAL_TO_MIXAMO[src_name])
		if si == -1 or di == -1:
			continue
		var depth := 0
		var p := dst.get_bone_parent(di)
		while p >= 0:
			depth += 1
			p = dst.get_bone_parent(p)
		pairs.append({"si": si, "di": di, "depth": depth})
	pairs.sort_custom(func(a, b): return a["depth"] < b["depth"])
	return pairs


func _retarget_clip(clip: Animation, src: Skeleton3D, dst: Skeleton3D) -> Animation:
	var tracks := _index_tracks(clip)
	var src_rests := _global_rests(src)
	var dst_rests := _global_rests(dst)
	var pairs := _resolve_mapping(src, dst)
	if pairs.is_empty():
		push_warning("[BakeAnimations] No mappable bones for retarget")
		return null

	var hips_si := src.find_bone("DEF-hips")
	var hips_di := dst.find_bone("mixamorig_Hips")
	var hip_scale := 1.0
	if hips_si >= 0 and hips_di >= 0 and src_rests[hips_si].origin.y > 0.001:
		hip_scale = dst_rests[hips_di].origin.y / src_rests[hips_si].origin.y

	var anim := Animation.new()
	anim.length = clip.length
	anim.step = 1.0 / RETARGET_FPS

	var rot_track_for := {}  # di -> track index
	for pair in pairs:
		var di: int = pair["di"]
		var tr := anim.add_track(Animation.TYPE_ROTATION_3D)
		anim.track_set_path(tr, NodePath("Skeleton3D:%s" % dst.get_bone_name(di)))
		rot_track_for[di] = tr
	var hips_pos_track := anim.add_track(Animation.TYPE_POSITION_3D)
	anim.track_set_path(hips_pos_track, NodePath("Skeleton3D:mixamorig_Hips"))

	var ctx := {
		"anim": anim,
		"clip": clip,
		"tracks": tracks,
		"src": src,
		"dst": dst,
		"pairs": pairs,
		"src_rests": src_rests,
		"dst_rests": dst_rests,
		"rot_track_for": rot_track_for,
		"hips_pos_track": hips_pos_track,
		"hips_si": hips_si,
		"hips_di": hips_di,
		"hip_scale": hip_scale,
	}
	var frame_count := int(ceil(clip.length * RETARGET_FPS)) + 1
	for f in frame_count:
		_retarget_frame(ctx, minf(f / RETARGET_FPS, clip.length))
	return anim


func _retarget_frame(ctx: Dictionary, t: float) -> void:
	var anim: Animation = ctx["anim"]
	var dst: Skeleton3D = ctx["dst"]
	var pairs: Array[Dictionary] = ctx["pairs"]
	var src_rests: Array[Transform3D] = ctx["src_rests"]
	var dst_rests: Array[Transform3D] = ctx["dst_rests"]
	var rot_track_for: Dictionary = ctx["rot_track_for"]
	var hips_si: int = ctx["hips_si"]
	var hips_di: int = ctx["hips_di"]
	var src_globals := _sample_src_globals(ctx["clip"], ctx["tracks"], ctx["src"], t)
	var dst_global_basis := {}  # di -> Basis (this frame)

	for pair in pairs:
		var si: int = pair["si"]
		var di: int = pair["di"]
		# World-space rotation delta from rest, applied to the target's rest.
		var delta := src_globals[si].basis * src_rests[si].basis.inverse()
		var gb := delta * dst_rests[di].basis
		dst_global_basis[di] = gb

		var parent := dst.get_bone_parent(di)
		var parent_basis: Basis = (
			dst_global_basis[parent]
			if parent in dst_global_basis
			else (dst_rests[parent].basis if parent >= 0 else Basis())
		)
		var local_rot := (parent_basis.inverse() * gb).get_rotation_quaternion().normalized()
		anim.rotation_track_insert_key(rot_track_for[di], t, local_rot)

		if di == hips_di and si == hips_si:
			var world_off: Vector3 = (
				(src_globals[si].origin - src_rests[si].origin) * ctx["hip_scale"]
			)
			var hips_global_pos := dst_rests[di].origin + world_off
			var parent_rest := dst_rests[parent] if parent >= 0 else Transform3D()
			var local_pos := parent_rest.affine_inverse() * hips_global_pos
			anim.position_track_insert_key(ctx["hips_pos_track"], t, local_pos)
