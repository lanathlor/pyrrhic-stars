extends AnimatableBody3D
## Enclosed elevator cab with sliding doors at ground and upper floors.
## Doors close → cab travels → doors open at destination.

enum State {
	IDLE_BOTTOM,
	CLOSING_BOTTOM,
	TRAVELING_UP,
	OPENING_TOP,
	IDLE_TOP,
	CLOSING_TOP,
	TRAVELING_DOWN,
	OPENING_BOTTOM,
}

const BOTTOM_Y := 0.0
const TOP_Y := 100.0
const TRAVEL_TIME := 8.0
const DOOR_ANIM_TIME := 0.6
const DOOR_OPEN_X := 2.0

var _state: State = State.IDLE_BOTTOM
var _travel_t: float = 0.0  # 0=bottom, 1=top
var _door_t: float = 1.0    # 0=closed, 1=open

var _door_gf_l: AnimatableBody3D
var _door_gf_r: AnimatableBody3D
var _door_uf_l: AnimatableBody3D
var _door_uf_r: AnimatableBody3D
var _cab_front_seal: CollisionShape3D


func _ready() -> void:
	_door_gf_l = get_parent().get_node("DoorGF_L")
	_door_gf_r = get_parent().get_node("DoorGF_R")
	_door_uf_l = get_parent().get_node("DoorUF_L")
	_door_uf_r = get_parent().get_node("DoorUF_R")
	_cab_front_seal = get_node("CabFrontSeal")

	# Start: ground floor, GF doors open, UF doors closed
	_apply_door_gf(1.0)
	_apply_door_uf(0.0)
	_cab_front_seal.disabled = true


func activate() -> void:
	if _state == State.IDLE_BOTTOM:
		_state = State.CLOSING_BOTTOM
		_door_t = 1.0
	elif _state == State.IDLE_TOP:
		_state = State.CLOSING_TOP
		_door_t = 1.0


func is_idle() -> bool:
	return _state == State.IDLE_BOTTOM or _state == State.IDLE_TOP


func is_at_bottom() -> bool:
	return _state == State.IDLE_BOTTOM


func is_at_top() -> bool:
	return _state == State.IDLE_TOP


func get_floor_label() -> String:
	match _state:
		State.IDLE_BOTTOM:
			return "Go up"
		State.IDLE_TOP:
			return "Go down"
		State.CLOSING_BOTTOM, State.TRAVELING_UP, State.OPENING_TOP:
			return "Going up..."
		State.CLOSING_TOP, State.TRAVELING_DOWN, State.OPENING_BOTTOM:
			return "Going down..."
	return ""


func _physics_process(delta: float) -> void:
	match _state:
		State.CLOSING_BOTTOM:
			_door_t = maxf(_door_t - delta / DOOR_ANIM_TIME, 0.0)
			_apply_door_gf(_door_t)
			if _door_t <= 0.0:
				_state = State.TRAVELING_UP
				_cab_front_seal.disabled = false

		State.TRAVELING_UP:
			_travel_t = minf(_travel_t + delta / TRAVEL_TIME, 1.0)
			position.y = lerpf(BOTTOM_Y, TOP_Y, _smoothstep(_travel_t))
			if _travel_t >= 1.0:
				_state = State.OPENING_TOP
				_door_t = 0.0
				_cab_front_seal.disabled = true

		State.OPENING_TOP:
			_door_t = minf(_door_t + delta / DOOR_ANIM_TIME, 1.0)
			_apply_door_uf(_door_t)
			if _door_t >= 1.0:
				_state = State.IDLE_TOP

		State.CLOSING_TOP:
			_door_t = maxf(_door_t - delta / DOOR_ANIM_TIME, 0.0)
			_apply_door_uf(_door_t)
			if _door_t <= 0.0:
				_state = State.TRAVELING_DOWN
				_cab_front_seal.disabled = false

		State.TRAVELING_DOWN:
			_travel_t = maxf(_travel_t - delta / TRAVEL_TIME, 0.0)
			position.y = lerpf(BOTTOM_Y, TOP_Y, _smoothstep(_travel_t))
			if _travel_t <= 0.0:
				_state = State.OPENING_BOTTOM
				_door_t = 0.0
				_cab_front_seal.disabled = true

		State.OPENING_BOTTOM:
			_door_t = minf(_door_t + delta / DOOR_ANIM_TIME, 1.0)
			_apply_door_gf(_door_t)
			if _door_t >= 1.0:
				_state = State.IDLE_BOTTOM


func _apply_door_gf(open_fraction: float) -> void:
	var slide := open_fraction * DOOR_OPEN_X
	_door_gf_l.position.x = -slide
	_door_gf_r.position.x = slide


func _apply_door_uf(open_fraction: float) -> void:
	var slide := open_fraction * DOOR_OPEN_X
	_door_uf_l.position.x = -slide
	_door_uf_r.position.x = slide


func _smoothstep(t: float) -> float:
	return t * t * (3.0 - 2.0 * t)
