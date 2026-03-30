class_name TestGunnerHUD
extends GdUnitTestSuite

## Tests for the Gunner HUD — health bar, roll cooldown, hit feedback state.

# The HUD is embedded in the gunner scene, so we test via the gunner.
const GUNNER_SCENE := "res://scenes/controllers/gunner/gunner.tscn"

var _gunner: CharacterBody3D
var _hud: Control


func before_test() -> void:
	_gunner = auto_free(load(GUNNER_SCENE).instantiate())
	add_child(_gunner)
	await get_tree().process_frame
	_hud = _gunner.hud


# --- Health bar ---

func test_health_bar_initial_value() -> void:
	var bar: ProgressBar = _hud.get_node("HealthBar")
	assert_float(bar.value).is_equal(100.0)
	assert_float(bar.max_value).is_equal(100.0)


func test_health_bar_updates_on_damage() -> void:
	_gunner.take_damage(40.0)
	var bar: ProgressBar = _hud.get_node("HealthBar")
	assert_float(bar.value).is_equal(60.0)


# --- Roll cooldown ---

func test_roll_cooldown_ratio_ready() -> void:
	_hud.update_roll_cooldown(0.0, 2.5)
	assert_float(_hud._roll_cooldown_ratio).is_equal(0.0)


func test_roll_cooldown_ratio_just_used() -> void:
	_hud.update_roll_cooldown(2.5, 2.5)
	assert_float(_hud._roll_cooldown_ratio).is_equal(1.0)


func test_roll_cooldown_ratio_half() -> void:
	_hud.update_roll_cooldown(1.25, 2.5)
	assert_float(_hud._roll_cooldown_ratio).is_equal(0.5)


# --- Hit marker ---

func test_hit_marker_sets_timer() -> void:
	_hud.show_hit_marker()
	assert_float(_hud._hit_marker_timer).is_greater(0.0)


func test_hit_marker_decays() -> void:
	_hud.show_hit_marker()
	var initial := _hud._hit_marker_timer
	_hud._process(1.0 / 60.0)
	assert_float(_hud._hit_marker_timer).is_less(initial)


# --- Damage flash ---

func test_damage_flash_sets_timer() -> void:
	_hud.show_damage_flash()
	assert_float(_hud._damage_flash_timer).is_greater(0.0)


func test_damage_flash_modulates_overlay() -> void:
	_hud.show_damage_flash()
	_hud._process(1.0 / 60.0)
	var overlay: ColorRect = _hud.get_node("DamageOverlay")
	assert_float(overlay.modulate.a).is_greater(0.0)


# --- Shoot recoil ---

func test_recoil_sets_timer() -> void:
	_hud.on_shoot()
	assert_float(_hud._recoil_timer).is_greater(0.0)
