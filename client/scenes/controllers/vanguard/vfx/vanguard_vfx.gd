extends Node

## Vanguard VFX manager subsystem.
## Spawns and manages visual effects for all Vanguard abilities.
## One-shot effects auto-free; looping effects are tracked and cleaned up on state exit.

const SwingTrailScene := preload("res://scenes/controllers/vanguard/vfx/swing_trail.tscn")
const HitImpactScene := preload("res://scenes/controllers/vanguard/vfx/hit_impact.tscn")
const BlockShieldScene := preload("res://scenes/controllers/vanguard/vfx/block_shield.tscn")
const ParryFlashScene := preload("res://scenes/controllers/vanguard/vfx/parry_flash.tscn")
const BladeSwirlAuraScene := preload("res://scenes/controllers/vanguard/vfx/blade_swirl_aura.tscn")
const GroundSlamShockwaveScene := preload(
	"res://scenes/controllers/vanguard/vfx/ground_slam_shockwave.tscn"
)

var ctrl: Node

# Active looping effects
var _active_block_shield: Node3D = null
var _active_blade_swirl: Node3D = null


func _ready() -> void:
	ctrl = get_parent()


func _scene_root() -> Node:
	return ctrl.get_tree().current_scene if ctrl.get_tree() else null


# --- Swing Trail ---


func start_swing_trail() -> void:
	var root := _scene_root()
	if not root:
		return
	# One-shot arc: spawns, sweeps, fades, auto-frees. No tracking needed.
	var trail: Node3D = SwingTrailScene.instantiate()
	root.add_child(trail)
	trail.start(ctrl)


func stop_swing_trail() -> void:
	pass  # Swing trail is one-shot, auto-frees


# --- Hit Impact ---


func spawn_hit_impact(hit_pos: Vector3) -> void:
	var root := _scene_root()
	if not root:
		return
	var impact: Node3D = HitImpactScene.instantiate()
	root.add_child(impact)
	impact.global_position = hit_pos


# --- Block Shield ---


func show_block_shield() -> void:
	if _active_block_shield and is_instance_valid(_active_block_shield):
		return
	_active_block_shield = BlockShieldScene.instantiate()
	ctrl.add_child(_active_block_shield)
	_active_block_shield.position = Vector3(0.0, 1.0, -0.5)


func hide_block_shield() -> void:
	if _active_block_shield and is_instance_valid(_active_block_shield):
		_active_block_shield.fade_out()
		_active_block_shield = null


# --- Parry Flash ---


func spawn_parry_flash() -> void:
	var root := _scene_root()
	if not root:
		return
	var flash: Node3D = ParryFlashScene.instantiate()
	root.add_child(flash)
	flash.global_position = (
		ctrl.global_position + Vector3(0.0, 1.0, 0.0) + (-ctrl.transform.basis.z * 0.5)
	)


# --- Blade Swirl Aura ---


func start_blade_swirl() -> void:
	stop_blade_swirl()
	var root := _scene_root()
	if not root:
		return
	_active_blade_swirl = BladeSwirlAuraScene.instantiate()
	root.add_child(_active_blade_swirl)
	_active_blade_swirl.start(ctrl)


func stop_blade_swirl() -> void:
	if _active_blade_swirl and is_instance_valid(_active_blade_swirl):
		_active_blade_swirl.stop()
		_active_blade_swirl = null


# --- Ground Slam Shockwave ---


func spawn_ground_slam_shockwave(pos: Vector3, rot_y: float) -> void:
	var root := _scene_root()
	if not root:
		return
	var shockwave: Node3D = GroundSlamShockwaveScene.instantiate()
	root.add_child(shockwave)
	shockwave.global_position = pos + Vector3(0.0, 0.05, 0.0)
	shockwave.rotation.y = rot_y
