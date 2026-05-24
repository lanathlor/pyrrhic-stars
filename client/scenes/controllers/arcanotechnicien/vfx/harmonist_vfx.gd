extends Node

## Harmonist VFX manager subsystem.
## Spawns and manages visual effects for all Harmonist abilities.
## One-shot effects auto-free; looping effects are tracked and cleaned up on state exit.

const CastFlashScene := preload("res://scenes/controllers/arcanotechnicien/vfx/cast_flash.tscn")
const HealBeamScene := preload("res://scenes/controllers/arcanotechnicien/vfx/heal_beam.tscn")
const HealPulseScene := preload("res://scenes/controllers/arcanotechnicien/vfx/heal_pulse.tscn")
const ZoneHealAuraScene := preload(
	"res://scenes/controllers/arcanotechnicien/vfx/zone_heal_aura.tscn"
)
const FrostWardScene := preload("res://scenes/controllers/arcanotechnicien/vfx/frost_ward.tscn")
const GustTrailScene := preload("res://scenes/controllers/arcanotechnicien/vfx/gust_trail.tscn")
const ChannelFluxScene := preload(
	"res://scenes/controllers/arcanotechnicien/vfx/channel_flux.tscn"
)
const ConfluenceOrbitScene := preload(
	"res://scenes/controllers/arcanotechnicien/vfx/confluence_orbit.tscn"
)
const SympatheticFieldScene := preload(
	"res://scenes/controllers/arcanotechnicien/vfx/sympathetic_field.tscn"
)

var ctrl: Node

# Active looping effects
var _active_beam: Node3D = null
var _active_zone: Node3D = null
var _active_channel: Node3D = null
var _active_confluence: Node3D = null
var _active_field: Node3D = null


func _ready() -> void:
	ctrl = get_parent()


func _scene_root() -> Node:
	return ctrl.get_tree().current_scene if ctrl.get_tree() else null


# --- Cast Flash (any direct ability) ---


func spawn_cast_flash() -> void:
	var root := _scene_root()
	if not root:
		return
	var flash: Node3D = CastFlashScene.instantiate()
	root.add_child(flash)
	# Position at caster's hands (chest height, slightly forward)
	flash.global_position = (
		ctrl.global_position + Vector3(0.0, 1.1, 0.0) + (-ctrl.transform.basis.z * 0.4)
	)


# --- Heal Beam (Mending Beam, slot 1) ---


func start_heal_beam(target: Node3D) -> void:
	stop_heal_beam()
	var root := _scene_root()
	if not root:
		return
	_active_beam = HealBeamScene.instantiate()
	root.add_child(_active_beam)
	_active_beam.start(ctrl, target)


func stop_heal_beam() -> void:
	if _active_beam and is_instance_valid(_active_beam):
		_active_beam.stop()
		_active_beam = null


# --- Heal Pulse (Mending Surge, slot 0) ---


func spawn_heal_pulse(pos: Vector3) -> void:
	var root := _scene_root()
	if not root:
		return
	var pulse: Node3D = HealPulseScene.instantiate()
	root.add_child(pulse)
	pulse.global_position = pos


# --- Zone Telegraph (Transfusion, slot 3) ---


func start_zone_telegraph(pos: Vector3, radius: float) -> void:
	stop_zone_telegraph()
	var root := _scene_root()
	if not root:
		return
	_active_zone = ZoneHealAuraScene.instantiate()
	root.add_child(_active_zone)
	_active_zone.start(ctrl, radius)


func stop_zone_telegraph() -> void:
	if _active_zone and is_instance_valid(_active_zone):
		_active_zone.stop()
		_active_zone = null


# --- Frost Ward (slot 4) ---


func spawn_frost_ward(target: Node3D) -> void:
	var root := _scene_root()
	if not root:
		return
	var ward: Node3D = FrostWardScene.instantiate()
	root.add_child(ward)
	ward.start(target)


# --- Gust Step Trail (slot 5) ---


func spawn_gust_trail() -> void:
	var root := _scene_root()
	if not root:
		return
	var trail: Node3D = GustTrailScene.instantiate()
	root.add_child(trail)
	trail.global_position = ctrl.global_position


# --- Channel Flux (any channel) ---


func start_channel_flux() -> void:
	stop_channel_flux()
	var root := _scene_root()
	if not root:
		return
	_active_channel = ChannelFluxScene.instantiate()
	root.add_child(_active_channel)
	_active_channel.start(ctrl)


func update_channel_flux(progress: float) -> void:
	if _active_channel and is_instance_valid(_active_channel):
		_active_channel.update_progress(progress)


func stop_channel_flux() -> void:
	if _active_channel and is_instance_valid(_active_channel):
		_active_channel.stop()
		_active_channel = null


# --- Confluence Orbit (persistent) ---


func update_confluence(tier: int, stacks: int) -> void:
	if not _active_confluence or not is_instance_valid(_active_confluence):
		var root := _scene_root()
		if not root:
			return
		_active_confluence = ConfluenceOrbitScene.instantiate()
		root.add_child(_active_confluence)
		_active_confluence.start(ctrl)
	_active_confluence.update(tier, stacks)


# --- Sympathetic Field (persistent) ---


func show_sympathetic_field() -> void:
	if _active_field and is_instance_valid(_active_field):
		return
	var root := _scene_root()
	if not root:
		return
	_active_field = SympatheticFieldScene.instantiate()
	root.add_child(_active_field)
	_active_field.start(ctrl)


func hide_sympathetic_field() -> void:
	if _active_field and is_instance_valid(_active_field):
		_active_field.stop()
		_active_field = null


# --- Remote Player VFX ---


func drive_remote_vfx(old_vs: int, new_vs: int) -> void:
	# Stop effects from previous state
	if old_vs == NetSerializer.VS_AT_CHANNELING_BEAM:
		stop_heal_beam()
	if old_vs == NetSerializer.VS_AT_CHANNELING_ZONE:
		stop_zone_telegraph()
	var channel_states: Array[int] = [
		NetSerializer.VS_AT_CHANNELING,
		NetSerializer.VS_AT_CHANNELING_BEAM,
		NetSerializer.VS_AT_CHANNELING_ZONE,
	]
	if old_vs in channel_states:
		stop_channel_flux()

	# Start effects for new state
	if new_vs == NetSerializer.VS_AT_CHANNELING_BEAM:
		var target := _find_nearest_ally()
		if target:
			start_heal_beam(target)
		start_channel_flux()
	elif new_vs == NetSerializer.VS_AT_CHANNELING_ZONE:
		start_zone_telegraph(ctrl.global_position, 6.0)
		start_channel_flux()
	elif new_vs == NetSerializer.VS_AT_CHANNELING:
		start_channel_flux()


func _find_nearest_ally() -> Node3D:
	var best: Node3D = null
	var best_dist: float = 30.0
	for p in GameManager.players:
		if not is_instance_valid(p) or not p.visible:
			continue
		if p == ctrl:
			continue
		var dist: float = ctrl.global_position.distance_to(p.global_position)
		if dist < best_dist:
			best_dist = dist
			best = p
	return best
