class_name MeterRenderer
extends RefCounted

## Damage and healing meter drawing logic extracted from shared_hud.gd.
## Stateless renderer — all data is passed in via method parameters.

# --- Colors ---
const HUD_BG := Color(0.02, 0.025, 0.035, 0.82)
const HUD_BORDER := Color(0.28, 0.3, 0.36, 0.85)
const TEXT_PRIMARY := Color(0.9, 0.92, 0.96, 0.95)
const TEXT_MUTED := Color(0.63, 0.67, 0.74, 0.92)

const CLASS_COLORS := {
	"gunner": Color(0.24, 0.62, 0.95),
	"vanguard": Color(0.82, 0.44, 0.24),
	"blade_dancer": Color(0.36, 0.82, 0.66),
	"arcanotechnicien": Color(0.3, 0.65, 0.85),
}

# =============================================================================
# Damage Meter
# =============================================================================


static func draw_damage_meter(
	ctrl: Control,
	damage_totals: Dictionary,
	fight_duration: float,
	player_names: Dictionary,
) -> void:
	if damage_totals.is_empty():
		return

	var font := ThemeDB.fallback_font
	var meter_w := 188.0
	var meter_x := ctrl.size.x - meter_w - 18.0
	var entry_h := 18.0
	var title_y := ctrl.size.y - 154.0

	var sorted_pids: Array = damage_totals.keys()
	sorted_pids.sort_custom(func(a, b): return damage_totals[a] > damage_totals[b])
	var max_damage: float = maxf(damage_totals.get(sorted_pids[0], 1.0), 1.0)
	var entry_count := mini(sorted_pids.size(), 5)

	var title := _build_damage_title(damage_totals, fight_duration)
	ctrl.draw_string(
		font,
		Vector2(meter_x, title_y + 8.0),
		title,
		HORIZONTAL_ALIGNMENT_LEFT,
		meter_w,
		10,
		TEXT_MUTED
	)

	for i in entry_count:
		var pid: int = sorted_pids[i]
		var dmg: float = damage_totals[pid]
		var y := title_y + 20.0 + i * entry_h
		var entry_rect := Rect2(meter_x, y, meter_w, entry_h)
		var entry := {pid = pid, ratio = dmg / max_damage, text = _format_amount(dmg)}
		_draw_meter_entry(ctrl, font, entry, entry_rect, player_names)


static func _build_damage_title(damage_totals: Dictionary, fight_duration: float) -> String:
	if fight_duration <= 0.0:
		return "Damage"
	var total_dmg: float = 0.0
	for pid in damage_totals:
		total_dmg += damage_totals[pid]
	return "Damage (%.0f DPS)" % (total_dmg / maxf(fight_duration, 1.0))


# =============================================================================
# Healing Meter
# =============================================================================


static func draw_healing_meter(
	ctrl: Control,
	healing_totals: Dictionary,
	overheal_totals: Dictionary,
	damage_totals: Dictionary,
	fight_duration: float,
	player_names: Dictionary,
) -> void:
	if healing_totals.is_empty():
		return

	var font := ThemeDB.fallback_font
	var meter_w := 188.0
	var meter_x := ctrl.size.x - meter_w - 18.0
	var entry_h := 18.0

	var sorted_pids: Array = healing_totals.keys()
	sorted_pids.sort_custom(func(a, b): return healing_totals[a] > healing_totals[b])
	var max_heal: float = maxf(healing_totals.get(sorted_pids[0], 1.0), 1.0)
	var entry_count := mini(sorted_pids.size(), 5)

	var dmg_entry_count := mini(damage_totals.size(), 5) if not damage_totals.is_empty() else 0
	var dmg_height := (20.0 + dmg_entry_count * entry_h + 10.0) if dmg_entry_count > 0 else 0.0
	var title_y := ctrl.size.y - 154.0 - dmg_height

	var title := _build_healing_title(healing_totals, overheal_totals, fight_duration)
	ctrl.draw_string(
		font,
		Vector2(meter_x, title_y + 8.0),
		title,
		HORIZONTAL_ALIGNMENT_LEFT,
		meter_w,
		10,
		TEXT_MUTED
	)

	for i in entry_count:
		var pid: int = sorted_pids[i]
		var heal: float = healing_totals[pid]
		var oh: float = overheal_totals.get(pid, 0.0)
		var y := title_y + 20.0 + i * entry_h
		var entry_rect := Rect2(meter_x, y, meter_w, entry_h)
		var entry := {pid = pid, ratio = heal / max_heal, text = _format_heal_amount(heal, oh)}
		_draw_meter_entry(ctrl, font, entry, entry_rect, player_names)


static func _build_healing_title(
	healing_totals: Dictionary, overheal_totals: Dictionary, fight_duration: float
) -> String:
	if fight_duration <= 0.0:
		return "Healing"
	var total_heal: float = 0.0
	var total_overheal: float = 0.0
	for pid in healing_totals:
		total_heal += healing_totals[pid]
	for pid in overheal_totals:
		total_overheal += overheal_totals[pid]
	var hps := total_heal / maxf(fight_duration, 1.0)
	var oh_pct := 0
	if total_heal + total_overheal > 0.0:
		oh_pct = int(total_overheal / (total_heal + total_overheal) * 100.0)
	return "Healing (%.0f HPS, %d%% OH)" % [hps, oh_pct]


static func _format_heal_amount(heal: float, oh: float) -> String:
	var text := _format_amount(heal)
	if oh > 0.0:
		text += " (%s)" % _format_amount(oh)
	return text


# =============================================================================
# Shared drawing utilities
# =============================================================================


static func _format_amount(value: float) -> String:
	if value >= 1000.0:
		return "%.1fk" % (value / 1000.0)
	return "%d" % int(value)


static func _draw_meter_entry(
	ctrl: Control,
	font: Font,
	entry: Dictionary,
	rect: Rect2,
	player_names: Dictionary,
) -> void:
	var pid: int = entry.pid
	var ratio: float = entry.ratio
	var value_text: String = entry.text
	var meter_x := rect.position.x
	var y := rect.position.y
	var meter_w := rect.size.x
	var entry_h := rect.size.y

	var cls: String = "gunner"
	if NetworkManager.player_info.has(pid):
		cls = NetworkManager.player_info[pid].get("class_name", "gunner")
	var bar_color: Color = CLASS_COLORS.get(cls, Color(0.5, 0.5, 0.5))
	_draw_status_bar(
		ctrl, Rect2(meter_x, y + 2.0, meter_w, entry_h - 6.0), ratio, Color(bar_color, 0.92)
	)

	var uname: String = player_names.get(pid, "Player_%d" % pid)
	if uname.length() > 10:
		uname = uname.substr(0, 10)
	ctrl.draw_string(
		font,
		Vector2(meter_x + 4.0, y + 14.0),
		uname,
		HORIZONTAL_ALIGNMENT_LEFT,
		meter_w * 0.45,
		10,
		TEXT_PRIMARY
	)
	ctrl.draw_string(
		font,
		Vector2(meter_x + meter_w - 70.0, y + 14.0),
		value_text,
		HORIZONTAL_ALIGNMENT_RIGHT,
		66,
		10,
		TEXT_PRIMARY
	)


static func _draw_status_bar(ctrl: Control, rect: Rect2, ratio: float, fill_color: Color) -> void:
	ctrl.draw_rect(rect, HUD_BG)
	if ratio > 0.0:
		var fill_width := maxf(rect.size.x * ratio, 0.0)
		ctrl.draw_rect(Rect2(rect.position, Vector2(fill_width, rect.size.y)), fill_color)
	ctrl.draw_rect(rect, HUD_BORDER, false, 1.0)
