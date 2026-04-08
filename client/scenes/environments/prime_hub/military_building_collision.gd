@tool
extends StaticBody3D
## Generates collision shapes for the Stark Tower building.
## Shaft is BEHIND the back wall. Positions match CSG exactly.

func _ready() -> void:
	_generate_wall_shapes()


func _generate_wall_shapes() -> void:
	var wall_shapes: Dictionary = {
		# Tower exterior walls
		"TowerWallNL": Vector3(16, 130, 0.5),
		"TowerWallNR": Vector3(16, 130, 0.5),
		"TowerWallSL": Vector3(23, 130, 0.4),
		"TowerWallSR": Vector3(23, 130, 0.4),
		"TowerWallSMid": Vector3(4, 97, 0.4),
		"TowerWallSTop": Vector3(4, 27, 0.4),
		"TowerWallW": Vector3(0.5, 130, 44),
		"TowerWallEBelow": Vector3(0.5, 100, 44),
		"TowerWallEAbove": Vector3(0.5, 26, 44),
		# GF floor & ceiling — one piece each
		"GFFloor": Vector3(50, 0.2, 44),
		"GFCeiling": Vector3(50, 0.3, 44),
		# Shaft walls (behind back wall)
		"ShaftWallW": Vector3(0.3, 130, 4),
		"ShaftWallE": Vector3(0.3, 130, 4),
		"ShaftWallBack": Vector3(4, 130, 0.3),
		# Ops floor & ceiling — one piece each
		"OpsFloor": Vector3(50, 0.3, 44),
		"OpsCeiling": Vector3(50, 0.3, 44),
		# Ops partition
		"OpsPartL": Vector3(12, 5, 0.3),
		"OpsPartC": Vector3(2, 5, 0.3),
		"OpsPartR": Vector3(12, 5, 0.3),
		# Room divider
		"OpsDivider": Vector3(0.3, 5, 11),
	}

	for node_name in wall_shapes:
		var col_node := get_node_or_null(node_name) as CollisionShape3D
		if col_node and not col_node.shape:
			var box := BoxShape3D.new()
			box.size = wall_shapes[node_name]
			col_node.shape = box
