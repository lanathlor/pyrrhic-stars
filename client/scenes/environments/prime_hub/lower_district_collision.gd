@tool
extends StaticBody3D
## Wall collision for the lower street district perimeter.

func _ready() -> void:
	var shapes: Dictionary = {
		"WallNCol": Vector3(200, 44, 0.5),
		"WallSCol": Vector3(200, 44, 0.5),
		"WallWCol": Vector3(0.5, 44, 200),
		"WallECol": Vector3(0.5, 44, 200),
	}
	for node_name in shapes:
		var col := get_node_or_null(node_name) as CollisionShape3D
		if col and not col.shape:
			var box := BoxShape3D.new()
			box.size = shapes[node_name]
			col.shape = box
