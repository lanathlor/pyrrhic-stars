shader_type spatial;
render_mode blend_add, depth_draw_opaque, cull_disabled, unshaded;

// Beam color parameters — heal green-cyan
uniform vec4 beam_color : source_color = vec4(0.2, 0.9, 0.5, 0.6);
uniform vec4 core_color : source_color = vec4(0.5, 1.0, 0.7, 0.9);
uniform float core_width : hint_range(0.0, 1.0) = 0.15;

// Animation
uniform float scroll_speed : hint_range(0.0, 5.0) = 0.8;
uniform float pulse_speed : hint_range(0.0, 5.0) = 1.5;
uniform float pulse_strength : hint_range(0.0, 1.0) = 0.3;

// Fresnel / edge glow
uniform float fresnel_power : hint_range(0.5, 8.0) = 2.5;
uniform float fresnel_strength : hint_range(0.0, 3.0) = 1.5;

// Noise-like pattern (procedural)
uniform float pattern_scale : hint_range(1.0, 40.0) = 12.0;
uniform float pattern_intensity : hint_range(0.0, 1.0) = 0.4;

// Fade for start/stop transitions
uniform float fade : hint_range(0.0, 1.0) = 1.0;

float hash(vec2 p) {
	vec3 p3 = fract(vec3(p.xyx) * 0.1031);
	p3 += dot(p3, p3.yzx + 33.33);
	return fract((p3.x + p3.y) * p3.z);
}

float noise(vec2 p) {
	vec2 i = floor(p);
	vec2 f = fract(p);
	f = f * f * (3.0 - 2.0 * f);

	float a = hash(i);
	float b = hash(i + vec2(1.0, 0.0));
	float c = hash(i + vec2(0.0, 1.0));
	float d = hash(i + vec2(1.0, 1.0));

	return mix(mix(a, b, f.x), mix(c, d, f.x), f.y);
}

float fbm(vec2 p) {
	float v = 0.0;
	float a = 0.5;
	vec2 shift = vec2(100.0);
	for (int i = 0; i < 3; i++) {
		v += a * noise(p);
		p = p * 2.0 + shift;
		a *= 0.5;
	}
	return v;
}

void fragment() {
	vec2 uv = UV;

	float time = TIME;
	vec2 scroll_uv = vec2(uv.x * pattern_scale, uv.y * pattern_scale - time * scroll_speed);
	float energy_pattern = fbm(scroll_uv);
	float energy_pattern2 = fbm(scroll_uv * 1.5 + vec2(time * 0.3, 0.0));
	float pattern = mix(energy_pattern, energy_pattern2, 0.5);

	float dist_from_center = abs(uv.x - 0.5) * 2.0;
	float core = smoothstep(core_width + 0.05, core_width - 0.05, dist_from_center);

	float edge_fade = smoothstep(1.0, 0.7, dist_from_center);

	float pulse = 1.0 + pulse_strength * sin(time * pulse_speed);

	float fresnel = pow(1.0 - abs(dot(NORMAL, VIEW)), fresnel_power) * fresnel_strength;

	vec3 base = beam_color.rgb * (1.0 + pattern * pattern_intensity);
	vec3 col = mix(base, core_color.rgb, core * 0.8);
	col += beam_color.rgb * fresnel;
	col *= pulse;

	float alpha = edge_fade * beam_color.a;
	alpha *= (1.0 + pattern * 0.3);
	alpha *= pulse;
	alpha = clamp(alpha, 0.0, 1.0);

	float y_fade = smoothstep(0.0, 0.08, uv.y) * smoothstep(1.0, 0.92, uv.y);
	alpha *= y_fade;

	// Apply fade for start/stop transitions
	alpha *= fade;

	ALBEDO = col;
	ALPHA = alpha;
}
