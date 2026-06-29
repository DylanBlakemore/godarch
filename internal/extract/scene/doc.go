// Package scene extracts nodes and edges from Godot's INI-style files:
// .tscn / .tres / .import / project.godot. It emits schema-conformant
// model types; it does not resolve cross-file references or persist anything.
//
// Implemented in milestone 01. Depends on internal/model only.
package scene
