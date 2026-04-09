extends Node3D

## Shared Mixamo character model wrapper.
## Loads a base FBX for the mesh/skeleton, imports additional animations from
## other FBX files, and exposes a simple play_anim() API.

const BASE_MODEL := "res://assets/models/characters/Idle.fbx"

## Map of logical name → FBX path. Controllers call play_anim("run"), etc.
## Each FBX's mixamo animation is imported and added to the AnimationPlayer.
const ANIM_SOURCES := {
	# Base
	"idle":              "res://assets/models/characters/Idle.fbx",
	"run":               "res://assets/models/characters/Running.fbx",
	"jump":              "res://assets/models/characters/Jump.fbx",
	"roll":              "res://assets/models/characters/Stand To Roll.fbx",
	# Rifle (old standalone)
	"rifle_idle":        "res://assets/models/characters/Rifle Idle.fbx",
	"rifle_run":         "res://assets/models/characters/Rifle Run.fbx",
	"rifle_jump":        "res://assets/models/characters/Rifle Jump.fbx",
	"rifle_shoot":       "res://assets/models/characters/Gunplay.fbx",
	# Rifle (aiming set)
	"rifle_aim_idle":    "res://assets/models/characters/rifle/idle aiming.fbx",
	"rifle_aim_walk":    "res://assets/models/characters/rifle/walk forward.fbx",
	"rifle_aim_run":     "res://assets/models/characters/rifle/run forward.fbx",
	# Great sword
	"slash":             "res://assets/models/characters/Great Sword Slash.fbx",
	"sword_idle":        "res://assets/models/characters/sword/great sword idle.fbx",
	"sword_run":         "res://assets/models/characters/sword/great sword run.fbx",
	"sword_walk":        "res://assets/models/characters/sword/great sword walk.fbx",
	"sword_slash_1":     "res://assets/models/characters/sword/great sword slash.fbx",
	"sword_slash_2":     "res://assets/models/characters/sword/great sword slash (2).fbx",
	"sword_slash_3":     "res://assets/models/characters/sword/great sword slash (3).fbx",
	"sword_heavy":       "res://assets/models/characters/sword/great sword attack.fbx",
	"sword_block":       "res://assets/models/characters/sword/great sword blocking.fbx",
	"sword_impact":      "res://assets/models/characters/sword/great sword impact.fbx",
	"sword_jump":        "res://assets/models/characters/sword/great sword jump.fbx",
	"sword_spin":        "res://assets/models/characters/sword/great sword high spin attack.fbx",
}

## Which animations should loop (others play once).
const LOOPING_ANIMS := [
	"idle", "run",
	"rifle_idle", "rifle_run", "rifle_aim_idle", "rifle_aim_walk", "rifle_aim_run",
	"sword_idle", "sword_run", "sword_walk", "sword_block",
]

var _anim_player: AnimationPlayer = null
var _mesh_instances: Array[MeshInstance3D] = []
var _skeleton: Skeleton3D = null
var _root_bone_idx: int = -1
var _hips_bone_idx: int = -1
var _original_materials: Array[Material] = []
var _current_anim: String = ""
var _loaded_anims: PackedStringArray = []

## The currently attached weapon node (if any), for external rotation.
var weapon_node: Node3D = null
var _bone_attachment: BoneAttachment3D = null


func _ready() -> void:
	# Load base model (provides mesh + skeleton)
	var base_scene := load(BASE_MODEL) as PackedScene
	if not base_scene:
		push_warning("CharacterModel: could not load base model %s" % BASE_MODEL)
		return

	var instance := base_scene.instantiate()
	add_child(instance)

	# Mixamo models face +Z, Godot characters face -Z
	instance.rotation.y = PI

	# Find nodes in the imported scene tree
	_anim_player = _find_child_of_type(instance, "AnimationPlayer") as AnimationPlayer
	_mesh_instances = _find_all_mesh_instances(instance)
	_skeleton = _find_child_of_type(instance, "Skeleton3D") as Skeleton3D

	# Find root bone and hips bone to strip root motion
	if _skeleton:
		for i in _skeleton.get_bone_count():
			if _skeleton.get_bone_parent(i) == -1:
				_root_bone_idx = i
			var bone_name := _skeleton.get_bone_name(i).to_lower()
			if "hips" in bone_name:
				_hips_bone_idx = i

	for mesh in _mesh_instances:
		for i in mesh.get_surface_override_material_count():
			_original_materials.append(mesh.get_surface_override_material(i))

	if not _anim_player:
		return

	# Import animations from all FBX sources
	for anim_name in ANIM_SOURCES:
		var fbx_path: String = ANIM_SOURCES[anim_name]
		_import_anim_from_fbx(anim_name, fbx_path)

	# Start at idle
	if "idle" in _loaded_anims:
		play_anim("idle")


func _physics_process(_delta: float) -> void:
	# Strip root motion — pin root bone XZ so the model doesn't drift horizontally.
	# Keep Y so the hips rest height is preserved (feet stay on ground).
	if _skeleton and _root_bone_idx >= 0:
		var pose := _skeleton.get_bone_pose(_root_bone_idx)
		pose.origin.x = 0.0
		pose.origin.z = 0.0
		_skeleton.set_bone_pose(_root_bone_idx, pose)


## Play a named animation. Does nothing if already playing that anim.
func play_anim(anim_name: String, speed: float = 1.0) -> void:
	if not _anim_player:
		return
	if _current_anim == anim_name and _anim_player.speed_scale == speed:
		return
	if anim_name not in _loaded_anims:
		return
	_current_anim = anim_name
	_anim_player.speed_scale = speed
	_anim_player.play(anim_name)


## Play an animation scaled to fit a target duration in seconds.
## E.g. a 1.5s roll animation played with target_duration=0.4 → speed_scale=3.75
func play_anim_timed(anim_name: String, target_duration: float) -> void:
	if not _anim_player or anim_name not in _loaded_anims:
		return
	var anim := _anim_player.get_animation(anim_name)
	if not anim or target_duration <= 0.0:
		play_anim(anim_name)
		return
	var speed := anim.length / target_duration
	play_anim(anim_name, speed)


## Set playback speed of the current animation.
func set_animation_speed(speed: float) -> void:
	if _anim_player:
		_anim_player.speed_scale = speed


## Hide the entire model (for FPS — local player shouldn't see their own body).
func hide_model() -> void:
	# Hide the whole instanced scene (mesh + skeleton + everything)
	for child in get_children():
		if child is Node3D:
			child.visible = false


## Attach a weapon model to a skeleton bone (e.g. Mixamo's right hand).
## Returns the instanced weapon Node3D, or null on failure.
func attach_weapon(scene_path: String, bone_name: String = "mixamorig:RightHand",
		offset_pos: Vector3 = Vector3.ZERO, offset_rot: Vector3 = Vector3.ZERO) -> Node3D:
	if not _skeleton:
		push_warning("CharacterModel: no skeleton to attach weapon to")
		return null

	var bone_idx := _skeleton.find_bone(bone_name)
	if bone_idx == -1:
		push_warning("CharacterModel: bone '%s' not found" % bone_name)
		return null

	var weapon_scene := load(scene_path) as PackedScene
	if not weapon_scene:
		push_warning("CharacterModel: could not load weapon '%s'" % scene_path)
		return null

	# Remove previous weapon if any
	if _bone_attachment and is_instance_valid(_bone_attachment):
		_bone_attachment.queue_free()

	_bone_attachment = BoneAttachment3D.new()
	_bone_attachment.bone_name = bone_name
	_skeleton.add_child(_bone_attachment)

	weapon_node = weapon_scene.instantiate()
	weapon_node.position = offset_pos
	weapon_node.rotation = offset_rot
	_bone_attachment.add_child(weapon_node)

	return weapon_node


## Flash the mesh white for damage feedback.
var _flash_tween: Tween = null

func flash_damage(color: Color = Color(1.0, 1.0, 1.0), duration: float = 0.12) -> void:
	if _mesh_instances.is_empty():
		return
	# Kill any existing flash tween to avoid conflicts
	if _flash_tween and _flash_tween.is_valid():
		_flash_tween.kill()

	var flash_mat := StandardMaterial3D.new()
	flash_mat.albedo_color = color
	flash_mat.emission_enabled = true
	flash_mat.emission = color
	flash_mat.emission_energy_multiplier = 4.0
	flash_mat.shading_mode = BaseMaterial3D.SHADING_MODE_UNSHADED
	for mesh in _mesh_instances:
		for i in mesh.get_surface_override_material_count():
			mesh.set_surface_override_material(i, flash_mat)

	_flash_tween = get_tree().create_tween()
	_flash_tween.tween_interval(duration)
	_flash_tween.tween_callback(_clear_flash)


func _clear_flash() -> void:
	var mat_idx: int = 0
	for mesh in _mesh_instances:
		for i in mesh.get_surface_override_material_count():
			if mat_idx < _original_materials.size():
				mesh.set_surface_override_material(i, _original_materials[mat_idx])
			else:
				mesh.set_surface_override_material(i, null)
			mat_idx += 1


## Extract the mixamo animation from an FBX and add it to our AnimationPlayer.
func _import_anim_from_fbx(anim_name: String, fbx_path: String) -> void:
	var fbx_scene := load(fbx_path) as PackedScene
	if not fbx_scene:
		push_warning("CharacterModel: could not load animation %s from %s" % [anim_name, fbx_path])
		return

	# Temporarily instance the FBX to grab its AnimationPlayer
	var temp := fbx_scene.instantiate()
	var temp_anim_player := _find_child_of_type(temp, "AnimationPlayer") as AnimationPlayer
	if not temp_anim_player:
		temp.queue_free()
		return

	# Find the actual animation (skip "Take 001" / "RESET")
	var source_anim_name := ""
	for name in temp_anim_player.get_animation_list():
		if name != "Take 001" and name != "RESET":
			source_anim_name = name
			break
	if source_anim_name == "":
		# Fallback to first available
		var list := temp_anim_player.get_animation_list()
		if list.size() > 0:
			source_anim_name = list[0]

	if source_anim_name == "":
		temp.queue_free()
		return

	var anim := temp_anim_player.get_animation(source_anim_name)
	if not anim:
		temp.queue_free()
		return

	# Duplicate to avoid ownership issues
	var anim_copy := anim.duplicate()

	# Strip root motion: zero out horizontal (X/Z) position on root/hips tracks
	_strip_root_motion(anim_copy)

	# Set loop mode
	if anim_name in LOOPING_ANIMS:
		anim_copy.loop_mode = Animation.LOOP_LINEAR
	else:
		anim_copy.loop_mode = Animation.LOOP_NONE

	# Add to our AnimationPlayer under the logical name
	if _anim_player.has_animation(anim_name):
		_anim_player.get_animation_library("").remove_animation(anim_name)
	_anim_player.get_animation_library("").add_animation(anim_name, anim_copy)
	_loaded_anims.append(anim_name)

	temp.queue_free()


## Strip root motion from animation tracks.
## Finds position tracks for root/hips bones and zeros keyframes.
## If strip_y is true, also zeros Y (used for combat anims that move hips down).
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


## Recursively find all MeshInstance3D nodes (handles subclasses too).
func _find_all_mesh_instances(node: Node) -> Array[MeshInstance3D]:
	var result: Array[MeshInstance3D] = []
	if node is MeshInstance3D:
		result.append(node)
	for child in node.get_children():
		result.append_array(_find_all_mesh_instances(child))
	return result


## Recursively find first child node of a given class name.
func _find_child_of_type(node: Node, type_name: String) -> Node:
	if node.get_class() == type_name:
		return node
	for child in node.get_children():
		var found := _find_child_of_type(child, type_name)
		if found:
			return found
	return null
