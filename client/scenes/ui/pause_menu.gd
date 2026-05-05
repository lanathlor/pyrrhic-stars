extends CanvasLayer

var _styled: bool = false

@onready var resume_btn: Button = %ResumeBtn
@onready var menu_btn: Button = %MenuBtn
@onready var quit_btn: Button = %QuitBtn


func _ready() -> void:
	_apply_styles()


func _apply_styles() -> void:
	if _styled:
		return
	_styled = true
	var ui_ctrl: Node = get_parent().get_node("UIController")
	ui_ctrl.apply_panel_style($Panel, false, 12)
	ui_ctrl.apply_overlay_label($Panel/VBox/Title, 22, ui_ctrl.UI_TEXT)
	ui_ctrl.apply_button_style(resume_btn)
	ui_ctrl.apply_button_style(menu_btn)
	ui_ctrl.apply_button_style(quit_btn, ui_ctrl.UI_DANGER)
