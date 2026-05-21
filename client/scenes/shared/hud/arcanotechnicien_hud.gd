extends Control

## Arcanotechnicien HUD -- Harmonist healer HUD.
## Components: flux bar, spell bar, confluence display, party frames, channel bar,
## lock-on reticle, damage/heal flash, hit marker.

const DAMAGE_FLASH_DURATION: float = 0.3
const HEAL_FLASH_DURATION: float = 0.4
const HIT_MARKER_DURATION: float = 0.15

const ARCANOTECHNICIEN_COLOR := Color(0.3, 0.65, 0.85)

# -- Confluence colors --
const CONFLUENCE_DIM := Color(0.15, 0.25, 0.35, 0.4)
const CONFLUENCE_ACTIVE := Color(0.3, 0.65, 0.85, 0.95)
const CONFLUENCE_MAX := Color(0.5, 0.85, 1.0, 1.0)

# -- Panel / bar constants (shared visual language) --
const PANEL_BG := Color(0.02, 0.025, 0.035, 0.82)
const PANEL_BORDER := Color(0.28, 0.3, 0.36, 0.85)
const TEXT_PRIMARY := Color(0.92, 0.94, 0.97, 0.95)
const TEXT_MUTED := Color(0.66, 0.7, 0.77, 0.9)
const HEALTH_GOOD := Color(0.28, 0.78, 0.4, 1.0)
const HEALTH_BAD := Color(0.82, 0.24, 0.24, 1.0)
const FLUX_COLOR := Color(0.25, 0.55, 0.9, 1.0)
const FLUX_LOW_COLOR := Color(0.6, 0.3, 0.8, 1.0)
const CHANNEL_BG := Color(0.04, 0.05, 0.07, 0.75)
const CHANNEL_FILL := Color(0.3, 0.65, 0.85, 0.9)

# -- Class max HP lookup for party frames --
const CLASS_MAX_HP := {
	"gunner": 150.0,
	"vanguard": 200.0,
	"blade_dancer": 150.0,
	"arcanotechnicien": 150.0,
}

# -- Party frame layout --
const PARTY_FRAME_X := 18.0
const PARTY_FRAME_W := 198.0
const PARTY_FRAME_H := 48.0
const PARTY_FRAME_GAP := 6.0
const PARTY_MAX_VISIBLE := 4

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

# -- Channel bar --
var _channel_progress: float = 0.0  # 0.0 = just started, 1.0 = complete
var _channel_spell_name: String = ""
var _channel_active: bool = false

# -- Party frames --
var _party_data: Array = []  # Array[Dictionary] with: peer_id, name, health, max_health, class_name
var _hovered_party_index: int = -1
var _party_frame_rects: Array[Rect2] = []  # cached rects for hover detection

@onready var damage_overlay: ColorRect = $DamageOverlay
@onready var lock_on_reticle: Control = $LockOnReticle
@onready var ability_bar = $AbilityBar


func _ready() -> void:
	damage_overlay.modulate.a = 0.0
	ability_bar.accent_color = ARCANOTECHNICIEN_COLOR
	# Enable mouse pass-through but track mouse for party frame hover
	mouse_filter = Control.MOUSE_FILTER_PASS


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
	_draw_confluence()
	_draw_flux_bar()
	_draw_party_frames()
	_draw_channel_bar()


# =============================================================================
# Public API
# =============================================================================


func update_confluence(tier: int, stacks: int) -> void:
	_confluence_tier = tier
	_confluence_stacks = stacks


func update_spells(spells: Array) -> void:
	ability_bar.update_spells(spells)


func update_gcd(ratio: float) -> void:
	_gcd_ratio = ratio
	ability_bar.update_gcd(ratio)


func update_flux(current: float, max_value: float) -> void:
	_flux_current = current
	_flux_max = max_value


func update_channel(progress: float, spell_name: String) -> void:
	_channel_progress = clampf(progress, 0.0, 1.0)
	_channel_spell_name = spell_name
	_channel_active = progress > 0.0 and progress < 1.0


func hide_channel() -> void:
	_channel_active = false
	_channel_progress = 0.0
	_channel_spell_name = ""


func update_party(party: Array) -> void:
	_party_data = party


## Returns the peer_id of the party member under the mouse, or -1 if none.
func get_mouseover_target() -> int:
	if _hovered_party_index < 0 or _hovered_party_index >= _party_data.size():
		return -1
	return _party_data[_hovered_party_index].get("peer_id", -1)


func show_damage_flash() -> void:
	_damage_flash_timer = DAMAGE_FLASH_DURATION


func show_heal_flash() -> void:
	_heal_flash_timer = HEAL_FLASH_DURATION


func show_hit_marker() -> void:
	_hit_marker_timer = HIT_MARKER_DURATION


func show_lock_on() -> void:
	lock_on_reticle.visible = true
	lock_on_reticle._lock_active = true


func hide_lock_on() -> void:
	lock_on_reticle._lock_active = false
	lock_on_reticle.remove_meta("lock_target")
	lock_on_reticle.remove_meta("lock_camera")


func update_lock_on(target: Node3D, cam: Camera3D) -> void:
	lock_on_reticle.set_meta("lock_target", target)
	lock_on_reticle.set_meta("lock_camera", cam)


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
		center + Vector2(gap + x_len, -gap - x_len),
		center + Vector2(gap, -gap),
		color,
		thick,
		true
	)
	draw_line(
		center + Vector2(-gap - x_len, gap + x_len),
		center + Vector2(-gap, gap),
		color,
		thick,
		true
	)
	draw_line(
		center + Vector2(gap + x_len, gap + x_len),
		center + Vector2(gap, gap),
		color,
		thick,
		true
	)


# =============================================================================
# Drawing -- Confluence (5-stack mechanic, center screen above ability bar)
# =============================================================================


func _draw_confluence() -> void:
	var font := ThemeDB.fallback_font
	var center_x := size.x / 2.0
	var y := size.y - 160.0

	# --- Stack counter (big number) ---
	var count_text := "%d" % _confluence_stacks
	var count_color: Color
	if _confluence_tier >= 2:
		count_color = CONFLUENCE_MAX
		count_color.a = 0.7 + 0.3 * absf(sin(float(Time.get_ticks_msec()) / 300.0))
	elif _confluence_tier == 1:
		count_color = CONFLUENCE_ACTIVE
	else:
		count_color = Color(0.35, 0.5, 0.6, 0.6) if _confluence_stacks > 0 else CONFLUENCE_DIM

	draw_string(
		font,
		Vector2(center_x - 10.0, y + 6.0),
		count_text,
		HORIZONTAL_ALIGNMENT_CENTER,
		20.0,
		24,
		count_color
	)

	# --- Tier label ---
	var label := "CONFLUENCE"
	var label_color := CONFLUENCE_DIM
	if _confluence_tier == 1:
		label_color = CONFLUENCE_ACTIVE
	elif _confluence_tier >= 2:
		label_color = CONFLUENCE_MAX
		label_color.a = 0.7 + 0.3 * absf(sin(float(Time.get_ticks_msec()) / 300.0))
	elif _confluence_stacks > 0:
		label_color = Color(0.35, 0.5, 0.6, 0.6)

	draw_string(
		font,
		Vector2(center_x + 14.0, y - 6.0),
		label,
		HORIZONTAL_ALIGNMENT_LEFT,
		90.0,
		11,
		label_color
	)

	# --- Tier pips (5 dots for max 5 Confluence stacks) ---
	var pip_w := 10.0
	var pip_h := 3.0
	var pip_gap := 4.0
	var pip_count := 5
	var total_w := pip_w * pip_count + pip_gap * (pip_count - 1)
	var pip_x := center_x - total_w / 2.0
	var pip_y := y + 12.0

	for i in pip_count:
		var px := pip_x + i * (pip_w + pip_gap)
		var pip_rect := Rect2(px, pip_y, pip_w, pip_h)
		var lit: bool = i < _confluence_stacks

		if lit:
			var pip_color: Color
			if _confluence_tier >= 2:
				pip_color = CONFLUENCE_MAX
				pip_color.a = 0.7 + 0.3 * absf(sin(float(Time.get_ticks_msec()) / 300.0))
			elif _confluence_tier == 1:
				pip_color = CONFLUENCE_ACTIVE
			else:
				pip_color = Color(0.3, 0.65, 0.85, 0.8)
			draw_rect(pip_rect, pip_color)
		else:
			draw_rect(pip_rect, CONFLUENCE_DIM)


# =============================================================================
# Drawing -- Flux Bar (bottom center, below the shared_hud HP bar area)
# =============================================================================


func _draw_flux_bar() -> void:
	var font := ThemeDB.fallback_font
	var center_x := size.x / 2.0
	var bar_w := 248.0
	var bar_h := 10.0
	# Position just below where shared_hud draws the player HP bar (size.y - 126 + 14 gap)
	var bar_x := center_x - bar_w / 2.0
	var bar_y := size.y - 108.0

	var ratio := clampf(_flux_current / maxf(_flux_max, 1.0), 0.0, 1.0)

	# Flux color shifts to purple when low (below 20%)
	var fill_color: Color
	if ratio < 0.2:
		fill_color = FLUX_LOW_COLOR
	else:
		fill_color = FLUX_COLOR

	# Background
	draw_rect(Rect2(bar_x, bar_y, bar_w, bar_h), PANEL_BG)

	# Fill
	if ratio > 0.0:
		var fill_w := bar_w * ratio
		draw_rect(Rect2(bar_x, bar_y, fill_w, bar_h), fill_color)

	# Border
	draw_rect(Rect2(bar_x, bar_y, bar_w, bar_h), PANEL_BORDER, false, 1.0)

	# Label
	draw_string(
		font,
		Vector2(bar_x + 6.0, bar_y + 9.0),
		"FLUX",
		HORIZONTAL_ALIGNMENT_LEFT,
		40.0,
		8,
		TEXT_MUTED
	)

	# Value text
	var flux_text := "%d / %d" % [int(_flux_current), int(_flux_max)]
	draw_string(
		font,
		Vector2(bar_x + bar_w - 80.0, bar_y + 9.0),
		flux_text,
		HORIZONTAL_ALIGNMENT_RIGHT,
		74.0,
		9,
		TEXT_PRIMARY
	)


# =============================================================================
# Drawing -- Party Frames (left side, vertical list of ally HP bars)
# =============================================================================


func _draw_party_frames() -> void:
	if _party_data.is_empty():
		return

	var font := ThemeDB.fallback_font
	var frame_y := size.y * 0.5 - 120.0

	_party_frame_rects.clear()

	var drawn: int = 0
	for i in _party_data.size():
		if drawn >= PARTY_MAX_VISIBLE:
			break

		var member: Dictionary = _party_data[i]
		var pid: int = member.get("peer_id", 0)
		var uname: String = member.get("name", "Player_%d" % pid)
		var health: float = member.get("health", 0.0)
		var max_health: float = member.get("max_health", 150.0)
		var cls: String = member.get("class_name", "unknown")

		var y := frame_y + drawn * (PARTY_FRAME_H + PARTY_FRAME_GAP)
		var frame_rect := Rect2(PARTY_FRAME_X, y, PARTY_FRAME_W, PARTY_FRAME_H)
		_party_frame_rects.append(frame_rect)

		# Background -- slightly brighter on hover
		var is_hovered: bool = (i == _hovered_party_index)
		var bg_color := Color(0.06, 0.08, 0.12, 0.65) if is_hovered else Color(0.03, 0.04, 0.06, 0.5)
		draw_rect(frame_rect, bg_color)

		# Border -- accent on hover
		var border_color := ARCANOTECHNICIEN_COLOR if is_hovered else Color(PANEL_BORDER, 0.4)
		draw_rect(frame_rect, border_color, false, 1.0 if not is_hovered else 1.5)

		# Name (truncated)
		if uname.length() > 14:
			uname = uname.substr(0, 14)
		draw_string(
			font,
			Vector2(PARTY_FRAME_X + 6.0, y + 14.0),
			uname,
			HORIZONTAL_ALIGNMENT_LEFT,
			PARTY_FRAME_W - 70.0,
			10,
			TEXT_PRIMARY
		)

		# Class label (right side of name row)
		draw_string(
			font,
			Vector2(PARTY_FRAME_X + PARTY_FRAME_W - 60.0, y + 14.0),
			cls.replace("_", " ").to_upper(),
			HORIZONTAL_ALIGNMENT_RIGHT,
			56.0,
			8,
			TEXT_MUTED
		)

		# HP bar
		var hp_bar_x := PARTY_FRAME_X + 6.0
		var hp_bar_y := y + 24.0
		var hp_bar_w := PARTY_FRAME_W - 12.0
		var hp_bar_h := 16.0
		var hp_ratio := clampf(health / maxf(max_health, 1.0), 0.0, 1.0)
		var bar_color := HEALTH_GOOD if hp_ratio > 0.3 else HEALTH_BAD

		# Bar background
		draw_rect(Rect2(hp_bar_x, hp_bar_y, hp_bar_w, hp_bar_h), PANEL_BG)
		# Bar fill
		if hp_ratio > 0.0:
			draw_rect(
				Rect2(hp_bar_x, hp_bar_y, hp_bar_w * hp_ratio, hp_bar_h), bar_color
			)
		# Bar border
		draw_rect(Rect2(hp_bar_x, hp_bar_y, hp_bar_w, hp_bar_h), PANEL_BORDER, false, 1.0)

		# HP text
		var hp_text := "%d / %d" % [int(health), int(max_health)]
		draw_string(
			font,
			Vector2(hp_bar_x + hp_bar_w - 84.0, hp_bar_y + 13.0),
			hp_text,
			HORIZONTAL_ALIGNMENT_RIGHT,
			78.0,
			9,
			TEXT_PRIMARY
		)

		drawn += 1


func _update_party_hover() -> void:
	_hovered_party_index = -1
	var mouse_pos := get_local_mouse_position()
	for i in _party_frame_rects.size():
		if _party_frame_rects[i].has_point(mouse_pos):
			_hovered_party_index = i
			break


# =============================================================================
# Drawing -- Channel Bar (center screen, above confluence)
# =============================================================================


func _draw_channel_bar() -> void:
	if not _channel_active:
		return

	var font := ThemeDB.fallback_font
	var center_x := size.x / 2.0
	var bar_w := 220.0
	var bar_h := 16.0
	# Position above confluence display
	var bar_x := center_x - bar_w / 2.0
	var bar_y := size.y - 200.0

	# Background
	draw_rect(Rect2(bar_x, bar_y, bar_w, bar_h), CHANNEL_BG)

	# Fill (progresses left to right)
	if _channel_progress > 0.0:
		var fill_w := bar_w * _channel_progress
		draw_rect(Rect2(bar_x, bar_y, fill_w, bar_h), CHANNEL_FILL)

	# Border
	draw_rect(Rect2(bar_x, bar_y, bar_w, bar_h), PANEL_BORDER, false, 1.0)

	# Spell name (centered above bar)
	if _channel_spell_name != "":
		draw_string(
			font,
			Vector2(bar_x, bar_y - 4.0),
			_channel_spell_name,
			HORIZONTAL_ALIGNMENT_CENTER,
			bar_w,
			11,
			ARCANOTECHNICIEN_COLOR
		)

	# Progress percentage (inside bar, right-aligned)
	var pct_text := "%d%%" % int(_channel_progress * 100.0)
	draw_string(
		font,
		Vector2(bar_x + bar_w - 40.0, bar_y + 13.0),
		pct_text,
		HORIZONTAL_ALIGNMENT_RIGHT,
		36.0,
		10,
		TEXT_PRIMARY
	)
