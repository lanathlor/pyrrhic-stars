extends RefCounted
class_name MapData

## Pure-data script holding floor geometry and waypoint targets for the minimap
## and full map overlay. Shared by shared_hud.gd, map_overlay.gd, and portal_trail.gd.

# =============================================================================
# Floor definitions — waypoint targets per floor
# =============================================================================

const FLOORS := [
	{
		"id": "lower_district",
		"name": "Lower District",
		"target": Vector3(5.0, -199.8, -55.0),
		"y_min": -210.0, "y_max": -150.0,
		"arrival_radius": 5.0,
	},
	{
		"id": "tower_lobby",
		"name": "Tower Lobby",
		"target": Vector3(0.0, 0.2, 43.0),
		"y_min": -5.0, "y_max": 10.0,
		"arrival_radius": 3.0,
		"bounds_min": Vector3(-24, 0, -1),
		"bounds_max": Vector3(24, 0, 43),
	},
	{
		"id": "plaza",
		"name": "Plaza",
		"target": Vector3(0.0, 0.2, -1.0),
		"y_min": -5.0, "y_max": 10.0,
		"arrival_radius": 4.0,
	},
	{
		"id": "ops",
		"name": "Operations Deck",
		"target": Vector3(33.0, 100.2, 5.5),
		"y_min": 95.0, "y_max": 110.0,
		"arrival_radius": 4.0,
	},
]


# =============================================================================
# Lower District (Y = -200)
# =============================================================================

const LOWER_DISTRICT := {
	"center": Vector2(5.0, -55.0),
	"size": Vector2(200.0, 200.0),
	"buildings": [
		# Row A (north, Z -100 to -150)
		{"center": Vector2(-65, -125), "size": Vector2(50, 45)},   # A1
		{"center": Vector2(-20, -130), "size": Vector2(30, 30)},   # A2
		{"center": Vector2(30, -125),  "size": Vector2(40, 40)},   # A3
		{"center": Vector2(75, -130),  "size": Vector2(25, 35)},   # A4
		# Row B (west of plaza)
		{"center": Vector2(-65, -80),  "size": Vector2(40, 15)},   # B1
		{"center": Vector2(-68, -55),  "size": Vector2(35, 14)},   # B2
		{"center": Vector2(-62, -30),  "size": Vector2(42, 16)},   # B3
		# Row C (east of plaza)
		{"center": Vector2(30, -82),   "size": Vector2(30, 12)},   # C1
		{"center": Vector2(65, -80),   "size": Vector2(30, 14)},   # C2
		{"center": Vector2(35, -42),   "size": Vector2(28, 18)},   # C3
		{"center": Vector2(70, -38),   "size": Vector2(35, 15)},   # C4
		# Row D (south)
		{"center": Vector2(-60, 15),   "size": Vector2(45, 40)},   # D1
		{"center": Vector2(-20, 20),   "size": Vector2(30, 30)},   # D2
		{"center": Vector2(30, 18),    "size": Vector2(35, 35)},   # D3
		{"center": Vector2(75, 15),    "size": Vector2(28, 35)},   # D4
		# Misc
		{"center": Vector2(-8, -55),   "size": Vector2(2, 2)},     # Monument
		{"center": Vector2(5, -55),    "size": Vector2(10, 10)},   # Lift shaft
	],
}


# =============================================================================
# Plaza (Y = 0)
# =============================================================================

const PLAZA := {
	"center": Vector2(0.0, 21.0),
	"size": Vector2(250.0, 272.0),
	"buildings": [
		# Lift shaft hole
		{"center": Vector2(5, -55),   "size": Vector2(10, 10)},
		# Tower walls (split for entrance gap)
		{"center": Vector2(-25, 21),  "size": Vector2(1, 44)},    # West wall
		{"center": Vector2(25, 21),   "size": Vector2(1, 44)},    # East wall
		{"center": Vector2(0, 43),    "size": Vector2(50, 1)},    # South wall
		{"center": Vector2(-17, -1),  "size": Vector2(16, 1)},    # North wall left
		{"center": Vector2(17, -1),   "size": Vector2(16, 1)},    # North wall right
	],
}


# =============================================================================
# Operations Deck (Y = 100)
# =============================================================================

const OPS := {
	"center": Vector2(8.0, 21.0),
	"size": Vector2(66.0, 44.0),
	"buildings": [
		# Elevator shaft
		{"center": Vector2(0, 45),    "size": Vector2(5, 5)},
		# Ops partitions
		{"center": Vector2(-18, 10),  "size": Vector2(12, 0.5)},
		{"center": Vector2(18, 10),   "size": Vector2(12, 0.5)},
		{"center": Vector2(0, 21),    "size": Vector2(2, 0.5)},
		{"center": Vector2(0, 21),    "size": Vector2(0.5, 11)},
	],
	# Walkable floor zones — the map draws these instead of one big bounds rect
	"floors": [
		{"center": Vector2(0.0, 21.0), "size": Vector2(50.0, 44.0)},   # Tower interior
		{"center": Vector2(31.0, 5.5), "size": Vector2(16.0, 12.0)},   # Landing pad ramp
	],
	# Kept for portal_trail navmesh baking
	"extra_floors": [
		{"center": Vector2(31.0, 5.5), "size": Vector2(16.0, 12.0)},
	],
}


# =============================================================================
# Arena
# =============================================================================

const ARENA := {
	"center": Vector2(0.0, 19.0),
	"size": Vector2(40.0, 68.0),
	"buildings": [
		# Boss room walls
		{"center": Vector2(0, -15),   "size": Vector2(40, 0.5)},  # North wall
		{"center": Vector2(20, -1.5), "size": Vector2(0.5, 27)},  # East wall
		{"center": Vector2(-20, -1.5),"size": Vector2(0.5, 27)},  # West wall
		# Hallway walls
		{"center": Vector2(-8, 26),   "size": Vector2(0.5, 28)},  # Hallway left
		{"center": Vector2(8, 26),    "size": Vector2(0.5, 28)},  # Hallway right
		# Connector walls at Z=12
		{"center": Vector2(-14, 12),  "size": Vector2(12, 0.5)},  # Left connector
		{"center": Vector2(14, 12),   "size": Vector2(12, 0.5)},  # Right connector
		# Boss room pillars
		{"center": Vector2(-8, -6),   "size": Vector2(1.5, 1.5)},
		{"center": Vector2(8, -6),    "size": Vector2(1.5, 1.5)},
		{"center": Vector2(-8, 6),    "size": Vector2(1.5, 1.5)},
		{"center": Vector2(8, 6),     "size": Vector2(1.5, 1.5)},
		{"center": Vector2(0, -10),   "size": Vector2(1.5, 1.5)},
		{"center": Vector2(0, 10),    "size": Vector2(1.5, 1.5)},
		# Hallway cover
		{"center": Vector2(-4, 27),   "size": Vector2(2, 1)},
		{"center": Vector2(4, 27),    "size": Vector2(2, 1)},
		{"center": Vector2(-4, 17),   "size": Vector2(2, 1)},
		{"center": Vector2(4, 17),    "size": Vector2(2, 1)},
	],
}


# =============================================================================
# Helpers
# =============================================================================

static func get_floor_for_position(pos: Vector3) -> Dictionary:
	for floor_def in FLOORS:
		if pos.y < floor_def["y_min"] or pos.y > floor_def["y_max"]:
			continue
		if floor_def.has("bounds_min"):
			var bmin: Vector3 = floor_def["bounds_min"]
			var bmax: Vector3 = floor_def["bounds_max"]
			if pos.x < bmin.x or pos.x > bmax.x \
					or pos.z < bmin.z or pos.z > bmax.z:
				continue
		return floor_def
	return {}


static func get_geometry_for_floor(floor_id: String) -> Dictionary:
	match floor_id:
		"lower_district":
			return LOWER_DISTRICT
		"plaza":
			return PLAZA
		"tower_lobby":
			return PLAZA  # same geometry, different waypoint target
		"ops":
			return OPS
		"arena":
			return ARENA
		_:
			return {}


# =============================================================================
# Scene scanner — shared by minimap and full map overlay
# =============================================================================

## Scan an environment scene tree and return classified shapes for the given
## player Y position.  Returns {rects: Array[{rect, type}], circles: Array[{center, radius, green}]}.
## Types: "floor", "garden", "ground", "green", "wall", "" (skip).
static func scan_scene(env: Node3D, player_y: float) -> Dictionary:
	var rects: Array = []
	var circles: Array = []
	var y_min := player_y - 0.5  # just below feet for thin floor surfaces
	var y_max := player_y + 100.0  # tower walls are 130m tall, centered at Y=65
	_scan_node_recursive(env, y_min, y_max, rects, circles)
	return {"rects": rects, "circles": circles}


static func _scan_node_recursive(node: Node, y_min: float, y_max: float,
		rects: Array, circles: Array) -> void:
	if node is CSGBox3D:
		var csg: CSGBox3D = node
		if not csg.visible:
			return
		var pos: Vector3 = csg.global_position
		var s: Vector3 = csg.size
		var box_y_min: float = pos.y - s.y / 2.0
		var box_y_max: float = pos.y + s.y / 2.0
		# Box center must be near or above player level.
		# Excludes sub-floor fills centered well below the floor.
		var box_center_y: float = pos.y
		if box_center_y >= y_min and box_center_y <= y_max:
			var area: float = s.x * s.z
			if area >= 0.1:
				var node_type: String = _classify(csg.name, s, area)
				if node_type != "":
					rects.append({"rect": Rect2(pos.x - s.x / 2.0, pos.z - s.z / 2.0, s.x, s.z), "type": node_type})

	elif node is CSGCylinder3D:
		var cyl: CSGCylinder3D = node
		if not cyl.visible:
			return
		var pos: Vector3 = cyl.global_position
		if pos.y >= y_min and pos.y <= y_max:
			if cyl.radius >= 0.3:
				circles.append({"center": Vector2(pos.x, pos.z), "radius": cyl.radius, "green": _is_green(cyl.name)})

	elif node is CSGSphere3D:
		var sph: CSGSphere3D = node
		if not sph.visible:
			return
		var pos: Vector3 = sph.global_position
		var r: float = sph.radius
		if pos.y >= y_min and pos.y <= y_max:
			if r >= 0.3:
				circles.append({"center": Vector2(pos.x, pos.z), "radius": r, "green": _is_green(sph.name)})

	for child in node.get_children():
		_scan_node_recursive(child, y_min, y_max, rects, circles)


static func _classify(node_name: StringName, box_size: Vector3, area: float) -> String:
	# Massive underground fills — not visible, skip
	if box_size.y > 50.0 and area > 5000.0:
		return ""
	# Large flat green = garden
	if _is_green(node_name) and box_size.y < 2.0 and area > 3000.0:
		return "garden"
	# Small green = hedge/planter
	if _is_green(node_name):
		return "green"
	# Large flat = floor (plaza, lobby)
	if box_size.y < 2.0 and area > 3000.0:
		return "floor"
	# Small flat = path/sidewalk
	if box_size.y < 2.0:
		return "ground"
	# Tall but huge area = background fill
	if area > 3000.0:
		return ""
	return "wall"


static func _is_green(node_name: StringName) -> bool:
	var n: String = node_name.to_lower()
	return n.contains("hedge") or n.contains("flower") or n.contains("grass") \
		or n.contains("planter") or n.contains("tree") or n.contains("canopy") \
		or n.contains("shrub") or n.contains("bush")
