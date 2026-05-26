extends CanvasLayer

var cards: Dictionary = {}  # class_name_str -> PanelContainer
var _styled: bool = false

@onready var cards_hbox: HBoxContainer = %CardsHBox
@onready var name_input: LineEdit = %NameInput
@onready var error_label: Label = %ErrorLabel
@onready var back_btn: Button = %BackBtn
@onready var create_btn: Button = %CreateBtn


func _ready() -> void:
	_apply_styles()
	_build_class_cards()


func _apply_styles() -> void:
	if _styled:
		return
	_styled = true
	var ui_ctrl: Node = get_parent().get_node("UIController")
	ui_ctrl.apply_panel_style($Panel, false, 14)
	ui_ctrl.apply_overlay_label($Panel/Outer/Title, 26, ui_ctrl.UI_TEXT)
	ui_ctrl.apply_overlay_label($Panel/Outer/NameLabel, 16, ui_ctrl.UI_TEXT_MUTED)
	ui_ctrl.apply_overlay_label(error_label, 13, ui_ctrl.UI_DANGER)
	ui_ctrl.apply_line_edit_style(name_input)
	ui_ctrl.apply_button_style(back_btn)
	ui_ctrl.apply_button_style(create_btn)


func _build_class_cards() -> void:
	var ui_ctrl: Node = get_parent().get_node("UIController")
	var ctrl: Node = get_parent()

	var normal_style := StyleBoxFlat.new()
	normal_style.bg_color = ui_ctrl.UI_SURFACE
	normal_style.border_color = ui_ctrl.UI_BORDER
	normal_style.set_border_width_all(1)
	normal_style.set_corner_radius_all(0)
	normal_style.set_content_margin_all(12)

	var selected_style := StyleBoxFlat.new()
	selected_style.bg_color = ui_ctrl.UI_SURFACE_ACTIVE
	selected_style.border_color = ui_ctrl.UI_BORDER_ACTIVE
	selected_style.set_border_width_all(1)
	selected_style.set_corner_radius_all(0)
	selected_style.set_content_margin_all(12)

	for cls in ctrl.CLASS_INFO:
		_build_single_card(cls, ctrl.CLASS_INFO[cls], normal_style, selected_style, ui_ctrl)


func _build_single_card(
	cls: String,
	info: Dictionary,
	normal_style: StyleBoxFlat,
	selected_style: StyleBoxFlat,
	ui_ctrl: Node
) -> void:
	var ctrl: Node = get_parent()
	var card := PanelContainer.new()
	card.custom_minimum_size = Vector2(190.0, 220.0)
	card.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	card.add_theme_stylebox_override("panel", normal_style)
	card.set_meta("normal_style", normal_style)
	card.set_meta("selected_style", selected_style)
	cards_hbox.add_child(card)
	cards[cls] = card

	var vbox := VBoxContainer.new()
	vbox.add_theme_constant_override("separation", 8)
	card.add_child(vbox)

	var name_label := Label.new()
	name_label.text = info.name
	name_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	ui_ctrl.apply_overlay_label(name_label, 21, ui_ctrl.UI_TEXT)
	vbox.add_child(name_label)

	var genre_label := Label.new()
	genre_label.text = info.genre
	genre_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	ui_ctrl.apply_overlay_label(genre_label, 13, ui_ctrl.UI_BORDER_ACTIVE)
	vbox.add_child(genre_label)

	var sep := HSeparator.new()
	sep.add_theme_color_override("separator", Color(ui_ctrl.UI_BORDER, 0.75))
	vbox.add_child(sep)

	var desc_label := Label.new()
	desc_label.text = info.desc
	desc_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	ui_ctrl.apply_overlay_label(desc_label, 13, ui_ctrl.UI_TEXT_DIM)
	desc_label.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	vbox.add_child(desc_label)

	var click_btn := Button.new()
	click_btn.flat = true
	click_btn.anchor_right = 1.0
	click_btn.anchor_bottom = 1.0
	click_btn.mouse_filter = Control.MOUSE_FILTER_STOP
	var cls_capture: String = cls
	click_btn.pressed.connect(func(): ctrl._select_create_class(cls_capture))
	card.add_child(click_btn)
