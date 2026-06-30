class_name TestGraphicsQuality
extends GdUnitTestSuite

## Graphics quality must actually reach the expensive screen-space effects. The
## quality slider used to only change MSAA and render scale, leaving volumetric
## fog / SSR / SSAO baked-on at every tier - the exact effects that make the game
## crawl on integrated GPUs. apply_quality_to_environment() is the bridge: given
## a tier, it strips or enables those effects on a scene's Environment.

var _saved_quality: int


func before_test() -> void:
	_saved_quality = int(SettingsManager.get_value("graphics", "quality", 2))


func after_test() -> void:
	SettingsManager._settings["graphics"]["quality"] = _saved_quality


func _env_at(tier: int) -> Environment:
	SettingsManager._settings["graphics"]["quality"] = tier
	var env: Environment = auto_free(Environment.new())
	# Author everything ON, as the scene resources do, then let quality strip it.
	env.volumetric_fog_enabled = true
	env.ssr_enabled = true
	env.ssao_enabled = true
	env.ssil_enabled = true
	env.glow_enabled = true
	SettingsManager.apply_quality_to_environment(env)
	return env


func test_low_strips_the_expensive_effects() -> void:
	var env := _env_at(0)
	assert_bool(env.volumetric_fog_enabled).is_false()
	assert_bool(env.ssr_enabled).is_false()
	assert_bool(env.ssao_enabled).is_false()
	assert_bool(env.ssil_enabled).is_false()
	assert_bool(env.glow_enabled).is_false()


func test_high_keeps_fog_and_ssr() -> void:
	var env := _env_at(2)
	assert_bool(env.volumetric_fog_enabled).is_true()
	assert_bool(env.ssr_enabled).is_true()
	assert_bool(env.ssao_enabled).is_true()


func test_ultra_enables_ssil() -> void:
	assert_bool(_env_at(3).ssil_enabled).is_true()
	# SSIL is the one effect reserved for Ultra alone.
	assert_bool(_env_at(2).ssil_enabled).is_false()


func test_null_environment_is_safe() -> void:
	SettingsManager.apply_quality_to_environment(null)


func test_quality_tier_clamps() -> void:
	SettingsManager._settings["graphics"]["quality"] = 99
	assert_int(SettingsManager.quality_tier()).is_equal(3)
	SettingsManager._settings["graphics"]["quality"] = -5
	assert_int(SettingsManager.quality_tier()).is_equal(0)
