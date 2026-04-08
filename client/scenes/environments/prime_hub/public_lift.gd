extends AnimatableBody3D
## Open-air public lift platform.
## Railings on 3 sides permanent, west railing deploys when moving.

enum State { IDLE_TOP, MOVING_DOWN, IDLE_BOTTOM, MOVING_UP }

const TOP_Y := 0.0
const BOTTOM_Y := -150.0
const TRAVEL_TIME := 20.0

var _state: State = State.IDLE_TOP
var _progress: float = 0.0  # 0=top, 1=bottom

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
		State.MOVING_UP:
			_progress = maxf(_progress - delta / TRAVEL_TIME, 0.0)
			if _progress <= 0.0:
				_state = State.IDLE_TOP
				_set_west_rail(false)

	var t := _progress * _progress * (3.0 - 2.0 * _progress)
	position.y = lerpf(TOP_Y, BOTTOM_Y, t)


func _set_west_rail(deployed: bool) -> void:
	if _rail_west_col:
		_rail_west_col.disabled = not deployed
	if _rail_west_vis:
		_rail_west_vis.visible = deployed
