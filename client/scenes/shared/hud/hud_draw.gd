extends RefCounted

## Stateless HUD drawing/formatting helpers shared across HUD scripts.
## Referenced via preload (const HudDraw) rather than a global class_name.


static func format_mmss(seconds: float) -> String:
	var total := int(seconds)
	return "%02d:%02d" % [total / 60, total % 60]


static func status_bar(
	ci: CanvasItem, rect: Rect2, ratio: float, fill_color: Color, bg: Color, border: Color
) -> void:
	ci.draw_rect(rect, bg)
	if ratio > 0.0:
		var fill_width := maxf(rect.size.x * ratio, 0.0)
		ci.draw_rect(Rect2(rect.position, Vector2(fill_width, rect.size.y)), fill_color)
	ci.draw_rect(rect, border, false, 1.0)
