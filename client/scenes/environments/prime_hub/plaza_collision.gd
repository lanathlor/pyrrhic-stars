@tool
extends StaticBody3D
## Generates collision shapes for the outdoor plaza.
## Ground is split into 4 pieces to leave a hole for the public lift shaft
## centered at X=35, Z=-12 (7x7 opening).

const SHAFT_X := 5.0
const SHAFT_Z := -55.0
const SHAFT_HALF := 3.0  # half of 6m opening, matches platform size

# Full ground: 250 wide, 272 deep, centered at (0, -0.2, 21)
# X: -125 to 125, Z: -115 to 157
const GX_MIN := -125.0
const GX_MAX := 125.0
const GZ_MIN := -115.0
const GZ_MAX := 157.0
const GY := -0.2


func _ready() -> void:
	# Ground split into 4 pieces around shaft hole
	var shaft_x_min := SHAFT_X - SHAFT_HALF
	var shaft_x_max := SHAFT_X + SHAFT_HALF
	var shaft_z_min := SHAFT_Z - SHAFT_HALF
	var shaft_z_max := SHAFT_Z + SHAFT_HALF

	var ground_shapes: Dictionary = {
		"GroundW":
		[
			# West strip: full Z, X from -125 to shaft left edge
			Vector3((GX_MIN + shaft_x_min) / 2.0, GY, (GZ_MIN + GZ_MAX) / 2.0),
			Vector3(shaft_x_min - GX_MIN, 0.2, GZ_MAX - GZ_MIN)
		],
		"GroundE":
		[
			# East strip: full Z, X from shaft right edge to 125
			Vector3((shaft_x_max + GX_MAX) / 2.0, GY, (GZ_MIN + GZ_MAX) / 2.0),
			Vector3(GX_MAX - shaft_x_max, 0.2, GZ_MAX - GZ_MIN)
		],
		"GroundN":
		[
			# North fill: shaft X range, Z from -115 to shaft north edge
			Vector3(SHAFT_X, GY, (GZ_MIN + shaft_z_min) / 2.0),
			Vector3(shaft_x_max - shaft_x_min, 0.2, shaft_z_min - GZ_MIN)
		],
		"GroundS":
		[
			# South fill: shaft X range, Z from shaft south edge to 157
			Vector3(SHAFT_X, GY, (shaft_z_max + GZ_MAX) / 2.0),
			Vector3(shaft_x_max - shaft_x_min, 0.2, GZ_MAX - shaft_z_max)
		],
	}

	# Create ground pieces dynamically
	for shape_name in ground_shapes:
		var data: Array = ground_shapes[shape_name]
		var pos: Vector3 = data[0]
		var size: Vector3 = data[1]
		var existing := get_node_or_null(shape_name) as CollisionShape3D
		if not existing:
			existing = CollisionShape3D.new()
			existing.name = shape_name
			add_child(existing)
		existing.transform.origin = pos
		var box := BoxShape3D.new()
		box.size = size
		existing.shape = box

	# Original GroundShape is no longer needed (replaced by 4 pieces)
	var old_ground := get_node_or_null("GroundShape") as CollisionShape3D
	if old_ground:
		old_ground.shape = null

	var edge_shapes: Dictionary = {
		"EdgeNShape": Vector3(250, 3, 0.5),
		"EdgeSShape": Vector3(250, 3, 0.5),
		"EdgeWShape": Vector3(0.5, 3, 272),
		"EdgeEShape": Vector3(0.5, 3, 272),
	}

	for node_name in edge_shapes:
		var col := get_node_or_null(node_name) as CollisionShape3D
		if col and not col.shape:
			var box := BoxShape3D.new()
			box.size = edge_shapes[node_name]
			col.shape = box
