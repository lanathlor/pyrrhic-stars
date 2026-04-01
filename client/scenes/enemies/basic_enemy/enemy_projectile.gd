extends Area3D

## Enemy ranged projectile — visual only. Server handles damage.
## Position is updated from WorldState projectile data.

@export var speed: float = 22.0
@export var lifetime: float = 5.0

var direction: Vector3 = Vector3.FORWARD
var damage: float = 15.0
var _timer: float = 0.0


func setup(dir: Vector3, dmg: float) -> void:
	direction = dir.normalized()
	damage = dmg
	if direction.length() > 0.1:
		look_at(global_position + direction, Vector3.UP)


func _physics_process(delta: float) -> void:
	# Client-side movement prediction — server is authoritative
	global_position += direction * speed * delta
	_timer += delta
	if _timer >= lifetime:
		queue_free()


func _on_body_entered(_body: Node3D) -> void:
	# Visual: destroy on contact. Server already handled the damage.
	queue_free()
