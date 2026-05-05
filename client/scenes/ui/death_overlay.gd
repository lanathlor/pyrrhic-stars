extends CanvasLayer

var _styled: bool = false

@onready var background: ColorRect = %Background
@onready var respawn_btn: Button = %RespawnBtn
@onready var respawn_hub_btn: Button = %RespawnHubBtn


func _ready() -> void:
	_apply_styles()


func _apply_styles() -> void:
	if _styled:
		return
	_styled = true
	var ui_ctrl: Node = get_parent().get_node("UIController")
	ui_ctrl.apply_overlay_label($DeathLabel, 56, ui_ctrl.UI_DANGER)
	ui_ctrl.apply_button_style(respawn_btn)
	ui_ctrl.apply_button_style(respawn_hub_btn)
