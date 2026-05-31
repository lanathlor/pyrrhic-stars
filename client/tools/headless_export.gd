extends SceneTree
## Headless full level export — extracts all zone data + bakes navmesh.
## Produces identical output to the editor plugin's on-save export.
##
## Usage: godot --headless --script tools/headless_export.gd

const Lib := preload("res://addons/collision_exporter/level_export_lib.gd")

const SCENES: Dictionary = {
	"arena": "res://scenes/environments/arena/arena.tscn",
	"hub": "res://scenes/environments/prime_hub/military_building.tscn",
}


func _init() -> void:
	for zone_name in SCENES:
		_export_scene(zone_name, SCENES[zone_name])
	quit()


func _export_scene(zone_name: String, scene_path: String) -> void:
	var packed := load(scene_path) as PackedScene
	if packed == null:
		printerr("headless_export: cannot load %s" % scene_path)
		return
	var root := packed.instantiate()
	if root == null:
		printerr("headless_export: cannot instantiate %s" % scene_path)
		return

	var data := Lib.extract_level(root, scene_path)
	root.free()

	var output_dir := ProjectSettings.globalize_path("res://") + "../shared/levels/"
	var output_path := Lib.write_level(data, output_dir)
	if output_path.is_empty():
		return

	var poly_count: int = 0
	var nm: Dictionary = data.get("navmesh", {})
	if nm.has("polygons"):
		poly_count = nm["polygons"].size()
	print("headless_export: %s — %d obstacles, %d navmesh polys" % [
		zone_name, data["obstacles"].size(), poly_count])
