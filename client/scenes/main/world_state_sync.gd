extends Node

## Processes world state ticks and damage events from the server.

var ctrl: Node


func _ready() -> void:
	ctrl = get_parent()


func on_world_state(data: Dictionary) -> void:
	if ctrl.state == ctrl.GameState.MENU:
		return

	var entity_mgr: Node = ctrl.entity_mgr
	var players_data: Array = data.get("players", [])
	var seen_peers: Dictionary = {}
	var my_id: int = NetworkManager.get_my_id()

	for p_data in players_data:
		var pid: int = p_data["peer_id"]
		seen_peers[pid] = true

		# Store player names
		var uname: String = p_data.get("username", "")
		if uname != "":
			ctrl._player_names[pid] = uname

		# Spawn remote player if needed
		if pid != my_id and pid not in entity_mgr.spawned_players:
			var cls: String = p_data.get("class_name", "gunner")
			var spec: String = p_data.get("spec_name", "")
			entity_mgr.spawn_player(pid, cls, p_data["pos"], spec)

		# Sync local spec from server (handles reconnect with persisted spec)
		if pid == my_id:
			var server_spec: String = p_data.get("spec_name", "")
			if server_spec != "" and server_spec != ctrl._local_spec:
				ctrl._local_spec = server_spec

		if pid not in entity_mgr.spawned_players:
			continue

		var player: CharacterBody3D = entity_mgr.spawned_players[pid]
		if not is_instance_valid(player):
			continue

		if pid == my_id and ctrl.state == ctrl.GameState.HUB:
			# Hub: client-authoritative movement, only snap on extreme desync
			var server_pos: Vector3 = p_data["pos"]
			if player.global_position.distance_to(server_pos) > 8.0:
				player.global_position = server_pos
			# Still apply server state for stamina/health sync
			if player.has_method("apply_server_state"):
				player.apply_server_state(p_data)
		elif player.has_method("apply_server_state"):
			player.apply_server_state(p_data)

		# Update overhead name for remote players in hub
		if ctrl.state == ctrl.GameState.HUB and pid != my_id:
			entity_mgr.update_overhead_name(player, pid)

	# Despawn players no longer in state
	var to_remove: Array = []
	for pid in entity_mgr.spawned_players:
		if pid not in seen_peers:
			to_remove.append(pid)
	for pid in to_remove:
		var player = entity_mgr.spawned_players[pid]
		if is_instance_valid(player):
			player.queue_free()
		entity_mgr.spawned_players.erase(pid)

	# Enemies — dynamically spawn/update/despawn from server state
	var enemies_data: Array = data.get("enemies", [])
	entity_mgr.update_enemies(enemies_data)

	# NPCs — hub ambient characters
	var npcs_data: Array = data.get("npcs", [])
	entity_mgr.update_npcs(npcs_data)

	# Projectiles: spawn/update/remove
	var stale: Array = []
	for pid in entity_mgr.spawned_projectiles:
		if not is_instance_valid(entity_mgr.spawned_projectiles[pid]):
			stale.append(pid)
	for pid in stale:
		entity_mgr.spawned_projectiles.erase(pid)

	var proj_data: Array = data.get("projectiles", [])
	var active_ids: Dictionary = {}
	for p in proj_data:
		var pid: int = p["proj_id"]
		active_ids[pid] = true
		if pid not in entity_mgr.spawned_projectiles:
			entity_mgr.spawn_projectile(
				pid,
				p["pos"],
				p["direction"],
				p.get("speed", 22.0),
				p.get("angular_velocity", 0.0),
				p.get("visual_tag", "")
			)
		else:
			entity_mgr.spawned_projectiles[pid].global_position = p["pos"]
	var proj_to_remove: Array = []
	for pid in entity_mgr.spawned_projectiles:
		if pid not in active_ids:
			proj_to_remove.append(pid)
	for pid in proj_to_remove:
		var proj = entity_mgr.spawned_projectiles[pid]
		if is_instance_valid(proj):
			proj.queue_free()
		entity_mgr.spawned_projectiles.erase(pid)

	# Feed shared HUD
	if ctrl._shared_hud:
		ctrl._shared_hud.update_world_state(data)

	# Feed map overlay (only when visible to avoid unnecessary work)
	if ctrl._map_overlay and ctrl._map_overlay.visible:
		var local_pos := Vector3.ZERO
		var local_rot := 0.0
		if (
			my_id in entity_mgr.spawned_players
			and is_instance_valid(entity_mgr.spawned_players[my_id])
		):
			local_pos = entity_mgr.spawned_players[my_id].global_position
			local_rot = entity_mgr.spawned_players[my_id].rotation.y
		(
			ctrl
			. _map_overlay
			. update_state(
				{
					"player_pos": local_pos,
					"player_rot_y": local_rot,
					"players": ctrl._shared_hud._world_players,
					"npcs": ctrl._shared_hud._npc_positions,
					"enemies": ctrl._shared_hud._enemy_positions,
				}
			)
		)
		if ctrl._portal_trail and local_pos != Vector3.ZERO:
			ctrl._map_overlay.set_waypoint_path(ctrl._portal_trail.get_path_to_target(local_pos))


func on_damage_event(data: Dictionary) -> void:
	var entity_mgr: Node = ctrl.entity_mgr
	var target_peer: int = data.get("target_peer_id", -1)
	var source_peer: int = data.get("source_peer_id", 0)
	var amount: float = data.get("amount", 0.0)
	var hit_pos: Vector3 = data.get("hit_pos", Vector3.ZERO)
	var source_type: int = data.get("source_type", 0)
	if target_peer >= 1000:
		# Player hit an enemy (enemy IDs are >= 1000)
		if target_peer in entity_mgr.enemy_nodes:
			var enode: CharacterBody3D = entity_mgr.enemy_nodes[target_peer]
			if is_instance_valid(enode) and enode.has_method("on_damage_visual"):
				enode.on_damage_visual(amount, hit_pos)
		# Server-confirmed hit marker on the attacker's HUD
		if source_peer == NetworkManager.get_my_id():
			var local_player: CharacterBody3D = entity_mgr.spawned_players.get(source_peer)
			if is_instance_valid(local_player) and local_player.has_method("on_hit_confirmed"):
				local_player.on_hit_confirmed(amount, hit_pos)
		# Floating damage number
		spawn_damage_number(amount, hit_pos)
	elif target_peer in entity_mgr.spawned_players:
		var player: CharacterBody3D = entity_mgr.spawned_players[target_peer]
		if is_instance_valid(player) and player.has_method("on_damage_visual"):
			player.on_damage_visual(amount, hit_pos)

	# Feed shared HUD damage meter
	if ctrl._shared_hud:
		ctrl._shared_hud.on_damage_event(data)


func spawn_damage_number(amount: float, world_pos: Vector3) -> void:
	var label := Label3D.new()
	label.text = str(int(amount))
	label.font_size = 48
	label.outline_size = 8
	label.modulate = Color(1.0, 0.95, 0.3, 1.0)
	label.outline_modulate = Color(0.0, 0.0, 0.0, 0.8)
	label.billboard = BaseMaterial3D.BILLBOARD_ENABLED
	label.no_depth_test = true
	label.pixel_size = 0.005
	# Slight random offset so stacked hits don't overlap
	var offset := Vector3(randf_range(-0.3, 0.3), randf_range(0.0, 0.3), randf_range(-0.3, 0.3))
	label.position = world_pos + offset + Vector3(0.0, 0.5, 0.0)
	ctrl.add_child(label)

	var tween: Tween = ctrl.create_tween()
	tween.set_parallel(true)
	(
		tween
		. tween_property(label, "position:y", label.position.y + 1.5, 0.8)
		. set_ease(Tween.EASE_OUT)
		. set_trans(Tween.TRANS_QUAD)
	)
	tween.tween_property(label, "modulate:a", 0.0, 0.8).set_delay(0.3)
	tween.tween_property(label, "outline_modulate:a", 0.0, 0.8).set_delay(0.3)
	tween.chain().tween_callback(label.queue_free)
