extends Node

## Arcanotechnicien WoW-style click targeting system.

var ctrl: Node


func _ready() -> void:
	ctrl = get_parent()


func try_click_target(screen_pos: Vector2) -> void:
	# Check HUD party frames first (UI priority over world)
	if ctrl.hud and ctrl.hud.has_method("get_clicked_target"):
		var party_pid: int = ctrl.hud.get_clicked_target(screen_pos)
		if party_pid > 0:
			select_target_by_peer_id(party_pid)
			return

	# Raycast into 3D world: mask 6 = layer 2 (Player) | layer 3 (Enemy)
	var from: Vector3 = ctrl.camera.project_ray_origin(screen_pos)
	var dir: Vector3 = ctrl.camera.project_ray_normal(screen_pos)
	var to: Vector3 = from + dir * 100.0
	var space: PhysicsDirectSpaceState3D = ctrl.get_world_3d().direct_space_state
	if not space:
		return
	var query := PhysicsRayQueryParameters3D.create(from, to, 6)
	query.exclude = [ctrl.get_rid()]
	var result: Dictionary = space.intersect_ray(query)
	if result:
		var hit_node: Node3D = result.collider
		# Walk up to find a node with peer_id (player or enemy)
		while hit_node and not ("peer_id" in hit_node):
			hit_node = hit_node.get_parent()
		if hit_node and "peer_id" in hit_node and hit_node != ctrl:
			select_target(hit_node)
			return
	# Clicked empty space -- clear selection
	clear_selection()


func select_target(target: Node3D) -> void:
	ctrl._selected_target = target
	ctrl.hud.show_selected_target(target, ctrl.camera)


func select_target_by_peer_id(pid: int) -> void:
	for player in GameManager.players:
		if is_instance_valid(player) and player.visible and "peer_id" in player:
			if player.peer_id == pid and player != ctrl:
				select_target(player)
				return
	for enemy in GameManager.enemies:
		if is_instance_valid(enemy) and enemy.visible and "peer_id" in enemy:
			if enemy.peer_id == pid:
				select_target(enemy)
				return


func clear_selection() -> void:
	ctrl._selected_target = null
	ctrl.hud.hide_selected_target()
