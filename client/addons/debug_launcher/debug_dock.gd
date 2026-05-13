@tool
extends VBoxContainer
## Debug Launcher dock: class selector + auto-detected zone display.
## Writes .dev_config.json for main.gd to read on F5.

const CLASSES := ["gunner", "vanguard", "blade_dancer"]
const CLASS_LABELS := ["Gunner (FPS)", "Vanguard (Souls)", "Blade Dancer (State Machine)"]
const CONFIG_PATH := "res://.dev_config.json"

var _class_option: OptionButton
var _zone_label: Label
var _selected_class: String = "gunner"
var _detected_zone: String = "arena"


func _ready() -> void:
	name = "DebugLauncher"
	custom_minimum_size = Vector2(200, 0)

	# Title
	var title := Label.new()
	title.text = "Debug Launcher"
	title.add_theme_font_size_override("font_size", 14)
	add_child(title)

	# Separator
	add_child(HSeparator.new())

	# Class selector
	var class_label := Label.new()
	class_label.text = "Class"
	class_label.add_theme_font_size_override("font_size", 12)
	add_child(class_label)

	_class_option = OptionButton.new()
	for i in range(CLASSES.size()):
		_class_option.add_item(CLASS_LABELS[i], i)
	_class_option.item_selected.connect(_on_class_selected)
	add_child(_class_option)

	# Zone display
	add_child(HSeparator.new())

	var zone_title := Label.new()
	zone_title.text = "Zone (auto-detected)"
	zone_title.add_theme_font_size_override("font_size", 12)
	add_child(zone_title)

	_zone_label = Label.new()
	_zone_label.text = _detected_zone
	_zone_label.add_theme_font_size_override("font_size", 13)
	add_child(_zone_label)

	# Hint
	add_child(HSeparator.new())
	var hint := Label.new()
	hint.text = "Save a level scene to update zone.\nPress F5 to launch."
	hint.add_theme_font_size_override("font_size", 10)
	hint.autowrap_mode = TextServer.AUTOWRAP_WORD
	add_child(hint)

	# Load existing config
	_load_config()


func set_zone(zone_name: String) -> void:
	_detected_zone = zone_name
	if _zone_label:
		_zone_label.text = zone_name


func _on_class_selected(idx: int) -> void:
	_selected_class = CLASSES[idx]
	save_config()


func save_config() -> void:
	var config := {"class": _selected_class, "zone": _detected_zone}
	var path: String = ProjectSettings.globalize_path(CONFIG_PATH)
	var f := FileAccess.open(path, FileAccess.WRITE)
	if f == null:
		push_warning("DebugLauncher: could not write %s" % path)
		return
	f.store_string(JSON.stringify(config, "\t") + "\n")


func _load_config() -> void:
	var path: String = ProjectSettings.globalize_path(CONFIG_PATH)
	if not FileAccess.file_exists(path):
		save_config()
		return
	var f := FileAccess.open(path, FileAccess.READ)
	if f == null:
		return
	var json := JSON.new()
	if json.parse(f.get_as_text()) != OK:
		return
	var data: Dictionary = json.data
	_selected_class = data.get("class", "gunner")
	_detected_zone = data.get("zone", "arena")
	# Update UI
	var idx: int = CLASSES.find(_selected_class)
	if idx >= 0 and _class_option:
		_class_option.selected = idx
	if _zone_label:
		_zone_label.text = _detected_zone
