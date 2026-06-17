extends Node

## Tracks all players and enemies in the scene for AI targeting.

var players: Array[CharacterBody3D] = []
var enemies: Array[CharacterBody3D] = []


## True when a text field (e.g. the social panel inputs) has keyboard focus.
## Controllers consult this so typing does not leak into gameplay actions, since
## polled input (Input.get_vector / is_action_pressed) ignores GUI focus.
func text_input_active() -> bool:
	var focus := get_viewport().gui_get_focus_owner()
	return focus is LineEdit or focus is TextEdit


## Movement vector for the four directional actions, zeroed while typing.
func move_vector() -> Vector2:
	if text_input_active():
		return Vector2.ZERO
	return Input.get_vector("move_left", "move_right", "move_forward", "move_backward")


func register_player(player: CharacterBody3D) -> void:
	players.append(player)


func unregister_player(player: CharacterBody3D) -> void:
	players.erase(player)


func register_enemy(enemy: CharacterBody3D) -> void:
	enemies.append(enemy)


func unregister_enemy(enemy: CharacterBody3D) -> void:
	enemies.erase(enemy)


func get_nearest_player(from_position: Vector3) -> CharacterBody3D:
	var nearest: CharacterBody3D = null
	var nearest_dist: float = INF
	for player in players:
		if not is_instance_valid(player) or not player.visible:
			continue
		var dist: float = from_position.distance_squared_to(player.global_position)
		if dist < nearest_dist:
			nearest_dist = dist
			nearest = player
	return nearest


func get_farthest_player(from_position: Vector3) -> CharacterBody3D:
	var farthest: CharacterBody3D = null
	var farthest_dist: float = -1.0
	for player in players:
		if not is_instance_valid(player) or not player.visible:
			continue
		var dist: float = from_position.distance_squared_to(player.global_position)
		if dist > farthest_dist:
			farthest_dist = dist
			farthest = player
	return farthest
