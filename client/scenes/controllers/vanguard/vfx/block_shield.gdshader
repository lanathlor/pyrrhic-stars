shader_type spatial;
render_mode blend_add, depth_draw_never, cull_disabled, unshaded;

uniform vec4 shield_color : source_color = vec4(0.9, 0.15, 0.1, 0.6);
uniform float pulse_speed : hint_range(0.0, 10.0) = 2.0;
uniform float opacity : hint_range(0.0, 1.0) = 1.0;
uniform float hex_scale : hint_range(1.0, 20.0) = 8.0;

// Hexagonal distance function
float hex_dist(vec2 p) {
	p = abs(p);
	return max(dot(p, normalize(vec2(1.0, 1.73))), p.x);
}

vec4 hex_coords(vec2 uv) {
	vec2 r = vec2(1.0, 1.73);
	vec2 h = r * 0.5;
	vec2 a = mod(uv, r) - h;
	vec2 b = mod(uv - h, r) - h;
	vec2 gv;
	if (length(a) < length(b)) {
		gv = a;
	} else {
		gv = b;
	}
	float d = hex_dist(gv);
	return vec4(gv, d, 0.0);
}

void fragment() {
	// Fresnel — glows at glancing angles, near-invisible straight on
	float fresnel = pow(1.0 - abs(dot(NORMAL, VIEW)), 1.5);

	// Hex pattern
	vec2 hex_uv = UV * hex_scale;
	vec4 hc = hex_coords(hex_uv);
	float hex_edge = smoothstep(0.4, 0.5, hc.z);

	// Pulse animation
	float pulse = 0.7 + 0.3 * sin(TIME * pulse_speed);

	// Combine: fresnel glow + hex pattern edges
	float pattern = mix(fresnel * 0.5, 1.0, hex_edge * fresnel);
	float alpha = pattern * shield_color.a * opacity * pulse;

	ALBEDO = shield_color.rgb * (1.0 + fresnel * 0.5);
	ALPHA = alpha;
}
