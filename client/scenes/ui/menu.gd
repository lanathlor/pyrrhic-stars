extends CanvasLayer

var register_mode: bool = false
var _styled: bool = false

@onready var welcome_label: Label = %WelcomeLabel
@onready var email_input: LineEdit = %EmailInput
@onready var password_input: LineEdit = %PasswordInput
@onready var username_input: LineEdit = %UsernameInput
@onready var status_label: Label = %StatusLabel
@onready var play_btn: Button = %PlayBtn
@onready var toggle_mode_btn: Button = %ToggleModeBtn
@onready var switch_account_btn: Button = %SwitchAccountBtn
@onready var replays_btn: Button = %ReplaysBtn
@onready var settings_btn: Button = %SettingsBtn


func _ready() -> void:
	_apply_styles()
	toggle_mode_btn.pressed.connect(_on_toggle_mode)
	_update_mode()


func _apply_styles() -> void:
	if _styled:
		return
	_styled = true
	var ui_ctrl: Node = get_parent().get_node("UIController")
	ui_ctrl.apply_panel_style($Panel, false, 14)
	ui_ctrl.apply_overlay_label($Panel/VBox/Title, 34, ui_ctrl.UI_TEXT)
	ui_ctrl.apply_overlay_label($Panel/VBox/Subtitle, 14, ui_ctrl.UI_TEXT_DIM)
	ui_ctrl.apply_overlay_label(welcome_label, 18, ui_ctrl.UI_TEXT_MUTED)
	ui_ctrl.apply_overlay_label(status_label, 14, ui_ctrl.UI_DANGER)
	ui_ctrl.apply_line_edit_style(email_input)
	ui_ctrl.apply_line_edit_style(password_input)
	ui_ctrl.apply_line_edit_style(username_input)
	ui_ctrl.apply_button_style(play_btn)
	ui_ctrl.apply_button_style(replays_btn)
	ui_ctrl.apply_button_style(settings_btn)
	# Flat "link" buttons: tint so they read as actions on the dark panel.
	toggle_mode_btn.add_theme_color_override("font_color", ui_ctrl.UI_BORDER_ACTIVE)
	toggle_mode_btn.add_theme_color_override("font_hover_color", ui_ctrl.UI_TEXT)
	switch_account_btn.add_theme_color_override("font_color", ui_ctrl.UI_TEXT_MUTED)
	switch_account_btn.add_theme_color_override("font_hover_color", ui_ctrl.UI_TEXT)


## Shows an error/info message under the form.
func show_status(message: String) -> void:
	status_label.text = message
	status_label.visible = message != ""


## Switches the visible form between the "returning user" (saved token) view and
## the credential entry view.
func set_returning(is_returning: bool, account_label: String = "") -> void:
	email_input.visible = not is_returning
	password_input.visible = not is_returning
	toggle_mode_btn.visible = not is_returning
	switch_account_btn.visible = is_returning
	if is_returning:
		username_input.visible = false
		welcome_label.text = "Welcome back, %s" % account_label
		welcome_label.visible = account_label != ""
		play_btn.text = "Play"
	else:
		welcome_label.visible = false
		_update_mode()


func _on_toggle_mode() -> void:
	register_mode = not register_mode
	show_status("")
	_update_mode()


func _update_mode() -> void:
	username_input.visible = register_mode
	if register_mode:
		play_btn.text = "Register"
		toggle_mode_btn.text = "Have an account? Log in"
	else:
		play_btn.text = "Log In"
		toggle_mode_btn.text = "Need an account? Register"
