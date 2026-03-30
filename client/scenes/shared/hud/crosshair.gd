extends Control

## Crosshair drawing node — delegates to parent GunnerHUD.


func _draw() -> void:
	var hud := get_parent() as Control
	if hud and hud.has_method("draw_crosshair"):
		hud.draw_crosshair(self)
