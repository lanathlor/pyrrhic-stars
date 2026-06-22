extends CanvasLayer

## First-session onboarding: explains the core gameplay loop (pick a class, gear
## up, group, enter the arena, beat the boss in time, spend and push harder).
## Built entirely in code like the overflux and settings panels. Auto-shown once
## on the first hub entry (tracked via SettingsManager "ui/seen_how_to_play") and
## re-openable from the pause menu or the [H] hotkey.

signal closed

const SEEN_SECTION := "ui"
const SEEN_KEY := "seen_how_to_play"

## name + body for each loop step. Keybinds are inline so a brand-new player can
## act on them immediately. No em dashes per the project copy rules.
const STEPS := [
	{
		"name": "Know your class",
		"body":
		"Each class plays a different genre. Press [N] anytime to see your abilities and switch spec.",
	},
	{
		"name": "Gear up in the hub",
		"body":
		"Spend scrip with a merchant to upgrade your kit. Take the lift to reach other floors.",
	},
	{
		"name": "Team up (optional)",
		"body":
		(
			"Press [G] for the social panel. Aim at a player and press [E] to invite them."
			+ " Up to 5 per group."
		),
	},
	{
		"name": "Enter the Arena",
		"body": "Walk to the portal and press [E], pick your Overflux conditions, then drop in.",
	},
	{
		"name": "Beat the boss in time",
		"body":
		(
			"Press [R] to ready up. Defeat the boss before the clear timer ends for the full"
			+ " reward. Finishing over time still pays out, but less."
		),
	},
	{
		"name": "Spend, then push higher",
		"body":
		(
			"Claim your scrip, upgrade at the merchant, and raise your Overflux for harder"
			+ " fights and better rewards."
		),
	},
]
const CONTROLS_HINT := (
	"Move: WASD  ·  Look: Mouse  ·  Free the cursor: hold Alt"
	+ "  ·  Reopen this: [H] or pause menu"
)

var ui_ctrl: Node = null

var _panel: PanelContainer


func _ready() -> void:
	layer = 28
	process_mode = Node.PROCESS_MODE_ALWAYS
	visible = false
	if ui_ctrl == null:
		ui_ctrl = get_parent().get_node_or_null("UIController")
	_build()


## Opens the guide and marks it as seen so it never auto-pops again.
func open() -> void:
	visible = true
	Input.set_mouse_mode(Input.MOUSE_MODE_VISIBLE)
	SettingsManager.set_value(SEEN_SECTION, SEEN_KEY, true)


## Shows the guide only the first time a player ever reaches the hub.
func open_if_first_time() -> void:
	if bool(SettingsManager.get_value(SEEN_SECTION, SEEN_KEY, false)):
		return
	open()


func close() -> void:
	visible = false
	closed.emit()


# =============================================================================
# Construction
# =============================================================================


func _build() -> void:
	var bg := ColorRect.new()
	bg.color = Color(0, 0, 0, 0.72)
	bg.set_anchors_preset(Control.PRESET_FULL_RECT)
	bg.mouse_filter = Control.MOUSE_FILTER_STOP
	add_child(bg)

	var center := CenterContainer.new()
	center.set_anchors_preset(Control.PRESET_FULL_RECT)
	add_child(center)

	_panel = PanelContainer.new()
	_panel.custom_minimum_size = Vector2(620, 0)
	center.add_child(_panel)
	if ui_ctrl:
		ui_ctrl.apply_panel_style(_panel, false, 24)

	var vbox := VBoxContainer.new()
	vbox.add_theme_constant_override("separation", 14)
	_panel.add_child(vbox)

	_build_header(vbox)

	var step_no := 1
	for step in STEPS:
		vbox.add_child(_build_step(step_no, step.name, step.body))
		step_no += 1

	_build_footer(vbox)


func _build_header(vbox: VBoxContainer) -> void:
	var title := Label.new()
	title.text = "How to Play"
	title.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	vbox.add_child(title)
	if ui_ctrl:
		ui_ctrl.apply_overlay_label(title, 24, ui_ctrl.UI_TEXT)

	var intro := Label.new()
	intro.text = "Pyrrhic Stars is a co-op boss hunt. Here is the loop."
	intro.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	intro.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	vbox.add_child(intro)
	if ui_ctrl:
		ui_ctrl.apply_overlay_label(intro, 14, ui_ctrl.UI_TEXT_MUTED)

	var sep := HSeparator.new()
	if ui_ctrl:
		sep.add_theme_stylebox_override("separator", _separator_style())
	vbox.add_child(sep)


func _build_footer(vbox: VBoxContainer) -> void:
	var sep2 := HSeparator.new()
	if ui_ctrl:
		sep2.add_theme_stylebox_override("separator", _separator_style())
	vbox.add_child(sep2)

	var controls := Label.new()
	controls.text = CONTROLS_HINT
	controls.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	controls.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	vbox.add_child(controls)
	if ui_ctrl:
		ui_ctrl.apply_overlay_label(controls, 12, ui_ctrl.UI_TEXT_DIM)

	var footer := HBoxContainer.new()
	footer.alignment = BoxContainer.ALIGNMENT_CENTER
	vbox.add_child(footer)
	var got_it := Button.new()
	got_it.text = "Got it"
	got_it.custom_minimum_size = Vector2(160, 38)
	got_it.pressed.connect(close)
	footer.add_child(got_it)
	if ui_ctrl:
		ui_ctrl.apply_button_style(got_it)


func _build_step(number: int, step_name: String, body: String) -> HBoxContainer:
	var row := HBoxContainer.new()
	row.add_theme_constant_override("separation", 14)

	var num := Label.new()
	num.text = str(number)
	num.custom_minimum_size = Vector2(28, 0)
	num.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	num.vertical_alignment = VERTICAL_ALIGNMENT_TOP
	row.add_child(num)
	if ui_ctrl:
		ui_ctrl.apply_overlay_label(num, 20, ui_ctrl.UI_BORDER_ACTIVE)

	var text_col := VBoxContainer.new()
	text_col.add_theme_constant_override("separation", 2)
	text_col.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	row.add_child(text_col)

	var name_lbl := Label.new()
	name_lbl.text = step_name
	text_col.add_child(name_lbl)
	if ui_ctrl:
		ui_ctrl.apply_overlay_label(name_lbl, 16, ui_ctrl.UI_TEXT)

	var body_lbl := Label.new()
	body_lbl.text = body
	body_lbl.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	body_lbl.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	text_col.add_child(body_lbl)
	if ui_ctrl:
		ui_ctrl.apply_overlay_label(body_lbl, 13, ui_ctrl.UI_TEXT_MUTED)

	return row


func _separator_style() -> StyleBoxLine:
	var s := StyleBoxLine.new()
	s.color = ui_ctrl.UI_BORDER
	s.thickness = 1
	return s


func _input(event: InputEvent) -> void:
	if not visible:
		return
	# Esc is handled centrally (main._input -> ui_controller.close_open_overlay),
	# but a click outside the panel dismisses it like the other overlays.
	if event is InputEventMouseButton and event.pressed and event.button_index == MOUSE_BUTTON_LEFT:
		if not _panel.get_global_rect().has_point(event.position):
			close()
			get_viewport().set_input_as_handled()
