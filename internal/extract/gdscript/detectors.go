package gdscript

import (
	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/dylanblakemore/godarch/internal/model"
)

// The functions below are the egress detectors of DESIGN §3.2/§4: each emits the
// edge and boundary for one reference kind. A target that cannot be pinned from
// the file alone (a dynamic string, an untyped emitter) is left unresolved with
// its raw expression retained, never dropped.

func (e *extractor) emitSignal(owner, emitterType, signal string, line int) {
	key := model.SignalKey(emitterType, signal)
	meta := map[string]any{"signal": signal}
	e.addEdge(model.EdgeEmitsSignal, owner, string(key), false, 1.0, line, meta)
	e.addBoundary(model.DirectionEgress, model.BoundarySignalEmit, owner, key, line, meta)
}

func (e *extractor) connectSignal(owner, emitterType, signal string, line int) {
	key := model.SignalKey(emitterType, signal)
	meta := map[string]any{"signal": signal}
	e.addEdge(model.EdgeConnectsSignal, owner, string(key), false, 1.0, line, meta)
	e.addBoundary(model.DirectionEgress, model.BoundarySignalConnect, owner, key, line, meta)
}

// loadResource emits a loads_resource edge for preload()/load(). A literal path
// resolves to its res:// node when known; a dynamic argument is kept as the raw
// expression, unresolved and low-confidence.
func (e *extractor) loadResource(owner string, args *sitter.Node, idx, line int, static bool) {
	props := map[string]any{}
	if static {
		props["via"] = "preload"
	}
	if path, ok := e.stringArg(args, idx); ok {
		key := model.ResourceKey(path)
		_, exists := e.p.Nodes[model.NormalizePath(path)]
		e.addEdge(model.EdgeLoadsResource, owner, string(key), exists, 1.0, line, nilIfEmpty(props))
		e.addBoundary(model.DirectionEgress, model.BoundaryResourceLoad, owner, key, line, nilIfEmpty(props))
		return
	}
	raw := e.argOrExpr(args, idx)
	props["dynamic"] = true
	props["expr"] = raw
	e.addEdge(model.EdgeLoadsResource, owner, raw, false, 0.5, line, props)
	e.addBoundary(model.DirectionEgress, model.BoundaryResourceLoad, owner, "", line, props)
}

// nodeReachArg handles the string-argument form get_node("Path").
func (e *extractor) nodeReachArg(owner string, args *sitter.Node, line int) {
	if path, ok := e.stringArg(args, 0); ok {
		e.nodeReach(owner, path, line, 1.0)
		return
	}
	raw := e.argOrExpr(args, 0)
	key := model.NodePathKey(raw)
	props := map[string]any{"dynamic": true, "expr": raw}
	e.addEdge(model.EdgeReferencesNode, owner, string(key), false, 0.5, line, props)
	e.addBoundary(model.DirectionEgress, model.BoundaryNodeReach, owner, key, line, props)
}

func (e *extractor) nodeReach(owner, path string, line int, conf float64) {
	key := model.NodePathKey(path)
	meta := map[string]any{"path": path}
	e.addEdge(model.EdgeReferencesNode, owner, string(key), false, conf, line, meta)
	e.addBoundary(model.DirectionEgress, model.BoundaryNodeReach, owner, key, line, meta)
}

// inGroup records add_to_group("g"): the script's node joins group g.
func (e *extractor) inGroup(group string, line int) {
	e.addEdge(model.EdgeInGroup, e.file, model.GroupID(group), true, 1.0, line, map[string]any{"group": group})
}

func (e *extractor) sceneChange(owner string, args *sitter.Node, line int) {
	if path, ok := e.stringArg(args, 0); ok {
		key := model.ResourceKey(path)
		_, exists := e.p.Nodes[model.NormalizePath(path)]
		e.addEdge(model.EdgeChangesSceneTo, owner, string(key), exists, 1.0, line, nil)
		e.addBoundary(model.DirectionEgress, model.BoundarySceneChange, owner, key, line, nil)
		return
	}
	raw := e.argOrExpr(args, 0)
	props := map[string]any{"dynamic": true, "expr": raw}
	e.addEdge(model.EdgeChangesSceneTo, owner, raw, false, 0.5, line, props)
	e.addBoundary(model.DirectionEgress, model.BoundarySceneChange, owner, "", line, props)
}

func (e *extractor) rpcCall(owner string, args *sitter.Node, methodIdx, line int) {
	m, ok := e.stringArg(args, methodIdx)
	if !ok {
		return
	}
	key := model.RPCKey("*", m)
	meta := map[string]any{"method": m}
	e.addEdge(model.EdgeRPCCall, owner, string(key), false, 1.0, line, meta)
	e.addBoundary(model.DirectionEgress, model.BoundaryRPCCall, owner, key, line, meta)
}

func (e *extractor) usesAction(owner, action string, line int) {
	e.addEdge(model.EdgeUsesAction, owner, model.ActionID(action), true, 1.0, line, map[string]any{"action": action})
}

func (e *extractor) callsGroup(owner, group string, line int) {
	e.addEdge(model.EdgeCallsGroup, owner, model.GroupID(group), true, 1.0, line, map[string]any{"group": group})
	e.addBoundary(model.DirectionEgress, model.BoundaryGroupCall, owner, model.GroupKey(group), line, map[string]any{"group": group})
}

// fileIO emits the file_io egress boundary (there is no file_io edge type;
// file access is a boundary, not a graph dependency).
func (e *extractor) fileIO(owner, path string, line int) {
	var key model.MatchKey
	meta := map[string]any{}
	if path != "" {
		key = model.ResourceKey(path)
		meta["path"] = path
	}
	e.addBoundary(model.DirectionEgress, model.BoundaryFileIO, owner, key, line, nilIfEmpty(meta))
}

// autoloadAccess records <Autoload>.member access, deduplicated per owner.
func (e *extractor) autoloadAccess(owner, name string, line int) {
	sig := owner + "|" + name
	if e.seenAutoload[sig] {
		return
	}
	e.seenAutoload[sig] = true
	e.addEdge(model.EdgeAccessesAutoload, owner, model.AutoloadID(name), true, 1.0, line, nil)
	e.addBoundary(model.DirectionEgress, model.BoundaryAutoloadAccess, owner, model.AutoloadKey(name), line, nil)
}

// nilIfEmpty returns nil for an empty map so addEdge/addBoundary omit empty
// property/meta objects from the golden output.
func nilIfEmpty(m map[string]any) map[string]any {
	if len(m) == 0 {
		return nil
	}
	return m
}
