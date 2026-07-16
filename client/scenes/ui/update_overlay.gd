extends CanvasLayer

## Blocking "update required" screen, shown before login when this build's
## stamped release tag differs from the server's. One click downloads the
## exact release asset the server reports, swaps the executable in place and
## relaunches. Built entirely in code (no .tscn).

const UI_SURFACE := Color(0.035, 0.045, 0.065, 0.92)
const UI_BORDER := Color(0.32, 0.58, 0.92, 0.95)
const UI_TEXT := Color(0.9, 0.93, 0.98, 0.96)
const UI_TEXT_MUTED := Color(0.6, 0.66, 0.75, 0.95)
const UI_DANGER := Color(0.86, 0.28, 0.28, 0.96)

var _server_version: String = ""
var _title: Label
var _detail: Label
var _status: Label
var _progress: ProgressBar
var _update_btn: Button
var _http: HTTPRequest
var _downloading: bool = false


func _ready() -> void:
	process_mode = Node.PROCESS_MODE_ALWAYS
	layer = 20
	visible = false
	_http = HTTPRequest.new()
	_http.request_completed.connect(_on_download_completed)
	add_child(_http)
	_build_ui()


func open(client_version: String, server_version: String) -> void:
	_server_version = server_version
	_detail.text = "Installed: %s   ·   Server: %s" % [client_version, server_version]
	_status.text = ""
	_progress.visible = false
	_update_btn.disabled = false
	visible = true


func _process(_delta: float) -> void:
	if not _downloading:
		return
	var total := _http.get_body_size()
	if total > 0:
		_progress.max_value = total
		_progress.value = _http.get_downloaded_bytes()


func _build_ui() -> void:
	var bg := ColorRect.new()
	bg.color = Color(0.0, 0.0, 0.0, 0.85)
	bg.set_anchors_preset(Control.PRESET_FULL_RECT)
	bg.mouse_filter = Control.MOUSE_FILTER_STOP
	add_child(bg)

	var center := CenterContainer.new()
	center.set_anchors_preset(Control.PRESET_FULL_RECT)
	add_child(center)

	var panel := PanelContainer.new()
	var style := StyleBoxFlat.new()
	style.bg_color = UI_SURFACE
	style.border_color = UI_BORDER
	style.set_border_width_all(1)
	style.set_corner_radius_all(6)
	style.set_content_margin_all(32)
	panel.add_theme_stylebox_override("panel", style)
	center.add_child(panel)

	var box := VBoxContainer.new()
	box.add_theme_constant_override("separation", 16)
	box.custom_minimum_size = Vector2(520, 0)
	panel.add_child(box)
	_build_content(box)


func _build_content(box: VBoxContainer) -> void:
	_title = Label.new()
	_title.text = "Update required"
	_title.add_theme_font_size_override("font_size", 34)
	_title.add_theme_color_override("font_color", UI_TEXT)
	_title.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	box.add_child(_title)

	_detail = Label.new()
	_detail.add_theme_color_override("font_color", UI_TEXT_MUTED)
	_detail.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	box.add_child(_detail)

	_progress = ProgressBar.new()
	_progress.custom_minimum_size = Vector2(0, 24)
	_progress.show_percentage = true
	_progress.visible = false
	box.add_child(_progress)

	_status = Label.new()
	_status.add_theme_color_override("font_color", UI_DANGER)
	_status.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	_status.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	box.add_child(_status)

	_update_btn = Button.new()
	_update_btn.text = "Update & restart"
	_update_btn.custom_minimum_size = Vector2(0, 44)
	_update_btn.pressed.connect(_on_update_pressed)
	box.add_child(_update_btn)


func _on_update_pressed() -> void:
	var url: String = VersionCheck.download_url(_server_version)
	if url == "":
		_fail("No release download is available for this platform.")
		return
	_http.download_file = OS.get_executable_path() + ".new"
	var err := _http.request(url)
	if err != OK:
		_fail("Could not start the download: %s" % error_string(err))
		return
	_downloading = true
	_update_btn.disabled = true
	_progress.visible = true
	_progress.value = 0
	_status.text = ""


func _on_download_completed(
	result: int, code: int, _headers: PackedStringArray, _body: PackedByteArray
) -> void:
	_downloading = false
	if result != HTTPRequest.RESULT_SUCCESS or code != 200:
		DirAccess.remove_absolute(_http.download_file)
		_fail("Download failed (result %d, HTTP %d). Retry?" % [result, code])
		return
	_swap_and_relaunch()


## Puts the downloaded binary in place of the running one and restarts.
## The rename dance keeps a working install at every step: the new binary is
## only moved over the old one after the download fully succeeded, and on
## Windows (which cannot overwrite a running exe) the old one is first renamed
## aside; main.gd deletes the leftover ".old" on next launch.
func _swap_and_relaunch() -> void:
	var exe := OS.get_executable_path()
	var incoming := exe + ".new"
	if OS.get_name() == "Windows":
		DirAccess.remove_absolute(exe + ".old")
		if DirAccess.rename_absolute(exe, exe + ".old") != OK:
			_fail("Could not move the current executable aside. Retry?")
			return
	else:
		# 493 = 0o755 (gdtoolkit cannot parse octal literals).
		FileAccess.set_unix_permissions(incoming, 493)
	if DirAccess.rename_absolute(incoming, exe) != OK:
		if OS.get_name() == "Windows":
			DirAccess.rename_absolute(exe + ".old", exe)
		_fail("Could not install the new executable. Retry?")
		return
	OS.create_process(exe, [])
	get_tree().quit()


func _fail(message: String) -> void:
	_status.text = message
	_progress.visible = false
	_update_btn.disabled = false
