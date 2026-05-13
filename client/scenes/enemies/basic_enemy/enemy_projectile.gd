extends Area3D

## Enemy ranged projectile — visual only. Server handles damage.
## Position is updated from WorldState projectile data.
## Supports curved motion via angular velocity for bullet-hell patterns.

@export var lifetime: float = 5.0

var direction: Vector3 = Vector3.FORWARD
var speed: float = 22.0
var angular_velocity: float = 0.0
var visual_tag: String = ""
var _timer: float = 0.0


func setup(dir: Vector3, spd: float, ang_vel: float = 0.0, tag: String = "") -> void:
	direction = dir.normalized()
	speed = spd
	angular_velocity = ang_vel
	visual_tag = tag
	if direction.length() > 0.1:
		look_at(global_position + direction, Vector3.UP)


func _physics_process(delta: float) -> void:
	# Angular velocity: rotate direction around Y axis (curved projectiles)
	if angular_velocity != 0.0:
		direction = direction.rotated(Vector3.UP, angular_velocity * delta)

	# Client-side movement prediction — server is authoritative
	global_position += direction * speed * delta

	# Orient mesh along flight direction
	if direction.length() > 0.1:
		look_at(global_position + direction, Vector3.UP)

	_timer += delta
	if _timer >= lifetime:
		queue_free()
