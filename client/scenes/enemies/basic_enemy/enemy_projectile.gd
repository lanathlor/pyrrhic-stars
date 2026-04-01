extends Area3D

## Enemy ranged projectile — moves in a straight line, damages players on contact.

@export var speed: float = 22.0
@export var lifetime: float = 5.0

var direction: Vector3 = Vector3.FORWARD
var damage: float = 15.0
var _timer: float = 0.0
var _is_host: bool = true


func setup(dir: Vector3, dmg: float) -> void:
	direction = dir.normalized()
	damage = dmg
	_is_host = not NetworkManager.is_active or NetworkManager.is_host
	if direction.length() > 0.1:
		look_at(global_position + direction, Vector3.UP)


func _physics_process(delta: float) -> void:
	global_position += direction * speed * delta
	_timer += delta
	if _timer >= lifetime:
		queue_free()


func _on_body_entered(body: Node3D) -> void:
	if _is_host and body.has_method("take_damage"):
		if NetworkManager.is_active:
			var target_peer: int = body.peer_id if "peer_id" in body else 0
			NetworkManager.send_msg(NetSerializer.OP_DAMAGE,
				NetSerializer.encode_damage(target_peer, damage, global_position))
		else:
			body.take_damage(damage, global_position)
		CombatLog.log_boss_hit("ranged", damage, body.name, body.global_position)
	queue_free()
