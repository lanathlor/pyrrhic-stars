class_name ArcanotechnicienHudDrawHelpers
extends RefCounted

## Static drawing helpers for ArcanotechnicienHud.
## All methods take a CanvasItem (ci) as the first parameter so they can issue
## draw_* calls during the parent Control's _draw() callback.

# -- Panel / bar constants (visual language) --
const PANEL_BG := Color(0.02, 0.025, 0.035, 0.82)
const PANEL_BORDER := Color(0.28, 0.3, 0.36, 0.85)
const TEXT_PRIMARY := Color(0.92, 0.94, 0.97, 0.95)
const TEXT_MUTED := Color(0.66, 0.7, 0.77, 0.9)
const HEALTH_GOOD := Color(0.28, 0.78, 0.4, 1.0)
const HEALTH_BAD := Color(0.82, 0.24, 0.24, 1.0)
const FLUX_COLOR := Color(0.25, 0.55, 0.9, 1.0)
const FLUX_LOW_COLOR := Color(0.6, 0.3, 0.8, 1.0)

const ARCANOTECHNICIEN_COLOR := Color(0.3, 0.65, 0.85)

# -- School colors for segmented flux bar --
const SCHOOL_COLORS := {
	"bioarcanotechnic": Color(0.25, 0.55, 0.9, 1.0),
	"biometabolic": Color(0.45, 0.8, 0.35, 1.0),
	"frost": Color(0.6, 0.85, 0.95, 1.0),
	"aerokinetic": Color(0.8, 0.85, 0.65, 1.0),
}
const SCHOOL_LABELS := {
	"bioarcanotechnic": "BIO",
	"biometabolic": "META",
	"frost": "FROST",
	"aerokinetic": "AERO",
}

# -- Confluence colors --
const CONFLUENCE_DIM := Color(0.15, 0.25, 0.35, 0.4)
const CONFLUENCE_ACTIVE := Color(0.3, 0.65, 0.85, 0.95)
const CONFLUENCE_MAX := Color(0.5, 0.85, 1.0, 1.0)

# -- Channel bar colors --
const CHANNEL_BG := Color(0.04, 0.05, 0.07, 0.75)
const CHANNEL_FILL := Color(0.3, 0.65, 0.85, 0.9)
const SUSTAIN_FILL := Color(0.4, 0.8, 0.6, 0.9)

# -- Party frame layout --
const PARTY_FRAME_X := 18.0
const PARTY_FRAME_W := 198.0
const PARTY_FRAME_H := 48.0
const PARTY_FRAME_GAP := 6.0
const PARTY_MAX_VISIBLE := 4

# =============================================================================
# Confluence (5-stack mechanic, center screen above ability bar)
# =============================================================================


static func draw_confluence(
	ci: CanvasItem,
	hud_size: Vector2,
	confluence_tier: int,
	confluence_stacks: int,
) -> void:
	var font := ThemeDB.fallback_font
	var center_x := hud_size.x / 2.0
	var y := hud_size.y - 160.0

	var count_color := _confluence_color(confluence_tier, confluence_stacks)
	ci.draw_string(
		font,
		Vector2(center_x - 10.0, y + 6.0),
		"%d" % confluence_stacks,
		HORIZONTAL_ALIGNMENT_CENTER,
		20.0,
		24,
		count_color
	)

	var label_color := _confluence_color(confluence_tier, confluence_stacks)
	ci.draw_string(
		font,
		Vector2(center_x + 14.0, y - 6.0),
		"CONFLUENCE",
		HORIZONTAL_ALIGNMENT_LEFT,
		90.0,
		11,
		label_color
	)

	_draw_confluence_pips(ci, center_x, y + 12.0, confluence_tier, confluence_stacks)


# =============================================================================
# Flux Bar (bottom center, below the shared_hud HP bar area)
# =============================================================================


static func draw_flux_bar(
	ci: CanvasItem,
	hud_size: Vector2,
	flux_current: float,
	flux_max: float,
	flux_pools: Array,
) -> void:
	var font := ThemeDB.fallback_font
	var center_x := hud_size.x / 2.0
	var bar_w := 248.0
	var bar_h := 10.0
	var bar_x := center_x - bar_w / 2.0
	var bar_y := hud_size.y - 108.0

	if flux_pools.size() > 0:
		_draw_segmented_flux_bar(ci, font, bar_x, bar_y, bar_w, bar_h, flux_max, flux_pools)
	else:
		_draw_single_flux_bar(ci, font, bar_x, bar_y, bar_w, bar_h, flux_current, flux_max)


static func _draw_single_flux_bar(
	ci: CanvasItem,
	font: Font,
	bar_x: float,
	bar_y: float,
	bar_w: float,
	bar_h: float,
	flux_current: float,
	flux_max: float,
) -> void:
	var ratio := clampf(flux_current / maxf(flux_max, 1.0), 0.0, 1.0)

	# Flux color shifts to purple when low (below 20%)
	var fill_color: Color
	if ratio < 0.2:
		fill_color = FLUX_LOW_COLOR
	else:
		fill_color = FLUX_COLOR

	# Background
	ci.draw_rect(Rect2(bar_x, bar_y, bar_w, bar_h), PANEL_BG)

	# Fill
	if ratio > 0.0:
		var fill_w := bar_w * ratio
		ci.draw_rect(Rect2(bar_x, bar_y, fill_w, bar_h), fill_color)

	# Border
	ci.draw_rect(Rect2(bar_x, bar_y, bar_w, bar_h), PANEL_BORDER, false, 1.0)

	# Label
	ci.draw_string(
		font,
		Vector2(bar_x + 6.0, bar_y + 9.0),
		"FLUX",
		HORIZONTAL_ALIGNMENT_LEFT,
		40.0,
		8,
		TEXT_MUTED
	)

	# Value text
	var flux_text := "%d / %d" % [int(flux_current), int(flux_max)]
	ci.draw_string(
		font,
		Vector2(bar_x + bar_w - 80.0, bar_y + 9.0),
		flux_text,
		HORIZONTAL_ALIGNMENT_RIGHT,
		74.0,
		9,
		TEXT_PRIMARY
	)


static func _draw_segmented_flux_bar(
	ci: CanvasItem,
	font: Font,
	bar_x: float,
	bar_y: float,
	bar_w: float,
	bar_h: float,
	flux_max: float,
	flux_pools: Array,
) -> void:
	var total_max := maxf(flux_max, 1.0)
	ci.draw_rect(Rect2(bar_x, bar_y, bar_w, bar_h), PANEL_BG)

	var seg_x := bar_x
	var flux_current_sum := 0.0
	for i in range(flux_pools.size()):
		var pool: Dictionary = flux_pools[i]
		var current: float = pool.get("current", 0.0)
		flux_current_sum += current
		var seg_w: float = (pool.get("max", 0.0) / total_max) * bar_w
		if seg_w >= 1.0:
			_draw_flux_segment(ci, font, seg_x, bar_y, seg_w, bar_h, pool, i > 0)
		seg_x += seg_w

	ci.draw_rect(Rect2(bar_x, bar_y, bar_w, bar_h), PANEL_BORDER, false, 1.0)
	var flux_text := "%d / %d" % [int(flux_current_sum), int(flux_max)]
	ci.draw_string(
		font,
		Vector2(bar_x + bar_w - 80.0, bar_y + 9.0),
		flux_text,
		HORIZONTAL_ALIGNMENT_RIGHT,
		74.0,
		9,
		TEXT_PRIMARY
	)


# =============================================================================
# Party Frames (left side, vertical list of ally HP bars)
# =============================================================================


## Draws party frames and returns the array of Rect2 for hover detection.
static func draw_party_frames(
	ci: CanvasItem,
	hud_size: Vector2,
	party_data: Array,
	selected_peer_id: int,
	hovered_party_index: int,
) -> Array[Rect2]:
	var rects: Array[Rect2] = []
	if party_data.is_empty():
		return rects

	var font := ThemeDB.fallback_font
	var frame_y := hud_size.y * 0.5 - 120.0

	var drawn: int = 0
	for i in party_data.size():
		if drawn >= PARTY_MAX_VISIBLE:
			break
		var member: Dictionary = party_data[i]
		var pid: int = member.get("peer_id", 0)
		var y := frame_y + drawn * (PARTY_FRAME_H + PARTY_FRAME_GAP)
		var frame_rect := Rect2(PARTY_FRAME_X, y, PARTY_FRAME_W, PARTY_FRAME_H)
		rects.append(frame_rect)
		var is_selected: bool = pid == selected_peer_id and selected_peer_id > 0
		var is_hovered: bool = i == hovered_party_index
		_draw_single_party_frame(ci, font, frame_rect, member, is_selected, is_hovered)
		drawn += 1

	return rects


# =============================================================================
# Channel Bar (center screen, above confluence)
# =============================================================================


static func draw_channel_bar(
	ci: CanvasItem,
	hud_size: Vector2,
	channel_active: bool,
	channel_progress: float,
	channel_ability_name: String,
	sustain_active: bool,
	sustain_elapsed: float,
) -> void:
	if not channel_active:
		return

	var font := ThemeDB.fallback_font
	var center_x := hud_size.x / 2.0
	var bar_w := 220.0
	var bar_h := 16.0
	var bar_x := center_x - bar_w / 2.0
	var bar_y := hud_size.y - 200.0
	var bar_rect := Rect2(bar_x, bar_y, bar_w, bar_h)

	ci.draw_rect(bar_rect, CHANNEL_BG)
	_draw_channel_fill(ci, bar_rect, sustain_active, channel_progress)
	ci.draw_rect(bar_rect, PANEL_BORDER, false, 1.0)
	_draw_channel_label(ci, font, bar_x, bar_y, bar_w, channel_ability_name, sustain_active)
	_draw_channel_status(
		ci, font, bar_x, bar_y, bar_w, sustain_active, sustain_elapsed, channel_progress
	)


# =============================================================================
# Private helpers
# =============================================================================


static func _confluence_color(tier: int, stacks: int) -> Color:
	if tier >= 2:
		var c := CONFLUENCE_MAX
		c.a = 0.7 + 0.3 * absf(sin(float(Time.get_ticks_msec()) / 300.0))
		return c
	if tier == 1:
		return CONFLUENCE_ACTIVE
	if stacks > 0:
		return Color(0.35, 0.5, 0.6, 0.6)
	return CONFLUENCE_DIM


static func _draw_confluence_pips(
	ci: CanvasItem, center_x: float, pip_y: float, tier: int, stacks: int
) -> void:
	var pip_w := 10.0
	var pip_h := 3.0
	var pip_gap := 4.0
	var pip_count := 5
	var total_w := pip_w * pip_count + pip_gap * (pip_count - 1)
	var pip_x := center_x - total_w / 2.0
	for i in pip_count:
		var px := pip_x + i * (pip_w + pip_gap)
		var pip_rect := Rect2(px, pip_y, pip_w, pip_h)
		if i < stacks:
			ci.draw_rect(pip_rect, _confluence_pip_color(tier))
		else:
			ci.draw_rect(pip_rect, CONFLUENCE_DIM)


static func _confluence_pip_color(tier: int) -> Color:
	if tier >= 2:
		var c := CONFLUENCE_MAX
		c.a = 0.7 + 0.3 * absf(sin(float(Time.get_ticks_msec()) / 300.0))
		return c
	if tier == 1:
		return CONFLUENCE_ACTIVE
	return Color(0.3, 0.65, 0.85, 0.8)


static func _draw_flux_segment(
	ci: CanvasItem,
	font: Font,
	seg_x: float,
	bar_y: float,
	seg_w: float,
	bar_h: float,
	pool: Dictionary,
	draw_sep: bool,
) -> void:
	var school: String = pool.get("school", "")
	var current: float = pool.get("current", 0.0)
	var pool_max: float = pool.get("max", 0.0)
	var fill_ratio := clampf(current / maxf(pool_max, 0.1), 0.0, 1.0)
	var school_color: Color = SCHOOL_COLORS.get(school, FLUX_COLOR)
	if fill_ratio < 0.2 and fill_ratio > 0.0:
		school_color = school_color.lerp(FLUX_LOW_COLOR, 0.5)
	if fill_ratio > 0.0:
		ci.draw_rect(Rect2(seg_x, bar_y, seg_w * fill_ratio, bar_h), school_color)
	if draw_sep:
		ci.draw_line(Vector2(seg_x, bar_y), Vector2(seg_x, bar_y + bar_h), PANEL_BORDER, 1.0)
	var label: String = SCHOOL_LABELS.get(school, "")
	if label != "" and seg_w > 24.0:
		ci.draw_string(
			font,
			Vector2(seg_x + 2.0, bar_y - 2.0),
			label,
			HORIZONTAL_ALIGNMENT_LEFT,
			seg_w - 4.0,
			7,
			TEXT_MUTED.lerp(school_color, 0.4)
		)


static func _draw_single_party_frame(
	ci: CanvasItem,
	font: Font,
	frame_rect: Rect2,
	member: Dictionary,
	is_selected: bool,
	is_hovered: bool,
) -> void:
	var bg_color: Color
	if is_selected:
		bg_color = Color(0.1, 0.15, 0.25, 0.75)
	elif is_hovered:
		bg_color = Color(0.06, 0.08, 0.12, 0.65)
	else:
		bg_color = Color(0.03, 0.04, 0.06, 0.5)
	ci.draw_rect(frame_rect, bg_color)

	var border_color: Color
	var border_width: float
	if is_selected:
		border_color = Color(0.4, 0.75, 1.0, 0.95)
		border_width = 2.0
	elif is_hovered:
		border_color = ARCANOTECHNICIEN_COLOR
		border_width = 1.5
	else:
		border_color = Color(PANEL_BORDER, 0.4)
		border_width = 1.0
	ci.draw_rect(frame_rect, border_color, false, border_width)

	var y := frame_rect.position.y
	var pid: int = member.get("peer_id", 0)
	var uname: String = member.get("name", "Player_%d" % pid)
	if uname.length() > 14:
		uname = uname.substr(0, 14)
	ci.draw_string(
		font,
		Vector2(PARTY_FRAME_X + 6.0, y + 14.0),
		uname,
		HORIZONTAL_ALIGNMENT_LEFT,
		PARTY_FRAME_W - 70.0,
		10,
		TEXT_PRIMARY
	)
	var cls: String = member.get("class_name", "unknown")
	ci.draw_string(
		font,
		Vector2(PARTY_FRAME_X + PARTY_FRAME_W - 60.0, y + 14.0),
		cls.replace("_", " ").to_upper(),
		HORIZONTAL_ALIGNMENT_RIGHT,
		56.0,
		8,
		TEXT_MUTED
	)
	_draw_party_hp_bar(ci, font, y, member)


static func _draw_party_hp_bar(ci: CanvasItem, font: Font, y: float, member: Dictionary) -> void:
	var health: float = member.get("health", 0.0)
	var max_health: float = member.get("max_health", 150.0)
	var hp_bar_x := PARTY_FRAME_X + 6.0
	var hp_bar_y := y + 24.0
	var hp_bar_w := PARTY_FRAME_W - 12.0
	var hp_bar_h := 16.0
	var hp_ratio := clampf(health / maxf(max_health, 1.0), 0.0, 1.0)
	var bar_color := HEALTH_GOOD if hp_ratio > 0.3 else HEALTH_BAD
	ci.draw_rect(Rect2(hp_bar_x, hp_bar_y, hp_bar_w, hp_bar_h), PANEL_BG)
	if hp_ratio > 0.0:
		ci.draw_rect(Rect2(hp_bar_x, hp_bar_y, hp_bar_w * hp_ratio, hp_bar_h), bar_color)
	ci.draw_rect(Rect2(hp_bar_x, hp_bar_y, hp_bar_w, hp_bar_h), PANEL_BORDER, false, 1.0)
	var hp_text := "%d / %d" % [int(health), int(max_health)]
	ci.draw_string(
		font,
		Vector2(hp_bar_x + hp_bar_w - 84.0, hp_bar_y + 13.0),
		hp_text,
		HORIZONTAL_ALIGNMENT_RIGHT,
		78.0,
		9,
		TEXT_PRIMARY
	)


static func _draw_channel_fill(
	ci: CanvasItem, bar_rect: Rect2, sustain_active: bool, progress: float
) -> void:
	if sustain_active:
		var pulse := 0.7 + 0.3 * absf(sin(float(Time.get_ticks_msec()) / 400.0))
		var fill_color := SUSTAIN_FILL
		fill_color.a = pulse
		ci.draw_rect(bar_rect, fill_color)
	elif progress > 0.0:
		var fill_w := bar_rect.size.x * progress
		ci.draw_rect(
			Rect2(bar_rect.position.x, bar_rect.position.y, fill_w, bar_rect.size.y), CHANNEL_FILL
		)


static func _draw_channel_label(
	ci: CanvasItem,
	font: Font,
	bar_x: float,
	bar_y: float,
	bar_w: float,
	ability_name: String,
	sustain_active: bool,
) -> void:
	if ability_name == "":
		return
	var label := ability_name + " (SUSTAIN)" if sustain_active else ability_name
	var lc := SUSTAIN_FILL if sustain_active else ARCANOTECHNICIEN_COLOR
	ci.draw_string(
		font, Vector2(bar_x, bar_y - 4.0), label, HORIZONTAL_ALIGNMENT_CENTER, bar_w, 11, lc
	)


static func _draw_channel_status(
	ci: CanvasItem,
	font: Font,
	bar_x: float,
	bar_y: float,
	bar_w: float,
	sustain_active: bool,
	sustain_elapsed: float,
	progress: float,
) -> void:
	var status_text: String
	if sustain_active:
		status_text = "+%d%%" % int(sustain_elapsed * 5.0)
	else:
		status_text = "%d%%" % int(progress * 100.0)
	ci.draw_string(
		font,
		Vector2(bar_x + bar_w - 40.0, bar_y + 13.0),
		status_text,
		HORIZONTAL_ALIGNMENT_RIGHT,
		36.0,
		10,
		TEXT_PRIMARY
	)
