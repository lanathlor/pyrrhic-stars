extends Camera3D
## Noclip free-fly camera for replay viewing.
## WASD + mouse-look (hold right-click). Q/E for vertical. Shift for speed.

@export var move_speed: float = 10.0
@export var fast_multiplier: float = 3.0
@export var sensitivity: float = 0.002

var _yaw: float = 0.0
var _pitch: float = 0.0
var _looking: bool = false


func _ready() -> void:
	current = true
	# Start facing -Z (toward the arena center)
	_yaw = 0.0
	_pitch = -0.3


func _unhandled_input(event: InputEvent) -> void:
	if event is InputEventMouseButton:
		if event.button_index == MOUSE_BUTTON_RIGHT:
			_looking = event.pressed
			Input.mouse_mode = (
				Input.MOUSE_MODE_CAPTURED if event.pressed else Input.MOUSE_MODE_VISIBLE
			)
		# Scroll wheel adjusts base speed
		if event.button_index == MOUSE_BUTTON_WHEEL_UP:
			move_speed = minf(move_speed * 1.2, 100.0)
		elif event.button_index == MOUSE_BUTTON_WHEEL_DOWN:
			move_speed = maxf(move_speed / 1.2, 1.0)

	if event is InputEventMouseMotion and _looking:
		_yaw -= event.relative.x * sensitivity
		_pitch = clampf(_pitch - event.relative.y * sensitivity, -PI / 2.0 * 0.95, PI / 2.0 * 0.95)
		transform.basis = Basis.from_euler(Vector3(_pitch, _yaw, 0.0))


func _process(delta: float) -> void:
	var dir := Vector3.ZERO
	if Input.is_action_pressed("move_forward"):
		dir.z -= 1.0
	if Input.is_action_pressed("move_backward"):
		dir.z += 1.0
	if Input.is_action_pressed("move_left"):
		dir.x -= 1.0
	if Input.is_action_pressed("move_right"):
		dir.x += 1.0
	if Input.is_key_pressed(KEY_Q):
		dir.y -= 1.0
	if Input.is_key_pressed(KEY_E):
		dir.y += 1.0

	if dir != Vector3.ZERO:
		var speed := move_speed * (fast_multiplier if Input.is_key_pressed(KEY_SHIFT) else 1.0)
		global_translate(global_transform.basis * dir.normalized() * speed * delta)
