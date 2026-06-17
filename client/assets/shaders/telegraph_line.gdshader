shader_type spatial;
render_mode unshaded, cull_disabled;

// Rectangular telegraph (charge / ranged aim). UV.y runs from the source (0)
// to the far end (1). The danger fills along the length as fill 0..1.
uniform vec4 color : source_color = vec4(1.0, 0.3, 0.0, 0.35);
uniform vec4 edge_color : source_color = vec4(1.0, 0.5, 0.0, 0.9);
uniform float edge_width : hint_range(0.0, 0.2) = 0.06;
uniform float fade : hint_range(0.0, 1.0) = 1.0;
uniform float fill : hint_range(0.0, 1.0) = 1.0;

void fragment() {
	float bx = min(UV.x, 1.0 - UV.x);
	float by = min(UV.y, 1.0 - UV.y);
	float border = min(bx, by);
	float ring = 1.0 - smoothstep(0.0, edge_width, border);
	float interior = 1.0 - smoothstep(fill - 0.03, fill, UV.y);
	ALBEDO = mix(color.rgb, edge_color.rgb, ring);
	ALPHA = max(color.a * interior, edge_color.a * ring) * fade;
}
