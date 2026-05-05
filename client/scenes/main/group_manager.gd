extends Node

## Group state management: invite popup, group panel, error handling.

var ctrl: Node

var group_data: Dictionary = {}  # current group state
var pending_invite_group_id: int = 0


func _ready() -> void:
	ctrl = get_parent()


func on_group_state(data: Dictionary) -> void:
	group_data = data
	update_group_panel()
	if ctrl._shared_hud:
		ctrl._shared_hud.update_group_members(data)


func on_group_invite(group_id: int, leader_name: String) -> void:
	pending_invite_group_id = group_id
	ctrl._invite_label.text = "%s invites you to a group\n[Accept]  [Decline]" % leader_name
	ctrl._invite_popup.visible = true
	Input.set_mouse_mode(Input.MOUSE_MODE_VISIBLE)
	# Auto-decline after 30 seconds
	var captured_id: int = group_id
	ctrl.get_tree().create_timer(30.0).timeout.connect(
		func():
			if ctrl._invite_popup.visible and pending_invite_group_id == captured_id:
				decline_invite()
	)


func on_group_error(_code: int, msg: String) -> void:
	print("[Main] Group error: %s" % msg)
	if ctrl._hub_status_label:
		ctrl._hub_status_label.text = "Error: %s" % msg


func accept_invite() -> void:
	if pending_invite_group_id > 0:
		NetworkManager.send_group_invite_reply(pending_invite_group_id, true)
		pending_invite_group_id = 0
	ctrl._invite_popup.visible = false
	if ctrl.state == ctrl.GameState.HUB:
		Input.set_mouse_mode(Input.MOUSE_MODE_CAPTURED)


func decline_invite() -> void:
	if pending_invite_group_id > 0:
		NetworkManager.send_group_invite_reply(pending_invite_group_id, false)
		pending_invite_group_id = 0
	ctrl._invite_popup.visible = false
	if ctrl.state == ctrl.GameState.HUB:
		Input.set_mouse_mode(Input.MOUSE_MODE_CAPTURED)


func update_group_panel() -> void:
	if not ctrl._group_panel:
		return
	var gid: int = group_data.get("group_id", 0)
	if gid == 0:
		ctrl._group_label.text = "No group\n[G] Create group"
		ctrl._group_leave_btn.visible = false
		ctrl._group_panel.visible = ctrl.state == ctrl.GameState.HUB
		return

	var leader_peer: int = group_data.get("leader_peer", 0)
	var members: Array = group_data.get("members", [])
	var text := "Group:\n"
	for m in members:
		var uname: String = m.get("username", "???")
		var pid: int = m.get("peer_id", 0)
		var leader_str := " *" if pid == leader_peer else ""
		var you_str := " (you)" if pid == NetworkManager.get_my_id() else ""
		text += "  %s%s%s\n" % [uname, leader_str, you_str]
	ctrl._group_label.text = text
	ctrl._group_leave_btn.visible = true
	ctrl._group_panel.visible = ctrl.state == ctrl.GameState.HUB
