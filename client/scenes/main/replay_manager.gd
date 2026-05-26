extends Node

## Replay system: browser and playback.

var ctrl: Node

var replay_browser: CanvasLayer = null
var replay_scene: Node3D = null


func _ready() -> void:
	ctrl = get_parent()


func enter_replay_browser() -> void:
	ctrl.state = ctrl.GameState.REPLAY_BROWSER
	ctrl._menu_layer.visible = false

	var browser_script: GDScript = load("res://scripts/replay/replay_browser.gd")
	replay_browser = CanvasLayer.new()
	replay_browser.set_script(browser_script)
	ctrl.add_child(replay_browser)
	replay_browser.replay_selected.connect(_on_replay_selected)
	replay_browser.browser_closed.connect(_on_browser_closed)


func _on_replay_selected(replay: Variant) -> void:
	# Clean up browser
	if replay_browser:
		replay_browser.queue_free()
		replay_browser = null

	ctrl.state = ctrl.GameState.REPLAY

	# Create replay scene
	var scene_script: GDScript = load("res://scripts/replay/replay_scene.gd")
	replay_scene = Node3D.new()
	replay_scene.set_script(scene_script)
	ctrl.add_child(replay_scene)
	replay_scene.replay_exited.connect(_on_replay_exited)
	replay_scene.start_replay(replay)


func _on_browser_closed() -> void:
	if replay_browser:
		replay_browser.queue_free()
		replay_browser = null
	ctrl.state = ctrl.GameState.MENU
	ctrl._menu_layer.visible = true


func _on_replay_exited() -> void:
	if replay_scene:
		replay_scene.queue_free()
		replay_scene = null
	ctrl.state = ctrl.GameState.MENU
	ctrl._menu_layer.visible = true
	Input.mouse_mode = Input.MOUSE_MODE_VISIBLE
