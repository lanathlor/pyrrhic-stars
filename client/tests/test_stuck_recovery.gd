class_name TestStuckRecovery
extends GdUnitTestSuite

## Tests for the StuckRecovery wedge detector. Pure logic, no physics body.

const StuckRecoveryScript := preload("res://scenes/shared/stuck_recovery.gd")

const DELTA := 0.1
# 0.5u per 0.1s = 5 u/s, comfortably above SAFE_SPEED.
const STEP := Vector3(0.5, 0.0, 0.0)


# Feed `frames` of healthy on-floor movement starting at `start`. Returns the
# last position that was actually fed to track() (the safe fallback spot).
func _move_freely(sr: RefCounted, frames: int, start: Vector3) -> Vector3:
	var pos := start
	var last := start
	for i in frames:
		var out: Vector3 = sr.track(pos, true, true, DELTA)
		assert_vector(out).is_equal(pos)  # healthy movement is never overridden
		last = pos
		pos += STEP
	return last


func test_no_recovery_while_moving() -> void:
	var sr := StuckRecoveryScript.new()
	_move_freely(sr, 20, Vector3.ZERO)


func test_recovers_to_last_safe_when_wedged() -> void:
	var sr := StuckRecoveryScript.new()
	var safe := _move_freely(sr, 6, Vector3.ZERO)
	# Hit a wall: creep forward a hair (below MOVE_EPS*delta) and stick there.
	var wedge := safe + Vector3(0.04, 0.0, 0.0)

	var recovered_at := -1
	for i in 20:
		var out: Vector3 = sr.track(wedge, true, true, DELTA)
		if not out.is_equal_approx(wedge):
			recovered_at = i
			assert_vector(out).is_equal(safe)
			break
	# STUCK_TIME = 1.2s / 0.1s per frame = recovery on the 12th wedged frame.
	assert_int(recovered_at).is_greater_equal(11)


func test_no_recovery_when_not_commanding() -> void:
	var sr := StuckRecoveryScript.new()
	var safe := _move_freely(sr, 6, Vector3.ZERO)
	var wedge := safe + Vector3(0.04, 0.0, 0.0)

	# Standing still by choice (not commanding) must never trigger recovery.
	for i in 30:
		var out: Vector3 = sr.track(wedge, true, false, DELTA)
		assert_vector(out).is_equal(wedge)


func test_escalates_to_server_when_no_safe_position() -> void:
	var sr := StuckRecoveryScript.new()
	var wedge := Vector3(3.0, 100.0, 13.0)

	# Wedged immediately (e.g. logged in on a wedge), never moved freely: there is
	# no local position to recover to, so track must leave the position untouched
	# and ask the caller to fire the server unstuck.
	var escalated := false
	for i in 20:
		var out: Vector3 = sr.track(wedge, true, true, DELTA)
		assert_vector(out).is_equal(wedge)  # never invents a local position
		if sr.wants_server_unstuck:
			escalated = true
			# STUCK_TIME = 1.2s / 0.1s per frame.
			assert_int(i).is_greater_equal(11)
			break
	assert_bool(escalated).is_true()


func test_escalation_is_rate_limited() -> void:
	var sr := StuckRecoveryScript.new()
	var wedge := Vector3(3.0, 100.0, 13.0)

	# Run long enough to wedge twice over; the cooldown must prevent a second
	# escalation from firing back-to-back.
	var escalations := 0
	for i in 30:
		sr.track(wedge, true, true, DELTA)
		if sr.wants_server_unstuck:
			escalations += 1
	assert_int(escalations).is_equal(1)


func test_no_escalation_when_not_commanding() -> void:
	var sr := StuckRecoveryScript.new()
	var wedge := Vector3(3.0, 100.0, 13.0)

	for i in 30:
		sr.track(wedge, true, false, DELTA)
		assert_bool(sr.wants_server_unstuck).is_false()
