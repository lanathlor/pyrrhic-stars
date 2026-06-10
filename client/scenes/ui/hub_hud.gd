extends CanvasLayer

var _styled: bool = false

@onready var class_label: Label = %ClassLabel
@onready var status_label: Label = %StatusLabel
@onready var portal_prompt: Label = %PortalPrompt
@onready var lift_prompt: Label = %LiftPrompt
@onready var merchant_prompt: Label = %MerchantPrompt
@onready var group_panel: PanelContainer = %GroupPanel
@onready var group_label: Label = %GroupLabel
@onready var group_leave_btn: Button = %GroupLeaveBtn


func _ready() -> void:
	_apply_styles()


func _apply_styles() -> void:
	if _styled:
		return
	_styled = true
	var ui_ctrl: Node = get_parent().get_node("UIController")
	ui_ctrl.apply_overlay_label(class_label, 15, ui_ctrl.UI_TEXT_MUTED)
	ui_ctrl.apply_overlay_label(status_label, 13, ui_ctrl.UI_TEXT_DIM)
	ui_ctrl.apply_overlay_label(portal_prompt, 24, ui_ctrl.UI_BORDER_ACTIVE)
	ui_ctrl.apply_overlay_label(lift_prompt, 20, ui_ctrl.UI_TEXT_MUTED)
	ui_ctrl.apply_overlay_label(merchant_prompt, 20, ui_ctrl.UI_BORDER_ACTIVE)
	ui_ctrl.apply_panel_style(group_panel, false, 10)
	ui_ctrl.apply_overlay_label(group_label, 13, ui_ctrl.UI_TEXT)
	ui_ctrl.apply_button_style(group_leave_btn, ui_ctrl.UI_DANGER)
