class_name Player
extends CharacterBody2D

## Movement speed, exposed to the inspector (exports_var).
@export var speed: int = 200

## Emitted when the player fires (declares_signal).
signal fired(weapon)

## Node reached at ready time (references_node, node_reach egress).
@onready var sprite = $Body/Sprite

## Static resource dependency (loads_resource via preload, resource_load egress).
const BULLET = preload("res://bullet.tscn")

## Lifecycle ingress; connects a signal and joins a group.
func _ready() -> void:
	add_to_group("players")
	fired.connect(_on_fired)

## Input-handler ingress; reads an input action.
func _input(event) -> void:
	if Input.is_action_pressed("jump"):
		jump()

## Notification ingress.
func _notification(what) -> void:
	pass

## Signal-handler ingress (_on_ prefix); accesses an autoload and emits a signal.
func _on_fired(weapon) -> void:
	GameState.score += 1
	emit_signal("fired", weapon)

## Exercises the remaining egress detectors and a self-call.
func jump() -> void:
	fired.emit()
	var scene = load("res://level.tscn")
	get_tree().change_scene_to_file("res://level.tscn")
	get_tree().call_group("enemies", "flee")
	get_node("Body/Sprite").play("jump")
	rpc("take_hit", 5)
	rpc_id(1, "take_hit", 5)
	save_state()

## File I/O egress.
func save_state() -> void:
	var f = FileAccess.open("user://save.dat", FileAccess.WRITE)

## RPC endpoint ingress + rpc_endpoint edge.
@rpc("any_peer")
func take_hit(amount) -> void:
	GameState.score -= amount
