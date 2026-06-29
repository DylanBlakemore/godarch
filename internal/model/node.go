package model

// Kind enumerates the entity types that become graph nodes (DESIGN §3.1).
type Kind string

const (
	KindScript    Kind = "script"     // a .gd/.cs file
	KindScene     Kind = "scene"      // a .tscn/.scn file
	KindSceneNode Kind = "scene_node" // a [node] inside a scene tree
	KindResource  Kind = "resource"   // a .tres/.res file
	KindAutoload  Kind = "autoload"   // a singleton registered in project.godot
	KindAsset     Kind = "asset"      // textures, audio, models, fonts, shaders, themes
	KindAction    Kind = "action"     // an input action
	KindGroup     Kind = "group"      // a node group
	KindLayer     Kind = "layer"      // a collision layer
	KindSignal    Kind = "signal"     // a signal declared in a script
	KindClass     Kind = "class"      // a class_name / built-in / native class
	KindExtension Kind = "extension"  // a .gdextension and the classes it registers
	KindDoc       Kind = "doc"        // a developer documentation file
)

// AllKinds lists every Kind. Used by tests to guarantee DESIGN §3.1 coverage
// and available to callers that need to enumerate node types.
var AllKinds = []Kind{
	KindScript, KindScene, KindSceneNode, KindResource, KindAutoload,
	KindAsset, KindAction, KindGroup, KindLayer, KindSignal,
	KindClass, KindExtension, KindDoc,
}

// Node is a single entity in the dependency graph. It mirrors archi's generic
// node with typed properties: Identity holds the kind-specific facts known at
// extraction time, Properties holds values derived later by analysis.
type Node struct {
	ID         string         `json:"id"`                   // canonical, per the identifier scheme
	Kind       Kind           `json:"kind"`                 //
	Path       string         `json:"path,omitempty"`       // res:// path, repo-relative for docs, "" for pure concepts
	Line       int            `json:"line,omitempty"`       // 0 if N/A
	Identity   map[string]any `json:"identity,omitempty"`   // {name, node_type, parent_path, arity, ...}
	Properties map[string]any `json:"properties,omitempty"` // {fan_in, in_cycle, complexity, ...}
}
