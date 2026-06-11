extends RefCounted

## Test context passed to E2EScenario.run(). Provides wait helpers,
## navigation, input simulation, and assertions against the live game.

const GUNNER_BOT := preload("res://scripts/bot/bot_controller.gd")
const VANGUARD_BOT := preload("res://scripts/bot/vanguard_bot_controller.gd")
const BD_BOT := preload("res://scripts/bot/blade_dancer_bot_controller.gd")

var main: Node3D
var tree: SceneTree
var entity_mgr: Node
var env_builder: Node
var game_flow: Node
var failures: Array[String] = []
var headed: bool

var _bot: Node = null


func _init(p_main: Node3D) -> void:
	main = p_main
	tree = main.get_tree()
	entity_mgr = main.entity_mgr
	env_builder = main.env_builder
	game_flow = main.game_flow
	headed = DisplayServer.get_name() != "headless"


# =============================================================================
# Wait helpers
# =============================================================================


func wait_for_state(target: int, timeout := 15.0) -> bool:
	var elapsed := 0.0
	while main.state != target:
		await tree.process_frame
		elapsed += tree.root.get_process_delta_time()
		if elapsed >= timeout:
			trace("TIMEOUT waiting for state %d (current=%d)" % [target, main.state])
			return false
	return true


func wait_for_signal(sig: Signal, timeout := 10.0) -> bool:
	var state := [false]
	var cb := func() -> void: state[0] = true
	sig.connect(cb, CONNECT_ONE_SHOT)
	var elapsed := 0.0
	while not state[0]:
		await tree.process_frame
		elapsed += tree.root.get_process_delta_time()
		if elapsed >= timeout:
			if sig.is_connected(cb):
				sig.disconnect(cb)
			trace("TIMEOUT waiting for signal")
			return false
	return true


func wait_frames(count := 2) -> void:
	for i in count:
		await tree.process_frame


func wait_seconds(seconds: float) -> void:
	await tree.create_timer(seconds).timeout


func wait_until(condition: Callable, timeout := 10.0) -> bool:
	var elapsed := 0.0
	while not condition.call():
		await tree.process_frame
		elapsed += tree.root.get_process_delta_time()
		if elapsed >= timeout:
			trace("TIMEOUT in wait_until")
			return false
	return true


# =============================================================================
# Connection
# =============================================================================


func dev_connect(p_class := "gunner", zone := "hub", use_menu := false) -> bool:
	main._local_class = p_class
	main._menu_layer.visible = false
	NetworkManager.username = "E2EBot"

	if use_menu:
		return await _menu_connect(p_class)

	NetworkManager.dev_params = {"class": p_class, "zone": zone}

	# Array wrapper: lambdas capture objects by ref, not bools by value
	var state := [false]
	var cb := func(_zt: int, _pid: int) -> void: state[0] = true
	NetworkManager.zone_transfer_received.connect(cb, CONNECT_ONE_SHOT)

	var max_attempts := 40
	for attempt in max_attempts:
		if state[0]:
			trace("connected on attempt %d" % (attempt + 1))
			return true
		if not NetworkManager.is_active:
			NetworkManager.connect_to_server("127.0.0.1")
		await tree.create_timer(0.5).timeout

	if NetworkManager.zone_transfer_received.is_connected(cb):
		NetworkManager.zone_transfer_received.disconnect(cb)
	fail("could not connect after %d attempts" % max_attempts)
	return false


func _menu_connect(p_class: String) -> bool:
	NetworkManager.dev_params = {}

	# Connect with retry (server may need a moment)
	var got_list := [false]
	var list_cb := func(_d: Dictionary) -> void: got_list[0] = true
	NetworkManager.character_list_received.connect(list_cb, CONNECT_ONE_SHOT)

	for attempt in 20:
		if got_list[0]:
			break
		if not NetworkManager.is_active:
			NetworkManager.connect_to_server("127.0.0.1")
		await tree.create_timer(0.5).timeout

	if NetworkManager.character_list_received.is_connected(list_cb):
		NetworkManager.character_list_received.disconnect(list_cb)
	if not got_list[0]:
		fail("no character list received")
		return false

	var chars: Array = main._char_list_data.get("characters", [])
	var char_id := 0
	for ch in chars:
		if ch.get("class_name", "") == p_class:
			char_id = ch.get("char_id", 0)
			break
	if char_id == 0 and chars.size() > 0:
		char_id = chars[0].get("char_id", 0)

	if char_id == 0:
		NetworkManager.send_create_character(p_class, "E2EBot")
	else:
		main.state = main.GameState.CHARACTER_SELECT
		NetworkManager.send_select_character(char_id)

	if not await wait_for_state(main.GameState.HUB, 20.0):
		fail("never reached HUB after character select")
		return false

	trace("menu_connect: in hub as %s" % p_class)
	return true


# =============================================================================
# Navigation
# =============================================================================


func walk_to(target: Vector3, threshold := 2.0, timeout := 20.0) -> bool:
	var player := _get_local_player()
	if not player:
		fail("walk_to: no local player")
		return false

	var elapsed := 0.0
	while player.global_position.distance_to(target) > threshold:
		_steer_toward(player, target)
		await tree.process_frame
		elapsed += tree.root.get_process_delta_time()
		if elapsed >= timeout:
			_release_movement()
			trace(
				(
					"TIMEOUT walk_to %s (dist=%.1f)"
					% [target, player.global_position.distance_to(target)]
				)
			)
			return false
	_release_movement()
	return true


func walk_to_portal(timeout := 20.0) -> bool:
	var current_env: Node3D = env_builder.current_env
	if not current_env:
		fail("walk_to_portal: no environment loaded")
		return false
	var portal_area: Node3D = current_env.get_node_or_null("PortalArea")
	if not portal_area:
		fail("walk_to_portal: no PortalArea node in environment")
		return false

	var target: Vector3 = portal_area.global_position
	var player := _get_local_player()
	if not player:
		fail("walk_to_portal: no local player")
		return false

	var elapsed := 0.0
	while not main.hub_interact.near_portal:
		_steer_toward(player, target)
		await tree.process_frame
		elapsed += tree.root.get_process_delta_time()
		if elapsed >= timeout:
			_release_movement()
			trace("TIMEOUT walk_to_portal (dist=%.1f)" % player.global_position.distance_to(target))
			return false
	_release_movement()
	return true


func walk_to_exit_portal(timeout := 20.0) -> bool:
	var target := Vector3(0.0, 0.1, 0.0)
	var player := _get_local_player()
	if not player:
		fail("walk_to_exit_portal: no local player")
		return false

	var elapsed := 0.0
	while not env_builder.is_near_exit_portal():
		_steer_toward(player, target)
		await tree.process_frame
		elapsed += tree.root.get_process_delta_time()
		if elapsed >= timeout:
			_release_movement()
			trace(
				(
					"TIMEOUT walk_to_exit_portal (dist=%.1f)"
					% player.global_position.distance_to(target)
				)
			)
			return false
	_release_movement()
	return true


func _steer_toward(player: CharacterBody3D, target: Vector3) -> void:
	var to_target := target - player.global_position
	to_target.y = 0.0
	if to_target.length() < 0.5:
		_release_movement()
		return

	# Face target
	player.rotation.y = atan2(-to_target.x, -to_target.z)

	# Convert world direction to player-local and press inputs
	var dir := to_target.normalized()
	var local := player.transform.basis.inverse() * dir
	if local.z < -0.3:
		Input.action_press("move_forward")
		Input.action_release("move_backward")
	elif local.z > 0.3:
		Input.action_press("move_backward")
		Input.action_release("move_forward")
	else:
		Input.action_release("move_forward")
		Input.action_release("move_backward")

	if local.x > 0.3:
		Input.action_press("move_right")
		Input.action_release("move_left")
	elif local.x < -0.3:
		Input.action_press("move_left")
		Input.action_release("move_right")
	else:
		Input.action_release("move_left")
		Input.action_release("move_right")


func _release_movement() -> void:
	for action in ["move_forward", "move_backward", "move_left", "move_right", "sprint"]:
		Input.action_release(action)


func _get_local_player() -> CharacterBody3D:
	var my_id: int = NetworkManager.get_my_id()
	if my_id <= 0:
		return null
	var player: CharacterBody3D = entity_mgr.spawned_players.get(my_id)
	if player and is_instance_valid(player):
		return player
	return null


# =============================================================================
# Actions
# =============================================================================


func enter_portal(overflux_conditions: Array = []) -> void:
	if overflux_conditions.is_empty():
		NetworkManager.send_enter_portal()
	else:
		NetworkManager.send_enter_portal_with_conditions(overflux_conditions)


func press_action(action: String, duration := 0.1) -> void:
	Input.action_press(action)
	await tree.create_timer(duration).timeout
	Input.action_release(action)


func hold_action(action: String) -> void:
	Input.action_press(action)


func release_action(action: String) -> void:
	Input.action_release(action)


func release_all() -> void:
	for action in [
		"move_forward",
		"move_backward",
		"move_left",
		"move_right",
		"sprint",
		"shoot",
		"dodge",
		"jump",
		"light_attack",
		"heavy_attack",
		"block",
	]:
		Input.action_release(action)


# =============================================================================
# Bot
# =============================================================================


func attach_bot(p_class := "gunner") -> void:
	detach_bot()
	var player := _get_local_player()
	if not player:
		trace("attach_bot: no local player, skipping")
		return

	_bot = Node.new()
	match p_class:
		"blade_dancer":
			_bot.set_script(BD_BOT)
		"vanguard":
			_bot.set_script(VANGUARD_BOT)
		_:
			_bot.set_script(GUNNER_BOT)
	player.add_child(_bot)
	trace("bot attached (%s)" % p_class)


func detach_bot() -> void:
	if _bot and is_instance_valid(_bot):
		_bot.queue_free()
		_bot = null


# =============================================================================
# Assertions
# =============================================================================


func assert_true(condition: bool, msg := "") -> void:
	if not condition:
		var text := "assert_true failed"
		if msg != "":
			text += ": %s" % msg
		fail(text)


func assert_eq(a: Variant, b: Variant, msg := "") -> void:
	if a != b:
		var text := "assert_eq failed: %s != %s" % [a, b]
		if msg != "":
			text += " (%s)" % msg
		fail(text)


func assert_state(expected: int) -> void:
	if main.state != expected:
		fail("expected state %d, got %d" % [expected, main.state])


func assert_env_valid() -> void:
	var env: Node3D = env_builder.current_env
	if not env or not is_instance_valid(env):
		fail("environment is null or freed")
		return
	if not env.is_inside_tree():
		fail("environment is not in scene tree")
	# Also check HUD doesn't hold a freed player reference
	var hud: Control = main._shared_hud
	if hud:
		var ref: Variant = hud._local_player
		if ref != null and not is_instance_valid(ref):
			fail("shared_hud._local_player is a freed reference")


func assert_player_alive() -> void:
	var player := _get_local_player()
	if not player:
		fail("no local player found")
		return
	if "health" in player and player.health <= 0:
		fail("local player is dead (hp=%d)" % player.health)


func assert_single_world_environment() -> void:
	var count := _count_world_environments(tree.root)
	if count > 1:
		fail("found %d WorldEnvironment nodes (expected <= 1)" % count)


func assert_viewport_not_grey(label := "") -> void:
	if not headed:
		trace("SKIP viewport check (headless): %s" % label)
		return

	var image: Image = main.get_viewport().get_texture().get_image()
	if not image:
		fail("viewport image is null: %s" % label)
		return

	# Sample a 3x3 grid from the center of the viewport (avoids HUD edges)
	var w := image.get_width()
	var h := image.get_height()
	var samples: Array[Color] = []
	for gy in [0.35, 0.5, 0.65]:
		for gx in [0.35, 0.5, 0.65]:
			samples.append(image.get_pixel(int(w * gx), int(h * gy)))

	# Check if all samples are nearly identical and grey
	var avg_r := 0.0
	var avg_g := 0.0
	var avg_b := 0.0
	for s in samples:
		avg_r += s.r
		avg_g += s.g
		avg_b += s.b
	avg_r /= samples.size()
	avg_g /= samples.size()
	avg_b /= samples.size()

	# Compute variance across samples
	var variance := 0.0
	for s in samples:
		variance += (s.r - avg_r) ** 2 + (s.g - avg_g) ** 2 + (s.b - avg_b) ** 2
	variance /= samples.size()

	# Grey screen: all pixels nearly identical, RGB close together, mid-range brightness
	var rgb_spread := maxf(maxf(absf(avg_r - avg_g), absf(avg_g - avg_b)), absf(avg_r - avg_b))
	var brightness := (avg_r + avg_g + avg_b) / 3.0
	var is_grey := variance < 0.001 and rgb_spread < 0.05 and brightness > 0.2 and brightness < 0.8

	if is_grey:
		fail(
			(
				"viewport is grey (avg=%.2f,%.2f,%.2f var=%.4f): %s"
				% [avg_r, avg_g, avg_b, variance, label]
			)
		)


# =============================================================================
# Logging
# =============================================================================


func fail(msg: String) -> void:
	failures.append(msg)
	print("[E2E FAIL] %s" % msg)


func trace(msg: String) -> void:
	print("[E2E] %s" % msg)


# =============================================================================
# Internal
# =============================================================================


func _count_world_environments(node: Node) -> int:
	var count := 0
	if node is WorldEnvironment:
		count += 1
	for child in node.get_children():
		count += _count_world_environments(child)
	return count
