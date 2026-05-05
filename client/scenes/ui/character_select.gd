extends CanvasLayer

var _styled: bool = false

@onready var welcome_label: Label = %WelcomeLabel
@onready var char_list_container: VBoxContainer = %CharListContainer
@onready var back_btn: Button = %BackBtn
@onready var create_btn: Button = %CreateBtn
@onready var enter_world_btn: Button = %EnterWorldBtn


func _ready() -> void:
	_apply_styles()


func _apply_styles() -> void:
	if _styled:
		return
	_styled = true
	var ui_ctrl: Node = get_parent().get_node("UIController")
	ui_ctrl.apply_panel_style($Panel, false, 14)
	ui_ctrl.apply_overlay_label($Panel/Outer/Title, 26, ui_ctrl.UI_TEXT)
	ui_ctrl.apply_overlay_label(welcome_label, 14, ui_ctrl.UI_TEXT_DIM)
	ui_ctrl.apply_button_style(back_btn)
	ui_ctrl.apply_button_style(create_btn)
	ui_ctrl.apply_button_style(enter_world_btn)
