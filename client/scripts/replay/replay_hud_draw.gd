class_name ReplayHudDraw
## Static drawing helpers for replay HUD overlay elements.
## Renders player status bars, boss frame, and damage meter onto a Control node.

const CLASS_MAX_HP := {
	"gunner": 150.0,
	"vanguard": 200.0,
	"blade_dancer": 150.0,
}
const CLASS_COLORS := {
	"gunner": Color(0.24, 0.62, 0.95),
	"vanguard": Color(0.82, 0.44, 0.24),
	"blade_dancer": Color(0.36, 0.82, 0.66),
}


## Draw player HP bars in the top-left corner.
static func draw_player_status(overlay: Control, players: Array) -> void:
	var x := 12.0
	var y := 12.0
	var panel_h := 38.0  # name_h(18) + bar_h(16) + 4
	var gap := 4.0
	for pdata in players:
		_draw_single_player_bar(overlay, pdata, x, y)
		y += panel_h + gap


static func _draw_single_player_bar(
	overlay: Control, pdata: Dictionary, x: float, y: float
) -> void:
	var font := ThemeDB.fallback_font
	var panel_w := 200.0
	var name_h := 18.0
	var bar_h := 16.0
	var panel_h := name_h + bar_h + 4.0

	var hp: float = pdata.get("health", 0.0)
	var cls: String = pdata.get("class_name", "gunner")
	var max_hp: float = CLASS_MAX_HP.get(cls, 150.0)
	var username: String = pdata.get("username", "Player")
	var ratio := clampf(hp / maxf(max_hp, 1.0), 0.0, 1.0)
	var dead := hp <= 0.0

	overlay.draw_rect(Rect2(x, y, panel_w, panel_h), Color(0.05, 0.05, 0.08, 0.85))
	var name_color := Color(0.4, 0.4, 0.4) if dead else Color.WHITE
	overlay.draw_string(
		font,
		Vector2(x + 6, y + 14),
		username,
		HORIZONTAL_ALIGNMENT_LEFT,
		panel_w - 12,
		12,
		name_color
	)

	var bar_y := y + name_h
	overlay.draw_rect(Rect2(x + 4, bar_y, panel_w - 8, bar_h), Color(0.15, 0.15, 0.15, 0.9))
	if not dead:
		var bar_color := Color(0.8, 0.2, 0.2) if ratio <= 0.3 else Color(0.2, 0.75, 0.3)
		overlay.draw_rect(Rect2(x + 4, bar_y, (panel_w - 8) * ratio, bar_h), bar_color)

	var class_label := cls.replace("_", " ").to_upper()
	var text_color := Color(0.5, 0.5, 0.5) if dead else Color(0.8, 0.8, 0.8, 0.7)
	overlay.draw_string(
		font, Vector2(x + 8, bar_y + 12), class_label, HORIZONTAL_ALIGNMENT_LEFT, 80, 9, text_color
	)
	var hp_text := "DEAD" if dead else "%d / %d" % [int(hp), int(max_hp)]
	var hp_color := Color(0.6, 0.2, 0.2) if dead else Color.WHITE
	overlay.draw_string(
		font,
		Vector2(x + panel_w - 100, bar_y + 12),
		hp_text,
		HORIZONTAL_ALIGNMENT_RIGHT,
		88,
		9,
		hp_color
	)


## Draw boss health bar centered at the top.
static func draw_boss_frame(
	overlay: Control,
	boss_name: String,
	boss_health: float,
	boss_max_health: float,
	boss_phase: int,
) -> void:
	var font := ThemeDB.fallback_font
	var vp_size := overlay.get_viewport_rect().size
	var panel_x := vp_size.x / 2.0 - 216.0
	var panel_w := 432.0
	var panel_y := 14.0

	_draw_boss_header(overlay, font, panel_x, panel_w, panel_y, boss_name, boss_phase)
	_draw_boss_health_bar(
		overlay, font, panel_x, panel_w, panel_y, boss_health, boss_max_health, boss_phase
	)


static func _draw_boss_header(
	overlay: Control,
	font: Font,
	px: float,
	pw: float,
	py: float,
	boss_name: String,
	boss_phase: int,
) -> void:
	overlay.draw_string(
		font,
		Vector2(px, py + 9.0),
		boss_name,
		HORIZONTAL_ALIGNMENT_LEFT,
		240.0,
		12,
		Color(0.93, 0.9, 0.8, 0.97)
	)
	var phase_color: Color
	match boss_phase:
		1:
			phase_color = Color(0.56, 0.74, 0.28)
		2:
			phase_color = Color(0.93, 0.7, 0.25)
		3:
			phase_color = Color(0.93, 0.34, 0.34)
		_:
			phase_color = Color(0.5, 0.5, 0.5)
	overlay.draw_string(
		font,
		Vector2(px + pw - 36.0, py + 9.0),
		"P%d" % boss_phase,
		HORIZONTAL_ALIGNMENT_RIGHT,
		32.0,
		11,
		phase_color
	)


static func _draw_boss_health_bar(
	overlay: Control,
	font: Font,
	px: float,
	pw: float,
	py: float,
	boss_health: float,
	boss_max_health: float,
	boss_phase: int,
) -> void:
	var bar_rect := Rect2(px, py + 14.0, pw, 12.0)
	var hp_ratio := clampf(boss_health / maxf(boss_max_health, 1.0), 0.0, 1.0)
	var bar_color: Color
	match boss_phase:
		1:
			bar_color = Color(0.56, 0.22, 0.22)
		2:
			bar_color = Color(0.74, 0.44, 0.18)
		3:
			bar_color = Color(0.78, 0.18, 0.18)
		_:
			bar_color = Color(0.5, 0.5, 0.5)

	overlay.draw_rect(bar_rect, Color(0.08, 0.08, 0.1, 0.9))
	if hp_ratio > 0.0:
		overlay.draw_rect(
			Rect2(bar_rect.position, Vector2(bar_rect.size.x * hp_ratio, bar_rect.size.y)),
			bar_color
		)
	overlay.draw_rect(bar_rect, Color(0.25, 0.25, 0.3, 0.8), false, 1.0)
	var hp_text := "%d / %d" % [int(boss_health), int(boss_max_health)]
	overlay.draw_string(
		font,
		Vector2(bar_rect.position.x + bar_rect.size.x - 118.0, bar_rect.position.y + 12.0),
		hp_text,
		HORIZONTAL_ALIGNMENT_RIGHT,
		110.0,
		10,
		Color(0.9, 0.92, 0.96, 0.95)
	)


## Draw the damage meter in the top-right corner.
static func draw_damage_meter(
	overlay: Control,
	damage_totals: Dictionary,
	current_tick: int,
	tick_rate: int,
	replay: Variant,
) -> void:
	if damage_totals.is_empty():
		return

	var font := ThemeDB.fallback_font
	var vp_size := overlay.get_viewport_rect().size
	var meter_w := 200.0
	var meter_x := vp_size.x - meter_w - 12.0
	var entry_h := 20.0
	var title_h := 22.0
	var y := 12.0

	var sorted_ids: Array = damage_totals.keys()
	sorted_ids.sort_custom(func(a, b): return damage_totals[a] > damage_totals[b])
	var max_damage: float = damage_totals.get(sorted_ids[0], 1.0) if sorted_ids.size() > 0 else 1.0
	if max_damage <= 0.0:
		max_damage = 1.0

	var entry_count := mini(sorted_ids.size(), 5)
	var panel_h := title_h + entry_count * entry_h + 4.0
	overlay.draw_rect(Rect2(meter_x, y, meter_w, panel_h), Color(0.05, 0.05, 0.08, 0.85))

	var elapsed: float = float(current_tick) / float(tick_rate)
	overlay.draw_string(
		font,
		Vector2(meter_x + 6, y + 15),
		_build_title(damage_totals, elapsed),
		HORIZONTAL_ALIGNMENT_LEFT,
		meter_w - 12,
		11,
		Color(0.7, 0.7, 0.7)
	)

	for i in entry_count:
		var ey := y + title_h + i * entry_h
		_draw_meter_entry(
			overlay,
			font,
			sorted_ids[i],
			damage_totals,
			max_damage,
			elapsed,
			meter_x,
			meter_w,
			entry_h,
			ey,
			replay
		)


static func _draw_meter_entry(
	overlay: Control,
	font: Font,
	eid: String,
	damage_totals: Dictionary,
	max_damage: float,
	elapsed: float,
	mx: float,
	mw: float,
	eh: float,
	ey: float,
	replay: Variant,
) -> void:
	var dmg: float = damage_totals[eid]
	var cls: String = _get_participant_class(eid, replay)
	var bar_color: Color = CLASS_COLORS.get(cls, Color(0.5, 0.5, 0.5))
	var ratio := dmg / max_damage
	overlay.draw_rect(Rect2(mx + 4, ey + 2, (mw - 8) * ratio, eh - 4), Color(bar_color, 0.85))

	var pname: String = replay.get_participant_name(eid) if replay else eid
	if pname.length() > 12:
		pname = pname.substr(0, 12)
	overlay.draw_string(
		font, Vector2(mx + 8, ey + 15), pname, HORIZONTAL_ALIGNMENT_LEFT, mw * 0.5, 10, Color.WHITE
	)
	overlay.draw_string(
		font,
		Vector2(mx + mw - 94, ey + 15),
		_format_damage(dmg, elapsed),
		HORIZONTAL_ALIGNMENT_RIGHT,
		86,
		10,
		Color.WHITE
	)


static func _build_title(damage_totals: Dictionary, elapsed: float) -> String:
	if elapsed <= 0.0:
		return "Damage"
	var total_dmg: float = 0.0
	for eid in damage_totals:
		total_dmg += damage_totals[eid]
	var dps := total_dmg / maxf(elapsed, 1.0)
	return "Damage (%.0f DPS)" % dps


static func _format_damage(dmg: float, elapsed: float) -> String:
	var dmg_text: String
	if dmg >= 1000.0:
		dmg_text = "%.1fk" % (dmg / 1000.0)
	else:
		dmg_text = "%d" % int(dmg)
	if elapsed > 0.0:
		var pdps := dmg / maxf(elapsed, 1.0)
		dmg_text += " (%.0f)" % pdps
	return dmg_text


static func _get_participant_class(entity_id: String, replay: Variant) -> String:
	if replay == null:
		return "gunner"
	for p in replay.participants:
		if p.get("entity_id", "") == entity_id:
			return p.get("class", "gunner")
	return "gunner"
