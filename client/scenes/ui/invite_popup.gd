extends CanvasLayer

var _styled: bool = false

@onready var invite_label: Label = %InviteLabel
@onready var accept_btn: Button = %AcceptBtn
@onready var decline_btn: Button = %DeclineBtn


func _ready() -> void:
	_apply_styles()


func _apply_styles() -> void:
	if _styled:
		return
	_styled = true
	var ui_ctrl: Node = get_parent().get_node("UIController")
	ui_ctrl.apply_panel_style($Panel, false, 12)
	ui_ctrl.apply_overlay_label(invite_label, 16, ui_ctrl.UI_TEXT)
	ui_ctrl.apply_button_style(accept_btn)
	ui_ctrl.apply_button_style(decline_btn, ui_ctrl.UI_DANGER)
