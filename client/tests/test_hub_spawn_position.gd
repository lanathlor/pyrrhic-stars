class_name TestHubSpawnPosition
extends GdUnitTestSuite

## Regression: grey screen returning from arena to hub. The server sends
## OP_CHARACTER_STATE with the arena position right before OP_ZONE_TRANSFER.
## character_manager sets _has_saved_state=true with that arena position.
## enter_hub() then spawns the player at the arena coords inside the hub
## geometry (grey screen). Fix: on_zone_transfer clears _has_saved_state
## before calling enter_hub.

const GFMScript := preload("res://scenes/main/game_flow_manager.gd")


## on_zone_transfer must clear _has_saved_state BEFORE the enter_hub call.
## Without this, a stale OP_CHARACTER_STATE from the same packet batch
## makes enter_hub spawn the player at the arena position inside the hub.
func test_zone_transfer_clears_saved_state_before_enter_hub() -> void:
	var src: String = (GFMScript as GDScript).source_code
	var fn_idx := src.find("func on_zone_transfer")
	assert_int(fn_idx).is_greater(-1)

	var enter_hub_idx := src.find("enter_hub()", fn_idx)
	assert_int(enter_hub_idx).is_greater(-1)

	var before_enter_hub := src.substr(fn_idx, enter_hub_idx - fn_idx)
	var clears_saved := before_enter_hub.find("_has_saved_state = false") != -1
	assert_bool(clears_saved).is_true()


## The hub spawn path must not use _saved_hub_position when it equals
## Vector3.ZERO. The server sends (0,0,0) when it has no hub position (player
## was in arena). The guard lives in _spawn_local_hub_player, called by enter_hub.
func test_enter_hub_guards_zero_saved_position() -> void:
	var src: String = (GFMScript as GDScript).source_code
	var fn_idx := src.find("func _spawn_local_hub_player")
	assert_int(fn_idx).is_greater(-1)

	var fn_end := src.find("\nfunc ", fn_idx + 1)
	if fn_end < 0:
		fn_end = src.length()
	var fn_body := src.substr(fn_idx, fn_end - fn_idx)

	var has_zero_guard := fn_body.find("Vector3.ZERO") != -1
	assert_bool(has_zero_guard).is_true()
