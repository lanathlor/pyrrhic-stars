extends Node

## Character selection, creation, and network callbacks for character flow.

var ctrl: Node


func _ready() -> void:
	ctrl = get_parent()


# =============================================================================
# Character state callbacks
# =============================================================================


func on_character_state(data: Dictionary) -> void:
	# Server confirmed character selection. Restore position and enter hub.
	ctrl._selected_char_id = data.get("char_id", 0)
	if data.class_name != "":
		ctrl._local_class = data.class_name
	if data.position != Vector3.ZERO:
		ctrl._saved_hub_position = data.position
		ctrl._saved_hub_rot_y = data.rot_y
		ctrl._has_saved_state = true
	var char_name: String = data.get("char_name", "")
	print(
		(
			"[Main] Character confirmed: id=%d class=%s name=%s pos=%s"
			% [ctrl._selected_char_id, ctrl._local_class, char_name, ctrl._saved_hub_position]
		)
	)


func on_character_list(data: Dictionary) -> void:
	ctrl._char_list_data = data
	ctrl._account_username = data.get("username", "")
	var last_id: int = data.get("last_char_id", 0)
	ctrl._selected_char_id = last_id
	# Set local class from the last played character.
	for ch in data.get("characters", []):
		if ch.char_id == last_id:
			ctrl._local_class = ch.class_name
			break
	print(
		(
			"[Main] Character list: %d characters, username=%s"
			% [data.characters.size(), ctrl._account_username]
		)
	)
	enter_character_select()


func on_character_error(data: Dictionary) -> void:
	print("[Main] Character error: %s" % data.message)
	if ctrl._char_create_error_label:
		ctrl._char_create_error_label.text = data.message
		ctrl._char_create_error_label.visible = true
	if ctrl._char_create_btn:
		ctrl._char_create_btn.disabled = false


func on_net_connected() -> void:
	if (
		ctrl.state == ctrl.GameState.CHARACTER_SELECT
		or ctrl.state == ctrl.GameState.CREATE_CHARACTER
	):
		# ZoneJoined after character selection/creation -- enter hub
		print("[Main] Joined hub as peer %d" % NetworkManager.get_my_id())
		ctrl._enter_hub()
	else:
		print("[Main] Connected, waiting for character list...")


func on_net_connection_failed() -> void:
	print("[Main] Connection failed")
	ctrl._enter_menu()


func on_net_player_disconnected(peer_id: int) -> void:
	print("[Main] Peer %d disconnected" % peer_id)
	if peer_id in ctrl.entity_mgr.spawned_players:
		var player: CharacterBody3D = ctrl.entity_mgr.spawned_players[peer_id]
		if is_instance_valid(player):
			player.queue_free()
		ctrl.entity_mgr.spawned_players.erase(peer_id)


# =============================================================================
# Character Select
# =============================================================================


func enter_character_select() -> void:
	ctrl.state = ctrl.GameState.CHARACTER_SELECT
	ctrl._menu_layer.visible = false
	ctrl._hub_layer.visible = false
	ctrl._char_create_layer.visible = false
	ctrl._char_select_layer.visible = true
	ctrl._inventory_layer.toolbar_panel.visible = false
	Input.set_mouse_mode(Input.MOUSE_MODE_VISIBLE)
	ctrl.ui_ctrl.populate_char_select()


func select_character_row(char_id: int, class_name_str: String) -> void:
	ctrl._selected_char_id = char_id
	ctrl._local_class = class_name_str
	# Update row highlights.
	for row in ctrl._char_list_container.get_children():
		if row is PanelContainer and row.has_meta("char_id"):
			if row.get_meta("char_id") == char_id:
				row.add_theme_stylebox_override("panel", row.get_meta("selected_style"))
			else:
				row.add_theme_stylebox_override("panel", row.get_meta("normal_style"))


func on_enter_world_pressed() -> void:
	if ctrl._selected_char_id == 0:
		return
	ctrl._enter_world_btn.disabled = true
	NetworkManager.send_select_character(ctrl._selected_char_id)


# =============================================================================
# Create Character
# =============================================================================


func enter_create_character() -> void:
	ctrl.state = ctrl.GameState.CREATE_CHARACTER
	ctrl._char_select_layer.visible = false
	ctrl._char_create_layer.visible = true
	ctrl._inventory_layer.toolbar_panel.visible = false
	ctrl._char_create_error_label.visible = false
	ctrl._char_name_input.text = ""
	ctrl._char_create_btn.disabled = false
	Input.set_mouse_mode(Input.MOUSE_MODE_VISIBLE)
	# Default select gunner.
	select_create_class("gunner")


func select_create_class(cls: String) -> void:
	ctrl._local_class = cls
	for c_name in ctrl._char_create_cards:
		var card: PanelContainer = ctrl._char_create_cards[c_name]
		if c_name == cls:
			card.add_theme_stylebox_override("panel", card.get_meta("selected_style"))
		else:
			card.add_theme_stylebox_override("panel", card.get_meta("normal_style"))


func on_create_character_pressed() -> void:
	var char_name: String = ctrl._char_name_input.text.strip_edges()
	if char_name.length() < 2 or char_name.length() > 20:
		ctrl._char_create_error_label.text = "Name must be 2-20 characters."
		ctrl._char_create_error_label.visible = true
		return
	ctrl._char_create_error_label.visible = false
	ctrl._char_create_btn.disabled = true
	NetworkManager.send_create_character(ctrl._local_class, char_name)


# =============================================================================
# Connection / username
# =============================================================================


func on_connect_pressed() -> void:
	# If no saved username, require input.
	if ctrl._username == "":
		ctrl._username = ctrl._username_input.text.strip_edges()
		if ctrl._username == "":
			ctrl._username_input.grab_focus()
			return
		_save_username(ctrl._username)

	NetworkManager.username = ctrl._username
	NetworkManager.disconnect_game()
	var err: int = NetworkManager.connect_to_server(ctrl.server_address)
	if err != OK:
		print("[Main] Failed to connect: %s" % error_string(err))
		return
	print("[Main] Connecting to %s:%d..." % [ctrl.server_address, NetworkManager.DEFAULT_PORT])
	ctrl._menu_layer.visible = false


func load_saved_username() -> String:
	if not FileAccess.file_exists(ctrl.USERNAME_SAVE_PATH):
		return ""
	var f: FileAccess = FileAccess.open(ctrl.USERNAME_SAVE_PATH, FileAccess.READ)
	if f == null:
		return ""
	var uname: String = f.get_as_text().strip_edges()
	f.close()
	return uname


func _save_username(uname: String) -> void:
	var f: FileAccess = FileAccess.open(ctrl.USERNAME_SAVE_PATH, FileAccess.WRITE)
	if f == null:
		return
	f.store_string(uname)
	f.close()


# =============================================================================
# Class/spec selection
# =============================================================================


func select_class(class_name_str: String) -> void:
	ctrl._local_class = class_name_str
	if NetworkManager.is_active:
		NetworkManager.set_player_class(class_name_str)
	ctrl.hub_interact.update_hub_display()


func toggle_spec_panel() -> void:
	ctrl._spec_panel.set_specs(ctrl.SPEC_INFO.get(ctrl._local_class, []), ctrl._local_spec)
	ctrl._spec_panel.toggle()
	ctrl._update_cursor_mode()
	ctrl._sync_toolbar_active()


func on_spec_selected(spec_id: String) -> void:
	if spec_id == ctrl._local_spec:
		return
	ctrl._local_spec = spec_id
	if NetworkManager.is_active:
		NetworkManager.set_player_spec(spec_id)
	ctrl._spec_panel.set_specs(ctrl.SPEC_INFO.get(ctrl._local_class, []), ctrl._local_spec)
	ctrl.hub_interact.update_hub_display()
	# Immediately switch the live player's spec (don't wait for server round-trip).
	var my_id: int = NetworkManager.get_my_id()
	if my_id in ctrl.entity_mgr.spawned_players:
		var player: CharacterBody3D = ctrl.entity_mgr.spawned_players[my_id]
		if is_instance_valid(player) and "spec_id" in player:
			player._switch_spec(spec_id, true)
