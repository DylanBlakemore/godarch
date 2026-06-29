package model

import (
	"strconv"
	"strings"
)

// Identifier scheme (DESIGN §8a). IDs are the deterministic, human-readable
// join keys for resolution and the anchors docs reference. The forms are:
//
//	file (script/scene/resource/asset)  res://<path>
//	symbol inside a script              <file>::<symbol>
//	signal declared in a script         <file>::signal:<name>
//	scene-internal node                 <scene>::<NodePath>
//	autoload                            autoload:<Name>
//	input action                        action:<name>
//	group                               group:<name>
//	collision layer                     layer:<index>
//	class                               class:<Name>
//	GDExtension                         ext:<res://path>
//	doc                                 <repo-relative path>
//
// File-scoped IDs share the "<file>::<rest>" shape; the parsers below
// disambiguate symbol / signal / scene-node by the "signal:" marker and the
// owning file's extension.

const (
	scopeSep   = "::"
	signalMark = "signal:"
)

// FileID returns the canonical ID for a file: its normalized res:// path. The
// kind-specific aliases below exist for call-site clarity; all four produce the
// same form because a file's identity is its path.
func FileID(resPath string) string { return NormalizePath(resPath) }

// ScriptID returns the ID of a .gd/.cs file.
func ScriptID(resPath string) string { return FileID(resPath) }

// SceneID returns the ID of a .tscn/.scn file.
func SceneID(resPath string) string { return FileID(resPath) }

// ResourceID returns the ID of a .tres/.res file.
func ResourceID(resPath string) string { return FileID(resPath) }

// AssetID returns the ID of an asset file (texture, audio, model, ...).
func AssetID(resPath string) string { return FileID(resPath) }

// SymbolID returns <file>::<symbol> for a method/var/inner symbol in a script.
func SymbolID(file, symbol string) string {
	return NormalizePath(file) + scopeSep + symbol
}

// SignalDeclID returns <file>::signal:<name> for a signal declared in a script.
func SignalDeclID(file, name string) string {
	return NormalizePath(file) + scopeSep + signalMark + name
}

// SceneNodeID returns <scene>::<NodePath> for a node inside a scene tree.
func SceneNodeID(scene, nodePath string) string {
	return NormalizePath(scene) + scopeSep + nodePath
}

// AutoloadID returns autoload:<Name>.
func AutoloadID(name string) string { return "autoload:" + name }

// ActionID returns action:<name>.
func ActionID(name string) string { return "action:" + name }

// GroupID returns group:<name>.
func GroupID(name string) string { return "group:" + name }

// LayerID returns layer:<index>.
func LayerID(index int) string { return "layer:" + strconv.Itoa(index) }

// ClassID returns class:<Name>.
func ClassID(name string) string { return "class:" + name }

// ExtensionID returns ext:<res://path> for a .gdextension file.
func ExtensionID(resPath string) string { return "ext:" + NormalizePath(resPath) }

// DocID returns the ID of a developer doc: its repo-relative path, cleaned but
// left scheme-less (docs live in the repo, not under res://).
func DocID(repoRelPath string) string {
	return strings.TrimPrefix(strings.TrimSpace(repoRelPath), "./")
}

// splitFileScoped splits a "<file>::<rest>" ID on the first scope separator.
func splitFileScoped(id string) (file, rest string, ok bool) {
	return strings.Cut(id, scopeSep)
}

// isSceneFile reports whether a file ID names a scene (so a file-scoped ID on it
// is a scene node rather than a script symbol).
func isSceneFile(file string) bool {
	return strings.HasSuffix(file, ".tscn") || strings.HasSuffix(file, ".scn")
}

// ParseSymbolID parses <file>::<symbol>. It returns ok=false for signal-decl
// IDs (the rest starts with "signal:") and for scene-node IDs (the file is a
// scene), so each ID form parses unambiguously.
func ParseSymbolID(id string) (file, symbol string, ok bool) {
	file, rest, ok := splitFileScoped(id)
	if !ok || strings.HasPrefix(rest, signalMark) || isSceneFile(file) {
		return "", "", false
	}
	return file, rest, true
}

// ParseSignalDeclID parses <file>::signal:<name>.
func ParseSignalDeclID(id string) (file, name string, ok bool) {
	file, rest, ok := splitFileScoped(id)
	if !ok || !strings.HasPrefix(rest, signalMark) {
		return "", "", false
	}
	return file, strings.TrimPrefix(rest, signalMark), true
}

// ParseSceneNodeID parses <scene>::<NodePath>.
func ParseSceneNodeID(id string) (scene, nodePath string, ok bool) {
	scene, rest, ok := splitFileScoped(id)
	if !ok || !isSceneFile(scene) {
		return "", "", false
	}
	return scene, rest, true
}
