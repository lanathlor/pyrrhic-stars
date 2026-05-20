extends Node

## Blade Dancer blade formation visuals: materials, positions, rotations.

const BLADE_SCENE := "res://assets/models/weapons/weapon_floating_blade.glb"

var ctrl: Node

var _blade_nodes: Array[Node3D] = []
var _blade_orbit_angle: float = 0.0
var _blade_lerp_speed: float = 12.0
var _blade_spin: float = 0.0

# Materials -- one per config
var _orbit_material: StandardMaterial3D
var _fan_material: StandardMaterial3D
var _lance_material: StandardMaterial3D
var _scatter_material: StandardMaterial3D
var _crown_material: StandardMaterial3D


func _ready() -> void:
	ctrl = get_parent()


func setup() -> void:
	_setup_blade_materials()
	_setup_blades()


func _setup_blade_materials() -> void:
	_orbit_material = _make_material(Color(0.2, 0.8, 0.9), Color(0.1, 0.6, 0.8))
	_fan_material = _make_material(Color(1.0, 0.5, 0.1), Color(0.9, 0.35, 0.05))
	_lance_material = _make_material(Color(0.9, 0.2, 0.1), Color(0.8, 0.1, 0.05))
	_scatter_material = _make_material(Color(0.6, 0.2, 0.9), Color(0.5, 0.1, 0.8))
	_crown_material = _make_material(Color(1.0, 0.85, 0.3), Color(0.9, 0.75, 0.2))


func _make_material(albedo: Color, emission: Color) -> StandardMaterial3D:
	var mat := StandardMaterial3D.new()
	mat.albedo_color = albedo
	mat.emission_enabled = true
	mat.emission = emission
	mat.emission_energy_multiplier = 2.0
	return mat


func get_config_material(cfg: int) -> StandardMaterial3D:
	match cfg:
		ctrl.Config.ORBIT:
			return _orbit_material
		ctrl.Config.FAN:
			return _fan_material
		ctrl.Config.LANCE:
			return _lance_material
		ctrl.Config.SCATTER:
			return _scatter_material
		ctrl.Config.CROWN:
			return _crown_material
	return _orbit_material


func _setup_blades() -> void:
	var blade_scene := load(BLADE_SCENE) as PackedScene
	if not blade_scene:
		push_warning("BladeDancer: could not load blade model %s" % BLADE_SCENE)
		return
	for i in 6:
		var blade := blade_scene.instantiate()
		ctrl.blade_pivot.add_child(blade)
		_blade_nodes.append(blade)
		_apply_blade_material(blade, _orbit_material)


func update_blade_visual(delta: float) -> void:
	if _blade_nodes.is_empty():
		return

	_blade_spin += delta * 2.0
	_blade_orbit_angle += 120.0 * delta

	var targets: Array[Vector3] = []
	var target_rots: Array[float] = []
	var mat: StandardMaterial3D
	var lerp_speed: float = _blade_lerp_speed

	match ctrl.state:
		ctrl.State.CASTING:
			# During casting, blend toward destination config formation
			var dest_cfg: int = ctrl._casting_spell.get("dest", ctrl.Config.ORBIT)
			mat = get_config_material(dest_cfg)
			lerp_speed = 20.0
			var dur: float = ctrl._casting_spell.get("dur", 0.4)
			var progress: float = 1.0 - (ctrl._cast_timer / dur) if dur > 0.0 else 1.0
			progress = clampf(progress, 0.0, 1.0)

			# Sweep blades forward during first half, settle into dest formation second half
			if progress < 0.5:
				var sweep: float = progress * 2.0  # 0->1 in first half
				var sweep_angle: float = lerpf(-60.0, 60.0, sweep)
				for i in _blade_nodes.size():
					var a: float = deg_to_rad(sweep_angle + (i - 2.5) * 12.0)
					var r: float = 1.8
					targets.append(Vector3(sin(a) * r, 0.9, -cos(a) * r))
					target_rots.append(a)
			else:
				var settle: float = (progress - 0.5) * 2.0  # 0->1 in second half
				var dest_targets: Array[Vector3] = _get_formation_positions(dest_cfg)
				var dest_rots: Array[float] = _get_formation_rotations(dest_cfg)
				for i in _blade_nodes.size():
					var sweep_a: float = deg_to_rad(60.0 + (i - 2.5) * 12.0)
					var sweep_pos := Vector3(sin(sweep_a) * 1.8, 0.9, -cos(sweep_a) * 1.8)
					targets.append(sweep_pos.lerp(dest_targets[i], settle))
					target_rots.append(lerp_angle(sweep_a, dest_rots[i], settle))

		ctrl.State.DASH:
			mat = get_config_material(ctrl.config)
			lerp_speed = 15.0
			for i in _blade_nodes.size():
				var spread: float = (i - 2.5) * 0.4
				targets.append(Vector3(spread, 0.9, 1.5))
				target_rots.append(PI)

		_:
			# Idle / Move -- use current config formation
			mat = get_config_material(ctrl.config)
			targets = _get_formation_positions(ctrl.config)
			target_rots = _get_formation_rotations(ctrl.config)

	for i in _blade_nodes.size():
		if i >= targets.size():
			break
		_blade_nodes[i].position = _blade_nodes[i].position.lerp(targets[i], lerp_speed * delta)
		_blade_nodes[i].rotation.y = lerp_angle(
			_blade_nodes[i].rotation.y, target_rots[i], lerp_speed * delta
		)
		_blade_nodes[i].rotation.x = sin(_blade_spin + i * 2.0) * 0.15
		_apply_blade_material(_blade_nodes[i], mat)


## Get idle formation positions for a given config.
func _get_formation_positions(cfg: int) -> Array[Vector3]:
	var positions: Array[Vector3] = []
	match cfg:
		ctrl.Config.ORBIT:
			# Circular orbit around player -- 6 blades, 60 deg spacing
			for i in 6:
				var angle: float = deg_to_rad(_blade_orbit_angle + i * 60.0)
				positions.append(Vector3(cos(angle) * 1.0, 0.9, sin(angle) * 1.0))
		ctrl.Config.FAN:
			# Arc spread in front of player -- 6 blades, 20 deg spacing (-50 to +50)
			for i in 6:
				var angle: float = deg_to_rad(-50.0 + i * 20.0)
				positions.append(Vector3(sin(angle) * 1.5, 0.9, -cos(angle) * 1.5))
		ctrl.Config.LANCE:
			# Staggered double line aimed forward
			var local_dir: Vector3
			if ctrl._lock_on_active and ctrl._lock_target and is_instance_valid(ctrl._lock_target):
				var to_target: Vector3 = ctrl._lock_target.global_position - ctrl.global_position
				to_target.y = 0.0
				local_dir = ctrl.transform.basis.inverse() * to_target.normalized()
			else:
				local_dir = Vector3(0.0, 0.0, -1.0)
			# Inner 3 blades in center line
			for i in 3:
				var d: float = 2.0 + i * 0.4
				positions.append(Vector3(local_dir.x * d, 0.9, local_dir.z * d))
			# Outer 3 blades offset sideways
			var side := Vector3(-local_dir.z, 0.0, local_dir.x)
			for i in 3:
				var d: float = 2.1 + i * 0.4
				var offset: float = 0.2 if i % 2 == 0 else -0.2
				positions.append(Vector3(
					local_dir.x * d + side.x * offset,
					0.9,
					local_dir.z * d + side.z * offset
				))
		ctrl.Config.SCATTER:
			# Blades spread outward in different directions -- 6 blades, staggered heights
			for i in 6:
				var angle: float = deg_to_rad(_blade_orbit_angle * 0.7 + i * 60.0)
				var r: float = 1.8
				positions.append(Vector3(cos(angle) * r, 0.6 + (i % 3) * 0.3, sin(angle) * r))
		ctrl.Config.CROWN:
			# Hover above head in a halo -- 6 blades, 60 deg spacing
			for i in 6:
				var angle: float = deg_to_rad(_blade_orbit_angle * 0.5 + i * 60.0)
				var r: float = 0.6
				positions.append(Vector3(cos(angle) * r, 1.8, sin(angle) * r))
	return positions


## Get idle formation rotations for a given config.
func _get_formation_rotations(cfg: int) -> Array[float]:
	var rotations: Array[float] = []
	match cfg:
		ctrl.Config.ORBIT:
			for i in 6:
				var angle: float = deg_to_rad(_blade_orbit_angle + i * 60.0)
				rotations.append(angle + PI / 2.0)
		ctrl.Config.FAN:
			for i in 6:
				var angle: float = deg_to_rad(-50.0 + i * 20.0)
				rotations.append(angle)
		ctrl.Config.LANCE:
			var local_dir: Vector3
			if ctrl._lock_on_active and ctrl._lock_target and is_instance_valid(ctrl._lock_target):
				var to_target: Vector3 = ctrl._lock_target.global_position - ctrl.global_position
				to_target.y = 0.0
				local_dir = ctrl.transform.basis.inverse() * to_target.normalized()
			else:
				local_dir = Vector3(0.0, 0.0, -1.0)
			for i in 6:
				rotations.append(atan2(local_dir.x, local_dir.z))
		ctrl.Config.SCATTER:
			for i in 6:
				var angle: float = deg_to_rad(_blade_orbit_angle * 0.7 + i * 60.0)
				rotations.append(angle + PI / 4.0)
		ctrl.Config.CROWN:
			for i in 6:
				var angle: float = deg_to_rad(_blade_orbit_angle * 0.5 + i * 60.0)
				rotations.append(angle)
	return rotations


## Apply a material override to all MeshInstance3D children in a GLB instance.
func _apply_blade_material(node: Node3D, mat: StandardMaterial3D) -> void:
	for child in node.get_children():
		if child is MeshInstance3D:
			for s in child.get_surface_override_material_count():
				child.set_surface_override_material(s, mat)
		if child.get_child_count() > 0:
			_apply_blade_material(child, mat)
