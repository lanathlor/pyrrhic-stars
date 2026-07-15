extends Area3D

## Enemy ranged projectile — visual only. Server handles damage.
## Position is updated from WorldState projectile data.
## Supports curved motion via angular velocity for bullet-hell patterns.

## Per-ability projectile tint so overlapping bullet-hell patterns stay
## readable. Unknown/empty tags keep the default orange from the scene.
const TAG_COLORS := {
	"spiral_lattice": Color(0.4, 0.6, 1.0),
	"aimed_lattice": Color(1.0, 0.85, 0.25),
	"gravity_well": Color(0.7, 0.3, 1.0),
	"close_barrage": Color(1.0, 0.35, 0.55),
}

# Materials cached per tag: bullet-hell spawns hundreds of projectiles, so
# tinted materials are shared instead of duplicated per instance.
static var _tag_materials: Dictionary = {}

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
	_apply_tag_color()
	if direction.length() > 0.1:
		look_at(global_position + direction, Vector3.UP)


func _apply_tag_color() -> void:
	if not TAG_COLORS.has(visual_tag):
		return
	var color: Color = TAG_COLORS[visual_tag]
	var mesh: MeshInstance3D = $Mesh
	if not _tag_materials.has(visual_tag):
		var mat: StandardMaterial3D = mesh.get_surface_override_material(0).duplicate()
		mat.albedo_color = color
		mat.emission = color
		_tag_materials[visual_tag] = mat
	mesh.set_surface_override_material(0, _tag_materials[visual_tag])
	($Light as OmniLight3D).light_color = color


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
