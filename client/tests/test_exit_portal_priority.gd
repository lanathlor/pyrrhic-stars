class_name TestExitPortalPriority
extends GdUnitTestSuite

## Regression test: pressing E at a portal in an instanced zone must
## send the player back directly. It must NOT open the overflux panel.
## The overflux panel only makes sense when entering an instance from the hub.
##
## Bug: the E-key handler opened _overflux_panel.open() whenever near_portal
## was true, regardless of zone type. Fix: overflux only in HUB state; all
## other states send enter_portal directly.

const MAIN_SCRIPT := preload("res://scenes/main.gd")


## Verify that the overflux panel only opens in HUB state.
## In any other state, pressing E at a portal should send enter_portal.
func test_overflux_only_opens_in_hub_state() -> void:
	var src: String = (MAIN_SCRIPT as GDScript).source_code

	# Find the E-key handler block
	var handler_idx := src.find("func _handle_gameplay_input")
	assert_int(handler_idx).is_greater(-1)
	var handler_block := src.substr(handler_idx, 1200)

	# The overflux_panel.open() call must be guarded by a HUB state check.
	# Find the overflux open line and check it's inside a HUB-state block.
	var overflux_idx := handler_block.find("_overflux_panel.open")
	assert_int(overflux_idx).is_greater(-1)

	# There should be a "state == GameState.HUB" check before _overflux_panel.open
	var before_overflux := handler_block.substr(0, overflux_idx)
	assert_str(before_overflux).contains("GameState.HUB")


## Verify that the non-HUB portal path sends enter_portal directly.
func test_instance_portal_sends_enter_portal_directly() -> void:
	var src: String = (MAIN_SCRIPT as GDScript).source_code
	var handler_idx := src.find("func _handle_gameplay_input")
	assert_int(handler_idx).is_greater(-1)
	var handler_block := src.substr(handler_idx, 1200)

	# There should be an else branch after the "state == GameState.HUB" check
	# that calls send_enter_portal (no overflux).
	var hub_check := handler_block.find("state == GameState.HUB")
	assert_int(hub_check).is_greater(-1)
	var after_hub := handler_block.substr(hub_check, 400)
	# The else branch should contain send_enter_portal
	assert_str(after_hub).contains("send_enter_portal")
