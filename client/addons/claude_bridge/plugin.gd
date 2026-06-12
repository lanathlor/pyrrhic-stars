@tool
extends EditorPlugin

## File-based editor control bridge so tooling (Claude Code) can drive the
## editor: open scenes, inspect and edit node properties, select nodes, and
## render screenshots of the edited world from arbitrary camera positions.
##
## Commands:  client/editor_bridge/commands.json - JSON object or array of
##            objects, each {"cmd": "..."}. The file is deleted after reading.
## Results:   client/editor_bridge/result.json - array, one entry per command.
## Shots:     client/editor_bridge/shots/<name>.png
##
## Verbs:
##   {"cmd": "status"}
##   {"cmd": "open_scene", "path": "res://scenes/environments/arena/arena.tscn"}
##   {"cmd": "list_tree", "path": "Geometry", "depth": 3}
##   {"cmd": "inspect", "path": "Boss", "props": ["position", "mesh:material:albedo_color"]}
##   {"cmd": "set", "path": "Sun", "props": {"light_energy": 1.4, "light_color": "#ffe9c4"}}
##   {"cmd": "select", "path": "Portal"}
##   {"cmd": "save_scene"}
##   {"cmd": "screenshot", "name": "arena_north"}                       <- editor camera view
##   {"cmd": "screenshot", "name": "x", "pos": [0, 10, 30], "look_at": [0, 2, 0],
##    "fov": 60, "width": 1920, "height": 1080}                         <- free camera

const BRIDGE_DIR := "res://editor_bridge/"
const SHOTS_DIR := BRIDGE_DIR + "shots/"
const POLL_FRAMES := 10

var _tick: int = 0
var _busy: bool = false


func _enter_tree() -> void:
	DirAccess.make_dir_recursive_absolute(ProjectSettings.globalize_path(SHOTS_DIR))
	var ignore_path := ProjectSettings.globalize_path(BRIDGE_DIR + ".gdignore")
	if not FileAccess.file_exists(ignore_path):
		var f := FileAccess.open(ignore_path, FileAccess.WRITE)
		if f:
			f.store_string("")
	print("[ClaudeBridge] watching %s" % ProjectSettings.globalize_path(BRIDGE_DIR + "commands.json"))


func _process(_delta: float) -> void:
	_tick += 1
	if _busy or _tick % POLL_FRAMES != 0:
		return
	var path := ProjectSettings.globalize_path(BRIDGE_DIR + "commands.json")
	if not FileAccess.file_exists(path):
		return
	var f := FileAccess.open(path, FileAccess.READ)
	if f == null:
		return
	var text := f.get_as_text()
	f.close()
	DirAccess.remove_absolute(path)
	if text.strip_edges().is_empty():
		return

	var json := JSON.new()
	if json.parse(text) != OK:
		_write_result([{"ok": false, "error": "JSON parse: %s" % json.get_error_message()}])
		return
	_run_commands(json.data)


func _run_commands(data: Variant) -> void:
	_busy = true
	var cmds: Array = data if data is Array else [data]
	var results: Array = []
	for c in cmds:
		if c is Dictionary:
			results.append(await _run_one(c))
		else:
			results.append({"ok": false, "error": "command must be a JSON object"})
	_write_result(results)
	_busy = false


func _run_one(cmd: Dictionary) -> Dictionary:
	var verb := str(cmd.get("cmd", ""))
	var result: Dictionary
	match verb:
		"status":
			result = _cmd_status()
		"open_scene":
			result = await _cmd_open_scene(cmd)
		"list_tree":
			result = _cmd_list_tree(cmd)
		"inspect":
			result = _cmd_inspect(cmd)
		"set":
			result = _cmd_set(cmd)
		"select":
			result = _cmd_select(cmd)
		"save_scene":
			result = _cmd_save_scene()
		"screenshot":
			result = await _cmd_screenshot(cmd)
		_:
			result = {"ok": false, "error": "unknown cmd: %s" % verb}
	return result


# =============================================================================
# Commands
# =============================================================================


func _cmd_status() -> Dictionary:
	var root := EditorInterface.get_edited_scene_root()
	return {
		"ok": true,
		"cmd": "status",
		"scene": root.scene_file_path if root else null,
		"unsaved": root != null and EditorInterface.get_edited_scene_root().scene_file_path == "",
	}


func _cmd_open_scene(cmd: Dictionary) -> Dictionary:
	var path := str(cmd.get("path", ""))
	if not ResourceLoader.exists(path):
		return {"ok": false, "error": "scene not found: %s" % path}
	EditorInterface.open_scene_from_path(path)
	for i in 3:
		await get_tree().process_frame
	return {"ok": true, "cmd": "open_scene", "scene": path}


func _cmd_list_tree(cmd: Dictionary) -> Dictionary:
	var node := _resolve_node(cmd)
	if node == null:
		return _node_error(cmd)
	var depth := int(cmd.get("depth", 3))
	return {"ok": true, "cmd": "list_tree", "tree": _tree_dump(node, depth)}


func _cmd_inspect(cmd: Dictionary) -> Dictionary:
	var node := _resolve_node(cmd)
	if node == null:
		return _node_error(cmd)

	var out := {
		"ok": true,
		"cmd": "inspect",
		"name": str(node.name),
		"type": node.get_class(),
		"children": node.get_child_count(),
	}
	var script: Script = node.get_script()
	if script:
		out["script"] = script.resource_path
	if node is Node3D:
		out["position"] = var_to_str(node.position)
		out["rotation_deg"] = var_to_str(node.rotation_degrees)
		out["scale"] = var_to_str(node.scale)
		out["visible"] = node.visible
	if node is MeshInstance3D and node.mesh:
		out["mesh"] = node.mesh.resource_path

	var props := {}
	for p in cmd.get("props", []):
		props[str(p)] = var_to_str(node.get_indexed(NodePath(str(p))))
	if not props.is_empty():
		out["props"] = props
	return out


func _cmd_set(cmd: Dictionary) -> Dictionary:
	var node := _resolve_node(cmd)
	if node == null:
		return _node_error(cmd)
	var props := cmd.get("props", {}) as Dictionary
	if props.is_empty():
		return {"ok": false, "error": "set: no props given"}

	var applied := {}
	for p in props:
		var prop_path := NodePath(str(p))
		var current: Variant = node.get_indexed(prop_path)
		node.set_indexed(prop_path, _coerce(current, props[p]))
		applied[str(p)] = var_to_str(node.get_indexed(prop_path))
	EditorInterface.mark_scene_as_unsaved()
	return {"ok": true, "cmd": "set", "applied": applied}


func _cmd_select(cmd: Dictionary) -> Dictionary:
	var node := _resolve_node(cmd)
	if node == null:
		return _node_error(cmd)
	var sel := EditorInterface.get_selection()
	sel.clear()
	sel.add_node(node)
	return {"ok": true, "cmd": "select", "selected": str(node.name)}


func _cmd_save_scene() -> Dictionary:
	var err := EditorInterface.save_scene()
	return {"ok": err == OK, "cmd": "save_scene", "error_code": err}


func _cmd_screenshot(cmd: Dictionary) -> Dictionary:
	var shot_name := str(cmd.get("name", "shot_%d" % _tick)).trim_suffix(".png") + ".png"
	var out_path := ProjectSettings.globalize_path(SHOTS_DIR + shot_name)

	var img: Image
	if cmd.has("pos"):
		img = await _render_free_camera(cmd)
	else:
		var vp := EditorInterface.get_editor_viewport_3d(0)
		img = vp.get_texture().get_image()
	if img == null:
		return {"ok": false, "error": "screenshot: render produced no image"}

	img.save_png(out_path)
	return {
		"ok": true,
		"cmd": "screenshot",
		"file": out_path,
		"size": [img.get_width(), img.get_height()],
	}


# =============================================================================
# Free camera rendering
# =============================================================================


func _render_free_camera(cmd: Dictionary) -> Image:
	var world := EditorInterface.get_editor_viewport_3d(0).find_world_3d()
	if world == null:
		return null

	var vp := SubViewport.new()
	vp.size = Vector2i(int(cmd.get("width", 1280)), int(cmd.get("height", 720)))
	vp.render_target_update_mode = SubViewport.UPDATE_ALWAYS
	vp.world_3d = world
	add_child(vp)

	var cam := Camera3D.new()
	vp.add_child(cam)
	cam.current = true
	cam.fov = float(cmd.get("fov", 60.0))
	cam.global_position = _to_v3(cmd["pos"])
	if cmd.has("look_at"):
		var target := _to_v3(cmd["look_at"])
		if not cam.global_position.is_equal_approx(target):
			cam.look_at(target)

	for i in 4:
		await get_tree().process_frame

	var img := vp.get_texture().get_image()
	vp.queue_free()
	return img


# =============================================================================
# Helpers
# =============================================================================


func _resolve_node(cmd: Dictionary) -> Node:
	var root := EditorInterface.get_edited_scene_root()
	if root == null:
		return null
	var path := str(cmd.get("path", "."))
	if path == "." or path.is_empty():
		return root
	return root.get_node_or_null(path)


func _node_error(cmd: Dictionary) -> Dictionary:
	if EditorInterface.get_edited_scene_root() == null:
		return {"ok": false, "error": "no scene open in editor"}
	return {"ok": false, "error": "node not found: %s" % cmd.get("path", ".")}


func _tree_dump(node: Node, depth: int) -> Dictionary:
	var d := {"name": str(node.name), "type": node.get_class()}
	if node is Node3D and not node.visible:
		d["visible"] = false
	if depth > 0 and node.get_child_count() > 0:
		var kids := []
		for c in node.get_children():
			kids.append(_tree_dump(c, depth - 1))
		d["children"] = kids
	elif node.get_child_count() > 0:
		d["children_omitted"] = node.get_child_count()
	return d


func _coerce(current: Variant, value: Variant) -> Variant:
	var out: Variant = null
	match typeof(current):
		TYPE_VECTOR3:
			if value is Array and value.size() == 3:
				out = Vector3(value[0], value[1], value[2])
		TYPE_VECTOR2:
			if value is Array and value.size() == 2:
				out = Vector2(value[0], value[1])
		TYPE_COLOR:
			if value is String:
				out = Color(value)
			elif value is Array and value.size() >= 3:
				out = Color(value[0], value[1], value[2], value[3] if value.size() > 3 else 1.0)
		TYPE_FLOAT:
			out = float(value)
		TYPE_INT:
			if value is float or value is int:
				out = int(value)
		TYPE_BOOL:
			out = bool(value)
	if out == null and value is String:
		var parsed: Variant = str_to_var(value)
		if parsed != null and typeof(parsed) == typeof(current):
			out = parsed
	if out == null:
		out = value
	return out


func _to_v3(value: Variant) -> Vector3:
	if value is Array and value.size() == 3:
		return Vector3(value[0], value[1], value[2])
	return Vector3.ZERO


func _write_result(results: Array) -> void:
	var f := FileAccess.open(
		ProjectSettings.globalize_path(BRIDGE_DIR + "result.json"), FileAccess.WRITE
	)
	if f:
		f.store_string(JSON.stringify(results, "  "))
