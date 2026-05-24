class_name TestGunnerHUD
extends GdUnitTestSuite

## Tests for the Gunner HUD — hit feedback, damage flash, recoil, ability bar.

const GunnerHudScript := preload("res://scenes/shared/hud/gunner_hud.gd")
const GUNNER_SCENE := "res://scenes/controllers/gunner/gunner.tscn"
const DELTA := 1.0 / 60.0

var _gunner: CharacterBody3D
var _hud: GunnerHudScript


func before_test() -> void:
	_gunner = auto_free(load(GUNNER_SCENE).instantiate()) as CharacterBody3D
	add_child(_gunner)
	await get_tree().process_frame
	_hud = _gunner.hud as GunnerHudScript


# --- Ability bar ---


func test_update_abilities_passes_to_ability_bar() -> void:
	var abilities := [
		{name = "Shoot", keybind = "LMB", desc = "10 dmg.", cooldown = 0.0, cooldown_max = 0.0},
		{name = "Roll", keybind = "C", desc = "Dodge.", cooldown = 1.0, cooldown_max = 2.5},
	]
	_hud.update_abilities(abilities)
	var bar: Control = _hud.get_node("AbilityBar")
	assert_int(bar._abilities.size()).is_equal(2)
	assert_str(bar._abilities[0].name).is_equal("Shoot")
	assert_str(bar._abilities[1].name).is_equal("Roll")


func test_ability_bar_accent_color_is_blue() -> void:
	var bar: Control = _hud.get_node("AbilityBar")
	assert_float(bar.accent_color.r).is_equal_approx(0.24, 0.01)
	assert_float(bar.accent_color.g).is_equal_approx(0.62, 0.01)
	assert_float(bar.accent_color.b).is_equal_approx(0.95, 0.01)


# --- Hit marker ---


func test_hit_marker_sets_timer() -> void:
	_hud.show_hit_marker()
	assert_float(_hud._hit_marker_timer).is_greater(0.0)


func test_hit_marker_decays() -> void:
	_hud.show_hit_marker()
	var initial := _hud._hit_marker_timer
	_hud._process(DELTA)
	assert_float(_hud._hit_marker_timer).is_less(initial)


# --- Damage flash ---


func test_damage_flash_sets_timer() -> void:
	_hud.show_damage_flash()
	assert_float(_hud._damage_flash_timer).is_greater(0.0)


func test_damage_flash_modulates_overlay() -> void:
	_hud.show_damage_flash()
	_hud._process(DELTA)
	var overlay: ColorRect = _hud.get_node("DamageOverlay")
	assert_float(overlay.modulate.a).is_greater(0.0)


# --- Shoot recoil ---


func test_recoil_sets_timer() -> void:
	_hud.on_shoot()
	assert_float(_hud._recoil_timer).is_greater(0.0)
