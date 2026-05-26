class_name CityscapeHelpers
## Static helpers for cityscape mesh building, materials, and data definitions.
## Used by cityscape.gd to keep procedural generation under 500 lines.

## 20 landmark mega-towers scattered in the north view.
## Format: [x, z, width, depth, height]
const LANDMARKS := [
	[-400.0, -600.0, 50.0, 45.0, 600.0],
	[300.0, -800.0, 60.0, 50.0, 700.0],
	[-150.0, -1100.0, 55.0, 50.0, 750.0],
	[600.0, -500.0, 45.0, 40.0, 550.0],
	[-700.0, -900.0, 65.0, 55.0, 680.0],
	[100.0, -1400.0, 55.0, 50.0, 800.0],
	[-500.0, -1600.0, 60.0, 55.0, 720.0],
	[450.0, -1200.0, 50.0, 45.0, 650.0],
	[-250.0, -2000.0, 70.0, 65.0, 850.0],
	[700.0, -1800.0, 65.0, 55.0, 780.0],
	[-900.0, -700.0, 55.0, 50.0, 620.0],
	[200.0, -2400.0, 75.0, 70.0, 900.0],
	[-600.0, -2200.0, 60.0, 55.0, 810.0],
	[900.0, -1000.0, 50.0, 45.0, 640.0],
	[-100.0, -400.0, 40.0, 35.0, 500.0],
	[500.0, -2600.0, 70.0, 65.0, 860.0],
	[-800.0, -1400.0, 55.0, 50.0, 700.0],
	[350.0, -350.0, 35.0, 30.0, 480.0],
	[-350.0, -1800.0, 60.0, 55.0, 740.0],
	[800.0, -2200.0, 65.0, 60.0, 820.0],
]

## Monolith lights stacked up the structure.
## Format: [height_fraction, color, energy, range]
const MONOLITH_LIGHT_DEFS := [
	[0.15, Color(0.95, 0.8, 0.45), 30.0, 2000.0],
	[0.3, Color(0.95, 0.8, 0.45), 40.0, 2500.0],
	[0.45, Color(0.95, 0.8, 0.45), 40.0, 2500.0],
	[0.55, Color(0.95, 0.8, 0.45), 30.0, 2000.0],
	[0.65, Color(0.4, 0.55, 0.95), 35.0, 2000.0],
	[0.75, Color(0.95, 0.8, 0.45), 40.0, 2500.0],
	[0.85, Color(0.4, 0.55, 0.95), 50.0, 3000.0],
	[0.95, Color(0.4, 0.55, 0.95), 60.0, 3500.0],
]

## Tall buildings sitting on the UndergroundBase ceiling.
## Format: [x, z, width, depth, height]
const LOWER_TALL := [
	[-60.0, -90.0, 30.0, 25.0, 150.0],
	[-100.0, -60.0, 25.0, 30.0, 148.0],
	[60.0, -100.0, 28.0, 22.0, 149.0],
	[90.0, -70.0, 22.0, 28.0, 147.0],
]

## City glow light rings. Format: [distance, count, energy, y_offset]
const GLOW_RINGS := [
	[400.0, 14, 10.0, -100.0],
	[800.0, 16, 15.0, -80.0],
	[1500.0, 14, 25.0, -50.0],
	[3000.0, 12, 35.0, -20.0],
]


## Box-Muller gaussian. Returns a sample with given mean and std_dev.
static func gaussian(rng: RandomNumberGenerator, mean: float, std_dev: float) -> float:
	var u1: float = maxf(rng.randf(), 0.0001)
	var u2: float = rng.randf()
	var z: float = sqrt(-2.0 * log(u1)) * cos(TAU * u2)
	return mean + std_dev * z


## Create a solid-color StandardMaterial3D.
static func solid_mat(color: Color, rough: float) -> StandardMaterial3D:
	var m := StandardMaterial3D.new()
	m.albedo_color = color
	m.roughness = rough
	return m


## Create a glowing StandardMaterial3D with emission.
static func glow_mat(color: Color, emission: Color, energy: float) -> StandardMaterial3D:
	var m := StandardMaterial3D.new()
	m.albedo_color = color
	m.emission_enabled = true
	m.emission = emission
	m.emission_energy_multiplier = energy
	m.roughness = 0.2
	return m


## Build a MultiMeshInstance3D from transforms with per-instance color variation.
static func build_mm_varied(
	nm: String, xforms: Array[Transform3D], rng: RandomNumberGenerator
) -> MultiMeshInstance3D:
	if xforms.is_empty():
		return null
	var mesh := BoxMesh.new()
	mesh.size = Vector3.ONE
	var mm := MultiMesh.new()
	mm.transform_format = MultiMesh.TRANSFORM_3D
	mm.use_custom_data = true
	mm.instance_count = xforms.size()
	mm.mesh = mesh
	for i in xforms.size():
		mm.set_instance_transform(i, xforms[i])
		var shade: float = rng.randf_range(0.7, 1.3)
		var warm: float = rng.randf_range(-0.03, 0.03)
		mm.set_instance_custom_data(i, Color(shade, warm, 0.0, 0.0))
	var mmi := MultiMeshInstance3D.new()
	mmi.multimesh = mm
	mmi.name = nm
	var shader := Shader.new()
	shader.code = """shader_type spatial;
render_mode cull_back;
uniform vec3 base_color = vec3(0.35, 0.36, 0.38);
uniform float roughness_val : hint_range(0.0, 1.0) = 0.75;
varying float v_shade;
varying float v_warm;
void vertex() {
	v_shade = INSTANCE_CUSTOM.r;
	v_warm = INSTANCE_CUSTOM.g;
}
void fragment() {
	ALBEDO = base_color * v_shade + vec3(v_warm, v_warm * 0.5, -v_warm);
	ROUGHNESS = roughness_val;
}
"""
	var mat := ShaderMaterial.new()
	mat.shader = shader
	mat.set_shader_parameter("base_color", Vector3(0.35, 0.36, 0.38))
	mat.set_shader_parameter("roughness_val", 0.75)
	mmi.material_override = mat
	return mmi


## Build a MultiMeshInstance3D from transforms with a single material.
static func build_mm(
	nm: String, xforms: Array[Transform3D], mat: StandardMaterial3D
) -> MultiMeshInstance3D:
	if xforms.is_empty():
		return null
	var mesh := BoxMesh.new()
	mesh.size = Vector3.ONE
	var mm := MultiMesh.new()
	mm.transform_format = MultiMesh.TRANSFORM_3D
	mm.instance_count = xforms.size()
	mm.mesh = mesh
	for i in xforms.size():
		mm.set_instance_transform(i, xforms[i])
	var mmi := MultiMeshInstance3D.new()
	mmi.multimesh = mm
	mmi.name = nm
	mmi.material_override = mat
	return mmi


## Build a wall MeshInstance3D.
static func build_wall(
	nm: String, pos: Vector3, size: Vector3, mat: StandardMaterial3D
) -> MeshInstance3D:
	var mi := MeshInstance3D.new()
	var mesh := BoxMesh.new()
	mesh.size = size
	mi.mesh = mesh
	mi.position = pos
	mi.material_override = mat
	mi.name = nm
	return mi


## Build the ground plane MeshInstance3D.
static func build_ground(ground_y: float, extent: float) -> MeshInstance3D:
	var ground_mi := MeshInstance3D.new()
	var ground_mesh := BoxMesh.new()
	ground_mesh.size = Vector3(extent * 3.0, 2.0, extent * 3.0)
	ground_mi.mesh = ground_mesh
	ground_mi.position = Vector3(0.0, ground_y - 1.0, 10.0)
	var ground_mat := StandardMaterial3D.new()
	ground_mat.albedo_color = Color(0.06, 0.065, 0.08)
	ground_mat.roughness = 0.5
	ground_mat.metallic = 0.2
	ground_mi.material_override = ground_mat
	ground_mi.name = "CityGround"
	return ground_mi


## Build monolith omni lights as a Node3D container.
static func build_monolith_lights(
	ground_y: float, mono_x: float, mono_z: float, mono_h: float
) -> Node3D:
	var container := Node3D.new()
	container.name = "MonolithLights"
	for li in MONOLITH_LIGHT_DEFS.size():
		var def: Array = MONOLITH_LIGHT_DEFS[li]
		var ml := OmniLight3D.new()
		ml.name = "Mono_%d" % li
		ml.light_color = def[1]
		ml.light_energy = def[2]
		ml.omni_range = def[3]
		ml.light_volumetric_fog_energy = def[2] * 0.8
		ml.omni_attenuation = 0.8
		ml.shadow_enabled = false
		ml.position = Vector3(mono_x, ground_y + mono_h * def[0], mono_z)
		container.add_child(ml)
	return container


## Build city glow lights as a Node3D container.
static func build_city_glow(rng: RandomNumberGenerator) -> Node3D:
	var container := Node3D.new()
	container.name = "CityGlow"

	var idx := 0
	for ring in GLOW_RINGS:
		var dist: float = ring[0]
		var count: int = ring[1]
		var energy: float = ring[2]
		var y_off: float = ring[3]

		for i in count:
			var angle: float = (TAU / count) * i + rng.randf_range(-0.3, 0.3)
			var px: float = cos(angle) * dist + rng.randf_range(-dist * 0.15, dist * 0.15)
			var pz: float = sin(angle) * dist + 10.0 + rng.randf_range(-dist * 0.15, dist * 0.15)

			var light := OmniLight3D.new()
			light.name = "Glow_%d" % idx
			idx += 1

			if rng.randf() < 0.7:
				light.light_color = Color(0.95, 0.8, 0.5)
			else:
				light.light_color = Color(0.5, 0.65, 0.9)

			var light_dist: float = Vector2(px, pz - 10.0).length()
			light.light_energy = energy
			light.omni_range = light_dist + 300.0
			light.light_volumetric_fog_energy = energy
			light.omni_attenuation = 1.5
			light.shadow_enabled = false
			light.position = Vector3(px, y_off, pz)

			container.add_child(light)
	return container
