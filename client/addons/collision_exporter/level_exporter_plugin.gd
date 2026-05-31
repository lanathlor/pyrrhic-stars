@tool
extends EditorPlugin
## Auto-exports level data to JSON for the Go server whenever a scene is saved.
##
## Delegates all extraction and serialization to LevelExportLib.
## Enable in Project > Project Settings > Plugins.

var _gizmo_plugin: EditorNode3DGizmoPlugin


func _enter_tree() -> void:
	_gizmo_plugin = preload("res://addons/collision_exporter/level_marker_gizmo.gd").new()
	add_node_3d_gizmo_plugin(_gizmo_plugin)
	scene_saved.connect(_on_scene_saved)
	print("LevelExporter: plugin loaded — gizmos + auto-export on save")


func _exit_tree() -> void:
	scene_saved.disconnect(_on_scene_saved)
	remove_node_3d_gizmo_plugin(_gizmo_plugin)


func _on_scene_saved(path: String) -> void:
	var root := get_editor_interface().get_edited_scene_root()
	if root == null:
		return
	if root.scene_file_path != path:
		return
	if not _has_server_nodes(root):
		return

	var data := LevelExportLib.extract_level(root, root.scene_file_path)
	var output_dir := ProjectSettings.globalize_path("res://") + "../shared/levels/"
	var output_path := LevelExportLib.write_level(data, output_dir)
	if output_path.is_empty():
		return

	var poly_count: int = 0
	var nm: Dictionary = data.get("navmesh", {})
	if nm.has("polygons"):
		poly_count = nm["polygons"].size()
	print("LevelExporter: exported %s (%d obstacles, %d navmesh polys)" % [
		output_path, data["obstacles"].size(), poly_count])


func _has_server_nodes(node: Node) -> bool:
	if node.is_in_group("server_ignore"):
		return false
	if node is CSGBox3D:
		return true
	for g in node.get_groups():
		if str(g).begins_with("server_"):
			return true
	for child in node.get_children():
		if _has_server_nodes(child):
			return true
	return false
