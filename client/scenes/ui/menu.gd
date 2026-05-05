extends CanvasLayer

var _styled: bool = false

@onready var welcome_label: Label = %WelcomeLabel
@onready var username_input: LineEdit = %UsernameInput
@onready var play_btn: Button = %PlayBtn
@onready var replays_btn: Button = %ReplaysBtn


func _ready() -> void:
	_apply_styles()


func _apply_styles() -> void:
	if _styled:
		return
	_styled = true
	var ui_ctrl: Node = get_parent().get_node("UIController")
	ui_ctrl.apply_panel_style($Panel, false, 14)
	ui_ctrl.apply_overlay_label($Panel/VBox/Title, 34, ui_ctrl.UI_TEXT)
	ui_ctrl.apply_overlay_label($Panel/VBox/Subtitle, 14, ui_ctrl.UI_TEXT_DIM)
	ui_ctrl.apply_overlay_label(welcome_label, 18, ui_ctrl.UI_TEXT_MUTED)
	ui_ctrl.apply_line_edit_style(username_input)
	ui_ctrl.apply_button_style(play_btn)
	ui_ctrl.apply_button_style(replays_btn)
