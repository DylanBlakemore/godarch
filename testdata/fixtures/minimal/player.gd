extends Node

## The player's movement speed, exposed to the editor (one @export).
@export var speed: int = 100

## Emitted when the player dies (one declared signal).
signal died

## Editor-connected handler for the Timer's timeout signal.
func _on_timer_timeout() -> void:
	died.emit()
