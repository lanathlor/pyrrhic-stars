class_name TestGunnerMagazineSync
extends GdUnitTestSuite

## Adversarial tests for magazine/reload client prediction vs server sync.
##
## These reproduce the exact bugs:
## 1. Client-predicted reload cancelled by stale server state
##    (server hasn't seen reload request yet, says reloading=false,
##    client interprets as "reload completed" and accepts magazine=30)
## 2. Sustained fire rollback: every shot rolls back magazine by 1
##    because server world state is always 1 tick behind client prediction

const GunnerScript := preload("res://scenes/controllers/gunner/gunner.gd")
const GUNNER_SCENE := "res://scenes/controllers/gunner/gunner.tscn"

var _gunner: GunnerScript


func before_test() -> void:
	_gunner = auto_free(load(GUNNER_SCENE).instantiate()) as GunnerScript
	_gunner.position = Vector3(0.0, 5.0, 0.0)
	add_child(_gunner)
	await get_tree().process_frame


func after_test() -> void:
	for action in ["shoot", "reload", "dodge"]:
		if Input.is_action_pressed(action):
			Input.action_release(action)


## Helper: build a minimal server state dict for a local gunner.
func _server_state(overrides: Dictionary = {}) -> Dictionary:
	var base := {
		"pos": Vector3.ZERO,
		"rot_y": 0.0,
		"health": 150.0,
		"max_health": 150.0,
		"state": 0,
		"visual_state": 0,
		"aim_pitch": 0.0,
		"magazine": 30,
		"mag_max": 30,
		"reloading": false,
		"stability": 1.0,
		"pressure_stacks": 0,
		"enhanced_loaded": 0,
		"mag_dump_active": false,
	}
	base.merge(overrides, true)
	return base


# ==========================================================================
# BUG 1: Client-predicted reload cancelled by stale server state
# ==========================================================================


func test_reload_not_cancelled_by_stale_server_state() -> void:
	# Client fires all ammo, starts reloading
	_gunner._magazine = 0
	_gunner._reloading = true
	_gunner._reload_timer = 2.0
	_gunner._reload_total = 2.2

	# Server state arrives from BEFORE it saw the reload request.
	# Server still says: reloading=false, magazine=0
	_gunner.apply_server_state(_server_state({
		"magazine": 0,
		"reloading": false,
	}))

	# Client reload must NOT be cancelled — server just hasn't seen it yet
	assert_bool(_gunner._reloading).is_true()
	assert_float(_gunner._reload_timer).is_greater(0.0)


func test_reload_not_cancelled_by_stale_server_with_higher_mag() -> void:
	# Client optimistically decremented to 0 and started reload.
	# Server hasn't processed last few shots: magazine=3, reloading=false
	_gunner._magazine = 0
	_gunner._reloading = true
	_gunner._reload_timer = 2.0
	_gunner._reload_total = 2.2

	_gunner.apply_server_state(_server_state({
		"magazine": 3,
		"reloading": false,
	}))

	# Must NOT cancel reload or slam magazine to 3
	assert_bool(_gunner._reloading).is_true()
	assert_int(_gunner._magazine).is_equal(0)


func test_reload_completes_when_server_confirms() -> void:
	# Client is reloading, server confirms reloading too
	_gunner._magazine = 0
	_gunner._reloading = true
	_gunner._reload_timer = 1.0
	_gunner._reload_total = 2.2

	# Server acks reload
	_gunner.apply_server_state(_server_state({
		"magazine": 0,
		"reloading": true,
	}))
	assert_bool(_gunner._reloading).is_true()

	# Later: server says reload done, magazine full
	_gunner.apply_server_state(_server_state({
		"magazine": 30,
		"reloading": false,
	}))

	# NOW it should accept the completion
	assert_bool(_gunner._reloading).is_false()
	assert_int(_gunner._magazine).is_equal(30)
	assert_float(_gunner._reload_timer).is_equal(0.0)


func test_server_initiated_reload_accepted() -> void:
	# Client doesn't think it's reloading, server says it is
	_gunner._magazine = 5
	_gunner._reloading = false

	_gunner.apply_server_state(_server_state({
		"magazine": 0,
		"reloading": true,
	}))

	assert_bool(_gunner._reloading).is_true()
	assert_int(_gunner._magazine).is_equal(0)


# ==========================================================================
# BUG 2: Sustained fire rollback — server_mag > client_mag every shot
# ==========================================================================


func test_sustained_fire_no_rollback() -> void:
	# Simulate: client fired 2 shots (magazine 30 → 28)
	# Server only processed 1 so far (magazine=29)
	_gunner._magazine = 28
	_gunner._reloading = false

	_gunner.apply_server_state(_server_state({
		"magazine": 29,
		"reloading": false,
	}))

	# Client should keep its prediction (28), not rollback to 29
	assert_int(_gunner._magazine).is_equal(28)


func test_sustained_fire_full_sequence_no_rollback() -> void:
	# Simulate the exact pattern from the bug report:
	# Client fires a shot each step, server is always 1 behind
	_gunner._magazine = 30
	_gunner._reloading = false
	var rollbacks := 0

	for i in range(29):
		# Client fires
		_gunner._magazine -= 1
		var client_mag: int = _gunner._magazine

		# Server state from previous tick (1 behind)
		var server_mag: int = 30 - i  # server hasn't processed this shot yet
		_gunner.apply_server_state(_server_state({
			"magazine": server_mag,
			"reloading": false,
		}))

		if _gunner._magazine > client_mag:
			rollbacks += 1

	assert_int(rollbacks).is_equal(0)
	# Client should show 1 (fired 29 shots from 30)
	assert_int(_gunner._magazine).is_equal(1)


func test_server_downward_correction_accepted() -> void:
	# Server says fewer rounds than client — genuine rejection, accept it
	_gunner._magazine = 25
	_gunner._reloading = false

	_gunner.apply_server_state(_server_state({
		"magazine": 23,
		"reloading": false,
	}))

	assert_int(_gunner._magazine).is_equal(23)


func test_server_equal_magazine_no_change() -> void:
	# Server and client agree
	_gunner._magazine = 20
	_gunner._reloading = false

	_gunner.apply_server_state(_server_state({
		"magazine": 20,
		"reloading": false,
	}))

	assert_int(_gunner._magazine).is_equal(20)


# ==========================================================================
# Reload bar must clear when server says done (no lingering bar)
# ==========================================================================


func test_reload_bar_clears_on_server_completion() -> void:
	# Client is reloading, server has already confirmed
	_gunner._magazine = 0
	_gunner._reloading = true
	_gunner._reload_timer = 0.5  # client bar still has time left
	_gunner._reload_total = 2.2

	# Server confirms reload in progress first
	_gunner.apply_server_state(_server_state({
		"magazine": 0,
		"reloading": true,
	}))
	assert_bool(_gunner._reloading).is_true()

	# Server says done
	_gunner.apply_server_state(_server_state({
		"magazine": 30,
		"reloading": false,
	}))

	assert_bool(_gunner._reloading).is_false()
	assert_int(_gunner._magazine).is_equal(30)
	assert_float(_gunner._reload_timer).is_equal(0.0)
	assert_float(_gunner._reload_total).is_equal(0.0)


# ==========================================================================
# No double-reload: client completes reload, server still says reloading
# ==========================================================================


func test_no_double_reload_after_client_completion() -> void:
	# Client starts reload, server acks it
	_gunner._magazine = 0
	_gunner._reloading = true
	_gunner._reload_timer = 0.1
	_gunner._reload_total = 2.2
	_gunner._reload_server_acked = true

	# Client timer expires — reload completes locally
	_gunner._magazine = 30
	_gunner._reloading = false
	_gunner._reload_timer = 0.0
	_gunner._reload_total = 0.0
	# This is what the timer completion sets:
	_gunner._reload_server_acked = true

	# Server state arrives still showing reloading=true (hasn't finished yet)
	_gunner.apply_server_state(_server_state({
		"magazine": 0,
		"reloading": true,
	}))

	# Must NOT re-enter reload — client already completed
	assert_bool(_gunner._reloading).is_false()
	assert_int(_gunner._magazine).is_equal(30)


func test_ack_flag_resets_for_next_reload_cycle() -> void:
	# After a reload cycle completes, the ack flag must reset
	# so the next reload can be tracked properly
	_gunner._magazine = 30
	_gunner._reloading = false
	_gunner._reload_server_acked = true  # leftover from previous reload

	# Server agrees: not reloading
	_gunner.apply_server_state(_server_state({
		"magazine": 30,
		"reloading": false,
	}))

	# Ack flag should be cleared
	assert_bool(_gunner._reload_server_acked).is_false()

	# Now a new reload should be accepted from server
	_gunner._magazine = 0
	_gunner.apply_server_state(_server_state({
		"magazine": 0,
		"reloading": true,
	}))

	assert_bool(_gunner._reloading).is_true()
	assert_bool(_gunner._reload_server_acked).is_true()
