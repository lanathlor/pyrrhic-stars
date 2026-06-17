shader_type spatial;
render_mode unshaded, cull_disabled;

uniform vec4 color : source_color = vec4(1.0, 0.1, 0.1, 0.45);
uniform vec4 edge_color : source_color = vec4(1.0, 0.2, 0.1, 0.9);
uniform float edge_width : hint_range(0.0, 0.2) = 0.08;
uniform float fade : hint_range(0.0, 1.0) = 1.0;
// fill: interior fill radius 0..1. 0 = empty outline, 1 = full disc.
// Server-driven telegraphs animate this from 0 (commit) to 1 (execution).
uniform float fill : hint_range(0.0, 1.0) = 1.0;

void fragment() {
	vec2 center_uv = UV * 2.0 - 1.0;
	float dist = length(center_uv);
	if (dist > 1.0) {
		discard;
	}
	float edge_inner = 1.0 - edge_width;
	float ring = smoothstep(edge_inner - 0.02, edge_inner, dist);
	float interior = 1.0 - smoothstep(fill - 0.03, fill, dist);
	ALBEDO = mix(color.rgb, edge_color.rgb, ring);
	ALPHA = max(color.a * interior, edge_color.a * ring) * fade;
}
