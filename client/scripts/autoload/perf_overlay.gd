extends Node

## Toggleable on-screen performance readout (F3). Off by default; costs nothing
## until shown. Exists to answer one question on weak hardware: are we CPU-bound
## (script/physics) or GPU-bound (fill rate, effects)? When the frame budget is
## much larger than CPU process + physics time, the GPU is the bottleneck and the
## fix is graphics quality (see SettingsManager); otherwise look at the server
## state rate or per-frame script work.

const TOGGLE_KEY := KEY_F3
const UPDATE_INTERVAL := 0.25

var _layer: CanvasLayer
var _label: Label
var _accum := 0.0


func _ready() -> void:
	process_mode = Node.PROCESS_MODE_ALWAYS
	_layer = CanvasLayer.new()
	_layer.layer = 128
	_layer.visible = false
	add_child(_layer)
	SettingsManager.settings_changed.connect(_apply_visibility)

	var panel := PanelContainer.new()
	panel.position = Vector2(8, 8)
	panel.modulate = Color(1, 1, 1, 0.92)
	_layer.add_child(panel)

	_label = Label.new()
	_label.add_theme_font_size_override("font_size", 13)
	_label.add_theme_color_override("font_color", Color(0.7, 0.95, 1.0))
	panel.add_child(_label)

	# The toggle is a persisted setting so the Settings panel and F3 stay in sync
	# and the overlay survives a restart if left on.
	_apply_visibility()


func _input(event: InputEvent) -> void:
	if event is InputEventKey and event.pressed and not event.echo:
		if (event as InputEventKey).keycode == TOGGLE_KEY:
			var on := bool(SettingsManager.get_value("ui", "perf_overlay", false))
			SettingsManager.set_value("ui", "perf_overlay", not on)


## Mirrors the persisted toggle onto the overlay. Driven by settings_changed, so
## both F3 and the Settings panel checkbox flow through the same path.
func _apply_visibility() -> void:
	_layer.visible = bool(SettingsManager.get_value("ui", "perf_overlay", false))
	if _layer.visible:
		_refresh()


func _process(delta: float) -> void:
	if not _layer.visible:
		return
	_accum += delta
	if _accum >= UPDATE_INTERVAL:
		_accum = 0.0
		_refresh()


func _refresh() -> void:
	var fps := Performance.get_monitor(Performance.TIME_FPS)
	var frame_ms := (1000.0 / fps) if fps > 0.0 else 0.0
	var process_ms := Performance.get_monitor(Performance.TIME_PROCESS) * 1000.0
	var physics_ms := Performance.get_monitor(Performance.TIME_PHYSICS_PROCESS) * 1000.0
	var cpu_ms := process_ms + physics_ms
	var draw_calls := Performance.get_monitor(Performance.RENDER_TOTAL_DRAW_CALLS_IN_FRAME)
	var prims := Performance.get_monitor(Performance.RENDER_TOTAL_PRIMITIVES_IN_FRAME)
	var vram := Performance.get_monitor(Performance.RENDER_VIDEO_MEM_USED) / (1024.0 * 1024.0)
	var nodes := Performance.get_monitor(Performance.OBJECT_NODE_COUNT)

	# If the frame takes much longer than the CPU spent on it, the GPU is the wall.
	var bound := "GPU-bound" if frame_ms > cpu_ms * 1.4 + 1.0 else "CPU-bound"

	var q := SettingsManager.quality_tier()
	var q_name: String = SettingsManager.QUALITY_LEVELS[q]
	var renderer := str(
		ProjectSettings.get_setting("rendering/renderer/rendering_method", "forward_plus")
	)
	var scale := get_tree().root.scaling_3d_scale

	_label.text = (
		"%.0f FPS  (%.1f ms/frame)   [%s]\n" % [fps, frame_ms, bound]
		+ "CPU: %.1f ms (proc %.1f + phys %.1f)\n" % [cpu_ms, process_ms, physics_ms]
		+ "Draw calls: %d   Prims: %.0fk\n" % [draw_calls, prims / 1000.0]
		+ "VRAM: %.0f MB   Nodes: %d\n" % [vram, nodes]
		+ "Quality: %s   Scale: %.0f%%   %s" % [q_name, scale * 100.0, renderer]
	)
