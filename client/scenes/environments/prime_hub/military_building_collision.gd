@tool
extends StaticBody3D
## Generates collision shapes for the Stark Tower building.
## Shaft is BEHIND the back wall. Positions match CSG exactly.

func _ready() -> void:
	_generate_wall_shapes()


func _generate_wall_shapes() -> void:
	var wall_shapes: Dictionary = {
		# Tower exterior walls
		"TowerWallN": Vector3(24, 130, 0.5),
		"TowerWallSL": Vector3(10, 130, 0.4),
		"TowerWallSR": Vector3(10, 130, 0.4),
		"TowerWallSMid": Vector3(4, 97, 0.4),
		"TowerWallSTop": Vector3(4, 27, 0.4),
		"TowerWallW": Vector3(0.5, 130, 22),
		"TowerWallEBelow": Vector3(0.5, 100, 22),
		"TowerWallEAbove": Vector3(0.5, 26, 22),
		# GF floor & ceiling — one piece each
		"GFFloor": Vector3(24, 0.2, 22),
		"GFCeiling": Vector3(24, 0.3, 22),
		# Shaft walls (behind back wall)
		"ShaftWallW": Vector3(0.3, 130, 4),
		"ShaftWallE": Vector3(0.3, 130, 4),
		"ShaftWallBack": Vector3(4, 130, 0.3),
		# Ops floor & ceiling — one piece each
		"OpsFloor": Vector3(24, 0.3, 22),
		"OpsCeiling": Vector3(24, 0.3, 22),
		# Ops partition
		"OpsPartL": Vector3(5, 5, 0.3),
		"OpsPartC": Vector3(2, 5, 0.3),
		"OpsPartR": Vector3(5, 5, 0.3),
		# Room divider
		"OpsDivider": Vector3(0.3, 5, 10),
	}

	for node_name in wall_shapes:
		var col_node := get_node_or_null(node_name) as CollisionShape3D
		if col_node and not col_node.shape:
			var box := BoxShape3D.new()
			box.size = wall_shapes[node_name]
			col_node.shape = box
