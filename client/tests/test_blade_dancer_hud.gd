class_name TestBladeDancerHUD
extends GdUnitTestSuite

## Tests for the Blade Dancer HUD — config display, GCD, damage flash, hit marker, lock-on.

const BD_SCENE := "res://scenes/controllers/blade_dancer/blade_dancer.tscn"
const DELTA := 1.0 / 60.0

var _bd: CharacterBody3D
var _hud: Control


func before_test() -> void:
	_bd = auto_free(load(BD_SCENE).instantiate())
	add_child(_bd)
	await get_tree().process_frame
	_hud = _bd.hud


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
	assert_float(overlay.color.r).is_equal_approx(0.8, 0.01)
	assert_float(overlay.color.g).is_equal(0.0)


func test_damage_flash_decays_over_time() -> void:
	_hud.show_damage_flash()
	var initial := _hud._damage_flash_timer
	_hud._process(DELTA)
	assert_float(_hud._damage_flash_timer).is_less(initial)


func test_damage_flash_reaches_zero() -> void:
	_hud.show_damage_flash()
	var frames := ceili(_hud.DAMAGE_FLASH_DURATION / DELTA) + 2
	for i in frames:
		_hud._process(DELTA)
	assert_float(_hud._damage_flash_timer).is_less_equal(0.0)


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


# --- Config display ---

func test_update_config_stores_orbit() -> void:
	_hud.update_config(0)
	assert_int(_hud._current_config).is_equal(0)


func test_update_config_stores_lance() -> void:
	_hud.update_config(1)
	assert_int(_hud._current_config).is_equal(1)


func test_update_config_initial_orbit() -> void:
	# HUD starts in orbit (set by blade_dancer _ready when local)
	assert_int(_hud._current_config).is_equal(0)


# --- GCD ---

func test_update_gcd_stores_ratio() -> void:
	_hud.update_gcd(0.5)
	assert_float(_hud._gcd_ratio).is_equal(0.5)


func test_update_gcd_clamps_above_one() -> void:
	_hud.update_gcd(1.5)
	assert_float(_hud._gcd_ratio).is_equal(1.0)


func test_update_gcd_clamps_below_zero() -> void:
	_hud.update_gcd(-0.5)
	assert_float(_hud._gcd_ratio).is_equal(0.0)


func test_update_gcd_zero_means_ready() -> void:
	_hud.update_gcd(0.0)
	assert_float(_hud._gcd_ratio).is_equal(0.0)


func test_update_gcd_one_means_just_used() -> void:
	_hud.update_gcd(1.0)
	assert_float(_hud._gcd_ratio).is_equal(1.0)


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


func test_hit_marker_duration_constant() -> void:
	assert_float(_hud.HIT_MARKER_DURATION).is_equal(0.15)


# --- Color constants ---

func test_orbit_color_is_cyan() -> void:
	assert_float(_hud.ORBIT_COLOR.r).is_equal_approx(0.2, 0.01)
	assert_float(_hud.ORBIT_COLOR.g).is_equal_approx(0.8, 0.01)
	assert_float(_hud.ORBIT_COLOR.b).is_equal_approx(0.9, 0.01)


func test_lance_color_is_orange() -> void:
	assert_float(_hud.LANCE_COLOR.r).is_equal_approx(1.0, 0.01)
	assert_float(_hud.LANCE_COLOR.g).is_equal_approx(0.6, 0.01)
	assert_float(_hud.LANCE_COLOR.b).is_equal_approx(0.1, 0.01)


# --- Ability bar data ---

func test_ability_names_has_four_entries() -> void:
	assert_int(_hud.ABILITY_NAMES.size()).is_equal(4)


func test_ability_keybinds_has_four_entries() -> void:
	assert_int(_hud.ABILITY_KEYBINDS.size()).is_equal(4)


func test_ability_names_orbit_index() -> void:
	# Each ability name array: [orbit_name, lance_name]
	assert_str(_hud.ABILITY_NAMES[0][0]).is_equal("Slash")
	assert_str(_hud.ABILITY_NAMES[1][0]).is_equal("Launch")
	assert_str(_hud.ABILITY_NAMES[2][0]).is_equal("Barrier")
	assert_str(_hud.ABILITY_NAMES[3][0]).is_equal("Dash")


func test_ability_names_lance_index() -> void:
	assert_str(_hud.ABILITY_NAMES[0][1]).is_equal("Pierce")
	assert_str(_hud.ABILITY_NAMES[1][1]).is_equal("Impale")
	assert_str(_hud.ABILITY_NAMES[2][1]).is_equal("Recall")
	assert_str(_hud.ABILITY_NAMES[3][1]).is_equal("Retreat")
