class_name TestVersionCheck
extends GdUnitTestSuite

## Pre-login version gate: the client must run the exact release tag the
## server was built from. Unstamped builds (editor, PR exports) carry the
## "dev" placeholder and skip the gate entirely; the server rejects them at
## the WS handshake instead when it is a stamped production build.

const VersionCheckScript := preload("res://scripts/autoload/version_check.gd")

const VERSION_SETTING := "application/config/version"
const BASE_SETTING := "pyrrhic/update/download_base"

var _checker: Node
var _saved_version: Variant
var _saved_base: Variant


func before_test() -> void:
	_checker = auto_free(VersionCheckScript.new())
	_saved_version = ProjectSettings.get_setting(VERSION_SETTING, "dev")
	_saved_base = ProjectSettings.get_setting(BASE_SETTING, "")


func after_test() -> void:
	ProjectSettings.set_setting(VERSION_SETTING, _saved_version)
	ProjectSettings.set_setting(BASE_SETTING, _saved_base)


func test_dev_placeholder_means_unstamped() -> void:
	ProjectSettings.set_setting(VERSION_SETTING, "dev")
	assert_str(_checker.client_version()).is_empty()


func test_stamped_version_is_reported() -> void:
	ProjectSettings.set_setting(VERSION_SETTING, "v0.4.2")
	assert_str(_checker.client_version()).is_equal("v0.4.2")


func test_update_required_only_on_stamped_mismatch() -> void:
	# Unstamped client: never gate, whatever the server runs.
	assert_bool(_checker.is_update_required("", "v0.4.2")).is_false()
	# Unstamped server (local dev gateway): nothing to compare against.
	assert_bool(_checker.is_update_required("v0.4.1", "")).is_false()
	assert_bool(_checker.is_update_required("v0.4.2", "v0.4.2")).is_false()
	assert_bool(_checker.is_update_required("v0.4.1", "v0.4.2")).is_true()


func test_download_url_per_platform() -> void:
	ProjectSettings.set_setting(BASE_SETTING, "https://example.com/releases/download")
	assert_str(_checker.download_url("v0.4.2", "Linux")).is_equal(
		"https://example.com/releases/download/v0.4.2/pyrrhic-stars-linux.x86_64"
	)
	assert_str(_checker.download_url("v0.4.2", "Windows")).is_equal(
		"https://example.com/releases/download/v0.4.2/pyrrhic-stars-windows.exe"
	)
	# Platforms without a released binary get no URL (no self-update offered).
	assert_str(_checker.download_url("v0.4.2", "Web")).is_empty()


func test_download_url_requires_stamped_base() -> void:
	ProjectSettings.set_setting(BASE_SETTING, "")
	assert_str(_checker.download_url("v0.4.2", "Linux")).is_empty()


func test_ws_version_param_appended_only_when_stamped() -> void:
	ProjectSettings.set_setting(VERSION_SETTING, "dev")
	assert_str(_checker.ws_version_param()).is_empty()
	ProjectSettings.set_setting(VERSION_SETTING, "v0.4.2")
	assert_str(_checker.ws_version_param()).is_equal("&v=v0.4.2")


func test_parse_response_extracts_version() -> void:
	var body := '{"version":"v0.4.2"}'.to_utf8_buffer()
	var parsed: Dictionary = _checker.parse_response(200, body)
	assert_bool(parsed.ok).is_true()
	assert_str(parsed.version).is_equal("v0.4.2")


func test_parse_response_rejects_errors_and_garbage() -> void:
	assert_bool(_checker.parse_response(500, "{}".to_utf8_buffer()).ok).is_false()
	assert_bool(_checker.parse_response(200, "not json".to_utf8_buffer()).ok).is_false()
	assert_bool(_checker.parse_response(200, "[1,2]".to_utf8_buffer()).ok).is_false()
