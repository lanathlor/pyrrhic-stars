class_name TestVanguardHUD
extends GdUnitTestSuite

## Tests for the Vanguard HUD — damage/parry flash, hit marker, lock-on reticle.

const VANGUARD_SCENE := "res://scenes/controllers/vanguard/vanguard.tscn"
const DELTA := 1.0 / 60.0

var _vanguard: CharacterBody3D
var _hud: Control


func before_test() -> void:
	_vanguard = auto_free(load(VANGUARD_SCENE).instantiate())
	add_child(_vanguard)
	await get_tree().process_frame
	_hud = _vanguard.hud


# --- Damage flash ---

func test_damage_flash_sets_timer() -> void:
	_hud.show_damage_flash()
	assert_float(_hud._damage_flash_timer).is_greater(0.0)


func test_damage_flash_modulates_overlay() -> void:
	_hud.show_damage_flash()
	_hud._process(DELTA)
	var overlay: ColorRect = _hud.get_node("DamageOverlay")
	assert_float(overlay.modulate.a).is_greater(0.0)


func test_damage_flash_sets_red_color() -> void:
	_hud.show_damage_flash()
	var overlay: ColorRect = _hud.get_node("DamageOverlay")
	# Damage flash sets overlay to red
	assert_float(overlay.color.r).is_equal_approx(0.8, 0.01)
	assert_float(overlay.color.g).is_equal(0.0)


func test_damage_flash_decays_over_time() -> void:
	_hud.show_damage_flash()
	var initial := _hud._damage_flash_timer
	_hud._process(DELTA)
	assert_float(_hud._damage_flash_timer).is_less(initial)


# --- Parry flash ---

func test_parry_flash_sets_timer() -> void:
	_hud.show_parry_flash()
	assert_float(_hud._parry_flash_timer).is_greater(0.0)


func test_parry_flash_turns_overlay_white() -> void:
	_hud.show_parry_flash()
	_hud._process(DELTA)
	var overlay: ColorRect = _hud.get_node("DamageOverlay")
	# While parry timer is active, overlay should be white
	assert_float(overlay.color.r).is_equal(1.0)
	assert_float(overlay.color.g).is_equal(1.0)
	assert_float(overlay.color.b).is_equal(1.0)


func test_parry_flash_restores_red_after_expiry() -> void:
	_hud.show_parry_flash()
	# Simulate enough frames to expire parry timer
	var frames := ceili(_hud.PARRY_FLASH_DURATION / DELTA) + 2
	for i in frames:
		_hud._process(DELTA)
	var overlay: ColorRect = _hud.get_node("DamageOverlay")
	assert_float(overlay.color.r).is_equal_approx(0.8, 0.01)
	assert_float(overlay.color.g).is_equal(0.0)


func test_parry_flash_decays_over_time() -> void:
	_hud.show_parry_flash()
	var initial := _hud._parry_flash_timer
	_hud._process(DELTA)
	assert_float(_hud._parry_flash_timer).is_less(initial)


# --- Hit marker ---

func test_hit_marker_sets_timer() -> void:
	_hud.show_hit_marker()
	assert_float(_hud._hit_marker_timer).is_greater(0.0)


func test_hit_marker_decays() -> void:
	_hud.show_hit_marker()
	var initial := _hud._hit_marker_timer
	_hud._process(DELTA)
	assert_float(_hud._hit_marker_timer).is_less(initial)


func test_hit_marker_reaches_zero() -> void:
	_hud.show_hit_marker()
	var frames := ceili(_hud.HIT_MARKER_DURATION / DELTA) + 2
	for i in frames:
		_hud._process(DELTA)
	assert_float(_hud._hit_marker_timer).is_less_equal(0.0)


# --- Lock-on reticle ---

func test_lock_on_show() -> void:
	_hud.show_lock_on()
	var reticle: Control = _hud.get_node("LockOnReticle")
	assert_bool(reticle.visible).is_true()
	assert_bool(reticle._lock_active).is_true()


func test_lock_on_hide() -> void:
	_hud.show_lock_on()
	_hud.hide_lock_on()
	var reticle: Control = _hud.get_node("LockOnReticle")
	assert_bool(reticle._lock_active).is_false()


func test_lock_on_update_stores_meta() -> void:
	var target := auto_free(Node3D.new())
	var cam := auto_free(Camera3D.new())
	_hud.update_lock_on(target, cam)
	var reticle: Control = _hud.get_node("LockOnReticle")
	assert_that(reticle.get_meta("lock_target")).is_same(target)
	assert_that(reticle.get_meta("lock_camera")).is_same(cam)


# --- Duration constants ---

func test_damage_flash_duration_constant() -> void:
	assert_float(_hud.DAMAGE_FLASH_DURATION).is_equal(0.3)


func test_parry_flash_duration_constant() -> void:
	assert_float(_hud.PARRY_FLASH_DURATION).is_equal(0.25)


func test_hit_marker_duration_constant() -> void:
	assert_float(_hud.HIT_MARKER_DURATION).is_equal(0.15)
