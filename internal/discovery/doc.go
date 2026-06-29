// Package discovery walks a Godot project directory, classifies its files
// (scripts, scenes, resources, assets), parses project.godot, and builds the
// uid:// ↔ res:// map that the rest of the pipeline relies on.
//
// Discovery depends on internal/model only.
package discovery
