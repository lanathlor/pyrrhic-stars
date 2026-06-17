class_name TestDungeonTimer
extends GdUnitTestSuite

## The dungeon clear timer (HUD count-down) must keep running through a party
## wipe. On the server the run clock (FightStartTick) never stops for a wipe -
## players respawn and run back while the timer keeps ticking, so the run can
## roll into OVERTIME. The HUD must mirror that: a wipe is not a fight end.
##
## Regression: on_all_dead() used to call on_fight_end(), which froze
## _fight_active and stopped the count-down, making the timer freeze on death
## and OVERTIME unreachable on the HUD.

const SharedHudScript := preload("res://scenes/shared/hud/shared_hud.gd")

var _hud: Control


func before_test() -> void:
	_hud = auto_free(Control.new())
	_hud.set_script(SharedHudScript)


# Drive the per-frame timer the way _process does, without a render frame.
func _advance(seconds: float, steps: int = 10) -> void:
	var dt := seconds / float(steps)
	for _i in range(steps):
		_hud._process(dt)


func test_timer_advances_during_fight() -> void:
	_hud.set_time_limit(300.0)
	_hud.on_fight_start()
	_advance(5.0)
	assert_float(_hud._fight_duration).is_greater(4.5)


func test_wipe_does_not_freeze_timer() -> void:
	_hud.set_time_limit(300.0)
	_hud.on_fight_start()
	_advance(5.0)
	var before: float = _hud._fight_duration

	# Party wipe: the run continues, the clock must keep ticking.
	_hud.on_wipe()
	_advance(5.0)

	assert_bool(_hud._fight_active).is_true()
	assert_float(_hud._fight_duration).is_greater(before + 4.5)


func test_victory_stops_timer() -> void:
	# Contrast: a boss kill (real fight end) freezes the count-down.
	_hud.set_time_limit(300.0)
	_hud.on_fight_start()
	_advance(5.0)
	var before: float = _hud._fight_duration

	_hud.on_fight_end()
	_advance(5.0)

	assert_bool(_hud._fight_active).is_false()
	assert_float(_hud._fight_duration).is_equal_approx(before, 0.001)
