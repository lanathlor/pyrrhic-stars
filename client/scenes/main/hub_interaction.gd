extends Node

## Hub proximity checks: portal, lift, aim-at-player for invite.

var ctrl: Node

var near_portal: bool = false
var near_lift: bool = false
var aimed_peer_id: int = 0


func _ready() -> void:
	ctrl = get_parent()


func check_portal_proximity() -> void:
	var entity_mgr: Node = ctrl.entity_mgr
	var my_id: int = NetworkManager.get_my_id()
	if my_id not in entity_mgr.spawned_players:
		near_portal = false
		if ctrl._portal_prompt:
			ctrl._portal_prompt.visible = false
		return
	var player: CharacterBody3D = entity_mgr.spawned_players[my_id]
	if not is_instance_valid(player):
		return
	# Use the actual PortalArea node position
	var current_env: Node3D = ctrl.env_builder.current_env
	var portal_area: Node3D = current_env.get_node_or_null("PortalArea") if current_env else null
	if not portal_area:
		near_portal = false
		if ctrl._portal_prompt:
			ctrl._portal_prompt.visible = false
		return
	var dist: float = player.global_position.distance_to(portal_area.global_position)
	near_portal = dist < 4.0
	if ctrl._portal_prompt:
		ctrl._portal_prompt.visible = near_portal
		if near_portal:
			var target: String = portal_area.get_meta("target_zone", "Arena")
			ctrl._portal_prompt.text = "Press [E] to enter %s" % target.capitalize()


func check_lift_proximity() -> void:
	var entity_mgr: Node = ctrl.entity_mgr
	var my_id: int = NetworkManager.get_my_id()
	if my_id not in entity_mgr.spawned_players:
		near_lift = false
		if ctrl._lift_prompt:
			ctrl._lift_prompt.visible = false
		return
	var player: CharacterBody3D = entity_mgr.spawned_players[my_id]
	if not is_instance_valid(player):
		return

	var pos: Vector3 = player.global_position
	var found_lift: Node = null
	var current_env: Node3D = ctrl.env_builder.current_env

	# Check interior elevator (ElevatorCab)
	var elev: Node = current_env.get_node_or_null("ElevatorCab") if current_env else null
	if elev and elev.is_idle():
		var elev_pos: Vector3 = (elev as Node3D).global_position
		var door_z: float = elev_pos.z - 2.0
		var in_x: bool = absf(pos.x - elev_pos.x) < 2.5
		var near_door: bool = in_x and absf(pos.z - door_z) < 3.0
		var bottom_y: float = elev.get("BOTTOM_Y")
		var top_y: float = elev.get("TOP_Y")
		if near_door and (pos.y < bottom_y + 5.0 or pos.y > top_y - 5.0):
			found_lift = elev

	# Check public lift — detect at BOTH stations (top and bottom)
	if not found_lift:
		var plift: Node = current_env.get_node_or_null("Plaza/PublicLift") if current_env else null
		if plift and plift.is_idle():
			var station_x := 5.0
			var station_z := -55.0
			var dist_xz: float = Vector2(pos.x - station_x, pos.z - station_z).length()
			if dist_xz < 6.0:
				# Near top station (Y around 0)
				var near_top: bool = pos.y > -5.0 and pos.y < 5.0
				# Near bottom station (Y around -200)
				var near_bottom: bool = pos.y > -205.0 and pos.y < -195.0
				if near_top or near_bottom:
					found_lift = plift

	near_lift = found_lift != null
	if ctrl._lift_prompt:
		if found_lift:
			var plift_pos_y: float = (found_lift as Node3D).global_position.y
			var lift_here: bool = absf(pos.y - plift_pos_y) < 5.0
			if lift_here:
				ctrl._lift_prompt.text = "Press [E] — %s" % found_lift.get_floor_label()
			else:
				ctrl._lift_prompt.text = "Press [E] — Call lift"
		ctrl._lift_prompt.visible = near_lift


func interact_lift() -> void:
	var entity_mgr: Node = ctrl.entity_mgr
	var pos := Vector3.ZERO
	var my_id: int = NetworkManager.get_my_id()
	if my_id in entity_mgr.spawned_players:
		pos = entity_mgr.spawned_players[my_id].global_position

	var current_env: Node3D = ctrl.env_builder.current_env

	# Try interior elevator first
	var elev: Node = current_env.get_node_or_null("ElevatorCab") if current_env else null
	if elev and elev.is_idle():
		var elev_pos: Vector3 = (elev as Node3D).global_position
		var door_z: float = elev_pos.z - 2.0
		if absf(pos.x - elev_pos.x) < 2.5 and absf(pos.z - door_z) < 3.0:
			elev.activate()
			return

	# Try public lift — fixed stations at top (Y=0) and bottom (Y=-200)
	var plift: Node = current_env.get_node_or_null("Plaza/PublicLift") if current_env else null
	if plift and plift.is_idle():
		var dist_xz: float = Vector2(pos.x - 5.0, pos.z - (-55.0)).length()
		var near_top: bool = pos.y > -5.0 and pos.y < 5.0
		var near_bottom: bool = pos.y > -205.0 and pos.y < -195.0
		if dist_xz < 6.0 and (near_top or near_bottom):
			plift.activate()
			return


func check_aim_at_player() -> void:
	var entity_mgr: Node = ctrl.entity_mgr
	aimed_peer_id = 0
	var my_id: int = NetworkManager.get_my_id()
	if my_id not in entity_mgr.spawned_players:
		return
	var local_player: CharacterBody3D = entity_mgr.spawned_players[my_id]
	if not is_instance_valid(local_player):
		return
	# Get camera
	var camera: Camera3D = ctrl.get_viewport().get_camera_3d()
	if not camera:
		return
	var from: Vector3 = camera.global_position
	var forward: Vector3 = -camera.global_transform.basis.z

	# Simple distance-based check against remote players
	var best_dist := 3.0  # max aim distance
	for pid in entity_mgr.spawned_players:
		if pid == my_id:
			continue
		var p: CharacterBody3D = entity_mgr.spawned_players[pid]
		if not is_instance_valid(p):
			continue
		# Project point onto ray
		var to_player: Vector3 = p.global_position - from
		var dot: float = to_player.dot(forward)
		if dot < 0 or dot > 15.0:
			continue
		var closest_on_ray: Vector3 = from + forward * dot
		var dist: float = closest_on_ray.distance_to(p.global_position + Vector3(0, 1, 0))
		if dist < best_dist:
			best_dist = dist
			aimed_peer_id = pid

	# Update hub status with aim info
	update_hub_display()


func update_hub_display() -> void:
	if ctrl._hub_class_label:
		ctrl._hub_class_label.text = "Class: %s" % ctrl._local_class.to_upper()

	if ctrl._hub_status_label:
		if aimed_peer_id > 0 and not near_portal:
			var uname: String = ctrl._player_names.get(aimed_peer_id, "Player_%d" % aimed_peer_id)
			ctrl._hub_status_label.text = "Press [E] to invite %s" % uname
		elif not near_portal:
			if ctrl._group_data.get("group_id", 0) > 0:
				ctrl._hub_status_label.text = "In group - Walk to portal | [G] Leave group"
			else:
				ctrl._hub_status_label.text = "[G] Create group | Aim at player + [E] to invite"
