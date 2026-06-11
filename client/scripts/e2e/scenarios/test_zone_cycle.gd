extends "res://scripts/e2e/e2e_scenario.gd"

## Regression test for rendering bugs after zone transfers.
## Cycles hub -> arena -> hub N times using the portal at the arena
## entrance. Tests both normal and overflux portal entry paths.
## Waits for boss world state before exiting to exercise the minimap
## draw path. Asserts environment integrity at each transition.

const CYCLES := 3


func scenario_name() -> String:
	return "test_zone_cycle"


func run(ctx: RefCounted) -> void:
	if not await ctx.dev_connect("gunner", "hub"):
		ctx.fail("connect failed")
		return

	if not await ctx.wait_for_state(ctx.main.GameState.HUB):
		ctx.fail("never reached HUB state")
		return

	ctx.assert_env_valid()
	ctx.assert_viewport_not_grey("hub initial")

	for i in CYCLES:
		var cycle := i + 1
		# Alternate between normal and overflux portal entry
		var use_overflux := cycle % 2 == 0
		if not await _do_cycle(ctx, cycle, use_overflux):
			return

	ctx.trace("all %d cycles passed" % CYCLES)


func _do_cycle(ctx: RefCounted, cycle: int, use_overflux: bool) -> bool:
	var mode := "overflux" if use_overflux else "normal"
	ctx.trace("cycle %d/%d (%s): hub -> arena" % [cycle, CYCLES, mode])

	if not await ctx.walk_to_portal():
		ctx.fail("could not reach hub portal (cycle %d)" % cycle)
		return false

	# Pass dummy overflux conditions on overflux cycles to exercise that path
	if use_overflux:
		ctx.enter_portal(["fortified"])
	else:
		ctx.enter_portal()

	if not await ctx.wait_for_state(ctx.main.GameState.ARENA_LOBBY, 20.0):
		ctx.fail("never reached ARENA_LOBBY (cycle %d)" % cycle)
		return false
	ctx.assert_env_valid()
	ctx.assert_single_world_environment()
	ctx.assert_viewport_not_grey("arena cycle %d" % cycle)

	# Brief pause to let server ticks arrive (world state, overflux state)
	await ctx.wait_seconds(0.5)

	# Arena -> hub via the entrance portal
	ctx.trace("cycle %d/%d: arena -> hub" % [cycle, CYCLES])
	if not await ctx.walk_to_portal():
		ctx.fail("could not reach arena portal (cycle %d)" % cycle)
		return false
	ctx.enter_portal()

	if not await ctx.wait_for_state(ctx.main.GameState.HUB, 20.0):
		ctx.fail("never returned to HUB (cycle %d)" % cycle)
		return false

	# Check freed-reference bug and environment integrity
	# Check integrity over several frames to catch deferred-free timing
	for f in 5:
		await ctx.tree.process_frame
		ctx.assert_env_valid()
		ctx.assert_single_world_environment()

	ctx.assert_env_valid()
	ctx.assert_viewport_not_grey("hub return cycle %d" % cycle)
	ctx.trace("cycle %d/%d: OK" % [cycle, CYCLES])
	return true
