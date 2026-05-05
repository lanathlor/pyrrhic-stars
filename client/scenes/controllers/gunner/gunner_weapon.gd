extends Node

## Gunner weapon: shooting, tracers, muzzle flash, viewmodel setup/update, weapon attachment.

const WEAPON_SCENE := "res://assets/models/weapons/weapon_rifle.glb"

var ctrl: Node

# Viewmodel state
var _viewmodel: Node3D
var _viewmodel_weapon: Node3D
var _recoil_offset: float = 0.0
var _bob_time: float = 0.0
var _muzzle_flash_timer: float = 0.0


func _ready() -> void:
	ctrl = get_parent()


func attach_weapon() -> void:
	var offset_pos: Vector3 = ctrl._weapon_offset_pos
	var offset_rot := Vector3(
		deg_to_rad(ctrl._weapon_offset_rot_deg.x),
		deg_to_rad(ctrl._weapon_offset_rot_deg.y),
		deg_to_rad(ctrl._weapon_offset_rot_deg.z)
	)
	ctrl.character_model.attach_weapon(WEAPON_SCENE, "mixamorig_RightHand", offset_pos, offset_rot)


func setup_viewmodel() -> void:
	_viewmodel = Node3D.new()
	_viewmodel.name = "Viewmodel"
	ctrl.camera.add_child(_viewmodel)
	_viewmodel.position = ctrl._vm_pos
	_viewmodel.rotation = Vector3(
		deg_to_rad(ctrl._vm_rot_deg.x),
		deg_to_rad(ctrl._vm_rot_deg.y),
		deg_to_rad(ctrl._vm_rot_deg.z)
	)
	_viewmodel.scale = ctrl._vm_scale

	var weapon_scene: PackedScene = load(WEAPON_SCENE) as PackedScene
	if weapon_scene:
		_viewmodel_weapon = weapon_scene.instantiate()
		_viewmodel.add_child(_viewmodel_weapon)


func update_weapon_live() -> void:
	# Live-update weapon offset from inspector while game runs
	if ctrl.character_model.weapon_node:
		ctrl.character_model.weapon_node.position = ctrl._weapon_offset_pos
		ctrl.character_model.weapon_node.rotation = Vector3(
			deg_to_rad(ctrl._weapon_offset_rot_deg.x),
			deg_to_rad(ctrl._weapon_offset_rot_deg.y),
			deg_to_rad(ctrl._weapon_offset_rot_deg.z)
		)
	# Live-update viewmodel from inspector
	if _viewmodel and ctrl._is_local():
		if _recoil_offset <= 0.001:
			_viewmodel.rotation = Vector3(
				deg_to_rad(ctrl._vm_rot_deg.x),
				deg_to_rad(ctrl._vm_rot_deg.y),
				deg_to_rad(ctrl._vm_rot_deg.z)
			)
		_viewmodel.scale = ctrl._vm_scale


func handle_shooting(delta: float) -> void:
	ctrl._fire_cooldown -= delta
	# During rechamber timing window, shoot button confirms the rechamber instead
	var rechamber_phase: int = ctrl._rechamber_phase
	if rechamber_phase == 2:
		if (
			Input.is_action_just_pressed("shoot")
			and Input.get_mouse_mode() == Input.MOUSE_MODE_CAPTURED
		):
			ctrl._rechamber_phase = 0
			ctrl._rechamber_buff = true
			ctrl._rechamber_buff_timer = ctrl.RECHAMBER_BUFF_DURATION
			if NetworkManager.is_active:
				NetworkManager.send_ability(12, ctrl.head.rotation.x, ctrl.rotation.y)
		return
	# Normal shooting — can't shoot during rechamber windup or lockout
	if rechamber_phase != 0:
		return
	var overclock_active: bool = ctrl._overclock_active
	var current_fire_rate: float = ctrl.OVERCLOCK_FIRE_RATE if overclock_active else ctrl.fire_rate
	if (
		Input.is_action_pressed("shoot")
		and ctrl._fire_cooldown <= 0.0
		and Input.get_mouse_mode() == Input.MOUSE_MODE_CAPTURED
	):
		_shoot()
		ctrl._fire_cooldown = current_fire_rate


func _shoot() -> void:
	ctrl.gun_ray.force_raycast_update()
	_muzzle_flash_timer = 0.05
	ctrl.muzzle_light.visible = true
	ctrl.hud.on_shoot()
	_recoil_offset = 0.06

	# Tracer line from weapon muzzle to hit (or max range)
	var tracer_from: Vector3 = get_muzzle_pos()
	var tracer_to: Vector3
	if ctrl.gun_ray.is_colliding():
		tracer_to = ctrl.gun_ray.get_collision_point()
	else:
		tracer_to = (
			ctrl.head.global_position + ctrl.head.global_transform.basis * Vector3(0, 0, -100)
		)
	spawn_tracer(tracer_from, tracer_to)

	# Tell server we fired
	if NetworkManager.is_active:
		NetworkManager.send_ability(0, ctrl.head.rotation.x, ctrl.rotation.y)


func update_muzzle_flash(delta: float) -> void:
	if _muzzle_flash_timer > 0.0:
		_muzzle_flash_timer -= delta
		if _muzzle_flash_timer <= 0.0:
			ctrl.muzzle_light.visible = false


func update_viewmodel(delta: float) -> void:
	if not _viewmodel:
		return
	var flat_vel := Vector3(ctrl.velocity.x, 0.0, ctrl.velocity.z)
	var speed: float = flat_vel.length()

	# Walk bob
	if speed > 0.5 and ctrl.is_on_floor():
		_bob_time += delta * speed * 1.2
		var bob_y: float = sin(_bob_time * 2.0) * 0.006
		var bob_x: float = cos(_bob_time) * 0.003
		_viewmodel.position = ctrl._vm_pos + Vector3(bob_x, bob_y, 0.0)
	else:
		_bob_time = 0.0
		_viewmodel.position = _viewmodel.position.lerp(ctrl._vm_pos, delta * 10.0)

	# Recoil recovery
	if _recoil_offset > 0.001:
		_recoil_offset = lerpf(_recoil_offset, 0.0, delta * 18.0)
	else:
		_recoil_offset = 0.0
	_viewmodel.rotation.x = deg_to_rad(ctrl._vm_rot_deg.x) - _recoil_offset


## Get the muzzle position — from viewmodel for local, from bone weapon for remote.
func get_muzzle_pos() -> Vector3:
	if ctrl._is_local() and _viewmodel:
		return _viewmodel.global_position
	if ctrl.character_model.weapon_node and is_instance_valid(ctrl.character_model.weapon_node):
		return ctrl.character_model.weapon_node.global_position
	return ctrl.global_position + Vector3(0.0, 1.4, 0.0)


## Spawn a bullet tracer line in world space from origin to end point.
func spawn_tracer(from_pos: Vector3, to_pos: Vector3) -> void:
	var diff := to_pos - from_pos
	var length: float = diff.length()
	if length < 0.1:
		return

	var dir := diff.normalized()
	var mid := (from_pos + to_pos) / 2.0

	# Build transform manually — no need for look_at or being in tree
	var tracer := MeshInstance3D.new()
	var box := BoxMesh.new()
	box.size = Vector3(0.03, 0.03, length)
	tracer.mesh = box
	tracer.cast_shadow = GeometryInstance3D.SHADOW_CASTING_SETTING_OFF

	var mat := StandardMaterial3D.new()
	mat.albedo_color = Color(1.0, 0.95, 0.6, 0.7)
	mat.emission_enabled = true
	mat.emission = Color(1.0, 0.85, 0.3)
	mat.emission_energy_multiplier = 6.0
	mat.shading_mode = BaseMaterial3D.SHADING_MODE_UNSHADED
	mat.transparency = BaseMaterial3D.TRANSPARENCY_ALPHA
	tracer.material_override = mat

	# Orient along the line using a manual basis
	var up := Vector3.UP
	if absf(dir.dot(up)) > 0.99:
		up = Vector3.RIGHT
	var z_axis := -dir
	var x_axis := up.cross(z_axis).normalized()
	var y_axis := z_axis.cross(x_axis).normalized()
	tracer.transform = Transform3D(Basis(x_axis, y_axis, z_axis), mid)

	var scene_root: Node = (
		ctrl.get_tree().current_scene if ctrl.get_tree().current_scene else ctrl.get_tree().root
	)
	scene_root.add_child(tracer)

	# Fade out and free
	var tween: Tween = ctrl.get_tree().create_tween()
	tween.tween_property(mat, "albedo_color:a", 0.0, 0.12)
	tween.parallel().tween_property(mat, "emission_energy_multiplier", 0.0, 0.12)
	tween.tween_callback(tracer.queue_free)


## Spawn a tracer for a remote gunner using their synced aim data.
func fire_remote_tracer() -> void:
	var net_position: Vector3 = ctrl._net_position
	var net_rotation_y: float = ctrl._net_rotation_y
	var net_aim_pitch: float = ctrl._net_aim_pitch
	var shoulder: Vector3 = net_position + Vector3(0.0, 1.3, 0.0)
	var fwd := Vector3(0, 0, -1).rotated(Vector3(0, 1, 0), net_rotation_y)
	var from_pos: Vector3 = shoulder + fwd * 0.5
	var dir := Vector3(0, 0, -1)
	dir = dir.rotated(Vector3(1, 0, 0), net_aim_pitch)
	dir = dir.rotated(Vector3(0, 1, 0), net_rotation_y)
	var to_pos: Vector3 = from_pos + dir * 100.0
	spawn_tracer(from_pos, to_pos)
