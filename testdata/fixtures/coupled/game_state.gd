extends Node

## Global score, registered as the GameState autoload in project.godot.
var score: int = 0

## Emitted when the score changes; the HUD connects to it.
signal score_changed(value)
