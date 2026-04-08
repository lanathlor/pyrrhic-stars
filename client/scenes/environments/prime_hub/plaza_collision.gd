@tool
extends StaticBody3D
## Generates collision shapes for the outdoor plaza.

func _ready() -> void:
	var shapes: Dictionary = {
		"GroundShape": Vector3(250, 0.2, 272),
		"EdgeNShape": Vector3(250, 3, 0.5),
		"EdgeSShape": Vector3(250, 3, 0.5),
		"EdgeWShape": Vector3(0.5, 3, 272),
		"EdgeEShape": Vector3(0.5, 3, 272),
	}

	for node_name in shapes:
		var col := get_node_or_null(node_name) as CollisionShape3D
		if col and not col.shape:
			var box := BoxShape3D.new()
			box.size = shapes[node_name]
			col.shape = box
