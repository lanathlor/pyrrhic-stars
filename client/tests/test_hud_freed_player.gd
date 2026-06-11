class_name TestHudFreedPlayer
extends GdUnitTestSuite

## Regression: after arena -> hub zone transfer, shared_hud._draw() assigned
## a freed _local_player reference to _minimap.local_player, crashing with:
##   "Invalid assignment of property 'local_player' with value of type
##    'previously freed' on a base object of type 'RefCounted (MinimapRenderer)'"
## The guard at _draw() line 117 (_boss_visible = true) allows the minimap
## block to run even after the arena player is freed but before clear is called.

const SharedHudScript := preload("res://scenes/shared/hud/shared_hud.gd")

var _hud: Control


func before_test() -> void:
	_hud = auto_free(Control.new())
	_hud.set_script(SharedHudScript)
	add_child(_hud)
	await get_tree().process_frame


## Core crash scenario: _boss_visible is true (from arena world state),
## _local_player is a freed node (arena player was queue_freed during zone
## transfer), and _draw() fires before clear_local_player() nulls the ref.
func test_draw_with_freed_player_does_not_crash() -> void:
	var player := CharacterBody3D.new()
	add_child(player)
	_hud.set_local_player(player, "gunner", 1)
	_hud._boss_visible = true

	# Free the player WITHOUT calling clear_local_player — this simulates
	# the window during zone transfer where despawn queues the free but
	# _draw fires before the reference is nulled.
	player.queue_free()
	await get_tree().process_frame  # player is now freed

	# _local_player is a dangling reference. _draw() must handle this
	# without crashing. Trigger a redraw cycle.
	_hud.queue_redraw()
	await get_tree().process_frame

	# If we got here without a SCRIPT ERROR, the guard works.
	# The minimap's local_player must be null (not a freed ref).
	assert_that(_hud._minimap.local_player).is_null()


## Same scenario but with _hub_mode = true (returning to hub).
func test_draw_with_freed_player_hub_mode() -> void:  # gdlint:ignore
	var player := CharacterBody3D.new()
	add_child(player)
	_hud.set_local_player(player, "vanguard", 2)
	_hud._hub_mode = true

	player.queue_free()
	await get_tree().process_frame

	_hud.queue_redraw()
	await get_tree().process_frame

	var minimap_ref: CharacterBody3D = _hud._minimap.local_player
	var ref_ok := minimap_ref == null or is_instance_valid(minimap_ref)
	assert_bool(ref_ok).is_true()
