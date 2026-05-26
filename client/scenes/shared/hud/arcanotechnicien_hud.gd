extends Control

## Arcanotechnicien HUD -- Harmonist healer HUD.
## Components: flux bar, ability bar, confluence display, party frames, channel bar,
## lock-on reticle, damage/heal flash, hit marker.
##
## Heavy drawing logic is delegated to ArcanotechnicienHudDrawHelpers.

const DAMAGE_FLASH_DURATION: float = 0.3
const HEAL_FLASH_DURATION: float = 0.4
const HIT_MARKER_DURATION: float = 0.15

const ARCANOTECHNICIEN_COLOR := Color(0.3, 0.65, 0.85)

# -- Class max HP lookup for party frames --
const CLASS_MAX_HP := {
	"gunner": 150.0,
	"vanguard": 200.0,
	"blade_dancer": 150.0,
	"arcanotechnicien": 150.0,
}

# -- Timers --
var _damage_flash_timer: float = 0.0
var _heal_flash_timer: float = 0.0
var _hit_marker_timer: float = 0.0

# -- Confluence --
var _confluence_tier: int = 0
var _confluence_stacks: int = 0

# -- GCD --
var _gcd_ratio: float = 0.0

# -- Flux bar --
var _flux_current: float = 100.0
var _flux_max: float = 100.0
var _flux_pools: Array = []  # [{school, current, max}, ...]

# -- Channel bar --
var _channel_progress: float = 0.0  # 0.0 = just started, 1.0 = complete
var _channel_ability_name: String = ""
var _channel_active: bool = false
var _sustain_active: bool = false
var _sustain_elapsed: float = 0.0

# -- Party frames --
var _party_data: Array = []  # Array[Dictionary] with: peer_id, name, health, max_health, class_name
var _hovered_party_index: int = -1
var _party_frame_rects: Array[Rect2] = []  # cached rects for hover detection
var _selected_peer_id: int = -1  # click-selected target peer_id

var _codex_panel: Control

@onready var damage_overlay: ColorRect = $DamageOverlay
@onready var lock_on_reticle: Control = $LockOnReticle
@onready var ability_bar = $AbilityBar


func _ready() -> void:
	damage_overlay.modulate.a = 0.0
	ability_bar.accent_color = ARCANOTECHNICIEN_COLOR
	# Enable mouse pass-through but track mouse for party frame hover
	mouse_filter = Control.MOUSE_FILTER_PASS
	# WoW-style: reticle shows "SELECTED" instead of lock-on hints
	lock_on_reticle.hint_text = ""
	lock_on_reticle.lock_label = "SELECTED"

	# Codex panel (full-screen overlay, hidden by default)
	var CodexPanelScript := preload("res://scenes/shared/hud/codex_panel.gd")
	_codex_panel = CodexPanelScript.new()
	_codex_panel.name = "CodexPanel"
	_codex_panel.set_anchors_and_offsets_preset(Control.PRESET_FULL_RECT)
	add_child(_codex_panel)
	_codex_panel.loadout_applied.connect(_on_loadout_applied)
	_codex_panel.commitment_applied.connect(_on_commitment_applied)
	_codex_panel.preset_saved.connect(_on_preset_saved)
	_codex_panel.preset_deleted.connect(_on_preset_deleted)


func _process(delta: float) -> void:
	if _damage_flash_timer > 0.0:
		_damage_flash_timer -= delta
		damage_overlay.color = Color(0.8, 0.0, 0.0, 1.0)
		damage_overlay.modulate.a = _damage_flash_timer / DAMAGE_FLASH_DURATION * 0.4
	if _heal_flash_timer > 0.0:
		_heal_flash_timer -= delta
		damage_overlay.color = Color(0.2, 0.8, 0.4, 1.0)
		damage_overlay.modulate.a = _heal_flash_timer / HEAL_FLASH_DURATION * 0.25
	if _hit_marker_timer > 0.0:
		_hit_marker_timer -= delta

	# Update party frame hover detection
	_update_party_hover()

	lock_on_reticle.queue_redraw()
	queue_redraw()


func _draw() -> void:
	_draw_hit_marker()
	ArcanotechnicienHudDrawHelpers.draw_confluence(self, size, _confluence_tier, _confluence_stacks)
	ArcanotechnicienHudDrawHelpers.draw_flux_bar(self, size, _flux_current, _flux_max, _flux_pools)
	_party_frame_rects = ArcanotechnicienHudDrawHelpers.draw_party_frames(
		self, size, _party_data, _selected_peer_id, _hovered_party_index
	)
	(
		ArcanotechnicienHudDrawHelpers
		. draw_channel_bar(
			self,
			size,
			_channel_active,
			_channel_progress,
			_channel_ability_name,
			_sustain_active,
			_sustain_elapsed,
		)
	)


# =============================================================================
# Public API
# =============================================================================


func update_confluence(tier: int, stacks: int) -> void:
	_confluence_tier = tier
	_confluence_stacks = stacks


func update_abilities(abilities: Array) -> void:
	ability_bar.update_abilities(abilities)


func update_gcd(ratio: float) -> void:
	_gcd_ratio = ratio
	ability_bar.update_gcd(ratio)


func update_flux(current: float, max_value: float, pools: Array = []) -> void:
	_flux_current = current
	_flux_max = max_value
	_flux_pools = pools


func update_channel(progress: float, ability_name: String) -> void:
	_channel_progress = clampf(progress, 0.0, 1.0)
	_channel_ability_name = ability_name
	_channel_active = true
	_sustain_active = false


func update_sustain(ability_name: String, elapsed: float) -> void:
	_channel_ability_name = ability_name
	_sustain_active = true
	_sustain_elapsed = elapsed
	_channel_active = true


func hide_channel() -> void:
	_channel_active = false
	_sustain_active = false
	_channel_progress = 0.0
	_channel_ability_name = ""


func update_party(party: Array) -> void:
	_party_data = party


## Returns the peer_id of the party member under the mouse, or -1 if none.
func get_mouseover_target() -> int:
	if _hovered_party_index < 0 or _hovered_party_index >= _party_data.size():
		return -1
	return _party_data[_hovered_party_index].get("peer_id", -1)


## Returns the peer_id of the party member clicked at screen_pos, or -1 if none.
func get_clicked_target(screen_pos: Vector2) -> int:
	for i in _party_frame_rects.size():
		if i < _party_data.size() and _party_frame_rects[i].has_point(screen_pos):
			return _party_data[i].get("peer_id", -1)
	return -1


func show_selected_target(target: Node3D, cam: Camera3D) -> void:
	if "peer_id" in target:
		_selected_peer_id = target.peer_id
	else:
		_selected_peer_id = -1
	lock_on_reticle.set_meta("lock_target", target)
	lock_on_reticle.set_meta("lock_camera", cam)
	lock_on_reticle._lock_active = true
	lock_on_reticle.visible = true


func hide_selected_target() -> void:
	_selected_peer_id = -1
	lock_on_reticle._lock_active = false
	lock_on_reticle.remove_meta("lock_target")
	lock_on_reticle.remove_meta("lock_camera")


func update_selected_target(target: Node3D, cam: Camera3D) -> void:
	lock_on_reticle.set_meta("lock_target", target)
	lock_on_reticle.set_meta("lock_camera", cam)


func show_damage_flash() -> void:
	_damage_flash_timer = DAMAGE_FLASH_DURATION


func show_heal_flash() -> void:
	_heal_flash_timer = HEAL_FLASH_DURATION


func toggle_codex() -> void:
	if _codex_panel.is_open():
		_codex_panel.close()
	else:
		_codex_panel.open()


func close_codex() -> void:
	if _codex_panel and _codex_panel.is_open():
		_codex_panel.close()


func is_codex_open() -> bool:
	return _codex_panel and _codex_panel.is_open()


func show_hit_marker() -> void:
	_hit_marker_timer = HIT_MARKER_DURATION


# =============================================================================
# Drawing -- Hit Marker
# =============================================================================


func _draw_hit_marker() -> void:
	if _hit_marker_timer <= 0.0:
		return
	var center := size / 2.0
	var t: float = _hit_marker_timer / HIT_MARKER_DURATION
	var color := Color(0.3, 0.85, 0.5, t)
	var gap: float = 5.0
	var x_len: float = 10.0
	var thick: float = 2.5
	draw_line(
		center + Vector2(-gap - x_len, -gap - x_len),
		center + Vector2(-gap, -gap),
		color,
		thick,
		true
	)
	draw_line(
		center + Vector2(gap + x_len, -gap - x_len), center + Vector2(gap, -gap), color, thick, true
	)
	draw_line(
		center + Vector2(-gap - x_len, gap + x_len), center + Vector2(-gap, gap), color, thick, true
	)
	draw_line(
		center + Vector2(gap + x_len, gap + x_len), center + Vector2(gap, gap), color, thick, true
	)


# =============================================================================
# Internal -- Party frame hover detection
# =============================================================================


func _update_party_hover() -> void:
	_hovered_party_index = -1
	var mouse_pos := get_local_mouse_position()
	for i in _party_frame_rects.size():
		if _party_frame_rects[i].has_point(mouse_pos):
			_hovered_party_index = i
			break


# =============================================================================
# Codex panel callbacks
# =============================================================================


func _on_loadout_applied(slots: Array) -> void:
	NetworkManager.loadout.send_set_loadout(slots)


func _on_commitment_applied(entries: Array) -> void:
	NetworkManager.loadout.send_set_flux_commitment(entries)


func _on_preset_saved(preset_name: String, slots: Array, commitment: String) -> void:
	NetworkManager.loadout.send_save_preset(preset_name, slots, commitment)


func _on_preset_deleted(preset_id: int) -> void:
	NetworkManager.loadout.send_delete_preset(preset_id)
