class_name CodexPanelDrawCards
extends RefCounted

## Draw helpers for Codex Panel: ability grid, individual cards, tooltip, drag ghost.

const T := preload("res://scenes/shared/hud/codex_panel_theme.gd")


static func draw_grid(ci: CanvasItem, s: Dictionary) -> void:
	var panel_rect: Rect2 = s.panel
	var grid_rect: Rect2 = s.grid
	var card_rects: Array = s.cards
	var filtered_abilities: Array = s.abilities
	var hovered_ability_idx: int = s.hovered
	var pending_loadout: Array = s.loadout
	var scroll_offset: float = s.scroll
	var max_scroll: float = s.max_scroll
	# Separator
	ci.draw_line(
		Vector2(panel_rect.position.x + 8, grid_rect.position.y - 2),
		Vector2(panel_rect.end.x - 8, grid_rect.position.y - 2),
		T.PANEL_BORDER,
		1.0
	)

	if filtered_abilities.is_empty():
		var font := ThemeDB.fallback_font
		var msg := (
			"Waiting for ability catalog..."
			if AbilityCatalog.catalog.is_empty()
			else "No abilities in this school."
		)
		ci.draw_string(
			font,
			Vector2(grid_rect.position.x + 20, grid_rect.position.y + 40),
			msg,
			HORIZONTAL_ALIGNMENT_LEFT,
			grid_rect.size.x - 40,
			12,
			T.TEXT_DIM
		)
		return

	for i in card_rects.size():
		var rect: Rect2 = card_rects[i]
		if rect.end.y < grid_rect.position.y or rect.position.y > grid_rect.end.y:
			continue
		if rect.position.y < grid_rect.position.y - 1 or rect.end.y > grid_rect.end.y + 1:
			continue
		if i < filtered_abilities.size():
			_draw_ability_card(
				ci, rect, filtered_abilities[i], i == hovered_ability_idx, pending_loadout
			)

	if max_scroll > 0:
		_draw_scroll_indicators(ci, grid_rect, scroll_offset, max_scroll)


static func _draw_ability_card(
	ci: CanvasItem,
	rect: Rect2,
	ability: Dictionary,
	hovered: bool,
	pending_loadout: Array[String],
) -> void:
	var font := ThemeDB.fallback_font
	var school: String = ability.get("school", "")
	var school_color := T.get_school_color(school)
	var ability_id: String = ability.get("id", "")
	var implemented: bool = ability.get("implemented", false)
	var affinity: String = ability.get("affinity", "off")
	var slot := T.get_slot_for_ability(ability_id, pending_loadout)
	var border_color := _affinity_border_color(affinity, school_color)

	# Background
	var bg := Color(0.04, 0.05, 0.07, 0.7)
	if hovered and implemented:
		bg = Color(0.08, 0.1, 0.14, 0.85)
	if slot >= 0:
		bg = Color(school_color.r * 0.15, school_color.g * 0.15, school_color.b * 0.15, 0.7)

	ci.draw_rect(rect, bg)
	ci.draw_rect(Rect2(rect.position.x, rect.position.y, 3, rect.size.y), border_color)
	ci.draw_rect(rect, border_color if hovered else T.PANEL_BORDER, false, 1.5 if hovered else 1.0)

	_draw_card_text(ci, font, rect, ability, implemented, school_color)
	_draw_slot_badge(ci, font, rect, slot)

	if not implemented:
		ci.draw_rect(rect, T.UNIMPL_OVERLAY)
		_draw_hatch(ci, rect)


static func _draw_hatch(ci: CanvasItem, rect: Rect2) -> void:
	var hatch_step := 18.0
	var hatch_color := Color(0.3, 0.3, 0.35, 0.25)
	var offset := 0.0
	while offset < rect.size.x + rect.size.y:
		var sx := rect.position.x + offset
		var sy := rect.position.y
		var ex := rect.position.x
		var ey := rect.position.y + offset
		if sx > rect.end.x:
			sy += sx - rect.end.x
			sx = rect.end.x
		if ey > rect.end.y:
			ex += ey - rect.end.y
			ey = rect.end.y
		if sy <= rect.end.y and ex <= rect.end.x:
			ci.draw_line(Vector2(sx, sy), Vector2(ex, ey), hatch_color, 1.0)
		offset += hatch_step


static func draw_tooltip(
	ci: CanvasItem,
	canvas_size: Vector2,
	card_rects: Array[Rect2],
	filtered_abilities: Array,
	hovered_ability_idx: int,
) -> void:
	var font := ThemeDB.fallback_font
	var ability: Dictionary = filtered_abilities[hovered_ability_idx]
	var d := _extract_tooltip_data(ability, font)

	var tip_pos := _compute_tooltip_pos(
		card_rects[hovered_ability_idx], canvas_size, d.tip_w, d.tip_h
	)
	var tip_x := tip_pos.x
	var tip_y := tip_pos.y

	ci.draw_rect(Rect2(tip_x, tip_y, d.tip_w, d.tip_h), Color(0.02, 0.025, 0.04, 0.95))
	ci.draw_rect(Rect2(tip_x, tip_y, d.tip_w, d.tip_h), Color(d.school_color, 0.4), false, 1.0)
	ci.draw_rect(Rect2(tip_x, tip_y, 4, d.tip_h), d.school_color)

	var ty := tip_y + 22
	_draw_tooltip_header(ci, font, tip_x, d.tip_w, ty, d.ability_name, d.school_color)
	ty += 22
	ty = _draw_tooltip_info(ci, font, tip_x, d.tip_w, ty, d.school, d.ability_type, d.delivery)
	ty = _draw_tooltip_stats(
		ci, font, tip_x, d.tip_w, ty, d.flux_amount, d.flux_cost, d.commit_time, d.cooldown
	)
	ty = _draw_tooltip_details(ci, font, tip_x, d.tip_w, ty, d.stat_details)
	ty = _draw_tooltip_affinity(ci, font, tip_x, d.tip_w, ty, d.affinity)
	_draw_tooltip_description(ci, font, tip_x, d.tip_w, ty, d.desc_lines, d.implemented)


static func _draw_tooltip_header(
	ci: CanvasItem,
	font: Font,
	tip_x: float,
	tip_w: float,
	ty: float,
	ability_name: String,
	school_color: Color,
) -> void:
	ci.draw_string(
		font,
		Vector2(tip_x + 14, ty),
		ability_name,
		HORIZONTAL_ALIGNMENT_LEFT,
		tip_w - 28,
		T.TIP_NAME_SIZE,
		school_color
	)


static func _draw_tooltip_info(
	ci: CanvasItem,
	font: Font,
	tip_x: float,
	tip_w: float,
	ty: float,
	school: String,
	ability_type: String,
	delivery: String,
) -> float:
	var info: Array[String] = []
	info.append(T.SCHOOL_LABELS.get(school, school))
	if ability_type != "":
		info.append(ability_type.capitalize())
	if delivery != "":
		info.append(delivery.capitalize())
	ci.draw_string(
		font,
		Vector2(tip_x + 14, ty),
		" \u00b7 ".join(info),
		HORIZONTAL_ALIGNMENT_LEFT,
		tip_w - 28,
		T.TIP_INFO_SIZE,
		T.TEXT_MUTED
	)
	return ty + T.TIP_LINE_H


static func _draw_tooltip_stats(
	ci: CanvasItem,
	font: Font,
	tip_x: float,
	tip_w: float,
	ty: float,
	flux_amount: float,
	flux_cost: String,
	commit_time: float,
	cooldown: float,
) -> float:
	var stats: Array[String] = []
	if flux_amount > 0.01:
		stats.append("Flux: %.0f" % flux_amount)
	elif flux_cost != "":
		stats.append("Flux: %s" % flux_cost.capitalize())
	if commit_time > 0.01:
		stats.append("%.1fs commit" % commit_time)
	if cooldown > 0.01:
		stats.append("%.0fs CD" % cooldown)
	if stats.size() > 0:
		ci.draw_string(
			font,
			Vector2(tip_x + 14, ty),
			" | ".join(stats),
			HORIZONTAL_ALIGNMENT_LEFT,
			tip_w - 28,
			T.TIP_STAT_SIZE,
			Color(0.85, 0.75, 0.3, 0.8)
		)
		ty += T.TIP_LINE_H
	return ty


static func _draw_tooltip_details(
	ci: CanvasItem,
	font: Font,
	tip_x: float,
	tip_w: float,
	ty: float,
	stat_details: Array[String],
) -> float:
	var col_w := (tip_w - 28) / 2.0
	var detail_i := 0
	while detail_i < stat_details.size():
		var left: String = stat_details[detail_i]
		ci.draw_string(
			font,
			Vector2(tip_x + 14, ty),
			left,
			HORIZONTAL_ALIGNMENT_LEFT,
			col_w,
			T.TIP_DETAIL_SIZE,
			Color(0.7, 0.85, 0.75, 0.85)
		)
		if detail_i + 1 < stat_details.size():
			var right: String = stat_details[detail_i + 1]
			ci.draw_string(
				font,
				Vector2(tip_x + 14 + col_w, ty),
				right,
				HORIZONTAL_ALIGNMENT_LEFT,
				col_w,
				T.TIP_DETAIL_SIZE,
				Color(0.7, 0.85, 0.75, 0.85)
			)
		detail_i += 2
		ty += T.TIP_LINE_H
	return ty


static func _draw_tooltip_affinity(
	ci: CanvasItem,
	font: Font,
	tip_x: float,
	tip_w: float,
	ty: float,
	affinity: String,
) -> float:
	var aff_color: Color
	match affinity:
		"primary":
			aff_color = Color(0.3, 0.8, 0.4)
		"secondary":
			aff_color = Color(0.7, 0.7, 0.3)
		_:
			aff_color = Color(0.7, 0.35, 0.3)
	var aff_labels := {"primary": "PRIMARY", "secondary": "SECONDARY", "off": "OFF-SPEC"}
	ci.draw_string(
		font,
		Vector2(tip_x + 14, ty),
		aff_labels.get(affinity, "OFF-SPEC"),
		HORIZONTAL_ALIGNMENT_LEFT,
		tip_w - 28,
		T.TIP_AFF_SIZE,
		aff_color
	)
	ty += 16
	# Separator
	ci.draw_line(
		Vector2(tip_x + 10, ty - 2),
		Vector2(tip_x + tip_w - 10, ty - 2),
		Color(T.PANEL_BORDER, 0.5),
		1.0
	)
	return ty + 6


static func _draw_tooltip_description(
	ci: CanvasItem,
	font: Font,
	tip_x: float,
	tip_w: float,
	ty: float,
	desc_lines: Array[String],
	implemented: bool,
) -> void:
	for line in desc_lines:
		ci.draw_string(
			font,
			Vector2(tip_x + 14, ty),
			line,
			HORIZONTAL_ALIGNMENT_LEFT,
			tip_w - 28,
			T.TIP_DESC_SIZE,
			Color(0.82, 0.84, 0.88, 0.9)
		)
		ty += T.TIP_DESC_LINE_H

	if not implemented:
		ty += 6
		ci.draw_string(
			font,
			Vector2(tip_x + 14, ty),
			"NOT YET IMPLEMENTED",
			HORIZONTAL_ALIGNMENT_LEFT,
			tip_w - 28,
			T.TIP_WARN_SIZE,
			Color(0.8, 0.4, 0.3, 0.9)
		)


static func _draw_scroll_indicators(
	ci: CanvasItem, grid_rect: Rect2, scroll_offset: float, max_scroll: float
) -> void:
	var font := ThemeDB.fallback_font
	if scroll_offset > 0:
		ci.draw_string(
			font,
			Vector2(grid_rect.get_center().x - 6, grid_rect.position.y + 14),
			"^",
			HORIZONTAL_ALIGNMENT_CENTER,
			12,
			12,
			T.TEXT_DIM
		)
	if scroll_offset < max_scroll:
		ci.draw_string(
			font,
			Vector2(grid_rect.get_center().x - 6, grid_rect.end.y - 4),
			"v",
			HORIZONTAL_ALIGNMENT_CENTER,
			12,
			12,
			T.TEXT_DIM
		)


static func _affinity_border_color(affinity: String, school_color: Color) -> Color:
	match affinity:
		"primary":
			return school_color
		"secondary":
			return Color(school_color, 0.5)
		_:
			return Color(0.5, 0.25, 0.25, 0.5)


static func _draw_card_text(
	ci: CanvasItem,
	font: Font,
	rect: Rect2,
	ability: Dictionary,
	implemented: bool,
	school_color: Color,
) -> void:
	var ability_name: String = ability.get("name", "???")
	var name_color := T.TEXT_PRIMARY if implemented else T.TEXT_DIM
	var name_parts := ability_name.split(" ", true, 1)
	ci.draw_string(
		font,
		Vector2(rect.position.x + 8, rect.position.y + 16),
		name_parts[0],
		HORIZONTAL_ALIGNMENT_LEFT,
		rect.size.x - 14,
		11,
		name_color
	)
	if name_parts.size() > 1:
		ci.draw_string(
			font,
			Vector2(rect.position.x + 8, rect.position.y + 30),
			name_parts[1],
			HORIZONTAL_ALIGNMENT_LEFT,
			rect.size.x - 14,
			11,
			name_color
		)
	var school: String = ability.get("school", "")
	var school_abbrev: String = T.SCHOOL_LABELS.get(school, school).to_upper().left(3)
	var type_letter: String = T.TYPE_LETTERS.get(ability.get("ability_type", ""), "?")
	ci.draw_string(
		font,
		Vector2(rect.position.x + 8, rect.end.y - 22),
		"%s  %s" % [school_abbrev, type_letter],
		HORIZONTAL_ALIGNMENT_LEFT,
		rect.size.x - 14,
		9,
		Color(school_color, 0.7) if implemented else T.TEXT_DIM
	)
	var flux_cost: String = ability.get("flux_cost", "")
	var dot_count: int = T.FLUX_DOT_COUNT.get(flux_cost, 0)
	for d in dot_count:
		ci.draw_rect(
			Rect2(rect.position.x + 8.0 + d * 10.0, rect.end.y - 10.0, 6.0, 4.0),
			T.ACCENT if implemented else T.TEXT_DIM
		)


static func _draw_slot_badge(ci: CanvasItem, font: Font, rect: Rect2, slot: int) -> void:
	if slot < 0:
		return
	var badge := Rect2(rect.end.x - 22, rect.position.y + 4, 18, 16)
	ci.draw_rect(badge, Color(T.ACCENT, 0.8))
	ci.draw_string(
		font,
		Vector2(badge.position.x + 3, badge.position.y + 12),
		T.SLOT_KEYBINDS[slot],
		HORIZONTAL_ALIGNMENT_CENTER,
		12,
		10,
		Color.WHITE
	)


static func _extract_tooltip_data(ability: Dictionary, font: Font) -> Dictionary:
	var school: String = ability.get("school", "")
	var description: String = ability.get("description", "")
	var implemented: bool = ability.get("implemented", false)
	var desc_lines := T.wrap_text(font, description, 360.0, T.TIP_DESC_SIZE)
	var stat_details := _build_stat_details(ability)
	var detail_rows := ceili(stat_details.size() / 2.0) if stat_details.size() > 0 else 0
	var tip_w := 400.0
	var tip_h := (
		100.0
		+ detail_rows * T.TIP_LINE_H
		+ desc_lines.size() * T.TIP_DESC_LINE_H
		+ (22.0 if not implemented else 0.0)
	)
	return {
		ability_name = ability.get("name", "???"),
		school = school,
		school_color = T.get_school_color(school),
		ability_type = ability.get("ability_type", ""),
		delivery = ability.get("delivery", ""),
		flux_cost = ability.get("flux_cost", ""),
		flux_amount = ability.get("flux_amount", 0.0),
		cooldown = ability.get("cooldown", 0.0),
		commit_time = ability.get("commit_time", 0.0),
		affinity = ability.get("affinity", "off"),
		implemented = implemented,
		stat_details = stat_details,
		desc_lines = desc_lines,
		tip_w = tip_w,
		tip_h = tip_h,
	}


static func _build_stat_details(ability: Dictionary) -> Array[String]:
	var details: Array[String] = []
	var checks := [
		["base_heal", "Heal: %.0f"],
		["base_damage", "Damage: %.0f"],
		["zone_heal_tick", "Tick Heal: %.0f"],
		["range", "Range: %.0fm"],
		["zone_radius", "Radius: %.0fm"],
		["zone_duration", "Duration: %.1fs"],
		["commit_time", "Channel: %.1fs"],
		["gcd", "GCD: %.1fs"],
	]
	for c in checks:
		var val: float = ability.get(c[0], 0.0)
		if val > 0.01:
			details.append(c[1] % val)
	return details


static func _compute_tooltip_pos(
	card_rect: Rect2, canvas_size: Vector2, tip_w: float, tip_h: float
) -> Vector2:
	var tip_x := card_rect.end.x + 8.0
	if tip_x + tip_w > canvas_size.x - 10:
		tip_x = card_rect.position.x - tip_w - 8.0
	var tip_y := clampf(card_rect.position.y, 10.0, canvas_size.y - tip_h - 10.0)
	return Vector2(tip_x, tip_y)
