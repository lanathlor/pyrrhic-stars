@tool
extends Node3D
## Spatial debug: prints bounding boxes and alignment checks for all key nodes.
## Attach to the root MilitaryBuilding node. Runs in editor and at runtime.

func _ready() -> void:
	_audit()


func _audit() -> void:
	print("\n=== SPATIAL AUDIT ===")

	# Check CSG geometry bounds
	_print_section("GROUND FLOOR")
	_report_child("GroundFloor/Floor")
	_report_child("GroundFloor/GlassNorth")
	_report_child("GroundFloor/GlassWest")
	_report_child("GroundFloor/GlassEast")
	_report_child("GroundFloor/WallSouthLeft")
	_report_child("GroundFloor/WallSouthRight")

	_print_section("LIFT SHAFT")
	_report_child("LiftShaft/ShaftFloor")
	_report_child("LiftShaft/ShaftWallWest")
	_report_child("LiftShaft/ShaftWallBack")

	_print_section("ELEVATOR & DOORS")
	_report_child("ElevatorCab")
	_report_child("DoorGF_L")
	_report_child("DoorGF_R")
	_report_child("DoorUF_L")
	_report_child("DoorUF_R")

	_print_section("UPPER FLOOR")
	_report_child("UpperFloor/LandingFloorLeft")
	_report_child("UpperFloor/LandingFloorRight")
	_report_child("UpperFloor/BriefingFloor")
	_report_child("UpperFloor/PortalFloor")

	_print_section("PROPS")
	_report_child("Props/TowerExterior")
	_report_child("Props/HoloDesk")
	_report_child("Props/ConsoleDesk")

	_print_section("PORTAL AREA")
	_report_child("PortalArea")

	# Alignment checks
	_print_section("ALIGNMENT CHECKS")
	_check_elevator_shaft_alignment()
	_check_door_shaft_alignment()
	_check_floor_continuity()

	print("=== END AUDIT ===\n")


func _report_child(path: String) -> void:
	var node := get_node_or_null(path)
	if not node:
		print("  MISSING: %s" % path)
		return

	var pos := Vector3.ZERO
	if node is Node3D:
		pos = (node as Node3D).global_position
	print("  %s -> pos=(%.1f, %.1f, %.1f)" % [path, pos.x, pos.y, pos.z])

	# For CSGBox3D, also report the box extents
	if node is CSGBox3D:
		var box := node as CSGBox3D
		var half := box.size / 2.0
		print("    bounds: X[%.1f, %.1f] Y[%.1f, %.1f] Z[%.1f, %.1f]" % [
			pos.x - half.x, pos.x + half.x,
			pos.y - half.y, pos.y + half.y,
			pos.z - half.z, pos.z + half.z,
		])


func _check_elevator_shaft_alignment() -> void:
	var cab := get_node_or_null("ElevatorCab") as Node3D
	var shaft_w := get_node_or_null("LiftShaft/ShaftWallWest") as CSGBox3D
	if not cab or not shaft_w:
		print("  SKIP: elevator or shaft missing")
		return

	var cab_x := cab.global_position.x
	var cab_z := cab.global_position.z
	var shaft_center_x := (shaft_w.global_position.x + 1.65)  # shaft is 3m wide
	var shaft_center_z := shaft_w.global_position.z
	print("  Cab center: X=%.1f Z=%.1f" % [cab_x, cab_z])
	print("  Shaft center: X=%.1f Z=%.1f" % [shaft_center_x, shaft_center_z])
	if absf(cab_x - shaft_center_x) > 0.5 or absf(cab_z - shaft_center_z) > 0.5:
		print("  WARNING: Elevator cab not centered in shaft!")
	else:
		print("  OK: Elevator aligned with shaft")


func _check_door_shaft_alignment() -> void:
	var door_gf := get_node_or_null("DoorGF_L") as Node3D
	var door_uf := get_node_or_null("DoorUF_L") as Node3D
	if not door_gf or not door_uf:
		return
	print("  GF doors at Z=%.1f, UF doors at Z=%.1f" % [
		door_gf.global_position.z, door_uf.global_position.z])

	# GF doors should be at shaft front wall Z
	# UF doors should be at landing front
	var shaft_front_z := get_node_or_null("LiftShaft/GFLintel")
	if shaft_front_z:
		print("  GF shaft opening at Z=%.1f" % shaft_front_z.global_position.z)
		if absf(door_gf.global_position.z - shaft_front_z.global_position.z) > 0.5:
			print("  WARNING: GF doors not aligned with shaft opening!")
		else:
			print("  OK: GF doors aligned")


func _check_floor_continuity() -> void:
	var gf_floor := get_node_or_null("GroundFloor/Floor") as CSGBox3D
	var uf_landing := get_node_or_null("UpperFloor/LandingFloorLeft") as CSGBox3D
	if not gf_floor or not uf_landing:
		return
	var gf_top := gf_floor.global_position.y + gf_floor.size.y / 2.0
	var uf_floor_y := uf_landing.global_position.y - uf_landing.size.y / 2.0
	print("  GF floor top: Y=%.2f" % gf_top)
	print("  UF floor bottom: Y=%.2f" % uf_floor_y)
	print("  Gap between floors: %.2fm" % (uf_floor_y - gf_top))


func _print_section(title: String) -> void:
	print("\n--- %s ---" % title)
