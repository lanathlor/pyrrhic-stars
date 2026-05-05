extends Node

## All UI creation: menu, character select/create, hub HUD, pause, group panel,
## invite popup, death overlay, shared HUD. Also style helpers.

const CLASS_SCENES := {
	"gunner": "res://scenes/controllers/gunner/gunner.tscn",
	"vanguard": "res://scenes/controllers/vanguard/vanguard.tscn",
	"blade_dancer": "res://scenes/controllers/blade_dancer/blade_dancer.tscn",
}
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
		"desc": "5 configurations, 4 spells each.\nHighest skill ceiling."
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
# Style helpers (public — used by other sub-systems)
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
# Pause menu
# =============================================================================


func create_pause_menu() -> void:
	ctrl._pause_layer = CanvasLayer.new()
	ctrl._pause_layer.layer = 20
	ctrl._pause_layer.process_mode = Node.PROCESS_MODE_ALWAYS
	ctrl._pause_layer.visible = false
	ctrl.add_child(ctrl._pause_layer)

	var bg := ColorRect.new()
	bg.color = Color(0.0, 0.0, 0.0, 0.72)
	bg.anchor_right = 1.0
	bg.anchor_bottom = 1.0
	bg.mouse_filter = Control.MOUSE_FILTER_STOP
	ctrl._pause_layer.add_child(bg)

	var panel := PanelContainer.new()
	panel.anchor_left = 0.5
	panel.anchor_right = 0.5
	panel.anchor_top = 0.35
	panel.anchor_bottom = 0.65
	panel.offset_left = -140.0
	panel.offset_right = 140.0
	apply_panel_style(panel, false, 12)
	ctrl._pause_layer.add_child(panel)

	var vbox := VBoxContainer.new()
	vbox.add_theme_constant_override("separation", 10)
	panel.add_child(vbox)

	var title := Label.new()
	title.text = "Paused"
	title.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	apply_overlay_label(title, 22, UI_TEXT)
	vbox.add_child(title)

	var resume_btn := Button.new()
	resume_btn.text = "Resume"
	resume_btn.custom_minimum_size.y = 38.0
	apply_button_style(resume_btn)
	resume_btn.pressed.connect(ctrl._toggle_pause)
	vbox.add_child(resume_btn)

	var menu_btn := Button.new()
	menu_btn.text = "Back to Menu"
	menu_btn.custom_minimum_size.y = 38.0
	apply_button_style(menu_btn)
	menu_btn.pressed.connect(
		func():
			ctrl.get_tree().paused = false
			ctrl.paused = false
			ctrl.entity_mgr.despawn_all_players()
			ctrl._enter_menu()
	)
	vbox.add_child(menu_btn)

	var quit_btn := Button.new()
	quit_btn.text = "Quit"
	quit_btn.custom_minimum_size.y = 38.0
	apply_button_style(quit_btn, UI_DANGER)
	quit_btn.pressed.connect(func(): ctrl.get_tree().quit())
	vbox.add_child(quit_btn)


# =============================================================================
# Menu UI
# =============================================================================


func create_menu_ui() -> void:
	ctrl._menu_layer = CanvasLayer.new()
	ctrl._menu_layer.layer = 18
	ctrl._menu_layer.visible = false
	ctrl.add_child(ctrl._menu_layer)

	var bg := ColorRect.new()
	bg.color = Color(0.02, 0.025, 0.04, 0.96)
	bg.anchor_right = 1.0
	bg.anchor_bottom = 1.0
	bg.mouse_filter = Control.MOUSE_FILTER_STOP
	ctrl._menu_layer.add_child(bg)

	var panel := PanelContainer.new()
	panel.anchor_left = 0.5
	panel.anchor_right = 0.5
	panel.anchor_top = 0.25
	panel.anchor_bottom = 0.75
	panel.offset_left = -170.0
	panel.offset_right = 170.0
	apply_panel_style(panel, false, 14)
	ctrl._menu_layer.add_child(panel)

	var vbox := VBoxContainer.new()
	vbox.add_theme_constant_override("separation", 12)
	panel.add_child(vbox)

	var title := Label.new()
	title.text = "CODEX ONLINE"
	title.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	apply_overlay_label(title, 34, UI_TEXT)
	vbox.add_child(title)

	var subtitle := Label.new()
	subtitle.text = "Phase 0 -- Server Authoritative"
	subtitle.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	apply_overlay_label(subtitle, 14, UI_TEXT_DIM)
	vbox.add_child(subtitle)

	var spacer := Control.new()
	spacer.custom_minimum_size.y = 10.0
	vbox.add_child(spacer)

	# Welcome label — shown for returning players.
	ctrl._menu_welcome_label = Label.new()
	ctrl._menu_welcome_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	apply_overlay_label(ctrl._menu_welcome_label, 18, UI_TEXT_MUTED)
	ctrl._menu_welcome_label.visible = false
	vbox.add_child(ctrl._menu_welcome_label)

	# Username input — only shown for new players (no saved username).
	ctrl._username_input = LineEdit.new()
	ctrl._username_input.placeholder_text = "Choose a username..."
	ctrl._username_input.custom_minimum_size.y = 42.0
	ctrl._username_input.max_length = 20
	ctrl._username_input.alignment = HORIZONTAL_ALIGNMENT_CENTER
	apply_line_edit_style(ctrl._username_input)
	vbox.add_child(ctrl._username_input)

	# Load saved username — if exists, show welcome instead of input.
	var saved: String = ctrl._load_saved_username()
	if saved != "":
		ctrl._username = saved
		ctrl._username_input.visible = false
		ctrl._menu_welcome_label.text = "Welcome back, %s" % saved
		ctrl._menu_welcome_label.visible = true

	var play_btn := Button.new()
	play_btn.text = "Play"
	play_btn.custom_minimum_size = Vector2(200.0, 42.0)
	apply_button_style(play_btn)
	play_btn.pressed.connect(ctrl._on_connect_pressed)
	vbox.add_child(play_btn)

	var replay_btn := Button.new()
	replay_btn.text = "Replays"
	replay_btn.custom_minimum_size = Vector2(200.0, 42.0)
	apply_button_style(replay_btn)
	replay_btn.pressed.connect(ctrl._enter_replay_browser)
	vbox.add_child(replay_btn)


# =============================================================================
# Character Select UI
# =============================================================================


func create_char_select_ui() -> void:
	ctrl._char_select_layer = CanvasLayer.new()
	ctrl._char_select_layer.layer = 18
	ctrl._char_select_layer.visible = false
	ctrl.add_child(ctrl._char_select_layer)

	var bg := ColorRect.new()
	bg.color = Color(0.02, 0.025, 0.04, 0.96)
	bg.anchor_right = 1.0
	bg.anchor_bottom = 1.0
	bg.mouse_filter = Control.MOUSE_FILTER_STOP
	ctrl._char_select_layer.add_child(bg)

	var panel := PanelContainer.new()
	panel.anchor_left = 0.5
	panel.anchor_right = 0.5
	panel.anchor_top = 0.1
	panel.anchor_bottom = 0.9
	panel.offset_left = -320.0
	panel.offset_right = 320.0
	apply_panel_style(panel, false, 14)
	ctrl._char_select_layer.add_child(panel)

	var outer := VBoxContainer.new()
	outer.add_theme_constant_override("separation", 12)
	panel.add_child(outer)

	# Title
	var title := Label.new()
	title.text = "Select Character"
	title.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	apply_overlay_label(title, 26, UI_TEXT)
	outer.add_child(title)

	# Welcome label
	ctrl._char_select_welcome = Label.new()
	ctrl._char_select_welcome.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	apply_overlay_label(ctrl._char_select_welcome, 14, UI_TEXT_DIM)
	outer.add_child(ctrl._char_select_welcome)

	# Scrollable character list
	var scroll := ScrollContainer.new()
	scroll.size_flags_vertical = Control.SIZE_EXPAND_FILL
	scroll.custom_minimum_size.y = 220.0
	outer.add_child(scroll)

	ctrl._char_list_container = VBoxContainer.new()
	ctrl._char_list_container.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	ctrl._char_list_container.add_theme_constant_override("separation", 4)
	scroll.add_child(ctrl._char_list_container)

	# Buttons
	var btn_hbox := HBoxContainer.new()
	btn_hbox.alignment = BoxContainer.ALIGNMENT_CENTER
	btn_hbox.add_theme_constant_override("separation", 20)
	outer.add_child(btn_hbox)

	var back_btn := Button.new()
	back_btn.text = "Back"
	back_btn.custom_minimum_size = Vector2(100.0, 38.0)
	apply_button_style(back_btn)
	back_btn.pressed.connect(
		func():
			NetworkManager.disconnect_game()
			ctrl._enter_menu()
	)
	btn_hbox.add_child(back_btn)

	var create_btn := Button.new()
	create_btn.text = "Create New Character"
	create_btn.custom_minimum_size = Vector2(200.0, 38.0)
	apply_button_style(create_btn)
	create_btn.pressed.connect(ctrl._enter_create_character)
	btn_hbox.add_child(create_btn)

	ctrl._enter_world_btn = Button.new()
	ctrl._enter_world_btn.text = "Enter World"
	ctrl._enter_world_btn.custom_minimum_size = Vector2(160.0, 38.0)
	apply_button_style(ctrl._enter_world_btn)
	ctrl._enter_world_btn.pressed.connect(ctrl._on_enter_world_pressed)
	btn_hbox.add_child(ctrl._enter_world_btn)


# =============================================================================
# Character Create UI
# =============================================================================


func create_char_create_ui() -> void:
	ctrl._char_create_layer = CanvasLayer.new()
	ctrl._char_create_layer.layer = 18
	ctrl._char_create_layer.visible = false
	ctrl.add_child(ctrl._char_create_layer)

	var bg := ColorRect.new()
	bg.color = Color(0.02, 0.025, 0.04, 0.96)
	bg.anchor_right = 1.0
	bg.anchor_bottom = 1.0
	bg.mouse_filter = Control.MOUSE_FILTER_STOP
	ctrl._char_create_layer.add_child(bg)

	var panel := PanelContainer.new()
	panel.anchor_left = 0.5
	panel.anchor_right = 0.5
	panel.anchor_top = 0.08
	panel.anchor_bottom = 0.92
	panel.offset_left = -360.0
	panel.offset_right = 360.0
	apply_panel_style(panel, false, 14)
	ctrl._char_create_layer.add_child(panel)

	var outer := VBoxContainer.new()
	outer.add_theme_constant_override("separation", 14)
	panel.add_child(outer)

	var title := Label.new()
	title.text = "Create Character"
	title.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	apply_overlay_label(title, 26, UI_TEXT)
	outer.add_child(title)

	# Class cards
	var cards_hbox := HBoxContainer.new()
	cards_hbox.add_theme_constant_override("separation", 12)
	cards_hbox.alignment = BoxContainer.ALIGNMENT_CENTER
	cards_hbox.size_flags_vertical = Control.SIZE_EXPAND_FILL
	outer.add_child(cards_hbox)

	var normal_style := StyleBoxFlat.new()
	normal_style.bg_color = UI_SURFACE
	normal_style.border_color = UI_BORDER
	normal_style.set_border_width_all(1)
	normal_style.set_corner_radius_all(0)
	normal_style.set_content_margin_all(12)

	var selected_style := StyleBoxFlat.new()
	selected_style.bg_color = UI_SURFACE_ACTIVE
	selected_style.border_color = UI_BORDER_ACTIVE
	selected_style.set_border_width_all(1)
	selected_style.set_corner_radius_all(0)
	selected_style.set_content_margin_all(12)

	for cls in CLASS_INFO:
		var info: Dictionary = CLASS_INFO[cls]
		var card := PanelContainer.new()
		card.custom_minimum_size = Vector2(190.0, 220.0)
		card.size_flags_horizontal = Control.SIZE_EXPAND_FILL
		card.add_theme_stylebox_override("panel", normal_style)
		card.set_meta("normal_style", normal_style)
		card.set_meta("selected_style", selected_style)
		cards_hbox.add_child(card)
		ctrl._char_create_cards[cls] = card

		var vbox := VBoxContainer.new()
		vbox.add_theme_constant_override("separation", 8)
		card.add_child(vbox)

		var name_label := Label.new()
		name_label.text = info.name
		name_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
		apply_overlay_label(name_label, 21, UI_TEXT)
		vbox.add_child(name_label)

		var genre_label := Label.new()
		genre_label.text = info.genre
		genre_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
		apply_overlay_label(genre_label, 13, UI_BORDER_ACTIVE)
		vbox.add_child(genre_label)

		var sep := HSeparator.new()
		sep.add_theme_color_override("separator", Color(UI_BORDER, 0.75))
		vbox.add_child(sep)

		var desc_label := Label.new()
		desc_label.text = info.desc
		desc_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
		apply_overlay_label(desc_label, 13, UI_TEXT_DIM)
		desc_label.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
		vbox.add_child(desc_label)

		# Click detection
		var click_btn := Button.new()
		click_btn.flat = true
		click_btn.anchor_right = 1.0
		click_btn.anchor_bottom = 1.0
		click_btn.mouse_filter = Control.MOUSE_FILTER_STOP
		var cls_capture: String = cls
		click_btn.pressed.connect(func(): ctrl._select_create_class(cls_capture))
		card.add_child(click_btn)

	# Name input
	var name_label := Label.new()
	name_label.text = "Character Name"
	name_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	apply_overlay_label(name_label, 16, UI_TEXT_MUTED)
	outer.add_child(name_label)

	ctrl._char_name_input = LineEdit.new()
	ctrl._char_name_input.placeholder_text = "Enter a name (2-20 characters)..."
	ctrl._char_name_input.custom_minimum_size.y = 40.0
	ctrl._char_name_input.max_length = 20
	ctrl._char_name_input.alignment = HORIZONTAL_ALIGNMENT_CENTER
	apply_line_edit_style(ctrl._char_name_input)
	outer.add_child(ctrl._char_name_input)

	# Error label
	ctrl._char_create_error_label = Label.new()
	ctrl._char_create_error_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	apply_overlay_label(ctrl._char_create_error_label, 13, UI_DANGER)
	ctrl._char_create_error_label.visible = false
	outer.add_child(ctrl._char_create_error_label)

	# Buttons
	var btn_hbox := HBoxContainer.new()
	btn_hbox.alignment = BoxContainer.ALIGNMENT_CENTER
	btn_hbox.add_theme_constant_override("separation", 20)
	outer.add_child(btn_hbox)

	var back_btn := Button.new()
	back_btn.text = "Back"
	back_btn.custom_minimum_size = Vector2(100.0, 38.0)
	apply_button_style(back_btn)
	back_btn.pressed.connect(
		func():
			ctrl._char_create_layer.visible = false
			ctrl._enter_character_select()
	)
	btn_hbox.add_child(back_btn)

	ctrl._char_create_btn = Button.new()
	ctrl._char_create_btn.text = "Create"
	ctrl._char_create_btn.custom_minimum_size = Vector2(160.0, 38.0)
	apply_button_style(ctrl._char_create_btn)
	ctrl._char_create_btn.pressed.connect(ctrl._on_create_character_pressed)
	btn_hbox.add_child(ctrl._char_create_btn)


# =============================================================================
# Hub UI
# =============================================================================


func create_hub_ui() -> void:
	ctrl._hub_layer = CanvasLayer.new()
	ctrl._hub_layer.layer = 14
	ctrl._hub_layer.visible = false
	ctrl.add_child(ctrl._hub_layer)

	# Class selection (top-center)
	ctrl._hub_class_label = Label.new()
	ctrl._hub_class_label.text = "[1] Gunner  [2] Vanguard  [3] Blade Dancer"
	ctrl._hub_class_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	ctrl._hub_class_label.anchor_left = 0.0
	ctrl._hub_class_label.anchor_right = 1.0
	ctrl._hub_class_label.anchor_top = 0.0
	ctrl._hub_class_label.anchor_bottom = 0.0
	ctrl._hub_class_label.offset_top = 10.0
	ctrl._hub_class_label.offset_bottom = 60.0
	apply_overlay_label(ctrl._hub_class_label, 15, UI_TEXT_MUTED)
	ctrl._hub_layer.add_child(ctrl._hub_class_label)

	# Status (top-left)
	ctrl._hub_status_label = Label.new()
	ctrl._hub_status_label.text = "Hub - Walk to the portal to enter the arena"
	ctrl._hub_status_label.anchor_left = 0.0
	ctrl._hub_status_label.anchor_right = 1.0
	ctrl._hub_status_label.anchor_top = 0.0
	ctrl._hub_status_label.anchor_bottom = 0.0
	ctrl._hub_status_label.offset_top = 65.0
	ctrl._hub_status_label.offset_bottom = 90.0
	ctrl._hub_status_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	apply_overlay_label(ctrl._hub_status_label, 13, UI_TEXT_DIM)
	ctrl._hub_layer.add_child(ctrl._hub_status_label)

	# Portal prompt (center-bottom)
	ctrl._portal_prompt = Label.new()
	ctrl._portal_prompt.text = "Press [E] to enter Arena"
	ctrl._portal_prompt.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	ctrl._portal_prompt.anchor_left = 0.0
	ctrl._portal_prompt.anchor_right = 1.0
	ctrl._portal_prompt.anchor_top = 0.7
	ctrl._portal_prompt.anchor_bottom = 0.75
	apply_overlay_label(ctrl._portal_prompt, 24, UI_BORDER_ACTIVE)
	ctrl._portal_prompt.visible = false
	ctrl._hub_layer.add_child(ctrl._portal_prompt)

	# Lift prompt
	ctrl._lift_prompt = Label.new()
	ctrl._lift_prompt.text = "Press [E] — Go up"
	ctrl._lift_prompt.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	ctrl._lift_prompt.anchor_left = 0.0
	ctrl._lift_prompt.anchor_right = 1.0
	ctrl._lift_prompt.anchor_top = 0.65
	ctrl._lift_prompt.anchor_bottom = 0.7
	apply_overlay_label(ctrl._lift_prompt, 20, UI_TEXT_MUTED)
	ctrl._lift_prompt.visible = false
	ctrl._hub_layer.add_child(ctrl._lift_prompt)


# =============================================================================
# Group panel
# =============================================================================


func create_group_panel() -> void:
	ctrl._group_panel = PanelContainer.new()
	ctrl._group_panel.anchor_left = 0.0
	ctrl._group_panel.anchor_right = 0.0
	ctrl._group_panel.anchor_top = 0.0
	ctrl._group_panel.anchor_bottom = 0.0
	ctrl._group_panel.offset_left = 10.0
	ctrl._group_panel.offset_top = 80.0
	ctrl._group_panel.offset_right = 228.0
	ctrl._group_panel.offset_bottom = 220.0
	ctrl._group_panel.visible = false
	apply_panel_style(ctrl._group_panel, false, 10)
	ctrl._hub_layer.add_child(ctrl._group_panel)

	var vbox := VBoxContainer.new()
	vbox.add_theme_constant_override("separation", 6)
	ctrl._group_panel.add_child(vbox)

	ctrl._group_label = Label.new()
	ctrl._group_label.text = "No group\n[G] Create group"
	apply_overlay_label(ctrl._group_label, 13, UI_TEXT)
	vbox.add_child(ctrl._group_label)

	ctrl._group_leave_btn = Button.new()
	ctrl._group_leave_btn.text = "Leave Group [G]"
	ctrl._group_leave_btn.custom_minimum_size.y = 34.0
	apply_button_style(ctrl._group_leave_btn, UI_DANGER)
	ctrl._group_leave_btn.visible = false
	ctrl._group_leave_btn.pressed.connect(func(): NetworkManager.send_group_leave())
	vbox.add_child(ctrl._group_leave_btn)


# =============================================================================
# Invite popup
# =============================================================================


func create_invite_popup() -> void:
	ctrl._invite_popup = CanvasLayer.new()
	ctrl._invite_popup.layer = 21
	ctrl._invite_popup.visible = false
	ctrl.add_child(ctrl._invite_popup)

	var bg := ColorRect.new()
	bg.color = Color(0.0, 0.0, 0.0, 0.7)
	bg.anchor_right = 1.0
	bg.anchor_bottom = 1.0
	bg.mouse_filter = Control.MOUSE_FILTER_STOP
	ctrl._invite_popup.add_child(bg)

	var panel := PanelContainer.new()
	panel.anchor_left = 0.5
	panel.anchor_right = 0.5
	panel.anchor_top = 0.35
	panel.anchor_bottom = 0.5
	panel.offset_left = -170.0
	panel.offset_right = 170.0
	apply_panel_style(panel, false, 12)
	ctrl._invite_popup.add_child(panel)

	var vbox := VBoxContainer.new()
	vbox.add_theme_constant_override("separation", 10)
	panel.add_child(vbox)

	ctrl._invite_label = Label.new()
	ctrl._invite_label.text = "Group invitation"
	ctrl._invite_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	apply_overlay_label(ctrl._invite_label, 16, UI_TEXT)
	vbox.add_child(ctrl._invite_label)

	var hbox := HBoxContainer.new()
	hbox.add_theme_constant_override("separation", 16)
	hbox.alignment = BoxContainer.ALIGNMENT_CENTER
	vbox.add_child(hbox)

	var accept_btn := Button.new()
	accept_btn.text = "Accept"
	accept_btn.custom_minimum_size = Vector2(100, 36)
	apply_button_style(accept_btn)
	accept_btn.pressed.connect(ctrl.group_mgr.accept_invite)
	hbox.add_child(accept_btn)

	var decline_btn := Button.new()
	decline_btn.text = "Decline"
	decline_btn.custom_minimum_size = Vector2(100, 36)
	apply_button_style(decline_btn, UI_DANGER)
	decline_btn.pressed.connect(ctrl.group_mgr.decline_invite)
	hbox.add_child(decline_btn)


# =============================================================================
# Shared HUD
# =============================================================================


func create_shared_hud() -> void:
	ctrl._shared_hud_layer = CanvasLayer.new()
	ctrl._shared_hud_layer.layer = 9  # below class HUDs (10), below damage overlay
	ctrl.add_child(ctrl._shared_hud_layer)

	ctrl._shared_hud = preload("res://scenes/shared/hud/shared_hud.gd").new()
	ctrl._shared_hud.name = "SharedHUD"
	ctrl._shared_hud.anchor_right = 1.0
	ctrl._shared_hud.anchor_bottom = 1.0
	ctrl._shared_hud.mouse_filter = Control.MOUSE_FILTER_IGNORE
	ctrl._shared_hud_layer.add_child(ctrl._shared_hud)
	ctrl._shared_hud.set_player_names(ctrl._player_names)

	ctrl._map_overlay = preload("res://scenes/shared/hud/map_overlay.gd").new()
	ctrl._map_overlay.name = "MapOverlay"
	ctrl._map_overlay.anchor_right = 1.0
	ctrl._map_overlay.anchor_bottom = 1.0
	ctrl._map_overlay.mouse_filter = Control.MOUSE_FILTER_IGNORE
	ctrl._map_overlay.visible = false
	ctrl._shared_hud_layer.add_child(ctrl._map_overlay)


# =============================================================================
# Death overlay
# =============================================================================


func create_death_overlay() -> void:
	ctrl._death_overlay_layer = CanvasLayer.new()
	ctrl._death_overlay_layer.layer = 12
	ctrl._death_overlay_layer.process_mode = Node.PROCESS_MODE_ALWAYS
	ctrl._death_overlay_layer.visible = false
	ctrl.add_child(ctrl._death_overlay_layer)

	ctrl._death_overlay_bg = ColorRect.new()
	ctrl._death_overlay_bg.color = Color(0.0, 0.0, 0.0, 0.7)
	ctrl._death_overlay_bg.anchor_right = 1.0
	ctrl._death_overlay_bg.anchor_bottom = 1.0
	ctrl._death_overlay_bg.mouse_filter = Control.MOUSE_FILTER_STOP
	ctrl._death_overlay_layer.add_child(ctrl._death_overlay_bg)

	ctrl._death_label = Label.new()
	ctrl._death_label.text = "YOU DIED"
	ctrl._death_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	ctrl._death_label.vertical_alignment = VERTICAL_ALIGNMENT_CENTER
	ctrl._death_label.anchor_left = 0.0
	ctrl._death_label.anchor_right = 1.0
	ctrl._death_label.anchor_top = 0.3
	ctrl._death_label.anchor_bottom = 0.45
	apply_overlay_label(ctrl._death_label, 56, UI_DANGER)
	ctrl._death_overlay_layer.add_child(ctrl._death_label)

	var btn_container := VBoxContainer.new()
	btn_container.anchor_left = 0.5
	btn_container.anchor_right = 0.5
	btn_container.anchor_top = 0.55
	btn_container.anchor_bottom = 0.7
	btn_container.offset_left = -120.0
	btn_container.offset_right = 120.0
	btn_container.add_theme_constant_override("separation", 12)
	ctrl._death_overlay_layer.add_child(btn_container)

	ctrl._respawn_btn = Button.new()
	ctrl._respawn_btn.text = "Respawn"
	ctrl._respawn_btn.custom_minimum_size.y = 38.0
	apply_button_style(ctrl._respawn_btn)
	ctrl._respawn_btn.disabled = true
	ctrl._respawn_btn.pressed.connect(ctrl._on_respawn)
	btn_container.add_child(ctrl._respawn_btn)

	ctrl._respawn_hub_btn = Button.new()
	ctrl._respawn_hub_btn.text = "Return to Hub"
	ctrl._respawn_hub_btn.custom_minimum_size.y = 38.0
	apply_button_style(ctrl._respawn_hub_btn)
	ctrl._respawn_hub_btn.pressed.connect(ctrl._on_respawn_hub)
	btn_container.add_child(ctrl._respawn_hub_btn)


# =============================================================================
# Character select logic
# =============================================================================


func populate_char_select() -> void:
	# Update welcome label.
	if ctrl._account_username != "":
		ctrl._char_select_welcome.text = "Welcome, %s" % ctrl._account_username
	else:
		ctrl._char_select_welcome.text = ""

	# Clear existing rows.
	for child in ctrl._char_list_container.get_children():
		child.queue_free()

	var characters: Array = ctrl._char_list_data.get("characters", [])
	var last_id: int = ctrl._char_list_data.get("last_char_id", 0)

	var normal_style := StyleBoxFlat.new()
	normal_style.bg_color = UI_SURFACE
	normal_style.border_color = UI_BORDER
	normal_style.set_border_width_all(1)
	normal_style.set_corner_radius_all(0)
	normal_style.set_content_margin_all(10)

	var selected_style := StyleBoxFlat.new()
	selected_style.bg_color = UI_SURFACE_ACTIVE
	selected_style.border_color = UI_BORDER_ACTIVE
	selected_style.set_border_width_all(1)
	selected_style.set_corner_radius_all(0)
	selected_style.set_content_margin_all(10)

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

		# Click detection
		var btn := Button.new()
		btn.flat = true
		btn.anchor_right = 1.0
		btn.anchor_bottom = 1.0
		btn.mouse_filter = Control.MOUSE_FILTER_STOP
		var id_capture: int = char_id
		var cls_capture: String = ch.class_name
		btn.pressed.connect(func(): ctrl._select_character_row(id_capture, cls_capture))
		row.add_child(btn)

	# Auto-select last played if none selected.
	if ctrl._selected_char_id == 0 and not characters.is_empty():
		ctrl._select_character_row(characters[0].char_id, characters[0].class_name)
