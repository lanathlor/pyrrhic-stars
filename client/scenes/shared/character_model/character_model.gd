extends Node3D

## Shared Mixamo character model wrapper.
## Loads a pre-baked AnimationLibrary and exposes a simple play_anim() API.
## To regenerate the library, edit animation_manifest.yaml and run bake_animations.gd.

const DEFAULT_BASE_MODEL := "res://assets/animations/Idle.fbx"
const ANIM_LIBRARY := "res://assets/animations/mixamo_anims.tres"

## Override the base model FBX/GLB. Leave empty to use the default Mixamo Idle model.
@export var base_model_override: String = ""

## The currently attached weapon node (if any), for external rotation.
var weapon_node: Node3D = null

## AnimationTree and playback — set up via setup_state_machine().
var anim_tree: AnimationTree = null
var state_playback: AnimationNodeStateMachinePlayback = null

var _anim_player: AnimationPlayer = null
var _mesh_instances: Array[MeshInstance3D] = []
var _skeleton: Skeleton3D = null
var _root_bone_idx: int = -1
var _hips_bone_idx: int = -1
var _original_materials: Array[Material] = []
var _current_anim: String = ""
var _loaded_anims: PackedStringArray = []
var _bone_attachment: BoneAttachment3D = null
var _flash_tween: Tween = null


func _ready() -> void:
	var instance := _load_base_model()
	if not instance:
		return

	_anim_player = _find_child_of_type(instance, "AnimationPlayer") as AnimationPlayer
	_mesh_instances = _find_all_mesh_instances(instance)
	_skeleton = _find_child_of_type(instance, "Skeleton3D") as Skeleton3D

	# FBX/GLB models exported without animations lack an AnimationPlayer.
	# Create one as a sibling of Skeleton3D so the shared animation library
	# can be loaded, and set its root_node to Skeleton3D's parent so track
	# paths like "Skeleton3D:bone_name" resolve correctly.
	if not _anim_player and _skeleton:
		_anim_player = AnimationPlayer.new()
		_skeleton.get_parent().add_child(_anim_player)
		_anim_player.root_node = _anim_player.get_path_to(_skeleton.get_parent())

	_normalize_bone_names(instance)
	_find_root_and_hips_bones()

	for mesh in _mesh_instances:
		for i in mesh.get_surface_override_material_count():
			_original_materials.append(mesh.get_surface_override_material(i))

	if not _anim_player:
		return

	_load_animation_library()
	if "idle" in _loaded_anims:
		play_anim("idle")


func _load_base_model() -> Node:
	var model_path: String = (
		base_model_override if base_model_override != "" else DEFAULT_BASE_MODEL
	)
	var instance: Node = null
	if base_model_override == "":
		instance = get_node_or_null("BaseModel")
	if not instance:
		var old := get_node_or_null("BaseModel")
		if old:
			old.queue_free()
		var base_scene := load(model_path) as PackedScene
		if not base_scene:
			push_warning("CharacterModel: could not load base model %s" % model_path)
			return null
		instance = base_scene.instantiate()
		instance.name = "BaseModel"
		instance.rotation.y = PI
		add_child(instance)
	return instance


func _normalize_bone_names(instance: Node) -> void:
	if not _skeleton:
		return
	# Map all Mixamo bone name variants to the canonical "mixamorig_" prefix
	# used by the baked animation library.
	var needs_remap := false
	for i in _skeleton.get_bone_count():
		var bname := _skeleton.get_bone_name(i)
		var new_name := ""
		if bname.begins_with("mixamorig1_"):
			new_name = "mixamorig_" + bname.substr(len("mixamorig1_"))
		elif bname.begins_with("mixamorig:"):
			new_name = "mixamorig_" + bname.substr(len("mixamorig:"))
		if new_name != "":
			_skeleton.set_bone_name(i, new_name)
			needs_remap = true
	if needs_remap:
		for mesh in _find_all_mesh_instances(instance):
			var skin: Skin = mesh.skin
			if not skin:
				continue
			skin = skin.duplicate() as Skin
			for i in skin.get_bind_count():
				var bind_name := skin.get_bind_name(i)
				var new_bind := ""
				if bind_name.begins_with("mixamorig1_"):
					new_bind = "mixamorig_" + bind_name.substr(len("mixamorig1_"))
				elif bind_name.begins_with("mixamorig:"):
					new_bind = "mixamorig_" + bind_name.substr(len("mixamorig:"))
				if new_bind != "":
					skin.set_bind_name(i, new_bind)
			mesh.skin = skin


func _find_root_and_hips_bones() -> void:
	if not _skeleton:
		return
	for i in _skeleton.get_bone_count():
		if _skeleton.get_bone_parent(i) == -1:
			_root_bone_idx = i
		var bone_name := _skeleton.get_bone_name(i).to_lower()
		if "hips" in bone_name:
			_hips_bone_idx = i


func _load_animation_library() -> void:
	var library := load(ANIM_LIBRARY) as AnimationLibrary
	if library:
		if _anim_player.has_animation_library(""):
			_anim_player.remove_animation_library("")
		_anim_player.add_animation_library("", library)
		_loaded_anims = PackedStringArray(library.get_animation_list())
	else:
		push_warning("CharacterModel: could not load animation library %s" % ANIM_LIBRARY)


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


## Build an AnimationTree with a StateMachine from a state→clip mapping.
## Returns the StateMachinePlayback for travel()/start() calls.
## Call once from the controller's _ready() after character_model is ready.
##
## states: { "idle": "sword_idle", "run": "sword_run", "dodge": "roll", ... }
## crossfade: default crossfade time for transitions between states
func setup_state_machine(
	states: Dictionary, crossfade: float = 0.1
) -> AnimationNodeStateMachinePlayback:
	if not _anim_player:
		push_warning("CharacterModel: no AnimationPlayer — cannot create AnimationTree")
		return null

	var sm := AnimationNodeStateMachine.new()

	# Add a node for each state
	for state_name: String in states:
		var clip_name: String = states[state_name]
		var anim_node := AnimationNodeAnimation.new()
		anim_node.animation = clip_name
		sm.add_node(state_name, anim_node, Vector2(0, 0))

	# Add transitions between all pairs for crossfade blending
	var state_names: Array = states.keys()
	for i in state_names.size():
		for j in state_names.size():
			if i == j:
				continue
			var t := AnimationNodeStateMachineTransition.new()
			t.xfade_time = crossfade
			t.switch_mode = AnimationNodeStateMachineTransition.SWITCH_MODE_IMMEDIATE
			sm.add_transition(state_names[i], state_names[j], t)

	anim_tree = AnimationTree.new()
	anim_tree.tree_root = sm
	add_child(anim_tree)
	anim_tree.anim_player = anim_tree.get_path_to(_anim_player)
	# Set root_node to the Skeleton3D's parent so track paths like
	# "Skeleton3D:mixamorig_Hips" resolve correctly for both FBX (flat)
	# and GLB (has Armature wrapper) scene trees.
	if _skeleton:
		anim_tree.root_node = anim_tree.get_path_to(_skeleton.get_parent())
	anim_tree.active = true

	state_playback = anim_tree["parameters/playback"] as AnimationNodeStateMachinePlayback
	return state_playback


## Travel to a state in the AnimationTree state machine.
## Falls back to play_anim() if no state machine is set up.
func travel(state_name: String, speed: float = 1.0) -> void:
	if _anim_player:
		_anim_player.speed_scale = speed
	if state_playback:
		if state_playback.get_current_node() != state_name:
			state_playback.travel(state_name)
	else:
		play_anim(state_name, speed)


## Drive the locomotion blend from current planar speed: idle below a small
## deadzone, sprint near sprint_speed, run in between. Each clip is time-scaled
## so its foot speed roughly matches actual movement. Requires "idle", "run",
## and "sprint" states in the state machine.
func travel_locomotion(speed: float, run_speed: float, sprint_speed: float) -> void:
	if speed < 0.5:
		travel("idle")
	elif speed >= sprint_speed * 0.85:
		travel("sprint", clampf(speed / sprint_speed, 0.85, 1.2))
	else:
		travel("run", clampf(speed / run_speed, 0.6, 1.3))


## Travel to a state scaled to fit a target duration.
func travel_timed(state_name: String, target_duration: float) -> void:
	if not _anim_player:
		return
	var anim_name: String = state_name
	# Resolve clip name from state machine if available
	if anim_tree and anim_tree.tree_root is AnimationNodeStateMachine:
		var sm := anim_tree.tree_root as AnimationNodeStateMachine
		if sm.has_node(state_name):
			var node := sm.get_node(state_name)
			if node is AnimationNodeAnimation:
				anim_name = node.animation
	if _anim_player.has_animation(anim_name):
		var anim := _anim_player.get_animation(anim_name)
		if anim and target_duration > 0.0:
			var speed := anim.length / target_duration
			travel(state_name, speed)
			return
	travel(state_name)


## Hide the entire model (for FPS — local player shouldn't see their own body).
func hide_model() -> void:
	# Hide the whole instanced scene (mesh + skeleton + everything)
	for child in get_children():
		if child is Node3D:
			child.visible = false


## Attach a weapon model to a skeleton bone (e.g. Mixamo's right hand).
## Returns the instanced weapon Node3D, or null on failure.
func attach_weapon(
	scene_path: String,
	bone_name: String = "mixamorig:RightHand",
	offset_pos: Vector3 = Vector3.ZERO,
	offset_rot: Vector3 = Vector3.ZERO
) -> Node3D:
	# Weapons are purely visual — skip in headless mode (e.g. tests)
	if DisplayServer.get_name() == "headless":
		return null

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
