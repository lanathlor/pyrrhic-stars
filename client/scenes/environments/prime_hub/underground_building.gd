extends Node3D
## Generates window bands and trim strips on the south building's north face.

@export var glass_mat: Material
@export var trim_mat: Material

const FACE_Z := 49.9
const TRIM_Z := 49.85
const FLOOR_INTERVAL := 4.0
const WIN_HEIGHT := 3.0
const WIDTH := 250.0
const Y_START := -6.0
const Y_END := -150.0


func _ready() -> void:
	var y := Y_START
	var idx := 1
	while y >= Y_END:
		# Glass window band
		var win := CSGBox3D.new()
		win.name = "Win%02d" % idx
		win.size = Vector3(WIDTH, WIN_HEIGHT, 0.15)
		win.position = Vector3(0, y, FACE_Z)
		if glass_mat:
			win.material = glass_mat
		add_child(win)

		# Blue trim strip at floor edge (top of window)
		var trim := CSGBox3D.new()
		trim.name = "Trim%02d" % idx
		trim.size = Vector3(WIDTH, 0.08, 0.08)
		trim.position = Vector3(0, y + WIN_HEIGHT / 2.0 + 0.5, TRIM_Z)
		if trim_mat:
			trim.material = trim_mat
		add_child(trim)

		y -= FLOOR_INTERVAL
		idx += 1
