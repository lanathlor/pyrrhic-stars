extends CanvasLayer

## Group + Friends management overlay, opened with [G] in the hub/lobby.
## Built entirely in code (no .tscn). Non-pausing: the world keeps running while
## open; main.gd owns cursor mode. Group actions (create/leave/invite/kick) and
## friend actions (request/accept/decline/remove, invite-to-group) live here.

signal closed

const ONLINE_COLOR := Color(0.36, 0.82, 0.46, 0.96)
const NAME_ACCOUNT := 0
const NAME_CHARACTER := 1

var ctrl: Node  # main.gd
var ui: Node  # UIController

var _group_data: Dictionary = {}
var _friends: Array = []  # [{user_id, name, online}]
var _requests: Array = []  # [{user_id, name}]
var _current_tab: String = "group"

# Widget references (dynamic lists are rebuilt on refresh).
var _panel: PanelContainer
var _group_tab_btn: Button
var _friends_tab_btn: Button
var _group_tab: VBoxContainer
var _friends_tab: VBoxContainer
var _group_status: Label
var _member_list: VBoxContainer
var _invite_row: HBoxContainer
var _invite_input: LineEdit
var _invite_by_char: CheckButton
var _create_btn: Button
var _leave_btn: Button
var _friend_input: LineEdit
var _friend_by_char: CheckButton
var _request_list: VBoxContainer
var _friend_list: VBoxContainer


func _ready() -> void:
	process_mode = Node.PROCESS_MODE_ALWAYS
	layer = 19
	visible = false
	ctrl = get_parent()
	ui = ctrl.get_node("UIController")
	_build_ui()
	_show_tab("group")


# =============================================================================
# Lifecycle
# =============================================================================


func open() -> void:
	visible = true
	NetworkManager.social.send_friend_list_request()
	_refresh_group()
	_refresh_friends()
	_show_tab(_current_tab)


func close() -> void:
	visible = false
	closed.emit()


func toggle() -> void:
	if visible:
		close()
	else:
		open()


# =============================================================================
# UI construction
# =============================================================================


func _build_ui() -> void:
	var bg := ColorRect.new()
	bg.color = Color(0.0, 0.0, 0.0, 0.55)
	bg.set_anchors_preset(Control.PRESET_FULL_RECT)
	bg.mouse_filter = Control.MOUSE_FILTER_STOP
	add_child(bg)

	var center := CenterContainer.new()
	center.set_anchors_preset(Control.PRESET_FULL_RECT)
	center.mouse_filter = Control.MOUSE_FILTER_PASS
	bg.add_child(center)

	_panel = PanelContainer.new()
	_panel.custom_minimum_size = Vector2(560, 520)
	ui.apply_panel_style(_panel, false, 16)
	center.add_child(_panel)

	var root := VBoxContainer.new()
	root.add_theme_constant_override("separation", 12)
	_panel.add_child(root)

	_build_header(root)
	_build_tab_row(root)
	_build_group_tab(root)
	_build_friends_tab(root)


func _build_header(parent: VBoxContainer) -> void:
	var header := HBoxContainer.new()
	header.add_theme_constant_override("separation", 8)
	parent.add_child(header)

	var title := Label.new()
	title.text = "Social"
	ui.apply_overlay_label(title, 22, ui.UI_TEXT)
	title.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	header.add_child(title)

	var close_btn := Button.new()
	close_btn.text = "X"
	close_btn.custom_minimum_size = Vector2(36, 0)
	ui.apply_button_style(close_btn)
	close_btn.pressed.connect(close)
	header.add_child(close_btn)


func _build_tab_row(parent: VBoxContainer) -> void:
	var row := HBoxContainer.new()
	row.add_theme_constant_override("separation", 8)
	parent.add_child(row)

	_group_tab_btn = Button.new()
	_group_tab_btn.text = "Group"
	_group_tab_btn.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	ui.apply_button_style(_group_tab_btn)
	_group_tab_btn.pressed.connect(func(): _show_tab("group"))
	row.add_child(_group_tab_btn)

	_friends_tab_btn = Button.new()
	_friends_tab_btn.text = "Friends"
	_friends_tab_btn.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	ui.apply_button_style(_friends_tab_btn)
	_friends_tab_btn.pressed.connect(func(): _show_tab("friends"))
	row.add_child(_friends_tab_btn)


func _build_group_tab(parent: VBoxContainer) -> void:
	_group_tab = VBoxContainer.new()
	_group_tab.add_theme_constant_override("separation", 10)
	_group_tab.size_flags_vertical = Control.SIZE_EXPAND_FILL
	parent.add_child(_group_tab)

	_group_status = Label.new()
	ui.apply_overlay_label(_group_status, 15, ui.UI_TEXT_MUTED)
	_group_tab.add_child(_group_status)

	_member_list = VBoxContainer.new()
	_member_list.add_theme_constant_override("separation", 4)
	_member_list.size_flags_vertical = Control.SIZE_EXPAND_FILL
	_group_tab.add_child(_member_list)

	_group_tab.add_child(_make_separator())

	# Invite-by-name row.
	_invite_row = HBoxContainer.new()
	_invite_row.add_theme_constant_override("separation", 6)
	_group_tab.add_child(_invite_row)

	_invite_input = LineEdit.new()
	_invite_input.placeholder_text = "Player name..."
	_invite_input.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	ui.apply_line_edit_style(_invite_input)
	_invite_input.text_submitted.connect(func(_t): _on_invite_submit())
	_invite_row.add_child(_invite_input)

	_invite_by_char = CheckButton.new()
	_invite_by_char.text = "By character"
	_invite_by_char.button_pressed = true
	_invite_row.add_child(_invite_by_char)

	var invite_btn := Button.new()
	invite_btn.text = "Invite"
	ui.apply_button_style(invite_btn)
	invite_btn.pressed.connect(_on_invite_submit)
	_invite_row.add_child(invite_btn)

	# Create / Leave actions.
	var actions := HBoxContainer.new()
	actions.add_theme_constant_override("separation", 8)
	_group_tab.add_child(actions)

	_create_btn = Button.new()
	_create_btn.text = "Create Group"
	_create_btn.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	ui.apply_button_style(_create_btn)
	_create_btn.pressed.connect(NetworkManager.send_group_create)
	actions.add_child(_create_btn)

	_leave_btn = Button.new()
	_leave_btn.text = "Leave Group"
	_leave_btn.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	ui.apply_button_style(_leave_btn, ui.UI_DANGER)
	_leave_btn.pressed.connect(NetworkManager.send_group_leave)
	actions.add_child(_leave_btn)


func _build_friends_tab(parent: VBoxContainer) -> void:
	_friends_tab = VBoxContainer.new()
	_friends_tab.add_theme_constant_override("separation", 10)
	_friends_tab.size_flags_vertical = Control.SIZE_EXPAND_FILL
	_friends_tab.visible = false
	parent.add_child(_friends_tab)

	# Add-friend row.
	var add_row := HBoxContainer.new()
	add_row.add_theme_constant_override("separation", 6)
	_friends_tab.add_child(add_row)

	_friend_input = LineEdit.new()
	_friend_input.placeholder_text = "Account name..."
	_friend_input.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	ui.apply_line_edit_style(_friend_input)
	_friend_input.text_submitted.connect(func(_t): _on_add_friend_submit())
	add_row.add_child(_friend_input)

	_friend_by_char = CheckButton.new()
	_friend_by_char.text = "By character"
	add_row.add_child(_friend_by_char)

	var add_btn := Button.new()
	add_btn.text = "Send Request"
	ui.apply_button_style(add_btn)
	add_btn.pressed.connect(_on_add_friend_submit)
	add_row.add_child(add_btn)

	var req_header := Label.new()
	req_header.text = "Incoming Requests"
	ui.apply_overlay_label(req_header, 14, ui.UI_BORDER_ACTIVE)
	_friends_tab.add_child(req_header)

	_request_list = VBoxContainer.new()
	_request_list.add_theme_constant_override("separation", 4)
	_friends_tab.add_child(_request_list)

	_friends_tab.add_child(_make_separator())

	var friends_header := Label.new()
	friends_header.text = "Friends"
	ui.apply_overlay_label(friends_header, 14, ui.UI_BORDER_ACTIVE)
	_friends_tab.add_child(friends_header)

	_friend_list = VBoxContainer.new()
	_friend_list.add_theme_constant_override("separation", 4)
	_friend_list.size_flags_vertical = Control.SIZE_EXPAND_FILL
	_friends_tab.add_child(_friend_list)


# =============================================================================
# Tabs
# =============================================================================


func _show_tab(which: String) -> void:
	_current_tab = which
	var group_active := which == "group"
	_group_tab.visible = group_active
	_friends_tab.visible = not group_active
	ui.apply_button_style(_group_tab_btn, ui.UI_BORDER_ACTIVE if group_active else ui.UI_BORDER)
	ui.apply_button_style(_friends_tab_btn, ui.UI_BORDER if group_active else ui.UI_BORDER_ACTIVE)
	if not group_active:
		_clear_request_badge()


func _clear_request_badge() -> void:
	_friends_tab_btn.text = "Friends" if _requests.is_empty() else "Friends (%d)" % _requests.size()


# =============================================================================
# Group tab
# =============================================================================


func update_group(data: Dictionary) -> void:
	_group_data = data
	if visible:
		_refresh_group()


func _refresh_group() -> void:
	for child in _member_list.get_children():
		child.queue_free()

	var gid: int = _group_data.get("group_id", 0)
	var in_group := gid > 0
	_create_btn.visible = not in_group
	_leave_btn.visible = in_group
	_invite_row.visible = in_group

	if not in_group:
		_group_status.text = "Not in a group. Create one to invite players."
		return

	var members: Array = _group_data.get("members", [])
	var leader_peer: int = _group_data.get("leader_peer", 0)
	var am_leader := leader_peer == NetworkManager.get_my_id()
	_group_status.text = "Group (%d/5)" % members.size()
	for m in members:
		_member_list.add_child(_build_member_row(m, am_leader, leader_peer))


func _build_member_row(m: Dictionary, am_leader: bool, leader_peer: int) -> PanelContainer:
	var pid: int = m.get("peer_id", 0)
	var uname: String = m.get("username", "???")

	var row := PanelContainer.new()
	ui.apply_panel_style(row, false, 8)
	var hbox := HBoxContainer.new()
	hbox.add_theme_constant_override("separation", 8)
	row.add_child(hbox)

	var label := Label.new()
	var suffix := ""
	if pid == leader_peer:
		suffix += " (leader)"
	if pid == NetworkManager.get_my_id():
		suffix += " (you)"
	label.text = uname + suffix
	ui.apply_overlay_label(label, 15, ui.UI_TEXT)
	label.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	hbox.add_child(label)

	# Leader can kick everyone except themselves.
	if am_leader and pid != leader_peer:
		var kick := Button.new()
		kick.text = "Kick"
		ui.apply_button_style(kick, ui.UI_DANGER)
		var captured: int = pid
		kick.pressed.connect(func(): NetworkManager.social.send_group_kick(captured))
		hbox.add_child(kick)

	return row


func _on_invite_submit() -> void:
	var entered := _invite_input.text.strip_edges()
	if entered == "":
		return
	var flag := NAME_CHARACTER if _invite_by_char.button_pressed else NAME_ACCOUNT
	NetworkManager.social.send_group_invite_by_name(flag, entered)
	_invite_input.clear()


# =============================================================================
# Friends tab
# =============================================================================


func update_friends(friends: Array) -> void:
	_friends = friends
	if visible:
		_refresh_friends()


func on_friend_request(user_id: String, name: String) -> void:
	# Avoid duplicates if the same request arrives twice (live + replay).
	for r in _requests:
		if r.get("user_id", "") == user_id:
			return
	_requests.append({"user_id": user_id, "name": name})
	if visible and _current_tab == "friends":
		_refresh_friends()
	else:
		_clear_request_badge()
		if ctrl.group_mgr:
			ctrl.group_mgr.on_group_error(0, "%s sent a friend request" % name)


func on_friend_error(msg: String) -> void:
	# Reuse the hub status line for transient friend errors (ambiguous name, etc.).
	if ctrl.group_mgr:
		ctrl.group_mgr.on_group_error(0, msg)


func on_friend_status(data: Dictionary) -> void:
	var uid: String = data.get("user_id", "")
	for f in _friends:
		if f.get("user_id", "") == uid:
			f["online"] = data.get("online", false)
			break
	if visible:
		_refresh_friends()


func _refresh_friends() -> void:
	_clear_request_badge()
	_rebuild_request_list()
	_rebuild_friend_list()


func _rebuild_request_list() -> void:
	for child in _request_list.get_children():
		child.queue_free()
	if _requests.is_empty():
		var none := Label.new()
		none.text = "No incoming requests."
		ui.apply_overlay_label(none, 13, ui.UI_TEXT_DIM)
		_request_list.add_child(none)
		return
	for r in _requests:
		_request_list.add_child(_build_request_row(r))


func _build_request_row(r: Dictionary) -> PanelContainer:
	var uid: String = r.get("user_id", "")
	var row := PanelContainer.new()
	ui.apply_panel_style(row, false, 8)
	var hbox := HBoxContainer.new()
	hbox.add_theme_constant_override("separation", 8)
	row.add_child(hbox)

	var label := Label.new()
	label.text = r.get("name", "???")
	ui.apply_overlay_label(label, 15, ui.UI_TEXT)
	label.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	hbox.add_child(label)

	var accept := Button.new()
	accept.text = "Accept"
	ui.apply_button_style(accept)
	accept.pressed.connect(func(): _respond_request(uid, true))
	hbox.add_child(accept)

	var decline := Button.new()
	decline.text = "Decline"
	ui.apply_button_style(decline, ui.UI_DANGER)
	decline.pressed.connect(func(): _respond_request(uid, false))
	hbox.add_child(decline)
	return row


func _respond_request(user_id: String, accept: bool) -> void:
	NetworkManager.social.send_friend_respond(accept, user_id)
	for i in range(_requests.size()):
		if _requests[i].get("user_id", "") == user_id:
			_requests.remove_at(i)
			break
	_refresh_friends()


func _rebuild_friend_list() -> void:
	for child in _friend_list.get_children():
		child.queue_free()
	if _friends.is_empty():
		var none := Label.new()
		none.text = "No friends yet. Send a request above."
		ui.apply_overlay_label(none, 13, ui.UI_TEXT_DIM)
		_friend_list.add_child(none)
		return
	var in_group: bool = _group_data.get("group_id", 0) > 0
	for f in _friends:
		_friend_list.add_child(_build_friend_row(f, in_group))


func _build_friend_row(f: Dictionary, in_group: bool) -> PanelContainer:
	var uid: String = f.get("user_id", "")
	var fname: String = f.get("name", "???")
	var online: bool = f.get("online", false)

	var row := PanelContainer.new()
	ui.apply_panel_style(row, false, 8)
	var hbox := HBoxContainer.new()
	hbox.add_theme_constant_override("separation", 8)
	row.add_child(hbox)

	var label := Label.new()
	label.text = "%s  %s" % [fname, "online" if online else "offline"]
	ui.apply_overlay_label(label, 15, ONLINE_COLOR if online else ui.UI_TEXT_DIM)
	label.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	hbox.add_child(label)

	if in_group and online:
		var invite := Button.new()
		invite.text = "Invite"
		ui.apply_button_style(invite)
		var captured: String = fname
		invite.pressed.connect(
			func(): NetworkManager.social.send_group_invite_by_name(NAME_CHARACTER, captured)
		)
		hbox.add_child(invite)

	var remove := Button.new()
	remove.text = "Remove"
	ui.apply_button_style(remove, ui.UI_DANGER)
	remove.pressed.connect(func(): NetworkManager.social.send_friend_remove(uid))
	hbox.add_child(remove)
	return row


func _on_add_friend_submit() -> void:
	var entered := _friend_input.text.strip_edges()
	if entered == "":
		return
	var flag := NAME_CHARACTER if _friend_by_char.button_pressed else NAME_ACCOUNT
	NetworkManager.social.send_friend_request(flag, entered)
	_friend_input.clear()


func _make_separator() -> HSeparator:
	var sep := HSeparator.new()
	var s := StyleBoxLine.new()
	s.color = ui.UI_BORDER
	s.thickness = 1
	sep.add_theme_stylebox_override("separator", s)
	return sep
