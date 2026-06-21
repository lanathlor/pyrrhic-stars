class_name TestHowToPlayPanel
extends GdUnitTestSuite

## Tests for the first-session How-to-Play guide: open/close visibility, the
## first-time gate, and that its "seen" flag round-trips through SettingsManager's
## free-form "ui" section (a regression: graphics/audio only merge known keys).

const PanelScript := preload("res://scenes/ui/how_to_play_panel.gd")

var _panel: PanelScript


func before_test() -> void:
	var node: CanvasLayer = auto_free(CanvasLayer.new())
	node.set_script(PanelScript)
	_panel = node as PanelScript
	add_child(_panel)
	await get_tree().process_frame


func test_starts_hidden() -> void:
	assert_bool(_panel.visible).is_false()


func test_open_shows_panel() -> void:
	_panel.open()
	assert_bool(_panel.visible).is_true()


func test_close_hides_and_emits_signal() -> void:
	_panel.open()
	var emitted := [false]
	_panel.closed.connect(func(): emitted[0] = true)
	_panel.close()
	assert_bool(_panel.visible).is_false()
	assert_bool(emitted[0]).is_true()


func test_open_marks_seen() -> void:
	SettingsManager.set_value(PanelScript.SEEN_SECTION, PanelScript.SEEN_KEY, false)
	_panel.open()
	var seen := bool(
		SettingsManager.get_value(PanelScript.SEEN_SECTION, PanelScript.SEEN_KEY, false)
	)
	assert_bool(seen).is_true()


func test_first_time_gate_skips_when_already_seen() -> void:
	SettingsManager.set_value(PanelScript.SEEN_SECTION, PanelScript.SEEN_KEY, true)
	_panel.open_if_first_time()
	assert_bool(_panel.visible).is_false()


func test_first_time_gate_shows_when_unseen() -> void:
	SettingsManager.set_value(PanelScript.SEEN_SECTION, PanelScript.SEEN_KEY, false)
	_panel.open_if_first_time()
	assert_bool(_panel.visible).is_true()


func test_ui_section_survives_document_merge() -> void:
	# Regression: the free-form "ui" section must carry arbitrary keys through
	# _merge_document, otherwise the "seen" flag is dropped on reload/server sync.
	SettingsManager.set_value(PanelScript.SEEN_SECTION, PanelScript.SEEN_KEY, false)
	SettingsManager._merge_document({"ui": {PanelScript.SEEN_KEY: true}})
	var seen := bool(
		SettingsManager.get_value(PanelScript.SEEN_SECTION, PanelScript.SEEN_KEY, false)
	)
	assert_bool(seen).is_true()
