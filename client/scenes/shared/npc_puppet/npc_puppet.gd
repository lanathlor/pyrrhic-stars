extends Node3D

## Lightweight NPC visual puppet.
## Pins CharacterModel Y to 0 every frame to prevent animation drift.

@onready var character_model: Node3D = $CharacterModel


func _physics_process(_delta: float) -> void:
	if character_model:
		character_model.position.y = 0.0
