extends "res://scripts/e2e/e2e_scenario.gd"

## Smoke test: connect to server, land in hub, verify basics.


func scenario_name() -> String:
	return "test_connect_hub"


func run(ctx: RefCounted) -> void:
	if not await ctx.dev_connect("gunner", "hub"):
		ctx.fail("connect failed")
		return

	if not await ctx.wait_for_state(ctx.main.GameState.HUB):
		ctx.fail("never reached HUB state")
		return

	ctx.assert_state(ctx.main.GameState.HUB)
	ctx.assert_env_valid()
	ctx.assert_player_alive()
	ctx.assert_single_world_environment()
	ctx.assert_viewport_not_grey("hub after connect")

	ctx.trace("smoke test passed")
