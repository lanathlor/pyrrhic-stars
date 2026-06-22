class_name TestZoneTransferEnv
extends GdUnitTestSuite

## Regression: on_zone_transfer to hub loaded the environment twice because
## it called unload+load explicitly and then enter_hub() which did the same
## check against current_env.name != "Hub" (root node is "MilitaryBuilding").
## This caused a zombie environment node and broke the next arena entry.

const GFM_SCRIPT := preload("res://scenes/main/game_flow_manager.gd")


## The hub path in on_zone_transfer must NOT load the environment before
## calling enter_hub, since enter_hub already handles environment loading.
## Alternatively, enter_hub must not reload if the environment was just loaded.
func test_hub_path_does_not_double_load() -> void:
	var src: String = (GFM_SCRIPT as GDScript).source_code

	# Extract the full on_zone_transfer body (up to the next top-level func), so
	# the search is not truncated by a fixed window as the function grows.
	var zt_idx := src.find("func on_zone_transfer")
	assert_int(zt_idx).is_greater(-1)
	var fn_tail := src.substr(zt_idx)
	var next_fn := fn_tail.find("\nfunc ")
	var zt_block: String = fn_tail.substr(0, next_fn) if next_fn != -1 else fn_tail

	# The hub path is the top-level else branch (one tab indent); the arena
	# branch's nested elses are deeper indented and must not be matched here.
	var else_idx := zt_block.find("\n\telse:")
	assert_int(else_idx).is_greater(-1)
	var hub_path := zt_block.substr(else_idx, 300)

	# The hub path must NOT call both load_environment AND enter_hub,
	# because enter_hub also calls load_environment internally.
	var has_load := hub_path.find("load_environment") != -1
	var has_enter_hub := hub_path.find("enter_hub") != -1

	# Either: hub path calls enter_hub only (which handles env loading itself)
	# Or: hub path loads env but doesn't call enter_hub
	# Both calling load_environment AND enter_hub is the double-load bug.
	if has_load and has_enter_hub:
		# Check if enter_hub was fixed to skip reload
		var enter_idx := src.find("func enter_hub")
		assert_int(enter_idx).is_greater(-1)
		var enter_block := src.substr(enter_idx, 600)
		# If enter_hub still has load_environment, we have a double-load
		var enter_loads := enter_block.find("load_environment") != -1
		assert_bool(enter_loads).is_false()
