extends "res://scripts/e2e/e2e_scenario.gd"

## Reproduce: vanguard grey screen on arena -> hub transfer.
## The vanguard camera has top_level=true and positions itself in
## _physics_process. After zone transfer, camera may stay at (0,0,0)
## while the player is at Y=-200.


func scenario_name() -> String:
	return "test_vanguard_zone_cycle"


func run(ctx: RefCounted) -> void:
	if not await ctx.dev_connect("vanguard", "hub", true):
		ctx.fail("connect failed")
		return

	if not await ctx.wait_for_state(ctx.main.GameState.HUB):
		ctx.fail("never reached HUB state")
		return

	await ctx.wait_seconds(0.5)
	await _assert_camera_near_player(ctx, "hub initial")

	# Teleport player to portal area and enter
	var player: CharacterBody3D = ctx._get_local_player()
	if not player:
		ctx.fail("no local player in hub")
		return
	var portal: Node3D = ctx.env_builder.current_env.get_node_or_null("PortalArea")
	if portal:
		player.global_position = portal.global_position
	await ctx.wait_seconds(0.2)
	ctx.enter_portal()

	if not await ctx.wait_for_state(ctx.main.GameState.ARENA_LOBBY, 20.0):
		ctx.fail("never reached ARENA_LOBBY")
		return

	await ctx.wait_seconds(0.5)
	await _assert_camera_near_player(ctx, "arena")

	# Teleport to arena exit portal and return
	player = ctx._get_local_player()
	if not player:
		ctx.fail("no local player in arena")
		return
	portal = ctx.env_builder.current_env.get_node_or_null("PortalArea")
	if portal:
		player.global_position = portal.global_position
	await ctx.wait_seconds(0.2)
	ctx.enter_portal()

	if not await ctx.wait_for_state(ctx.main.GameState.HUB, 20.0):
		ctx.fail("never returned to HUB")
		return

	await ctx.wait_seconds(0.5)
	await _assert_camera_near_player(ctx, "hub return")
	_dump_ui_state(ctx, "hub return")


func _dump_ui_state(ctx: RefCounted, label: String) -> void:
	var layers: Array[String] = []
	for child in ctx.main.get_children():
		if child is CanvasLayer and child.visible:
			layers.append(child.name)
	ctx.trace("visible CanvasLayers [%s]: %s" % [label, ", ".join(layers)])


func _assert_camera_near_player(ctx: RefCounted, label: String) -> void:
	var player: CharacterBody3D = ctx._get_local_player()
	if not player:
		ctx.fail("no local player: %s" % label)
		return
	var cam: Camera3D = ctx.main.get_viewport().get_camera_3d()
	if not cam:
		ctx.fail("no active camera: %s" % label)
		return
	var dist := cam.global_position.distance_to(player.global_position)
	ctx.trace(
		(
			"cam=%s player=%s dist=%.1f [%s]"
			% [cam.global_position, player.global_position, dist, label]
		)
	)
	if dist > 20.0:
		ctx.fail(
			(
				"camera %.1f from player [%s] cam=%s player=%s"
				% [dist, label, cam.global_position, player.global_position]
			)
		)
