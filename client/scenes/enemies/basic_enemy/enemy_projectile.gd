extends Area3D

## Enemy ranged projectile — moves in a straight line, damages players on contact.

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
	global_position += direction * speed * delta
	_timer += delta
	if _timer >= lifetime:
		queue_free()


func _on_body_entered(body: Node3D) -> void:
	if body.has_method("take_damage"):
		body.take_damage(damage, global_position)
		CombatLog.log_boss_hit("ranged", damage, body.name, body.global_position)
	queue_free()
