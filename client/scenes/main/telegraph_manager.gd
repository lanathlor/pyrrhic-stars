extends Node

## Server-authoritative telegraph renderer.
##
## The server sends, each world-state tick, the full set of active telegraphs
## (shape, world position, size, category, and the commit→execute tick window).
## This manager is a dumb renderer: it reconciles one mesh per telegraph id and
## animates the radial "fill" from the commit tick to the execution tick using a
## local clock interpolated against the snapshot tick.

const FLOOR_Y := 0.04
const TICK_HZ := 20.0

const SHADER_CIRCLE: Shader = preload("res://assets/shaders/telegraph_circle.gdshader")
const SHADER_CONE: Shader = preload("res://assets/shaders/telegraph_cone.gdshader")
const SHADER_LINE: Shader = preload("res://assets/shaders/telegraph_line.gdshader")

# category -> [fill_color, edge_color]. 0=unavoidable 1=parryable 2=blockable 3=heal.
const COLORS := {
	0: [Color(0.9, 0.1, 0.1, 0.32), Color(1.0, 0.25, 0.15, 0.95)],
	1: [Color(1.0, 0.75, 0.1, 0.30), Color(1.0, 0.85, 0.2, 0.95)],
	2: [Color(0.1, 0.7, 0.95, 0.30), Color(0.3, 0.85, 1.0, 0.95)],
	3: [Color(0.2, 0.9, 0.3, 0.28), Color(0.4, 1.0, 0.5, 0.9)],
}

var ctrl: Node
var _container: Node3D
var _active: Dictionary = {}  # id -> { node, mats, shape, category, start, exec }
var _latest_tick: float = 0.0
var _latest_tick_ms: int = 0


func _ready() -> void:
	ctrl = get_parent()
	_container = ctrl.get_node("Telegraphs")


## update_telegraphs reconciles the rendered set against the server snapshot.
func update_telegraphs(telegraphs: Array, tick: int) -> void:
	_latest_tick = float(tick)
	_latest_tick_ms = Time.get_ticks_msec()

	var seen: Dictionary = {}
	for t: Dictionary in telegraphs:
		var id: int = t["id"]
		seen[id] = true
		var entry: Dictionary = _active.get(id, {})
		if entry.is_empty() or entry["shape"] != t["shape"]:
			if not entry.is_empty():
				(entry["node"] as Node).queue_free()
			entry = _build(t)
			_active[id] = entry
		_update_geometry(entry, t)
		entry["start"] = float(t["start_tick"])
		entry["exec"] = float(t["execute_tick"])

	for id: int in _active.keys():
		if not seen.has(id):
			(_active[id]["node"] as Node).queue_free()
			_active.erase(id)


func _process(_delta: float) -> void:
	if _active.is_empty():
		return
	var now_tick := _latest_tick + float(Time.get_ticks_msec() - _latest_tick_ms) / 1000.0 * TICK_HZ
	for id: int in _active:
		var e: Dictionary = _active[id]
		var span: float = maxf(e["exec"] - e["start"], 1.0)
		var fill: float = clampf((now_tick - e["start"]) / span, 0.0, 1.0)
		for mat: ShaderMaterial in e["mats"]:
			mat.set_shader_parameter("fill", fill)


func _build(t: Dictionary) -> Dictionary:
	var shape: int = t["shape"]
	var category: int = t["category"]
	var entry := {"shape": shape, "category": category, "mats": []}
	if shape == 3:  # multi: a parent grouping N rings
		var parent := Node3D.new()
		_container.add_child(parent)
		entry["node"] = parent
	else:
		var shader: Shader = SHADER_CIRCLE
		if shape == 1:
			shader = SHADER_CONE
		elif shape == 2:
			shader = SHADER_LINE
		var mi := _make_ring(shader, category)
		_container.add_child(mi)
		entry["node"] = mi
		entry["plane"] = mi.mesh
		entry["mats"] = [mi.material_override]
	return entry


func _make_ring(shader: Shader, category: int) -> MeshInstance3D:
	var mi := MeshInstance3D.new()
	mi.top_level = true  # world-space; ignore container transform
	mi.mesh = PlaneMesh.new()
	var mat := ShaderMaterial.new()
	mat.shader = shader
	var pal: Array = COLORS.get(category, COLORS[0])
	mat.set_shader_parameter("color", pal[0])
	mat.set_shader_parameter("edge_color", pal[1])
	mat.set_shader_parameter("fill", 0.0)
	mi.material_override = mat
	return mi


func _update_geometry(entry: Dictionary, t: Dictionary) -> void:
	match int(t["shape"]):
		0:  # circle
			var r: float = t["radius"]
			(entry["plane"] as PlaneMesh).size = Vector2(r * 2.0, r * 2.0)
			(entry["node"] as Node3D).position = Vector3(t["cx"], FLOOR_Y, t["cz"])
		1:  # cone
			var rng: float = t["range"]
			(entry["plane"] as PlaneMesh).size = Vector2(rng * 2.0, rng * 2.0)
			var node := entry["node"] as Node3D
			node.position = Vector3(t["cx"], FLOOR_Y, t["cz"])
			node.rotation.y = t["facing"]
			(entry["mats"][0] as ShaderMaterial).set_shader_parameter("half_angle", t["half_angle"])
		2:  # line
			var w: float = t["width"]
			var ln: float = t["length"]
			(entry["plane"] as PlaneMesh).size = Vector2(w, ln)
			var node := entry["node"] as Node3D
			node.rotation.y = atan2(float(t["dir_x"]), float(t["dir_z"]))
			node.position = Vector3(
				t["cx"] + t["dir_x"] * ln * 0.5, FLOOR_Y, t["cz"] + t["dir_z"] * ln * 0.5
			)
		3:  # multi_circle
			_update_multi(entry, t)


func _update_multi(entry: Dictionary, t: Dictionary) -> void:
	var parent := entry["node"] as Node3D
	var centers: Array = t["centers"]
	var r: float = t["radius"]
	if parent.get_child_count() != centers.size():
		for ch in parent.get_children():
			parent.remove_child(ch)
			ch.free()
		for i in range(centers.size()):
			parent.add_child(_make_ring(SHADER_CIRCLE, entry["category"]))
	var mats: Array = []
	for i in range(centers.size()):
		var mi := parent.get_child(i) as MeshInstance3D
		(mi.mesh as PlaneMesh).size = Vector2(r * 2.0, r * 2.0)
		var c: Vector2 = centers[i]
		mi.position = Vector3(c.x, FLOOR_Y, c.y)
		mats.append(mi.material_override)
	entry["mats"] = mats


## clear removes all telegraphs (called on zone transfer / despawn).
func clear() -> void:
	for id: int in _active.keys():
		(_active[id]["node"] as Node).queue_free()
	_active.clear()
