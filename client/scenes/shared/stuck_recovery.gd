class_name StuckRecovery
extends RefCounted

## Detects when a local player capsule is wedged in level geometry and frees it.
##
## A wedge presents as: the player is actively commanding movement but the body
## barely moves for a sustained window (classic CharacterBody3D corner pinch on
## thin colliders). Two recovery paths:
##
##   1. Local: snap back to the last position where the body was moving healthily
##      on the floor - a spot it legitimately occupied moments ago, so the
##      correction is small and the server accepts it without a teleport reject.
##   2. Escalate: when no safe position was ever recorded (e.g. the player logged
##      in directly on top of a wedge), there is nothing local to fall back to, so
##      ask the caller to fire the server-authoritative unstuck (respawn type 2),
##      the same teleport-to-hub-spawn the manual button uses. A cooldown keeps it
##      from spamming the server while the snap-back propagates.
##
## `track()` is a pure function over its inputs and internal state so it can be
## unit-tested without a live physics body. The owning controller applies the
## returned position and honours `wants_server_unstuck`.

## Seconds of commanded-but-stuck movement before a recovery fires.
const STUCK_TIME := 1.2
## Horizontal speed (units/sec) below which the body counts as "not moving".
const MOVE_EPS := 0.5
## Horizontal speed (units/sec) above which we record a fresh safe position.
const SAFE_SPEED := 2.0
## Seconds to wait between escalations so we do not spam respawn requests.
const ESCALATE_COOLDOWN := 3.0

## Set true by track() for one tick when the caller should fire the server
## unstuck (no local safe position to recover to). Read it after track()/apply().
var wants_server_unstuck := false

var _last_pos := Vector3.ZERO
var _has_last := false
var _last_safe := Vector3.ZERO
var _has_safe := false
var _stuck_time := 0.0
var _escalate_cooldown := 0.0


## Returns the position the body should occupy this frame. Normally equal to
## `pos`; when a wedge is detected with a known safe position it returns that
## position (the caller should teleport there and zero velocity). When wedged
## with no safe position it leaves `pos` unchanged and sets wants_server_unstuck.
func track(pos: Vector3, on_floor: bool, commanding: bool, delta: float) -> Vector3:
	wants_server_unstuck = false
	if _escalate_cooldown > 0.0:
		_escalate_cooldown -= delta

	if not _has_last:
		_last_pos = pos
		_has_last = true
		return pos

	var d := delta if delta > 0.0001 else 0.0001
	var horiz_speed := Vector2(pos.x - _last_pos.x, pos.z - _last_pos.z).length() / d
	_last_pos = pos

	# Remember the most recent spot where the player was moving freely on ground.
	if on_floor and horiz_speed > SAFE_SPEED:
		_last_safe = pos
		_has_safe = true

	# Commanding movement but going nowhere -> accumulate stuck time.
	if commanding and horiz_speed < MOVE_EPS:
		_stuck_time += delta
	else:
		_stuck_time = 0.0

	if _stuck_time < STUCK_TIME:
		return pos

	_stuck_time = 0.0
	if _has_safe:
		_last_pos = _last_safe
		return _last_safe
	# Nothing local to recover to (logged in on a wedge): escalate to the server.
	if _escalate_cooldown <= 0.0:
		wants_server_unstuck = true
		_escalate_cooldown = ESCALATE_COOLDOWN
	return pos


## Convenience for controllers: runs track() against `body`, teleports it free
## (zeroing velocity) on a local recovery, and returns true when the caller
## should fire the server unstuck (NetworkManager.send_respawn_request(2)).
## `commanding` is whether the player is actively requesting movement this frame.
## Call once per physics frame after move_and_slide(), for the local player only.
func apply(body: CharacterBody3D, commanding: bool, delta: float) -> bool:
	var pos: Vector3 = track(body.global_position, body.is_on_floor(), commanding, delta)
	if not pos.is_equal_approx(body.global_position):
		body.global_position = pos
		body.velocity = Vector3.ZERO
	return wants_server_unstuck
