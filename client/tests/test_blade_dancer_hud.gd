class_name TestBladeDancerHUD
extends GdUnitTestSuite

## Tests for the Blade Dancer HUD — config display, GCD, damage flash, hit marker, lock-on, ability bar.

const BDScript := preload("res://scenes/controllers/blade_dancer/blade_dancer.gd")
const BDHudScript := preload("res://scenes/shared/hud/blade_dancer_hud.gd")
const BD_SCENE := "res://scenes/controllers/blade_dancer/blade_dancer.tscn"
const DELTA := 1.0 / 60.0

var _bd: BDScript
var _hud: BDHudScript


func before_test() -> void:
	_bd = auto_free(load(BD_SCENE).instantiate()) as BDScript
	add_child(_bd)
	await get_tree().process_frame
	_hud = _bd.hud as BDHudScript


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


func test_update_config_stores_fan() -> void:
	_hud.update_config(1)
	assert_int(_hud._current_config).is_equal(1)


func test_update_config_initial_orbit() -> void:
	assert_int(_hud._current_config).is_equal(0)


func test_update_config_updates_accent_color() -> void:
	_hud.update_config(2)  # Lance -- red
	var bar: Control = _hud.get_node("AbilityBar")
	assert_float(bar.accent_color.r).is_equal_approx(0.9, 0.01)
	assert_float(bar.accent_color.g).is_equal_approx(0.2, 0.01)


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


func test_update_gcd_passes_to_ability_bar() -> void:
	_hud.update_gcd(0.7)
	var bar: Control = _hud.get_node("AbilityBar")
	assert_float(bar._gcd_ratio).is_equal_approx(0.7, 0.01)


# --- Ability bar ---


func test_update_abilities_enriches_keybinds() -> void:
	var abilities := [
		{name = "Test Ability", desc = "A test.", dest = 1, dur = 0.3},
	]
	_hud.update_abilities(abilities)
	var bar: Control = _hud.get_node("AbilityBar")
	assert_str(bar._abilities[0].keybind).is_equal("LMB")


func test_update_abilities_passes_four_slots() -> void:
	var abilities := [
		{name = "A", desc = "", dest = 1, dur = 0.3},
		{name = "B", desc = "", dest = 2, dur = 0.3},
		{name = "C", desc = "", dest = 3, dur = 0.4},
		{name = "D", desc = "", dest = 4, dur = 0.5},
	]
	_hud.update_abilities(abilities)
	var bar: Control = _hud.get_node("AbilityBar")
	assert_int(bar._abilities.size()).is_equal(4)
	assert_str(bar._abilities[3].keybind).is_equal("E")


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
	var target: Node3D = auto_free(Node3D.new())
	var cam: Camera3D = auto_free(Camera3D.new())
	add_child(target)
	add_child(cam)
	_hud.update_lock_on(target, cam)
	var reticle: Control = _hud.get_node("LockOnReticle")
	assert_that(reticle.get_meta("lock_target")).is_same(target)
	assert_that(reticle.get_meta("lock_camera")).is_same(cam)


# --- Duration constants ---


func test_damage_flash_duration_constant() -> void:
	assert_float(_hud.DAMAGE_FLASH_DURATION).is_equal(0.3)


func test_hit_marker_duration_constant() -> void:
	assert_float(_hud.HIT_MARKER_DURATION).is_equal(0.15)


# --- Config color constants ---


func test_config_colors_has_five_entries() -> void:
	assert_int(_hud.CONFIG_COLORS.size()).is_equal(5)


func test_ability_keybinds_has_four_entries() -> void:
	assert_int(_hud.ABILITY_KEYBINDS.size()).is_equal(4)


# --- Flow mastery ---


func test_update_flow_stores_tier_and_stacks() -> void:
	_hud.update_flow(1, 5)
	assert_int(_hud._flow_tier).is_equal(1)
	assert_int(_hud._flow_stacks).is_equal(5)


func test_update_flow_defaults_to_zero() -> void:
	assert_int(_hud._flow_tier).is_equal(0)
	assert_int(_hud._flow_stacks).is_equal(0)


func test_update_flow_overwrites_previous() -> void:
	_hud.update_flow(2, 10)
	_hud.update_flow(1, 3)
	assert_int(_hud._flow_tier).is_equal(1)
	assert_int(_hud._flow_stacks).is_equal(3)


func test_flow_color_constants_exist() -> void:
	assert_float(_hud.FLOW_DIM.a).is_equal_approx(0.4, 0.01)
	assert_float(_hud.FLOW_EMPOWERED.a).is_equal_approx(0.95, 0.01)
	assert_float(_hud.FLOW_MAXIMUM.a).is_equal_approx(1.0, 0.01)


func test_controller_flow_state_defaults() -> void:
	assert_int(_bd._flow_tier).is_equal(0)
	assert_int(_bd._flow_stacks).is_equal(0)
