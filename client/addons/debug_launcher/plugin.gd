@tool
extends EditorPlugin
## Debug Launcher: editor dock for dev mode quick-start.
##
## Provides a dock with class selection and auto-detected zone.
## Writes .dev_config.json so main.gd knows which class/zone to use.
## Sets editor run args to include --dev automatically.

var _dock: Control


func _enter_tree() -> void:
	_dock = preload("res://addons/debug_launcher/debug_dock.gd").new()
	_dock.name = "Debug Launcher"
	add_control_to_dock(DOCK_SLOT_RIGHT_BL, _dock)
	scene_saved.connect(_on_scene_saved)

	# Ensure --dev is passed when running from editor.
	var current_args: String = ProjectSettings.get_setting("editor/run/main_run_args", "")
	if "--dev" not in current_args:
		if current_args.strip_edges() == "":
			ProjectSettings.set_setting("editor/run/main_run_args", "-- --dev")
		else:
			ProjectSettings.set_setting("editor/run/main_run_args", current_args + " --dev")
		print("DebugLauncher: set run args to include --dev")


func _exit_tree() -> void:
	scene_saved.disconnect(_on_scene_saved)
	remove_control_from_docks(_dock)
	_dock.queue_free()


func _on_scene_saved(path: String) -> void:
	if not _is_level_scene(path):
		return
	var zone_name: String = _infer_zone_name(path)
	_dock.set_zone(zone_name)
	_dock.save_config()
	print("DebugLauncher: zone updated to '%s' from %s" % [zone_name, path])


func _is_level_scene(path: String) -> bool:
	return "arena" in path or "hub" in path or "prime_hub" in path or "dungeon" in path


func _infer_zone_name(scene_path: String) -> String:
	if "hub" in scene_path or "prime_hub" in scene_path:
		return "hub"
	if "arena" in scene_path:
		return "arena"
	return scene_path.get_base_dir().get_file()
