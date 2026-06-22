extends Node

## Style helpers and dynamic UI population (character list rows).
## Static UI scenes are in scenes/ui/ and instanced in main.tscn.

const CLASS_INFO := {
	"gunner":
	{
		"name": "Gunner",
		"genre": "FPS",
		"desc": "Fast movement, high fire rate.\nRelentless aggression."
	},
	"vanguard":
	{
		"name": "Vanguard",
		"genre": "Souls-like",
		"desc": "Big AoE swings, punish windows.\nHeavy and deliberate."
	},
	"blade_dancer":
	{
		"name": "Blade Dancer",
		"genre": "State Machine",
		"desc": "5 configurations, 4 abilities each.\nHighest skill ceiling."
	},
}
const UI_SURFACE := Color(0.035, 0.045, 0.065, 0.88)
const UI_SURFACE_ALT := Color(0.05, 0.06, 0.085, 0.92)
const UI_SURFACE_ACTIVE := Color(0.08, 0.1, 0.15, 0.96)
const UI_BORDER := Color(0.28, 0.31, 0.37, 0.9)
const UI_BORDER_ACTIVE := Color(0.32, 0.58, 0.92, 0.95)
const UI_TEXT := Color(0.9, 0.93, 0.98, 0.96)
const UI_TEXT_MUTED := Color(0.6, 0.66, 0.75, 0.95)
const UI_TEXT_DIM := Color(0.48, 0.53, 0.6, 0.95)
const UI_DANGER := Color(0.86, 0.28, 0.28, 0.96)

var ctrl: Node


func _ready() -> void:
	ctrl = get_parent()


# =============================================================================
# Code-built overlay panels (created at startup, owned by main via ctrl fields)
# =============================================================================


func build_overlay_panels() -> void:
	# Overflux selection panel.
	var overflux_panel := preload("res://scenes/ui/overflux_panel.gd").new()
	ctrl.add_child(overflux_panel)
	overflux_panel.confirmed.connect(ctrl._on_overflux_confirmed)
	overflux_panel.cancelled.connect(ctrl._on_overflux_cancelled)
	ctrl._overflux_panel = overflux_panel

	# Merchant shop panel.
	var merchant_layer := CanvasLayer.new()
	merchant_layer.layer = 18
	ctrl.add_child(merchant_layer)
	var merchant_panel: Control = preload("res://scenes/ui/merchant_panel.gd").new()
	merchant_panel.set_anchors_preset(Control.PRESET_FULL_RECT)
	merchant_layer.add_child(merchant_panel)
	merchant_panel.closed.connect(ctrl._update_cursor_mode)
	merchant_panel.closed.connect(
		func():
			ctrl._inventory_layer.bag_panel.merchant_open = false
			ctrl._inventory_layer.bag_panel.queue_redraw()
	)
	ctrl._merchant_layer = merchant_layer
	ctrl._merchant_panel = merchant_panel

	# Settings overlay (opened from the pause and main menus).
	var settings_panel := preload("res://scenes/ui/settings_panel.gd").new()
	settings_panel.ui_ctrl = self
	ctrl.add_child(settings_panel)
	ctrl._settings_panel = settings_panel

	# Social overlay (group + friends), opened with [G].
	var social_panel := preload("res://scenes/ui/social_panel.gd").new()
	ctrl.add_child(social_panel)
	social_panel.closed.connect(ctrl._update_cursor_mode)
	social_panel.closed.connect(ctrl._sync_toolbar_active)
	ctrl._social_panel = social_panel

	# How-to-play guide: auto-shown once on first hub entry, re-openable via [H]
	# or the pause menu.
	var how_to_play := preload("res://scenes/ui/how_to_play_panel.gd").new()
	how_to_play.ui_ctrl = self
	ctrl.add_child(how_to_play)
	how_to_play.closed.connect(ctrl._update_cursor_mode)
	how_to_play.closed.connect(ctrl._sync_toolbar_active)
	ctrl._how_to_play_panel = how_to_play


## Close every open gameplay overlay (social, inventory, bag, spec, merchant,
## overflux). Returns true if at least one was closed, so the caller can swallow
## the Esc keypress before it reaches the pause menu.
func close_open_overlay() -> bool:
	var closed_any := false
	if ctrl._social_panel and ctrl._social_panel.visible:
		ctrl._social_panel.close()
		closed_any = true
	if ctrl._inventory_layer.equip_panel.visible:
		ctrl._inventory_layer.equip_panel.toggle()
		closed_any = true
	if ctrl._inventory_layer.bag_panel.visible:
		ctrl._inventory_layer.bag_panel.toggle()
		closed_any = true
	if ctrl._spec_panel.visible:
		ctrl._spec_panel.toggle()
		closed_any = true
	if ctrl._merchant_panel.visible:
		ctrl._merchant_panel.close_shop()
		closed_any = true
	if ctrl._overflux_panel.visible:
		ctrl._overflux_panel.close()
		closed_any = true
	if ctrl._how_to_play_panel and ctrl._how_to_play_panel.visible:
		ctrl._how_to_play_panel.close()
		closed_any = true
	if ctrl._map_overlay and ctrl._map_overlay.visible:
		ctrl._map_overlay.toggle()
		closed_any = true
	if closed_any:
		ctrl._update_cursor_mode()
		ctrl._sync_toolbar_active()
	return closed_any


## Sets the OS mouse mode from the current game/overlay state, without pausing.
## Cursor is visible while paused, for always-visible classes, or whenever any
## overlay is open (inventory, spec, bot, overflux, merchant, social, help);
## captured otherwise. Also visible when Alt is held or the backtick toggle is on.
func update_cursor_mode() -> void:
	if ctrl.paused:
		Input.set_mouse_mode(Input.MOUSE_MODE_VISIBLE)
		return
	if ctrl._is_cursor_always_visible_class():
		Input.set_mouse_mode(Input.MOUSE_MODE_VISIBLE)
		return
	var inv: CanvasLayer = ctrl._inventory_layer
	var inv_open: bool = inv.equip_panel.visible or inv.bag_panel.visible
	var bot_open: bool = ctrl.dev_mgr.bot_panel != null and ctrl.dev_mgr.bot_panel.visible
	var want_cursor: bool = (
		ctrl._cursor_toggled
		or ctrl._alt_held
		or inv_open
		or ctrl._spec_panel.visible
		or bot_open
		or (ctrl._overflux_panel != null and ctrl._overflux_panel.visible)
		or (ctrl._merchant_panel != null and ctrl._merchant_panel.visible)
		or (ctrl._social_panel != null and ctrl._social_panel.visible)
		or (ctrl._how_to_play_panel != null and ctrl._how_to_play_panel.visible)
	)
	Input.set_mouse_mode(Input.MOUSE_MODE_VISIBLE if want_cursor else Input.MOUSE_MODE_CAPTURED)


func toggle_map_overlay() -> void:
	var map_overlay: Control = ctrl._map_overlay
	var entity_mgr: Node = ctrl.entity_mgr
	var env_builder: Node = ctrl.env_builder
	var my_id: int = NetworkManager.get_my_id()
	if my_id in entity_mgr.spawned_players and is_instance_valid(entity_mgr.spawned_players[my_id]):
		var player: CharacterBody3D = entity_mgr.spawned_players[my_id]
		map_overlay._player_pos = player.global_position
		map_overlay._player_rot_y = player.rotation.y
	map_overlay.toggle()
	if map_overlay.visible:
		map_overlay.scan_environment(env_builder.current_env)
		map_overlay._recompute_scale()
		if env_builder.portal_trail:
			if (
				my_id in entity_mgr.spawned_players
				and is_instance_valid(entity_mgr.spawned_players[my_id])
			):
				map_overlay.set_waypoint_path(
					env_builder.portal_trail.get_path_to_target(
						entity_mgr.spawned_players[my_id].global_position
					)
				)


# =============================================================================
# Style helpers (public — used by UI scene scripts and other sub-systems)
# =============================================================================


func style_box(
	fill: Color, border: Color, border_width: int = 1, padding: int = 10
) -> StyleBoxFlat:
	var style := StyleBoxFlat.new()
	style.bg_color = fill
	style.border_color = border
	style.set_border_width_all(border_width)
	style.set_corner_radius_all(0)
	style.set_content_margin_all(padding)
	return style


func apply_button_style(btn: Button, accent: Color = UI_BORDER_ACTIVE) -> void:
	btn.add_theme_stylebox_override("normal", style_box(UI_SURFACE, UI_BORDER, 1, 10))
	btn.add_theme_stylebox_override("hover", style_box(UI_SURFACE_ALT, accent, 1, 10))
	btn.add_theme_stylebox_override("pressed", style_box(UI_SURFACE_ACTIVE, accent, 1, 10))
	btn.add_theme_stylebox_override("focus", style_box(UI_SURFACE_ACTIVE, accent, 1, 10))
	btn.add_theme_stylebox_override(
		"disabled", style_box(Color(UI_SURFACE, 0.45), Color(UI_BORDER, 0.4), 1, 10)
	)
	btn.add_theme_color_override("font_color", UI_TEXT)
	btn.add_theme_color_override("font_hover_color", UI_TEXT)
	btn.add_theme_color_override("font_pressed_color", UI_TEXT)
	btn.add_theme_color_override("font_focus_color", UI_TEXT)
	btn.add_theme_color_override("font_disabled_color", UI_TEXT_DIM)
	btn.add_theme_constant_override("h_separation", 8)
	# Every styled button routes its click through the SFX layer (single chokepoint).
	if not btn.pressed.is_connected(_on_styled_button_pressed):
		btn.pressed.connect(_on_styled_button_pressed)


func _on_styled_button_pressed() -> void:
	AudioManager.play_ui(&"ui_click")


func apply_line_edit_style(input: LineEdit) -> void:
	input.add_theme_stylebox_override("normal", style_box(UI_SURFACE, UI_BORDER, 1, 10))
	input.add_theme_stylebox_override("focus", style_box(UI_SURFACE_ALT, UI_BORDER_ACTIVE, 1, 10))
	input.add_theme_stylebox_override("read_only", style_box(UI_SURFACE, UI_BORDER, 1, 10))
	input.add_theme_color_override("font_color", UI_TEXT)
	input.add_theme_color_override("font_placeholder_color", UI_TEXT_DIM)
	input.add_theme_color_override("caret_color", UI_TEXT)
	input.add_theme_color_override("selection_color", Color(UI_BORDER_ACTIVE, 0.35))


func apply_panel_style(panel: PanelContainer, active: bool = false, padding: int = 10) -> void:
	var fill: Color = UI_SURFACE_ACTIVE if active else UI_SURFACE
	var border: Color = UI_BORDER_ACTIVE if active else UI_BORDER
	panel.add_theme_stylebox_override("panel", style_box(fill, border, 1, padding))


func apply_overlay_label(label: Label, font_size: int, color: Color = UI_TEXT) -> void:
	label.add_theme_font_size_override("font_size", font_size)
	label.add_theme_color_override("font_color", color)


# =============================================================================
# Character select logic (dynamic population from server data)
# =============================================================================


func populate_char_select() -> void:
	if ctrl._account_username != "":
		ctrl._char_select_welcome.text = "Welcome, %s" % ctrl._account_username
	else:
		ctrl._char_select_welcome.text = ""

	for child in ctrl._char_list_container.get_children():
		child.queue_free()

	var characters: Array = ctrl._char_list_data.get("characters", [])
	var normal_style := _make_char_row_style(UI_SURFACE, UI_BORDER)
	var selected_style := _make_char_row_style(UI_SURFACE_ACTIVE, UI_BORDER_ACTIVE)

	if characters.is_empty():
		var empty_label := Label.new()
		empty_label.text = "No characters yet. Create one to get started!"
		empty_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
		apply_overlay_label(empty_label, 14, UI_TEXT_DIM)
		ctrl._char_list_container.add_child(empty_label)
		ctrl._enter_world_btn.disabled = true
		return

	ctrl._enter_world_btn.disabled = false
	for ch in characters:
		_build_char_row(ch, normal_style, selected_style)

	if ctrl._selected_char_id == 0 and not characters.is_empty():
		ctrl._select_character_row(characters[0].char_id, characters[0].class_name)


func _make_char_row_style(bg: Color, border: Color) -> StyleBoxFlat:
	var s := StyleBoxFlat.new()
	s.bg_color = bg
	s.border_color = border
	s.set_border_width_all(1)
	s.set_corner_radius_all(0)
	s.set_content_margin_all(10)
	return s


func _build_char_row(
	ch: Dictionary, normal_style: StyleBoxFlat, selected_style: StyleBoxFlat
) -> void:
	var char_id: int = ch.char_id
	var class_display: String = CLASS_INFO.get(ch.class_name, {}).get("name", ch.class_name)

	var row := PanelContainer.new()
	row.custom_minimum_size.y = 44.0
	row.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	row.set_meta("char_id", char_id)
	row.set_meta("normal_style", normal_style)
	row.set_meta("selected_style", selected_style)
	if char_id == ctrl._selected_char_id:
		row.add_theme_stylebox_override("panel", selected_style)
	else:
		row.add_theme_stylebox_override("panel", normal_style)
	ctrl._char_list_container.add_child(row)

	var hbox := HBoxContainer.new()
	hbox.add_theme_constant_override("separation", 14)
	row.add_child(hbox)

	var name_lbl := Label.new()
	name_lbl.text = ch.char_name
	apply_overlay_label(name_lbl, 16, UI_TEXT)
	name_lbl.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	hbox.add_child(name_lbl)

	var class_lbl := Label.new()
	class_lbl.text = class_display
	apply_overlay_label(class_lbl, 14, UI_BORDER_ACTIVE)
	hbox.add_child(class_lbl)

	var btn := Button.new()
	btn.flat = true
	btn.anchor_right = 1.0
	btn.anchor_bottom = 1.0
	btn.mouse_filter = Control.MOUSE_FILTER_STOP
	var id_capture: int = char_id
	var cls_capture: String = ch.class_name
	btn.pressed.connect(func(): ctrl._select_character_row(id_capture, cls_capture))
	row.add_child(btn)
