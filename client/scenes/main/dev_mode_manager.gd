extends Node

## Dev mode: auto-start server, connect with retry, bot management.

var ctrl: Node

var debug_panel: CanvasLayer = null
var bot_panel: CanvasLayer = null
var server_pid: int = -1
var dev_class: String = "gunner"
var dev_zone: String = "arena"
var dev_connected: bool = false


func _ready() -> void:
	ctrl = get_parent()


## Load .dev_config.json written by the debug_launcher editor plugin.
## Only sets values not already overridden by CLI args.
func load_dev_config() -> void:
	var config_path: String = ProjectSettings.globalize_path("res://.dev_config.json")
	if not FileAccess.file_exists(config_path):
		return
	var f: FileAccess = FileAccess.open(config_path, FileAccess.READ)
	if f == null:
		return
	var json := JSON.new()
	if json.parse(f.get_as_text()) != OK:
		return
	var data: Dictionary = json.data
	# CLI args (already parsed) take priority -- only apply config if still default
	var has_class_arg := false
	var has_zone_arg := false
	for arg in OS.get_cmdline_user_args():
		if arg.begins_with("--class="):
			has_class_arg = true
		elif arg.begins_with("--zone="):
			has_zone_arg = true
	if not has_class_arg and data.has("class"):
		dev_class = data["class"]
	if not has_zone_arg and data.has("zone"):
		dev_zone = data["zone"]


## Start the Go gateway server and connect automatically.
func dev_auto_start() -> void:
	ctrl._menu_layer.visible = false
	ctrl._local_class = dev_class

	# Start server subprocess.
	# Uses bash to cd into server dir, build (cached), then exec the binary
	# so the PID points directly to the gateway process.
	var client_dir: String = ProjectSettings.globalize_path("res://").rstrip("/")
	var project_root: String = client_dir.get_base_dir()
	var server_dir: String = project_root + "/server"
	OS.set_environment("CODEX_DEV", "1")
	OS.set_environment("GOPATH", project_root + "/.go")
	print("[Main] Starting dev server from %s..." % server_dir)
	server_pid = (
		OS
		. create_process(
			"bash",
			[
				"-c",
				"cd '%s' && go build -o bin/gateway ./cmd/gateway && exec bin/gateway" % server_dir,
			]
		)
	)
	if server_pid <= 0:
		push_error("[Main] Failed to start dev server -- falling back to menu")
		ctrl._enter_menu()
		return
	print("[Main] Dev server started (PID %d)" % server_pid)

	# Connect with retries (server needs a moment to build + bind)
	NetworkManager.username = "Dev"
	NetworkManager.dev_params = {"class": dev_class, "zone": dev_zone}
	await dev_connect_with_retry()


## Retry connecting to the dev server until it's ready.
## Waits for zone_transfer_received signal which fires when dev auto-join completes.
func dev_connect_with_retry() -> void:
	dev_connected = false
	NetworkManager.zone_transfer_received.connect(_on_dev_zone_transfer, CONNECT_ONE_SHOT)

	var max_attempts: int = 40  # 40 * 0.5s = 20s max wait (build + start)
	for attempt in range(max_attempts):
		if dev_connected:
			print("[Main] Dev server connected on attempt %d" % (attempt + 1))
			return
		# Only initiate a new connection if the previous attempt failed.
		if not NetworkManager.is_active:
			NetworkManager.connect_to_server("127.0.0.1")
		await ctrl.get_tree().create_timer(0.5).timeout

	# Cleanup
	if NetworkManager.zone_transfer_received.is_connected(_on_dev_zone_transfer):
		NetworkManager.zone_transfer_received.disconnect(_on_dev_zone_transfer)
	push_error("[Main] Could not connect to dev server after %d attempts" % max_attempts)
	stop_dev_server()
	ctrl._enter_menu()


func _on_dev_zone_transfer(_zone_type: int, _peer_id: int) -> void:
	dev_connected = true


func stop_dev_server() -> void:
	if server_pid > 0:
		OS.kill(server_pid)
		print("[Main] Dev server stopped (PID %d)" % server_pid)
		server_pid = -1


## Initialize dev mode: parse CLI args, load config, create debug panels.
func initialize(user_args: PackedStringArray) -> void:
	# Parse optional --class=X and --zone=X overrides
	for arg in user_args:
		if arg.begins_with("--class="):
			dev_class = arg.split("=")[1]
		elif arg.begins_with("--zone="):
			dev_zone = arg.split("=")[1]
	# Load editor config (CLI args take priority, already parsed above)
	load_dev_config()
	var DebugPanelScript := preload("res://scenes/ui/debug_panel.gd")
	debug_panel = DebugPanelScript.new()
	ctrl.add_child(debug_panel)
	var BotPanelScript := preload("res://scenes/ui/bot_panel.gd")
	bot_panel = BotPanelScript.new()
	bot_panel.closed.connect(ctrl._update_cursor_mode)
	ctrl.add_child(bot_panel)
	print("[Main] Dev mode enabled -- class=%s zone=%s" % [dev_class, dev_zone])


func toggle_debug_panel() -> void:
	if debug_panel:
		debug_panel.toggle()


func toggle_bot_panel() -> void:
	if bot_panel != null:
		bot_panel.toggle()
		ctrl._update_cursor_mode()


func respawn_bots_after_transfer() -> void:
	if bot_panel == null:
		return
	var configs: Array = bot_panel.get_bot_configs()
	for cfg in configs:
		NetworkManager.debug.send_spawn_bot(cfg["class"], cfg["spec"])
