shader_type spatial;
render_mode blend_add, depth_draw_never, cull_disabled, unshaded;

uniform vec4 ring_color : source_color = vec4(0.9, 0.15, 0.1, 1.0);
uniform float expand_progress : hint_range(0.0, 1.0) = 0.0;
uniform float ring_thickness : hint_range(0.01, 0.2) = 0.12;
uniform float half_angle : hint_range(0.0, 3.14159) = 0.785; // 45 degrees = 90 degree arc

void fragment() {
	vec2 center_uv = UV * 2.0 - 1.0;
	float dist = length(center_uv);

	// Cone mask: discard fragments outside the forward arc
	// Forward direction is -Z in local space, which maps to UV.y > 0 (top half)
	float angle = atan(center_uv.x, -center_uv.y);
	if (abs(angle) > half_angle) {
		discard;
	}

	// Expanding ring
	float current_radius = expand_progress;
	float ring_dist = abs(dist - current_radius);
	float ring = 1.0 - smoothstep(0.0, ring_thickness, ring_dist);

	// Fade as ring expands
	float brightness = (1.0 - expand_progress * 0.7);

	// Discard outside unit circle
	if (dist > 1.0) {
		discard;
	}

	ALBEDO = ring_color.rgb * brightness;
	ALPHA = ring * ring_color.a * brightness;
}
