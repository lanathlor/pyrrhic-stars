extends Node

## Persistent client settings (graphics, audio, keybinds).
##
## The server database is the source of truth so settings follow the account
## across devices, but a local JSON mirror (user://settings.json) lets everything
## apply instantly at boot and keeps working offline. On login we fetch the
## server document (server wins); a brand-new account is seeded with the local
## one. All values are stored in a single JSON document that is byte-identical
## locally and server-side, so there is a single serialization path.
##
## Key bindings are stored as physical keycodes. Because the InputMap binds by
## physical key, movement already lands on the correct physical keys for AZERTY
## and every other layout; get_action_label() translates those physical keys to
## the labels actually printed on the player's keyboard via the active OS layout.

signal settings_changed

const SAVE_PATH := "user://settings.json"
const SETTINGS_VERSION := 1
const SERVER_PUSH_DEBOUNCE := 0.6

const RESOLUTIONS: Array[String] = [
	"1280x720",
	"1600x900",
	"1920x1080",
	"2560x1440",
	"3840x2160",
]
const DISPLAY_MODES: Array[String] = ["Windowed", "Fullscreen", "Borderless"]
const QUALITY_LEVELS: Array[String] = ["Low", "Medium", "High", "Ultra"]

## Rebindable binds grouped for the controls UI: "Core" holds the universal
## movement/UI binds; the rest mirror each class's in-game ability bar slot-for-slot
## (including the mouse-bound and dodge slots), numbered to match the HUD. The
## InputMap is global, so an action shared by several classes (e.g. heavy_attack,
## dodge) appears under each class that uses it - rebinding it changes the key for
## every class listed. Slot order/count matches each controller's
## ABILITY_SLOT_ACTIONS / HUD ability bar. "Core" must stay first.
const KEYBIND_GROUPS: Array = [
	{
		"name": "Core",
		"binds":
		[
			{"action": "move_forward", "label": "Move Forward"},
			{"action": "move_backward", "label": "Move Backward"},
			{"action": "move_left", "label": "Move Left"},
			{"action": "move_right", "label": "Move Right"},
			{"action": "jump", "label": "Jump"},
			{"action": "sprint", "label": "Sprint"},
			{"action": "dodge", "label": "Dodge / Roll"},
			{"action": "lock_on", "label": "Lock On"},
			{"action": "toggle_codex", "label": "Toggle Codex"},
		],
	},
	{
		"name": "Gunner",
		"binds":
		[
			{"action": "shoot", "label": "Ability 1"},
			{"action": "dodge", "label": "Ability 2"},
			{"action": "ability_1", "label": "Ability 3"},
			{"action": "ability_2", "label": "Ability 4"},
			{"action": "mag_dump", "label": "Ability 5"},
			{"action": "load_enhanced", "label": "Ability 6"},
			{"action": "reload", "label": "Reload"},
		],
	},
	{
		"name": "Blade Dancer",
		"binds":
		[
			{"action": "light_attack", "label": "Ability 1"},
			{"action": "block", "label": "Ability 2"},
			{"action": "heavy_attack", "label": "Ability 3"},
			{"action": "ability_2", "label": "Ability 4"},
		],
	},
	{
		"name": "Arcanotechnicien",
		"binds":
		[
			{"action": "harmonist_slot_0", "label": "Ability 1"},
			{"action": "harmonist_slot_1", "label": "Ability 2"},
			{"action": "heavy_attack", "label": "Ability 3"},
			{"action": "ability_2", "label": "Ability 4"},
			{"action": "ability_1", "label": "Ability 5"},
			{"action": "dodge", "label": "Ability 6"},
		],
	},
	{
		"name": "Vanguard",
		"binds":
		[
			{"action": "light_attack", "label": "Ability 1"},
			{"action": "block", "label": "Ability 2"},
			{"action": "heavy_attack", "label": "Ability 3"},
			{"action": "dodge", "label": "Ability 4"},
			{"action": "ability_1", "label": "Ability 5"},
			{"action": "ability_2", "label": "Ability 6"},
		],
	},
]

# Labels for negative (mouse-button) binds, keyed by button_index.
const MOUSE_BUTTON_LABELS: Dictionary = {
	MOUSE_BUTTON_LEFT: "Mouse Left",
	MOUSE_BUTTON_RIGHT: "Mouse Right",
	MOUSE_BUTTON_MIDDLE: "Mouse Middle",
	MOUSE_BUTTON_WHEEL_UP: "Wheel Up",
	MOUSE_BUTTON_WHEEL_DOWN: "Wheel Down",
}

var _settings: Dictionary = {}
var _default_keybinds: Dictionary = {}
var _server_host: String = ""
var _dirty: bool = false

var _get_http: HTTPRequest
var _put_http: HTTPRequest
var _push_timer: Timer


func _ready() -> void:
	_snapshot_default_keybinds()
	_settings = _default_settings()

	_get_http = HTTPRequest.new()
	add_child(_get_http)
	_get_http.request_completed.connect(_on_get_completed)
	_put_http = HTTPRequest.new()
	add_child(_put_http)

	_push_timer = Timer.new()
	_push_timer.one_shot = true
	_push_timer.wait_time = SERVER_PUSH_DEBOUNCE
	add_child(_push_timer)
	_push_timer.timeout.connect(_push_to_server)

	_load_local()
	apply_all()


# =============================================================================
# Public API
# =============================================================================


## The grouped keybind layout for the controls UI (Core + per-class tabs).
func keybind_groups() -> Array:
	return KEYBIND_GROUPS


## Actions whose keys must stay distinct when editing the given group, to prevent
## binding one key to two actions the same class reads. Editing a class group:
## Core + that group (a class only ever uses Core + its own bar). Editing Core:
## every group, since a Core key is shared by all classes.
func conflict_scope_actions(group_name: String) -> Array:
	var actions: Array = []
	for group in KEYBIND_GROUPS:
		if group_name == "Core" or group.name == "Core" or group.name == group_name:
			for bind in group.binds:
				if not actions.has(bind.action):
					actions.append(bind.action)
	return actions


func get_value(section: String, key: String, fallback: Variant = null) -> Variant:
	var s: Dictionary = _settings.get(section, {})
	return s.get(key, fallback)


## Updates one setting, applies the relevant subsystem, and persists.
func set_value(section: String, key: String, value: Variant) -> void:
	if not _settings.has(section):
		_settings[section] = {}
	_settings[section][key] = value
	match section:
		"graphics":
			_apply_graphics()
		"audio":
			_apply_audio()
	_persist()


## Returns the physical keycode currently bound to an action.
func get_keybind(action: String) -> int:
	var kb: Dictionary = _settings.get("keybinds", {})
	return int(kb.get(action, 0))


## Rebinds an action to a physical keycode and persists.
func set_keybind(action: String, physical_keycode: int) -> void:
	if not _settings.has("keybinds"):
		_settings["keybinds"] = {}
	_settings["keybinds"][action] = physical_keycode
	_set_action_key(action, physical_keycode)
	_persist()


## Restores all keybinds to their project defaults.
func reset_keybinds() -> void:
	_settings["keybinds"] = _default_keybinds.duplicate()
	_apply_keybinds()
	_persist()


## The label printed on the player's physical key for an action, under the active
## OS keyboard layout. This is the whole of the "AZERTY auto-detection".
func get_action_label(action: String) -> String:
	return keycode_to_label(get_keybind(action))


## Binds are stored as ints: 0 = unbound, >0 = physical keycode, <0 = mouse button
## (negated button_index). This keeps the JSON document a flat int map while
## supporting the mouse-bound ability slots (primary/secondary/block).
func keycode_to_label(code: int) -> String:
	if code == 0:
		return "Unbound"
	if code < 0:
		return MOUSE_BUTTON_LABELS.get(-code, "Mouse %d" % (-code))
	var kc := DisplayServer.keyboard_get_keycode_from_physical(code as Key)
	if kc == 0:
		kc = code as Key
	return OS.get_keycode_string(kc)


func apply_all() -> void:
	_apply_graphics()
	_apply_audio()
	_apply_keybinds()


## Fetches the server document after login. Server wins; an empty server document
## is seeded with the current local settings. Safe to call in dev mode (uses the
## ?uuid= bypass when no Kratos token is present).
func sync_from_server(host: String) -> void:
	_server_host = host
	var req := _build_request("GET")
	if req.is_empty():
		return
	_get_http.cancel_request()
	var err := _get_http.request(req.url, req.headers, HTTPClient.METHOD_GET)
	if err != OK:
		push_warning("[Settings] sync GET error: %s" % error_string(err))


# =============================================================================
# Defaults / serialization
# =============================================================================


func _default_settings() -> Dictionary:
	return {
		"version": SETTINGS_VERSION,
		"graphics":
		{
			"resolution": "1920x1080",
			"display_mode": 0,
			"quality": 2,
			"vsync": true,
		},
		"audio":
		{
			"master": 1.0,
			"music": 0.8,
			"sfx": 1.0,
		},
		"keybinds": _default_keybinds.duplicate(),
	}


## Records the project-default physical keycode for every keyboard-bound action,
## before any overrides are applied. Mouse-only actions (shoot, block, ...) have
## no InputEventKey and are intentionally skipped.
func _snapshot_default_keybinds() -> void:
	_default_keybinds.clear()
	for action in InputMap.get_actions():
		var action_name := String(action)
		if action_name.begins_with("ui_"):
			continue
		for ev in InputMap.action_get_events(action):
			if ev is InputEventKey:
				_default_keybinds[action_name] = (ev as InputEventKey).physical_keycode
				break
			elif ev is InputEventMouseButton:
				_default_keybinds[action_name] = -(ev as InputEventMouseButton).button_index
				break


## Deep-merges a stored document onto the defaults so new settings keys added in
## later versions still get sane values for existing users.
func _merge_document(doc: Dictionary) -> void:
	for section in ["graphics", "audio"]:
		var stored: Dictionary = doc.get(section, {})
		for key in stored.keys():
			if _settings[section].has(key):
				_settings[section][key] = stored[key]
	var kb: Dictionary = doc.get("keybinds", {})
	for action in kb.keys():
		_settings["keybinds"][action] = int(kb[action])


# =============================================================================
# Apply
# =============================================================================


func _apply_graphics() -> void:
	var g: Dictionary = _settings.get("graphics", {})
	_apply_display(g)
	var vsync: bool = bool(g.get("vsync", true))
	var vsync_mode := DisplayServer.VSYNC_ENABLED if vsync else DisplayServer.VSYNC_DISABLED
	DisplayServer.window_set_vsync_mode(vsync_mode as DisplayServer.VSyncMode)
	_apply_quality(int(g.get("quality", 2)))


func _apply_display(g: Dictionary) -> void:
	var mode: int = int(g.get("display_mode", 0))
	match mode:
		1:
			DisplayServer.window_set_mode(DisplayServer.WINDOW_MODE_EXCLUSIVE_FULLSCREEN)
		2:
			DisplayServer.window_set_mode(DisplayServer.WINDOW_MODE_FULLSCREEN)
		_:
			DisplayServer.window_set_mode(DisplayServer.WINDOW_MODE_WINDOWED)
			DisplayServer.window_set_flag(DisplayServer.WINDOW_FLAG_BORDERLESS, false)
			var dims: PackedStringArray = String(g.get("resolution", "1920x1080")).split("x")
			if dims.size() == 2:
				DisplayServer.window_set_size(Vector2i(int(dims[0]), int(dims[1])))
				_center_window()


func _center_window() -> void:
	var screen := DisplayServer.window_get_current_screen()
	var size := DisplayServer.window_get_size()
	var origin := DisplayServer.screen_get_position(screen)
	var area := DisplayServer.screen_get_size(screen)
	DisplayServer.window_set_position(origin + (area - size) / 2)


func _apply_quality(q: int) -> void:
	q = clampi(q, 0, 3)
	var root := get_tree().root
	var msaa := [
		Viewport.MSAA_DISABLED,
		Viewport.MSAA_2X,
		Viewport.MSAA_4X,
		Viewport.MSAA_8X,
	]
	var render_scale := [0.75, 0.85, 1.0, 1.0]
	root.msaa_3d = msaa[q] as Viewport.MSAA
	root.scaling_3d_scale = render_scale[q]


func _apply_audio() -> void:
	var a: Dictionary = _settings.get("audio", {})
	_set_bus_volume("Master", float(a.get("master", 1.0)))
	_set_bus_volume("Music", float(a.get("music", 0.8)))
	_set_bus_volume("SFX", float(a.get("sfx", 1.0)))


func _set_bus_volume(bus_name: String, v: float) -> void:
	var idx := AudioServer.get_bus_index(bus_name)
	if idx < 0:
		return
	AudioServer.set_bus_mute(idx, v <= 0.001)
	AudioServer.set_bus_volume_db(idx, linear_to_db(clampf(v, 0.0001, 1.0)))


func _apply_keybinds() -> void:
	var kb: Dictionary = _settings.get("keybinds", {})
	for action in kb.keys():
		if InputMap.has_action(action):
			_set_action_key(action, int(kb[action]))


## Rebinds an action to a stored code (0 = unbound, >0 = physical keycode,
## <0 = mouse button). Replaces the existing key/mouse-button event, leaving any
## other event kinds (joypad, etc.) untouched.
func _set_action_key(action: String, code: int) -> void:
	for ev in InputMap.action_get_events(action):
		if ev is InputEventKey or ev is InputEventMouseButton:
			InputMap.action_erase_event(action, ev)
	if code > 0:
		var ev := InputEventKey.new()
		ev.physical_keycode = code as Key
		InputMap.action_add_event(action, ev)
	elif code < 0:
		var mb := InputEventMouseButton.new()
		mb.button_index = (-code) as MouseButton
		InputMap.action_add_event(action, mb)


# =============================================================================
# Persistence (local immediate, server debounced)
# =============================================================================


func _persist() -> void:
	_save_local()
	settings_changed.emit()
	_dirty = true
	if _server_host != "":
		_push_timer.start()


func _save_local() -> void:
	var f := FileAccess.open(SAVE_PATH, FileAccess.WRITE)
	if f == null:
		push_warning("[Settings] could not write %s" % SAVE_PATH)
		return
	f.store_string(JSON.stringify(_settings))
	f.close()


func _load_local() -> void:
	if not FileAccess.file_exists(SAVE_PATH):
		return
	var f := FileAccess.open(SAVE_PATH, FileAccess.READ)
	if f == null:
		return
	var text := f.get_as_text()
	f.close()
	var parsed: Variant = JSON.parse_string(text)
	if parsed is Dictionary:
		_merge_document(parsed)


func _push_to_server() -> void:
	if not _dirty or _server_host == "":
		return
	var req := _build_request("PUT")
	if req.is_empty():
		return
	var headers: Array = req.headers.duplicate()
	headers.append("Content-Type: application/json")
	var err := _put_http.request(req.url, headers, HTTPClient.METHOD_PUT, JSON.stringify(_settings))
	if err != OK:
		push_warning("[Settings] push PUT error: %s" % error_string(err))
		return
	_dirty = false


func _on_get_completed(
	_result: int, code: int, _headers: PackedStringArray, body: PackedByteArray
) -> void:
	if code != 200:
		# No server document yet (or transient error): seed the server with local.
		_dirty = true
		_push_to_server()
		return
	var parsed: Variant = JSON.parse_string(body.get_string_from_utf8())
	if parsed is Dictionary and not (parsed as Dictionary).is_empty():
		_settings = _default_settings()
		_merge_document(parsed)
		apply_all()
		_save_local()
		settings_changed.emit()
	else:
		# Empty server document: push local up as the first-time sync.
		_dirty = true
		_push_to_server()


# =============================================================================
# REST helpers
# =============================================================================


## Builds {url, headers} for a settings request. Uses the Kratos token via the
## Authorization header when available, else the dev ?uuid= bypass. Returns an
## empty dict when neither identity is available.
func _build_request(_method: String) -> Dictionary:
	var url := "%s/api/v1/settings" % ServerConfig.gateway_http_base(_server_host)
	var headers: Array = ["Accept: application/json"]
	if AuthManager.has_token():
		headers.append("Authorization: Bearer %s" % AuthManager.get_token())
		return {"url": url, "headers": headers}
	var uuid := IdentityManager.get_player_id()
	if uuid != "":
		url += "?uuid=%s" % uuid.uri_encode()
		return {"url": url, "headers": headers}
	return {}
