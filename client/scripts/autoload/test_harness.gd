extends Node

## E2E test instrumentation. Activated by command-line flags:
##   --bot       Attach a BotController to the player
##   --remote    Enable file-based remote control
##   --capture   Periodic screenshot + state capture
##   --e2e       Run scenario then quit
##
## Output goes to res://test_output/ (client/test_output/ on disk).

const GUNNER_BOT_SCRIPT := preload("res://scripts/bot/bot_controller.gd")
const VANGUARD_BOT_SCRIPT := preload("res://scripts/bot/vanguard_bot_controller.gd")
const BLADE_DANCER_BOT_SCRIPT := preload("res://scripts/bot/blade_dancer_bot_controller.gd")
const E2E_CONTEXT_SCRIPT := preload("res://scripts/e2e/e2e_context.gd")

const REMOTE_ACTIONS: Array[String] = [
	"move_forward",
	"move_backward",
	"move_left",
	"move_right",
	"jump",
	"sprint",
	"shoot",
	"dodge",
	"light_attack",
	"heavy_attack",
	"block",
	"lock_on",
	"ability_1",
	"ability_2",
	"reload",
	"load_enhanced",
	"mag_dump",
]

var bot_mode: bool = false
var remote_mode: bool = false
var capture_mode: bool = false
var e2e_mode: bool = false
var bot_class: String = "gunner"  # "gunner", "vanguard", or "blade_dancer"
var scenario_names: PackedStringArray = []

var output_dir: String = "res://test_output/"
var _tick: int = 0
var _capture_interval: int = 30  # frames between auto-captures
var _remote_poll_interval: int = 5  # frames between command reads
var _bot: Node = null
var _e2e_timer: float = 0.0
var _e2e_duration: float = 90.0  # seconds for e2e run (boss fight takes ~40-60s)
var _ctx: RefCounted = null
var _walk_target: Variant = null  # Vector3 while walking, null when idle
var _connect_status: String = "idle"  # idle | connecting | connected | failed


func _ready() -> void:
	var args := OS.get_cmdline_user_args()
	bot_mode = "--bot" in args
	remote_mode = "--remote" in args
	capture_mode = "--capture" in args or remote_mode
	e2e_mode = "--e2e" in args

	# Parse flags: --duration=60, --class=vanguard, --e2e-scenarios=a,b
	for arg in args:
		if arg.begins_with("--duration="):
			_e2e_duration = arg.split("=")[1].to_float()
		elif arg.begins_with("--class="):
			bot_class = arg.split("=")[1]
		elif arg.begins_with("--e2e-scenarios="):
			scenario_names = PackedStringArray(arg.split("=")[1].split(","))

	DirAccess.make_dir_recursive_absolute(ProjectSettings.globalize_path(output_dir))

	if bot_mode or remote_mode or scenario_names.size() > 0:
		# Wait one frame for scene tree to be ready, then attach
		get_tree().process_frame.connect(_on_first_frame, CONNECT_ONE_SHOT)

	print(
		(
			"[TestHarness] bot=%s remote=%s capture=%s e2e=%s"
			% [bot_mode, remote_mode, capture_mode, e2e_mode]
		)
	)


func _on_first_frame() -> void:
	if scenario_names.size() > 0:
		_start_scenario_runner()
		return
	if bot_mode:
		_attach_bot()
	if remote_mode:
		_write_state()  # initial state


func _start_scenario_runner() -> void:
	var RunnerScript: GDScript = load("res://scripts/e2e/e2e_runner.gd")
	var runner: Node = RunnerScript.new()
	runner.scenario_names = scenario_names
	add_child(runner)
	print("[TestHarness] E2E scenario runner started: %s" % ", ".join(scenario_names))


func _process(_delta: float) -> void:
	_tick += 1

	if _walk_target != null:
		var ctx := _get_ctx()
		if ctx == null or ctx.steer_step(_walk_target):
			_walk_target = null

	if capture_mode and _tick % _capture_interval == 0:
		_write_state()

	if remote_mode and _tick % _remote_poll_interval == 0:
		_read_commands()

	if e2e_mode:
		_e2e_timer += _delta
		if _e2e_timer >= _e2e_duration:
			_finish_e2e()


# --- Bot ---


func _attach_bot() -> void:
	await get_tree().process_frame
	# Select the right class via main scene
	var main := get_tree().current_scene
	if main and "player" in main:
		if bot_class == "vanguard" and "vanguard" in main and main.has_method("_select_player"):
			main._select_player(main.vanguard)
			await get_tree().process_frame
		elif (
			bot_class == "blade_dancer"
			and "blade_dancer" in main
			and main.has_method("_select_player")
		):
			main._select_player(main.blade_dancer)
			await get_tree().process_frame

	if GameManager.players.is_empty():
		push_warning("[TestHarness] No players found, can't attach bot")
		return

	# Find the active player
	var player: CharacterBody3D = null
	if main and "player" in main:
		player = main.player
	else:
		player = GameManager.players[0]

	_bot = Node.new()
	if "config" in player:
		_bot.set_script(BLADE_DANCER_BOT_SCRIPT)
		print("[TestHarness] Blade Dancer bot attached to %s" % player.name)
	elif "stamina" in player:
		_bot.set_script(VANGUARD_BOT_SCRIPT)
		print("[TestHarness] Vanguard bot attached to %s" % player.name)
	else:
		_bot.set_script(GUNNER_BOT_SCRIPT)
		print("[TestHarness] Gunner bot attached to %s" % player.name)
	player.add_child(_bot)


# --- Screenshot ---


func capture_screenshot(name: String = "") -> String:
	var image := get_viewport().get_texture().get_image()
	if not image:
		return ""
	var filename := name if name != "" else "screenshot_%d.png" % _tick
	var path := output_dir + filename
	var global_path := ProjectSettings.globalize_path(path)
	image.save_png(global_path)
	return global_path


# --- State dump ---


func get_state() -> Dictionary:
	var state := {
		"tick": _tick,
		"time": _e2e_timer,
		"players": [],
		"enemies": [],
	}

	var ctx := _get_ctx()
	if ctx:
		var main: Node3D = ctx.main
		var app := {
			"state": main.GameState.keys()[main.state],
			"connected": NetworkManager.is_active,
			"connect_status": _connect_status,
			"my_id": NetworkManager.get_my_id(),
			"walk_target": _vec3_to_array(_walk_target) if _walk_target != null else null,
			"near_portal": main.hub_interact.near_portal if main.hub_interact else false,
			"near_exit_portal":
			main.env_builder.is_near_exit_portal() if main.env_builder else false,
		}
		var local: CharacterBody3D = ctx.local_player()
		if local:
			app["local_player"] = {
				"position": _vec3_to_array(local.global_position),
				"health": local.health if "health" in local else -1,
			}
		state["app"] = app

	for player in GameManager.players:
		if not is_instance_valid(player):
			continue
		var p := {
			"name": player.name,
			"position": _vec3_to_array(player.global_position),
			"velocity": _vec3_to_array(player.velocity),
			"health": player.health if "health" in player else -1,
			"max_health": player.max_health if "max_health" in player else -1,
			"is_rolling": player._is_rolling if "_is_rolling" in player else false,
			"roll_cooldown":
			player._roll_cooldown_timer if "_roll_cooldown_timer" in player else 0.0,
		}
		state["players"].append(p)

	for enemy in GameManager.enemies:
		if not is_instance_valid(enemy):
			continue
		var e := {
			"name": enemy.name,
			"position": _vec3_to_array(enemy.global_position),
			"health": enemy.health if "health" in enemy else -1,
			"max_health": enemy.max_health if "max_health" in enemy else -1,
			"state": enemy.State.keys()[enemy.state] if "state" in enemy else "UNKNOWN",
			"phase": enemy._current_phase if "_current_phase" in enemy else 1,
		}
		state["enemies"].append(e)

	return state


func _write_state() -> void:
	var state := get_state()
	# Also capture screenshot
	var screenshot_path := capture_screenshot("latest.png")
	state["screenshot"] = screenshot_path
	var json := JSON.stringify(state, "  ")
	var path := ProjectSettings.globalize_path(output_dir + "state.json")
	var file := FileAccess.open(path, FileAccess.WRITE)
	if file:
		file.store_string(json)


# --- Remote control ---


func _read_commands() -> void:
	var path := ProjectSettings.globalize_path(output_dir + "commands.json")
	if not FileAccess.file_exists(path):
		return
	var file := FileAccess.open(path, FileAccess.READ)
	if not file:
		return
	var text := file.get_as_text()
	if text.is_empty():
		return

	var json := JSON.new()
	if json.parse(text) != OK:
		return
	var cmd: Dictionary = json.data
	_apply_commands(cmd)

	# Delete file after reading so we don't re-process
	DirAccess.remove_absolute(path)


func _apply_commands(cmd: Dictionary) -> void:
	var ctx := _get_ctx()

	if cmd.has("connect") and cmd["connect"] is Dictionary and ctx:
		_remote_connect(ctx, cmd["connect"])

	if cmd.get("stop", false):
		_walk_target = null
		if ctx:
			ctx.release_all()

	if cmd.has("walk_to"):
		_walk_target = _resolve_walk_target(cmd["walk_to"])

	if cmd.get("enter_portal", false):
		NetworkManager.send_enter_portal()

	if cmd.has("attach_bot") and ctx:
		ctx.attach_bot(str(cmd["attach_bot"]))
	if cmd.get("detach_bot", false) and ctx:
		ctx.detach_bot()

	# Held actions: {"shoot": true} presses, {"shoot": false} releases
	var actions := cmd.get("actions", {}) as Dictionary
	for action_name in REMOTE_ACTIONS:
		if actions.has(action_name):
			if actions[action_name]:
				Input.action_press(action_name)
			else:
				Input.action_release(action_name)

	# One-shot press+release
	if cmd.has("tap") and cmd["tap"] is Array:
		for action_name in cmd["tap"]:
			if str(action_name) in REMOTE_ACTIONS:
				_tap_action(str(action_name))

	# Raw key taps for physical_keycode handlers (R, E, G, 1/2/3, ...)
	if cmd.has("press_key") and cmd["press_key"] is Array:
		for key_name in cmd["press_key"]:
			_tap_physical_key(str(key_name))

	# Aim at world position
	if cmd.has("look_at") and cmd["look_at"] is Array and cmd["look_at"].size() == 3:
		var player := _remote_player(ctx)
		if player:
			var target := Vector3(cmd["look_at"][0], cmd["look_at"][1], cmd["look_at"][2])
			_aim_player_at(player, target)

	# Request screenshot + state dump
	if cmd.get("screenshot", false):
		_write_state()


func _remote_connect(ctx: RefCounted, params: Dictionary) -> void:
	_connect_status = "connecting"
	var ok: bool = await ctx.dev_connect(
		str(params.get("class", "gunner")), str(params.get("zone", "hub"))
	)
	_connect_status = "connected" if ok else "failed"


func _tap_action(action: String) -> void:
	Input.action_press(action)
	await get_tree().create_timer(0.12).timeout
	Input.action_release(action)


func _tap_physical_key(key_name: String) -> void:
	var keycode := OS.find_keycode_from_string(key_name)
	if keycode == KEY_NONE:
		return
	var ev := InputEventKey.new()
	ev.keycode = keycode
	ev.physical_keycode = keycode
	ev.pressed = true
	Input.parse_input_event(ev)
	await get_tree().create_timer(0.1).timeout
	var ev_up := InputEventKey.new()
	ev_up.keycode = keycode
	ev_up.physical_keycode = keycode
	ev_up.pressed = false
	Input.parse_input_event(ev_up)


func _resolve_walk_target(value: Variant) -> Variant:
	if value is Array and value.size() == 3:
		return Vector3(value[0], value[1], value[2])
	if value is String:
		var main := get_tree().current_scene
		if value == "exit_portal":
			return Vector3(0.0, 0.1, 0.0)
		if value == "portal" and main and "env_builder" in main:
			var env: Node3D = main.env_builder.current_env
			var portal: Node3D = env.get_node_or_null("PortalArea") if env else null
			if portal:
				return portal.global_position
	return null


func _remote_player(ctx: RefCounted) -> CharacterBody3D:
	if ctx:
		var player: CharacterBody3D = ctx.local_player()
		if player:
			return player
	if GameManager.players.is_empty():
		return null
	return GameManager.players[0]


func _get_ctx() -> RefCounted:
	if _ctx != null:
		return _ctx
	var main := get_tree().current_scene
	if main == null or not ("entity_mgr" in main):
		return null
	_ctx = E2E_CONTEXT_SCRIPT.new(main)
	return _ctx


func _aim_player_at(player: CharacterBody3D, target: Vector3) -> void:
	var to_target := target - player.global_position
	# Yaw (horizontal)
	player.rotation.y = atan2(-to_target.x, -to_target.z)
	# Pitch (vertical) on head node
	var head: Node3D = player.get_node_or_null("Head")
	if head:
		var flat_dist := Vector2(to_target.x, to_target.z).length()
		head.rotation.x = clampf(
			atan2(to_target.y - 1.6, flat_dist), deg_to_rad(-89.0), deg_to_rad(89.0)
		)


# --- E2E finish ---


func _finish_e2e() -> void:
	_write_state()
	capture_screenshot("e2e_final.png")
	var state := get_state()

	# Write summary
	var summary := {
		"duration": _e2e_timer,
		"player_health": state["players"][0]["health"] if state["players"].size() > 0 else 0,
		"enemies_alive": state["enemies"].size(),
		"result":
		"PASS" if state["players"].size() > 0 and state["players"][0]["health"] > 0 else "FAIL",
	}
	var json := JSON.stringify(summary, "  ")
	var path := ProjectSettings.globalize_path(output_dir + "e2e_result.json")
	var file := FileAccess.open(path, FileAccess.WRITE)
	if file:
		file.store_string(json)
	print("[TestHarness] E2E complete: %s" % summary["result"])
	get_tree().quit(0 if summary["result"] == "PASS" else 1)


# --- Util ---


func _vec3_to_array(v: Vector3) -> Array:
	return [snappedf(v.x, 0.01), snappedf(v.y, 0.01), snappedf(v.z, 0.01)]
