extends Node

## Game state transitions: menu, hub, arena warmup, fight, zone transfers, death.

var ctrl: Node


func _ready() -> void:
	ctrl = get_parent()


# =============================================================================
# Menu
# =============================================================================


func enter_menu() -> void:
	ctrl.state = ctrl.GameState.MENU
	NetworkManager.disconnect_game()
	ctrl._menu_layer.visible = true
	ctrl._hub_layer.visible = false
	ctrl._char_select_layer.visible = false
	ctrl._char_create_layer.visible = false
	ctrl._pause_layer.visible = false
	ctrl._lobby_panel.visible = false
	ctrl._inventory_layer.equip_panel.visible = false
	ctrl._inventory_layer.bag_panel.visible = false
	ctrl._inventory_layer.toolbar_panel.visible = false
	ctrl._spec_panel.visible = false
	ctrl._sync_toolbar_active()
	Input.set_mouse_mode(Input.MOUSE_MODE_VISIBLE)
	ctrl.env_builder.unload_environment()
	if ctrl._enter_world_btn:
		ctrl._enter_world_btn.disabled = false
	# Show the returning-user view or the login form depending on whether we
	# still hold a Kratos session token.
	if AuthManager.has_token():
		ctrl._menu_layer.set_returning(true, ctrl.char_mgr.load_saved_username())
	else:
		ctrl._menu_layer.set_returning(false)


# =============================================================================
# Hub
# =============================================================================


func show_portal_prompt_only() -> void:
	ctrl._hub_layer.visible = true
	ctrl._hub_class_label.visible = false
	ctrl._hub_status_label.visible = false
	if ctrl._lift_prompt:
		ctrl._lift_prompt.visible = false
	ctrl._group_panel.visible = false
	ctrl._portal_prompt.visible = false
	ctrl.hub_interact.near_portal = false


func enter_hub() -> void:
	ctrl.state = ctrl.GameState.HUB
	ctrl.get_tree().paused = false
	ctrl.paused = false
	ctrl._pause_layer.visible = false
	ctrl._menu_layer.visible = false
	ctrl._char_select_layer.visible = false
	ctrl._char_create_layer.visible = false
	ctrl._hub_layer.visible = true
	ctrl._lobby_panel.visible = false
	ctrl._inventory_layer.toolbar_panel.visible = true
	if not ctrl._is_cursor_always_visible_class():
		Input.set_mouse_mode(Input.MOUSE_MODE_CAPTURED)
	else:
		Input.set_mouse_mode(Input.MOUSE_MODE_VISIBLE)
	ctrl.hub_interact.near_portal = false
	ctrl._portal_prompt.visible = false
	ctrl.hub_interact.near_lift = false
	if ctrl._lift_prompt:
		ctrl._lift_prompt.visible = false

	# Load hub scene if not already loaded
	if ctrl.env_builder.current_env == null or ctrl.env_builder.current_env.name != "Hub":
		ctrl.env_builder.unload_environment()
		ctrl.env_builder.load_environment(ctrl.HUB_SCENE)

	# Despawn existing players
	ctrl.entity_mgr.despawn_all_players()

	# Spawn local player in hub (use saved position if returning player)
	var my_id: int = NetworkManager.get_my_id()
	if my_id > 0:
		var spawn_pos: Vector3 = ctrl.HUB_SPAWNS[0]
		if ctrl._has_saved_state and ctrl._saved_hub_position != Vector3.ZERO:
			spawn_pos = ctrl._saved_hub_position
		ctrl._has_saved_state = false
		print(
			(
				"[enter_hub] spawning id=%d at %s (saved=%s)"
				% [my_id, spawn_pos, ctrl._saved_hub_position]
			)
		)
		ctrl.entity_mgr.spawn_player(my_id, ctrl._local_class, spawn_pos, ctrl._local_spec)
		if ctrl._saved_hub_rot_y != 0.0:
			var player: CharacterBody3D = ctrl.entity_mgr.spawned_players.get(my_id)
			if player:
				player.rotation.y = ctrl._saved_hub_rot_y
			ctrl._saved_hub_rot_y = 0.0

	# Debug: camera state after spawn
	var cam3d: Camera3D = ctrl.get_viewport().get_camera_3d()
	var player_children: int = ctrl.entity_mgr._players_node.get_child_count()
	print(
		(
			"[enter_hub] spawn done: players_children=%d active_cam=%s cam_pos=%s"
			% [
				player_children,
				cam3d.name if cam3d else "NONE",
				str(cam3d.global_position) if cam3d else "N/A",
			]
		)
	)

	ctrl.hub_interact.update_hub_display()
	ctrl.group_mgr.update_group_panel()
	if ctrl._shared_hud:
		ctrl._shared_hud.on_enter_hub()
	if ctrl._map_overlay:
		ctrl._map_overlay.reset_floor()
	ctrl.env_builder.create_portal_trail()


# =============================================================================
# Arena warmup
# =============================================================================


func enter_arena_warmup() -> void:
	ctrl.state = ctrl.GameState.ARENA_LOBBY
	ctrl.get_tree().paused = false
	ctrl.paused = false
	ctrl._pause_layer.visible = false
	ctrl._menu_layer.visible = false
	ctrl.env_builder.remove_exit_portal()
	show_portal_prompt_only()
	ctrl._lobby_panel.visible = true
	if ctrl._shared_hud:
		ctrl._shared_hud.on_enter_arena()
	if ctrl._map_overlay:
		ctrl._map_overlay.set_floor("arena", "Arena")

	# Load arena scene if not already loaded
	if ctrl.env_builder.current_env == null or ctrl.env_builder.current_env.name != "Arena":
		ctrl.env_builder.unload_environment()
		ctrl.env_builder.load_environment(ctrl.ARENA_SCENE)


# =============================================================================
# Spawning
# =============================================================================


func spawn_multiplayer_players() -> void:
	var spawn_idx := 0
	for pid in NetworkManager.player_info:
		var info: Dictionary = NetworkManager.player_info[pid]
		var class_name_str: String = info["class_name"]
		if not ctrl.CLASS_SCENES.has(class_name_str):
			push_error("[Main] Unknown class: %s" % class_name_str)
			continue
		var spawn_pos: Vector3 = ctrl.PLAYER_SPAWNS[spawn_idx % ctrl.PLAYER_SPAWNS.size()]
		var spec: String = ctrl._local_spec if pid == NetworkManager.get_my_id() else ""
		ctrl.entity_mgr.spawn_player(pid, class_name_str, spawn_pos, spec)
		spawn_idx += 1

	if not ctrl._is_cursor_always_visible_class():
		Input.set_mouse_mode(Input.MOUSE_MODE_CAPTURED)
	else:
		Input.set_mouse_mode(Input.MOUSE_MODE_VISIBLE)


# =============================================================================
# Fight
# =============================================================================


func start_fight(time_limit_text := "") -> void:
	ctrl.state = ctrl.GameState.FIGHT
	show_portal_prompt_only()
	ctrl._lobby_panel.visible = false
	ctrl._cursor_toggled = false
	ctrl._alt_held = false
	ctrl._inventory_layer.equip_panel.visible = false
	ctrl._inventory_layer.bag_panel.visible = false
	ctrl._sync_toolbar_active()

	# Enemies are managed dynamically via update_enemies from world state
	CombatLog.start_fight()
	if ctrl._shared_hud:
		# Server sends the dungeon time limit (seconds) in the fight-start text.
		# Fall back to the default if absent so older flows still show a timer.
		var limit := float(time_limit_text) if time_limit_text.is_valid_float() else 300.0
		ctrl._shared_hud.set_time_limit(limit)
		ctrl._shared_hud.on_fight_start()


func on_boss_dead() -> void:
	ctrl.state = ctrl.GameState.FIGHT_OVER
	ctrl.env_builder.spawn_exit_portal()
	if ctrl._local_player_dead and ctrl._death_overlay_layer.visible:
		ctrl._respawn_btn.disabled = false
	CombatLog.end_fight("VICTORY")
	if ctrl._shared_hud:
		ctrl._shared_hud.on_fight_end()


func on_all_dead() -> void:
	ctrl.state = ctrl.GameState.FIGHT_OVER
	if ctrl._local_player_dead and ctrl._death_overlay_layer.visible:
		ctrl._respawn_btn.disabled = false
	CombatLog.end_fight("WIPE")
	if ctrl._shared_hud:
		# A wipe does not stop the clear timer: the run continues while players
		# respawn and run back, so the count-down keeps ticking toward OVERTIME.
		ctrl._shared_hud.on_wipe()
	# Show reset button for group leader on wipe
	var gdata: Dictionary = ctrl.group_mgr.group_data
	var leader_peer: int = gdata.get("leader_peer", 0)
	var my_id: int = NetworkManager.get_my_id()
	ctrl._death_overlay_layer.reset_instance_btn.visible = (
		leader_peer > 0 and leader_peer == my_id
	)


# =============================================================================
# Zone transfer
# =============================================================================


func on_zone_transfer(zone_type: int, _new_peer_id: int) -> void:
	var prev_state: int = ctrl.state
	var prev_zone: Node3D = ctrl.env_builder.current_env
	var prev_env_name: String = "null"
	if prev_zone and is_instance_valid(prev_zone):
		prev_env_name = prev_zone.name
	print(
		(
			"[ZoneTransfer] START type=%d peer=%d prev_state=%d prev_env=%s"
			% [
				zone_type,
				_new_peer_id,
				prev_state,
				prev_env_name,
			]
		)
	)
	hide_death_overlay()
	# Discard any saved hub position set by OP_CHARACTER_STATE that arrived
	# in the same packet batch. The server sends the player's current
	# (arena) position, not their last hub position.
	ctrl._has_saved_state = false
	ctrl.env_builder.remove_exit_portal()
	print("[ZoneTransfer] despawn players: %d alive" % ctrl.entity_mgr.spawned_players.size())
	ctrl.entity_mgr.despawn_all_players()
	ctrl.entity_mgr.clear_all_enemies()
	if ctrl._map_overlay:
		ctrl._map_overlay.visible = false
	ctrl._inventory_layer.equip_panel.visible = false
	ctrl._inventory_layer.bag_panel.visible = false
	ctrl._sync_toolbar_active()
	ctrl.entity_mgr.clear_all_npcs()

	if zone_type == NetSerializer.ZONE_TYPE_ARENA:
		print("[ZoneTransfer] -> ARENA: unloading %s" % prev_env_name)
		ctrl.env_builder.unload_environment()
		ctrl.env_builder.load_environment(ctrl.ARENA_SCENE)
		var new_env: Node3D = ctrl.env_builder.current_env
		print(
			(
				"[ZoneTransfer] -> ARENA: loaded %s valid=%s"
				% [
					new_env.name if new_env else "null",
					str(is_instance_valid(new_env)) if new_env else "false",
				]
			)
		)
		ctrl.group_mgr.pending_instance_zone = ""
		ctrl.state = ctrl.GameState.ARENA_LOBBY
		show_portal_prompt_only()
		ctrl._lobby_panel.visible = true
		ctrl._menu_layer.visible = false
		ctrl._char_select_layer.visible = false
		ctrl._char_create_layer.visible = false
		if ctrl._shared_hud:
			ctrl._shared_hud.on_enter_arena()
		if not ctrl._is_cursor_always_visible_class():
			Input.set_mouse_mode(Input.MOUSE_MODE_CAPTURED)
		else:
			Input.set_mouse_mode(Input.MOUSE_MODE_VISIBLE)
		var my_id: int = NetworkManager.get_my_id()
		print(
			(
				"[ZoneTransfer] -> ARENA: spawning local player id=%d class=%s"
				% [
					my_id,
					ctrl._local_class,
				]
			)
		)
		if my_id > 0:
			ctrl.entity_mgr.spawn_player(
				my_id, ctrl._local_class, ctrl.LOBBY_SPAWN, ctrl._local_spec
			)
			var spawned: bool = my_id in ctrl.entity_mgr.spawned_players
			print(
				(
					"[ZoneTransfer] -> ARENA: spawn result=%s players_node_children=%d"
					% [
						str(spawned),
						ctrl.entity_mgr._players_node.get_child_count(),
					]
				)
			)
		else:
			print("[ZoneTransfer] -> ARENA: WARNING my_id <= 0, no player spawned")
	else:
		print("[ZoneTransfer] -> HUB: delegating to enter_hub (handles env loading)")
		enter_hub()
	var final_env: Node3D = ctrl.env_builder.current_env
	var final_env_name: String = final_env.name if final_env else "null"
	var we_final: int = ctrl.env_builder._count_world_environments(ctrl.get_tree().root)
	var done_cam: Camera3D = ctrl.get_viewport().get_camera_3d()
	var done_cam_info := "NONE"
	if done_cam:
		done_cam_info = (
			"%s at %s (parent=%s)"
			% [
				done_cam.name,
				str(done_cam.global_position),
				done_cam.get_parent().name if done_cam.get_parent() else "?",
			]
		)
	# Dump visible CanvasLayers to catch UI covering the 3D view
	var visible_layers: Array[String] = []
	for child in ctrl.get_children():
		if child is CanvasLayer and child.visible:
			visible_layers.append(child.name)
	print("[ZoneTransfer] visible layers: %s" % ", ".join(visible_layers))
	print(
		(
			"[ZoneTransfer] DONE state=%d env=%s WorldEnvs=%d cam=%s mouse=%d"
			% [ctrl.state, final_env_name, we_final, done_cam_info, Input.get_mouse_mode()]
		)
	)

	# Re-spawn bots after zone transfer (they are zone-local on the server)
	ctrl.dev_mgr.respawn_bots_after_transfer()


# =============================================================================
# Server-authoritative game flow events
# =============================================================================


func on_game_flow_event(flow_type: int, _text: String) -> void:
	match flow_type:
		NetSerializer.FLOW_SPAWN_PLAYERS:
			spawn_multiplayer_players()
		NetSerializer.FLOW_FIGHT_START:
			start_fight(_text)
		NetSerializer.FLOW_BOSS_DEAD:
			on_boss_dead()
		NetSerializer.FLOW_ALL_DEAD:
			on_all_dead()
		NetSerializer.FLOW_RETURN_LOBBY:
			hide_death_overlay()
			ctrl.entity_mgr.clear_all_enemies()
			ctrl.entity_mgr.clear_all_npcs()
			enter_arena_warmup()
		NetSerializer.FLOW_BOSS_ACTIVATED:
			pass  # Gate close handled by FLOW_GATE_CLOSE
		NetSerializer.FLOW_BOSS_RESET:
			pass  # Gate open handled by FLOW_GATE_OPEN
		NetSerializer.FLOW_GATE_CLOSE:
			ctrl.env_builder.close_gate(_text)
		NetSerializer.FLOW_GATE_OPEN:
			ctrl.env_builder.open_gate(_text)


# =============================================================================
# Death overlay
# =============================================================================


func on_local_player_died() -> void:
	ctrl._local_player_dead = true
	ctrl._death_overlay_layer.visible = true
	ctrl._respawn_btn.disabled = (ctrl.state == ctrl.GameState.FIGHT)
	Input.set_mouse_mode(Input.MOUSE_MODE_VISIBLE)


func on_respawn() -> void:
	NetworkManager.send_respawn_request(0)
	hide_death_overlay()
	if not ctrl._is_cursor_always_visible_class():
		Input.set_mouse_mode(Input.MOUSE_MODE_CAPTURED)
	else:
		Input.set_mouse_mode(Input.MOUSE_MODE_VISIBLE)


func on_respawn_hub() -> void:
	NetworkManager.send_respawn_request(1)
	hide_death_overlay()


func hide_death_overlay() -> void:
	ctrl._local_player_dead = false
	ctrl._death_overlay_layer.visible = false
	ctrl._respawn_btn.disabled = true
	ctrl._death_overlay_layer.reset_instance_btn.visible = false
