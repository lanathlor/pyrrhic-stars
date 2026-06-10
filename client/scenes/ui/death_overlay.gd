extends CanvasLayer

var reset_instance_btn: Button

var _styled: bool = false

@onready var background: ColorRect = %Background
@onready var respawn_btn: Button = %RespawnBtn
@onready var respawn_hub_btn: Button = %RespawnHubBtn


func _ready() -> void:
	_apply_styles()
	_build_reset_button()


func _build_reset_button() -> void:
	reset_instance_btn = Button.new()
	reset_instance_btn.text = "Reset Instance"
	reset_instance_btn.visible = false
	# Place it next to the respawn buttons
	respawn_hub_btn.get_parent().add_child(reset_instance_btn)


func _apply_styles() -> void:
	if _styled:
		return
	_styled = true
	var ui_ctrl: Node = get_parent().get_node("UIController")
	ui_ctrl.apply_overlay_label($DeathLabel, 56, ui_ctrl.UI_DANGER)
	ui_ctrl.apply_button_style(respawn_btn)
	ui_ctrl.apply_button_style(respawn_hub_btn)
	if reset_instance_btn:
		ui_ctrl.apply_button_style(reset_instance_btn)
