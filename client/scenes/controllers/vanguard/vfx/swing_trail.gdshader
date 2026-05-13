shader_type spatial;
render_mode blend_add, depth_draw_never, cull_disabled, unshaded;

uniform vec4 trail_color : source_color = vec4(0.9, 0.15, 0.1, 1.0);
uniform vec4 core_color : source_color = vec4(1.0, 0.4, 0.2, 1.0);
uniform float fade : hint_range(0.0, 1.0) = 1.0;
uniform float emission_boost : hint_range(1.0, 8.0) = 3.0;
// Progressive reveal: 0=nothing visible, 1=full arc
uniform float reveal : hint_range(0.0, 1.0) = 1.0;

void vertex() {
	// Scale ribbon height based on position along the arc.
	// Near UV.y=0 (trailing edge) the ribbon is thin,
	// near UV.y=reveal (leading edge) it's full size.
	float along = UV.y; // 0=start of arc, 1=end of arc
	float growth = smoothstep(0.0, max(reveal, 0.01), along);
	// Pinch vertices toward center (UV.x=0.5) when growth is small
	float pinch = mix(0.15, 1.0, growth);
	VERTEX.y = mix(VERTEX.y, 0.0, 1.0 - pinch);
}

void fragment() {
	float along = UV.y;

	// Discard fragments past the reveal point
	if (along > reveal) {
		discard;
	}

	// Width fade — soft falloff at ribbon edges
	float edge_dist = abs(UV.x - 0.5) * 2.0;
	float edge_fade = 1.0 - smoothstep(0.7, 1.0, edge_dist);

	// Trailing fade — old part of the arc fades out behind the leading edge
	float trail_fade = smoothstep(0.0, reveal * 0.6, along);
	// Bright leading edge pulse
	float lead_glow = smoothstep(reveal - 0.15, reveal, along);

	// Color: hot core center, red outer
	float core_blend = smoothstep(0.0, 0.4, edge_dist);
	vec3 col = mix(core_color.rgb, trail_color.rgb, core_blend);
	// Leading edge gets extra brightness
	col *= emission_boost * (1.0 + lead_glow * 0.5);

	float alpha = edge_fade * trail_fade * fade;

	ALBEDO = col;
	ALPHA = alpha;
}
