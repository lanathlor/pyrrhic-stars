extends AnimatableBody3D
## Open-air public lift platform.
## Railings on 3 sides permanent, west railing deploys when moving.

enum State { IDLE_TOP, MOVING_DOWN, IDLE_BOTTOM, MOVING_UP }

const TOP_Y := 0.0
const BOTTOM_Y := -200.0
const TRAVEL_TIME := 20.0

var _state: State = State.IDLE_BOTTOM
var _progress: float = 1.0  # 0=top, 1=bottom

@onready var _rail_west_col: CollisionShape3D = $RailWestCol
@onready var _rail_west_vis: Node3D = $RailWestVis


func _ready() -> void:
	_set_west_rail(false)


func activate() -> void:
	if _state == State.IDLE_TOP:
		_state = State.MOVING_DOWN
		_set_west_rail(true)
	elif _state == State.IDLE_BOTTOM:
		_state = State.MOVING_UP
		_set_west_rail(true)


func is_idle() -> bool:
	return _state == State.IDLE_TOP or _state == State.IDLE_BOTTOM


func get_floor_label() -> String:
	match _state:
		State.IDLE_TOP:
			return "Go down to streets"
		State.IDLE_BOTTOM:
			return "Go up to plaza"
		State.MOVING_DOWN:
			return "Descending..."
		State.MOVING_UP:
			return "Ascending..."
	return ""


func _physics_process(delta: float) -> void:
	match _state:
		State.MOVING_DOWN:
			_progress = minf(_progress + delta / TRAVEL_TIME, 1.0)
			if _progress >= 1.0:
				_state = State.IDLE_BOTTOM
				_set_west_rail(false)
			else:
				_push_players_from_destination(delta)
		State.MOVING_UP:
			_progress = maxf(_progress - delta / TRAVEL_TIME, 0.0)
			if _progress <= 0.0:
				_state = State.IDLE_TOP
				_set_west_rail(false)
			else:
				_push_players_from_destination(delta)

	var t := _progress * _progress * (3.0 - 2.0 * _progress)
	position.y = lerpf(TOP_Y, BOTTOM_Y, t)


func _push_players_from_destination(_delta: float) -> void:
	# Push the LOCAL player out of the shaft zone at the destination floor.
	# Only affects the local player — no authority over remote players.
	var dest_y: float
	if _state == State.MOVING_DOWN:
		dest_y = BOTTOM_Y
	else:
		dest_y = TOP_Y

	var my_id := NetworkManager.get_my_id()
	for node in get_tree().get_nodes_in_group("players"):
		if not (node is CharacterBody3D):
			continue
		var p: CharacterBody3D = node
		if p.peer_id != my_id:
			continue
		var pos := p.global_position
		var dx := pos.x - global_position.x
		var dz := pos.z - global_position.z
		var dist_xz := sqrt(dx * dx + dz * dz)
		if dist_xz > 4.5:
			continue
		if absf(pos.y - dest_y) > 3.0:
			continue
		var push_dir := Vector3(dx, 0, dz)
		if push_dir.length() < 0.1:
			push_dir = Vector3(-1, 0, 0)
		push_dir = push_dir.normalized()
		# Set velocity directly (not accumulate) — just enough to walk out
		var push_speed := 5.0
		p.velocity.x = push_dir.x * push_speed
		p.velocity.z = push_dir.z * push_speed


func _set_west_rail(deployed: bool) -> void:
	if _rail_west_col:
		_rail_west_col.disabled = not deployed
	if _rail_west_vis:
		_rail_west_vis.visible = deployed
